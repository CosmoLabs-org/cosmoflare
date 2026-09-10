Below is the practical Wrangler → Cloudflare API mapping for **D1 database management**.

> Assumption: current Wrangler v3/v4 and Cloudflare REST API v4.
> Base API URL:
>
> ```text
> https://api.cloudflare.com/client/v4
> ```
>
> D1 endpoints are **account-scoped**:
>
> ```text
> /accounts/{account_id}/d1/database/...
> ```

---

# Qwen results — Wrangler D1 commands mapped to Cloudflare API endpoints

Provenance: asked 2026-09-11 in the Qwen research conversation. Prompt:
"What are the specific CLI commands for managing D1 databases in Wrangler,
and how do they map to Cloudflare's API endpoints?" Captured
2026-09-11. Feeds FEAT-022 (D1 depth: migrations, time-travel, backups,
export/import).

---

# 1. Core Wrangler D1 commands

The main Wrangler D1 commands are:

```bash
wrangler d1 create <name>
wrangler d1 list
wrangler d1 delete <name-or-id>
wrangler d1 execute <database>
wrangler d1 export <database>
wrangler d1 migrations create <database> <migration-name>
wrangler d1 migrations list <database>
wrangler d1 migrations apply <database>
```

Depending on Wrangler version, you may also see:

```bash
wrangler d1 time-travel <database>
```

Treat `time-travel` and some export flags as newer/plan-sensitive commands and verify against your installed Wrangler version.

---

# 2. Command-to-API mapping table

| Wrangler command | What it does | Cloudflare API endpoint | HTTP method | Notes |
|---|---|---:|---:|---|
| `wrangler d1 create <name>` | Creates a new D1 database. | `/accounts/{account_id}/d1/database` | `POST` | Body usually contains `{ "name": "<name>" }`. Returns database ID. |
| `wrangler d1 list` | Lists D1 databases in the account. | `/accounts/{account_id}/d1/database` | `GET` | May support pagination/search flags depending on Wrangler version. |
| `wrangler d1 delete <name-or-id>` | Deletes a D1 database. | `/accounts/{account_id}/d1/database/{database_id}` | `DELETE` | Destructive. Wrangler may resolve database name to ID first. |
| `wrangler d1 execute <db> --command "SQL"` | Executes SQL against a database. | `/accounts/{account_id}/d1/database/{database_id}/query` | `POST` | Body contains SQL and optional params. |
| `wrangler d1 execute <db> --file file.sql` | Executes a SQL file. | `/accounts/{account_id}/d1/database/{database_id}/query` or repeated query calls | `POST` | Wrangler parses/sends statements. |
| `wrangler d1 execute <db> --batch batch.json` | Executes batched SQL statements. | D1 bulk/batch query endpoint, or repeated `/query` calls depending on Wrangler version | `POST` | Batch payload is usually an array of SQL/params objects. |
| `wrangler d1 export <database>` | Exports database contents as SQL. | `/accounts/{account_id}/d1/database/{database_id}/export` or related export flow | `POST`/`GET` | Export behavior may be synchronous or asynchronous depending on API version. Verify current docs. |
| `wrangler d1 migrations create <db> <name>` | Creates a local migration file. | No direct API call | — | Creates a file such as `migrations/0001_name.sql`. |
| `wrangler d1 migrations list <database>` | Lists local migrations and applied state. | Local filesystem + optional D1 query to migration tracking table | Usually `POST /query` | Wrangler checks which migrations have been applied. |
| `wrangler d1 migrations apply <database>` | Applies pending migrations. | `/accounts/{account_id}/d1/database/{database_id}/query` | `POST` | Runs migration SQL and records applied migrations. |
| `wrangler d1 time-travel <database>` | Restores/inspects a point-in-time state. | `/accounts/{account_id}/d1/database/{database_id}/time_travel` or `/restore`-style endpoint | `POST` | Verify exact endpoint/path in current D1 API docs. |

---

# 3. Detailed command mapping

## 3.1 `wrangler d1 create`

### CLI

```bash
wrangler d1 create my-database
```

### What Wrangler does

1. Reads your account ID from config, environment, or flags.
2. Calls the Cloudflare API to create a D1 database.
3. Returns the new database’s ID and often prints an example `wrangler.toml` binding.

