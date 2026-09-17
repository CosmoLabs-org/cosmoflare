package helpers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

// doMockRequest performs an HTTP request against the mock server and returns
// the status code and response body.
func doMockRequest(t *testing.T, method, url string) (int, string) {
	t.Helper()

	req, err := http.NewRequest(method, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, string(body)
}

// decodeMockJSON unmarshals a JSON body into a generic map.
func decodeMockJSON(t *testing.T, body string) map[string]interface{} {
	t.Helper()

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(body), &parsed))
	return parsed
}

// TestMockCloudflareServer validates that the mock Cloudflare API server
// exposes the expected endpoints with the correct methods, status codes, and
// JSON payloads, and rejects unsupported methods with 405.
func TestMockCloudflareServer(t *testing.T) {
	server := MockCloudflareServer()
	t.Cleanup(server.Close)

	base := server.URL

	t.Run("accounts GET returns test account", func(t *testing.T) {
		status, body := doMockRequest(t, http.MethodGet, base+"/accounts")
		require.Equal(t, http.StatusOK, status)

		parsed := decodeMockJSON(t, body)
		require.Equal(t, true, parsed["success"])

		results, ok := parsed["result"].([]interface{})
		require.True(t, ok, "result should be an array")
		require.Len(t, results, 1)

		account, ok := results[0].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "test-account-id", account["id"])
		require.Equal(t, "Test Account", account["name"])
		require.Equal(t, "active", account["status"])
	})

	t.Run("accounts non-GET method is rejected", func(t *testing.T) {
		for _, method := range []string{http.MethodPost, http.MethodDelete, http.MethodPut} {
			status, _ := doMockRequest(t, method, base+"/accounts")
			require.Equal(t, http.StatusMethodNotAllowed, status, "method %s should be rejected", method)
		}
	})

	t.Run("buckets GET lists test bucket", func(t *testing.T) {
		status, body := doMockRequest(t, http.MethodGet, base+"/accounts/test-account-id/r2/buckets")
		require.Equal(t, http.StatusOK, status)

		parsed := decodeMockJSON(t, body)
		require.Equal(t, true, parsed["success"])

		results, ok := parsed["result"].([]interface{})
		require.True(t, ok, "result should be an array")
		require.Len(t, results, 1)

		bucket, ok := results[0].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "test-bucket", bucket["name"])
	})

	t.Run("buckets POST creates new bucket", func(t *testing.T) {
		status, body := doMockRequest(t, http.MethodPost, base+"/accounts/test-account-id/r2/buckets")
		require.Equal(t, http.StatusOK, status)

		parsed := decodeMockJSON(t, body)
		require.Equal(t, true, parsed["success"])

		result, ok := parsed["result"].(map[string]interface{})
		require.True(t, ok, "result should be an object")
		require.Equal(t, "new-bucket", result["name"])
	})

	t.Run("buckets unsupported method is rejected", func(t *testing.T) {
		for _, method := range []string{http.MethodPut, http.MethodPatch} {
			status, _ := doMockRequest(t, method, base+"/accounts/test-account-id/r2/buckets")
			require.Equal(t, http.StatusMethodNotAllowed, status, "method %s should be rejected", method)
		}
	})

	t.Run("bucket DELETE succeeds", func(t *testing.T) {
		status, body := doMockRequest(t, http.MethodDelete, base+"/accounts/test-account-id/r2/buckets/test-bucket")
		require.Equal(t, http.StatusOK, status)

		parsed := decodeMockJSON(t, body)
		require.Equal(t, true, parsed["success"])
	})

	t.Run("bucket non-DELETE method is rejected", func(t *testing.T) {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			status, _ := doMockRequest(t, method, base+"/accounts/test-account-id/r2/buckets/test-bucket")
			require.Equal(t, http.StatusMethodNotAllowed, status, "method %s should be rejected", method)
		}
	})

	t.Run("unknown path returns 404", func(t *testing.T) {
		status, _ := doMockRequest(t, http.MethodGet, base+"/unknown")
		require.Equal(t, http.StatusNotFound, status)
	})

	t.Run("responses are JSON content type", func(t *testing.T) {
		for _, path := range []string{"/accounts", "/accounts/test-account-id/r2/buckets"} {
			resp, err := http.Get(base + path)
			require.NoError(t, err)
			defer resp.Body.Close()
			io.Copy(io.Discard, resp.Body)
			require.Equal(t, "application/json", resp.Header.Get("Content-Type"), "path %s", path)
		}
	})
}

// TestNewMockS3Client validates that the constructor returns a usable client
// with initialized bucket and object maps.
func TestNewMockS3Client(t *testing.T) {
	client := NewMockS3Client()

	require.NotNil(t, client)
	require.NotNil(t, client.Buckets, "Buckets map should be initialized")
	require.NotNil(t, client.Objects, "Objects map should be initialized")
	require.Empty(t, client.Buckets)
	require.Empty(t, client.Objects)

	t.Run("fresh instance per call", func(t *testing.T) {
		other := NewMockS3Client()
		other.Buckets["isolated"] = true
		require.NotContains(t, client.Buckets, "isolated", "instances should not share state")
	})
}

