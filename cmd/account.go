package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage multiple Cloudflare accounts",
	Long: `Manage multiple Cloudflare accounts beyond environment variables.

Accounts are stored in ~/.cosmoflare/accounts.yaml (permissions 0600).
The active account is tracked in ~/.cosmoflare/active-account.

Commands:
  list      List all configured accounts
  add       Add a new account
  switch    Switch the active account
  remove    Remove an account
  current   Show the current active account
  verify    Verify account credentials work

Examples:
  cosmoflare account list                                         # List all accounts
  cosmoflare account add prod --account-id ABC --api-token XYZ    # Add account
  cosmoflare account switch prod                                  # Switch active account
  cosmoflare account current                                      # Show active account
  cosmoflare account verify prod                                  # Verify credentials
  cosmoflare account remove staging --force                       # Remove account`,
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured accounts",
	Long: `List all Cloudflare accounts configured in ~/.cosmoflare/accounts.yaml.

Displays account name, masked account ID, email, and whether the account
is currently active. API tokens are never shown.

Examples:
  cosmoflare account list         # Human-readable table
  cosmoflare account list --json  # Machine-readable JSON output`,
	RunE: runAccountList,
}

var accountAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new account",
	Long: `Add a new Cloudflare account configuration.

The account name must be unique and contain only alphanumeric characters,
hyphens, and underscores. The --account-id and --api-token flags are required.

If this is the first account added, it automatically becomes the active account.

Examples:
  cosmoflare account add production --account-id abc123 --api-token tok_xxx
  cosmoflare account add staging --account-id def456 --api-token tok_yyy --email user@example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runAccountAdd,
}

var accountSwitchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch the active account",
	Long: `Switch the active Cloudflare account.

The named account must already be configured. After switching, all subsequent
cosmoflare commands will use the new account's credentials (unless overridden
by --account-id / --api-token flags or environment variables).

Examples:
  cosmoflare account switch production
  cosmoflare account switch staging`,
	Args: cobra.ExactArgs(1),
	RunE: runAccountSwitch,
}

var accountRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an account",
	Long: `Remove a configured Cloudflare account.

If the account is currently active, use --force to remove it anyway.
This permanently deletes the account configuration from accounts.yaml.

Examples:
  cosmoflare account remove old-staging
  cosmoflare account remove production --force    # Remove even if active`,
	Args: cobra.ExactArgs(1),
	RunE: runAccountRemove,
}

var accountCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current active account",
	Long: `Show the currently active Cloudflare account.

Displays the account name, masked account ID, and email. If no account
is active, suggests how to set one.

Examples:
  cosmoflare account current
  cosmoflare account current --json`,
	RunE: runAccountCurrent,
}

