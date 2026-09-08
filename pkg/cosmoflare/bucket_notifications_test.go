package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// bucketNotificationHandler captures the last request seen by the test server.
type bucketNotificationHandler struct {
	method string
	path   string
	auth   string
	body   []byte
	inner  http.HandlerFunc
}

func (h *bucketNotificationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.method = r.Method
	h.path = r.URL.Path
	h.auth = r.Header.Get("Authorization")
	h.body = make([]byte, 0)
	if r.Body != nil {
		buf := make([]byte, 64*1024)
		n, _ := r.Body.Read(buf)
		h.body = buf[:n]
	}
	if h.inner != nil {
		h.inner(w, r)
	}
}

func newBucketNotificationTestServer(t *testing.T, h *bucketNotificationHandler) *BucketNotificationService {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewBucketNotificationService("ACC", "tok-secret", WithBucketNotificationBaseURL(srv.URL))
}

func TestBucketNotificationList(t *testing.T) {
	h := &bucketNotificationHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"errors":[],"result":{"bucketName":"bk","queues":[` +
				`{"queueId":"q1","queueName":"queue-one","rules":[{"ruleId":"r1","createdAt":"2024-09-19T21:54:48.405Z","actions":["PutObject"],"description":"uploads","prefix":"img/","suffix":".jpeg"}]},` +
				`{"queueId":"q2","queueName":"queue-two","rules":[]}]}}`))
		},
	}
	svc := newBucketNotificationTestServer(t, h)

	queues, err := svc.List(context.Background(), "bk")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if h.method != http.MethodGet {
		t.Errorf("method = %s, want GET", h.method)
	}
	if h.path != "/accounts/ACC/event_notifications/r2/bk/configuration" {
		t.Errorf("path = %s", h.path)
	}
	if h.auth != "Bearer tok-secret" {
		t.Errorf("auth = %s", h.auth)
	}
	if len(queues) != 2 {
		t.Fatalf("queues = %d, want 2", len(queues))
	}
	if queues[0].QueueID != "q1" || queues[0].QueueName != "queue-one" {
		t.Errorf("queue[0] = %+v", queues[0])
	}
	if len(queues[0].Rules) != 1 {
		t.Fatalf("queue[0] rules = %d, want 1", len(queues[0].Rules))
	}
	rule := queues[0].Rules[0]
	if rule.RuleID != "r1" {
		t.Errorf("ruleId = %s, want r1", rule.RuleID)
	}
	wantTime := time.Date(2024, 9, 19, 21, 54, 48, 405000000, time.UTC)
	if !rule.CreatedAt.Equal(wantTime) {
		t.Errorf("createdAt = %s, want %s", rule.CreatedAt, wantTime)
	}
	if len(rule.Actions) != 1 || rule.Actions[0] != ActionPutObject {
		t.Errorf("actions = %v", rule.Actions)
	}
	if rule.Description != "uploads" || rule.Prefix != "img/" || rule.Suffix != ".jpeg" {
		t.Errorf("rule fields = %+v", rule)
	}
	if queues[1].QueueID != "q2" || len(queues[1].Rules) != 0 {
		t.Errorf("queue[1] = %+v", queues[1])
	}
}

func TestBucketNotificationSet(t *testing.T) {
	h := &bucketNotificationHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"errors":[],"result":{"bucketName":"bk","queues":[{"queueId":"q1","queueName":"queue-one","rules":[]}]}}`))
		},
	}
	svc := newBucketNotificationTestServer(t, h)

	rules := []NotificationRule{
		{
			Actions:     []NotificationAction{ActionPutObject, ActionCopyObject},
			Description: "uploads",
			Prefix:      "img/",
			Suffix:      ".jpeg",
		},
	}
	if err := svc.Set(context.Background(), "bk", "q1", rules); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if h.method != http.MethodPut {
		t.Errorf("method = %s, want PUT", h.method)
	}
	if h.path != "/accounts/ACC/event_notifications/r2/bk/configuration/queues/q1" {
		t.Errorf("path = %s", h.path)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(h.body, &got); err != nil {
		t.Fatalf("body decode: %v (body=%s)", err, h.body)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"rules":[{"actions":["PutObject","CopyObject"],"description":"uploads","prefix":"img/","suffix":".jpeg"}]}`
	if strings.TrimSpace(string(raw)) != want {
		t.Errorf("body = %s, want %s", raw, want)
	}

	// Empty rule set must be rejected before any HTTP call.
	h2 := &bucketNotificationHandler{}
	svc2 := newBucketNotificationTestServer(t, h2)
	if err := svc2.Set(context.Background(), "bk", "q1", nil); err == nil {
		t.Error("Set with no rules: want error")
	} else if h2.method != "" {
		t.Error("Set with no rules: server must not be hit")
	}
}

func TestBucketNotificationDeleteWithIds(t *testing.T) {
	h := &bucketNotificationHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"errors":[],"result":null}`))
		},
	}
	svc := newBucketNotificationTestServer(t, h)

	if err := svc.Delete(context.Background(), "bk", "q1", []string{"a", "b"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if h.method != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", h.method)
	}
	if h.path != "/accounts/ACC/event_notifications/r2/bk/configuration/queues/q1" {
		t.Errorf("path = %s", h.path)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(h.body, &got); err != nil {
		t.Fatalf("body decode: %v (body=%s)", err, h.body)
	}
	ids, ok := got["ruleIds"].([]interface{})
	if !ok {
		t.Fatalf("body %s: missing ruleIds array", h.body)
	}
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Errorf("ruleIds = %v, want [a b]", ids)
	}
}

