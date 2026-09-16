package migration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// --- test helpers -----------------------------------------------------------

func newAtomicFalse() *atomic.Bool { var b atomic.Bool; return &b }

func newAtomicTrue() *atomic.Bool { b := new(atomic.Bool); b.Store(true); return b }

// fakeR2 implements cosmoflare.R2Client for tests. The embedded nil interface
// means any unexpected method call panics, which is a useful test signal.
type fakeR2 struct {
	cosmoflare.R2Client

	mu        sync.Mutex
	uploads   map[string]int64
	headFn    func(key string) (*cosmoflare.HeadResult, error)
	uploadErr error
}

func newFakeR2() *fakeR2 {
	return &fakeR2{uploads: map[string]int64{}}
}

func (f *fakeR2) Upload(_ context.Context, _, key string, reader io.Reader, size int64, _ ...cosmoflare.UploadOption) (*cosmoflare.UploadResult, error) {
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	n, _ := io.Copy(io.Discard, reader)
	f.mu.Lock()
	f.uploads[key] = n
	f.mu.Unlock()
	return &cosmoflare.UploadResult{Key: key, Size: n}, nil
}

func (f *fakeR2) MultipartUpload(_ context.Context, _, key string, reader io.Reader, size int64, _ ...cosmoflare.UploadOption) (*cosmoflare.UploadResult, error) {
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	n, _ := io.Copy(io.Discard, reader)
	f.mu.Lock()
	f.uploads[key] = n
	f.mu.Unlock()
	return &cosmoflare.UploadResult{Key: key, Size: n}, nil
}

func (f *fakeR2) HeadObject(_ context.Context, _, key string) (*cosmoflare.HeadResult, error) {
	if f.headFn != nil {
		return f.headFn(key)
	}
	return &cosmoflare.HeadResult{Key: key}, nil
}

// s3Fixture is a fake S3 backend served over httptest.
type s3Fixture struct {
	server *httptest.Server
}

