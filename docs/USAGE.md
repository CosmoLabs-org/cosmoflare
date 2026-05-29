# Cosmoflare Usage Guide

Cosmoflare is a CLI tool for managing the full Cloudflare developer platform: R2 (storage), Workers (compute), KV (key-value), DNS, Zones, SSL/TLS, Cache, and more. All commands support `--json` for machine-readable output.

> **Binary names:** `cosmoflare` is the primary binary name. `r2go2` remains available as a backward-compatible alias and all examples below work with either name.

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

## Dev Server

Start a local development proxy that routes requests to Cloudflare services through your configured credentials.

### Start dev server
```bash
cosmoflare dev                          # All services on port 8787
cosmoflare dev --port 3000              # Custom port
cosmoflare dev --services r2,kv         # Only proxy R2 and KV
cosmoflare dev --profile staging        # Use 'staging' credentials
cosmoflare dev --watch=false            # Disable config hot-reload
cosmoflare dev --json                   # Machine-readable startup events
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8787` | Local port to listen on (matches Wrangler convention) |
| `--watch` | `true` | Watch `.cosmoflare.yaml` for changes and hot-reload |
| `--services` | all | Comma-separated: `r2,kv,workers,dns,zones,ssl,cache,d1,pages,queues` |
| `--profile` | active | Credential profile to use |

### Health check
```bash
curl http://localhost:8787/health
```
JSON output:
```json
{"status":"ok"}
```

### Startup event (--json)
```json
{
  "port": 8787,
  "address": "127.0.0.1:8787",
  "services": ["r2", "kv", "workers", "dns", "zones", "ssl", "cache", "d1", "pages", "queues"],
  "watch": true
}
```

The server shuts down cleanly on SIGINT/SIGTERM.

## Bucket Commands

