import type { Guide } from "./guides";

// Observability education series for the Guides section: static article
// data only. Guidance prose, no usage claims, so the "real numbers only"
// rule does not apply. Facts reflect stable public Cloudflare behavior as
// of 2025-2026. Plan-specific numbers are deliberately absent: the prose
// describes the concept and points to the dashboard limits page.

export const OBSERVABILITY_GUIDES: readonly Guide[] = [
  {
    id: "graphql-analytics-api",
    title: "The GraphQL Analytics API",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/analytics/graphql-api/",
    body: [
      {
        kind: "p",
        text: "Cloudflare exposes nearly all product analytics through one GraphQL endpoint: https://api.cloudflare.com/client/v4/graphql. One POST with your API token returns JSON. You never query raw events. You query pre-aggregated groups, and each product publishes its own dataset for that.",
      },
      { kind: "h3", text: "One dataset per product" },
      {
        kind: "list",
        text: [
          "workersInvocationsAdaptive: per-invocation Worker analytics. Dimensions cover script, status, colo and time. Sums cover requests, errors and subrequests. Quantiles cover CPU time.",
          "kvOperationsAdaptiveGroups: KV operations bucketed by namespace, action type and status.",
          "d1AnalyticsAdaptiveGroups: per-database query counts, rows read and rows written.",
          "httpRequests1dGroups: zone HTTP traffic aggregated per day, including requests, bytes, cache status and threat counts.",
        ],
      },
      { kind: "h3", text: "Dimensions, sums and maxes" },
      {
        kind: "p",
        text: "Every dataset splits its fields into three roles. Dimensions describe the buckets: scriptName, country, datetimeMinute. Sums add up events inside each bucket: requests, bytes, errors. Maxes and quantiles describe distributions inside the bucket, such as peak CPU time or CPU p99. A query groups by dimensions and reads sums or quantiles from each group. The schema cannot mix a raw event field with an aggregate, which is what makes the results stable.",
      },
      { kind: "h3", text: "The 10,000 group cap" },
      {
        kind: "p",
        text: "A query may return at most 10,000 groups. Wide windows with fine granularity blow past the cap, and the API rejects the query rather than truncating the result. The fix is always one of three moves: shorten the window, drop the highest-cardinality dimension, or step up to a coarser dataset. A good habit is to iterate: run one day at minute granularity first, then widen the window only as the grouping gets coarser.",
      },
      { kind: "h3", text: "Granularity: adaptive versus 1d" },
      {
        kind: "p",
        text: "Datasets whose name ends in Adaptive record events at roughly minute resolution and sample adaptively: quiet periods stay near-exact, and heavy periods are counted statistically. Datasets ending in 1dGroups are pre-aggregated per day. They answer long-range questions cheaply and hit the group cap far more slowly, but they cannot show intra-day spikes. Rule of thumb: adaptive datasets for the last 24 to 72 hours and for alerting, 1d datasets for weekly and monthly trends.",
      },
    ],
  },
  {
    id: "logpush",
    title: "Logpush",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/logs/",
    body: [
      {
        kind: "p",
        text: "Analytics answers questions about aggregates. Logpush ships the raw records themselves: every request, every Worker invocation event, every audit entry, delivered as compressed batches to storage you own. Once the records land in your bucket, retention and queries are yours.",
      },
      { kind: "h3", text: "Destinations" },
      {
        kind: "list",
        text: [
          "S3-compatible object storage: AWS S3, Cloudflare R2, Google Cloud Storage, and any endpoint that speaks the S3 API.",
          "HTTP receivers: Datadog, New Relic, Splunk, Sumo Logic, Elasticsearch, and a generic HTTPS endpoint for anything else.",
          "Other first-class bindings such as Azure Blob Storage.",
        ],
      },
      { kind: "h3", text: "Field selection" },
      {
        kind: "p",
        text: "Each job declares one dataset and one field set. You pick the fields per job from that dataset's catalog, so a job for security review can ship the wide set while a job for traffic accounting ships ten fields. Fewer fields means smaller objects and faster downstream loads, and it is the only cost lever you control at job level.",
      },
      { kind: "h3", text: "Fidelity and sampling" },
      {
        kind: "p",
        text: "Logpush fidelity is plan-dependent. Higher tiers stream every event. Lower tiers stream a sample, and Workers trace events sample below the top tier as well. Before you build anything that assumes every request is present, read the sampling terms on the current product page for your plan.",
      },
      { kind: "h3", text: "How jobs move data" },
      {
        kind: "list",
        text: [
          "A job pushes accumulated records in batches, typically minutes apart. It is a pipeline, not a live tail.",
          "Before the first push, Logpush writes a challenge file into the destination. Confirming it proves write access.",
          "Object-store jobs write compressed files named by timestamp. Treat each file downstream as an immutable shard.",
        ],
      },
      { kind: "h3", text: "Cost model" },
      {
        kind: "p",
        text: "On some plans Logpush bills per log push request. Cost then scales with how much traffic you push and how finely jobs batch, not with how many fields you select. Check the current Logpush page for your plan's billing unit before you add jobs or shrink the batching interval.",
      },
    ],
  },
  {
    id: "workers-observability",
    title: "Observing Workers",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/workers/observability/",
    body: [
      {
        kind: "p",
        text: "A Worker that fails silently is the most expensive failure mode on the platform. Every failed invocation is still billed, and nothing tells you unless you look. Workers give you four layers of observation. Wire all four before you need any of them.",
      },
      { kind: "h3", text: "Four layers" },
      {
        kind: "list",
        text: [
          "Tail workers: attach a second Worker as a tail consumer. It receives an event for every invocation of the source Worker, including logs and exceptions, after the response has been sent. It never adds latency to the user path.",
          "wrangler tail and the tail API: stream live invocation logs to your terminal, or open the WebSocket session yourself for custom tooling. Both are session-scoped: nothing is captured once the session closes.",
          "Workers Observability: enable it once and console.log output plus errors are captured, sampled, indexed and queryable in the dashboard, with no tail session attached.",
          "Per-script analytics: invocation counts, error counts and CPU time quantiles per script, in the dashboard and through the workersInvocationsAdaptive GraphQL dataset.",
        ],
      },
      { kind: "h3", text: "What per-script analytics give you" },
      {
        kind: "p",
        text: "The invocation dataset carries per-invocation events and aggregates by script, so it answers the questions the dashboard buries: error rate per script over the last hour, which script leads CPU p99, how many subrequests an average invocation makes. These are the numbers an alert rule should read. A cron Worker stuck in an error loop is invisible to users for days while it burns billed requests. The per-script error count sees it immediately.",
      },
      { kind: "h3", text: "The discipline" },
      {
        kind: "p",
        text: "Set thresholds per script, not per account. Alert when one script's error rate crosses a few percent, when its CPU p99 approaches its CPU limit, and when a cron-triggered script errors on consecutive runs. CPU creep and error loops are the two failure modes that arrive slowly and bill continuously. Catching either before a customer notices is the entire return on an observability budget.",
      },
    ],
  },
  {
    id: "web-analytics",
    title: "Cloudflare Web Analytics",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/web-analytics/",
    body: [
      {
        kind: "p",
        text: "Web Analytics measures real browsers, not edge traffic. You add one small script tag, each page load sends one beam, and the dashboard records what actually rendered. It sets no cookies and does not fingerprint, so it collects none of the data a consent banner exists for.",
      },
      { kind: "h3", text: "What one page load sends" },
      {
        kind: "p",
        text: "The snippet loads beacon.min.js from Cloudflare's edge. On each page load the script sends a beam: a small request carrying the URL, the referrer, the country resolved at the edge, and rendering timings for Core Web Vitals. For zones proxied through Cloudflare you can enable automatic setup and skip the snippet entirely.",
      },
      { kind: "h3", text: "What it measures" },
      {
        kind: "list",
        text: [
          "Page views and visits, including entry and exit pages.",
          "Countries and referrers.",
          "Browser, device class and operating system breakdowns.",
          "Core Web Vitals sampled from real page loads.",
        ],
      },
      {
        kind: "p",
        text: "What it cannot do: build funnels, follow a user across visits, or attribute a conversion. There is no identity, no session stitching and no user profile, by design. If a product decision depends on conversion attribution or cohort behavior, that is a job for product analytics, not Web Analytics.",
      },
      { kind: "h3", text: "Browser truth versus edge truth" },
      {
        kind: "p",
        text: "The GraphQL zone datasets count every request that reaches the edge: bots, scrapers, API clients, health checks, and cache hits that run no JavaScript. Web Analytics counts only page loads in browsers that executed the beacon, and ad blockers suppress a share of those beams. The two numbers will never match, and they should not. Ask the edge dataset how much traffic your infrastructure serves. Ask Web Analytics how many humans rendered your pages.",
      },
    ],
  },
  {
    id: "audit-logs",
    title: "Audit logs",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/logs/",
    body: [
      {
        kind: "p",
        text: "Audit logs answer one question: who changed what, and when. Every account-level mutation lands as a record: the actor, which may be a human, an API token or an API key, plus the action, the resource touched, the source IP and the timestamp. DNS edits, firewall rule changes, Worker deploys, member invites and token creation all appear here.",
      },
      { kind: "h3", text: "Where to read them" },
      {
        kind: "list",
        text: [
          "Dashboard: Manage Account, then Audit Log. Filter by actor, resource or time window.",
          "API: the audit_logs endpoint on the account. Same records, paginated and scriptable.",
          "Logpush: an audit logs dataset ships each record to your own storage, the way request logs ship.",
        ],
      },
      {
        kind: "p",
        text: "Retention length depends on plan tier, and the dashboard shows the window your plan keeps. If compliance or postmortems need longer history, ship audit logs through Logpush into storage you control from day one. You cannot export what the retention window has already dropped.",
      },
      { kind: "h3", text: "Incident review starts here" },
      {
        kind: "p",
        text: "When something breaks, resist debugging the code first. Pull the audit trail for the incident window and check whether anything changed at all: a firewall rule, a DNS record, a token with a new scope, a deploy by a teammate or a CI credential. A five-minute audit review settles whether you are chasing a bug or a change. It costs far less than an hour of source-level debugging that assumed the config was clean.",
      },
      {
        kind: "p",
        text: "Audit records are also your token-hygiene evidence. A token that still appears as an actor after the person who created it has left is an open door. The log names it, and the cleanup is one revocation.",
      },
    ],
  },
];
