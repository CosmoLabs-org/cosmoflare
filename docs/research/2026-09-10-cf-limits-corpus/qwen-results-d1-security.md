Yes. The most important mental model is:

> D1 does not use traditional database usernames/passwords for Workers.  
> Access is controlled through **Cloudflare account permissions**, **API tokens**, **Wrangler identity**, and **Worker bindings**.

So securing D1 with Wrangler is mostly about securing:

1. who can call Cloudflare APIs,
2. which environments they can touch,
3. how SQL and migrations are executed,
4. how Worker code accesses the database,
5. where local database files and exports are stored.

---

# Qwen results — Best practices for securing D1 database access via Wrangler

Provenance: asked 2026-09-11 in the Qwen research conversation. Prompt:
"Are there any best practices for securing D1 database access when using
Wrangler CLI?" Captured 2026-09-11. Feeds FEAT-019 (token scopes),
FEAT-020 (danger_level/destructive command metadata), FEAT-018
(push-sql dry-run/confirmation behavior).

---

# 1. Use least-privilege authentication

## Prefer `wrangler login` for humans

For interactive development, prefer OAuth login:

```bash
wrangler login
```

Then verify identity:

```bash
wrangler whoami
```

This avoids putting long-lived API keys in shell history or config files.

## Avoid the Global API Key

Do not use the account-wide Global API Key for Wrangler or CI if you can avoid it. Prefer scoped API tokens.

## Use scoped API tokens for automation

For CI/CD, create a dedicated Cloudflare API token with only the permissions needed.

Typical minimum permissions:

| Task | Minimum likely permissions |
|---|---|
| List/inspect D1 databases | D1 Read |
| Run read-only SQL remotely | D1 Read, if API model permits read-only distinction |
| Create/delete D1 databases | D1 Edit |
| Run migrations/write SQL | D1 Edit |
| Deploy Worker that binds to D1 | Workers Scripts Edit/Upload |
| Manage Worker routes/custom domains | Workers Routes Edit, if deploying routes |
| Tail Worker logs | Workers Tail Read |
| Manage secrets | Workers Scripts Edit or equivalent secret permission |

If a job only validates SQL or lists databases, do not give it D1 Edit.

## Use separate tokens for separate jobs

Example token separation:

```text
ci-worker-deploy-token
  - Workers Scripts Edit
  - Workers Routes Edit if needed
  - No D1 Edit

ci-d1-migration-token
  - D1 Edit
  - Maybe D1 Read
  - No account admin

readonly-debug-token
  - D1 Read
  - Workers Tail Read
```

This limits blast radius if one token leaks.

## Use short-lived tokens where possible

If your CI provider supports OIDC or short-lived Cloudflare authentication, prefer that over long-lived static tokens.

If you must use static tokens:

- set expiry,
- rotate regularly,
- revoke unused tokens,
- store them in a secret manager,
- never print them in logs.

Example:

```bash
export CLOUDFLARE_API_TOKEN="$(vault read -field=token secret/cloudflare/ci/d1-migrator)"
wrangler whoami
```

---

# 2. Treat D1 access as account access, not just database access

D1 database IDs are identifiers, not passwords.

For example, this is normal in `wrangler.toml`:

```toml
[[d1_databases]]
binding = "DB"
database_name = "my-app-dev"
database_id = "12345678-aaaa-bbbb-cccc-123456789abc"
```

The `database_id` is not a secret by itself. However, combined with a valid API token, it tells an attacker exactly which database to target.

More importantly, many Cloudflare permissions are account-scoped. A token with D1 Edit may be able to affect all D1 databases in the account, not just one database.

## Best practice

If you need strong isolation between teams or environments, use separate Cloudflare accounts:

```text
account: myapp-dev
account: myapp-staging
account: myapp-prod
```

Then give developers broad access to dev, but restricted access to prod.

If you keep dev/staging/prod in the same account, enforce process controls carefully:

- separate Wrangler environments,
- separate tokens,
- explicit production confirmation,
- separate CI jobs,
- audit logging.

---

# 3. Separate dev, staging, and production databases explicitly

Do not rely on humans remembering which database they are targeting.

Use Wrangler environments.

Example:

```toml
name = "my-app"
main = "src/index.ts"
compatibility_date = "2026-01-01"

[[d1_databases]]
binding = "DB"
database_name = "my-app-dev"
database_id = "dev-db-id"

[env.staging]
name = "my-app-staging"

[[env.staging.d1_databases]]
binding = "DB"
database_name = "my-app-staging"
database_id = "staging-db-id"

[env.production]
name = "my-app"

[[env.production.d1_databases]]
binding = "DB"
database_name = "my-app-prod"
database_id = "prod-db-id"
```

