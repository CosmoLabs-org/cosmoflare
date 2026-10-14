import type { Guide } from "./guides";

// Platform-level articles for the Guides section (education hub,
// ROAD-p9X66J9): DDoS protection, bot management, Zero Trust Gateway,
// Spectrum and account structure. Prose only, no usage claims, so the
// "real numbers only" rule does not apply. Same block model as guides.ts;
// this module is the data. Plan-specific quotas are deliberately not
// quoted: articles describe the concept and point at the dashboard.

export const PLATFORM_GUIDES: readonly Guide[] = [
  {
    id: "ddos-protection",
    title: "DDoS protection: what is on by default",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/ddos-protection/",
    body: [
      {
        kind: "p",
        text: "Every plan on Cloudflare includes DDoS mitigation by default. It is always on, it is unmetered, and it does not bill you when an attack arrives. There is nothing to enable and no capacity to run out of.",
      },
      { kind: "h3", text: "Network layer (L3/L4)" },
      {
        kind: "p",
        text: "Volumetric floods that target addresses and ports never reach your origin. Cloudflare's network-layer managed rulesets drop or absorb this traffic at the edge, before it consumes anything you pay for. These rulesets are automatic. You cannot disable them or tune their sensitivity from the dashboard.",
      },
      { kind: "h3", text: "Application layer (L7)" },
      {
        kind: "p",
        text: "HTTP floods are handled by the managed DDoS ruleset for HTTP traffic. It carries rules that match known attack patterns, such as floods against a single path or with a single user agent. On paid plans you can override individual rules: lower the sensitivity or change the action. Free zones take the defaults. Check the dashboard's DDoS page for what your plan exposes before planning around overrides.",
      },
      {
        kind: "p",
        text: "Treat sensitivity overrides as a last resort. Lower sensitivity means the ruleset tolerates more traffic before it acts, which is exactly what an attacker wants. If legitimate users are caught by a rule, prefer a scoped allow rule in the WAF over a global sensitivity change.",
      },
      { kind: "h3", text: "What you actually configure" },
      {
        kind: "list",
        text: [
          "Alerts. Dashboard notifications can tell you when an attack against your zone is detected or mitigated, so you learn about floods from a notification, not from users.",
          "WAF custom rules. Application-layer abuse that is not a volumetric flood belongs in scoped rules that challenge or block the offending pattern.",
          "Rate limiting. Login endpoints and APIs take sustained single-source abuse that DDoS rulesets ignore. Rate limiting rules are the right tool there.",
          "Origin hygiene. Keep the origin address out of public DNS and accept traffic only from Cloudflare addresses. Edge mitigation does not help an attacker who found the origin directly.",
        ],
      },
      {
        kind: "p",
        text: "Verify after an incident in Security Events, which records mitigated requests and the ruleset that acted on them. If a rule caught legitimate traffic, that log is where you will see it.",
      },
    ],
  },
  {
    id: "bot-management",
    title: "Bot management basics",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/bots/",
    body: [
      {
        kind: "p",
        text: "Cloudflare classifies every request it proxies as it passes the edge. The classification lands in two places: a numeric bot score, and a set of flags for verified bots. Both feed analytics, and both can drive rules, depending on your plan.",
      },
      { kind: "h3", text: "The bot score" },
      {
        kind: "p",
        text: "The score runs from 1 to 99. Low scores mean the request looks automated, high scores mean it looks human. The score is computed per request from signals such as the TLS fingerprint, header order, and behavioral traits, so the same client can score differently across requests. It is a probability, not a verdict.",
      },
      { kind: "h3", text: "Verified bots" },
      {
        kind: "p",
        text: "Search engine crawlers and other legitimate automation are checked against a verified list, normally by reverse DNS lookup. A verified Googlebot is automation you almost never want to block. Verification exists so your rules can treat it separately from scraper traffic that merely claims a browser user agent.",
      },
      { kind: "h3", text: "Fields you can match in rules" },
      {
        kind: "list",
        text: [
          "cf.client.bot: a boolean, true when the request comes from a verified bot. The simplest and most portable bot field.",
          "cf.verified_bot_category: the category of the verified bot, such as search engine crawler or monitoring agent.",
          "cf.bot_score: the 1 to 99 score, available in custom rules on plans that include Bot Management. Lower plans get category-based protection through Bot Fight Mode or Super Bot Fight Mode instead. The dashboard shows what your plan exposes.",
        ],
      },
      { kind: "h3", text: "The practical ladder" },
      {
        kind: "list",
        text: [
          "Start by logging. Where your plan supports a log action for bot traffic, leave it there for a week and read the results in Security Events before blocking anything.",
          "Separate verified bots first. An expression that allows cf.client.bot ahead of your bot rules keeps search indexing working.",
          "Challenge mid-range scores. A managed challenge on likely-automated traffic catches most scrapers without touching humans.",
          "Block only at the extremes. Reserve block for definitely-automated traffic on sensitive paths, after the log data supports it.",
        ],
      },
      {
        kind: "p",
        text: "Whatever the plan, the loop is the same: observe in Security Events, tighten one rule at a time, and keep search and monitoring traffic out of the blast radius.",
      },
    ],
  },
  {
    id: "zero-trust-gateway",
    title: "Gateway and the Secure Web Gateway story",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/cloudflare-one/policies/gateway/",
    body: [
      {
        kind: "p",
        text: "Gateway is the egress half of Cloudflare Zero Trust. It filters the traffic your devices send out: which destinations they can reach, and what they can download. Access, by contrast, controls inbound traffic, deciding who may reach an application you published. One filters destinations, the other gates identity.",
      },
      { kind: "h3", text: "Three filter points" },
      {
        kind: "table",
        head: ["Policy type", "What it inspects", "Needs"],
        rows: [
          ["DNS", "Domain names being resolved, matched against security categories", "A resolver pointed at Gateway, nothing more"],
          ["Network", "Raw connections by IP, port, protocol and ASN", "The WARP client routing the traffic"],
          ["HTTP", "Full URLs, categories, uploads and downloads, file contents", "The WARP client proxying the traffic, TLS decrypted"],
        ],
      },
      { kind: "h3", text: "DNS filtering" },
      {
        kind: "p",
        text: "DNS policies are the cheapest control. The resolver sees the name being requested, matches it against categories such as malware or phishing, and refuses to answer for blocked names. No traffic is intercepted, so it is fast and privacy-light. But it only sees hostnames: a request that hardcodes an address, or a tool that ignores the system resolver, passes straight through.",
      },
      { kind: "h3", text: "HTTP filtering" },
      {
        kind: "p",
        text: "HTTP policies see the full request, which means URL paths, response types, and file transfers. That is where antivirus scanning, upload and download controls, and data loss prevention live. The cost is TLS decryption: the device trusts Cloudflare to terminate and re-encrypt the session, which needs a managed certificate on the fleet.",
      },
      { kind: "h3", text: "Device client versus WARP-less modes" },
      {
        kind: "p",
        text: "The WARP client, registered to your Zero Trust team, routes traffic through Cloudflare and evaluates every policy type. Without the client you still have options: point resolvers at Gateway's DNS endpoints for DNS-only filtering, or use a browser proxy auto-config file where your deployment supports it. WARP-less setups trade coverage for convenience, and network and HTTP policies lose most of their reach.",
      },
      { kind: "h3", text: "Where Gateway sits next to Access" },
      {
        kind: "p",
        text: "A rule of thumb: Access answers who may enter, Gateway answers where users may go. A common pairing puts Access in front of internal applications and Gateway DNS policies on every device as the baseline, with HTTP policies added where inspection is worth the decryption cost.",
      },
    ],
  },
  {
    id: "spectrum",
    title: "Spectrum: proxying TCP and UDP",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/spectrum/",
    body: [
      {
        kind: "p",
        text: "Cloudflare's standard proxy speaks HTTP. Spectrum speaks everything else: it carries arbitrary TCP and UDP services through Cloudflare's network. SSH, RDP, game servers, and other non-HTTP protocols get an edge address, DDoS mitigation, and origin hiding, the same treatment web traffic gets from the orange cloud.",
      },
      { kind: "h3", text: "How an app is exposed" },
      {
        kind: "p",
        text: "A Spectrum application points at an origin, by hostname or address, and listens on the port or port range you configure. Clients connect to a Cloudflare edge address for the app, and Spectrum forwards the connection to your origin. The protocol is passed through untouched. Spectrum is a TCP and UDP proxy, not an interpreter.",
      },
      { kind: "h3", text: "Proxy mode versus direct mode" },
      {
        kind: "p",
        text: "Proxy mode is the default and the useful one: DNS and client connections land on Cloudflare, traffic is forwarded to the origin, and the origin address stays private behind edge protection. Direct mode publishes your origin address in DNS and connections skip the edge, so protection does not apply. Run direct mode only for a deliberate reason, such as testing, and expect no mitigation on that traffic.",
      },
      { kind: "h3", text: "Protocols in practice" },
      {
        kind: "list",
        text: [
          "SSH and RDP: put edge access rules on the app so only known addresses reach the port at all.",
          "Game servers: the classic Spectrum workload, since protocol floods are a common attack against them.",
          "Anything with a fixed port: mail submission, MQTT brokers, or a database behind strict IP allowlists.",
        ],
      },
      { kind: "h3", text: "What Spectrum does not do" },
      {
        kind: "p",
        text: "There is no cache, no WAF ruleset, and no request inspection, because there are no HTTP semantics to inspect. Web traffic belongs on the standard proxy, which carries those features. Spectrum's job is transport: availability and hiding for services the orange cloud cannot carry.",
      },
      {
        kind: "p",
        text: "Spectrum is a paid product with a small free allowance for a narrow set of applications, and the details live on the plan's limits page in the dashboard. Static IP addresses for an app are an option on higher tiers, worth checking before you build DNS around one.",
      },
    ],
  },
  {
    id: "account-structure",
    title: "Accounts, zones, and multi-tenant patterns",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/fundamentals/",
    body: [
      {
        kind: "p",
        text: "Two containers matter in Cloudflare: the account and the zone. The account is the administrative and billing wrapper that holds members, API tokens, and account-scoped products. A zone is one domain's presence at the edge: its DNS records, certificates, and per-domain rules. Almost every permission and product scopes to one of the two.",
      },
      { kind: "h3", text: "What scopes where" },
      {
        kind: "table",
        head: ["Level", "Scoped there"],
        rows: [
          ["Account", "Billing, members and roles, API tokens, Workers, R2, account-level WAF rulesets"],
          ["Zone", "DNS records, SSL/TLS mode, cache and redirect rules, zone-level WAF rules, rate limiting"],
        ],
      },
      { kind: "h3", text: "Account-level WAF rules" },
      {
        kind: "p",
        text: "Where the plan supports them, account-level WAF rulesets evaluate before zone rules and cover every zone in the account at once. One rule there protects all domains, which is the right place for an organization-wide block, such as a country restriction or a known attacker list. Zone rules stay for domain-specific logic. Check the dashboard for whether your plan exposes account-level custom rules.",
      },
      { kind: "h3", text: "The SSL for SaaS pattern" },
      {
        kind: "p",
        text: "Cloudflare for SaaS lets customer hostnames ride on your zone. You register each customer hostname as a custom hostname, Cloudflare issues and renews its certificate, and the customer points their DNS at you with a CNAME. Your zone's edge configuration serves their traffic, while the hostname still belongs to their domain. A fallback origin answers for hostnames with no matching registration.",
      },
      { kind: "h3", text: "Blast radius by construction" },
      {
        kind: "list",
        text: [
          "Keep internal and experimental projects in their own zones. A broken redirect rule in the sandbox zone cannot take down production.",
          "Scope API tokens to zones, not the account, for tooling that touches one domain. A leaked token then exposes one domain's configuration.",
          "Separate accounts are the hard boundary: distinct billing, distinct members, nothing shared. Reserve them for tenants or business units that must not see each other.",
          "One account with many zones is the normal, simple layout. Split only when a boundary has a reason to exist.",
        ],
      },
      {
        kind: "p",
        text: "When something behaves oddly, ask first which level owns it. Account-level and zone-level rules can both match a request, and account rules evaluate first. Reading both layers beats guessing.",
      },
    ],
  },
];
