/*
Package webhook tests — ID uniqueness regression guards.

Copyright (c) 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package webhook

import (
	"fmt"
	"sync"
	"testing"
)

// TestGenerateID_Uniqueness guards against the root cause behind the flaky
// TestTriggerAlert_WithStoreMultipleWebhooks: generateID() previously used
// time.Now().UnixNano() alone, which collided ~70% of the time in tight
// loops. Collisions silently overwrote ID-keyed map entries (lost webhooks).
func TestGenerateID_Uniqueness(t *testing.T) {
	const n = 5000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := generateID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id generated after %d calls: %q", i, id)
		}
		seen[id] = struct{}{}
	}
}

// TestGenerateID_ConcurrentUniqueness ensures uniqueness holds under
// concurrent creation — the webhook Manager must be safe for parallel use.
func TestGenerateID_ConcurrentUniqueness(t *testing.T) {
	const goroutines = 16
	const perG = 1000
	seen := make(map[string]struct{}, goroutines*perG)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				id := generateID()
				mu.Lock()
				if _, dup := seen[id]; dup {
					t.Errorf("duplicate id under concurrency: %q", id)
				}
				seen[id] = struct{}{}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
}

// TestCreateWebhook_NoSilentOverwrite verifies the user-facing impact:
// rapidly created webhooks must each occupy a distinct map slot.
func TestCreateWebhook_NoSilentOverwrite(t *testing.T) {
	m := NewManager(nil, "acct")
	const n = 1000
	for i := 0; i < n; i++ {
		if _, err := m.CreateWebhook(&Webhook{Name: fmt.Sprintf("w-%d", i), URL: "http://x"}); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	if got := len(m.webhooks); got != n {
		t.Fatalf("expected %d stored webhooks, got %d (ID collisions silently overwrote entries)", n, got)
	}
}
