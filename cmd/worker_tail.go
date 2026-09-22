package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// Live Worker log tail (FEAT-021 wave 2). Subscribes to
// WorkerService.TailLogs and streams entries until the channel closes or
// the context is cancelled (ctrl-c exits 0 with a summary line).

var workerTailCmd = &cobra.Command{
	Use:   "tail <name>",
	Short: "Stream live logs for a Worker",
	Long: `Stream log entries for a Worker as they are produced.

The stream runs until you press ctrl-c (which exits 0 with a summary
line) or the server closes it. Use --format json for one JSON object per
line, suitable for piping into jq.`,
	Example: `  # Follow a Worker's logs (text)
  cosmoflare worker tail api-gateway

  # One JSON object per line
  cosmoflare worker tail api-gateway --format json

  # Pipe JSON entries through jq
  cosmoflare worker tail api-gateway --format json | jq .message`,
	Args: prefixedResourceArgs(cobra.ExactArgs(1)),
	RunE: runWorkerTail,
}

var workerTailFormat string

// registerWorkerTailCmds attaches the tail command to the worker tree. The
// flag guard keeps re-registration idempotent.
func registerWorkerTailCmds(parent *cobra.Command) {
	if workerTailCmd.Flags().Lookup("format") == nil {
		workerTailCmd.Flags().StringVar(&workerTailFormat, "format", "text", "Output format (text|json)")
	}
	parent.AddCommand(workerTailCmd)
}

func runWorkerTail(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("worker name is required")
	}
	name := args[0]

	jsonMode := workerTailFormat == "json"
	if workerTailFormat != "text" && workerTailFormat != "json" {
		return outErrf("invalid --format %q: must be text or json", workerTailFormat)
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ch, err := svc.TailLogs(ctx, name, &cosmoflare.TailOptions{})
	if err != nil {
		return outErr(fmt.Sprintf("failed to tail logs for worker %q", name), err)
	}

	return streamTailEntries(ctx, ch, jsonMode)
}

func streamTailEntries(ctx context.Context, ch <-chan *cosmoflare.LogEntry, jsonMode bool) error {
	count := 0
	for {
		select {
		case <-ctx.Done():
			printInfo("streamed %d entries", count)
			return nil
		case e, ok := <-ch:
			if !ok {
				printInfo("streamed %d entries", count)
				return nil
			}
			fmt.Println(formatTailEntry(e, jsonMode))
			count++
		}
	}
}

// formatTailEntry renders one log line: a JSON object per line in json
// mode, "TIMESTAMP LEVEL MESSAGE" in text mode.
func formatTailEntry(e *cosmoflare.LogEntry, jsonMode bool) string {
	if e == nil {
		return ""
	}
	if jsonMode {
		b, err := json.Marshal(e)
		if err != nil {
			return fmt.Sprintf("timestamp=%s level=%s message=%q", e.Timestamp.Format(time.RFC3339), e.Level, e.Message)
		}
		return string(b)
	}
	return fmt.Sprintf("%s %s %s", e.Timestamp.UTC().Format(time.RFC3339), e.Level, e.Message)
}
