package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// newRegistrarService is a package-level factory var so tests can override
// the service constructor without live Cloudflare credentials (same seam
// pattern as newRedirectService).
var newRegistrarService = func() (*cosmoflare.RegistrarService, error) {
	return cosmoflare.NewRegistrarServiceFromCreds(AccountID, APIToken)
}

var (
	regForce            bool
	regAutoRenewState   string
	regContactRole      string
	regContactAllRoles  bool
	regContactFirstName string
	regContactLastName  string
	regContactOrg       string
	regContactAddress   string
	regContactAddress2  string
	regContactCity      string
	regContactState     string
	regContactZip       string
	regContactCountry   string
	regContactPhone     string
	regContactEmail     string
	regContactFax       string
)

var domainsRegistrarCmd = &cobra.Command{
	Use:   "registrar",
	Short: "Manage Cloudflare Registrar domains: register, transfer, renew, lock, contacts, DNSSEC",
	Long: `Registrar operations for domains registered with Cloudflare Registrar.

Registration data, transfers, renewals, transfer locks, registration
contacts and DNSSEC delegation — all in one place.

Sequencing rules encoded as pre-flight checks:
- transfer requires the domain to be UNLOCKED at the current registrar
- DNSSEC disable requires DS records to be removed at the parent zone first
- renew only works inside the domain's renewable window (API errors are
  surfaced verbatim)

register, transfer and renew are cost-incurring mutations. Preview them
with --dry-run before running for real.

Examples:
  cosmoflare domains registrar list
  cosmoflare domains registrar get example.com
  cosmoflare domains registrar register example.com --dry-run
  cosmoflare domains registrar transfer example.com
  cosmoflare domains registrar cancel-transfer example.com
  cosmoflare domains registrar renew example.com
  cosmoflare domains registrar auto-renew example.com on
  cosmoflare domains registrar lock example.com
  cosmoflare domains registrar unlock example.com
  cosmoflare domains registrar contacts get example.com
  cosmoflare domains registrar contacts update example.com --first-name=Jane --last-name=Doe --email=jane@example.com
  cosmoflare domains registrar dnssec get example.com
  cosmoflare domains registrar dnssec enable example.com
  cosmoflare domains registrar dnssec disable example.com`,
}

var domainsRegistrarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Cloudflare-registered domains with registration status",
	Long: `List every domain registered with Cloudflare Registrar, with expiry,
transfer lock and registrar name.

Domains registered elsewhere are absent from this list (see "cosmoflare
domains" for the full zone overview).

Examples:
  cosmoflare domains registrar list
  cosmoflare domains registrar list --json`,
	Args: cobra.NoArgs,
	RunE: runRegistrarList,
}

var domainsRegistrarGetCmd = &cobra.Command{
	Use:   "get DOMAIN",
	Short: "Show full registrar details for a domain",
	Long: `Show the full registrar state for a single domain: transfer-in
checklist, registry statuses, expiry, lock state and registrant contact.

Works for domains not yet at Cloudflare too — the API reports availability
and transfer eligibility.

Examples:
  cosmoflare domains registrar get example.com
  cosmoflare domains registrar get example.com --json`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarGet,
}

var domainsRegistrarRegisterCmd = &cobra.Command{
	Use:   "register DOMAIN",
	Short: "Register a domain with Cloudflare Registrar",
	Long: `Register a new domain with Cloudflare Registrar at cost.

COST-INCURRING MUTATION: registration is billed to your Cloudflare account.
Use --dry-run to preview the request before committing.

The domain may enter a pending state — poll its status with
"cosmoflare domains registrar get DOMAIN".

Examples:
  cosmoflare domains registrar register example.com --dry-run
  cosmoflare domains registrar register example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarRegister,
}

var domainsRegistrarTransferCmd = &cobra.Command{
	Use:   "transfer DOMAIN",
	Short: "Transfer a domain into Cloudflare Registrar",
	Long: `Initiate a transfer of a domain from its current registrar into
Cloudflare Registrar (at-cost renewal included).

COST-INCURRING MUTATION: transfers add a year of registration and are
billed to your Cloudflare account. Use --dry-run to preview.

Pre-flight: the domain MUST be unlocked (and WHOIS privacy disabled) at the
current registrar. Warnings block the transfer unless --force is passed.
Transfers are asynchronous — poll "cosmoflare domains registrar get DOMAIN"
or cancel with "cancel-transfer".

Examples:
  cosmoflare domains registrar transfer example.com --dry-run
  cosmoflare domains registrar transfer example.com
  cosmoflare domains registrar transfer example.com --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarTransfer,
}