### API mapping

```http
POST https://api.cloudflare.com/client/v4/accounts/{account_id}/d1/database
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "my-database"
}
```

### Typical response shape

```json
{
  "result": {
    "uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "name": "my-database",
    "version": "beta",
    "num_tables": 0,
    "file_size": 0,
    "created_at": "2026-01-01T00:00:00Z"
  },
  "success": true,
  "errors": [],
  "messages": []
}
```

The exact response fields can vary, but the important value is the database `uuid`/`id`.

### Binding example

After creating a database, Wrangler often suggests something like:

```toml
[[d1_databases]]
binding = "DB"
database_name = "my-database"
database_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

That `database_id` is what Workers use to bind to the database.

---

## 3.2 `wrangler d1 list`

### CLI

```bash
wrangler d1 list
```

Possible flags depending on version:

```bash
wrangler d1 list --json
```

### What Wrangler does

Lists all D1 databases in the account.

### API mapping

```http
GET https://api.cloudflare.com/client/v4/accounts/{account_id}/d1/database
```

Possibly with pagination/query parameters such as:

```http
GET /accounts/{account_id}/d1/database?page=1&per_page=20
```

### Use in wrappers

This endpoint is useful for:

- resolving database name to database ID,
- checking whether a database exists,
- enforcing account database count limits,
- building autocomplete.

---

## 3.3 `wrangler d1 delete`

### CLI

```bash
wrangler d1 delete my-database
```

or:

```bash
wrangler d1 delete <database-id>
```

Depending on Wrangler version, there may be a confirmation prompt or a `--force` flag.

### What Wrangler does

1. If given a database name, resolves it to a database ID using the list endpoint.
2. Calls the delete endpoint.

### API mapping

```http
DELETE https://api.cloudflare.com/client/v4/accounts/{account_id}/d1/database/{database_id}
```

### Important wrapper behavior

Your CLI should treat this as a high-risk destructive command.

Recommended safeguards:

- Require explicit confirmation.
- Support `--yes` or `--force` for automation.
- Show the database name and ID before deletion.
- Warn if the database is referenced by any Worker bindings in local config.
- Optionally require the user to pass the database name again:
  
  ```bash
  --confirm my-database
  ```

---

## 3.4 `wrangler d1 execute`

This is the most important D1 command.

### Basic CLI

```bash
wrangler d1 execute my-database --command "SELECT 1"
```

### Common flags

Depending on Wrangler version:

```bash
wrangler d1 execute <database> --command "<sql>"
wrangler d1 execute <database> --file ./schema.sql
wrangler d1 execute <database> --batch ./batch.json
wrangler d1 execute <database> --local
wrangler d1 execute <database> --remote
wrangler d1 execute <database> --json
wrangler d1 execute <database> --persist-to ./local-db.sqlite
```

---

## 3.5 `wrangler d1 execute --command`

### CLI

```bash
wrangler d1 execute my-database \
  --command "CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT UNIQUE)"
```

### Remote API mapping

For a remote D1 database:

```http
POST https://api.cloudflare.com/client/v4/accounts/{account_id}/d1/database/{database_id}/query
Authorization: Bearer <token>
Content-Type: application/json

