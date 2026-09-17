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

func TestLimitsSnapshotCache_ServesOncePerTTL(t *testing.T) {
	src := &fakeLimitsSource{snap: &cosmoflare.LimitsSnapshot{}}
	c := NewLimitsSnapshotCache(time.Hour)
	c.SetSource(src, "acct-1")

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

func TestLimitsSnapshotCache_RefetchesAfterInvalidation(t *testing.T) {
	src := &fakeLimitsSource{snap: &cosmoflare.LimitsSnapshot{}}
	c := NewLimitsSnapshotCache(time.Hour)
	c.SetSource(src, "acct-1")

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

func TestLimitsSnapshotCache_ZeroTTLDisablesCaching(t *testing.T) {
	src := &fakeLimitsSource{snap: &cosmoflare.LimitsSnapshot{}}
	c := NewLimitsSnapshotCache(0)
	c.SetSource(src, "acct-1")

	for i := 0; i < 2; i++ {
		if _, err := c.Snapshot(context.Background(), ""); err != nil {
			t.Fatal(err)
		}
	}
	if src.calls != 2 {
		t.Errorf("source fetched %d time(s) with ttl=0, want 2", src.calls)
	}
}

func TestLimitsSnapshotCache_ErrorsNotCached(t *testing.T) {
	src := &fakeLimitsSource{err: errors.New("boom")}
	c := NewLimitsSnapshotCache(time.Hour)
	c.SetSource(src, "acct-1")

	if _, err := c.Snapshot(context.Background(), ""); err == nil {
		t.Fatal("first call should fail")
	}
	src.err = nil
	src.snap = &cosmoflare.LimitsSnapshot{}
	if _, err := c.Snapshot(context.Background(), ""); err != nil {
		t.Fatalf("second call should succeed after error cleared: %v", err)
	}
	if src.calls != 2 {
		t.Errorf("source fetched %d time(s), want 2 (error must not be cached)", src.calls)
	}
}

func TestLimitsSnapshotCache_NoSourceErrors(t *testing.T) {
	c := NewLimitsSnapshotCache(time.Hour)
	if _, err := c.Snapshot(context.Background(), ""); err == nil {
		t.Fatal("Snapshot with no source set must error")
	}
}

