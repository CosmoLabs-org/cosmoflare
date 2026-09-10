# Qwen results — operational command surface (wrangler parity + beyond + permissions)

Provenance: asked 2026-09-11 in the same Qwen conversation as the limits
corpus. Prompt (full text in `qwen-prompt-command-coverage.md`): enumerate
every CLI command cosmoflare must cover for D1, KV, Objects (R2), Pages,
Workers etc.; everything wrangler covers; and beyond that everything the
Access API can set for permissions — the project must be aware of all of
them. Captured 2026-09-11T09:20:00+04:00.

---

Below is the **operational command surface** `cosmoflare` should support. It is split into:

1. **Wrangler-compatible commands** — what your wrapper should mirror, proxy, or improve on.
2. **Beyond Wrangler** — Cloudflare REST API capabilities Wrangler does not fully cover.
3. **Permission / Access inventory** — what API token permissions and Access/Zero Trust objects you need to be aware of.
4. **Command metadata model** — how to encode this in `cosmoflare` for limits-aware enforcement.

> Important: lock the exact Wrangler command tree in CI by parsing:
>
> ```bash
> wrangler --help
> wrangler <group> --help
> wrangler <group> <subcommand> --help
> ```
>
> Wrangler changes quickly. The list below is the design-complete surface you should plan for, including stable commands, likely commands, legacy aliases, and API-only gaps.

---

# 1. Wrangler command surface to cover

## 1.1 Global / auth / session / project commands

These are the root-level commands every wrapper must understand.

| Command group | Commands to support | Notes |
|---|---|---|
| Auth | `wrangler login` | Opens browser OAuth flow. Wrapper should also support token-based auth. |
| Auth | `wrangler logout` | Clears stored credentials. |
| Auth | `wrangler whoami` | Shows current user/account identity. Important for permission diagnostics. |
| Docs | `wrangler docs [search]` | Optional but useful for CLI discoverability. |
| Scaffold | `wrangler init [name]` | Creates a new Worker/project. |
| Scaffold | `wrangler create [name]` | Alias/deprecated form of `init` in some versions. |
| Types | `wrangler types` | Generates TypeScript types from `wrangler.toml` / `wrangler.json`. |
| Telemetry | `wrangler telemetry enable` | Verify exact availability. |
| Telemetry | `wrangler telemetry disable` | Verify exact availability. |
| Telemetry | `wrangler telemetry status` | Verify exact availability. |
| Config | `wrangler config list` | Legacy global config. |
| Config | `wrangler config get <key>` | Legacy global config. |
| Config | `wrangler config set <key> <value>` | Legacy global config. |
| Config | `wrangler config delete <key>` | Legacy global config. |

Wrapper requirements:

- Support `--account-id`, `--env`, `--profile`, `--api-token`, `--output json`, `--dry-run` where meaningful.
- Support both `wrangler.toml` and `wrangler.json`/`wrangler.jsonc` configs.
- Support environment inheritance: default env, named envs, vars, secrets, bindings.

---

## 1.2 Workers core lifecycle

| Command | Purpose |
|---|---|
| `wrangler dev` | Local development server. |
| `wrangler deploy` | Deploy Worker script, assets, bindings, migrations, etc. |
| `wrangler publish` | Deprecated alias for `deploy`; support for backward compatibility. |
| `wrangler delete [worker]` | Delete a Worker script. |
| `wrangler tail [worker]` | Stream live Worker logs. |
| `wrangler rollback [deployment-id]` | Roll back to a previous deployment. |
| `wrangler types` | Generate types from config. |

### Deployment commands

| Command | Purpose |
|---|---|
| `wrangler deployments list` | List Worker deployments. |
| `wrangler deployments view [deployment-id]` | View a deployment. |
| `wrangler rollback [deployment-id]` | Roll back to a deployment. |

### Versioned deployment commands

| Command | Purpose |
|---|---|
| `wrangler versions upload` | Upload a new Worker version without necessarily deploying it. |
| `wrangler versions list` | List versions. |
| `wrangler versions view [version-id]` | View a version. |
| `wrangler versions deploy [version-id]` | Deploy a specific version. |
| `wrangler versions delete [version-id]` | Delete a version. |
| `wrangler versions rollback [version-id]` | Roll back to a version. |

Wrapper requirements:

- Understand the difference between:
  - script upload,
  - deployment,
  - version,
  - route attachment,
  - custom domain attachment,
  - static asset upload.
- Enforce:
  - uncompressed Worker size limit,
  - environment variable count/size,
  - static asset count,
  - route count,
  - Worker count per account,
  - Free plan daily request cap behavior.

---

## 1.3 Worker secrets

| Command | Purpose |
|---|---|
| `wrangler secret put <key>` | Create/update a secret. |
| `wrangler secret delete <key>` | Delete a secret. |
| `wrangler secret list` | List secret names, not values. |
| `wrangler secret bulk <file>` | Bulk upload secrets from JSON, if supported in current Wrangler. |

Wrapper requirements:

- Never print secret values unless explicitly requested and permitted.
- Support secret diffing: local manifest vs remote secret names.
- Support per-environment secrets.
- Support safe redaction in logs and error output.

---

## 1.4 Workers KV

Wrangler has both modern and legacy command styles. Support both if possible.

### Modern KV commands

| Command | Purpose |
|---|---|
| `wrangler kv namespace create <namespace>` | Create KV namespace. |
| `wrangler kv namespace delete <namespace>` | Delete KV namespace. |
| `wrangler kv namespace list` | List KV namespaces. |
| `wrangler kv key put <key> [value/file]` | Write a key/value. |
| `wrangler kv key get <key>` | Read a key/value. |
| `wrangler kv key delete <key>` | Delete a key. |
| `wrangler kv key list [prefix]` | List keys. |
| `wrangler kv bulk put <file>` | Bulk write keys from JSON/file. |
| `wrangler kv bulk delete <file>` | Bulk delete keys from JSON/file. |

### Legacy KV aliases

| Command | Purpose |
|---|---|
| `wrangler kv:namespace create <namespace>` | Legacy namespace create. |
| `wrangler kv:namespace delete <namespace>` | Legacy namespace delete. |
| `wrangler kv:namespace list` | Legacy namespace list. |
| `wrangler kv:key put <key> [value/file]` | Legacy key write. |
| `wrangler kv:key get <key>` | Legacy key read. |
| `wrangler kv:key delete <key>` | Legacy key delete. |
| `wrangler kv:key list [prefix]` | Legacy key list. |
| `wrangler kv:bulk put <file>` | Legacy bulk write. |
| `wrangler kv:bulk delete <file>` | Legacy bulk delete. |

Wrapper requirements:

- Resolve namespace by:
  - ID,
  - binding name,
  - preview namespace,
  - environment-specific namespace.
- Enforce:
  - key size max 512 bytes,
  - metadata max 1024 bytes,
  - value size max 25 MiB,
  - same-key write rate 1/sec,
  - Free tier read/write quotas.

---

## 1.5 R2 buckets and objects

Wrangler coverage for R2 is narrower than the full R2/S3 API. Your wrapper should support Wrangler commands and then extend into API/S3 operations.

### R2 bucket commands

| Command | Purpose |
|---|---|
| `wrangler r2 bucket create <bucket>` | Create bucket. |
| `wrangler r2 bucket list` | List buckets. |
| `wrangler r2 bucket delete <bucket>` | Delete bucket. |

### R2 object commands

| Command | Purpose |
|---|---|
| `wrangler r2 object put <bucket>/<key>` | Upload object. |
| `wrangler r2 object get <bucket>/<key>` | Download object. |
| `wrangler r2 object list <bucket>` | List objects. |
| `wrangler r2 object delete <bucket>/<key>` | Delete object. |

Some R2 subcommands may vary by Wrangler version, so verify:

