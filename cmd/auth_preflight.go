/*
Package cmd — auth preflight (FEAT-072): check token scopes BEFORE an
operation fails with an opaque "request is not authorized".

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli"
	"github.com/spf13/cobra"
)

// authPreflightCmd represents the auth preflight command
var authPreflightCmd = &cobra.Command{
	Use:   "preflight <command-family>",
	Short: "Check the token's REAL scopes against a command family's requirements",
	Long: `Permissions preflight: compare the token's actual scopes (read live via
Cloudflare's token-verification endpoint) with what a command family needs,
before an operation fails with an opaque "request is not authorized".

Known families:
` + preflightFamilyList() + `
Example:
  cosmoflare auth preflight cache
  cosmoflare auth preflight dns --json

Exit status: 0 when every required scope is present, 1 with a one-line
"missing scope: Zone:Rulesets:Edit" list when gaps exist.`,
	Args: cobra.ExactArgs(1),
	RunE: runAuthPreflight,
}

func init() {
	authCmd.AddCommand(authPreflightCmd)
}

// preflightFamilyList renders the known families for the long help.
func preflightFamilyList() string {
	return "  " + strings.Join(cli.ScopeFamilies(), ", ") + "\n"
}

// preflightMissingScopes and preflightRequiredScopes are thin seams over
// internal/cli so tests can exercise the matcher without network access.
func preflightMissingScopes(family string, present []string) []string {
	return cli.MissingScopes(family, present)
}

func preflightRequiredScopes(family string) ([]cli.TokenScope, bool) {
	return cli.RequiredScopes(family)
}

func runAuthPreflight(cmd *cobra.Command, args []string) error {
	family := args[0]

	required, ok := cli.RequiredScopes(family)
	if !ok {
		return fmt.Errorf("unknown command family %q. Known families: %s",
			family, strings.Join(cli.ScopeFamilies(), ", "))
	}
	requiredNames := make([]string, len(required))
	for i, r := range required {
		requiredNames[i] = r.String()
	}

	// Resolve the token exactly like auth token/status: environment wins,
	// then the stored profile.
	token, _, _, err := resolveStoredCredentials()
	if err != nil {
		return fmt.Errorf("preflight needs a token: %w", err)
	}

	present, err := fetchTokenScopes(context.Background(), token)
	if err != nil {
		return fmt.Errorf("could not read token scopes: %w", err)
	}

	missing := cli.MissingScopes(family, present)

	if JSONOutput {
		if len(missing) > 0 {
			_ = printJSON(map[string]interface{}{
				"success":        false,
				"command_family": family,
				"required":       requiredNames,
				"present":        present,
				"missing":        missing,
			})
			// Returned error makes the process exit 1 (Decision
			// 2026-09-14: JSON-mode failures must exit non-zero).
			return fmt.Errorf("missing scope: %s", strings.Join(missing, ", "))
		}
		return printSuccessJSON("preflight passed: all required scopes present", map[string]interface{}{
			"command_family": family,
			"required":       requiredNames,
			"present":        present,
			"missing":        []string{},
		})
	}

	printInfo("Command family: %s", family)
	printInfo("Required scopes: %s", strings.Join(requiredNames, ", "))
	printInfo("Token scopes: %s", strings.Join(present, ", "))

	if len(missing) > 0 {
		// One-line diagnosis instead of a mid-task API rejection (FB-30).
		fmt.Println("missing scope: " + strings.Join(missing, ", "))
		// Silence cobra's "Error: …" echo so the line above stays the
		// single failure output; the returned error still exits 1.
		cmd.SilenceErrors = true
		return errors.New("missing scope: " + strings.Join(missing, ", "))
	}

	printSuccess("✅ preflight passed: all required scopes present")
	return nil
}
