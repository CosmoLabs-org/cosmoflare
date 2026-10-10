// Rules editor as a real React view (P-06, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// rules.ts's renderRules. Editable rule cards (name + enabled toggle,
// condition select, threshold with the count hint, exclude chips, delete),
// an action bar (Add rule / Save), and the save ritual: client-side
// validation with per-card highlight + action-bar error, PUT of the whole
// draft set, then the server echo is adopted and written into the ApiClient
// cache so a second Save can never PUT stale rules (BUG-057 defects 1-7).
// The pure helpers (validateRuleDraft, addExclude, removeExclude,
// thresholdHint, saveErrorLabel, RulesHttpError) and the AlertRule /
// RulesPayload types are imported verbatim from rules.ts. The section
// masthead owns the page title.

import { useCallback, useEffect, useRef, useState } from "react";
import { api, LoginExpiredError } from "../../api";
import {
  addExclude,
  removeExclude,
  RulesHttpError,
  saveErrorLabel,
  thresholdHint,
  validateRuleDraft,
  type AlertRule,
  type RulesPayload,
} from "../../rules";
import type { ViewFetcher } from "./WorkersView";

/** Deep-ish copy of the server's rules into editable drafts. */
function copyRules(rules: AlertRule[]): AlertRule[] {
  return rules.map((r) => ({ ...r, exclude: r.exclude ? [...r.exclude] : undefined }));
}

// putRules PUTs the full draft set — the same wire contract as rules.ts's
// internal putRules: 200 echoes `{ rules }`; 400 carries `{ error, index? }`
// and surfaces as RulesHttpError; an expired Access session answers
// opaqueredirect/403 and surfaces as LoginExpiredError.
async function putRules(rules: AlertRule[]): Promise<RulesPayload> {
  let res: Response;
  try {
    res = await fetch("/api/rules", {
      method: "PUT",
      credentials: "same-origin",
      redirect: "manual",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ rules }),
    });
  } catch (err) {
    throw new Error(`Could not reach the Ops Worker: ${err instanceof Error ? err.message : String(err)}`);
  }
  if (res.type === "opaqueredirect" || res.status === 403) throw new LoginExpiredError();
  const body = (await res.json().catch(() => undefined)) as
    | { rules?: AlertRule[]; error?: string; index?: number }
    | undefined;
  if (res.status === 400) throw new RulesHttpError(body?.error ?? "The server rejected the rules.", body?.index);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  if (!body || !Array.isArray(body.rules)) throw new Error("The server returned no rules — try Refresh.");
  return { rules: body.rules, conditions: [], starter: false };
}

/** One editable rule card. Structural edits go up through onChange; the
 *  chip add input is card-local state. */
