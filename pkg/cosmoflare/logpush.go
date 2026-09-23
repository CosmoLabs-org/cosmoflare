package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// LogpushJob is the public view of an account-scoped Logpush job. Timestamps
// are rendered as RFC 3339 strings, empty when the API reports none.
type LogpushJob struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Dataset            string `json:"dataset"`
	Enabled            bool   `json:"enabled"`
	DestinationConf    string `json:"destination_conf"`
	OwnershipChallenge string `json:"ownership_challenge,omitempty"`
	Frequency          string `json:"frequency,omitempty"`
	LastComplete       string `json:"last_complete,omitempty"`
	LastError          string `json:"last_error,omitempty"`
	ErrorMessage       string `json:"error_message,omitempty"`
}

// LogpushOwnershipChallenge is the destination ownership challenge the API
// issues for a destination_conf. The caller uploads Filename to the
// destination, then validates the challenge file's contents.
type LogpushOwnershipChallenge struct {
	Filename string `json:"filename"`
	Valid    bool   `json:"valid"`
	Message  string `json:"message,omitempty"`
}

// LogpushJobCreate carries the required fields for job creation. Enabled
// defaults to true — a job that ships nothing on creation is almost always
// a misconfiguration.
type LogpushJobCreate struct {
	Name            string
	Dataset         string
	DestinationConf string
	Frequency       string
}

// LogpushJobUpdate carries optional changes to an existing job. Zero values
// leave the corresponding field untouched except Enabled, which is a pointer
// so "disable" is expressible.
type LogpushJobUpdate struct {
	Name            string
	Dataset         string
	DestinationConf string
	Frequency       string
	Enabled         *bool
}

// LogpushService implements account-scoped Logpush job operations on top of
// the typed cloudflare-go resources.
type LogpushService struct {
	cf        *cloudflare.API
	accountID string
}

// NewLogpushService creates a new Logpush service client.
func NewLogpushService(api *cloudflare.API, accountID string) (*LogpushService, error) {
	if api == nil {
		return nil, validationError("NewLogpushService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewLogpushService", "account ID is required")
	}
	return &LogpushService{cf: api, accountID: accountID}, nil
}

// NewLogpushServiceFromCreds creates a LogpushService from account ID and API token.
func NewLogpushServiceFromCreds(accountID, apiToken string) (*LogpushService, error) {
	if accountID == "" {
		return nil, validationError("NewLogpushServiceFromCreds", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewLogpushServiceFromCreds", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewLogpushServiceFromCreds", "failed to create Cloudflare API client", err)
	}
	return &LogpushService{cf: cf, accountID: accountID}, nil
}

