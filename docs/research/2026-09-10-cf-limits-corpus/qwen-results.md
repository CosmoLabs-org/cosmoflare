# cosmoflare Limits Catalog

## 1. Workers

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| workers.requests_daily | Daily request cap | quota | requests_per_day | account | 100000 | null | null | opaque | hard | true | https://developers.cloudflare.com/workers/platform/limits/ | 2026-09-10T10:00:00Z | Resets at midnight UTC. Can fail open (bypass) or closed (Error 1027). |
| workers.cpu_time | CPU time per request | size | ms | invocation | 10 | 300000 | 300000 | opaque | soft | true | https://developers.cloudflare.com/workers/platform/limits/ | 2026-09-10T10:00:00Z | Paid defaults to 30s, configurable to 5m. Excludes `await` on I/O. |
| workers.memory | Memory per isolate | size | mb | isolate | 128 | 128 | 128 | opaque | soft | false | https://developers.cloudflare.com/workers/platform/limits/ | 2026-09-10T10:00:00Z | Soft limit: runtime lets in-flight requests complete before spawning a new isolate. |
| workers.subrequests | Subrequests per invocation | quota | count | invocation | 50 | 10000 | 10000 | opaque | hard | true | https://developers.cloudflare.com/workers/platform/limits/ | 2026-09-10T10:00:00Z | Paid configurable up to 10M. Each redirect chain counts. |
| workers.script_size | Worker bundle size (uncompressed) | size | mb | script | 64 | 64 | 64 | local | hard | true | https://developers.cloudflare.com/workers/platform/limits/ | 2026-09-10T10:00:00Z | No compressed size limit. Only uncompressed bundle counts. |
| workers.env_vars | Environment variables per worker | quota | count | script | 64 | 128 | 128 | local | hard | true | https://developers.cloudflare.com/workers/platform/limits/ | 2026-09-10T10:00:00Z | Secrets + text vars combined. Max 5 KB per var. |

## 2. R2

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r2.object_size | Maximum object size | size | tb | object | 5 | 5 | 5 | local | hard | false | https://developers.cloudflare.com/r2/platform/limits/ | 2026-09-10T10:05:00Z | 5 TiB per object. |
| r2.upload_single | Single-part upload max size | size | gb | object | 5 | 5 | 5 | local | hard | false | https://developers.cloudflare.com/r2/platform/limits/ | 2026-09-10T10:05:00Z | 5 GiB. Use multipart for larger files. |
| r2.upload_multipart | Multipart upload max size | size | tb | object | 4.995 | 4.995 | 4.995 | local | hard | false | https://developers.cloudflare.com/r2/platform/limits/ | 2026-09-10T10:05:00Z | 4.995 TiB. Max 10,000 parts per upload. |
| r2.object_metadata | Object metadata size | size | bytes | object | 8192 | 8192 | 8192 | local | hard | false | https://developers.cloudflare.com/r2/platform/limits/ | 2026-09-10T10:05:00Z | 8,192 bytes (8 KB). |
| r2.api_rate | REST API rate limit | rate | requests_per_5min | account | 1200 | 1200 | 1200 | api-counter | hard | true | https://developers.cloudflare.com/r2/platform/limits/ | 2026-09-10T10:05:00Z | Applies only to Cloudflare REST API. S3/Workers APIs have separate/higher limits. |
| r2.concurrent_writes | Concurrent writes to same key | rate | ops_per_second | object | 1 | 1 | 1 | opaque | hard | false | https://developers.cloudflare.com/r2/platform/limits/ | 2026-09-10T10:05:00Z | 1 per second. |

## 3. KV

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| kv.reads_daily | Daily reads | quota | ops_per_day | account | 100000 | null | null | opaque | hard | true | https://developers.cloudflare.com/kv/platform/limits/ | 2026-09-10T10:10:00Z | Resets daily. |
| kv.writes_daily | Daily writes (different keys) | quota | ops_per_day | account | 1000 | null | null | opaque | hard | true | https://developers.cloudflare.com/kv/platform/limits/ | 2026-09-10T10:10:00Z | Resets daily. |
| kv.writes_same_key | Writes to same key | rate | ops_per_second | namespace | 1 | 1 | 1 | opaque | hard | false | https://developers.cloudflare.com/kv/platform/limits/ | 2026-09-10T10:10:00Z | Strict 1/sec limit per key. |
| kv.value_size | Maximum value size | size | mb | object | 25 | 25 | 25 | local | hard | false | https://developers.cloudflare.com/kv/platform/limits/ | 2026-09-10T10:10:00Z | 25 MiB. Metadata max 1024 bytes. |
| kv.key_size | Maximum key size | size | bytes | object | 512 | 512 | 512 | local | hard | false | https://developers.cloudflare.com/kv/platform/limits/ | 2026-09-10T10:10:00Z | 512 bytes. |

## 4. D1

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| d1.db_count | Databases per account | quota | count | account | 10 | 50000 | 50000 | api-list | hard | true | https://developers.cloudflare.com/d1/platform/limits/ | 2026-09-10T10:15:00Z | Paid requires Workers Paid plan. |
| d1.db_size | Maximum database size | size | gb | database | 0.5 | 10 | 10 | opaque | hard | true | https://developers.cloudflare.com/d1/platform/limits/ | 2026-09-10T10:15:00Z | 500 MB free, 10 GB paid. |
| d1.time_travel_restores | Time travel restore ops | rate | ops_per_10min | database | 10 | 10 | 10 | opaque | hard | false | https://developers.cloudflare.com/d1/platform/limits/ | 2026-09-10T10:15:00Z | Max 10 restores per 10 mins. |
| d1.queries_per_invocation | Queries per Worker invocation | quota | count | invocation | 50 | 1000 | 1000 | opaque | hard | true | https://developers.cloudflare.com/d1/platform/limits/ | 2026-09-10T10:15:00Z | Read subrequest limits. |
| d1.import_size | Max file import size (`d1 execute`) | size | gb | database | 5 | 5 | 5 | local | hard | false | https://developers.cloudflare.com/d1/platform/limits/ | 2026-09-10T10:15:00Z | 5 GB max SQL file. |

