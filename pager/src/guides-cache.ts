import type { Guide } from "./guides";

// Cache education series for the Guides section: static article data only.
// Guidance prose, no usage claims, so the "real numbers only" rule does not
// apply. Facts reflect stable public Cloudflare behavior as of 2025-2026.

export const CACHE_GUIDES: readonly Guide[] = [
  {
    id: "edge-cache-explained",
    title: "How the edge cache works",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Cloudflare runs data centers in hundreds of cities. Engineers call each one a colo, a colocation data center that acts as the local point of presence. Anycast routing sends every visitor to the nearest colo, and each colo keeps its own copy of your cache. The same URL can be a HIT in Frankfurt and a MISS in Tokyo at the same minute.",
      },
      { kind: "h3", text: "The request path" },
      {
        kind: "list",
        text: [
          "The browser resolves your hostname and lands on the nearest colo.",
          "The colo checks its cache under the URL's cache key.",
          "On a HIT, the colo serves the stored response and the request ends there.",
          "On a MISS, the colo fetches from your origin, returns the response, and stores a copy for the TTL.",
        ],
      },
      {
        kind: "p",
        text: "A HIT does no origin work at all: no origin bandwidth, no origin compute, no origin latency. The visitor gets the response from hardware a few milliseconds away. If a Workers route matches the request, the Worker runs before the cache lookup by default, so a HIT inside its fetch() saves the origin subrequest while the invocation itself is still billed. Requests served by Workers static assets bypass the Worker entirely and are never billed as invocations.",
      },
      { kind: "h3", text: "Reading cf-cache-status" },
      {
        kind: "p",
        text: "Every response carries a cf-cache-status header. It is the ground truth for what the edge actually did. Decode it like this.",
      },
      {
        kind: "table",
        head: ["Status", "Meaning"],
        rows: [
          ["HIT", "Served from this colo's cache. The origin was not contacted."],
          ["MISS", "Not cached here. Fetched from origin and stored for next time."],
          ["DYNAMIC", "Deemed not eligible for caching. Served without storing anything."],
          ["EXPIRED", "Was cached but past its TTL. Refetched from origin and stored again."],
          ["BYPASS", "Cache skipped on purpose: a bypass rule, a no-store directive, or a Set-Cookie on the response."],
          ["REVALIDATED", "Stale object confirmed unchanged by a conditional origin request (304). Cheaper than EXPIRED."],
          ["STALE", "Stale copy served, for example while revalidation is in progress or the origin could not confirm freshness."],
          ["UPDATING", "Stale copy served while the fresh copy is being fetched in the background."],
        ],
      },
      {
        kind: "p",
        text: "DYNAMIC on a URL you expected to cache is the most useful signal in the header. It means eligibility failed before any TTL was read, which is the subject of the next article.",
      },
    ],
  },
  {
    id: "html-cache-trap",
    title: "Why s-maxage on HTML does nothing by default",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "By default Cloudflare decides cache eligibility by file extension. A static list covers css, js, images, fonts and similar asset types. HTML, JSON APIs and extensionless routes are not on that list. Eligibility is decided first. The edge reads your Cache-Control headers only after a response is already eligible.",
      },
      {
        kind: "p",
        text: "That ordering is the trap. Your origin sends Cache-Control: public, s-maxage=86400 on a text/html document. The header is real, curl shows it, the deploy log shows it. The edge still answers DYNAMIC, because an HTML response is never eligible under the default rules and s-maxage never gets read. The fix looks shipped and does nothing. Hit rate stays flat and origin keeps paying for every request.",
      },
      { kind: "h3", text: "The two real fixes" },
      {
        kind: "list",
        text: [
          "Zone Cache Rule: match the HTML paths by hostname and URI path, set Eligible for cache. With the default Edge TTL mode (respect origin), the s-maxage you already emit starts working with no other change.",
          "Worker-side caching: a Worker returning its own Response is not written to the zone cache automatically, s-maxage or not. Cache it explicitly with the Cache API (caches.default.put), or add a Cache Rule that makes the path eligible and let the normal path work.",
        ],
      },
      {
        kind: "p",
        text: "The Cache Rule route is usually the smaller change. It is one object in the zone, it needs no redeploy, and it activates headers you already send.",
      },
      { kind: "h3", text: "How to verify you were trapped" },
      {
        kind: "p",
        text: "Run curl -sI on an HTML URL and read two headers. Cache-Control proves the origin intent. cf-cache-status proves what the edge actually did. DYNAMIC next to a correct s-maxage is the trap, confirmed.",
      },
      {
        kind: "p",
        text: "Each colo caches independently, so a fresh deployment can show MISS in one city and HIT in another. DYNAMIC, by contrast, is uniform everywhere. That makes it a reliable verdict on eligibility, not on local cache state.",
      },
    ],
  },
  {
    id: "cache-control-layers",
    title: "Cache-Control and TTLs: which layer wins",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "A single response can be cached in two independent places: the visitor's browser and the Cloudflare edge. Each layer reads its own directives from the same Cache-Control family, and each layer can be overridden by a zone Cache Rule.",
      },
      { kind: "kv", term: "max-age", desc: "Browser directive. Seconds the browser may reuse the response without asking anyone." },
      { kind: "kv", term: "private, no-store", desc: "Browser directives. private keeps shared caches out. no-store forbids storage everywhere." },
      { kind: "kv", term: "s-maxage", desc: "Shared-cache directive. Sets the edge TTL for responses already cache eligible. Falls back to max-age when absent." },
      { kind: "kv", term: "CDN-Cache-Control", desc: "Cloudflare-supported header that targets the edge only. It overrides s-maxage and max-age at the edge while leaving the browser directive untouched." },
      { kind: "h3", text: "Precedence" },
      {
        kind: "list",
        text: [
          "At the edge, CDN-Cache-Control wins over s-maxage, which wins over max-age.",
          "A zone Cache Rule Edge TTL set to ignore origin headers overrides all of them for matched requests.",
          "A Cache Rule Browser TTL similarly overrides the browser-facing header for matched requests.",
          "The browser never sees Cache Rules. It obeys whatever header finally reaches it.",
        ],
      },
      { kind: "h3", text: "The asymmetry that sets your TTLs" },
      {
        kind: "p",
        text: "Edge cache can be purged programmatically: purge by URL, purge everything, or purge by prefix and cache key on higher plans. Browser cache cannot be purged at all. Once a response is on the visitor's machine it stays until its TTL expires, whatever you deploy.",
      },
      {
        kind: "p",
        text: "This asymmetry drives the standard layout. Keep browser max-age short for documents, since you cannot recall them. Keep edge TTL long, since purge-on-deploy is one API call. Hashed, immutable assets are the exception: long max-age is safe there because each deploy produces new URLs.",
      },
    ],
  },
  {
    id: "cache-rules-vs-headers",
    title: "Cache Rules vs origin headers",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Origin headers and Cache Rules are two control surfaces over the same edge cache. Headers describe intent per response and travel with each deploy. Rules override that intent per request match, from the zone side, without touching your code.",
      },
      { kind: "h3", text: "What a Cache Rule can do that headers cannot" },
      {
        kind: "list",
        text: [
          "Force eligibility. HTML and extensionless routes stay DYNAMIC no matter what headers say. Only a rule, or explicit Cache API writes, makes them cacheable.",
          "Override Edge TTL. Ignore origin headers entirely and pin a fixed edge TTL for matched paths.",
          "Override Browser TTL. Pin a browser lifetime for matched paths, independent of what the origin emits.",
          "Reshape the cache key. Include or ignore query parameters, and on higher plans headers or cookies. No response header can merge cache entries; that is key surgery, and rules own it.",
          "Bypass deliberately. Keep authenticated or volatile paths out of cache even when their headers claim to be cacheable.",
        ],
      },
      { kind: "h3", text: "When headers alone are enough" },
      {
        kind: "p",
        text: "If a response is already cache eligible, which for a default zone means its extension is on the static asset list, correct headers are the whole job. Emit max-age, or s-maxage when the edge TTL should differ from the browser TTL, and let it ride. A rule that repeats what headers already do is one more object to keep in sync.",
      },
      { kind: "h3", text: "When a rule is the only tool" },
      {
        kind: "p",
        text: "Three cases force a rule. HTML or JSON routes that must become cacheable. Query strings that must not fragment the cache, such as analytics parameters on otherwise identical assets. Per-path TTLs on an origin you cannot redeploy, for example a vendor-served asset path.",
      },
      {
        kind: "p",
        text: "The healthy split: headers carry freshness, rules carry exceptions. When both try to carry freshness for the same path, the rule wins silently, and the header becomes documentation that lies.",
      },
    ],
  },
  {
    id: "hit-rate-cost-lever",
    title: "Hit rate is a cost lever",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Every cache MISS pays twice: the visitor waits on origin latency, and the platform pays for the work the edge skipped. A HIT pays neither. On a traffic-heavy zone, the ratio between the two is the largest single cost dial you own.",
      },
      { kind: "h3", text: "What a MISS bills" },
      {
        kind: "list",
        text: [
          "Origin bandwidth and origin compute for the refetch.",
          "A Worker invocation for every request on a matched route, whether the inner fetch hits cache or not.",
          "D1 row reads when the origin path queries the database on every request.",
          "Repeated image transformations or API calls that could have been stored once and reused.",
        ],
      },
      {
        kind: "p",
        text: "The invocation happens before the cache lookup when a route matches, so a HIT inside the Worker's fetch() saves the origin subrequest but not the invocation. Moving hot paths off routes, or serving them as static assets, is what removes the invocation itself.",
      },
      { kind: "h3", text: "The practical ladder, cheapest step first" },
      {
        kind: "list",
        text: [
          "1. Emit Cache-Control on every cacheable response. Zero rules, zero cost, immediate HITs on assets.",
          "2. Confirm static assets are covered. They are eligible by extension, so a correct max-age is all they need.",
          "3. Add Cache Rules for HTML paths. Eligibility plus a respect-origin Edge TTL activates the s-maxage you already send.",
          "4. Measure per zone, before and after. Use the cache status breakdown in zone analytics or the cacheStatus dimension in GraphQL analytics, and track the HIT share over a full week.",
        ],
      },
      {
        kind: "p",
        text: "The measurement step is not optional. A caching change that raises the HIT share is real. One that leaves cf-cache-status on DYNAMIC for the paths you meant to fix is the HTML trap from the earlier article, and only the metrics show it.",
      },
      {
        kind: "p",
        text: "One benchmark to remember: a near-zero hit rate on a traffic-heavy zone is the single largest efficiency finding an audit can surface. Nothing else on the bill moves per-request cost by that ratio.",
      },
    ],
  },
];