{
  "sql": "CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT UNIQUE)"
}
```

With parameters:

```json
{
  "sql": "SELECT * FROM users WHERE id = ?",
  "params": [42]
}
```

### Local behavior

If you use:

```bash
wrangler d1 execute my-database --local --command "SELECT 1"
```

Wrangler does **not** call the Cloudflare D1 API.

Instead, it executes the SQL against a local SQLite-compatible database used by local Workers development.

Typical local state may live under something like:

```text
.wrangler/state/
```

or a custom location if you use:

```bash
--persist-to ./path/to/db.sqlite
```

---

## 3.6 `wrangler d1 execute --file`

### CLI

```bash
wrangler d1 execute my-database --file ./schema.sql
```

### What Wrangler does

Reads the SQL file and sends the statements to D1.

### API mapping

Logically:

```http
POST /accounts/{account_id}/d1/database/{database_id}/query
```

Depending on implementation, Wrangler may:

- send one combined request,
- split statements and send multiple requests,
- use a batch endpoint where available.

### Wrapper guidance

If your CLI implements `--file`, you should decide explicitly:

- whether statements run sequentially,
- whether one failed statement aborts the rest,
- whether the file is wrapped in a transaction,
- whether output is JSON or table-formatted.

For safety, do not implicitly run arbitrary multi-statement SQL without a clear execution model.

---

## 3.7 `wrangler d1 execute --batch`

### CLI

```bash
wrangler d1 execute my-database --batch ./batch.json
```

### Conceptual batch payload

A batch file usually contains multiple SQL statements, often with parameters:

```json
[
  {
    "sql": "INSERT INTO users (email) VALUES (?)",
    "params": ["alice@example.com"]
  },
  {
    "sql": "INSERT INTO users (email) VALUES (?)",
    "params": ["bob@example.com"]
  }
]
```

### API mapping

Depending on Wrangler version and current D1 API surface, this maps to one of:

```http
POST /accounts/{account_id}/d1/database/{database_id}/bulk
```

or multiple calls to:

```http
POST /accounts/{account_id}/d1/database/{database_id}/query
```

### Important edge case

D1 batch semantics matter a lot for wrappers.

You need to know:

- Is the batch atomic?
- Does one failed statement abort the whole batch?
- Are statements executed in order?
- Are results returned per statement?
- Does the batch count as one billing operation or multiple rows read/written?

For `cosmoflare`, expose this explicitly rather than hiding it.

---

## 3.8 `wrangler d1 execute --local` vs `--remote`

### Remote

```bash
wrangler d1 execute my-database --remote --command "SELECT 1"
```

Calls Cloudflare D1 API.

Endpoint:

```http
POST /accounts/{account_id}/d1/database/{database_id}/query
```

### Local

```bash
wrangler d1 execute my-database --local --command "SELECT 1"
```

Does not call Cloudflare.

Uses local SQLite-like emulation.

### Why this matters

Your CLI should show the execution target clearly:

```text
Target: local
Database: my-database
Path: .wrangler/state/v3/d1/miniflare-D1DatabaseObject.sqlite
```

or:

```text
Target: remote
Database: my-database
Database ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
Account ID: yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy
```

This prevents accidental production writes.

---

# 4. D1 migrations commands

D1 migrations in Wrangler are not a direct Cloudflare REST resource in the same way that databases and queries are. They are primarily a Wrangler convention.

---

## 4.1 `wrangler d1 migrations create`

### CLI

```bash
wrangler d1 migrations create my-database add_users_table
```

### What it does

Creates a local migration file, usually something like:

```text
migrations/0001_add_users_table.sql
```

or:

```text
migrations/202601010000_add_users_table.sql
```

depending on Wrangler version/configuration.

### API mapping

No direct Cloudflare API call.

This is local filesystem behavior.

---

## 4.2 `wrangler d1 migrations list`

### CLI

```bash
wrangler d1 migrations list my-database
```

### What it does

Lists migrations in the local `migrations` directory and indicates which ones have been applied.

### API mapping

Combination of:

1. Local filesystem scan.
2. Query against the D1 database to read migration state.

Usually, Wrangler queries a migration tracking table, commonly named something like:

```text
d1_migrations
```

The remote query path is therefore:

```http
POST /accounts/{account_id}/d1/database/{database_id}/query
```

with SQL similar in spirit to:

```sql
SELECT * FROM d1_migrations;
```

The exact schema/table name should be treated as Wrangler-internal unless documented.

---

## 4.3 `wrangler d1 migrations apply`

### CLI

```bash
wrangler d1 migrations apply my-database
```

### What it does

Applies pending migrations from the local `migrations` directory to the target D1 database.

### API mapping

For each pending migration, Wrangler sends SQL through:

```http
POST /accounts/{account_id}/d1/database/{database_id}/query
```

It also records the migration as applied, typically by inserting into the migration tracking table.

### Conceptual sequence

```text
1. List local migration files.
2. Query database for already-applied migrations.
3. For each pending migration:
   a. Read SQL file.
   b. Execute SQL against D1.
   c. Record migration tag as applied.