// TestMockS3ClientCreateBucket validates the mocked CreateBucket API: success
// registers the bucket, duplicates return an error, and empty names are
// accepted as-is by the mock.
func TestMockS3ClientCreateBucket(t *testing.T) {
	client := NewMockS3Client()
	ctx := context.Background()

	t.Run("success registers bucket", func(t *testing.T) {
		out, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("my-bucket")})
		require.NoError(t, err)
		require.NotNil(t, out)
		require.Equal(t, "/my-bucket", aws.ToString(out.Location))
		require.True(t, client.Buckets["my-bucket"])
		require.Empty(t, client.Objects["my-bucket"])
	})

	t.Run("duplicate bucket returns error", func(t *testing.T) {
		_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("my-bucket")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "already exists")
	})

	t.Run("second bucket is independent", func(t *testing.T) {
		_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("other-bucket")})
		require.NoError(t, err)
		require.Len(t, client.Buckets, 2)
	})
}

// TestMockS3ClientListBuckets validates the mocked ListBuckets API for the
// empty state and after buckets are created.
func TestMockS3ClientListBuckets(t *testing.T) {
	ctx := context.Background()

	t.Run("empty client lists no buckets", func(t *testing.T) {
		client := NewMockS3Client()
		out, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
		require.NoError(t, err)
		require.Empty(t, out.Buckets)
	})

	t.Run("created buckets are listed", func(t *testing.T) {
		client := NewMockS3Client()
		_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("alpha")})
		require.NoError(t, err)
		_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("beta")})
		require.NoError(t, err)

		out, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
		require.NoError(t, err)
		require.Len(t, out.Buckets, 2)

		names := make(map[string]bool)
		for _, b := range out.Buckets {
			names[aws.ToString(b.Name)] = true
			require.NotNil(t, b.CreationDate, "creation date should be set")
		}
		require.True(t, names["alpha"])
		require.True(t, names["beta"])
	})
}

// TestMockS3ClientDeleteBucket validates the mocked DeleteBucket API: existing
// buckets are removed together with their objects and unknown buckets return
// an error.
func TestMockS3ClientDeleteBucket(t *testing.T) {
	client := NewMockS3Client()
	ctx := context.Background()

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("doomed")})
	require.NoError(t, err)
	_, err = client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String("doomed"), Key: aws.String("k")})
	require.NoError(t, err)

	t.Run("existing bucket is removed", func(t *testing.T) {
		out, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String("doomed")})
		require.NoError(t, err)
		require.NotNil(t, out)
		require.NotContains(t, client.Buckets, "doomed")
		require.NotContains(t, client.Objects, "doomed", "objects should be removed with the bucket")
	})

	t.Run("unknown bucket returns error", func(t *testing.T) {
		_, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String("missing")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})
}

// TestMockS3ClientPutObject validates the mocked PutObject API: keys are
// appended to the bucket, an ETag is returned, and unknown buckets fail.
func TestMockS3ClientPutObject(t *testing.T) {
	client := NewMockS3Client()
	ctx := context.Background()

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("uploads")})
	require.NoError(t, err)

	t.Run("success appends key and returns etag", func(t *testing.T) {
		out, err := client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String("uploads"),
			Key:    aws.String("path/file.txt"),
		})
		require.NoError(t, err)
		require.NotNil(t, out)

		etag := aws.ToString(out.ETag)
		require.NotEmpty(t, etag)
		require.True(t, strings.HasPrefix(etag, "\""), "etag should be quoted")
		require.Equal(t, []string{"path/file.txt"}, client.Objects["uploads"])
	})

	t.Run("multiple keys accumulate in order", func(t *testing.T) {
		_, err := client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String("uploads"), Key: aws.String("second")})
		require.NoError(t, err)
		require.Equal(t, []string{"path/file.txt", "second"}, client.Objects["uploads"])
	})

	t.Run("unknown bucket returns error", func(t *testing.T) {
		_, err := client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String("nope"), Key: aws.String("k")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})
}