func TestBucketNotificationDeleteAll(t *testing.T) {
	h := &bucketNotificationHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"errors":[],"result":null}`))
		},
	}
	svc := newBucketNotificationTestServer(t, h)

	if err := svc.Delete(context.Background(), "bk", "q1", nil); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if h.method != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", h.method)
	}
	if h.path != "/accounts/ACC/event_notifications/r2/bk/configuration/queues/q1" {
		t.Errorf("path = %s", h.path)
	}
	if strings.TrimSpace(string(h.body)) != "" {
		t.Errorf("body = %s, want empty", h.body)
	}
	var got map[string]interface{}
	_ = json.Unmarshal(h.body, &got) // intentionally ignored: body must be empty
	if _, present := got["ruleIds"]; present {
		t.Errorf("body %s must not contain ruleIds key", h.body)
	}
}

func TestBucketNotificationValidation(t *testing.T) {
	h := &bucketNotificationHandler{}
	svc := newBucketNotificationTestServer(t, h)

	// Invalid action string: rejected before any HTTP call.
	err := svc.Set(context.Background(), "bk", "q1", []NotificationRule{
		{Actions: []NotificationAction{NotificationAction("HeadObject")}},
	})
	if err == nil {
		t.Fatal("invalid action: want error")
	}
	if h.method != "" {
		t.Error("invalid action: server must not be hit")
	}

	// Empty actions: rejected before any HTTP call.
	h2 := &bucketNotificationHandler{}
	svc2 := newBucketNotificationTestServer(t, h2)
	if err := svc2.Set(context.Background(), "bk", "q1", []NotificationRule{{Actions: nil}}); err == nil {
		t.Error("nil actions: want error")
	} else if h2.method != "" {
		t.Error("nil actions: server must not be hit")
	}

	// Duplicate actions within one rule: rejected before any HTTP call.
	h3 := &bucketNotificationHandler{}
	svc3 := newBucketNotificationTestServer(t, h3)
	err = svc3.Set(context.Background(), "bk", "q1", []NotificationRule{
		{Actions: []NotificationAction{ActionPutObject, ActionPutObject}},
	})
	if err == nil {
		t.Fatal("duplicate actions: want error")
	}
	if h3.method != "" {
		t.Error("duplicate actions: server must not be hit")
	}
}

func TestBucketNotificationAPIError(t *testing.T) {
	h := &bucketNotificationHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":false,"errors":[{"code":10001,"message":"boom"}],"result":null}`))
		},
	}
	svc := newBucketNotificationTestServer(t, h)

	if _, err := svc.List(context.Background(), "bk"); err == nil {
		t.Fatal("List: want error")
	} else if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %v, want it to mention boom", err)
	}
}
