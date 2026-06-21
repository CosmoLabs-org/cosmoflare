package server

import (
	"context"
	"encoding/json"
	"net/http"
)

// ServeSource is the daemon's data-access seam. It is a NEW abstraction for the
// desktop daemon — it is NOT internal/tui.DataSource (which is R2-storage-only).
// Injecting it lets the REST handlers be unit-tested with no live Cloudflare
// credentials: tests pass a fake, the daemon passes an adapter over the real
// per-service constructors (see cmd/serve.go).
//
// Account model (v1, read-only): an "account" is a local config profile. The CF
// methods take a profile name (empty = current/default profile) so the daemon
// re-resolves credentials per request. There is no write-side "switch".
type ServeSource interface {
	// Accounts returns the local config profiles. This is a local read, NOT a
	// Cloudflare API call — it must not affect cloudflare_online.
	Accounts(ctx context.Context) (any, error)
	// The following methods resolve credentials from the named profile and
	// call the corresponding Cloudflare service. Success → caller records
	// cloudflare_online=true; error → cloudflare_online=false + 502.
	Zones(ctx context.Context, profile string) (any, error)
	R2Buckets(ctx context.Context, profile string) (any, error)
	Workers(ctx context.Context, profile string) (any, error)
	KV(ctx context.Context, profile string) (any, error)
}

// SetData attaches the data source backing the REST read endpoints. The server
// returns 503 for the CF endpoints until a source is attached.
func (s *Server) SetData(src ServeSource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.src = src
}

// registerRESTRoutes mounts the read-only REST endpoints. It is called from
// Server.registerRoutes.
func (s *Server) registerRESTRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/accounts", s.handleAccounts)
	mux.HandleFunc("/zones", s.withCloudflare(func(ctx context.Context, profile string) (any, error) {
		return s.src.Zones(ctx, profile)
	}))
	mux.HandleFunc("/r2/buckets", s.withCloudflare(func(ctx context.Context, profile string) (any, error) {
		return s.src.R2Buckets(ctx, profile)
	}))
	mux.HandleFunc("/workers", s.withCloudflare(func(ctx context.Context, profile string) (any, error) {
		return s.src.Workers(ctx, profile)
	}))
	mux.HandleFunc("/kv", s.withCloudflare(func(ctx context.Context, profile string) (any, error) {
		return s.src.KV(ctx, profile)
	}))
}

// handleAccounts serves the local profile list. It is a local read; it does not
// touch the cloudflare_online tier.
func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	src := s.src
	s.mu.Unlock()
	if src == nil {
		http.Error(w, `{"error":"no data source configured"}`, http.StatusServiceUnavailable)
		return
	}
	data, err := src.Accounts(r.Context())
	if err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, data)
}

// cfHandler is a Cloudflare-backed read: resolve ?profile, call the service,
// record cloudflare_online, and return JSON (200) or 502 + error on failure.
type cfHandler func(ctx context.Context, profile string) (any, error)

// withCloudflare wraps a CF read with the source-nil check, profile parsing,
// cloudflare_online bookkeeping, and error mapping.
func (s *Server) withCloudflare(h cfHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		src := s.src
		s.mu.Unlock()
		if src == nil {
			http.Error(w, `{"error":"no data source configured"}`, http.StatusServiceUnavailable)
			return
		}
		profile := r.URL.Query().Get("profile")
		data, err := h(r.Context(), profile)
		if err != nil {
			s.SetCloudflareOnline(false)
			writeJSONStatus(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		s.SetCloudflareOnline(true)
		writeJSON(w, data)
	}
}

// writeJSONStatus encodes v at the given HTTP status.
func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