## 5. Queues

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| queues.message_size | Max message size | size | kb | queue | 128 | 128 | 128 | local | hard | false | https://developers.cloudflare.com/queues/platform/limits/ | 2026-09-10T10:20:00Z | 1 KB = 1000 bytes. Includes ~100 bytes internal overhead. |
| queues.batch_size | Max consumer batch size | quota | count | invocation | 100 | 100 | 100 | local | hard | false | https://developers.cloudflare.com/queues/platform/limits/ | 2026-09-10T10:20:00Z | Max 100 messages. |
| queues.send_batch | Max messages per `sendBatch` | quota | count | invocation | 100 | 100 | 100 | local | hard | false | https://developers.cloudflare.com/queues/platform/limits/ | 2026-09-10T10:20:00Z | Or 256 KB total. |
| queues.retention | Message retention period | size | days | queue | 14 | 14 | 14 | local | hard | false | https://developers.cloudflare.com/queues/platform/limits/ | 2026-09-10T10:20:00Z | Configurable up to 14 days. |
| queues.backlog | Per-queue backlog size | size | gb | queue | 25 | 25 | 25 | opaque | hard | false | https://developers.cloudflare.com/queues/platform/limits/ | 2026-09-10T10:20:00Z | 25 GB max per queue. |

## 6. Pages

| id | name | kind | unit | scope | free | pro | business | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| pages.files | Files per site | quota | count | project | 20000 | 100000 | 100000 | 100000 | api-list | hard | true | https://developers.cloudflare.com/pages/platform/limits/ | 2026-09-10T10:25:00Z | Paid requires `PAGES_WRANGLER_MAJOR_VERSION=4` env var. |
| pages.file_size | Max single file size | size | mb | file | 25 | 25 | 25 | 25 | local | hard | false | https://developers.cloudflare.com/pages/platform/limits/ | 2026-09-10T10:25:00Z | 25 MiB. |
| pages.builds_monthly | Builds per month | quota | count | account | 500 | 5000 | 20000 | null | opaque | hard | true | https://developers.cloudflare.com/pages/platform/limits/ | 2026-09-10T10:25:00Z | Resets monthly. |
| pages.redirects | Max redirects in `_redirects` | quota | count | project | 2100 | 2100 | 2100 | 2100 | local | hard | false | https://developers.cloudflare.com/pages/platform/limits/ | 2026-09-10T10:25:00Z | 2,000 static + 100 dynamic. |

## 7. Images

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| images.transformations | Unique transformations/month | quota | ops_per_month | account | 5000 | null | null | opaque | hard | true | https://developers.cloudflare.com/images/pricing/ | 2026-09-10T10:30:00Z | Billed per *unique* combination of parameters per calendar month. Free returns Error 9422 when exceeded. |
| images.hosted_size | Hosted image file size | size | mb | object | 10 | 10 | 10 | local | hard | false | https://developers.cloudflare.com/images/get-started/limits/ | 2026-09-10T10:30:00Z | 10 MB max upload. Remote images allow 100 MB. |
| images.animated_area | Animated GIF/WebP area | size | mp | object | 100 | 100 | 100 | opaque | soft | false | https://developers.cloudflare.com/images/get-started/limits/ | 2026-09-10T10:30:00Z | 100 MP max total area across frames. Transformations ignored if > 50 MP. |

## 8. Stream

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| stream.upload_size | Max video upload size | size | gb | object | 30 | 30 | 30 | local | hard | false | https://developers.cloudflare.com/stream/faq/ | 2026-09-10T10:35:00Z | 30 GB max file size. |
| stream.concurrent_encoding | Concurrent queued/encoding videos | quota | count | account | 120 | 120 | 120 | opaque | hard | false | https://developers.cloudflare.com/stream/faq/ | 2026-09-10T10:35:00Z | Returns 429 if exceeded. Contact support to raise. |

## 9. Vectorize

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| vectorize.indexes | Indexes per account | quota | count | account | 100 | 50000 | 50000 | api-list | hard | true | https://developers.cloudflare.com/vectorize/platform/limits/ | 2026-09-10T10:40:00Z | |
| vectorize.dimensions | Max dimensions per vector | size | dimensions | index | 1536 | 1536 | 1536 | local | hard | false | https://developers.cloudflare.com/vectorize/platform/limits/ | 2026-09-10T10:40:00Z | 32 bits precision. |
| vectorize.metadata | Metadata per vector | size | kb | object | 10 | 10 | 10 | local | hard | false | https://developers.cloudflare.com/vectorize/platform/limits/ | 2026-09-10T10:40:00Z | 10 KiB. |
| vectorize.upsert_batch | Max upsert batch size (Workers) | quota | count | invocation | 1000 | 1000 | 1000 | local | hard | false | https://developers.cloudflare.com/vectorize/platform/limits/ | 2026-09-10T10:40:00Z | HTTP API allows 5000. |
| vectorize.vectors_per_index | Max vectors per index | quota | count | index | 20000000 | 20000000 | 20000000 | opaque | hard | true | https://developers.cloudflare.com/vectorize/platform/limits/ | 2026-09-10T10:40:00Z | 20M vectors. |

## 10. Workers AI + AI Gateway

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| workers_ai.text_gen_rate | Text generation rate limit (default) | rate | ops_per_minute | account | 300 | 300 | 300 | opaque | hard | true | https://developers.cloudflare.com/workers-ai/platform/limits/ | 2026-09-10T10:45:00Z | Varies by model. Frontier models (e.g., Kimi) are 20 req/min standard, 50 req/min with prepaid Gateway credits. |
| workers_ai.embeddings_rate | Text embeddings rate limit | rate | ops_per_minute | account | 3000 | 3000 | 3000 | opaque | hard | true | https://developers.cloudflare.com/workers-ai/platform/limits/ | 2026-09-10T10:45:00Z | BGE-large is 1500 req/min. |

## 11. Hyperdrive

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| hyperdrive.configs | Max configured databases | quota | count | account | 10 | 25 | 25 | api-list | hard | true | https://developers.cloudflare.com/hyperdrive/platform/limits/ | 2026-09-10T10:50:00Z | |
| hyperdrive.origin_conns | Max origin DB connections per config | quota | count | database | 20 | 100 | 100 | opaque | hard | false | https://developers.cloudflare.com/hyperdrive/platform/limits/ | 2026-09-10T10:50:00Z | ~20 free, ~100 paid. |

## 12. DNS

| id | name | kind | unit | scope | free | pro | business | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| dns.records_zone | DNS records per zone | quota | count | zone | 200 | 3500 | 3500 | null | api-list | hard | true | https://developers.cloudflare.com/dns/manage-dns-records/ | 2026-09-10T10:55:00Z | Free zones created before 2024-09-01 get 1,000. Email Routing records get a small buffer. |
| dns.records_account | DNS records per account (Enterprise) | quota | count | account | null | null | null | 1000000 | api-list | hard | true | https://developers.cloudflare.com/dns/manage-dns-records/ | 2026-09-10T10:55:00Z | Public and internal zones counted separately. |