| Possible command | Purpose |
|---|---|
| `wrangler r2 bucket info` | Bucket details, if available. |
| `wrangler r2 bucket event add` | Event notifications, if available. |
| `wrangler r2 bucket event delete` | Event notifications, if available. |
| `wrangler r2 bucket event list` | Event notifications, if available. |
| `wrangler r2 bucket cors` | CORS management, if available. |
| `wrangler r2 bucket lifecycle` | Lifecycle management, if available. |
| `wrangler r2 bucket policy` | Bucket policy management, if available. |

Wrapper requirements:

- Support multipart uploads.
- Enforce:
  - single-part upload max 5 GiB,
  - multipart max 4.995 TiB,
  - max 10,000 parts,
  - object metadata max 8,192 bytes,
  - object key max 1,024 bytes.
- Handle:
  - Cloudflare R2 REST API,
  - S3-compatible API,
  - presigned URLs,
  - Workers bindings.

---

## 1.6 D1

| Command | Purpose |
|---|---|
| `wrangler d1 create <name>` | Create database. |
| `wrangler d1 list` | List databases. |
| `wrangler d1 delete <name>` | Delete database. |
| `wrangler d1 execute <database>` | Run SQL. |
| `wrangler d1 export <database>` | Export database, if supported in current Wrangler. |
| `wrangler d1 migrations create <database> <migration-name>` | Create migration file. |
| `wrangler d1 migrations list <database>` | List migrations. |
| `wrangler d1 migrations apply <database>` | Apply migrations. |
| `wrangler d1 time-travel <database>` | Point-in-time restore, verify exact command name/flags. |

Important `d1 execute` modes:

| Mode/flag | Purpose |
|---|---|
| `--command "<sql>"` | Execute inline SQL. |
| `--file <file.sql>` | Execute SQL file. |
| `--batch <file.json>` | Execute batch statements. |
| `--local` | Run against local development D1. |
| `--remote` | Run against remote production D1. |
| `--json` | Output JSON. |
| `--persist-to <path>` | Persist local database location. |

Wrapper requirements:

- Enforce:
  - max database size,
  - max SQL statement length,
  - max bound parameters,
  - max row/BLOB size,
  - max columns,
  - import file size,
  - time-travel restore rate,
  - queries per Worker invocation.
- Support migration state tracking.
- Warn before destructive SQL: `DROP`, `TRUNCATE`, `DELETE` without `WHERE`, etc.

---

## 1.7 Queues

| Command | Purpose |
|---|---|
| `wrangler queues create <queue>` | Create queue. |
| `wrangler queues delete <queue>` | Delete queue. |
| `wrangler queues list` | List queues. |
| `wrangler queues send <queue>` | Send message, verify exact syntax. |
| `wrangler queues tail <queue>` | Tail queue messages, verify availability. |
| `wrangler queues consumer add <queue> <worker>` | Attach consumer Worker. |
| `wrangler queues consumer remove <queue> <worker>` | Detach consumer Worker. |
| `wrangler queues consumer list` | List consumers, verify availability. |
| `wrangler queues consumer update` | Update consumer settings, verify availability. |

Wrapper requirements:

- Enforce:
  - message size max 128 KB,
  - batch size max 100,
  - `sendBatch` max 100 messages or 256 KB,
  - retention max 14 days,
  - backlog max 25 GB,
  - per-queue throughput 5,000 messages/sec.
- Support DLQ configuration.
- Support retry settings, batch size, max retries, visibility timeout.

---

## 1.8 Pages

| Command | Purpose |
|---|---|
| `wrangler pages project list` | List Pages projects. |
| `wrangler pages project create [name]` | Create Pages project. |
| `wrangler pages project delete [name]` | Delete Pages project. |
| `wrangler pages deploy [directory]` | Deploy static site or output directory. |
| `wrangler pages dev [directory]` | Local Pages development server. |
| `wrangler pages deployment list` | List deployments. |
| `wrangler pages deployment view [deployment-id]` | View deployment, verify availability. |
| `wrangler pages deployment delete [deployment-id]` | Delete deployment. |
| `wrangler pages deployment retry [deployment-id]` | Retry failed deployment, verify availability. |
| `wrangler pages deployment tail [deployment-id]` | Stream deployment/function logs, verify availability. |
| `wrangler pages secret put <key>` | Set Pages secret/env secret, verify availability. |
| `wrangler pages secret delete <key>` | Delete Pages secret, verify availability. |
| `wrangler pages secret list` | List Pages secrets, verify availability. |
| `wrangler pages secret bulk <file>` | Bulk secrets, verify availability. |

Wrapper requirements:

- Enforce:
  - max 100 projects per account,
  - max files per site,
  - max file size 25 MiB,
  - `_headers` rule count,
  - `_redirects` rule count,
  - build concurrency,
  - monthly build quota.
- Support:
  - direct upload,
  - Git-based deployment metadata,
  - preview deployments,
  - custom domains,
  - Pages Functions,
  - environment variables/secrets.

---

## 1.9 Legacy/deprecated commands to tolerate

These should be recognized, normalized, and mapped to modern equivalents.

| Legacy command | Modern equivalent |
|---|---|
| `wrangler publish` | `wrangler deploy` |
| `wrangler build` | Usually handled by build config, not a first-class deploy command. |
| `wrangler config` | Modern config file / auth flows. |
| `wrangler subdomain` | Workers subdomain API. |
| `wrangler kv:namespace` | `wrangler kv namespace` |
| `wrangler kv:key` | `wrangler kv key` |
| `wrangler kv:bulk` | `wrangler kv bulk` |
| `wrangler pages publish` | `wrangler pages deploy` |

---

# 2. Beyond Wrangler: API operations your wrapper must support

Wrangler is not enough for a full Cloudflare developer platform manager. `cosmoflare` should model the following API surfaces as first-class commands.

---

## 2.1 Accounts and organization

| Capability | API operations to support |
|---|---|
| List accounts | `GET /accounts` |
| Get account | `GET /accounts/:account_id` |
| Account settings | `GET/PATCH /accounts/:account_id/settings` |
| Members | list, invite, remove, update roles |
| Roles | list roles, assign roles, remove roles |
| API tokens | list, create, verify, roll, delete |
| Audit logs | list/export account audit logs |
| Billing | subscriptions, plans, usage, invoices |
| Alerts | alert policies, notification channels |
| Notifications | list/create/update notification policies |
| Logpush | jobs, datasets, fields, destinations |
| Usage analytics | Workers, R2, KV, D1, Pages usage metrics |

---

## 2.2 Zones

| Capability | API operations to support |
|---|---|
| List zones | `GET /zones` |
| Create zone | `POST /zones` |
| Get zone | `GET /zones/:zone_id` |
| Delete zone | `DELETE /zones/:zone_id` |
| Zone settings | get/update individual zone settings |
| Activation check | zone activation/recheck |
| Custom nameservers | manage custom NS |
| Vanity nameservers | manage vanity NS where available |
| Zone holds | support organizational zone hold behavior |

Wrapper requirements:

- Understand account-level vs zone-level scope.
- Understand Free/Pro/Business/Enterprise zone plan differences.
- Respect organization limits where applicable.

---

## 2.3 DNS

| Capability | API operations to support |
|---|---|
| List records | `GET /zones/:zone_id/dns_records` |
| Create record | `POST /zones/:zone_id/dns_records` |
| Update record | `PATCH /zones/:zone_id/dns_records/:id` |
| Delete record | `DELETE /zones/:zone_id/dns_records/:id` |
| Import DNS records | zone file import |
| Export DNS records | zone file export |
| DNS settings | secondary DNS, DNSSEC where applicable |
| Batch record changes | where supported by plan |

Wrapper requirements:

- Enforce:
  - per-zone record quotas,
  - Enterprise per-account record quota,
  - batch limits,
  - API rate limits.
- Handle service-created records from Email Routing, Pages, Workers custom domains, etc.

---

## 2.4 Workers script management API

Wrangler covers script deploy, but the wrapper should understand the underlying API.

