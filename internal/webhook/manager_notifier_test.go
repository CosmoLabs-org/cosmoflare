package webhook

import (
	"testing"
)

// FEAT-008: TriggerAlert fans out to an optional in-process notifier so the
// serve daemon can bridge alerts to its SSE hub (notifications channel).
// The notifier must receive the payload built for webhook delivery, fire
// once per trigger, and be safe to leave unset.

func TestTriggerAlertInvokesNotifier(t *testing.T) {
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
	if p.Event != "alert_triggered" {
		t.Errorf("payload event = %q, want alert_triggered", p.Event)
	}
	if p.Alert != alert {
		t.Error("payload alert mismatch: notifier must receive the triggered alert")
	}
	if p.Value != 7.5 {
		t.Errorf("payload value = %v, want 7.5", p.Value)
	}
	if p.Threshold != 5 {
		t.Errorf("payload threshold = %v, want 5", p.Threshold)
	}
	if p.Message != "error rate above threshold" {
		t.Errorf("payload message = %q", p.Message)
	}
	if p.Source != "cosmoflare" {
		t.Errorf("payload source = %q, want cosmoflare", p.Source)
	}
	if p.Timestamp.IsZero() {
		t.Error("payload timestamp is zero")
	}
}

func TestTriggerAlertWithoutNotifierDoesNotPanic(t *testing.T) {
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