```

### Local vs remote

```bash
wrangler d1 migrations apply my-database --local
```

Applies to local development database.

```bash
wrangler d1 migrations apply my-database --remote
```

Applies to remote Cloudflare D1 database.

---

# 5. D1 export

## 5.1 `wrangler d1 export`

### CLI

```bash
wrangler d1 export my-database
```

Possible flags depending on version:

```bash
wrangler d1 export my-database --output ./backup.sql
```

### What it does

Exports database contents, usually as a SQL dump.

### API mapping

Likely maps to a D1 export endpoint under the database resource:

```http
POST /accounts/{account_id}/d1/database/{database_id}/export
```

or a related export retrieval flow.

Because export implementations can evolve, verify the exact current endpoint and response behavior.

### Wrapper behavior

Your CLI should support:

- output to stdout,
- output to file,
- remote vs local export,
- progress indication for large databases,
- size-limit enforcement.

Remember D1 has import/export and database size constraints. Exporting very large databases may be operationally constrained even if the command exists.

---

# 6. D1 time travel

Depending on Wrangler version, you may have:

```bash
wrangler d1 time-travel <database>
```

or something similar.

### Conceptual CLI

```bash
wrangler d1 time-travel my-database --timestamp "2026-01-01T00:00:00Z"
```

### What it does

Uses D1 point-in-time recovery to inspect or restore a database state.

### API mapping

Likely one of:

```http
POST /accounts/{account_id}/d1/database/{database_id}/time_travel
```

or:

```http
POST /accounts/{account_id}/d1/database/{database_id}/restore
```

with a timestamp payload such as:

```json
{
  "timestamp": "2026-01-01T00:00:00Z"
}
```

or Unix epoch seconds, depending on API design.

### Important notes

Time-travel retention depends on plan:

- Free tier has shorter retention.
- Workers Paid tier has longer retention.
- Restore operations may be rate-limited.

For `cosmoflare`, model this as a separate command with explicit confirmation because it can mutate or replace database state.

---

# 7. Database name vs database ID resolution

Many Wrangler commands accept a **database name**:

```bash
wrangler d1 execute my-database --command "SELECT 1"
```

But the Cloudflare API generally operates on a **database ID**:

```text
/accounts/{account_id}/d1/database/{database_id}/query
```

So Wrangler often does this internally:

```text
1. GET /accounts/{account_id}/d1/database
2. Find database where name == "my-database"
3. Use its uuid as database_id
4. Call the specific database endpoint
```

Your wrapper should support both:

```bash
cosmoflare d1 sql execute --database my-database
cosmoflare d1 sql execute --database-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

Prefer database ID in automation to avoid ambiguity.

---

# 8. Local development vs Cloudflare API

A very important mapping distinction:

| Command target | Uses Cloudflare API? | Backing store |
|---|---:|---|
| `wrangler d1 execute --remote` | Yes | Cloudflare D1 |
| `wrangler d1 execute --local` | No | Local SQLite-like database |
| `wrangler dev` with D1 binding | Usually no for D1 data | Local D1 emulation |
| `wrangler deploy` with D1 binding | Yes for Worker script upload | Does not migrate D1 data by itself |
| `wrangler d1 migrations apply --remote` | Yes | Cloudflare D1 |
| `wrangler d1 migrations apply --local` | No | Local SQLite-like database |

Your CLI should make this explicit in every D1 command.

Example output:

```text
Executing SQL against remote D1 database:
  Account:   <account-id>
  Database:  my-database
  ID:        xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

or:

```text
Executing SQL against local D1 database:
  Path: .wrangler/state/v3/d1/...
```

---

# 9. API request examples

## 9.1 Create database

```bash
curl -X POST \
  "https://api.cloudflare.com/client/v4/accounts/$ACCOUNT_ID/d1/database" \
  -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{
    "name": "my-database"
  }'
```

## 9.2 List databases

```bash
curl -X GET \
  "https://api.cloudflare.com/client/v4/accounts/$ACCOUNT_ID/d1/database" \
  -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN"
```

## 9.3 Delete database

```bash
curl -X DELETE \
  "https://api.cloudflare.com/client/v4/accounts/$ACCOUNT_ID/d1/database/$DATABASE_ID" \
  -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN"
