// Cosmoflare Pager service worker (FEAT-045).
//
// push event -> validate the envelope inline (mirrors src/payload.ts; a
// service worker here stays import-free) -> append to IndexedDB
// (db "cosmoflare-pager", store "alerts", keyPath "id") ->
// self.registration.showNotification tagged by severity.
// notificationclick -> focus an existing window or open one.
// fetch (FEAT-pRXPMMY offline app shell) -> cache-first for the hashed build
// assets and precached shell files, network-first for navigations, and an
// explicit pass-through for /api/* (real numbers are NEVER served by the
// service worker — cached API data lives only in the ApiClient).
//
// Payload protocol: pkg/cosmoflare/alertspush/payload.go. No credentials ever
// appear in a payload.

const DB_NAME = "cosmoflare-pager";
const DB_VERSION = 1;
const ALERTS_STORE = "alerts";
const SEVERITIES = ["info", "warning", "critical"];

// ---- Offline app shell (FEAT-pRXPMMY) ----

// Bump CACHE_NAME on every deploy: activate deletes any older cache, so a
// new build's assets replace the old ones wholesale.
const CACHE_NAME = "pager-shell-v2";

// The un-hashed shell files precached at install. Hashed /assets/* files are
// cached lazily on first fetch (cache-first), so this list stays short.
const PRECACHE_URLS = [
  "/index.html",
  "/manifest.webmanifest",
  "/icon-192.png",
  "/icon-512.png",
  "/icon-maskable-192.png",
  "/icon-maskable-512.png",
  "/favicon.svg",
];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches
      .open(CACHE_NAME)
      .then((cache) => cache.addAll(PRECACHE_URLS))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((names) =>
        Promise.all(
          names
            .filter((name) => name !== CACHE_NAME)
            .map((name) => caches.delete(name))
        )
      )
      .then(() => self.clients.claim())
  );
});

// isShellAsset: same-origin GET for a hashed build asset or one of the
// precached shell files — safe to serve cache-first (content is addressed or
// replaced wholesale on deploy).
function isShellAsset(url) {
  if (url.pathname.startsWith("/assets/")) return true;
  return PRECACHE_URLS.includes(url.pathname);
}

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") return;

  const url = new URL(request.url);

  // REAL-NUMBERS RULE (FEAT-pRXPMMY): /api/* responses are NEVER served by
  // the service worker — no cache reads, no cache writes. Cached API data
  // lives only in the ApiClient, which labels it with its real age. This
  // early return is load-bearing: every later branch must stay unreachable
  // for API URLs.
  if (url.pathname.startsWith("/api/")) return;

  // Everything non-same-origin (fonts, CDNs) passes through untouched.
  if (url.origin !== self.location.origin) return;

  // Hashed build assets + precached shell files: cache-first. Immutable
  // content (hashed filenames) or wholesale-replaced on deploy (shell files),
  // so a cache hit is always correct.
  if (isShellAsset(url)) {
    event.respondWith(
      caches.match(request).then(
        (hit) =>
          hit ||
          fetch(request).then((response) => {
            if (response.ok) {
              const copy = response.clone();
              caches.open(CACHE_NAME).then((cache) => cache.put(request, copy));
            }
            return response;
          })
      )
    );
    return;
  }

  // Navigations: network-first so a deploy's new index.html (and its new
  // hashed assets) apply whenever online; the cached shell opens the app
  // when offline.
  if (request.mode === "navigate") {
    event.respondWith(
      fetch(request)
        .then((response) => {
          if (response.ok) {
            const copy = response.clone();
            caches.open(CACHE_NAME).then((cache) => cache.put("/index.html", copy));
          }
          return response;
        })
        .catch(() =>
          caches
            .match("/index.html")
            .then((hit) => hit || new Response("offline", { status: 503, statusText: "offline" }))
        )
    );
    return;
  }

  // Everything else (other same-origin GETs) passes through untouched.
});

// Vibration patterns (ms on/off) escalate with severity.
const VIBRATE = {
  info: [100],
  warning: [200, 100, 200],
  critical: [300, 100, 300, 100, 300],
};

// validateEnvelope mirrors alertspush.Payload.Validate: non-empty
// id/service/title and a severity from the known vocabulary. detail is
// optional. Like parsePayload, fired_at must be a parseable date.
function validateEnvelope(data) {
  if (typeof data !== "object" || data === null) return null;
  const id = data.id;
  const severity = data.severity;
  const service = data.service;
  const title = data.title;
  const detail = data.detail;
  const firedAt = data.fired_at;
  if (typeof id !== "string" || id === "") return null;
  if (!SEVERITIES.includes(severity)) return null;
  if (typeof service !== "string" || service === "") return null;
  if (typeof title !== "string" || title === "") return null;
  if (typeof firedAt !== "string" || Number.isNaN(Date.parse(firedAt))) return null;
  return {
    id: id,
    severity: severity,
    service: service,
    title: title,
    detail: typeof detail === "string" ? detail : "",
    fired_at: firedAt,
  };
}

function openAlertsDB() {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(ALERTS_STORE)) {
        db.createObjectStore(ALERTS_STORE, { keyPath: "id" });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

// appendAlert stores the record keyed by id (put, so a re-delivered alert
// updates rather than duplicates).
async function appendAlert(record) {
  const db = await openAlertsDB();
  try {
    await new Promise((resolve, reject) => {
      const tx = db.transaction(ALERTS_STORE, "readwrite");
      tx.objectStore(ALERTS_STORE).put(record);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
      tx.onabort = () => reject(tx.error);
    });
  } finally {
    db.close();
  }
}

async function showPushNotification(payload) {
  const critical = payload.severity === "critical";
  await self.registration.showNotification(payload.title, {
    body: payload.detail || payload.service,
    // Tag by severity: a newer alert of the same severity replaces the older
    // one instead of stacking notifications.
    tag: payload.severity,
    icon: "/icon-192.png",
    badge: "/icon-192.png",
    vibrate: VIBRATE[payload.severity],
    requireInteraction: critical, // critical alerts stay until dismissed
    data: payload,
  });
}

self.addEventListener("push", (event) => {
  let payload = null;
  try {
    payload = validateEnvelope(event.data.json());
  } catch (err) {
    payload = null; // malformed JSON body
  }
  if (payload === null) {
    // Malformed pushes are dropped, mirroring parsePayload's null contract.
    return;
  }
  const record = Object.assign({}, payload, {
    received_at: Date.now(),
    acked: false,
  });
  event.waitUntil(
    // History append must not block the notification, and an IndexedDB
    // failure must not lose the alert surface.
    appendAlert(record)
      .catch((err) => {
        console.error("pager: IndexedDB append failed", err);
      })
      .then(() => showPushNotification(payload))
      // Live in-app appearance (operator 2026-10-12): tell every open window
      // an alert landed so the mounted views re-fetch instead of waiting for
      // the next manual refresh.
      .then(() =>
        self.clients
          .matchAll({ type: "window", includeUncontrolled: true })
          .then((windowClients) => {
            for (const client of windowClients) client.postMessage({ type: "alerts-updated" });
          })
      )
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((windowClients) => {
      for (const client of windowClients) {
        if ("focus" in client) return client.focus();
      }
      return self.clients.openWindow("/");
    })
  );
});
