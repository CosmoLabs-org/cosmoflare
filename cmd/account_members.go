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

var accountMemberCmd = &cobra.Command{
	Use:   "member",
	Short: "Manage Cloudflare account members",
	Long: `Manage the members of your Cloudflare account: list who has access,
invite new members by email, change their roles, and remove them.

Members are identified by member ID for update/remove (see
'cosmoflare account member list --json'), and by email when inviting.
Roles are assigned by role ID — list the available roles with
'cosmoflare account role list'.

Commands:
  list     List all account members
  invite   Invite a new member by email
  update   Change a member's roles
  remove   Remove a member from the account

Examples:
  cosmoflare account member list --json
  cosmoflare account member invite dev@example.com --role aaaa-account-settings-edit
  cosmoflare account member update MEMBER_ID --role bbbb-workers-scripts-edit
  cosmoflare account member remove MEMBER_ID --force`,
}

var accountMemberListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all account members",
	Long: `List every member of the account with their email, name, status
(invited/pending/active) and assigned roles.

Examples:
  cosmoflare account member list
  cosmoflare account member list --json`,
	RunE: runAccountMemberList,
}

var accountMemberInviteCmd = &cobra.Command{
	Use:   "invite <email>",
	Short: "Invite a new member to the account",
	Long: `Invite a new member to the account by email address.

The member receives a confirmation email and starts in "pending" status
until they accept. At least one role ID is required — resolve available
roles with 'cosmoflare account role list'.

Examples:
  cosmoflare account member invite dev@example.com --role aaaa-account-settings-edit
  cosmoflare account member invite ops@example.com --role ROLE_A --role ROLE_B --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAccountMemberInvite,
}

var accountMemberUpdateCmd = &cobra.Command{
	Use:   "update <member-id>",
	Short: "Change a member's roles",
	Long: `Replace the set of roles held by an account member.

--role is the FULL replacement set, not a delta: every role you list
replaces the member's current roles. Resolve role IDs with
'cosmoflare account role list'.

Examples:
  cosmoflare account member update MEMBER_ID --role bbbb-workers-scripts-edit
  cosmoflare account member update MEMBER_ID --role ROLE_A --role ROLE_B --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAccountMemberUpdate,
}

var accountMemberRemoveCmd = &cobra.Command{
	Use:   "remove <member-id>",
	Short: "Remove a member from the account",
	Long: `Remove a member from the account.

The member loses all access immediately; a pending invitation is revoked.
This is a destructive action: it runs in dry-run mode unless --force is
passed (or the command is registered non-destructive).

Examples:
  cosmoflare account member remove MEMBER_ID
  cosmoflare account member remove MEMBER_ID --force`,
	Args: cobra.ExactArgs(1),
	RunE: runAccountMemberRemove,
}

var accountRoleCmd = &cobra.Command{
	Use:   "role",
	Short: "List roles available in the account",
	Long: `List the roles available for assignment to account members.

Role IDs from this list are what 'account member invite' and
'account member update' expect in --role.

Commands:
  list   List available account roles

Examples:
  cosmoflare account role list
  cosmoflare account role list --json`,
}

var accountRoleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available account roles",
	Long: `List the roles available for assignment in this account, with the
role ID to pass to 'account member invite/update'.

Examples:
  cosmoflare account role list
  cosmoflare account role list --json`,
	RunE: runAccountRoleList,
}

// Flags for member invite/update.
var (
	accountMemberRoles []string
	accountMemberForce bool
)

func init() {
	accountCmd.AddCommand(accountMemberCmd)
	accountCmd.AddCommand(accountRoleCmd)

	accountMemberCmd.AddCommand(accountMemberListCmd)
	accountMemberCmd.AddCommand(accountMemberInviteCmd)
	accountMemberCmd.AddCommand(accountMemberUpdateCmd)
	accountMemberCmd.AddCommand(accountMemberRemoveCmd)

	accountRoleCmd.AddCommand(accountRoleListCmd)

	accountMemberInviteCmd.Flags().StringSliceVar(&accountMemberRoles, "role", []string{}, "Role ID to assign (repeatable, required)")
	accountMemberUpdateCmd.Flags().StringSliceVar(&accountMemberRoles, "role", []string{}, "Role ID to assign (repeatable; replaces the full role set)")
	accountMemberRemoveCmd.Flags().BoolVar(&accountMemberForce, "force", false, "Skip the destructive dry-run default and remove for real")

	_ = accountMemberInviteCmd.MarkFlagRequired("role")
	_ = accountMemberUpdateCmd.MarkFlagRequired("role")
}

