// PurgeButton (operator 2026-10-12: action buttons): per-zone cache purge
// in the Zones table. Two-step confirm inline (no modal churn): the first
// click arms the danger state, the second fires POST /api/zones/purge —
// the same raw-fetch mutation pattern PairingView uses. Self-resetting:
// armed times out after 4s, done/error fades back to idle so the table
// stays quiet.
import { useEffect, useRef, useState } from "react";

// Default mutation call — the Worker's error JSON (e.g. the missing
// Cache Purge permission hint) is surfaced verbatim.
async function purgeZone(zone: string): Promise<void> {
  const res = await fetch("/api/zones/purge", {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ zone }),
  });
  if (res.ok) return;
  let detail = `HTTP ${res.status}`;
  try {
    const body = (await res.json()) as { error?: string };
    if (body.error) detail = body.error;
  } catch {
    // non-JSON error body: keep the HTTP status text
  }
  throw new Error(detail);
}

type Phase = "idle" | "armed" | "busy" | "done" | { error: string };

export default function PurgeButton({ zone, onPurge = purgeZone }: { zone: string; onPurge?: (zone: string) => Promise<void> }): React.JSX.Element {
  const [phase, setPhase] = useState<Phase>("idle");
  const timer = useRef<number | undefined>(undefined);

  useEffect(() => {
    if (phase !== "armed" && phase !== "done" && typeof phase !== "object") return;
    const ms = phase === "armed" ? 4000 : 3000;
    timer.current = window.setTimeout(() => setPhase("idle"), ms);
    return () => window.clearTimeout(timer.current);
  }, [phase]);

  const onClick = (): void => {
    if (phase === "idle") {
      setPhase("armed");
      return;
    }
    if (phase !== "armed") return;
    setPhase("busy");
    void onPurge(zone)
      .then(() => setPhase("done"))
      .catch((err: unknown) => setPhase({ error: err instanceof Error ? err.message : String(err) }));
  };

  const label = phase === "armed"
    ? "Confirm purge"
    : phase === "busy"
      ? "Purging…"
      : phase === "done"
        ? "Purged"
        : typeof phase === "object"
          ? "Retry purge"
          : "Purge cache";

  return (
    <span className="cf-purge">
      <button
        type="button"
        className={`cf-btn cf-btn-sm ${phase === "armed" ? "cf-btn-danger" : ""}`}
        aria-label={`Purge cache for ${zone}`}
        disabled={phase === "busy"}
        onClick={onClick}
      >
        {label}
      </button>
      {typeof phase === "object" ? <span className="cf-purge-error" role="alert">{phase.error}</span> : null}
    </span>
  );
}
