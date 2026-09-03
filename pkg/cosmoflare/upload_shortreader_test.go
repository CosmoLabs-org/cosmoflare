package cosmoflare

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// s3CallLog records the multipart protocol calls a fake S3 endpoint receives.
// Handler goroutines append concurrently, so all access is mutex-guarded.
type s3CallLog struct {
	mu    sync.Mutex
	calls []string
}

func (l *s3CallLog) add(call string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, call)
}

func (l *s3CallLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.calls))
	copy(out, l.calls)
	return out
}

// waitFor polls until the exact call is observed or the timeout expires.
func (l *s3CallLog) waitFor(t *testing.T, want string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, c := range l.snapshot() {
			if c == want {
				return true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// newFakeS3MultipartClient returns a client whose S3 endpoint is a local test
// server implementing the multipart upload protocol (create / upload part /
// complete / abort), plus the call log for assertions.
func newFakeS3MultipartClient(t *testing.T) (*client, *s3CallLog) {
	t.Helper()

	log := &s3CallLog{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.Method == http.MethodPost && q.Has("uploads"):
			log.add("create")
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><InitiateMultipartUploadResult><UploadId>bug026-upload-1</UploadId></InitiateMultipartUploadResult>`)
		case r.Method == http.MethodPut && q.Has("partNumber"):
			log.add(fmt.Sprintf("part:%s:%d", q.Get("partNumber"), r.ContentLength))
			w.Header().Set("ETag", `"etag-`+q.Get("partNumber")+`"`)
		case r.Method == http.MethodPost && q.Has("uploadId"):
			log.add("complete")
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><CompleteMultipartUploadResult><ETag>"final-etag"</ETag></CompleteMultipartUploadResult>`)
		case r.Method == http.MethodDelete && q.Has("uploadId"):
			log.add("abort")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("fake S3: unexpected request %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "test-access", SecretAccessKey: "test-secret"}, nil
		})),
	)
	if err != nil {
		t.Fatalf("failed to load AWS config: %v", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(server.URL)
		o.UsePathStyle = true
	})

	return &client{accountID: "test-account", s3: s3Client, cfg: &clientConfig{region: "auto"}}, log
}

// assertShortReadError verifies the BUG-026 contract: a reader that delivers
// fewer bytes than the declared size must produce an R2ValidationError that
// names the observed and declared byte counts.
func assertShortReadError(t *testing.T, err error, gotBytes, wantBytes int64) {
	t.Helper()
	if err == nil {
		t.Fatalf("BUG-026: expected error for short reader, got nil (truncated object stored without error)")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *R2ValidationError, got %T: %v", err, err)
	}
	want := fmt.Sprintf("reader shorter than declared size (%d of %d bytes)", gotBytes, wantBytes)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error should contain %q, got: %v", want, err)
	}
}

// assertNoCompleteOrEmptyParts verifies the truncated upload was never
// completed and no empty part was ever sent.
func assertNoCompleteOrEmptyParts(t *testing.T, log *s3CallLog) {
	t.Helper()
	for _, call := range log.snapshot() {
		if call == "complete" {
			t.Error("BUG-026: truncated multipart upload must not be completed")
		}
		if strings.HasPrefix(call, "part:") && strings.HasSuffix(call, ":0") {
			t.Errorf("BUG-026: empty part must not be uploaded, got call %q", call)
		}
	}
}

// --- BUG-026: short reader must abort, not silently store a truncated object ---

func TestMultipartUpload_ReaderShorterThanDeclaredSize(t *testing.T) {
	c, log := newFakeS3MultipartClient(t)

	const (
		size     = 10 * 1024 * 1024 // declared 10MB
		partSize = 4 * 1024 * 1024
	)
	// Reader delivers 5MB then EOF: part 1 reads 4MB, part 2 gets 1MB and
	// hits io.ErrUnexpectedEOF at 5MB of the declared 10MB.
	shortReader := bytes.NewReader(make([]byte, 5*1024*1024))

	result, err := c.MultipartUpload(t.Context(), "bug026-bucket", "short-reader.bin",
		shortReader, size, WithPartSize(partSize))
	assertShortReadError(t, err, 5*1024*1024, size)
	if result != nil {
		t.Errorf("expected nil result on short reader, got %+v", result)
	}

	if !log.waitFor(t, "abort", 5*time.Second) {
		t.Error("expected AbortMultipartUpload to be called for the truncated upload")
	}
	assertNoCompleteOrEmptyParts(t, log)
}

