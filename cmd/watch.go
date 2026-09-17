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

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
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
  cosmoflare watch my-bucket                              # Watch cwd, sync to bucket root
  cosmoflare watch my-bucket ./dist                        # Watch ./dist directory
  cosmoflare watch my-bucket ./build --prefix=assets/      # Upload under assets/ prefix
  cosmoflare watch my-bucket . --exclude="*.log,*.tmp"     # Skip log and tmp files
  cosmoflare watch my-bucket . --interval=5s               # Poll every 5 seconds
  cosmoflare watch my-bucket . --delete                    # Sync deletions too
  cosmoflare watch my-bucket . --dry-run                   # Preview without uploading
  cosmoflare watch my-bucket . --json                      # NDJSON event stream`,
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

// watchState carries the per-run state shared between the watch loop and its
// helpers.
type watchState struct {
	ctx      context.Context
	fw       *cosmoflare.FileWatcher
	r2client cosmoflare.R2Client
	bucket   string
}

// scopedWatchBucket applies the active profile's resource prefix to the
// watch bucket argument (FEAT-026). The bucket arg is a bare name here — the
// R2 key prefix is a separate --prefix flag — so it is scoped whole.
func scopedWatchBucket(arg string) string {
	return applyResourcePrefix(arg)
}

func runWatch(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("bucket name is required\n\nUsage: cosmoflare watch <bucket> [directory]")
	}

	bucket := scopedWatchBucket(args[0])
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
	fw, err := newWatchWatcher(absDir)
	if err != nil {
		return err
	}

	// Create R2 client (unless dry-run)
	r2client, err := newWatchR2Client(absDir)
	if err != nil {
		return err
	}

	// Take initial snapshot
	lastSnap, err := fw.Snapshot()
	if err != nil {
		return outErr("failed to scan directory", err)
	}

	watchPrintHeader(absDir, bucket, len(lastSnap))

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

	state := &watchState{ctx: ctx, fw: fw, r2client: r2client, bucket: bucket}

	for {
		select {
		case <-ctx.Done():
			if !JSONOutput {
				printInfo("Stopping watch...")
			}
			return nil
		case <-ticker.C:
			lastSnap = state.tick(lastSnap)
		}
	}
}

// newWatchWatcher creates the polling file watcher for the directory.
func newWatchWatcher(absDir string) (*cosmoflare.FileWatcher, error) {
	fw, err := cosmoflare.NewFileWatcher(absDir, &cosmoflare.WatcherOptions{
		Prefix:   watchPrefix,
		Exclude:  watchExclude,
		Interval: watchInterval,
		Delete:   watchDelete,
	})
	if err != nil {
		return nil, outErr("failed to create watcher", err)
	}
	return fw, nil
}

// newWatchR2Client creates the R2 client (unless dry-run), with project
// guardrails attached so uploads violating allowed_buckets / max_file_size /
// blocked_keys fail.
func newWatchR2Client(absDir string) (cosmoflare.R2Client, error) {
	var r2client cosmoflare.R2Client
	if DryRun {
		return r2client, nil
	}

	var err error
	r2client, err = cosmoflare.NewClient(append([]cosmoflare.ClientOption{
		cosmoflare.WithAccountID(AccountID),
		cosmoflare.WithAPIToken(APIToken),
	}, projectConfigOptions(absDir)...)...)
	if err != nil {
		return nil, outErr("failed to create R2 client", err)
	}
	return r2client, nil
}

// watchPrintHeader prints the human-readable startup banner.
func watchPrintHeader(absDir, bucket string, initialFiles int) {
	if !JSONOutput {
		printInfo("Watching %s → bucket %q (prefix=%q, interval=%s, delete=%v)",
			absDir, bucket, watchPrefix, watchInterval, watchDelete)
		printInfo("Initial scan: %d file(s)", initialFiles)
		if DryRun {
			printWarning("DRY RUN: no files will be uploaded or deleted")
		}
		printInfo("Press Ctrl+C to stop")
	}
}

// tick scans the directory, syncs any detected changes, and returns the
// snapshot to use as the baseline for the next tick.
func (s *watchState) tick(lastSnap cosmoflare.FileSnapshot) cosmoflare.FileSnapshot {
	curSnap, err := s.fw.Snapshot()
	if err != nil {
		emitWatchError("scan error: "+err.Error(), "")
		return lastSnap
	}

	changes := s.fw.Diff(lastSnap, curSnap)
	if len(changes) == 0 {
		return lastSnap
	}

	s.applyChanges(changes)

	return curSnap
}

// applyChanges dispatches each detected change to the matching sync action.
func (s *watchState) applyChanges(changes []cosmoflare.FileChange) {
	for _, change := range changes {
		switch change.Type {
		case cosmoflare.ChangeAdded, cosmoflare.ChangeModified:
			syncUpload(s.ctx, s.r2client, s.bucket, change)
		case cosmoflare.ChangeDeleted:
			if watchDelete {
				syncDelete(s.ctx, s.r2client, s.bucket, change)
			} else if Verbose && !JSONOutput {
				printInfo("Skipped deletion of %s (use --delete to sync)", change.Path)
			}
		}
	}
}

func syncUpload(ctx context.Context, client cosmoflare.R2Client, bucket string, change cosmoflare.FileChange) {
	action := "uploaded"
	symbol := "[+]"
	if change.Type == cosmoflare.ChangeModified {
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

func syncDelete(ctx context.Context, client cosmoflare.R2Client, bucket string, change cosmoflare.FileChange) {
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

func emitWatchEvent(action string, change cosmoflare.FileChange, dryRun bool) {
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
