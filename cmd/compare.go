package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var comparePrefix string

var compareCmd = &cobra.Command{
	Use:   "compare [source-bucket] [dest-bucket]",
	Short: "Compare objects between two buckets",
	Long: `Compare objects between two R2 buckets.

Shows objects only in source, only in dest, with different sizes, and identical.

Examples:
  cosmoflare compare src-bucket dst-bucket
  cosmoflare compare src-bucket dst-bucket --prefix=images/
  cosmoflare compare src-bucket dst-bucket --json`,
	RunE: runCompare,
}

func init() {
	rootCmd.AddCommand(compareCmd)
	compareCmd.Flags().StringVar(&comparePrefix, "prefix", "", "Filter by key prefix")
}

type CompareDiff struct {
	Key        string `json:"key"`
	SourceSize int64  `json:"source_size"`
	DestSize   int64  `json:"dest_size"`
}

type CompareResult struct {
	OnlyInSource  []string      `json:"only_in_source"`
	OnlyInDest    []string      `json:"only_in_dest"`
	DifferentSize []CompareDiff `json:"different_size"`
	Same          []string      `json:"same"`
}

func runCompare(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("source and destination bucket names are required")
	}
	srcBucket := args[0]
	dstBucket := args[1]

	printInfo("Comparing buckets: %s <-> %s", srcBucket, dstBucket)

	client, err := getAPIClient()
	if err != nil {
		return outErr("failed to create API client", err)
	}

	srcMap, dstMap, err := compareListBuckets(client, srcBucket, dstBucket)
	if err != nil {
		return err
	}

	result := compareDiffBuckets(srcMap, dstMap)

	return outPayload("Comparison complete", func() any {
		return result
	}, func() {
		renderCompareResult(result, srcMap, dstMap)
	})
}

// compareListBuckets lists both buckets (filtered by the compare prefix) and
// returns key→size maps for the source and destination.
func compareListBuckets(client cosmoflare.R2Client, srcBucket, dstBucket string) (map[string]int64, map[string]int64, error) {
	srcResult, err := client.ListObjects(context.Background(), srcBucket, comparePrefix, "", 0, "")
	if err != nil {
		return nil, nil, outErr("failed to list source bucket", err)
	}

	dstResult, err := client.ListObjects(context.Background(), dstBucket, comparePrefix, "", 0, "")
	if err != nil {
		return nil, nil, outErr("failed to list destination bucket", err)
	}

	srcMap := make(map[string]int64, len(srcResult.Items))
	for _, obj := range srcResult.Items {
		srcMap[obj.Key] = obj.Size
	}

	dstMap := make(map[string]int64, len(dstResult.Items))
	for _, obj := range dstResult.Items {
		dstMap[obj.Key] = obj.Size
	}

	return srcMap, dstMap, nil
}

// compareDiffBuckets classifies keys as only-in-source, only-in-dest,
// different-size, or same.
func compareDiffBuckets(srcMap, dstMap map[string]int64) CompareResult {
	result := CompareResult{}

	for key, srcSize := range srcMap {
		dstSize, inDst := dstMap[key]
		if !inDst {
			result.OnlyInSource = append(result.OnlyInSource, key)
		} else if srcSize != dstSize {
			result.DifferentSize = append(result.DifferentSize, CompareDiff{Key: key, SourceSize: srcSize, DestSize: dstSize})
		} else {
			result.Same = append(result.Same, key)
		}
	}

	for key := range dstMap {
		if _, inSrc := srcMap[key]; !inSrc {
			result.OnlyInDest = append(result.OnlyInDest, key)
		}
	}

	return result
}

// renderCompareResult prints the tabular comparison report and summary line.
func renderCompareResult(result CompareResult, srcMap, dstMap map[string]int64) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if len(result.OnlyInSource) > 0 {
		fmt.Fprintln(w, "ONLY IN SOURCE\tSIZE")
		for _, key := range result.OnlyInSource {
			fmt.Fprintf(w, "%s\t%s\n", key, utils.FormatBytes(srcMap[key]))
		}
		fmt.Fprintln(w)
	}

	if len(result.OnlyInDest) > 0 {
		fmt.Fprintln(w, "ONLY IN DEST\tSIZE")
		for _, key := range result.OnlyInDest {
			fmt.Fprintf(w, "%s\t%s\n", key, utils.FormatBytes(dstMap[key]))
		}
		fmt.Fprintln(w)
	}

	if len(result.DifferentSize) > 0 {
		fmt.Fprintln(w, "DIFFERENT SIZE\tSOURCE\tDEST")
		for _, d := range result.DifferentSize {
			fmt.Fprintf(w, "%s\t%s\t%s\n", d.Key, utils.FormatBytes(d.SourceSize), utils.FormatBytes(d.DestSize))
		}
		fmt.Fprintln(w)
	}

	w.Flush()

	printInfo("Summary: %d only in source, %d only in dest, %d different size, %d same",
		len(result.OnlyInSource), len(result.OnlyInDest), len(result.DifferentSize), len(result.Same))
}
