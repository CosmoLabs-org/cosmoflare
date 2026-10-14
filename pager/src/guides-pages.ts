// Pages platform articles for the Guides section: deployment shapes,
// preview deployments, custom domains, git-connected builds and
// configuration as code. Prose only, no usage claims, so the
// "real numbers only" rule does not apply. Same block model as
// guides.ts; this module is the data.

import type { Guide } from "./guides";

export const PAGES_GUIDES: readonly Guide[] = [
  {
    id: "pages-vs-workers",
    title: "Pages, Workers, or Workers static assets",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/workers/static-assets/",
    body: [
      {
        kind: "p",
        text: "A front-end project on Cloudflare lands in one of three shapes: a Pages project, a Pages project with Functions, or a Workers project with static assets. All three serve from the same edge network. The difference is the deployment model, the feature set, and where Cloudflare invests new work.",
      },
      {
        kind: "table",
        head: ["Concern", "Pages", "Workers with static assets"],
        rows: [
          ["Best for", "Existing sites and the classic static-plus-functions split", "New full-stack projects deployed as one unit"],
          ["Build model", "Git integration built in, or direct upload", "wrangler deploy, or git via Workers Builds"],
          ["Functions", "Pages Functions with file-based routing", "One Worker script handling what assets do not"],
          ["Preview URLs", "Unique URL per deployment plus branch aliases", "Version preview URLs, and branch previews under Builds"],
          ["Custom domains", "Per project, on zones in the same account", "Custom domains or routes, same account rule"],
          ["Status", "Supported and maintained, few new features", "Recommended for all new projects"],
        ],
      },
      { kind: "h3", text: "New projects start on Workers" },
      {
        kind: "p",
        text: "Cloudflare recommends Workers with static assets for new projects. A static assets directory deploys beside the Worker script. Requests for files serve directly from the edge, and everything else invokes the Worker. Bindings attach to the same project, so the front-end and the API share one deployment and one preview flow.",
      },
      {
        kind: "list",
        text: [
          "Static assets and server code deploy as one unit.",
          "Bindings to KV, R2, D1, Queues and other services attach to the same project.",
          "The compatibility date and flags pin runtime behavior per project.",
          "New platform features arrive on Workers first.",
        ],
      },
      { kind: "h3", text: "Pages stays supported" },
      {
        kind: "p",
        text: "Existing Pages projects keep working. Cloudflare maintains the product and its documentation, and it does not remove features. What it does not do is build new platform capability into Pages. New deployment, build and billing features appear on Workers first. A Pages project you are happy with does not need to move.",
      },
      { kind: "h3", text: "What migration looks like" },
      {
        kind: "p",
        text: "A migration is mostly mechanical. Static content moves into an assets directory, Functions collapse into the Worker script, and project settings move into the wrangler configuration file.",
      },
      {
        kind: "list",
        text: [
          "Move build output into an assets directory and declare it with the assets field in wrangler.toml or wrangler.jsonc.",
          "Port Functions logic into the Worker, one route at a time.",
          "Move redirect and header rules from the _redirects and _headers files into the assets configuration, which supports both.",
          "Recreate custom domains on the Worker, deploy, then repoint DNS.",
        ],
      },
      {
        kind: "p",
        text: "Keep the Pages project until the Worker serves production traffic cleanly. Both can run at the same time on different hostnames during the move.",
      },
    ],
  },
  {
    id: "preview-deployments",
    title: "Preview deployments",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/workers/configuration/previews/",
    body: [
      {
        kind: "p",
        text: "Every non-production deployment gets a URL of its own. The URL is unique per deployment, stays up after you close the pull request, and serves the exact build it was created for. Review then runs against real edge behavior instead of a local approximation.",
      },
      { kind: "h3", text: "Production and preview environments" },
      {
        kind: "p",
        text: "The production deployment serves your production domains. Previews are separate deployments, not overrides of production. Each has its own URL, its own build and its own traffic. A push to the production branch targets production. A push anywhere else targets a preview. Production never changes until a build for the production branch succeeds.",
      },
      {
        kind: "list",
        text: [
          "Pages preview: <hash>.<project>.pages.dev, one URL per deployment.",
          "Pages branch alias: <branch>.<project>.pages.dev, refreshed by the latest build of that branch.",
          "Pages production: <project>.pages.dev plus your custom domains.",
          "Workers previews: a URL that embeds the version id under your account workers.dev subdomain.",
        ],
      },
      { kind: "h3", text: "Sharing previews for review" },
      {
        kind: "p",
        text: "Paste the deployment URL into a pull request or a chat, and reviewers open the real build on the real network. Branch alias URLs are useful for long-lived branches: the alias always serves the newest build of that branch, so the link never goes stale while the branch lives.",
      },
      { kind: "h3", text: "Access control on previews" },
      {
        kind: "p",
        text: "Unlisted does not mean protected. Anyone who learns the URL can open a public preview. Pages projects can put preview deployments behind Cloudflare Access from the dashboard, which challenges visitors before a request reaches the deployment. For Workers previews, protect a custom domain with an Access self-hosted application, or check a session inside the Worker itself. Keep anything private behind a real gate.",
      },
      { kind: "h3", text: "How preview traffic is billed" },
      {
        kind: "p",
        text: "Preview traffic is real traffic. Requests to preview URLs are served from the same edge network and bill like requests to production, at the same rates and against the same plan limits. Load testing a preview shows up on the same bill as production. Current request and bandwidth limits for your plan are listed on the dashboard limits page.",
      },
    ],
  },
  {
    id: "custom-domains-workers",
    title: "Custom domains and routes on Workers",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/workers/configuration/routing/custom-domains/",
    body: [
      {
        kind: "p",
        text: "A Worker is code on the edge until something routes traffic to it. Three mechanisms do that: the workers.dev subdomain, a custom domain, and a route pattern. They attach at different layers and they behave differently when they overlap.",
      },
      {
        kind: "kv",
        term: "workers.dev subdomain",
        desc: "Every account gets one subdomain, and each Worker can publish under it as <name>.<subdomain>.workers.dev. No zone or DNS setup. Best for tests and internal tools.",
      },
      {
        kind: "kv",
        term: "Custom domain",
        desc: "Cloudflare creates the DNS record on your zone and issues the certificate automatically. The hostname belongs to the Worker. Recommended for production.",
      },
      {
        kind: "kv",
        term: "Route pattern",
        desc: "A zone-level pattern such as example.com/api/* that sends matching requests to the Worker. The hostname must be proxied for the pattern to see traffic. Best when a Worker takes over part of an existing site.",
      },
      {
        kind: "table",
        head: ["Aspect", "workers.dev", "Custom domain", "Route pattern"],
        rows: [
          ["DNS record", "None", "Created and managed by Cloudflare", "Existing records stay in place"],
          ["TLS certificate", "Managed by Cloudflare", "Issued and renewed automatically", "Zone certificate, already in place"],
          ["Needs a zone", "No", "Yes, in the same account", "Yes, in the same account"],
          ["Scope", "One Worker name", "One hostname per domain", "Any pattern on the zone"],
          ["Best for", "Testing and internal tools", "Production hostnames", "Part of an existing site"],
        ],
      },
      { kind: "h3", text: "Precedence when both match" },
      {
        kind: "p",
        text: "If a hostname has a custom domain and a route pattern that also matches, the custom domain wins. Cloudflare serves the custom domain first, and the route applies only where no custom domain covers the request. Adding a custom domain over a hostname that a route already serves stops that route from firing. Check the Worker's domains and routes list after the change so stale patterns do not surprise you later.",
      },
      {
        kind: "p",
        text: "Between two route patterns that both match, the more specific pattern wins. example.com/api/v2/* beats example.com/api/* on the same zone.",
      },
      { kind: "h3", text: "Practical notes" },
      {
        kind: "list",
        text: [
          "Custom domains require the zone to live in the same Cloudflare account as the Worker.",
          "Disabling the workers.dev subdomain removes that public URL without touching custom domains or routes.",
          "A broad pattern such as example.com/* also captures paths the Worker never handles. Keep patterns narrow.",
        ],
      },
    ],
  },
  {
    id: "workers-builds-ci",
    title: "Workers Builds",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/workers/ci-cd/builds/",
    body: [
      {
        kind: "p",
        text: "Workers Builds is Cloudflare's git-connected build system for Workers. Connect a GitHub or GitLab repository, and each push produces a build and a deployment without your CI running wrangler. It covers the common path end to end.",
      },
      {
        kind: "list",
        text: [
          "Builds on every push to the configured production branch, and deploys to production on success.",
          "Builds non-production branches as preview deployments with their own URLs.",
          "Watches paths, so a docs-only commit can skip the build.",
          "Runs your build command, then your deploy command, in the project's build image.",
        ],
      },
      { kind: "h3", text: "Build configuration per project" },
      {
        kind: "p",
        text: "Configuration lives with the project rather than in a pipeline file. The build settings hold the production branch, the build and deploy commands, the build image, watch paths and build variables. Build variables marked as secret stay encrypted and reach the build process only.",
      },
      { kind: "h3", text: "Previews per branch" },
      {
        kind: "p",
        text: "Each non-production branch builds to preview deployments. Open a pull request and the build posts a preview URL against it. New commits to the branch replace the preview, so the URL tracks the tip of the branch. The production deployment stays untouched until the production branch builds.",
      },
      { kind: "h3", text: "Where a pipeline still earns its place" },
      {
        kind: "p",
        text: "Built-in builds deploy code. A pipeline does more than deploy. Keep one when you need any of the following.",
      },
      {
        kind: "list",
        text: [
          "Run tests, type checks and lint before anything deploys, and block on failure.",
          "Scan dependencies for vulnerabilities and enforce policy on what merges.",
          "Deploy to several targets from one commit, such as staging first and production on approval.",
          "Build artifacts outside the Worker, such as packages, docs or images.",
          "Coordinate database migrations or backfills around the deploy.",
        ],
      },
      {
        kind: "p",
        text: "A common shape keeps both systems. The pipeline runs checks on the pull request. The merge then triggers Workers Builds, which deploys production. One system gates, the other ships.",
      },
    ],
  },
  {
    id: "project-config",
    title: "wrangler.toml and configuration as code",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/workers/wrangler/configuration/",
    body: [
      {
        kind: "p",
        text: "A Worker can exist as a dashboard-only project, but then nothing about it is reviewable or reproducible. Declaring the project in wrangler.toml or wrangler.jsonc and committing that file turns the project into code. Anyone with access can deploy it, review changes to it, and rebuild it from a fresh checkout.",
      },
      { kind: "h3", text: "What lives in configuration" },
      {
        kind: "list",
        text: [
          "Project identity: name, main entry point, compatibility_date and compatibility_flags.",
          "Bindings: KV namespaces, R2 buckets, D1 databases, service bindings, Queues, Durable Objects and AI.",
          "Traffic: routes, custom domains, workers.dev enablement and cron triggers.",
          "Assets: the static assets directory and how it serves requests.",
          "Plain-text vars and observability settings.",
        ],
      },
      { kind: "h3", text: "Environment sections" },
      {
        kind: "p",
        text: "The same file can describe several environments. A named section such as [env.staging] starts from the top-level configuration and overrides what it redefines: a different deployed name, different vars, different routes. Keep the shared shape at the top level and name only the differences. Before you trust a section, run wrangler deploy --dry-run for that environment and read the output to see exactly what would ship.",
      },
      { kind: "h3", text: "Secrets versus vars" },
      {
        kind: "p",
        text: "Vars are plaintext values in the config file. Everyone who can read the file sees them, and every deploy injects them. Secrets are written outside the file, with wrangler secret put or the dashboard, stored encrypted and injected at runtime. Never put a secret in vars to save a step, and never commit one. A secret only needs to be set once, and later deploys keep it until you change it.",
      },
      { kind: "h3", text: "Configuration in git" },
      {
        kind: "p",
        text: "Committed configuration gives you three properties. Review: a binding or route change appears in the pull request diff instead of appearing silently in the dashboard. Reproducibility: a fresh clone plus one deploy command rebuilds the project. Recovery: the file records what the project was, independent of who clicked what. Pair it with one rule. Any dashboard change gets copied back into the config before the next deploy, so the file never drifts.",
      },
    ],
  },
];
