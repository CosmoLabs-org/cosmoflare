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

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
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
	vectorizeDimensions   int
	vectorizeMetric       string
	vectorizeForce        bool
	vectorizeFile         string
	vectorizeID           string
	vectorizeValues       string
	vectorizeTopK         int
	vectorizeUpsertFile   string
	vectorizeGetVectorID  string
	vectorizeDeleteIDsCSV string
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

var vectorizeUpsertCmd = &cobra.Command{
	Use:   "upsert [index-name]",
	Short: "Upsert vectors into an index",
	Long: `Upsert (insert or update) vectors in a Vectorize index from a JSON file.

Unlike insert, upsert overwrites vectors that already exist by ID. The file
must be a JSON array (not NDJSON):
  [{"id":"vec-1","values":[0.1,0.2,0.3],"metadata":{"label":"example"}}]

Batches are capped at 50 vectors per call.

Examples:
  cosmoflare vectorize upsert my-index --file vectors.json
  cosmoflare vectorize upsert my-index --file vectors.json --json`,
	Args: cobra.ExactArgs(1),
	RunE: runVectorizeUpsert,
}

var vectorizeGetVectorCmd = &cobra.Command{
	Use:   "get-vector [index-name]",
	Short: "Get a single vector by ID",
	Long: `Get a single vector's values and metadata from a Vectorize index.

Named "get-vector" (not "get") to avoid colliding with the existing
"vectorize get" command, which fetches index details.

Examples:
  cosmoflare vectorize get-vector my-index --id vec-1
  cosmoflare vectorize get-vector my-index --id vec-1 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runVectorizeGetVector,
}

var vectorizeDeleteVectorsCmd = &cobra.Command{
	Use:   "delete-vectors [index-name]",
	Short: "Delete vectors by ID",
	Long: `Delete one or more vectors from a Vectorize index by ID.

Named "delete-vectors" (not "delete") to avoid colliding with the existing
"vectorize delete" command, which deletes the whole index.

Examples:
  cosmoflare vectorize delete-vectors my-index --ids vec-1,vec-2
  cosmoflare vectorize delete-vectors my-index --ids vec-1 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runVectorizeDeleteVectors,
}

var vectorizeNamespacesCmd = &cobra.Command{
	Use:   "namespaces [index-name]",
	Short: "List namespaces in an index",
	Long: `List the distinct vector namespaces present in a Vectorize index.

Examples:
  cosmoflare vectorize namespaces my-index
  cosmoflare vectorize namespaces my-index --json`,
	Args: cobra.ExactArgs(1),
	RunE: runVectorizeNamespaces,
}

func init() {
	rootCmd.AddCommand(vectorizeCmd)

	vectorizeCmd.AddCommand(vectorizeCreateCmd)
	vectorizeCmd.AddCommand(vectorizeListCmd)
	vectorizeCmd.AddCommand(vectorizeGetCmd)
	vectorizeCmd.AddCommand(vectorizeDeleteCmd)
	vectorizeCmd.AddCommand(vectorizeInsertCmd)
	vectorizeCmd.AddCommand(vectorizeQueryCmd)
	vectorizeCmd.AddCommand(vectorizeUpsertCmd)
	vectorizeCmd.AddCommand(vectorizeGetVectorCmd)
	vectorizeCmd.AddCommand(vectorizeDeleteVectorsCmd)
	vectorizeCmd.AddCommand(vectorizeNamespacesCmd)

	vectorizeCreateCmd.Flags().IntVar(&vectorizeDimensions, "dimensions", 0, "Number of dimensions for vectors")
	vectorizeCreateCmd.Flags().StringVar(&vectorizeMetric, "metric", "cosine", "Distance metric: cosine|euclidean|dot-product")

	vectorizeDeleteCmd.Flags().BoolVar(&vectorizeForce, "force", false, "Skip confirmation prompt")

	vectorizeInsertCmd.Flags().StringVar(&vectorizeFile, "file", "", "NDJSON file containing vectors")
	vectorizeInsertCmd.Flags().StringVar(&vectorizeID, "id", "", "Vector ID (for single vector insert)")
	vectorizeInsertCmd.Flags().StringVar(&vectorizeValues, "values", "", "Comma-separated float values (for single vector or query)")

	vectorizeQueryCmd.Flags().StringVar(&vectorizeValues, "values", "", "Comma-separated float values for query vector")
	vectorizeQueryCmd.Flags().IntVar(&vectorizeTopK, "top-k", 10, "Number of nearest neighbors to return")

	vectorizeUpsertCmd.Flags().StringVar(&vectorizeUpsertFile, "file", "", "JSON file containing an array of vectors")

	vectorizeGetVectorCmd.Flags().StringVar(&vectorizeGetVectorID, "id", "", "Vector ID to fetch")

	vectorizeDeleteVectorsCmd.Flags().StringVar(&vectorizeDeleteIDsCSV, "ids", "", "Comma-separated vector IDs to delete")
}