// getAccountMemberService creates the member/role service from the
// credential globals.
func getAccountMemberService() (*cosmoflare.AccountMemberService, error) {
	return cosmoflare.NewAccountMemberServiceFromCreds(AccountID, APIToken)
}

// accountMemberRoleNames collapses a member's roles to a display string.
func accountMemberRoleNames(roles []cosmoflare.AccountMemberRoleView) string {
	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, r.Name)
	}
	if len(names) == 0 {
		return "-"
	}
	return strings.Join(names, ", ")
}

func runAccountMemberList(cmd *cobra.Command, args []string) error {
	svc, err := getAccountMemberService()
	if err != nil {
		return outErr("failed to create account member service", err)
	}

	members, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list account members", err)
	}

	return outResult(members, func() {
		if len(members) == 0 {
			printInfo("No members found for this account")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "MEMBER ID\tEMAIL\tNAME\tSTATUS\t2FA\tROLES")
		for _, m := range members {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\n",
				m.ID, m.Email, m.Name, m.Status, m.TwoFactor, accountMemberRoleNames(m.Roles))
		}
		w.Flush()
		printInfo("Total: %d member(s)", len(members))
	})
}

func runAccountMemberInvite(cmd *cobra.Command, args []string) error {
	email := args[0]

	svc, err := getAccountMemberService()
	if err != nil {
		return outErr("failed to create account member service", err)
	}

	member, err := svc.Invite(context.Background(), email, accountMemberRoles)
	if err != nil {
		return outErr("failed to invite member", err)
	}

	return outPayload("Member invited successfully", func() any {
		return member
	}, func() {
		printSuccess("Invitation sent to '%s' (member ID: %s)", member.Email, member.ID)
		printInfo("The member starts in 'pending' status until they accept the email invitation")
	})
}

func runAccountMemberUpdate(cmd *cobra.Command, args []string) error {
	memberID := args[0]

	svc, err := getAccountMemberService()
	if err != nil {
		return outErr("failed to create account member service", err)
	}

	member, err := svc.UpdateRoles(context.Background(), memberID, accountMemberRoles)
	if err != nil {
		return outErr("failed to update member roles", err)
	}

	return outPayload("Member roles updated successfully", func() any {
		return member
	}, func() {
		printSuccess("Roles of member '%s' updated: %s", memberID, accountMemberRoleNames(member.Roles))
	})
}

func runAccountMemberRemove(cmd *cobra.Command, args []string) error {
	memberID := args[0]

	cliPath := cliPathOfCmdOr(cmd, "account member remove")
	// Registry-flagged destructive: runs dry unless --force, so no prompt.
	dry := destructiveDryRun(cliPath, accountMemberForce)
	if dry {
		return outPayload("DRY RUN: Would remove account member", func() any {
			return map[string]string{"memberId": memberID}
		}, func() {
			printInfo("DRY RUN: Would remove account member '%s'", memberID)
			printInfo("Re-run with --force to remove the member for real")
		})
	}

	svc, err := getAccountMemberService()
	if err != nil {
		return outErr("failed to create account member service", err)
	}

	if err := svc.Remove(context.Background(), memberID); err != nil {
		return outErr("failed to remove account member", err)
	}

	return outPayload("Member removed successfully", func() any {
		return map[string]string{"memberId": memberID}
	}, func() {
		printSuccess("Account member '%s' removed", memberID)
	})
}

func runAccountRoleList(cmd *cobra.Command, args []string) error {
	svc, err := getAccountMemberService()
	if err != nil {
		return outErr("failed to create account member service", err)
	}

	roles, err := svc.RolesList(context.Background())
	if err != nil {
		return outErr("failed to list account roles", err)
	}

	return outResult(roles, func() {
		if len(roles) == 0 {
			printInfo("No roles found for this account")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ROLE ID\tNAME\tDESCRIPTION")
		for _, r := range roles {
			desc := r.Description
			if desc == "" {
				desc = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.ID, r.Name, desc)
		}
		w.Flush()
		printInfo("Total: %d role(s)", len(roles))
	})
}
