import { beforeEach, describe, expect, it } from "vitest";
import "fake-indexeddb/auto";
import { IDBFactory } from "fake-indexeddb";
import {
  acknowledge,
  getAlert,
  listAlerts,
  MAX_ALERTS,
  putAlert,
  snooze,
  type AlertRecord,
} from "./store";

// Fresh empty database per test — fake-indexeddb persists within a process.
beforeEach(() => {
  indexedDB = new IDBFactory();
});

let seq = 0;
// makeRecord builds a valid AlertRecord with a strictly increasing
// received_at so insertion order is observable.
function makeRecord(overrides: Partial<AlertRecord> = {}): AlertRecord {
  seq += 1;
  return {
    id: `alert-${seq}`,
    severity: "info",
    service: "api-gateway",
    title: `Alert ${seq}`,
    detail: "",
    fired_at: "2026-10-07T13:00:00Z",
    received_at: seq,
    acked: false,
    ...overrides,
  };
}

describe("listAlerts", () => {
  it("returns records newest-first", async () => {
    const old = makeRecord();
    const mid = makeRecord();
    const fresh = makeRecord();
    for (const record of [old, mid, fresh]) {
      await putAlert(record);
    }
    expect((await listAlerts()).map((r) => r.id)).toEqual([fresh.id, mid.id, old.id]);
  });

  it("hides records snoozed into the future and shows them once the snooze passes", async () => {
    const record = makeRecord();
    await putAlert(record);
    await putAlert({ ...record, snoozed_until: Date.now() + 60_000 });
    expect(await listAlerts()).toEqual([]);

    await putAlert({ ...record, snoozed_until: Date.now() - 1 });
    expect(await listAlerts()).toHaveLength(1);
  });

  it("returns an empty list on an empty store", async () => {
    expect(await listAlerts()).toEqual([]);
  });
});

describe("acknowledge", () => {
  it("sets the acked flag on one record without touching the others", async () => {
    const first = makeRecord();
    const second = makeRecord();
    const third = makeRecord();
    for (const record of [first, second, third]) {
      await putAlert(record);
    }
    expect((await getAlert(second.id))?.acked).toBe(false);

    await acknowledge(second.id);

    expect((await getAlert(second.id))?.acked).toBe(true);
    expect((await getAlert(first.id))?.acked).toBe(false);
    expect((await getAlert(third.id))?.acked).toBe(false);
  });

  it("is a no-op for an unknown id", async () => {
    await acknowledge("missing");
    expect(await listAlerts()).toEqual([]);
  });
});

describe("snooze", () => {
  it("sets snoozed_until and filters the record from listAlerts", async () => {
    const record = makeRecord();
    await putAlert(record);

    await snooze(record.id, 1);

    const stored = await getAlert(record.id);
    expect(stored?.snoozed_until).toBeGreaterThan(Date.now());
    expect(await listAlerts()).toEqual([]);
  });

  it("keeps other records visible while one is snoozed", async () => {
    const first = makeRecord();
    const second = makeRecord();
    await putAlert(first);
    await putAlert(second);

    await snooze(first.id, 1);

    const visible = await listAlerts();
    expect(visible.map((r) => r.id)).toEqual([second.id]);
  });
});

describe("FIFO cap", () => {
  it("keeps at most MAX_ALERTS records, evicting the oldest first", async () => {
    const total = MAX_ALERTS + 2;
    for (let i = 0; i < total; i++) {
      await putAlert(makeRecord({ received_at: i }));
    }

    const alerts = await listAlerts();
    expect(alerts).toHaveLength(MAX_ALERTS);
    // Newest first: the two oldest (received_at 0 and 1) were evicted.
    expect(alerts[0].received_at).toBe(total - 1);
    expect(alerts[MAX_ALERTS - 1].received_at).toBe(2);
  });
});
