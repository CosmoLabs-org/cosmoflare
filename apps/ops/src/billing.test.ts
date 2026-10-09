// Cosmoflare Ops billing collector tests (FEAT-052): period fallback chain,
// day/days projection math, overage arithmetic, per-dataset failure
// isolation, project attribution, and the upstream-call budget — all against
// a mocked global fetch. No live calls here; the live check is a separate
// manual run.

import { describe, expect, it } from "vitest";
import { classifyR2, collectBilling, makePeriod, overageUsd, project, resolvePeriod } from "./billing";

// Day 8 of the 31-day calendar period October 2026 (00:00 UTC).
const OCT8 = new Date("2026-10-08T00:00:00Z");

const REST_OK = () => new Response(JSON.stringify({ success: true, result: [], result_info: { total_pages: 1 } }));
const SUB_403 = () => new Response(JSON.stringify({ success: false, errors: [{ message: "Authentication error" }] }), { status: 403 });
const SUB_OK = () =>
  new Response(
    JSON.stringify({
      success: true,
      result: [{ current_period: { start: "2026-09-15T00:00:00Z", end: "2026-10-15T00:00:00Z" } }],
      result_info: { total_pages: 1 },
    }),
  );

type GqlRow = { dimensions: Record<string, string>; sum?: Record<string, number>; max?: Record<string, number> };

// Full happy-path dataset: every flow dataset returns one row; the storage
// snapshot returns D1 bytes; REST name lists resolve the ids.
const DATASETS: Record<string, GqlRow[]> = {
  w: [{ dimensions: { scriptName: "churches-api" }, sum: { requests: 6_000_000, errors: 100 } }],
  d: [{ dimensions: { databaseId: "db1" }, sum: { rowsRead: 6_900_000_000, rowsWritten: 60_000_000, readQueries: 100, writeQueries: 10 } }],
  r: [
    { dimensions: { bucketName: "churches-bucket", actionType: "GetObject" }, sum: { requests: 12_000_000 } },
    { dimensions: { bucketName: "churches-bucket", actionType: "PutObject" }, sum: { requests: 500_000 } },
  ],
  k: [{ dimensions: { namespaceId: "ns1", actionType: "read" }, sum: { requests: 2_000_000 } }],
  o: [],
};

function billingGql(aliases: Record<string, GqlRow[] | undefined>): Response {
  const payload: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(aliases)) if (v) payload[k] = v;
  return new Response(JSON.stringify({ data: { viewer: { accounts: [payload] } } }));
}

// Install a fetch mock for the duration of a test. `gql` decides flow (all
// datasets merged) vs single-dataset queries by alias membership; `rest`
// routes by path fragment.
async function billingWithFetch(
  handler: (url: string, body: string) => Response,
  run: () => Promise<void>,
): Promise<void> {
  const real = globalThis.fetch;
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) =>
    handler(String(input), init?.body ? String(init.body) : "")) as typeof fetch;
  try {
    await run();
  } finally {
    globalThis.fetch = real;
  }
}

const isFlowBody = (body: string): boolean =>
  body.includes("workersInvocationsAdaptive") && body.includes("d1AnalyticsAdaptiveGroups");

// Route a happy-path mock: subscription 403 → calendar period, merged flow
// call succeeds, snapshot + name lists resolve.
function billingHappyHandler(options: {
  flowOk?: boolean;
  failWorkersSingle?: boolean;
  d1Names?: { uuid: string; name: string }[];
  kvNames?: { id: string; title: string }[];
  snap?: GqlRow[];
} = {}) {
  const { flowOk = true, failWorkersSingle = false, d1Names = [{ uuid: "db1", name: "mycarguide-db" }], kvNames = [{ id: "ns1", title: "cosmoflare-kv" }], snap = [{ dimensions: { databaseId: "db1" }, max: { databaseSizeBytes: 2_000_000_000 } }] } = options;
  return (url: string, body: string): Response => {
    if (url.includes("/subscriptions")) return SUB_403();
    if (url.includes("/graphql")) {
      const isFlow = isFlowBody(body);
      if (isFlow) return flowOk ? billingGql(DATASETS) : new Response(JSON.stringify({ errors: [{ message: "merged query rejected" }] }));
      if (body.includes("workersInvocationsAdaptive")) return failWorkersSingle ? gqlError("workers dataset down") : billingGql({ w: DATASETS.w });
      if (body.includes("d1AnalyticsAdaptiveGroups")) return billingGql({ d: DATASETS.d });
      if (body.includes("r2OperationsAdaptiveGroups")) return billingGql({ r: DATASETS.r });
      if (body.includes("kvOperationsAdaptiveGroups")) return billingGql({ k: DATASETS.k });
      if (body.includes("durableObjectsInvocationsAdaptiveGroups")) return billingGql({ o: DATASETS.o });
      if (body.includes("d1StorageAdaptiveGroups")) return billingGql({ s: snap });
      return new Response(JSON.stringify({ errors: [{ message: "unexpected query" }] }), { status: 500 });
    }
    if (url.includes("/d1/database")) return new Response(JSON.stringify({ success: true, result: d1Names, result_info: { total_pages: 1 } }));
    if (url.includes("/storage/kv/namespaces")) return billingKvNames(kvNames);
    return REST_OK();
  };
}

