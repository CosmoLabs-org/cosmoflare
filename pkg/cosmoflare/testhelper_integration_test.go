//go:build integration

package cosmoflare

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func strReader(s string) io.Reader {
	return strings.NewReader(s)
}

type cfResponse struct {
	Result     interface{} `json:"result"`
	ResultInfo interface{} `json:"result_info,omitempty"`
	Success    bool        `json:"success"`
	Errors     []struct{}  `json:"errors"`
	Messages   []struct{}  `json:"messages"`
}

func writeCFJSON(w http.ResponseWriter, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfResponse{
		Result:  result,
		Success: true,
		Errors:  []struct{}{},
	})
}

func writeCFJSONWithInfo(w http.ResponseWriter, result, info interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfResponse{
		Result:     result,
		ResultInfo: info,
		Success:    true,
		Errors:     []struct{}{},
	})
}

func writeCFError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"errors":  []map[string]interface{}{{"code": code, "message": message}},
	})
}

func newTestClientWithServer(t *testing.T, handler http.Handler) (*client, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfAPI, err := cloudflare.NewWithAPIToken("test-token")
	if err != nil {
		t.Fatalf("failed to create CF API: %v", err)
	}
	cfAPI.BaseURL = server.URL

	c := &client{
		cf:         cfAPI,
		accountID:  "test-account-id",
		apiToken:   "test-token",
		httpClient: server.Client(),
		cfg:        &clientConfig{region: "auto"},
	}

	return c, server
}
