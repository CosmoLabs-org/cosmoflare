package cmd

import (
	"os"
	"strings"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"

	"github.com/CosmoLabs-org/cosmoflare/internal/cmdmanifest"
)

// auditLogPathEnv lets tests (and operators) redirect the mutation audit
// log without touching the home-directory default.
const auditLogPathEnv = "COSMOFLARE_AUDIT_LOG"

// mutationAuditLogPath returns the audit log destination: the
// COSMOFLARE_AUDIT_LOG override when set, otherwise the default path.
func mutationAuditLogPath() string {
	if p := os.Getenv(auditLogPathEnv); p != "" {
		return p
	}
	return getAuditLogPath()
}

// auditMutation stamps one executed mutation into the audit log, decorated
// with the command registry's danger metadata (FEAT-020 wave 1 consumer).
// Failures are never surfaced — audit must not break the audited command —
// and an unregistered path logs bare details so coverage gaps are visible
// in the log itself.
func auditMutation(cliPath []string, resource string, success bool) {
	logger, err := cosmoflare.NewAuditLogger(mutationAuditLogPath())
	if err != nil || logger == nil {
		return
	}

	details := map[string]string{}
	op := strings.Join(cliPath, " ")
	service, action := "", ""
	if c, ok := cmdmanifest.Load().ResolveCLI(cliPath...); ok {
		details["manifest_id"] = c.ID
		details["danger_level"] = c.DangerLevel
		if c.Destructive {
			details["destructive"] = "true"
		}
		service, action = c.Service, c.Verb
	} else {
		details["manifest_id"] = "unregistered"
	}
	logger.LogMutation(op, service, resource, action, details, success)
}
