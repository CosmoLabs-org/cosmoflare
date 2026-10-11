// @vitest-environment jsdom
// DomainsView (P-05b): sorted rows with expiry attention coloring and the
// "external" handling, the in-component profile drill-down (tap a row →
// Registration/DNS/Security/Renewal sections; "All domains" returns), and
// the /api/domains/detail enrichment (the historical "Previous nameservers
// (before Cloudflare)" label) — with every fetch injected
// (the worker_profile.test.ts pattern).
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api, type FetchResult } from "../../api";
import type { AttachmentsPayload, AuditEntry, AuditPayload, DomainDetailPayload, DomainRecord, DomainsPayload } from "../../domains";
import DomainsView from "./DomainsView";

afterEach(cleanup);

const DAY = 86_400_000;
const iso = (daysFromNow: number): string => new Date(Date.now() + daysFromNow * DAY).toISOString();

function domain(partial: Partial<DomainRecord> & { id: string; name: string }): DomainRecord {
  return {
    status: "active",
    paused: false,
    expiresAt: null,
    plan: "Pro",
    type: "full",
    createdOn: "2024-01-15T00:00:00Z",
    modifiedOn: "2026-09-01T00:00:00Z",
    nameServers: ["ada.ns.cloudflare.com", "bob.ns.cloudflare.com"],
    ...partial,
  };
}

const DOMAINS: DomainsPayload = {
  generatedAt: "2026-10-10T00:00:00Z",
  domains: [
    domain({ id: "z3", name: "later.org", expiresAt: iso(90) }),
    domain({ id: "z1", name: "soon.dev", expiresAt: iso(10) }),
    domain({ id: "z4", name: "outside.example", expiresAt: null }),
    domain({ id: "z2", name: "warn.net", expiresAt: iso(20), status: "pending" }),
  ],
  errors: [],
};

const DETAIL: DomainDetailPayload = {
  zone: {
    ...domain({ id: "z1", name: "soon.dev", expiresAt: iso(10) }),
    originalNameServers: ["ns1.dreamhost.com", "ns2.dreamhost.com"],
    activatedOn: "2024-01-16T00:00:00Z",
    ownerType: "organization",
    dnssecStatus: "active",
    sslMode: "strict",
  },
  errors: [],
};

const ATTACHMENTS: AttachmentsPayload = {
  generatedAt: "2026-10-10T00:00:00Z",
  pages: [
    {
      project: "noble-web",
      domains: [
        { hostname: "staging.noble.coffee", status: "active", ssl: "active", verification: "active" },
        { hostname: "noble.coffee", status: "pending", ssl: null, verification: null },
      ],
    },
  ],
  workers: [{ hostname: "api.noble.coffee", service: "noble-api", environment: "production" }],
  zones: [],
  errors: [],
};

/** Stub fetcher: answers api/domains and the detail/summary endpoints; the
 *  summary deliberately fails to prove enrichment is soft. */
function domainsFetcher(opts: { detail?: DomainDetailPayload; failFirstLoad?: Error; attachments?: AttachmentsPayload } = {}): Parameters<typeof DomainsView>[0]["fetcher"] {
  return async <T,>(endpoint: string): Promise<FetchResult<T>> => {
    if (opts.failFirstLoad) throw opts.failFirstLoad;
    if (endpoint === "api/domains") return { data: DOMAINS as unknown as T, source: "network", ageSec: 0, demo: false };
    if (endpoint === "api/attachments") {
      if (opts.attachments) return { data: opts.attachments as unknown as T, source: "network", ageSec: 0, demo: false };
      throw new Error("HTTP 503");
    }
    if (endpoint.startsWith("api/domains/detail")) {
      if (opts.detail) return { data: opts.detail as unknown as T, source: "network", ageSec: 0, demo: false };
      throw new Error("HTTP 503");
    }
    throw new Error("HTTP 404");
  };
}

