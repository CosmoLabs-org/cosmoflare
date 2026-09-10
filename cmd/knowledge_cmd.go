package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

var knowledgeCmd = &cobra.Command{
	Use:   "knowledge",
	Short: "Inspect the loaded Cloudflare knowledge packs",
	Long: `List the knowledge packs compiled into this binary.

Each pack carries an endpoint registry, error decodes, plan caps, and field
invariants for one Cloudflare product. Routes inside a pack's scope that are
not registered are blocked before send.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		packs, err := knowledge.Load()
		if err != nil {
			return fmt.Errorf("knowledge packs failed to load: %w", err)
		}
		if JSONOutput {
			return printJSON(packs)
		}
		for _, p := range packs {
			fmt.Printf("%s: %d endpoints, %d error decodes, %d plan caps, %d invariants, %d traffic classes (scopes: %v)\n",
				p.Product, len(p.Endpoints), len(p.Errors), len(p.PlanCaps), len(p.Invariants), len(p.TrafficClasses), p.Scopes)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(knowledgeCmd)
}
