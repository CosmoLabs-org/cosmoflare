package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// logpushKnownDatasets lists datasets Logpush commonly ships. It exists only
// for the help-text hint on create — Cloudflare owns the real validation, so
// an unknown name is passed through untouched rather than blocked.
var logpushKnownDatasets = map[string]bool{
	"http_requests":              true,
	"http_requests_crawlerhints": true,
	"spectrum_events":            true,
	"firewalls":                  true,
	"dns_logs":                   true,
	"nel_reports":                true,
	"analytics_engine_datasets":  true,
	"workers_trace_events":       true,
	"ghost_locator":              true,
	"instant_logs":               true,
}

var (
	logpushName        string
	logpushDataset     string
	logpushDestination string
	logpushFrequency   string
	logpushForce       bool
	logpushEnable      bool
	logpushDisable     bool
)

var logpushCmd = &cobra.Command{
	Use:   "logpush",
	Short: "Manage Logpush jobs",
	Long: `Cloudflare Logpush management — create, inspect, update, and delete
account Logpush jobs, plus destination ownership verification.

Commands:
  job create       Create a Logpush job for a dataset
  job list         List all Logpush jobs in the account
  job get          Get a job's details by ID
  job update       Update a job (partial; unspecified fields are kept)
  job delete       Delete a job (irreversible; dry-run by default)
  ownership verify  Get and validate a destination ownership challenge

Logpush continuously uploads a dataset's logs (http_requests, firewalls,
dns_logs, ...) to a destination you own — an R2/S3/GCS bucket, Azure blob
storage, Datadog, or an HTTPS endpoint. All commands here are account-scoped
and require an API token with the account-level "Logs" permission; writes
("Logs: Edit") for every operation including reads, per the permission
dataset this CLI is built against.

Wrangler has no Logpush equivalent — this group is the observability
surface of the CLI.

Examples:
  cosmoflare logpush job list --json
  cosmoflare logpush job create --dataset http_requests --destination-conf 'r2://my-bucket/logs?account=...' --name edge-logs
  cosmoflare logpush ownership verify --dataset http_requests --destination-conf 'r2://my-bucket/logs?account=...'`,
}

var logpushJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Manage Logpush jobs",
	Long: `Create, list, get, update, and delete Logpush jobs.

A job binds one dataset to one destination_conf. New jobs start enabled;
pause one with 'job update <id> --disable'.`,
}

var logpushJobCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Logpush job",
	Long: `Create a Logpush job that ships a dataset to a destination.

The destination must already be verified (see 'logpush ownership verify')
when the destination type requires a challenge file; the create call fails
with the API's guidance otherwise.

Common datasets: http_requests, firewalls, dns_logs, spectrum_events,
nel_reports, workers_trace_events (non-exhaustive — Cloudflare validates
the real list server-side).

Frequency accepts "high" (whenever files fill) or "low" (every 5 minutes,
the default when omitted).

Examples:
  cosmoflare logpush job create --dataset http_requests --destination-conf 'r2://logs-bucket/http?account=<acct>' --name edge-logs
  cosmoflare logpush job create --dataset firewalls --destination-conf 'datadog://?ddsource=cfwaf' --frequency high --json`,
	RunE: runLogpushJobCreate,
}

var logpushJobListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Logpush jobs",
	Long: `List every Logpush job in the account, across all datasets.

Examples:
  cosmoflare logpush job list
  cosmoflare logpush job list --json`,
	RunE: runLogpushJobList,
}

var logpushJobGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a Logpush job's details",
	Long: `Get the details of one Logpush job by its numeric ID (see
'logpush job list' for IDs).

Examples:
  cosmoflare logpush job get 100237
  cosmoflare logpush job get 100237 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runLogpushJobGet,
}

var logpushJobUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a Logpush job",
	Long: `Update a Logpush job with partial changes.

Only the flags you pass change the job; everything else keeps its current
value. The Cloudflare update is a full replace under the hood, so this
command fetches the current job first and merges your changes on top.

Use --disable to pause log delivery (the job and its config are kept) and
--enable to resume it.

Examples:
  cosmoflare logpush job update 100237 --name edge-logs-v2
  cosmoflare logpush job update 100237 --disable
  cosmoflare logpush job update 100237 --destination-conf 'r2://other-bucket/logs?account=<acct>'`,
	Args: cobra.ExactArgs(1),
	RunE: runLogpushJobUpdate,
}

var logpushJobDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a Logpush job",
	Long: `Delete a Logpush job by its numeric ID.

WARNING: This action is irreversible. Log delivery for the job's dataset
stops immediately; the destination and any already-uploaded files are
untouched.