// Create adds a new Logpush job for a dataset, pushing to destination_conf.
// The returned job carries the ID Cloudflare assigned.
func (s *LogpushService) Create(ctx context.Context, opts LogpushJobCreate) (*LogpushJob, error) {
	if opts.Dataset == "" {
		return nil, validationError("LogpushService.Create", "dataset is required")
	}
	if opts.DestinationConf == "" {
		return nil, validationError("LogpushService.Create", "destination_conf is required")
	}
	if s.accountID == "" {
		return nil, validationError("LogpushService.Create", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	job, err := s.cf.CreateLogpushJob(ctx, rc, cloudflare.CreateLogpushJobParams{
		Name:            opts.Name,
		Dataset:         opts.Dataset,
		DestinationConf: opts.DestinationConf,
		Frequency:       opts.Frequency,
		Enabled:         true,
	})
	if err != nil {
		return nil, newError("LogpushService.Create", fmt.Sprintf("failed to create logpush job for dataset %q", opts.Dataset), err)
	}
	mapped := logpushJobFromCF(*job)
	return &mapped, nil
}

// List returns every Logpush job in the account across all datasets.
func (s *LogpushService) List(ctx context.Context) ([]LogpushJob, error) {
	if s.accountID == "" {
		return nil, validationError("LogpushService.List", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	jobs, err := s.cf.ListLogpushJobs(ctx, rc, cloudflare.ListLogpushJobsParams{})
	if err != nil {
		return nil, newError("LogpushService.List", "failed to list logpush jobs", err)
	}
	out := make([]LogpushJob, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, logpushJobFromCF(j))
	}
	return out, nil
}

// Get retrieves a single Logpush job by its numeric ID.
func (s *LogpushService) Get(ctx context.Context, jobID int) (*LogpushJob, error) {
	if jobID <= 0 {
		return nil, validationError("LogpushService.Get", "job ID must be a positive integer")
	}
	if s.accountID == "" {
		return nil, validationError("LogpushService.Get", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	job, err := s.cf.GetLogpushJob(ctx, rc, jobID)
	if err != nil {
		return nil, newError("LogpushService.Get", fmt.Sprintf("failed to get logpush job %d", jobID), err)
	}
	mapped := logpushJobFromCF(job)
	return &mapped, nil
}

// Update applies partial changes to an existing Logpush job. The Cloudflare
// PUT replaces the whole job, so the current state is fetched first and the
// non-zero update fields merged on top; the returned job reflects the state
// after the update.
func (s *LogpushService) Update(ctx context.Context, jobID int, opts LogpushJobUpdate) (*LogpushJob, error) {
	if jobID <= 0 {
		return nil, validationError("LogpushService.Update", "job ID must be a positive integer")
	}
	if s.accountID == "" {
		return nil, validationError("LogpushService.Update", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	current, err := s.cf.GetLogpushJob(ctx, rc, jobID)
	if err != nil {
		return nil, newError("LogpushService.Update", fmt.Sprintf("failed to fetch logpush job %d before update", jobID), err)
	}

	params := cloudflare.UpdateLogpushJobParams{
		ID:                 jobID,
		Name:               current.Name,
		Dataset:            current.Dataset,
		Enabled:            current.Enabled,
		DestinationConf:    current.DestinationConf,
		OwnershipChallenge: current.OwnershipChallenge,
		Frequency:          current.Frequency,
		Filter:             current.Filter,
		OutputOptions:      current.OutputOptions,
		MaxUploadBytes:     current.MaxUploadBytes,
		MaxUploadRecords:   current.MaxUploadRecords,
	}
	if opts.Name != "" {
		params.Name = opts.Name
	}
	if opts.Dataset != "" {
		params.Dataset = opts.Dataset
	}
	if opts.DestinationConf != "" {
		params.DestinationConf = opts.DestinationConf
	}
	if opts.Frequency != "" {
		params.Frequency = opts.Frequency
	}
	if opts.Enabled != nil {
		params.Enabled = *opts.Enabled
	}

	if err := s.cf.UpdateLogpushJob(ctx, rc, params); err != nil {
		return nil, newError("LogpushService.Update", fmt.Sprintf("failed to update logpush job %d", jobID), err)
	}
	return s.Get(ctx, jobID)
}

// Delete permanently removes a Logpush job. This is irreversible: log data
// for the dataset stops flowing to the destination immediately.
func (s *LogpushService) Delete(ctx context.Context, jobID int) error {
	if jobID <= 0 {
		return validationError("LogpushService.Delete", "job ID must be a positive integer")
	}
	if s.accountID == "" {
		return validationError("LogpushService.Delete", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	if err := s.cf.DeleteLogpushJob(ctx, rc, jobID); err != nil {
		return newError("LogpushService.Delete", fmt.Sprintf("failed to delete logpush job %d", jobID), err)
	}
	return nil
}

// OwnershipChallenge requests a destination ownership challenge: the
// filename the caller must upload to the destination before the job can be
// created. An invalid destination surfaces as an error carrying the API's
// message.
func (s *LogpushService) OwnershipChallenge(ctx context.Context, destinationConf string) (*LogpushOwnershipChallenge, error) {
	if destinationConf == "" {
		return nil, validationError("LogpushService.OwnershipChallenge", "destination_conf is required")
	}
	if s.accountID == "" {
		return nil, validationError("LogpushService.OwnershipChallenge", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	challenge, err := s.cf.GetLogpushOwnershipChallenge(ctx, rc, cloudflare.GetLogpushOwnershipChallengeParams{
		DestinationConf: destinationConf,
	})
	if err != nil {
		return nil, newError("LogpushService.OwnershipChallenge", fmt.Sprintf("failed to get ownership challenge for destination %q", destinationConf), err)
	}
	return &LogpushOwnershipChallenge{
		Filename: challenge.Filename,
		Valid:    challenge.Valid,
		Message:  challenge.Message,
	}, nil
}

// OwnershipValidate checks the file-based ownership challenge: the caller
// uploads the challenge file's contents to the destination and this call
// confirms the API can read it back.
func (s *LogpushService) OwnershipValidate(ctx context.Context, destinationConf, challenge string) (bool, error) {
	if destinationConf == "" {
		return false, validationError("LogpushService.OwnershipValidate", "destination_conf is required")
	}
	if challenge == "" {
		return false, validationError("LogpushService.OwnershipValidate", "ownership challenge is required")
	}
	if s.accountID == "" {
		return false, validationError("LogpushService.OwnershipValidate", "account ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	valid, err := s.cf.ValidateLogpushOwnershipChallenge(ctx, rc, cloudflare.ValidateLogpushOwnershipChallengeParams{
		DestinationConf:    destinationConf,
		OwnershipChallenge: challenge,
	})
	if err != nil {
		return false, newError("LogpushService.OwnershipValidate", fmt.Sprintf("failed to validate ownership challenge for destination %q", destinationConf), err)
	}
	return valid, nil
}

// logpushJobFromCF maps the cloudflare-go job type to the public LogpushJob
// view, rendering timestamps as RFC 3339 strings and omitting zero times.
func logpushJobFromCF(j cloudflare.LogpushJob) LogpushJob {
	lastComplete := ""
	if j.LastComplete != nil {
		lastComplete = j.LastComplete.Format(time.RFC3339)
	}
	lastError := ""
	if j.LastError != nil {
		lastError = j.LastError.Format(time.RFC3339)
	}
	return LogpushJob{
		ID:                 j.ID,
		Name:               j.Name,
		Dataset:            j.Dataset,
		Enabled:            j.Enabled,
		DestinationConf:    j.DestinationConf,
		OwnershipChallenge: j.OwnershipChallenge,
		Frequency:          j.Frequency,
		LastComplete:       lastComplete,
		LastError:          lastError,
		ErrorMessage:       j.ErrorMessage,
	}
}