| Capability | API operations to support |
|---|---|
| List scripts | list Worker scripts in account |
| Get script | fetch script content/metadata |
| Upload script | upload Worker code + metadata |
| Delete script | delete Worker script |
| Script settings | compatibility date/flags, tags, usage model |
| Bindings | plain text, secrets, KV, R2, D1, Queues, Services, AI, Vectorize, Hyperdrive, etc. |
| Routes | list/create/update/delete Worker routes |
| Custom domains | attach/detach Worker custom domains |
| Subdomain | manage account workers subdomain |
| Tails | create/list/delete live tail sessions |
| Deployments | list/view/rollback deployments |
| Versions | upload/list/deploy/delete versions |
| Cron triggers | manage scheduled Workers |
| Script analytics | usage, errors, invocation outcomes |

Wrapper requirements:

- Enforce:
  - script size,
  - number of Workers,
  - env var count/size,
  - route limits,
  - custom domain limits,
  - static asset limits,
  - Free plan request cap.

---

## 2.5 KV API

| Capability | API operations to support |
|---|---|
| List namespaces | account-level namespace list |
| Create namespace | create namespace |
| Delete namespace | delete namespace |
| Namespace details | get namespace metadata |
| List keys | list keys with prefix/limit/cursor |
| Read value | get key value |
| Write value | put key value |
| Write metadata | put key metadata |
| Delete key | delete key |
| Bulk write | bulk put |
| Bulk delete | bulk delete |

Wrapper requirements:

- Support local development namespaces vs remote namespaces.
- Support preview namespaces if used.
- Track eventual consistency semantics for reads after writes.

---

## 2.6 R2 API

Wrangler gives basic bucket/object management. Full coverage requires R2 admin API and S3 API.

### Bucket management

| Capability | API operations to support |
|---|---|
| List buckets | list all buckets |
| Create bucket | create bucket |
| Delete bucket | delete bucket |
| Bucket location/jurisdiction | where supported |
| Bucket custom domains | attach/detach custom domains |
| Bucket CORS | get/put/delete CORS config |
| Bucket lifecycle | get/put/delete lifecycle rules |
| Bucket policy | get/put/delete bucket policy |
| Event notifications | configure/list/delete notifications |
| Public bucket settings | managed public bucket config |
| Bucket analytics/metrics | where available |

### Object operations

| Capability | API operations to support |
|---|---|
| PUT object | single-part upload |
| GET object | download |
| HEAD object | metadata |
| DELETE object | delete |
| LIST objects | list with prefix/delimiter/cursor |
| COPY object | copy object |
| Multipart create | create multipart upload |
| Multipart upload part | upload part |
| Multipart complete | complete multipart upload |
| Multipart abort | abort multipart upload |
| Multipart list parts | list uploaded parts |
| Presigned URLs | generate presigned GET/PUT URLs |

Wrapper requirements:

- Support both Cloudflare REST API and S3-compatible API.
- Respect R2 REST API global rate limit.
- Prefer S3-compatible API for high-throughput object operations.
- Enforce local file-size checks before upload.

---

## 2.7 D1 API

| Capability | API operations to support |
|---|---|
| List databases | list D1 databases |
| Create database | create D1 database |
| Delete database | delete D1 database |
| Database details | size, version, binding info |
| Query database | execute SQL |
| Batch query | execute batched SQL |
| Export database | export SQL dump |
| Time travel | point-in-time restore where available |
| Backups | list/restore backups where available |
| Migrations | track/apply migration state |

Wrapper requirements:

- Distinguish local development D1 vs remote production D1.
- Prevent accidental destructive operations.
- Track row reads/written for billing awareness.
- Enforce SQL statement size and import size.

---

## 2.8 Queues API

| Capability | API operations to support |
|---|---|
| List queues | list queues |
| Create queue | create queue |
| Delete queue | delete queue |
| Queue details | backlog, retention, consumer info |
| Send message | send single message |
| Send batch | send multiple messages |
| Consumers | add/update/remove/list consumers |
| DLQ | configure dead-letter queue |
| Messages | pull/acknowledge where API supports |

Wrapper requirements:

- Enforce message size including internal metadata overhead.
- Enforce batch limits.
- Support consumer Worker binding in `wrangler.toml`.

---

## 2.9 Pages API

| Capability | API operations to support |
|---|---|
| List projects | list Pages projects |
| Create project | create Pages project |
| Delete project | delete Pages project |
| Project settings | build config, env vars, custom domains |
| Deployments | list/view/delete/retry |
| Direct upload | upload assets without Git |
| Deployment logs | view/stream logs where available |
| Functions | manage Pages Functions settings |
| Environment variables | list/create/update/delete |
| Secrets | list/create/update/delete |
| Custom domains | attach/detach domains |
| Usage/analytics | builds, requests, function invocations |

Wrapper requirements:

- Support `_headers` and `_redirects` validation before deploy.
- Enforce file count and file size limits locally.
- Detect whether `PAGES_WRANGLER_MAJOR_VERSION=4` is needed for higher file counts.

---

## 2.10 AI / Workers AI / AI Gateway

| Capability | API operations to support |
|---|---|
| List models | list Workers AI catalog/models |
| Run inference | execute model tasks |
| Task types | text generation, embeddings, ASR, image-to-text, object detection, classification, summarization, translation, text-to-image |
| AI Gateway endpoints | create/list/update/delete gateways |
| Gateway keys | manage gateway API keys where applicable |
| Gateway logs | view logs/analytics |
| Gateway caching/rate limiting | configure gateway behavior |
| Billing/credits | prepaid credits, unified billing where documented |

Wrapper requirements:

- Enforce per-task and per-model rate limits.
- Handle frontier model special limits.
- Support model metadata/context windows where documented.

---

## 2.11 Vectorize API

| Capability | API operations to support |
|---|---|
| List indexes | list Vectorize indexes |
| Create index | create index |
| Delete index | delete index |
| Index details | dimensions, metadata config, namespaces |
| Upsert vectors | batch upsert |
| Query vectors | similarity query with topK/filters |
| Get vector | fetch by ID |
| Delete vector | delete by ID |
| List vectors | paginated listing |
| Namespaces | manage namespaces within index |
| Metadata indexes | configure metadata filtering |

Wrapper requirements:

- Enforce:
  - dimensions,
  - metadata size,
  - vector ID length,
  - batch size,
  - topK limits,
  - upload size.

---

## 2.12 Hyperdrive API

| Capability | API operations to support |
|---|---|
| List configs | list Hyperdrive configurations |
| Create config | create database acceleration config |
| Update config | update origin DB settings |
| Delete config | delete config |
| Connection settings | timeouts, pooling behavior |
| Bindings | bind Hyperdrive to Workers |

Wrapper requirements:

- Do not store database passwords in plain logs.
- Support secret-based origin credentials.
- Enforce config count and connection limits.

---

## 2.13 SSL/TLS and custom hostnames

| Capability | API operations to support |
|---|---|
| Edge certificates | list/get certificate packs |
| Custom certificates | upload/update/delete custom certs |
| Advanced Certificate Manager | ACM certificate management where applicable |
| Universal SSL | settings/status |
| Custom hostnames | create/list/update/delete custom hostnames |
| Custom hostname verification | DCV/status checks |
| Custom hostname certificates | certificate status, fallback certs |
| Minimum TLS version | zone setting |
| TLS ciphers | cipher suite settings where applicable |
| Authenticated origin pulls | global/zone/hostname AOP |

Wrapper requirements:

- Understand certificate pack semantics.
- Track expiry/renewal.
- Respect plan availability.

---

## 2.14 Cache API

| Capability | API operations to support |
|---|---|
| Purge by URL | single-file purge |
| Purge by hostname | hostname purge |
| Purge by tag | cache-tag purge |
| Purge by prefix | prefix purge |
| Purge everything | full zone purge |
| Cache rules | CRUD cache rules |
| Cache key | custom cache key settings |
| TTL settings | edge/browser cache TTL |
| Cache Reserve | settings where applicable |

Wrapper requirements:

- Respect token bucket purge limits.
- Respect max operations per purge request.
- Respect per-account and per-plan shared limits.

---

## 2.15 Rules API

