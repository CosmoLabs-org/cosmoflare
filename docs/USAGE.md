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

## D1 Commands

D1 database management for Cloudflare's serverless SQL databases.

### Create a D1 database
```bash
cosmoflare d1 create my-database
cosmoflare d1 create production-db --json
```
JSON output:
```json
{"success":true,"message":"D1 database created","data":{"id":"480f4f69-...","name":"my-database"}}
```

### List D1 databases
```bash
cosmoflare d1 list
cosmoflare d1 list --json
```

### Get D1 database details
```bash
cosmoflare d1 get <database-id>
cosmoflare d1 get <database-id> --json
```

### Delete a D1 database
```bash
cosmoflare d1 delete <database-id>
cosmoflare d1 delete <database-id> --force
```

### Execute SQL query
```bash
cosmoflare d1 query <database-id> --sql="SELECT * FROM users"
cosmoflare d1 query <database-id> --sql="SELECT * FROM users WHERE id = ?1" --param="42"
cosmoflare d1 query <database-id> --sql="CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)" --json
cosmoflare d1 query <database-id> --sql="INSERT INTO users (name) VALUES (?1)" --param="Alice" --json
```
JSON output:
```json
{"success":true,"data":{"columns":["id","name"],"rows":[[1,"Alice"]],"meta":{"changes":0,"duration":0.5}}}
```

| Flag | Description |
|------|-------------|
| `--sql` | SQL query to execute (required) |
| `--param` | Positional query parameter (repeatable for ?1, ?2, ...) |
| `--force` | Skip confirmation prompt (delete only) |

## Email Routing Commands

Email routing configuration for domains. Route incoming emails to destinations based on rules. All email commands are zone-scoped.

### List routing rules
```bash
cosmoflare email rules list <zone-id>
cosmoflare email rules list <zone-id> --json
```

### Get a routing rule
```bash
cosmoflare email rules get <zone-id> <rule-id>
cosmoflare email rules get <zone-id> <rule-id> --json
```

### Create a routing rule
```bash
cosmoflare email rules create <zone-id> --match-to="support@example.com" --forward-to="team@company.com" --name="Support routing"
cosmoflare email rules create <zone-id> --match-all --forward-to="admin@company.com" --name="Forward all"
cosmoflare email rules create <zone-id> --match-to="spam@example.com" --drop --name="Drop spam"
cosmoflare email rules create <zone-id> --match-to="info@example.com" --forward-to="info@company.com" --name="Info" --priority=10 --enabled=false
```
JSON output:
```json
{"success":true,"message":"Email routing rule created","data":{"tag":"rule-abc123","name":"Support routing","enabled":true,"priority":0}}
```

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | (required) | Rule name |
| `--match-to` | | Email address to match |
| `--match-all` | `false` | Match all incoming email |
| `--forward-to` | | Destination email to forward to |
| `--drop` | `false` | Drop matching email instead of forwarding |
| `--priority` | `0` | Rule priority (lower = higher) |
| `--enabled` | `true` | Whether the rule is enabled |

### Update a routing rule
```bash
cosmoflare email rules update <zone-id> <rule-id> --match-to="new@example.com" --forward-to="dest@example.com" --name="Updated"
cosmoflare email rules update <zone-id> <rule-id> --enabled=false
cosmoflare email rules update <zone-id> <rule-id> --priority=5 --json
```

### Delete a routing rule
```bash
cosmoflare email rules delete <zone-id> <rule-id>
cosmoflare email rules delete <zone-id> <rule-id> --force
```

### List destination addresses
```bash
cosmoflare email destinations list <zone-id>
cosmoflare email destinations list <zone-id> --json
```

### Add a destination address
```bash
cosmoflare email destinations add <zone-id> --email="team@company.com"
cosmoflare email destinations add <zone-id> --email="admin@company.com" --json
```
The address receives a verification email that must be confirmed before use.