var accountVerifyCmd = &cobra.Command{
	Use:   "verify <name>",
	Short: "Verify account credentials work",
	Long: `Verify that an account's API token is valid by calling the Cloudflare API.

Makes a lightweight request to the /user/tokens/verify endpoint. The result
indicates whether the token is accepted.

Examples:
  cosmoflare account verify production
  cosmoflare account verify staging --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAccountVerify,
}

// Flags for account add
var (
	accountAddAccountID string
	accountAddAPIToken  string
	accountAddEmail     string
)

// Flag for account remove
var accountRemoveForce bool

func init() {
	rootCmd.AddCommand(accountCmd)

	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountSwitchCmd)
	accountCmd.AddCommand(accountRemoveCmd)
	accountCmd.AddCommand(accountCurrentCmd)
	accountCmd.AddCommand(accountVerifyCmd)

	// account add flags
	accountAddCmd.Flags().StringVar(&accountAddAccountID, "account-id", "", "Cloudflare Account ID (required)")
	accountAddCmd.Flags().StringVar(&accountAddAPIToken, "api-token", "", "Cloudflare API token (required)")
	accountAddCmd.Flags().StringVar(&accountAddEmail, "email", "", "Account email address (optional)")
	_ = accountAddCmd.MarkFlagRequired("account-id")
	_ = accountAddCmd.MarkFlagRequired("api-token")

	// account remove flags
	accountRemoveCmd.Flags().BoolVar(&accountRemoveForce, "force", false, "Force removal even if account is active")
}

// getAccountService creates an AccountService with the default config directory.
func getAccountService() (*cosmoflare.AccountService, error) {
	return cosmoflare.NewAccountService("")
}

func runAccountList(cmd *cobra.Command, args []string) error {
	p := NewPresenter()

	svc, err := getAccountService()
	if err != nil {
		return fmt.Errorf("failed to initialize account service: %w", err)
	}

	accounts, err := svc.List()
	if err != nil {
		return p.ErrorWrap("failed to list accounts", err)
	}

	// Determine active account
	cur, _ := svc.Current()
	activeName := ""
	if cur != nil {
		activeName = cur.Name
	}

	return p.SuccessPayload("accounts listed", func() any {
		type jsonAccount struct {
			Name      string `json:"name"`
			AccountID string `json:"account_id"`
			Email     string `json:"email,omitempty"`
			Active    bool   `json:"active"`
			CreatedAt string `json:"created_at"`
		}
		out := make([]jsonAccount, 0, len(accounts))
		for _, a := range accounts {
			out = append(out, jsonAccount{
				Name:      a.Name,
				AccountID: a.AccountID,
				Email:     a.Email,
				Active:    a.Name == activeName,
				CreatedAt: a.CreatedAt,
			})
		}
		return map[string]interface{}{
			"accounts": out,
			"count":    len(out),
			"active":   activeName,
		}
	}, func() {
		if len(accounts) == 0 {
			printInfo("No accounts configured")
			printInfo("Add one: cosmoflare account add <name> --account-id ID --api-token TOKEN")
			return
		}

		fmt.Printf("\nConfigured Accounts (%d)\n", len(accounts))
		fmt.Println(strings.Repeat("─", 60))

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  \tNAME\tACCOUNT ID\tEMAIL")
		fmt.Fprintln(w, "  \t────\t──────────\t─────")
		for _, a := range accounts {
			marker := "  "
			if a.Name == activeName {
				marker = "* "
			}
			maskedID := maskID(a.AccountID)
			email := a.Email
			if email == "" {
				email = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", marker, a.Name, maskedID, email)
		}
		w.Flush()

		if activeName != "" {
			fmt.Printf("\n* = active account (%s)\n", activeName)
		}
	})
}

func runAccountAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	p := NewPresenter()

	svc, err := getAccountService()
	if err != nil {
		return fmt.Errorf("failed to initialize account service: %w", err)
	}

	if DryRun {
		printWarning("DRY RUN: Would add account %q", name)
		return nil
	}

	acct, err := svc.Add(name, accountAddAccountID, accountAddAPIToken, accountAddEmail)
	if err != nil {
		return p.ErrorWrap("failed to add account", err)
	}

	return p.SuccessPayload("account added", func() any {
		return map[string]interface{}{
			"name":       acct.Name,
			"account_id": acct.AccountID,
			"email":      acct.Email,
			"created_at": acct.CreatedAt,
		}
	}, func() {
		printSuccess("Account %q added (account ID: %s)", acct.Name, maskID(acct.AccountID))
	})
}

func runAccountSwitch(cmd *cobra.Command, args []string) error {
	name := args[0]
	p := NewPresenter()

	svc, err := getAccountService()
	if err != nil {
		return fmt.Errorf("failed to initialize account service: %w", err)
	}

	acct, err := svc.Switch(name)
	if err != nil {
		return p.ErrorWrap("failed to switch account", err)
	}

	return p.SuccessPayload("account switched", func() any {
		return map[string]interface{}{
			"name":       acct.Name,
			"account_id": acct.AccountID,
		}
	}, func() {
		printSuccess("Switched to account %q (account ID: %s)", acct.Name, maskID(acct.AccountID))
	})
}

func runAccountRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	p := NewPresenter()

	svc, err := getAccountService()
	if err != nil {
		return fmt.Errorf("failed to initialize account service: %w", err)
	}

	if DryRun {
		printWarning("DRY RUN: Would remove account %q", name)
		return nil
	}

	if err := svc.Remove(name, accountRemoveForce); err != nil {
		return p.ErrorWrap("failed to remove account", err)
	}

	return p.SuccessPayload("account removed", func() any {
		return map[string]interface{}{"name": name}
	}, func() {
		printSuccess("Account %q removed", name)
	})
}

func runAccountCurrent(cmd *cobra.Command, args []string) error {
	p := NewPresenter()

	svc, err := getAccountService()
	if err != nil {
		return fmt.Errorf("failed to initialize account service: %w", err)
	}

	acct, err := svc.Current()
	if err != nil {
		return p.ErrorWrap("failed to get current account", err)
	}

	if acct == nil {
		return p.SuccessPayload("no active account", func() any {
			return map[string]interface{}{"active": false}
		}, func() {
			printInfo("No active account")
			printInfo("Set one: cosmoflare account switch <name>")
		})
	}

	return p.SuccessPayload("current account", func() any {
		return map[string]interface{}{
			"name":       acct.Name,
			"account_id": acct.AccountID,
			"email":      acct.Email,
			"active":     true,
		}
	}, func() {
		fmt.Printf("\nActive Account\n")
		fmt.Println(strings.Repeat("─", 40))
		fmt.Printf("  Name:       %s\n", acct.Name)
		fmt.Printf("  Account ID: %s\n", maskID(acct.AccountID))
		if acct.Email != "" {
			fmt.Printf("  Email:      %s\n", acct.Email)
		}
	})
}

func runAccountVerify(cmd *cobra.Command, args []string) error {
	name := args[0]
	p := NewPresenter()

	svc, err := getAccountService()
	if err != nil {
		return fmt.Errorf("failed to initialize account service: %w", err)
	}

	result, err := svc.Verify(name)
	if err != nil {
		return p.ErrorWrap("failed to verify account", err)
	}

	return p.SuccessPayload("account verified", func() any {
		return map[string]interface{}{
			"name":    result.Name,
			"valid":   result.Valid,
			"message": result.Message,
		}
	}, func() {
		if result.Valid {
			printSuccess("Account %q: %s", result.Name, result.Message)
		} else {
			printError("Account %q: %s", result.Name, result.Message)
		}
	})
}

// maskID shows the first 4 and last 4 characters of an ID, masking the middle.
func maskID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:4] + strings.Repeat("*", len(id)-8) + id[len(id)-4:]
}