function gqlError(message: string): Response {
  return new Response(JSON.stringify({ errors: [{ message }] }));
}

function billingKvNames(ns: { id: string; title: string }[]): Response {
  return new Response(JSON.stringify({ success: true, result: ns, result_info: { total_pages: 1 } }));
}

// ---------------------------------------------------------------------------
// Period math

describe("makePeriod", () => {
  it("computes 1-based day and period length", () => {
    const p = makePeriod(new Date("2026-10-01T00:00:00Z"), new Date("2026-11-01T00:00:00Z"), "calendar", OCT8);
    expect(p.day).toBe(8);
    expect(p.days).toBe(31);
    expect(p.source).toBe("calendar");
  });
});

describe("resolvePeriod", () => {
  it("uses the subscription period when the endpoint answers with a covering period", async () => {
    await billingWithFetch(
      (url) => (url.includes("/subscriptions") ? SUB_OK() : SUB_403()),
      async () => {
        const p = await resolvePeriod("acct", "tok", OCT8);
        expect(p.source).toBe("subscription");
        expect(p.day).toBe(24); // Sep 15 → Oct 8 is 23 full days elapsed
        expect(p.days).toBe(30);
      },
    );
  });

  it("falls back to the anchor day when the subscription endpoint returns 403", async () => {
    await billingWithFetch(
      () => SUB_403(),
      async () => {
        const p = await resolvePeriod("acct", "tok", OCT8, { anchorDay: 15 });
        expect(p.source).toBe("anchor");
        expect(p.start).toBe("2026-09-15T00:00:00.000Z"); // anchor 15 not yet reached in October
        expect(p.end).toBe("2026-10-15T00:00:00.000Z");
        expect(p.day).toBe(24);
        expect(p.days).toBe(30);
      },
    );
  });

  it("falls back to the calendar month with no anchor day", async () => {
    await billingWithFetch(
      () => SUB_403(),
      async () => {
        const p = await resolvePeriod("acct", "tok", OCT8);
        expect(p.source).toBe("calendar");
        expect(p.day).toBe(8);
        expect(p.days).toBe(31);
      },
    );
  });
});

// ---------------------------------------------------------------------------
// Pure overage helpers

