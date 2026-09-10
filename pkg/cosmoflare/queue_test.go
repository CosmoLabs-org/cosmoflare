package cosmoflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func queueMockSetup(handler http.HandlerFunc) (*QueueService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewQueueService(cf, "account-test-123")
	return svc, server
}

func queueWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- Constructor validation ---

func TestNewQueueServiceValidation(t *testing.T) {
	_, err := NewQueueService(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewQueueService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}
}

func TestNewQueueServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewQueueService(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

func TestNewQueueServiceFromCredsValidation(t *testing.T) {
	_, err := NewQueueServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewQueueServiceFromCreds("account123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
}

func TestNewQueueServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewQueueServiceFromCreds("account123", "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

// --- Method validation ---

func TestQueueGetValidation(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Get(context.Background(), "")
	if err == nil {
		t.Error("expected error when queue name is empty")
	}
}

func TestQueueCreateValidation(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Create(context.Background(), "")
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestQueueUpdateValidation(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Update(context.Background(), "", "new-name")
	if err == nil {
		t.Error("expected error when queue name is empty")
	}

	_, err = svc.Update(context.Background(), "old-name", "")
	if err == nil {
		t.Error("expected error when new name is empty")
	}
}

func TestQueueDeleteValidation(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	err := svc.Delete(context.Background(), "")
	if err == nil {
		t.Error("expected error when queue name is empty")
	}
}

func TestQueueListConsumersValidation(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.ListConsumers(context.Background(), "")
	if err == nil {
		t.Error("expected error when queue name is empty")
	}
}

func TestQueueCreateConsumerValidation(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.CreateConsumer(context.Background(), "", QueueConsumer{ScriptName: "worker"})
	if err == nil {
		t.Error("expected error when queue name is empty")
	}

	_, err = svc.CreateConsumer(context.Background(), "my-queue", QueueConsumer{})
	if err == nil {
		t.Error("expected error when script name is empty")
	}
}

func TestQueueDeleteConsumerValidation(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	err := svc.DeleteConsumer(context.Background(), "", "consumer")
	if err == nil {
		t.Error("expected error when queue name is empty")
	}

	err = svc.DeleteConsumer(context.Background(), "my-queue", "")
	if err == nil {
		t.Error("expected error when consumer name is empty")
	}
}

// --- API success tests ---

func TestQueueCreateSuccess(t *testing.T) {
	now := time.Now().UTC()
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		queueWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"queue_id":   "q-uuid-123",
				"queue_name": "my-queue",
				"created_on": now.Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	q, err := svc.Create(context.Background(), "my-queue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.ID != "q-uuid-123" {
		t.Errorf("expected ID=q-uuid-123, got %s", q.ID)
	}
	if q.Name != "my-queue" {
		t.Errorf("expected Name=my-queue, got %s", q.Name)
	}
}

func TestQueueCreateError(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		queueWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "internal error"}},
		})
	})
	defer server.Close()

	_, err := svc.Create(context.Background(), "my-queue")
	if err == nil {
		t.Error("expected error on server failure")
	}
}

func TestQueueListSuccess(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {
		queueWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"queue_id": "q1", "queue_name": "first", "producers_total_count": 1, "consumers_total_count": 2},
				{"queue_id": "q2", "queue_name": "second", "producers_total_count": 0, "consumers_total_count": 1},
			},
			"result_info": map[string]interface{}{
				"page": 1, "per_page": 50, "total_count": 2, "count": 2,
			},
		})
	})
	defer server.Close()

	queues, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queues) != 2 {
		t.Fatalf("expected 2 queues, got %d", len(queues))
	}
	if queues[0].ID != "q1" || queues[0].Name != "first" {
		t.Errorf("first queue mismatch: %+v", queues[0])
	}
	if queues[1].ConsumersTotalCount != 1 {
		t.Errorf("expected ConsumersTotalCount=1, got %d", queues[1].ConsumersTotalCount)
	}
}

func TestQueueListError(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		queueWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "forbidden"}},
		})
	})
	defer server.Close()

	_, err := svc.List(context.Background())
	if err == nil {
		t.Error("expected error on forbidden")
	}
}

func TestQueueGetSuccess(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {
		queueWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"queue_id":              "q-get-123",
				"queue_name":            "test-queue",
				"producers_total_count": 3,
				"consumers_total_count": 2,
			},
		})
	})
	defer server.Close()

	q, err := svc.Get(context.Background(), "test-queue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.ID != "q-get-123" {
		t.Errorf("expected ID=q-get-123, got %s", q.ID)
	}
	if q.ProducersTotalCount != 3 {
		t.Errorf("expected ProducersTotalCount=3, got %d", q.ProducersTotalCount)
	}
}

