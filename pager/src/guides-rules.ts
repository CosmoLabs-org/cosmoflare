// Static article data for the Guides section: the Cloudflare rules
// families. Same block contract as guides.ts; the renderer needs no
// changes. Content only, no fetching. Quota figures appear only where
// stable and public; every other limit points at the dashboard limits
// page instead of an invented number.

import type { Guide } from "./guides";

export const RULES_GUIDES: readonly Guide[] = [
  {
    id: "rules-families",
    title: "Rules at Cloudflare: every family in one table",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Cloudflare splits its rules into families. Each family answers one question about a request: where should it go, is it allowed, how should it be cached, what should change on the way through. The table lists every family, its free plan allowance per zone, and where in the request it runs.",
      },
      {
        kind: "table",
        head: ["Family", "What it does", "Free allowance per zone", "Where it runs"],
        rows: [
          ["Page Rules", "Applies per-URL settings, including URL forwarding", "3", "Edge, per URL match (legacy)"],
          ["Redirect Rules", "Sends a matching request to another URL", "10", "Before cache lookup, terminates the request"],
          ["WAF Custom Rules", "Blocks, challenges, or allows traffic by expression", "5", "Before the origin fetch"],
          ["Cache Rules", "Overrides cache behavior per match", "10", "At cache lookup"],
          ["Configuration Rules", "Turns zone settings on or off for matching requests", "10", "Early, before other phases act"],
          ["Rate Limiting Rules", "Counts matching requests over time and acts on excess", "Plan dependent, check the dashboard limits page", "At the edge, over a counting period"],
          ["Origin Rules", "Sets origin fetch properties: host header, SNI, port, DNS", "Plan dependent, check the dashboard limits page", "At the origin fetch"],
          ["Transform Rules", "Rewrites URL parts, request headers, or response headers", "Plan dependent, check the dashboard limits page", "Request phase or response phase"],
          ["Snippets", "Runs a small JavaScript on matching requests", "Plan dependent, check the dashboard limits page", "At the edge, on matched requests"],
        ],
      },
      { kind: "h3", text: "One engine underneath" },
      {
        kind: "p",
        text: "All of these families run on the Ruleset Engine, the shared core Cloudflare uses for its own products too. A rule is a filter expression plus an action, stored in a phase-specific ruleset. The dashboard, the API, and Terraform write to the same rulesets, so nothing is dashboard-only.",
      },
      { kind: "h3", text: "Execution order" },
      {
        kind: "p",
        text: "Rules run in phases, and the phases have a fixed order. A redirect fires before the cache is consulted, so a matched redirect never reaches your origin. WAF custom rules evaluate before the request leaves for your origin. Cache Rules apply when Cloudflare decides what to serve. Origin Rules apply when Cloudflare builds the fetch to your origin. Transform Rules exist in both a request phase and a response phase.",
      },
      { kind: "h3", text: "Quota behavior" },
      {
        kind: "p",
        text: "Each family has a per-zone count limit, and your plan sets the number. Hitting the cap blocks creating more rules in that family. Nothing breaks at runtime: existing rules keep firing, and only new creation stops until you delete one or change plan. For your plan's current numbers, check the Rules limits page in the Cloudflare dashboard.",
      },
    ],
  },
  {
    id: "page-rules-vs-redirect-rules",
    title: "Page Rules vs Redirect Rules: which to use",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Cloudflare ships two products that both redirect traffic. Page Rules are the legacy product. Redirect Rules are the replacement. For new work, use Redirect Rules.",
      },
      { kind: "h3", text: "Page Rules" },
      {
        kind: "p",
        text: "A Page Rule matches a URL pattern and applies a bundle of settings at once: a forwarding URL with a status code, Cache Level including the Cache Everything option, Edge Cache TTL, Disable Performance, and more. Patterns are glob-like, with * matching any sequence of characters. The free plan allows 3 Page Rules per zone, which runs out fast.",
      },
      { kind: "h3", text: "Redirect Rules" },
      {
        kind: "p",
        text: "A Redirect Rule matches on a rules-language expression and redirects to a static or dynamically built target. Expressions read any request field: host, path, query, headers, cookies, country, ASN. The free plan allows 10 redirect rules per zone, and Bulk Redirects handles large static lists as a separate feature.",
      },
      {
        kind: "p",
        text: "Recommendation. Point every new redirect at Redirect Rules. They are more precise, more expressive, and far less quota-starved. Migrate old Page Rules one at a time as you touch them: write the equivalent Redirect Rule, test it, delete the Page Rule, keep the change small.",
      },
      { kind: "h3", text: "The www to apex pattern" },
      {
        kind: "p",
        text: "Redirecting www.example.com to example.com shows the gap between the two. A Page Rule needs one rule per hostname, because its pattern is anchored to a single host: www.example.com/* forwarded to https://example.com/$1. A Redirect Rule matches an expression instead, so one rule covers every www host on the zone.",
      },
      {
        kind: "kv",
        term: "starts_with(http.host, \"www.\")",
        desc: "Matches any host beginning with www., so www.example.com, www.staging.example.com, and every other www subdomain fall under one rule. A Page Rule pattern cannot express this; its wildcard cannot span hostnames.",
      },
    ],
  },
  {
    id: "rate-limiting-rules",
    title: "Rate limiting rules: the first lines of defense",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "A rate limiting rule counts matching requests over a period and acts when the count crosses your threshold. It is the first tool against abuse that no static rule can catch, because each individual request looks harmless.",
      },
      { kind: "h3", text: "Match and counting" },
      {
        kind: "p",
        text: "The match is a rules-language expression, the same language every rule family uses. The counting characteristics sit next to it: a counting period, and a counting key. The key decides what counts as one unit. The client IP is the default key. A request characteristic such as a header or a cookie gives per-session counting. Requests with different keys never share a count.",
      },
      { kind: "h3", text: "Actions" },
      {
        kind: "list",
        text: [
          "Block with a timeout: the client is refused for a fixed duration after the threshold trips.",
          "Managed challenge: the client must pass an interactive challenge before continuing.",
          "Log: the rule records the event and takes no action.",
        ],
      },
      {
        kind: "p",
        text: "Classic use cases: the login path, to slow credential stuffing; API endpoints without server-side throttling of their own; expensive paths such as search or export that an abuser can call in a loop; and form POST endpoints that collect spam.",
      },
      {
        kind: "p",
        text: "The rule of thumb: start with log-only. Watch the logs for a day or two and check that your threshold sits comfortably above your legitimate peak. Then switch the action to challenge or block. A rule that enforces from the first minute locks out real users on the first traffic spike, or on the first scanner arriving from a shared exit IP.",
      },
    ],
  },
  {
    id: "rules-expression-language",
    title: "The rules expression language in five minutes",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Every rule family shares one expression language. Learn it once and you can write redirects, WAF rules, cache overrides, and origin picks without new syntax. An expression is a boolean test over fields of the current request. The result decides whether the rule's action runs.",
      },
      { kind: "h3", text: "Fields" },
      {
        kind: "p",
        text: "Fields are read-only request values. The common ones: http.host is the requested hostname, http.request.uri.path is the path without the query string, http.request.uri.query is the query string, http.user_agent is the User-Agent header. Many more exist: method, individual headers, cookies, ASN, country, IP version, and bot score fields.",
      },
      { kind: "h3", text: "Operators and combination" },
      {
        kind: "p",
        text: "Compare with eq, ne, lt, gt. Test membership with in against a set literal such as {\"a\" \"b\"} or a rule list. Test substrings with contains, starts_with, and ends_with. Combine tests with and, or, and not, and group with parentheses. Precedence is standard, but parentheses make the intent explicit.",
      },
      { kind: "h3", text: "String functions" },
      {
        kind: "p",
        text: "Functions transform a value before comparison: lower and upper for case-insensitive matching, len for length, substring to cut a range, concat to build a value, url_decode to unescape an encoded value. A few functions are plan-gated. The dashboard expression builder validates syntax while you type, so it tells you before deploy.",
      },
      { kind: "h3", text: "Three realistic expressions" },
      {
        kind: "kv",
        term: "http.request.uri.path contains \"/api/\" and http.request.method eq \"POST\"",
        desc: "Every POST whose path contains /api/. A base for rate limiting an API or for blocking write traffic from unknown clients.",
      },
      {
        kind: "kv",
        term: "starts_with(http.user_agent, \"curl\") or http.user_agent eq \"\"",
        desc: "Command-line clients plus requests with no User-Agent header at all. Browsers and mainstream HTTP libraries always send one, so an empty value is usually a script.",
      },
      {
        kind: "kv",
        term: "http.host eq \"example.com\" and http.request.uri.path starts_with \"/blog/\"",
        desc: "Blog paths on the apex host only. The www host carries a different http.host value, so this rule does not match it.",
      },
      {
        kind: "p",
        text: "Write expressions in the dashboard editor or through the API. The editor autocompletes field names and flags syntax errors as you type, which is the fastest way to learn the vocabulary.",
      },
    ],
  },
  {
    id: "snippets-and-transform",
    title: "Snippets and Transform Rules: small code, no Worker",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Between plain rules and a full Worker sit two lighter tools. Transform Rules rewrite request and response attributes through a form, no code. Snippets run a small piece of JavaScript on matched requests without a separate deploy. Both are configured per zone.",
      },
      { kind: "h3", text: "Transform Rules" },
      {
        kind: "p",
        text: "Transform Rules are declarative: you pick a source, an operation, and a target. They can rewrite the URL path, rewrite the query string, and set or modify request headers and response headers. Each rule runs in a fixed phase, so ordering is predictable and there is no runtime to debug. Typical uses: strip tracking query parameters before caching, add a security header, rewrite a legacy path to a new one.",
      },
      { kind: "h3", text: "Snippets" },
      {
        kind: "p",
        text: "A Snippet is a small JavaScript file attached to a match expression. When a request matches, the edge runs the code in place and returns your response, or passes the request through with your modifications. There is no build step and no separate deploy; the dashboard is the deployment. Quotas cap the number of snippets per zone, the snippet size, and the CPU time per request. Storage bindings such as KV, R2, D1, and Durable Objects are not available. For current figures, check the Snippets limits page in the dashboard.",
      },
      {
        kind: "p",
        text: "Typical Snippet uses: a conditional header, a small HTML fix on one path, a cookie tweak, a short synthetic response. The moment the logic needs stored data or grows past a screen, it outgrew the tool.",
      },
      { kind: "h3", text: "When a full Worker is warranted" },
      {
        kind: "list",
        text: [
          "Storage: KV, R2, D1, or Durable Objects bindings. Snippets have none.",
          "Long work: streaming responses, slow upstream calls, or anything near the snippet CPU ceiling.",
          "A real workflow: git, CI, preview deployments, staged rollouts, and rollbacks.",
          "Observability: structured logs, metrics, and tail-based debugging.",
        ],
      },
      {
        kind: "p",
        text: "Rule of thumb: if the logic fits in a few dozen lines and only touches one zone's headers, URLs, or cookies, start with Transform Rules or a Snippet. Move to a Worker when you need storage, complex branching across many routes, or a test suite.",
      },
    ],
  },
];
