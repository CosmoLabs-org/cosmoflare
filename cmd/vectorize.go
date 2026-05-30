package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"
)

var vectorizeCmd = &cobra.Command{
	Use:   "vectorize",
	Short: "Manage Cloudflare Vectorize indexes",
	Long: `Vector database management for AI workloads.

Commands:
  create    Create a vector index
  list      List all indexes
  get       Get index details
  delete    Delete an index
  insert    Insert vectors into an index
  query     Query an index for nearest neighbors

Examples:
  cosmoflare vectorize create my-index --dimensions=768 --metric=cosine
  cosmoflare vectorize list --json
  cosmoflare vectorize get my-index
  cosmoflare vectorize delete my-index --force
  cosmoflare vectorize insert my-index --file vectors.ndjson
  cosmoflare vectorize query my-index --values=0.1,0.2,0.3 --top-k=5`,
}

var (
	vectorizeDimensions int
	vectorizeMetric     string
	vectorizeForce      bool
	vectorizeFile       string
	vectorizeID         string
	vectorizeValues     string
	vectorizeTopK       int
)

var vectorizeCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a vector index",
	Long: `Create a new Cloudflare Vectorize index.

The index name must be unique within your account. Dimensions and metric
determine how vectors are stored and compared.

Supported metrics: cosine (default), euclidean, dot-product.

Examples:
  cosmoflare vectorize create embeddings --dimensions=768 --metric=cosine
  cosmoflare vectorize create search-idx --dimensions=1536 --metric=dot-product --json`,
	Args: cobra.ExactArgs(1),
	RunE: runVectorizeCreate,
}

var vectorizeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all vector indexes",
	Long: `List all Cloudflare Vectorize indexes in the current account.

Examples:
  cosmoflare vectorize list
  cosmoflare vectorize list --json`,
	RunE: runVectorizeList,
}

var vectorizeGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get index details",
	Long: `Get details of a specific Vectorize index.

Examples:
  cosmoflare vectorize get my-index
  cosmoflare vectorize get my-index --json`,
	Args: cobra.ExactArgs(1),
	RunE: runVectorizeGet,
}

var vectorizeDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a vector index",
	Long: `Delete a Cloudflare Vectorize index and all its vectors.

Examples:
  cosmoflare vectorize delete my-index
  cosmoflare vectorize delete my-index --force`,
	Args: cobra.ExactArgs(1),
	RunE: runVectorizeDelete,
}

var vectorizeInsertCmd = &cobra.Command{
	Use:   "insert [index-name]",
	Short: "Insert vectors into an index",
	Long: `Insert vectors into a Vectorize index from an NDJSON file.

Each line should be a JSON object with id, values, and optional metadata:
  {"id":"vec-1","values":[0.1,0.2,0.3],"metadata":{"label":"example"}}

Or provide a single vector inline with --id and --values.

Examples:
  cosmoflare vectorize insert my-index --file vectors.ndjson
  cosmoflare vectorize insert my-index --id=vec-1 --values=0.1,0.2,0.3
  cosmoflare vectorize insert my-index --file vectors.ndjson --json`,
	Args: cobra.ExactArgs(1),
	RunE: runVectorizeInsert,
}