## 13. Zones

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| zones.organization | Zones per Organization | quota | count | account | null | null | 5000 | api-list | hard | true | https://developers.cloudflare.com/fundamentals/organizations/limitations/ | 2026-09-10T11:00:00Z | Standard accounts have undocumented soft limits (typically ~50). Organizations (Ent/MSSP) have 5,000. |

## 14. SSL/TLS

| id | name | kind | unit | scope | free | pro | business | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| ssl.custom_certs | Custom certificate packs | quota | count | zone | 0 | 0 | 6 | 6 | api-list | hard | true | https://developers.cloudflare.com/ssl/edge-certificates/custom-certificates/ | 2026-09-10T11:05:00Z | Biz/Ent get 5 Modern + 1 Legacy pack. Each pack can hold 3 certs (RSA/ECDSA/SHA1). |
| ssl.custom_hostnames | Custom hostnames (SaaS Pay-as-you-go) | quota | count | zone | null | null | null | 50000 | api-list | hard | true | https://developers.cloudflare.com/ssl/changelog/ | 2026-09-10T11:05:00Z | Raised from 5,000 in 2026. |

## 15. Cache

| id | name | kind | unit | scope | free | pro | business | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| cache.purge_rate | Purge requests rate | rate | requests_per_sec | account | 0.083 | 5 | 10 | 50 | opaque | soft | true | https://developers.cloudflare.com/cache/how-to/purge-cache/ | 2026-09-10T11:10:00Z | Free is 5/min. Token bucket based. Shared across zones of same plan. |
| cache.purge_batch_urls | Max URLs per single-file purge req | quota | count | request | 100 | 100 | 100 | 500 | local | hard | false | https://developers.cloudflare.com/cache/how-to/purge-cache/ | 2026-09-10T11:10:00Z | |
| cache.file_size_max | Max cacheable file size | size | mb | object | 512 | 512 | 512 | 5120 | opaque | hard | false | https://developers.cloudflare.com/cache/concepts/default-cache-behavior/ | 2026-09-10T11:10:00Z | 512 MB (Free/Pro/Biz), 5 GB (Ent). |
| cache.tags_header_size | Max Cache-Tag header size | size | kb | request | 16 | 16 | 16 | 16 | local | hard | false | https://developers.cloudflare.com/cache/how-to/purge-cache/purge-by-tags/ | 2026-09-10T11:10:00Z | ~1,000 unique tags. |
| cache.edge_ttl_min | Minimum Edge Cache TTL | size | s | zone | 7200 | 3600 | 1 | 1 | local | hard | false | https://developers.cloudflare.com/cache/how-to/edge-browser-cache-ttl/ | 2026-09-10T11:10:00Z | Free: 2h, Pro: 1h, Biz/Ent: 1s. |

## 16. Rules

| id | name | kind | unit | scope | free | pro | business | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| rules.page_rules | Page Rules per zone | quota | count | zone | 3 | 20 | 50 | 125 | api-list | hard | true | https://developers.cloudflare.com/rules/page-rules/ | 2026-09-10T11:15:00Z | Disabled rules still count against quota. |
| rules.single_redirects | Single Redirects per zone | quota | count | zone | 10 | 25 | 50 | 300 | api-list | hard | true | https://developers.cloudflare.com/rules/url-forwarding/ | 2026-09-10T11:15:00Z | |

## 17. WAF

| id | name | kind | unit | scope | free | pro | business | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| waf.custom_rules | Custom rules per zone | quota | count | zone | 5 | 20 | 100 | 1000 | api-list | hard | true | https://developers.cloudflare.com/waf/custom-rules/ | 2026-09-10T11:20:00Z | Free/Pro/Biz lack Regex and Log actions. |
| waf.rate_limiting_rules | Rate limiting rules per zone | quota | count | zone | 1 | 2 | 5 | 100 | api-list | hard | true | https://developers.cloudflare.com/waf/rate-limiting-rules/ | 2026-09-10T11:20:00Z | |

## 18. Email Routing

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| email_routing.message_size_in | Inbound message size | size | mb | request | 25 | 25 | 25 | local | hard | false | https://developers.cloudflare.com/email-routing/limits/ | 2026-09-10T11:25:00Z | |
| email_routing.rules | Routing rules per domain | quota | count | zone | 200 | 200 | 200 | api-list | hard | true | https://developers.cloudflare.com/email-routing/limits/ | 2026-09-10T11:25:00Z | |

## 19. Health Checks

| id | name | kind | unit | scope | free | pro | business | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| healthchecks.count | Health checks per account | quota | count | account | 0 | 10 | 50 | 1000 | api-list | hard | true | https://developers.cloudflare.com/health-checks/ | 2026-09-10T11:30:00Z | Not available on Free. |

## 20. Domains/Registrar

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| registrar.max_term | Max registration term | size | years | object | 10 | 10 | 10 | opaque | hard | false | https://developers.cloudflare.com/registrar/faq/ | 2026-09-10T11:35:00Z | 10 years for most TLDs, 5 years for .co. |

## 21. Cloudflare REST API

| id | name | kind | unit | scope | free | paid | enterprise | enforceability | soft/hard | trackable | source_url | verified_on | notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| api.global_rate | Global API rate limit | rate | requests_per_5min | account | 1200 | 1200 | 1200 | api-counter | hard | true | https://developers.cloudflare.com/fundamentals/api/reference/limits/ | 2026-09-10T11:40:00Z | Per user/account token. |
| api.ip_rate | Client API per IP | rate | requests_per_sec | account | 200 | 200 | 200 | opaque | hard | false | https://developers.cloudflare.com/fundamentals/api/reference/limits/ | 2026-09-10T11:40:00Z | 200/sec. |
| api.token_quota | User API token quota | quota | count | account | 50 | 50 | 50 | api-list | hard | true | https://developers.cloudflare.com/fundamentals/api/reference/limits/ | 2026-09-10T11:40:00Z | Account API tokens have quota of 500. |

## EDGE CASES

