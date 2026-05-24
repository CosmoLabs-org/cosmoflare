package r2go2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
						"batch_size":      10,
						"max_retries":     3,
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
		Name:        "consumer-1",
		Service:     "svc",
		ScriptName:  "my-worker",
		Environment: "production",
		QueueName:   "my-queue",
		CreatedOn:   &now,
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
