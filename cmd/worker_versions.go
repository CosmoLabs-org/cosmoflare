package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// Worker script VERSIONS subcommands (FEAT-021).
//
// cloudflare-go v0.116 has no versions API, so the library layer (see
// pkg/cosmoflare/worker_versions.go) issues raw REST calls and therefore
// requires credentials-based service construction (NewWorkerServiceFromCreds).

var (
	workerVersionScript     string
	workerVersionCompatDate string
	workerVersionModule     bool
	workerVersionForce      bool
)

var workerVersionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "Manage Worker script versions",
	Long: `Manage immutable versions of a Worker script.

Each upload creates a new version; deploying a version points the Worker's
live deployment at it. Rollback is a redeploy of an earlier version ID.

Commands:
  upload    Upload a new script version (does not deploy)
  list      List all versions of a Worker
  view      Show one version in detail
  deploy    Promote a version to the live deployment
  delete    Delete an undeployed version
  rollback  Redeploy an earlier version

Examples:
  cosmoflare worker versions upload my-worker --script=worker.js
  cosmoflare worker versions upload my-worker --script=worker.js --compatibility-date=2024-09-01 --module
  cosmoflare worker versions list my-worker --json
  cosmoflare worker versions view my-worker <version-id>
  cosmoflare worker versions deploy my-worker <version-id>
  cosmoflare worker versions delete my-worker <version-id>
  cosmoflare worker versions rollback my-worker <version-id>`,
}

var workerVersionUploadCmd = &cobra.Command{
	Use:   "upload [name]",
	Short: "Upload a new Worker script version",
	Long: `Upload a new immutable version of a Worker script.

The version is created but NOT deployed — promote it afterwards with
"worker versions deploy".

Examples:
  cosmoflare worker versions upload my-worker --script=worker.js
  cosmoflare worker versions upload my-worker --script=worker.js --compatibility-date=2024-09-01
  cosmoflare worker versions upload my-worker --script=worker.js --module`,
	RunE: runWorkerVersionUpload,
}

var workerVersionListCmd = &cobra.Command{
	Use:   "list [name]",
	Short: "List Worker script versions",
	Long: `List every version of a Worker script, newest last.

Examples:
  cosmoflare worker versions list my-worker
  cosmoflare worker versions list my-worker --json`,
	RunE: runWorkerVersionList,
}

var workerVersionViewCmd = &cobra.Command{
	Use:   "view [name] [version-id]",
	Short: "Show one Worker script version",
	Long: `Show the details of a single Worker script version.

Examples:
  cosmoflare worker versions view my-worker 0f2ac1af-3fb2-4bf0-a40f-3300e6172e0f
  cosmoflare worker versions view my-worker <version-id> --json`,
	RunE: runWorkerVersionView,
}

var workerVersionDeployCmd = &cobra.Command{
	Use:   "deploy [name] [version-id]",
	Short: "Deploy a Worker script version",
	Long: `Promote a version to the Worker's live deployment.

Examples:
  cosmoflare worker versions deploy my-worker 0f2ac1af-3fb2-4bf0-a40f-3300e6172e0f`,
	RunE: runWorkerVersionDeploy,
}

var workerVersionDeleteCmd = &cobra.Command{
	Use:   "delete [name] [version-id]",
	Short: "Delete a Worker script version",
	Long: `Delete a version that is not currently deployed.

WARNING: deleting the live version is rejected by the API. This action is
irreversible.

Examples:
  cosmoflare worker versions delete my-worker 0f2ac1af-3fb2-4bf0-a40f-3300e6172e0f
  cosmoflare worker versions delete my-worker <version-id> --force`,
	RunE: runWorkerVersionDelete,
}

var workerVersionRollbackCmd = &cobra.Command{
	Use:   "rollback [name] [version-id]",
	Short: "Roll back to a previous Worker script version",
	Long: `Roll the Worker back by redeploying an earlier version.

There is no dedicated rollback endpoint: rollback deploys the given
version_id, making it live again.

Examples:
  cosmoflare worker versions rollback my-worker 0f2ac1af-3fb2-4bf0-a40f-3300e6172e0f`,
	RunE: runWorkerVersionRollback,
}