### Get destination details
```bash
cosmoflare email destinations get <zone-id> <address-id>
cosmoflare email destinations get <zone-id> <address-id> --json
```

### Delete a destination address
```bash
cosmoflare email destinations delete <zone-id> <address-id>
cosmoflare email destinations delete <zone-id> <address-id> --force
```

### Catch-all rule
```bash
cosmoflare email catchall <zone-id>
cosmoflare email catchall <zone-id> --json
cosmoflare email catchall update <zone-id> --forward-to="catchall@example.com"
cosmoflare email catchall update <zone-id> --forward-to="admin@example.com" --json
```

### Email routing settings
```bash
cosmoflare email settings <zone-id>
cosmoflare email settings <zone-id> --json
```

### Enable/disable email routing
```bash
cosmoflare email enable <zone-id>
cosmoflare email disable <zone-id>
```

## Firewall Commands

Firewall rule management for Cloudflare zones. Create and manage rules using Cloudflare filter expressions.

### List firewall rules
```bash
cosmoflare firewall list <zone-id>
cosmoflare firewall list <zone-id> --json
cosmoflare firewall list <zone-id> --json | jq '.[] | select(.action=="block")'
```

### Get firewall rule details
```bash
cosmoflare firewall get <zone-id> <rule-id>
cosmoflare firewall get <zone-id> <rule-id> --json
```

### Create a firewall rule
```bash
cosmoflare firewall create <zone-id> --expression='(ip.src eq 1.2.3.4)' --action=block --description="Block bad IP"
cosmoflare firewall create <zone-id> --expression='(http.request.uri.path contains "/wp-admin")' --action=challenge
cosmoflare firewall create <zone-id> --expression='(cf.threat_score gt 50)' --action=js_challenge --description="Challenge high threat"
cosmoflare firewall create <zone-id> --expression='(ip.geoip.country eq "CN")' --action=block --json
```
JSON output:
```json
{"success":true,"message":"Firewall rule created","data":{"id":"rule-abc123","action":"block","description":"Block bad IP","paused":false}}
```

| Flag | Description |
|------|-------------|
| `--expression` | Cloudflare filter expression (required) |
| `--action` | Rule action: `block`, `challenge`, `js_challenge`, `allow`, `log`, `bypass` (required) |
| `--description` | Human-readable rule description |
| `--priority` | Rule priority (lower = higher priority) |
| `--paused` | Create the rule in a paused state |

### Update a firewall rule
```bash
cosmoflare firewall update <zone-id> <rule-id> --expression='(ip.src eq 5.6.7.8)' --action=block --description="Updated IP"
cosmoflare firewall update <zone-id> <rule-id> --expression='(cf.threat_score gt 30)' --action=challenge --json
```
Both `--expression` and `--action` are required on update (full replacement).

### Delete a firewall rule
```bash
cosmoflare firewall delete <zone-id> <rule-id>
cosmoflare firewall delete <zone-id> <rule-id> --force
```

## WAF Commands

WAF (Web Application Firewall) managed ruleset and IP access rule management. Zone-scoped.

### List WAF packages
```bash
cosmoflare waf packages <zone-id>
cosmoflare waf packages <zone-id> --json
```

### List WAF rules in a package
```bash
cosmoflare waf rules <zone-id> <package-id>
cosmoflare waf rules <zone-id> <package-id> --json
```

### Get or update a WAF rule
```bash
cosmoflare waf rule <zone-id> <package-id> <rule-id>
cosmoflare waf rule <zone-id> <package-id> <rule-id> --mode=block
cosmoflare waf rule <zone-id> <package-id> <rule-id> --mode=simulate --json
```

Valid `--mode` values: `block`, `simulate`, `disable`, `default`, `challenge`.

### List IP access rules
```bash
cosmoflare waf access list <zone-id>
cosmoflare waf access list <zone-id> --json
```

