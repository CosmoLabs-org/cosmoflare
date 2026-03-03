/*
Package api provides enhanced R2 API client with progress tracking

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package api

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/visual"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// UploadProgressCallback is called during upload to report progress
type UploadProgressCallback func(uploadedBytes int64, totalBytes int64, speed float64)

// UploadResult contains the result of an upload operation
type UploadResult struct {
	Key         string            `json:"key"`
	Bucket      string            `json:"bucket"`
	Size        int64             `json:"size"`
	ETag        string            `json:"etag"`
	UploadID    string            `json:"upload_id"`
	Duration    time.Duration     `json:"duration"`
	Speed       float64           `json:"speed_mbps"`
	Parts       []CompletedPart   `json:"parts,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	URL         string            `json:"url,omitempty"`
}

// CompletedPart represents a completed multipart upload part
type CompletedPart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
	Size       int64  `json:"size"`
}

// UploadOptions contains options for enhanced uploads
type UploadOptions struct {
	ContentType   string            `json:"content_type,omitempty"`
	CacheControl  string            `json:"cache_control,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	ChunkSize     int64             `json:"chunk_size,omitempty"`
	ShowProgress  bool              `json:"show_progress,omitempty"`
	Quiet         bool              `json:"quiet,omitempty"`
	Retries       int               `json:"retries,omitempty"`
	Timeout       time.Duration     `json:"timeout,omitempty"`
	ProgressFunc  UploadProgressCallback `json:"-"`
}

// S3API defines the S3 operations used by EnhancedClient
type S3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	CreateMultipartUpload(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error)
	UploadPart(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error)
	CompleteMultipartUpload(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error)
	AbortMultipartUpload(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error)
}

// EnhancedClient extends the basic client with enhanced upload capabilities
type EnhancedClient struct {
	*Client
	s3Client S3API
}

// NewEnhancedClient creates a new enhanced R2 client
func NewEnhancedClient(baseClient *Client) (*EnhancedClient, error) {
	if baseClient == nil {
		return nil, fmt.Errorf("base client cannot be nil")
	}

	// Create S3 client from base client's S3 client
	// Note: In production, this would create a new S3 client with additional configuration
	return &EnhancedClient{
		Client:   baseClient,
		s3Client: baseClient.s3,
	}, nil
}

// NewEnhancedClientWithS3 creates a new enhanced R2 client with an explicit S3 API
func NewEnhancedClientWithS3(baseClient *Client, s3api S3API) (*EnhancedClient, error) {
	if baseClient == nil {
		return nil, fmt.Errorf("base client cannot be nil")
	}

	return &EnhancedClient{
		Client:   baseClient,
		s3Client: s3api,
	}, nil
}

// UploadFile uploads a file with real-time progress tracking
func (ec *EnhancedClient) UploadFile(ctx context.Context, bucket, key, filePath string, opts *UploadOptions) (*UploadResult, error) {
	if opts == nil {
		opts = &UploadOptions{
			ShowProgress: true,
			ChunkSize:    8 * 1024 * 1024, // 8MB default
			Retries:      3,
			Timeout:      30 * time.Minute,
		}
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	fileSize := fileInfo.Size()

	// Determine if multipart upload is needed
	useMultipart := fileSize > opts.ChunkSize

	if !opts.Quiet {
		visual.ShowSpinner(fmt.Sprintf("Analyzing %s for upload...", key), 2*time.Second)
	}

	var result *UploadResult
	startTime := time.Now()

	if useMultipart {
		result, err = ec.multipartUpload(ctx, bucket, key, filePath, fileSize, opts)
	} else {
		result, err = ec.singlePartUpload(ctx, bucket, key, filePath, fileSize, opts)
	}

	if result != nil {
		result.Duration = time.Since(startTime)
	}

	return result, err
}

// singlePartUpload handles small files with simple upload
func (ec *EnhancedClient) singlePartUpload(ctx context.Context, bucket, key, filePath string, fileSize int64, opts *UploadOptions) (*UploadResult, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create progress reader
	progressReader := visual.NewR2ProgressReader(file, nil, nil) // We'll enhance this later

	// Prepare upload input
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   progressReader,
	}

	// Set content type if not provided
	if opts.ContentType == "" {
		input.ContentType = aws.String("application/octet-stream")
	} else {
		input.ContentType = aws.String(opts.ContentType)
	}

	// Set cache control if provided
	if opts.CacheControl != "" {
		input.CacheControl = aws.String(opts.CacheControl)
	}

	// Set metadata if provided
	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}

	if !opts.Quiet && opts.ShowProgress {
		visual.ShowProgress(0, fileSize, fmt.Sprintf("Uploading %s", key))
	}

	// Perform upload
	resp, err := ec.s3Client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object: %w", err)
	}

	if !opts.Quiet && opts.ShowProgress {
		visual.ShowProgress(fileSize, fileSize, fmt.Sprintf("Completed %s", key))
	}

	// Calculate speed
	speed := float64(fileSize) / time.Since(time.Now()).Seconds() / (1024 * 1024)

	return &UploadResult{
		Key:      key,
		Bucket:   bucket,
		Size:     fileSize,
		ETag:     aws.ToString(resp.ETag),
		Speed:    speed,
		Metadata: opts.Metadata,
	}, nil
}

// multipartUpload handles large files with multipart upload
func (ec *EnhancedClient) multipartUpload(ctx context.Context, bucket, key, filePath string, fileSize int64, opts *UploadOptions) (*UploadResult, error) {
	// Create multipart upload
	createResp, err := ec.s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart upload: %w", err)
	}

	uploadID := aws.ToString(createResp.UploadId)

	if !opts.Quiet && opts.ShowProgress {
		fmt.Printf("📦 Starting multipart upload: %s\n", key)
		fmt.Printf("   File size: %s | Parts: %d\n", utils.FormatBytes(fileSize), int(fileSize/opts.ChunkSize)+1)
	}

	// Calculate number of parts
	partSize := opts.ChunkSize
	numParts := int(fileSize/partSize) + 1
	completedParts := make([]CompletedPart, 0, numParts)

	// Upload parts concurrently or sequentially
	for partNumber := 1; partNumber <= numParts; partNumber++ {
		partResult, err := ec.uploadPart(ctx, bucket, key, uploadID, partNumber, filePath, partSize, opts)
		if err != nil {
			// Abort multipart upload on failure
			ec.abortMultipartUpload(ctx, bucket, key, uploadID)
			return nil, fmt.Errorf("failed to upload part %d: %w", partNumber, err)
		}

		completedParts = append(completedParts, *partResult)

		if !opts.Quiet && opts.ShowProgress {
			uploadedBytes := int64(partNumber) * partSize
			if uploadedBytes > fileSize {
				uploadedBytes = fileSize
			}
			visual.ShowProgress(uploadedBytes, fileSize, fmt.Sprintf("Part %d/%d", partNumber, numParts))
		}
	}

	// Complete multipart upload
	completedUploadParts := make([]types.CompletedPart, len(completedParts))
	for i, part := range completedParts {
		partNum := int32(part.PartNumber)
		completedUploadParts[i] = types.CompletedPart{
			PartNumber: &partNum,
			ETag:       aws.String(part.ETag),
		}
	}

	completeResp, err := ec.s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedUploadParts,
		},
	})
	if err != nil {
		ec.abortMultipartUpload(ctx, bucket, key, uploadID)
		return nil, fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	// Calculate average speed
	speed := float64(fileSize) / time.Since(time.Now()).Seconds() / (1024 * 1024)

	if !opts.Quiet && opts.ShowProgress {
		visual.ShowSuccess(fmt.Sprintf("Multipart upload completed: %s", key))
	}

	return &UploadResult{
		Key:      key,
		Bucket:   bucket,
		Size:     fileSize,
		ETag:     aws.ToString(completeResp.ETag),
		UploadID: uploadID,
		Speed:    speed,
		Parts:    completedParts,
		Metadata: opts.Metadata,
	}, nil
}

// uploadPart uploads a single part of a multipart upload
func (ec *EnhancedClient) uploadPart(ctx context.Context, bucket, key, uploadID string, partNumber int, filePath string, partSize int64, opts *UploadOptions) (*CompletedPart, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Seek to part position
	offset := int64(partNumber-1) * partSize
	_, err = file.Seek(offset, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to part position: %w", err)
	}

	// Create limited reader for this part
	var reader io.Reader = file
	if offset+partSize > getFileSize(filePath) {
		partSize = getFileSize(filePath) - offset
	}
	reader = io.LimitReader(file, partSize)

	// Upload part
	partNum := int32(partNumber)
	resp, err := ec.s3Client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: &partNum,
		Body:       reader,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to upload part: %w", err)
	}

	return &CompletedPart{
		PartNumber: partNumber,
		ETag:       aws.ToString(resp.ETag),
		Size:       partSize,
	}, nil
}

// abortMultipartUpload aborts a multipart upload
func (ec *EnhancedClient) abortMultipartUpload(ctx context.Context, bucket, key, uploadID string) {
	_, err := ec.s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		visual.ShowError(fmt.Sprintf("Failed to abort multipart upload: %v", err))
	}
}

// getFileSize returns the size of a file
func getFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// UploadWithRealTimeProgress uploads a file with rich visual feedback
func (ec *EnhancedClient) UploadWithRealTimeProgress(ctx context.Context, bucket, key, filePath string, opts *UploadOptions) error {
	// Show startup animation
	visual.ShowStartupAnimation()

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Create visual progress tracker
	rp := visual.NewR2UploadProgress()
	upload := rp.AddUpload(filepath.Base(filePath), bucket, key, fileInfo.Size())

	// Add visual progress tracking to options
	originalProgressFunc := opts.ProgressFunc
	opts.ProgressFunc = func(uploadedBytes, totalBytes int64, speed float64) {
		rp.UpdateProgress(upload, uploadedBytes, speed)

		// Call original progress function if provided
		if originalProgressFunc != nil {
			originalProgressFunc(uploadedBytes, totalBytes, speed)
		}
	}

	// Start upload
	rp.StartUpload(upload)

	// Set up real-time display
	go func() {
		rp.ShowSimpleProgress()
	}()

	// Perform upload
	result, err := ec.UploadFile(ctx, bucket, key, filePath, opts)

	// Handle result
	if err != nil {
		rp.FailUpload(upload, err)
		visual.ShowError(fmt.Sprintf("Upload failed: %v", err))
		return err
	}

	rp.CompleteUpload(upload)

	// Show beautiful result display
	resultDetails := map[string]interface{}{
		"File":       key,
		"Bucket":     bucket,
		"Size":       utils.FormatBytes(result.Size),
		"Duration":   result.Duration.String(),
		"Speed":      fmt.Sprintf("%.2f MB/s", result.Speed),
		"Upload ID":  result.UploadID,
		"ETag":       result.ETag,
	}

	if result.Parts != nil && len(result.Parts) > 0 {
		resultDetails["Parts"] = fmt.Sprintf("%d completed", len(result.Parts))
	}

	visual.ShowResult("🚀 Upload Complete!", resultDetails)
	visual.ShowSuccess(fmt.Sprintf("Successfully uploaded %s to R2!", key))

	return nil
}