Then:

```bash
wrangler deploy
wrangler deploy --env staging
wrangler deploy --env production
```

For D1 commands, also be explicit about target:

```bash
wrangler d1 execute my-app-dev --local --command "SELECT 1"
wrangler d1 execute my-app-staging --remote --command "SELECT 1"
wrangler d1 execute my-app-prod --remote --command "SELECT 1"
```

Your wrapper should display the target clearly:

```text
Target: remote
Account: 1234abcd...
Database: my-app-prod
Database ID: 12345678-aaaa-bbbb-cccc-123456789abc
```

---

# 4. Protect local D1 state

When you use local D1, Wrangler stores SQLite-like local state on disk.

Commands like:

```bash
wrangler d1 execute my-db --local --command "SELECT 1"
wrangler dev
```

may create or modify local state under directories such as:

```text
.wrangler/
```

or a custom path if using:

```bash
--persist-to ./local-db.sqlite
```

## Best practices

Add Wrangler state to `.gitignore`:

```gitignore
.wrangler/
.dev.vars
node_modules/
*.sqlite
*.sqlite3
```

Do not commit local database files.

If your dev database contains realistic production-like data:

- encrypt developer disks,
- restrict access to CI artifacts,
- clean workspaces after jobs,
- avoid persisting local DBs unnecessarily,
- prefer synthetic seed data instead of production dumps.

For CI, avoid uploading `.wrangler/state` as an artifact unless required.

---

# 5. Keep secrets out of `wrangler.toml`

`wrangler.toml` should contain configuration, not credentials.

Okay to put in `wrangler.toml`:

```toml
name = "my-app"
main = "src/index.ts"
compatibility_date = "2026-01-01"

[[d1_databases]]
binding = "DB"
database_name = "my-app-dev"
database_id = "..."
```

Avoid putting in `wrangler.toml`:

```toml
api_token = "..."
CLOUDFLARE_API_TOKEN = "..."
STRIPE_SECRET_KEY = "..."
DATABASE_PASSWORD = "..."
```

For local development secrets, use `.dev.vars`:

```text
# .dev.vars
MY_THIRD_PARTY_API_KEY="..."
```

And gitignore it:

```gitignore
.dev.vars
```

For production secrets, use Worker secrets:

```bash
wrangler secret put MY_SECRET
```

Not D1.

Do not store application secrets in D1 unless you have an application-level encryption strategy. D1 is a database, not a secret manager.

---

# 6. Protect production SQL execution

The most dangerous D1 commands are:

```bash
wrangler d1 execute <database> --remote
wrangler d1 migrations apply <database> --remote
wrangler d1 delete <database>
```

These can modify or destroy production data.

## Best practices

### Require confirmation for remote writes

For humans:

```text
You are about to execute SQL against production database:

  Account:     my-prod-account
  Database:    my-app-prod
  Database ID: 12345678-aaaa-bbbb-cccc-123456789abc

Type the database name to confirm:
```

### Avoid arbitrary remote SQL in production

Prefer:

```text
1. Migration file created in PR.
2. Migration reviewed.
3. Migration tested on local/dev/staging.
4. Migration applied by CI with approved token.
```

Instead of:

```bash
wrangler d1 execute my-app-prod --remote --command "UPDATE users SET admin = 1 WHERE email = 'someone@example.com'"
```

Ad hoc production SQL should be treated as an incident/break-glass operation.

### Use dry runs where possible

If your tooling supports it, validate SQL before executing:

```bash
wrangler d1 execute my-db --local --file ./schema.sql
```

Then:

```bash
wrangler d1 execute my-db --remote --file ./schema.sql
```

For your own CLI, add:

```text
--dry-run
--plan
--require-target=staging|production
--confirm <database-name>
```

### Use separate migration tokens

The token used to run migrations should be separate from the token used to deploy Workers or inspect logs.

---

# 7. Secure migrations

Wrangler migrations are local SQL files applied through D1 query execution. That means migration security is both code security and database security.

## Best practices

### Store migrations in source control

Example:

```text
migrations/
  0001_create_users.sql
  0002_add_sessions.sql
```

Migrations should be reviewed like code.

### Do not generate migration SQL from untrusted input

Migrations should not be dynamically built from user input.

### Test migrations locally first

```bash
wrangler d1 migrations apply my-db --local
```

Then staging:

```bash
wrangler d1 migrations apply my-db --remote --env staging
```