### Create a bucket
```bash
r2go2 bucket create my-bucket
r2go2 bucket create my-bucket --location=eu --tags=env=prod
r2go2 bucket create my-bucket --metadata=team=platform
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
r2go2 bucket list --tag=env=prod
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
r2go2 bucket update my-bucket --metadata=team=platform
r2go2 bucket update my-bucket --add-tags=v2
r2go2 bucket update my-bucket --remove-tags=deprecated
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

### Follow Worker logs (real-time tailing)
```bash
r2go2 worker logs my-worker --follow
r2go2 worker logs my-worker -f --level=error
r2go2 worker logs my-worker -f --since=15m --interval=5
r2go2 worker logs my-worker -f --json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--follow, -f` | `false` | Continuously poll for new log entries |
| `--interval` | `2` | Polling interval in seconds |
| `--level` | all | Filter by level: `error`, `warn`, `info`, `debug` |
| `--since` | none | Show logs since duration (e.g. `15m`, `1h`) |

`--json` mode streams newline-delimited JSON events. Clean exit on SIGINT.

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

## DNS Commands

DNS record management is zone-scoped. All DNS commands require a `<zone-id>` as the first argument.

### Create a DNS record
```bash
r2go2 dns create <zone-id> --type=A --name=www --content=1.2.3.4
r2go2 dns create <zone-id> --type=A --name=www --content=1.2.3.4 --proxied --ttl=300
r2go2 dns create <zone-id> --type=CNAME --name=blog --content=blog.example.com --comment="Blog subdomain"
r2go2 dns create <zone-id> --type=MX --name=@ --content=mail.example.com --priority=10
```
JSON output:
```json
{"success":true,"message":"DNS record created successfully","data":{"id":"rec-abc123","type":"A","name":"www.example.com","content":"1.2.3.4","proxied":true,"ttl":300}}
```

### List DNS records
```bash
r2go2 dns list <zone-id>
r2go2 dns list <zone-id> --json
r2go2 dns list <zone-id> --type=CNAME
r2go2 dns list <zone-id> --name=www --content=1.2.3.4
```

### Get a DNS record
```bash
r2go2 dns get <zone-id> <record-id>
r2go2 dns get <zone-id> <record-id> --json
```

### Update a DNS record
```bash
r2go2 dns update <zone-id> <record-id> --content=5.6.7.8
r2go2 dns update <zone-id> <record-id> --ttl=300 --proxied --comment="Updated IP" --json
```
JSON output:
```json
{"success":true,"message":"DNS record updated successfully","data":{"id":"rec-abc123","type":"A","name":"www.example.com","content":"5.6.7.8","proxied":true,"ttl":300}}
```

### Delete a DNS record
```bash
r2go2 dns delete <zone-id> <record-id>
r2go2 dns delete <zone-id> <record-id> --force
```
JSON output:
```json
{"success":true,"message":"DNS record deleted successfully","data":{"zone_id":"zone-abc","record_id":"rec-abc123"}}
```

## Zone Commands

Zone management is account-scoped and uses the configured account ID.

### Create a zone
```bash
r2go2 zone create example.com
r2go2 zone create example.com --type=full --json
```
JSON output:
```json
{"success":true,"message":"Zone created successfully","data":{"id":"zone-abc123","name":"example.com","status":"pending","type":"full"}}
```

### List zones
```bash
r2go2 zone list
r2go2 zone list --json
```

### Get zone details
```bash
r2go2 zone get <zone-id>
r2go2 zone get <zone-id> --json
```

### Get zone settings
```bash
r2go2 zone settings <zone-id>
r2go2 zone settings <zone-id> --json
```

### Delete a zone
```bash
r2go2 zone delete <zone-id>
r2go2 zone delete <zone-id> --force
```
JSON output:
```json
{"success":true,"message":"Zone deleted successfully","data":{"id":"zone-abc123"}}
```

## SSL/TLS Commands

SSL/TLS management is zone-scoped. Inspect and configure encryption settings for a zone.

### Check SSL status
```bash
r2go2 ssl status <zone-id>
r2go2 ssl status <zone-id> --json
```
JSON output:
```json
{"success":true,"data":{"zone_id":"zone-abc123","mode":"full","status":"active","certificate_status":"active"}}
```

### Get SSL settings
```bash
r2go2 ssl settings <zone-id>
r2go2 ssl settings <zone-id> --json
```

### Update SSL settings
```bash
r2go2 ssl update <zone-id> --mode=full
r2go2 ssl update <zone-id> --mode=full --min-tls=1.2 --always-https --auto-rewrites --json
```
JSON output:
```json
{"success":true,"message":"SSL settings updated successfully","data":{"zone_id":"zone-abc123","mode":"full","min_tls_version":"1.2","always_use_https":true,"automatic_https_rewrites":true}}
```

Supported `--mode` values: `off`, `flexible`, `full`, `strict` (full strict).

### Verify SSL certificate
```bash
r2go2 ssl verify <zone-id>
r2go2 ssl verify <zone-id> --json
```
JSON output:
```json
{"success":true,"data":{"zone_id":"zone-abc123","certificate_status":"active","issuer":"DigiCert","expires_on":"2027-01-15T00:00:00Z"}}
```

## Cache Commands

Cache management is zone-scoped. Purge cached content and configure caching behavior.

### Purge all cached content
```bash
r2go2 cache purge <zone-id> --all
r2go2 cache purge <zone-id> --all --force
```
JSON output:
```json
{"success":true,"message":"Cache purged successfully","data":{"zone_id":"zone-abc123","purge_type":"all"}}
```

### Purge by URL
```bash
r2go2 cache purge <zone-id> --url=https://example.com/style.css
r2go2 cache purge <zone-id> --url=https://example.com/a.js --url=https://example.com/b.js
```

### Purge by cache tag
```bash
r2go2 cache purge <zone-id> --tag=static
r2go2 cache purge <zone-id> --tag=static --tag=images
```

### Purge by hostname
```bash
r2go2 cache purge <zone-id> --host=example.com
r2go2 cache purge <zone-id> --host=example.com --host=cdn.example.com
```

### Get cache settings
```bash
r2go2 cache settings <zone-id>
r2go2 cache settings <zone-id> --json
```

### Update cache settings
```bash
r2go2 cache settings <zone-id> --browser-ttl=3600 --dev-mode --cache-level=aggressive
```
JSON output:
```json
{"success":true,"message":"Cache settings updated successfully","data":{"zone_id":"zone-abc123","browser_ttl":3600,"development_mode":true,"cache_level":"aggressive"}}
```

Supported `--cache-level` values: `basic`, `simplified`, `aggressive`.

## Pages Commands

Manage Cloudflare Pages projects and deployments.

### Create a Pages project
```bash
r2go2 pages create my-site --branch main
r2go2 pages create my-blog --branch master --json
```

### List Pages projects
```bash
r2go2 pages list
r2go2 pages list --json
```

### Get Pages project details
```bash
r2go2 pages get my-site
r2go2 pages get my-site --json
```

### Delete a Pages project
```bash
r2go2 pages delete my-site
r2go2 pages delete my-site --force
```

### List deployments
```bash
r2go2 pages deployments my-site
r2go2 pages deployments my-site --json
```

## Queue Commands

Manage Cloudflare Queues for message-based communication between Workers.

### Create a queue
```bash
r2go2 queue create my-queue
r2go2 queue create production-events --json
```

### List queues
```bash
r2go2 queue list
r2go2 queue list --json
```

### Get queue details
```bash
r2go2 queue get my-queue
r2go2 queue get my-queue --json
```

### Update (rename) a queue
```bash
r2go2 queue update my-queue --name new-name
```

### Delete a queue
```bash
r2go2 queue delete my-queue
r2go2 queue delete my-queue --force
```

### List consumers
```bash
r2go2 queue consumers my-queue
r2go2 queue consumers my-queue --json
```

## Init Command

Initialize a new Cosmoflare project with a `.cosmoflare.yaml` configuration file.

```bash
cosmoflare init                     # Interactive setup (auto-detects framework)
cosmoflare init --yes               # Non-interactive with defaults
cosmoflare init --template worker   # Worker project template
cosmoflare init --template pages    # Pages static site template
cosmoflare init --template r2       # R2 storage-focused template
cosmoflare init --template full     # All services enabled
cosmoflare init --json              # Machine-readable output
```

Templates: `worker` (default), `pages`, `r2`, `full`. Auto-detection checks for `wrangler.toml` (worker), `go.mod` (full), `package.json` (pages).

## Images Commands

Manage Cloudflare Images — upload, store, resize, and deliver optimized images.

### Upload an image (file)
```bash
r2go2 images upload photo.jpg
r2go2 images upload banner.png --metadata '{"project":"website"}'
r2go2 images upload photo.jpg --require-signed-urls
r2go2 images upload photo.jpg --json
```

### Upload an image (URL)
```bash
r2go2 images upload --url https://example.com/photo.jpg
r2go2 images upload --url https://example.com/img.png --require-signed-urls
r2go2 images upload --url https://example.com/img.png --metadata '{"source":"external"}' --json
```

### List images
```bash
r2go2 images list
r2go2 images list --json
```

### Get image details
```bash
r2go2 images get IMG_ID
r2go2 images get IMG_ID --json
```

### Delete an image
```bash
r2go2 images delete IMG_ID
r2go2 images delete IMG_ID --force
r2go2 images delete IMG_ID --json
```

### List delivery variants
```bash
r2go2 images variants list
r2go2 images variants list --json
```

### Create a delivery variant
```bash
r2go2 images variants create hero --fit=cover --width=1200 --height=630
r2go2 images variants create thumb --fit=cover --width=150 --height=150
r2go2 images variants create avatar --fit=crop --width=100 --height=100 --metadata-mode=none
r2go2 images variants create public --fit=scale-down --width=1920 --height=1080 --never-require-signed-urls
r2go2 images variants create hero --json
```

Fit modes: `scale-down` (default), `contain`, `cover`, `crop`, `pad`.
Metadata modes: `none` (default), `keep`, `copyright`.

### Delete a delivery variant
```bash
r2go2 images variants delete old-variant
r2go2 images variants delete old-variant --force
```
## Hyperdrive Management

Hyperdrive accelerates database connections from Cloudflare Workers. Manage Hyperdrive configs to connect Workers to your existing databases with connection pooling and query caching.

### Create a Hyperdrive Config

```bash
cosmoflare hyperdrive create my-db \
  --origin-host=db.example.com \
  --origin-port=5432 \
  --origin-scheme=postgres \
  --database=mydb \
  --user=admin \
  --password=secret

