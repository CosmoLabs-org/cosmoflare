// Cosmoflare Ops project attribution (FEAT-052): map a Cloudflare resource
// name (Worker script, D1 database, KV namespace, R2 bucket, DO namespace) to
// the CosmoLabs project that drives it. Pure functions, no I/O.

/** Default operator map. The heuristics must not need these — they exist to
 *  catch names the heuristics cannot see through. */
export const DEFAULT_PROJECT_MAP: Record<string, string> = {
  mycarguide: "mycarguide",
  cosmolearning: "cosmolearning",
  churches: "churches",
  noblecoffee: "noblecoffee",
  handleshop: "handleshop",
  noelymaria: "noelymaria",
  cosmolabs: "cosmolabs",
  cosmoflare: "cosmoflare",
};

const AFFIXES = [
  "-db",
  "-prod",
  "-staging",
  "-worker",
  "-api",
  "-cache",
  "-kv",
  "-bucket",
  "-assets",
  "cosmoflare-",
];

// TLDs treated as noise: churches.app → churches (first label only).
const COMMON_TLDS = new Set(["com", "org", "net", "app", "dev", "io", "co", "ai"]);

function normalise(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]/g, "");
}

/** Attribute a resource name to a project: explicit map first (case-insensitive
 *  exact), then affix stripping, then domain first-label joining. */
export function projectFor(name: string, map: Record<string, string> = DEFAULT_PROJECT_MAP): string {
  const lower = name.toLowerCase();
  if (map[lower]) return map[lower];
  const stripped = AFFIXES.reduce(
    (s, a) => (a.endsWith("-") ? s.startsWith(a) ? s.slice(a.length) : s : s.endsWith(a) ? s.slice(0, -a.length) : s),
    lower,
  );
  // Domain-ish names: mycar.guide → mycarguide (labels joined); when the TLD
  // is generic noise, keep only the first label (churches.app → churches).
  if (stripped.includes(".")) {
    const parts = stripped.split(".");
    const tld = parts[parts.length - 1];
    if (COMMON_TLDS.has(tld)) return normalise(parts.slice(0, -1).join(""));
    return normalise(stripped);
  }
  return normalise(stripped);
}