### Create an IP access rule
```bash
cosmoflare waf access create <zone-id> --ip=1.2.3.4 --mode=block
cosmoflare waf access create <zone-id> --ip=192.168.0.0/24 --mode=whitelist --note="Office network"
```

| Flag | Description |
|------|-------------|
| `--ip` | IP address or CIDR range |
| `--mode` | Access rule mode: `block`, `challenge`, `whitelist`, `js_challenge` |
| `--note` | Note for the access rule |

### Delete an IP access rule
```bash
cosmoflare waf access delete <zone-id> <rule-id>
cosmoflare waf access delete <zone-id> <rule-id> --force
```

## CORS Commands

CORS response header management via Cloudflare Transform Rules. No Worker required. Zone-scoped. Requires a Cloudflare Pro plan or higher.

### Show active CORS rules
```bash
cosmoflare cors settings <zone-id>
cosmoflare cors settings <zone-id> --json
```
JSON output:
```json
[{"name":"cosmoflare-cors","allow_origins":["*"],"allow_methods":["GET","POST","OPTIONS"],"allow_credentials":false,"expression":"true"}]
```

### Create or replace a CORS rule
```bash
cosmoflare cors set <zone-id> --origins "*" --methods "GET,POST,OPTIONS"
cosmoflare cors set <zone-id> --origins "https://app.example.com" --credentials --max-age 86400
cosmoflare cors set <zone-id> --origins "*" --expression 'http.request.uri.path matches "^/api/"'
cosmoflare cors set <zone-id> --origins "https://app.example.com" --credentials --json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--origins` | (required) | Comma-separated allowed origins (e.g., `"*"` or `"https://app.example.com"`) |
| `--methods` | `GET, POST, OPTIONS` | Comma-separated HTTP methods |
| `--headers` | `Content-Type, Authorization` | Comma-separated request header names |
| `--max-age` | `86400` | Preflight cache max-age in seconds |
| `--credentials` | `false` | Allow credentials (cannot combine with `--origins "*"`) |
| `--rule-name` | `cosmoflare-cors` | Name/identifier for the rule |
| `--expression` | `true` | Wirefilter expression to scope the rule |

### Remove a CORS rule
```bash
cosmoflare cors remove <zone-id>
cosmoflare cors remove <zone-id> --rule-name my-cors-rule
cosmoflare cors remove <zone-id> --json
```

## Page Rules Commands

Page Rule management for URL-based settings and redirects. Zone-scoped.

### List page rules
```bash
cosmoflare pagerules list <zone-id>
cosmoflare pagerules list <zone-id> --json
cosmoflare pagerules list <zone-id> --json | jq '.[].targets[0].constraint.value'
```

### Get page rule details
```bash
cosmoflare pagerules get <zone-id> <rule-id>
cosmoflare pagerules get <zone-id> <rule-id> --json
```

### Create a page rule
```bash
cosmoflare pagerules create <zone-id> --url="*.example.com/old/*" --action=forwarding_url --action-value="https://example.com/new/$1"
cosmoflare pagerules create <zone-id> --url="example.com/secure/*" --action=always_https --status=active
cosmoflare pagerules create <zone-id> --url="example.com/static/*" --action=cache_level --action-value=cache_everything --priority=2
```
JSON output:
```json
{"success":true,"message":"Page rule created","data":{"id":"rule-abc123","status":"active","priority":1}}
```

| Flag | Default | Description |
|------|---------|-------------|
| `--url` | (required) | URL match pattern (e.g., `*.example.com/path/*`) |
| `--action` | (required) | Action: `forwarding_url`, `always_https`, `cache_level`, `ssl`, `browser_cache_ttl` |
| `--action-value` | | Value for the action (URL for forwarding, cache level, etc.) |
| `--status` | `active` | Rule status: `active` or `disabled` |
| `--priority` | `1` | Rule priority (1 = highest, evaluated first) |

