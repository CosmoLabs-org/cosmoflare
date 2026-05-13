package r2go2

import (
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func TestNewWorkerServiceValidation(t *testing.T) {
	_, err := NewWorkerService(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewWorkerService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewWorkerService(nil, "")
	if err == nil {
		t.Error("expected error when both API client and accountID are empty")
	}
}

func TestNewWorkerServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewWorkerService(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestWorkerOptions(t *testing.T) {
	cfg := &workerConfig{}
	WithWorkerCompatibilityDate("2024-01-01")(cfg)
	if cfg.compatibilityDate != "2024-01-01" {
		t.Errorf("expected compatibilityDate=2024-01-01, got %s", cfg.compatibilityDate)
	}

	bindings := []WorkerBinding{
		{Name: "MY_KV", Type: "kv", ID: "ns-123"},
		{Name: "MY_R2", Type: "r2", ID: "my-bucket"},
	}
	WithWorkerBindings(bindings)(cfg)
	if len(cfg.bindings) != 2 {
		t.Errorf("expected 2 bindings, got %d", len(cfg.bindings))
	}
	if cfg.bindings[0].Name != "MY_KV" {
		t.Errorf("expected first binding name=MY_KV, got %s", cfg.bindings[0].Name)
	}

	tags := []string{"production", "v2"}
	WithWorkerTags(tags)(cfg)
	if len(cfg.tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(cfg.tags))
	}

	WithWorkerModule(true)(cfg)
	if !cfg.module {
		t.Error("expected module=true")
	}
}

func TestLogOptions(t *testing.T) {
	cfg := &logConfig{}
	WithLogLimit(50)(cfg)
	if cfg.limit != 50 {
		t.Errorf("expected limit=50, got %d", cfg.limit)
	}

	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	WithLogSince(since)(cfg)
	if !cfg.since.Equal(since) {
		t.Errorf("expected since=%v, got %v", since, cfg.since)
	}
}

func TestWorkerTypes(t *testing.T) {
	w := &Worker{
		Name:        "my-worker",
		Modified:    time.Now(),
		Size:        1024,
		Runtime:     "workers",
		Script:      "export default {}",
		Bindings:    []WorkerBinding{{Name: "KV", Type: "kv", ID: "ns-1"}},
		Tags:        []string{"prod"},
		CompatibilityDate: "2024-01-01",
	}
	if w.Name != "my-worker" {
		t.Errorf("unexpected name: %s", w.Name)
	}
	if w.Size != 1024 {
		t.Errorf("unexpected size: %d", w.Size)
	}
	if len(w.Bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(w.Bindings))
	}
	if len(w.Tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(w.Tags))
	}
}

func TestWorkerSettingsType(t *testing.T) {
	s := WorkerSettings{
		CompatibilityDate: "2024-01-01",
		UsageModel:        "bundled",
		Bindings:          []WorkerBinding{{Name: "D1", Type: "d1", ID: "db-1"}},
	}
	if s.CompatibilityDate != "2024-01-01" {
		t.Errorf("unexpected compatibility date: %s", s.CompatibilityDate)
	}
	if s.UsageModel != "bundled" {
		t.Errorf("unexpected usage model: %s", s.UsageModel)
	}
	if len(s.Bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(s.Bindings))
	}
}

func TestLogEntryType(t *testing.T) {
	e := &LogEntry{
		Timestamp: time.Now(),
		Level:     "error",
		Message:   "uncaught exception",
		Event:     "exception",
	}
	if e.Level != "error" {
		t.Errorf("unexpected level: %s", e.Level)
	}
}

func TestToCFBindings(t *testing.T) {
	bindings := []WorkerBinding{
		{Name: "MY_KV", Type: "kv", ID: "ns-123"},
		{Name: "MY_R2", Type: "r2", ID: "my-bucket"},
		{Name: "MY_D1", Type: "d1", ID: "db-456"},
		{Name: "MY_QUEUE", Type: "queue", ID: "my-queue"},
		{Name: "MY_SERVICE", Type: "service", ID: "other-worker"},
		{Name: "MY_VAR", Type: "plain_text", ID: "hello"},
		{Name: "MY_SECRET", Type: "secret_text", ID: "s3cret"},
	}

	result := toCFBindings(bindings)
	if len(result) != 7 {
		t.Fatalf("expected 7 bindings, got %d", len(result))
	}

	for name, binding := range result {
		switch name {
		case "MY_KV":
			if _, ok := binding.(cloudflare.WorkerKvNamespaceBinding); !ok {
				t.Errorf("expected WorkerKvNamespaceBinding for %s", name)
			}
		case "MY_R2":
			if _, ok := binding.(cloudflare.WorkerR2BucketBinding); !ok {
				t.Errorf("expected WorkerR2BucketBinding for %s", name)
			}
		case "MY_D1":
			if _, ok := binding.(cloudflare.WorkerD1DatabaseBinding); !ok {
				t.Errorf("expected WorkerD1DatabaseBinding for %s", name)
			}
		case "MY_QUEUE":
			if _, ok := binding.(cloudflare.WorkerQueueBinding); !ok {
				t.Errorf("expected WorkerQueueBinding for %s", name)
			}
		case "MY_SERVICE":
			if _, ok := binding.(cloudflare.WorkerServiceBinding); !ok {
				t.Errorf("expected WorkerServiceBinding for %s", name)
			}
		case "MY_VAR":
			if _, ok := binding.(cloudflare.WorkerPlainTextBinding); !ok {
				t.Errorf("expected WorkerPlainTextBinding for %s", name)
			}
		case "MY_SECRET":
			if _, ok := binding.(cloudflare.WorkerSecretTextBinding); !ok {
				t.Errorf("expected WorkerSecretTextBinding for %s", name)
			}
		}
	}
}

func TestToCFBindingsEmpty(t *testing.T) {
	result := toCFBindings(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 bindings for nil input, got %d", len(result))
	}
}

func TestWorkerDeployValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.Deploy(nil, "", nil)
	if err == nil {
		t.Error("expected error when name is empty")
	}
	if !strings.Contains(err.Error(), "worker name is required") {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = svc.Deploy(nil, "my-worker", nil)
	if err == nil {
		t.Error("expected error when script is nil")
	}
	if !strings.Contains(err.Error(), "script content is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorkerGetValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.Get(nil, "")
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestWorkerDeleteValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	err := svc.Delete(nil, "")
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestWorkerLogsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.Logs(nil, "")
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestWorkerUpdateSettingsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	err := svc.UpdateSettings(nil, "", WorkerSettings{})
	if err == nil {
		t.Error("expected error when name is empty")
	}
}