Then production:

```bash
wrangler d1 migrations apply my-db --remote --env production
```

Depending on your setup, flags may differ, but the principle is: local → dev → staging → production.

### Prevent concurrent migrations

In CI, use concurrency locks:

```yaml
concurrency:
  group: d1-migrations-production
  cancel-in-progress: false
```

Two concurrent migration jobs can cause confusing state, especially if migration tracking is partially applied.

### Keep destructive migrations explicit

Dangerous examples:

```sql
DROP TABLE users;
DELETE FROM users;
ALTER TABLE users DROP COLUMN email;
```

Require extra review or special labels for these.

---

# 8. Secure Worker runtime access to D1

Workers access D1 through bindings.

Example:

```ts
export interface Env {
  DB: D1Database;
}

export default {
  async fetch(request: Request, env: Env) {
    const result = await env.DB.prepare("SELECT COUNT(*) AS count FROM users").first();
    return Response.json(result);
  }
}
```

There is no D1 password inside the Worker. The binding itself grants access.

Therefore:

> Anyone who can deploy the Worker can usually change how the Worker uses the D1 binding.

## Best practices

### Protect deployment as strongly as database access

A Worker with a D1 binding can read/write that database according to the code deployed. Secure:

- repo access,
- branch protection,
- CI permissions,
- Wrangler deploy tokens,
- production deploy approvals.

### Do not expose raw SQL controls to users

Never do something like this:

```ts
const sql = await request.text();
const result = await env.DB.prepare(sql).all();
```

That is effectively remote code execution against your database.

Instead, expose narrow operations:

```ts
const url = new URL(request.url);
const id = url.searchParams.get("id");

const user = await env.DB
  .prepare("SELECT id, name, email FROM users WHERE id = ?")
  .bind(id)
  .first();
```

### Use prepared statements and bind parameters

Unsafe:

```ts
const query = `SELECT * FROM users WHERE email = '${email}'`;
const user = await env.DB.prepare(query).first();
```

Safe:

```ts
const user = await env.DB
  .prepare("SELECT * FROM users WHERE email = ?")
  .bind(email)
  .first();
```

D1 follows SQLite-style parameter binding. Use it.

### Avoid dynamic table/column names from user input

Bind parameters do not protect identifiers like table names or column names.

Avoid:

```ts
const table = userInput.table;
const query = `SELECT * FROM ${table}`;
```

If dynamic identifiers are necessary, validate against a strict allowlist:

```ts
const ALLOWED_TABLES = new Set(["users", "posts", "comments"]);

if (!ALLOWED_TABLES.has(table)) {
  throw new Error("Invalid table");
}
```

Still prefer fixed queries.

### Implement application-level authorization

D1 does not give you full database-user-level row permissions inside Workers. There is no built-in concept of:

```text
app_user can read posts but not delete users
```

If you need row-level or role-based controls, implement them in application logic.

Example:

```ts
if (!currentUser.canModerate(post.authorId)) {
  return new Response("Forbidden", { status: 403 });
}
```

Do not rely on the database alone for authorization.

---

# 9. Protect HTTP endpoints that touch D1

If your Worker exposes an API that queries D1, protect that API.

## Use authentication

Do not expose sensitive D1-backed endpoints anonymously.

Use one or more of:

- session cookies,
- JWTs,
- Cloudflare Access,
- API keys,
- OAuth,
- mTLS,
- signed requests.

## Rate limit expensive queries

Use Cloudflare WAF rate limiting for endpoints that perform:

- full-table scans,
- search,
- report generation,
- bulk exports,
- complex joins.

## Paginate results

Avoid returning unbounded data:

```ts
const rows = await env.DB.prepare("SELECT * FROM logs").all();
```

Prefer:

```ts
const rows = await env.DB
  .prepare("SELECT * FROM logs ORDER BY id DESC LIMIT ?")
  .bind(limit)
  .all();
```

And validate `limit`:

```ts
const safeLimit = Math.min(Number(limit) || 50, 100);
```

## Avoid exposing internal tables

Do not expose migration tables, internal metadata, or raw SQL errors to end users.

Return generic error messages publicly, and log details internally.

---

# 10. Secure direct D1 API usage outside Workers

If you call the D1 REST API directly from scripts, treat that like database admin access.

Example conceptual request:

```bash
curl -X POST \
  "https://api.cloudflare.com/client/v4/accounts/$ACCOUNT_ID/d1/database/$DATABASE_ID/query" \
  -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{
    "sql": "SELECT COUNT(*) FROM users"
  }'
```