### Update a page rule
```bash
cosmoflare pagerules update <zone-id> <rule-id> --url="*.example.com/new/*" --action=forwarding_url --action-value="https://example.com/latest/$1"
cosmoflare pagerules update <zone-id> <rule-id> --url="example.com/*" --action=always_https --status=disabled
```
All fields are required on update (full replacement).

### Delete a page rule
```bash
cosmoflare pagerules delete <zone-id> <rule-id>
cosmoflare pagerules delete <zone-id> <rule-id> --force
```

## Domains Command

Domain overview with health indicators for all zones in your account.

```bash
cosmoflare domains                        # Overview table of all zones
cosmoflare domains --detail               # Per-domain cards with full info
cosmoflare domains --filter=active        # Only active zones
cosmoflare domains --name="*.com"         # Filter by domain pattern
cosmoflare domains --page=2 --per-page=25 # Pagination
cosmoflare domains --sort=status          # Sort by zone status
cosmoflare domains --enrich               # Add live health probes (slower)
cosmoflare domains --json                 # Machine-readable output
cosmoflare domains --json | jq '.domains[].zone.name'
```
JSON output:
```json
{"domains":[{"zone":{"id":"zone-abc123","name":"example.com","status":"active"},"dns_records":5,"ssl_status":"active"}],"pagination":{"page":1,"per_page":50,"total_count":3}}
```

| Flag | Default | Description |
|------|---------|-------------|
| `--filter` | | Filter by status: `active`, `paused` |
| `--name` | | Filter by domain name pattern (e.g., `*.com`) |
| `--page` | `1` | Page number |
| `--per-page` | `50` | Results per page |
| `--sort` | `name` | Sort by: `name`, `status`, `records` |
| `--detail` | `false` | Show detailed per-domain cards |
| `--enrich` | `false` | Run live health probes (HTTP, SSL) |

## Doctor Command

Deep diagnostic health checks for domains. Runs 4 probes: DNS propagation, SSL certificate, HTTP response, and nameserver consistency.

```bash
cosmoflare doctor example.com              # Full diagnostics
cosmoflare doctor example.com --fix        # Include fix suggestions
cosmoflare doctor example.com --json       # Machine-readable output
cosmoflare doctor --all                    # Run on all domains (slow)
cosmoflare doctor --all --json             # All domains, JSON output
```

Fix suggestions are valid cosmoflare commands that can be executed directly:
```bash
cosmoflare doctor example.com --fix --json | jq '.issues[].fix'
```

JSON output:
```json
{"domain":"example.com","score":95,"probes":{"dns":{"status":"pass"},"ssl":{"status":"pass","expires":"2027-01-15T00:00:00Z"},"http":{"status":"pass","response_time_ms":150},"nameservers":{"status":"pass"}},"issues":[]}
```

| Flag | Description |
|------|-------------|
| `--fix` | Show actionable fix commands for each issue |
| `--all` | Run diagnostics on all domains |

Exit codes: 0=healthy/warning, 1=critical.

## Status Command

At-a-glance infrastructure dashboard showing resource counts and health across all Cloudflare services.

```bash
cosmoflare status                  # Quick dashboard
cosmoflare status --json           # Machine-readable output
cosmoflare status --verbose        # Include per-zone breakdown
```
JSON output:
```json
{"timestamp":"2026-05-29T12:00:00Z","account_id":"abc123","zones":{"count":5},"workers":{"count":3},"kv_namespaces":{"count":2},"r2_buckets":{"count":4},"dns_records":{"count":25},"ssl":{"active":5,"pending":0,"inactive":0},"query_time_ms":450}
```

## Auth Commands

Manage authentication with Cloudflare.

### Login
```bash
cosmoflare auth login                                                  # Interactive login
cosmoflare auth login --token=your_api_token --account-id=your_id     # Non-interactive
cosmoflare auth login --profile=production --interactive              # Create named profile
```