func getVectorizeService() (*cosmoflare.VectorizeService, error) {
	return cosmoflare.NewVectorizeServiceFromCreds(AccountID, APIToken)
}

func runVectorizeCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	if DryRun {
		return outResult(map[string]interface{}{
			"dry_run": true, "action": "create_index",
			"name": name, "dimensions": vectorizeDimensions, "metric": vectorizeMetric,
		}, func() {
			printInfo("DRY RUN: Would create index '%s' (dimensions=%d, metric=%s)", name, vectorizeDimensions, vectorizeMetric)
		})
	}

	svc, err := getVectorizeService()
	if err != nil {
		return outErr("failed to create vectorize service", err)
	}

	idx, err := svc.CreateIndex(context.Background(), name, vectorizeDimensions, vectorizeMetric)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return outResult(map[string]interface{}{"success": true, "data": idx}, func() {
		printSuccess("Created index '%s' (dimensions=%d, metric=%s)", idx.Name, idx.Dimensions, idx.Metric)
	})
}

func runVectorizeList(cmd *cobra.Command, args []string) error {
	svc, err := getVectorizeService()
	if err != nil {
		return outErr("failed to create vectorize service", err)
	}

	indexes, err := svc.ListIndexes(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list indexes: %w", err)
	}

	return outResult(indexes, func() {
		if len(indexes) == 0 {
			printInfo("No vectorize indexes found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDIMENSIONS\tMETRIC\tVECTORS")
		for _, idx := range indexes {
			fmt.Fprintf(w, "%s\t%d\t%s\t%d\n", idx.Name, idx.Dimensions, idx.Metric, idx.VectorCount)
		}
		w.Flush()
	})
}

func runVectorizeGet(cmd *cobra.Command, args []string) error {
	name := args[0]

	svc, err := getVectorizeService()
	if err != nil {
		return outErr("failed to create vectorize service", err)
	}

	idx, err := svc.GetIndex(context.Background(), name)
	if err != nil {
		return fmt.Errorf("failed to get index: %w", err)
	}

	return outResult(idx, func() {
		fmt.Printf("Name:       %s\n", idx.Name)
		fmt.Printf("Dimensions: %d\n", idx.Dimensions)
		fmt.Printf("Metric:     %s\n", idx.Metric)
		fmt.Printf("Vectors:    %d\n", idx.VectorCount)
		if idx.Description != "" {
			fmt.Printf("Description: %s\n", idx.Description)
		}
	})
}

func runVectorizeDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	if DryRun {
		return outResult(map[string]interface{}{"dry_run": true, "action": "delete_index", "name": name}, func() {
			printInfo("DRY RUN: Would delete index '%s'", name)
		})
	}

	svc, err := getVectorizeService()
	if err != nil {
		return outErr("failed to create vectorize service", err)
	}

	if err := svc.DeleteIndex(context.Background(), name); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	return outResult(map[string]interface{}{"success": true, "message": fmt.Sprintf("Index '%s' deleted", name)}, func() {
		printSuccess("Deleted index '%s'", name)
	})
}

