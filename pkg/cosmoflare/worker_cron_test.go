package cosmoflare

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// Reuses workerMockSetup / workerWriteJSON from worker_test.go (same package).

// TestWorkerCronListWithMock verifies CronList GETs the script schedules
// endpoint and maps the nested schedules array to the public view.
func TestWorkerCronListWithMock(t *testing.T) {
	var gotMethod, gotPath string
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"schedules": []map[string]interface{}{
					{"cron": "*/5 * * * *", "modified_on": "2026-01-02T03:04:05Z"},
					{"cron": "0 12 * * 1"},
				},
			},
		})
	})
	defer server.Close()

	triggers, err := svc.CronList(context.Background(), "my-worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/workers/scripts/my-worker/schedules") {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if len(triggers) != 2 {
		t.Fatalf("expected 2 triggers, got %d", len(triggers))
	}
	if triggers[0].Cron != "*/5 * * * *" || triggers[0].Modified != "2026-01-02T03:04:05Z" {
		t.Errorf("unexpected first trigger: %+v", triggers[0])
	}
	// Absent modified_on must map to the omitted empty string, not a
	// zero-time render.
	if triggers[1].Cron != "0 12 * * 1" || triggers[1].Modified != "" {
		t.Errorf("unexpected second trigger: %+v", triggers[1])
	}
}

// TestWorkerCronReplaceWithMock verifies CronReplace PUTs the full
// schedule set and returns the post-replace view.
func TestWorkerCronReplaceWithMock(t *testing.T) {
	var gotMethod, gotBody string
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		buf := make([]byte, 512)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"schedules": []map[string]interface{}{
					{"cron": "*/10 * * * *", "modified_on": "2026-02-03T04:05:06Z"},
				},
			},
		})
	})
	defer server.Close()

	triggers, err := svc.CronReplace(context.Background(), "my-worker", []string{"*/10 * * * *", "0 0 * * *"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("expected PUT, got %s", gotMethod)
	}
	// The update payload is a bare JSON array of {cron} objects.
	var sent []map[string]string
	if err := json.Unmarshal([]byte(gotBody), &sent); err != nil {
		t.Fatalf("failed to decode request body %q: %v", gotBody, err)
	}
	if len(sent) != 2 || sent[0]["cron"] != "*/10 * * * *" || sent[1]["cron"] != "0 0 * * *" {
		t.Errorf("unexpected request payload: %s", gotBody)
	}
	if len(triggers) != 1 || triggers[0].Cron != "*/10 * * * *" {
		t.Errorf("unexpected replace result: %+v", triggers)
	}
}

// TestWorkerCronCreateWithMock verifies create fetches the current set,
// appends the new expression, and replaces.
func TestWorkerCronCreateWithMock(t *testing.T) {
	existing := true
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			result := []map[string]interface{}{}
			if existing {
				result = append(result, map[string]interface{}{"cron": "*/5 * * * *"})
			}
			workerWriteJSON(w, map[string]interface{}{
				"success": true,
				"errors":  []interface{}{},
				"result":  map[string]interface{}{"schedules": result},
			})
			return
		}
		var sent []struct {
			Cron string `json:"cron"`
		}
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if existing {
			if len(sent) != 2 || sent[0].Cron != "*/5 * * * *" || sent[1].Cron != "0 12 * * 1" {
				t.Errorf("expected current + appended schedule, got %+v", sent)
			}
		} else if len(sent) != 1 || sent[0].Cron != "*/5 * * * *" {
			t.Errorf("expected single appended schedule into empty set, got %+v", sent)
		}
		schedules := []map[string]interface{}{}
		for _, c := range sent {
			schedules = append(schedules, map[string]interface{}{"cron": c.Cron})
		}
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"schedules": schedules},
		})
	})
	defer server.Close()

	triggers, err := svc.CronCreate(context.Background(), "my-worker", "0 12 * * 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(triggers) != 2 {
		t.Fatalf("expected 2 triggers after create, got %d", len(triggers))
	}

	// Flip the mock: no existing schedules, create into an empty set.
	existing = false
	triggers, err = svc.CronCreate(context.Background(), "my-worker", "*/5 * * * *")
	if err != nil {
		t.Fatalf("unexpected error creating into empty set: %v", err)
	}
	if len(triggers) != 1 || triggers[0].Cron != "*/5 * * * *" {
		t.Errorf("unexpected create result: %+v", triggers)
	}
}

