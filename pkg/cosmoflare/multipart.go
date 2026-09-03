package cosmoflare

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	// MinPartSize is the minimum part size allowed by the S3 API (5MB).
	MinPartSize = 5 * 1024 * 1024

	// MaxPartSize is the maximum part size allowed (5GB).
	MaxPartSize = 5 * 1024 * 1024 * 1024

	// stateFilePrefix is the prefix for upload state files.
	stateFilePrefix = ".cosmoflare-upload-"
)

// CompletedPartInfo tracks a successfully uploaded part.
type CompletedPartInfo struct {
	PartNumber int32  `json:"part_number"`
	ETag       string `json:"etag"`
	Size       int64  `json:"size"`
}

// MultipartUploadState tracks the state of a resumable multipart upload.
// This is persisted to disk so interrupted uploads can be resumed.
type MultipartUploadState struct {
	UploadID       string              `json:"upload_id"`
	Bucket         string              `json:"bucket"`
	Key            string              `json:"key"`
	TotalSize      int64               `json:"total_size"`
	PartSize       int64               `json:"part_size"`
	TotalParts     int64               `json:"total_parts"`
	CompletedParts []CompletedPartInfo `json:"completed_parts"`
	StartedAt      time.Time           `json:"started_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	ContentType    string              `json:"content_type,omitempty"`
	CacheControl   string              `json:"cache_control,omitempty"`
	Metadata       map[string]string   `json:"metadata,omitempty"`
	Checksum       string              `json:"checksum"`
}

// stateFilePath returns the path to the upload state file for a given bucket/key.
func stateFilePath(bucket, key string) string {
	// Hash the key to create a safe filename
	h := sha256.Sum256([]byte(bucket + "/" + key))
	hash := hex.EncodeToString(h[:8])
	return filepath.Join(os.TempDir(), stateFilePrefix+hash+".json")
}

// StateFilePath returns the path to the upload state file for a given bucket/key.
// Exported for CLI usage.
func StateFilePath(bucket, key string) string {
	return stateFilePath(bucket, key)
}

// SaveUploadState persists the upload state to disk for resume capability.
func SaveUploadState(state *MultipartUploadState) error {
	if state == nil {
		return validationError("SaveUploadState", "state is nil")
	}
	if state.UploadID == "" {
		return validationError("SaveUploadState", "upload ID is required")
	}
	if state.Bucket == "" {
		return validationError("SaveUploadState", "bucket is required")
	}
	if state.Key == "" {
		return validationError("SaveUploadState", "key is required")
	}

	state.UpdatedAt = time.Now().UTC()
	state.Checksum = computeStateChecksum(state)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return newError("SaveUploadState", "failed to marshal state", err)
	}

	path := stateFilePath(state.Bucket, state.Key)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return newError("SaveUploadState", "failed to write state file", err)
	}

	return nil
}

// LoadUploadState loads the upload state from disk.
// Returns nil, nil if no state file exists.
func LoadUploadState(bucket, key string) (*MultipartUploadState, error) {
	if bucket == "" {
		return nil, validationError("LoadUploadState", "bucket is required")
	}
	if key == "" {
		return nil, validationError("LoadUploadState", "key is required")
	}

	path := stateFilePath(bucket, key)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, newError("LoadUploadState", "failed to read state file", err)
	}

	var state MultipartUploadState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, newError("LoadUploadState", "failed to parse state file (corrupted?)", err)
	}

	// Verify checksum integrity
	savedChecksum := state.Checksum
	state.Checksum = ""
	expected := computeStateChecksum(&state)
	if savedChecksum != expected {
		return nil, newError("LoadUploadState", "state file checksum mismatch (corrupted?)", nil)
	}
	state.Checksum = savedChecksum

	return &state, nil
}

// RemoveUploadState deletes the upload state file.
func RemoveUploadState(bucket, key string) error {
	path := stateFilePath(bucket, key)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return newError("RemoveUploadState", "failed to remove state file", err)
	}
	return nil
}

// computeStateChecksum computes a checksum of the state for integrity verification.
func computeStateChecksum(state *MultipartUploadState) string {
	// Hash the essential fields to detect corruption
	data := fmt.Sprintf("%s:%s:%s:%d:%d:%d",
		state.UploadID, state.Bucket, state.Key,
		state.TotalSize, state.PartSize, len(state.CompletedParts))
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// IsPartCompleted checks if a specific part number has already been uploaded.
func (s *MultipartUploadState) IsPartCompleted(partNumber int32) bool {
	for _, p := range s.CompletedParts {
		if p.PartNumber == partNumber {
			return true
		}
	}
	return false
}

// CompletedBytes returns the total bytes already uploaded.
func (s *MultipartUploadState) CompletedBytes() int64 {
	var total int64
	for _, p := range s.CompletedParts {
		total += p.Size
	}
	return total
}

// RemainingParts returns the part numbers that still need to be uploaded.
func (s *MultipartUploadState) RemainingParts() []int32 {
	completed := make(map[int32]bool, len(s.CompletedParts))
	for _, p := range s.CompletedParts {
		completed[p.PartNumber] = true
	}

	var remaining []int32
	for i := int32(1); i <= int32(s.TotalParts); i++ {
		if !completed[i] {
			remaining = append(remaining, i)
		}
	}
	return remaining
}

// ShouldUseMultipart returns true if the file size exceeds the multipart threshold.
func ShouldUseMultipart(size int64, threshold int64) bool {
	if threshold <= 0 {
		threshold = multipartThreshold
	}
	return size > threshold
}

// ValidatePartSize checks that the part size meets S3 API requirements.
func ValidatePartSize(partSize int64) error {
	if partSize < MinPartSize {
		return fmt.Errorf("part size %d bytes is below minimum of %d bytes (5MB)", partSize, MinPartSize)
	}
	if partSize > MaxPartSize {
		return fmt.Errorf("part size %d bytes exceeds maximum of %d bytes (5GB)", partSize, MaxPartSize)
	}
	return nil
}

// ResumeMultipartUpload resumes an interrupted multipart upload from saved state.
// The reader must be seekable (e.g., *os.File) to skip already-uploaded parts.
func (c *client) ResumeMultipartUpload(ctx context.Context, bucket, key string, reader io.ReadSeeker, size int64, opts ...UploadOption) (*UploadResult, error) {
	if err := validateBucketName(bucket); err != nil {
		return nil, validationError("ResumeMultipartUpload", err.Error())
	}
	if key == "" {
		return nil, validationError("ResumeMultipartUpload", "object key is required")
	}
	if reader == nil {
		return nil, validationError("ResumeMultipartUpload", "reader is required")
	}
	if err := c.enforceUploadGuardrails("ResumeMultipartUpload", bucket, key, size); err != nil {
		return nil, err
	}

	// Load saved state
	state, err := LoadUploadState(bucket, key)
	if err != nil {
		return nil, newError("ResumeMultipartUpload", "failed to load upload state", err)
	}
	if state == nil {
		return nil, newError("ResumeMultipartUpload", "no upload state found for this object; start a new upload instead", nil)
	}

	// Validate state consistency
	if state.TotalSize != size {
		return nil, newError("ResumeMultipartUpload", fmt.Sprintf(
			"file size mismatch: state has %d bytes but file is %d bytes", state.TotalSize, size), nil)
	}

	cfg := &uploadConfig{
		partSize:    state.PartSize,
		concurrency: defaultConcurrency,
	}
	for _, o := range opts {
		o(cfg)
	}

	// Get remaining parts to upload
	remaining := state.RemainingParts()
	if len(remaining) == 0 {
		// All parts uploaded, just complete
		return c.completeMultipartFromState(ctx, state)
	}

	// Upload remaining parts concurrently
	type partResult struct {
		info CompletedPartInfo
		err  error
	}

	var mu sync.Mutex
	sem := make(chan struct{}, cfg.concurrency)
	results := make(chan partResult, len(remaining))

	uploadedBytes := state.CompletedBytes()

	for _, partNum := range remaining {
		offset := int64(partNum-1) * state.PartSize

		// Read this part's data (BUG-026: a reader shorter than the declared
		// size fails the resume instead of storing a truncated object; the
		// saved state is kept so a corrected retry can continue)
		if _, err := reader.Seek(offset, io.SeekStart); err != nil {
			return nil, newError("ResumeMultipartUpload", fmt.Sprintf("failed to seek to part %d offset", partNum), err)
		}
		buf, err := readUploadPart("ResumeMultipartUpload", reader, state.PartSize, size, offset, int64(partNum))
		if err != nil {
			return nil, err
		}
		pn := partNum

		sem <- struct{}{}
		go func(partNumber int32, data []byte) {
			defer func() { <-sem }()

			resp, err := c.s3Client().UploadPart(ctx, &s3.UploadPartInput{
				Bucket:     aws.String(bucket),
				Key:        aws.String(key),
				UploadId:   aws.String(state.UploadID),
				PartNumber: aws.Int32(partNumber),
				Body:       bytes.NewReader(data),
			})
			if err != nil {
				results <- partResult{err: fmt.Errorf("part %d: %w", partNumber, err)}
				return
			}

			info := CompletedPartInfo{
				PartNumber: partNumber,
				ETag:       aws.ToString(resp.ETag),
				Size:       int64(len(data)),
			}

			// Save progress after each part
			mu.Lock()
			state.CompletedParts = append(state.CompletedParts, info)
			_ = SaveUploadState(state)
			mu.Unlock()

			results <- partResult{info: info}
		}(pn, buf)

		uploadedBytes += int64(len(buf))
		if cfg.progressCallback != nil {
			cfg.progressCallback(uploadedBytes, size)
		}
	}

	// Collect results
	for range remaining {
		r := <-results
		if r.err != nil {
			// Save current progress before returning error
			_ = SaveUploadState(state)
			return nil, newError("ResumeMultipartUpload", "part upload failed", r.err)
		}
	}

	// Complete the upload
	result, err := c.completeMultipartFromState(ctx, state)
	if err != nil {
		return nil, err
	}

	// Clean up state file on success
	_ = RemoveUploadState(bucket, key)

	return result, nil
}

// completeMultipartFromState completes a multipart upload using the parts tracked in state.
func (c *client) completeMultipartFromState(ctx context.Context, state *MultipartUploadState) (*UploadResult, error) {
	// Build completed parts list
	var completedParts []types.CompletedPart
	for _, p := range state.CompletedParts {
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       aws.String(p.ETag),
			PartNumber: aws.Int32(p.PartNumber),
		})
	}

	// Sort by part number
	sort.Slice(completedParts, func(i, j int) bool {
		return *completedParts[i].PartNumber < *completedParts[j].PartNumber
	})

	completeResp, err := c.s3Client().CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(state.Bucket),
		Key:      aws.String(state.Key),
		UploadId: aws.String(state.UploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		return nil, newError("CompleteMultipartUpload", "failed to complete multipart upload", err)
	}

	return &UploadResult{
		Key:      state.Key,
		Bucket:   state.Bucket,
		Size:     state.TotalSize,
		ETag:     aws.ToString(completeResp.ETag),
		Uploaded: time.Now().UTC(),
		Parts:    int(state.TotalParts),
	}, nil
}

// ResumableMultipartUpload performs a multipart upload with state tracking for resume capability.
// Unlike the base MultipartUpload, this persists progress after each part so interrupted
// uploads can be resumed with ResumeMultipartUpload.
func (c *client) ResumableMultipartUpload(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts ...UploadOption) (*UploadResult, error) {
	if err := validateBucketName(bucket); err != nil {
		return nil, validationError("ResumableMultipartUpload", err.Error())
	}
	if key == "" {
		return nil, validationError("ResumableMultipartUpload", "object key is required")
	}
	if reader == nil {
		return nil, validationError("ResumableMultipartUpload", "reader is required")
	}
	if err := c.enforceUploadGuardrails("ResumableMultipartUpload", bucket, key, size); err != nil {
		return nil, err
	}

	cfg := &uploadConfig{
		partSize:    defaultPartSize,
		concurrency: defaultConcurrency,
	}
	for _, o := range opts {
		o(cfg)
	}

	// Validate part size
	if err := ValidatePartSize(cfg.partSize); err != nil {
		return nil, validationError("ResumableMultipartUpload", err.Error())
	}

	// Apply cache policy
	if c.cfg.cacheControl && cfg.cacheControl == "" {
		cfg.cacheControl = defaultCachePolicy(key)
	}

	// Initiate multipart upload
	createInput := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if cfg.contentType != "" {
		createInput.ContentType = aws.String(cfg.contentType)
	}
	if cfg.cacheControl != "" {
		createInput.CacheControl = aws.String(cfg.cacheControl)
	}
	if len(cfg.metadata) > 0 {
		createInput.Metadata = cfg.metadata
	}

	createResp, err := c.s3Client().CreateMultipartUpload(ctx, createInput)
	if err != nil {
		return nil, newError("ResumableMultipartUpload", "failed to initiate multipart upload", err)
	}
	uploadID := aws.ToString(createResp.UploadId)

	// Calculate parts
	partSize := cfg.partSize
	numParts := (size + partSize - 1) / partSize
	if numParts == 0 {
		numParts = 1
	}

	// Create initial state
	state := &MultipartUploadState{
		UploadID:     uploadID,
		Bucket:       bucket,
		Key:          key,
		TotalSize:    size,
		PartSize:     partSize,
		TotalParts:   numParts,
		StartedAt:    time.Now().UTC(),
		ContentType:  cfg.contentType,
		CacheControl: cfg.cacheControl,
		Metadata:     cfg.metadata,
	}

	// Save initial state
	if err := SaveUploadState(state); err != nil {
		// Non-fatal: upload can proceed without resume capability
		_ = err
	}

	// Abort helper
	abort := func() {
		c.s3Client().AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(bucket),
			Key:      aws.String(key),
			UploadId: aws.String(uploadID),
		})
		_ = RemoveUploadState(bucket, key)
	}

	// Upload parts with state tracking
	type partResult struct {
		info CompletedPartInfo
		err  error
	}

	var mu sync.Mutex
	sem := make(chan struct{}, cfg.concurrency)
	results := make(chan partResult, numParts)
	var uploadedBytes int64

	for partNum := int64(1); partNum <= numParts; partNum++ {
		// Read this part's data (BUG-026: a reader shorter than the declared
		// size aborts the upload instead of storing a truncated object)
		buf, err := readUploadPart("ResumableMultipartUpload", reader, partSize, size, uploadedBytes, partNum)
		if err != nil {
			abort()
			return nil, err
		}
		pn := int32(partNum)

		sem <- struct{}{}
		go func(partNumber int32, data []byte) {
			defer func() { <-sem }()

			resp, err := c.s3Client().UploadPart(ctx, &s3.UploadPartInput{
				Bucket:     aws.String(bucket),
				Key:        aws.String(key),
				UploadId:   aws.String(uploadID),
				PartNumber: aws.Int32(partNumber),
				Body:       bytes.NewReader(data),
			})
			if err != nil {
				results <- partResult{err: fmt.Errorf("part %d: %w", partNumber, err)}
				return
			}

			info := CompletedPartInfo{
				PartNumber: partNumber,
				ETag:       aws.ToString(resp.ETag),
				Size:       int64(len(data)),
			}

			// Save progress after each part
			mu.Lock()
			state.CompletedParts = append(state.CompletedParts, info)
			_ = SaveUploadState(state)
			mu.Unlock()

			results <- partResult{info: info}
		}(pn, buf)

		uploadedBytes += int64(len(buf))
		if cfg.progressCallback != nil {
			cfg.progressCallback(uploadedBytes, size)
		}
	}

	// Collect results
	for i := int64(0); i < numParts; i++ {
		r := <-results
		if r.err != nil {
			// Save progress so upload can be resumed
			_ = SaveUploadState(state)
			return nil, newError("ResumableMultipartUpload", "part upload failed (state saved for resume)", r.err)
		}
	}

	// Complete the upload
	result, err := c.completeMultipartFromState(ctx, state)
	if err != nil {
		abort()
		return nil, err
	}

	// Clean up state file on success
	_ = RemoveUploadState(bucket, key)

	return result, nil
}