| Capability | API operations to support |
|---|---|
| Page Rules | CRUD page rules |
| Single Redirects | CRUD redirect rules |
| Bulk Redirects | account-level bulk redirects/lists |
| Transform Rules | request/response header, URI rewrite |
| Configuration Rules | zone setting overrides |
| Cache Rules | cache behavior rules |
| Origin Rules | origin/host/header overrides |
| Custom Errors | error page rules |
| Compression Rules | compression behavior |
| Snippets | code snippets where available |
| Rulesets | ruleset APIs for WAF/managed rules |

Wrapper requirements:

- Count disabled rules toward quotas.
- Understand execution order between rule products.
- Validate expression syntax locally where possible.

---

## 2.16 WAF API

| Capability | API operations to support |
|---|---|
| Custom rules | CRUD custom firewall rules |
| Rate limiting rules | CRUD rate limiting rules |
| Managed rulesets | list/enable/configure managed rules |
| Custom rulesets | zone/account custom rulesets |
| Lists | IP lists, bulk lists, redirect lists |
| Firewall events | query firewall analytics/events |
| Expression validation | validate rule expressions |

Wrapper requirements:

- Respect plan-specific rule counts.
- Handle regex availability differences.
- Validate expression complexity where documented.

---

## 2.17 Email Routing / Email Sending

| Capability | API operations to support |
|---|---|
| Email Routing settings | enable/disable zone routing |
| Routing rules | CRUD routing rules |
| Destination addresses | add/verify/list/delete destinations |
| Custom domains | manage sending/routing domains |
| Email Sending | send APIs where available |
| Worker email handlers | bindings and message limits |
| Logs/metrics | delivery/drop metrics |

Wrapper requirements:

- Enforce inbound message size.
- Enforce recipient count.
- Enforce routing rule/destination limits.
- Understand verified destination behavior.

---

## 2.18 Stream / Images / Media

### Stream

| Capability | API operations to support |
|---|---|
| Upload video | direct upload, TUS, creator upload, upload from URL |
| List videos | paginated listing |
| Delete video | delete video |
| Video details | status, duration, playback info |
| Live inputs | RTMP/SRT/WebRTC inputs where applicable |
| Live outputs | simulcast/output config |
| Analytics | delivery/storage metrics |
| Signed URLs | tokenized playback URLs |
| Downloads | MP4 download settings |

### Images

| Capability | API operations to support |
|---|---|
| Upload image | hosted image upload |
| List images | list stored images |
| Delete image | delete stored image |
| Variants | create/list/update/delete variants |
| Transformations | URL-based transformations |
| Images binding | Worker binding operations |
| Analytics | transformation/delivery usage |

Wrapper requirements:

- Enforce upload size limits.
- Enforce transformation uniqueness/billing semantics.
- Understand animated GIF/WebP area limits.

---

## 2.19 Registrar / Domains

| Capability | API operations to support |
|---|---|
| Search domains | availability search |
| Register domain | register supported TLDs |
| Transfer domain | initiate/manage transfers |
| List domains | list registered domains |
| Domain details | status, expiration, locks |
| Contact info | registrant/admin/tech contacts |
| Auto-renew | enable/disable |
| Transfer lock | lock/unlock where supported |
| DNSSEC | enable/disable for registrar domains |
| Renewal | renew domain |

Wrapper requirements:

- Respect TLD-specific constraints.
- Respect max registration term.
- Handle redemption grace periods.

---

## 2.20 Health Checks / Load Balancing / Traffic

| Capability | API operations to support |
|---|---|
| Health checks | CRUD health checks |
| Health check analytics | status/latency/failures |
| Load balancers | CRUD load balancers |
| Pools | CRUD origin pools |
| Monitors | CRUD monitors |
| Steering policies | traffic steering where available |
| Notifications | health alert notifications |

Wrapper requirements:

- Enforce plan availability.
- Enforce check counts and monitoring intervals.

---

## 2.21 Analytics / Logs / GraphQL

| Capability | API operations to support |
|---|---|
| REST analytics | zone/account analytics |
| GraphQL Analytics API | flexible analytics queries |
| Logpush | configure/export logs |
| Workers Tracing | tracing where available |
| Audit logs | account audit events |
| Firewall events | security events |
| R2/D1/KV usage | usage metrics where available |

Wrapper requirements:

- Respect GraphQL query-cost rate limits.
- Respect `Retry-After` on 429.
- Cache analytics responses where appropriate.

---

# 3. Permission / Access inventory

There are two different “permission” systems you need to model:

1. **Cloudflare API token permissions** — what a token can do in the Cloudflare API.
2. **Cloudflare Access / Zero Trust permissions** — who can access protected applications/resources.

You likely need both.

---

# 3A. Cloudflare API token permission inventory

Your wrapper should maintain a permission manifest that maps every command to the minimum token permissions required.

## 3A.1 Account-level permission families

| Permission family | Read | Edit | Used by |
|---|---:|---:|---|
| Account Settings | ✅ | ✅ | account settings, org settings |
| API Tokens | ✅ | ✅ | token management, auth diagnostics |
| Members and Roles | ✅ | ✅ | account member/role management |
| Audit Logs | ✅ | — | audit log export |
| Billing | ✅ | ✅ | plan/subscription/usage |
| Analytics | ✅ | — | usage metrics |
| Alerts | ✅ | ✅ | alert policies |
| Notifications | ✅ | ✅ | notification policies |
| Logpush | ✅ | ✅ | log export jobs |
| D1 | ✅ | ✅ | D1 CRUD/query |
| KV Storage | ✅ | ✅ | Workers KV namespaces/keys |
| Queues | ✅ | ✅ | queue CRUD/messages/consumers |
| R2 Storage | ✅ | ✅ | bucket/object management |
| R2 Admin | ✅ | ✅ | advanced bucket/admin operations, where modeled separately |
| Workers Scripts | ✅ | ✅ | Worker script CRUD |
| Workers Routes | ✅ | ✅ | route attachment |
| Workers Tail | ✅ | — | live Worker logs |
| Workers Tracing | ✅ | ✅ | tracing where available |
| Workers AI | ✅ | ✅ | AI inference/catalog where applicable |
| AI Gateway | ✅ | ✅ | gateway configuration/logs |
| Vectorize | ✅ | ✅ | vector indexes/queries |
| Hyperdrive | ✅ | ✅ | Hyperdrive configs |
| Images | ✅ | ✅ | Images storage/transformations |
| Stream | ✅ | ✅ | Stream video management |
| Registrar | ✅ | ✅ | domain registration/transfer |
| SSL and Certificates | ✅ | ✅ | cert management |
| Rulesets | ✅ | ✅ | WAF/ruleset APIs |
| Access | ✅ | ✅ | Zero Trust Access objects |
| Teams / Zero Trust | ✅ | ✅ | broader Zero Trust settings |
| Turnstile | ✅ | ✅ | Turnstile widgets |
| DNS Firewall / DNS settings | ✅ | ✅ | account DNS features where applicable |
| Cache Purge | — | ✅ | cache purge operations |
| Organizations | ✅ | ✅ | organization-level management |

## 3A.2 Zone-level permission families