var domainsRegistrarCancelTransferCmd = &cobra.Command{
	Use:   "cancel-transfer DOMAIN",
	Short: "Cancel a pending domain transfer",
	Long: `Cancel a pending transfer-in for a domain.

Only works while the transfer is still pending (check
transfer_in.can_cancel_transfer via "cosmoflare domains registrar get").

Examples:
  cosmoflare domains registrar cancel-transfer example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarCancelTransfer,
}

var domainsRegistrarRenewCmd = &cobra.Command{
	Use:   "renew DOMAIN",
	Short: "Renew a domain for one year",
	Long: `Renew a Cloudflare-registered domain for one year, at cost.

COST-INCURRING MUTATION: renewal is billed to your Cloudflare account. Use
--dry-run to preview.

Renewals only succeed inside the domain's renewable window; outside it the
API error is surfaced verbatim (check "expires_at" via "get").

Examples:
  cosmoflare domains registrar renew example.com --dry-run
  cosmoflare domains registrar renew example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarRenew,
}

var domainsRegistrarAutoRenewCmd = &cobra.Command{
	Use:   "auto-renew DOMAIN STATE",
	Short: "Toggle auto-renew for a domain (on|off)",
	Long: `Enable or disable auto-renew for a Cloudflare-registered domain.

STATE is one of: on, off, true, false.

Examples:
  cosmoflare domains registrar auto-renew example.com on
  cosmoflare domains registrar auto-renew example.com off`,
	Args: cobra.ExactArgs(2),
	RunE: runRegistrarAutoRenew,
}

var domainsRegistrarLockCmd = &cobra.Command{
	Use:   "lock DOMAIN",
	Short: "Lock a domain against transfers",
	Long: `Enable the transfer lock for a domain. Locked domains cannot be
transferred away from Cloudflare Registrar.

Examples:
  cosmoflare domains registrar lock example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarLock,
}

var domainsRegistrarUnlockCmd = &cobra.Command{
	Use:   "unlock DOMAIN",
	Short: "Unlock a domain for transfer",
	Long: `Disable the transfer lock for a domain, allowing it to be
transferred away from Cloudflare Registrar.

Only unlock when you intend to transfer — re-lock afterwards.

Examples:
  cosmoflare domains registrar unlock example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarUnlock,
}

var domainsRegistrarContactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Manage registration contacts for a domain",
	Long: `Get or update the registration contacts (registrant, billing,
admin, tech) for a Cloudflare-registered domain.

Examples:
  cosmoflare domains registrar contacts get example.com
  cosmoflare domains registrar contacts update example.com --first-name=Jane --last-name=Doe --email=jane@example.com`,
}

var domainsRegistrarContactsGetCmd = &cobra.Command{
	Use:   "get DOMAIN",
	Short: "Show registration contacts for a domain",
	Long: `Show the registration contacts for a Cloudflare-registered
domain. The read model exposes the registrant contact; billing/admin/tech
are write-only (see "contacts update").

Examples:
  cosmoflare domains registrar contacts get example.com
  cosmoflare domains registrar contacts get example.com --json`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarContactsGet,
}

