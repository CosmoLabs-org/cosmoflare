package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"
)

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Manage Cloudflare DNS records",
	Long: `DNS record management for Cloudflare zones.

Commands:
  create    Create a DNS record
  list      List DNS records
  get       Get a DNS record
  update    Update a DNS record
  delete    Delete a DNS record

DNS records are zone-scoped, so a zone ID is required for all operations.

Examples:
  cosmoflare dns create ZONE_ID --type=A --name=www --content=1.2.3.4
  cosmoflare dns list ZONE_ID --json
  cosmoflare dns get ZONE_ID RECORD_ID
  cosmoflare dns update ZONE_ID RECORD_ID --content=5.6.7.8
  cosmoflare dns delete ZONE_ID RECORD_ID --force`,
}

var (
	dnsRecordType string
	dnsName       string
	dnsContent    string
	dnsTTL        int
	dnsProxied    bool
	dnsPriority   uint16
	dnsComment    string
	dnsForce      bool
	dnsFilterType string
	dnsFilterName string
	dnsFilterContent string
)

var dnsCreateCmd = &cobra.Command{
	Use:   "create [zone-id]",
	Short: "Create a DNS record",
	Long: `Create a new DNS record in a Cloudflare zone.

Supported record types: A, AAAA, CNAME, MX, TXT, NS, SRV, CAA, LOC, SPF, CERT, DNSKEY, DS, NAPTR, SMIMEA, SSHFP, TLSA, URI.

Examples:
  cosmoflare dns create ZONE_ID --type=A --name=www --content=1.2.3.4
  cosmoflare dns create ZONE_ID --type=AAAA --name=www --content=2001:db8::1
  cosmoflare dns create ZONE_ID --type=CNAME --name=blog --content=example.com
  cosmoflare dns create ZONE_ID --type=MX --name=@ --content=mail.example.com --priority=10
  cosmoflare dns create ZONE_ID --type=TXT --name=@ --content="v=spf1 include:example.com ~all"
  cosmoflare dns create ZONE_ID --type=A --name=api --content=1.2.3.4 --proxied --ttl=1 --comment="API endpoint"`,
	RunE: runDNSCreate,
}

var dnsListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List DNS records",
	Long: `List all DNS records in a zone with optional filtering.

Examples:
  cosmoflare dns list ZONE_ID
  cosmoflare dns list ZONE_ID --type=A
  cosmoflare dns list ZONE_ID --name=www.example.com
  cosmoflare dns list ZONE_ID --content=1.2.3.4
  cosmoflare dns list ZONE_ID --json`,
	RunE: runDNSList,
}

var dnsGetCmd = &cobra.Command{
	Use:   "get [zone-id] [record-id]",
	Short: "Get a DNS record",
	Long: `Get details of a single DNS record by its ID.

Examples:
  cosmoflare dns get ZONE_ID RECORD_ID
  cosmoflare dns get ZONE_ID RECORD_ID --json`,
	RunE: runDNSGet,
}

var dnsUpdateCmd = &cobra.Command{
	Use:   "update [zone-id] [record-id]",
	Short: "Update a DNS record",
	Long: `Update an existing DNS record.

Provide only the fields you want to change. Unspecified fields are left unchanged.

Examples:
  cosmoflare dns update ZONE_ID RECORD_ID --content=5.6.7.8
  cosmoflare dns update ZONE_ID RECORD_ID --ttl=300
  cosmoflare dns update ZONE_ID RECORD_ID --proxied --comment="Updated endpoint"
  cosmoflare dns update ZONE_ID RECORD_ID --content=5.6.7.8 --ttl=300 --json`,
	RunE: runDNSUpdate,
}

var dnsDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [record-id]",
	Short: "Delete a DNS record",
	Long: `Delete a DNS record from a zone.

WARNING: This action is irreversible.

Examples:
  cosmoflare dns delete ZONE_ID RECORD_ID
  cosmoflare dns delete ZONE_ID RECORD_ID --force`,
	RunE: runDNSDelete,
}

