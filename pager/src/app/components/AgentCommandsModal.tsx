// AgentCommandsModal (P-18, FEAT-pC1EV32): the copy-paste dispatch surface
// for the deterministic advisory brief. Opened from a firing alert card, it
// fetches /api/advisory?zone=<zone> fresh on every open (the endpoint is
// private, no-store — plain fetch, never the caching api client), shimmers
// while collecting, and renders the agentBrief as a selectable mono block
// the operator pastes to their coding agent. Copy crossfades to a checkmark
// (150ms ease-out, opacity only — no bounce) and auto-reverts after 1.5s.
// Esc and overlay-click close; Tab is trapped (shared trapTab).
import { useEffect, useRef, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { trapTab } from "./focus";

// The /api/advisory envelope (apps/ops/src/index.ts): brief is null when the
// zone has zero findings, and then agentBrief is "" — nothing to dispatch.
interface AdvisoryEnvelope {
  zone: string;
  zoneId: string;
  generatedAt: string;
  facts: { window: string };
  brief: { findings: unknown[] } | null;
  rendered: { agentBrief: string };
}

type LoadState = "loading" | "error" | "ready";

export default function AgentCommandsModal({ zone, onClose }: { zone: string; onClose: () => void }): React.JSX.Element {
  const reduced = useReducedMotion();
  const [state, setState] = useState<LoadState>("loading");
  const [error, setError] = useState<string | null>(null);
  const [envelope, setEnvelope] = useState<AdvisoryEnvelope | null>(null);
  // Retry re-runs the fetch effect; the tick is never read.
  const [retryTick, setRetryTick] = useState(0);
  const [copied, setCopied] = useState(false);
  const copyTimer = useRef<number | null>(null);
  useEffect(() => () => {
    if (copyTimer.current !== null) window.clearTimeout(copyTimer.current);
  }, []);

  // Fresh fetch on every open and on Retry: the advisory is measured now,
  // not a cacheable view — the endpoint answers private, no-store.
  useEffect(() => {
    let cancelled = false;
    setState("loading");
    setError(null);
    void (async () => {
      try {
        const res = await fetch(`/api/advisory?zone=${encodeURIComponent(zone)}`, {
          credentials: "same-origin",
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const body = (await res.json()) as AdvisoryEnvelope;
        if (cancelled) return;
        setEnvelope(body);
        setState("ready");
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : String(err));
        setState("error");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [zone, retryTick]);

  // Zero findings → brief is null and agentBrief is "": an honest empty
  // state, and a Copy button that must not look available.
  const agentBrief = envelope?.rendered.agentBrief ?? "";
  const hasBrief = state === "ready" && envelope !== null && envelope.brief !== null && agentBrief.length > 0;

  const copy = (): void => {
    if (!hasBrief || copied) return;
    void navigator.clipboard.writeText(agentBrief).then(
      () => {
        setCopied(true);
        copyTimer.current = window.setTimeout(() => setCopied(false), 1500);
      },
      () => undefined, // clipboard refused: the label stays honest (Copy)
    );
  };

  const dialogRef = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    const onKey = (event: KeyboardEvent): void => {
      if (event.key === "Escape") {
        onClose();
        return;
      }
      if (event.key === "Tab" && dialogRef.current) trapTab(dialogRef.current, event);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <motion.div
      className="cf-modal-backdrop"
      initial={reduced ? false : { opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={reduced ? { duration: 0 } : { duration: 0.15, ease: "easeOut" }}
      onClick={onClose}
    >
      <motion.div
        role="dialog"
        aria-modal="true"
        aria-labelledby="cf-agent-title"
        ref={dialogRef}
        className="cf-modal"
        initial={reduced ? false : { opacity: 0, scale: 0.96, y: 8 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={reduced ? { duration: 0 } : { duration: 0.2, ease: "easeOut" }}
        onClick={(event) => event.stopPropagation()}
      >
        <p className="cf-modal-eyebrow">CosmoLabs Ops</p>
        <h2 id="cf-agent-title" className="cf-modal-title">Commands to AI Agent · {zone}</h2>
        {state === "ready" && envelope !== null ? (
          <p className="cf-modal-body">measured {envelope.facts.window} · CosmoLabs Ops</p>
        ) : null}

        {state === "loading" ? (
          <div className="cf-loading">
            <div className="cf-skeleton" />
            <div className="cf-skeleton" />
            <div className="cf-skeleton" />
          </div>
        ) : state === "error" ? (
          <>
            <p className="cf-modal-body">Could not load the advisory brief: {error}</p>
            <div className="cf-modal-actions">
              <button type="button" className="cf-btn cf-btn-primary" onClick={() => setRetryTick((t) => t + 1)}>
                Retry
              </button>
              <button type="button" className="cf-btn cf-btn-ghost" onClick={onClose}>
                Close
              </button>
            </div>
          </>
        ) : (
          <>
            {hasBrief ? (
              <pre className="cf-agent-brief">{agentBrief}</pre>
            ) : (
              <p className="cf-modal-body">Nothing to dispatch — no concerns measured for this zone in the current window.</p>
            )}
            <div className="cf-modal-actions">
              <button
                type="button"
                className="cf-btn cf-btn-primary cf-agent-copy"
                disabled={!hasBrief}
                onClick={copy}
              >
                <span className={`cf-agent-face ${copied ? "is-off" : "is-on"}`}>Copy</span>
                <span className={`cf-agent-face cf-agent-check ${copied ? "is-on" : "is-off"}`} aria-hidden="true">
                  ✓ Copied
                </span>
              </button>
              <button type="button" className="cf-btn cf-btn-ghost" onClick={onClose}>
                Close
              </button>
            </div>
          </>
        )}
      </motion.div>
    </motion.div>
  );
}