| Permission family | Read | Edit | Used by |
|---|---:|---:|---|
| Zone | ✅ | ✅ | zone CRUD/settings |
| Zone Settings | ✅ | ✅ | zone settings updates |
| DNS | ✅ | ✅ | DNS records |
| SSL and Certificates | ✅ | ✅ | zone certificates |
| Custom Hostnames | ✅ | ✅ | custom hostname management |
| Cache Purge | — | ✅ | purge operations |
| Page Rules | ✅ | ✅ | legacy page rules |
| Redirect Rules | ✅ | ✅ | Single/Bulk Redirects |
| Transform Rules | ✅ | ✅ | request/response transforms |
| Configuration Rules | ✅ | ✅ | zone setting overrides |
| Cache Rules | ✅ | ✅ | cache behavior |
| Origin Rules | ✅ | ✅ | origin routing/header overrides |
| Custom Errors | ✅ | ✅ | custom error pages |
| WAF | ✅ | ✅ | custom rules, managed rules |
| Rate Limiting | ✅ | ✅ | rate limiting rules |
| IP Access Rules | ✅ | ✅ | IP access rules |
| Zone Lockdown | ✅ | ✅ | zone lockdown |
| Workers Routes | ✅ | ✅ | route management |
| Workers Scripts | ✅ | ✅ | zone-scoped script operations |
| Workers Tail | ✅ | — | tail logs |
| KV Storage | ✅ | ✅ | KV operations when zone-scoped |
| D1 | ✅ | ✅ | D1 operations where applicable |
| R2 | ✅ | ✅ | R2 where applicable |
| Queues | ✅ | ✅ | queues where applicable |
| Email Routing | ✅ | ✅ | routing rules/destinations |
| Email Security | ✅ | ✅ | email security features |
| Analytics | ✅ | — | zone analytics |
| Logs | ✅ | — | zone logs |
| Load Balancing | ✅ | ✅ | LB pools/monitors |
| Health Checks | ✅ | ✅ | health checks |
| Page Shield | ✅ | ✅ | Page Shield |
| Spectrum | ✅ | ✅ | Spectrum apps |
| Bot Management | ✅ | ✅ | bot settings |
| Waiting Room | ✅ | ✅ | waiting rooms |
| Web Analytics | ✅ | ✅ | web analytics |
| Image Resizing | ✅ | ✅ | image resizing settings |
| Stream | ✅ | ✅ | zone Stream settings where applicable |
| Access | ✅ | ✅ | zone-scoped Access apps/policies |

## 3A.3 User-level permission families

| Permission family | Read | Edit | Used by |
|---|---:|---:|---|
| User Details | ✅ | ✅ | profile info |
| User API Tokens | ✅ | ✅ | personal token management |
| Memberships | ✅ | ✅ | account memberships |
| Sessions | ✅ | ✅ | session management where available |
| Notifications | ✅ | ✅ | user notification settings |

## 3A.4 Recommended least-privilege matrix

| Command group | Minimum permissions |
|---|---|
| `login`, `whoami` | User Details Read |
| `deploy` | Workers Scripts Edit/Upload, Workers Routes Edit if routes, Account Settings Read maybe |
| `delete worker` | Workers Scripts Edit/Delete |
| `tail` | Workers Tail Read |
| `secret put/delete/list` | Workers Scripts Edit or Secrets Edit depending on API model |
| `kv namespace/key/bulk` | Workers KV Storage Edit |
| `r2 bucket create/delete/list` | R2 Storage Edit/Admin |
| `r2 object put/get/list/delete` | R2 Storage Edit |
| `d1 create/delete/execute` | D1 Edit |
| `queues create/delete/send/consumer` | Queues Edit |
| `pages project/deployment` | Pages Edit / Account Pages Edit depending on API model |
| `dns records` | Zone DNS Edit |
| `cache purge` | Zone Cache Purge Edit |
| `rules` | Zone Rules Edit |
| `waf` | Zone WAF Edit / Rulesets Edit |
| `ssl/custom hostnames` | Zone SSL Edit / Custom Hostnames Edit |
| `email routing` | Zone Email Routing Edit |
| `health checks` | Zone Health Checks Edit |
| `registrar` | Account Registrar Edit |
| `analytics/logs` | Analytics Read / Logs Read |
| `API token management` | API Tokens Edit |
| `Access apps/policies` | Access Apps and Policies Edit |

---

# 3B. Cloudflare Access / Zero Trust permission objects

If by “Access API” you mean Cloudflare Access / Zero Trust, these are the objects that grant or restrict access.

## 3B.1 Access organizations

| Object | Operations |
|---|---|
| Access organization | get/update settings |
| Login page settings | customize login page |
| Authentication settings | session duration, MFA policies |
| Audit logs | read/export |
| Seats | add/remove users, seat management |

## 3B.2 Identity providers

| Object | Operations |
|---|---|
| Identity providers | create/list/get/update/delete |
| OAuth providers | configure client ID/secret/scopes |
| SAML providers | metadata, certificates, assertions fields |
| OIDC providers | issuer, client config |
| SCIM | user/group provisioning settings |
| One-time-pin / OTP | email OTP settings |
| GitHub/Google/Okta/Azure AD/etc. | provider-specific settings |

## 3B.3 Groups

| Object | Operations |
|---|---|
| Access groups | create/list/get/update/delete |
| Group membership rules | include/exclude rules |
| Static groups | explicit user assignment |
| Dynamic groups | attribute/email/domain rules |
| Service token groups | group by service token |

## 3B.4 Applications

| Object | Operations |
|---|---|
| Self-hosted apps | create/list/get/update/delete |
| SaaS apps | SaaS app config |
| SSH apps | SSH access apps |
| RDP apps | RDP access apps |
| Domain/zone-level apps | broad domain protection |
| App settings | session duration, cookies, CORS |
| App policies | attach policies |
| Custom pages | block/forbid pages |
| Bypass codes | generate/manage bypass codes where available |

## 3B.5 Policies

| Policy type | Operations |
|---|---|
| Allow policies | create/update/delete |
| Block policies | create/update/delete |
| Bypass policies | create/update/delete |
| Service auth policies | service token authentication |
| MFA policies | require MFA |
| Device posture policies | require device checks |
| Include rules | users, groups, emails, service tokens, IPs |
| Exclude rules | exceptions |
| Require rules | additional requirements |

## 3B.6 Service tokens

| Object | Operations |
|---|---|
| Service tokens | create/list/get/delete |
| Renew service token | rotate/renew |
| Token policies | bind token to Access policies |
| Token expiration | manage expiry |
| Token usage | audit usage |

## 3B.7 Certificates

| Object | Operations |
|---|---|
| mTLS certificates | upload/list/delete |
| Certificate settings | trust settings |
| Short-lived certificates | SSH certificate settings |
| Certificate authorities | configure CA where available |
| Device certificates | device auth certificates |

## 3B.8 Devices and posture

If you want full Zero Trust coverage, include:

| Object | Operations |
|---|---|
| Device profiles | create/update/delete |
| Device posture rules | create/update/delete |
| Device settings | OS requirements, firewall checks |
| Device enrollment | list/revoke devices |
| WARP settings | client settings |

## 3B.9 Access audit and reporting

| Object | Operations |
|---|---|
| Access audit logs | list/export |
| Application usage | analytics |
| Policy hits | inspect decisions |
| Failed logins | security diagnostics |

---

# 4. Command metadata model for `cosmoflare`

Every command in your wrapper should be described by a manifest entry. This will let you enforce limits, permissions, and API behavior consistently.

## 4.1 Suggested command schema

```json
{
  "id": "r2.object.put",
  "cli_path": ["r2", "object", "put"],
  "wrangler_equivalent": "wrangler r2 object put",
  "service": "r2",
  "scope": "bucket/object",
  "verb": "write",
  "api_ops": [
    "PUT /<bucket>/<key>",
    "POST multipart-upload",
    "POST upload-part",
    "POST complete-multipart-upload"
  ],
  "permissions": {
    "account": ["R2 Storage Edit"],
    "zone": [],
    "user": []
  },
  "limits": [
    "r2.upload_single",
    "r2.upload_multipart",
    "r2.object_metadata",
    "r2.api_rate"
  ],
  "local_checks": [
    "file_size",
    "metadata_size",
    "key_length"
  ],
  "api_checks": [
    "bucket_exists",
    "object_count_if_limited",
    "api_rate_remaining"
  ],
  "rate_limit_behavior": {
    "http_status": 429,
    "retry_after": true,
    "bucket": "r2_rest_or_s3"
  },
  "danger_level": "low",
  "destructive": false,
  "trackable": true
}
```

---

# 5. Full command inventory by domain

Below is the consolidated inventory your wrapper should implement.

## 5.1 Auth

```text
cosmoflare auth login
cosmoflare auth logout
cosmoflare auth whoami
cosmoflare auth token create
cosmoflare auth token list
cosmoflare auth token verify
cosmoflare auth token revoke
cosmoflare auth token permissions
```