| Flag | Description |
|------|-------------|
| `--token` | API token |
| `--account-id` | Cloudflare account ID |
| `--profile` | Profile name to authenticate |
| `--email` | Account email |
| `--method` | Auth method: `token`, `service-key` |
| `--interactive` | Force interactive mode |
| `--scope` | Token permission scope |

### Rotate tokens
```bash
cosmoflare auth rotate --profile=production
```

### Check auth status
```bash
cosmoflare auth status
cosmoflare auth status --json
```

### Logout
```bash
cosmoflare auth logout
```
Clears in-memory credentials. Profile configurations in `~/.r2go2/config.yaml` are preserved.

## Setup Command

Interactive setup wizard for first-time configuration.

```bash
cosmoflare setup                         # Interactive setup
cosmoflare setup --profile=prod          # Create specific profile
cosmoflare setup --quiet                 # Non-interactive (uses env vars)
cosmoflare setup --switch                # Switch profiles interactively
cosmoflare setup --welcome               # Show welcome message
cosmoflare setup --backup                # Backup profiles
cosmoflare setup --restore               # Restore profiles
```

| Flag | Default | Description |
|------|---------|-------------|
| `--profile` | | Profile name to create |
| `--quiet` | `false` | Non-interactive mode (use environment variables) |
| `--skip-test` | `false` | Skip connection testing |
| `--auto-detect` | `true` | Auto-detect account information |
| `--switch` | `false` | Switch profiles interactively |
| `--welcome` | `false` | Show welcome message for new users |
| `--backup` | `false` | Backup profiles |
| `--restore` | `false` | Restore profiles |

## Backup Command

Secure profile backup with encryption and format options.

```bash
cosmoflare backup                        # Interactive backup with format selection
cosmoflare backup --format=enc           # Create encrypted backup
cosmoflare backup --format=json          # Create plain JSON backup
cosmoflare backup --format=env           # Create environment variable script
```

Supported `--format` values: `enc` (encrypted), `json` (plain JSON), `env` (environment variable script). Backups exclude sensitive API tokens for security.

## Copy Command

Enhanced file copy with progress monitoring, resume capability, and batch support.

```bash
cosmoflare copy source.txt dest.txt                          # Single file copy
cosmoflare copy --recursive source/ dest/                    # Copy directory
cosmoflare copy --progress --verify --resume source.txt dest.txt  # With resume and verification
cosmoflare copy --batch files.txt destination/               # Batch copy from file list
cosmoflare copy --parallel 8 --chunk-size 16MB source/ dest/ # High-performance copy
cosmoflare copy --dry-run --verbose source/ dest/            # Dry run
```

| Flag | Default | Description |
|------|---------|-------------|
| `-r, --recursive` | `false` | Copy directories recursively |
| `--resume` | `false` | Resume interrupted transfers |
| `-V, --verify` | `true` | Verify file integrity after copy |
| `-o, --overwrite` | `false` | Overwrite existing files |
| `-p, --preserve` | `true` | Preserve file attributes |
| `-P, --progress` | `true` | Show progress bars |
| `-q, --quiet` | `false` | Suppress output except errors |
| `-s, --stats` | `false` | Show detailed statistics |
| `-j, --parallel` | `4` | Number of parallel operations |
| `--chunk-size` | `8MB` | Chunk size for large files |
| `-b, --batch` | `false` | Batch operation mode |
| `--format` | `table` | Output format: `table`, `json`, `csv` |

## Completion Command

Generate shell completion scripts for all commands.

```bash
# Bash
source <(cosmoflare completion bash)
cosmoflare completion bash > /etc/bash_completion.d/cosmoflare     # Linux
cosmoflare completion bash > /usr/local/etc/bash_completion.d/cosmoflare  # macOS

# Zsh
cosmoflare completion zsh > "${fpath[1]}/_cosmoflare"

# Fish
cosmoflare completion fish | source
cosmoflare completion fish > ~/.config/fish/completions/cosmoflare.fish

# PowerShell
cosmoflare completion powershell | Out-String | Invoke-Expression
```

