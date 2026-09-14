package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

var decodeContext string

var decodeCmd = &cobra.Command{
	Use:   "decode <code>",
	Short: "Decode a Cloudflare API error code into its cause and fix",
	Long: `Decode a Cloudflare API error code using cosmoflare's knowledge packs.

Prints the cause and the fix for codes the knowledge layer understands.
Unknown codes exit non-zero with a clear message — no fabricated verdicts.

Examples:
  cosmoflare decode 10405
  cosmoflare decode 1000 --context phase-entrypoint
  cosmoflare decode 20155 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid error code %q: expected an integer", args[0])
		}
		d := knowledge.LookupDecode(code, decodeContext)
		if d == nil {
			return fmt.Errorf("no knowledge entry for code %d (context %q) — the knowledge layer never fabricates a verdict", code, decodeContext)
		}
		return outResult(d, func() {
			cmd.Printf("code:    %d\ncontext: %s\ncause:   %s\nfix:     %s\n", d.Code, d.Context, d.Cause, d.Fix)
		})
	},
}

func init() {
	rootCmd.AddCommand(decodeCmd)
	decodeCmd.Flags().StringVar(&decodeContext, "context", "", "refine the decode by context (e.g. phase-entrypoint)")
}