// registerWorkerVersionCmds wires the "versions" command group onto parent.
func registerWorkerVersionCmds(parent *cobra.Command) {
	parent.AddCommand(workerVersionsCmd)

	workerVersionsCmd.AddCommand(workerVersionUploadCmd)
	workerVersionsCmd.AddCommand(workerVersionListCmd)
	workerVersionsCmd.AddCommand(workerVersionViewCmd)
	workerVersionsCmd.AddCommand(workerVersionDeployCmd)
	workerVersionsCmd.AddCommand(workerVersionDeleteCmd)
	workerVersionsCmd.AddCommand(workerVersionRollbackCmd)

	if workerVersionUploadCmd.Flags().Lookup("script") == nil {
		workerVersionUploadCmd.Flags().StringVarP(&workerVersionScript, "script", "s", "", "Path to Worker script file")
	}
	if workerVersionUploadCmd.Flags().Lookup("compatibility-date") == nil {
		workerVersionUploadCmd.Flags().StringVar(&workerVersionCompatDate, "compatibility-date", "", "Workers runtime compatibility date (yyyy-mm-dd)")
	}
	if workerVersionUploadCmd.Flags().Lookup("module") == nil {
		workerVersionUploadCmd.Flags().BoolVar(&workerVersionModule, "module", false, "Treat script as ES module")
	}

	if workerVersionDeleteCmd.Flags().Lookup("force") == nil {
		workerVersionDeleteCmd.Flags().BoolVar(&workerVersionForce, "force", false, "Skip confirmation prompt")
	}
}

// workerVersionArgs validates the positional arguments of a command that
// takes a worker name and (optionally) a version ID.
func workerVersionArgs(args []string, needVersion bool) (string, string, error) {
	if len(args) < 1 || args[0] == "" {
		return "", "", fmt.Errorf("worker name is required")
	}
	name := args[0]
	if !needVersion {
		return name, "", nil
	}
	if len(args) < 2 || args[1] == "" {
		return "", "", fmt.Errorf("version ID is required")
	}
	return name, args[1], nil
}

