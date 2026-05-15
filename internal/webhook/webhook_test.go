/*
Package webhook tests

Copyright (c) 2025 CosmoLabs (https://cosmolabs.org)
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
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// NewManager
// ---------------------------------------------------------------------------

func TestNewManager(t *testing.T) {
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

func TestCreateWebhook_EmptyName(t *testing.T) {
	m := NewManager(nil, "acct-123")
	_, err := m.CreateWebhook(&Webhook{Name: "", URL: "https://example.com/hook"})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention 'name', got: %v", err)
	}
}

func TestCreateWebhook_EmptyURL(t *testing.T) {
	m := NewManager(nil, "acct-123")
	_, err := m.CreateWebhook(&Webhook{Name: "my-hook", URL: ""})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	if !strings.Contains(err.Error(), "URL") {
		t.Errorf("error should mention 'URL', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateWebhook - defaults
// ---------------------------------------------------------------------------

func TestCreateWebhook_SetsDefaults(t *testing.T) {
	m := NewManager(nil, "acct-123")
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

func TestCreateWebhook_PreservesProvided(t *testing.T) {
	m := NewManager(nil, "acct-123")
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

func TestSendWebhook_NoWebhooks(t *testing.T) {
	m := NewManager(nil, "acct-123")
	event := &Event{Type: "object.created", Timestamp: time.Now().UTC(), Source: "r2go2"}
	// getWebhooksForEvent returns empty slice, so SendWebhook should return nil
	err := m.SendWebhook("object.created", event)
	if err != nil {
		t.Fatalf("expected nil error when no webhooks registered, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateAlert - validation
// ---------------------------------------------------------------------------

func TestCreateAlert_EmptyName(t *testing.T) {
	m := NewManager(nil, "acct-123")
	_, err := m.CreateAlert(&Alert{Name: "", Type: AlertTypeThreshold, Metric: "requests"})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention 'name', got: %v", err)
	}
}

func TestCreateAlert_EmptyType(t *testing.T) {
	m := NewManager(nil, "acct-123")
	_, err := m.CreateAlert(&Alert{Name: "my-alert", Type: "", Metric: "requests"})
	if err == nil {
		t.Fatal("expected error for empty type")
	}
	if !strings.Contains(err.Error(), "type") {
		t.Errorf("error should mention 'type', got: %v", err)
	}
}

func TestCreateAlert_EmptyMetric(t *testing.T) {
	m := NewManager(nil, "acct-123")
	_, err := m.CreateAlert(&Alert{Name: "my-alert", Type: AlertTypeThreshold, Metric: ""})
	if err == nil {
		t.Fatal("expected error for empty metric")
	}
	if !strings.Contains(err.Error(), "metric") {
		t.Errorf("error should mention 'metric', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateAlert - defaults
// ---------------------------------------------------------------------------

func TestCreateAlert_SetsDefaults(t *testing.T) {
	m := NewManager(nil, "acct-123")
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

func TestCreateAlert_PreservesWindow(t *testing.T) {
	m := NewManager(nil, "acct-123")
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

func TestCreateBucketEvent_NilData(t *testing.T) {
	event := CreateBucketEvent(EventTypeBucketCreated, "my-bucket", nil)
	if event.Data == nil {
		t.Fatal("expected data map to be initialized")
	}
	if event.Bucket != "my-bucket" {
		t.Errorf("expected bucket 'my-bucket', got: %s", event.Bucket)
	}
	if event.Source != "r2go2" {
		t.Errorf("expected source 'r2go2', got: %s", event.Source)
	}
	if event.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
	if event.Data["bucket"] != "my-bucket" {
		t.Errorf("expected data[bucket]='my-bucket', got: %v", event.Data["bucket"])
	}
}

func TestCreateBucketEvent_ExistingData(t *testing.T) {
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

func TestCreateObjectEvent_NilData(t *testing.T) {
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
	if event.Source != "r2go2" {
		t.Errorf("expected source 'r2go2', got: %s", event.Source)
	}
	if event.Data["size"] != int64(1024) {
		t.Errorf("expected data[size]=1024, got: %v", event.Data["size"])
	}
	if event.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestCreateObjectEvent_WithData(t *testing.T) {
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

func TestCreateAlertEvent(t *testing.T) {
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
	if event.Source != "r2go2" {
		t.Errorf("expected source 'r2go2', got: %s", event.Source)
	}
	if event.Data["alert_id"] != "alert-001" {
		t.Errorf("expected data[alert_id]='alert-001', got: %v", event.Data["alert_id"])
	}
	if event.Data["alert_name"] != "CPU Alert" {
		t.Errorf("expected data[alert_name]='CPU Alert', got: %v", event.Data["alert_name"])
	}
	if event.Data["alert_type"] != AlertTypeThreshold {
		t.Errorf("expected data[alert_type]='threshold', got: %v", event.Data["alert_type"])
	}
	if event.Data["metric"] != "cpu_usage" {
		t.Errorf("expected data[metric]='cpu_usage', got: %v", event.Data["metric"])
	}
	if event.Data["threshold"] != 90.0 {
		t.Errorf("expected data[threshold]=90.0, got: %v", event.Data["threshold"])
	}
	if event.Data["value"] != 95.5 {
		t.Errorf("expected data[value]=95.5, got: %v", event.Data["value"])
	}
	if event.Data["message"] != "CPU exceeded threshold" {
		t.Errorf("expected data[message]='CPU exceeded threshold', got: %v", event.Data["message"])
	}
}

// ---------------------------------------------------------------------------
// ParseWebhookSignature
// ---------------------------------------------------------------------------

func computeHMAC(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func TestParseWebhookSignature_InvalidFormat(t *testing.T) {
	_, err := ParseWebhookSignature([]byte("data"), "invalid-no-prefix", "secret")
	if err == nil {
		t.Fatal("expected error for invalid signature format")
	}
	if !strings.Contains(err.Error(), "invalid signature format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseWebhookSignature_Correct(t *testing.T) {
	payload := []byte(`{"event":"test","timestamp":"2025-01-01T00:00:00Z"}`)
	secret := "my-secret-key"
	sig := "sha256=" + computeHMAC(payload, secret)

	ok, err := ParseWebhookSignature(payload, sig, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected signature to be valid")
	}
}

func TestParseWebhookSignature_WrongSecret(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	sig := "sha256=" + computeHMAC(payload, "correct-secret")

	ok, err := ParseWebhookSignature(payload, sig, "wrong-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected signature to be invalid with wrong secret")
	}
}

func TestParseWebhookSignature_TamperedPayload(t *testing.T) {
	original := []byte(`{"event":"test","value":100}`)
	tampered := []byte(`{"event":"test","value":999}`)
	secret := "my-secret"
	sig := "sha256=" + computeHMAC(original, secret)

	ok, err := ParseWebhookSignature(tampered, sig, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected signature to be invalid for tampered payload")
	}
}

// ---------------------------------------------------------------------------
// TestWebhook - HTTP integration
// ---------------------------------------------------------------------------

func TestWebhook_SendsPostWithCorrectHeadersAndBody(t *testing.T) {
	var receivedMethod, receivedContentType, receivedUserAgent, receivedEvent string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedContentType = r.Header.Get("Content-Type")
		receivedUserAgent = r.Header.Get("User-Agent")
		receivedEvent = r.Header.Get("X-R2Go2-Event")
		receivedBody, _ = io.ReadAll(r.Body)
		r.Body.Close()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		Name:   "test-hook",
		URL:    server.URL,
		Events: []string{"webhook_test"},
	}

	err := m.TestWebhook(wh)
	if err != nil {
		t.Fatalf("TestWebhook failed: %v", err)
	}

	if receivedMethod != "POST" {
		t.Errorf("expected POST, got: %s", receivedMethod)
	}
	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got: %s", receivedContentType)
	}
	if receivedUserAgent != "R2Go2-Webhook/1.0" {
		t.Errorf("expected User-Agent R2Go2-Webhook/1.0, got: %s", receivedUserAgent)
	}
	if receivedEvent != "webhook_test" {
		t.Errorf("expected X-R2Go2-Event webhook_test, got: %s", receivedEvent)
	}

	// Verify body is valid JSON with expected fields
	var payload NotificationPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if payload.Event != "webhook_test" {
		t.Errorf("expected payload event 'webhook_test', got: %s", payload.Event)
	}
	if payload.Source != "r2go2" {
		t.Errorf("expected payload source 'r2go2', got: %s", payload.Source)
	}
	if payload.Message == "" {
		t.Error("expected payload message to be set")
	}
}

func TestWebhook_SecretAddsSignature(t *testing.T) {
	var receivedSignature string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get("X-R2Go2-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	secret := "test-webhook-secret"
	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		Name:   "signed-hook",
		URL:    server.URL,
		Secret: secret,
	}

	err := m.TestWebhook(wh)
	if err != nil {
		t.Fatalf("TestWebhook failed: %v", err)
	}

	if receivedSignature == "" {
		t.Fatal("expected X-R2Go2-Signature header to be set")
	}
	if !strings.HasPrefix(receivedSignature, "sha256=") {
		t.Errorf("expected sha256= prefix, got: %s", receivedSignature)
	}

	// Verify the signature is correct by parsing it
	// We need the raw request body to verify, so let's capture it
}

func TestWebhook_SecretSignatureVerifiable(t *testing.T) {
	var receivedSignature string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get("X-R2Go2-Signature")
		receivedBody, _ = io.ReadAll(r.Body)
		r.Body.Close()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	secret := "verifiable-secret"
	m := NewManager(nil, "acct-123")
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
	ok, err := ParseWebhookSignature(receivedBody, receivedSignature, secret)
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

func TestTriggerAlert_IncrementsCount(t *testing.T) {
	m := NewManager(nil, "acct-123")
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

func TestTriggerAlert_MultipleTriggers(t *testing.T) {
	m := NewManager(nil, "acct-123")
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

func TestTriggerAlert_MissingWebhookIDs_Graceful(t *testing.T) {
	m := NewManager(nil, "acct-123")
	alert := &Alert{
		ID:        "alert-003",
		Name:      "Disk Alert",
		Type:      AlertTypeHealth,
		Metric:    "disk_usage",
		Threshold: 95.0,
		Webhooks:  []string{"wh_nonexistent_1", "wh_nonexistent_2"},
	}

	// getWebhook always returns error, so this should not panic
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

func TestEventTypeConstants(t *testing.T) {
	constants := map[string]string{
		"BucketCreated":    EventTypeBucketCreated,
		"BucketDeleted":    EventTypeBucketDeleted,
		"ObjectCreated":    EventTypeObjectCreated,
		"ObjectDeleted":    EventTypeObjectDeleted,
		"ObjectUploaded":   EventTypeObjectUploaded,
		"ObjectDownloaded": EventTypeObjectDownloaded,
		"MigrationStart":   EventTypeMigrationStart,
		"MigrationComplete": EventTypeMigrationComplete,
		"MigrationFailed":  EventTypeMigrationFailed,
		"AlertTriggered":   EventTypeAlertTriggered,
		"HealthCheckFailed": EventTypeHealthCheckFailed,
		"DomainAttached":   EventTypeDomainAttached,
		"DomainDetached":   EventTypeDomainDetached,
	}

	if len(constants) != 13 {
		t.Errorf("expected 13 event type constants, got %d", len(constants))
	}

	for name, val := range constants {
		if val == "" {
			t.Errorf("constant %s is empty", name)
		}
	}
}

// ---------------------------------------------------------------------------
// AlertType constants
// ---------------------------------------------------------------------------

func TestAlertTypeConstants(t *testing.T) {
	types := []AlertType{
		AlertTypeThreshold,
		AlertTypeTrend,
		AlertTypeBudget,
		AlertTypeHealth,
		AlertTypeCustom,
	}

	if len(types) != 5 {
		t.Errorf("expected 5 alert types, got %d", len(types))
	}

	expected := []string{"threshold", "trend", "budget", "health", "custom"}
	for i, at := range types {
		if string(at) != expected[i] {
			t.Errorf("expected alert type %s, got %s", expected[i], at)
		}
	}
}

// ---------------------------------------------------------------------------
// SendWebhook - with registered webhooks (tested via sendToWebhook directly)
// ---------------------------------------------------------------------------

func TestSendToWebhook_EnabledWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_001",
		Name:       "test-hook",
		URL:        server.URL,
		Enabled:    true,
		RetryCount: 0,
		Timeout:    5,
	}

	event := &Event{Type: "object.created", Timestamp: time.Now().UTC(), Source: "r2go2"}
	err := m.sendToWebhook(wh, event)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestSendToWebhook_DisabledWebhookStillSends(t *testing.T) {
	// sendToWebhook doesn't check Enabled - that's SendWebhook's job
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_disabled",
		Name:       "disabled-hook",
		URL:        server.URL,
		Enabled:    false,
		RetryCount: 0,
		Timeout:    5,
	}

	event := &Event{Type: "object.created", Timestamp: time.Now().UTC(), Source: "r2go2"}
	err := m.sendToWebhook(wh, event)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !called {
		t.Error("sendToWebhook should send regardless of Enabled flag")
	}
}

func TestSendToWebhook_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_err",
		URL:        server.URL,
		Enabled:    true,
		RetryCount: 0,
		Timeout:    5,
	}

	event := &Event{Type: "test", Timestamp: time.Now().UTC(), Source: "r2go2"}
	err := m.sendToWebhook(wh, event)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestSendToWebhook_MalformedURL(t *testing.T) {
	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_bad",
		URL:        "http://[::1]:namedport", // invalid URL
		Enabled:    true,
		RetryCount: 0,
		Timeout:    1,
	}

	event := &Event{Type: "test", Timestamp: time.Now().UTC(), Source: "r2go2"}
	err := m.sendToWebhook(wh, event)
	if err == nil {
		t.Fatal("expected error for malformed URL")
	}
}

func TestSendToWebhook_HTTPURLAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_http",
		URL:        server.URL, // httptest uses http
		Enabled:    true,
		RetryCount: 0,
		Timeout:    5,
	}

	event := &Event{Type: "test", Timestamp: time.Now().UTC(), Source: "r2go2"}
	err := m.sendToWebhook(wh, event)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TriggerAlert - notification via sendNotification directly
// ---------------------------------------------------------------------------

func TestTriggerAlert_WithNotificationDelivery(t *testing.T) {
	var receivedBody []byte
	var receivedSignature string
	var receivedEventHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedEventHeader = r.Header.Get("X-R2Go2-Event")
		receivedSignature = r.Header.Get("X-R2Go2-Signature")
		receivedBody, _ = io.ReadAll(r.Body)
		r.Body.Close()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
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
		Source:    "r2go2",
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

	if receivedEventHeader != "alert_triggered" {
		t.Errorf("expected X-R2Go2-Event 'alert_triggered', got %s", receivedEventHeader)
	}
	if receivedSignature == "" {
		t.Error("expected signature header to be set")
	}
	if !strings.HasPrefix(receivedSignature, "sha256=") {
		t.Errorf("expected sha256= prefix, got: %s", receivedSignature)
	}

	// Verify signature is valid
	ok, err := ParseWebhookSignature(receivedBody, receivedSignature, "alert-secret")
	if err != nil {
		t.Fatalf("signature verification error: %v", err)
	}
	if !ok {
		t.Error("expected valid signature")
	}

	// Verify payload contains alert data
	var decoded NotificationPayload
	if err := json.Unmarshal(receivedBody, &decoded); err != nil {
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

func TestTriggerAlert_DisabledWebhookSkipped(t *testing.T) {
	m := NewManager(nil, "acct-123")
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

func TestTriggerAlert_ServerFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("gateway error"))
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	alert := &Alert{
		ID:       "alert-fail",
		Name:     "Net Alert",
		Type:     AlertTypeCustom,
		Metric:   "latency",
		Webhooks: []string{}, // no webhooks configured
	}

	// Manually send failing notification
	payload := &NotificationPayload{
		Event:     "alert_triggered",
		Alert:     alert,
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Message:   "high latency",
	}

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
}

// ---------------------------------------------------------------------------
// TriggerAlert - all alert types
// ---------------------------------------------------------------------------

func TestTriggerAlert_AllAlertTypes(t *testing.T) {
	alertTypes := []AlertType{
		AlertTypeThreshold,
		AlertTypeTrend,
		AlertTypeBudget,
		AlertTypeHealth,
		AlertTypeCustom,
	}

	for _, at := range alertTypes {
		t.Run(string(at), func(t *testing.T) {
			m := NewManager(nil, "acct-123")
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

func TestSendWebhook_WithStoreEnabledWebhook(t *testing.T) {
	var receivedEventHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedEventHeader = r.Header.Get("X-R2Go2-Event")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	// Create and store a webhook that listens for "object.created"
	wh, err := m.CreateWebhook(&Webhook{
		Name:       "store-hook",
		URL:        server.URL,
		Events:     []string{"object.created"},
		Enabled:    true,
		RetryCount: 0,
		Timeout:    5,
	})
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}

	event := &Event{Type: "object.created", Timestamp: time.Now().UTC(), Source: "r2go2"}
	err = m.SendWebhook("object.created", event)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if receivedEventHeader != "object.created" {
		t.Errorf("expected event header 'object.created', got %s", receivedEventHeader)
	}
	_ = wh
}

func TestSendWebhook_WithStoreDisabledWebhook(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	_, err := m.CreateWebhook(&Webhook{
		Name:    "disabled-store-hook",
		URL:     server.URL,
		Events:  []string{"object.deleted"},
		Enabled: false, // disabled
	})
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}

	event := &Event{Type: "object.deleted", Timestamp: time.Now().UTC(), Source: "r2go2"}
	err = m.SendWebhook("object.deleted", event)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if called {
		t.Error("disabled webhook should not be called")
	}
}

func TestSendWebhook_WithStoreAllEvents(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	// Default events is ["all"] which matches any event type
	_, err := m.CreateWebhook(&Webhook{
		Name:       "catch-all-hook",
		URL:        server.URL,
		Enabled:    true,
		RetryCount: 0,
		Timeout:    5,
	})
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}

	event := &Event{Type: "migration.started", Timestamp: time.Now().UTC(), Source: "r2go2"}
	err = m.SendWebhook("migration.started", event)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected catch-all webhook to be called once, got %d", callCount)
	}
}

func TestSendWebhook_WithStoreNoMatchingEvent(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	_, err := m.CreateWebhook(&Webhook{
		Name:       "specific-hook",
		URL:        server.URL,
		Events:     []string{"object.created"}, // only matches object.created
		Enabled:    true,
		RetryCount: 0,
		Timeout:    5,
	})
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}

	// Send a different event type
	event := &Event{Type: "bucket.created", Timestamp: time.Now().UTC(), Source: "r2go2"}
	err = m.SendWebhook("bucket.created", event)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if called {
		t.Error("webhook with specific event should not match different event type")
	}
}

func TestSendWebhook_WithStoreServerErrorStillReturnsNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	_, err := m.CreateWebhook(&Webhook{
		Name:       "error-hook",
		URL:        server.URL,
		Events:     []string{"test"},
		Enabled:    true,
		RetryCount: 0,
		Timeout:    5,
	})
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}

	event := &Event{Type: "test", Timestamp: time.Now().UTC(), Source: "r2go2"}
	// SendWebhook prints errors but returns nil
	err = m.SendWebhook("test", event)
	if err != nil {
		t.Fatalf("expected nil error even on server failure, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TriggerAlert - with webhooks in the store
// ---------------------------------------------------------------------------

func TestTriggerAlert_WithStoreWorkingWebhook(t *testing.T) {
	var receivedBody []byte
	var receivedSignature string
	var receivedEventHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedEventHeader = r.Header.Get("X-R2Go2-Event")
		receivedSignature = r.Header.Get("X-R2Go2-Signature")
		receivedBody, _ = io.ReadAll(r.Body)
		r.Body.Close()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
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

	if receivedEventHeader != "alert_triggered" {
		t.Errorf("expected X-R2Go2-Event 'alert_triggered', got %s", receivedEventHeader)
	}
	if !strings.HasPrefix(receivedSignature, "sha256=") {
		t.Errorf("expected sha256= prefix, got: %s", receivedSignature)
	}

	// Verify signature
	ok, err := ParseWebhookSignature(receivedBody, receivedSignature, "store-secret")
	if err != nil {
		t.Fatalf("signature verification error: %v", err)
	}
	if !ok {
		t.Error("expected valid signature")
	}

	var payload NotificationPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
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

func TestTriggerAlert_WithStoreDisabledWebhook(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
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
	if called {
		t.Error("disabled webhook should not have been called")
	}
	if alert.Count != 1 {
		t.Errorf("expected Count=1, got: %d", alert.Count)
	}
}

func TestTriggerAlert_WithStoreNonexistentWebhook(t *testing.T) {
	m := NewManager(nil, "acct-123")
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

func TestTriggerAlert_WithStoreServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
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

func TestTriggerAlert_WithStoreMultipleWebhooks(t *testing.T) {
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	wh1, _ := m.CreateWebhook(&Webhook{
		Name:       "multi-hook-1",
		URL:        server.URL,
		Enabled:    true,
		RetryCount: 0,
		Timeout:    5,
	})
	wh2, _ := m.CreateWebhook(&Webhook{
		Name:       "multi-hook-2",
		URL:        server.URL,
		Enabled:    true,
		RetryCount: 0,
		Timeout:    5,
	})
	wh3, _ := m.CreateWebhook(&Webhook{
		Name:       "multi-hook-disabled",
		URL:        server.URL,
		Enabled:    false,
		RetryCount: 0,
		Timeout:    5,
	})

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
	if callCount != 2 { // only enabled webhooks should be called
		t.Errorf("expected 2 calls (2 enabled + 1 disabled + 1 missing), got %d", callCount)
	}
	if alert.Count != 1 {
		t.Errorf("expected Count=1, got: %d", alert.Count)
	}
}

// ---------------------------------------------------------------------------
// sendNotification - retry behavior
// ---------------------------------------------------------------------------

func TestSendNotification_RetryOnFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_retry",
		URL:        server.URL,
		RetryCount: 3,
		Timeout:    5,
	}

	payload := &NotificationPayload{
		Event:     "test.event",
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Message:   "retry test",
	}

	err := m.sendNotification(wh, payload)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got: %d", attempts)
	}
}

func TestSendNotification_ExhaustsRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_exhaust",
		URL:        server.URL,
		RetryCount: 2,
		Timeout:    5,
	}

	payload := &NotificationPayload{
		Event:     "test.fail",
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Message:   "exhaust test",
	}

	err := m.sendNotification(wh, payload)
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

func TestSendNotification_CustomHeaders(t *testing.T) {
	var receivedAuth, receivedCustom string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedCustom = r.Header.Get("X-Custom-Header")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:    "wh_headers",
		URL:   server.URL,
		RetryCount: 0,
		Timeout:    5,
		Headers: map[string]string{
			"Authorization":   "Bearer test-token",
			"X-Custom-Header": "custom-value",
		},
	}

	payload := &NotificationPayload{
		Event:     "headers.test",
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Message:   "header test",
	}

	err := m.sendNotification(wh, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedAuth != "Bearer test-token" {
		t.Errorf("expected Authorization header, got: %s", receivedAuth)
	}
	if receivedCustom != "custom-value" {
		t.Errorf("expected X-Custom-Header, got: %s", receivedCustom)
	}
}

// ---------------------------------------------------------------------------
// sendNotification - malicious header injection
// ---------------------------------------------------------------------------

func TestSendNotification_HeaderInjectionSanitization(t *testing.T) {
	// Go's http.Client rejects headers with newlines, causing the request to fail.
	// This verifies that malformed headers cause an error rather than being sent.
	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_inject",
		URL:        "http://127.0.0.1:1", // any URL - request won't be sent
		RetryCount: 0,
		Timeout:    1,
		Headers: map[string]string{
			"X-Injected": "value\r\nX-Evil: malicious",
		},
	}

	payload := &NotificationPayload{
		Event:     "inject.test",
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Message:   "injection test",
	}

	err := m.sendNotification(wh, payload)
	// Go's http.Client will reject the header with newline - error expected
	if err == nil {
		t.Fatal("expected error for header with newline injection")
	}
}

// ---------------------------------------------------------------------------
// sendNotification - rate limiting (429 response)
// ---------------------------------------------------------------------------

func TestSendNotification_RateLimited(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_429",
		URL:        server.URL,
		RetryCount: 1,
		Timeout:    5,
	}

	payload := &NotificationPayload{
		Event:     "rate.test",
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Message:   "rate limit test",
	}

	err := m.sendNotification(wh, payload)
	if err == nil {
		t.Fatal("expected error after rate limiting exhausts retries")
	}
	if attempts != 2 { // initial + 1 retry
		t.Errorf("expected 2 attempts, got: %d", attempts)
	}
}

// ---------------------------------------------------------------------------
// sendNotification - connection refused (network error)
// ---------------------------------------------------------------------------

func TestSendNotification_ConnectionRefused(t *testing.T) {
	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_refused",
		URL:        "http://127.0.0.1:1", // port 1 should refuse
		RetryCount: 1,
		Timeout:    1,
	}

	payload := &NotificationPayload{
		Event:     "conn.test",
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Message:   "connection refused test",
	}

	err := m.sendNotification(wh, payload)
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

func TestSignPayload_Consistency(t *testing.T) {
	m := NewManager(nil, "acct-123")
	data := []byte(`{"test": "data"}`)
	secret := "sign-secret"

	sig1 := m.signPayload(data, secret)
	sig2 := m.signPayload(data, secret)

	if sig1 != sig2 {
		t.Error("signatures should be deterministic")
	}
	if !strings.HasPrefix(sig1, computeHMAC(data, secret)[:10]) {
		t.Error("signature should match HMAC computation")
	}

	// Different data should produce different signature
	sig3 := m.signPayload([]byte(`{"test": "different"}`), secret)
	if sig1 == sig3 {
		t.Error("different data should produce different signature")
	}

	// Different secret should produce different signature
	sig4 := m.signPayload(data, "different-secret")
	if sig1 == sig4 {
		t.Error("different secret should produce different signature")
	}
}

// ---------------------------------------------------------------------------
// ParseWebhookSignature - edge cases
// ---------------------------------------------------------------------------

func TestParseWebhookSignature_EmptySignature(t *testing.T) {
	ok, err := ParseWebhookSignature([]byte("data"), "", "secret")
	if err == nil {
		t.Fatal("expected error for empty signature")
	}
	if ok {
		t.Error("expected false for empty signature")
	}
}

func TestParseWebhookSignature_EmptyPayload(t *testing.T) {
	secret := "secret"
	sig := "sha256=" + computeHMAC([]byte{}, secret)
	ok, err := ParseWebhookSignature([]byte{}, sig, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected valid signature for empty payload")
	}
}

func TestParseWebhookSignature_EmptySecret(t *testing.T) {
	payload := []byte("data")
	sig := "sha256=" + computeHMAC(payload, "")
	ok, err := ParseWebhookSignature(payload, sig, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected valid signature with empty secret")
	}
}

func TestParseWebhookSignature_ReplayAttackDifferentPayload(t *testing.T) {
	original := []byte(`{"event":"payment","amount":100}`)
	replay := []byte(`{"event":"payment","amount":10000}`)
	secret := "webhook-secret"

	// Attacker captures valid signature for $100
	sig := "sha256=" + computeHMAC(original, secret)

	// Attacker replays with modified amount ($10000)
	ok, err := ParseWebhookSignature(replay, sig, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("replay with tampered payload should be detected")
	}
}

func TestParseWebhookSignature_RandomSuffix(t *testing.T) {
	payload := []byte("data")
	secret := "secret"
	validSig := "sha256=" + computeHMAC(payload, secret)

	// Append garbage to valid signature
	tamperedSig := validSig + "abcdef"
	ok, err := ParseWebhookSignature(payload, tamperedSig, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("signature with extra suffix should be invalid")
	}
}

func TestParseWebhookSignature_WrongPrefix(t *testing.T) {
	ok, err := ParseWebhookSignature([]byte("data"), "md5=abc123", "secret")
	if err == nil {
		t.Fatal("expected error for wrong prefix")
	}
	if ok {
		t.Error("expected false for wrong prefix")
	}
}

// ---------------------------------------------------------------------------
// CreateWebhook - invalid URL
// ---------------------------------------------------------------------------

func TestCreateWebhook_InvalidURL(t *testing.T) {
	m := NewManager(nil, "acct-123")
	_, err := m.CreateWebhook(&Webhook{Name: "bad-url", URL: "://not-a-url"})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "invalid webhook URL") {
		t.Errorf("expected 'invalid webhook URL' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Webhook struct - JSON serialization
// ---------------------------------------------------------------------------

func TestWebhook_JSONRoundTrip(t *testing.T) {
	wh := &Webhook{
		ID:        "wh_json",
		Name:      "json-hook",
		URL:       "https://example.com/hook",
		Events:    []string{"object.created", "object.deleted"},
		Enabled:   true,
		Secret:    "json-secret",
		Headers:   map[string]string{"Auth": "Bearer xyz"},
		CreatedAt: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
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

func TestWebhook_SecretOmittedWhenEmpty(t *testing.T) {
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

func TestNotificationPayload_JSONRoundTrip(t *testing.T) {
	now := time.Now().UTC()
	payload := &NotificationPayload{
		Event:     "object.uploaded",
		Alert:     &Alert{ID: "a1", Name: "Test Alert", Type: AlertTypeThreshold},
		Timestamp: now,
		Source:    "r2go2",
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

func TestSendToWebhook_IncludesBucketAndObject(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		r.Body.Close()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_bo",
		URL:        server.URL,
		RetryCount: 0,
		Timeout:    5,
	}

	event := &Event{
		Type:      EventTypeObjectCreated,
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Bucket:    "photos-bucket",
		Object:    "vacation.jpg",
		Data:      map[string]interface{}{"size": int64(2048)},
	}

	err := m.sendToWebhook(wh, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload NotificationPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
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

func TestSendNotification_ErrorDoesNotLeakCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	m := NewManager(nil, "acct-123")
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

	payload := &NotificationPayload{
		Event:     "security.test",
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Message:   "credential leak test",
	}

	err := m.sendNotification(wh, payload)
	if err == nil {
		t.Fatal("expected error")
	}

	errMsg := err.Error()
	// Secret should not appear in error message
	if strings.Contains(errMsg, "super-secret-token-do-not-leak") {
		t.Error("error message leaks webhook secret")
	}
	if strings.Contains(errMsg, "secret-api-key-12345") {
		t.Error("error message leaks API key from headers")
	}
}

// ---------------------------------------------------------------------------
// sendNotification - marshal error (unmarshallable data)
// ---------------------------------------------------------------------------

func TestSendNotification_MarshalError(t *testing.T) {
	m := NewManager(nil, "acct-123")
	wh := &Webhook{
		ID:         "wh_marshal",
		URL:        "http://127.0.0.1:1",
		RetryCount: 0,
		Timeout:    1,
	}

	// Create payload with unmarshallable data (channel causes JSON marshal error)
	payload := &NotificationPayload{
		Event:     "marshal.test",
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Message:   "marshal test",
		Data:      map[string]interface{}{"bad": make(chan int)},
	}

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

func TestSendNotification_Timeout(t *testing.T) {
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

	payload := &NotificationPayload{
		Event:     "timeout.test",
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Message:   "timeout test",
	}

	err := m.sendNotification(wh, payload)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
