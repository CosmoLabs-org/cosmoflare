package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// waitingRoomFixture mirrors the Cloudflare waiting-room JSON shape used by
// cloudflare-go's WaitingRoomDetailResponse.
func waitingRoomFixture(id, name string) map[string]interface{} {
	return map[string]interface{}{
		"id":                    id,
		"name":                  name,
		"host":                  "example.com",
		"path":                  "/",
		"queueing_method":       "fifo",
		"new_users_per_minute":  200,
		"total_active_users":    500,
		"session_duration":      1,
		"queue_all":             false,
		"suspended":             false,
		"json_response_enabled": false,
		"queueing_status_code":  429,
		"cookie_suffix":         "abc123",
		"created_on":            "2024-01-02T03:04:05Z",
		"modified_on":           "2024-06-07T08:09:10Z",
	}
}

func waitingRoomWriteJSON(t *testing.T, w http.ResponseWriter, room map[string]interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"errors":  []interface{}{},
		"result":  room,
	})
}

// TestWaitingRoomService_List verifies List issues a GET on the zone's
// waiting_rooms endpoint and maps every returned room.
func TestWaitingRoomService_List(t *testing.T) {
	t.Parallel()
	const zoneID = "zone-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/zones/"+zoneID+"/waiting_rooms"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				waitingRoomFixture("room-1", "launch"),
				waitingRoomFixture("room-2", "sale"),
			},
		})
	}))
	defer server.Close()

	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, err := NewWaitingRoomService(cf, zoneID)
	if err != nil {
		t.Fatalf("service: %v", err)
	}

	rooms, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("expected 2 rooms, got %d", len(rooms))
	}
	first := rooms[0]
	if first.ID != "room-1" || first.Name != "launch" || first.Host != "example.com" {
		t.Errorf("unexpected mapping: %+v", first)
	}
	if first.NewUsersPerMinute != 200 || first.TotalActiveUsers != 500 || first.SessionDuration != 1 {
		t.Errorf("unexpected capacity mapping: %+v", first)
	}
	if first.CreatedOn == "" || first.ModifiedOn == "" {
		t.Errorf("expected timestamps rendered, got %+v", first)
	}
	if first.CookieSuffix != "abc123" {
		t.Errorf("expected cookie suffix mapped, got %q", first.CookieSuffix)
	}
}

// TestWaitingRoomService_Create verifies Create POSTs the required fields
// to the zone endpoint and maps the assigned ID.
func TestWaitingRoomService_Create(t *testing.T) {
	t.Parallel()
	const zoneID = "zone-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/zones/"+zoneID+"/waiting_rooms"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		for _, key := range []string{"name", "host", "new_users_per_minute", "total_active_users"} {
			if _, ok := body[key]; !ok {
				t.Errorf("request body missing %q: %v", key, body)
			}
		}
		if body["name"] != "launch" || body["host"] != "example.com" {
			t.Errorf("unexpected name/host: %v", body)
		}
		waitingRoomWriteJSON(t, w, waitingRoomFixture("room-new", "launch"))
	}))
	defer server.Close()

	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewWaitingRoomService(cf, zoneID)

	room, err := svc.Create(context.Background(), WaitingRoomCreate{
		Name:               "launch",
		Host:               "example.com",
		Path:               "/",
		NewUsersPerMinute:  200,
		TotalActiveUsers:   500,
		SessionDuration:    1,
		QueueingStatusCode: 429,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.ID != "room-new" || room.Name != "launch" {
		t.Errorf("unexpected mapped room: %+v", room)
	}
}

// TestWaitingRoomService_Create_Validation verifies Create rejects missing
// required fields before any network call.
func TestWaitingRoomService_Create_Validation(t *testing.T) {
	t.Parallel()
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL("http://127.0.0.1:1"))
	svc, _ := NewWaitingRoomService(cf, "zone-123")

	cases := []struct {
		name string
		opts WaitingRoomCreate
		want string
	}{
		{"missing name", WaitingRoomCreate{Host: "h", NewUsersPerMinute: 1, TotalActiveUsers: 1}, "name is required"},
		{"missing host", WaitingRoomCreate{Name: "n", NewUsersPerMinute: 1, TotalActiveUsers: 1}, "host is required"},
		{"missing users/min", WaitingRoomCreate{Name: "n", Host: "h", TotalActiveUsers: 1}, "new users per minute"},
		{"missing active users", WaitingRoomCreate{Name: "n", Host: "h", NewUsersPerMinute: 1}, "total active users"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

// TestWaitingRoomService_Get verifies Get targets the single-room endpoint
// and maps the response.
func TestWaitingRoomService_Get(t *testing.T) {
	t.Parallel()
	const zoneID = "zone-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/zones/"+zoneID+"/waiting_rooms/room-9"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		waitingRoomWriteJSON(t, w, waitingRoomFixture("room-9", "launch"))
	}))
	defer server.Close()

	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewWaitingRoomService(cf, zoneID)

	room, err := svc.Get(context.Background(), "room-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.ID != "room-9" || room.Host != "example.com" {
		t.Errorf("unexpected room: %+v", room)
	}
	if _, err := svc.Get(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "waiting room ID is required") {
		t.Fatalf("expected empty-ID validation error, got %v", err)
	}
}

