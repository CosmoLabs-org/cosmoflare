package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// bucketLifecycleHandler captures the last request seen by the test server.
type bucketLifecycleHandler struct {
	method string
	path   string
	auth   string
	body   []byte
	inner  http.HandlerFunc
}

func (h *bucketLifecycleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func newBucketLifecycleTestServer(t *testing.T, h *bucketLifecycleHandler) *BucketLifecycleService {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewBucketLifecycleService("ACC", "tok-secret", WithBucketLifecycleBaseURL(srv.URL))
}

func lifecycleAge(maxAge int64) *LifecycleTransition {
	return &LifecycleTransition{Condition: LifecycleCondition{Type: "Age", MaxAge: &maxAge}}
}

func TestBucketLifecycleGet(t *testing.T) {
	h := &bucketLifecycleHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"rules":[
				{"id":"expire imgs","enabled":true,"conditions":{"prefix":"img/"},
				 "deleteObjectsTransition":{"condition":{"type":"Age","maxAge":7776000}}},
				{"id":"abort mpu","enabled":false,"conditions":{"prefix":""},
				 "abortMultipartUploadsTransition":{"condition":{"type":"Age","maxAge":604800}},
				 "storageClassTransitions":[{"condition":{"type":"Age","maxAge":2592000},"storageClass":"InfrequentAccess"}]}
			]}}`))
		},
	}
	svc := newBucketLifecycleTestServer(t, h)

	rules, err := svc.Get(context.Background(), "bk")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if h.method != http.MethodGet {
		t.Errorf("method = %s, want GET", h.method)
	}
	if h.path != "/accounts/ACC/r2/buckets/bk/lifecycle" {
		t.Errorf("path = %s", h.path)
	}
	if h.auth != "Bearer tok-secret" {
		t.Errorf("Authorization = %q", h.auth)
	}
	if len(rules) != 2 {
		t.Fatalf("len(rules) = %d, want 2", len(rules))
	}
	r0 := rules[0]
	if r0.ID != "expire imgs" || !r0.Enabled || r0.Conditions.Prefix != "img/" {
		t.Errorf("rule 0 = %+v", r0)
	}
	if r0.DeleteObjectsTransition == nil || r0.DeleteObjectsTransition.Condition.Type != "Age" ||
		r0.DeleteObjectsTransition.Condition.MaxAge == nil || *r0.DeleteObjectsTransition.Condition.MaxAge != 7776000 {
		t.Errorf("rule 0 delete transition = %+v", r0.DeleteObjectsTransition)
	}
	r1 := rules[1]
	if r1.AbortMultipartUploadsTransition == nil || r1.AbortMultipartUploadsTransition.Condition.MaxAge == nil ||
		*r1.AbortMultipartUploadsTransition.Condition.MaxAge != 604800 {
		t.Errorf("rule 1 abort transition = %+v", r1.AbortMultipartUploadsTransition)
	}
	if len(r1.StorageClassTransitions) != 1 ||
		r1.StorageClassTransitions[0].StorageClass != "InfrequentAccess" {
		t.Errorf("rule 1 storage class transitions = %+v", r1.StorageClassTransitions)
	}
}

func TestBucketLifecycleGetEmpty(t *testing.T) {
	h := &bucketLifecycleHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"rules":[]}}`))
		},
	}
	svc := newBucketLifecycleTestServer(t, h)
	rules, err := svc.Get(context.Background(), "bk")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rules == nil {
		t.Fatal("rules should be non-nil when none exist")
	}
	if len(rules) != 0 {
		t.Errorf("len(rules) = %d, want 0", len(rules))
	}
}

func TestBucketLifecycleSetReplaces(t *testing.T) {
	h := &bucketLifecycleHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"rules":[]}}`))
		},
	}
	svc := newBucketLifecycleTestServer(t, h)

	maxAge := int64(86400)
	rules := []LifecycleRule{{
		ID:         "daily expire",
		Enabled:    true,
		Conditions: LifecycleConditions{Prefix: "tmp/"},
		DeleteObjectsTransition: &LifecycleTransition{
			Condition: LifecycleCondition{Type: "Age", MaxAge: &maxAge},
		},
	}}
	if err := svc.Set(context.Background(), "bk", rules); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if h.method != http.MethodPut {
		t.Errorf("method = %s, want PUT", h.method)
	}
	if h.path != "/accounts/ACC/r2/buckets/bk/lifecycle" {
		t.Errorf("path = %s", h.path)
	}
	want, _ := json.Marshal(map[string]interface{}{"rules": rules})
	var gotBody map[string]json.RawMessage
	if err := json.Unmarshal(h.body, &gotBody); err != nil {
		t.Fatalf("body not JSON: %v (%s)", err, h.body)
	}
	var gotRules []json.RawMessage
	if err := json.Unmarshal(gotBody["rules"], &gotRules); err != nil {
		t.Fatalf("rules not a list: %v (%s)", err, gotBody["rules"])
	}
	var wantRules []json.RawMessage
	wantBody := map[string]json.RawMessage{}
	if err := json.Unmarshal(want, &wantBody); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(wantBody["rules"], &wantRules); err != nil {
		t.Fatalf("marshal rules: %v", err)
	}
	if len(gotRules) != len(wantRules) {
		t.Fatalf("rule count = %d, want %d (%s)", len(gotRules), len(wantRules), h.body)
	}
	var g, w interface{}
	json.Unmarshal(gotRules[0], &g)
	json.Unmarshal(wantRules[0], &w)
	if !jsonEqual(g, w) {
		t.Errorf("body rules mismatch:\n got: %s\nwant: %s", gotRules[0], wantRules[0])
	}
}

func jsonEqual(a, b interface{}) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func TestBucketLifecycleSetClear(t *testing.T) {
	h := &bucketLifecycleHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{"rules":[]}}`))
		},
	}
	svc := newBucketLifecycleTestServer(t, h)

	if err := svc.Set(context.Background(), "bk", nil); err != nil {
		t.Fatalf("Set(nil): %v", err)
	}
	if strings.TrimSpace(string(h.body)) != `{"rules":[]}` {
		t.Errorf("body = %s, want {\"rules\":[]}", h.body)
	}
	if h.method != http.MethodPut {
		t.Errorf("method = %s, want PUT", h.method)
	}
}