func runWorkerVersionUpload(cmd *cobra.Command, args []string) error {
	name, _, err := workerVersionArgs(args, false)
	if err != nil {
		return err
	}

	if workerVersionScript == "" {
		return fmt.Errorf("script file is required (--script or -s)")
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	f, err := os.Open(workerVersionScript)
	if err != nil {
		return outErr("failed to open script file", err)
	}
	defer f.Close()

	var opts []cosmoflare.WorkerOption
	if workerVersionCompatDate != "" {
		opts = append(opts, cosmoflare.WithWorkerCompatibilityDate(workerVersionCompatDate))
	}
	if len(workerBindings) > 0 {
		bindings, err := parseWorkerBindings(workerBindings)
		if err != nil {
			return err
		}
		opts = append(opts, cosmoflare.WithWorkerBindings(bindings))
	}
	if len(workerTags) > 0 {
		opts = append(opts, cosmoflare.WithWorkerTags(workerTags))
	}
	if workerVersionModule {
		opts = append(opts, cosmoflare.WithWorkerModule(true))
	}

	if DryRun {
		return outPayload("DRY RUN: Would upload worker version", func() any {
			return map[string]string{"name": name, "script": workerVersionScript}
		}, func() {
			printInfo("DRY RUN: Would upload a new version of worker '%s' from %s", name, workerVersionScript)
		})
	}

	version, err := svc.VersionUpload(context.Background(), name, f, opts...)
	if err != nil {
		return outErr("failed to upload worker version", err)
	}

	return outPayload("Worker version uploaded", func() any {
		return version
	}, func() {
		printSuccess("Version %d (%s) of worker '%s' uploaded", version.Number, version.ID, name)
		printInfo("Promote it with: cosmoflare worker versions deploy %s %s", name, version.ID)
	})
}

func runWorkerVersionList(cmd *cobra.Command, args []string) error {
	name, _, err := workerVersionArgs(args, false)
	if err != nil {
		return err
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	versions, err := svc.VersionList(context.Background(), name)
	if err != nil {
		return outErr("failed to list worker versions", err)
	}

	return outResult(versions, func() {
		if len(versions) == 0 {
			printInfo("No versions found for worker '%s'", name)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNUMBER\tCREATED")
		for _, v := range versions {
			fmt.Fprintf(w, "%s\t%d\t%s\n",
				v.ID,
				v.Number,
				v.CreatedAt.Format("2006-01-02 15:04:05"),
			)
		}
		w.Flush()

		printInfo("Total: %d version(s)", len(versions))
	})
}

func runWorkerVersionView(cmd *cobra.Command, args []string) error {
	name, versionID, err := workerVersionArgs(args, true)
	if err != nil {
		return err
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	version, err := svc.VersionGet(context.Background(), name, versionID)
	if err != nil {
		return outErr("failed to get worker version", err)
	}

	return outResult(version, func() {
		fmt.Printf("Worker: %s\n", name)
		fmt.Printf("Version ID: %s\n", version.ID)
		fmt.Printf("Number: %d\n", version.Number)
		fmt.Printf("Created: %s\n", version.CreatedAt.Format("2006-01-02 15:04:05"))
		if version.Source != "" {
			fmt.Printf("Source: %s\n", version.Source)
		}
	})
}

func runWorkerVersionDeploy(cmd *cobra.Command, args []string) error {
	name, versionID, err := workerVersionArgs(args, true)
	if err != nil {
		return err
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would deploy worker version", func() any {
			return map[string]string{"name": name, "version_id": versionID}
		}, func() {
			printInfo("DRY RUN: Would deploy version %s of worker '%s'", versionID, name)
		})
	}

	deployment, err := svc.VersionDeploy(context.Background(), name, versionID)
	if err != nil {
		return outErr("failed to deploy worker version", err)
	}

	return outPayload("Worker version deployed", func() any {
		return deployment
	}, func() {
		printSuccess("Version %s of worker '%s' is now live (deployment %s)", versionID, name, deployment.ID)
	})
}

func runWorkerVersionDelete(cmd *cobra.Command, args []string) error {
	name, versionID, err := workerVersionArgs(args, true)
	if err != nil {
		return err
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if !workerVersionForce && !DryRun && !confirmWorkerVersionDelete(name, versionID) {
		printInfo("Version deletion cancelled")
		return nil
	}

	if DryRun {
		return outPayload("DRY RUN: Would delete worker version", func() any {
			return map[string]string{"name": name, "version_id": versionID}
		}, func() {
			printInfo("DRY RUN: Would delete version %s of worker '%s'", versionID, name)
		})
	}

	if err := svc.VersionDelete(context.Background(), name, versionID); err != nil {
		return outErr("failed to delete worker version", err)
	}

	return outPayload("Worker version deleted", func() any {
		return map[string]string{"name": name, "version_id": versionID}
	}, func() {
		printSuccess("Version %s of worker '%s' deleted", versionID, name)
	})
}

// confirmWorkerVersionDelete asks for interactive confirmation. It always
// returns false when stdin is not a usable terminal-safe reader (Scanln
// returns an error), so scripted pipelines must pass --force.
func confirmWorkerVersionDelete(name, versionID string) bool {
	fmt.Printf("Are you sure you want to delete version %s of worker '%s'? [y/N]: ", versionID, name)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.TrimSpace(strings.ToLower(response)) == "y" ||
		strings.TrimSpace(strings.ToLower(response)) == "yes"
}

func runWorkerVersionRollback(cmd *cobra.Command, args []string) error {
	name, versionID, err := workerVersionArgs(args, true)
	if err != nil {
		return err
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would roll back worker", func() any {
			return map[string]string{"name": name, "version_id": versionID}
		}, func() {
			printInfo("DRY RUN: Would roll back worker '%s' to version %s", name, versionID)
		})
	}

	deployment, err := svc.VersionRollback(context.Background(), name, versionID)
	if err != nil {
		return outErr("failed to roll back worker", err)
	}

	return outPayload("Worker rolled back", func() any {
		return deployment
	}, func() {
		printSuccess("Worker '%s' rolled back to version %s (deployment %s)", name, versionID, deployment.ID)
	})
}