func TestResumableMultipartUpload_ReaderShorterThanDeclaredSize(t *testing.T) {
	c, log := newFakeS3MultipartClient(t)

	const (
		size     = 12 * 1024 * 1024 // declared 12MB
		partSize = 5 * 1024 * 1024  // minimum allowed part size
	)
	// Reader delivers 7MB then EOF: part 1 reads 5MB, part 2 gets 2MB and
	// hits io.ErrUnexpectedEOF at 7MB of the declared 12MB.
	shortReader := bytes.NewReader(make([]byte, 7*1024*1024))

	defer RemoveUploadState("bug026-bucket", "resumable-short.bin")

	result, err := c.ResumableMultipartUpload(t.Context(), "bug026-bucket", "resumable-short.bin",
		shortReader, size, WithPartSize(partSize))
	assertShortReadError(t, err, 7*1024*1024, size)
	if result != nil {
		t.Errorf("expected nil result on short reader, got %+v", result)
	}

	if !log.waitFor(t, "abort", 5*time.Second) {
		t.Error("expected AbortMultipartUpload to be called for the truncated upload")
	}
	assertNoCompleteOrEmptyParts(t, log)
}

func TestResumeMultipartUpload_ReaderShorterThanDeclaredSize(t *testing.T) {
	c, log := newFakeS3MultipartClient(t)

	const (
		size     = 12 * 1024 * 1024
		partSize = 5 * 1024 * 1024
	)
	bucket, key := "bug026-bucket", "resume-short.bin"

	state := &MultipartUploadState{
		UploadID:   "bug026-upload-1",
		Bucket:     bucket,
		Key:        key,
		TotalSize:  size,
		PartSize:   partSize,
		TotalParts: 3,
		StartedAt:  time.Now().UTC(),
	}
	if err := SaveUploadState(state); err != nil {
		t.Fatal(err)
	}
	defer RemoveUploadState(bucket, key)

	// Reader delivers 7MB then EOF: part 2 seek-read gets 2MB and hits
	// io.ErrUnexpectedEOF at 7MB of the declared 12MB.
	shortReader := bytes.NewReader(make([]byte, 7*1024*1024))

	result, err := c.ResumeMultipartUpload(t.Context(), bucket, key, shortReader, size, WithPartSize(partSize))
	assertShortReadError(t, err, 7*1024*1024, size)
	if result != nil {
		t.Errorf("expected nil result on short reader, got %+v", result)
	}

	// Resume keeps the server-side upload alive (no abort) so a corrected
	// retry can continue, but the truncated set of parts must never complete.
	assertNoCompleteOrEmptyParts(t, log)
}

// --- Guards: a reader that delivers exactly the declared size still succeeds ---

func TestMultipartUpload_ReaderExactlyDeclaredSize(t *testing.T) {
	c, log := newFakeS3MultipartClient(t)

	const (
		size     = 10 * 1024 * 1024
		partSize = 4 * 1024 * 1024
	)
	fullReader := bytes.NewReader(make([]byte, size))

	result, err := c.MultipartUpload(t.Context(), "bug026-bucket", "exact-reader.bin",
		fullReader, size, WithPartSize(partSize))
	if err != nil {
		t.Fatalf("clean EOF exactly at size boundary must succeed, got: %v", err)
	}
	if result == nil || result.Size != size {
		t.Fatalf("expected result with Size=%d, got %+v", size, result)
	}
	if result.Parts != 3 {
		t.Errorf("expected 3 parts, got %d", result.Parts)
	}

	if !log.waitFor(t, "complete", 5*time.Second) {
		t.Fatal("expected CompleteMultipartUpload to be called")
	}
	for _, call := range log.snapshot() {
		if call == "abort" {
			t.Errorf("exact-size upload must not be aborted, calls: %v", log.snapshot())
		}
	}
}

func TestResumableMultipartUpload_ReaderExactlyDeclaredSize(t *testing.T) {
	c, log := newFakeS3MultipartClient(t)

	const (
		size     = 10 * 1024 * 1024
		partSize = 5 * 1024 * 1024
	)
	fullReader := bytes.NewReader(make([]byte, size))

	defer RemoveUploadState("bug026-bucket", "resumable-exact.bin")

	result, err := c.ResumableMultipartUpload(t.Context(), "bug026-bucket", "resumable-exact.bin",
		fullReader, size, WithPartSize(partSize))
	if err != nil {
		t.Fatalf("clean EOF exactly at size boundary must succeed, got: %v", err)
	}
	if result == nil || result.Size != size {
		t.Fatalf("expected result with Size=%d, got %+v", size, result)
	}

	if !log.waitFor(t, "complete", 5*time.Second) {
		t.Fatal("expected CompleteMultipartUpload to be called")
	}
	for _, call := range log.snapshot() {
		if call == "abort" {
			t.Errorf("exact-size upload must not be aborted, calls: %v", log.snapshot())
		}
	}
}

