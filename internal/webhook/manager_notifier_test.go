package webhook

import (
	"fmt"
	"testing"
)

// FEAT-008: TriggerAlert fans out to an optional in-process notifier so the
// serve daemon can bridge alerts to its SSE hub (notifications channel).
// The notifier must receive the payload built for webhook delivery, fire
// once per trigger, and be safe to leave unset.

// TestTriggerAlertInvokesNotifier verifies that the notifier receives exactly
// one payload per trigger and that the payload is the one built for webhook
// delivery: every projected field (event, value, threshold, message, source,
// timestamp) is asserted as its own subtest so a regression names the exact
// field that was dropped or corrupted.
func TestTriggerAlertInvokesNotifier(t *testing.T) {
	t.Parallel()
	m := NewManager(nil, "acct-123")

	var got []*NotificationPayload
	m.SetNotifier(func(p *NotificationPayload) {
		got = append(got, p)
	})

	alert := &Alert{ID: "a-1", Name: "high-error-rate", Type: AlertTypeThreshold, Threshold: 5, Enabled: true}
	if err := m.TriggerAlert(alert, 7.5, "error rate above threshold", nil); err != nil {
		t.Fatalf("TriggerAlert: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("notifier calls = %d, want 1", len(got))
	}
	p := got[0]
	// Pointer identity matters: the SSE bridge must observe the very alert
	// object that fired, not a copy that can diverge from later bookkeeping.
	if p.Alert != alert {
		t.Fatal("payload alert mismatch: notifier must receive the triggered alert")
	}

	checks := map[string]func() string{
		"event is alert_triggered": func() string {
			if p.Event != "alert_triggered" {
				return fmt.Sprintf("got %q", p.Event)
			}
			return ""
		},
		"value is the observed metric": func() string {
			if p.Value != 7.5 {
				return fmt.Sprintf("got %v", p.Value)
			}
			return ""
		},
		"threshold comes from the alert": func() string {
			if p.Threshold != 5 {
				return fmt.Sprintf("got %v", p.Threshold)
			}
			return ""
		},
		"message is passed through": func() string {
			if p.Message != "error rate above threshold" {
				return fmt.Sprintf("got %q", p.Message)
			}
			return ""
		},
		"source is cosmoflare": func() string {
			if p.Source != "cosmoflare" {
				return fmt.Sprintf("got %q", p.Source)
			}
			return ""
		},
		"timestamp is stamped": func() string {
			if p.Timestamp.IsZero() {
				return "timestamp is zero"
			}
			return ""
		},
	}
	runFieldChecks(t, checks)
}

// TestTriggerAlertWithoutNotifierDoesNotPanic verifies the notifier is
// optional: with none registered, triggering still succeeds and updates the
// alert's own bookkeeping instead of nil-panicking.
func TestTriggerAlertWithoutNotifierDoesNotPanic(t *testing.T) {
	t.Parallel()
	m := NewManager(nil, "acct-123")

	alert := &Alert{ID: "a-2", Name: "budget", Type: AlertTypeBudget, Threshold: 100, Enabled: true}
	if err := m.TriggerAlert(alert, 120, "over budget", nil); err != nil {
		t.Fatalf("TriggerAlert without notifier: %v", err)
	}
	if alert.Count != 1 {
		t.Errorf("alert.Count = %d, want 1 (alert state must still update)", alert.Count)
	}
	if alert.LastTrigger.IsZero() {
		t.Error("alert.LastTrigger is zero (alert state must still update)")
	}
}
