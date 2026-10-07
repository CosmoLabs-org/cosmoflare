// IndexedDB history store for Cosmoflare Pager (FEAT-045).
//
// The service worker (public/sw.js) appends push records into the shared
// database (db "cosmoflare-pager", store "alerts", keyPath "id") with the
// shape { ...payload, received_at, acked: false }. This module is the
// read/write wrapper the views use: newest-first listing with snoozed
// records filtered out, acknowledge/snooze updates, and the 500-record
// FIFO cap enforced on every write.
//
// The DB name, version and store name here MUST stay in lockstep with
// public/sw.js — bump both files together.

import type { Payload } from "./payload";

export const DB_NAME = "cosmoflare-pager";
export const DB_VERSION = 1;
export const ALERTS_STORE = "alerts";

/** History cap: when a write pushes the store past this size, the oldest records are evicted first (FIFO). */
export const MAX_ALERTS = 500;

/** A stored alert: the wire Payload plus pager-side bookkeeping fields. */
export type AlertRecord = Payload & {
  /** Epoch ms when the record was appended by the service worker. */
  received_at: number;
  /** Set by acknowledge(). */
  acked: boolean;
  /** Epoch ms until which the record is hidden from listAlerts (set by snooze()). Absent = not snoozed. */
  snoozed_until?: number;
};

function openAlertsDB(): Promise<IDBDatabase> {
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

// putAlert stores a record (keyed by id, so a re-delivered alert updates
// rather than duplicates) and enforces the FIFO cap: after the put, records
// beyond MAX_ALERTS are evicted oldest-first. getAll + an in-memory sort is
// plenty at pager history scale (<= MAX_ALERTS + 1 records per write).
export async function putAlert(record: AlertRecord): Promise<void> {
  const db = await openAlertsDB();
  try {
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(ALERTS_STORE, "readwrite");
      const store = tx.objectStore(ALERTS_STORE);
      store.put(record);
      const allReq = store.getAll();
      allReq.onsuccess = () => {
        const all = allReq.result as AlertRecord[];
        // Ascending by received_at = oldest first; everything before the
        // newest MAX_ALERTS records is stale.
        all.sort((a, b) => a.received_at - b.received_at);
        const stale = all.slice(0, all.length - MAX_ALERTS);
        for (const old of stale) {
          store.delete(old.id);
        }
      };
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
      tx.onabort = () => reject(tx.error);
    });
  } finally {
    db.close();
  }
}

// listAlerts returns the visible history newest-first: records snoozed into
// the future (snoozed_until > now) are filtered out; everything else sorts by
// received_at descending.
export async function listAlerts(): Promise<AlertRecord[]> {
  const db = await openAlertsDB();
  try {
    return await new Promise<AlertRecord[]>((resolve, reject) => {
      const tx = db.transaction(ALERTS_STORE, "readonly");
      const req = tx.objectStore(ALERTS_STORE).getAll();
      tx.oncomplete = () => {
        const now = Date.now();
        const visible = (req.result as AlertRecord[]).filter(
          (record) => record.snoozed_until === undefined || record.snoozed_until <= now,
        );
        visible.sort((a, b) => b.received_at - a.received_at);
        resolve(visible);
      };
      tx.onerror = () => reject(tx.error);
      tx.onabort = () => reject(tx.error);
    });
  } finally {
    db.close();
  }
}

/** getAlert fetches a single record by id (undefined when unknown). */
export async function getAlert(id: string): Promise<AlertRecord | undefined> {
  const db = await openAlertsDB();
  try {
    return await new Promise<AlertRecord | undefined>((resolve, reject) => {
      const tx = db.transaction(ALERTS_STORE, "readonly");
      const req = tx.objectStore(ALERTS_STORE).get(id);
      tx.oncomplete = () => resolve(req.result as AlertRecord | undefined);
      tx.onerror = () => reject(tx.error);
      tx.onabort = () => reject(tx.error);
    });
  } finally {
    db.close();
  }
}

// updateAlert applies update() to the stored record in one readwrite
// transaction; unknown ids resolve as a no-op.
async function updateAlert(
  id: string,
  update: (record: AlertRecord) => AlertRecord,
): Promise<void> {
  const db = await openAlertsDB();
  try {
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(ALERTS_STORE, "readwrite");
      const store = tx.objectStore(ALERTS_STORE);
      const getReq = store.get(id);
      getReq.onsuccess = () => {
        const existing = getReq.result as AlertRecord | undefined;
        if (existing !== undefined) {
          store.put(update(existing));
        }
      };
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
      tx.onabort = () => reject(tx.error);
    });
  } finally {
    db.close();
  }
}

/** acknowledge marks the alert as acknowledged (acked flag set). */
export function acknowledge(id: string): Promise<void> {
  return updateAlert(id, (record) => ({ ...record, acked: true }));
}

/** snooze hides the alert from listAlerts until the given minutes have passed. */
export function snooze(id: string, minutes: number): Promise<void> {
  return updateAlert(id, (record) => ({
    ...record,
    snoozed_until: Date.now() + minutes * 60_000,
  }));
}
