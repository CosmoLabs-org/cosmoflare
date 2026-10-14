import type { Guide } from "./guides";

// Traffic management education series for the Guides section: static
// article data only. Guidance prose, no usage claims, so the "real
// numbers only" rule does not apply. Facts reflect stable public
// Cloudflare behavior as of 2025-2026. Where a quota or price varies
// by plan, the text says so and points at the dashboard instead of
// quoting a number.

export const TRAFFIC_GUIDES: readonly Guide[] = [
  {
    id: "load-balancing",
    title: "Load Balancing",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/load-balancing/",
    body: [
      {
        kind: "p",
        text: "Load Balancing is Cloudflare's managed traffic director. The hostname resolves to Cloudflare, and each query returns an address from a healthy pool according to your steering policy. You define pools of origin servers, attach health monitors, and order pools for failover. It is a paid add-on, so the closing question of this article matters: does your topology need it.",
      },
      { kind: "kv", term: "Pool", desc: "A named group of origin servers, each with its address, port and weight. A pool receives traffic only while it is healthy." },
      { kind: "kv", term: "Monitor", desc: "A probe definition, HTTP, HTTPS or TCP, applied to a pool's origins. Monitor results set origin health, and origin health sets pool health." },
      { kind: "kv", term: "Steering policy", desc: "The rule for choosing an origin inside a pool and a pool among pools. Pool priority order drives failover." },
      { kind: "h3", text: "Steering policies" },
      {
        kind: "list",
        text: [
          "Round Robin: origins take turns. The right default when origins are identical.",
          "Weighted Round Robin: origins take turns in proportion to a weight you set. Use it when capacity is uneven across the pool.",
          "Least outstanding requests: each new request goes to the origin with the fewest requests in flight. Use it when request cost varies.",
          "Latency steering: each query goes to the pool with the lowest measured response time for the visitor's region. Use it when pools live in different regions.",
        ],
      },
      { kind: "h3", text: "Failover across pools" },
      {
        kind: "p",
        text: "Pools have a priority order. The balancer answers from the first healthy pool and drops to the next when the first fails. The switch happens per query at the edge, in seconds, not at DNS TTL speed. Set a grace period so a pool that recently failed probes keeps serving while it recovers, and mark a fallback pool for the case where every prioritized pool is down.",
      },
      { kind: "h3", text: "Session affinity" },
      {
        kind: "p",
        text: "Affinity pins a visitor to one origin for a configurable duration. Modes are a Cloudflare-issued cookie, an IP-based cookie, or pass-through of the origin's own session cookie. Use it whenever origins hold server-side state. Accept the trade: pinned traffic does not rebalance until sessions expire or an origin dies, so the pool carries more variance than pure steering would.",
      },
      { kind: "h3", text: "When plain DNS failover is enough" },
      {
        kind: "list",
        text: [
          "One region, one cloud, one address per hostname. A DNS-level failover at your DNS provider covers the whole failure mode you actually have.",
          "Failure tolerance measured in minutes. Cached DNS answers keep flowing past a failure until the TTL expires, and that is acceptable for many sites.",
          "Cost-constrained setups where the outage mode is rare and manual recovery is fine.",
        ],
      },
      {
        kind: "p",
        text: "Load Balancing earns its cost when pools span regions or clouds, when you want per-request steering instead of per-record, when failover must complete in seconds, or when sessions must survive origin changes. If none of those apply, a simple DNS failover is the honest answer.",
      },
    ],
  },
  {
    id: "waiting-rooms",
    title: "Waiting Rooms",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/waiting-rooms/",
    body: [
      {
        kind: "p",
        text: "A Waiting Room holds excess visitors in a queue at the edge instead of forwarding them to an origin that cannot take the load. Traffic keeps flowing at a rate your origin can absorb, and everyone else sees a position in line. The room runs entirely at the edge, so no origin change and no client change is needed.",
      },
      { kind: "h3", text: "How the queue works" },
      {
        kind: "list",
        text: [
          "Visitors queue in first-in-first-out order. A first-party cookie holds each visitor's position across requests, so reloading does not lose the spot.",
          "Queued visitors receive an HTML page showing their position and an estimated wait time. The template is yours to restyle.",
          "Admission is metered: the room lets new visitors in at a configured rate and holds total active users at or under a configured ceiling.",
          "Dynamic rules can adjust admission to measured traffic instead of holding to fixed numbers for the whole event.",
        ],
      },
      { kind: "kv", term: "Total active users", desc: "The ceiling on simultaneous active sessions. Everyone above it queues." },
      { kind: "kv", term: "New users per minute", desc: "The admission rate. This is the lever that actually protects the origin during a spike." },
      { kind: "kv", term: "Queue all", desc: "An event mode that queues everyone and admits visitors on your schedule. Fits launches and ticket drops." },
      { kind: "kv", term: "Session duration", desc: "How long an admitted visitor stays counted as active after their last request." },
      { kind: "h3", text: "Enable before the launch, not during" },
      {
        kind: "p",
        text: "The room must exist and cover the hostname before demand arrives. The hostname has to be proxied through Cloudflare and matched by the room's rules in advance, because attaching a room to a live site mid-incident leaves a window of unprotected traffic and a cold queue with no order. The operating pattern is simple: define the room against normal traffic, keep queueing disabled day to day, and enable queueing for the event window. Rehearse the toggle.",
      },
      {
        kind: "p",
        text: "Limits such as room counts and session ceilings vary by plan. Check the Waiting Room limits page in the dashboard before you commit to a design, since the numbers change and the dashboard is authoritative.",
      },
    ],
  },
  {
    id: "argo-tiered-cache",
    title: "Argo Smart Routing and Tiered Cache",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/cache/how-to/tiered-cache/",
    body: [
      {
        kind: "p",
        text: "Argo Smart Routing is a paid add-on that changes how traffic moves through Cloudflare's own network. Instead of fixed public-internet paths, Cloudflare picks routes from continuous performance measurements between its colos, and applies TCP-level optimizations on the chosen path. Origin fetches and requests that transit multiple colos both benefit.",
      },
      {
        kind: "p",
        text: "The route choice is per request and per destination, from live measurements, so the fastest path today is not assumed to be the fastest path tomorrow. The gain depends on geography. An origin near most of your visitors sees little benefit. An origin far from most of your traffic sees the most. Billing is a subscription plus a per-GB charge on traffic served through the feature, so the fee scales with what actually uses it.",
      },
      { kind: "h3", text: "Tiered Cache" },
      {
        kind: "p",
        text: "Tiered Cache changes what a cache miss costs your origin. Without it, every colo that misses fetches from origin directly. With it, lower-tier colos fetch from an upper-tier colo in their region, and only the upper tier fetches from origin. One origin pull per region serves the whole region. The feature is free to enable, in the Caching section of the dashboard.",
      },
      {
        kind: "p",
        text: "Smart Tiered Cache Topology picks the upper tiers for you. The Generic topology lets you pin specific colos as the upper tier. Smart is the right choice for almost everyone. The visible win is fewer origin requests and fewer simultaneous origin connections, which matters most when origin bandwidth or connection limits are the constraint.",
      },
      {
        kind: "table",
        head: ["", "Argo Smart Routing", "Tiered Cache"],
        rows: [
          ["What it changes", "The path requests take between Cloudflare colos and to your origin", "Which colo fetches from your origin on a miss"],
          ["Mechanism", "Live measurements pick the fastest path, plus TCP optimizations", "Lower tiers fetch from a regional upper tier, which fetches from origin once"],
          ["Cost", "Paid add-on, subscription plus a per-GB fee", "Free"],
          ["Enable when", "Visitors and origins sit far apart and slow origin fetches are measurable", "Many colos serve the same content and origin pulls add up"],
        ],
      },
      {
        kind: "p",
        text: "The two compose. Smart Tiered Cache Topology is designed to work with Argo, and together they cut both origin load and inter-colo latency. If you enable only one, Tiered Cache is usually the first move: it is free and it attacks origin load directly, while Argo is a latency purchase you should justify with your own measurements.",
      },
    ],
  },
  {
    id: "cache-reserve",
    title: "Cache Reserve",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/cache/how-to/cache-reserve/",
    body: [
      {
        kind: "p",
        text: "The edge cache holds a copy per colo and evicts under pressure. Long-tail content, objects requested rarely, falls out first, and every eviction becomes an origin pull on the next request. Cache Reserve adds a persistent storage tier behind the edge cache, backed by R2. Misses check Cache Reserve before reaching origin, and fills from origin keep a copy there.",
      },
      { kind: "h3", text: "What changes for hit rate" },
      {
        kind: "list",
        text: [
          "Objects survive edge eviction. A rarely requested file stays reachable at the edge tier instead of forcing an origin pull after every quiet period.",
          "Your cache rules still decide eligibility and freshness. Reserve extends reach, not the freshness logic you already configured.",
          "It runs with Tiered Cache, specifically Smart Tiered Cache Topology. Enable the hierarchical cache alongside it so fills concentrate at the upper tier.",
        ],
      },
      { kind: "h3", text: "Cost model" },
      {
        kind: "p",
        text: "Billing follows R2: storage for the bytes held, plus Class A operations on writes and Class B operations on reads. There is no separate Cache Reserve line item. The number to watch is steady-state storage, because long-tail content accumulates and never leaves on its own.",
      },
      { kind: "h3", text: "The decision" },
      {
        kind: "list",
        text: [
          "Worth it when origin pulls are the expensive part: origin egress fees, bandwidth caps, or an origin that slows down under load.",
          "Not worth it when the content set is small and hot. The edge cache alone covers it, and reserve storage bills you for little benefit.",
          "Check the analytics first. The dashboard reports reserve hit rate and estimated origin savings, which turn the decision into arithmetic instead of taste.",
        ],
      },
      {
        kind: "p",
        text: "Run it for a month on one zone before rolling it out widely. Storage cost is predictable from your content set, but the hit-rate lift depends on your traffic shape, and only your own numbers settle that.",
      },
    ],
  },
  {
    id: "health-checks",
    title: "Health checks and origin monitoring",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/load-balancing/health-checks/",
    body: [
      {
        kind: "p",
        text: "Health checks are the feedback loop that makes pools and failover real. A monitor defines the probe, an origin is the target, and the results drive both load balancing decisions and alerting. Monitors work with or without a Load Balancing deployment, and the two arrangements differ in purpose.",
      },
      { kind: "h3", text: "Standalone versus LB-attached" },
      {
        kind: "table",
        head: ["", "Standalone health checks", "LB-attached monitors"],
        rows: [
          ["Created", "On their own, against any endpoint they can reach", "With a pool, against that pool's origins"],
          ["Purpose", "Visibility and alerting, with no steering involved", "Health state directly drives which origins receive traffic"],
          ["Counts", "Quotas vary by plan; check the dashboard limits page", "Included with Load Balancing, one monitor applied per origin"],
        ],
      },
      { kind: "h3", text: "What a monitor checks" },
      {
        kind: "list",
        text: [
          "Protocol and request: HTTP, HTTPS or TCP, with a path, method, headers and the response codes that count as healthy.",
          "Timing: the check interval, the timeout, and how many consecutive failures flip an origin to unhealthy.",
          "Content assertions: an expected response body substring, for checks that must catch an origin that answers 200 with a broken page.",
        ],
      },
      { kind: "h3", text: "Notifications" },
      {
        kind: "p",
        text: "Cloudflare Notifications can watch health status and deliver through email, webhooks and third-party paging integrations. Keep the two concerns separate: steering is automatic, alerting is not. Without a notification policy attached, a dead origin simply disappears from rotation and nobody hears about it until traffic lands somewhere else.",
      },
      { kind: "h3", text: "Health in steering" },
      {
        kind: "p",
        text: "A pool is healthy while enough of its origins pass the monitor. The balancer steers away from unhealthy pools on its own, and session affinity breaks for visitors pinned to a failed origin. Combine this with the grace period described in the load balancing article, so a probe flap does not evict a pool that still serves. Simulated health states let you test failover behavior without killing real origins.",
      },
      {
        kind: "p",
        text: "Keep monitor paths cheap. The probe hits your origin at the interval you set, so point it at a lightweight endpoint that still exercises the critical dependency. A check that touches the database on every probe is a load source of its own.",
      },
    ],
  },
];
