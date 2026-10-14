// Static article data for the Guides section (FEAT-pDQEFQV, FEAT-pAGY842):
// plain-language operator guidance — guidance prose only, never usage
// claims, so the "real numbers only" rule does not apply. No fetching:
// this module is the data.
//
// The education hub (ROAD-p9X66J9) adds topic sets authored in parallel —
// one file per topic, concatenated here. New articles follow the operator's
// style rules: direct, technical, no em-dashes.

import { RULES_GUIDES } from "./guides-rules";
import { CACHE_GUIDES } from "./guides-cache";
import { COMPUTE_GUIDES } from "./guides-compute";
import { STORAGE_GUIDES } from "./guides-storage";
import { NETWORK_GUIDES } from "./guides-network";
import { SECURITY_GUIDES } from "./guides-security";

/** One block of article body prose. Paragraphs, sub-headings and bullet
 *  lists carry text; kv rows carry a term + desc pair rendered with the
 *  cf-domain-field pattern; tables carry a head row plus string rows for
 *  reference material like rule-family quotas. */
export type GuideBlock =
  | { kind: "p" | "h3"; text: string }
  | { kind: "list"; text: string[] }
  | { kind: "kv"; term: string; desc: string }
  | { kind: "table"; head: string[]; rows: string[][] };

/** One guide article. `id` is the hash-route target (#/guides/<id>);
 *  `source`, when present, is the canonical Cloudflare docs page the
 *  article draws on. */
export interface Guide {
  id: string;
  title: string;
  updated: string;
  body: GuideBlock[];
  source?: string;
}

/** The grant path, referenced twice in article 1 — one constant so it
 *  cannot drift between the intro and the close. */
const GRANT_PATH = "My Profile → API Tokens → Edit";

const BASE_GUIDES: readonly Guide[] = [
  {
    id: "permissions",
    title: "Token permissions: what Ops can see",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Ops reads through your account API token. Each section is as deep as the scopes on that token allow — a section that needs a scope the token lacks falls back to estimates or hides rows. Grant one scope at a time and watch the section fill in. Grant any of these in the dashboard under " + GRANT_PATH + ".",
      },
      { kind: "kv", term: "Billing Read", desc: "Subscription-true billing periods and usage-v2 numbers — without it the app falls back to anchor-day estimates." },
      { kind: "kv", term: "Email Routing Read", desc: "Bounce and delivery metrics." },
      { kind: "kv", term: "Registrar Read", desc: "Authoritative domain expiry, auto-renew and transfer lock." },
      { kind: "kv", term: "Workers Scripts Read", desc: "Bindings, cron triggers, routes and versions on worker profiles." },
      { kind: "kv", term: "Rulesets / WAF Read", desc: "Rule inventory and the quota gauges in the WAF section." },
      { kind: "kv", term: "D1 Read", desc: "Per-database usage detail." },
      {
        kind: "p",
        text: "One trap: the R2 data plane does not use the account token. The CLI's S3 traffic authenticates with a separate 32-character access key pair (R2 → Manage R2 API Tokens). That is a different token, not a scope — granting account scopes never covers it.",
      },
    ],
  },
  {
    id: "waf-starter-rules",
    title: "WAF starter rules: first week",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Your first week with the WAF, in plain language. One move at a time, and verify each one in Ops before the next.",
      },
      {
        kind: "h3",
        text: "1. Turn on the managed ruleset at default sensitivity",
      },
      { kind: "p", text: "What: deploy Cloudflare's managed ruleset with its default sensitivity, changing nothing. Why: it carries the current exploit protection so you start covered on day one. Verify: the WAF section in Ops shows the rule inventory and the quota gauges — confirm the managed count appears against your plan's quota." },
      {
        kind: "h3",
        text: "2. Add one custom rule at a time, and watch the count",
      },
      { kind: "p", text: "What: add a single custom rule, then let it run before adding another. Why: when something breaks you know which rule did it, and every rule eats plan quota. Verify: the WAF section's quota gauges track the count after each addition." },
      {
        kind: "h3",
        text: "3. Use rule lists for repeat offenders",
      },
      { kind: "p", text: "What: put repeating IP addresses into a rule list and match the list, instead of growing a long expression. Why: lists are one object to update and do not bloat a rule toward the expression-size limit. Verify: the rule inventory in Ops shows the list-based rule as one entry, not a wall of conditions." },
      {
        kind: "h3",
        text: "4. Rate-limit the login and API paths first",
      },
      { kind: "p", text: "What: put rate-limiting rules on the login and API endpoints before anything else. Why: they are the usual abuse vector — credential stuffing and API hammering. Verify: alert rules on rule counts in Ops tell you when the inventory changes without you watching it." },
      {
        kind: "h3",
        text: "5. Never put a permanent allow for your own IP on a shared connection",
      },
      { kind: "p", text: "What: prefer temporary or scoped exceptions for your own addresses. Why: on a shared connection the IP is not yours alone — a permanent allow exempts everyone on it, permanently. Verify: the rule inventory in Ops makes the standing allow visible so it cannot be forgotten." },
    ],
  },
];

export const GUIDES: readonly Guide[] = [
  ...BASE_GUIDES,
  ...RULES_GUIDES,
  ...CACHE_GUIDES,
  ...COMPUTE_GUIDES,
  ...STORAGE_GUIDES,
  ...NETWORK_GUIDES,
  ...SECURITY_GUIDES,
];
