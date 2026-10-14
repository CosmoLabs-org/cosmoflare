// D1 database profile collector: one database's metadata — created date,
// version, region, table count, file size, read-replication mode — shaped
// for the pager's D1 profile view. Follows the house rule every collector
// obeys: the section fails independently, a failure becomes an errors[] line
// and an omitted section, never a failed response and never an invented
// number. No live SQL: the pager renders the 24h read/write analytics the
// summary already carries, so this endpoint sticks to REST metadata.

import { fetchRetry } from "./retry";
import { fetchD1List } from "./summary";
import type { OpsEnv } from "./scheduled";

const API = "https://api.cloudflare.com/client/v4";

export interface D1ReadReplicationRow {
  mode: string;
  parts: number | null;
}

export interface D1MetadataRow {
  name: string;
  /** ISO timestamp the database was created. */
  createdAt: string | null;
  /** D1 version (e.g. "production"). */
  version: string | null;
  /** Region the primary lives in (e.g. "WNAM"). */
  region: string | null;
  numTables: number | null;
  /** File size in bytes. */
  fileSize: number | null;
  readReplication: D1ReadReplicationRow | null;
}

export interface D1ProfilePayload {
  generatedAt: string;
  /** null when the name→uuid lookup or the metadata GET fails — the errors[]
   *  line carries why, the view renders no metadata card. */
  metadata: D1MetadataRow | null;
  /** One line per failed section — an omitted section is data absent, never invented. */
  errors: string[];
}

// Injectable list resolution (collectSummary's deps pattern): the index.ts
// route hands in the cached "d1-list" fetcher so a profile view reuses the
// same cached list the summary already paid for, instead of a second
// /d1/database page per profile view.
export interface D1ProfileDeps {
  d1List?: (token: string, accountId: string) => Promise<{ uuid: string; name: string }[]>;
}

// One soft-failing GET: !success or a transport error becomes a thrown Error
// the caller's catch turns into an errors[] line (getSection's pattern in
// worker_profile.ts).
async function getSection<T>(token: string, path: string): Promise<T> {
  const res = await fetchRetry(`${API}${path}`, { headers: { Authorization: `Bearer ${token}` } });
  const body = (await res.json()) as { success?: boolean; result?: T; errors?: { message: string }[] };
  if (!res.ok || body.success !== true) throw new Error(body.errors?.[0]?.message ?? `HTTP ${res.status}`);
  return body.result as T;
}

export async function collectD1Profile(
  env: OpsEnv,
  name: string,
  deps: D1ProfileDeps = {},
  now = new Date(),
): Promise<D1ProfilePayload> {
  const out: D1ProfilePayload = { generatedAt: now.toISOString(), metadata: null, errors: [] };
  const list = deps.d1List ?? ((tok: string, acct: string) => fetchD1List(tok, acct));

  // 1. Metadata: resolve name→uuid off the account's database list, then GET
  //    the database. Either leg failing degrades to an errors line.
  try {
    const databases = await list(env.CF_API_TOKEN, env.CF_ACCOUNT_ID);
    const db = databases.find((d) => d.name === name);
    if (db === undefined) {
      out.errors.push(`metadata unavailable: no database named ${name}`);
      return out;
    }
    const meta = await getSection<{
      name?: string;
      created_at?: string | null;
      version?: string | null;
      running_in_region?: string | null;
      num_tables?: number | null;
      file_size?: number | null;
      read_replication?: { mode?: string | null; parts?: number | null } | null;
    } | null>(env.CF_API_TOKEN, `/accounts/${env.CF_ACCOUNT_ID}/d1/database/${db.uuid}`);
    out.metadata = {
      name: meta?.name ?? name,
      createdAt: meta?.created_at ?? null,
      version: meta?.version ?? null,
      region: meta?.running_in_region ?? null,
      numTables: meta?.num_tables ?? null,
      fileSize: meta?.file_size ?? null,
      readReplication: meta?.read_replication?.mode
        ? { mode: meta.read_replication.mode, parts: meta.read_replication.parts ?? null }
        : null,
    };
  } catch (err) {
    out.errors.push(`metadata unavailable: ${err instanceof Error ? err.message : String(err)}`);
  }

  return out;
}
