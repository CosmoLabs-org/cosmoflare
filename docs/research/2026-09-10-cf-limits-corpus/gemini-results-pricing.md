# Gemini deep-research results — Cloudflare pricing and billing mechanics

Provenance: deep research run 2026-09-12 from the pricing prompt in
deep-research-prompts.md (Prompt: pricing corpus; original draft on clipboard
2026-09-12). Exported by Gemini as 'Cloudflare Pricing and Billing
Mechanics.md' from Downloads. Feeds D5 budget projections, BR-06 alert
thresholds, cost command upgrade, FEAT-018 cost-awareness.

ACCOUNT CONTEXT (owner, 2026-09-12): CosmoLabs accounts run Workers Paid
($5/mo) — monthly auto-bill semantics, no daily hard-fails on the umbrella
services. Set workers_plan: paid as the config default for our own
accounts. NOTE: MyCarGuide production evidence (FEAT-018 item 3) describes
free-tier daily-cap behavior — client accounts may differ; never assume
the umbrella.

GAPS: (1) the Vectorize queried/stored-dimension formulas exported as
image placeholders (![][imageN]) — unrecoverable here; re-fetch from the
Vectorize pricing page before implementing cost math for it. (2) entry
as_of dates say 2026-04-21 (Gemini's research corpus date, not today) —
re-verify at synthesis. (3) superscript footnote refs dangle.

---

# **Cloudflare Platform Billing Mechanics and Unit Economics Specification**

Cloudflare operates a split billing topology: a legacy zone-level model applied per apex domain for network edge services, alongside an account-level serverless developer platform1. The commercial foundation of the developer platform is the Workers Paid subscription, a $5.00 per month baseline that bundles compute, key-value storage, relational database queries, asynchronous message queues, and database connection pooling under a single account-level entitlement1. Across both developer primitives and edge caching, Cloudflare enforces an explicit zero-egress ($0.00 per gigabyte) architectural invariant, shifting cloud infrastructure unit economics from data transfer bandwidth to execution state mutations, storage capacity, and active CPU compute time3.

Platform usage allocations operate across two distinct temporal windows: rolling daily UTC quotas and monthly subscription billing cycles5. Free tiers across serverless products (Workers, KV, D1, Workers AI) enforce hard daily limits that reset uniformly at 00:00 UTC5. When an account on a free allocation breaches its daily threshold, the platform executes a hard failure mode—returning HTTP 429 Too Many Requests responses, internal runtime error codes, or database execution rejections—rather than silently degrading quality of service or automatically invoicing overages5. Conversely, upgrading to Workers Paid eliminates daily operational caps, converting resource metering into monthly baseline allocations backed by pay-as-you-go elastic overage billing3.

Cost modeling across the platform requires accounting for compounding operation counters and fractional rounding rules10. Because data transfer to the public internet is zero-rated across Workers, Pages, R2, KV, and D1, financial liability concentrates in high-frequency mutation patterns3. In particular, asynchronous primitives such as Queues expand single-message ingest into multi-operation lifecycles11, and storage classes such as R2 Infrequent Access enforce billable unit rounding that can bill isolated requests at full integer million-operation increments10.

## **Cloudflare Workers**

Cloudflare Workers compute billing is structured under the Standard usage model1. Metering tracks incoming requests and active CPU execution time, whereas passive wall-clock duration spent awaiting asynchronous network I/O, subrequests, or storage queries does not accrue CPU time or compute charges1. Inbound HTTP requests directed to a Worker from external clients increment the request counter, but subrequests issued from inside the Worker via fetch() to external third-party origins, other Cloudflare edge zones, or platform storage bindings do not incur additional request fees9.

On the Workers Free plan, the platform allocates 100,000 requests per day across all account scripts, resetting at 00:00 UTC, with a strict compute ceiling of 10 milliseconds of CPU execution time per invocation1. Breaching the daily request cap results in an immediate hard failure: the edge proxy terminates incoming traffic with an HTTP 429 Too Many Requests status code (displaying error code 1027: "Worker rate limit exceeded")2. Exceeding the 10 ms CPU ceiling terminates the runtime isolate mid-execution, raising an uncatchable script cancellation exception2.

The Workers Paid plan ($5.00 per month) transitions accounts to monthly aggregate metering, providing 10 million requests and 30 million CPU-milliseconds within the base subscription1. Overages beyond these baselines bill at $0.30 per additional 1 million requests and $0.02 per additional 1 million CPU-milliseconds3. Invocation execution ceilings expand to 30 seconds of CPU time for standard HTTP requests and up to 15 minutes of continuous CPU time for Cron Triggers and Queue consumers2. Network egress is explicitly zero-rated ($0.00 per gigabyte)1. Integrated Workers Builds CI/CD infrastructure provides 3,000 build minutes monthly with 1 concurrent runner on the Free plan, expanding to 6,000 build minutes and 6 concurrent runners on Paid, with excess build minutes billed at $0.005 per minute14.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Inbound Requests | million requests | 0.1 | 0.30 | daily\_utc | hard\_fail | https://developers.cloudflare.com/workers/platform/pricing/ | 2026-04-21 |
| CPU Time | million CPU-ms | null | 0.02 | calendar\_month | auto\_bill | https://developers.cloudflare.com/workers/platform/pricing/ | 2026-04-21 |
| Base Subscription | month | 0 | 5.00 | calendar\_month | auto\_bill | https://developers.cloudflare.com/workers/platform/pricing/ | 2026-04-21 |
| Data Egress | GB | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/workers/platform/pricing/ | 2026-04-21 |
| Build Minutes | minutes | 3000 | 0.005 | calendar\_month | auto\_bill | https://developers.cloudflare.com/workers/ci-cd/builds/limits-and-pricing/ | 2026-04-21 |
| Concurrent Builds | concurrent slots | 1 | null | none | hard\_fail | https://developers.cloudflare.com/workers/ci-cd/builds/limits-and-pricing/ | 2026-04-21 |

## **Cloudflare R2**

Cloudflare R2 provides globally distributed, S3-compatible object storage characterized by an absolute zero-egress pricing model4. Egress data transfer to the public internet or into other Cloudflare developer primitives is permanently billed at $0.00 per gigabyte, eliminating bandwidth retrieval variability4. Invoicing decomposes strictly into monthly average storage volume, Class A state-mutating operations (PutObject, CopyObject, CreateMultipartUpload, UploadPart, CompleteMultipartUpload, and ListObjects), Class B state-reading operations (GetObject and HeadObject), and data retrieval processing volume where applicable4. Deletion operations do not incur Class A or Class B request charges10.

R2 Standard storage includes an account-level monthly free tier comprising 10 GB-months of storage, 1 million Class A operations, and 10 million Class B operations4. Beyond this tier, usage transitions to pay-as-you-go rates: $0.015 per GB-month of storage, $4.50 per million Class A operations, and $0.36 per million Class B operations4. R2 Infrequent Access storage targets long-term archive workloads, lowering baseline storage to $0.01 per GB-month while increasing Class A operations to $9.00 per million, Class B operations to $0.90 per million, and introducing a data retrieval processing fee of $0.01 per gigabyte2. Infrequent Access provides no free tier10.

R2 enforces calendar month billing windows, tracking hourly storage averages across the billing cycle10. When the free tier on Standard storage is exhausted on an account with a verified payment method, operations continue uninterrupted and overages are billed automatically10. A defining financial constraint of R2 is billable unit rounding: Cloudflare rounds billable operations up to the next integer unit (1 million requests)10. Because Infrequent Access includes no free baseline, executing a single Class A operation triggers a billable increment of 1 full million operations, generating an immediate $9.00 charge on the monthly invoice13.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Standard Storage | GB-month | 10 | 0.015 | calendar\_month | auto\_bill | https://developers.cloudflare.com/r2/pricing/ | 2026-04-21 |
| Standard Class A Operations | million requests | 1.0 | 4.50 | calendar\_month | auto\_bill | https://developers.cloudflare.com/r2/pricing/ | 2026-04-21 |
| Standard Class B Operations | million requests | 10.0 | 0.36 | calendar\_month | auto\_bill | https://developers.cloudflare.com/r2/pricing/ | 2026-04-21 |
| Standard Data Egress | GB | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/r2/pricing/ | 2026-04-21 |
| Infrequent Access Storage | GB-month | null | 0.01 | calendar\_month | auto\_bill | https://developers.cloudflare.com/r2/pricing/ | 2026-04-21 |
| Infrequent Access Class A | million requests | null | 9.00 | calendar\_month | auto\_bill | https://developers.cloudflare.com/r2/pricing/ | 2026-04-21 |
| Infrequent Access Class B | million requests | null | 0.90 | calendar\_month | auto\_bill | https://developers.cloudflare.com/r2/pricing/ | 2026-04-21 |
| Infrequent Access Retrieval | GB | null | 0.01 | calendar\_month | auto\_bill | https://developers.cloudflare.com/r2/pricing/ | 2026-04-21 |

## **Cloudflare Workers KV**

Workers KV is an eventually consistent, globally distributed key-value storage engine designed for low-latency read caching at the edge2. Billing meters operations on a strictly per-key basis rather than per HTTP payload5. Bulk operations process multiple keys in a single invocation; writing or reading 500 keys in a single batch request via the REST API or Workers runtime bindings is calculated as 500 individual write or read operations5. Furthermore, read operations targeting nonexistent keys that return null in a Worker or HTTP 404 via the API traverse the routing fabric and are fully billable5.

Under the Workers Free plan, KV usage is governed by daily hard caps that reset at 00:00 UTC: 100,000 read operations, 1,000 write operations, 1,000 delete operations, and 1,000 key listing requests, alongside 1 GB of stored data2. Breaching any single daily operational limit halts execution of that operation type immediately, causing runtime bindings to return errors and reject traffic5.

Subscribing to the Workers Paid plan ($5.00 per month) transitions KV into calendar-month metering, bundling 10 million reads, 1 million writes, 1 million deletes, 1 million listing requests, and 1 GB of storage into the base subscription5. Overages beyond these monthly baselines are billed at $0.50 per million additional reads, $5.00 per million additional writes, deletes, or list requests, and $0.50 per GB-month of storage5. Data transfer out to the public internet or between edge nodes is unmetered and free ($0.00/GB)5.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Keys Read | million keys | 0.1 | 0.50 | daily\_utc | hard\_fail | https://developers.cloudflare.com/kv/platform/pricing/ | 2026-04-21 |
| Keys Written | million keys | 0.001 | 5.00 | daily\_utc | hard\_fail | https://developers.cloudflare.com/kv/platform/pricing/ | 2026-04-21 |
| Keys Deleted | million keys | 0.001 | 5.00 | daily\_utc | hard\_fail | https://developers.cloudflare.com/kv/platform/pricing/ | 2026-04-21 |
| List Requests | million requests | 0.001 | 5.00 | daily\_utc | hard\_fail | https://developers.cloudflare.com/kv/platform/pricing/ | 2026-04-21 |
| Stored Data | GB-month | 1.0 | 0.50 | calendar\_month | hard\_fail | https://developers.cloudflare.com/kv/platform/pricing/ | 2026-04-21 |
| Data Egress | GB | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/kv/platform/pricing/ | 2026-04-21 |

## **Cloudflare D1**

Cloudflare D1 is a serverless relational SQL database built upon an SQLite dialect, decoupling storage capacity from transactional execution6. D1 adheres to a scale-to-zero operational model: idle databases incur zero virtual CPU, memory, or instance-hour charges6. Costs are governed exclusively by three dimensions: rows read, rows written, and total storage across all databases in an account6. Egress bandwidth is free, and edge read replication does not carry a replica-hour surcharge; queries executing against read replicas are billed identically to primary reads based on total rows scanned6.

A row read represents any row scanned by the database engine to resolve a query, independent of whether that row is returned to the client or filtered via an unindexed WHERE clause6. Full table scans across unindexed tables read every record present, whereas index seeks restrict row read counts to the relevant index leaves and matching records6. A row written represents any mutation operation (INSERT, UPDATE, DELETE, or table-altering DDL statements)6. Mutating a column that belongs to an index generates at least one additional write operation to account for updating the corresponding index structure6.

On the Workers Free plan, D1 enforces daily operational ceilings resetting at 00:00 UTC: 5 million rows read per day, 100,000 rows written per day, and a cumulative 5 GB storage limit across all databases6. Exceeding daily read or write allocations causes subsequent queries to hard-fail with database API exceptions8. Breaching the 5 GB storage boundary blocks write operations, schema changes, and index creation until data is purged8. Under the Workers Paid plan ($5.00 per month), D1 lifts daily quotas, providing a monthly baseline of 25 billion rows read, 50 million rows written, and 5 GB of cumulative storage6. Overages bill at $0.001 per million rows read, $1.00 per million rows written, and $0.75 per GB-month of storage6. Individual D1 databases carry an architectural capacity ceiling of 10 GB8.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Rows Read | million rows | 5.0 | 0.001 | daily\_utc | hard\_fail | https://developers.cloudflare.com/d1/platform/pricing/ | 2026-04-21 |
| Rows Written | million rows | 0.1 | 1.00 | daily\_utc | hard\_fail | https://developers.cloudflare.com/d1/platform/pricing/ | 2026-04-21 |
| Storage | GB-month | 5.0 | 0.75 | calendar\_month | hard\_fail | https://developers.cloudflare.com/d1/platform/pricing/ | 2026-04-21 |
| Data Egress | GB | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/d1/platform/pricing/ | 2026-04-21 |

## **Cloudflare Queues**

Cloudflare Queues provides asynchronous message queuing between Workers without requiring provisioning of message broker clusters22. Financial metering is based entirely on operational volume, calculated in 64 KB message payload increments11. If a message payload is 120 KB, publishing, consuming, or acknowledging that message incurs two billable operations per lifecycle stage11. A standard end-to-end message lifecycle—producing a message, reading it via a consumer Worker, and successfully acknowledging (deleting) it—incurs 3 distinct operations per 64 KB segment (1 write \+ 1 read \+ 1 delete)11.

Queues requires an underlying Workers subscription and cannot be provisioned as an isolated service11. Under Workers Free, Queues provides limited daily operation allocations intended for non-production development with constrained retention periods11. Reaching daily free thresholds triggers a hard failure, causing subsequent send() and sendBatch() invocations to fail with runtime exceptions11.

The Workers Paid plan ($5.00 per month) bundles 1 million operations per calendar month11. Usage beyond this allowance is billed at $0.40 per million additional operations11. Because each message under 64 KB requires 3 operations across its lifecycle, processing 1 million delivered messages incurs 3 million operations, resulting in an effective overage cost of $1.20 per million fully processed messages11. Egress data transfer and message storage (with configurable retention up to 14 days) do not incur extra charges11.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Operations | million operations | null | 0.40 | calendar\_month | auto\_bill | https://developers.cloudflare.com/queues/platform/pricing/ | 2026-04-21 |
| Data Egress | GB | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/queues/platform/pricing/ | 2026-04-21 |

## **Cloudflare Pages**

Cloudflare Pages provides Git-integrated static site hosting and full-stack serverless deployment18. Its billing engine separates static asset delivery from dynamic function execution and build pipelines1. Serving static assets (HTML, client JavaScript, CSS, media) is unmetered across all plans; Cloudflare charges zero fees for static request volume, bandwidth, or egress data transfer3.

Dynamic edge routing implemented via Pages Functions (code executing in the /functions directory) is integrated into the Workers runtime infrastructure and billed under Workers pricing1. On the Free tier, Pages Functions share the account-level pool of 100,000 requests per day, failing with HTTP 429 errors upon exhaustion1. On the Workers Paid plan ($5.00 per month), Pages Functions draw from the shared monthly baseline of 10 million requests and 30 million CPU-milliseconds, with overages billed at $0.30 per million requests and $0.02 per million CPU-ms1.

Pages continuous integration pipelines are governed by Workers Builds14. Free tier accounts receive 3,000 build minutes per calendar month with a concurrency limit of 1 active build14. If an account consumes all 3,000 minutes within a calendar month, further builds fail to initiate until the cycle resets (hard fail)14. Accounts on a paid plan receive 6,000 build minutes monthly and 6 concurrent build runners, with additional build time billed automatically at $0.005 per minute14.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Static Asset Requests | requests | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/workers/platform/pricing/ | 2026-04-21 |
| Static Bandwidth / Egress | GB | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/workers/platform/pricing/ | 2026-04-21 |
| Build Minutes | minutes | 3000 | 0.005 | calendar\_month | auto\_bill | https://developers.cloudflare.com/workers/ci-cd/builds/limits-and-pricing/ | 2026-04-21 |
| Concurrent Builds | concurrent slots | 1 | null | none | hard\_fail | https://developers.cloudflare.com/workers/ci-cd/builds/limits-and-pricing/ | 2026-04-21 |
| Functions Invocations | million requests | 0.1 | 0.30 | daily\_utc | hard\_fail | https://developers.cloudflare.com/workers/platform/pricing/ | 2026-04-21 |

## **Cloudflare Images**

Cloudflare Images provides hosted image storage, dynamic variant transformation, and edge caching24. Billing separates hosted image storage from transformation pipelines applied to remote assets24. Delivering optimized images from the edge cache to end clients does not accrue egress bandwidth charges3.

Image storage within Cloudflare's hosted pipeline is billed at $5.00 per month per 100,000 images stored, with no persistent free tier24. Remote transformations—operating on images stored externally in Amazon S3, origin web servers, or Cloudflare R2—are metered based on unique transformations executed within a calendar month24. A unique transformation is defined as a specific combination of source image URL and manipulation parameters (width, height, format, compression quality)26. Subsequent requests for the same transformed derivative within the same calendar month are served from the edge cache and do not accrue transformation charges26.

The Images Paid plan includes 5,000 unique transformations per calendar month24. Additional transformations beyond this allowance are billed at $0.50 per 1,000 unique transformations24. Accounts attempting to perform transformations without an active Images subscription or beyond entitlement boundaries encounter hard execution rejections24.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Unique Transformations | thousand transformations | 5.0 | 0.50 | calendar\_month | auto\_bill | https://developers.cloudflare.com/images/pricing/ | 2026-04-21 |
| Hosted Image Storage | hundred thousand images | null | 5.00 | calendar\_month | hard\_fail | https://developers.cloudflare.com/images/pricing/ | 2026-04-21 |
| Image Delivery / Egress | GB | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/images/pricing/ | 2026-04-21 |

## **Cloudflare Stream**

Cloudflare Stream delivers video ingestion, automated multi-bitrate encoding (HLS and DASH), edge storage, and global video playback1. Stream operates on an unmetered bandwidth model: accounts are not billed for gigabytes of video traffic delivered across the CDN, but rather on two operational metrics: minutes of video stored and minutes of video viewed1.

Stream does not offer a recurring free tier and requires a paid subscription starting at a $5.00 monthly baseline1. Storage is priced at $5.00 per 1,000 minutes of stored video per month ($0.005 per minute)1. Viewing is priced at $1.00 per 1,000 minutes of video streamed to end users per month ($0.001 per minute)1. Storage metering tracks the total duration of encoded source video residing on the platform during the billing cycle, averaged monthly1. Viewing minutes measure the cumulative playback time across all active viewer sessions1.

Account overages in storage and viewing minutes are automatically calculated and added to the monthly billing invoice1. If an account without an active Stream subscription attempts to ingest video via the API or dashboard, the control plane enforces a hard failure, rejecting the upload request1.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Stored Video | thousand minutes | null | 5.00 | calendar\_month | auto\_bill | https://developers.cloudflare.com/stream/ | 2026-04-21 |
| Viewed Video | thousand minutes | null | 1.00 | calendar\_month | auto\_bill | https://developers.cloudflare.com/stream/ | 2026-04-21 |
| Video Delivery / Egress | GB | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/stream/ | 2026-04-21 |

## **Cloudflare Vectorize**

Cloudflare Vectorize is a distributed vector database designed to store and query vector embeddings for semantic search and Retrieval-Augmented Generation (RAG)27. Vectorize does not meter CPU utilization, memory allocations, or provisioned index hours27. Billing is governed by two metrics: queried vector dimensions and stored vector dimensions27. Network egress transfer is free ($0.00/GB)27.

Queried vector dimensions reflect the computational work of evaluating query vectors against stored index vectors27. The formula is:

![][image1]

Stored vector dimensions track persistent capacity:

![][image2]

Under the Workers Free plan, accounts receive 30 million queried vector dimensions and 5 million stored vector dimensions per calendar month, with a limit of 100 indexes per account27. Breaching these monthly allocations triggers a hard failure, rejecting subsequent vector upserts and query operations27. The Workers Paid plan ($5.00 per month) includes 50 million queried vector dimensions and 10 million stored vector dimensions monthly, raising the index ceiling to 50,00027. Overages bill at $0.01 per million queried dimensions and $0.05 per 100 million stored dimensions ($0.0005 per 1 million stored dimensions)27.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Queried Dimensions | million dimensions | 30.0 | 0.01 | calendar\_month | hard\_fail | https://developers.cloudflare.com/vectorize/platform/pricing/ | 2026-04-21 |
| Stored Dimensions | million dimensions | 5.0 | 0.0005 | calendar\_month | hard\_fail | https://developers.cloudflare.com/vectorize/platform/pricing/ | 2026-04-21 |
| Data Egress | GB | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/vectorize/platform/pricing/ | 2026-04-21 |

## **Cloudflare Workers AI**

Cloudflare Workers AI executes machine learning inference on GPUs distributed across Cloudflare edge data centers31. The billing engine operates via two metering frameworks: an abstract compute currency termed Neurons, which normalizes compute across varied tasks (text embeddings, image generation, audio transcription)31, alongside direct per-token pricing for popular open-source Large Language Models (LLMs)31.

All accounts receive a recurring free tier of 10,000 Neurons per day, resetting at 00:00 UTC2. This allocation provides non-production capacity for prototyping (e.g., approximately 10,000 text embedding tokens or basic Whisper transcription)2. When an account on the free allocation exhausts its 10,000 daily Neurons, subsequent inference requests hard-fail with HTTP 429 Too Many Requests errors until the 00:00 UTC boundary resets the pool2.

On the Workers Paid plan ($5.00 per month baseline), compute beyond the free baseline is billed pay-as-you-go31. Standard Neuron consumption is billed at $0.011 per 1,000 Neurons ($0.000011 per Neuron)31. Under direct token-based pricing, models are billed on input, cached input, and output tokens31. Representative production model rates include: Meta Llama 3.1 8B Instruct FP8 Fast at $0.045 per million input tokens and $0.384 per million output tokens; Meta Llama 3.3 70B Instruct FP8 Fast at $0.293 per million input tokens and $2.253 per million output tokens; and DeepSeek R1 Distill Qwen 32B at $0.497 per million input tokens and $4.881 per million output tokens31. Egress data transfer is free4.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Compute Neurons | thousand neurons | 10.0 | 0.011 | daily\_utc | hard\_fail | https://developers.cloudflare.com/workers-ai/platform/pricing/ | 2026-04-21 |
| Data Egress | GB | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/workers-ai/platform/pricing/ | 2026-04-21 |

## **Cloudflare Hyperdrive**

Cloudflare Hyperdrive accelerates centralized relational database access (PostgreSQL, MySQL, CockroachDB) from distributed Workers isolates1. Hyperdrive maintains persistent pools of pre-warmed TCP and TLS connections across Cloudflare’s global network, routing database queries over optimized backbone paths and caching read responses at the edge1.

Hyperdrive does not bill for connection pooling, proxy bandwidth, edge caching, or transactional query execution1. Its access and usage are fully bundled within the core Workers subscription tiers1.

On the Workers Free plan, accounts are entitled to 1 Hyperdrive database configuration to validate connectivity and connection multiplexing1. Creating additional configurations beyond this limit fails at the API provisioning layer1. On the Workers Paid plan ($5.00 per month), Hyperdrive enables up to 50 active database configurations at zero additional cost ($0.00)1. Invocation compute and execution duration are billed solely under standard Workers CPU-time and request rates1. Data egress from Hyperdrive back to the calling Worker or client is free1.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Database Configurations | configuration | 1.0 | 0.00 | none | hard\_fail | https://developers.cloudflare.com/workers/platform/pricing/ | 2026-04-21 |
| Query Pooling / Caching | million queries | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/workers/platform/pricing/ | 2026-04-21 |
| Data Transfer / Egress | GB | null | 0.00 | none | auto\_bill | https://developers.cloudflare.com/workers/platform/pricing/ | 2026-04-21 |

## **Cloudflare Zones**

Zone plans govern apex domain edge infrastructure, encompassing CDN caching, DDoS mitigation, Universal SSL, and Web Application Firewall (WAF) enforcement2. Unlike developer services metered across the entire account, zone subscriptions are contracted per domain (zone)2. Egress bandwidth and traffic volume delivered to web clients are unmetered and free across all tiers3.

The Free Zone plan ($0/month) includes unmetered Layer 7 DDoS protection, global CDN caching, and Universal SSL, but excludes managed WAF rule evaluation2. Traffic spikes and volumetric attacks do not generate financial overages or throttle edge delivery2. The Pro Zone plan ($20/month billed annually or $25/month billed monthly per domain) adds the Cloudflare Managed Ruleset WAF, Polish image optimization, Mirage mobile compression, and 20 Cache Rules2. The Business Zone plan ($200/month per domain) provides custom SSL certificate uploads, a 100% uptime SLA backed by service credits, advanced DDoS threshold management, PCI DSS compliance controls, and 50 Cache Rules2.

The Enterprise Zone plan requires custom annual contracts (typically starting between $2,000 and $5,000+ monthly)1. It includes 24/7/365 phone support with a 15-minute emergency response SLA, multi-user role-based access control (RBAC), Logpush event streaming, and custom contractually defined usage structures1. Zones do not experience throughput throttling or degrade under legitimate volume surges; features exceeding plan scope are blocked at configuration time2.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| Free Plan | zone-month | 1.0 | 0.00 | calendar\_month | auto\_bill | https://www.cloudflare.com/plans/ | 2026-04-21 |
| Pro Plan | zone-month | null | 20.00 | calendar\_month | auto\_bill | https://www.cloudflare.com/plans/ | 2026-04-21 |
| Business Plan | zone-month | null | 200.00 | calendar\_month | auto\_bill | https://www.cloudflare.com/plans/ | 2026-04-21 |
| Enterprise Plan | zone-month | null | null | calendar\_month | unknown | https://www.cloudflare.com/plans/ | 2026-04-21 |
| Zone Egress Traffic | GB | null | 0.00 | none | auto\_bill | https://www.cloudflare.com/plans/ | 2026-04-21 |

## **Cloudflare REST API**

The Cloudflare REST API provides programmatic control across the entire platform, including DNS record management, firewall rule configuration, cache purges, serverless deployments, and analytics retrieval5. Cloudflare levies zero financial fees for REST API calls across all account tiers5.

API access is constrained through defensive rate limiting enforced at the edge5. The global baseline rate limit is 1,200 requests per 5-minute rolling window per user authentication context (API Token or Global API Key), establishing an effective average throughput ceiling of 4 requests per second5. Specific administrative sub-endpoints (such as bulk cache purges or rapid zone creation) enforce lower secondary burst limits to protect back-end control plane databases5.

The REST API enforces a rolling-window rate-limiting mechanic rather than a static daily reset5. When a client exceeds 1,200 calls within any trailing 5-minute period, the API gateway executes a hard failure: further requests are rejected with HTTP 429 Too Many Requests responses containing Retry-After headers until the rolling request count falls below the limit5. Enterprise contracts can negotiate elevated API rate limit ceilings through their account team1.

| metric | unit | free\_included | paid\_price\_usd | billing\_window | overflow\_behavior | source\_url | as\_of |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| API Invocations | requests per 5 min | 1200.0 | 0.00 | rolling | hard\_fail | https://developers.cloudflare.com/fundamentals/api/reference/limits/ | 2026-04-21 |
| API Platform Access | month | 1.0 | 0.00 | none | auto\_bill | https://developers.cloudflare.com/fundamentals/api/reference/limits/ | 2026-04-21 |

## **Platform Bundling Topology and Plan Dependencies**

The $5.00 per month Workers Paid subscription serves as the umbrella commercial gateway across the developer platform1. Subscribing to Workers Paid does not merely unlock additional compute; it alters the plan boundaries and operational ceilings across dependent storage, database, and messaging primitives1.

The complete architectural bundling structure spans the following resource entitlements:

* **Compute**: Workers Paid includes 10 million requests and 30 million CPU-milliseconds monthly, raising the CPU limit per HTTP request from 10 milliseconds to 30 seconds1.  
* **Workers KV**: Elevates daily caps (100k reads, 1k writes) to a monthly baseline of 10 million reads, 1 million writes, 1 million deletes, and 1 million listing requests, enabling pay-as-you-go scaling5.  
* **Cloudflare D1**: Elevates daily caps (5M reads, 100k writes) to a monthly allocation of 25 billion rows read and 50 million rows written, while retaining 5 GB of included storage6.  
* **Cloudflare Queues**: Unlocks 1 million queue operations per month, after which operations bill at $0.40 per million11.  
* **Cloudflare Vectorize**: Unlocks 50 million queried vector dimensions and 10 million stored vector dimensions monthly, raising the maximum index count from 100 to 50,00027.  
* **Cloudflare Hyperdrive**: Increases active database configurations from 1 to 50 at zero operational cost1.  
* **Workers Builds**: Increases monthly build capacity from 3,000 to 6,000 minutes and expands build concurrency from 1 runner to 614.

Conversely, several services remain decoupled from the Workers Paid umbrella: Cloudflare R2 maintains its own independent 10 GB and operation allocation on Standard storage regardless of compute plan4; Cloudflare Images requires an isolated $5.00/100k images subscription24; Cloudflare Stream requires a separate $5.00/1k minutes base commitment1; and Zone plans (Free, Pro, Business, Enterprise) remain metered independently per apex domain2.

## **Conflicts Between Commercial Pricing and Technical Documentation**

Discrepancies between Cloudflare's commercial marketing pages and developer technical documentation reveal operational nuances that directly impact production billing:

> 1. **R2 Infrequent Access Billable Unit Rounding**: Marketing tables advertise Infrequent Access Class A operations at "$9.00 / million requests" and Class B operations at "$0.90 / million requests"2. However, developer documentation specifies that Cloudflare rounds billable usage up to the next integer billing unit10. Because the billable unit is 1 million requests and Infrequent Access includes no free baseline, executing a single Class A operation rounds up to 1 full million operations, generating an immediate $9.00 charge on the invoice13. Marketing pages describe this as utility pricing, concealing the rounding mechanics that affect test or low-frequency workloads10.  
> 2. **Workers Subrequest Billing Clarity**: Commercial documentation advertises Workers Paid at "$0.30 per million requests"3. Developer documentation clarifies that only inbound edge requests to the Worker are charged; subrequests made via fetch() to external origins or internal bindings do not increment the request counter9. However, subrequests that make external network calls can extend isolate execution; while wall-clock I/O wait is unbilled under the Standard usage model, any concurrent CPU processing during streaming responses consumes billable CPU-milliseconds1.  
> 3. **Queues Operations vs. Delivered Messages**: Marketing summaries state that Queues costs "$0.40 per million operations"11. Developer documentation defines an operation as each 64 KB chunk handled during writing, reading, or deleting11. A standard message delivery pipeline requires at least three distinct operations (1 write, 1 read, 1 delete/ack)11. Consequently, delivering 1 million basic messages consumes 3 million billable operations, resulting in an effective delivery cost of $1.20 per million messages11. Payloads exceeding 64 KB scale this multiplier linearly11.  
> 4. **Workers AI Dual-Currency Model**: Earlier documentation and dashboard calculators state that Workers AI is billed exclusively via Neurons ($0.011 per 1,000 Neurons with 10,000 Neurons/day free)2. The current Workers AI pricing specification details direct per-million token pricing for modern LLMs (e.g., Llama 3.1, Llama 3.3, DeepSeek R1 Distill) alongside Neuron pricing31. The documentation does not fully reconcile whether the 10,000 free daily Neurons apply as a credit against token-metered models or apply solely to legacy Neuron-metered models2.  
> 5. **D1 Daily Hard Quotas vs. Monthly Paid Pools**: Marketing pages advertise D1 Free as offering "5 million rows read / day and 100,000 rows written / day"6. Technical documentation notes that this daily limit resets strictly at 00:00 UTC, terminating subsequent queries upon exhaustion6. On Workers Paid, the billing window shifts entirely from daily quotas to a monthly pool of 25 billion rows read and 50 million rows written6. If an unindexed query performs a full table scan across a 6-million-row database, it instantly hard-fails on the Free plan, whereas on Workers Paid, it consumes 0.024% of the monthly allowance without error6.

## **PRICING JSON**

(Editorial note: the export mangled this block — escaped brackets, no fence.
Recovered programmatically into the 52 objects below and mirrored to
pricing-catalog-draft.json. Values are Gemini-reported and carry as_of
2026-04-21 — re-verify at catalog synthesis before embedding.)

```json
[
  {
    "id": "workers.inbound_requests",
    "service": "workers",
    "metric": "Inbound Requests",
    "unit": "million requests",
    "free_included": 0.1,
    "paid_price_usd": 0.3,
    "billing_window": "daily_utc",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/workers/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "workers.cpu_time",
    "service": "workers",
    "metric": "CPU Time",
    "unit": "million CPU-ms",
    "free_included": null,
    "paid_price_usd": 0.02,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/workers/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "workers.base_subscription",
    "service": "workers",
    "metric": "Base Subscription",
    "unit": "month",
    "free_included": 0.0,
    "paid_price_usd": 5.0,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/workers/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "workers.data_egress",
    "service": "workers",
    "metric": "Data Egress",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/workers/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "workers.build_minutes",
    "service": "workers",
    "metric": "Build Minutes",
    "unit": "minutes",
    "free_included": 3000.0,
    "paid_price_usd": 0.005,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/workers/ci-cd/builds/limits-and-pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "workers.concurrent_builds",
    "service": "workers",
    "metric": "Concurrent Builds",
    "unit": "concurrent slots",
    "free_included": 1.0,
    "paid_price_usd": null,
    "billing_window": "none",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/workers/ci-cd/builds/limits-and-pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "r2.standard_storage",
    "service": "r2",
    "metric": "Standard Storage",
    "unit": "GB-month",
    "free_included": 10.0,
    "paid_price_usd": 0.015,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/r2/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "r2.standard_class_a_operations",
    "service": "r2",
    "metric": "Standard Class A Operations",
    "unit": "million requests",
    "free_included": 1.0,
    "paid_price_usd": 4.5,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/r2/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "r2.standard_class_b_operations",
    "service": "r2",
    "metric": "Standard Class B Operations",
    "unit": "million requests",
    "free_included": 10.0,
    "paid_price_usd": 0.36,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/r2/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "r2.standard_data_egress",
    "service": "r2",
    "metric": "Standard Data Egress",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/r2/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "r2.infrequent_access_storage",
    "service": "r2",
    "metric": "Infrequent Access Storage",
    "unit": "GB-month",
    "free_included": null,
    "paid_price_usd": 0.01,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/r2/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "r2.infrequent_access_class_a",
    "service": "r2",
    "metric": "Infrequent Access Class A",
    "unit": "million requests",
    "free_included": null,
    "paid_price_usd": 9.0,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/r2/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "r2.infrequent_access_class_b",
    "service": "r2",
    "metric": "Infrequent Access Class B",
    "unit": "million requests",
    "free_included": null,
    "paid_price_usd": 0.9,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/r2/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "r2.infrequent_access_retrieval",
    "service": "r2",
    "metric": "Infrequent Access Retrieval",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.01,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/r2/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "kv.keys_read",
    "service": "kv",
    "metric": "Keys Read",
    "unit": "million keys",
    "free_included": 0.1,
    "paid_price_usd": 0.5,
    "billing_window": "daily_utc",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/kv/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "kv.keys_written",
    "service": "kv",
    "metric": "Keys Written",
    "unit": "million keys",
    "free_included": 0.001,
    "paid_price_usd": 5.0,
    "billing_window": "daily_utc",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/kv/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "kv.keys_deleted",
    "service": "kv",
    "metric": "Keys Deleted",
    "unit": "million keys",
    "free_included": 0.001,
    "paid_price_usd": 5.0,
    "billing_window": "daily_utc",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/kv/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "kv.list_requests",
    "service": "kv",
    "metric": "List Requests",
    "unit": "million requests",
    "free_included": 0.001,
    "paid_price_usd": 5.0,
    "billing_window": "daily_utc",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/kv/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "kv.stored_data",
    "service": "kv",
    "metric": "Stored Data",
    "unit": "GB-month",
    "free_included": 1.0,
    "paid_price_usd": 0.5,
    "billing_window": "calendar_month",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/kv/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "kv.data_egress",
    "service": "kv",
    "metric": "Data Egress",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/kv/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "d1.rows_read",
    "service": "d1",
    "metric": "Rows Read",
    "unit": "million rows",
    "free_included": 5.0,
    "paid_price_usd": 0.001,
    "billing_window": "daily_utc",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/d1/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "d1.rows_written",
    "service": "d1",
    "metric": "Rows Written",
    "unit": "million rows",
    "free_included": 0.1,
    "paid_price_usd": 1.0,
    "billing_window": "daily_utc",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/d1/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "d1.storage",
    "service": "d1",
    "metric": "Storage",
    "unit": "GB-month",
    "free_included": 5.0,
    "paid_price_usd": 0.75,
    "billing_window": "calendar_month",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/d1/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "d1.data_egress",
    "service": "d1",
    "metric": "Data Egress",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/d1/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "queues.operations",
    "service": "queues",
    "metric": "Operations",
    "unit": "million operations",
    "free_included": null,
    "paid_price_usd": 0.4,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/queues/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "queues.data_egress",
    "service": "queues",
    "metric": "Data Egress",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/queues/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "pages.static_asset_requests",
    "service": "pages",
    "metric": "Static Asset Requests",
    "unit": "requests",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/workers/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "pages.static_bandwidth_egress",
    "service": "pages",
    "metric": "Static Bandwidth / Egress",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/workers/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "pages.build_minutes",
    "service": "pages",
    "metric": "Build Minutes",
    "unit": "minutes",
    "free_included": 3000.0,
    "paid_price_usd": 0.005,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/workers/ci-cd/builds/limits-and-pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "pages.concurrent_builds",
    "service": "pages",
    "metric": "Concurrent Builds",
    "unit": "concurrent slots",
    "free_included": 1.0,
    "paid_price_usd": null,
    "billing_window": "none",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/workers/ci-cd/builds/limits-and-pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "pages.functions_invocations",
    "service": "pages",
    "metric": "Functions Invocations",
    "unit": "million requests",
    "free_included": 0.1,
    "paid_price_usd": 0.3,
    "billing_window": "daily_utc",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/workers/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "images.unique_transformations",
    "service": "images",
    "metric": "Unique Transformations",
    "unit": "thousand transformations",
    "free_included": 5.0,
    "paid_price_usd": 0.5,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/images/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "images.hosted_image_storage",
    "service": "images",
    "metric": "Hosted Image Storage",
    "unit": "hundred thousand images",
    "free_included": null,
    "paid_price_usd": 5.0,
    "billing_window": "calendar_month",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/images/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "images.image_delivery_egress",
    "service": "images",
    "metric": "Image Delivery / Egress",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/images/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "stream.stored_video",
    "service": "stream",
    "metric": "Stored Video",
    "unit": "thousand minutes",
    "free_included": null,
    "paid_price_usd": 5.0,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/stream/",
    "as_of": "2026-04-21"
  },
  {
    "id": "stream.viewed_video",
    "service": "stream",
    "metric": "Viewed Video",
    "unit": "thousand minutes",
    "free_included": null,
    "paid_price_usd": 1.0,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/stream/",
    "as_of": "2026-04-21"
  },
  {
    "id": "stream.video_delivery_egress",
    "service": "stream",
    "metric": "Video Delivery / Egress",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/stream/",
    "as_of": "2026-04-21"
  },
  {
    "id": "vectorize.queried_dimensions",
    "service": "vectorize",
    "metric": "Queried Dimensions",
    "unit": "million dimensions",
    "free_included": 30.0,
    "paid_price_usd": 0.01,
    "billing_window": "calendar_month",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/vectorize/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "vectorize.stored_dimensions",
    "service": "vectorize",
    "metric": "Stored Dimensions",
    "unit": "million dimensions",
    "free_included": 5.0,
    "paid_price_usd": 0.0005,
    "billing_window": "calendar_month",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/vectorize/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "vectorize.data_egress",
    "service": "vectorize",
    "metric": "Data Egress",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/vectorize/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "workers_ai.compute_neurons",
    "service": "workers_ai",
    "metric": "Compute Neurons",
    "unit": "thousand neurons",
    "free_included": 10.0,
    "paid_price_usd": 0.011,
    "billing_window": "daily_utc",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/workers-ai/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "workers_ai.data_egress",
    "service": "workers_ai",
    "metric": "Data Egress",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/workers-ai/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "hyperdrive.database_configurations",
    "service": "hyperdrive",
    "metric": "Database Configurations",
    "unit": "configuration",
    "free_included": 1.0,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/workers/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "hyperdrive.query_pooling_caching",
    "service": "hyperdrive",
    "metric": "Query Pooling / Caching",
    "unit": "million queries",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/workers/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "hyperdrive.data_transfer_egress",
    "service": "hyperdrive",
    "metric": "Data Transfer / Egress",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/workers/platform/pricing/",
    "as_of": "2026-04-21"
  },
  {
    "id": "zones.free_plan",
    "service": "zones",
    "metric": "Free Plan",
    "unit": "zone-month",
    "free_included": 1.0,
    "paid_price_usd": 0.0,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://www.cloudflare.com/plans/",
    "as_of": "2026-04-21"
  },
  {
    "id": "zones.pro_plan",
    "service": "zones",
    "metric": "Pro Plan",
    "unit": "zone-month",
    "free_included": null,
    "paid_price_usd": 20.0,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://www.cloudflare.com/plans/",
    "as_of": "2026-04-21"
  },
  {
    "id": "zones.business_plan",
    "service": "zones",
    "metric": "Business Plan",
    "unit": "zone-month",
    "free_included": null,
    "paid_price_usd": 200.0,
    "billing_window": "calendar_month",
    "overflow_behavior": "auto_bill",
    "source_url": "https://www.cloudflare.com/plans/",
    "as_of": "2026-04-21"
  },
  {
    "id": "zones.enterprise_plan",
    "service": "zones",
    "metric": "Enterprise Plan",
    "unit": "zone-month",
    "free_included": null,
    "paid_price_usd": null,
    "billing_window": "calendar_month",
    "overflow_behavior": "unknown",
    "source_url": "https://www.cloudflare.com/plans/",
    "as_of": "2026-04-21"
  },
  {
    "id": "zones.zone_egress_traffic",
    "service": "zones",
    "metric": "Zone Egress Traffic",
    "unit": "GB",
    "free_included": null,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://www.cloudflare.com/plans/",
    "as_of": "2026-04-21"
  },
  {
    "id": "rest_api.api_invocations",
    "service": "rest_api",
    "metric": "API Invocations",
    "unit": "requests per 5 min",
    "free_included": 1200.0,
    "paid_price_usd": 0.0,
    "billing_window": "rolling",
    "overflow_behavior": "hard_fail",
    "source_url": "https://developers.cloudflare.com/fundamentals/api/reference/limits/",
    "as_of": "2026-04-21"
  },
  {
    "id": "rest_api.api_platform_access",
    "service": "rest_api",
    "metric": "API Platform Access",
    "unit": "month",
    "free_included": 1.0,
    "paid_price_usd": 0.0,
    "billing_window": "none",
    "overflow_behavior": "auto_bill",
    "source_url": "https://developers.cloudflare.com/fundamentals/api/reference/limits/",
    "as_of": "2026-04-21"
  }
]
```