The token in `Authorization` is the credential.

## Best practices

- Do not hardcode tokens in scripts.
- Do not pass tokens as CLI args visible in `ps`.
- Use environment variables from a secret manager.
- Redact tokens from logs.
- Use different tokens for read-only and write workloads.
- Prefer short-lived credentials.

Bad:

```bash
curl -H "Authorization: Bearer abc123..." ...
```

Better:

```bash
export CLOUDFLARE_API_TOKEN="$(secret-manager get cloudflare/d1-readonly)"
curl -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" ...
```

---

# 11. Protect D1 exports

If you use:

```bash
wrangler d1 export my-db
```

or an equivalent API export, the output may contain your entire database schema and data.

Treat exports as sensitive backups.

## Best practices

- Store exports in encrypted storage.
- Restrict download access.
- Do not commit exports to Git.
- Do not paste exports into issue trackers.
- Redact or sanitize data before sharing.
- Delete temporary export files after use.
- Avoid exporting production data into developer laptops unless necessary.

Example `.gitignore`:

```gitignore
*.sql
*.dump
*.sqlite
backups/
```

If developers need realistic data, prefer generated fixture data over production exports.

---

# 12. Use backups and time travel defensively

Before risky migrations or destructive operations, know your recovery path.

D1 has time-travel/restore behavior depending on plan and current features.

Best practices:

- Know your retention window.
- Test restore procedures before you need them.
- Take note of timestamps before risky operations.
- Prefer reversible migrations.
- Avoid immediate hard deletes for critical data; use soft deletes where appropriate.
- For especially risky changes, create a backup/export first.

Example operational sequence:

```text
1. Export or verify time-travel timestamp.
2. Apply migration on staging.
3. Validate app behavior.
4. Apply migration to production.
5. Monitor errors.
6. Keep restore point until confident.
```

---

# 13. Audit and monitor D1-related actions

Cloudflare account audit logs can help track who changed what.

Monitor or alert on:

- D1 database creation,
- D1 database deletion,
- API token creation,
- new Worker deployments,
- changes to Worker bindings,
- changes to account members/roles,
- unusual migration runs,
- large exports,
- production SQL execution.

For your own CLI wrapper, log:

```text
timestamp
actor
account ID
database ID
database name
environment
command
target: local/remote
operation type: read/write/admin
request ID if available
```

Redact secrets and tokens.

---

# 14. CI/CD hardening checklist

For GitHub Actions, GitLab CI, CircleCI, etc.:

## Use separate CI tokens

Do not reuse developer tokens.

## Pin Wrangler version

Avoid supply-chain surprises by pinning versions.

Example with npm:

```bash
npm exec --yes -- wrangler@4.0.0 --version
```

Or in `package.json`:

```json
{
  "devDependencies": {
    "wrangler": "^4.0.0"
  }
}
```

Verify major version compatibility for your project.

## Store tokens in CI secrets

Do not store them in plain files.

GitHub Actions example:

```yaml
env:
  CLOUDFLARE_API_TOKEN: ${{ secrets.CLOUDFLARE_D1_DEPLOY_TOKEN }}
```

## Restrict production deploys

Use:

- protected branches,
- required reviews,
- required status checks,
- environment protection rules,
- manual approval for production,
- separate production tokens.

Example conceptual flow:

```text
PR opened
  -> lint SQL
  -> run local D1 migrations
  -> run tests against local D1
  -> deploy staging automatically

Production approval
  -> deploy Worker
  -> apply migrations with production token
```

## Do not give PR builds production credentials

Pull requests from forks or untrusted branches should not receive production Cloudflare tokens.

Use read-only or no credentials for most checks.

---

# 15. Local development security checklist

When developing locally:

```bash
wrangler dev
```

with D1 bindings, remember:

- local D1 state may be plaintext on disk,
- `.dev.vars` may contain local secrets,
- seed data may contain sensitive values,
- local SQLite files can be copied accidentally.

Use:

```gitignore
.wrangler/
.dev.vars
*.sqlite
*.sqlite3
```

Prefer:

```bash
wrangler d1 execute my-db --local --file ./seed.sql
```

with fake/synthetic data instead of production data.

---

# 16. Recommended permission model

A practical model for teams:

## Developer, day-to-day

Permissions:

```text
Workers Scripts Edit for dev account/project
D1 Edit for dev account
Workers Tail Read
```

No production token.

## CI deploy to staging

Permissions:

```text
Workers Scripts Edit
Workers Routes Edit if required
D1 Edit for staging account or restricted environment
```

