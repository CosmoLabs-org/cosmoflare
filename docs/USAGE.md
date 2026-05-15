# R2Go2 Usage Guide

R2Go2 is a CLI tool for managing the full Cloudflare developer platform: R2 (storage), Workers (compute), and KV (key-value). All commands support `--json` for machine-readable output.

## Setup

Set environment variables:
```bash
export CLOUDFLARE_ACCOUNT_ID="your-account-id"
export CLOUDFLARE_API_TOKEN="your-api-token"
```

Or pass via flags: `--account-id` and `--api-token`.

## Global Flags

| Flag | Description |
|------|-------------|
| `--account-id` | Cloudflare Account ID |
| `--api-token` | Cloudflare API token |
| `--dry-run` | Show what would happen without executing |
| `--json` | Output in JSON format |
| `-v, --verbose` | Enable verbose output |

## Bucket Commands

### Create a bucket
```bash
r2go2 bucket create my-bucket
r2go2 bucket create my-bucket --location=eu --tags=env=prod
```
JSON output:
```json
{"success":true,"message":"Bucket created successfully","data":{"name":"my-bucket","created_at":"2026-05-12T..."}}
```

### List buckets
```bash
r2go2 bucket list
r2go2 bucket list --json
r2go2 bucket list --format=csv
r2go2 bucket list --prefix=prod-
```

### Get bucket details
```bash
r2go2 bucket get my-bucket
r2go2 bucket get my-bucket --output json
r2go2 bucket get my-bucket --include-objects
```

### Check if bucket exists
```bash
r2go2 bucket exists my-bucket && echo "exists"
```
Exit codes: 0=exists, 1=not found, 2=error.
JSON output:
```json
{"exists":true,"bucket":"my-bucket"}
```

### Update bucket metadata
```bash
r2go2 bucket update my-bucket --tags=env=staging
```

### Delete a bucket
```bash
r2go2 bucket delete my-bucket
r2go2 bucket delete my-bucket --force
r2go2 bucket delete my-bucket --dry-run
```

### Import buckets from spec
```bash
r2go2 bucket import buckets.json
r2go2 bucket import buckets.yaml --continue
```
JSON output:
```json
{"success":true,"message":"Import complete","data":{"successful":3,"failed":0,"total":3}}
```

## Object Commands

### List objects
```bash
r2go2 object ls my-bucket
r2go2 object ls my-bucket --prefix=images/ --recursive
r2go2 object ls my-bucket --max-keys=10 --json
```

### Upload an object
```bash
r2go2 object put my-bucket file.txt
r2go2 object put my-bucket image.jpg --key=assets/logo.jpg
r2go2 object put my-bucket data.csv --content-type=text/csv --metadata=source=api
```
JSON output:
```json
{"success":true,"message":"Upload successful","data":{"key":"file.txt","bucket":"my-bucket","size":1024,"etag":"abc123","uploaded":"2026-05-12T..."}}
```

Files over 100MB automatically use multipart upload for better throughput.

### Download an object
```bash
r2go2 object get my-bucket file.txt
r2go2 object get my-bucket file.txt --output=local.txt
r2go2 object get my-bucket large.zip --range-start=0 --range-end=1023
```
JSON output:
```json
{"success":true,"message":"Download successful","data":{"key":"file.txt","bucket":"my-bucket","output":"file.txt","size":1024}}
```

Progress bars are shown by default for downloads.

### Get object metadata
```bash
r2go2 object head my-bucket file.txt
r2go2 object head my-bucket file.txt --output json
```

### Delete an object
```bash
r2go2 object delete my-bucket file.txt
```
JSON output:
```json
{"success":true,"message":"Object deleted successfully","data":{"bucket":"my-bucket","key":"file.txt"}}
```

### Copy an object
```bash
r2go2 object copy source-bucket/file.txt dest-bucket/backup.txt
```
JSON output:
```json
{"success":true,"message":"Object copied successfully","data":{"key":"backup.txt","source_key":"file.txt","bucket":"dest-bucket","etag":"def456"}}
```