var vectorizeQueryCmd = &cobra.Command{
	Use:   "query [index-name]",
	Short: "Query an index for nearest neighbors",
	Long: `Query a Vectorize index to find the most similar vectors.

Provide query values as a comma-separated list of floats.

Examples:
  cosmoflare vectorize query my-index --values=0.1,0.2,0.3 --top-k=5
  cosmoflare vectorize query my-index --values=0.1,0.2,0.3 --top-k=10 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runVectorizeQuery,
}

func init() {
	rootCmd.AddCommand(vectorizeCmd)

	vectorizeCmd.AddCommand(vectorizeCreateCmd)
	vectorizeCmd.AddCommand(vectorizeListCmd)
	vectorizeCmd.AddCommand(vectorizeGetCmd)
	vectorizeCmd.AddCommand(vectorizeDeleteCmd)
	vectorizeCmd.AddCommand(vectorizeInsertCmd)
	vectorizeCmd.AddCommand(vectorizeQueryCmd)

	vectorizeCreateCmd.Flags().IntVar(&vectorizeDimensions, "dimensions", 0, "Number of dimensions for vectors")
	vectorizeCreateCmd.Flags().StringVar(&vectorizeMetric, "metric", "cosine", "Distance metric: cosine|euclidean|dot-product")

	vectorizeDeleteCmd.Flags().BoolVar(&vectorizeForce, "force", false, "Skip confirmation prompt")

	vectorizeInsertCmd.Flags().StringVar(&vectorizeFile, "file", "", "NDJSON file containing vectors")
	vectorizeInsertCmd.Flags().StringVar(&vectorizeID, "id", "", "Vector ID (for single vector insert)")
	vectorizeInsertCmd.Flags().StringVar(&vectorizeValues, "values", "", "Comma-separated float values (for single vector or query)")

	vectorizeQueryCmd.Flags().StringVar(&vectorizeValues, "values", "", "Comma-separated float values for query vector")
	vectorizeQueryCmd.Flags().IntVar(&vectorizeTopK, "top-k", 10, "Number of nearest neighbors to return")
}

func getVectorizeService() (*r2go2.VectorizeService, error) {
	return r2go2.NewVectorizeServiceFromCreds(AccountID, APIToken)
}

func runVectorizeCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	if DryRun {
		if JSONOutput {
			return printJSON(map[string]interface{}{
				"dry_run": true, "action": "create_index",
				"name": name, "dimensions": vectorizeDimensions, "metric": vectorizeMetric,
			})
		}
		printInfo("DRY RUN: Would create index '%s' (dimensions=%d, metric=%s)", name, vectorizeDimensions, vectorizeMetric)
		return nil
	}

	svc, err := getVectorizeService()
	if err != nil {
		return fmt.Errorf("failed to create vectorize service: %w", err)
	}

	idx, err := svc.CreateIndex(context.Background(), name, vectorizeDimensions, vectorizeMetric)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	if JSONOutput {
		return printJSON(map[string]interface{}{"success": true, "data": idx})
	}
	printSuccess("Created index '%s' (dimensions=%d, metric=%s)", idx.Name, idx.Dimensions, idx.Metric)
	return nil
}

func runVectorizeList(cmd *cobra.Command, args []string) error {
	svc, err := getVectorizeService()
	if err != nil {
		return fmt.Errorf("failed to create vectorize service: %w", err)
	}

	indexes, err := svc.ListIndexes(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list indexes: %w", err)
	}

	if JSONOutput {
		return printJSON(indexes)
	}

	if len(indexes) == 0 {
		printInfo("No vectorize indexes found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDIMENSIONS\tMETRIC\tVECTORS")
	for _, idx := range indexes {
		fmt.Fprintf(w, "%s\t%d\t%s\t%d\n", idx.Name, idx.Dimensions, idx.Metric, idx.VectorCount)
	}
	w.Flush()
	return nil
}

func runVectorizeGet(cmd *cobra.Command, args []string) error {
	name := args[0]

	svc, err := getVectorizeService()
	if err != nil {
		return fmt.Errorf("failed to create vectorize service: %w", err)
	}

	idx, err := svc.GetIndex(context.Background(), name)
	if err != nil {
		return fmt.Errorf("failed to get index: %w", err)
	}

	if JSONOutput {
		return printJSON(idx)
	}

	fmt.Printf("Name:       %s\n", idx.Name)
	fmt.Printf("Dimensions: %d\n", idx.Dimensions)
	fmt.Printf("Metric:     %s\n", idx.Metric)
	fmt.Printf("Vectors:    %d\n", idx.VectorCount)
	if idx.Description != "" {
		fmt.Printf("Description: %s\n", idx.Description)
	}
	return nil
}

func runVectorizeDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	if DryRun {
		if JSONOutput {
			return printJSON(map[string]interface{}{"dry_run": true, "action": "delete_index", "name": name})
		}
		printInfo("DRY RUN: Would delete index '%s'", name)
		return nil
	}

	svc, err := getVectorizeService()
	if err != nil {
		return fmt.Errorf("failed to create vectorize service: %w", err)
	}

	if err := svc.DeleteIndex(context.Background(), name); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	if JSONOutput {
		return printJSON(map[string]interface{}{"success": true, "message": fmt.Sprintf("Index '%s' deleted", name)})
	}
	printSuccess("Deleted index '%s'", name)
	return nil
}

func runVectorizeInsert(cmd *cobra.Command, args []string) error {
	indexName := args[0]

	svc, err := getVectorizeService()
	if err != nil {
		return fmt.Errorf("failed to create vectorize service: %w", err)
	}

	var vectors []r2go2.VectorizeVector

	if vectorizeFile != "" {
		data, err := os.ReadFile(vectorizeFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line == "" {
				continue
			}
			var v r2go2.VectorizeVector
			if err := json.Unmarshal([]byte(line), &v); err != nil {
				return fmt.Errorf("failed to parse vector line: %w", err)
			}
			vectors = append(vectors, v)
		}
	} else if vectorizeID != "" && vectorizeValues != "" {
		vals, err := parseFloatSlice(vectorizeValues)
		if err != nil {
			return fmt.Errorf("invalid --values: %w", err)
		}
		vectors = append(vectors, r2go2.VectorizeVector{ID: vectorizeID, Values: vals})
	} else {
		return fmt.Errorf("provide --file or both --id and --values")
	}

	if DryRun {
		if JSONOutput {
			return printJSON(map[string]interface{}{"dry_run": true, "action": "insert", "index": indexName, "count": len(vectors)})
		}
		printInfo("DRY RUN: Would insert %d vectors into '%s'", len(vectors), indexName)
		return nil
	}

	if err := svc.InsertVectors(context.Background(), indexName, vectors); err != nil {
		return fmt.Errorf("failed to insert vectors: %w", err)
	}

	if JSONOutput {
		return printJSON(map[string]interface{}{"success": true, "inserted": len(vectors), "index": indexName})
	}
	printSuccess("Inserted %d vectors into '%s'", len(vectors), indexName)
	return nil
}

func runVectorizeQuery(cmd *cobra.Command, args []string) error {
	indexName := args[0]

	if vectorizeValues == "" {
		return fmt.Errorf("--values is required for query")
	}

	vals, err := parseFloatSlice(vectorizeValues)
	if err != nil {
		return fmt.Errorf("invalid --values: %w", err)
	}

	svc, err := getVectorizeService()
	if err != nil {
		return fmt.Errorf("failed to create vectorize service: %w", err)
	}

	results, err := svc.QueryVectors(context.Background(), indexName, vals, vectorizeTopK)
	if err != nil {
		return fmt.Errorf("failed to query index: %w", err)
	}

	if JSONOutput {
		return printJSON(map[string]interface{}{"success": true, "matches": results})
	}

	if len(results) == 0 {
		printInfo("No matches found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSCORE")
	for _, r := range results {
		fmt.Fprintf(w, "%s\t%.4f\n", r.ID, r.Score)
	}
	w.Flush()
	return nil
}

func parseFloatSlice(s string) ([]float64, error) {
	parts := strings.Split(s, ",")
	vals := make([]float64, 0, len(parts))
	for _, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float %q: %w", p, err)
		}
		vals = append(vals, f)
	}
	return vals, nil
}
