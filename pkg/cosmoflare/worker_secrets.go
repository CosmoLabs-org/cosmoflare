package cosmoflare

import (
	"context"
	"fmt"
	"sort"

	"github.com/cloudflare/cloudflare-go"
)

// WorkerSecret is the public view of a Worker secret binding: the NAME and
// binding type only. Secret VALUES are never represented, returned, or
// logged by this package — the Cloudflare API does not expose them either.
type WorkerSecret struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// SecretBulkResult reports the per-key outcome of a SecretsBulk upload.
// A failed key does not abort the remaining keys; Error carries the
// upstream message (names/opcodes only, never a secret value).
type SecretBulkResult struct {
	Key     string `json:"key"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// SecretPut creates or updates a single secret on a Worker.
// The value is sent to the API and never appears in any error message.
func (s *WorkerService) SecretPut(ctx context.Context, worker, key, value string) error {
	if worker == "" {
		return validationError("WorkerService.SecretPut", "worker name is required")
	}
	if key == "" {
		return validationError("WorkerService.SecretPut", "secret name is required")
	}
	if value == "" {
		return validationError("WorkerService.SecretPut", "secret value is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	_, err := s.cf.SetWorkersSecret(ctx, rc, cloudflare.SetWorkersSecretParams{
		ScriptName: worker,
		Secret: &cloudflare.WorkersPutSecretRequest{
			Name: key,
			Text: value,
			Type: cloudflare.WorkerSecretTextBindingType,
		},
	})
	if err != nil {
		return newError("WorkerService.SecretPut", fmt.Sprintf("failed to set secret %q on worker %q", key, worker), err)
	}
	return nil
}

// SecretDelete removes a secret from a Worker.
func (s *WorkerService) SecretDelete(ctx context.Context, worker, key string) error {
	if worker == "" {
		return validationError("WorkerService.SecretDelete", "worker name is required")
	}
	if key == "" {
		return validationError("WorkerService.SecretDelete", "secret name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	_, err := s.cf.DeleteWorkersSecret(ctx, rc, cloudflare.DeleteWorkersSecretParams{
		ScriptName: worker,
		SecretName: key,
	})
	if err != nil {
		return newError("WorkerService.SecretDelete", fmt.Sprintf("failed to delete secret %q on worker %q", key, worker), err)
	}
	return nil
}

// SecretList returns the secret names bound to a Worker, sorted by name.
// Cloudflare does not return secret values, so neither do we.
func (s *WorkerService) SecretList(ctx context.Context, worker string) ([]WorkerSecret, error) {
	if worker == "" {
		return nil, validationError("WorkerService.SecretList", "worker name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	resp, err := s.cf.ListWorkersSecrets(ctx, rc, cloudflare.ListWorkersSecretsParams{
		ScriptName: worker,
	})
	if err != nil {
		return nil, newError("WorkerService.SecretList", fmt.Sprintf("failed to list secrets for worker %q", worker), err)
	}

	secrets := make([]WorkerSecret, 0, len(resp.Result))
	for _, sec := range resp.Result {
		secretType := sec.Type
		// cloudflare-go v0.116 tags WorkersSecret.Type as json:"secret_text"
		// instead of the API's "type", so the field arrives empty for the
		// real response shape. Every secret on this endpoint is a
		// secret_text binding, so normalize the blank case.
		if secretType == "" {
			secretType = "secret_text"
		}
		secrets = append(secrets, WorkerSecret{Name: sec.Name, Type: secretType})
	}
	sort.Slice(secrets, func(i, j int) bool { return secrets[i].Name < secrets[j].Name })
	return secrets, nil
}

// SecretsBulk uploads every entry of secrets to a Worker, collecting a
// per-key result. Keys are processed in sorted order for deterministic
// output; a failing key is recorded and does NOT abort the remaining keys.
// Bulk itself only fails on validation (empty worker / empty map) — the
// returned slice always has one entry per input key.
func (s *WorkerService) SecretsBulk(ctx context.Context, worker string, secrets map[string]string) ([]SecretBulkResult, error) {
	if worker == "" {
		return nil, validationError("WorkerService.SecretsBulk", "worker name is required")
	}
	if len(secrets) == 0 {
		return nil, validationError("WorkerService.SecretsBulk", "at least one secret is required")
	}

	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	results := make([]SecretBulkResult, 0, len(keys))
	for _, key := range keys {
		res := SecretBulkResult{Key: key}
		if err := s.SecretPut(ctx, worker, key, secrets[key]); err != nil {
			res.Error = err.Error()
		} else {
			res.Success = true
		}
		results = append(results, res)
	}
	return results, nil
}
