/*
Package webhook provides webhook and alerting functionality for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// Manager handles webhook operations and alerting
type Manager struct {
	ctx        context.Context
	cf         *cloudflare.API
	accountID  string
	httpClient *http.Client
	secrets    map[string]string // Webhook secrets for signature verification
}

// NewManager creates a new webhook manager
func NewManager(cf *cloudflare.API, accountID string) *Manager {
	return &Manager{
		ctx:        context.Background(),
		cf:         cf,
		accountID:  accountID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		secrets:    make(map[string]string),
	}
}

// Webhook represents a webhook configuration
type Webhook struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Events      []string          `json:"events"`
	Enabled     bool              `json:"enabled"`
	Secret      string            `json:"secret,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	RetryCount  int               `json:"retry_count"`
	Timeout     int               `json:"timeout"` // in seconds
}

// Event represents an event that can trigger webhooks
type Event struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Bucket    string                 `json:"bucket,omitempty"`
	Object    string                 `json:"object,omitempty"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
}

// Alert represents an alert configuration
type Alert struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        AlertType         `json:"type"`
	Threshold   float64           `json:"threshold"`
	Metric      string            `json:"metric"`
	Window      string            `json:"window"`     // time window (e.g., "1h", "24h")
	Enabled     bool              `json:"enabled"`
	Webhooks    []string          `json:"webhooks"`   // webhook IDs to notify
	Conditions  map[string]string `json:"conditions"` // additional conditions
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	LastTrigger time.Time         `json:"last_trigger,omitempty"`
	Count       int               `json:"count"`       // how many times triggered
}

// AlertType represents different types of alerts
type AlertType string

const (
	AlertTypeThreshold AlertType = "threshold"
	AlertTypeTrend     AlertType = "trend"
	AlertTypeBudget    AlertType = "budget"
	AlertTypeHealth    AlertType = "health"
	AlertTypeCustom    AlertType = "custom"
)

// NotificationPayload represents a webhook notification payload
type NotificationPayload struct {
	Event      string                 `json:"event"`
	Alert      *Alert                 `json:"alert,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Source     string                 `json:"source"`
	Bucket     string                 `json:"bucket,omitempty"`
	Object     string                 `json:"object,omitempty"`
	Value      float64                `json:"value,omitempty"`
	Threshold  float64                `json:"threshold,omitempty"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Signature  string                 `json:"signature,omitempty"`
}

// CreateWebhook creates a new webhook
func (m *Manager) CreateWebhook(webhook *Webhook) (*Webhook, error) {
	if webhook.Name == "" {
		return nil, fmt.Errorf("webhook name is required")
	}
	if webhook.URL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	// Validate URL
	if _, err := url.Parse(webhook.URL); err != nil {
		return nil, fmt.Errorf("invalid webhook URL: %w", err)
	}

	// Generate ID and timestamps
	webhook.ID = generateID()
	webhook.CreatedAt = time.Now().UTC()
	webhook.UpdatedAt = time.Now().UTC()

	// Set defaults
	if webhook.Timeout == 0 {
		webhook.Timeout = 30
	}
	if webhook.RetryCount == 0 {
		webhook.RetryCount = 3
	}
	if len(webhook.Events) == 0 {
		webhook.Events = []string{"all"}
	}

	// Store webhook (in a real implementation, this would be stored in a database)
	// For now, we'll return the created webhook

	return webhook, nil
}

// SendWebhook sends an event to webhooks
func (m *Manager) SendWebhook(eventType string, event *Event) error {
	// Get webhooks that should receive this event
	webhooks, err := m.getWebhooksForEvent(eventType)
	if err != nil {
		return fmt.Errorf("failed to get webhooks for event: %w", err)
	}

	// Send to each webhook
	for _, webhook := range webhooks {
		if !webhook.Enabled {
			continue
		}

		if err := m.sendToWebhook(webhook, event); err != nil {
			fmt.Printf("Failed to send webhook to %s: %v\n", webhook.URL, err)
		}
	}

	return nil
}

// TriggerAlert triggers an alert
func (m *Manager) TriggerAlert(alert *Alert, value float64, message string, data map[string]interface{}) error {
	alert.LastTrigger = time.Now().UTC()
	alert.Count++

	// Create notification payload
	payload := &NotificationPayload{
		Event:     "alert_triggered",
		Alert:     alert,
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Value:     value,
		Threshold: alert.Threshold,
		Message:   message,
		Data:      data,
	}

	// Send to configured webhooks
	for _, webhookID := range alert.Webhooks {
		webhook, err := m.getWebhook(webhookID)
		if err != nil {
			fmt.Printf("Failed to get webhook %s: %v\n", webhookID, err)
			continue
		}

		if !webhook.Enabled {
			continue
		}

		if err := m.sendNotification(webhook, payload); err != nil {
			fmt.Printf("Failed to send alert notification to %s: %v\n", webhook.URL, err)
		}
	}

	return nil
}

// CreateAlert creates a new alert
func (m *Manager) CreateAlert(alert *Alert) (*Alert, error) {
	if alert.Name == "" {
		return nil, fmt.Errorf("alert name is required")
	}
	if alert.Type == "" {
		return nil, fmt.Errorf("alert type is required")
	}
	if alert.Metric == "" {
		return nil, fmt.Errorf("metric is required")
	}

	// Generate ID and timestamps
	alert.ID = generateID()
	alert.CreatedAt = time.Now().UTC()
	alert.UpdatedAt = time.Now().UTC()

	// Set defaults
	if alert.Window == "" {
		alert.Window = "1h"
	}

	return alert, nil
}

// TestWebhook tests a webhook configuration
func (m *Manager) TestWebhook(webhook *Webhook) error {
	testEvent := &Event{
		Type:      "webhook_test",
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Data: map[string]interface{}{
			"message": "This is a test webhook from R2Go2",
		},
	}

	return m.sendToWebhook(webhook, testEvent)
}

// Helper methods

func (m *Manager) getWebhooksForEvent(eventType string) ([]*Webhook, error) {
	// In a real implementation, this would query a database
	// For now, return empty slice
	return []*Webhook{}, nil
}

func (m *Manager) getWebhook(id string) (*Webhook, error) {
	// In a real implementation, this would query a database
	// For now, return error
	return nil, fmt.Errorf("webhook not found: %s", id)
}

func (m *Manager) sendToWebhook(webhook *Webhook, event *Event) error {
	// Create event-specific payload
	payload := &NotificationPayload{
		Event:     event.Type,
		Timestamp: event.Timestamp,
		Source:    event.Source,
		Bucket:    event.Bucket,
		Object:    event.Object,
		Data:      event.Data,
		Message:   fmt.Sprintf("Event: %s", event.Type),
	}

	return m.sendNotification(webhook, payload)
}

func (m *Manager) sendNotification(webhook *Webhook, payload *NotificationPayload) error {
	// Serialize payload
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "R2Go2-Webhook/1.0")
	req.Header.Set("X-R2Go2-Event", payload.Event)
	req.Header.Set("X-R2Go2-Timestamp", payload.Timestamp.Format(time.RFC3339))

	// Add custom headers
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	// Add signature if secret is configured
	if webhook.Secret != "" {
		signature := m.signPayload(data, webhook.Secret)
		req.Header.Set("X-R2Go2-Signature", "sha256="+signature)
	}

	// Send request with retries
	var lastErr error
	for attempt := 0; attempt <= webhook.RetryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
		}

		resp, err := m.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Check response
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			return nil
		}

		// Read error response for debugging
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		lastErr = fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Errorf("failed to send webhook after %d attempts: %w", webhook.RetryCount+1, lastErr)
}

func (m *Manager) signPayload(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("wh_%d", time.Now().UnixNano())
}

// Common event types
const (
	EventTypeBucketCreated    = "bucket.created"
	EventTypeBucketDeleted    = "bucket.deleted"
	EventTypeObjectCreated    = "object.created"
	EventTypeObjectDeleted    = "object.deleted"
	EventTypeObjectUploaded   = "object.uploaded"
	EventTypeObjectDownloaded = "object.downloaded"
	EventTypeMigrationStart   = "migration.started"
	EventTypeMigrationComplete = "migration.completed"
	EventTypeMigrationFailed   = "migration.failed"
	EventTypeAlertTriggered    = "alert.triggered"
	EventTypeHealthCheckFailed = "health_check.failed"
	EventTypeDomainAttached   = "domain.attached"
	EventTypeDomainDetached   = "domain.detached"
)

// Utility functions for creating common events

// CreateBucketEvent creates a bucket-related event
func CreateBucketEvent(eventType, bucket string, data map[string]interface{}) *Event {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["bucket"] = bucket

	return &Event{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Bucket:    bucket,
		Source:    "r2go2",
		Data:      data,
	}
}

// CreateObjectEvent creates an object-related event
func CreateObjectEvent(eventType, bucket, object string, size int64, data map[string]interface{}) *Event {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["bucket"] = bucket
	data["object"] = object
	data["size"] = size

	return &Event{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Bucket:    bucket,
		Object:    object,
		Source:    "r2go2",
		Data:      data,
	}
}

// CreateAlertEvent creates an alert-related event
func CreateAlertEvent(alert *Alert, value float64, message string) *Event {
	return &Event{
		Type:      EventTypeAlertTriggered,
		Timestamp: time.Now().UTC(),
		Source:    "r2go2",
		Data: map[string]interface{}{
			"alert_id":   alert.ID,
			"alert_name": alert.Name,
			"alert_type": alert.Type,
			"metric":     alert.Metric,
			"threshold":  alert.Threshold,
			"value":      value,
			"message":    message,
		},
	}
}

// ParseWebhookSignature parses and verifies a webhook signature
func ParseWebhookSignature(payload []byte, signature, secret string) (bool, error) {
	if !strings.HasPrefix(signature, "sha256=") {
		return false, fmt.Errorf("invalid signature format")
	}

	signatureHex := strings.TrimPrefix(signature, "sha256=")

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(signatureHex), []byte(expectedSignature)), nil
}