func TestQueueDeleteSuccess(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		queueWriteJSON(w, map[string]interface{}{"success": true, "errors": []interface{}{}})
	})
	defer server.Close()

	err := svc.Delete(context.Background(), "my-queue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueueDeleteError(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		queueWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "not found"}},
		})
	})
	defer server.Close()

	err := svc.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error on not found")
	}
}

func TestQueueUpdateSuccess(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		queueWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"queue_id":   "q-update-123",
				"queue_name": "new-name",
			},
		})
	})
	defer server.Close()

	q, err := svc.Update(context.Background(), "old-name", "new-name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Name != "new-name" {
		t.Errorf("expected Name=new-name, got %s", q.Name)
	}
}

func TestQueueListConsumersSuccess(t *testing.T) {
	svc, server := queueMockSetup(func(w http.ResponseWriter, r *http.Request) {
		queueWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"script_name": "worker-1",
					"environment": "production",
					"queue_name":  "my-queue",
					"settings": map[string]interface{}{
						"batch_size":       10,
						"max_retries":      3,
						"max_wait_time_ms": 5000,
					},
				},
			},
			"result_info": map[string]interface{}{
				"page": 1, "per_page": 50, "total_count": 1, "count": 1,
			},
		})
	})
	defer server.Close()

	consumers, err := svc.ListConsumers(context.Background(), "my-queue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(consumers) != 1 {
		t.Fatalf("expected 1 consumer, got %d", len(consumers))
	}
	if consumers[0].ScriptName != "worker-1" {
		t.Errorf("expected ScriptName=worker-1, got %s", consumers[0].ScriptName)
	}
	if consumers[0].Settings.BatchSize != 10 {
		t.Errorf("expected BatchSize=10, got %d", consumers[0].Settings.BatchSize)
	}
	if consumers[0].Settings.MaxRetries != 3 {
		t.Errorf("expected MaxRetries=3, got %d", consumers[0].Settings.MaxRetries)
	}
}

// --- Mapper tests ---

func TestMapQueue(t *testing.T) {
	now := time.Now().UTC()
	q := mapQueue(cloudflare.Queue{
		ID:                  "test-queue-id",
		Name:                "test-queue",
		CreatedOn:           &now,
		ProducersTotalCount: 2,
		ConsumersTotalCount: 1,
		Producers: []cloudflare.QueueProducer{
			{Service: "worker-a", Environment: "production"},
		},
		Consumers: []cloudflare.QueueConsumer{
			{ScriptName: "consumer-worker", Environment: "staging"},
		},
	})

	if q.ID != "test-queue-id" {
		t.Errorf("expected ID=test-queue-id, got %s", q.ID)
	}
	if q.ProducersTotalCount != 2 {
		t.Errorf("expected ProducersTotalCount=2, got %d", q.ProducersTotalCount)
	}
	if len(q.Producers) != 1 || q.Producers[0].Service != "worker-a" {
		t.Errorf("producers mismatch: %+v", q.Producers)
	}
	if len(q.Consumers) != 1 || q.Consumers[0].ScriptName != "consumer-worker" {
		t.Errorf("consumers mismatch: %+v", q.Consumers)
	}
	if q.CreatedOn == nil || !q.CreatedOn.Equal(now) {
		t.Error("CreatedOn mismatch")
	}
}

func TestMapQueueConsumer(t *testing.T) {
	now := time.Now().UTC()
	c := mapQueueConsumer(cloudflare.QueueConsumer{
		Name:            "consumer-1",
		Service:         "svc",
		ScriptName:      "my-worker",
		Environment:     "production",
		QueueName:       "my-queue",
		CreatedOn:       &now,
		DeadLetterQueue: "dlq",
		Settings: cloudflare.QueueConsumerSettings{
			BatchSize:   5,
			MaxRetires:  2,
			MaxWaitTime: 1000,
		},
	})

	if c.Name != "consumer-1" {
		t.Errorf("expected Name=consumer-1, got %s", c.Name)
	}
	if c.ScriptName != "my-worker" {
		t.Errorf("expected ScriptName=my-worker, got %s", c.ScriptName)
	}
	if c.Settings.BatchSize != 5 {
		t.Errorf("expected BatchSize=5, got %d", c.Settings.BatchSize)
	}
	if c.Settings.MaxRetries != 2 {
		t.Errorf("expected MaxRetries=2, got %d", c.Settings.MaxRetries)
	}
	if c.Settings.MaxWaitTime != 1000 {
		t.Errorf("expected MaxWaitTime=1000, got %d", c.Settings.MaxWaitTime)
	}
	if c.DeadLetterQueue != "dlq" {
		t.Errorf("expected DeadLetterQueue=dlq, got %s", c.DeadLetterQueue)
	}
}

