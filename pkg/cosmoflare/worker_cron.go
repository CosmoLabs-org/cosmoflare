package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// WorkerCronTrigger is the public view of a cron trigger attached to a
// Worker script. The Cloudflare API models schedules as a full set on the
// script: there is no per-trigger CRUD, so every mutation is a replace of
// the whole schedule list.
type WorkerCronTrigger struct {
	Cron     string `json:"cron"`
	Modified string `json:"modified_on,omitempty"`
}

// CronList returns the cron schedules attached to a Worker, in the order
// the API returns them (insertion order, not sorted).
func (s *WorkerService) CronList(ctx context.Context, worker string) ([]WorkerCronTrigger, error) {
	if worker == "" {
		return nil, validationError("WorkerService.CronList", "worker name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	triggers, err := s.cf.ListWorkerCronTriggers(ctx, rc, cloudflare.ListWorkerCronTriggersParams{
		ScriptName: worker,
	})
	if err != nil {
		return nil, newError("WorkerService.CronList", fmt.Sprintf("failed to list cron triggers for worker %q", worker), err)
	}
	return workerCronTriggersFromCF(triggers), nil
}

// CronReplace atomically replaces the full set of cron schedules on a
// Worker. An empty schedules slice removes all triggers. The returned
// slice is the API's view of the schedules after the replace.
func (s *WorkerService) CronReplace(ctx context.Context, worker string, schedules []string) ([]WorkerCronTrigger, error) {
	if worker == "" {
		return nil, validationError("WorkerService.CronReplace", "worker name is required")
	}

	crons := make([]cloudflare.WorkerCronTrigger, 0, len(schedules))
	for _, expr := range schedules {
		crons = append(crons, cloudflare.WorkerCronTrigger{Cron: expr})
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	triggers, err := s.cf.UpdateWorkerCronTriggers(ctx, rc, cloudflare.UpdateWorkerCronTriggersParams{
		ScriptName: worker,
		Crons:      crons,
	})
	if err != nil {
		return nil, newError("WorkerService.CronReplace", fmt.Sprintf("failed to replace cron triggers for worker %q", worker), err)
	}
	return workerCronTriggersFromCF(triggers), nil
}

// CronCreate appends one cron expression to a Worker's schedule set.
// Duplicates are rejected before the API call so a repeated create is a
// no-op rather than a replace round trip.
func (s *WorkerService) CronCreate(ctx context.Context, worker, cron string) ([]WorkerCronTrigger, error) {
	if worker == "" {
		return nil, validationError("WorkerService.CronCreate", "worker name is required")
	}
	if cron == "" {
		return nil, validationError("WorkerService.CronCreate", "cron expression is required")
	}

	current, err := s.CronList(ctx, worker)
	if err != nil {
		return nil, err
	}
	for _, t := range current {
		if t.Cron == cron {
			return nil, validationError("WorkerService.CronCreate", fmt.Sprintf("cron %q already exists on worker %q", cron, worker))
		}
	}

	schedules := make([]string, 0, len(current)+1)
	for _, t := range current {
		schedules = append(schedules, t.Cron)
	}
	schedules = append(schedules, cron)
	return s.CronReplace(ctx, worker, schedules)
}

// CronDelete removes one cron expression from a Worker's schedule set.
// Deleting an expression that is not present is an error, not a no-op.
func (s *WorkerService) CronDelete(ctx context.Context, worker, cron string) ([]WorkerCronTrigger, error) {
	if worker == "" {
		return nil, validationError("WorkerService.CronDelete", "worker name is required")
	}
	if cron == "" {
		return nil, validationError("WorkerService.CronDelete", "cron expression is required")
	}

	current, err := s.CronList(ctx, worker)
	if err != nil {
		return nil, err
	}
	found := false
	schedules := make([]string, 0, len(current))
	for _, t := range current {
		if t.Cron == cron {
			found = true
			continue
		}
		schedules = append(schedules, t.Cron)
	}
	if !found {
		return nil, validationError("WorkerService.CronDelete", fmt.Sprintf("cron %q not found on worker %q", cron, worker))
	}
	return s.CronReplace(ctx, worker, schedules)
}

// workerCronTriggersFromCF maps the cloudflare-go trigger type to the
// public WorkerCronTrigger view, rendering timestamps as RFC3339 strings
// and omitting zero times entirely.
func workerCronTriggersFromCF(triggers []cloudflare.WorkerCronTrigger) []WorkerCronTrigger {
	out := make([]WorkerCronTrigger, 0, len(triggers))
	for _, t := range triggers {
		modified := ""
		if t.ModifiedOn != nil {
			modified = t.ModifiedOn.Format(time.RFC3339)
		}
		out = append(out, WorkerCronTrigger{Cron: t.Cron, Modified: modified})
	}
	return out
}
