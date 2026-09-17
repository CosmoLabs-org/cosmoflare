package webhook

import (
	"context"
	"errors"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// fakeLimitsSource counts Snapshot calls and returns a fixed snapshot (or an
// error) so cache behavior is observable without network.
type fakeLimitsSource struct {
	calls int
	snap  *cosmoflare.LimitsSnapshot
	err   error
}

func (f *fakeLimitsSource) Snapshot(ctx context.Context, bucket string) (*cosmoflare.LimitsSnapshot, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.snap, nil
}

// newLimitsTestCache builds a cache already bound to src for the given
// account, collapsing the two-line construction every cache test repeated.
func newLimitsTestCache(ttl time.Duration, src *fakeLimitsSource, account string) *LimitsSnapshotCache {
	c := NewLimitsSnapshotCache(ttl)
	c.SetSource(src, account)
	return c
}

// TestLimitsSnapshotCache_ServesOncePerTTL verifies the cache's core promise:
// repeated reads inside the TTL are served from memory (exactly one fetch)
// and hand back the identical snapshot pointer the source produced.
func TestLimitsSnapshotCache_ServesOncePerTTL(t *testing.T) {
	t.Parallel()
	src := &fakeLimitsSource{snap: &cosmoflare.LimitsSnapshot{}}
	c := newLimitsTestCache(time.Hour, src, "acct-1")

	for i := 0; i < 3; i++ {
		snap, err := c.Snapshot(context.Background(), "")
		if err != nil {
			t.Fatalf("Snapshot call %d: %v", i, err)
		}
		if snap != src.snap {
			t.Fatalf("call %d returned a different snapshot pointer", i)
		}
	}
	if src.calls != 1 {
		t.Errorf("source fetched %d time(s) inside TTL, want 1", src.calls)
	}
}

// TestLimitsSnapshotCache_RefetchesAfterInvalidation verifies that switching
// the account invalidates the cached snapshot, forcing a fresh fetch instead
// of leaking one account's limits into another's view.
func TestLimitsSnapshotCache_RefetchesAfterInvalidation(t *testing.T) {
	t.Parallel()
	src := &fakeLimitsSource{snap: &cosmoflare.LimitsSnapshot{}}
	c := newLimitsTestCache(time.Hour, src, "acct-1")

	if _, err := c.Snapshot(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	c.SetSource(src, "acct-2") // account switch invalidates
	if _, err := c.Snapshot(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if src.calls != 2 {
		t.Errorf("source fetched %d time(s) after account switch, want 2", src.calls)
	}
}

// TestLimitsSnapshotCache_ZeroTTLDisablesCaching verifies that ttl=0 is an
// explicit "always fetch" mode: every read goes through to the source.
func TestLimitsSnapshotCache_ZeroTTLDisablesCaching(t *testing.T) {
	t.Parallel()
	src := &fakeLimitsSource{snap: &cosmoflare.LimitsSnapshot{}}
	c := newLimitsTestCache(0, src, "acct-1")

	for i := 0; i < 2; i++ {
		if _, err := c.Snapshot(context.Background(), ""); err != nil {
			t.Fatal(err)
		}
	}
	if src.calls != 2 {
		t.Errorf("source fetched %d time(s) with ttl=0, want 2", src.calls)
	}
}

// TestLimitsSnapshotCache_ErrorsNotCached verifies that a failed fetch is
// never memoized: once the source recovers, the very next read succeeds and
// re-consults the source, so a transient outage cannot pin a stale error.
func TestLimitsSnapshotCache_ErrorsNotCached(t *testing.T) {
	t.Parallel()
	src := &fakeLimitsSource{err: errors.New("boom")}
	c := newLimitsTestCache(time.Hour, src, "acct-1")

	if _, err := c.Snapshot(context.Background(), ""); err == nil {
		t.Fatal("first call should fail")
	}
	// Simulate the outage ending between two reads.
	src.err = nil
	src.snap = &cosmoflare.LimitsSnapshot{}
	if _, err := c.Snapshot(context.Background(), ""); err != nil {
		t.Fatalf("second call should succeed after error cleared: %v", err)
	}
	if src.calls != 2 {
		t.Errorf("source fetched %d time(s), want 2 (error must not be cached)", src.calls)
	}
}

// TestLimitsSnapshotCache_NoSourceErrors verifies that reading before any
// source is configured returns an error rather than a nil snapshot that
// callers would dereference.
func TestLimitsSnapshotCache_NoSourceErrors(t *testing.T) {
	t.Parallel()
	c := NewLimitsSnapshotCache(time.Hour)
	if _, err := c.Snapshot(context.Background(), ""); err == nil {
		t.Fatal("Snapshot with no source set must error")
	}
}
