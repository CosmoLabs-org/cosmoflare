// Storage guides for the education section (Ops education hub, 2026-10-14):
// operator-facing articles on Cloudflare storage services. Prose only, never
// usage claims, so the "real numbers only" rule does not apply. Plan-specific
// quotas are described conceptually and pointed at the dashboard limits page.

import type { Guide } from "./guides";

export const STORAGE_GUIDES: readonly Guide[] = [
  {
    id: "kv-essentials",
    title: "Workers KV: what it is good for",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Workers KV is a global key-value store. It is built for read-heavy workloads where values change rarely. The design trades immediate consistency for low-latency reads everywhere.",
      },
      { kind: "h3", text: "Consistency model" },
      {
        kind: "p",
        text: "Writes propagate to all locations in up to about 60 seconds. A write from one region may not be visible in another region right away. Treat KV as eventually consistent. Do not use it for counters, locks, or any value where a stale read breaks correctness.",
      },
      { kind: "h3", text: "Per-location caching" },
      {
        kind: "p",
        text: "Each colo keeps a local cache of values it has read. A repeated read in the same location is served from that cache and does not touch the central store. The cache holds a value for about 60 seconds by default. A write invalidates caches near the writer, but copies in distant colos can still serve the old value until they expire.",
      },
      { kind: "h3", text: "Limits that shape design" },
      { kind: "kv", term: "Value size", desc: "Up to 25 MiB per value. Store large payloads in R2 and keep a pointer in KV." },
      { kind: "kv", term: "Key size", desc: "Up to 512 bytes per key." },
      { kind: "kv", term: "Write rate", desc: "One write per second per key. A hot key that takes frequent updates belongs in D1." },
      { kind: "kv", term: "TTL", desc: "Set an expiration at write time. The minimum is 60 seconds. Expired keys disappear without a cleanup job." },
      { kind: "h3", text: "When to choose KV" },
      {
        kind: "p",
        text: "The rule: read often, write rarely, and tolerate some staleness. Feature flags, service config, and routing tables fit. Per-request session state does not.",
      },
      {
        kind: "table",
        head: ["Access pattern", "KV", "R2", "D1"],
        rows: [
          ["Read-heavy, rarely written", "Best fit", "Works", "Works"],
          ["Large blobs or files", "No", "Best fit", "No"],
          ["Relational queries", "No", "No", "Best fit"],
          ["Reads that must be current", "No", "No", "Best fit"],
          ["Frequent writes to one key", "No", "Limited", "Best fit"],
        ],
      },
    ],
  },
  {
    id: "r2-essentials",
    title: "R2: storage without egress fees",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "R2 is object storage with an S3-compatible API and zero egress fees. Bytes leaving R2 cost nothing, including over the public internet. That is the headline difference from Amazon S3 and the reason data can be read freely without a bill shock.",
      },
      { kind: "h3", text: "Two APIs, one bucket" },
      {
        kind: "p",
        text: "The S3 API works with existing tools and SDKs, signed with access keys. The Workers API binds a bucket to a Worker and gives typed operations without credentials in code. Both reach the same objects, so a Worker can serve what a script uploaded.",
      },
      { kind: "h3", text: "What you pay for" },
      {
        kind: "list",
        text: [
          "Storage, billed per GB-month.",
          "Class A operations: writes, lists, copies, and other mutations.",
          "Class B operations: reads and HEAD requests.",
          "Zero egress: no charge for bytes transferred out.",
        ],
      },
      {
        kind: "p",
        text: "The free tier includes 10 GB-month of storage, 1 million Class A operations, and 10 million Class B operations per month. Confirm current numbers on the R2 pricing page in the dashboard before you budget.",
      },
      { kind: "h3", text: "Presigned URLs" },
      {
        kind: "p",
        text: "The S3 API can issue presigned URLs. A client uploads or downloads directly against R2 with a URL that expires, so large transfers never pass through your server. Generate presigned URLs server-side. Never put long-lived credentials in client code.",
      },
      { kind: "h3", text: "Lifecycle rules for cold data" },
      {
        kind: "p",
        text: "Lifecycle rules delete objects by age and prefix and abort incomplete multipart uploads. Use them for logs past a retention window, stale exports, and abandoned uploads. Rules run on the bucket without a Worker or a cron job.",
      },
      {
        kind: "p",
        text: "In an ops app, R2 fits exports, log archives, and report files: data where size matters, writes are batchy, and reads are occasional.",
      },
    ],
  },
  {
    id: "d1-economics",
    title: "D1: rows read is the meter",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "D1 is serverless SQLite that runs at the edge. You get real SQL with replication. Billing counts two units: rows read and rows written. Rows read dominates, and it is the unit most teams overlook.",
      },
      { kind: "h3", text: "Why the meter matters" },
      {
        kind: "p",
        text: "A query bills every row it scans, not every row it returns. A table with 100,000 rows and a query with no usable index bills 100,000 rows read even if the result is one row. A dashboard that runs that query on every page view burns budget fast, and the query itself still looks innocent in code review.",
      },
      { kind: "h3", text: "EXPLAIN QUERY PLAN" },
      {
        kind: "p",
        text: "Run any query with EXPLAIN QUERY PLAN in the D1 console. The output tells you the plan: SCAN means a full table scan. SEARCH with USING INDEX means the query walks an index. This is the single most useful tool for D1 cost control, and it costs nothing to run.",
      },
      { kind: "h3", text: "Index the exact filter path" },
      {
        kind: "p",
        text: "Create indexes for the columns in your WHERE clause and ORDER BY. A composite index on the exact query shape beats several single-column guesses. Verify each new index with EXPLAIN QUERY PLAN, then run PRAGMA optimize so the query planner picks up fresh table statistics.",
      },
      { kind: "h3", text: "Cache what repeats" },
      {
        kind: "p",
        text: "If every visitor requests the same aggregate, query it once and cache the result in KV or in memory with a short TTL. A cached answer costs zero rows read. Re-querying identical data per request is the most common D1 cost leak.",
      },
      {
        kind: "list",
        text: [
          "Run EXPLAIN QUERY PLAN on every query in a hot path.",
          "Index the exact WHERE and ORDER BY shape.",
          "Run PRAGMA optimize after schema or index changes.",
          "Cache repeated aggregates behind a short TTL.",
          "Watch rows-read metrics in the dashboard after each deploy.",
        ],
      },
    ],
  },
  {
    id: "vectorize-hyperdrive",
    title: "Vectorize and Hyperdrive",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Two services cover storage cases KV, R2, and D1 do not. Vectorize stores embeddings for similarity search. Hyperdrive connects Workers to databases that live outside Cloudflare.",
      },
      { kind: "h3", text: "Vectorize" },
      {
        kind: "p",
        text: "Vectorize is a vector database. You store embeddings, arrays of numbers produced by a model, and query by similarity. The use case is semantic search, recommendations, and retrieval for language models, where a match is by meaning rather than by exact text.",
      },
      {
        kind: "p",
        text: "The workflow: generate embeddings with Workers AI or any model, insert them into an index, then query with a new embedding to get the nearest stored vectors. Index dimensions must match the model output. Current indexes support up to 3072 dimensions. Free and paid plans differ in vector count and query volume, so check the limits page in the dashboard for the numbers that apply to your plan.",
      },
      { kind: "h3", text: "Hyperdrive" },
      {
        kind: "p",
        text: "Hyperdrive is a pooling and caching layer for existing PostgreSQL and MySQL databases. The database stays where it is. Hyperdrive keeps warm connections ready so a Worker skips repeated connection setup round trips, and it caches common query results near users.",
      },
      {
        kind: "p",
        text: "Use it when a legacy database must serve Workers traffic and a migration is not an option. Hyperdrive does not move data and does not provide SQL itself. Configure the connection string and credentials on the Hyperdrive instance, then bind that instance to the Worker.",
      },
    ],
  },
  {
    id: "choosing-storage",
    title: "Choosing between KV, R2 and D1",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Pick storage by access pattern first and by convenience second. The wrong pick shows up later as a cost problem or a consistency bug, and both are expensive to fix after launch.",
      },
      { kind: "h3", text: "The decision tree" },
      {
        kind: "list",
        text: [
          "Blobs and files of any size, read occasionally: R2.",
          "Config, feature flags, routing tables, values read constantly and changed rarely: KV.",
          "Relational queries, joins, aggregates, anything that must be consistent at read time: D1.",
          "Semantic search or retrieval over embeddings: Vectorize.",
          "An existing external database that Workers must query: Hyperdrive.",
        ],
      },
      { kind: "h3", text: "Consistency decides first" },
      {
        kind: "p",
        text: "Ask one question before anything else: can a read return a value up to a minute old? If no, the answer is D1. If yes, KV or R2, whichever matches the data shape.",
      },
      { kind: "h3", text: "The hybrid pattern" },
      {
        kind: "p",
        text: "Store the source of truth in R2 or D1 and cache hot reads in KV with a short TTL. Example: report files live in R2, the index of recent reports lives in KV with a 60 second TTL, and the usage numbers that must be exact come from D1.",
      },
      {
        kind: "p",
        text: "The pattern splits the jobs: cheap durable storage, fast global reads, and exact numbers where exactness matters. No single store has to carry all three.",
      },
    ],
  },
];
