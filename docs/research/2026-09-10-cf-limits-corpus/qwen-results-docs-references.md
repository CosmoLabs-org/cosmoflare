# Qwen results — Cloudflare docs references (D1 security, wrangler auth, token permissions)

Provenance: asked 2026-09-11 in the Qwen research conversation. Prompt:
"Give the exact docs references and save descriptions from the website
directly in the chat for our own knowledge please." Captured 2026-09-11.
Feeds BR-01 catalog provenance, FEAT-011/019 (permissions reference page),
FEAT-022 (D1 migrations/time-travel/backups pages). Notable recent
changelogs: wrangler device-flow login (2026-08-04), `wrangler auth token`
command (2025-12-18).

---

Here are the exact documentation references and descriptions from the Cloudflare developer website regarding D1 security, Wrangler authentication, and API token permissions for your team's knowledge base.

### D1 Data Security & Best Practices

* **Data Security**
  * URL: `https://developers.cloudflare.com/d1/reference/data-security/`
  * Description: D1 encrypts data at rest and in transit, and is covered by Cloudflare's SOC 2 and ISO 27001 compliance certifications .

* **Best Practices**
  * URL: `https://developers.cloudflare.com/d1/best-practices/`
  * Description: This section provides recommended patterns and techniques for building with D1 databases .

* **Local Development**
  * URL: `https://developers.cloudflare.com/d1/best-practices/local-development/`
  * Description: Migrations can be run locally as part of your CI/CD setup by passing the --local flag to the command .

### D1 Migrations & Backups

* **Migrations**
  * URL: `https://developers.cloudflare.com/d1/reference/migrations/`
  * Description: Database migrations are a way of versioning your database . Each migration is stored as an .sql file in your migrations folder .

* **Time Travel and Backups**
  * URL: `https://developers.cloudflare.com/d1/reference/time-travel/`
  * Description: Time Travel is D1's approach to backups and point-in-time-recovery, and allows you to restore a database to any minute within the last 30 days .

* **Backups (Legacy)**
  * URL: `https://developers.cloudflare.com/d1/reference/backups/`
  * Description: Time Travel replaces the snapshot based backups used for legacy alpha databases .

### Wrangler CLI Authentication

* **General Commands**
  * URL: `https://developers.cloudflare.com/workers/wrangler/commands/general/`
  * Description: General Wrangler commands for authentication, telemetry, and shell completions are documented here . You can authorize Wrangler with your Cloudflare account using OAuth .

* **Authentication Profiles**
  * URL: `https://developers.cloudflare.com/workers/wrangler/`
  * Description: Authentication profiles allow you to maintain separate logins and switch accounts per directory .

* **Device Flow Login**
  * URL: `https://developers.cloudflare.com/changelog/post/2026-08-04-wrangler-login-device-flow/`
  * Description: The wrangler login command now supports the OAuth 2.0 Device Authorization Grant ↗ . You can pass --device to authenticate without starting a temporary callback .

* **Auth Token Retrieval**
  * URL: `https://developers.cloudflare.com/changelog/post/2025-12-18-wrangler-auth-token/`
  * Description: Wrangler now includes a new wrangler auth token command that retrieves your current authentication token or credentials for use with other tools .

### API Tokens & Permissions

* **API Token Permissions**
  * URL: `https://developers.cloudflare.com/fundamentals/api/reference/permissions/`
  * Description: This reference documents all available token permissions . The "API Tokens Read" permission grants read access to a user's API tokens . The "API Tokens Edit" permission grants write access to a user's API tokens .

* **Create API Token**
  * URL: `https://developers.cloudflare.com/fundamentals/api/get-started/create-token/`
  * Description: You can modify the token's permissions during the creation flow . After selecting a permissions group (Account, User, or Zone), you must choose what level of access to grant the token .

* **Account API Tokens**
  * URL: `https://developers.cloudflare.com/fundamentals/api/get-started/account-owned-tokens/`
  * Description: Account API tokens allow you to set up durable integrations that can act as service principals with their own specific set of permissions .