This command is registry-flagged destructive: it runs as a dry-run unless
--force is passed.

Examples:
  cosmoflare logpush job delete 100237
  cosmoflare logpush job delete 100237 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runLogpushJobDelete,
}

var logpushOwnershipCmd = &cobra.Command{
	Use:   "ownership",
	Short: "Verify Logpush destination ownership",
	Long: `Destination ownership verification for Logpush.

Before a job can push to a bucket or endpoint, Cloudflare must see proof
you control the destination. 'verify' performs the two-step flow in one
call: it requests the challenge filename for the destination, then
validates the challenge file's contents once you have uploaded it.`,
}

var logpushOwnershipVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Get and validate a destination ownership challenge",
	Long: `Request the ownership challenge for a destination, then validate it.

Step one (challenge): the API returns a filename that must exist at the
destination root, e.g. "challenge-xyz.txt" in the bucket.

Step two (validate): download that challenge file from the destination and
pass its contents via --challenge. The API confirms it can read the file
back and marks the destination verified.

Omitting --challenge runs only step one and prints the filename to upload.

Examples:
  cosmoflare logpush ownership verify --dataset http_requests --destination-conf 'r2://logs-bucket/http?account=<acct>'
  cosmoflare logpush ownership verify --dataset http_requests --destination-conf 'r2://logs-bucket/http?account=<acct>' --challenge "$(wrangler r2 object get logs-bucket/challenge-xyz.txt --file=/dev/stdout)" --json`,
	RunE: runLogpushOwnershipVerify,
}

var logpushOwnershipChallengeFlag string

func init() {
	rootCmd.AddCommand(logpushCmd)

	logpushCmd.AddCommand(logpushJobCmd, logpushOwnershipCmd)
	logpushJobCmd.AddCommand(
		logpushJobCreateCmd,
		logpushJobListCmd,
		logpushJobGetCmd,
		logpushJobUpdateCmd,
		logpushJobDeleteCmd,
	)
	logpushOwnershipCmd.AddCommand(logpushOwnershipVerifyCmd)

	logpushJobCreateCmd.Flags().StringVar(&logpushDataset, "dataset", "", "Dataset to push (e.g. http_requests, firewalls, dns_logs)")
	logpushJobCreateCmd.Flags().StringVar(&logpushDestination, "destination-conf", "", "Destination URI with credentials and parameters (r2://bucket/prefix?...)")
	logpushJobCreateCmd.Flags().StringVar(&logpushName, "name", "", "Human-readable job name")
	logpushJobCreateCmd.Flags().StringVar(&logpushFrequency, "frequency", "", "Push frequency: high (when files fill) or low (every 5 min)")

	logpushJobUpdateCmd.Flags().StringVar(&logpushName, "name", "", "New job name")
	logpushJobUpdateCmd.Flags().StringVar(&logpushDataset, "dataset", "", "New dataset")
	logpushJobUpdateCmd.Flags().StringVar(&logpushDestination, "destination-conf", "", "New destination URI")
	logpushJobUpdateCmd.Flags().StringVar(&logpushFrequency, "frequency", "", "New push frequency (high or low)")
	logpushJobUpdateCmd.Flags().BoolVar(&logpushEnable, "enable", false, "Resume log delivery")
	logpushJobUpdateCmd.Flags().BoolVar(&logpushDisable, "disable", false, "Pause log delivery")

	logpushJobDeleteCmd.Flags().BoolVar(&logpushForce, "force", false, "Skip the destructive dry-run default")

	logpushOwnershipVerifyCmd.Flags().StringVar(&logpushDataset, "dataset", "", "Dataset the destination will serve")
	logpushOwnershipVerifyCmd.Flags().StringVar(&logpushDestination, "destination-conf", "", "Destination URI to verify")
	logpushOwnershipVerifyCmd.Flags().StringVar(&logpushOwnershipChallengeFlag, "challenge", "", "Contents of the uploaded challenge file (omit to only fetch the filename)")
}

func getLogpushService() (*cosmoflare.LogpushService, error) {
	return cosmoflare.NewLogpushServiceFromCreds(AccountID, APIToken)
}

// logpushResetFlags returns the logpush flag variables to their zero values
// so each test subtest starts from a pristine command state.
func logpushResetFlags() {
	logpushName = ""
	logpushDataset = ""
	logpushDestination = ""
	logpushFrequency = ""
	logpushForce = false
	logpushEnable = false
	logpushDisable = false
	logpushOwnershipChallengeFlag = ""
}

