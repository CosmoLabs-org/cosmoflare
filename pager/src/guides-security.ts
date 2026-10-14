// Security education articles for the Guides section: WAF actions,
// Zero Trust, Email Routing and API token hygiene. Guidance prose only,
// never usage claims, so the "real numbers only" rule does not apply.
// Static data, no fetching.

import type { Guide } from "./guides";

export const SECURITY_GUIDES: readonly Guide[] = [
  {
    id: "waf-managed-vs-custom",
    title: "WAF: managed rules vs custom rules",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "The WAF gives you two different tools. Managed rulesets are curated by Cloudflare and updated as threats change. Custom rules are yours alone: you write the expression, you pick the action. Both run on the same engine, and knowing which one to reach for avoids most WAF mistakes.",
      },
      { kind: "h3", text: "Managed rulesets" },
      {
        kind: "p",
        text: "A managed ruleset is a bundle of rules Cloudflare authors and maintains. You deploy it at a sensitivity level and the platform scores each request against it. The Cloudflare Managed Ruleset is built on the OWASP Core Rule Set, so its coverage follows the classic web-attack taxonomy: injection, protocol attacks, scanner detection. Rules update continuously with no action on your side.",
      },
      {
        kind: "list",
        text: [
          "Sensitivity levels raise or lower the score a request must reach before the ruleset fires.",
          "Deploy at a lower sensitivity and read the logs before raising it.",
          "Individual rules can be excluded when they false-positive on your traffic.",
          "Every plan also gets the Cloudflare Free Managed Ruleset, a smaller subset that runs automatically and needs no deployment.",
        ],
      },
      { kind: "h3", text: "Custom rules" },
      {
        kind: "p",
        text: "A custom rule is a filter expression plus an action. The expression matches on any field the rules language exposes: IP, ASN, path, headers, country, TLS fingerprint. The action is block, challenge, log or skip. This is where knowledge of your own traffic lives: only you know which endpoint is an API path and which client is your monitoring.",
      },
      {
        kind: "list",
        text: [
          "Match narrow. A rule that fires on one path is easier to reason about than one that fires on an ASN.",
          "Use rule lists for sets of addresses you update over time, instead of growing a long expression.",
          "Rate limiting is its own ruleset, for per-client request rates rather than per-request content.",
        ],
      },
      { kind: "h3", text: "The quota split" },
      {
        kind: "p",
        text: "Deploying managed rulesets does not consume your custom rules quota. The quota counts custom rules only. On the free plan that allowance is five custom rules per zone, which is small, and paid plans raise the ceiling. The practical consequence: managed protection is free to turn on, and every custom rule you write costs a slot, so spend slots deliberately.",
      },
      { kind: "h3", text: "Deployment order" },
      {
        kind: "p",
        text: "Deploy the managed ruleset first, at default sensitivity, and leave it alone for a week. It covers generic attacks immediately and its updates need no maintenance from you. Then add custom rules one at a time, for what managed rules cannot know about: your login path, your internal API range, the country that never appears in legitimate traffic. One rule at a time means a regression points at one rule. The WAF section in Ops shows the rule inventory and quota gauges, so each addition is visible.",
      },
    ],
  },
  {
    id: "challenge-actions",
    title: "Block, challenge, log: choosing the action",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Every rule ends in an action. The menu looks simple and the differences matter in production. The table is the whole decision; the notes below cover the two that people get wrong.",
      },
      {
        kind: "table",
        head: ["Action", "What the client sees", "When to use"],
        rows: [
          [
            "Block",
            "A 403 error page. No retry, no way through.",
            "Traffic that is never legitimate. Known-bad ASNs, exploit paths, addresses you have already confirmed.",
          ],
          [
            "Managed Challenge",
            "An interstitial. Low-risk clients pass without interaction; suspicious ones must solve a visual challenge.",
            "The default for anything that might include humans. Login pages, forms, anything you cannot safely allowlist.",
          ],
          [
            "JS Challenge",
            "Nothing visible. A script runs in the browser and the client passes automatically.",
            "Non-interactive gating. Bots that do not run JavaScript fail it silently.",
          ],
          [
            "Log",
            "Nothing. The request proceeds exactly as if no rule matched.",
            "Measuring a rule before enforcing it. The event still appears in security logs, tagged with the action.",
          ],
          [
            "Skip",
            "Nothing. The request proceeds and the skipped components do not run.",
            "Exceptions. Let your office range past the managed ruleset, or stop further rules from running.",
          ],
        ],
      },
      {
        kind: "p",
        text: "Two nuances. A Managed Challenge turns interactive only when the client looks suspicious, so most real users see nothing at all. A JS Challenge is the older, weaker option: it stops trivial non-browser traffic but not a modern headless browser. Prefer Managed Challenge when in doubt.",
      },
      { kind: "h3", text: "The rollout discipline" },
      {
        kind: "p",
        text: "Never deploy a new enforcement rule straight to Block. The sequence that avoids outages:",
      },
      {
        kind: "list",
        text: [
          "Set the action to Log. Let it run long enough to cover both weekday and weekend traffic.",
          "Read the events. If legitimate requests appear, fix the expression, not the action.",
          "When the log is clean, move the rule to Managed Challenge. Keep it there if the path has any human traffic.",
          "Move to Block only when no legitimate client can match, or for challenge-hostile endpoints such as raw API paths.",
        ],
      },
      {
        kind: "p",
        text: "The security events view in Ops gives hit counts per rule, so the log phase is measurable rather than a guess.",
      },
    ],
  },
  {
    id: "zero-trust-quickstart",
    title: "Zero Trust in ten minutes: Access and Tunnels",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Access and Tunnels are the two halves of Cloudflare Zero Trust. A Tunnel connects your infrastructure to the edge without any inbound port. Access sits in front of an application and demands identity before a request reaches it. Together they give you an internal app reachable only through authentication, with no open firewall port at all.",
      },
      { kind: "h3", text: "Access: authentication at the edge" },
      {
        kind: "p",
        text: "An Access application wraps a path on your zone. Before serving it, Cloudflare redirects the client to your chosen identity provider. Google, GitHub, Okta, Entra ID and generic SAML or OIDC all work, and so does a simple one-time PIN sent by email. A policy then decides who may continue.",
      },
      {
        kind: "list",
        text: [
          "Allow everyone on your email domain. The fastest useful policy for a small team.",
          "Allow specific addresses, groups from your identity provider, or a country.",
          "Add require blocks when a stricter gate matters, such as admin paths.",
          "Deny rules exist, and evaluation order matters: put denies above allows.",
        ],
      },
      {
        kind: "p",
        text: "Access does not replace your application's own sessions. It proves identity at the edge and forwards a signed JWT your app can verify if it wants to know who is calling.",
      },
      { kind: "h3", text: "Tunnels: outbound-only connectors" },
      {
        kind: "p",
        text: "The cloudflared daemon runs next to your service and holds outbound connections to the edge. Incoming requests arrive down those connections. The host needs no open inbound port, no public IP, and no firewall exception. An attacker scanning the address finds nothing to connect to.",
      },
      {
        kind: "list",
        text: [
          "Create the tunnel in the Zero Trust dashboard or with cloudflared tunnel login.",
          "Install the connector and map a public hostname to a local service, or keep the hostname private and reach it through WARP.",
          "Run two connectors for redundancy. The edge load-balances across them.",
        ],
      },
      { kind: "h3", text: "The two together" },
      {
        kind: "p",
        text: "Point the tunnel's hostname at your internal app, then add an Access application on that hostname. The result is a private service with zero public ingress: the port stays closed, and the only path in runs through your identity provider. Ten minutes of setup replaces a VPN endpoint for most internal tools.",
      },
    ],
  },
  {
    id: "email-routing",
    title: "Email Routing",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Email Routing forwards mail for addresses on your domain to mailboxes you already own. It runs at the edge, costs nothing, and needs no server. It is a forwarding service, not a mail platform, and the boundaries are worth knowing before you depend on it.",
      },
      { kind: "h3", text: "Addresses and catch-all" },
      {
        kind: "p",
        text: "You create custom addresses, such as hello@example.com, and map each to one or more destinations. A destination must be verified first: Cloudflare sends a confirmation mail, and forwarding to it stays off until confirmed. A catch-all forwards everything else on the domain. Point it at a real mailbox while you are testing, and it collects mail sent to addresses that were never configured.",
      },
      { kind: "h3", text: "The DNS records" },
      {
        kind: "p",
        text: "Enabling routing on a Cloudflare zone adds the records for you. The set it needs:",
      },
      {
        kind: "kv",
        term: "MX",
        desc: "Routes inbound mail for the domain to Cloudflare's edge. Several MX records with priorities are added automatically.",
      },
      {
        kind: "kv",
        term: "SPF (TXT)",
        desc: "v=spf1 include:_spf.mx.cloudflare.net ~all. Authorizes Cloudflare's forwarders for the mail they relay onward.",
      },
      { kind: "h3", text: "What it does not do" },
      {
        kind: "p",
        text: "No sending. Replies leave through whatever mailbox received the forwarded mail, not through Cloudflare. No storage. If no destination accepts the message, it is dropped, and the routing dashboard reports the drops. No folders, search or webmail. Treat it as a router in front of a real mailbox, not as a mailbox.",
      },
      { kind: "h3", text: "Workers for programmatic handling" },
      {
        kind: "p",
        text: "An email Worker receives inbound messages as an object with headers and a readable body. In the handler you can reject with an SMTP status, drop silently, or forward to another address, and you can write to KV or D1 as the message passes through. Common uses: routing support@ by content, triaging bounces, feeding a ticketing pipeline. Deploy the Worker, then select it as the send action on a route. No other glue is needed.",
      },
    ],
  },
  {
    id: "api-token-hygiene",
    title: "API tokens: scopes, least privilege, rotation",
    updated: "2026-10-14",
    body: [
      {
        kind: "p",
        text: "Two credentials authenticate against the Cloudflare API. The Global API Key is the legacy one: a single key per account, full permission over everything, no scoping, no expiry. Every use of it carries the whole account. API tokens are the replacement: scoped, revocable, expirable. Use tokens. The global key is rarely required by current tooling and never the right choice.",
      },
      { kind: "h3", text: "Permission groups and scope" },
      {
        kind: "p",
        text: "A token's power is defined by permission groups and by the resources it applies to. Permission groups are product bundles, such as Zone Read, Workers Scripts Read or Billing Read. Scope decides where they apply: the whole account, specific zones, or a single worker. A token that reads analytics for one zone cannot touch another zone, even in the same account.",
      },
      {
        kind: "kv",
        term: "Read versus edit",
        desc: "Most automation only observes. Grant the Read group, not Edit, and a leaked token becomes an information exposure rather than a mutation risk.",
      },
      {
        kind: "kv",
        term: "Account versus zone scope",
        desc: "Prefer the narrowest resource set the tool actually touches. Zone-scoped tokens survive account growth.",
      },
      {
        kind: "kv",
        term: "Expiry and IP filter",
        desc: "Both are optional per token. A CI token with an expiry and an IP filter beats a longer-lived unfiltered one.",
      },
      { kind: "h3", text: "One token per automation" },
      {
        kind: "p",
        text: "Give each script, integration and app its own token with only the scopes it needs. The cost is a little bookkeeping. The payoff is precise blast radius: when one token leaks or misbehaves, you revoke exactly it, and every other automation keeps working. Shared tokens destroy both properties at once.",
      },
      { kind: "h3", text: "Audit before and after granting" },
      {
        kind: "p",
        text: "You can see what a token can reach before it is ever used. The token record lists its permission groups and scoped resources, and the verify endpoint (GET /user/tokens/verify) confirms the token is live and reports its id. After granting, the account Audit Logs attribute every API call to the token that made it. That attribution is why one-token-per-automation works: the log answers which integration did this, by name.",
      },
      {
        kind: "p",
        text: "Ops itself follows this model: it asks for one read-scoped token, and every section is as deep as that token's scopes allow. Grant read groups first. Add an edit group only where a tool truly mutates something.",
      },
    ],
  },
];
