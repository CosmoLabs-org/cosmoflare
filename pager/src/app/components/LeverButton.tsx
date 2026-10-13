// LeverButton (operator 2026-10-12): THE reusable action button for every
// write-path lever in the app — purge, DNS pause/enable, WAF enable/
// disable, and whatever comes next. One theme, one state machine: an icon
// with the label BELOW it; optional two-step confirm (the first tap arms to
// danger red, the second fires — arming self-cancels after 4s); busy while
// the mutation runs; the Worker's error text surfaces inline with a Retry
// label; an optional done flash ("Purged") fades back to idle.
import { useEffect, useRef, useState } from "react";
import type { LucideIcon } from "lucide-react";

type Phase = "idle" | "armed" | "busy" | "done" | { error: string };

export interface LeverButtonProps {
  icon: LucideIcon;
  /** Idle label ("Pause"). */
  label: string;
  /** Armed label ("Confirm pause"); omit to fire without confirming. */
  confirmLabel?: string;
  /** What happens if the armed action fires ("record stops resolving") —
   *  shown under the lever while armed (critique P2: state the impact). */
  armedHint?: string;
  busyLabel?: string;
  /** Success flash label ("Purged"); omit to return straight to idle. */
  doneLabel?: string;
  /** Accessible description of what the button does. */
  ariaLabel?: string;
  /** The mutation. Resolves on success; a rejection becomes the inline error. */
  onFire: () => Promise<void>;
}

export default function LeverButton({ icon: Icon, label, confirmLabel, armedHint, busyLabel = "Working…", doneLabel, ariaLabel, onFire }: LeverButtonProps): React.JSX.Element {
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
      if (confirmLabel) {
        setPhase("armed");
        return;
      }
    } else if (phase !== "armed") {
      return;
    }
    setPhase("busy");
    void onFire()
      .then(() => setPhase(doneLabel ? "done" : "idle"))
      .catch((err: unknown) => setPhase({ error: err instanceof Error ? err.message : String(err) }));
  };

  const armed = phase === "armed";
  const textLabel = armed
    ? confirmLabel!
    : phase === "busy"
      ? busyLabel
      : phase === "done"
        ? doneLabel!
        : typeof phase === "object"
          ? `Retry ${label.toLowerCase()}`
          : label;

  return (
    <span className="cf-lever-wrap">
      <button
        type="button"
        className={`cf-lever${armed ? " is-armed" : ""}`}
        aria-label={ariaLabel ?? textLabel}
        disabled={phase === "busy"}
        onClick={onClick}
      >
        <span className="cf-lever-icon" aria-hidden="true">
          <Icon size={16} strokeWidth={1.9} />
        </span>
        <span className="cf-lever-label">{textLabel}</span>
      </button>
      {armed && armedHint ? <span className="cf-lever-hint" role="status">{armedHint}</span> : null}
      {typeof phase === "object" ? <span className="cf-lever-error" role="alert">{phase.error}</span> : null}
    </span>
  );
}
