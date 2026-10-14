// @vitest-environment jsdom
// WorkerProfileView (P-05a): the React profile reuses the statRow visual
// rules (cf-stat-value, zero-errors green, error-rate tint), the back
// control, the dashboard link, and the not-found quiet state — with the
// summary fetch injected (the worker_profile.test.ts pattern).
// The live-tail describe (2026-10-14) covers the bounded NDJSON tail UI:
// events render from a gated mock stream, a stream {error} line renders the
// note, the end line auto-stops cleanly, and the Stop control aborts.
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { LoginExpiredError, type FetchResult, type Summary, type WorkerProfilePayload, type WorkerRow } from "../../api";
import WorkerProfileView from "./WorkerProfileView";

afterEach(cleanup);

const ROW: WorkerRow = {
  script: "api-gateway",
  requests: 52_332_502,
  errors: 1234,
  errorPct: 5.5,
  cpuP50Ms: 3.42,
  cpuP99Ms: null,
};

const SUMMARY = (workers: WorkerRow[]): Summary => ({
  generatedAt: "2026-10-10T00:00:00Z",
  windowHours: 24,
  usage: [],
  d1: [],
  zones: [],
  workers,
  errors: [],
});

function fetcherReturning(summary: Summary, opts: { throwErr?: Error } = {}): Parameters<typeof WorkerProfileView>[0]["fetcher"] {
  return async <T,>(_endpoint: string): Promise<FetchResult<T>> => {
    if (opts.throwErr) throw opts.throwErr;
    return { data: summary as unknown as T, source: "network", ageSec: 0, demo: false };
  };
}

/** Skeleton probe: the fetch hangs until go() releases it. */
function gatedSummaryFetcher(summary: Summary): { fetcher: Parameters<typeof WorkerProfileView>[0]["fetcher"]; go: () => void } {
  let release!: (r: FetchResult<Summary>) => void;
  const gate = new Promise<FetchResult<Summary>>((res) => {
    release = res;
  });
  const fetcher: Parameters<typeof WorkerProfileView>[0]["fetcher"] = async <T,>(): Promise<FetchResult<T>> =>
    gate as unknown as Promise<FetchResult<T>>;
  return {
    fetcher,
    go: () => release({ data: summary, source: "network", ageSec: 0, demo: false }),
  };
}

/** Endpoint-aware fetcher: maps "api/summary" and "api/worker-profile?…"
 *  to injected payloads (the multi-fetch views' injection pattern). */
function multiFetcher(endpoints: Record<string, unknown>): Parameters<typeof WorkerProfileView>[0]["fetcher"] {
  return async <T,>(endpoint: string): Promise<FetchResult<T>> => {
    for (const [prefix, body] of Object.entries(endpoints)) {
      if (endpoint === prefix || endpoint.startsWith(`${prefix}?`)) {
        return { data: body as T, source: "network", ageSec: 0, demo: false };
      }
    }
    throw new Error(`unexpected endpoint ${endpoint}`);
  };
}

// ---- live-tail stream mocks ----
// A gated NDJSON stream: the test holds the controller and pushes lines, so
// assertions run against exact render timing (no timers, no fake clocks).
interface TailStream {
  push: (line: string) => void;
  close: () => void;
}

function gatedTailStreamFetch(): { tailFetch: typeof fetch; stream: TailStream; signal: () => AbortSignal | null } {
  let controller!: ReadableStreamDefaultController<Uint8Array>;
  let captured: AbortSignal | null = null;
  const body = new ReadableStream<Uint8Array>({
    start(c) {
      controller = c;
    },
  });
  const tailFetch: typeof fetch = async (_input, init) => {
    captured = init?.signal ?? null;
    return { ok: true, status: 200, body } as unknown as Response;
  };
  return {
    tailFetch,
    stream: {
      push: (line) => controller.enqueue(new TextEncoder().encode(`${line}\n`)),
      close: () => controller.close(),
    },
    signal: () => captured,
  };
}

/** The view's tail button by its exact label. */
function tailButton(label: string): HTMLButtonElement | null {
  return Array.from(document.querySelectorAll("button")).find((b) => b.textContent === label) ?? null;
}

const PROFILE: WorkerProfilePayload = {
  generatedAt: "2026-10-13T00:00:00.000Z",
  bindings: [{ type: "kv_namespace", name: "CACHE" }, { type: "plain_text", name: "FLAG" }],
  cronTriggers: ["*/5 * * * *"],
  routes: [{ zone: "cosmolabs.org", pattern: "cosmolabs.org/api*" }],
  versions: [{ number: 7, createdOn: "2026-10-05T00:00:00Z" }, { number: 6, createdOn: null }],
  errors: [],
};