1. **Workers CPU Time (Soft/Hard semantics)**: CPU time excludes `await` time (I/O, fetch, KV/R2 reads). The 10ms/30s/5m limits are enforced per isolate; the runtime allows some flexibility ("built-in flexibility") but terminates execution and returns Error 1102 (`exceededCpu`) when consistently hit.
2. **Workers Daily Requests (Timezone & Fail-open)**: The 100,000 daily cap on the Free plan resets strictly at **midnight UTC**, not a rolling 24-hour window. When exceeded, the behavior is configurable per route: it can return a 1027 error page (Fail Closed) or bypass the worker entirely and proxy directly to the origin (Fail Open).
3. **R2 Multipart vs. Single PUT**: A single `PUT` request caps at 5 GiB. To upload up to 4.995 TiB, you *must* use multipart upload, which itself is capped at exactly 10,000 parts per upload.
4. **R2 API Rate Limits (Endpoint Specificity)**: The 1,200 requests / 5 min limit applies *only* to the Cloudflare REST API (management). High-throughput object operations via the S3-compatible API or Workers API (`env.R2_BUCKET`) are subject to entirely different, much higher, undocumented limits.
5. **KV Same-Key Write Limit**: KV enforces a strict 1 write per second *per key*. Writing to different keys is subject to the daily account quota, but rapid updates to a single key (e.g., a counter or session state) will trigger rate limiting.
6. **D1 Time Travel Restores**: Limited to 10 restores per 10 minutes *per database*. This is a hard quota to prevent rollback loops, not a daily limit.
7. **Queues Message Size Math**: The 128 KB message size limit uses base-10 math (1 KB = 1000 bytes, not 1024). Furthermore, the message includes ~100 bytes of internal Queues metadata, meaning your actual payload must be smaller than 128,000 bytes.
8. **Pages Files Limit Environment Variable**: Paid plans support 100,000 files per deployment (vs 20,000 on Free), but *only* if you explicitly set the environment variable `PAGES_WRANGLER_MAJOR_VERSION=4` in your Pages project settings.
9. **Images Transformations Uniqueness**: You are billed (and capped on the Free plan) per *unique* transformation combination per calendar month. If 1 million users request the exact same resized image, it counts as 1 transformation. Hitting the 5,000 Free cap returns Error 9422, but cached assets continue to serve.
10. **Images Animated Transformations**: GIF/WebP animations are limited to 100 total megapixels across all frames. If the animation exceeds 50 megapixels, Cloudflare will deliver the file but *silently drop all transformation parameters* (no resizing/cropping applied).
11. **Stream Storage Reservation**: When generating a Direct Creator Upload (DCU) link with a `maxDurationSeconds` limit, Stream immediately deducts that duration from your purchased storage quota. If the link expires unused, the storage is released back to your account.
12. **DNS Records Cutoff Date & Buffer**: Free zones created before `2024-09-01 00:00:00 UTC` have a 1,000 record limit; zones created on or after that date have a 200 limit. Additionally, internal Cloudflare services (like Email Routing adding TXT/MX records) are granted a "small buffer" that can slightly exceed the strict zone limit.
13. **Cache Purge Token Buckets**: Purge limits use a token bucket algorithm. A Free account gets 5 requests per minute, but the bucket holds 25 tokens. This allows a short burst of 25 purge requests instantly, but subsequent requests will 429 until tokens refill. These limits are shared across *all* zones on the same plan within an account.
14. **API 429 Semantics & Retry-After**: When hitting the 1,200 / 5 min global API limit, Cloudflare returns a 429 status with a `Retry-After` header indicating the number of seconds (rounded up) until the bucket refills. Note that the `Ratelimit` header returns remaining quota (`r`) and reset time (`t`), but the global limit is per-token, not per-account (though there is a separate 200 req/sec per IP limit).

## CONFLICTS

- **Workers "Number of Workers" Limit**: The documentation table lists "Number of Workers" as 100 (Free) and 500 (Paid). However, older documentation and community forum posts often reference a 1,000 limit for paid plans. The current docs explicitly state 500, but advise using "Workers for Platforms" if you need more than 500 Workers or 1,000 routes.
- **Stream Video Limits**: The FAQ states "By default, a video upload can be at most 30 GB" and "up to 120 videos queued or being encoding simultaneously." However, older pricing pages occasionally referenced "unlimited" video counts. The 120 concurrent encoding limit is strictly enforced via 429 errors and requires a support ticket to raise.
- **WAF Custom Rules vs Rate Limiting Rules**: WAF Custom Rules limit is 5 (Free) / 20 (Pro) / 100 (Biz) / 1000 (Ent). Rate Limiting Rules have a separate, much lower limit of 1 (Free) / 2 (Pro) / 5 (Biz) / 100 (Ent). Users often confuse the two, attempting to create multiple rate limiting rules on the Free plan and hitting the "1 rule" hard limit despite having unused Custom Rule quota.

## CATALOG JSON

