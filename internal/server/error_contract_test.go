package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// TestErrorContract verifies that every daemon error path maps typed
// pkg/cosmoflare errors to a stable HTTP status + {"error","code"} body,
// instead of the old blanket 502.
func TestErrorContract(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "validation",
			err:        &cosmoflare.R2ValidationError{R2Error: cosmoflare.R2Error{Op: "op", Message: "bad bucket name"}},
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation_failed",
		},
		{
			name:       "auth",
			err:        &cosmoflare.R2AuthError{R2Error: cosmoflare.R2Error{Op: "op", Message: "invalid api token"}},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:       "not found",
			err:        &cosmoflare.R2NotFoundError{R2Error: cosmoflare.R2Error{Op: "op", Bucket: "bucket", Key: "key", Message: "resource not found"}},
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name:       "rate limited",
			err:        &cosmoflare.R2QuotaError{R2Error: cosmoflare.R2Error{Op: "op", Message: "rate limit exceeded"}},
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "rate_limited",
		},
		{
			name:       "unclassified upstream",
			err:        errors.New("connection reset by peer"),
			wantStatus: http.StatusBadGateway,
			wantCode:   "upstream_error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := &fakeSource{zoneErr: tc.err}
			_, url := newSourcedServer(t, src)

			resp := get(t, url+"/zones", "t")
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}

			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if _, ok := body["error"]; !ok {
				t.Fatalf("body missing \"error\" field: %v", body)
			}
			if code, _ := body["code"].(string); code != tc.wantCode {
				t.Fatalf("body[code] = %q, want %q (body=%v)", code, tc.wantCode, body)
			}
		})
	}
}