### Search for objects
```bash
r2go2 object search my-bucket ".jpg"
r2go2 object search my-bucket "image-*" --type=glob
r2go2 object search my-bucket ".*\.png$" --type=regex
```

### Batch operations
```bash
r2go2 object batch my-bucket operations.json
r2go2 object batch my-bucket operations.json --continue --dry-run
```
JSON output:
```json
{"success":true,"message":"Batch operations complete","data":{"successful":5,"failed":0,"total":5}}
```

## Worker Commands

### Deploy a Worker
```bash
r2go2 worker deploy my-worker --script=worker.js
r2go2 worker deploy my-worker --script=worker.js --compatibility-date=2024-01-01
r2go2 worker deploy my-worker --script=worker.js --module --bindings=MY_KV:kv:ns-123 --tags=prod,v2
```
JSON output:
```json
{"success":true,"message":"Worker deployed successfully","data":{"name":"my-worker","size":1024}}
```

### List Workers
```bash
r2go2 worker list
r2go2 worker list --json
```

### Get Worker script
```bash
r2go2 worker get my-worker
r2go2 worker get my-worker --json
```

### Delete a Worker
```bash
r2go2 worker delete my-worker
r2go2 worker delete my-worker --force
```

### View Worker logs
```bash
r2go2 worker logs my-worker
r2go2 worker logs my-worker --limit=50 --json
```

### Update Worker settings
```bash
r2go2 worker settings my-worker --compatibility-date=2024-01-01
r2go2 worker settings my-worker --usage-model=bundled --bindings=MY_R2:r2:my-bucket
```

## KV Commands

### Create a KV namespace
```bash
r2go2 kv namespace create my-cache
r2go2 kv namespace create production-data --json
```
JSON output:
```json
{"success":true,"message":"Namespace created successfully","data":{"id":"ns-abc123","title":"my-cache"}}
```

### List KV namespaces
```bash
r2go2 kv namespace list
r2go2 kv namespace list --json
```

### Delete a KV namespace
```bash
r2go2 kv namespace delete ns-abc123
r2go2 kv namespace delete ns-abc123 --force
```

### Write a key-value pair
```bash
r2go2 kv put ns-abc123 my-key --value="hello world"
r2go2 kv put ns-abc123 config.json --file=config.json
r2go2 kv put ns-abc123 session-123 --value="data" --ttl=3600
```

### Read a key-value pair
```bash
r2go2 kv get ns-abc123 my-key
r2go2 kv get ns-abc123 my-key --json
```

### Delete a key
```bash
r2go2 kv delete ns-abc123 my-key
```

### List keys in a namespace
```bash
r2go2 kv list ns-abc123
r2go2 kv list ns-abc123 --prefix=cache/
r2go2 kv list ns-abc123 --limit=100 --json
```

## Library Usage (Workers and KV)

### Workers

```go
import r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"

// Create a Worker service
ws, err := r2go2.NewWorkerServiceFromCreds("account-id", "api-token")

// Deploy a Worker
script := strings.NewReader("export default { fetch() { return new Response('hello') } }")
worker, err := ws.Deploy(ctx, "my-worker", script,
    r2go2.WithWorkerCompatibilityDate("2024-01-01"),
    r2go2.WithWorkerBindings([]r2go2.WorkerBinding{
        {Name: "MY_KV", Type: "kv", ID: "ns-123"},
    }),
)

// List Workers
workers, err := ws.List(ctx)

// Update settings
err = ws.UpdateSettings(ctx, "my-worker", r2go2.WorkerSettings{
    CompatibilityDate: "2024-06-01",
    UsageModel:        "bundled",
})
```

### KV

