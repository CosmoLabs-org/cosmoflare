package migration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

const (
	maxRetries       = 3
	multipartMinSize = 100 * 1024 * 1024
)

type transferResult struct {
	Key  string
	Size int64
	Err  error
}

func retryDelay(attempt int) time.Duration {
	base := time.Second
	for i := 0; i < attempt; i++ {
		base *= 4
	}
	return base
}

func transferTimeout(size int64) time.Duration {
	minTimeout := 5 * time.Minute
	sizeBasedTimeout := time.Duration(size/(1024*1024)) * time.Second
	if sizeBasedTimeout > minTimeout {
		return sizeBasedTimeout
	}
	return minTimeout
}

func shouldUseMultipart(size int64) bool {
	return size >= multipartMinSize
}

func transferWorker(
	s3Client *s3.Client,
	r2Client cosmoflare.R2Client,
	s3Bucket, r2Bucket string,
	work <-chan *S3Object,
	results chan<- transferResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	for obj := range work {
		results <- transferObject(s3Client, r2Client, s3Bucket, r2Bucket, obj)
	}
}

func transferObject(
	s3Client *s3.Client,
	r2Client cosmoflare.R2Client,
	s3Bucket, r2Bucket string,
	obj *S3Object,
) transferResult {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay(attempt - 1))
		}
		err := doTransfer(s3Client, r2Client, s3Bucket, r2Bucket, obj)
		if err == nil {
			return transferResult{Key: obj.Key, Size: obj.Size}
		}
		lastErr = err
	}
	return transferResult{Key: obj.Key, Err: fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastErr)}
}

func doTransfer(
	s3Client *s3.Client,
	r2Client cosmoflare.R2Client,
	s3Bucket, r2Bucket string,
	obj *S3Object,
) error {
	timeout := transferTimeout(obj.Size)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	getOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(obj.Key),
	})
	if err != nil {
		return fmt.Errorf("S3 GetObject failed for %s: %w", obj.Key, err)
	}
	defer getOutput.Body.Close()

	size := obj.Size
	if getOutput.ContentLength != nil {
		size = *getOutput.ContentLength
	}

	if shouldUseMultipart(size) {
		_, err = r2Client.MultipartUpload(ctx, r2Bucket, obj.Key, getOutput.Body, size)
	} else {
		_, err = r2Client.Upload(ctx, r2Bucket, obj.Key, getOutput.Body, size)
	}
	if err != nil {
		return fmt.Errorf("R2 upload failed for %s: %w", obj.Key, err)
	}
	return nil
}
