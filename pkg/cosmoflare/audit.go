package cosmoflare

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AuditEntry represents a single audit log entry for a CLI mutation.
type AuditEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Operation string            `json:"operation"`
	Service   string            `json:"service"`
	Resource  string            `json:"resource"`
	Action    string            `json:"action"`
	User      string            `json:"user"`
	Details   map[string]string `json:"details,omitempty"`
	Success   bool              `json:"success"`
}

// AuditLogger writes newline-delimited JSON audit log entries to a file.
type AuditLogger struct {
	mu   sync.Mutex
	file *os.File
	path string
}

// DefaultAuditLogPath returns the default audit log path (~/.cosmoflare/audit.log).
func DefaultAuditLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cosmoflare", "audit.log")
}

// NewAuditLogger creates a new audit logger that appends to the given path.
// If path is empty, returns nil (audit logging disabled).
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
	if entry.User == "" {
		if u, err := user.Current(); err == nil {
			entry.User = u.Username
		}
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

// LogMutation is a convenience method for logging any CLI mutation.
func (l *AuditLogger) LogMutation(op, service, resource, action string, details map[string]string, success bool) {
	entry := AuditEntry{
		Operation: op,
		Service:   service,
		Resource:  resource,
		Action:    action,
		Details:   details,
		Success:   success,
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

// Path returns the file path of the audit log.
func (l *AuditLogger) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// ReadAuditLog reads all entries from an audit log file.
func ReadAuditLog(path string) ([]AuditEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
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

// ReadAuditLogWithLimit reads up to limit entries from the end of the audit log.
// If limit <= 0, returns all entries.
func ReadAuditLogWithLimit(path string, limit int) ([]AuditEntry, error) {
	entries, err := ReadAuditLog(path)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || len(entries) <= limit {
		return entries, nil
	}
	return entries[len(entries)-limit:], nil
}

// ReadAuditLogSince reads entries since the given time.
func ReadAuditLogSince(path string, since time.Time) ([]AuditEntry, error) {
	entries, err := ReadAuditLog(path)
	if err != nil {
		return nil, err
	}
	var filtered []AuditEntry
	for _, e := range entries {
		if !e.Timestamp.Before(since) {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}

// SearchAuditLog searches entries matching a query string across all fields.
// Optionally filter by action type (create, delete, update).
func SearchAuditLog(path string, query string, actionFilter string) ([]AuditEntry, error) {
	entries, err := ReadAuditLog(path)
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var results []AuditEntry
	for _, e := range entries {
		if actionFilter != "" && !strings.EqualFold(e.Action, actionFilter) {
			continue
		}
		if matchesQuery(e, queryLower) {
			results = append(results, e)
		}
	}
	return results, nil
}

// matchesQuery returns true if any field in the entry contains the query string.
func matchesQuery(e AuditEntry, queryLower string) bool {
	fields := []string{
		e.Operation,
		e.Service,
		e.Resource,
		e.Action,
		e.User,
		e.Timestamp.Format(time.RFC3339),
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), queryLower) {
			return true
		}
	}
	for k, v := range e.Details {
		if strings.Contains(strings.ToLower(k), queryLower) || strings.Contains(strings.ToLower(v), queryLower) {
			return true
		}
	}
	return false
}

// ExportAuditLogJSON exports audit entries to a JSON file (array format).
func ExportAuditLogJSON(entries []AuditEntry, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

// ExportAuditLogCSV exports audit entries to CSV format.
func ExportAuditLogCSV(entries []AuditEntry, w io.Writer) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header
	if err := cw.Write([]string{"timestamp", "operation", "service", "resource", "action", "user", "success", "details"}); err != nil {
		return err
	}

	for _, e := range entries {
		detailParts := make([]string, 0, len(e.Details))
		for k, v := range e.Details {
			detailParts = append(detailParts, k+"="+v)
		}
		successStr := "true"
		if !e.Success {
			successStr = "false"
		}
		row := []string{
			e.Timestamp.Format(time.RFC3339),
			e.Operation,
			e.Service,
			e.Resource,
			e.Action,
			e.User,
			successStr,
			strings.Join(detailParts, "; "),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// ClearAuditLog removes all entries from the audit log file.
// If before is non-zero, only entries before that time are removed.
func ClearAuditLog(path string, before time.Time) (int, error) {
	if before.IsZero() {
		// Truncate entire file
		entries, err := ReadAuditLog(path)
		if err != nil {
			return 0, err
		}
		count := len(entries)
		if err := os.Truncate(path, 0); err != nil {
			return 0, fmt.Errorf("failed to truncate audit log: %w", err)
		}
		return count, nil
	}

	// Partial clear: keep entries >= before
	entries, err := ReadAuditLog(path)
	if err != nil {
		return 0, err
	}

	var kept []AuditEntry
	removed := 0
	for _, e := range entries {
		if e.Timestamp.Before(before) {
			removed++
		} else {
			kept = append(kept, e)
		}
	}

	if removed == 0 {
		return 0, nil
	}

	// Rewrite file with kept entries
	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("failed to rewrite audit log: %w", err)
	}
	defer f.Close()

	for _, e := range kept {
		data, err := json.Marshal(e)
		if err != nil {
			return removed, fmt.Errorf("failed to marshal audit entry: %w", err)
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return removed, err
		}
	}

	return removed, nil
}