var domainsRegistrarContactsUpdateCmd = &cobra.Command{
	Use:   "update DOMAIN",
	Short: "Update registration contacts for a domain",
	Long: `Update the registration contacts for a Cloudflare-registered
domain.

By default the flags update the REGISTRANT contact; use --role to target
billing, admin or tech instead, or --all-roles to apply the same contact to
all four roles at once.

Examples:
  cosmoflare domains registrar contacts update example.com --first-name=Jane --last-name=Doe --email=jane@example.com
  cosmoflare domains registrar contacts update example.com --role=tech --email=tech@example.com
  cosmoflare domains registrar contacts update example.com --all-roles --email=ops@example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarContactsUpdate,
}

var domainsRegistrarDNSSECCmd = &cobra.Command{
	Use:   "dnssec",
	Short: "Manage DNSSEC for a registrar domain",
	Long: `Get, enable or disable DNSSEC delegation for a Cloudflare-registered
domain.

Enabling DNSSEC returns DS records that MUST be added at the parent zone's
registrar for delegation to complete. Disabling requires those DS records
to be removed at the parent first — the pre-flight check warns and blocks
unless --force is passed.

Examples:
  cosmoflare domains registrar dnssec get example.com
  cosmoflare domains registrar dnssec enable example.com
  cosmoflare domains registrar dnssec disable example.com`,
}

var domainsRegistrarDNSSECGetCmd = &cobra.Command{
	Use:   "get DOMAIN",
	Short: "Show DNSSEC status and DS records for a domain",
	Long: `Show the DNSSEC status for a domain and the DS records that must
be published at the parent zone while delegation is active.

Examples:
  cosmoflare domains registrar dnssec get example.com
  cosmoflare domains registrar dnssec get example.com --json`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarDNSSECGet,
}

var domainsRegistrarDNSSECEnableCmd = &cobra.Command{
	Use:   "enable DOMAIN",
	Short: "Enable DNSSEC for a domain",
	Long: `Enable DNSSEC delegation for a domain.

The response includes the DS records to add at the parent zone's registrar
— until they are published, delegation is incomplete. This is a follow-up
note, not a blocker: the command succeeds and prints the records.

Examples:
  cosmoflare domains registrar dnssec enable example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarDNSSECEnable,
}

