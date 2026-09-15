package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// workerDomainCmd is the "worker domain" group command. Registration is
// exported (registerWorkerDomainCmds) so the orchestrator wires it onto
// the worker command without this file editing cmd/worker.go.
var workerDomainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage Worker custom domains",
	Long: `Manage custom domains attached to Cloudflare Workers.

A custom domain routes a hostname in one of your zones to a Worker
script. Attaching a domain creates or updates the routing so requests
to the hostname are handled by the Worker; detaching removes the
route but leaves the zone and its DNS records untouched.

Commands:
  list    List every custom domain in the account
  attach  Attach a hostname to a Worker
  detach  Detach a custom domain by ID

Examples:
  cosmoflare worker domain list`,
}

var workerDomainListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Worker custom domains",
	Long: `List every custom domain attached to a Worker in the account.

Each row shows the domain ID (needed for detach), the hostname, the
Worker script it routes to, and the zone that contains it. Output is
sorted by hostname.

Examples:
  cosmoflare worker domain list
  cosmoflare worker domain list --json`,
	RunE: runWorkerDomainList,
}

var workerDomainAttachCmd = &cobra.Command{
	Use:   "attach HOSTNAME",
	Short: "Attach a custom domain to a Worker",
	Long: `Attach a hostname to a Worker script so the Worker handles
requests for that hostname.

The hostname must live in a zone in your account; --zone takes that
zone's ID and --service names the Worker script. If the hostname is
already attached, the attachment is updated to the new service.

Examples:
  cosmoflare worker domain attach app.example.com --service=my-worker --zone=023e105f4ecef8ad9ca31a8372d0c353
  cosmoflare worker domain attach app.example.com --service=my-worker --zone=023e... --json`,
	RunE: runWorkerDomainAttach,
}

var workerDomainDetachCmd = &cobra.Command{
	Use:   "detach DOMAIN_ID",
	Short: "Detach a custom domain from its Worker",
	Long: `Detach a custom domain by its domain ID (shown by
'worker domain list').

The hostname immediately stops routing to the Worker. The zone and its
DNS records are not modified. Use --force to skip the confirmation
prompt.

Examples:
  cosmoflare worker domain detach dom-123 --force
  cosmoflare worker domain detach dom-123`,
	RunE: runWorkerDomainDetach,
}

var (
	workerDomainService string
	workerDomainZone    string
	workerDomainForce   bool
)

// registerWorkerDomainCmds wires the "worker domain" command group onto
// the given parent command (the worker command).
func registerWorkerDomainCmds(parent *cobra.Command) {
	parent.AddCommand(workerDomainCmd)

	workerDomainCmd.AddCommand(workerDomainListCmd)
	workerDomainCmd.AddCommand(workerDomainAttachCmd)
	workerDomainCmd.AddCommand(workerDomainDetachCmd)

	if workerDomainAttachCmd.Flags().Lookup("service") == nil {
		workerDomainAttachCmd.Flags().StringVar(&workerDomainService, "service", "", "Worker script name to route the hostname to")
	}
	if workerDomainAttachCmd.Flags().Lookup("zone") == nil {
		workerDomainAttachCmd.Flags().StringVar(&workerDomainZone, "zone", "", "Zone ID that contains the hostname")
	}
	if workerDomainDetachCmd.Flags().Lookup("force") == nil {
		workerDomainDetachCmd.Flags().BoolVar(&workerDomainForce, "force", false, "Skip confirmation prompt")
	}
}

func runWorkerDomainList(cmd *cobra.Command, args []string) error {
	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	domains, err := svc.DomainList(context.Background())
	if err != nil {
		return outErr("failed to list worker domains", err)
	}

	return outResult(domains, func() {
		if len(domains) == 0 {
			printInfo("No custom domains attached to workers")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tHOSTNAME\tSERVICE\tZONE ID")
		for _, d := range domains {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.ID, d.Hostname, d.Service, d.ZoneID)
		}
		w.Flush()

		printInfo("Total: %d domain(s)", len(domains))
	})
}

func runWorkerDomainAttach(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return outErrf("hostname is required")
	}
	hostname := args[0]
	if strings.TrimSpace(workerDomainService) == "" {
		return outErrf("service name is required (--service)")
	}
	if strings.TrimSpace(workerDomainZone) == "" {
		return outErrf("zone ID is required (--zone)")
	}
	service, zoneID := workerDomainService, workerDomainZone

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would attach worker domain", func() any {
			return map[string]string{"hostname": hostname, "service": service, "zone_id": zoneID}
		}, func() {
			printInfo("DRY RUN: Would attach '%s' to worker '%s' in zone '%s'", hostname, service, zoneID)
		})
	}

	dom, err := svc.DomainAttach(context.Background(), hostname, service, zoneID)
	if err != nil {
		return outErr("failed to attach worker domain", err)
	}

	return outPayload("Worker domain attached successfully", func() any {
		return dom
	}, func() {
		printSuccess("Domain '%s' attached to worker '%s'", dom.Hostname, dom.Service)
	})
}

func runWorkerDomainDetach(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return outErrf("domain ID is required")
	}
	domainID := args[0]
	if strings.TrimSpace(domainID) == "" {
		return outErrf("domain ID is required")
	}

	if !workerDomainForce && !DryRun {
		fmt.Printf("Are you sure you want to detach domain '%s'? [y/N]: ", domainID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Domain detach cancelled")
			return nil
		}
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would detach worker domain", func() any {
			return map[string]string{"domain_id": domainID}
		}, func() {
			printInfo("DRY RUN: Would detach domain '%s'", domainID)
		})
	}

	if err := svc.DomainDetach(context.Background(), domainID); err != nil {
		return outErr("failed to detach worker domain", err)
	}

	return outPayload("Worker domain detached successfully", func() any {
		return map[string]string{"domain_id": domainID}
	}, func() {
		printSuccess("Domain '%s' detached", domainID)
	})
}
