/*
Package migration provides real S3 to R2 migration functionality for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cheggaaa/pb/v3"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
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
func (m *S3Migration) Execute(r2Client cosmoflare.R2Client) (*MigrationResult, error) {
	startTime := time.Now()
	if m.Concurrency <= 0 {
		m.Concurrency = 10
	}

	printInfo("🚀 Starting S3 to R2 migration")
	printInfo("Source: s3://%s", m.S3Bucket)
	printInfo("Destination: %s", m.R2Bucket)
	printInfo("Concurrency: %d", m.Concurrency)

	s3Client, err := m.createS3Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	objects, err := m.listS3Objects(s3Client)
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	if len(objects) == 0 {
		printInfo("No objects to migrate")
		return &MigrationResult{Duration: time.Since(startTime)}, nil
	}

	totalSize := m.getTotalSize(objects)
	printInfo("Found %d objects (%s)", len(objects), FormatBytes(totalSize))

	if m.DryRun {
		return m.performDryRun(objects)
	}

	cpPath := m.checkpointPath()
	var cp *Checkpoint

	if m.Resume {
		cp, err = loadCheckpoint(cpPath)
		if err != nil {
			printWarning("Corrupt checkpoint, starting fresh: %v", err)
			cp = nil
		}
	} else {
		existing, _ := loadCheckpoint(cpPath)
		if existing != nil {
			printWarning("Stale checkpoint found. Use --resume to continue or delete %s", cpPath)
		}
	}

	if cp == nil {
		cp = &Checkpoint{
			Version:      1,
			S3Bucket:     m.S3Bucket,
			R2Bucket:     m.R2Bucket,
			Filter:       m.Filter,
			StartedAt:    startTime,
			TotalObjects: int64(len(objects)),
			TotalSize:    totalSize,
			Completed:    make(map[string]int64),
			path:         cpPath,
		}
	} else {
		cp.path = cpPath
		printInfo("Resuming: %d/%d already transferred (%s)",
			len(cp.Completed), cp.TotalObjects, FormatBytes(cp.TransferredSize))
	}

	var remaining []*S3Object
	for _, obj := range objects {
		if !cp.isCompleted(obj.Key) {
			remaining = append(remaining, obj)
		}
	}

	if len(remaining) == 0 {
		printInfo("All objects already transferred")
		deleteCheckpoint(cpPath)
		return &MigrationResult{
			TotalObjects:    int64(len(objects)),
			SuccessCount:    int64(len(objects)),
			TransferredSize: totalSize,
			Duration:        time.Since(startTime),
		}, nil
	}

	printInfo("%d objects remaining", len(remaining))

	var stopping atomic.Bool
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	work := make(chan *S3Object, m.Concurrency)
	results := make(chan transferResult, m.Concurrency)

	var wg sync.WaitGroup
	for i := 0; i < m.Concurrency; i++ {
		wg.Add(1)
		go transferWorker(s3Client, r2Client, m.S3Bucket, m.R2Bucket, work, results, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, obj := range remaining {
			if stopping.Load() {
				break
			}
			work <- obj
		}
		close(work)
	}()

	go func() {
		<-sigCh
		stopping.Store(true)
		signal.Stop(sigCh)
	}()

	bar := pb.StartNew(len(remaining))
	bar.SetTemplateString(`{{counters . }} {{bar . }} {{percent . }} {{rtime . }} {{etime . }}`)

	flushTicker := time.NewTicker(flushInterval)
	defer flushTicker.Stop()
	sinceLastFlush := 0

	result := &MigrationResult{
		TotalObjects: int64(len(objects)),
		SkippedCount: int64(len(objects) - len(remaining)),
	}

	for r := range results {
		if r.Err != nil {
			cp.markFailed(r.Key)
			result.ErrorCount++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", r.Key, r.Err))
		} else {
			cp.markCompleted(r.Key, r.Size)
			result.SuccessCount++
			result.TransferredSize += r.Size
		}
		bar.Increment()
		sinceLastFlush++

		select {
		case <-flushTicker.C:
			cp.flush()
			sinceLastFlush = 0
		default:
			if sinceLastFlush >= flushCount {
				cp.flush()
				sinceLastFlush = 0
			}
		}
	}

	bar.Finish()
	result.Duration = time.Since(startTime)

	if result.ErrorCount == 0 && !stopping.Load() {
		deleteCheckpoint(cpPath)
		printSuccess("✅ Migration completed successfully!")
	} else if stopping.Load() {
		cp.flush()
		printWarning("Interrupted: %d/%d transferred. Run with --resume to continue.",
			result.SuccessCount+result.SkippedCount, result.TotalObjects)
		printMigrationSummary(result)
		return result, fmt.Errorf("migration interrupted")
	} else {
		cp.flush()
		printWarning("%d objects failed. Run with --resume to retry.", result.ErrorCount)
	}

	if m.Verify && !stopping.Load() && result.ErrorCount == 0 {
		m.verifyTransfers(s3Client, r2Client, objects, result)
	}

	printMigrationSummary(result)
	return result, nil
}

func (m *S3Migration) verifyTransfers(
	s3Client *s3.Client,
	r2Client cosmoflare.R2Client,
	objects []*S3Object,
	result *MigrationResult,
) {
	printInfo("🔍 Verifying transferred objects...")

	mismatchCount := 0
	bar := pb.StartNew(len(objects))
	bar.SetTemplateString(`Verifying: {{counters . }} {{bar . }} {{percent . }}`)

	for _, obj := range objects {
		bar.Increment()

		r2Head, err := r2Client.HeadObject(context.Background(), m.R2Bucket, obj.Key)
		if err != nil {
			printWarning("⚠ Verify failed for %s: %v", obj.Key, err)
			mismatchCount++
			continue
		}

		s3ETag := strings.Trim(obj.ETag, "\"")
		r2ETag := strings.Trim(r2Head.ETag, "\"")

		if s3ETag != r2ETag {
			printWarning("⚠ ETag mismatch: %s (S3: %s, R2: %s)", obj.Key, s3ETag, r2ETag)
			mismatchCount++
		}
	}

	bar.Finish()

	if mismatchCount == 0 {
		printSuccess("✅ All %d objects verified", len(objects))
	} else {
		printWarning("⚠ %d verification mismatches found", mismatchCount)
	}
}

// createS3Client creates an AWS S3 client
func (m *S3Migration) createS3Client() (*s3.Client, error) {
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(m.AWSRegion),
		config.WithSharedConfigProfile(m.AWSProfile),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)
	return client, nil
}

// listS3Objects lists objects from S3
func (m *S3Migration) listS3Objects(s3Client *s3.Client) ([]*S3Object, error) {
	var objects []*S3Object

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

			// Apply filter
			if !m.matchesFilter(key) {
				continue
			}

			objects = append(objects, &S3Object{
				Key:          key,
				Size:         aws.ToInt64(obj.Size),
				ETag:         *obj.ETag,
				LastModified: *obj.LastModified,
				StorageClass: string(obj.StorageClass),
			})
		}
	}

	return objects, nil
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

func printInfo(format string, args ...interface{}) {
	fmt.Printf("  ℹ "+format+"\n", args...)
}

func printSuccess(format string, args ...interface{}) {
	fmt.Printf("  ✓ "+format+"\n", args...)
}

func printWarning(format string, args ...interface{}) {
	fmt.Printf("  ⚠ "+format+"\n", args...)
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