describe("project / overageUsd / classifyR2", () => {
  it("projects linearly and holds storage", () => {
    expect(project(100, 0.4, false)).toBe(250);
    expect(project(100, 0, false)).toBe(0);
    expect(project(50, 0.9, true)).toBe(50);
  });

  it("computes overage beyond the allowance in price units", () => {
    expect(overageUsd(26_737_500_000, 25_000_000_000, 0.001, 1_000_000)).toBeCloseTo(1.7375, 9);
    expect(overageUsd(10_000_000, 10_000_000, 0.3, 1_000_000)).toBe(0);
  });

  it("classifies R2 operations and drops unknown ones", () => {
    expect(classifyR2("GetObject")).toBe("B");
    expect(classifyR2("PutObject")).toBe("A");
    expect(classifyR2("DeleteObject")).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// collectBilling happy path

describe("collectBilling", () => {
  it("projects every product from the merged dataset and attributes overage to projects", async () => {
    await billingWithFetch(billingHappyHandler(), async () => {
      const b = await collectBilling("acct", "tok", OCT8);
      // Day 8 of 31 → fraction 8/31 → projected = used × 31/8
      expect(b.period).toMatchObject({ source: "calendar", day: 8, days: 31 });
      const ids = b.products.map((p) => p.id);
      expect(ids.slice(0, 5)).toEqual(["d1.rows_written", "r2.class_b", "r2.class_a", "workers.requests", "d1.rows_read"]);
      const written = b.products.find((p) => p.id === "d1.rows_written")!;
      expect(written.used).toBe(60_000_000);
      expect(written.projected).toBeCloseTo(232_500_000, 4);
      expect(written.projectedOverageUsd).toBeCloseTo(182.5, 6);
      const read = b.products.find((p) => p.id === "d1.rows_read")!;
      expect(read.projected).toBeCloseTo(26_737_500_000, 4); // 6.9B × 31/8
      expect(read.projectedOverageUsd).toBeCloseTo(1.7375, 6);
      const classB = b.products.find((p) => p.id === "r2.class_b")!;
      expect(classB.projectedOverageUsd).toBeCloseTo(13.14, 6);
      expect(b.totalProjectedOverageUsd).toBeCloseTo(205.57125, 4);
      // Storage converts bytes → GB and holds (no projection)
      const storage = b.products.find((p) => p.id === "d1.storage")!;
      expect(storage.used).toBe(2); // 2e9 bytes
      expect(storage.projected).toBe(2);
      expect(storage.topConsumers[0].name).toBe("all databases (1)");
      expect(storage.topConsumers[0].project).toBe("shared");
      expect(b.projects.map((p) => p.project)).toEqual(["mycarguide", "churches", "cosmoflare"]);
      expect(b.projects[0].projectedOverageUsd).toBeCloseTo(184.2375, 4); // 182.5 + 1.7375, share 1
      expect(b.projects[0].drivers).toContain("d1.rows_written");
      expect(b.projects[1].projectedOverageUsd).toBeCloseTo(21.33375, 4);
      expect(b.projects[2]).toMatchObject({ project: "cosmoflare", projectedOverageUsd: 0 });
      expect(b.upstreamCalls).toBe(5);
    });
  });

  it("keeps other products alive when one dataset fails, and records the error", async () => {
    await billingWithFetch(billingHappyHandler({ flowOk: false, failWorkersSingle: true }), async () => {
      const b = await collectBilling("acct", "tok", OCT8);
      expect(b.errors.some((e) => e.startsWith("w:"))).toBe(true);
      expect(b.products.find((p) => p.id === "workers.requests")!.used).toBe(0);
      expect(b.products.find((p) => p.id === "d1.rows_written")!.projectedOverageUsd).toBeCloseTo(182.5, 6);
      expect(b.products.find((p) => p.id === "r2.class_b")!.used).toBe(12_000_000);
    });
  });

  it("honors the projectMap override for attribution", async () => {
    await billingWithFetch(billingHappyHandler(), async () => {
      const b = await collectBilling("acct", "tok", OCT8, { projectMap: { "mycarguide-db": "acme" } });
      expect(b.projects.map((p) => p.project)).toEqual(["acme", "churches", "cosmoflare"]);
    });
  });

  it("creates no project row from an empty consumer name", async () => {
    await billingWithFetch(
      (url, body) => {
        if (url.includes("/subscriptions")) return SUB_403();
        if (url.includes("/graphql")) {
          if (isFlowBody(body)) return billingGql({ w: [{ dimensions: { scriptName: "" }, sum: { requests: 10 } }] });
          if (body.includes("workersInvocationsAdaptive")) return billingGql({ w: [{ dimensions: { scriptName: "" }, sum: { requests: 10 } }] });
          return billingGql({});
        }
        return REST_OK();
      },
      async () => {
        const b = await collectBilling("acct", "tok", OCT8);
        expect(b.projects.map((p) => p.project)).not.toContain("");
      },
    );
  });

  it("records the verified-unavailable metrics as gap notes, not fabricated numbers", async () => {
    await billingWithFetch(billingHappyHandler(), async () => {
      const b = await collectBilling("acct", "tok", OCT8);
      for (const id of ["workers.cpu_ms", "do.duration", "r2.storage", "kv.storage"]) {
        expect(b.errors.some((e) => e.startsWith(`${id}:`)), id).toBe(true);
        expect(b.products.find((p) => p.id === id)!.used).toBe(0);
      }
    });
  });

  it("uses the subscription period when available end to end", async () => {
    await billingWithFetch(
      (url, body) => {
        if (url.includes("/subscriptions")) return SUB_OK();
        return billingHappyHandler()(url, body);
      },
      async () => {
        const b = await collectBilling("acct", "tok", OCT8);
        expect(b.period.source).toBe("subscription");
        expect(b.period.days).toBe(30);
      },
    );
  });
});
