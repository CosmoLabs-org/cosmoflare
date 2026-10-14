// Network and domain articles for the Guides section: proxy status, DNS
// record types, TLS modes, custom hostnames and registrar basics. Prose
// only, no usage claims, so the "real numbers only" rule does not apply.
// Same block model as guides.ts; this module is the data.

import type { Guide } from "./guides";

export const NETWORK_GUIDES: readonly Guide[] = [
  {
    id: "proxied-vs-dns-only",
    title: "Proxied vs DNS-only: the orange cloud",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/dns/manage-dns-records/reference/proxied-dns-records/",
    body: [
      {
        kind: "p",
        text: "Every record in a Cloudflare zone has a proxy status. The orange cloud means proxied, the grey cloud means DNS-only. The toggle changes far more than name resolution, and several Cloudflare features depend on it completely.",
      },
      { kind: "h3", text: "What the orange cloud does" },
      {
        kind: "p",
        text: "With proxying on, Cloudflare publishes its own anycast addresses in the DNS answer. Visitors connect to the nearest Cloudflare edge, not to your server. Cloudflare terminates TLS, evaluates WAF and rate limiting rules, serves cacheable content from cache, applies redirect and configuration rules, and only then opens a connection to your origin.",
      },
      {
        kind: "p",
        text: "The origin address stays out of public DNS, which makes direct-to-origin attacks harder. It is not anonymity. The address can leak through mail headers, historical DNS data, or another hostname on the same server.",
      },
      { kind: "h3", text: "What DNS-only mode does" },
      {
        kind: "p",
        text: "A grey-cloud record answers with your real address and nothing else. Traffic flows straight from visitor to origin. No cache, no WAF, no redirect rules, no Cloudflare TLS. Features that require proxying do nothing on that hostname, and they fail silently rather than reporting an error.",
      },
      {
        kind: "p",
        text: "Some records must stay DNS-only. Hostnames referenced by MX records cannot run through the HTTP proxy, because SMTP is not HTTP. Non-web protocols work proxied only on Cloudflare's fixed list of supported ports.",
      },
      {
        kind: "table",
        head: ["Concern", "Proxied", "DNS-only"],
        rows: [
          ["Edge caching", "Active", "Not applied"],
          ["WAF and rate limiting", "Active", "Not applied"],
          ["Redirect and config rules", "Active", "Not applied"],
          ["Origin IP exposure", "Hidden behind Cloudflare IPs", "Visible in every answer"],
          ["TTL control", "Forced to automatic", "Honored as configured"],
        ],
      },
      {
        kind: "p",
        text: "TTL deserves a note. A proxied answer always points at Cloudflare, so lowering the TTL does nothing useful: changing your origin does not require a DNS change at all. A DNS-only record honors its TTL, so propagation speed after a change depends on the value you set.",
      },
      {
        kind: "p",
        text: "When a redirect rule, cache setting or WAF rule appears to do nothing, check the proxy status of the hostname first. It is the most common cause.",
      },
    ],
  },
  {
    id: "dns-record-types",
    title: "DNS record types you actually use",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/dns/manage-dns-records/reference/dns-record-types/",
    body: [
      {
        kind: "p",
        text: "Most zones run on a short list of record types. Each type maps one kind of data onto a name. The table covers the types a typical operator touches, in the order they tend to appear.",
      },
      {
        kind: "table",
        head: ["Type", "What it maps", "Typical use"],
        rows: [
          ["A", "Name to an IPv4 address", "Pointing a hostname at a server"],
          ["AAAA", "Name to an IPv6 address", "IPv6 endpoints, usually beside an A record"],
          ["CNAME", "Name to another name", "www to the apex, or a host to a vendor endpoint"],
          ["ALIAS / apex CNAME", "Apex to another name, served flattened", "Apex pointing at a load balancer or vendor host"],
          ["MX", "Domain to mail servers", "Email delivery routing"],
          ["TXT", "Free-form text", "SPF, DKIM, domain verification strings"],
          ["NS", "Zone to name servers", "Delegating a subdomain to another DNS provider"],
          ["SRV", "Service location with port, weight and priority", "Protocols that require discovery, such as SIP"],
          ["CAA", "Domain to allowed certificate authorities", "Restricting which CAs may issue certificates"],
        ],
      },
      { kind: "h3", text: "CNAME flattening at the apex" },
      {
        kind: "p",
        text: "DNS rules forbid a CNAME at the zone apex. The apex must carry SOA and NS records, and a CNAME cannot coexist with any other record type on the same name. A literal apex CNAME would break the delegation itself.",
      },
      {
        kind: "p",
        text: "Cloudflare works around the limit with flattening. You store a CNAME value on the apex, and Cloudflare resolves the target internally when it answers a query, then returns ordinary A and AAAA records on the wire. Resolvers see a standards-compliant apex.",
      },
      {
        kind: "p",
        text: "The practical result: you can aim the apex at a hostname, such as a load balancer or a vendor endpoint, without breaking apex rules. When the apex record is proxied, flattening still applies on the DNS side, while traffic behavior is governed by the proxy as described in the orange-cloud guide.",
      },
      {
        kind: "p",
        text: "TXT records deserve a second look. Nearly every external service, from mail providers to certificate authorities to SaaS vendors, asks for a verification string as a TXT record. They accumulate over time. Stale ones are harmless, but reading them all takes longer than adding a new one.",
      },
    ],
  },
  {
    id: "ssl-tls-modes",
    title: "SSL/TLS modes and what each guarantees",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/ssl/",
    body: [
      {
        kind: "p",
        text: "The zone-level SSL/TLS mode decides how Cloudflare talks to visitors and to your origin. Four modes exist, and they are not four levels of the same setting. Two encrypt the full path, one encrypts half of it, and one turns encryption off.",
      },
      {
        kind: "table",
        head: ["Mode", "Visitor to edge", "Edge to origin", "Origin certificate requirement"],
        rows: [
          ["Off (deprecated)", "Plain HTTP only", "Plain HTTP", "None"],
          ["Flexible", "HTTPS", "Plain HTTP", "Never validated"],
          ["Full", "HTTPS", "HTTPS", "Any certificate, including self-signed or expired"],
          ["Full (strict)", "HTTPS", "HTTPS", "Valid, unexpired, hostname matches, publicly trusted or Cloudflare Origin CA"],
        ],
      },
      { kind: "h3", text: "The Flexible trap" },
      {
        kind: "p",
        text: "Flexible looks like it works. The browser shows a secure connection because the visitor leg is encrypted. The edge-to-origin leg is plain HTTP, so the half of the path that crosses the public internet to your server carries no encryption at all.",
      },
      {
        kind: "p",
        text: "Flexible is also the classic cause of redirect loops. If the origin itself forces HTTPS, it answers Cloudflare's HTTP request with a redirect to the HTTPS URL. Cloudflare retries per the mode's contract, still over HTTP, and the origin redirects again. The visitor sees ERR_TOO_MANY_REDIRECTS. Raising the mode to Full or Full (strict) fixes it.",
      },
      { kind: "h3", text: "Why Full (strict) is the recommendation" },
      {
        kind: "p",
        text: "Full encrypts both legs but accepts any certificate, so an attacker positioned between Cloudflare and the origin can present a self-signed certificate and read the traffic. Full (strict) closes that gap: the origin certificate must be valid, unexpired, and issued for the hostname. Pair it with a Cloudflare Origin CA certificate, which is free, trusted only by Cloudflare's edge, and can be issued for up to 15 years.",
      },
      { kind: "h3", text: "Always Use HTTPS and HSTS" },
      {
        kind: "p",
        text: "Always Use HTTPS sends a 301 redirect for HTTP requests at the edge. It fixes the mixed-visitor problem for the current visit and says nothing about future ones.",
      },
      {
        kind: "p",
        text: "HSTS closes that gap. The Strict-Transport-Security header tells the browser to use HTTPS only for the domain for the declared max-age, with optional includeSubDomains and preload. Enable it only after Full (strict) works across the zone. A wrong HSTS policy pins browsers into HTTPS for the whole max-age, and walking it back takes at least that long.",
      },
    ],
  },
  {
    id: "custom-hostnames",
    title: "Custom hostnames: SaaS on your zone",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/cloudflare-for-platforms/cloudflare-for-saas/",
    body: [
      {
        kind: "p",
        text: "Cloudflare for SaaS, historically called SSL for SaaS, lets other people's domains terminate TLS on your zone. Your customer keeps their own domain at their own registrar and DNS provider, points one record at your service, and you present a valid certificate for their hostname.",
      },
      { kind: "h3", text: "How it differs from a custom domain" },
      {
        kind: "p",
        text: "A custom domain is one you control end to end. You own the zone, add the records, and certificates follow from your own configuration. A custom hostname belongs to someone else. You never edit their DNS, you only ask for the single pointing record, and you carry the responsibility for issuing and renewing a certificate for a name you do not own.",
      },
      {
        kind: "p",
        text: "The flow is mechanical. The customer adds a CNAME from app.customer.example to your SaaS endpoint, for example tenants.yourapp.example. Cloudflare validates control of the requested hostname with an HTTP or TXT challenge, then issues a certificate for it on your zone. Renewal repeats the validation on schedule without customer action.",
      },
      { kind: "h3", text: "Fallback origin" },
      {
        kind: "p",
        text: "Every custom hostname configuration requires a fallback origin: one hostname inside your own zone that receives traffic for custom hostnames. It must be a proxied record in your zone and must resolve before any hostname activates. Unless you configure per-hostname origins through custom origin servers or origin rules, all custom hostname traffic lands on the fallback origin, and your application routes on the Host header.",
      },
      {
        kind: "p",
        text: "The service requires a paid plan. A base allotment of custom hostnames per zone is included, and hostnames beyond it bill per hostname per month at Cloudflare's current rate.",
      },
      {
        kind: "p",
        text: "Check two states before you tell a customer their domain is live. The hostname status and the certificate status are separate values, and a hostname can be active while its certificate is still pending validation.",
      },
    ],
  },
  {
    id: "registrar-basics",
    title: "Registrar and transfer locks",
    updated: "2026-10-14",
    source: "https://developers.cloudflare.com/registrar/",
    body: [
      {
        kind: "p",
        text: "Cloudflare Registrar sells domains at wholesale cost: the registry fee plus the ICANN fee, no markup, no upsells. Renewal prices match registration prices. The tradeoff is a deliberately small feature set, so most of the work is knowing the few rules that do exist.",
      },
      { kind: "h3", text: "Auto-renew" },
      {
        kind: "p",
        text: "Auto-renew is on by default. Renewal charges the payment method on file shortly before expiry. Turning it off moves the risk to you, and an expired domain takes mail and web with it regardless of how correct your DNS records are.",
      },
      { kind: "h3", text: "The transfer lock and the 60-day rule" },
      {
        kind: "p",
        text: "Newly registered and newly transferred domains come with the registrar transfer lock enabled. The lock rejects transfer attempts at the registrar until you remove it deliberately. Leave it on. Some registries add a second, registry-level lock on top.",
      },
      {
        kind: "p",
        text: "ICANN policy blocks transfers between registrars for 60 days after registration and for 60 days after a transfer into the current registrar. The window can also apply after certain registrant contact changes. Plan a transfer for after the window closes, not into it, because a rejected transfer costs you the attempt and the waiting time.",
      },
      { kind: "h3", text: "Expiry and the grace period" },
      {
        kind: "p",
        text: "When a domain is not renewed, most TLDs follow the same sequence. First a renewal grace period, up to roughly 45 days depending on the TLD, where renewal costs the normal price. Then a redemption period of about 30 days, where recovery costs a per-TLD restore fee on top of renewal. Then a short pending delete stage, about 5 days, after which the name drops and anyone can register it. Mail usually stops first.",
      },
      {
        kind: "p",
        text: "Monitoring expiry beats remembering dates. An ops tool reads the expiry date and the lock state for every domain in the account, alerts at 30, 7 and 1 days, and catches the failure modes a calendar cannot: an expired card on file, auto-renew left off, or a lock removed during a transfer that never finished. Expiry monitoring is cheap. Redemption restores are not.",
      },
    ],
  },
];