// TestWaitingRoomService_Update verifies Update merges the requested
// fields onto the room's live state: the re-fetch after the PATCH must
// observe the WRITTEN state (stateful-stub lesson from logpush).
func TestWaitingRoomService_Update(t *testing.T) {
	t.Parallel()
	const zoneID = "zone-123"

	current := waitingRoomFixture("room-9", "launch")
	var patchBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			waitingRoomWriteJSON(t, w, current)
		case http.MethodPatch:
			if err := json.NewDecoder(r.Body).Decode(&patchBody); err != nil {
				t.Fatalf("decode patch: %v", err)
			}
			// Apply the patch to the served state so the re-fetch (if any)
			// and the PATCH response agree.
			for k, v := range patchBody {
				current[k] = v
			}
			waitingRoomWriteJSON(t, w, current)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewWaitingRoomService(cf, zoneID)

	newTotal := 750
	room, err := svc.Update(context.Background(), "room-9", WaitingRoomUpdate{TotalActiveUsers: &newTotal})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.TotalActiveUsers != 750 {
		t.Errorf("expected total active users 750, got %d", room.TotalActiveUsers)
	}
	// Untouched required fields must ride along at their live values.
	if patchBody["host"] != "example.com" || patchBody["name"] != "launch" {
		t.Errorf("patch dropped live required fields: %v", patchBody)
	}
	if patchBody["new_users_per_minute"].(float64) != 200 {
		t.Errorf("patch changed new_users_per_minute: %v", patchBody)
	}
}

// TestWaitingRoomService_Delete verifies Delete issues a DELETE on the
// single-room endpoint.
func TestWaitingRoomService_Delete(t *testing.T) {
	t.Parallel()
	const zoneID = "zone-123"

	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/zones/"+zoneID+"/waiting_rooms/room-9"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		deleted = true
		waitingRoomWriteJSON(t, w, waitingRoomFixture("room-9", "launch"))
	}))
	defer server.Close()

	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewWaitingRoomService(cf, zoneID)

	if err := svc.Delete(context.Background(), "room-9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected delete endpoint hit")
	}
}

// TestWaitingRoomService_APIErrorWrapped verifies API failures surface as
// wrapped service errors.
func TestWaitingRoomService_APIErrorWrapped(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 9103, "message": "Unauthorized to access requested resource"}},
		})
	}))
	defer server.Close()

	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewWaitingRoomService(cf, "zone-123")

	if _, err := svc.List(context.Background()); err == nil || !strings.Contains(err.Error(), "failed to list waiting rooms") {
		t.Fatalf("expected wrapped list error, got %v", err)
	}
}

// TestNewWaitingRoomService_Validation verifies the constructors reject a
// nil client and an empty zone ID.
func TestNewWaitingRoomService_Validation(t *testing.T) {
	if _, err := NewWaitingRoomService(nil, "zone-123"); err == nil {
		t.Error("expected nil-client error")
	}
	cf, _ := cloudflare.NewWithAPIToken("t", cloudflare.BaseURL("http://127.0.0.1:1"))
	if _, err := NewWaitingRoomService(cf, ""); err == nil {
		t.Error("expected empty-zone error")
	}
	if _, err := NewWaitingRoomServiceFromCreds("", "tok"); err == nil {
		t.Error("expected empty-zone error from creds")
	}
	if _, err := NewWaitingRoomServiceFromCreds("zone-123", ""); err == nil {
		t.Error("expected empty-token error from creds")
	}
}
