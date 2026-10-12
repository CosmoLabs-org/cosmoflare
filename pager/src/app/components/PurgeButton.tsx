// PurgeButton (operator 2026-10-12: action buttons): per-zone cache purge
// in the Zones table — a LeverButton with the house theme. Two-step
// confirm; the Worker's error JSON (permission hint) surfaces verbatim.
import { Eraser } from "lucide-react";
import LeverButton from "./LeverButton";

// Default mutation call — the Worker's error JSON is surfaced verbatim.
export async function purgeZone(zone: string): Promise<void> {
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

export default function PurgeButton({ zone, onPurge = purgeZone }: { zone: string; onPurge?: (zone: string) => Promise<void> }): React.JSX.Element {
  return (
    <span className="cf-purge">
      <LeverButton
        icon={Eraser}
        label="Purge cache"
        confirmLabel="Confirm purge"
        busyLabel="Purging…"
        doneLabel="Purged"
        ariaLabel={`Purge cache for ${zone}`}
        onFire={() => onPurge(zone)}
      />
    </span>
  );
}