cosmoflare hyperdrive create staging-db \
  --origin-host=staging.example.com \
  --origin-port=5432 \
  --origin-scheme=postgres \
  --database=staging \
  --user=reader \
  --password=pass123 \
  --json
```

### List Hyperdrive Configs

```bash
cosmoflare hyperdrive list
cosmoflare hyperdrive list --json
```

### Get Hyperdrive Config Details

```bash
cosmoflare hyperdrive get CONFIG_ID
cosmoflare hyperdrive get CONFIG_ID --json
```

### Update a Hyperdrive Config

All origin fields are required for update (the API replaces the entire config).

```bash
cosmoflare hyperdrive update CONFIG_ID \
  --name=new-name \
  --origin-host=db2.example.com \
  --origin-port=5432 \
  --origin-scheme=postgres \
  --database=mydb \
  --user=admin \
  --password=newsecret
```

### Delete a Hyperdrive Config

```bash
cosmoflare hyperdrive delete CONFIG_ID
cosmoflare hyperdrive delete CONFIG_ID --force   # Skip confirmation
```

---

## Diff Commands

Compare your local `.cosmoflare.yaml` configuration against live Cloudflare state. This is a read-only command that never modifies anything.

### Compare all configured services
```bash
r2go2 diff                           # Full diff across all services
r2go2 diff --output=summary          # Show counts only
r2go2 diff --json                    # Structured JSON output
```

### Compare only Workers
```bash
r2go2 diff workers
r2go2 diff workers --json
r2go2 diff workers --output=summary
```

### Compare only DNS records
```bash
r2go2 diff dns
r2go2 diff dns --json
```
Requires `dns.zone_id` to be set in `.cosmoflare.yaml`.

### Compare only KV namespaces
```bash
r2go2 diff kv
r2go2 diff kv --json
```

### Compare only R2 buckets
```bash
r2go2 diff r2
r2go2 diff r2 --json
```

### Output format

In human mode, output uses `+`/`-`/`~` prefixes like git diff:
```
=== workers ===
  + my-new-worker    worker "my-new-worker" exists in config but not deployed
  - old-worker       worker "old-worker" is deployed but not in config

=== dns ===
  ~ A www 1.2.3.4    changes: ttl: 300 -> 600, proxied: false -> true
```

JSON output:
```json
{"success":true,"data":{"results":[{"service":"workers","additions":[{"action":"add","service":"workers","resource":"my-new-worker"}],"deletions":[],"changes":[]}],"total_additions":1,"total_deletions":0,"total_modifications":0,"has_changes":true}}
```

Flags:
- `--output=full` (default) — show detailed per-resource diff lines
- `--output=summary` — show change counts per service only
- `--json` — structured JSON output (works with all subcommands)

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
