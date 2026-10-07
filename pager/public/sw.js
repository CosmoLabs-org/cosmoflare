// Cosmoflare Pager service worker (FEAT-045).
//
// push event -> validate the envelope inline (mirrors src/payload.ts; a
// service worker here stays import-free) -> append to IndexedDB
// (db "cosmoflare-pager", store "alerts", keyPath "id") ->
// self.registration.showNotification tagged by severity.
// notificationclick -> focus an existing window or open one.
//
// Payload protocol: pkg/cosmoflare/alertspush/payload.go. No credentials ever
// appear in a payload.

const DB_NAME = "cosmoflare-pager";
const DB_VERSION = 1;
const ALERTS_STORE = "alerts";
const SEVERITIES = ["info", "warning", "critical"];

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