Wrangler equivalents:

```text
wrangler login
wrangler logout
wrangler whoami
```

---

## 5.2 Workers

```text
cosmoflare workers init
cosmoflare workers create
cosmoflare workers dev
cosmoflare workers deploy
cosmoflare workers delete
cosmoflare workers tail
cosmoflare workers list
cosmoflare workers get
cosmoflare workers rollback
cosmoflare workers deployments list
cosmoflare workers deployments view
cosmoflare workers versions upload
cosmoflare workers versions list
cosmoflare workers versions view
cosmoflare workers versions deploy
cosmoflare workers versions delete
cosmoflare workers versions rollback
cosmoflare workers secret put
cosmoflare workers secret delete
cosmoflare workers secret list
cosmoflare workers secret bulk
cosmoflare workers routes list
cosmoflare workers routes create
cosmoflare workers routes update
cosmoflare workers routes delete
cosmoflare workers domains attach
cosmoflare workers domains detach
cosmoflare workers domains list
cosmoflare workers subdomain get
cosmoflare workers subdomain set
cosmoflare workers cron list
cosmoflare workers cron create
cosmoflare workers cron update
cosmoflare workers cron delete
cosmoflare workers bindings list
cosmoflare workers types generate
```

Wrangler equivalents:

```text
wrangler init
wrangler create
wrangler dev
wrangler deploy
wrangler publish
wrangler delete
wrangler tail
wrangler rollback
wrangler deployments list
wrangler deployments view
wrangler versions upload
wrangler versions list
wrangler versions view
wrangler versions deploy
wrangler versions delete
wrangler versions rollback
wrangler secret put
wrangler secret delete
wrangler secret list
wrangler secret bulk
wrangler types
```

---

## 5.3 KV

```text
cosmoflare kv namespace create
cosmoflare kv namespace delete
cosmoflare kv namespace list
cosmoflare kv key put
cosmoflare kv key get
cosmoflare kv key delete
cosmoflare kv key list
cosmoflare kv bulk put
cosmoflare kv bulk delete
```

Wrangler equivalents:

```text
wrangler kv namespace create
wrangler kv namespace delete
wrangler kv namespace list
wrangler kv key put
wrangler kv key get
wrangler kv key delete
wrangler kv key list
wrangler kv bulk put
wrangler kv bulk delete
```

---

## 5.4 R2 / Objects

```text
cosmoflare r2 bucket create
cosmoflare r2 bucket list
cosmoflare r2 bucket delete
cosmoflare r2 bucket get
cosmoflare r2 bucket cors get
cosmoflare r2 bucket cors set
cosmoflare r2 bucket cors delete
cosmoflare r2 bucket lifecycle get
cosmoflare r2 bucket lifecycle set
cosmoflare r2 bucket lifecycle delete
cosmoflare r2 bucket policy get
cosmoflare r2 bucket policy set
cosmoflare r2 bucket policy delete
cosmoflare r2 bucket domain attach
cosmoflare r2 bucket domain detach
cosmoflare r2 bucket domain list
cosmoflare r2 bucket events list
cosmoflare r2 bucket events create
cosmoflare r2 bucket events delete
cosmoflare r2 object put
cosmoflare r2 object get
cosmoflare r2 object head
cosmoflare r2 object delete
cosmoflare r2 object list
cosmoflare r2 object copy
cosmoflare r2 multipart create
cosmoflare r2 multipart upload-part
cosmoflare r2 multipart complete
cosmoflare r2 multipart abort
cosmoflare r2 multipart list-parts
cosmoflare r2 presign get
cosmoflare r2 presign put
```

Wrangler equivalents:

```text
wrangler r2 bucket create
wrangler r2 bucket list
wrangler r2 bucket delete
wrangler r2 object put
wrangler r2 object get
wrangler r2 object list
wrangler r2 object delete
```

---

## 5.5 D1

```text
cosmoflare d1 database create
cosmoflare d1 database list
cosmoflare d1 database delete
cosmoflare d1 database get
cosmoflare d1 sql execute
cosmoflare d1 sql file
cosmoflare d1 sql batch
cosmoflare d1 export
cosmoflare d1 import
cosmoflare d1 migrations create
cosmoflare d1 migrations list
cosmoflare d1 migrations apply
cosmoflare d1 time-travel restore
cosmoflare d1 backups list
cosmoflare d1 backups restore
```

Wrangler equivalents:

```text
wrangler d1 create
wrangler d1 list
wrangler d1 delete
wrangler d1 execute
wrangler d1 export
wrangler d1 migrations create
wrangler d1 migrations list
wrangler d1 migrations apply
wrangler d1 time-travel
```

---

## 5.6 Queues

```text
cosmoflare queues create
cosmoflare queues list
cosmoflare queues delete
cosmoflare queues get
cosmoflare queues send
cosmoflare queues send-batch
cosmoflare queues consumer add
cosmoflare queues consumer update
cosmoflare queues consumer remove
cosmoflare queues consumer list
cosmoflare queues dlq configure
```

Wrangler equivalents:

```text
wrangler queues create
wrangler queues list
wrangler queues delete
wrangler queues send
wrangler queues consumer add
wrangler queues consumer remove
wrangler queues consumer list
```

---

## 5.7 Pages

```text
cosmoflare pages project create
cosmoflare pages project list
cosmoflare pages project delete
cosmoflare pages project get
cosmoflare pages deploy
cosmoflare pages dev
cosmoflare pages deployment list
cosmoflare pages deployment view
cosmoflare pages deployment delete
cosmoflare pages deployment retry
cosmoflare pages deployment logs
cosmoflare pages deployment tail
cosmoflare pages upload
cosmoflare pages env list
cosmoflare pages env set
cosmoflare pages env delete
cosmoflare pages secret put
cosmoflare pages secret delete
cosmoflare pages secret list
cosmoflare pages secret bulk
cosmoflare pages domain attach
cosmoflare pages domain detach
cosmoflare pages domain list
```

Wrangler equivalents:

```text
wrangler pages project create
wrangler pages project list
wrangler pages project delete
wrangler pages deploy
wrangler pages dev
wrangler pages deployment list
wrangler pages deployment delete
```

---

## 5.8 DNS

```text
cosmoflare dns record list
cosmoflare dns record create
cosmoflare dns record update
cosmoflare dns record delete
cosmoflare dns record import
cosmoflare dns record export
cosmoflare dns zone list
cosmoflare dns zone create
cosmoflare dns zone delete
cosmoflare dns zone get
cosmoflare dns settings get
cosmoflare dns settings update
```

No direct Wrangler equivalent.

---

## 5.9 Zones

```text
cosmoflare zone list
cosmoflare zone create
cosmoflare zone delete
cosmoflare zone get
cosmoflare zone settings list
cosmoflare zone settings get
cosmoflare zone settings update
cosmoflare zone activation check
```

No direct Wrangler equivalent.

---

## 5.10 Cache

```text
cosmoflare cache purge url
cosmoflare cache purge hostname
cosmoflare cache purge tag
cosmoflare cache purge prefix
cosmoflare cache purge everything
cosmoflare cache rules list
cosmoflare cache rules create
cosmoflare cache rules update
cosmoflare cache rules delete
```

No direct Wrangler equivalent.

---

## 5.11 Rules

```text
cosmoflare rules page list
cosmoflare rules page create
cosmoflare rules page update
cosmoflare rules page delete

cosmoflare rules redirect single list
cosmoflare rules redirect single create
cosmoflare rules redirect single update
cosmoflare rules redirect single delete

cosmoflare rules redirect bulk list
cosmoflare rules redirect bulk create
cosmoflare rules redirect bulk update
cosmoflare rules redirect bulk delete

cosmoflare rules transform request list
cosmoflare rules transform request create
cosmoflare rules transform request update
cosmoflare rules transform request delete

cosmoflare rules transform response list
cosmoflare rules transform response create
cosmoflare rules transform response update
cosmoflare rules transform response delete

cosmoflare rules config list
cosmoflare rules config create
cosmoflare rules config update
cosmoflare rules config delete

cosmoflare rules origin list
cosmoflare rules origin create
cosmoflare rules origin update
cosmoflare rules origin delete

cosmoflare rules custom-errors list
cosmoflare rules custom-errors create
cosmoflare rules custom-errors update
cosmoflare rules custom-errors delete
```

