package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"
	"github.com/spf13/cobra"
)

var (
	watchPrefix   string
	watchExclude  []string
	watchInterval time.Duration
	watchDelete   bool
)

var watchCmd = &cobra.Command{
	Use:   "watch [bucket] [directory]",
	Short: "Auto-sync a local directory to an R2 bucket on file changes",
	Long: `Watch a local directory for file changes and automatically sync them to an R2 bucket.

Uses polling-based change detection (no external dependencies). On each tick,
the directory tree is walked and compared to the previous snapshot. New and
modified files are uploaded; deleted files are removed from the bucket when
--delete is enabled.

The command runs until interrupted with Ctrl+C (SIGINT) or SIGTERM.

Output:
  [+] uploaded   path/to/new-file.txt
  [~] updated    path/to/changed.txt
  [-] deleted    path/to/removed.txt

In --json mode, each event is emitted as a single NDJSON line.

Examples:
  r2go2 watch my-bucket                              # Watch cwd, sync to bucket root
  r2go2 watch my-bucket ./dist                        # Watch ./dist directory
  r2go2 watch my-bucket ./build --prefix=assets/      # Upload under assets/ prefix
  r2go2 watch my-bucket . --exclude="*.log,*.tmp"     # Skip log and tmp files
  r2go2 watch my-bucket . --interval=5s               # Poll every 5 seconds
  r2go2 watch my-bucket . --delete                    # Sync deletions too
  r2go2 watch my-bucket . --dry-run                   # Preview without uploading
  r2go2 watch my-bucket . --json                      # NDJSON event stream`,
	RunE: runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().StringVar(&watchPrefix, "prefix", "", "R2 key prefix (prepended to relative file paths)")
	watchCmd.Flags().StringSliceVar(&watchExclude, "exclude", nil, "Glob patterns to exclude (comma-separated, e.g. '*.log,*.tmp')")
	watchCmd.Flags().DurationVar(&watchInterval, "interval", time.Second, "Poll interval for change detection")
	watchCmd.Flags().BoolVar(&watchDelete, "delete", false, "Sync file deletions (remove from R2 when local file is deleted)")
}

// watchEvent is the JSON representation of a single sync event.
type watchEvent struct {
	Time   string `json:"time"`
	Type   string `json:"type"`   // "uploaded", "updated", "deleted", "error"
	Path   string `json:"path"`   // Local relative path
	R2Key  string `json:"r2_key"` // R2 object key
	Size   int64  `json:"size,omitempty"`
	DryRun bool   `json:"dry_run,omitempty"`
	Error  string `json:"error,omitempty"`
}

func runWatch(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("bucket name is required\n\nUsage: r2go2 watch <bucket> [directory]")
	}

	bucket := args[0]
	dir := "."
	if len(args) >= 2 {
		dir = args[1]
	}

	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Create the file watcher
	fw, err := r2go2.NewFileWatcher(absDir, &r2go2.WatcherOptions{
		Prefix:   watchPrefix,
		Exclude:  watchExclude,
		Interval: watchInterval,
		Delete:   watchDelete,
	})
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create watcher: %v", err))
		}
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Create R2 client (unless dry-run)
	var r2client r2go2.R2Client
	if !DryRun {
		r2client, err = r2go2.NewClient(
			r2go2.WithAccountID(AccountID),
			r2go2.WithAPIToken(APIToken),
		)
		if err != nil {
			if JSONOutput {
				return printErrorJSON(fmt.Sprintf("failed to create R2 client: %v", err))
			}
			return fmt.Errorf("failed to create R2 client: %w", err)
		}
	}

	// Take initial snapshot
	lastSnap, err := fw.Snapshot()
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to scan directory: %v", err))
		}
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if !JSONOutput {
		printInfo("Watching %s → bucket %q (prefix=%q, interval=%s, delete=%v)",
			absDir, bucket, watchPrefix, watchInterval, watchDelete)
		printInfo("Initial scan: %d file(s)", len(lastSnap))
		if DryRun {
			printWarning("DRY RUN: no files will be uploaded or deleted")
		}
		printInfo("Press Ctrl+C to stop")
	}

	// Set up signal handling for clean exit
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Polling loop
	ticker := time.NewTicker(fw.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if !JSONOutput {
				printInfo("Stopping watch...")
			}
			return nil
		case <-ticker.C:
			curSnap, err := fw.Snapshot()
			if err != nil {
				emitWatchError("scan error: "+err.Error(), "")
				continue
			}

			changes := fw.Diff(lastSnap, curSnap)
			if len(changes) == 0 {
				continue
			}

			for _, change := range changes {
				switch change.Type {
				case r2go2.ChangeAdded, r2go2.ChangeModified:
					syncUpload(ctx, r2client, bucket, change)
				case r2go2.ChangeDeleted:
					if watchDelete {
						syncDelete(ctx, r2client, bucket, change)
					} else if Verbose && !JSONOutput {
						printInfo("Skipped deletion of %s (use --delete to sync)", change.Path)
					}
				}
			}

			lastSnap = curSnap
		}
	}
}

