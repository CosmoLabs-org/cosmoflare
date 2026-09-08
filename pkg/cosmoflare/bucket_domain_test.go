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

// bucketDomainHandler captures the last request seen by the test server.
type bucketDomainHandler struct {
	method string
	path   string
	auth   string
	body   []byte
	inner  http.HandlerFunc
}

func (h *bucketDomainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func newBucketDomainTestServer(t *testing.T, h *bucketDomainHandler) *BucketDomainService {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewBucketDomainService("ACC", "tok-secret", WithBucketDomainBaseURL(srv.URL))
}

func TestBucketDomainAttach(t *testing.T) {
	h := &bucketDomainHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"domain":"cdn.example.com","enabled":true,"zoneId":"z1","zoneName":"example.com"}}`))
		},
	}
	svc := newBucketDomainTestServer(t, h)

	d, err := svc.Attach(context.Background(), "bk", AttachBucketDomainRequest{
		Domain:  "cdn.example.com",
		Enabled: true,
		ZoneID:  "z1",
		MinTLS:  "1.2",
	})
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	if h.method != http.MethodPost {
		t.Errorf("method = %s, want POST", h.method)
	}
	if h.path != "/accounts/ACC/r2/buckets/bk/domains/custom" {
		t.Errorf("path = %s", h.path)
	}
	if h.auth != "Bearer tok-secret" {
		t.Errorf("Authorization = %q", h.auth)
	}
	var sent map[string]interface{}
	if err := json.Unmarshal(h.body, &sent); err != nil {
		t.Fatalf("body not JSON: %v (%s)", err, h.body)
	}
	if sent["domain"] != "cdn.example.com" || sent["zoneId"] != "z1" || sent["minTLS"] != "1.2" {
		t.Errorf("body = %s", h.body)
	}
	if sent["enabled"] != true {
		t.Errorf("enabled should be true in body, got %v", sent["enabled"])
	}
	if d.Domain != "cdn.example.com" || !d.Enabled || d.ZoneID != "z1" {
		t.Errorf("parsed domain = %+v", d)
	}
}

func TestBucketDomainAttachValidationError(t *testing.T) {
	h := &bucketDomainHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			t.Error("server must not be hit on validation failure")
			w.WriteHeader(http.StatusBadRequest)
		},
	}
	svc := newBucketDomainTestServer(t, h)

	if _, err := svc.Attach(context.Background(), "bk", AttachBucketDomainRequest{ZoneID: "z1"}); err == nil {
		t.Error("empty domain should be a validation error")
	}
	if _, err := svc.Attach(context.Background(), "bk", AttachBucketDomainRequest{Domain: "a.com"}); err == nil {
		t.Error("empty zone id should be a validation error")
	}
	if _, err := svc.Attach(context.Background(), "bk", AttachBucketDomainRequest{Domain: "a.com", ZoneID: "z", MinTLS: "0.9"}); err == nil {
		t.Error("bad minTLS should be a validation error")
	}
}

func TestBucketDomainList(t *testing.T) {
	h := &bucketDomainHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"domains":[{"domain":"a.example.com","enabled":true,"status":{"ownership":"active","ssl":"active"}},{"domain":"b.example.com","enabled":false,"status":{"ownership":"pending","ssl":"pending"}}]}}`))
		},
	}
	svc := newBucketDomainTestServer(t, h)

	domains, err := svc.List(context.Background(), "bk")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if h.method != http.MethodGet {
		t.Errorf("method = %s, want GET", h.method)
	}
	if len(domains) != 2 {
		t.Fatalf("len = %d, want 2", len(domains))
	}
	if domains[0].Status == nil || domains[0].Status.Ownership != "active" || domains[0].Status.SSL != "active" {
		t.Errorf("domains[0].Status = %+v", domains[0].Status)
	}
	if domains[1].Status == nil || domains[1].Status.Ownership != "pending" {
		t.Errorf("domains[1].Status = %+v", domains[1].Status)
	}

	// empty list
	h2 := &bucketDomainHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"domains":[]}}`))
		},
	}
	svc2 := newBucketDomainTestServer(t, h2)
	domains2, err := svc2.List(context.Background(), "bk")
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if domains2 == nil {
		t.Error("empty list should be non-nil empty slice")
	}
	if len(domains2) != 0 {
		t.Errorf("len = %d, want 0", len(domains2))
	}
}

func TestBucketDomainGet(t *testing.T) {
	h := &bucketDomainHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"domain":"cdn.example.com","enabled":true,"status":{"ownership":"pending","ssl":"active"}}}`))
		},
	}
	svc := newBucketDomainTestServer(t, h)

	d, err := svc.Get(context.Background(), "bk", "cdn.example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if h.path != "/accounts/ACC/r2/buckets/bk/domains/custom/cdn.example.com" {
		t.Errorf("path = %s", h.path)
	}
	if d.Status == nil || d.Status.Ownership != "pending" || d.Status.SSL != "active" {
		t.Errorf("Status = %+v", d.Status)
	}
}