func init() {
	rootCmd.AddCommand(dnsCmd)

	dnsCmd.AddCommand(dnsCreateCmd)
	dnsCmd.AddCommand(dnsListCmd)
	dnsCmd.AddCommand(dnsGetCmd)
	dnsCmd.AddCommand(dnsUpdateCmd)
	dnsCmd.AddCommand(dnsDeleteCmd)

	// Create flags
	dnsCreateCmd.Flags().StringVar(&dnsRecordType, "type", "", "Record type (A, AAAA, CNAME, MX, TXT, etc.)")
	dnsCreateCmd.Flags().StringVar(&dnsName, "name", "", "Record name (e.g., www, @, sub.domain)")
	dnsCreateCmd.Flags().StringVar(&dnsContent, "content", "", "Record content (IP address, hostname, text, etc.)")
	dnsCreateCmd.Flags().IntVar(&dnsTTL, "ttl", 0, "TTL in seconds (1 = automatic)")
	dnsCreateCmd.Flags().BoolVar(&dnsProxied, "proxied", false, "Enable Cloudflare proxy (orange cloud)")
	dnsCreateCmd.Flags().Uint16Var(&dnsPriority, "priority", 0, "Record priority (for MX, SRV records)")
	dnsCreateCmd.Flags().StringVar(&dnsComment, "comment", "", "Comment for the DNS record")
	_ = dnsCreateCmd.MarkFlagRequired("type")
	_ = dnsCreateCmd.MarkFlagRequired("name")
	_ = dnsCreateCmd.MarkFlagRequired("content")

	// List filter flags
	dnsListCmd.Flags().StringVar(&dnsFilterType, "type", "", "Filter by record type (A, CNAME, MX, etc.)")
	dnsListCmd.Flags().StringVar(&dnsFilterName, "name", "", "Filter by record name")
	dnsListCmd.Flags().StringVar(&dnsFilterContent, "content", "", "Filter by record content")

	// Update flags
	dnsUpdateCmd.Flags().IntVar(&dnsTTL, "ttl", 0, "TTL in seconds (1 = automatic)")
	dnsUpdateCmd.Flags().BoolVar(&dnsProxied, "proxied", false, "Enable Cloudflare proxy (orange cloud)")
	dnsUpdateCmd.Flags().Uint16Var(&dnsPriority, "priority", 0, "Record priority (for MX, SRV records)")
	dnsUpdateCmd.Flags().StringVar(&dnsComment, "comment", "", "Comment for the DNS record")

	// Delete flags
	dnsDeleteCmd.Flags().BoolVar(&dnsForce, "force", false, "Skip confirmation prompt")
}

func getDNSService(zoneID string) (*cosmoflare.DNSService, error) {
	return cosmoflare.NewDNSServiceFromCreds(zoneID, APIToken)
}

func runDNSCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getDNSService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create DNS service: %w", err)
	}

	var opts []cosmoflare.DNSOption
	if dnsTTL > 0 {
		opts = append(opts, cosmoflare.WithDNSTTL(dnsTTL))
	}
	if cmd.Flags().Changed("proxied") {
		opts = append(opts, cosmoflare.WithDNSProxied(dnsProxied))
	}
	if dnsPriority > 0 {
		opts = append(opts, cosmoflare.WithDNSPriority(dnsPriority))
	}
	if dnsComment != "" {
		opts = append(opts, cosmoflare.WithDNSComment(dnsComment))
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create DNS record", map[string]interface{}{
				"zone_id": zoneID,
				"type":    dnsRecordType,
				"name":    dnsName,
				"content": dnsContent,
			})
		}
		printInfo("DRY RUN: Would create %s record '%s' -> %s in zone %s", dnsRecordType, dnsName, dnsContent, zoneID)
		return nil
	}

	record, err := svc.Create(context.Background(), dnsRecordType, dnsName, dnsContent, opts...)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create DNS record: %v", err))
		}
		return fmt.Errorf("failed to create DNS record: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("DNS record created successfully", record)
	}

	printSuccess("DNS record created successfully!")
	printInfo("ID: %s", record.ID)
	printInfo("Type: %s", record.Type)
	printInfo("Name: %s", record.Name)
	printInfo("Content: %s", record.Content)
	printInfo("TTL: %d", record.TTL)
	printInfo("Proxied: %v", record.Proxied)
	return nil
}

func runDNSList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getDNSService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create DNS service: %w", err)
	}

	var opts []cosmoflare.DNSListOption
	if dnsFilterType != "" {
		opts = append(opts, cosmoflare.WithDNSType(dnsFilterType))
	}
	if dnsFilterName != "" {
		opts = append(opts, cosmoflare.WithDNSName(dnsFilterName))
	}
	if dnsFilterContent != "" {
		opts = append(opts, cosmoflare.WithDNSContent(dnsFilterContent))
	}

	records, err := svc.List(context.Background(), opts...)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list DNS records: %v", err))
		}
		return fmt.Errorf("failed to list DNS records: %w", err)
	}

	if JSONOutput {
		return printJSON(records)
	}

	if len(records) == 0 {
		printInfo("No DNS records found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tNAME\tCONTENT\tTTL\tPROXIED")
	for _, r := range records {
		ttlStr := fmt.Sprintf("%d", r.TTL)
		if r.TTL == 1 {
			ttlStr = "auto"
		}
		proxiedStr := "no"
		if r.Proxied {
			proxiedStr = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.ID,
			r.Type,
			r.Name,
			r.Content,
			ttlStr,
			proxiedStr,
		)
	}
	w.Flush()

	printInfo("Total: %d record(s)", len(records))
	return nil
}

func runDNSGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and record ID are required")
	}
	zoneID, recordID := args[0], args[1]

	svc, err := getDNSService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create DNS service: %w", err)
	}

	record, err := svc.Get(context.Background(), recordID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get DNS record: %v", err))
		}
		return fmt.Errorf("failed to get DNS record: %w", err)
	}

	if JSONOutput {
		return printJSON(record)
	}

	fmt.Printf("ID:         %s\n", record.ID)
	fmt.Printf("Type:       %s\n", record.Type)
	fmt.Printf("Name:       %s\n", record.Name)
	fmt.Printf("Content:    %s\n", record.Content)
	ttlStr := fmt.Sprintf("%d", record.TTL)
	if record.TTL == 1 {
		ttlStr = "auto (1)"
	}
	fmt.Printf("TTL:        %s\n", ttlStr)
	fmt.Printf("Proxied:    %v\n", record.Proxied)
	fmt.Printf("Proxiable:  %v\n", record.Proxiable)
	if record.Priority != nil {
		fmt.Printf("Priority:   %d\n", *record.Priority)
	}
	if record.Comment != "" {
		fmt.Printf("Comment:    %s\n", record.Comment)
	}
	fmt.Printf("Zone ID:    %s\n", record.ZoneID)
	fmt.Printf("Created:    %s\n", record.CreatedOn.Format("2006-01-02 15:04:05"))
	fmt.Printf("Modified:   %s\n", record.ModifiedOn.Format("2006-01-02 15:04:05"))
	return nil
}

func runDNSUpdate(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and record ID are required")
	}
	zoneID, recordID := args[0], args[1]

	var opts []cosmoflare.DNSOption
	if cmd.Flags().Changed("ttl") {
		opts = append(opts, cosmoflare.WithDNSTTL(dnsTTL))
	}
	if cmd.Flags().Changed("proxied") {
		opts = append(opts, cosmoflare.WithDNSProxied(dnsProxied))
	}
	if cmd.Flags().Changed("priority") {
		opts = append(opts, cosmoflare.WithDNSPriority(dnsPriority))
	}
	if cmd.Flags().Changed("comment") {
		opts = append(opts, cosmoflare.WithDNSComment(dnsComment))
	}

	if len(opts) == 0 {
		return fmt.Errorf("at least one update flag is required (--ttl, --proxied, --priority, --comment)")
	}

	svc, err := getDNSService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create DNS service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would update DNS record", map[string]string{
				"zone_id":   zoneID,
				"record_id": recordID,
			})
		}
		printInfo("DRY RUN: Would update DNS record '%s' in zone '%s'", recordID, zoneID)
		return nil
	}

	record, err := svc.Update(context.Background(), recordID, opts...)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to update DNS record: %v", err))
		}
		return fmt.Errorf("failed to update DNS record: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("DNS record updated successfully", record)
	}

	printSuccess("DNS record '%s' updated successfully!", recordID)
	printInfo("Type: %s  Name: %s  Content: %s", record.Type, record.Name, record.Content)
	return nil
}

func runDNSDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and record ID are required")
	}
	zoneID, recordID := args[0], args[1]

	if !dnsForce && !DryRun {
		fmt.Printf("Are you sure you want to delete DNS record '%s'? [y/N]: ", recordID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("DNS record deletion cancelled")
			return nil
		}
	}

	svc, err := getDNSService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create DNS service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete DNS record", map[string]string{
				"zone_id":   zoneID,
				"record_id": recordID,
			})
		}
		printInfo("DRY RUN: Would delete DNS record '%s' from zone '%s'", recordID, zoneID)
		return nil
	}

	if err := svc.Delete(context.Background(), recordID); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete DNS record: %v", err))
		}
		return fmt.Errorf("failed to delete DNS record: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("DNS record deleted successfully", map[string]string{
			"zone_id":   zoneID,
			"record_id": recordID,
		})
	}
	printSuccess("DNS record '%s' deleted successfully!", recordID)
	return nil
}
