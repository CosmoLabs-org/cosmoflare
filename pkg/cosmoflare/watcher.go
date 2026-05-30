package cosmoflare

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ChangeType describes the kind of filesystem change detected.
type ChangeType int

const (
	// ChangeAdded means a new file appeared.
	ChangeAdded ChangeType = iota
	// ChangeModified means an existing file was modified (size or mtime changed).
	ChangeModified
	// ChangeDeleted means a file was removed.
	ChangeDeleted
)

// String returns a human-readable label for the change type.
func (ct ChangeType) String() string {
	switch ct {
	case ChangeAdded:
		return "added"
	case ChangeModified:
		return "modified"
	case ChangeDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// Symbol returns a short prefix for display: [+] added, [~] modified, [-] deleted.
func (ct ChangeType) Symbol() string {
	switch ct {
	case ChangeAdded:
		return "[+]"
	case ChangeModified:
		return "[~]"
	case ChangeDeleted:
		return "[-]"
	default:
		return "[?]"
	}
}

// FileChange describes a single detected filesystem change.
type FileChange struct {
	Path     string     `json:"path"`      // Relative path from the watch directory
	R2Key    string     `json:"r2_key"`    // Key to use in R2 (prefix + path)
	Type     ChangeType `json:"type"`      // added, modified, deleted
	Size     int64      `json:"size"`      // File size (0 for deletes)
	FullPath string     `json:"full_path"` // Absolute path on disk
}

// FileInfo holds the last-known state of a file for comparison.
type FileInfo struct {
	ModTime time.Time
	Size    int64
}

// FileSnapshot maps relative file paths to their last-known state.
type FileSnapshot map[string]FileInfo

// WatcherOptions configures a FileWatcher.
type WatcherOptions struct {
	Prefix   string        // R2 key prefix (prepended to relative paths)
	Exclude  []string      // Glob patterns to skip (matched against filename)
	Interval time.Duration // Poll interval (default 1s)
	Delete   bool          // Whether to sync deletions
}

// FileWatcher polls a directory tree and detects file changes.
type FileWatcher struct {
	dir      string
	opts     WatcherOptions
	interval time.Duration
}

// NewFileWatcher creates a watcher for the given directory.
// Returns an error if the directory does not exist or is not a directory.
func NewFileWatcher(dir string, opts *WatcherOptions) (*FileWatcher, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, validationError("NewFileWatcher", fmt.Sprintf("directory %q does not exist: %v", dir, err))
	}
	if !info.IsDir() {
		return nil, validationError("NewFileWatcher", fmt.Sprintf("%q is not a directory", dir))
	}

	w := &FileWatcher{
		dir: dir,
	}

	if opts != nil {
		w.opts = *opts
	}

	w.interval = w.opts.Interval
	if w.interval == 0 {
		w.interval = time.Second
	}

	return w, nil
}

// Dir returns the watched directory path.
func (w *FileWatcher) Dir() string { return w.dir }

// Interval returns the configured poll interval.
func (w *FileWatcher) Interval() time.Duration { return w.interval }

// Snapshot walks the directory tree and returns a map of relative paths
// to file state (mod time + size). Hidden directories (starting with '.')
// are skipped. Files matching exclude patterns are skipped.
func (w *FileWatcher) Snapshot() (FileSnapshot, error) {
	snap := make(FileSnapshot)

	err := filepath.Walk(w.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip files/dirs we can't read
		}

		// Skip hidden directories (e.g. .git, .DS_Store dirs)
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && path != w.dir {
			return filepath.SkipDir
		}

		// Skip directories themselves (we only track files)
		if info.IsDir() {
			return nil
		}

		// Get relative path
		rel, err := filepath.Rel(w.dir, path)
		if err != nil {
			return nil
		}

		// Check exclude patterns against the filename
		if w.shouldExclude(rel) {
			return nil
		}

		snap[rel] = FileInfo{
			ModTime: info.ModTime(),
			Size:    info.Size(),
		}
		return nil
	})

	return snap, err
}

// shouldExclude checks if a relative path matches any exclude pattern.
func (w *FileWatcher) shouldExclude(rel string) bool {
	name := filepath.Base(rel)
	for _, pattern := range w.opts.Exclude {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
		// Also try matching against the full relative path
		if matched, _ := filepath.Match(pattern, rel); matched {
			return true
		}
	}
	return false
}

// Diff compares an old snapshot to a current snapshot and returns the
// list of changes (added, modified, deleted).
func (w *FileWatcher) Diff(old, cur FileSnapshot) []FileChange {
	var changes []FileChange

	// Detect added and modified files
	for path, curInfo := range cur {
		oldInfo, exists := old[path]
		if !exists {
			changes = append(changes, FileChange{
				Path:     path,
				R2Key:    w.r2Key(path),
				Type:     ChangeAdded,
				Size:     curInfo.Size,
				FullPath: filepath.Join(w.dir, path),
			})
		} else if curInfo.Size != oldInfo.Size || !curInfo.ModTime.Equal(oldInfo.ModTime) {
			changes = append(changes, FileChange{
				Path:     path,
				R2Key:    w.r2Key(path),
				Type:     ChangeModified,
				Size:     curInfo.Size,
				FullPath: filepath.Join(w.dir, path),
			})
		}
	}

	// Detect deleted files
	for path := range old {
		if _, exists := cur[path]; !exists {
			changes = append(changes, FileChange{
				Path:     path,
				R2Key:    w.r2Key(path),
				Type:     ChangeDeleted,
				Size:     0,
				FullPath: filepath.Join(w.dir, path),
			})
		}
	}

	// Sort for deterministic output
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Path < changes[j].Path
	})

	return changes
}

// r2Key builds the R2 object key from a relative file path.
// Uses forward slashes for R2 compatibility.
func (w *FileWatcher) r2Key(rel string) string {
	// Normalize to forward slashes for R2/S3 key
	key := filepath.ToSlash(rel)
	if w.opts.Prefix != "" {
		key = w.opts.Prefix + key
	}
	return key
}