// TestMockS3ClientListObjects validates the mocked ListObjects API including
// prefix filtering, empty buckets, and the unknown-bucket error path.
func TestMockS3ClientListObjects(t *testing.T) {
	client := NewMockS3Client()
	ctx := context.Background()

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("data")})
	require.NoError(t, err)
	for _, key := range []string{"a/one", "a/two", "b/three"} {
		_, err := client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String("data"), Key: aws.String(key)})
		require.NoError(t, err)
	}

	t.Run("lists all objects without prefix", func(t *testing.T) {
		out, err := client.ListObjects(ctx, &s3.ListObjectsInput{Bucket: aws.String("data")})
		require.NoError(t, err)
		require.Len(t, out.Contents, 3)

		for _, obj := range out.Contents {
			require.Equal(t, int64(1024), aws.ToInt64(obj.Size))
			require.Equal(t, "STANDARD", string(obj.StorageClass))
			require.NotEmpty(t, aws.ToString(obj.ETag))
		}
	})

	t.Run("filters by prefix", func(t *testing.T) {
		out, err := client.ListObjects(ctx, &s3.ListObjectsInput{
			Bucket: aws.String("data"),
			Prefix: aws.String("a/"),
		})
		require.NoError(t, err)
		require.Len(t, out.Contents, 2)

		for _, obj := range out.Contents {
			require.True(t, strings.HasPrefix(aws.ToString(obj.Key), "a/"))
		}
	})

	t.Run("prefix with no matches returns empty", func(t *testing.T) {
		out, err := client.ListObjects(ctx, &s3.ListObjectsInput{
			Bucket: aws.String("data"),
			Prefix: aws.String("zzz/"),
		})
		require.NoError(t, err)
		require.Empty(t, out.Contents)
	})

	t.Run("empty bucket returns no contents", func(t *testing.T) {
		_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("empty")})
		require.NoError(t, err)

		out, err := client.ListObjects(ctx, &s3.ListObjectsInput{Bucket: aws.String("empty")})
		require.NoError(t, err)
		require.Empty(t, out.Contents)
	})

	t.Run("unknown bucket returns error", func(t *testing.T) {
		_, err := client.ListObjects(ctx, &s3.ListObjectsInput{Bucket: aws.String("ghost")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})
}

// TestMockS3ClientDeleteObject validates the mocked DeleteObject API: the
// matching key is removed exactly once, missing keys are a no-op, and unknown
// buckets return an error.
func TestMockS3ClientDeleteObject(t *testing.T) {
	client := NewMockS3Client()
	ctx := context.Background()

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("del")})
	require.NoError(t, err)
	for _, key := range []string{"keep", "drop", "drop-dupe"} {
		_, err := client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String("del"), Key: aws.String(key)})
		require.NoError(t, err)
	}

	t.Run("removes only the matching key", func(t *testing.T) {
		out, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String("del"), Key: aws.String("drop")})
		require.NoError(t, err)
		require.NotNil(t, out)
		require.Equal(t, []string{"keep", "drop-dupe"}, client.Objects["del"])
	})

	t.Run("missing key is a no-op", func(t *testing.T) {
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String("del"), Key: aws.String("never-existed")})
		require.NoError(t, err)
		require.Equal(t, []string{"keep", "drop-dupe"}, client.Objects["del"])
	})

	t.Run("unknown bucket returns error", func(t *testing.T) {
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String("ghost"), Key: aws.String("k")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})
}

// TestNewMockTerminal validates that the constructor builds a terminal with
// the requested dimensions, a buffered input channel, and empty output.
func TestNewMockTerminal(t *testing.T) {
	term := NewMockTerminal(120, 40)

	require.NotNil(t, term)
	require.Equal(t, 120, term.Width)
	require.Equal(t, 40, term.Height)
	require.NotNil(t, term.Input)
	require.Empty(t, term.Output)

	t.Run("zero dimensions are preserved", func(t *testing.T) {
		term := NewMockTerminal(0, 0)
		require.Equal(t, 0, term.Width)
		require.Equal(t, 0, term.Height)
	})
}

// TestMockTerminalInputOutput validates SendInput buffering plus the
// CaptureOutput/GetOutput/ClearOutput lifecycle.
func TestMockTerminalInputOutput(t *testing.T) {
	term := NewMockTerminal(80, 24)
	defer term.Close()

	t.Run("send input buffers on channel", func(t *testing.T) {
		term.SendInput("first")
		term.SendInput("second")

		require.Equal(t, "first", <-term.Input)
		require.Equal(t, "second", <-term.Input)
	})

	t.Run("input channel has capacity 100", func(t *testing.T) {
		term := NewMockTerminal(80, 24)
		defer term.Close()

		for i := 0; i < 100; i++ {
			term.SendInput("msg")
		}

		count := 0
		for {
			select {
			case <-term.Input:
				count++
			default:
				require.Equal(t, 100, count)
				return
			}
		}
	})

	t.Run("capture and get output", func(t *testing.T) {
		term.CaptureOutput("line one")
		term.CaptureOutput("line two")

		require.Equal(t, []string{"line one", "line two"}, term.GetOutput())
	})

	t.Run("clear output resets slice", func(t *testing.T) {
		term.CaptureOutput("temp")
		term.ClearOutput()
		require.Empty(t, term.GetOutput())
	})
}

// TestMockTerminalClose validates that Close closes the input channel so
// subsequent receives report the channel as closed.
func TestMockTerminalClose(t *testing.T) {
	term := NewMockTerminal(80, 24)
	term.Close()

	_, ok := <-term.Input
	require.False(t, ok, "input channel should be closed")
}