func runVectorizeInsert(cmd *cobra.Command, args []string) error {
	indexName := args[0]

	svc, err := getVectorizeService()
	if err != nil {
		return outErr("failed to create vectorize service", err)
	}

	var vectors []cosmoflare.VectorizeVector

	if vectorizeFile != "" {
		data, err := os.ReadFile(vectorizeFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line == "" {
				continue
			}
			var v cosmoflare.VectorizeVector
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
		vectors = append(vectors, cosmoflare.VectorizeVector{ID: vectorizeID, Values: vals})
	} else {
		return fmt.Errorf("provide --file or both --id and --values")
	}

	if DryRun {
		return outResult(map[string]interface{}{"dry_run": true, "action": "insert", "index": indexName, "count": len(vectors)}, func() {
			printInfo("DRY RUN: Would insert %d vectors into '%s'", len(vectors), indexName)
		})
	}

	if err := svc.InsertVectors(context.Background(), indexName, vectors); err != nil {
		return fmt.Errorf("failed to insert vectors: %w", err)
	}

	return outResult(map[string]interface{}{"success": true, "inserted": len(vectors), "index": indexName}, func() {
		printSuccess("Inserted %d vectors into '%s'", len(vectors), indexName)
	})
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
		return outErr("failed to create vectorize service", err)
	}

	results, err := svc.QueryVectors(context.Background(), indexName, vals, vectorizeTopK)
	if err != nil {
		return fmt.Errorf("failed to query index: %w", err)
	}

	return outResult(map[string]interface{}{"success": true, "matches": results}, func() {
		if len(results) == 0 {
			printInfo("No matches found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSCORE")
		for _, r := range results {
			fmt.Fprintf(w, "%s\t%.4f\n", r.ID, r.Score)
		}
		w.Flush()
	})
}

func runVectorizeUpsert(cmd *cobra.Command, args []string) error {
	indexName := args[0]

	if vectorizeUpsertFile == "" {
		return fmt.Errorf("--file is required")
	}
	data, err := os.ReadFile(vectorizeUpsertFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	var vectors []cosmoflare.VectorizeVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		return fmt.Errorf("failed to parse vectors file: %w", err)
	}

	if DryRun {
		return outResult(map[string]interface{}{"dry_run": true, "action": "upsert", "index": indexName, "count": len(vectors)}, func() {
			printInfo("DRY RUN: Would upsert %d vectors into '%s'", len(vectors), indexName)
		})
	}

	svc, err := getVectorizeService()
	if err != nil {
		return outErr("failed to create vectorize service", err)
	}

	result, err := svc.UpsertVectors(context.Background(), indexName, vectors)
	if err != nil {
		return fmt.Errorf("failed to upsert vectors: %w", err)
	}

	return outResult(map[string]interface{}{"success": true, "data": result}, func() {
		printSuccess("Upserted %d vectors into '%s' (mutation: %s)", len(vectors), indexName, result.MutationID)
	})
}

func runVectorizeGetVector(cmd *cobra.Command, args []string) error {
	indexName := args[0]

	if vectorizeGetVectorID == "" {
		return fmt.Errorf("--id is required")
	}

	svc, err := getVectorizeService()
	if err != nil {
		return outErr("failed to create vectorize service", err)
	}

	v, err := svc.GetVector(context.Background(), indexName, vectorizeGetVectorID)
	if err != nil {
		return fmt.Errorf("failed to get vector: %w", err)
	}

	return outResult(v, func() {
		fmt.Printf("ID:     %s\n", v.ID)
		fmt.Printf("Values: %v\n", v.Values)
		if len(v.Metadata) > 0 {
			fmt.Printf("Metadata: %v\n", v.Metadata)
		}
	})
}

func runVectorizeDeleteVectors(cmd *cobra.Command, args []string) error {
	indexName := args[0]

	if vectorizeDeleteIDsCSV == "" {
		return fmt.Errorf("--ids is required")
	}
	parts := strings.Split(vectorizeDeleteIDsCSV, ",")
	ids := make([]string, 0, len(parts))
	for _, p := range parts {
		ids = append(ids, strings.TrimSpace(p))
	}

	if DryRun {
		return outResult(map[string]interface{}{"dry_run": true, "action": "delete_vectors", "index": indexName, "ids": ids}, func() {
			printInfo("DRY RUN: Would delete %d vectors from '%s'", len(ids), indexName)
		})
	}

	svc, err := getVectorizeService()
	if err != nil {
		return outErr("failed to create vectorize service", err)
	}

	result, err := svc.DeleteVectors(context.Background(), indexName, ids)
	if err != nil {
		return fmt.Errorf("failed to delete vectors: %w", err)
	}

	return outResult(map[string]interface{}{"success": true, "data": result}, func() {
		printSuccess("Deleted %d vector(s) from '%s'", len(ids), indexName)
	})
}

func runVectorizeNamespaces(cmd *cobra.Command, args []string) error {
	indexName := args[0]

	svc, err := getVectorizeService()
	if err != nil {
		return outErr("failed to create vectorize service", err)
	}

	namespaces, err := svc.ListNamespaces(context.Background(), indexName)
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	return outResult(namespaces, func() {
		if len(namespaces) == 0 {
			printInfo("No namespaces found in '%s'", indexName)
			return
		}
		for _, ns := range namespaces {
			fmt.Println(ns)
		}
	})
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
