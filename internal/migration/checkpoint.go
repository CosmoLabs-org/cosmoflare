package migration

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Checkpoint struct {
	Version         int              `json:"version"`
	S3Bucket        string           `json:"s3_bucket"`
	R2Bucket        string           `json:"r2_bucket"`
	Filter          string           `json:"filter"`
	StartedAt       time.Time        `json:"started_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	TotalObjects    int64            `json:"total_objects"`
	TotalSize       int64            `json:"total_size"`
	Completed       map[string]int64 `json:"completed"`
	Failed          []string         `json:"failed"`
	TransferredSize int64            `json:"transferred_size"`

	mu   sync.Mutex `json:"-"`
	path string     `json:"-"`
}

const (
	flushInterval = 30 * time.Second
	flushCount    = 10
)

func checkpointHash(s3Bucket, r2Bucket, filter string) string {
	h := sha256.Sum256([]byte(s3Bucket + ":" + r2Bucket + ":" + filter))
	return fmt.Sprintf("%x", h[:8])
}

func checkpointDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cosmoflare", "migrations")
}

func checkpointPathIn(dir, s3Bucket, r2Bucket, filter string) string {
	return filepath.Join(dir, checkpointHash(s3Bucket, r2Bucket, filter)+".json")
}

func (m *S3Migration) checkpointPath() string {
	return checkpointPathIn(checkpointDir(), m.S3Bucket, m.R2Bucket, m.Filter)
}

func loadCheckpoint(path string) (*Checkpoint, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint: %w", err)
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("corrupt checkpoint: %w", err)
	}
	cp.path = path
	return &cp, nil
}

func saveCheckpoint(cp *Checkpoint, path string) error {
	cp.UpdatedAt = time.Now()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint dir: %w", err)
	}
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize checkpoint: %w", err)
	}
	return nil
}

func deleteCheckpoint(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (cp *Checkpoint) markCompleted(key string, size int64) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.Completed[key] = size
	cp.TransferredSize += size
}

func (cp *Checkpoint) markFailed(key string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.Failed = append(cp.Failed, key)
}

func (cp *Checkpoint) isCompleted(key string) bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	_, ok := cp.Completed[key]
	return ok
}

func (cp *Checkpoint) flush() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return saveCheckpoint(cp, cp.path)
}