func TestBucketLifecycleValidation(t *testing.T) {
	svc := newBucketLifecycleTestServer(t, &bucketLifecycleHandler{})

	dateStr := "2026-01-01T00:00:00Z"
	maxAge := int64(0)

	// abortMultipartUploadsTransition with a Date condition is invalid.
	err := svc.Set(context.Background(), "bk", []LifecycleRule{{
		ID:         "bad abort",
		Enabled:    true,
		Conditions: LifecycleConditions{Prefix: ""},
		AbortMultipartUploadsTransition: &LifecycleTransition{
			Condition: LifecycleCondition{Type: "Date", Date: &dateStr},
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "Age") {
		t.Errorf("abort with Date: err = %v, want Age-only error", err)
	}

	// Age with MaxAge 0 is invalid.
	err = svc.Set(context.Background(), "bk", []LifecycleRule{{
		ID:         "zero age",
		Enabled:    true,
		Conditions: LifecycleConditions{Prefix: ""},
		DeleteObjectsTransition: &LifecycleTransition{
			Condition: LifecycleCondition{Type: "Age", MaxAge: &maxAge},
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "maxAge") {
		t.Errorf("maxAge 0: err = %v, want maxAge error", err)
	}

	// Unknown storage class is invalid.
	one := int64(3600)
	err = svc.Set(context.Background(), "bk", []LifecycleRule{{
		ID:         "bad class",
		Enabled:    true,
		Conditions: LifecycleConditions{Prefix: ""},
		StorageClassTransitions: []StorageClassTransition{{
			Condition:    LifecycleCondition{Type: "Age", MaxAge: &one},
			StorageClass: "Glacier",
		}},
	}})
	if err == nil || !strings.Contains(err.Error(), "InfrequentAccess") {
		t.Errorf("bad storage class: err = %v, want storage class error", err)
	}

	// Empty rule ID is invalid.
	err = svc.Set(context.Background(), "bk", []LifecycleRule{{
		ID:         "",
		Conditions: LifecycleConditions{Prefix: ""},
	}})
	if err == nil || !strings.Contains(err.Error(), "id") {
		t.Errorf("empty id: err = %v, want id error", err)
	}

	// More than 1000 rules is invalid.
	many := make([]LifecycleRule, 1001)
	for i := range many {
		many[i] = LifecycleRule{ID: "r", Conditions: LifecycleConditions{Prefix: ""}}
	}
	err = svc.Set(context.Background(), "bk", many)
	if err == nil || !strings.Contains(err.Error(), "1000") {
		t.Errorf(">1000 rules: err = %v, want limit error", err)
	}
}

func TestBucketLifecycleValidationSkipsHTTP(t *testing.T) {
	h := &bucketLifecycleHandler{}
	svc := newBucketLifecycleTestServer(t, h)
	dateStr := "2026-01-01T00:00:00Z"
	err := svc.Set(context.Background(), "bk", []LifecycleRule{{
		ID:         "bad abort",
		Conditions: LifecycleConditions{Prefix: ""},
		AbortMultipartUploadsTransition: &LifecycleTransition{
			Condition: LifecycleCondition{Type: "Date", Date: &dateStr},
		},
	}})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if h.method != "" {
		t.Errorf("server should never be hit on validation failure, got %s %s", h.method, h.path)
	}
}

func TestBucketLifecycleAPIError(t *testing.T) {
	h := &bucketLifecycleHandler{
		inner: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":false,"errors":[{"code":10001,"message":"boom"}],"messages":[],"result":null}`))
		},
	}
	svc := newBucketLifecycleTestServer(t, h)

	if _, err := svc.Get(context.Background(), "bk"); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("Get error = %v, want boom", err)
	}
	one := int64(60)
	err := svc.Set(context.Background(), "bk", []LifecycleRule{{
		ID:         "r",
		Conditions: LifecycleConditions{Prefix: ""},
		DeleteObjectsTransition: &LifecycleTransition{
			Condition: LifecycleCondition{Type: "Age", MaxAge: &one},
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("Set error = %v, want boom", err)
	}
}
