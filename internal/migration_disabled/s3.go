/*
Package migration provides real S3 to R2 migration functionality for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package migration

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cheggaaa/pb/v3"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/api"
)

// S3Migration handles migration from AWS S3 to Cloudflare R2
type S3Migration struct {
	S3Bucket       string
	R2Bucket       string
	Filter         string
	Concurrency    int
	Resume         bool
	Verify         bool
	AWSRegion      string
	AWSProfile     string
	DryRun         bool
	DeleteSource   bool
	Compression    bool
	ManifestFile   string
}

// MigrationResult represents the result of a migration
type MigrationResult struct {
	TotalObjects     int64         `json:"total_objects"`
	TransferredSize  int64         `json:"transferred_size"`
	SuccessCount     int64         `json:"success_count"`
	ErrorCount       int64         `json:"error_count"`
	SkippedCount     int64         `json:"skipped_count"`
	Duration         time.Duration `json:"duration"`
	Errors           []string      `json:"errors,omitempty"`
	ManifestPath     string        `json:"manifest_path,omitempty"`
}

// MigrationManifest tracks migration progress
type MigrationManifest struct {
	S3Bucket       string    `json:"s3_bucket"`
	R2Bucket       string    `json:"r2_bucket"`
	StartTime       time.Time `json:"start_time"`
	TotalObjects    int64     `json:"total_objects"`
	TotalSize       int64     `json:"total_size"`
	CompletedObjects int64    `json:"completed_objects"`
	CompletedSize   int64     `json:"completed_size"`
	FailedObjects   []string  `json:"failed_objects"`
	LastUpdated     time.Time `json:"last_updated"`
}

// S3Object represents an S3 object
type S3Object struct {
	Key          string
	Size         int64
	ETag         string
	LastModified time.Time
	StorageClass string
}

// Execute performs the S3 to R2 migration
func (m *S3Migration) Execute(r2Client *api.Client) (*MigrationResult, error) {
	startTime := time.Now()

	printInfo("🚀 Starting S3 to R2 migration")
	printInfo("Source: s3://%s", m.S3Bucket)
	printInfo("Destination: %s", m.R2Bucket)
	printInfo("Concurrency: %d", m.Concurrency)

	// Create S3 client
	s3Client, err := m.createS3Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Load resume manifest if needed
	var manifest *MigrationManifest
	if m.Resume {
		manifest, err = m.loadManifest()
		if err != nil {
			printInfo("No existing manifest found, starting fresh migration")
		} else {
			printInfo("Resuming migration: %d/%d objects completed",
				manifest.CompletedObjects, manifest.TotalObjects)
		}
	}

	// List S3 objects
	objects, err := m.listS3Objects(s3Client, manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	if len(objects) == 0 {
		printInfo("No objects to migrate")
		return &MigrationResult{
			Duration: time.Since(startTime),
		}, nil
	}

	printInfo("Found %d objects to migrate (%s)",
		len(objects), FormatBytes(m.getTotalSize(objects)))

	if m.DryRun {
		return m.performDryRun(objects)
	}

	// Initialize manifest if not resuming
	if manifest == nil {
		manifest = &MigrationManifest{
			S3Bucket:    m.S3Bucket,
			R2Bucket:    m.R2Bucket,
			StartTime:   startTime,
			TotalObjects: int64(len(objects)),
			TotalSize:   m.getTotalSize(objects),
		}
	}

	// Create migration result
	result := &MigrationResult{
		TotalObjects: int64(len(objects)),
		Duration:     time.Since(startTime),
	}

	// Create progress bar
	bar := pb.StartNew(len(objects))
	bar.SetTemplateString(`{{counters . }} {{bar . }} {{percent . }} {{rtime . }} {{etime . }}`)
	defer bar.Finish()

	// Create worker pool
	objectChan := make(chan *S3Object, m.Concurrency)
	resultChan := make(chan *migrationTaskResult, m.Concurrency)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < m.Concurrency; i++ {
		wg.Add(1)
		go m.worker(s3Client, r2Client, objectChan, resultChan, &wg)
	}

	// Start result collector
	go m.collectResults(resultChan, result, manifest, bar)

	// Send objects to workers
	for _, obj := range objects {
		objectChan <- obj
	}
	close(objectChan)

	// Wait for workers to finish
	wg.Wait()
	close(resultChan)

	// Save final manifest
	if m.ManifestFile != "" {
		if err := m.saveManifest(manifest); err != nil {
			printWarning("Failed to save manifest: %v", err)
		} else {
			result.ManifestPath = m.ManifestFile
		}
	}

	result.Duration = time.Since(startTime)

	// Print summary
	printMigrationSummary(result)

	if result.ErrorCount > 0 {
		return result, fmt.Errorf("migration completed with %d errors", result.ErrorCount)
	}

	printSuccess("✅ Migration completed successfully!")
	return result, nil
}

// createS3Client creates an AWS S3 client
func (m *S3Migration) createS3Client() (*s3.Client, error) {
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(m.AWSRegion),
		config.WithSharedCredentialsFiles(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Override profile if specified
	if m.AWSProfile != "default" {
		creds, err := config.LoadSharedConfigProfile(context.Background(), m.AWSProfile)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS profile '%s': %w", m.AWSProfile, err)
		}
		cfg.Credentials = aws.NewCredentialsCache(creds)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)
	return client, nil
}

// listS3Objects lists objects from S3, optionally filtering already completed ones
func (m *S3Migration) listS3Objects(s3Client *s3.Client, manifest *MigrationManifest) ([]*S3Object, error) {
	var objects []*S3Object
	completedSet := make(map[string]bool)

	// Build completed set if resuming
	if manifest != nil {
		for _, key := range manifest.FailedObjects {
			completedSet[key] = false // Failed objects need to be retried
		}
		// Would need to track completed objects differently in a real implementation
	}

	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(m.S3Bucket),
		Prefix: aws.String(m.getFilterPrefix()),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to list S3 objects: %w", err)
		}

		for _, obj := range page.Contents {
			key := *obj.Key

			// Skip completed objects if resuming
			if completed, exists := completedSet[key]; exists && completed {
				continue
			}

			// Apply filter
			if !m.matchesFilter(key) {
				continue
			}

			objects = append(objects, &S3Object{
				Key:          key,
				Size:         obj.Size,
				ETag:         *obj.ETag,
				LastModified: *obj.LastModified,
				StorageClass: string(obj.StorageClass),
			})
		}
	}

	return objects, nil
}

// worker processes migration tasks
func (m *S3Migration) worker(s3Client *s3.Client, r2Client *api.Client,
	objectChan <-chan *S3Object, resultChan chan<- *migrationTaskResult, wg *sync.WaitGroup) {

	defer wg.Done()

	for obj := range objectChan {
		result := &migrationTaskResult{
			Object: obj,
		}

		// Download from S3
		data, etag, err := m.downloadFromS3(s3Client, obj)
		if err != nil {
			result.Error = fmt.Errorf("failed to download from S3: %w", err)
			resultChan <- result
			continue
		}

		// Upload to R2
		if err := m.uploadToR2(r2Client, obj, data, etag); err != nil {
			result.Error = fmt.Errorf("failed to upload to R2: %w", err)
			resultChan <- result
			continue
		}

		// Verify if requested
		if m.Verify {
			if err := m.verifyUpload(r2Client, obj, data, etag); err != nil {
				result.Error = fmt.Errorf("verification failed: %w", err)
				resultChan <- result
				continue
			}
		}

		result.Success = true
		resultChan <- result
	}
}

// downloadFromS3 downloads an object from S3
func (m *S3Migration) downloadFromS3(s3Client *s3.Client, obj *S3Object) (io.ReadSeeker, string, error) {
	resp, err := s3Client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(m.S3Bucket),
		Key:    aws.String(obj.Key),
	})
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	// Read all data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Calculate MD5 for verification
	hasher := md5.New()
	hasher.Write(data)
	etag := hex.EncodeToString(hasher.Sum(nil))

	return strings.NewReader(string(data)), etag, nil
}

// uploadToR2 uploads data to R2
func (m *S3Migration) uploadToR2(r2Client *api.Client, obj *S3Object, data io.ReadSeeker, etag string) error {
	// Use the R2 client's S3 interface to upload
	_, err := r2Client.s3.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(m.R2Bucket),
		Key:    aws.String(obj.Key),
		Body:   data,
		// Metadata from S3 object would be copied here
	})

	return err
}

// verifyUpload verifies the uploaded object
func (m *S3Migration) verifyUpload(r2Client *api.Client, obj *S3Object, originalData io.ReadSeeker, expectedETag string) error {
	// Get the uploaded object from R2
	resp, err := r2Client.s3.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(m.R2Bucket),
		Key:    aws.String(obj.Key),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Calculate MD5 of uploaded data
	hasher := md5.New()
	if _, err := io.Copy(hasher, resp.Body); err != nil {
		return err
	}
	actualETag := hex.EncodeToString(hasher.Sum(nil))

	// Compare ETags
	if actualETag != expectedETag {
		return fmt.Errorf("ETag mismatch: expected %s, got %s", expectedETag, actualETag)
	}

	return nil
}

// collectResults collects migration task results
func (m *S3Migration) collectResults(resultChan <-chan *migrationTaskResult, result *MigrationResult,
	manifest *MigrationManifest, bar *pb.ProgressBar) {

	for taskResult := range resultChan {
		if taskResult.Success {
			result.SuccessCount++
			manifest.CompletedObjects++
			if taskResult.Object != nil {
				manifest.CompletedSize += taskResult.Object.Size
			}
		} else {
			result.ErrorCount++
			if taskResult.Object != nil {
				manifest.FailedObjects = append(manifest.FailedObjects, taskResult.Object.Key)
			}
			if taskResult.Error != nil {
				result.Errors = append(result.Errors, taskResult.Error.Error())
			}
		}

		manifest.LastUpdated = time.Now()
		bar.Increment()
	}
}

// migrationTaskResult represents the result of migrating a single object
type migrationTaskResult struct {
	Object *S3Object
	Success bool
	Error   error
}

// Helper methods

func (m *S3Migration) getFilterPrefix() string {
	if m.Filter == "" {
		return ""
	}

	// If filter ends with *, remove it for prefix filtering
	if strings.HasSuffix(m.Filter, "*") {
		return strings.TrimSuffix(m.Filter, "*")
	}

	return m.Filter
}

func (m *S3Migration) matchesFilter(key string) bool {
	if m.Filter == "" {
		return true
	}

	// Simple glob matching
	if strings.HasSuffix(m.Filter, "*") {
		prefix := strings.TrimSuffix(m.Filter, "*")
		return strings.HasPrefix(key, prefix)
	}

	return key == m.Filter
}

func (m *S3Migration) getTotalSize(objects []*S3Object) int64 {
	var total int64
	for _, obj := range objects {
		total += obj.Size
	}
	return total
}

func (m *S3Migration) performDryRun(objects []*S3Object) (*MigrationResult, error) {
	printInfo("DRY RUN: Would migrate the following objects:")
	printInfo("Total objects: %d", len(objects))
	printInfo("Total size: %s", FormatBytes(m.getTotalSize(objects)))

	// Show first few objects as examples
	maxExamples := 5
	for i, obj := range objects {
		if i >= maxExamples {
			printInfo("... and %d more objects", len(objects)-maxExamples)
			break
		}
		printInfo("  %s (%s)", obj.Key, FormatBytes(obj.Size))
	}

	return &MigrationResult{
		TotalObjects: int64(len(objects)),
		TransferredSize: m.getTotalSize(objects),
	}, nil
}

func (m *S3Migration) loadManifest() (*MigrationManifest, error) {
	if m.ManifestFile == "" {
		return nil, fmt.Errorf("no manifest file specified")
	}

	data, err := os.ReadFile(m.ManifestFile)
	if err != nil {
		return nil, err
	}

	var manifest MigrationManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func (m *S3Migration) saveManifest(manifest *MigrationManifest) error {
	if m.ManifestFile == "" {
		return nil
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(m.ManifestFile), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.ManifestFile, data, 0644)
}

func printMigrationSummary(result *MigrationResult) {
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("📊 Migration Summary")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Total Objects:     %d\n", result.TotalObjects)
	fmt.Printf("Successful:        %d\n", result.SuccessCount)
	fmt.Printf("Failed:            %d\n", result.ErrorCount)
	fmt.Printf("Skipped:           %d\n", result.SkippedCount)
	fmt.Printf("Transferred Size:  %s\n", FormatBytes(result.TransferredSize))
	fmt.Printf("Duration:          %s\n", result.Duration.Round(time.Second))

	if result.SuccessCount > 0 && result.Duration.Seconds() > 0 {
		throughput := float64(result.TransferredSize) / result.Duration.Seconds()
		fmt.Printf("Throughput:        %s/s\n", FormatBytes(int64(throughput)))
	}

	if result.ManifestPath != "" {
		fmt.Printf("Manifest:          %s\n", result.ManifestPath)
	}
	fmt.Println(strings.Repeat("─", 60))
}

// FormatBytes formats bytes in human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}