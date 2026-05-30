package cosmoflare

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Action    string                 `json:"action"`
	Bucket    string                 `json:"bucket,omitempty"`
	Key       string                 `json:"key,omitempty"`
	Size      int64                  `json:"size,omitempty"`
	Result    string                 `json:"result"` // "success" or "failure"
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLogger writes JSONL audit log entries.
type AuditLogger struct {
	mu   sync.Mutex
	file *os.File
	path string
}

// NewAuditLogger creates a new audit logger that appends to the given path.
func NewAuditLogger(path string) (*AuditLogger, error) {
	if path == "" {
		return nil, nil // audit logging disabled
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	return &AuditLogger{file: f, path: path}, nil
}

// Log writes an audit entry to the log file.
func (l *AuditLogger) Log(entry AuditEntry) error {
	if l == nil {
		return nil
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	_, err = l.file.Write(append(data, '\n'))
	return err
}

// LogUpload is a convenience method for logging upload operations.
func (l *AuditLogger) LogUpload(bucket, key string, size int64, result string, err error) {
	entry := AuditEntry{
		Action: "upload",
		Bucket: bucket,
		Key:    key,
		Size:   size,
		Result: result,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	_ = l.Log(entry)
}

// LogDelete is a convenience method for logging delete operations.
func (l *AuditLogger) LogDelete(bucket, key string, result string, err error) {
	entry := AuditEntry{
		Action: "delete",
		Bucket: bucket,
		Key:    key,
		Result: result,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	_ = l.Log(entry)
}

// Close closes the audit log file.
func (l *AuditLogger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

// ReadAuditLog reads all entries from an audit log file.
func ReadAuditLog(path string) ([]AuditEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []AuditEntry
	decoder := json.NewDecoder(f)
	for {
		var entry AuditEntry
		if err := decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return entries, fmt.Errorf("malformed audit entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