No direct Wrangler equivalent.

---

## 5.12 WAF

```text
cosmoflare waf custom-rules list
cosmoflare waf custom-rules create
cosmoflare waf custom-rules update
cosmoflare waf custom-rules delete
cosmoflare waf custom-rules validate

cosmoflare waf rate-limiting list
cosmoflare waf rate-limiting create
cosmoflare waf rate-limiting update
cosmoflare waf rate-limiting delete

cosmoflare waf managed-rulesets list
cosmoflare waf managed-rulesets get
cosmoflare waf managed-rulesets update

cosmoflare waf lists list
cosmoflare waf lists create
cosmoflare waf lists update
cosmoflare waf lists delete
cosmoflare waf lists items add
cosmoflare waf lists items remove
```

No direct Wrangler equivalent.

---

## 5.13 SSL/TLS

```text
cosmoflare ssl certificates list
cosmoflare ssl certificates get
cosmoflare ssl custom-certificates upload
cosmoflare ssl custom-certificates update
cosmoflare ssl custom-certificates delete
cosmoflare ssl universal-ssl get
cosmoflare ssl universal-ssl update
cosmoflare ssl custom-hostnames list
cosmoflare ssl custom-hostnames create
cosmoflare ssl custom-hostnames update
cosmoflare ssl custom-hostnames delete
cosmoflare ssl custom-hostnames verification status
```

No direct Wrangler equivalent.

---

## 5.14 Email Routing

```text
cosmoflare email routing status
cosmoflare email routing enable
cosmoflare email routing disable
cosmoflare email routing rules list
cosmoflare email routing rules create
cosmoflare email routing rules update
cosmoflare email routing rules delete
cosmoflare email routing destinations list
cosmoflare email routing destinations add
cosmoflare email routing destinations verify
cosmoflare email routing destinations delete
```

No direct Wrangler equivalent.

---

## 5.15 Health Checks

```text
cosmoflare healthchecks list
cosmoflare healthchecks create
cosmoflare healthchecks update
cosmoflare healthchecks delete
cosmoflare healthchecks status
cosmoflare healthchecks analytics
```

No direct Wrangler equivalent.

---

## 5.16 Registrar

```text
cosmoflare registrar domains list
cosmoflare registrar domains get
cosmoflare registrar domains search
cosmoflare registrar domains register
cosmoflare registrar domains transfer
cosmoflare registrar domains renew
cosmoflare registrar domains lock
cosmoflare registrar domains unlock
cosmoflare registrar contacts get
cosmoflare registrar contacts update
cosmoflare registrar dnssec enable
cosmoflare registrar dnssec disable
```

No direct Wrangler equivalent.

---

## 5.17 AI / AI Gateway

```text
cosmoflare ai models list
cosmoflare ai run
cosmoflare ai gateway list
cosmoflare ai gateway create
cosmoflare ai gateway update
cosmoflare ai gateway delete
cosmoflare ai gateway logs
cosmoflare ai gateway keys list
cosmoflare ai gateway keys create
cosmoflare ai gateway keys revoke
```

No stable Wrangler equivalent for most of these.

---

## 5.18 Vectorize

```text
cosmoflare vectorize index create
cosmoflare vectorize index list
cosmoflare vectorize index delete
cosmoflare vectorize index get
cosmoflare vectorize vectors upsert
cosmoflare vectorize vectors query
cosmoflare vectorize vectors get
cosmoflare vectorize vectors delete
cosmoflare vectorize vectors list
cosmoflare vectorize namespaces list
```

No stable Wrangler equivalent.

---

## 5.19 Hyperdrive

```text
cosmoflare hyperdrive config create
cosmoflare hyperdrive config list
cosmoflare hyperdrive config get
cosmoflare hyperdrive config update
cosmoflare hyperdrive config delete
```

No stable Wrangler equivalent.

---

## 5.20 Access / Zero Trust

```text
cosmoflare access organization get
cosmoflare access organization update

cosmoflare access idp list
cosmoflare access idp create
cosmoflare access idp update
cosmoflare access idp delete

cosmoflare access groups list
cosmoflare access groups create
cosmoflare access groups update
cosmoflare access groups delete

cosmoflare access apps list
cosmoflare access apps create
cosmoflare access apps update
cosmoflare access apps delete

cosmoflare access policies list
cosmoflare access policies create
cosmoflare access policies update
cosmoflare access policies delete

cosmoflare access service-tokens list
cosmoflare access service-tokens create
cosmoflare access service-tokens renew
cosmoflare access service-tokens delete

cosmoflare access certificates list
cosmoflare access certificates upload
cosmoflare access certificates delete

cosmoflare access seats list
cosmoflare access seats add
cosmoflare access seats remove

cosmoflare access audit-logs list
```

No Wrangler equivalent.

---

## 5.21 API tokens and permissions

```text
cosmoflare api tokens list
cosmoflare api tokens create
cosmoflare api tokens verify
cosmoflare api tokens roll
cosmoflare api tokens delete
cosmoflare api permissions list
cosmoflare api permissions check
cosmoflare api roles list
cosmoflare api roles assign
cosmoflare api roles revoke
```

No direct Wrangler equivalent.

---

# 6. Rate limits and error semantics your wrapper must respect

Your wrapper should not merely call APIs. It must respect Cloudflare’s rate-limit behavior.

## 6.1 Global API rate limits

| Limit | Value | Behavior |
|---|---:|---|
| Client API per user/account token | 1,200 requests / 5 minutes | 429 when exceeded |
| Client API per IP | 200 requests/second | 429 when exceeded |
| GraphQL Analytics API | variable by query cost, max 320 / 5 minutes | 429 when exceeded |

Wrapper behavior:

- Inspect `Retry-After`.
- Inspect `Ratelimit` and `Ratelimit-Policy` headers where present.
- Use exponential backoff with jitter.
- Never hammer on 429.
- Support `--max-retries`, `--retry-backoff`, and `--retry-budget`.

## 6.2 Service-specific rate limits

| Service | Limit to respect |
|---|---|
| R2 REST API | 1,200 requests / 5 minutes across R2 REST operations |
| Cache purge | token bucket limits by plan |
| KV same-key writes | 1 write/sec |
| Queues throughput | 5,000 messages/sec per queue |
| Workers AI | per-task and per-model request/min limits |
| Stream uploads | concurrent encoding queue limits |
| DNS API | endpoint-specific limits, plus global limits |
| Pages builds | concurrency/month quotas |
| D1 time travel | restore operations per time window |

---

# 7. Recommended enforcement model

For each command, classify enforcement as one of:

| Enforcement type | Meaning | Example |
|---|---|---|
| `local` | Checkable before API call | file size, key length, SQL statement size |
| `api-list` | Check by listing resources | number of Workers, buckets, namespaces, projects |
| `api-counter` | Requires analytics/usage endpoint | daily requests, transformations/month |
| `opaque` | Cannot reliably pre-check | same-key write rate, edge throughput |

## Example command guards

| Command | Local guard | API guard |
|---|---|---|
| `r2 object put` | file size vs 5 GiB single-part limit | bucket existence, multipart requirements |
| `kv key put` | key size, metadata size, value size | namespace existence |
| `d1 sql execute` | SQL statement length, file import size | database size, time-travel restore quota |
| `pages deploy` | file count, file size, `_headers`, `_redirects` | project count, build quota |
| `workers deploy` | bundle size, env var count/size | Worker count, route count, daily request cap |
| `cache purge` | URL batch size | purge rate bucket |
| `dns record create` | record payload validity | zone record quota |

---

# 8. Minimum viable `cosmoflare` command map

If you want a practical build order, implement in this order:

