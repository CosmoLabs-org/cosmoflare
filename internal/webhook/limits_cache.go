package webhook

import (
	"context"
	"fmt"
	"sync"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// LimitsSource is the snapshot seam the evaluator consumes — satisfied by
// *cosmoflare.LimitsService (live) and by LimitsSnapshotCache (FEAT-014).
type LimitsSource interface {
	Snapshot(ctx context.Context, bucket string) (*cosmoflare.LimitsSnapshot, error)
}

// LimitsSnapshotCache serves a limits snapshot at most once per TTL window
// (FEAT-014). Limits move on an hours timescale; the serve alert cycle used
// to collect a full snapshot (~45 API calls on a 41-zone account) on every
// evaluation tick. The cache is keyed by account: serve runs one account per
// daemon, but the current profile can change across ticks — a different
// account invalidates immediately.
type LimitsSnapshotCache struct {
	ttl     time.Duration
	mu      sync.Mutex
	source  LimitsSource
	account string
	snap    *cosmoflare.LimitsSnapshot
	fetched time.Time
}

// NewLimitsSnapshotCache creates an empty cache with the given TTL; the
// caller must SetSource before the first Snapshot (the daemon resolves
// credentials per tick). ttl <= 0 disables caching (every call fetches).
func NewLimitsSnapshotCache(ttl time.Duration) *LimitsSnapshotCache {
	return &LimitsSnapshotCache{ttl: ttl}
}

// SetSource swaps the underlying source (per-tick credential resolution) and
// drops the cached snapshot when the account changed.
func (c *LimitsSnapshotCache) SetSource(source LimitsSource, account string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.source = source
	if account != c.account {
		c.account = account
		c.snap = nil
	}
}

// Snapshot returns the cached snapshot when fresh, otherwise fetches a new
// one. Errors are never cached — the next tick retries.
func (c *LimitsSnapshotCache) Snapshot(ctx context.Context, bucket string) (*cosmoflare.LimitsSnapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.source == nil {
		return nil, fmt.Errorf("limits cache: no source set")
	}
	if c.ttl <= 0 {
		return c.source.Snapshot(ctx, bucket)
	}
	now := time.Now()
	if c.snap != nil && now.Sub(c.fetched) < c.ttl {
		return c.snap, nil
	}
	snap, err := c.source.Snapshot(ctx, bucket)
	if err != nil {
		return nil, err
	}
	c.snap, c.fetched = snap, now
	return snap, nil
}