function RuleCard({ draft, index, conditions, invalid, onChange, onDelete }: {
  draft: AlertRule;
  index: number;
  conditions: readonly string[];
  invalid: boolean;
  onChange: (patch: Partial<AlertRule>) => void;
  onDelete: () => void;
}): React.JSX.Element {
  const [addValue, setAddValue] = useState("");
  const exclude = draft.exclude ?? [];
  const hint = thresholdHint(draft.condition, draft.threshold);

  const chipRemove = (value: string): void => {
    onChange({ exclude: removeExclude(exclude, value) });
  };
  const chipAdd = (): void => {
    onChange({ exclude: addExclude(exclude, addValue) });
    setAddValue("");
  };

  return (
    <section
      className={`cf-card cf-rule${invalid ? " cf-rule-invalid" : ""}${draft.enabled ? "" : " cf-rule-off"}`}
      data-index={index}
    >
      {/* Name + enabled toggle share the first row. */}
      <div className="cf-rule-head">
        <input
          type="text"
          className="cf-input cf-rule-name"
          value={draft.name}
          placeholder="Rule name"
          aria-label={`Rule ${index + 1} name`}
          onChange={(e) => onChange({ name: e.target.value })}
        />
        <button
          type="button"
          className="cf-toggle"
          aria-pressed={draft.enabled}
          aria-label={`Rule ${index + 1} enabled`}
          onClick={() => onChange({ enabled: !draft.enabled })}
        >
          {draft.enabled ? "On" : "Off"}
        </button>
      </div>

      {/* Condition select from the server's list. */}
      <div className="cf-field">
        <label className="cf-field-label">Condition</label>
        <select
          className="cf-input cf-select"
          aria-label={`Rule ${index + 1} condition`}
          value={conditions.includes(draft.condition) ? draft.condition : conditions[0] ?? ""}
          onChange={(e) => onChange({ condition: e.target.value })}
        >
          {conditions.map((c) => (
            <option key={c} value={c}>{c}</option>
          ))}
        </select>
      </div>

      {/* Threshold number input + formatted hint (D1 rows-read, KV writes). */}
      <div className="cf-field">
        <label className="cf-field-label">Threshold</label>
        <div className="cf-field-row">
          <input
            type="number"
            className="cf-input"
            min="0"
            step="any"
            value={Number.isFinite(draft.threshold) ? String(draft.threshold) : ""}
            aria-label={`Rule ${index + 1} threshold`}
            onChange={(e) => onChange({ threshold: e.target.value === "" ? NaN : Number(e.target.value) })}
          />
          <span className="cf-rule-hint" hidden={hint === null}>{hint ?? ""}</span>
        </div>
      </div>

      {/* Excludes: removable chips + add input. */}
      <div className={`cf-field cf-exclude-field${exclude.length === 0 ? " is-empty" : ""}`}>
        <label className="cf-field-label">Exclude — names this rule ignores</label>
        <div className="cf-chips">
          {exclude.map((value) => (
            <span key={value} className="cf-chip">
              <span className="cf-chip-value">{value}</span>
              <button
                type="button"
                className="cf-chip-remove"
                aria-label={`Remove ${value}`}
                onClick={() => chipRemove(value)}
              >
                ×
              </button>
            </span>
          ))}
        </div>
        <div className="cf-field-row">
          <input
            type="text"
            className="cf-input cf-chip-add"
            placeholder="Add name to exclude"
            aria-label={`Rule ${index + 1} add exclude`}
            value={addValue}
            onChange={(e) => setAddValue(e.target.value)}
          />
          <button type="button" className="cf-btn cf-btn-ghost" onClick={chipAdd}>Add</button>
        </div>
      </div>

      {/* Per-card footer: Delete. Validation errors surface at the action
          bar with the offending card highlighted. */}
      <button type="button" className="cf-btn cf-btn-ghost cf-btn-danger" onClick={onDelete}>
        Delete
      </button>
    </section>
  );
}