Supports dynamic bucket name completion when `CLOUDFLARE_ACCOUNT_ID` and `CLOUDFLARE_API_TOKEN` are set.

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

## Cost Estimation

Estimate monthly Cloudflare costs based on current usage patterns. Provides per-service breakdowns for R2, Workers, and KV.

### Show all estimated costs
```bash
cosmoflare cost                         # Estimated monthly costs for all services
cosmoflare cost --period 7d             # Estimate based on last 7 days
cosmoflare cost --period 90d            # Estimate based on last 90 days
cosmoflare cost --json                  # JSON output
cosmoflare cost --format csv            # CSV output (for spreadsheets)
```

### R2 storage costs
```bash
cosmoflare cost r2                      # R2 storage cost estimate
cosmoflare cost r2 --json               # JSON output
```

Pricing: Storage $0.015/GB/month, Class A ops $4.50/M (PUT, POST, LIST), Class B ops $0.36/M (GET, HEAD).

### Workers costs
```bash
cosmoflare cost workers                 # Workers cost estimate
cosmoflare cost workers --json          # JSON output
```

Pricing: $0.50/M requests after 100K free tier.

### KV costs
```bash
cosmoflare cost kv                      # KV cost estimate
cosmoflare cost kv --json               # JSON output
```

Pricing: Reads $0.50/M, Writes $5.00/M, Storage $0.50/GB/month.

### Itemized breakdown
```bash
cosmoflare cost detail                  # Full itemized breakdown by service
cosmoflare cost detail --json           # JSON output
cosmoflare cost detail --format csv     # CSV for spreadsheet import
```

Flags: `--period` (7d, 30d, 90d; default 30d), `--format` (table, json, csv; default table).

NOTE: Estimates are approximate and based on Cloudflare published pricing. Actual costs may vary based on your plan, contract, and usage patterns.

## Watch Command

Auto-sync a local directory to an R2 bucket when files change. Uses polling-based change detection to upload new/modified files and optionally delete removed files.

### Start watching
```bash
r2go2 watch my-bucket                              # Watch cwd, sync to bucket root
r2go2 watch my-bucket ./dist                        # Watch ./dist directory
r2go2 watch my-bucket ./build --prefix=assets/      # Upload under assets/ prefix
r2go2 watch my-bucket . --exclude="*.log,*.tmp"     # Skip log and tmp files
r2go2 watch my-bucket . --interval=5s               # Poll every 5 seconds
r2go2 watch my-bucket . --delete                    # Sync deletions too
r2go2 watch my-bucket . --dry-run                   # Preview without uploading
r2go2 watch my-bucket . --json                      # NDJSON event stream
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--prefix` | `""` | R2 key prefix (prepended to relative file paths) |
| `--exclude` | none | Glob patterns to exclude (comma-separated) |
| `--interval` | `1s` | Poll interval for change detection |
| `--delete` | `false` | Sync file deletions (remove from R2 when local file is deleted) |

### Output

Human-readable (default):
```
[+] uploaded  src/index.html (1234 bytes)
[~] updated   src/style.css (5678 bytes)
[-] deleted   old/removed.txt
```

NDJSON (--json):
```json
{"time":"2026-01-15T10:30:00Z","type":"uploaded","path":"src/index.html","r2_key":"assets/src/index.html","size":1234}
{"time":"2026-01-15T10:30:01Z","type":"updated","path":"src/style.css","r2_key":"assets/src/style.css","size":5678}
{"time":"2026-01-15T10:30:02Z","type":"deleted","path":"old/removed.txt","r2_key":"assets/old/removed.txt"}
```

The command runs until interrupted with Ctrl+C (SIGINT) or SIGTERM. Hidden directories (`.git`, `.DS_Store`, etc.) are automatically excluded.

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