// newS3Fixture starts a fake S3 API. objects are served for ListObjectsV2
// (split over two pages when paginated is true) and GetObject.
func newS3Fixture(t *testing.T, objects []S3Object, paginated bool) *s3Fixture {
	t.Helper()
	f := &s3Fixture{}
	listPage := func(objs []S3Object, truncated bool, token string) string {
		var b strings.Builder
		b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
		b.WriteString(`<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
		b.WriteString(`<Name>src-bucket</Name><IsTruncated>`)
		if truncated {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
		b.WriteString(`</IsTruncated>`)
		if truncated {
			fmt.Fprintf(&b, `<NextContinuationToken>%s</NextContinuationToken>`, token)
		}
		for _, o := range objs {
			fmt.Fprintf(&b,
				`<Contents><Key>%s</Key><Size>%d</Size><ETag>%s</ETag><LastModified>%s</LastModified><StorageClass>%s</StorageClass></Contents>`,
				o.Key, o.Size, o.ETag, o.LastModified.UTC().Format(time.RFC3339), o.StorageClass)
		}
		b.WriteString(`</ListBucketResult>`)
		return b.String()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("list-type") == "2" {
			if r.URL.Query().Get("marker-error") == "1" {
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
				return
			}
			if paginated {
				if r.URL.Query().Get("continuation-token") == "" {
					w.Header().Set("Content-Type", "application/xml")
					io.WriteString(w, listPage(objects[:1], true, "tok1"))
					return
				}
				w.Header().Set("Content-Type", "application/xml")
				io.WriteString(w, listPage(objects[1:], false, ""))
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, listPage(objects, false, ""))
			return
		}
		// GetObject (path-style requests arrive as /<bucket>/<key>)
		key := strings.TrimPrefix(r.URL.Path, "/")
		if i := strings.Index(key, "/"); i >= 0 {
			key = key[i+1:]
		}
		for _, o := range objects {
			if o.Key == key {
				body := fmt.Sprintf("content-of-%s", key)
				w.Header().Set("Content-Length", fmt.Sprintf("%d", int64(len(body))))
				w.Header().Set("ETag", o.ETag)
				io.WriteString(w, body)
				return
			}
		}
		http.Error(w, `{"error":"NoSuchKey"}`, http.StatusNotFound)
	})
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

// s3ClientFor returns an s3.Client pointed at the fixture. listS3Objects,
// scanPhase and runTransfer all accept the client as a parameter, so tests
// inject the fixture-backed client directly.
func s3ClientFor(f *s3Fixture) *s3.Client {
	return s3.NewFromConfig(aws.Config{
		Region: "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider(
			"test", "test", "test"),
	}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(f.server.URL)
		o.UsePathStyle = true
	})
}

// testObjects returns a small deterministic object set.
func testObjects() []S3Object {
	return []S3Object{
		{Key: "a.jpg", Size: 100, ETag: `"etag-a"`, LastModified: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), StorageClass: "STANDARD"},
		{Key: "b.jpg", Size: 200, ETag: `"etag-b"`, LastModified: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), StorageClass: "STANDARD"},
		{Key: "c.txt", Size: 300, ETag: `"etag-c"`, LastModified: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), StorageClass: "STANDARD"},
	}
}

// withTestHome isolates checkpointDir() inside a temp dir.
func withTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

// withTestAWSEnv points config.LoadDefaultConfig at the fixture so Execute
// (which builds its own client) hits the fake S3.
func withTestAWSEnv(t *testing.T, url string) {
	t.Helper()
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ENDPOINT_URL_S3", url)
}

// --- pure helpers -----------------------------------------------------------

func TestS3FormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{5 * 1024 * 1024 * 1024, "5.0 GB"},
	}
	for _, c := range cases {
		if got := FormatBytes(c.in); got != c.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestS3GetFilterPrefix(t *testing.T) {
	cases := []struct {
		filter string
		want   string
	}{
		{"", ""},
		{"logs/*", "logs/"},
		{"logs/", "logs/"},
		{"exact.txt", "exact.txt"},
		{"*", ""},
	}
	for _, c := range cases {
		m := &S3Migration{Filter: c.filter}
		if got := m.getFilterPrefix(); got != c.want {
			t.Errorf("getFilterPrefix(%q) = %q, want %q", c.filter, got, c.want)
		}
	}
}

func TestS3MatchesFilter(t *testing.T) {
	cases := []struct {
		filter string
		key    string
		want   bool
	}{
		{"", "anything.bin", true},
		{"logs/*", "logs/2025/a.log", true},
		{"logs/*", "other/a.log", false},
		{"exact.txt", "exact.txt", true},
		{"exact.txt", "other.txt", false},
	}
	for _, c := range cases {
		m := &S3Migration{Filter: c.filter}
		if got := m.matchesFilter(c.key); got != c.want {
			t.Errorf("matchesFilter(filter=%q, key=%q) = %v, want %v", c.filter, c.key, got, c.want)
		}
	}
}

func TestS3GetTotalSize(t *testing.T) {
	m := &S3Migration{}
	objs := []*S3Object{{Size: 10}, {Size: 20}, {Size: 32}}
	if got := m.getTotalSize(objs); got != 62 {
		t.Errorf("getTotalSize = %d, want 62", got)
	}
	if got := m.getTotalSize(nil); got != 0 {
		t.Errorf("getTotalSize(nil) = %d, want 0", got)
	}
}

func TestS3PrintHelpersAndSummary(t *testing.T) {
	// These write to stdout; calling them verifies they do not panic.
	printInfo("hello %s", "world")
	printSuccess("done")
	printWarning("careful")
	printMigrationSummary(&MigrationResult{
		TotalObjects:    3,
		SuccessCount:    2,
		ErrorCount:      1,
		TransferredSize: 1024,
		Duration:        2 * time.Second,
		ManifestPath:    "/tmp/manifest.json",
	})
	printMigrationSummary(&MigrationResult{Duration: 0})
}

// --- performDryRun ----------------------------------------------------------

func TestS3PerformDryRun(t *testing.T) {
	m := &S3Migration{}
	objs := []*S3Object{{Key: "a", Size: 100}, {Key: "b", Size: 200}}
	res, err := m.performDryRun(objs)
	if err != nil {
		t.Fatalf("performDryRun failed: %v", err)
	}
	if res.TotalObjects != 2 {
		t.Errorf("TotalObjects = %d, want 2", res.TotalObjects)
	}
	if res.TransferredSize != 300 {
		t.Errorf("TransferredSize = %d, want 300", res.TransferredSize)
	}
}

func TestS3PerformDryRunMoreThanFive(t *testing.T) {
	m := &S3Migration{}
	objs := make([]*S3Object, 7)
	for i := range objs {
		objs[i] = &S3Object{Key: fmt.Sprintf("obj%d", i), Size: 1}
	}
	res, err := m.performDryRun(objs)
	if err != nil {
		t.Fatalf("performDryRun failed: %v", err)
	}
	if res.TotalObjects != 7 {
		t.Errorf("TotalObjects = %d, want 7", res.TotalObjects)
	}
}

// --- saveManifest -----------------------------------------------------------

func TestS3SaveManifestSkipsWhenEmpty(t *testing.T) {
	m := &S3Migration{}
	if err := m.saveManifest(&MigrationManifest{}); err != nil {
		t.Errorf("saveManifest with no ManifestFile should be a no-op, got %v", err)
	}
}

func TestS3SaveManifestWritesJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "manifest.json")
	m := &S3Migration{ManifestFile: path}
	manifest := &MigrationManifest{S3Bucket: "src", R2Bucket: "dst", TotalObjects: 5}
	if err := m.saveManifest(manifest); err != nil {
		t.Fatalf("saveManifest failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("manifest not written: %v", err)
	}
	if !strings.Contains(string(data), `"s3_bucket": "src"`) {
		t.Errorf("manifest JSON missing s3_bucket: %s", data)
	}
}

func TestS3SaveManifestMkdirError(t *testing.T) {
	// Reason: parent "dir" is a regular file, so MkdirAll fails.
	base := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(base, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	m := &S3Migration{ManifestFile: filepath.Join(base, "manifest.json")}
	if err := m.saveManifest(&MigrationManifest{}); err == nil {
		t.Error("expected error when manifest parent is a file")
	}
}

// --- createS3Client ---------------------------------------------------------

func TestS3CreateS3Client(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_REGION", "us-east-1")
	m := &S3Migration{AWSRegion: "us-east-1"}
	client, err := m.createS3Client()
	if err != nil {
		t.Fatalf("createS3Client failed: %v", err)
	}
	if client == nil {
		t.Error("client should not be nil")
	}
}

// --- listS3Objects ----------------------------------------------------------

func TestS3ListObjectsAll(t *testing.T) {
	f := newS3Fixture(t, testObjects(), false)
	m := &S3Migration{S3Bucket: "src-bucket"}
	objs, err := m.listS3Objects(s3ClientFor(f))
	if err != nil {
		t.Fatalf("listS3Objects failed: %v", err)
	}
	if len(objs) != 3 {
		t.Fatalf("got %d objects, want 3", len(objs))
	}
	if objs[0].Key != "a.jpg" || objs[0].Size != 100 || objs[0].ETag != `"etag-a"` {
		t.Errorf("first object wrong: %+v", objs[0])
	}
	if objs[0].StorageClass != "STANDARD" {
		t.Errorf("StorageClass = %q, want STANDARD", objs[0].StorageClass)
	}
	if objs[0].LastModified.IsZero() {
		t.Error("LastModified should be populated")
	}
}

func TestS3ListObjectsPaginated(t *testing.T) {
	// Reason: covers the paginator loop across continuation-token pages.
	f := newS3Fixture(t, testObjects(), true)
	m := &S3Migration{S3Bucket: "src-bucket"}
	objs, err := m.listS3Objects(s3ClientFor(f))
	if err != nil {
		t.Fatalf("listS3Objects failed: %v", err)
	}
	if len(objs) != 3 {
		t.Errorf("paginated listing should yield 3 objects, got %d", len(objs))
	}
}

func TestS3ListObjectsFilterApplied(t *testing.T) {
	f := newS3Fixture(t, testObjects(), false)
	m := &S3Migration{S3Bucket: "src-bucket", Filter: "a.jpg"}
	objs, err := m.listS3Objects(s3ClientFor(f))
	if err != nil {
		t.Fatalf("listS3Objects failed: %v", err)
	}
	if len(objs) != 1 || objs[0].Key != "a.jpg" {
		t.Errorf("exact filter should keep only a.jpg, got %d objects", len(objs))
	}
}

func TestS3ListObjectsPrefixFilter(t *testing.T) {
	// Reason: "logs/*"-style prefix filters are the supported glob form.
	f := newS3Fixture(t, testObjects(), false)
	m := &S3Migration{S3Bucket: "src-bucket", Filter: "b*"}
	objs, err := m.listS3Objects(s3ClientFor(f))
	if err != nil {
		t.Fatalf("listS3Objects failed: %v", err)
	}
	if len(objs) != 1 || objs[0].Key != "b.jpg" {
		t.Errorf("prefix filter should keep only b.jpg, got %d objects", len(objs))
	}
}

func TestS3ListObjectsSuffixGlobFilterMatchesNothing(t *testing.T) {
	// BUG (not fixed, exercising current behavior): a suffix glob like
	// "*.jpg" turns into the literal prefix "*." in getFilterPrefix /
	// matchesFilter, so it matches nothing. Users would expect it to mean
	// "ends with .jpg". Callers must use prefix globs ("logs/*") instead.
	f := newS3Fixture(t, testObjects(), false)
	m := &S3Migration{S3Bucket: "src-bucket", Filter: "*.jpg"}
	objs, err := m.listS3Objects(s3ClientFor(f))
	if err != nil {
		t.Fatalf("listS3Objects failed: %v", err)
	}
	if len(objs) != 0 {
		t.Errorf("current behavior: suffix glob filter matches nothing, got %d", len(objs))
	}
}

func TestS3ListObjectsServerErrorsAfterRetries(t *testing.T) {
	// Reason: backend returns 500 on every attempt; the SDK retryer exhausts
	// its budget and listS3Objects must propagate the error.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	m := &S3Migration{S3Bucket: "src-bucket"}
	_, err := m.listS3Objects(s3ClientFor(&s3Fixture{server: server}))
	if err == nil {
		t.Error("expected error when S3 listing always fails")
	}
	if !strings.Contains(err.Error(), "failed to list S3 objects") {
		t.Errorf("error should mention list failure, got: %v", err)
	}
}

// --- remainingObjects -------------------------------------------------------

func TestS3RemainingObjects(t *testing.T) {
	objs := []*S3Object{{Key: "a"}, {Key: "b"}, {Key: "c"}}
	cp := &Checkpoint{Completed: map[string]int64{"b": 1}}
	remaining := remainingObjects(objs, cp)
	if len(remaining) != 2 {
		t.Fatalf("got %d remaining, want 2", len(remaining))
	}
	if remaining[0].Key != "a" || remaining[1].Key != "c" {
		t.Errorf("remaining = [%s,%s], want [a,c]", remaining[0].Key, remaining[1].Key)
	}
	if got := remainingObjects(objs, &Checkpoint{Completed: map[string]int64{}}); len(got) != 3 {
		t.Errorf("fresh checkpoint should leave all 3, got %d", len(got))
	}
}

// --- loadOrCreateCheckpoint -------------------------------------------------

func TestS3LoadOrCreateCheckpointFresh(t *testing.T) {
	withTestHome(t)
	m := &S3Migration{S3Bucket: "src", R2Bucket: "dst", Filter: "*.jpg"}
	start := time.Now()
	cp := m.loadOrCreateCheckpoint(filepath.Join(t.TempDir(), "cp.json"), start,
		[]*S3Object{{Key: "a"}, {Key: "b"}}, 42)
	if cp == nil {
		t.Fatal("checkpoint should not be nil")
	}
	if cp.Version != 1 || cp.S3Bucket != "src" || cp.R2Bucket != "dst" || cp.Filter != "*.jpg" {
		t.Errorf("fresh checkpoint misconfigured: %+v", cp)
	}
	if cp.TotalObjects != 2 || cp.TotalSize != 42 {
		t.Errorf("TotalObjects/TotalSize = %d/%d, want 2/42", cp.TotalObjects, cp.TotalSize)
	}
	if cp.Completed == nil {
		t.Error("Completed map should be initialized")
	}
}

func TestS3LoadOrCreateCheckpointResume(t *testing.T) {
	// Reason: Resume=true with a valid checkpoint on disk must load it.
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	path := checkpointPathIn(checkpointDir(), "src", "dst", "")
	existing := &Checkpoint{
		Version:      1,
		S3Bucket:     "src",
		R2Bucket:     "dst",
		TotalObjects: 5,
		Completed:    map[string]int64{"a": 10},
	}
	if err := saveCheckpoint(existing, path); err != nil {
		t.Fatal(err)
	}
	m := &S3Migration{S3Bucket: "src", R2Bucket: "dst", Resume: true}
	cp := m.loadOrCreateCheckpoint(path, time.Now(), nil, 0)
	if cp == nil {
		t.Fatal("checkpoint should not be nil")
	}
	if len(cp.Completed) != 1 || cp.Completed["a"] != 10 {
		t.Errorf("resumed checkpoint should contain a=10, got %+v", cp.Completed)
	}
	if cp.path != path {
		t.Errorf("cp.path = %q, want %q", cp.path, path)
	}
}

func TestS3LoadOrCreateCheckpointResumeCorrupt(t *testing.T) {
	// Reason: Resume=true with a corrupt checkpoint falls back to fresh.
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	path := checkpointPathIn(checkpointDir(), "src", "dst", "")
	os.WriteFile(path, []byte("{invalid"), 0644)
	m := &S3Migration{S3Bucket: "src", R2Bucket: "dst", Resume: true}
	cp := m.loadOrCreateCheckpoint(path, time.Now(), []*S3Object{{Key: "a"}}, 1)
	if cp == nil {
		t.Fatal("checkpoint should not be nil")
	}
	if len(cp.Completed) != 0 {
		t.Error("corrupt checkpoint should start fresh with empty Completed")
	}
}

func TestS3LoadOrCreateCheckpointStaleWarning(t *testing.T) {
	// Reason: Resume=false with an existing checkpoint warns and starts fresh.
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	path := checkpointPathIn(checkpointDir(), "src", "dst", "")
	existing := &Checkpoint{Version: 1, Completed: map[string]int64{"old": 1}}
	if err := saveCheckpoint(existing, path); err != nil {
		t.Fatal(err)
	}
	m := &S3Migration{S3Bucket: "src", R2Bucket: "dst"}
	cp := m.loadOrCreateCheckpoint(path, time.Now(), []*S3Object{{Key: "a"}}, 1)
	if cp == nil {
		t.Fatal("checkpoint should not be nil")
	}
	if _, ok := cp.Completed["old"]; ok {
		t.Error("without --resume a stale checkpoint must not be reused")
	}
}

// --- verifyTransfers / finalizeMigration ------------------------------------

func TestS3VerifyTransfers(t *testing.T) {
	objs := []*S3Object{
		{Key: "match", ETag: `"same"`},
		{Key: "mismatch", ETag: `"s3etag"`},
		{Key: "missing", ETag: `"x"`},
	}
	r2 := newFakeR2()
	r2.headFn = func(key string) (*cosmoflare.HeadResult, error) {
		switch key {
		case "match":
			return &cosmoflare.HeadResult{ETag: `"same"`}, nil
		case "mismatch":
			return &cosmoflare.HeadResult{ETag: `"different"`}, nil
		default:
			return nil, errors.New("not found")
		}
	}
	// s3Client is unused by verifyTransfers; nil is safe.
	m := &S3Migration{R2Bucket: "dst"}
	m.verifyTransfers(nil, r2, objs, &MigrationResult{})
}

func TestS3FinalizeMigrationSuccess(t *testing.T) {
	// Reason: clean run deletes the checkpoint.
	dir := t.TempDir()
	cpPath := filepath.Join(dir, "cp.json")
	cp := &Checkpoint{Version: 1, Completed: map[string]int64{}}
	saveCheckpoint(cp, cpPath)
	m := &S3Migration{R2Bucket: "dst"}
	result := &MigrationResult{}
	err := m.finalizeMigration(nil, newFakeR2(), cpPath, cp, nil, result, newAtomicFalse())
	if err != nil {
		t.Fatalf("finalizeMigration should succeed: %v", err)
	}
	if _, statErr := os.Stat(cpPath); !os.IsNotExist(statErr) {
		t.Error("checkpoint should be deleted on clean finish")
	}
}

func TestS3FinalizeMigrationInterrupted(t *testing.T) {
	// Reason: stopping flag set means finalize reports and returns an error.
	dir := t.TempDir()
	cpPath := filepath.Join(dir, "cp.json")
	cp := &Checkpoint{Version: 1, Completed: map[string]int64{}}
	cp.path = cpPath
	m := &S3Migration{R2Bucket: "dst"}
	stopping := newAtomicTrue()
	err := m.finalizeMigration(nil, newFakeR2(), cpPath, cp, nil, &MigrationResult{TotalObjects: 3}, stopping)
	if err == nil {
		t.Fatal("interrupted migration should return an error")
	}
	if err.Error() != "migration interrupted" {
		t.Errorf("error = %q, want %q", err.Error(), "migration interrupted")
	}
	if _, statErr := os.Stat(cpPath); statErr != nil {
		t.Error("checkpoint must be kept for resume after interruption")
	}
}

func TestS3FinalizeMigrationWithErrors(t *testing.T) {
	// Reason: failed objects keep the checkpoint (for --resume) but finalize
	// itself does not error.
	dir := t.TempDir()
	cpPath := filepath.Join(dir, "cp.json")
	cp := &Checkpoint{Version: 1, Completed: map[string]int64{}}
	cp.path = cpPath
	m := &S3Migration{R2Bucket: "dst"}
	result := &MigrationResult{ErrorCount: 2}
	stopping := newAtomicFalse()
	if err := m.finalizeMigration(nil, newFakeR2(), cpPath, cp, nil, result, stopping); err != nil {
		t.Fatalf("finalizeMigration should not error on object failures: %v", err)
	}
	if _, statErr := os.Stat(cpPath); statErr != nil {
		t.Error("checkpoint must be kept when objects failed")
	}
}

func TestS3FinalizeMigrationVerifyPath(t *testing.T) {
	// Reason: Verify=true on a clean run triggers verifyTransfers (all match).
	dir := t.TempDir()
	cpPath := filepath.Join(dir, "cp.json")
	cp := &Checkpoint{Version: 1, Completed: map[string]int64{}}
	saveCheckpoint(cp, cpPath)
	objs := []*S3Object{{Key: "a", ETag: `"e1"`}}
	r2 := newFakeR2()
	r2.headFn = func(key string) (*cosmoflare.HeadResult, error) {
		return &cosmoflare.HeadResult{ETag: `"e1"`}, nil
	}
	m := &S3Migration{R2Bucket: "dst", Verify: true}
	if err := m.finalizeMigration(nil, r2, cpPath, cp, objs, &MigrationResult{}, newAtomicFalse()); err != nil {
		t.Fatalf("finalizeMigration with verify failed: %v", err)
	}
}

// --- Execute (end-to-end against fake S3) -----------------------------------

func TestS3ExecuteScanError(t *testing.T) {
	// Reason: the fake S3 always 500s listing, so Execute fails fast in
	// scanPhase without transferring anything.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	withTestHome(t)
	withTestAWSEnv(t, server.URL)
	m := &S3Migration{S3Bucket: "src", R2Bucket: "dst", Concurrency: 2}
	res, err := m.Execute(newFakeR2())
	if err == nil {
		t.Fatal("expected Execute to fail when listing fails")
	}
	if res != nil {
		t.Error("result should be nil on scan error")
	}
}

func TestS3ExecuteDryRun(t *testing.T) {
	f := newS3Fixture(t, testObjects(), false)
	withTestHome(t)
	withTestAWSEnv(t, f.server.URL)
	m := &S3Migration{S3Bucket: "src", R2Bucket: "dst", DryRun: true, Concurrency: 2}
	res, err := m.Execute(newFakeR2())
	if err != nil {
		t.Fatalf("Execute dry run failed: %v", err)
	}
	if res.TotalObjects != 3 {
		t.Errorf("TotalObjects = %d, want 3", res.TotalObjects)
	}
	if res.TransferredSize != 600 {
		t.Errorf("TransferredSize = %d, want 600", res.TransferredSize)
	}
}

func TestS3ExecuteEmptyBucket(t *testing.T) {
	// Reason: no objects means Execute returns early with a zero result.
	f := newS3Fixture(t, nil, false)
	withTestHome(t)
	withTestAWSEnv(t, f.server.URL)
	m := &S3Migration{S3Bucket: "src", R2Bucket: "dst", Concurrency: 2}
	res, err := m.Execute(newFakeR2())
	if err != nil {
		t.Fatalf("Execute on empty bucket failed: %v", err)
	}
	if res.TotalObjects != 0 {
		t.Errorf("TotalObjects = %d, want 0", res.TotalObjects)
	}
}

func TestS3ExecuteHappyPath(t *testing.T) {
	f := newS3Fixture(t, testObjects(), false)
	home := withTestHome(t)
	withTestAWSEnv(t, f.server.URL)
	r2 := newFakeR2()
	m := &S3Migration{S3Bucket: "src", R2Bucket: "dst", Concurrency: 2}
	res, err := m.Execute(r2)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if res.SuccessCount != 3 {
		t.Errorf("SuccessCount = %d, want 3", res.SuccessCount)
	}
	if res.ErrorCount != 0 {
		t.Errorf("ErrorCount = %d, want 0", res.ErrorCount)
	}
	if res.TransferredSize != 600 {
		t.Errorf("TransferredSize = %d, want 600", res.TransferredSize)
	}
	if res.SkippedCount != 0 {
		t.Errorf("SkippedCount = %d, want 0", res.SkippedCount)
	}
	r2.mu.Lock()
	defer r2.mu.Unlock()
	if len(r2.uploads) != 3 {
		t.Errorf("uploaded %d objects, want 3", len(r2.uploads))
	}
	if _, statErr := os.Stat(checkpointPathIn(
		filepath.Join(home, ".cosmoflare", "migrations"), "src", "dst", "")); !os.IsNotExist(statErr) {
		t.Error("checkpoint should be deleted after clean migration")
	}
}

func TestS3ExecuteResumeAllComplete(t *testing.T) {
	// Reason: when the checkpoint says everything is done, Execute reports
	// success without transferring and deletes the checkpoint.
	f := newS3Fixture(t, testObjects(), false)
	home := withTestHome(t)
	withTestAWSEnv(t, f.server.URL)
	cpDir := filepath.Join(home, ".cosmoflare", "migrations")
	path := checkpointPathIn(cpDir, "src", "dst", "")
	cp := &Checkpoint{
		Version:         1,
		S3Bucket:        "src",
		R2Bucket:        "dst",
		TotalObjects:    3,
		Completed:       map[string]int64{"a.jpg": 100, "b.jpg": 200, "c.txt": 300},
		TransferredSize: 600,
	}
	if err := saveCheckpoint(cp, path); err != nil {
		t.Fatal(err)
	}
	r2 := newFakeR2()
	m := &S3Migration{S3Bucket: "src", R2Bucket: "dst", Resume: true, Concurrency: 2}
	res, err := m.Execute(r2)
	if err != nil {
		t.Fatalf("Execute resume failed: %v", err)
	}
	if res.SuccessCount != 3 {
		t.Errorf("SuccessCount = %d, want 3 (all already done)", res.SuccessCount)
	}
	// BUG (not fixed, exercising current behavior): the "all objects already
	// transferred" early-return branch in Execute does not populate
	// SkippedCount, so it stays 0 even though every object was skipped.
	if res.SkippedCount != 0 {
		t.Errorf("current behavior: SkippedCount stays 0 in all-complete branch, got %d", res.SkippedCount)
	}
	r2.mu.Lock()
	uploads := len(r2.uploads)
	r2.mu.Unlock()
	if uploads != 0 {
		t.Errorf("no uploads should happen when everything is complete, got %d", uploads)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Error("completed checkpoint should be deleted")
	}
}