func TestBucketDomainUpdate(t *testing.T) {
	h := &bucketDomainHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"domain":"cdn.example.com","enabled":false}}`))
		},
	}
	svc := newBucketDomainTestServer(t, h)

	disabled := false
	tls := "1.3"
	d, err := svc.Update(context.Background(), "bk", "cdn.example.com", UpdateBucketDomainRequest{
		Enabled: &disabled,
		MinTLS:  &tls,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if h.method != http.MethodPut {
		t.Errorf("method = %s, want PUT", h.method)
	}
	body := string(h.body)
	if !strings.Contains(body, `"enabled":false`) || !strings.Contains(body, `"minTLS":"1.3"`) {
		t.Errorf("body = %s", body)
	}
	// pointer-nil fields must be absent
	var sent map[string]interface{}
	if err := json.Unmarshal(h.body, &sent); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if _, ok := sent["ciphers"]; ok {
		t.Errorf("unset ciphers should be absent from body: %s", body)
	}
	if d.Enabled {
		t.Errorf("parsed domain should be disabled")
	}

	// all-nil request → empty JSON object
	h2 := &bucketDomainHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"domain":"cdn.example.com"}}`))
		},
	}
	svc2 := newBucketDomainTestServer(t, h2)
	if _, err := svc2.Update(context.Background(), "bk", "cdn.example.com", UpdateBucketDomainRequest{}); err != nil {
		t.Fatalf("Update empty: %v", err)
	}
	var sent2 map[string]interface{}
	if err := json.Unmarshal(h2.body, &sent2); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if len(sent2) != 0 {
		t.Errorf("all-nil request should send empty object, got %s", h2.body)
	}
}

func TestBucketDomainDetach(t *testing.T) {
	h := &bucketDomainHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"domain":"cdn.example.com"}}`))
		},
	}
	svc := newBucketDomainTestServer(t, h)

	if err := svc.Detach(context.Background(), "bk", "cdn.example.com"); err != nil {
		t.Fatalf("Detach: %v", err)
	}
	if h.method != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", h.method)
	}
	if h.path != "/accounts/ACC/r2/buckets/bk/domains/custom/cdn.example.com" {
		t.Errorf("path = %s", h.path)
	}
}

func TestBucketDomainAPIError(t *testing.T) {
	h := &bucketDomainHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"success":false,"errors":[{"code":10001,"message":"boom"}],"messages":[],"result":null}`))
		},
	}
	svc := newBucketDomainTestServer(t, h)

	if _, err := svc.Get(context.Background(), "bk", "x.com"); err == nil {
		t.Fatal("expected error")
	} else if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error should mention 'boom', got: %v", err)
	}
}

func TestBucketDomainVerifyPolls(t *testing.T) {
	calls := 0
	h := &bucketDomainHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"domain":"cdn.example.com","enabled":true,"status":{"ownership":"pending","ssl":"pending"}}}`))
				return
			}
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"domain":"cdn.example.com","enabled":true,"status":{"ownership":"active","ssl":"active"}}}`))
		},
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	svc := NewBucketDomainService("ACC", "tok", WithBucketDomainBaseURL(srv.URL), WithBucketDomainPollInterval(time.Millisecond))

	d, err := svc.Verify(context.Background(), "bk", "cdn.example.com", 5*time.Second)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if d == nil || d.Status == nil || d.Status.Ownership != "active" || d.Status.SSL != "active" {
		t.Errorf("final domain = %+v", d)
	}
	if calls < 2 {
		t.Errorf("expected at least 2 polls, got %d", calls)
	}
}

func TestBucketDomainVerifyTimeout(t *testing.T) {
	h := &bucketDomainHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"domain":"cdn.example.com","enabled":true,"status":{"ownership":"pending","ssl":"pending"}}}`))
		},
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	svc := NewBucketDomainService("ACC", "tok", WithBucketDomainBaseURL(srv.URL), WithBucketDomainPollInterval(2*time.Millisecond))

	d, err := svc.Verify(context.Background(), "bk", "cdn.example.com", 30*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "pending") {
		t.Errorf("error should mention both statuses, got: %v", err)
	}
	if d == nil || d.Status == nil {
		t.Fatal("timeout should still return last domain state")
	}
	if d.Status.Ownership != "pending" || d.Status.SSL != "pending" {
		t.Errorf("last state = %+v", d.Status)
	}
}