describe("WorkerProfileView live tail", () => {
  it("renders events from a mocked NDJSON stream with a running count", async () => {
    const tail = gatedTailStreamFetch();
    render(
      <WorkerProfileView
        name="api-gateway"
        fetcher={multiFetcher({ "api/summary": SUMMARY([ROW]), "api/worker-profile": PROFILE })}
        tailFetch={tail.tailFetch}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    fireEvent.click(tailButton("Start tail") as HTMLButtonElement);
    tail.stream.push('{"type":"open"}');
    tail.stream.push('{"type":"event","event":{"outcome":"ok","exceptions":[],"eventTimestamp":"2026-10-14T00:00:01.000Z"}}');
    tail.stream.push('{"type":"event","event":{"outcome":"exceededCpu","exceptions":[{"message":"CPU time limit exceeded"}],"eventTimestamp":"2026-10-14T00:00:02.000Z"}}');
    await waitFor(() => {
      expect(document.querySelector(".cf-tail-count")?.textContent).toContain("2 events");
    });
    // Two compact rows: outcome + exceptions count, UTC time on the label side.
    const rows = Array.from(document.querySelectorAll(".cf-domain-field")).filter((r) =>
      r.textContent?.includes("exceededCpu") || r.textContent?.includes("00:00:01"),
    );
    expect(rows.length).toBe(2);
    const labels = rows.map((r) => r.querySelector(".cf-domain-field-label")?.textContent);
    expect(labels).toEqual(["00:00:01", "00:00:02"]);
    // Exceptions render as a count on the compact row.
    expect(rows[1].textContent).toContain("1 exception");
    // Streaming state shows the Stop control.
    expect(tailButton("Stop tail")).toBeTruthy();
  });

  it("renders a stream {error} line as the error note and stops", async () => {
    const tail = gatedTailStreamFetch();
    render(
      <WorkerProfileView
        name="api-gateway"
        fetcher={multiFetcher({ "api/summary": SUMMARY([ROW]), "api/worker-profile": PROFILE })}
        tailFetch={tail.tailFetch}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    fireEvent.click(tailButton("Start tail") as HTMLButtonElement);
    tail.stream.push('{"error":"tail unavailable: authentication failed"}');
    await waitFor(() => {
      expect(document.querySelector(".cf-tail-error")?.textContent).toContain("tail unavailable: authentication failed");
    });
    // The stream stopped: Start is offered again.
    expect(tailButton("Start tail")).toBeTruthy();
  });

  it("auto-stops cleanly at the end line (no hang, Start offered again)", async () => {
    const tail = gatedTailStreamFetch();
    render(
      <WorkerProfileView
        name="api-gateway"
        fetcher={multiFetcher({ "api/summary": SUMMARY([ROW]), "api/worker-profile": PROFILE })}
        tailFetch={tail.tailFetch}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    fireEvent.click(tailButton("Start tail") as HTMLButtonElement);
    tail.stream.push('{"type":"open"}');
    tail.stream.push('{"type":"end","reason":"duration-cutoff"}');
    tail.stream.close();
    await waitFor(() => {
      expect(document.querySelector(".cf-tail-note")?.textContent).toContain("duration-cutoff");
      expect(tailButton("Start tail")).toBeTruthy();
    });
    expect(tailButton("Stop tail")).toBeNull();
  });

  it("the Stop control aborts the stream and offers Start again", async () => {
    const tail = gatedTailStreamFetch();
    render(
      <WorkerProfileView
        name="api-gateway"
        fetcher={multiFetcher({ "api/summary": SUMMARY([ROW]), "api/worker-profile": PROFILE })}
        tailFetch={tail.tailFetch}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    fireEvent.click(tailButton("Start tail") as HTMLButtonElement);
    await waitFor(() => {
      expect(tailButton("Stop tail")).toBeTruthy();
    });
    fireEvent.click(tailButton("Stop tail") as HTMLButtonElement);
    // The fetch's abort signal fired; the UI is back to the idle affordance.
    expect(tail.signal()?.aborted).toBe(true);
    expect(tailButton("Start tail")).toBeTruthy();
    expect(document.querySelector(".cf-tail-note")?.textContent).toContain("Stopped");
  });
});

describe("WorkerProfileView worker-profile expansion", () => {
  it("renders the four configuration sections with a binding row and a version row", async () => {
    render(
      <WorkerProfileView
        name="api-gateway"
        fetcher={multiFetcher({ "api/summary": SUMMARY([ROW]), "api/worker-profile": PROFILE })}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    for (const title of ["Bindings", "Cron triggers", "Routes", "Versions"]) {
      const h2 = Array.from(document.querySelectorAll(".cf-card h2")).find((el) => el.textContent === title);
      expect(h2, title).toBeTruthy();
    }
    // A binding row: name left, type in the data face.
    const bindingLabel = Array.from(document.querySelectorAll(".cf-domain-field-label")).find((el) => el.textContent === "CACHE");
    expect(bindingLabel).toBeTruthy();
    // A version row: "v7" with its display-only date.
    const versionLabel = Array.from(document.querySelectorAll(".cf-domain-field-label")).find((el) => el.textContent === "v7");
    expect(versionLabel?.parentElement?.textContent).toContain("2026-10-05");
  });

  it("renders no Bindings card when the payload carries no bindings", async () => {
    render(
      <WorkerProfileView
        name="api-gateway"
        fetcher={multiFetcher({ "api/summary": SUMMARY([ROW]), "api/worker-profile": { ...PROFILE, bindings: [] } })}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    const headings = Array.from(document.querySelectorAll(".cf-card h2")).map((el) => el.textContent);
    expect(headings).not.toContain("Bindings");
    // Sections with data still render.
    expect(headings).toContain("Versions");
  });

  it("a failed side fetch leaves the analytics card untouched (no section cards)", async () => {
    const fetcher: Parameters<typeof WorkerProfileView>[0]["fetcher"] = async <T,>(endpoint: string): Promise<FetchResult<T>> => {
      if (endpoint === "api/summary") {
        return { data: SUMMARY([ROW]) as unknown as T, source: "network", ageSec: 0, demo: false };
      }
      throw new Error("HTTP 500");
    };
    render(<WorkerProfileView name="api-gateway" fetcher={fetcher} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-stat-value")).toBeTruthy();
    });
    const headings = Array.from(document.querySelectorAll(".cf-card h2")).map((el) => el.textContent);
    for (const title of ["Bindings", "Cron triggers", "Routes", "Versions"]) {
      expect(headings).not.toContain(title);
    }
  });
});

describe("WorkerProfileView (P-05a)", () => {
  it("renders the 24h stats with the statRow visual rules", async () => {
    render(<WorkerProfileView name="api-gateway" fetcher={fetcherReturning(SUMMARY([ROW]))} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    // Back control to the Workers section.
    const back = document.querySelector<HTMLAnchorElement>(".cf-profile-back");
    expect(back?.getAttribute("href")).toBe("#/workers");
    expect(back?.textContent).toBe("All workers");
    // Stats: the same numbers the table shows, with cf-stat-value faces.
    const values = Array.from(document.querySelectorAll(".cf-stat-value")).map((v) => v.textContent);
    expect(values).toEqual(["52.3M", "1.2k", "5.50%", "3.4ms", "—"]);
    // Zero-errors green reads on the Errors value; a 5.5% rate is critical.
    const errorValue = document.querySelectorAll(".cf-stat-value")[1];
    expect(errorValue.classList.contains("cf-level-critical-text")).toBe(true);
    const rateValue = document.querySelectorAll(".cf-stat-value")[2];
    expect(rateValue.classList.contains("cf-level-critical-text")).toBe(true);
    // Dashboard deep-link out.
    const dash = document.querySelector<HTMLAnchorElement>(".cf-link");
    expect(dash?.getAttribute("href")).toBe("https://dash.cloudflare.com/?to=/:account/workers/services/view/api-gateway");
    expect(dash?.getAttribute("target")).toBe("_blank");
    expect(dash?.getAttribute("rel")).toBe("noreferrer");
  });

  it("reads zero errors as green (cf-ok-text)", async () => {
    const clean: WorkerRow = { ...ROW, script: "cdn", errors: 0, errorPct: 0 };
    render(<WorkerProfileView name="cdn" fetcher={fetcherReturning(SUMMARY([clean]))} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-stat-value")).toBeTruthy();
    });
    const errorValue = document.querySelectorAll(".cf-stat-value")[1];
    expect(errorValue.classList.contains("cf-ok-text")).toBe(true);
  });

  it("shows the quiet not-found state — never an error", async () => {
    render(<WorkerProfileView name="ghost" fetcher={fetcherReturning(SUMMARY([ROW]))} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card h2")?.textContent).toBe("Worker not found");
    expect(document.querySelector(".cf-empty-quiet")?.textContent).toBe("No 24h analytics row for ghost.");
  });

  it("shows the login-expired message on LoginExpiredError", async () => {
    render(
      <WorkerProfileView
        name="api-gateway"
        fetcher={fetcherReturning(SUMMARY([]), { throwErr: new LoginExpiredError() })}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-login-expired")).toBeTruthy();
    });
    expect(document.querySelector(".cf-login-expired")?.textContent).toBe(
      "your login expired — close and reopen the app to sign in again",
    );
  });

  it("offers a Retry card after a first-load failure", async () => {
    render(
      <WorkerProfileView
        name="api-gateway"
        fetcher={fetcherReturning(SUMMARY([ROW]), { throwErr: new Error("HTTP 503") })}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-card.cf-level-warning")).toBeTruthy();
    });
    expect(document.querySelector(".cf-row-detail")?.textContent).toBe("HTTP 503");
    expect(document.querySelector(".cf-card .cf-btn")?.textContent).toBe("Retry");
  });

  it("renders the skeleton while the first fetch is in flight", async () => {
    const gated = gatedSummaryFetcher(SUMMARY([ROW]));
    render(<WorkerProfileView name="api-gateway" fetcher={gated.fetcher} />);
    expect(document.querySelector(".cf-loading")).toBeTruthy();
    expect(document.querySelectorAll(".cf-skeleton").length).toBe(3);
    expect(document.querySelector(".cf-profile")).toBeNull();
    gated.go();
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    expect(document.querySelector(".cf-loading")).toBeNull();
  });
});