func TestMapQueueConsumerSettingsToSDK(t *testing.T) {
	s := mapQueueConsumerSettingsToSDK(QueueConsumerSettings{
		BatchSize:   10,
		MaxRetries:  3,
		MaxWaitTime: 5000,
	})

	if s.BatchSize != 10 {
		t.Errorf("expected BatchSize=10, got %d", s.BatchSize)
	}
	if s.MaxRetires != 3 {
		t.Errorf("expected MaxRetires=3, got %d", s.MaxRetires)
	}
	if s.MaxWaitTime != 5000 {
		t.Errorf("expected MaxWaitTime=5000, got %d", s.MaxWaitTime)
	}
}

// --- Send / SendBatch (producer surface) ---

// queueProducerMockSetup returns a service backed by a mock server that
// resolves queue "my-queue" to ID "q-send-123" and records POSTs to the
// messages API.
func queueProducerMockSetup(t *testing.T, messagesHandler func(w http.ResponseWriter, r *http.Request)) (*QueueService, *[]byte, *httptest.Server) {
	t.Helper()
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account-test-123/workers/queues/my-queue":
			queueWriteJSON(w, map[string]interface{}{
				"success": true,
				"errors":  []interface{}{},
				"result": map[string]interface{}{
					"queue_id":   "q-send-123",
					"queue_name": "my-queue",
				},
			})
		case r.Method == http.MethodPost && (r.URL.Path == "/accounts/account-test-123/queues/q-send-123/messages" || r.URL.Path == "/accounts/account-test-123/queues/q-send-123/messages/batch"):
			data, _ := io.ReadAll(r.Body)
			capturedBody = data
			messagesHandler(w, r)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			queueWriteJSON(w, map[string]interface{}{"success": false, "errors": []interface{}{}})
		}
	}))
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewQueueService(cf, "account-test-123")
	return svc, &capturedBody, server
}

func TestQueueServiceSendValidation(t *testing.T) {
	svc, _, server := queueProducerMockSetup(t, func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Send(context.Background(), "", QueueMessage{Body: "hi"})
	if err == nil {
		t.Error("expected error when queue name is empty")
	}
}

func TestQueueServiceSendSuccess(t *testing.T) {
	svc, body, server := queueProducerMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		queueWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"message_id": "m-123"},
		})
	})
	defer server.Close()

	res, err := svc.Send(context.Background(), "my-queue", QueueMessage{
		Body:         `{"hello":"world"}`,
		ContentType:  "application/json",
		DelaySeconds: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Error("expected Success=true")
	}
	if res.MessageID != "m-123" {
		t.Errorf("expected MessageID=m-123, got %q", res.MessageID)
	}

	var sent struct {
		Body         string `json:"body"`
		ContentType  string `json:"content_type"`
		DelaySeconds int    `json:"delay_seconds"`
	}
	if err := json.Unmarshal(*body, &sent); err != nil {
		t.Fatalf("failed to decode sent body: %v", err)
	}
	if sent.Body != `{"hello":"world"}` {
		t.Errorf("body mismatch: %+v", sent)
	}
	if sent.ContentType != "application/json" {
		t.Errorf("content_type mismatch: %+v", sent)
	}
	if sent.DelaySeconds != 5 {
		t.Errorf("delay_seconds mismatch: %+v", sent)
	}
}

func TestQueueServiceSendOmitOptionalFields(t *testing.T) {
	svc, body, server := queueProducerMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		queueWriteJSON(w, map[string]interface{}{"success": true, "errors": []interface{}{}})
	})
	defer server.Close()

	if _, err := svc.Send(context.Background(), "my-queue", QueueMessage{Body: "plain"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(*body, &raw); err != nil {
		t.Fatalf("failed to decode sent body: %v", err)
	}
	if _, ok := raw["content_type"]; ok {
		t.Error("content_type should be omitted when empty")
	}
	if _, ok := raw["delay_seconds"]; ok {
		t.Error("delay_seconds should be omitted when zero")
	}
}

func TestQueueServiceSendError(t *testing.T) {
	svc, _, server := queueProducerMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		queueWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "internal error"}},
		})
	})
	defer server.Close()

	_, err := svc.Send(context.Background(), "my-queue", QueueMessage{Body: "hi"})
	if err == nil {
		t.Error("expected error on server failure")
	}
}

func TestQueueServiceSendUnknownQueue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		queueWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "queue not found"}},
		})
	}))
	defer server.Close()
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewQueueService(cf, "account-test-123")

	_, err := svc.Send(context.Background(), "missing-queue", QueueMessage{Body: "hi"})
	if err == nil {
		t.Error("expected error when queue does not exist")
	}
}