var domainsRegistrarDNSSECDisableCmd = &cobra.Command{
	Use:   "disable DOMAIN",
	Short: "Disable DNSSEC for a domain",
	Long: `Disable DNSSEC delegation for a domain.

Pre-flight: DS records must be REMOVED at the parent zone first, or the
domain stops resolving. The check blocks unless --force is passed.

Examples:
  cosmoflare domains registrar dnssec disable example.com
  cosmoflare domains registrar dnssec disable example.com --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRegistrarDNSSECDisable,
}

func init() {
	domainsCmd.AddCommand(domainsRegistrarCmd)

	domainsRegistrarCmd.AddCommand(domainsRegistrarListCmd)
	domainsRegistrarCmd.AddCommand(domainsRegistrarGetCmd)
	domainsRegistrarCmd.AddCommand(domainsRegistrarRegisterCmd)
	domainsRegistrarCmd.AddCommand(domainsRegistrarTransferCmd)
	domainsRegistrarCmd.AddCommand(domainsRegistrarCancelTransferCmd)
	domainsRegistrarCmd.AddCommand(domainsRegistrarRenewCmd)
	domainsRegistrarCmd.AddCommand(domainsRegistrarAutoRenewCmd)
	domainsRegistrarCmd.AddCommand(domainsRegistrarLockCmd)
	domainsRegistrarCmd.AddCommand(domainsRegistrarUnlockCmd)

	domainsRegistrarContactsCmd.AddCommand(domainsRegistrarContactsGetCmd)
	domainsRegistrarContactsCmd.AddCommand(domainsRegistrarContactsUpdateCmd)
	domainsRegistrarCmd.AddCommand(domainsRegistrarContactsCmd)

	domainsRegistrarDNSSECCmd.AddCommand(domainsRegistrarDNSSECGetCmd)
	domainsRegistrarDNSSECCmd.AddCommand(domainsRegistrarDNSSECEnableCmd)
	domainsRegistrarDNSSECCmd.AddCommand(domainsRegistrarDNSSECDisableCmd)
	domainsRegistrarCmd.AddCommand(domainsRegistrarDNSSECCmd)

	domainsRegistrarTransferCmd.Flags().BoolVar(&regForce, "force", false, "Proceed despite transfer pre-flight warnings")
	domainsRegistrarDNSSECDisableCmd.Flags().BoolVar(&regForce, "force", false, "Proceed despite DNSSEC disable pre-flight warnings")

	registrarContactFlag := func(cmd *cobra.Command) {
		cmd.Flags().StringVar(&regContactRole, "role", "registrant", "Contact role to update: registrant, billing, admin, tech")
		cmd.Flags().BoolVar(&regContactAllRoles, "all-roles", false, "Apply the contact to all four roles")
		cmd.Flags().StringVar(&regContactFirstName, "first-name", "", "Contact first name")
		cmd.Flags().StringVar(&regContactLastName, "last-name", "", "Contact last name")
		cmd.Flags().StringVar(&regContactOrg, "organization", "", "Contact organization")
		cmd.Flags().StringVar(&regContactAddress, "address", "", "Contact street address")
		cmd.Flags().StringVar(&regContactAddress2, "address2", "", "Contact street address (line 2)")
		cmd.Flags().StringVar(&regContactCity, "city", "", "Contact city")
		cmd.Flags().StringVar(&regContactState, "state", "", "Contact state/province")
		cmd.Flags().StringVar(&regContactZip, "zip", "", "Contact postal code")
		cmd.Flags().StringVar(&regContactCountry, "country", "", "Contact country code (e.g. US)")
		cmd.Flags().StringVar(&regContactPhone, "phone", "", "Contact phone (e.g. +1.5555555555)")
		cmd.Flags().StringVar(&regContactEmail, "email", "", "Contact email")
		cmd.Flags().StringVar(&regContactFax, "fax", "", "Contact fax")
	}
	registrarContactFlag(domainsRegistrarContactsUpdateCmd)
}

// registrarDomainEnvelope is the JSON envelope for single-domain registrar
// operations.
type registrarDomainEnvelope struct {
	Domain cosmoflare.RegistrarDomainDetail `json:"domain"`
}

// registrarService constructs the registrar service or renders the
// construction error through the active output mode.
func registrarService() (*cosmoflare.RegistrarService, error) {
	svc, err := newRegistrarService()
	if err != nil {
		return nil, outErr("failed to create registrar service", err)
	}
	return svc, nil
}

func registrarDomainArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// printRegistrarWarnings emits pre-flight warnings in plain mode; in JSON
// mode warnings ride on the error envelope (see registrarBlockOnWarnings).
func printRegistrarWarnings(warnings []string) {
	for _, w := range warnings {
		printWarning("pre-flight: %s", w)
	}
}

// registrarBlockOnWarnings returns a blocking error when warnings exist and
// --force was not passed; otherwise the warnings are printed and nil is
// returned.
func registrarBlockOnWarnings(action string, warnings []string, forced bool) error {
	if len(warnings) == 0 {
		return nil
	}
	if !forced {
		return outErrf("%s blocked by pre-flight warnings: %s — resolve them or pass --force", action, strings.Join(warnings, "; "))
	}
	printRegistrarWarnings(warnings)
	return nil
}

// renderRegistrarDetail prints the human view of a domain detail.
func renderRegistrarDetail(d cosmoflare.RegistrarDomainDetail) {
	fmt.Printf("Domain:            %s\n", d.Name)
	fmt.Printf("Registrar:         %s\n", d.CurrentRegistrar)
	fmt.Printf("Transfer lock:     %t\n", d.Locked)
	fmt.Printf("Available:         %t\n", d.Available)
	fmt.Printf("Can register:      %t\n", d.CanRegister)
	if d.ExpiresAt != nil {
		fmt.Printf("Expires at:        %s\n", d.ExpiresAt.Format("2006-01-02"))
	}
	if d.RegistryStatuses != "" {
		fmt.Printf("Registry statuses: %s\n", d.RegistryStatuses)
	}
	if d.TransferIn.CanCancelTransfer {
		printWarning("transfer-in pending — cancel with: cosmoflare domains registrar cancel-transfer %s", d.Name)
	}
	if d.Registrant != nil {
		fmt.Printf("Registrant:        %s %s <%s>\n", d.Registrant.FirstName, d.Registrant.LastName, d.Registrant.Email)
	}
}

func runRegistrarList(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	infos, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list registrar domains", err)
	}
	return outResult(struct {
		Domains map[string]cosmoflare.RegistrarInfo `json:"domains"`
	}{Domains: infos}, func() {
		if len(infos) == 0 {
			printInfo("No Cloudflare-registered domains found")
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "DOMAIN\tREGISTRAR\tLOCKED\tEXPIRES")
		for name, info := range infos {
			expires := "unknown"
			if info.ExpiresAt != nil {
				expires = info.ExpiresAt.Format("2006-01-02")
			}
			fmt.Fprintf(w, "%s\t%s\t%t\t%s\n", name, info.RegistrarName, info.TransferLock, expires)
		}
		w.Flush()
		printInfo("Total: %d domain(s)", len(infos))
	})
}

func runRegistrarGet(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	detail, err := svc.Get(context.Background(), registrarDomainArg(args))
	if err != nil {
		return outErr("failed to get registrar domain", err)
	}
	return outResult(registrarDomainEnvelope{Domain: detail}, func() {
		renderRegistrarDetail(detail)
	})
}

// registrarDryRun renders the standard dry-run payload for a cost-incurring
// registrar mutation.
func registrarDryRun(action, domain string, payload map[string]any) error {
	return outPayload("DRY RUN: Would "+action, func() any {
		return payload
	}, func() {
		printInfo("DRY RUN: Would %s %q (no API call made)", action, domain)
	})
}

func runRegistrarRegister(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	domain := registrarDomainArg(args)
	if DryRun {
		return registrarDryRun("register", domain, map[string]any{"domain": domain, "endpoint": "POST /registrar/domains"})
	}
	detail, err := svc.Register(context.Background(), domain)
	if err != nil {
		return outErr("failed to register domain", err)
	}
	return outResult(registrarDomainEnvelope{Domain: detail}, func() {
		printSuccess("Registration initiated for %s", domain)
		renderRegistrarDetail(detail)
		printInfo("Poll status with: cosmoflare domains registrar get %s", domain)
	})
}

func runRegistrarTransfer(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	domain := registrarDomainArg(args)
	ctx := context.Background()
	warnings, err := svc.TransferPreflight(ctx, domain)
	if err != nil {
		return outErr("failed to run transfer pre-flight", err)
	}
	if err := registrarBlockOnWarnings("transfer", warnings, regForce); err != nil {
		return err
	}
	if DryRun {
		return registrarDryRun("transfer", domain, map[string]any{"domain": domain, "warnings": warnings, "endpoint": "POST /registrar/domains/{domain}/transfer"})
	}
	detail, err := svc.Transfer(ctx, domain)
	if err != nil {
		return outErr("failed to transfer domain", err)
	}
	return outResult(registrarDomainEnvelope{Domain: detail}, func() {
		printSuccess("Transfer initiated for %s", domain)
		renderRegistrarDetail(detail)
		printInfo("Transfers are asynchronous — poll with: cosmoflare domains registrar get %s", domain)
	})
}

func runRegistrarCancelTransfer(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	domain := registrarDomainArg(args)
	detail, err := svc.CancelTransfer(context.Background(), domain)
	if err != nil {
		return outErr("failed to cancel transfer", err)
	}
	return outResult(registrarDomainEnvelope{Domain: detail}, func() {
		printSuccess("Transfer cancelled for %s", domain)
		renderRegistrarDetail(detail)
	})
}

func runRegistrarRenew(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	domain := registrarDomainArg(args)
	if DryRun {
		return registrarDryRun("renew", domain, map[string]any{"domain": domain, "endpoint": "POST /registrar/domains/{domain}/renew"})
	}
	detail, err := svc.Renew(context.Background(), domain)
	if err != nil {
		// Renewal window violations surface the API error verbatim.
		return outErr("failed to renew domain", err)
	}
	return outResult(registrarDomainEnvelope{Domain: detail}, func() {
		printSuccess("Renewed %s for one year", domain)
		renderRegistrarDetail(detail)
	})
}

// parseRegistrarBool accepts on/off/true/false/1/0/yes/no.
func parseRegistrarBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "on", "true", "1", "yes":
		return true, nil
	case "off", "false", "0", "no":
		return false, nil
	}
	return false, outErrf("invalid state %q — expected on, off, true or false", s)
}

func runRegistrarAutoRenew(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	enabled, err := parseRegistrarBool(args[1])
	if err != nil {
		return err
	}
	domain := args[0]
	detail, err := svc.SetAutoRenew(context.Background(), domain, enabled)
	if err != nil {
		return outErr("failed to set auto-renew", err)
	}
	state := "disabled"
	if enabled {
		state = "enabled"
	}
	return outResult(struct {
		Domain    cosmoflare.RegistrarDomainDetail `json:"domain"`
		AutoRenew bool                             `json:"auto_renew"`
	}{Domain: detail, AutoRenew: enabled}, func() {
		printSuccess("Auto-renew %s for %s", state, domain)
	})
}

func runRegistrarSetLock(locked bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		svc, err := registrarService()
		if err != nil {
			return err
		}
		domain := registrarDomainArg(args)
		detail, err := svc.SetLock(context.Background(), domain, locked)
		if err != nil {
			return outErr("failed to set transfer lock", err)
		}
		action := "Unlocked"
		if locked {
			action = "Locked"
		}
		return outResult(registrarDomainEnvelope{Domain: detail}, func() {
			printSuccess("%s %s", action, domain)
			if !locked {
				printWarning("domain can now be transferred away — re-lock when the transfer is done")
			}
		})
	}
}

func runRegistrarLock(cmd *cobra.Command, args []string) error {
	return runRegistrarSetLock(true)(cmd, args)
}

func runRegistrarUnlock(cmd *cobra.Command, args []string) error {
	return runRegistrarSetLock(false)(cmd, args)
}

func runRegistrarContactsGet(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	contacts, err := svc.GetContacts(context.Background(), registrarDomainArg(args))
	if err != nil {
		return outErr("failed to get contacts", err)
	}
	return outResult(contacts, func() {
		if contacts.Registrant == nil {
			printInfo("No registrant contact on record")
			return
		}
		c := contacts.Registrant
		fmt.Printf("Registrant: %s %s <%s>\n", c.FirstName, c.LastName, c.Email)
		if c.Organization != "" {
			fmt.Printf("  Org:      %s\n", c.Organization)
		}
		if c.Address != "" {
			fmt.Printf("  Address:  %s %s, %s %s %s\n", c.Address, c.Address2, c.City, c.State, c.Zip)
		}
		if c.Country != "" {
			fmt.Printf("  Country:  %s\n", c.Country)
		}
		if c.Phone != "" {
			fmt.Printf("  Phone:    %s\n", c.Phone)
		}
		printInfo("billing/admin/tech contacts are write-only — see 'contacts update --role'")
	})
}

func runRegistrarContactsUpdate(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	contact := &cosmoflare.RegistrarContact{
		FirstName:    regContactFirstName,
		LastName:     regContactLastName,
		Organization: regContactOrg,
		Address:      regContactAddress,
		Address2:     regContactAddress2,
		City:         regContactCity,
		State:        regContactState,
		Zip:          regContactZip,
		Country:      regContactCountry,
		Phone:        regContactPhone,
		Email:        regContactEmail,
		Fax:          regContactFax,
	}
	if contact.FirstName == "" && contact.LastName == "" && contact.Email == "" {
		return outErrf("at least one contact field is required (--first-name, --last-name, --email, ...)")
	}
	domain := registrarDomainArg(args)
	contacts := cosmoflare.RegistrarContacts{}
	switch {
	case regContactAllRoles:
		c := *contact
		contacts.Registrant, contacts.Billing, contacts.Admin, contacts.Tech = &c, &c, &c, &c
	default:
		switch strings.ToLower(regContactRole) {
		case "registrant":
			contacts.Registrant = contact
		case "billing":
			contacts.Billing = contact
		case "admin":
			contacts.Admin = contact
		case "tech":
			contacts.Tech = contact
		default:
			return outErrf("invalid --role %q — expected registrant, billing, admin or tech", regContactRole)
		}
	}
	updated, err := svc.UpdateContacts(context.Background(), domain, contacts)
	if err != nil {
		return outErr("failed to update contacts", err)
	}
	return outResult(updated, func() {
		printSuccess("Updated %s contact(s) for %s", regContactRole, domain)
		if b, err := json.Marshal(updated); err == nil {
			printInfo("updated contacts: %s", string(b))
		}
	})
}

func runRegistrarDNSSECGet(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	result, err := svc.GetDNSSEC(context.Background(), registrarDomainArg(args))
	if err != nil {
		return outErr("failed to get DNSSEC status", err)
	}
	return outResult(result, func() {
		fmt.Printf("DNSSEC status: %s\n", result.Status)
		if len(result.DSRecords) == 0 {
			printInfo("No DS records — delegation is not pending at the parent zone")
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "KEY TAG\tALGORITHM\tDIGEST TYPE\tDIGEST")
		for _, ds := range result.DSRecords {
			fmt.Fprintf(w, "%d\t%d\t%d\t%s\n", ds.KeyTag, ds.Algorithm, ds.DigestType, ds.Digest)
		}
		w.Flush()
		printInfo("These DS records must be published at the parent zone for delegation")
	})
}

func runRegistrarDNSSECEnable(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	domain := registrarDomainArg(args)
	result, err := svc.EnableDNSSEC(context.Background(), domain)
	if err != nil {
		return outErr("failed to enable DNSSEC", err)
	}
	return outResult(result, func() {
		printSuccess("DNSSEC %s for %s", strings.ToLower(result.Status), domain)
		if len(result.DSRecords) > 0 {
			printWarning("ACTION REQUIRED: add these DS records at the parent zone's registrar to complete delegation")
			for _, ds := range result.DSRecords {
				printInfo("  DS %d %d %d %s", ds.KeyTag, ds.Algorithm, ds.DigestType, ds.Digest)
			}
		}
	})
}

func runRegistrarDNSSECDisable(cmd *cobra.Command, args []string) error {
	svc, err := registrarService()
	if err != nil {
		return err
	}
	domain := registrarDomainArg(args)
	ctx := context.Background()
	warnings, err := svc.DisableDNSSECPreflight(ctx, domain)
	if err != nil {
		return outErr("failed to run DNSSEC disable pre-flight", err)
	}
	if err := registrarBlockOnWarnings("DNSSEC disable", warnings, regForce); err != nil {
		return err
	}
	result, err := svc.DisableDNSSEC(ctx, domain)
	if err != nil {
		return outErr("failed to disable DNSSEC", err)
	}
	return outResult(result, func() {
		printSuccess("DNSSEC %s for %s", strings.ToLower(result.Status), domain)
		if len(result.DSRecords) == 0 {
			printInfo("No DS records remain — delegation fully removed")
		}
	})
}
