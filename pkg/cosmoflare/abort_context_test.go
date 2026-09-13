package cosmoflare

import (
	"context"
	"testing"
	"time"
)

// BUG-043: aborting an incomplete multipart upload with the already-canceled
// request context makes the abort itself fail instantly — the upload leaks
// and R2 keeps billing the stored parts. The abort context must survive
// cancellation of the parent request context, while still being bounded.
func TestAbortContext_SurvivesCanceledParent(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel() // simulate the user interrupt / timeout that aborted the upload

	abortCtx, abortCancel := abortContext(parent)
	defer abortCancel()

	if err := abortCtx.Err(); err != nil {
		t.Fatalf("abort context inherited parent cancellation (%v) — the AbortMultipartUpload call would fail instantly and leak billed parts (BUG-043)", err)
	}
}

func TestAbortContext_IsBounded(t *testing.T) {
	abortCtx, abortCancel := abortContext(context.Background())
	defer abortCancel()

	// Bounded deadline so a hung abort endpoint cannot pin a goroutine forever.
	d, hasD := abortCtx.Deadline()
	if !hasD {
		t.Fatal("abort context has no deadline — a hung abort request would never be reaped")
	}
	if remaining := time.Until(d); remaining <= 0 || remaining > 60*time.Second {
		t.Errorf("abort deadline remaining = %v, want within (0, 60s]", remaining)
	}
}
