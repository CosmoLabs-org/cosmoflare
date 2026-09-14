/*
Package cosmoflare provides a Go library for managing the full Cloudflare
developer platform from Go code and CLIs.

It covers R2 (storage), Workers (compute), KV (key-value), DNS, Zones, SSL,
D1 (SQL database), Pages (static hosting), Queues (message queues), Images,
Hyperdrive, Vectorize, Workers AI, Stream, Email Routing, Healthchecks and
more — every service behind a *Service type constructed from an account ID
and API token.

Design:

  - Library-first: importable by any Go project
    (import cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare").
  - All services share one client construction pattern:
    NewClient with functional options (WithAccountID, WithAPIToken,
    WithHTTPClient, WithTimeout, ...), or per-service
    New<Name>ServiceFromCreds constructors.
  - Errors are wrapped R2Error values carrying Op, Bucket and Key so callers
    can branch on the failing operation without string matching.
  - The cosmoflare CLI and the Cosmoflare desktop app are both thin layers
    over this package — no separate API implementations per tier.

Credentials resolve with "options > environment > profile" precedence:
WithAccountID/WithAPIToken first, then CLOUDFLARE_ACCOUNT_ID and
CLOUDFLARE_API_TOKEN, then the named profile from the machine config
(~/.cosmoflare/config.yaml, legacy ~/.r2go2/config.yaml read for
compatibility).
*/
package cosmoflare