// TestWorkerCronCreateDuplicate verifies a duplicate create is rejected
// without hitting the replace endpoint.
func TestWorkerCronCreateDuplicate(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("duplicate create must not PUT, got %s", r.Method)
		}
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"schedules": []map[string]interface{}{{"cron": "*/5 * * * *"}},
			},
		})
	})
	defer server.Close()

	_, err := svc.CronCreate(context.Background(), "my-worker", "*/5 * * * *")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

// TestWorkerCronDeleteWithMock verifies delete removes only the matching
// expression from the set before replacing.
func TestWorkerCronDeleteWithMock(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			workerWriteJSON(w, map[string]interface{}{
				"success": true,
				"errors":  []interface{}{},
				"result": map[string]interface{}{
					"schedules": []map[string]interface{}{
						{"cron": "*/5 * * * *"},
						{"cron": "0 12 * * 1"},
						{"cron": "30 3 * * 0"},
					},
				},
			})
			return
		}
		var sent []struct {
			Cron string `json:"cron"`
		}
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if len(sent) != 2 {
			t.Fatalf("expected 2 remaining schedules, got %d", len(sent))
		}
		for _, c := range sent {
			if c.Cron == "0 12 * * 1" {
				t.Errorf("deleted expression still present in replace payload: %+v", sent)
			}
		}
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"schedules": []map[string]interface{}{
					{"cron": "*/5 * * * *"},
					{"cron": "30 3 * * 0"},
				},
			},
		})
	})
	defer server.Close()

	triggers, err := svc.CronDelete(context.Background(), "my-worker", "0 12 * * 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(triggers) != 2 {
		t.Fatalf("expected 2 triggers after delete, got %d", len(triggers))
	}
}

// TestWorkerCronDeleteAbsent verifies deleting an expression that is not
// scheduled is an error, not a silent replace.
func TestWorkerCronDeleteAbsent(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("absent delete must not PUT, got %s", r.Method)
		}
		workerWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"schedules": []map[string]interface{}{{"cron": "*/5 * * * *"}},
			},
		})
	})
	defer server.Close()

	_, err := svc.CronDelete(context.Background(), "my-worker", "9 9 9 9 9")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got %v", err)
	}
}

// --- validation ---

func TestWorkerCronListValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.CronList(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker-name error, got %v", err)
	}
}

func TestWorkerCronReplaceValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.CronReplace(context.Background(), "", []string{"*/5 * * * *"})
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker-name error, got %v", err)
	}
}

func TestWorkerCronCreateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.CronCreate(context.Background(), "", "*/5 * * * *")
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker-name error, got %v", err)
	}

	_, err = svc.CronCreate(context.Background(), "my-worker", "")
	if err == nil || !strings.Contains(err.Error(), "cron expression is required") {
		t.Fatalf("expected cron-expression error, got %v", err)
	}
}

func TestWorkerCronDeleteValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewWorkerService(cf, "account123")

	_, err := svc.CronDelete(context.Background(), "", "*/5 * * * *")
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker-name error, got %v", err)
	}

	_, err = svc.CronDelete(context.Background(), "my-worker", "")
	if err == nil || !strings.Contains(err.Error(), "cron expression is required") {
		t.Fatalf("expected cron-expression error, got %v", err)
	}
}

// --- API error mapping ---

func TestWorkerCronListAPIError(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 10000, "message": "workers.api.error.script_not_found"}},
		})
	})
	defer server.Close()

	_, err := svc.CronList(context.Background(), "gone")
	if err == nil {
		t.Fatal("expected error from API")
	}
	var cerr *R2Error
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *R2Error, got %T", err)
	}
	if !strings.Contains(err.Error(), "failed to list cron triggers") {
		t.Errorf("expected wrapped message, got %v", err)
	}
}

func TestWorkerCronReplaceAPIError(t *testing.T) {
	svc, server := workerMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		workerWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 10021, "message": "validation failed: invalid cron expression"}},
		})
	})
	defer server.Close()

	_, err := svc.CronReplace(context.Background(), "my-worker", []string{"not a cron"})
	if err == nil {
		t.Fatal("expected error from API")
	}
	if !strings.Contains(err.Error(), "failed to replace cron triggers") {
		t.Errorf("expected wrapped message, got %v", err)
	}
}