const text = (sel: string): string[] => Array.from(document.querySelectorAll(sel)).map((n) => n.textContent);

describe("DomainsView (P-05b)", () => {
  it("renders sorted rows with expiry attention coloring, external last", async () => {
    render(<DomainsView fetcher={domainsFetcher()} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    // sortDomains: soonest expiry first, no-expiry domains last by name.
    const names = text(".cf-domain-name");
    expect(names).toEqual(["soon.dev", "warn.net", "later.org", "outside.example"]);
    // Expiry levels: ≤14d critical, ≤30d warning, else ok, null carries none.
    const expiry = document.querySelectorAll(".cf-domain-expiry");
    expect(expiry[0].classList.contains("cf-level-critical-text")).toBe(true);
    expect(expiry[0].textContent).toMatch(/^\d+d · /);
    expect(expiry[1].classList.contains("cf-level-warning-text")).toBe(true);
    expect(expiry[2].classList.contains("cf-level-ok-text")).toBe(true);
    expect(expiry[3].classList.contains("cf-level-critical-text")).toBe(false);
    expect(expiry[3].textContent).toBe("external");
    // The pending (non-active) zone carries its muted badge.
    expect(text(".cf-domain-badges .cf-badge")).toEqual(["pending"]);
    // Subtitle carries the expiring count only; the domain total lives in the
    // masthead heading.
    expect(document.querySelector(".cf-row-detail")?.textContent).toBe("2 expiring within 30 days");
    expect(document.querySelector(".cf-card h2")).toBeNull();
  });

  it("renders no subtitle when no domains expire within 30 days", async () => {
    const quiet: DomainsPayload = {
      ...DOMAINS,
      domains: DOMAINS.domains.map((d) => domain({ id: d.id, name: d.name, expiresAt: iso(90) })),
    };
    const fetcher: Parameters<typeof DomainsView>[0]["fetcher"] = async <T,>(endpoint: string): Promise<FetchResult<T>> => {
      if (endpoint === "api/domains") return { data: quiet as unknown as T, source: "network", ageSec: 0, demo: false };
      throw new Error("HTTP 404");
    };
    render(<DomainsView fetcher={fetcher} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    expect(document.querySelector(".cf-row-detail")).toBeNull();
  });

  it("drills into the profile on row tap and returns on 'All domains'", async () => {
    render(<DomainsView fetcher={domainsFetcher()} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    const rows = document.querySelectorAll<HTMLButtonElement>(".cf-domain-row");
    rows[1].click(); // warn.net
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    expect(document.querySelector(".cf-profile-title")?.textContent).toBe("warn.net");
    expect(document.querySelector(".cf-profile-eyebrow")?.textContent).toBe("Domain profile");
    // The four sections, in the vanilla order.
    const sections = text(".cf-profile-section-title");
    expect(sections).toEqual(["Renewal", "Registration", "DNS", "Security"]);
    // Back control returns to the list.
    const back = document.querySelector<HTMLButtonElement>(".cf-profile-back");
    expect(back?.textContent).toBe("All domains");
    back!.click();
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    expect(document.querySelector(".cf-profile")).toBeNull();
  });

  it("enriches the profile from api/domains/detail — failures leave the base profile", async () => {
    render(<DomainsView fetcher={domainsFetcher({ detail: DETAIL })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    document.querySelectorAll<HTMLButtonElement>(".cf-domain-row")[0].click(); // soon.dev
    await waitFor(() => {
      expect(document.querySelector(".cf-profile-title")?.textContent).toBe("soon.dev");
    });
    // The historical label, verbatim.
    await waitFor(() => {
      expect(text(".cf-domain-field-label")).toContain("Previous nameservers (before Cloudflare)");
    });
    const labels = text(".cf-domain-field-label");
    const values = text(".cf-domain-field-value");
    const idx = labels.indexOf("Previous nameservers (before Cloudflare)");
    expect(values[idx]).toBe("ns1.dreamhost.com\nns2.dreamhost.com");
    expect(labels).toContain("DNSSEC");
    expect(labels).toContain("SSL mode");
    expect(labels).toContain("Activated");
    expect(labels).toContain("Owner");
    // Traffic stays absent — the summary fetch failed (soft join).
    expect(labels).not.toContain("Traffic (24h)");
  });

  it("keeps the base profile when the detail fetch fails", async () => {
    render(<DomainsView fetcher={domainsFetcher()} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    document.querySelectorAll<HTMLButtonElement>(".cf-domain-row")[0].click();
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    // Let any in-flight enrichment settle.
    await new Promise((r) => setTimeout(r, 20));
    expect(text(".cf-domain-field-label")).not.toContain("Previous nameservers (before Cloudflare)");
    // The base sections still paint.
    expect(text(".cf-profile-section-title")).toEqual(["Renewal", "Registration", "DNS", "Security"]);
  });

  it("paints the quiet empty state when no domains exist", async () => {
    const empty: DomainsPayload = { generatedAt: "2026-10-10T00:00:00Z", domains: [], errors: [] };
    const fetcher: Parameters<typeof DomainsView>[0]["fetcher"] = async <T,>(endpoint: string) => {
      if (endpoint === "api/domains") return { data: empty as unknown as T, source: "network", ageSec: 0, demo: false };
      throw new Error("HTTP 404");
    };
    render(<DomainsView fetcher={fetcher} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-empty-quiet")).toBeTruthy();
    });
    expect(document.querySelector(".cf-empty-quiet")?.textContent).toBe("No domains found on this account.");
  });

  it("does not loop fetches when no fetcher prop is injected (production path)", async () => {
    // A fresh FetchResult per call, as production returns — a stable reference
    // would let React bail out of the re-render and mask the loop.
    const spy = vi.spyOn(api, "fetchJson").mockImplementation(async () =>
      ({ data: DOMAINS, source: "network", ageSec: 0, demo: false }) as unknown as Awaited<ReturnType<typeof api.fetchJson>>,
    );
    render(<DomainsView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-card, .cf-empty, .cf-loading")).toBeTruthy();
    });
    const callCount = spy.mock.calls.length;
    await new Promise((r) => setTimeout(r, 300));
    expect(spy.mock.calls.length).toBeLessThanOrEqual(callCount + 1);
  });

  // Attachment inventory (FEAT-pEWA6N6 phase 1): the section under the zones
  // card, fed by its own api/attachments fetch.
  it("renders the attachments section from the stubbed api/attachments payload", async () => {
    render(<DomainsView fetcher={domainsFetcher({ attachments: ATTACHMENTS })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-attachment-row")).toBeTruthy();
    });
    // The zone card still renders above it.
    expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    // Subtitle, verbatim.
    expect(document.querySelector(".cf-attachments .cf-row-detail")?.textContent).toBe("Custom domains by project");
    // One group per Pages project, then the Workers group.
    expect(text(".cf-product-name")).toEqual(["noble-web", "Workers"]);
    expect(text(".cf-attachment-hostname")).toEqual(["staging.noble.coffee", "noble.coffee", "api.noble.coffee"]);
    // Muted status badges; the SSL state only when present; Workers rows
    // read service / environment.
    const badges = text(".cf-attachment-row .cf-badge");
    expect(badges).toEqual(["active", "ssl: active", "pending", "noble-api / production"]);
  });

  it("hides the attachments section when api/attachments fails — domains rows unaffected", async () => {
    render(<DomainsView fetcher={domainsFetcher()} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    // Let the failed attachments fetch settle.
    await new Promise((r) => setTimeout(r, 20));
    expect(document.querySelector(".cf-attachments")).toBeNull();
    expect(text(".cf-domain-name")).toEqual(["soon.dev", "warn.net", "later.org", "outside.example"]);
  });
});

// ---- Lifecycle UI (FEAT-pEWA6N6 phases 2-3): detach/attach rows, the
// gated zone-removal panel, and the audit card. Same injection patterns as
// above: reads through fetcher, mutations through mutator.

const LIFECYCLE_ATTACHMENTS: AttachmentsPayload = {
  generatedAt: "2026-10-12T00:00:00Z",
  pages: [
    {
      project: "noble-web",
      domains: [{ hostname: "staging.noble.coffee", status: "active", ssl: "active", verification: "active" }],
    },
  ],
  workers: [{ hostname: "api.example.dev", service: "noble-api", environment: "production", id: "dmn_123" }],
  zones: [
    { id: "z1", name: "noble.coffee", status: "active", recordCount: 42, protected: true },
    { id: "z2", name: "example.dev", status: "active", recordCount: 7 },
  ],
  errors: [],
};

const AUDIT_ENTRIES: AuditEntry[] = [
  { ts: "2026-10-12T01:00:00Z", action: "detach", target: "noble-web/staging.noble.coffee", operator: "owner@example.com", ok: true },
  { ts: "2026-10-12T00:30:00Z", action: "zone-remove", target: "example.dev", operator: "owner@example.com", ok: false, error: "typed name does not match" },
];

const AUDIT: AuditPayload = { audit: AUDIT_ENTRIES };

/** Fetcher over the lifecycle fixtures: domains + attachments + audit. */
function lifecycleFetcher(opts: { attachments?: AttachmentsPayload; audit?: AuditPayload; auditError?: Error } = {}): Parameters<typeof DomainsView>[0]["fetcher"] {
  return async <T,>(endpoint: string): Promise<FetchResult<T>> => {
    if (endpoint === "api/domains") return { data: DOMAINS as unknown as T, source: "network", ageSec: 0, demo: false };
    if (endpoint === "api/attachments") {
      if (opts.attachments) return { data: opts.attachments as unknown as T, source: "network", ageSec: 0, demo: false };
      throw new Error("HTTP 503");
    }
    if (endpoint === "api/audit") {
      if (opts.auditError) throw opts.auditError;
      if (opts.audit) return { data: opts.audit as unknown as T, source: "network", ageSec: 0, demo: false };
      throw new Error("HTTP 503");
    }
    throw new Error("HTTP 404");
  };
}

const buttonByText = (label: string): HTMLButtonElement | undefined =>
  Array.from(document.querySelectorAll<HTMLButtonElement>("button")).find((b) => b.textContent === label);

describe("DomainsView lifecycle (FEAT-pEWA6N6 phases 2-3)", () => {
  it("detaches a Pages domain after the inline confirm: Take offline → Confirm detach POSTs and the row flips to detached", async () => {
    const calls: { endpoint: string; body: unknown }[] = [];
    const mutator = async (endpoint: string, body: unknown): Promise<unknown> => {
      calls.push({ endpoint, body });
      return { ok: true };
    };
    render(<DomainsView fetcher={lifecycleFetcher({ attachments: LIFECYCLE_ATTACHMENTS })} mutator={mutator} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-attachment-row")).toBeTruthy();
    });
    // One inline confirm: Take offline turns into Confirm detach + Cancel.
    fireEvent.click(buttonByText("Take offline")!);
    await waitFor(() => {
      expect(buttonByText("Confirm detach")?.textContent).toBe("Confirm detach");
    });
    expect(buttonByText("Cancel")?.textContent).toBe("Cancel");
    fireEvent.click(buttonByText("Confirm detach")!);
    await waitFor(() => {
      expect(calls).toEqual([{ endpoint: "api/domains/detach", body: { kind: "pages", project: "noble-web", name: "staging.noble.coffee" } }]);
    });
    // The row flips to a detached state with a Put back control.
    await waitFor(() => {
      expect(buttonByText("Put back")).toBeTruthy();
    });
    expect(text(".cf-attachment-badges .cf-badge")).toContain("detached");
    // Put back POSTs the attach counterpart.
    fireEvent.click(buttonByText("Put back")!);
    await waitFor(() => {
      expect(calls[1]).toEqual({ endpoint: "api/domains/attach", body: { kind: "pages", project: "noble-web", name: "staging.noble.coffee" } });
    });
  });

  it("surfaces a detached-state API error verbatim under the row", async () => {
    const mutator = async (): Promise<unknown> => {
      throw new Error("routing conflict: domain is in use");
    };
    render(<DomainsView fetcher={lifecycleFetcher({ attachments: LIFECYCLE_ATTACHMENTS })} mutator={mutator} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-attachment-row")).toBeTruthy();
    });
    fireEvent.click(buttonByText("Take offline")!); // → confirm
    await waitFor(() => {
      expect(buttonByText("Confirm detach")).toBeTruthy();
    });
    fireEvent.click(buttonByText("Confirm detach")!);
    await waitFor(() => {
      expect(document.querySelector(".cf-attachment-error")?.textContent).toBe("routing conflict: domain is in use");
    });
    expect(document.querySelector(".cf-attachment-error")?.classList.contains("cf-level-warning-text")).toBe(true);
  });

  it("shows the workers certificate note in the detached state and sends the domainId on detach", async () => {
    const calls: { endpoint: string; body: unknown }[] = [];
    const mutator = async (endpoint: string, body: unknown): Promise<unknown> => {
      calls.push({ endpoint, body });
      return { ok: true };
    };
    render(<DomainsView fetcher={lifecycleFetcher({ attachments: LIFECYCLE_ATTACHMENTS })} mutator={mutator} />);
    await waitFor(() => {
      expect(text(".cf-attachment-hostname")).toContain("api.example.dev");
    });
    // The workers row is the second Take offline button (after the Pages row's).
    const takeOffline = Array.from(document.querySelectorAll<HTMLButtonElement>("button")).filter((b) => b.textContent === "Take offline");
    fireEvent.click(takeOffline[1]);
    await waitFor(() => {
      expect(buttonByText("Confirm detach")).toBeTruthy();
    });
    fireEvent.click(buttonByText("Confirm detach")!);
    await waitFor(() => {
      expect(calls).toEqual([{ endpoint: "api/domains/detach", body: { kind: "workers", name: "api.example.dev", domainId: "dmn_123" } }]);
    });
    await waitFor(() => {
      expect(document.querySelector(".cf-attachment-note")?.textContent).toContain("does not delete its advanced certificate");
    });
  });

  it("gates zone removal: exact typed name required, protected zones show the lock line with a disabled submit", async () => {
    const calls: { endpoint: string; body: unknown }[] = [];
    const mutator = async (endpoint: string, body: unknown): Promise<unknown> => {
      calls.push({ endpoint, body });
      return { ok: true };
    };
    render(<DomainsView fetcher={lifecycleFetcher({ attachments: LIFECYCLE_ATTACHMENTS })} mutator={mutator} />);
    await waitFor(() => {
      expect(text(".cf-product-name")).toContain("Zones");
    });
    const removeButtons = Array.from(document.querySelectorAll<HTMLButtonElement>("button")).filter((b) => b.textContent === "Remove");
    expect(removeButtons).toHaveLength(2);
    // Protected zone first (z1): the panel shows the lock line and the
    // submit stays disabled — no typed input is even offered.
    fireEvent.click(removeButtons[0]);
    await waitFor(() => {
      expect(document.querySelector(".cf-zone-lock")).toBeTruthy();
    });
    expect(document.querySelector(".cf-zone-lock")?.textContent).toContain("noble.coffee is a protected host");
    expect(document.querySelector(".cf-zone-lock")?.textContent).toContain("operator override");
    expect(document.querySelector(".cf-zone-warning")?.textContent).toContain("This cannot be undone");
    expect(buttonByText("Confirm removal")!.disabled).toBe(true);
    fireEvent.click(buttonByText("Confirm removal")!);
    expect(calls).toEqual([]);
    // Non-protected zone (z2): submit enables only on an exact typed match.
    fireEvent.click(removeButtons[1]);
    await waitFor(() => {
      expect(document.querySelector("input.cf-input")).toBeTruthy();
    });
    const input = document.querySelector<HTMLInputElement>("input.cf-input")!;
    const nonProtectedConfirm = Array.from(document.querySelectorAll<HTMLButtonElement>("button")).filter((b) => b.textContent === "Confirm removal")[1];
    expect(nonProtectedConfirm.disabled).toBe(true);
    fireEvent.change(input, { target: { value: "example.dev " } });
    expect(nonProtectedConfirm.disabled).toBe(true); // trailing space ≠ exact
    fireEvent.change(input, { target: { value: "example.dev" } });
    expect(nonProtectedConfirm.disabled).toBe(false);
    fireEvent.click(nonProtectedConfirm);
    await waitFor(() => {
      expect(calls).toEqual([{ endpoint: "api/zones/remove", body: { zoneId: "z2", typedName: "example.dev", override: false } }]);
    });
    // The row flips to its removed state.
    await waitFor(() => {
      expect(text(".cf-attachment-badges .cf-badge")).toContain("removed");
    });
  });

  it("renders zone-removal API refusals verbatim under the zone row", async () => {
    const mutator = async (): Promise<unknown> => {
      throw new Error("example.dev is a protected host — removal requires an operator override (ZONE_REMOVE_OVERRIDE=1 and override:true)");
    };
    const attachments: AttachmentsPayload = { ...LIFECYCLE_ATTACHMENTS, zones: [LIFECYCLE_ATTACHMENTS.zones[1]] };
    render(<DomainsView fetcher={lifecycleFetcher({ attachments })} mutator={mutator} />);
    await waitFor(() => {
      expect(buttonByText("Remove")).toBeTruthy();
    });
    fireEvent.click(buttonByText("Remove")!);
    await waitFor(() => {
      expect(document.querySelector("input.cf-input")).toBeTruthy();
    });
    fireEvent.change(document.querySelector<HTMLInputElement>("input.cf-input")!, { target: { value: "example.dev" } });
    fireEvent.click(buttonByText("Confirm removal")!);
    await waitFor(() => {
      expect(document.querySelector(".cf-attachment-error")?.textContent).toContain("operator override");
    });
  });

  it("renders the audit card newest-first from api/audit and shows skeleton/error states", async () => {
    render(<DomainsView fetcher={lifecycleFetcher({ attachments: LIFECYCLE_ATTACHMENTS, audit: AUDIT })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-audit-row")).toBeTruthy();
    });
    expect(document.querySelector(".cf-audit-title")?.textContent).toBe("Audit");
    // Newest first: the successful detach (01:00) above the failed removal (00:30).
    const actions = Array.from(document.querySelectorAll(".cf-audit-row .cf-audit-action")).map((n) => n.textContent);
    expect(actions).toEqual(["detach", "zone-remove"]);
    const fails = Array.from(document.querySelectorAll(".cf-audit-row .cf-audit-fail")).map((n) => n.textContent);
    expect(fails).toEqual(["error — typed name does not match"]);
    expect(document.querySelector(".cf-audit-row .cf-audit-operator")?.textContent).toBe("owner@example.com");
  });

  it("shows the audit error state when api/audit fails", async () => {
    render(<DomainsView fetcher={lifecycleFetcher({ attachments: LIFECYCLE_ATTACHMENTS, auditError: new Error("HTTP 503") })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-audit .cf-level-warning-text")?.textContent).toContain("Could not load the audit log — HTTP 503");
    });
    expect(document.querySelector(".cf-audit-row")).toBeNull();
  });
});
