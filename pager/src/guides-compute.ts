import type { Guide } from "./guides";

// Compute articles for the Guides education hub: the Workers execution model,
// Durable Objects, Queues, Workers AI / AI Gateway, and Cron Triggers. Same
// data-file pattern as guides.ts: static prose only, nothing fetched. Facts
// reflect public Cloudflare documentation as of 2025-2026. Plan-specific
// quotas are deliberately not hard-coded; those articles describe the
// concept and point to the dashboard limits page instead.

export const COMPUTE_GUIDES: readonly Guide[] = [
  {
    id: "workers-model",
    title: "Workers: the execution model",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "A Worker is your JavaScript, executed on Cloudflare's network on every incoming request. There is no server of yours behind it and no region to pick. Understanding the execution model explains most Workers behavior: what is fast, what is billed, and where the real limits sit.",
      },
      { kind: "h3", text: "V8 isolates, not containers" },
      {
        kind: "p",
        text: "Your code runs in a V8 isolate, the same sandboxing technology a browser uses to isolate tabs. An isolate lives inside an already-running process. There is no operating system to boot, no machine image to pull, no container to schedule. The cold-start question that dominates container planning barely applies here: starting a fresh isolate costs a few milliseconds, and warm isolates are reused across requests.",
      },
      { kind: "h3", text: "CPU time versus wall time" },
      {
        kind: "p",
        text: "Two clocks matter, and they are not the same clock. Wall time is elapsed time from invocation start to response finish. CPU time is the time your code actually spends executing on a core. Waiting on a fetch burns wall time and no CPU time. Parsing a large JSON payload burns CPU time. Workers limits are enforced on CPU time, which is why await-heavy code so rarely hits them.",
      },
      {
        kind: "list",
        text: [
          "An await on a slow upstream costs wall time, not CPU time, and is not what the CPU limit measures.",
          "Tight loops, heavy serialization, and image or crypto work are what actually exhaust the CPU budget.",
          "The free plan allows a small per-invocation CPU allowance measured in milliseconds.",
          "The paid plan allows seconds of CPU per invocation by default and is configurable upward.",
        ],
      },
      {
        kind: "p",
        text: "Exact CPU allowances per plan change over time. Read the current values on the dashboard limits page rather than treating any number here as a quota.",
      },
      { kind: "h3", text: "Paid plan billing units" },
      {
        kind: "p",
        text: "On the paid plan, Workers bill on two units: requests and CPU milliseconds. A request is one invocation, including scheduled and queue-consumer invocations. CPU milliseconds are the actual compute your code consumed, summed across the invocation. Wall-clock waiting does not appear on the bill. The dashboard usage pages show both units for the current period.",
      },
      { kind: "h3", text: "Bindings: reach everything without credentials" },
      {
        kind: "p",
        text: "Bindings are named references you declare in your Worker configuration, and the platform injects them into the handler's env object. A binding to a KV namespace, an R2 bucket, a D1 database, a Durable Object class, or a Queue gives you a typed client for that resource. No API token appears in your code, no secret is read at runtime, and no service authentication is your responsibility.",
      },
      {
        kind: "list",
        text: [
          "KV namespace: eventually consistent key-value reads and writes.",
          "R2 bucket: S3-compatible object storage, no access keys in play.",
          "D1 database: serverless SQL, queried over the binding.",
          "Durable Object namespace: a stub to fetch or RPC into one object instance.",
          "Queue producer: enqueue messages onto a Queues queue from inside a request.",
        ],
      },
      {
        kind: "p",
        text: "The practical consequence: a leaked repository leaks no storage credentials, because none exist in the code. Access is granted where the binding is declared, and revoked the same way.",
      },
    ],
  },
  {
    id: "durable-objects",
    title: "Durable Objects: state with one owner",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "A Durable Object is a small class you write, instantiated with a unique id. The platform guarantees that exactly one instance of a given id is active worldwide at any moment. All requests addressed to that id reach that one instance. That single guarantee is the whole product, and it is stronger than it first looks.",
      },
      { kind: "h3", text: "Single-threaded by construction" },
      {
        kind: "p",
        text: "Requests to one object are processed one at a time. There is no lock to take and no race window on in-memory state: while the object works on a request, the next one waits. Code that is unsafe under concurrency, like read-modify-write on an in-memory counter, is safe here without any locking discipline.",
      },
      { kind: "h3", text: "Strong consistency" },
      {
        kind: "p",
        text: "A read through the object observes every write the object has acknowledged. There is no replica lag and no stale read window inside the object. This is the property a read replica does not give you by default, and it is why Durable Objects sit under real-time features rather than beside them.",
      },
      { kind: "h3", text: "Canonical uses" },
      {
        kind: "kv",
        term: "WebSocket coordination",
        desc: "One object holds the sockets for a room, document, or game session and broadcasts from a single place.",
      },
      {
        kind: "kv",
        term: "Counters and rate limits",
        desc: "Increment a value with no race, as a global counter or a per-key limiter with a correct view of the window.",
      },
      {
        kind: "kv",
        term: "Coordination point",
        desc: "Leader election, genuinely simple distributed locks, and per-tenant state that lives next to its data.",
      },
      { kind: "h3", text: "When an object beats a database row lock" },
      {
        kind: "p",
        text: "A row lock holds a transaction open across a network round trip, contends under load, and ends in timeouts or deadlocks. An object serializes at the platform level instead: arrival order is execution order, and contention costs queueing, not lock churn. Reach for one when you need ordering across requests, not just atomicity on a single row. The two combine well: the object coordinates and decides, D1 or R2 stores the durable result, and the object's own storage holds its hot working state.",
      },
      { kind: "h3", text: "Billing" },
      {
        kind: "p",
        text: "Durable Objects bill on requests plus duration. Duration accrues while an object is active and is priced in GB-seconds of the memory the object is provisioned with. Objects backed by hibernating WebSockets are the exception: while the WebSocket hibernates between messages, duration does not accrue. Check the dashboard limits and pricing pages for the plan-specific numbers; the concept, active time times memory, is what stays stable.",
      },
    ],
  },
  {
    id: "queues",
    title: "Queues: moving work off the request path",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Queues decouple accepting work from doing work. A producer Worker sends a message, the queue holds it, and a consumer Worker processes it in batches. The request path returns as soon as the message is accepted, which is the entire point.",
      },
      { kind: "h3", text: "Producer and consumer" },
      {
        kind: "p",
        text: "Any Worker with a queue producer binding can send messages. The consumer is configured on the queue itself and is just another Worker: it implements a queue handler and receives messages in batches. Batches are acknowledged by returning successfully from the handler.",
      },
      { kind: "h3", text: "Delivery guarantees" },
      {
        kind: "list",
        text: [
          "Delivery is at-least-once. A message can be delivered more than once, so consumers must be idempotent.",
          "A failed batch, an exception or a non-success return, is retried automatically up to a configurable retry count with backoff.",
          "After retries are exhausted, the message is written to a dead letter queue that you configure on the queue.",
          "There is no strict global ordering guarantee. Treat messages as independent events and design accordingly.",
        ],
      },
      {
        kind: "p",
        text: "The dead letter queue is where you inspect and replay what repeatedly failed. An empty DLQ is the signal that retry policy and consumer health are both fine; a growing one means the consumer is rejecting work for a reason worth reading.",
      },
      { kind: "h3", text: "Acknowledge fast, work slow" },
      {
        kind: "p",
        text: "The standard pattern: the request handler validates input, enqueues a message, and returns immediately. The consumer does the slow part, calling third-party APIs, generating exports, sending email. User-facing latency stays flat regardless of the work, traffic spikes land in the queue instead of on the upstream, and a transient upstream failure becomes a retry instead of a failed request. The cost is that the caller no longer knows the outcome, so return an identifier the caller can poll or be notified about.",
      },
      { kind: "h3", text: "Limits" },
      {
        kind: "p",
        text: "Queues carry per-plan limits on message size, batch size, throughput, consumer concurrency, and consumer count. These numbers change with plan and over time, so read the dashboard limits page for current values. The stable part is the shape: per-message size caps, per-batch delivery, and a bounded number of concurrent consumer invocations.",
      },
    ],
  },
  {
    id: "workers-ai-gateway",
    title: "Workers AI and AI Gateway",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Workers AI runs inference on Cloudflare's network. You bind the AI object in your Worker configuration and call a model by name. There are no GPUs to provision and no inference infrastructure to run. Pricing is per model, based on the units that model consumes, such as input and output tokens.",
      },
      { kind: "h3", text: "The model catalog" },
      {
        kind: "p",
        text: "The catalog spans open-weight LLMs, image generation, speech-to-text, translation, and embedding models. Model identifiers look like @cf/meta/llama-3.1-8b-instruct, and new models appear regularly while others deprecate. Treat the dashboard's live model list as the source of truth, not any snapshot of it.",
      },
      { kind: "h3", text: "AI Gateway is a proxy, not a model" },
      {
        kind: "p",
        text: "AI Gateway sits in front of any AI provider, not only Workers AI. You point your existing OpenAI, Anthropic, Google, or AWS Bedrock calls at the gateway endpoint instead of the provider directly, or use the universal endpoint for anything else. Provider API keys can be stored in the gateway, so application code stops carrying them.",
      },
      { kind: "h3", text: "What the gateway adds" },
      {
        kind: "list",
        text: [
          "Response caching for identical prompts, which cuts repeat cost to zero.",
          "Rate limiting, retries, and fallback routing to another provider on failure.",
          "Request and response logging, so prompts and completions are auditable in one place.",
          "Analytics with token counts, cost, and latency per request.",
          "Custom metadata on requests, so usage is attributable to a team, feature, or customer.",
        ],
      },
      { kind: "h3", text: "Why route internal AI traffic through it" },
      {
        kind: "p",
        text: "Without a gateway, AI spend is a provider invoice with no attribution and per-service code holds provider credentials. With one, every call carries its metadata, cost breaks down per team and per feature in the analytics, and the prompt log is a single place to audit. Changing providers becomes a base URL change at the gateway rather than a rewrite in every service that calls a model. Workers AI calls can flow through the gateway too, so the same accounting covers them.",
      },
    ],
  },
  {
    id: "cron-triggers",
    title: "Cron Triggers",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "A Cron Trigger runs a Worker on a schedule. You declare cron expressions in the Worker configuration and implement the scheduled handler. There is no scheduler service to operate and nothing to keep alive; the schedule is part of the deployment.",
      },
      { kind: "h3", text: "Syntax specifics" },
      {
        kind: "p",
        text: "Cloudflare uses the standard five-field Unix cron syntax, evaluated in UTC: minute, hour, day-of-month, month, day-of-week. There is no seconds field and no year field. A Worker can carry multiple triggers, so one deployment can run several schedules.",
      },
      {
        kind: "list",
        text: [
          "\"*/5 * * * *\" runs every five minutes.",
          "\"0 3 * * *\" runs at 03:00 UTC every day.",
          "\"15 2 * * 1\" runs at 02:15 UTC on Mondays.",
        ],
      },
      {
        kind: "p",
        text: "Everything is UTC. Convert deliberately, and remember that a daily job at 03:00 UTC is a different local hour half the year in time zones with daylight saving.",
      },
      { kind: "h3", text: "Execution semantics" },
      {
        kind: "p",
        text: "Scheduled runs are best effort. They fire near the scheduled time, not on it, and there is no built-in retry contract for a failed or missed run. Do not assume a strict no-overlap guarantee within a script: a run that outlasts its interval can still be executing when the next trigger fires. Treat overlap as an expected condition, not an anomaly, and write the handler so two overlapping invocations do not corrupt anything.",
      },
      { kind: "h3", text: "Make runs idempotent" },
      {
        kind: "p",
        text: "Store run state outside the handler, in a Durable Object or KV, and check it before doing work. The durable pattern: derive a window from the scheduled time, key a marker by cron identifier plus window, and read that marker first. If the window was already processed, skip or resume. A Durable Object gives you an atomic check-and-set for the marker. KV gives you a cheap marker with eventual consistency, which suits jobs where a rare double run is harmless but a skipped one is not.",
      },
      {
        kind: "list",
        text: [
          "Key markers by cron identifier plus time window, so multiple schedules on one Worker stay independent.",
          "Write the marker only after the work succeeds, so a failed run is retried on the next trigger instead of skipped.",
          "Alert on a missing marker for an expected window. That catches missed runs, which the platform will not surface for you.",
        ],
      },
      {
        kind: "p",
        text: "The number of triggers allowed per Worker depends on plan; the dashboard limits page has current values. When the work per run grows past what one invocation should do, have the handler enqueue to a Queue or fan out through a Durable Object rather than stretching one run.",
      },
    ],
  },
];