func runLogpushJobCreate(cmd *cobra.Command, args []string) error {
	if logpushDataset == "" {
		return outErrf("dataset is required (--dataset, e.g. http_requests)")
	}
	if logpushDestination == "" {
		return outErrf("destination_conf is required (--destination-conf, e.g. r2://bucket/prefix?account=<acct>)")
	}

	if !logpushKnownDatasets[logpushDataset] {
		printInfo("note: %q is not in the common dataset list; Cloudflare will validate it server-side", logpushDataset)
	}

	if DryRun {
		return outPayload("DRY RUN: Would create logpush job", func() any {
			return map[string]string{
				"name":             logpushName,
				"dataset":          logpushDataset,
				"destination_conf": logpushDestination,
				"frequency":        logpushFrequency,
			}
		}, func() {
			printInfo("DRY RUN: Would create logpush job for dataset %q -> %s (name=%q frequency=%q)", logpushDataset, logpushDestination, logpushName, logpushFrequency)
		})
	}

	svc, err := getLogpushService()
	if err != nil {
		return outErr("failed to create logpush service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	job, err := svc.Create(context.Background(), cosmoflare.LogpushJobCreate{
		Name:            logpushName,
		Dataset:         logpushDataset,
		DestinationConf: logpushDestination,
		Frequency:       logpushFrequency,
	})
	if err != nil {
		return outErr("failed to create logpush job", err)
	}

	return outPayload("Logpush job created successfully", func() any {
		return job
	}, func() {
		printSuccess("Logpush job %d created for dataset %q", job.ID, job.Dataset)
		printInfo("Verify delivery state with: cosmoflare logpush job get %d", job.ID)
	})
}

