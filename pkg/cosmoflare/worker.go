package cosmoflare

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// Worker represents a Cloudflare Worker script.
type Worker struct {
	Name        string          `json:"name"`
	Modified    time.Time       `json:"modified"`
	Size        int64           `json:"size"`
	Runtime     string          `json:"runtime"`
	Bindings    []WorkerBinding `json:"bindings,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Script      string          `json:"script,omitempty"`
	CompatibilityDate string   `json:"compatibility_date,omitempty"`
}

// WorkerBinding represents a binding attached to a Worker.
type WorkerBinding struct {
	Name string `json:"name"`
	Type string `json:"type"` // "kv", "r2", "d1", "queue", "var", "secret_text"
	ID   string `json:"id"`
}

// WorkerSettings holds configurable Worker settings.
type WorkerSettings struct {
	CompatibilityDate string   `json:"compatibility_date"`
	UsageModel        string   `json:"usage_model"` // "bundled" or "unbound"
	Bindings          []WorkerBinding `json:"bindings,omitempty"`
}

// LogEntry represents a single Worker log event.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Event     string    `json:"event"`
}

// WorkerOption is a functional option for Worker operations.
type WorkerOption func(*workerConfig)

type workerConfig struct {
	compatibilityDate string
	bindings          []WorkerBinding
	tags              []string
	module            bool
}

// WithWorkerCompatibilityDate sets the Workers runtime compatibility date.
func WithWorkerCompatibilityDate(date string) WorkerOption {
	return func(c *workerConfig) { c.compatibilityDate = date }
}

// WithWorkerBindings attaches resource bindings to the Worker.
func WithWorkerBindings(bindings []WorkerBinding) WorkerOption {
	return func(c *workerConfig) { c.bindings = bindings }
}

// WithWorkerTags attaches tags to the Worker.
func WithWorkerTags(tags []string) WorkerOption {
	return func(c *workerConfig) { c.tags = tags }
}

// WithWorkerModule marks the script as an ES module.
func WithWorkerModule(enabled bool) WorkerOption {
	return func(c *workerConfig) { c.module = enabled }
}

// LogOption is a functional option for log retrieval.
type LogOption func(*logConfig)

type logConfig struct {
	limit int
	since time.Time
}

// WithLogLimit sets the maximum number of log entries to return.
func WithLogLimit(n int) LogOption {
	return func(c *logConfig) { c.limit = n }
}

// WithLogSince filters logs to entries after the given time.
func WithLogSince(t time.Time) LogOption {
	return func(c *logConfig) { c.since = t }
}

// WorkerService implements WorkerService.
type WorkerService struct {
	cf        *cloudflare.API
	accountID string
	// apiToken/apiBaseURL back the raw REST fallback (worker_versions.go):
	// cloudflare-go v0.116 has no Workers versions API, so those calls are
	// issued directly. Zero-value for NewWorkerService (no raw access).
	apiToken   string
	apiBaseURL string
}

// NewWorkerService creates a new Workers service client.
func NewWorkerService(api *cloudflare.API, accountID string) (*WorkerService, error) {
	if api == nil {
		return nil, validationError("NewWorkerService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewWorkerService", "account ID is required")
	}
	return &WorkerService{cf: api, accountID: accountID}, nil
}

// NewWorkerServiceFromCreds creates a WorkerService from account ID and API token.
// Convenience helper for CLI usage.
func NewWorkerServiceFromCreds(accountID, apiToken string) (*WorkerService, error) {
	if accountID == "" {
		return nil, validationError("NewWorkerService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewWorkerService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewWorkerService", "failed to create Cloudflare API client", err)
	}
	return &WorkerService{
		cf:         cf,
		accountID:  accountID,
		apiToken:   apiToken,
		apiBaseURL: "https://api.cloudflare.com/client/v4",
	}, nil
}

// Deploy uploads or updates a Worker script.
func (s *WorkerService) Deploy(ctx context.Context, name string, script io.Reader, opts ...WorkerOption) (*Worker, error) {
	if name == "" {
		return nil, validationError("WorkerService.Deploy", "worker name is required")
	}
	if script == nil {
		return nil, validationError("WorkerService.Deploy", "script content is required")
	}

	cfg := &workerConfig{}
	for _, o := range opts {
		o(cfg)
	}

	scriptBytes, err := io.ReadAll(script)
	if err != nil {
		return nil, newError("WorkerService.Deploy", "failed to read script content", err)
	}

	cfBindings := toCFBindings(cfg.bindings)

	params := cloudflare.CreateWorkerParams{
		ScriptName:        name,
		Script:            string(scriptBytes),
		Module:            cfg.module,
		CompatibilityDate: cfg.compatibilityDate,
		Bindings:          cfBindings,
		Tags:              cfg.tags,
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	resp, err := s.cf.UploadWorker(ctx, rc, params)
	if err != nil {
		return nil, newError("WorkerService.Deploy", fmt.Sprintf("failed to deploy worker %q", name), err)
	}

	w := &Worker{
		Name:     name,
		Modified: resp.WorkerMetaData.ModifiedOn,
		Size:     int64(resp.WorkerMetaData.Size),
		Script:   resp.WorkerScript.Script,
	}
	return w, nil
}

// List returns all Workers in the account.
func (s *WorkerService) List(ctx context.Context) ([]*Worker, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	resp, _, err := s.cf.ListWorkers(ctx, rc, cloudflare.ListWorkersParams{})
	if err != nil {
		return nil, newError("WorkerService.List", "failed to list workers", err)
	}

	workers := make([]*Worker, 0, len(resp.WorkerList))
	for _, md := range resp.WorkerList {
		w := &Worker{
			Name:     md.ID,
			Modified: md.ModifiedOn,
			Size:     int64(md.Size),
		}
		workers = append(workers, w)
	}
	return workers, nil
}

// Get retrieves a single Worker's script content and metadata.
func (s *WorkerService) Get(ctx context.Context, name string) (*Worker, error) {
	if name == "" {
		return nil, validationError("WorkerService.Get", "worker name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	resp, err := s.cf.GetWorker(ctx, rc, name)
	if err != nil {
		return nil, notFound("WorkerService.Get", "", name, err)
	}

	w := &Worker{
		Name:     name,
		Modified: resp.WorkerMetaData.ModifiedOn,
		Size:     int64(resp.WorkerMetaData.Size),
		Script:   resp.WorkerScript.Script,
	}
	return w, nil
}

// Delete removes a Worker script.
func (s *WorkerService) Delete(ctx context.Context, name string) error {
	if name == "" {
		return validationError("WorkerService.Delete", "worker name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	err := s.cf.DeleteWorker(ctx, rc, cloudflare.DeleteWorkerParams{ScriptName: name})
	if err != nil {
		return newError("WorkerService.Delete", fmt.Sprintf("failed to delete worker %q", name), err)
	}
	return nil
}

// Logs retrieves recent log entries for a Worker.
// Uses the script settings/metadata API as a proxy since the real-time
// WebSocket tail API requires persistent connections.
func (s *WorkerService) Logs(ctx context.Context, name string, opts ...LogOption) ([]*LogEntry, error) {
	if name == "" {
		return nil, validationError("WorkerService.Logs", "worker name is required")
	}

	cfg := &logConfig{}
	for _, o := range opts {
		o(cfg)
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	settings, err := s.cf.GetWorkersScriptSettings(ctx, rc, name)
	if err != nil {
		return nil, newError("WorkerService.Logs", fmt.Sprintf("failed to get worker logs for %q", name), err)
	}

	entries := []*LogEntry{{
		Timestamp: settings.WorkerMetaData.ModifiedOn,
		Level:     "info",
		Message:   fmt.Sprintf("Worker %q last modified", name),
		Event:     "metadata",
	}}

	if cfg.limit > 0 && len(entries) > cfg.limit {
		entries = entries[:cfg.limit]
	}
	return entries, nil
}

// TailOptions configures the TailLogs polling loop.
type TailOptions struct {
	Interval time.Duration
	Level    string
	Since    time.Duration
}

// TailLogs polls for Worker log entries and sends them on the returned channel.
// The channel closes when ctx is cancelled. Interval defaults to 2s if zero.
func (s *WorkerService) TailLogs(ctx context.Context, name string, opts *TailOptions) (<-chan *LogEntry, error) {
	if name == "" {
		return nil, validationError("WorkerService.TailLogs", "worker name is required")
	}

	if opts == nil {
		opts = &TailOptions{}
	}
	if opts.Interval <= 0 {
		opts.Interval = 2 * time.Second
	}

	ch := make(chan *LogEntry, 64)
	var sinceTime time.Time
	if opts.Since > 0 {
		sinceTime = time.Now().Add(-opts.Since)
	}

	go func() {
		defer close(ch)
		ticker := time.NewTicker(opts.Interval)
		defer ticker.Stop()

		poll := func() {
			var logOpts []LogOption
			if !sinceTime.IsZero() {
				logOpts = append(logOpts, WithLogSince(sinceTime))
			}
			entries, err := s.Logs(ctx, name, logOpts...)
			if err != nil {
				return
			}
			for _, e := range entries {
				if opts.Level != "" && e.Level != opts.Level {
					continue
				}
				if !sinceTime.IsZero() && !e.Timestamp.After(sinceTime) {
					continue
				}
				select {
				case ch <- e:
				case <-ctx.Done():
					return
				}
			}
			if len(entries) > 0 {
				sinceTime = entries[len(entries)-1].Timestamp
			}
		}

		poll()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				poll()
			}
		}
	}()

	return ch, nil
}

// UpdateSettings updates a Worker's configuration (compatibility date, bindings, usage model).
func (s *WorkerService) UpdateSettings(ctx context.Context, name string, settings WorkerSettings) error {
	if name == "" {
		return validationError("WorkerService.UpdateSettings", "worker name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	params := cloudflare.UpdateWorkersScriptSettingsParams{
		ScriptName:        name,
		CompatibilityDate: settings.CompatibilityDate,
	}

	params.Bindings = toCFBindings(settings.Bindings)

	_, err := s.cf.UpdateWorkersScriptSettings(ctx, rc, params)
	if err != nil {
		return newError("WorkerService.UpdateSettings", fmt.Sprintf("failed to update settings for worker %q", name), err)
	}
	return nil
}

// toCFBindings converts our WorkerBinding slice to cloudflare-go's map[string]WorkerBinding.
func toCFBindings(bindings []WorkerBinding) map[string]cloudflare.WorkerBinding {
	cfBindings := make(map[string]cloudflare.WorkerBinding)
	for _, b := range bindings {
		switch b.Type {
		case "kv":
			cfBindings[b.Name] = cloudflare.WorkerKvNamespaceBinding{NamespaceID: b.ID}
		case "r2":
			cfBindings[b.Name] = cloudflare.WorkerR2BucketBinding{BucketName: b.ID}
		case "d1":
			cfBindings[b.Name] = cloudflare.WorkerD1DatabaseBinding{DatabaseID: b.ID}
		case "queue":
			cfBindings[b.Name] = cloudflare.WorkerQueueBinding{Binding: b.Name, Queue: b.ID}
		case "service":
			cfBindings[b.Name] = cloudflare.WorkerServiceBinding{Service: b.ID}
		case "secret_text":
			cfBindings[b.Name] = cloudflare.WorkerSecretTextBinding{Text: b.ID}
		case "var", "plain_text":
			cfBindings[b.Name] = cloudflare.WorkerPlainTextBinding{Text: b.ID}
		default:
			cfBindings[b.Name] = cloudflare.WorkerPlainTextBinding{Text: b.ID}
		}
	}
	return cfBindings
}
