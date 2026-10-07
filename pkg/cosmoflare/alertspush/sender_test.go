package alertspush

import (
	"context"
	"strings"
	"testing"
	"time"
)

// fakeSender is an in-memory Sender that answers each endpoint with a canned
// HTTP status. It never touches the network.
type fakeSender struct {
	statuses map[string]int // endpoint -> HTTP status (0 = transport-style failure)
	calls    []string       // endpoints in the order they were attempted
}

var _ Sender = (*fakeSender)(nil)

func (f *fakeSender) Send(ctx context.Context, sub Subscription, envelope []byte) (int, error) {
	f.calls = append(f.calls, sub.Endpoint)
	return f.statuses[sub.Endpoint], nil
}

// dispatchTestPayload is a small valid payload for Dispatch tests. Named
// uniquely to avoid clashing with payload_test.go helpers after merge.
func dispatchTestPayload() Payload {
	return Payload{
		ID:       "rule-1",
		Severity: SeverityWarning,
		Service:  "cloudflare",
		Title:    "Origin 5xx spike",
		Detail:   "5xx rate above threshold",
		FiredAt:  time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
	}
}

func testStore(t *testing.T) *Store {
	t.Helper()
	return &Store{Subscriptions: []Subscription{
		{Endpoint: "https://push.example/e1", P256dh: "k1", Auth: "a1"},
		{Endpoint: "https://push.example/e2", P256dh: "k2", Auth: "a2"},
	}}
}

func TestDispatch_PrunesExpired404(t *testing.T) {
	st := testStore(t)
	fs := &fakeSender{statuses: map[string]int{
		"https://push.example/e1": 404,
		"https://push.example/e2": 200,
	}}
	sent, pruned, err := Dispatch(context.Background(), st, fs, dispatchTestPayload())
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if sent != 1 || pruned != 1 {
		t.Fatalf("sent=%d pruned=%d, want 1/1", sent, pruned)
	}
	if len(st.Subscriptions) != 1 || st.Subscriptions[0].Endpoint != "https://push.example/e2" {
		t.Fatalf("expired subscription not pruned, remaining: %+v", st.Subscriptions)
	}
	if len(fs.calls) != 2 {
		t.Fatalf("calls = %v, want every subscription attempted", fs.calls)
	}
}

func TestDispatch_PrunesGone410(t *testing.T) {
	st := &Store{Subscriptions: []Subscription{
		{Endpoint: "https://push.example/gone", P256dh: "k", Auth: "a"},
	}}
	fs := &fakeSender{statuses: map[string]int{
		"https://push.example/gone": 410,
	}}
	sent, pruned, err := Dispatch(context.Background(), st, fs, dispatchTestPayload())
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if sent != 0 || pruned != 1 {
		t.Fatalf("sent=%d pruned=%d, want 0/1", sent, pruned)
	}
	if len(st.Subscriptions) != 0 {
		t.Fatalf("410 subscription not pruned, remaining: %+v", st.Subscriptions)
	}
}

func TestDispatch_AllSuccess(t *testing.T) {
	st := testStore(t)
	fs := &fakeSender{statuses: map[string]int{
		"https://push.example/e1": 200,
		"https://push.example/e2": 201,
	}}
	sent, pruned, err := Dispatch(context.Background(), st, fs, dispatchTestPayload())
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if sent != 2 || pruned != 0 {
		t.Fatalf("sent=%d pruned=%d, want 2/0", sent, pruned)
	}
	if len(st.Subscriptions) != 2 {
		t.Fatalf("healthy subscriptions were pruned: %+v", st.Subscriptions)
	}
}

func TestDispatch_OneFailureDoesNotAbortRest(t *testing.T) {
	st := testStore(t)
	fs := &fakeSender{statuses: map[string]int{
		"https://push.example/e1": 500,
		"https://push.example/e2": 200,
	}}
	sent, pruned, err := Dispatch(context.Background(), st, fs, dispatchTestPayload())
	if err == nil {
		t.Fatal("want error for failing subscription, got nil")
	}
	if sent != 1 || pruned != 0 {
		t.Fatalf("sent=%d pruned=%d, want 1/0", sent, pruned)
	}
	if len(fs.calls) != 2 {
		t.Fatalf("calls = %v, want a failure not to abort the remaining subscriptions", fs.calls)
	}
	if !strings.Contains(err.Error(), "/e1") {
		t.Fatalf("error should name the failing endpoint, got: %v", err)
	}
}

func TestDispatch_FirstNonPruneErrorWins(t *testing.T) {
	st := testStore(t)
	fs := &fakeSender{statuses: map[string]int{
		"https://push.example/e1": 500,
		"https://push.example/e2": 503,
	}}
	_, _, err := Dispatch(context.Background(), st, fs, dispatchTestPayload())
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "/e1") || strings.Contains(err.Error(), "/e2") {
		t.Fatalf("want the FIRST non-prune error (e1), got: %v", err)
	}
}

func TestDispatch_InvalidPayloadSendsNothing(t *testing.T) {
	st := testStore(t)
	fs := &fakeSender{statuses: map[string]int{}}
	p := dispatchTestPayload()
	p.ID = "" // invalid: Validate fails
	sent, pruned, err := Dispatch(context.Background(), st, fs, p)
	if err == nil {
		t.Fatal("want marshal/validation error, got nil")
	}
	if sent != 0 || pruned != 0 {
		t.Fatalf("sent=%d pruned=%d, want 0/0", sent, pruned)
	}
	if len(fs.calls) != 0 {
		t.Fatalf("calls = %v, want no sends for an unmarshalable payload", fs.calls)
	}
}

func TestDispatch_OversizedEnvelopeSendsNothing(t *testing.T) {
	st := testStore(t)
	fs := &fakeSender{statuses: map[string]int{}}
	p := dispatchTestPayload()
	p.Detail = strings.Repeat("x", 2100) // pushes the envelope past 2048 bytes
	_, _, err := Dispatch(context.Background(), st, fs, p)
	if err == nil || !strings.Contains(err.Error(), "2048") {
		t.Fatalf("want 2048-byte cap error, got: %v", err)
	}
	if len(fs.calls) != 0 {
		t.Fatalf("calls = %v, want no sends for an oversized envelope", fs.calls)
	}
}
