package cosmoflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Workers script VERSIONS + DEPLOYMENTS via raw REST.
//
// cloudflare-go v0.116 ships no bindings for the script versions API, so
// these calls talk to the REST endpoints directly through rawRequest. They
// therefore require credentials-based construction (NewWorkerServiceFromCreds)
// — a service built from a bare *cloudflare.API has no API token to
// authenticate with and every method below returns a validation error.

// WorkerVersion is an immutable snapshot of a Worker script upload.
type WorkerVersion struct {
	ID        string    `json:"id"`
	Number    int       `json:"number"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source,omitempty"`
}

// WorkerDeployment records a point-in-time rollout of a version.
type WorkerDeployment struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	VersionID   string    `json:"version_id"`
	Source      string    `json:"source,omitempty"`
	AuthorEmail string    `json:"author_email,omitempty"`
}

// rawRequest issues one Cloudflare REST call against s.apiBaseURL.
//
// It speaks the standard CF envelope: {"success":bool,"result":...,
// "errors":[...]}. Non-2xx statuses and success:false envelopes both surface
// as *R2Error carrying the envelope's message. On success with out != nil the
// envelope's "result" field is unmarshalled into out.
func (s *WorkerService) rawRequest(ctx context.Context, method, path string, body, out any) error {
	if s.apiToken == "" {
		return validationError("rawRequest", "raw API requires credentials-based construction (NewWorkerServiceFromCreds)")
	}

	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return newError("rawRequest", "failed to marshal request body", err)
		}
		payload = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.apiBaseURL+path, payload)
	if err != nil {
		return newError("rawRequest", "failed to create request", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// controlPlaneClient() is the shared 30s-timeout control-plane client
	// (transport.go, FEAT-039) — the raw path inherits the same policy as
	// every other Cloudflare call in the package.
	resp, err := controlPlaneClient().Do(req)
	if err != nil {
		return newError("rawRequest", "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return newError("rawRequest", "failed to read response", err)
	}

	var envelope struct {
		Success bool            `json:"success"`
		Result  json.RawMessage `json:"result"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if uerr := json.Unmarshal(raw, &envelope); uerr != nil {
		if !httpStatusOK(resp.StatusCode) {
			return newError("rawRequest", fmt.Sprintf("API error %d: %s", resp.StatusCode, truncateSpace(string(raw))), nil)
		}
		return newError("rawRequest", "failed to decode response envelope", uerr)
	}

	msg := ""
	if len(envelope.Errors) > 0 {
		msg = envelope.Errors[0].Message
	}
	if !httpStatusOK(resp.StatusCode) || !envelope.Success {
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return newError("rawRequest", msg, nil)
	}

	if out != nil && len(envelope.Result) > 0 {
		if uerr := json.Unmarshal(envelope.Result, out); uerr != nil {
			return newError("rawRequest", "failed to decode result", uerr)
		}
	}
	return nil
}

// workerVersionsPath builds the script-versions path for a Worker.
func (s *WorkerService) workerVersionsPath(worker string) string {
	return fmt.Sprintf("/accounts/%s/workers/scripts/%s/versions", s.accountID, worker)
}

// VersionUpload creates a new version of a Worker script without deploying
// it. The JSON body mirrors Deploy's option set (script, bindings,
// compatibility_date, plus optional tags/module flags).
func (s *WorkerService) VersionUpload(ctx context.Context, worker string, script io.Reader, opts ...WorkerOption) (*WorkerVersion, error) {
	if worker == "" {
		return nil, validationError("WorkerService.VersionUpload", "worker name is required")
	}
	if script == nil {
		return nil, validationError("WorkerService.VersionUpload", "script content is required")
	}

	cfg := &workerConfig{}
	for _, o := range opts {
		o(cfg)
	}

	scriptBytes, err := io.ReadAll(script)
	if err != nil {
		return nil, newError("WorkerService.VersionUpload", "failed to read script content", err)
	}

	body := map[string]any{
		"script":             string(scriptBytes),
		"bindings":           toCFBindings(cfg.bindings),
		"compatibility_date": cfg.compatibilityDate,
	}
	if len(cfg.tags) > 0 {
		body["tags"] = cfg.tags
	}
	if cfg.module {
		body["module"] = true
	}

	var version WorkerVersion
	if err := s.rawRequest(ctx, http.MethodPost, s.workerVersionsPath(worker), body, &version); err != nil {
		return nil, err
	}
	return &version, nil
}

// VersionList returns all versions of a Worker script.
func (s *WorkerService) VersionList(ctx context.Context, worker string) ([]WorkerVersion, error) {
	if worker == "" {
		return nil, validationError("WorkerService.VersionList", "worker name is required")
	}

	var versions []WorkerVersion
	if err := s.rawRequest(ctx, http.MethodGet, s.workerVersionsPath(worker), nil, &versions); err != nil {
		return nil, err
	}
	if versions == nil {
		versions = []WorkerVersion{}
	}
	return versions, nil
}

// VersionGet retrieves a single version of a Worker script.
func (s *WorkerService) VersionGet(ctx context.Context, worker, versionID string) (*WorkerVersion, error) {
	if worker == "" {
		return nil, validationError("WorkerService.VersionGet", "worker name is required")
	}
	if versionID == "" {
		return nil, validationError("WorkerService.VersionGet", "version ID is required")
	}

	var version WorkerVersion
	path := s.workerVersionsPath(worker) + "/" + versionID
	if err := s.rawRequest(ctx, http.MethodGet, path, nil, &version); err != nil {
		return nil, err
	}
	return &version, nil
}

// VersionDeploy promotes a version to the live deployment.
func (s *WorkerService) VersionDeploy(ctx context.Context, worker, versionID string) (*WorkerDeployment, error) {
	if worker == "" {
		return nil, validationError("WorkerService.VersionDeploy", "worker name is required")
	}
	if versionID == "" {
		return nil, validationError("WorkerService.VersionDeploy", "version ID is required")
	}

	var deployment WorkerDeployment
	path := fmt.Sprintf("/accounts/%s/workers/scripts/%s/deployments", s.accountID, worker)
	body := map[string]string{"version_id": versionID}
	if err := s.rawRequest(ctx, http.MethodPut, path, body, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

// VersionDelete removes an undeployed version of a Worker script.
func (s *WorkerService) VersionDelete(ctx context.Context, worker, versionID string) error {
	if worker == "" {
		return validationError("WorkerService.VersionDelete", "worker name is required")
	}
	if versionID == "" {
		return validationError("WorkerService.VersionDelete", "version ID is required")
	}

	path := s.workerVersionsPath(worker) + "/" + versionID
	return s.rawRequest(ctx, http.MethodDelete, path, nil, nil)
}

// VersionRollback redeploys a previously uploaded version. The API has no
// dedicated rollback endpoint: rolling back IS deploying an older version_id,
// so this delegates to VersionDeploy and callers label it a rollback.
func (s *WorkerService) VersionRollback(ctx context.Context, worker, versionID string) (*WorkerDeployment, error) {
	return s.VersionDeploy(ctx, worker, versionID)
}

// httpStatusOK reports whether a response status is in the 2xx range.
func httpStatusOK(code int) bool { return code >= 200 && code <= 299 }

// truncateSpace trims surrounding whitespace for inline error bodies.
func truncateSpace(s string) string {
	if len(s) > 200 {
		s = s[:200]
	}
	return string(bytes.TrimSpace([]byte(s)))
}