func TestQueueServiceSendOversizedWarning(t *testing.T) {
	svc, _, server := queueProducerMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		queueWriteJSON(w, map[string]interface{}{"success": true, "errors": []interface{}{}})
	})
	defer server.Close()

	big := strings.Repeat("a", maxQueueMessageBytes)
	stderr := captureStderr(t)
	defer stderr.restore()

	res, err := svc.Send(context.Background(), "my-queue", QueueMessage{Body: big})
	if err != nil {
		t.Fatalf("oversized message should still be sent (soft warning), got error: %v", err)
	}
	if !res.Success {
		t.Error("expected Success=true despite warning")
	}
	out := stderr.output()
	if !strings.Contains(out, "128,000") {
		t.Errorf("warning should name the 128,000-byte limit, got: %q", out)
	}
	if !strings.Contains(out, fmt.Sprintf("%d", maxQueueMessageBytes+100)) {
		t.Errorf("warning should name the actual size, got: %q", out)
	}
}

func TestQueueServiceSendBatchValidation(t *testing.T) {
	svc, _, server := queueProducerMockSetup(t, func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.SendBatch(context.Background(), "", []QueueMessage{{Body: "hi"}})
	if err == nil {
		t.Error("expected error when queue name is empty")
	}

	_, err = svc.SendBatch(context.Background(), "my-queue", nil)
	if err == nil {
		t.Error("expected error when message list is empty")
	}

	tooMany := make([]QueueMessage, 101)
	for i := range tooMany {
		tooMany[i] = QueueMessage{Body: "m"}
	}
	_, err = svc.SendBatch(context.Background(), "my-queue", tooMany)
	if err == nil {
		t.Fatal("expected error when batch exceeds 100 messages")
	}
	msg := err.Error()
	if !strings.Contains(msg, "100") {
		t.Errorf("error should state the cap: %q", msg)
	}
	if !strings.Contains(msg, "split") {
		t.Errorf("error should explain how to split: %q", msg)
	}
}

func TestQueueServiceSendBatchSuccess(t *testing.T) {
	svc, body, server := queueProducerMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		queueWriteJSON(w, map[string]interface{}{"success": true, "errors": []interface{}{}})
	})
	defer server.Close()

	msgs := []QueueMessage{
		{Body: "one"},
		{Body: "two", ContentType: "text/plain", DelaySeconds: 3},
	}
	res, err := svc.SendBatch(context.Background(), "my-queue", msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Error("expected Success=true")
	}
	if res.Sent != 2 {
		t.Errorf("expected Sent=2, got %d", res.Sent)
	}

	var sent struct {
		Messages []map[string]interface{} `json:"messages"`
	}
	if err := json.Unmarshal(*body, &sent); err != nil {
		t.Fatalf("failed to decode sent body: %v", err)
	}
	if len(sent.Messages) != 2 {
		t.Fatalf("expected 2 messages in payload, got %d", len(sent.Messages))
	}
	if sent.Messages[0]["body"] != "one" {
		t.Errorf("first message mismatch: %+v", sent.Messages[0])
	}
	if sent.Messages[1]["content_type"] != "text/plain" {
		t.Errorf("second message content_type mismatch: %+v", sent.Messages[1])
	}
}

func TestQueueServiceSendBatchError(t *testing.T) {
	svc, _, server := queueProducerMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		queueWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "bad request"}},
		})
	})
	defer server.Close()

	_, err := svc.SendBatch(context.Background(), "my-queue", []QueueMessage{{Body: "hi"}})
	if err == nil {
		t.Error("expected error on bad request")
	}
}

func TestQueueServiceSendBatchOversizedWarning(t *testing.T) {
	svc, _, server := queueProducerMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		queueWriteJSON(w, map[string]interface{}{"success": true, "errors": []interface{}{}})
	})
	defer server.Close()

	msgs := []QueueMessage{
		{Body: "small"},
		{Body: strings.Repeat("b", maxQueueMessageBytes)},
	}
	stderr := captureStderr(t)
	defer stderr.restore()

	if _, err := svc.SendBatch(context.Background(), "my-queue", msgs); err != nil {
		t.Fatalf("oversized message should still be sent (soft warning), got error: %v", err)
	}
	if out := stderr.output(); !strings.Contains(out, "128,000") {
		t.Errorf("warning should name the 128,000-byte limit, got: %q", out)
	}
}

// captureStderr redirects os.Stderr to a pipe for the duration of a test.
type stderrCapture struct {
	old   *os.File
	read  *os.File
	write *os.File
}

func captureStderr(t *testing.T) *stderrCapture {
	t.Helper()
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	return &stderrCapture{old: old, read: r, write: w}
}

func (c *stderrCapture) restore() {
	os.Stderr = c.old
}

func (c *stderrCapture) output() string {
	c.write.Close()
	var buf bytes.Buffer
	buf.ReadFrom(c.read)
	return buf.String()
}