## Phase 1: Core developer platform

1. Auth
2. Workers deploy/dev/delete/tail
3. Workers secrets
4. KV namespaces/keys
5. R2 buckets/objects
6. D1 databases/migrations/execute
7. Pages projects/deployments
8. Queues/consumers

## Phase 2: Platform management

1. DNS records
2. Zones
3. Cache purge
4. Rules
5. WAF
6. SSL/custom hostnames
7. Email routing
8. Health checks

## Phase 3: Advanced services

1. Stream
2. Images
3. Vectorize
4. Hyperdrive
5. Workers AI / AI Gateway
6. Registrar
7. Analytics/GraphQL
8. Access/Zero Trust

---

# 9. Exact command-tree checklist

This is the compact checklist you can paste into the repo.

```text
auth login
auth logout
auth whoami
auth token create
auth token list
auth token verify
auth token revoke

workers init
workers create
workers dev
workers deploy
workers delete
workers tail
workers list
workers get
workers rollback
workers deployments list
workers deployments view
workers versions upload
workers versions list
workers versions view
workers versions deploy
workers versions delete
workers versions rollback
workers secret put
workers secret delete
workers secret list
workers secret bulk
workers routes list
workers routes create
workers routes update
workers routes delete
workers domains attach
workers domains detach
workers domains list
workers subdomain get
workers subdomain set
workers cron list
workers cron create
workers cron update
workers cron delete
workers bindings list
workers types generate

kv namespace create
kv namespace delete
kv namespace list
kv key put
kv key get
kv key delete
kv key list
kv bulk put
kv bulk delete

r2 bucket create
r2 bucket list
r2 bucket delete
r2 bucket get
r2 bucket cors get
r2 bucket cors set
r2 bucket cors delete
r2 bucket lifecycle get
r2 bucket lifecycle set
r2 bucket lifecycle delete
r2 bucket policy get
r2 bucket policy set
r2 bucket policy delete
r2 bucket domain attach
r2 bucket domain detach
r2 bucket domain list
r2 bucket events list
r2 bucket events create
r2 bucket events delete
r2 object put
r2 object get
r2 object head
r2 object delete
r2 object list
r2 object copy
r2 multipart create
r2 multipart upload-part
r2 multipart complete
r2 multipart abort
r2 multipart list-parts
r2 presign get
r2 presign put

d1 database create
d1 database list
d1 database delete
d1 database get
d1 sql execute
d1 sql file
d1 sql batch
d1 export
d1 import
d1 migrations create
d1 migrations list
d1 migrations apply
d1 time-travel restore
d1 backups list
d1 backups restore

queues create
queues list
queues delete
queues get
queues send
queues send-batch
queues consumer add
queues consumer update
queues consumer remove
queues consumer list
queues dlq configure

pages project create
pages project list
pages project delete
pages project get
pages deploy
pages dev
pages deployment list
pages deployment view
pages deployment delete
pages deployment retry
pages deployment logs
pages deployment tail
pages upload
pages env list
pages env set
pages env delete
pages secret put
pages secret delete
pages secret list
pages secret bulk
pages domain attach
pages domain detach
pages domain list

dns record list
dns record create
dns record update
dns record delete
dns record import
dns record export
dns zone list
dns zone create
dns zone delete
dns zone get
dns settings get
dns settings update

zone list
zone create
zone delete
zone get
zone settings list
zone settings get
zone settings update
zone activation check

cache purge url
cache purge hostname
cache purge tag
cache purge prefix
cache purge everything
cache rules list
cache rules create
cache rules update
cache rules delete

rules page list
rules page create
rules page update
rules page delete
rules redirect single list
rules redirect single create
rules redirect single update
rules redirect single delete
rules redirect bulk list
rules redirect bulk create
rules redirect bulk update
rules redirect bulk delete
rules transform request list
rules transform request create
rules transform request update
rules transform request delete
rules transform response list
rules transform response create
rules transform response update
rules transform response delete
rules config list
rules config create
rules config update
rules config delete
rules origin list
rules origin create
rules origin update
rules origin delete
rules custom-errors list
rules custom-errors create
rules custom-errors update
rules custom-errors delete

waf custom-rules list
waf custom-rules create
waf custom-rules update
waf custom-rules delete
waf custom-rules validate
waf rate-limiting list
waf rate-limiting create
waf rate-limiting update
waf rate-limiting delete
waf managed-rulesets list
waf managed-rulesets get
waf managed-rulesets update
waf lists list
waf lists create
waf lists update
waf lists delete
waf lists items add
waf lists items remove

ssl certificates list
ssl certificates get
ssl custom-certificates upload
ssl custom-certificates update
ssl custom-certificates delete
ssl universal-ssl get
ssl universal-ssl update
ssl custom-hostnames list
ssl custom-hostnames create
ssl custom-hostnames update
ssl custom-hostnames delete
ssl custom-hostnames verification status

email routing status
email routing enable
email routing disable
email routing rules list
email routing rules create
email routing rules update
email routing rules delete
email routing destinations list
email routing destinations add
email routing destinations verify
email routing destinations delete

healthchecks list
healthchecks create
healthchecks update
healthchecks delete
healthchecks status
healthchecks analytics

registrar domains list
registrar domains get
registrar domains search
registrar domains register
registrar domains transfer
registrar domains renew
registrar domains lock
registrar domains unlock
registrar contacts get
registrar contacts update
registrar dnssec enable
registrar dnssec disable

ai models list
ai run
ai gateway list
ai gateway create
ai gateway update
ai gateway delete
ai gateway logs
ai gateway keys list
ai gateway keys create
ai gateway keys revoke

vectorize index create
vectorize index list
vectorize index delete
vectorize index get
vectorize vectors upsert
vectorize vectors query
vectorize vectors get
vectorize vectors delete
vectorize vectors list
vectorize namespaces list

hyperdrive config create
hyperdrive config list
hyperdrive config get
hyperdrive config update
hyperdrive config delete

access organization get
access organization update
access idp list
access idp create
access idp update
access idp delete
access groups list
access groups create
access groups update
access groups delete
access apps list
access apps create
access apps update
access apps delete
access policies list
access policies create
access policies update
access policies delete
access service-tokens list
access service-tokens create
access service-tokens renew
access service-tokens delete
access certificates list
access certificates upload
access certificates delete
access seats list
access seats add
access seats remove
access audit-logs list

api tokens list
api tokens create
api tokens verify
api tokens roll
api tokens delete
api permissions list
api permissions check
api roles list
api roles assign
api roles revoke
```

---

# 10. What your wrapper must do beyond listing commands

To be production-grade, each command should include:

1. **Permission check**
   - Does the current token have required scopes?
   - Is the resource account-scoped or zone-scoped?

2. **Limits check**
   - Can this be checked locally?
   - Does it require an API list/count call?
   - Is it opaque?

3. **Rate-limit handling**
   - Respect `Retry-After`.
   - Understand per-service buckets.
   - Support backoff.

4. **Dry-run support**
   - Especially for deploy, SQL, purge, delete, and bulk operations.

5. **Auditability**
   - Log command, actor, account, resource, timestamp.
   - Redact secrets.

6. **Idempotency**
   - Support `--if-not-exists`, `--if-exists`, `--force`, etc.

7. **JSON output**
   - Machine-readable output for automation.

8. **Environment awareness**
   - dev/staging/prod environments.
   - local vs remote execution for D1, Workers, Pages.

---

# 11. Final recommendation

Treat the above as the `cosmoflare` command manifest.

The most important design rule is:

> Wrangler compatibility is the baseline, but `cosmoflare` should model Cloudflare as an API platform, not as a wrapper around Wrangler alone.

That means your command registry should contain:

- Wrangler-compatible commands.
- API-only commands.
- Required permissions.
- Rate-limit behavior.
- Limit checks.
- Enforcement class: `local`, `api-list`, `api-counter`, `opaque`.
- Destructiveness level.
- Dry-run support.
- Output schema.

If you do that, the limits-awareness layer can plug directly into the command execution pipeline.