export default function RulesView({ refreshSeq = 0, fetcher }: { refreshSeq?: number; fetcher?: ViewFetcher }): React.JSX.Element {
  // The Retry re-render must go through the SAME fetcher (injected or the
  // shared client) — hence the closure rather than a bare method reference.
  const fetchJson: ViewFetcher = fetcher ?? ((endpoint, opts) => api.fetchJson(endpoint, opts));
  const [payload, setPayload] = useState<RulesPayload | null>(null);
  const [drafts, setDrafts] = useState<AlertRule[]>([]);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  // rulesDirty: the user has interacted with the drafts since the last
  // paint. A background revalidation must NOT repaint over unsaved edits
  // (BUG-057 defect 2) — the repaint waits until the drafts are saved.
  const dirtyRef = useRef(false);
  const [highlight, setHighlight] = useState(-1);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveStatus, setSaveStatus] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const flashTimer = useRef<number | null>(null);

  // A background revalidation repaints the view with the fresh copy
  // (BUG-057 defect 2) — unless the user has unsaved draft edits.
  const repaint = useCallback((fresh: RulesPayload): void => {
    if (dirtyRef.current) return;
    setPayload(fresh);
    setDrafts(copyRules(fresh.rules));
  }, []);

  useEffect(() => {
    const refresh = refreshSeq > 0;
    let cancelled = false;
    void (async () => {
      try {
        const res = await fetchJson<RulesPayload>("api/rules", { refresh, onRevalidate: repaint });
        if (cancelled) return;
        dirtyRef.current = false;
        setPayload(res.data);
        setDrafts(copyRules(res.data.rules));
        setLoadError(null);
        setRefreshError(null);
      } catch (err) {
        if (cancelled) return;
        // A failed refresh keeps the last good data (BUG-057 defects 2/3);
        // only a first (skeleton) load shows the error/retry state.
        if (refresh) {
          setRefreshError(err instanceof Error ? err.message : String(err));
        } else {
          setLoadError({
            message: err instanceof Error ? err.message : String(err),
            expired: err instanceof LoginExpiredError,
          });
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [refreshSeq, retryTick, repaint, fetchJson]);

  useEffect(() => () => {
    if (flashTimer.current !== null) window.clearTimeout(flashTimer.current);
  }, []);

  if (loadError !== null) {
    if (loadError.expired) {
      return <p className="cf-empty cf-login-expired">{loadError.message}</p>;
    }
    return (
      <section className="cf-card cf-level-warning">
        <h3 className="cf-card-title">Could not load rules</h3>
        <p className="cf-row-detail">{loadError.message}</p>
        <button type="button" className="cf-btn" onClick={() => {
          setLoadError(null);
          setRetryTick((t) => t + 1);
        }}>
          Retry
        </button>
      </section>
    );
  }

  if (payload === null) {
    return (
      <div className="cf-loading">
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
      </div>
    );
  }

  const update = (index: number, patch: Partial<AlertRule>): void => {
    dirtyRef.current = true;
    setDrafts((ds) => ds.map((d, i) => (i === index ? { ...d, ...patch } : d)));
  };

  const addRule = (): void => {
    dirtyRef.current = true;
    setDrafts((ds) => [
      ...ds,
      {
        name: "",
        condition: payload.conditions[0] ?? "",
        threshold: 0,
        exclude: [],
        enabled: true,
      },
    ]);
    setHighlight(-1);
  };

  const deleteRule = (index: number): void => {
    dirtyRef.current = true;
    setDrafts((ds) => ds.filter((_, i) => i !== index));
    setHighlight(-1);
  };

  const flashStatus = (): void => {
    setSaveStatus("Saved");
    if (flashTimer.current !== null) window.clearTimeout(flashTimer.current);
    flashTimer.current = window.setTimeout(() => {
      setSaveStatus(null);
      flashTimer.current = null;
    }, 2500);
  };

  const save = (): void => {
    if (saving) return;
    setSaveError(null);

    // Client-side validation first: highlight the first bad card.
    for (let i = 0; i < drafts.length; i++) {
      const problems = validateRuleDraft(drafts[i], payload.conditions);
      if (problems.length > 0) {
        setHighlight(i);
        setSaveError(`Rule ${i + 1}: ${problems.join(" ")}`);
        return;
      }
    }

    setSaving(true);
    void putRules(drafts)
      .then((echo) => {
        // Adopt the server's echo so the next paint matches what is stored.
        setDrafts(copyRules(echo.rules));
        // The saved set is customised: the starter note clears and the
        // cached copy must agree (BUG-057 defect 1) — the server's
        // condition list is preserved for the <select> options.
        setPayload((p) => (p === null ? p : { ...p, starter: false }));
        api.updateCache("api/rules", { rules: echo.rules, conditions: payload.conditions, starter: false });
        dirtyRef.current = false;
        setHighlight(-1);
        flashStatus();
      })
      .catch((err: unknown) => {
        if (err instanceof RulesHttpError) {
          setHighlight(err.index ?? -1);
          setSaveError(saveErrorLabel(err.message, err.index));
        } else {
          setHighlight(-1);
          setSaveError(err instanceof Error ? err.message : String(err));
        }
      })
      .finally(() => {
        setSaving(false);
      });
  };

  return (
    <div>
      {refreshError !== null ? (
        <p className="cf-empty cf-level-warning-text" role="alert">{`Refresh failed — ${refreshError}`}</p>
      ) : null}

      {payload.starter ? <p className="cf-rule-note">Using starter rules — save to customise</p> : null}

      <div className="cf-rules-list">
        {drafts.length === 0 ? (
          <p className="cf-empty">No rules. Add one to start alerting.</p>
        ) : (
          drafts.map((draft, i) => (
            <RuleCard
              key={i}
              draft={draft}
              index={i}
              conditions={payload.conditions}
              invalid={i === highlight}
              onChange={(patch) => update(i, patch)}
              onDelete={() => deleteRule(i)}
            />
          ))
        )}
      </div>

      {saveError !== null ? (
        <p className="cf-save-error cf-level-critical-text" role="alert">{saveError}</p>
      ) : null}
      {saveStatus !== null ? <p className="cf-save-status">{saveStatus}</p> : null}

      <div className="cf-rules-actions">
        <button type="button" className="cf-btn" onClick={addRule}>Add rule</button>
        <button type="button" className="cf-btn cf-btn-primary" disabled={saving} onClick={save}>
          Save
        </button>
      </div>
    </div>
  );
}