func syncUpload(ctx context.Context, client r2go2.R2Client, bucket string, change r2go2.FileChange) {
	action := "uploaded"
	symbol := "[+]"
	if change.Type == r2go2.ChangeModified {
		action = "updated"
		symbol = "[~]"
	}

	if DryRun {
		emitWatchEvent(action, change, true)
		return
	}

	f, err := os.Open(change.FullPath)
	if err != nil {
		emitWatchError(fmt.Sprintf("failed to open %s: %v", change.Path, err), change.Path)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		emitWatchError(fmt.Sprintf("failed to stat %s: %v", change.Path, err), change.Path)
		return
	}

	_, err = client.Upload(ctx, bucket, change.R2Key, f, info.Size())
	if err != nil {
		emitWatchError(fmt.Sprintf("failed to upload %s: %v", change.Path, err), change.Path)
		return
	}

	if JSONOutput {
		emitWatchEvent(action, change, false)
	} else {
		fmt.Printf("%s %s %s (%d bytes)\n", symbol, action, change.Path, change.Size)
	}
}

func syncDelete(ctx context.Context, client r2go2.R2Client, bucket string, change r2go2.FileChange) {
	if DryRun {
		emitWatchEvent("deleted", change, true)
		return
	}

	err := client.DeleteObject(ctx, bucket, change.R2Key)
	if err != nil {
		emitWatchError(fmt.Sprintf("failed to delete %s: %v", change.R2Key, err), change.Path)
		return
	}

	if JSONOutput {
		emitWatchEvent("deleted", change, false)
	} else {
		fmt.Printf("[-] deleted  %s\n", change.Path)
	}
}

func emitWatchEvent(action string, change r2go2.FileChange, dryRun bool) {
	if JSONOutput {
		evt := watchEvent{
			Time:   time.Now().UTC().Format(time.RFC3339),
			Type:   action,
			Path:   change.Path,
			R2Key:  change.R2Key,
			Size:   change.Size,
			DryRun: dryRun,
		}
		data, _ := json.Marshal(evt)
		fmt.Println(string(data))
	} else {
		label := change.Type.Symbol()
		suffix := ""
		if dryRun {
			suffix = " (dry-run)"
		}
		if change.Size > 0 {
			fmt.Printf("%s %s %s (%d bytes)%s\n", label, action, change.Path, change.Size, suffix)
		} else {
			fmt.Printf("%s %s %s%s\n", label, action, change.Path, suffix)
		}
	}
}

func emitWatchError(msg, path string) {
	if JSONOutput {
		evt := watchEvent{
			Time:  time.Now().UTC().Format(time.RFC3339),
			Type:  "error",
			Path:  path,
			Error: msg,
		}
		data, _ := json.Marshal(evt)
		fmt.Println(string(data))
	} else {
		printError("%s", msg)
	}
}