```

## 9.4 Execute SQL

```bash
curl -X POST \
  "https://api.cloudflare.com/client/v4/accounts/$ACCOUNT_ID/d1/database/$DATABASE_ID/query" \
  -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{
    "sql": "SELECT * FROM users WHERE id = ?",
    "params": [1]
  }'
```

---

# 10. Recommended `cosmoflare` D1 command structure

You can model your wrapper commands cleanly:

```text
cosmoflare d1 database create
cosmoflare d1 database list
cosmoflare d1 database delete
cosmoflare d1 database get

cosmoflare d1 sql execute
cosmoflare d1 sql file
cosmoflare d1 sql batch

cosmoflare d1 export run

cosmoflare d1 migrations create
cosmoflare d1 migrations list
cosmoflare d1 migrations apply

cosmoflare d1 time-travel restore
```

Mapping to Wrangler:

| cosmoflare command | Wrangler equivalent |
|---|---|
| `d1 database create` | `wrangler d1 create` |
| `d1 database list` | `wrangler d1 list` |
| `d1 database delete` | `wrangler d1 delete` |
| `d1 database get` | No exact simple Wrangler equivalent; use list/API |
| `d1 sql execute` | `wrangler d1 execute --command` |
| `d1 sql file` | `wrangler d1 execute --file` |
| `d1 sql batch` | `wrangler d1 execute --batch` |
| `d1 export run` | `wrangler d1 export` |
| `d1 migrations create` | `wrangler d1 migrations create` |
| `d1 migrations list` | `wrangler d1 migrations list` |
| `d1 migrations apply` | `wrangler d1 migrations apply` |
| `d1 time-travel restore` | `wrangler d1 time-travel`, if available |

---

# 11. Permission expectations

For D1 operations, the relevant API token permission family is generally the D1 permission group.

| Operation | Minimum likely permission |
|---|---|
| List databases | D1 Read |
| Get database details | D1 Read |
| Create database | D1 Edit |
| Delete database | D1 Edit |
| Execute read-only SQL | D1 Read or D1 Edit depending on API model |
| Execute write SQL | D1 Edit |
| Migrations apply | D1 Edit |
| Export | D1 Read or D1 Edit depending on API model |
| Time-travel/restore | D1 Edit |

For a CLI wrapper, it is safer to require:

- `D1 Read` for read-only operations.
- `D1 Edit` for mutating operations.

---

# 12. Limits and guardrails your wrapper should apply

When mapping Wrangler D1 commands to API calls, enforce or warn about:

| Limit | Where it matters |
|---|---|
| Databases per account | `d1 database create` |
| Maximum database size | `sql execute`, `sql file`, `export`, `import` |
| Maximum SQL statement length | `sql execute --command`, `sql file`, migrations |
| Maximum bound parameters per query | `sql execute` with params |
| Maximum row/BLOB size | SQL inserts/updates |
| Maximum columns per table | migrations/schema creation |
| Import/export file size | `d1 export`, future import commands |
| Time-travel restore rate | `d1 time-travel restore` |
| Queries per Worker invocation | relevant when Workers call D1, not directly CLI |

---

# 13. Summary mapping

The concise mapping is:

```text
wrangler d1 create
  POST /accounts/{account_id}/d1/database

wrangler d1 list
  GET /accounts/{account_id}/d1/database

wrangler d1 delete
  DELETE /accounts/{account_id}/d1/database/{database_id}

wrangler d1 execute --command
  POST /accounts/{account_id}/d1/database/{database_id}/query

wrangler d1 execute --file
  POST /accounts/{account_id}/d1/database/{database_id}/query
  or multiple query calls

wrangler d1 execute --batch
  POST bulk/batch query endpoint
  or multiple query calls

wrangler d1 export
  D1 export endpoint under /accounts/{account_id}/d1/database/{database_id}
  verify exact current endpoint

wrangler d1 migrations create
  local filesystem only

wrangler d1 migrations list
  local filesystem + optional D1 query

wrangler d1 migrations apply
  POST /accounts/{account_id}/d1/database/{database_id}/query

wrangler d1 time-travel
  POST /accounts/{account_id}/d1/database/{database_id}/time_travel
  or restore-style endpoint
  verify exact current endpoint
```

The most important implementation detail: **Wrangler migrations are not a native Cloudflare REST resource; they are local files applied through D1 query execution.**