## CI deploy to production

Permissions:

```text
Workers Scripts Edit
Workers Routes Edit if required
No D1 Edit unless migration job
```

## CI D1 migration to production

Permissions:

```text
D1 Edit
```

No broader account admin.

## Break-glass admin

Permissions:

```text
D1 Edit
Workers Scripts Edit
Maybe account settings
```

Stored in emergency access workflow, not normal CI.

---

# 17. Practical secure workflow example

## Local development

```bash
wrangler login
wrangler whoami

wrangler d1 create app-dev

wrangler d1 migrations create app-dev create_users_table
wrangler d1 migrations apply app-dev --local

wrangler dev
```

## Test against dev database

```bash
wrangler d1 migrations apply app-dev --remote
wrangler d1 execute app-dev --remote --command "SELECT COUNT(*) FROM users"
```

## Deploy staging

```bash
wrangler deploy --env staging
```

## Apply staging migrations

```bash
wrangler d1 migrations apply app-staging --remote --env staging
```

## Production deploy with approval

```bash
wrangler deploy --env production
```

## Production migration with explicit token

```bash
CLOUDFLARE_API_TOKEN="$PROD_D1_MIGRATION_TOKEN" \
  wrangler d1 migrations apply app-prod --remote --env production
```

Your CLI should make each of these targets explicit and hard to confuse.

---

# 18. Common mistakes to avoid

## Mistake 1: Using one token for everything

Avoid a single account-admin token for dev, CI, migrations, and debugging.

## Mistake 2: Treating database ID as a secret

Database IDs are identifiers. They are okay to store in config, but do not rely on them for security.

## Mistake 3: Storing API tokens in `wrangler.toml`

Use environment variables, OAuth, or secret managers.

## Mistake 4: Letting `wrangler dev` accidentally hit production

Always distinguish local vs remote.

## Mistake 5: Running unreviewed remote SQL

Remote SQL execution should be reviewed, logged, and restricted.

## Mistake 6: Building SQL from user input

Use bind parameters and avoid dynamic SQL.

## Mistake 7: Exposing raw D1 results to users

Return only necessary fields.

## Mistake 8: Committing `.wrangler/` or `.dev.vars`

Gitignore them.

## Mistake 9: Exporting production data casually

Treat exports as sensitive backups.

## Mistake 10: No restore plan

Know how to recover before running destructive migrations.

---

# 19. Summary checklist

Secure D1 with Wrangler by doing the following:

```text
Authentication
  [ ] Use wrangler login for humans
  [ ] Use scoped API tokens for automation
  [ ] Avoid Global API Key
  [ ] Use short-lived/OIDC auth where possible
  [ ] Rotate and revoke tokens regularly

Authorization
  [ ] Use least-privilege D1 permissions
  [ ] Separate read-only and write tokens
  [ ] Separate Worker deploy tokens from D1 migration tokens
  [ ] Use separate accounts for prod/non-prod if strict isolation is needed

Configuration
  [ ] Keep database IDs in wrangler.toml if acceptable
  [ ] Keep tokens/secrets out of wrangler.toml
  [ ] Use .dev.vars for local secrets
  [ ] Gitignore .dev.vars and .wrangler/

Environments
  [ ] Separate dev/staging/prod databases
  [ ] Use Wrangler environments
  [ ] Require explicit production targeting
  [ ] Show account/database target before remote operations

SQL/Migrations
  [ ] Store migrations in source control
  [ ] Review migration SQL like code
  [ ] Test migrations locally and in staging
  [ ] Avoid ad hoc production SQL
  [ ] Confirm destructive operations
  [ ] Use concurrency locks in CI

Worker runtime
  [ ] Protect Worker deployment pipeline
  [ ] Use D1 prepared statements
  [ ] Never execute raw client-provided SQL
  [ ] Implement app-level authorization
  [ ] Paginate and rate limit D1-backed endpoints

Local state
  [ ] Protect local SQLite state
  [ ] Do not commit local DB files
  [ ] Use synthetic seed data where possible

Exports/Backups
  [ ] Treat exports as sensitive
  [ ] Encrypt and restrict export storage
  [ ] Know time-travel/restore retention
  [ ] Test restore procedures

Audit
  [ ] Log D1 admin actions
  [ ] Monitor database creation/deletion
  [ ] Monitor token creation and deployment changes
  [ ] Redact secrets from logs
```

The short version:

> Secure D1 by securing Wrangler’s identity, limiting API token permissions, separating environments, protecting local state, reviewing migrations, and never exposing raw SQL execution to untrusted input.