/*
Package webhook tests

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// newWebhookTestManager builds a Manager that is private to a single test.
// Every test constructs its own manager (and its own httptest server) so no
// case depends on another one's state and the suite is parallel-safe.
func newWebhookTestManager() *Manager {
	return NewManager(nil, "acct-123")
}

// newTestEvent returns a minimal, self-consistent event for the given type.
func newTestEvent(eventType string) *Event {
	return &Event{Type: eventType, Timestamp: time.Now().UTC(), Source: "cosmoflare"}
}

// newTestPayload returns a minimal notification payload used by delivery tests.
func newTestPayload(event, message string) *NotificationPayload {
	return &NotificationPayload{
		Event:     event,
		Timestamp: time.Now().UTC(),
		Source:    "cosmoflare",
		Message:   message,
	}
}

// capturedRequest records everything a test server observed. It is safe for
// concurrent handler goroutines, which matters for fan-out tests.
type capturedRequest struct {
	mu     sync.Mutex
	hits   int
	method string
	header http.Header
	body   []byte
}

// record stores one incoming request. The body is drained and closed so the
// connection can be reused for the next attempt.
func (c *capturedRequest) record(r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hits++
	c.method = r.Method
	c.header = r.Header
	c.body = body
}

// header returns the recorded value of a request header ("" when absent).
func (c *capturedRequest) headerValue(key string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.header == nil {
		return ""
	}
	return c.header.Get(key)
}

// snapshot returns the number of hits plus the last request seen.
func (c *capturedRequest) snapshot() (hits int, method string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits, c.method, c.body
}

// newCaptureServer starts an httptest server that always replies with status
// and records every request. The server is closed via t.Cleanup.
func newCaptureServer(t *testing.T, status int) (*httptest.Server, *capturedRequest) {
	t.Helper()
	capt := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capt.record(r)
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, capt
}

// computeHMAC returns the hex HMAC-SHA256 of payload using secret — the same
// digest scheme signPayload/ParseWebhookSignature use.
func computeHMAC(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// ---------------------------------------------------------------------------
// NewManager
// ---------------------------------------------------------------------------

// TestNewManager verifies that a freshly constructed Manager is immediately
// usable: it must ship a default HTTP client and initialized (empty) secret
// and webhook stores rather than nil maps that would panic on first write.
func TestNewManager(t *testing.T) {
	t.Parallel()
	m := NewManager(nil, "acct-123")
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.httpClient == nil {
		t.Error("expected default httpClient to be set")
	}
	if m.secrets == nil {
		t.Error("expected secrets map to be initialized")
	}
	if len(m.secrets) != 0 {
		t.Errorf("expected empty secrets map, got %d entries", len(m.secrets))
	}
}

// ---------------------------------------------------------------------------
// CreateWebhook - validation
// ---------------------------------------------------------------------------

// TestCreateWebhook_Validation covers every CreateWebhook rejection path in a
// single table: each case feeds an invalid webhook and asserts creation fails
// with an error that names the offending field (or URL problem).
func TestCreateWebhook_Validation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		webhook  *Webhook
		wantText string // substring the error must contain
	}{
		{
			name:     "empty name",
			webhook:  &Webhook{Name: "", URL: "https://example.com/hook"},
			wantText: "name",
		},
		{
			name:     "empty URL",
			webhook:  &Webhook{Name: "my-hook", URL: ""},
			wantText: "URL",
		},
		{
			name:     "unparseable URL",
			webhook:  &Webhook{Name: "bad-url", URL: "://not-a-url"},
			wantText: "invalid webhook URL",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := newWebhookTestManager()
			_, err := m.CreateWebhook(tc.webhook)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantText) {
				t.Errorf("error should mention %q, got: %v", tc.wantText, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CreateWebhook - defaults
// ---------------------------------------------------------------------------

// TestCreateWebhook_SetsDefaults verifies that CreateWebhook fills in every
// field a caller may omit: server-side ID, audit timestamps, a 30s timeout,
// 3 retries and a catch-all ["all"] event subscription.
func TestCreateWebhook_SetsDefaults(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	wh := &Webhook{Name: "test-hook", URL: "https://example.com/hook"}
	result, err := m.CreateWebhook(wh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID == "" {
		t.Error("expected ID to be set")
	}
	if !strings.HasPrefix(result.ID, "wh_") {
		t.Errorf("expected ID prefix 'wh_', got: %s", result.ID)
	}
	if result.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if result.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
	if result.Timeout != 30 {
		t.Errorf("expected default timeout 30, got %d", result.Timeout)
	}
	if result.RetryCount != 3 {
		t.Errorf("expected default RetryCount 3, got %d", result.RetryCount)
	}
	if len(result.Events) != 1 || result.Events[0] != "all" {
		t.Errorf("expected default events [all], got %v", result.Events)
	}
}

// TestCreateWebhook_PreservesProvided verifies that caller-supplied timeout,
// retry count and event list are NOT overwritten by the defaults logic.
func TestCreateWebhook_PreservesProvided(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	wh := &Webhook{
		Name:       "custom-hook",
		URL:        "https://example.com/hook",
		Timeout:    60,
		RetryCount: 5,
		Events:     []string{"object.created", "object.deleted"},
	}
	result, err := m.CreateWebhook(wh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Timeout != 60 {
		t.Errorf("expected preserved timeout 60, got %d", result.Timeout)
	}
	if result.RetryCount != 5 {
		t.Errorf("expected preserved RetryCount 5, got %d", result.RetryCount)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 preserved events, got %d", len(result.Events))
	}
}

// ---------------------------------------------------------------------------
// SendWebhook - no registered webhooks
// ---------------------------------------------------------------------------

// TestSendWebhook_NoWebhooks verifies that dispatching an event to a manager
// with no stored webhooks is a silent no-op (nil error) instead of a failure.
func TestSendWebhook_NoWebhooks(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	event := newTestEvent("object.created")
	// getWebhooksForEvent returns an empty slice, so SendWebhook must return nil
	err := m.SendWebhook("object.created", event)
	if err != nil {
		t.Fatalf("expected nil error when no webhooks registered, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateAlert - validation
// ---------------------------------------------------------------------------

// TestCreateAlert_Validation covers every CreateAlert rejection path in a
// single table: name, type and metric are all mandatory and the returned error
// must name the missing field so callers can act on it.
func TestCreateAlert_Validation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		alert   *Alert
		wantErr string // substring the error must contain
	}{
		{
			name:    "empty name",
			alert:   &Alert{Name: "", Type: AlertTypeThreshold, Metric: "requests"},
			wantErr: "name",
		},
		{
			name:    "empty type",
			alert:   &Alert{Name: "my-alert", Type: "", Metric: "requests"},
			wantErr: "type",
		},
		{
			name:    "empty metric",
			alert:   &Alert{Name: "my-alert", Type: AlertTypeThreshold, Metric: ""},
			wantErr: "metric",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := newWebhookTestManager()
			_, err := m.CreateAlert(tc.alert)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error should mention %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CreateAlert - defaults
// ---------------------------------------------------------------------------

// TestCreateAlert_SetsDefaults verifies that CreateAlert stamps an ID, audit
// timestamps and a default "1h" evaluation window on a minimal alert.
func TestCreateAlert_SetsDefaults(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	alert := &Alert{Name: "cpu-alert", Type: AlertTypeThreshold, Metric: "cpu_usage"}
	result, err := m.CreateAlert(alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID == "" {
		t.Error("expected ID to be set")
	}
	if result.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if result.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
	if result.Window != "1h" {
		t.Errorf("expected default window '1h', got: %s", result.Window)
	}
}

// TestCreateAlert_PreservesWindow verifies that an explicitly configured
// window (here "24h") is not replaced by the "1h" default.
func TestCreateAlert_PreservesWindow(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	alert := &Alert{
		Name:   "cpu-alert",
		Type:   AlertTypeThreshold,
		Metric: "cpu_usage",
		Window: "24h",
	}
	result, err := m.CreateAlert(alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Window != "24h" {
		t.Errorf("expected preserved window '24h', got: %s", result.Window)
	}
}

// ---------------------------------------------------------------------------
// CreateBucketEvent
// ---------------------------------------------------------------------------

// TestCreateBucketEvent_NilData verifies that CreateBucketEvent builds a
// usable event from scratch: nil caller data becomes an initialized map, the
// bucket name lands in both the field and Data, and the event is timestamped
// and attributed to the cosmoflare source.
func TestCreateBucketEvent_NilData(t *testing.T) {
	t.Parallel()
	event := CreateBucketEvent(EventTypeBucketCreated, "my-bucket", nil)
	if event.Data == nil {
		t.Fatal("expected data map to be initialized")
	}
	if event.Bucket != "my-bucket" {
		t.Errorf("expected bucket 'my-bucket', got: %s", event.Bucket)
	}
	if event.Source != "cosmoflare" {
		t.Errorf("expected source 'cosmoflare', got: %s", event.Source)
	}
	if event.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
	if event.Data["bucket"] != "my-bucket" {
		t.Errorf("expected data[bucket]='my-bucket', got: %v", event.Data["bucket"])
	}
}

// TestCreateBucketEvent_ExistingData verifies that caller-supplied Data is
// preserved (not clobbered) while the bucket key is still injected.
func TestCreateBucketEvent_ExistingData(t *testing.T) {
	t.Parallel()
	existing := map[string]interface{}{"region": "us-east-1", "owner": "team-a"}
	event := CreateBucketEvent(EventTypeBucketCreated, "my-bucket", existing)

	// Original data should be preserved
	if event.Data["region"] != "us-east-1" {
		t.Errorf("expected existing region preserved, got: %v", event.Data["region"])
	}
	if event.Data["owner"] != "team-a" {
		t.Errorf("expected existing owner preserved, got: %v", event.Data["owner"])
	}
	// Bucket should be added
	if event.Data["bucket"] != "my-bucket" {
		t.Errorf("expected data[bucket]='my-bucket', got: %v", event.Data["bucket"])
	}
}

// ---------------------------------------------------------------------------
// CreateObjectEvent
// ---------------------------------------------------------------------------

// TestCreateObjectEvent_NilData verifies the from-scratch path of
// CreateObjectEvent: nil data is initialized, bucket/object fields and Data
// entries agree, the size is stored as int64 and the event is timestamped.
func TestCreateObjectEvent_NilData(t *testing.T) {
	t.Parallel()
	event := CreateObjectEvent(EventTypeObjectCreated, "my-bucket", "file.txt", 1024, nil)
	if event.Data == nil {
		t.Fatal("expected data map to be initialized")
	}
	if event.Bucket != "my-bucket" {
		t.Errorf("expected bucket 'my-bucket', got: %s", event.Bucket)
	}
	if event.Object != "file.txt" {
		t.Errorf("expected object 'file.txt', got: %s", event.Object)
	}
	if event.Source != "cosmoflare" {
		t.Errorf("expected source 'cosmoflare', got: %s", event.Source)
	}
	if event.Data["size"] != int64(1024) {
		t.Errorf("expected data[size]=1024, got: %v", event.Data["size"])
	}
	if event.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

// TestCreateObjectEvent_WithData verifies that caller-supplied Data survives
// object-event construction while bucket, object and size are injected.
func TestCreateObjectEvent_WithData(t *testing.T) {
	t.Parallel()
	existing := map[string]interface{}{"content_type": "text/plain"}
	event := CreateObjectEvent(EventTypeObjectUploaded, "my-bucket", "file.txt", 2048, existing)

	if event.Data["content_type"] != "text/plain" {
		t.Errorf("expected existing content_type preserved, got: %v", event.Data["content_type"])
	}
	if event.Data["bucket"] != "my-bucket" {
		t.Errorf("expected data[bucket]='my-bucket', got: %v", event.Data["bucket"])
	}
	if event.Data["object"] != "file.txt" {
		t.Errorf("expected data[object]='file.txt', got: %v", event.Data["object"])
	}
	if event.Data["size"] != int64(2048) {
		t.Errorf("expected data[size]=2048, got: %v", event.Data["size"])
	}
}

// ---------------------------------------------------------------------------
// CreateAlertEvent
// ---------------------------------------------------------------------------

// TestCreateAlertEvent verifies that CreateAlertEvent projects every alert
// attribute plus the observed value and message into the event payload —
// receivers must be able to reconstruct the alert from Data alone.
func TestCreateAlertEvent(t *testing.T) {
	t.Parallel()
	alert := &Alert{
		ID:        "alert-001",
		Name:      "CPU Alert",
		Type:      AlertTypeThreshold,
		Metric:    "cpu_usage",
		Threshold: 90.0,
	}
	event := CreateAlertEvent(alert, 95.5, "CPU exceeded threshold")

	if event.Type != EventTypeAlertTriggered {
		t.Errorf("expected type %s, got: %s", EventTypeAlertTriggered, event.Type)
	}
	if event.Source != "cosmoflare" {
		t.Errorf("expected source 'cosmoflare', got: %s", event.Source)
	}

	// Every projected field is asserted as its own subtest so a regression
	// names the exact key that was dropped or corrupted.
	wantData := map[string]interface{}{
		"alert_id":   "alert-001",
		"alert_name": "CPU Alert",
		"alert_type": AlertTypeThreshold,
		"metric":     "cpu_usage",
		"threshold":  90.0,
		"value":      95.5,
		"message":    "CPU exceeded threshold",
	}
	for key, want := range wantData {
		t.Run("data["+key+"]", func(t *testing.T) {
			t.Parallel()
			if got := event.Data[key]; got != want {
				t.Errorf("expected data[%s]=%v, got: %v", key, want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ParseWebhookSignature
// ---------------------------------------------------------------------------

// TestParseWebhookSignature is the parameterized surface for signature
// verification. It covers the happy path plus every tampering/replay edge
// case that a receiver must reject, each as a named subtest.
func TestParseWebhookSignature(t *testing.T) {
	t.Parallel()
	standardPayload := []byte(`{"event":"test","timestamp":"2025-01-01T00:00:00Z"}`)

	cases := []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		wantOK    bool   // expected verification result when err == nil
		wantErr   string // non-empty => an error containing this text is required
	}{
		{
			name:      "correct signature verifies",
			payload:   standardPayload,
			signature: "sha256=" + computeHMAC(standardPayload, "my-secret-key"),
			secret:    "my-secret-key",
			wantOK:    true,
		},
		{
			name:      "signature without sha256 prefix is rejected",
			payload:   []byte("data"),
			signature: "invalid-no-prefix",
			secret:    "secret",
			wantErr:   "invalid signature format",
		},
		{
			name:      "wrong algorithm prefix is rejected",
			payload:   []byte("data"),
			signature: "md5=abc123",
			secret:    "secret",
			wantErr:   "invalid signature format",
		},
		{
			name:      "empty signature is rejected",
			payload:   []byte("data"),
			signature: "",
			secret:    "secret",
			wantOK:    false,
			wantErr:   "invalid signature format",
		},
		{
			name:      "wrong secret fails verification",
			payload:   []byte(`{"event":"test"}`),
			signature: "sha256=" + computeHMAC([]byte(`{"event":"test"}`), "correct-secret"),
			secret:    "wrong-secret",
			wantOK:    false,
		},
		{
			name:      "tampered payload fails verification",
			payload:   []byte(`{"event":"test","value":999}`),
			signature: "sha256=" + computeHMAC([]byte(`{"event":"test","value":100}`), "my-secret"),
			secret:    "my-secret",
			wantOK:    false,
		},
		{
			name:      "replayed signature on modified payload fails",
			payload:   []byte(`{"event":"payment","amount":10000}`),
			signature: "sha256=" + computeHMAC([]byte(`{"event":"payment","amount":100}`), "webhook-secret"),
			secret:    "webhook-secret",
			wantOK:    false,
		},
		{
			name:      "valid signature with garbage suffix fails",
			payload:   []byte("data"),
			signature: "sha256=" + computeHMAC([]byte("data"), "secret") + "abcdef",
			secret:    "secret",
			wantOK:    false,
		},
		{
			name:      "empty payload verifies against its own signature",
			payload:   []byte{},
			signature: "sha256=" + computeHMAC([]byte{}, "secret"),
			secret:    "secret",
			wantOK:    true,
		},
		{
			name:      "empty secret still produces a verifiable signature",
			payload:   []byte("data"),
			signature: "sha256=" + computeHMAC([]byte("data"), ""),
			secret:    "",
			wantOK:    true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ok, err := ParseWebhookSignature(tc.payload, tc.signature, tc.secret)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("unexpected error message: %v", err)
				}
				if ok {
					t.Error("expected ok=false alongside the error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tc.wantOK {
				t.Errorf("verification = %v, want %v", ok, tc.wantOK)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestWebhook - HTTP integration
// ---------------------------------------------------------------------------

// TestWebhook_SendsPostWithCorrectHeadersAndBody verifies the wire format of
// the TestWebhook probe: POST, JSON content type, the R2Go2 user agent and
// event header, and a JSON body carrying event, source and message.
func TestWebhook_SendsPostWithCorrectHeadersAndBody(t *testing.T) {
	t.Parallel()
	server, capt := newCaptureServer(t, http.StatusOK)

	m := newWebhookTestManager()
	wh := &Webhook{
		Name:   "test-hook",
		URL:    server.URL,
		Events: []string{"webhook_test"},
	}

	err := m.TestWebhook(wh)
	if err != nil {
		t.Fatalf("TestWebhook failed: %v", err)
	}

	_, method, body := capt.snapshot()

	headers := map[string]string{
		"Content-Type":  "application/json",
		"User-Agent":    "R2Go2-Webhook/1.0",
		"X-R2Go2-Event": "webhook_test",
	}
	for header, want := range headers {
		t.Run("header "+header, func(t *testing.T) {
			t.Parallel()
			if got := capt.headerValue(header); got != want {
				t.Errorf("expected %s %q, got: %q", header, want, got)
			}
		})
	}

	if method != "POST" {
		t.Errorf("expected POST, got: %s", method)
	}

	// Verify body is valid JSON with expected fields
	var payload NotificationPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if payload.Event != "webhook_test" {
		t.Errorf("expected payload event 'webhook_test', got: %s", payload.Event)
	}
	if payload.Source != "cosmoflare" {
		t.Errorf("expected payload source 'cosmoflare', got: %s", payload.Source)
	}
	if payload.Message == "" {
		t.Error("expected payload message to be set")
	}
}

// TestWebhook_SecretAddsSignature verifies that configuring a secret makes
// the delivery carry an X-R2Go2-Signature header with the sha256= prefix.
func TestWebhook_SecretAddsSignature(t *testing.T) {
	t.Parallel()
	server, capt := newCaptureServer(t, http.StatusOK)

	m := newWebhookTestManager()
	wh := &Webhook{
		Name:   "signed-hook",
		URL:    server.URL,
		Secret: "test-webhook-secret",
	}

	err := m.TestWebhook(wh)
	if err != nil {
		t.Fatalf("TestWebhook failed: %v", err)
	}

	receivedSignature := capt.headerValue("X-R2Go2-Signature")
	if receivedSignature == "" {
		t.Fatal("expected X-R2Go2-Signature header to be set")
	}
	if !strings.HasPrefix(receivedSignature, "sha256=") {
		t.Errorf("expected sha256= prefix, got: %s", receivedSignature)
	}
}

// TestWebhook_SecretSignatureVerifiable closes the loop: the signature that
// arrives alongside a signed body must verify with ParseWebhookSignature, so
// receivers implementing the same scheme accept our deliveries.
func TestWebhook_SecretSignatureVerifiable(t *testing.T) {
	t.Parallel()
	server, capt := newCaptureServer(t, http.StatusOK)

	secret := "verifiable-secret"
	m := newWebhookTestManager()
	wh := &Webhook{
		Name:   "verify-hook",
		URL:    server.URL,
		Secret: secret,
	}

	err := m.TestWebhook(wh)
	if err != nil {
		t.Fatalf("TestWebhook failed: %v", err)
	}

	// Verify the signature matches the body using ParseWebhookSignature
	_, _, body := capt.snapshot()
	ok, err := ParseWebhookSignature(body, capt.headerValue("X-R2Go2-Signature"), secret)
	if err != nil {
		t.Fatalf("signature parse error: %v", err)
	}
	if !ok {
		t.Error("expected signature to be valid for the received body")
	}
}

// ---------------------------------------------------------------------------
// TriggerAlert
// ---------------------------------------------------------------------------

// TestTriggerAlert_IncrementsCount verifies the state side of TriggerAlert:
// the alert's fire counter is incremented and LastTrigger is stamped, even
// though no webhook is configured.
func TestTriggerAlert_IncrementsCount(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	alert := &Alert{
		ID:        "alert-001",
		Name:      "CPU Alert",
		Type:      AlertTypeThreshold,
		Metric:    "cpu_usage",
		Threshold: 90.0,
		Count:     0,
	}

	before := time.Now()
	err := m.TriggerAlert(alert, 95.5, "CPU is high", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alert.Count != 1 {
		t.Errorf("expected Count=1, got: %d", alert.Count)
	}
	if alert.LastTrigger.Before(before) {
		t.Error("expected LastTrigger to be updated")
	}
}

// TestTriggerAlert_MultipleTriggers verifies that repeated triggers keep
// accumulating the fire counter instead of resetting it.
func TestTriggerAlert_MultipleTriggers(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	alert := &Alert{
		ID:        "alert-002",
		Name:      "Memory Alert",
		Type:      AlertTypeThreshold,
		Metric:    "memory",
		Threshold: 80.0,
	}

	for i := 0; i < 5; i++ {
		err := m.TriggerAlert(alert, float64(80+i), "high memory", nil)
		if err != nil {
			t.Fatalf("trigger %d failed: %v", i+1, err)
		}
	}

	if alert.Count != 5 {
		t.Errorf("expected Count=5, got: %d", alert.Count)
	}
}

// TestTriggerAlert_MissingWebhookIDs_Graceful verifies that referencing
// webhook IDs that do not exist neither fails nor panics — the alert state
// must still be updated after the skips.
func TestTriggerAlert_MissingWebhookIDs_Graceful(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	alert := &Alert{
		ID:        "alert-003",
		Name:      "Disk Alert",
		Type:      AlertTypeHealth,
		Metric:    "disk_usage",
		Threshold: 95.0,
		Webhooks:  []string{"wh_nonexistent_1", "wh_nonexistent_2"},
	}

	// getWebhook always returns an error, so this must not panic
	err := m.TriggerAlert(alert, 97.0, "disk full", map[string]interface{}{"path": "/var"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Alert mutation should still happen despite missing webhooks
	if alert.Count != 1 {
		t.Errorf("expected Count=1, got: %d", alert.Count)
	}
	if alert.LastTrigger.IsZero() {
		t.Error("expected LastTrigger to be set")
	}
}

// ---------------------------------------------------------------------------
// Event type constants
// ---------------------------------------------------------------------------

// TestEventTypeConstants pins the wire values of every exported event type.
// Receivers switch on these strings, so a silent rename would be a breaking
// protocol change — each constant is asserted as its own subtest.
func TestEventTypeConstants(t *testing.T) {
	t.Parallel()
	constants := []struct {
		name string
		got  string
		want string
	}{
		{"BucketCreated", EventTypeBucketCreated, "bucket.created"},
		{"BucketDeleted", EventTypeBucketDeleted, "bucket.deleted"},
		{"ObjectCreated", EventTypeObjectCreated, "object.created"},
		{"ObjectDeleted", EventTypeObjectDeleted, "object.deleted"},
		{"ObjectUploaded", EventTypeObjectUploaded, "object.uploaded"},
		{"ObjectDownloaded", EventTypeObjectDownloaded, "object.downloaded"},
		{"MigrationStart", EventTypeMigrationStart, "migration.started"},
		{"MigrationComplete", EventTypeMigrationComplete, "migration.completed"},
		{"MigrationFailed", EventTypeMigrationFailed, "migration.failed"},
		{"AlertTriggered", EventTypeAlertTriggered, "alert.triggered"},
		{"HealthCheckFailed", EventTypeHealthCheckFailed, "health_check.failed"},
		{"DomainAttached", EventTypeDomainAttached, "domain.attached"},
		{"DomainDetached", EventTypeDomainDetached, "domain.detached"},
	}
	if len(constants) != 13 {
		t.Fatalf("expected 13 event type constants, got %d", len(constants))
	}
	for _, c := range constants {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.got != c.want {
				t.Errorf("%s = %q, want %q (wire value changed — this is a breaking change for receivers)", c.name, c.got, c.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AlertType constants
// ---------------------------------------------------------------------------

// TestAlertTypeConstants pins the string values of every AlertType so the
// serialized alert type vocabulary stays stable across releases.
func TestAlertTypeConstants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		got  AlertType
		want string
	}{
		{AlertTypeThreshold, "threshold"},
		{AlertTypeTrend, "trend"},
		{AlertTypeBudget, "budget"},
		{AlertTypeHealth, "health"},
		{AlertTypeCustom, "custom"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			if string(tc.got) != tc.want {
				t.Errorf("expected alert type %s, got %s", tc.want, tc.got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// sendToWebhook - direct delivery
// ---------------------------------------------------------------------------

// TestSendToWebhook is the parameterized surface for the single-webhook
// delivery path. It documents the contract that sendToWebhook delivers
// regardless of the Enabled flag (filtering is SendWebhook's job) and that
// transport/HTTP failures surface as errors.
func TestSendToWebhook(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		url      string // non-empty overrides the test server URL
		status   int    // HTTP status the test server replies with
		enabled  bool
		wantErr  bool
		wantSent bool
	}{
		{
			name:     "enabled webhook is delivered",
			status:   http.StatusOK,
			enabled:  true,
			wantSent: true,
		},
		{
			// sendToWebhook does not consult Enabled — SendWebhook does.
			name:     "disabled webhook is still delivered",
			status:   http.StatusOK,
			enabled:  false,
			wantSent: true,
		},
		{
			name:     "plain http URL is accepted",
			status:   http.StatusOK,
			enabled:  true,
			wantSent: true,
		},
		{
			name:     "server error is reported",
			status:   http.StatusInternalServerError,
			enabled:  true,
			wantSent: true,
			wantErr:  true,
		},
		{
			name:     "malformed URL is reported",
			url:      "http://[::1]:namedport", // unparseable URL
			enabled:  true,
			wantSent: false,
			wantErr:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var serverURL string
			var capt *capturedRequest
			if tc.url == "" {
				server, c := newCaptureServer(t, tc.status)
				serverURL, capt = server.URL, c
			} else {
				serverURL = tc.url
			}

			m := newWebhookTestManager()
			wh := &Webhook{
				ID:         "wh_case",
				Name:       tc.name,
				URL:        serverURL,
				Enabled:    tc.enabled,
				RetryCount: 0,
				Timeout:    5,
			}

			err := m.sendToWebhook(wh, newTestEvent("object.created"))
			if (err != nil) != tc.wantErr {
				t.Fatalf("sendToWebhook error = %v, wantErr %v", err, tc.wantErr)
			}
			if capt == nil {
				return // no server: nothing else to observe
			}
			hits, _, _ := capt.snapshot()
			if sent := hits > 0; sent != tc.wantSent {
				t.Errorf("delivered = %v (hits=%d), want %v", sent, hits, tc.wantSent)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TriggerAlert - notification via sendNotification directly
// ---------------------------------------------------------------------------

// TestTriggerAlert_WithNotificationDelivery verifies the full wire contract
// of an alert notification: event header, HMAC signature that verifies
// against the delivered body, and round-tripped payload fields.
func TestTriggerAlert_WithNotificationDelivery(t *testing.T) {
	t.Parallel()
	server, capt := newCaptureServer(t, http.StatusOK)

	m := newWebhookTestManager()
	alert := &Alert{
		ID:        "alert-100",
		Name:      "CPU Alert",
		Type:      AlertTypeThreshold,
		Metric:    "cpu",
		Threshold: 90.0,
		Webhooks:  []string{}, // no webhook IDs - test just the mutation path
	}

	// Manually send the notification that TriggerAlert would send
	payload := &NotificationPayload{
		Event:     "alert_triggered",
		Alert:     alert,
		Timestamp: time.Now().UTC(),
		Source:    "cosmoflare",
		Value:     95.0,
		Threshold: 90.0,
		Message:   "CPU is high",
		Data:      map[string]interface{}{"host": "server-1"},
	}

	wh := &Webhook{
		ID:         "wh_001",
		Name:       "alert-hook",
		URL:        server.URL,
		Enabled:    true,
		Secret:     "alert-secret",
		RetryCount: 0,
		Timeout:    5,
	}

	err := m.sendNotification(wh, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := capt.headerValue("X-R2Go2-Event"); got != "alert_triggered" {
		t.Errorf("expected X-R2Go2-Event 'alert_triggered', got %s", got)
	}
	receivedSignature := capt.headerValue("X-R2Go2-Signature")
	if receivedSignature == "" {
		t.Fatal("expected signature header to be set")
	}
	if !strings.HasPrefix(receivedSignature, "sha256=") {
		t.Errorf("expected sha256= prefix, got: %s", receivedSignature)
	}

	// Verify signature is valid
	_, _, body := capt.snapshot()
	ok, err := ParseWebhookSignature(body, receivedSignature, "alert-secret")
	if err != nil {
		t.Fatalf("signature verification error: %v", err)
	}
	if !ok {
		t.Error("expected valid signature")
	}

	// Verify payload contains alert data
	var decoded NotificationPayload
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.Value != 95.0 {
		t.Errorf("expected value 95.0, got: %f", decoded.Value)
	}
	if decoded.Threshold != 90.0 {
		t.Errorf("expected threshold 90.0, got: %f", decoded.Threshold)
	}
	if decoded.Message != "CPU is high" {
		t.Errorf("expected message 'CPU is high', got: %s", decoded.Message)
	}
	if decoded.Data == nil || decoded.Data["host"] != "server-1" {
		t.Error("expected data to contain host=server-1")
	}
}

// TestTriggerAlert_DisabledWebhookSkipped verifies that an alert with no
// reachable webhooks still records its own trigger state.
func TestTriggerAlert_DisabledWebhookSkipped(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	alert := &Alert{
		ID:       "alert-dis",
		Name:     "Disk Alert",
		Type:     AlertTypeHealth,
		Metric:   "disk",
		Webhooks: []string{}, // no webhooks configured
	}

	err := m.TriggerAlert(alert, 99.0, "disk full", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Alert mutation should still happen
	if alert.Count != 1 {
		t.Errorf("expected Count=1, got: %d", alert.Count)
	}
}

// TestTriggerAlert_ServerFailure verifies that TriggerAlert itself succeeds
// (and still counts the fire) while a direct delivery to a failing endpoint
// returns an error — delivery failures must never block alert bookkeeping.
func TestTriggerAlert_ServerFailure(t *testing.T) {
	t.Parallel()
	server, capt := newCaptureServer(t, http.StatusBadGateway)

	m := newWebhookTestManager()
	alert := &Alert{
		ID:       "alert-fail",
		Name:     "Net Alert",
		Type:     AlertTypeCustom,
		Metric:   "latency",
		Webhooks: []string{}, // no webhooks configured
	}

	// Manually send failing notification
	payload := newTestPayload("alert_triggered", "high latency")
	payload.Alert = alert

	wh := &Webhook{
		ID:         "wh_fail",
		URL:        server.URL,
		Enabled:    true,
		RetryCount: 0,
		Timeout:    5,
	}

	// TriggerAlert itself still succeeds
	err := m.TriggerAlert(alert, 500.0, "high latency", nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if alert.Count != 1 {
		t.Errorf("expected Count=1 even on failure, got: %d", alert.Count)
	}

	// But direct notification to failing webhook should fail
	err = m.sendNotification(wh, payload)
	if err == nil {
		t.Error("expected error for failing webhook")
	}
	if hits, _, _ := capt.snapshot(); hits != 1 {
		t.Errorf("expected the failing endpoint to be hit once, got %d hits", hits)
	}
}

// ---------------------------------------------------------------------------
// TriggerAlert - all alert types
// ---------------------------------------------------------------------------

// TestTriggerAlert_AllAlertTypes verifies that every AlertType can be
// triggered end-to-end: unknown types must not be special-cased anywhere in
// the trigger path.
func TestTriggerAlert_AllAlertTypes(t *testing.T) {
	t.Parallel()
	alertTypes := []AlertType{
		AlertTypeThreshold,
		AlertTypeTrend,
		AlertTypeBudget,
		AlertTypeHealth,
		AlertTypeCustom,
	}

	for _, at := range alertTypes {
		t.Run(string(at), func(t *testing.T) {
			t.Parallel()
			m := newWebhookTestManager()
			alert := &Alert{
				ID:        "alert-" + string(at),
				Name:      string(at) + " alert",
				Type:      at,
				Metric:    "metric_" + string(at),
				Threshold: 50.0,
				Webhooks:  []string{"nonexistent"}, // will be gracefully skipped
			}
			err := m.TriggerAlert(alert, 75.0, "test message", nil)
			if err != nil {
				t.Fatalf("unexpected error for type %s: %v", at, err)
			}
			if alert.Count != 1 {
				t.Errorf("expected Count=1 for type %s, got: %d", at, alert.Count)
			}
			if alert.LastTrigger.IsZero() {
				t.Error("expected LastTrigger to be set")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SendWebhook - with webhooks in the store
// ---------------------------------------------------------------------------

// TestSendWebhook_WithStore is the parameterized fan-out surface for
// SendWebhook: each subtest stores one webhook, dispatches one event and
// asserts how many deliveries happened, covering enablement, event matching
// (including the "all" catch-all) and endpoint failures.
func TestSendWebhook_WithStore(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		status    int      // test server response status
		events    []string // webhook subscription (nil => default ["all"])
		enabled   bool
		sendEvent string // event type passed to SendWebhook
		wantCalls int
	}{
		{
			name:      "enabled webhook subscribed to the event is called",
			status:    http.StatusOK,
			events:    []string{"object.created"},
			enabled:   true,
			sendEvent: "object.created",
			wantCalls: 1,
		},
		{
			name:      "disabled webhook is skipped",
			status:    http.StatusOK,
			events:    []string{"object.deleted"},
			enabled:   false, // disabled
			sendEvent: "object.deleted",
			wantCalls: 0,
		},
		{
			// Default events is ["all"] which matches any event type.
			name:      "default catch-all subscription matches any event",
			status:    http.StatusOK,
			events:    nil,
			enabled:   true,
			sendEvent: "migration.started",
			wantCalls: 1,
		},
		{
			name:      "webhook subscribed to another event is skipped",
			status:    http.StatusOK,
			events:    []string{"object.created"}, // only matches object.created
			enabled:   true,
			sendEvent: "bucket.created",
			wantCalls: 0,
		},
		{
			// SendWebhook logs delivery errors but still returns nil.
			// RetryCount 0 is normalized to 3 by CreateWebhook, so a
			// failing endpoint is attempted 4 times (1 + 3 retries).
			name:      "endpoint failure still returns nil",
			status:    http.StatusInternalServerError,
			events:    []string{"test"},
			enabled:   true,
			sendEvent: "test",
			wantCalls: 4,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, capt := newCaptureServer(t, tc.status)

			m := newWebhookTestManager()
			// Create and store a webhook using the scenario's subscription.
			_, err := m.CreateWebhook(&Webhook{
				Name:       "store-hook",
				URL:        server.URL,
				Events:     tc.events,
				Enabled:    tc.enabled,
				RetryCount: 0,
				Timeout:    5,
			})
			if err != nil {
				t.Fatalf("create webhook: %v", err)
			}

			err = m.SendWebhook(tc.sendEvent, newTestEvent(tc.sendEvent))
			if err != nil {
				t.Fatalf("expected nil, got: %v", err)
			}
			hits, _, _ := capt.snapshot()
			if hits != tc.wantCalls {
				t.Errorf("deliveries = %d, want %d", hits, tc.wantCalls)
			}
			if tc.wantCalls > 0 {
				// A delivered notification must carry the event type on the wire.
				if got := capt.headerValue("X-R2Go2-Event"); got != tc.sendEvent {
					t.Errorf("expected event header %q, got %q", tc.sendEvent, got)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TriggerAlert - with webhooks in the store
// ---------------------------------------------------------------------------

// TestTriggerAlert_WithStoreWorkingWebhook verifies the integrated alert
// path against a real endpoint: the stored webhook receives a signed,
// parseable alert_triggered payload carrying value, threshold and message.
func TestTriggerAlert_WithStoreWorkingWebhook(t *testing.T) {
	t.Parallel()
	server, capt := newCaptureServer(t, http.StatusOK)

	m := newWebhookTestManager()
	wh, _ := m.CreateWebhook(&Webhook{
		Name:       "alert-webhook",
		URL:        server.URL,
		Enabled:    true,
		Secret:     "store-secret",
		RetryCount: 0,
		Timeout:    5,
	})

	alert := &Alert{
		ID:        "alert-store-1",
		Name:      "CPU Alert",
		Type:      AlertTypeThreshold,
		Metric:    "cpu",
		Threshold: 90.0,
		Webhooks:  []string{wh.ID},
	}

	err := m.TriggerAlert(alert, 95.0, "CPU is high", map[string]interface{}{"host": "server-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := capt.headerValue("X-R2Go2-Event"); got != "alert_triggered" {
		t.Errorf("expected X-R2Go2-Event 'alert_triggered', got %s", got)
	}
	receivedSignature := capt.headerValue("X-R2Go2-Signature")
	if !strings.HasPrefix(receivedSignature, "sha256=") {
		t.Errorf("expected sha256= prefix, got: %s", receivedSignature)
	}

	// Verify signature
	_, _, body := capt.snapshot()
	ok, err := ParseWebhookSignature(body, receivedSignature, "store-secret")
	if err != nil {
		t.Fatalf("signature verification error: %v", err)
	}
	if !ok {
		t.Error("expected valid signature")
	}

	var payload NotificationPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if payload.Value != 95.0 {
		t.Errorf("expected value 95.0, got: %f", payload.Value)
	}
	if payload.Threshold != 90.0 {
		t.Errorf("expected threshold 90.0, got: %f", payload.Threshold)
	}
	if payload.Message != "CPU is high" {
		t.Errorf("expected message 'CPU is high', got: %s", payload.Message)
	}
}

// TestTriggerAlert_WithStoreDisabledWebhook verifies that a stored but
// disabled webhook receives nothing while the alert still counts the fire.
func TestTriggerAlert_WithStoreDisabledWebhook(t *testing.T) {
	t.Parallel()
	server, capt := newCaptureServer(t, http.StatusOK)

	m := newWebhookTestManager()
	wh, _ := m.CreateWebhook(&Webhook{
		Name:    "disabled-alert-hook",
		URL:     server.URL,
		Enabled: false,
		Secret:  "secret",
		Timeout: 5,
	})

	alert := &Alert{
		ID:       "alert-dis-2",
		Name:     "Disk Alert",
		Type:     AlertTypeHealth,
		Metric:   "disk",
		Webhooks: []string{wh.ID},
	}

	err := m.TriggerAlert(alert, 99.0, "disk full", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hits, _, _ := capt.snapshot(); hits != 0 {
		t.Error("disabled webhook should not have been called")
	}
	if alert.Count != 1 {
		t.Errorf("expected Count=1, got: %d", alert.Count)
	}
}

// TestTriggerAlert_WithStoreNonexistentWebhook verifies that alert fan-out
// silently skips unknown webhook IDs instead of failing the trigger.
func TestTriggerAlert_WithStoreNonexistentWebhook(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	alert := &Alert{
		ID:       "alert-missing",
		Name:     "Missing Webhook Alert",
		Type:     AlertTypeBudget,
		Metric:   "cost",
		Webhooks: []string{"wh_does_not_exist"},
	}

	// Should not error, just skip missing webhook
	err := m.TriggerAlert(alert, 100.0, "over budget", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert.Count != 1 {
		t.Errorf("expected Count=1, got: %d", alert.Count)
	}
}

// TestTriggerAlert_WithStoreServerError verifies that an endpoint returning
// 5xx does not fail TriggerAlert and does not lose the fire count.
func TestTriggerAlert_WithStoreServerError(t *testing.T) {
	t.Parallel()
	server, _ := newCaptureServer(t, http.StatusBadGateway)

	m := newWebhookTestManager()
	wh, _ := m.CreateWebhook(&Webhook{
		Name:       "failing-alert-hook",
		URL:        server.URL,
		Enabled:    true,
		RetryCount: 0,
		Timeout:    5,
	})

	alert := &Alert{
		ID:       "alert-fail-2",
		Name:     "Net Alert",
		Type:     AlertTypeCustom,
		Metric:   "latency",
		Webhooks: []string{wh.ID},
	}

	// TriggerAlert prints errors but returns nil
	err := m.TriggerAlert(alert, 500.0, "high latency", nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if alert.Count != 1 {
		t.Errorf("expected Count=1 even on failure, got: %d", alert.Count)
	}
}

// TestTriggerAlert_WithStoreMultipleWebhooks verifies fan-out across a mixed
// set of targets: only the enabled, existing webhooks are called, missing IDs
// are skipped, and the alert counts exactly one fire.
func TestTriggerAlert_WithStoreMultipleWebhooks(t *testing.T) {
	t.Parallel()
	server, capt := newCaptureServer(t, http.StatusOK)

	m := newWebhookTestManager()
	newHook := func(name string, enabled bool) *Webhook {
		wh, err := m.CreateWebhook(&Webhook{
			Name:       name,
			URL:        server.URL,
			Enabled:    enabled,
			RetryCount: 0,
			Timeout:    5,
		})
		if err != nil {
			t.Fatalf("create webhook %s: %v", name, err)
		}
		return wh
	}
	wh1 := newHook("multi-hook-1", true)
	wh2 := newHook("multi-hook-2", true)
	wh3 := newHook("multi-hook-disabled", false)

	alert := &Alert{
		ID:       "alert-multi",
		Name:     "Multi Alert",
		Type:     AlertTypeThreshold,
		Metric:   "requests",
		Webhooks: []string{wh1.ID, wh2.ID, wh3.ID, "wh_nonexistent"},
	}

	err := m.TriggerAlert(alert, 1000.0, "threshold exceeded", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hits, _, _ := capt.snapshot()
	if hits != 2 { // only enabled webhooks should be called
		t.Errorf("expected 2 calls (2 enabled + 1 disabled + 1 missing), got %d", hits)
	}
	if alert.Count != 1 {
		t.Errorf("expected Count=1, got: %d", alert.Count)
	}
}

// ---------------------------------------------------------------------------
// sendNotification - retry behavior
// ---------------------------------------------------------------------------

// TestSendNotification_RetryOnFailure verifies transient failures are
// retried: an endpoint that fails twice then succeeds yields success after
// exactly three attempts.
func TestSendNotification_RetryOnFailure(t *testing.T) {
	t.Parallel()
	attempts := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		n := attempts
		mu.Unlock()
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := newWebhookTestManager()
	wh := &Webhook{
		ID:         "wh_retry",
		URL:        server.URL,
		RetryCount: 3,
		Timeout:    5,
	}

	err := m.sendNotification(wh, newTestPayload("test.event", "retry test"))
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	mu.Lock()
	got := attempts
	mu.Unlock()
	if got != 3 {
		t.Errorf("expected 3 attempts, got: %d", got)
	}
}

// TestSendNotification_ExhaustsRetries verifies that a permanently failing
// endpoint produces an error that reports the total attempt count
// (initial attempt + RetryCount).
func TestSendNotification_ExhaustsRetries(t *testing.T) {
	t.Parallel()
	server, _ := newCaptureServer(t, http.StatusInternalServerError)

	m := newWebhookTestManager()
	wh := &Webhook{
		ID:         "wh_exhaust",
		URL:        server.URL,
		RetryCount: 2,
		Timeout:    5,
	}

	err := m.sendNotification(wh, newTestPayload("test.fail", "exhaust test"))
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !strings.Contains(err.Error(), "failed to send webhook after") {
		t.Errorf("expected retry exhaustion message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "3 attempts") {
		t.Errorf("expected '3 attempts' in error (RetryCount=2 + initial), got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// sendNotification - custom headers
// ---------------------------------------------------------------------------

// TestSendNotification_CustomHeaders verifies that per-webhook header
// configuration is applied verbatim to the outgoing request.
func TestSendNotification_CustomHeaders(t *testing.T) {
	t.Parallel()
	server, capt := newCaptureServer(t, http.StatusOK)

	m := newWebhookTestManager()
	wh := &Webhook{
		ID:         "wh_headers",
		URL:        server.URL,
		RetryCount: 0,
		Timeout:    5,
		Headers: map[string]string{
			"Authorization":   "Bearer test-token",
			"X-Custom-Header": "custom-value",
		},
	}

	err := m.sendNotification(wh, newTestPayload("headers.test", "header test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for header, want := range wh.Headers {
		t.Run("header "+header, func(t *testing.T) {
			t.Parallel()
			if got := capt.headerValue(header); got != want {
				t.Errorf("expected %s %q, got: %q", header, want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// sendNotification - malicious header injection
// ---------------------------------------------------------------------------

// TestSendNotification_HeaderInjectionSanitization verifies that a header
// value containing CRLF (a header-splitting injection attempt) never reaches
// the wire: Go's net/http rejects it and the delivery fails instead.
func TestSendNotification_HeaderInjectionSanitization(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	wh := &Webhook{
		ID:         "wh_inject",
		URL:        "http://127.0.0.1:1", // any URL - request won't be sent
		RetryCount: 0,
		Timeout:    1,
		Headers: map[string]string{
			"X-Injected": "value\r\nX-Evil: malicious",
		},
	}

	err := m.sendNotification(wh, newTestPayload("inject.test", "injection test"))
	// Go's http.Client will reject the header with newline - error expected
	if err == nil {
		t.Fatal("expected error for header with newline injection")
	}
}

// ---------------------------------------------------------------------------
// sendNotification - rate limiting (429 response)
// ---------------------------------------------------------------------------

// TestSendNotification_RateLimited verifies that a 429 response is treated
// as a retryable failure and, once retries run out, surfaces as an error
// after exactly initial + retry attempts.
func TestSendNotification_RateLimited(t *testing.T) {
	t.Parallel()
	var capt capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capt.record(r)
		// A real rate limiter advertises when to retry.
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	m := newWebhookTestManager()
	wh := &Webhook{
		ID:         "wh_429",
		URL:        server.URL,
		RetryCount: 1,
		Timeout:    5,
	}

	err := m.sendNotification(wh, newTestPayload("rate.test", "rate limit test"))
	if err == nil {
		t.Fatal("expected error after rate limiting exhausts retries")
	}
	hits, _, _ := capt.snapshot()
	if hits != 2 { // initial + 1 retry
		t.Errorf("expected 2 attempts, got: %d", hits)
	}
}

// ---------------------------------------------------------------------------
// sendNotification - connection refused (network error)
// ---------------------------------------------------------------------------

// TestSendNotification_ConnectionRefused verifies that transport-level
// failures (unreachable endpoint) are retried and eventually reported with
// the retry-exhaustion error, not a raw net error.
func TestSendNotification_ConnectionRefused(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	wh := &Webhook{
		ID:         "wh_refused",
		URL:        "http://127.0.0.1:1", // port 1 should refuse
		RetryCount: 1,
		Timeout:    1,
	}

	err := m.sendNotification(wh, newTestPayload("conn.test", "connection refused test"))
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
	if !strings.Contains(err.Error(), "failed to send webhook after") {
		t.Errorf("expected retry exhaustion error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// signPayload
// ---------------------------------------------------------------------------

// TestSignPayload_Consistency verifies the cryptographic contract of
// signPayload: determinism for identical input, and distinct output whenever
// the payload or the secret changes.
func TestSignPayload_Consistency(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	data := []byte(`{"test": "data"}`)
	secret := "sign-secret"

	sig1 := m.signPayload(data, secret)

	t.Run("deterministic and matches raw HMAC", func(t *testing.T) {
		t.Parallel()
		sig2 := m.signPayload(data, secret)
		if sig1 != sig2 {
			t.Error("signatures should be deterministic")
		}
		if !strings.HasPrefix(sig1, computeHMAC(data, secret)[:10]) {
			t.Error("signature should match HMAC computation")
		}
	})
	t.Run("different data yields a different signature", func(t *testing.T) {
		t.Parallel()
		if sig3 := m.signPayload([]byte(`{"test": "different"}`), secret); sig1 == sig3 {
			t.Error("different data should produce different signature")
		}
	})
	t.Run("different secret yields a different signature", func(t *testing.T) {
		t.Parallel()
		if sig4 := m.signPayload(data, "different-secret"); sig1 == sig4 {
			t.Error("different secret should produce different signature")
		}
	})
}

// ---------------------------------------------------------------------------
// Webhook struct - JSON serialization
// ---------------------------------------------------------------------------

// TestWebhook_JSONRoundTrip verifies that every Webhook field survives a
// marshal/unmarshal cycle, so persisted configurations reload losslessly.
func TestWebhook_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	wh := &Webhook{
		ID:         "wh_json",
		Name:       "json-hook",
		URL:        "https://example.com/hook",
		Events:     []string{"object.created", "object.deleted"},
		Enabled:    true,
		Secret:     "json-secret",
		Headers:    map[string]string{"Auth": "Bearer xyz"},
		CreatedAt:  time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		RetryCount: 5,
		Timeout:    60,
	}

	data, err := json.Marshal(wh)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Webhook
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ID != wh.ID {
		t.Errorf("expected ID %s, got %s", wh.ID, decoded.ID)
	}
	if decoded.Name != wh.Name {
		t.Errorf("expected Name %s, got %s", wh.Name, decoded.Name)
	}
	if decoded.Secret != "json-secret" {
		t.Error("secret should be preserved in JSON")
	}
	if decoded.RetryCount != 5 {
		t.Errorf("expected RetryCount 5, got %d", decoded.RetryCount)
	}
	if decoded.Timeout != 60 {
		t.Errorf("expected Timeout 60, got %d", decoded.Timeout)
	}
}

// TestWebhook_SecretOmittedWhenEmpty verifies the `secret,omitempty` tag: an
// unset secret must not be serialized into stored/shared JSON documents.
func TestWebhook_SecretOmittedWhenEmpty(t *testing.T) {
	t.Parallel()
	wh := &Webhook{ID: "wh_1", Name: "no-secret", URL: "https://example.com"}
	data, err := json.Marshal(wh)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if strings.Contains(string(data), `"secret"`) {
		t.Error("secret field should be omitted when empty (omitempty)")
	}
}

// ---------------------------------------------------------------------------
// NotificationPayload - JSON serialization
// ---------------------------------------------------------------------------

// TestNotificationPayload_JSONRoundTrip verifies that a fully populated
// NotificationPayload (alert, metrics, message, signature) round-trips
// through JSON without dropping fields.
func TestNotificationPayload_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	payload := &NotificationPayload{
		Event:     "object.uploaded",
		Alert:     &Alert{ID: "a1", Name: "Test Alert", Type: AlertTypeThreshold},
		Timestamp: now,
		Source:    "cosmoflare",
		Bucket:    "my-bucket",
		Object:    "file.txt",
		Value:     1024.5,
		Threshold: 900.0,
		Message:   "Upload exceeded threshold",
		Data:      map[string]interface{}{"size": 1024},
		Signature: "sha256=abc",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded NotificationPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Event != "object.uploaded" {
		t.Errorf("expected event 'object.uploaded', got %s", decoded.Event)
	}
	if decoded.Alert == nil || decoded.Alert.ID != "a1" {
		t.Error("alert should be preserved")
	}
	if decoded.Value != 1024.5 {
		t.Errorf("expected value 1024.5, got %f", decoded.Value)
	}
	if decoded.Signature != "sha256=abc" {
		t.Errorf("expected signature 'sha256=abc', got %s", decoded.Signature)
	}
}

// ---------------------------------------------------------------------------
// sendToWebhook - bucket/object fields in payload
// ---------------------------------------------------------------------------

// TestSendToWebhook_IncludesBucketAndObject verifies that the event's bucket,
// object and data map are projected into the delivered JSON payload.
func TestSendToWebhook_IncludesBucketAndObject(t *testing.T) {
	t.Parallel()
	server, capt := newCaptureServer(t, http.StatusOK)

	m := newWebhookTestManager()
	wh := &Webhook{
		ID:         "wh_bo",
		URL:        server.URL,
		RetryCount: 0,
		Timeout:    5,
	}

	event := &Event{
		Type:      EventTypeObjectCreated,
		Timestamp: time.Now().UTC(),
		Source:    "cosmoflare",
		Bucket:    "photos-bucket",
		Object:    "vacation.jpg",
		Data:      map[string]interface{}{"size": int64(2048)},
	}

	err := m.sendToWebhook(wh, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, body := capt.snapshot()
	var payload NotificationPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if payload.Bucket != "photos-bucket" {
		t.Errorf("expected bucket 'photos-bucket', got %s", payload.Bucket)
	}
	if payload.Object != "vacation.jpg" {
		t.Errorf("expected object 'vacation.jpg', got %s", payload.Object)
	}
	if payload.Data == nil {
		t.Fatal("expected data to be present")
	}
	// JSON unmarshals numbers as float64
	sizeVal, ok := payload.Data["size"].(float64)
	if !ok || sizeVal != 2048 {
		t.Errorf("expected data[size]=2048, got: %v (type %T)", payload.Data["size"], payload.Data["size"])
	}
}

// ---------------------------------------------------------------------------
// sendNotification - error message does not leak credentials
// ---------------------------------------------------------------------------

// TestSendNotification_ErrorDoesNotLeakCredentials verifies that a delivery
// error never echoes back the webhook secret or header credentials, which
// would leak them into logs.
func TestSendNotification_ErrorDoesNotLeakCredentials(t *testing.T) {
	t.Parallel()
	server, _ := newCaptureServer(t, http.StatusUnauthorized)

	m := newWebhookTestManager()
	wh := &Webhook{
		ID:         "wh_leak",
		URL:        server.URL,
		RetryCount: 0,
		Timeout:    5,
		Secret:     "super-secret-token-do-not-leak",
		Headers: map[string]string{
			"Authorization": "Bearer secret-api-key-12345",
		},
	}

	err := m.sendNotification(wh, newTestPayload("security.test", "credential leak test"))
	if err == nil {
		t.Fatal("expected error")
	}

	errMsg := err.Error()
	for _, secret := range []string{wh.Secret, wh.Headers["Authorization"]} {
		if strings.Contains(errMsg, secret) {
			t.Errorf("error message leaks credential %q", secret)
		}
	}
}

// ---------------------------------------------------------------------------
// sendNotification - marshal error (unmarshallable data)
// ---------------------------------------------------------------------------

// TestSendNotification_MarshalError verifies that an unserializable payload
// (a channel in Data) fails fast with a marshal error before any HTTP
// attempt is made.
func TestSendNotification_MarshalError(t *testing.T) {
	t.Parallel()
	m := newWebhookTestManager()
	wh := &Webhook{
		ID:         "wh_marshal",
		URL:        "http://127.0.0.1:1",
		RetryCount: 0,
		Timeout:    1,
	}

	// Create payload with unmarshallable data (channel causes JSON marshal error)
	payload := newTestPayload("marshal.test", "marshal test")
	payload.Data = map[string]interface{}{"bad": make(chan int)}

	err := m.sendNotification(wh, payload)
	if err == nil {
		t.Fatal("expected error for unmarshallable payload")
	}
	if !strings.Contains(err.Error(), "failed to marshal payload") {
		t.Errorf("expected marshal error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// sendNotification - timeout
// ---------------------------------------------------------------------------

// TestSendNotification_Timeout verifies that a client-side timeout aborts a
// hanging endpoint instead of blocking until the server responds.
func TestSendNotification_Timeout(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // longer than client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := &Manager{
		ctx:        context.Background(),
		httpClient: &http.Client{Timeout: 100 * time.Millisecond},
		secrets:    make(map[string]string),
	}
	wh := &Webhook{
		ID:         "wh_timeout",
		URL:        server.URL,
		RetryCount: 0,
		Timeout:    1,
	}

	err := m.sendNotification(wh, newTestPayload("timeout.test", "timeout test"))
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
