import type { Guide } from "./guides";

// Media guides for the education section (Ops education hub, 2026-10-14):
// Images, Stream, Turnstile, media caching, and page-weight practice. Prose
// only, never usage claims, so the "real numbers only" rule does not apply.
// Plan-specific quotas and prices are described conceptually and pointed at
// the dashboard limits page.

export const MEDIA_GUIDES: readonly Guide[] = [
  {
    id: "images-essentials",
    title: "Cloudflare Images",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/images/",
    body: [
      {
        kind: "p",
        text: "Cloudflare Images covers two distinct jobs. It stores and delivers images you upload, and it transforms images on delivery that you keep somewhere else. The two jobs are billed and configured differently, so start by knowing which one you need.",
      },
      { kind: "h3", text: "Stored versus delivered" },
      {
        kind: "p",
        text: "An uploaded image is stored once as a master. Delivery happens through a URL on imagedelivery.net that carries the image id and a variant name. The master never changes. Every size and shape of the image is produced at delivery time from that one master.",
      },
      { kind: "h3", text: "Variants" },
      {
        kind: "p",
        text: "A variant is a named transform, defined once, applied by URL. Browsers and apps never ask for pixel dimensions. They ask for a variant.",
      },
      { kind: "kv", term: "Fitted", desc: "Fixed width, height, fit mode, and quality. Best for known slots like avatars and cards." },
      { kind: "kv", term: "Wildcard", desc: "The URL carries arbitrary transform parameters. Use it for one-off sizes that do not justify a named variant." },
      { kind: "kv", term: "Default", desc: "Every image has a default variant for unmodified delivery. Additional variants are free to define." },
      { kind: "h3", text: "Resizing images hosted elsewhere" },
      {
        kind: "p",
        text: "Image transformations also work without storage. A URL under /cdn-cgi/image/ carries the transform parameters plus the source URL on your own origin. The source image stays where it is, in R2, on a web server, or in any bucket. The transform runs at the edge and the result is cached. Availability of on-demand transformations depends on plan and subscription, so check the dashboard before building on it.",
      },
      { kind: "h3", text: "What gets billed" },
      {
        kind: "p",
        text: "Two meters apply to stored images: images stored, and unique images delivered per billing period. Repeated views of the same image in a period count once toward delivery. Each meter carries an included allowance, with overage billed in units beyond it. Current allowances sit on the dashboard limits page and change over time, so treat them as configuration, not constants.",
      },
      { kind: "h3", text: "Why delivery URLs beat self-hosted derivatives" },
      {
        kind: "list",
        text: [
          "No build step. Thumbnails and sizes exist the moment a variant URL is first requested, not at deploy time.",
          "One source of truth. Replacing an image updates every variant, so no derivative folder goes stale.",
          "Automatic format negotiation. Supporting browsers receive AVIF or WebP without client branching.",
          "Edge caching. Delivery responses are cached close to viewers, so repeat traffic does not re-transform.",
          "Clean output. Private metadata such as EXIF and GPS data is stripped from delivered variants.",
        ],
      },
    ],
  },
  {
    id: "stream-essentials",
    title: "Stream",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/stream/",
    body: [
      {
        kind: "p",
        text: "Stream is managed video. You hand it a video, it stores, transcodes, and serves it, and you embed a player or read an HLS or DASH manifest. The trade is convenience and per-minute billing against doing the encoding pipeline yourself.",
      },
      { kind: "h3", text: "Getting video in" },
      {
        kind: "list",
        text: [
          "Direct upload through the dashboard, the API, or the resumable upload protocol.",
          "Pull upload, where Stream fetches a source URL you provide.",
          "Live inputs over RTMPS or SRT. A finished live session becomes an on-demand recording automatically.",
        ],
      },
      { kind: "h3", text: "Delivery" },
      {
        kind: "p",
        text: "Each upload is re-encoded into a quality ladder, a set of renditions at different resolutions and bitrates. Stream generates HLS and DASH manifests, so any standards-based player works, and a hosted embed player is available when you do not want to run one. Playback picks a rendition as bandwidth changes. Thumbnails and a scrub storyboard are produced with the video.",
      },
      { kind: "h3", text: "Access control" },
      {
        kind: "p",
        text: "A video can require signed URLs. When that flag is set, playback needs a short-lived signed token that your backend mints and hands to the player. Unsigned requests fail. Use this for paywalls, private courses, and anything you would not leave public.",
      },
      { kind: "h3", text: "What gets billed" },
      {
        kind: "p",
        text: "Two meters apply: minutes stored, and minutes delivered, meaning minutes watched. Encoding is included in storage. Rates and included allowances change over time, so confirm both meters on the dashboard limits page before committing to a design that streams heavily.",
      },
      { kind: "h3", text: "When R2 plus a player beats Stream" },
      {
        kind: "list",
        text: [
          "Files are already encoded. If you ship fixed MP4 or WebM files, the transcode ladder buys nothing.",
          "Traffic is light or predictable. Stream meters every delivered minute, while R2 has no egress charge.",
          "Access control is simple or absent. Public files behind your own auth do not need signed tokens.",
          "Playback is basic. A video element with range request support covers it, and the CDN caches segments.",
        ],
      },
      {
        kind: "p",
        text: "Choose Stream when viewers need adaptive quality, when uploads arrive in unknown formats, or when short-lived signed access matters. Choose R2 when you control the files and the player.",
      },
    ],
  },
  {
    id: "turnstile",
    title: "Turnstile: CAPTCHA without puzzles",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/turnstile/",
    body: [
      {
        kind: "p",
        text: "Turnstile is a challenge widget for forms and APIs. In most cases the visitor sees nothing and passes silently. It runs a set of browser checks in the background and issues a proof token, so real users keep moving and scripted abuse has to fake a browser.",
      },
      { kind: "h3", text: "Three widget modes" },
      { kind: "kv", term: "Managed", desc: "The default. Runs invisibly when the signal is clear, falls back to an interactive checkbox when it is not." },
      { kind: "kv", term: "Non-interactive", desc: "Always invisible. Shows only a small compliance widget while the checks run. Choose it for polish on sign-up and checkout forms." },
      { kind: "kv", term: "Invisible", desc: "Fully invisible with no visible state at all. Strictest trade of user experience for pass rate." },
      { kind: "h3", text: "Embedding on a form" },
      {
        kind: "p",
        text: "Add the Turnstile script, then place a widget container with your sitekey on the form, or render one explicitly from JavaScript. On success the widget writes a token into the form field cf-turnstile-response, which submits with the rest of the form.",
      },
      { kind: "h3", text: "Server-side verification" },
      {
        kind: "p",
        text: "The token is a claim, not a fact. Your backend must verify it by posting the secret key and the token to the siteverify endpoint. The response says whether the token is valid. Tokens are single use and expire within about five minutes, so verify on the server on every submission and treat a missing or failed token as a rejection.",
      },
      { kind: "h3", text: "Pairing with a WAF rule" },
      {
        kind: "p",
        text: "Turnstile filters form submissions, but the request still reaches your origin. A rate-limiting rule or a custom WAF rule can stop the obvious floods earlier, at the edge, before they even render the form. The two layers complement each other. The edge rule drops volume cheaply. Turnstile separates real browsers from scripts among whatever remains. The siteverify call on your server is the final gate.",
      },
      {
        kind: "p",
        text: "Operationally, keep an eye on widget analytics for solve rates and interactive fallbacks. A falling pass rate on a Managed widget usually means traffic you want to look at, not a widget to reconfigure.",
      },
    ],
  },
  {
    id: "media-caching",
    title: "Serving media efficiently",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/cache/",
    body: [
      {
        kind: "p",
        text: "Media is where bandwidth budgets go to die, and where caching pays the most. A photo or video asset is large, rarely changes, and is requested many times. Get the cache headers right once and almost every later request is cheap.",
      },
      { kind: "h3", text: "Cache-Control on media assets" },
      {
        kind: "p",
        text: "Immutable assets want an aggressive header: public, a max-age in months or years, and immutable. This is the pattern for hashed filenames like hero.a3f19c.jpg. The browser stops revalidating, and the edge caches for the same duration. Content that changes gets a new filename, never an overwritten one. Files that must change in place need a shorter max-age and a real revalidation policy instead.",
      },
      { kind: "h3", text: "Cache keys for variant URLs" },
      {
        kind: "p",
        text: "The default cache key includes the query string, so image.jpg?width=640 and image.jpg?width=1024 are separate cache entries. That is usually what you want: each variant is transformed or stored once, then served from cache. Problems appear when URLs carry irrelevant parameters, such as analytics tags, because each tag value splits the cache. Cache Rules control which query parameters participate in the key, so normalize volatile ones out and keep meaningful ones in.",
      },
      { kind: "h3", text: "R2 as origin for large media" },
      {
        kind: "p",
        text: "A custom domain on an R2 bucket turns the bucket into a cacheable origin. Public assets are served directly. Private assets go through a Worker that authorizes and reads from the binding, with the public responses still cached. R2 charges no egress, and Tiered Cache means a median fetch misses only once, at an upper-tier colo, before other locations reuse it.",
      },
      { kind: "h3", text: "Range requests for video" },
      {
        kind: "p",
        text: "Video seeking sends Range requests, and the cache handles partial responses rather than forcing full downloads. Segment-based formats such as HLS and DASH fit this model naturally, since each segment is a small cacheable object. The maximum object size the cache stores on a given plan varies, so check the dashboard limits page before assuming a multi-gigabyte original will cache.",
      },
      { kind: "h3", text: "The cost view" },
      {
        kind: "p",
        text: "A cache hit touches no origin, and Cloudflare does not bill standard plan delivery per gigabyte. R2 charges no egress either. Together that makes cached media the cheapest bandwidth on the platform: the marginal cost of another view of an already-cached asset approaches zero. Uncached R2 delivery is still egress free, and you pay only storage and requests. The expensive path is the one you build yourself, on an origin that meters bandwidth.",
      },
    ],
  },
  {
    id: "fonts-and-optimization",
    title: "Fonts and page-weight optimization",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/images/",
    body: [
      {
        kind: "p",
        text: "Fonts are small files with outsized render cost, and page weight is a budget you cannot manage without measuring. This article covers both: host fonts well, negotiate image formats, and measure before you optimize anything.",
      },
      { kind: "h3", text: "Self-host versus third-party host" },
      {
        kind: "p",
        text: "Hosting font files on your own domain keeps the request first-party. A third-party font host adds a DNS lookup, a TLS handshake to another origin, and a privacy exposure. A German court ruled in 2022 that embedding fonts from a third-party host discloses visitor IP addresses to that host, which turned third-party font embedding into a legal question in the EU. Self-hosting ends the argument and removes the extra connection.",
      },
      { kind: "h3", text: "Serving fonts through the CDN" },
      {
        kind: "list",
        text: [
          "Ship WOFF2 only. It is supported everywhere current and compresses better than WOFF and TTF.",
          "Name font files with a content hash and cache them with a long max-age plus immutable. Fonts change at release time, not at request time.",
          "Preload only the one or two fonts above the fold. Preloading a full family wastes bandwidth.",
          "Set font-display: swap so text renders in a fallback face instead of hiding while the font loads.",
          "Serve font files with correct CORS headers when the CSS and the fonts come from different origins.",
        ],
      },
      { kind: "h3", text: "Image format negotiation" },
      {
        kind: "p",
        text: "Modern images come in AVIF and WebP at a fraction of the JPEG or PNG size for the same quality. Two mechanisms pick the format. Server-side negotiation reads the browser Accept header and returns the best supported format, which is what Image transformations and Polish do. Client-side negotiation uses a picture element with source elements, which also handles art direction and resolution switching through srcset. Use server-side negotiation as the default and reserve the picture element for cases where different formats need different crops.",
      },
      { kind: "h3", text: "Measure before optimizing" },
      {
        kind: "p",
        text: "Optimizing by intuition wastes effort on the wrong bytes. Start with a baseline: the browser DevTools network panel shows total transferred weight per page load, and Lighthouse or WebPageTest adds repeat-view numbers and field-style conditions. Sort requests by transferred size and note the top offenders. Optimize those, and only those. Then re-measure on the same pages and compare.",
      },
      {
        kind: "p",
        text: "Repeat the loop. A page-weight budget, a ceiling in KiB per template, keeps regressions visible in review. The first pass usually halves a page by fixing fonts and images. The second pass is where measurement, not instinct, decides what is left.",
      },
    ],
  },
];
