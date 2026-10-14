// Stamp the service worker on every build (BUG: the update modal never
// fired — sw.js is a static file whose bytes never changed, so browsers
// byte-compared it, saw "same worker" and never updated). Tying
// CACHE_NAME to the build's asset hash makes each deploy's sw.js unique;
// the SW's own activate handler already deletes older caches.
import { readdirSync, readFileSync, writeFileSync } from "node:fs";

const dist = new URL("./dist/", import.meta.url);
const assets = readdirSync(new URL("assets/", dist)).filter((f) => /^index-[A-Za-z0-9_-]+\.js$/.test(f));
if (assets.length === 0) throw new Error("stamp-sw: no index-*.js in dist/assets — run after vite build");
const hash = assets[0].match(/index-([A-Za-z0-9_-]+)\.js$/)?.[1];
if (!hash) throw new Error(`stamp-sw: cannot parse hash from ${assets[0]}`);

const swUrl = new URL("sw.js", dist);
const sw = readFileSync(swUrl, "utf8");
const stamped = sw.replace(/pager-shell-v[0-9a-zA-Z_-]+/, `pager-shell-${hash.slice(0, 12)}`);
if (stamped === sw) throw new Error("stamp-sw: CACHE_NAME literal not found in dist/sw.js");
writeFileSync(swUrl, stamped);

// Build marker: index.html carries the same stamp as a meta tag so the
// running app can SHOW which build it is (the "am I on the latest?"
// question becomes answerable on-device).
const indexUrl = new URL("index.html", dist);
const id = hash.slice(0, 12);
let html = readFileSync(indexUrl, "utf8");
const meta = `<meta name="pager-build" content="${id}" />`;
if (html.includes("pager-build")) {
  html = html.replace(/<meta name="pager-build" content="[^"]*" \/>/, meta);
} else {
  html = html.replace("</head>", `  ${meta}\n</head>`);
}
writeFileSync(indexUrl, html);
console.log(`stamp-sw: CACHE_NAME -> pager-shell-${id}, build marker -> ${id}`);