```json
[
  {
    "id": "workers.requests_daily",
    "service": "workers",
    "kind": "quota",
    "unit": "requests_per_day",
    "name": "Daily request cap",
    "scope": "account",
    "tiers": { "free": 100000, "paid": null, "enterprise": null },
    "trackable": true,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/workers/platform/limits/",
    "verified_on": "2026-09-10T10:00:00Z",
    "notes": "Resets at midnight UTC. Can fail open (bypass) or closed (Error 1027)."
  },
  {
    "id": "workers.cpu_time",
    "service": "workers",
    "kind": "size",
    "unit": "ms",
    "name": "CPU time per request",
    "scope": "invocation",
    "tiers": { "free": 10, "paid": 300000, "enterprise": 300000 },
    "trackable": true,
    "enforceability": "opaque",
    "soft": true,
    "source_url": "https://developers.cloudflare.com/workers/platform/limits/",
    "verified_on": "2026-09-10T10:00:00Z",
    "notes": "Paid defaults to 30s, configurable to 5m. Excludes await on I/O."
  },
  {
    "id": "workers.memory",
    "service": "workers",
    "kind": "size",
    "unit": "mb",
    "name": "Memory per isolate",
    "scope": "isolate",
    "tiers": { "free": 128, "paid": 128, "enterprise": 128 },
    "trackable": false,
    "enforceability": "opaque",
    "soft": true,
    "source_url": "https://developers.cloudflare.com/workers/platform/limits/",
    "verified_on": "2026-09-10T10:00:00Z",
    "notes": "Soft limit: runtime lets in-flight requests complete before spawning a new isolate."
  },
  {
    "id": "workers.subrequests",
    "service": "workers",
    "kind": "quota",
    "unit": "count",
    "name": "Subrequests per invocation",
    "scope": "invocation",
    "tiers": { "free": 50, "paid": 10000, "enterprise": 10000 },
    "trackable": true,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/workers/platform/limits/",
    "verified_on": "2026-09-10T10:00:00Z",
    "notes": "Paid configurable up to 10M. Each redirect chain counts."
  },
  {
    "id": "workers.script_size",
    "service": "workers",
    "kind": "size",
    "unit": "mb",
    "name": "Worker bundle size (uncompressed)",
    "scope": "script",
    "tiers": { "free": 64, "paid": 64, "enterprise": 64 },
    "trackable": true,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/workers/platform/limits/",
    "verified_on": "2026-09-10T10:00:00Z",
    "notes": "No compressed size limit. Only uncompressed bundle counts."
  },
  {
    "id": "workers.env_vars",
    "service": "workers",
    "kind": "quota",
    "unit": "count",
    "name": "Environment variables per worker",
    "scope": "script",
    "tiers": { "free": 64, "paid": 128, "enterprise": 128 },
    "trackable": true,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/workers/platform/limits/",
    "verified_on": "2026-09-10T10:00:00Z",
    "notes": "Secrets + text vars combined. Max 5 KB per var."
  },
  {
    "id": "r2.object_size",
    "service": "r2",
    "kind": "size",
    "unit": "tb",
    "name": "Maximum object size",
    "scope": "object",
    "tiers": { "free": 5, "paid": 5, "enterprise": 5 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/r2/platform/limits/",
    "verified_on": "2026-09-10T10:05:00Z",
    "notes": "5 TiB per object."
  },
  {
    "id": "r2.upload_single",
    "service": "r2",
    "kind": "size",
    "unit": "gb",
    "name": "Single-part upload max size",
    "scope": "object",
    "tiers": { "free": 5, "paid": 5, "enterprise": 5 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/r2/platform/limits/",
    "verified_on": "2026-09-10T10:05:00Z",
    "notes": "5 GiB. Use multipart for larger files."
  },
  {
    "id": "r2.upload_multipart",
    "service": "r2",
    "kind": "size",
    "unit": "tb",
    "name": "Multipart upload max size",
    "scope": "object",
    "tiers": { "free": 4.995, "paid": 4.995, "enterprise": 4.995 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/r2/platform/limits/",
    "verified_on": "2026-09-10T10:05:00Z",
    "notes": "4.995 TiB. Max 10,000 parts per upload."
  },
  {
    "id": "r2.object_metadata",
    "service": "r2",
    "kind": "size",
    "unit": "bytes",
    "name": "Object metadata size",
    "scope": "object",
    "tiers": { "free": 8192, "paid": 8192, "enterprise": 8192 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/r2/platform/limits/",
    "verified_on": "2026-09-10T10:05:00Z",
    "notes": "8,192 bytes (8 KB)."
  },
  {
    "id": "r2.api_rate",
    "service": "r2",
    "kind": "rate",
    "unit": "requests_per_5min",
    "name": "REST API rate limit",
    "scope": "account",
    "tiers": { "free": 1200, "paid": 1200, "enterprise": 1200 },
    "trackable": true,
    "enforceability": "api-counter",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/r2/platform/limits/",
    "verified_on": "2026-09-10T10:05:00Z",
    "notes": "Applies only to Cloudflare REST API. S3/Workers APIs have separate/higher limits."
  },
  {
    "id": "r2.concurrent_writes",
    "service": "r2",
    "kind": "rate",
    "unit": "ops_per_second",
    "name": "Concurrent writes to same key",
    "scope": "object",
    "tiers": { "free": 1, "paid": 1, "enterprise": 1 },
    "trackable": false,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/r2/platform/limits/",
    "verified_on": "2026-09-10T10:05:00Z",
    "notes": "1 per second."
  },
  {
    "id": "kv.reads_daily",
    "service": "kv",
    "kind": "quota",
    "unit": "ops_per_day",
    "name": "Daily reads",
    "scope": "account",
    "tiers": { "free": 100000, "paid": null, "enterprise": null },
    "trackable": true,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/kv/platform/limits/",
    "verified_on": "2026-09-10T10:10:00Z",
    "notes": "Resets daily."
  },
  {
    "id": "kv.writes_daily",
    "service": "kv",
    "kind": "quota",
    "unit": "ops_per_day",
    "name": "Daily writes (different keys)",
    "scope": "account",
    "tiers": { "free": 1000, "paid": null, "enterprise": null },
    "trackable": true,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/kv/platform/limits/",
    "verified_on": "2026-09-10T10:10:00Z",
    "notes": "Resets daily."
  },
  {
    "id": "kv.writes_same_key",
    "service": "kv",
    "kind": "rate",
    "unit": "ops_per_second",
    "name": "Writes to same key",
    "scope": "namespace",
    "tiers": { "free": 1, "paid": 1, "enterprise": 1 },
    "trackable": false,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/kv/platform/limits/",
    "verified_on": "2026-09-10T10:10:00Z",
    "notes": "Strict 1/sec limit per key."
  },
  {
    "id": "kv.value_size",
    "service": "kv",
    "kind": "size",
    "unit": "mb",
    "name": "Maximum value size",
    "scope": "object",
    "tiers": { "free": 25, "paid": 25, "enterprise": 25 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/kv/platform/limits/",
    "verified_on": "2026-09-10T10:10:00Z",
    "notes": "25 MiB. Metadata max 1024 bytes."
  },
  {
    "id": "kv.key_size",
    "service": "kv",
    "kind": "size",
    "unit": "bytes",
    "name": "Maximum key size",
    "scope": "object",
    "tiers": { "free": 512, "paid": 512, "enterprise": 512 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/kv/platform/limits/",
    "verified_on": "2026-09-10T10:10:00Z",
    "notes": "512 bytes."
  },
  {
    "id": "d1.db_count",
    "service": "d1",
    "kind": "quota",
    "unit": "count",
    "name": "Databases per account",
    "scope": "account",
    "tiers": { "free": 10, "paid": 50000, "enterprise": 50000 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/d1/platform/limits/",
    "verified_on": "2026-09-10T10:15:00Z",
    "notes": "Paid requires Workers Paid plan."
  },
  {
    "id": "d1.db_size",
    "service": "d1",
    "kind": "size",
    "unit": "gb",
    "name": "Maximum database size",
    "scope": "database",
    "tiers": { "free": 0.5, "paid": 10, "enterprise": 10 },
    "trackable": true,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/d1/platform/limits/",
    "verified_on": "2026-09-10T10:15:00Z",
    "notes": "500 MB free, 10 GB paid."
  },
  {
    "id": "d1.time_travel_restores",
    "service": "d1",
    "kind": "rate",
    "unit": "ops_per_10min",
    "name": "Time travel restore ops",
    "scope": "database",
    "tiers": { "free": 10, "paid": 10, "enterprise": 10 },
    "trackable": false,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/d1/platform/limits/",
    "verified_on": "2026-09-10T10:15:00Z",
    "notes": "Max 10 restores per 10 mins."
  },
  {
    "id": "d1.queries_per_invocation",
    "service": "d1",
    "kind": "quota",
    "unit": "count",
    "name": "Queries per Worker invocation",
    "scope": "invocation",
    "tiers": { "free": 50, "paid": 1000, "enterprise": 1000 },
    "trackable": true,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/d1/platform/limits/",
    "verified_on": "2026-09-10T10:15:00Z",
    "notes": "Read subrequest limits."
  },
  {
    "id": "d1.import_size",
    "service": "d1",
    "kind": "size",
    "unit": "gb",
    "name": "Max file import size (d1 execute)",
    "scope": "database",
    "tiers": { "free": 5, "paid": 5, "enterprise": 5 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/d1/platform/limits/",
    "verified_on": "2026-09-10T10:15:00Z",
    "notes": "5 GB max SQL file."
  },
  {
    "id": "queues.message_size",
    "service": "queues",
    "kind": "size",
    "unit": "kb",
    "name": "Max message size",
    "scope": "queue",
    "tiers": { "free": 128, "paid": 128, "enterprise": 128 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/queues/platform/limits/",
    "verified_on": "2026-09-10T10:20:00Z",
    "notes": "1 KB = 1000 bytes. Includes ~100 bytes internal overhead."
  },
  {
    "id": "queues.batch_size",
    "service": "queues",
    "kind": "quota",
    "unit": "count",
    "name": "Max consumer batch size",
    "scope": "invocation",
    "tiers": { "free": 100, "paid": 100, "enterprise": 100 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/queues/platform/limits/",
    "verified_on": "2026-09-10T10:20:00Z",
    "notes": "Max 100 messages."
  },
  {
    "id": "queues.send_batch",
    "service": "queues",
    "kind": "quota",
    "unit": "count",
    "name": "Max messages per sendBatch",
    "scope": "invocation",
    "tiers": { "free": 100, "paid": 100, "enterprise": 100 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/queues/platform/limits/",
    "verified_on": "2026-09-10T10:20:00Z",
    "notes": "Or 256 KB total."
  },
  {
    "id": "queues.retention",
    "service": "queues",
    "kind": "size",
    "unit": "days",
    "name": "Message retention period",
    "scope": "queue",
    "tiers": { "free": 14, "paid": 14, "enterprise": 14 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/queues/platform/limits/",
    "verified_on": "2026-09-10T10:20:00Z",
    "notes": "Configurable up to 14 days."
  },
  {
    "id": "queues.backlog",
    "service": "queues",
    "kind": "size",
    "unit": "gb",
    "name": "Per-queue backlog size",
    "scope": "queue",
    "tiers": { "free": 25, "paid": 25, "enterprise": 25 },
    "trackable": false,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/queues/platform/limits/",
    "verified_on": "2026-09-10T10:20:00Z",
    "notes": "25 GB max per queue."
  },
  {
    "id": "pages.files",
    "service": "pages",
    "kind": "quota",
    "unit": "count",
    "name": "Files per site",
    "scope": "project",
    "tiers": { "free": 20000, "pro": 100000, "business": 100000, "enterprise": 100000 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/pages/platform/limits/",
    "verified_on": "2026-09-10T10:25:00Z",
    "notes": "Paid requires PAGES_WRANGLER_MAJOR_VERSION=4 env var."
  },
  {
    "id": "pages.file_size",
    "service": "pages",
    "kind": "size",
    "unit": "mb",
    "name": "Max single file size",
    "scope": "file",
    "tiers": { "free": 25, "pro": 25, "business": 25, "enterprise": 25 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/pages/platform/limits/",
    "verified_on": "2026-09-10T10:25:00Z",
    "notes": "25 MiB."
  },
  {
    "id": "pages.builds_monthly",
    "service": "pages",
    "kind": "quota",
    "unit": "count",
    "name": "Builds per month",
    "scope": "account",
    "tiers": { "free": 500, "pro": 5000, "business": 20000, "enterprise": null },
    "trackable": true,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/pages/platform/limits/",
    "verified_on": "2026-09-10T10:25:00Z",
    "notes": "Resets monthly."
  },
  {
    "id": "pages.redirects",
    "service": "pages",
    "kind": "quota",
    "unit": "count",
    "name": "Max redirects in _redirects",
    "scope": "project",
    "tiers": { "free": 2100, "pro": 2100, "business": 2100, "enterprise": 2100 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/pages/platform/limits/",
    "verified_on": "2026-09-10T10:25:00Z",
    "notes": "2,000 static + 100 dynamic."
  },
  {
    "id": "images.transformations",
    "service": "images",
    "kind": "quota",
    "unit": "ops_per_month",
    "name": "Unique transformations/month",
    "scope": "account",
    "tiers": { "free": 5000, "paid": null, "enterprise": null },
    "trackable": true,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/images/pricing/",
    "verified_on": "2026-09-10T10:30:00Z",
    "notes": "Billed per unique combination of parameters per calendar month. Free returns Error 9422 when exceeded."
  },
  {
    "id": "images.hosted_size",
    "service": "images",
    "kind": "size",
    "unit": "mb",
    "name": "Hosted image file size",
    "scope": "object",
    "tiers": { "free": 10, "paid": 10, "enterprise": 10 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/images/get-started/limits/",
    "verified_on": "2026-09-10T10:30:00Z",
    "notes": "10 MB max upload. Remote images allow 100 MB."
  },
  {
    "id": "images.animated_area",
    "service": "images",
    "kind": "size",
    "unit": "mp",
    "name": "Animated GIF/WebP area",
    "scope": "object",
    "tiers": { "free": 100, "paid": 100, "enterprise": 100 },
    "trackable": false,
    "enforceability": "opaque",
    "soft": true,
    "source_url": "https://developers.cloudflare.com/images/get-started/limits/",
    "verified_on": "2026-09-10T10:30:00Z",
    "notes": "100 MP max total area across frames. Transformations ignored if > 50 MP."
  },
  {
    "id": "stream.upload_size",
    "service": "stream",
    "kind": "size",
    "unit": "gb",
    "name": "Max video upload size",
    "scope": "object",
    "tiers": { "free": 30, "paid": 30, "enterprise": 30 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/stream/faq/",
    "verified_on": "2026-09-10T10:35:00Z",
    "notes": "30 GB max file size."
  },
  {
    "id": "stream.concurrent_encoding",
    "service": "stream",
    "kind": "quota",
    "unit": "count",
    "name": "Concurrent queued/encoding videos",
    "scope": "account",
    "tiers": { "free": 120, "paid": 120, "enterprise": 120 },
    "trackable": false,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/stream/faq/",
    "verified_on": "2026-09-10T10:35:00Z",
    "notes": "Returns 429 if exceeded. Contact support to raise."
  },
  {
    "id": "vectorize.indexes",
    "service": "vectorize",
    "kind": "quota",
    "unit": "count",
    "name": "Indexes per account",
    "scope": "account",
    "tiers": { "free": 100, "paid": 50000, "enterprise": 50000 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/vectorize/platform/limits/",
    "verified_on": "2026-09-10T10:40:00Z",
    "notes": ""
  },
  {
    "id": "vectorize.dimensions",
    "service": "vectorize",
    "kind": "size",
    "unit": "dimensions",
    "name": "Max dimensions per vector",
    "scope": "index",
    "tiers": { "free": 1536, "paid": 1536, "enterprise": 1536 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/vectorize/platform/limits/",
    "verified_on": "2026-09-10T10:40:00Z",
    "notes": "32 bits precision."
  },
  {
    "id": "vectorize.metadata",
    "service": "vectorize",
    "kind": "size",
    "unit": "kb",
    "name": "Metadata per vector",
    "scope": "object",
    "tiers": { "free": 10, "paid": 10, "enterprise": 10 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/vectorize/platform/limits/",
    "verified_on": "2026-09-10T10:40:00Z",
    "notes": "10 KiB."
  },
  {
    "id": "vectorize.upsert_batch",
    "service": "vectorize",
    "kind": "quota",
    "unit": "count",
    "name": "Max upsert batch size (Workers)",
    "scope": "invocation",
    "tiers": { "free": 1000, "paid": 1000, "enterprise": 1000 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/vectorize/platform/limits/",
    "verified_on": "2026-09-10T10:40:00Z",
    "notes": "HTTP API allows 5000."
  },
  {
    "id": "vectorize.vectors_per_index",
    "service": "vectorize",
    "kind": "quota",
    "unit": "count",
    "name": "Max vectors per index",
    "scope": "index",
    "tiers": { "free": 20000000, "paid": 20000000, "enterprise": 20000000 },
    "trackable": true,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/vectorize/platform/limits/",
    "verified_on": "2026-09-10T10:40:00Z",
    "notes": "20M vectors."
  },
  {
    "id": "workers_ai.text_gen_rate",
    "service": "workers_ai",
    "kind": "rate",
    "unit": "ops_per_minute",
    "name": "Text generation rate limit (default)",
    "scope": "account",
    "tiers": { "free": 300, "paid": 300, "enterprise": 300 },
    "trackable": true,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/workers-ai/platform/limits/",
    "verified_on": "2026-09-10T10:45:00Z",
    "notes": "Varies by model. Frontier models (e.g., Kimi) are 20 req/min standard, 50 req/min with prepaid Gateway credits."
  },
  {
    "id": "workers_ai.embeddings_rate",
    "service": "workers_ai",
    "kind": "rate",
    "unit": "ops_per_minute",
    "name": "Text embeddings rate limit",
    "scope": "account",
    "tiers": { "free": 3000, "paid": 3000, "enterprise": 3000 },
    "trackable": true,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/workers-ai/platform/limits/",
    "verified_on": "2026-09-10T10:45:00Z",
    "notes": "BGE-large is 1500 req/min."
  },
  {
    "id": "hyperdrive.configs",
    "service": "hyperdrive",
    "kind": "quota",
    "unit": "count",
    "name": "Max configured databases",
    "scope": "account",
    "tiers": { "free": 10, "paid": 25, "enterprise": 25 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/hyperdrive/platform/limits/",
    "verified_on": "2026-09-10T10:50:00Z",
    "notes": ""
  },
  {
    "id": "hyperdrive.origin_conns",
    "service": "hyperdrive",
    "kind": "quota",
    "unit": "count",
    "name": "Max origin DB connections per config",
    "scope": "database",
    "tiers": { "free": 20, "paid": 100, "enterprise": 100 },
    "trackable": false,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/hyperdrive/platform/limits/",
    "verified_on": "2026-09-10T10:50:00Z",
    "notes": "~20 free, ~100 paid."
  },
  {
    "id": "dns.records_zone",
    "service": "dns",
    "kind": "quota",
    "unit": "count",
    "name": "DNS records per zone",
    "scope": "zone",
    "tiers": { "free": 200, "pro": 3500, "business": 3500, "enterprise": null },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/dns/manage-dns-records/",
    "verified_on": "2026-09-10T10:55:00Z",
    "notes": "Free zones created before 2024-09-01 get 1,000. Email Routing records get a small buffer."
  },
  {
    "id": "dns.records_account",
    "service": "dns",
    "kind": "quota",
    "unit": "count",
    "name": "DNS records per account (Enterprise)",
    "scope": "account",
    "tiers": { "free": null, "pro": null, "business": null, "enterprise": 1000000 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/dns/manage-dns-records/",
    "verified_on": "2026-09-10T10:55:00Z",
    "notes": "Public and internal zones counted separately."
  },
  {
    "id": "zones.organization",
    "service": "zones",
    "kind": "quota",
    "unit": "count",
    "name": "Zones per Organization",
    "scope": "account",
    "tiers": { "free": null, "paid": null, "enterprise": 5000 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/fundamentals/organizations/limitations/",
    "verified_on": "2026-09-10T11:00:00Z",
    "notes": "Standard accounts have undocumented soft limits (typically ~50). Organizations (Ent/MSSP) have 5,000."
  },
  {
    "id": "ssl.custom_certs",
    "service": "ssl",
    "kind": "quota",
    "unit": "count",
    "name": "Custom certificate packs",
    "scope": "zone",
    "tiers": { "free": 0, "pro": 0, "business": 6, "enterprise": 6 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/ssl/edge-certificates/custom-certificates/",
    "verified_on": "2026-09-10T11:05:00Z",
    "notes": "Biz/Ent get 5 Modern + 1 Legacy pack. Each pack can hold 3 certs (RSA/ECDSA/SHA1)."
  },
  {
    "id": "ssl.custom_hostnames",
    "service": "ssl",
    "kind": "quota",
    "unit": "count",
    "name": "Custom hostnames (SaaS Pay-as-you-go)",
    "scope": "zone",
    "tiers": { "free": null, "pro": null, "business": null, "enterprise": 50000 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/ssl/changelog/",
    "verified_on": "2026-09-10T11:05:00Z",
    "notes": "Raised from 5,000 in 2026."
  },
  {
    "id": "cache.purge_rate",
    "service": "cache",
    "kind": "rate",
    "unit": "requests_per_sec",
    "name": "Purge requests rate",
    "scope": "account",
    "tiers": { "free": 0.083, "pro": 5, "business": 10, "enterprise": 50 },
    "trackable": true,
    "enforceability": "opaque",
    "soft": true,
    "source_url": "https://developers.cloudflare.com/cache/how-to/purge-cache/",
    "verified_on": "2026-09-10T11:10:00Z",
    "notes": "Free is 5/min. Token bucket based. Shared across zones of same plan."
  },
  {
    "id": "cache.purge_batch_urls",
    "service": "cache",
    "kind": "quota",
    "unit": "count",
    "name": "Max URLs per single-file purge req",
    "scope": "request",
    "tiers": { "free": 100, "pro": 100, "business": 100, "enterprise": 500 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/cache/how-to/purge-cache/",
    "verified_on": "2026-09-10T11:10:00Z",
    "notes": ""
  },
  {
    "id": "cache.file_size_max",
    "service": "cache",
    "kind": "size",
    "unit": "mb",
    "name": "Max cacheable file size",
    "scope": "object",
    "tiers": { "free": 512, "pro": 512, "business": 512, "enterprise": 5120 },
    "trackable": false,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/cache/concepts/default-cache-behavior/",
    "verified_on": "2026-09-10T11:10:00Z",
    "notes": "512 MB (Free/Pro/Biz), 5 GB (Ent)."
  },
  {
    "id": "cache.tags_header_size",
    "service": "cache",
    "kind": "size",
    "unit": "kb",
    "name": "Max Cache-Tag header size",
    "scope": "request",
    "tiers": { "free": 16, "pro": 16, "business": 16, "enterprise": 16 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/cache/how-to/purge-cache/purge-by-tags/",
    "verified_on": "2026-09-10T11:10:00Z",
    "notes": "~1,000 unique tags."
  },
  {
    "id": "cache.edge_ttl_min",
    "service": "cache",
    "kind": "size",
    "unit": "s",
    "name": "Minimum Edge Cache TTL",
    "scope": "zone",
    "tiers": { "free": 7200, "pro": 3600, "business": 1, "enterprise": 1 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/cache/how-to/edge-browser-cache-ttl/",
    "verified_on": "2026-09-10T11:10:00Z",
    "notes": "Free: 2h, Pro: 1h, Biz/Ent: 1s."
  },
  {
    "id": "rules.page_rules",
    "service": "rules",
    "kind": "quota",
    "unit": "count",
    "name": "Page Rules per zone",
    "scope": "zone",
    "tiers": { "free": 3, "pro": 20, "business": 50, "enterprise": 125 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/rules/page-rules/",
    "verified_on": "2026-09-10T11:15:00Z",
    "notes": "Disabled rules still count against quota."
  },
  {
    "id": "rules.single_redirects",
    "service": "rules",
    "kind": "quota",
    "unit": "count",
    "name": "Single Redirects per zone",
    "scope": "zone",
    "tiers": { "free": 10, "pro": 25, "business": 50, "enterprise": 300 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/rules/url-forwarding/",
    "verified_on": "2026-09-10T11:15:00Z",
    "notes": ""
  },
  {
    "id": "waf.custom_rules",
    "service": "waf",
    "kind": "quota",
    "unit": "count",
    "name": "Custom rules per zone",
    "scope": "zone",
    "tiers": { "free": 5, "pro": 20, "business": 100, "enterprise": 1000 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/waf/custom-rules/",
    "verified_on": "2026-09-10T11:20:00Z",
    "notes": "Free/Pro/Biz lack Regex and Log actions."
  },
  {
    "id": "waf.rate_limiting_rules",
    "service": "waf",
    "kind": "quota",
    "unit": "count",
    "name": "Rate limiting rules per zone",
    "scope": "zone",
    "tiers": { "free": 1, "pro": 2, "business": 5, "enterprise": 100 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/waf/rate-limiting-rules/",
    "verified_on": "2026-09-10T11:20:00Z",
    "notes": ""
  },
  {
    "id": "email_routing.message_size_in",
    "service": "email_routing",
    "kind": "size",
    "unit": "mb",
    "name": "Inbound message size",
    "scope": "request",
    "tiers": { "free": 25, "paid": 25, "enterprise": 25 },
    "trackable": false,
    "enforceability": "local",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/email-routing/limits/",
    "verified_on": "2026-09-10T11:25:00Z",
    "notes": ""
  },
  {
    "id": "email_routing.rules",
    "service": "email_routing",
    "kind": "quota",
    "unit": "count",
    "name": "Routing rules per domain",
    "scope": "zone",
    "tiers": { "free": 200, "paid": 200, "enterprise": 200 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/email-routing/limits/",
    "verified_on": "2026-09-10T11:25:00Z",
    "notes": ""
  },
  {
    "id": "healthchecks.count",
    "service": "healthchecks",
    "kind": "quota",
    "unit": "count",
    "name": "Health checks per account",
    "scope": "account",
    "tiers": { "free": 0, "pro": 10, "business": 50, "enterprise": 1000 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/health-checks/",
    "verified_on": "2026-09-10T11:30:00Z",
    "notes": "Not available on Free."
  },
  {
    "id": "registrar.max_term",
    "service": "registrar",
    "kind": "size",
    "unit": "years",
    "name": "Max registration term",
    "scope": "object",
    "tiers": { "free": 10, "paid": 10, "enterprise": 10 },
    "trackable": false,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/registrar/faq/",
    "verified_on": "2026-09-10T11:35:00Z",
    "notes": "10 years for most TLDs, 5 years for .co."
  },
  {
    "id": "api.global_rate",
    "service": "api",
    "kind": "rate",
    "unit": "requests_per_5min",
    "name": "Global API rate limit",
    "scope": "account",
    "tiers": { "free": 1200, "paid": 1200, "enterprise": 1200 },
    "trackable": true,
    "enforceability": "api-counter",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/fundamentals/api/reference/limits/",
    "verified_on": "2026-09-10T11:40:00Z",
    "notes": "Per user/account token."
  },
  {
    "id": "api.ip_rate",
    "service": "api",
    "kind": "rate",
    "unit": "requests_per_sec",
    "name": "Client API per IP",
    "scope": "account",
    "tiers": { "free": 200, "paid": 200, "enterprise": 200 },
    "trackable": false,
    "enforceability": "opaque",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/fundamentals/api/reference/limits/",
    "verified_on": "2026-09-10T11:40:00Z",
    "notes": "200/sec."
  },
  {
    "id": "api.token_quota",
    "service": "api",
    "kind": "quota",
    "unit": "count",
    "name": "User API token quota",
    "scope": "account",
    "tiers": { "free": 50, "paid": 50, "enterprise": 50 },
    "trackable": true,
    "enforceability": "api-list",
    "soft": false,
    "source_url": "https://developers.cloudflare.com/fundamentals/api/reference/limits/",
    "verified_on": "2026-09-10T11:40:00Z",
    "notes": "Account API tokens have quota of 500."
  }
]
```