```go
// Create a KV service
ks, err := r2go2.NewKVServiceFromCreds("account-id", "api-token")

// Create a namespace
ns, err := ks.CreateNamespace(ctx, "my-cache")

// Write a key
err = ks.Put(ctx, ns.ID, "user:123", strings.NewReader(`{"name":"alice"}`),
    r2go2.WithKVTTL(3600),
)

// Read a key
data, err := ks.Get(ctx, ns.ID, "user:123")

// List keys
keys, err := ks.ListKeys(ctx, ns.ID,
    r2go2.WithKVPrefix("user:"),
    r2go2.WithKVLimit(100),
)
```

## Library Usage (R2 Storage)

Import as a Go library:
```go
import r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"

client, err := r2go2.NewClient(
    r2go2.WithAccountID("..."),
    r2go2.WithAPIToken("..."),
)

// Upload with options
result, err := client.Upload(ctx, "my-bucket", "key.txt", reader, size,
    r2go2.WithContentType("text/plain"),
    r2go2.WithMetadata(map[string]string{"env": "prod"}),
)

// Multipart upload (automatic for files > 100MB)
result, err := client.MultipartUpload(ctx, "my-bucket", "large.bin", reader, size,
    r2go2.WithPartSize(16*1024*1024),
    r2go2.WithConcurrency(5),
    r2go2.WithProgressCallback(func(uploaded, total int64) {
        fmt.Printf("\rProgress: %d/%d", uploaded, total)
    }),
)
```

### Pre-signed URLs

```go
url, err := client.PresignGetObject(ctx, "my-bucket", "file.txt", time.Hour)
fmt.Println("Download URL (expires in 1h):", url)
```

## Pre-signed URLs (CLI)

Generate temporary download URLs without exposing credentials:
```bash
r2go2 object presign my-bucket file.txt
r2go2 object presign my-bucket file.txt --expires=24h
r2go2 object presign my-bucket file.txt --expires=30m --json
```

## Pipe and Stdin Support

Upload from stdin (pipe):
```bash
echo "hello world" | r2go2 object put my-bucket - --key=stdin-data.txt
cat large.json | r2go2 object put my-bucket - --key=data.json --content-type=application/json
```

Download to stdout (pipe):
```bash
r2go2 object get my-bucket file.txt --output=- | gzip > file.gz
r2go2 object get my-bucket file.txt | cat
```

Automatic stdout detection: when stdout is not a terminal (piped), output goes to stdout without `--output=-`.

## Bucket Comparison

Compare objects between two buckets:
```bash
r2go2 compare src-bucket dst-bucket
r2go2 compare src-bucket dst-bucket --prefix=images/
r2go2 compare src-bucket dst-bucket --json
```

Output categories: `only_in_source`, `only_in_dest`, `different_size`, `same`.

## Analytics

Show R2 usage statistics:
```bash
r2go2 analytics
r2go2 analytics --bucket=my-bucket
r2go2 analytics --period=30d --json
```

Displays bucket sizes, object counts, and storage distribution. Note: `--period` is advisory until the Cloudflare Analytics API is integrated.

## Config Profile Commands

Manage named credential profiles for multi-account workflows. Profiles are stored in `~/.r2go2/config.yaml` with 0600 permissions.

### Initialize configuration
```bash
r2go2 config init
```

### List profiles
```bash
r2go2 config list
r2go2 config list --json
```

### Create or update a profile
```bash
r2go2 config set my-profile --account-id=1234567890abcdef1234567890abcdef --api-token=your_token
r2go2 config set prod --account-id=... --api-token=... --description="Production" --region=us-east-1
r2go2 config set staging --account-id=... --api-token=... --endpoint=https://custom.r2.cloudflarestorage.com
```

### Show profile details
```bash
r2go2 config show
r2go2 config show my-profile
r2go2 config show my-profile --show-secrets
```

### Validate a profile
```bash
r2go2 config validate
r2go2 config validate my-profile
```

### Switch active profile
```bash
r2go2 config switch staging
```

### Export profile as environment variables
```bash
eval $(r2go2 config export my-profile)
```

### Delete a profile
```bash
r2go2 config delete old-profile
```