// --- readUploadPart invariant tests ---

// eofReader yields its data, then io.EOF on every subsequent call — including
// calls with an empty buffer, unlike bytes.Reader.
type eofReader struct {
	data []byte
}

func (r *eofReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

// errDeviceFailure is the sentinel cause returned by failingReader.
var errDeviceFailure = errors.New("device failure")

// failingReader always fails with a non-EOF error.
type failingReader struct{}

func (failingReader) Read(p []byte) (int, error) {
	return 0, errDeviceFailure
}

func TestReadUploadPart_FullPart(t *testing.T) {
	reader := bytes.NewReader(make([]byte, 100))
	buf, err := readUploadPart("MultipartUpload", reader, 40, 100, 0, 1)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(buf) != 40 {
		t.Errorf("expected 40-byte part, got %d", len(buf))
	}
}

func TestReadUploadPart_LastPartCapped(t *testing.T) {
	reader := bytes.NewReader(make([]byte, 20))
	// 80 of 100 bytes already read; the final part must cap at 20 bytes.
	buf, err := readUploadPart("MultipartUpload", reader, 40, 100, 80, 3)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(buf) != 20 {
		t.Errorf("expected capped 20-byte final part, got %d", len(buf))
	}
}

func TestReadUploadPart_ShortReadMidPart(t *testing.T) {
	// Declared 100 bytes, reader delivers 50 then EOF.
	_, err := readUploadPart("MultipartUpload", &eofReader{data: make([]byte, 50)}, 80, 100, 0, 1)
	if err == nil {
		t.Fatal("BUG-026: expected validation error for short reader")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *R2ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "reader shorter than declared size (50 of 100 bytes)") {
		t.Errorf("error should report byte counts, got: %v", err)
	}
}

func TestReadUploadPart_EOFBetweenParts(t *testing.T) {
	// 80 of 100 bytes consumed; the reader is exhausted before the final
	// 20-byte part. This is the empty-part case from the bug report.
	_, err := readUploadPart("MultipartUpload", &eofReader{}, 40, 100, 80, 3)
	if err == nil {
		t.Fatal("BUG-026: expected validation error when reader ends between parts")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *R2ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "reader shorter than declared size (80 of 100 bytes)") {
		t.Errorf("error should report byte counts, got: %v", err)
	}
}

func TestReadUploadPart_CleanEOFAtSizeBoundary(t *testing.T) {
	// Every declared byte is already consumed; a reader that reports EOF even
	// for the empty final read must be treated as success.
	buf, err := readUploadPart("MultipartUpload", &eofReader{}, 40, 100, 100, 3)
	if err != nil {
		t.Fatalf("clean EOF exactly at size boundary must succeed, got %v", err)
	}
	if len(buf) != 0 {
		t.Errorf("expected empty final part, got %d bytes", len(buf))
	}
}

func TestReadUploadPart_ZeroSize(t *testing.T) {
	// Zero-size uploads loop once with an empty part; that must stay valid.
	buf, err := readUploadPart("MultipartUpload", &eofReader{}, 40, 0, 0, 1)
	if err != nil {
		t.Fatalf("zero-size upload must succeed, got %v", err)
	}
	if len(buf) != 0 {
		t.Errorf("expected empty part for zero size, got %d bytes", len(buf))
	}
}

func TestReadUploadPart_HardReadError(t *testing.T) {
	_, err := readUploadPart("MultipartUpload", failingReader{}, 40, 100, 0, 1)
	if err == nil {
		t.Fatal("expected error from failing reader")
	}
	var valErr *R2ValidationError
	if errors.As(err, &valErr) {
		t.Fatalf("hard read failure must not be a validation error, got %v", err)
	}
	if !strings.Contains(err.Error(), "failed to read part 1") {
		t.Errorf("error should name the failing part, got: %v", err)
	}
	if !errors.Is(err, errDeviceFailure) {
		t.Errorf("error should wrap the underlying failure (R2Error does not print it), got: %v", err)
	}
}