func runLogpushJobList(cmd *cobra.Command, args []string) error {
	svc, err := getLogpushService()
	if err != nil {
		return outErr("failed to create logpush service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	jobs, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list logpush jobs", err)
	}

	return outResult(jobs, func() {
		if len(jobs) == 0 {
			printInfo("No logpush jobs found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tDATASET\tENABLED\tFREQUENCY\tDESTINATION")
		for _, j := range jobs {
			fmt.Fprintf(w, "%d\t%s\t%s\t%v\t%s\t%s\n",
				j.ID, j.Name, j.Dataset, j.Enabled, logpushFreqString(j.Frequency), j.DestinationConf)
		}
		w.Flush()
		printInfo("Total: %d job(s)", len(jobs))
	})
}

func runLogpushJobGet(cmd *cobra.Command, args []string) error {
	jobID, err := logpushParseID(args)
	if err != nil {
		return outErr("failed to parse job ID", err)
	}

	svc, err := getLogpushService()
	if err != nil {
		return outErr("failed to create logpush service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	job, err := svc.Get(context.Background(), jobID)
	if err != nil {
		return outErr(fmt.Sprintf("failed to get logpush job %d", jobID), err)
	}

	return outResult(job, func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID:\t%d\n", job.ID)
		fmt.Fprintf(w, "Name:\t%s\n", job.Name)
		fmt.Fprintf(w, "Dataset:\t%s\n", job.Dataset)
		fmt.Fprintf(w, "Enabled:\t%v\n", job.Enabled)
		fmt.Fprintf(w, "Frequency:\t%s\n", logpushFreqString(job.Frequency))
		fmt.Fprintf(w, "Destination:\t%s\n", job.DestinationConf)
		if job.LastComplete != "" {
			fmt.Fprintf(w, "Last complete:\t%s\n", job.LastComplete)
		}
		if job.LastError != "" {
			fmt.Fprintf(w, "Last error:\t%s\n", job.LastError)
		}
		if job.ErrorMessage != "" {
			fmt.Fprintf(w, "Error message:\t%s\n", job.ErrorMessage)
		}
		w.Flush()
	})
}

func runLogpushJobUpdate(cmd *cobra.Command, args []string) error {
	jobID, err := logpushParseID(args)
	if err != nil {
		return outErr("failed to parse job ID", err)
	}
	if logpushEnable && logpushDisable {
		return outErrf("--enable and --disable are mutually exclusive")
	}

	var enabled *bool
	if logpushEnable {
		t := true
		enabled = &t
	} else if logpushDisable {
		f := false
		enabled = &f
	}

	if DryRun {
		return outPayload("DRY RUN: Would update logpush job", func() any {
			return map[string]any{
				"job_id":           jobID,
				"name":             logpushName,
				"dataset":          logpushDataset,
				"destination_conf": logpushDestination,
				"frequency":        logpushFrequency,
				"enabled":          enabled,
			}
		}, func() {
			printInfo("DRY RUN: Would update logpush job %d (name=%q dataset=%q destination=%q frequency=%q enabled=%v)",
				jobID, logpushName, logpushDataset, logpushDestination, logpushFrequency, enabled)
		})
	}

	svc, err := getLogpushService()
	if err != nil {
		return outErr("failed to create logpush service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	job, err := svc.Update(context.Background(), jobID, cosmoflare.LogpushJobUpdate{
		Name:            logpushName,
		Dataset:         logpushDataset,
		DestinationConf: logpushDestination,
		Frequency:       logpushFrequency,
		Enabled:         enabled,
	})
	if err != nil {
		return outErr(fmt.Sprintf("failed to update logpush job %d", jobID), err)
	}

	return outPayload("Logpush job updated successfully", func() any {
		return job
	}, func() {
		printSuccess("Logpush job %d updated (enabled=%v)", job.ID, job.Enabled)
	})
}

func runLogpushJobDelete(cmd *cobra.Command, args []string) error {
	jobID, err := logpushParseID(args)
	if err != nil {
		return outErr("failed to parse job ID", err)
	}

	cliPath := cliPathOfCmdOr(cmd, "logpush job delete")
	// Registry-flagged destructive: runs dry unless --force, so no prompt.
	dry := destructiveDryRun(cliPath, logpushForce)
	if dry {
		return outPayload("DRY RUN: Would delete logpush job", func() any {
			return map[string]any{"job_id": jobID}
		}, func() {
			printInfo("DRY RUN: Would delete logpush job %d (irreversible; pass --force to execute)", jobID)
		})
	}

	svc, err := getLogpushService()
	if err != nil {
		return outErr("failed to create logpush service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	if err := svc.Delete(context.Background(), jobID); err != nil {
		auditMutation(cliPath, strconv.Itoa(jobID), false)
		return outErr("failed to delete logpush job", err)
	}
	auditMutation(cliPath, strconv.Itoa(jobID), true)

	return outPayload("Logpush job deleted successfully", func() any {
		return map[string]any{"job_id": jobID}
	}, func() {
		printSuccess("Logpush job %d deleted successfully!", jobID)
	})
}

func runLogpushOwnershipVerify(cmd *cobra.Command, args []string) error {
	if logpushDataset == "" {
		return outErrf("dataset is required (--dataset, e.g. http_requests)")
	}
	if logpushDestination == "" {
		return outErrf("destination_conf is required (--destination-conf, e.g. r2://bucket/prefix?account=<acct>)")
	}

	svc, err := getLogpushService()
	if err != nil {
		return outErr("failed to create logpush service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	challenge, err := svc.OwnershipChallenge(context.Background(), logpushDestination)
	if err != nil {
		return outErr("failed to get ownership challenge (is the destination URI well-formed and reachable?)", err)
	}

	if logpushOwnershipChallengeFlag == "" {
		// Step one only: report the filename to upload.
		return outPayload("Ownership challenge issued", func() any {
			return map[string]string{
				"dataset":          logpushDataset,
				"destination_conf": logpushDestination,
				"filename":         challenge.Filename,
				"message":          challenge.Message,
			}
		}, func() {
			printSuccess("Ownership challenge for %s:", logpushDestination)
			printInfo("Upload a file named %q to the destination, then re-run with --challenge <file contents>", challenge.Filename)
		})
	}

	valid, err := svc.OwnershipValidate(context.Background(), logpushDestination, logpushOwnershipChallengeFlag)
	if err != nil {
		return outErr("failed to validate ownership challenge", err)
	}

	return outPayload("Ownership validation complete", func() any {
		return map[string]any{
			"dataset":          logpushDataset,
			"destination_conf": logpushDestination,
			"valid":            valid,
		}
	}, func() {
		if valid {
			printSuccess("Destination %s is verified — jobs can now push to it", logpushDestination)
		} else {
			printWarning("Destination %s is NOT verified — check the challenge file contents", logpushDestination)
		}
	})
}

// logpushParseID converts the command argument to a job ID. Logpush job IDs
// are plain integers; no prefix parsing applies (TASK-011 rule).
func logpushParseID(args []string) (int, error) {
	if len(args) != 1 || args[0] == "" {
		return 0, fmt.Errorf("logpush job ID is required (a plain integer, e.g. 100237)")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("logpush job ID must be a positive integer, got %q", args[0])
	}
	return id, nil
}

// logpushFreqString renders the frequency with the API default made
// explicit: empty means "low" (5-minute batches).
func logpushFreqString(f string) string {
	if f == "" {
		return "low"
	}
	return f
}
