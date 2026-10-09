// Rules view (FEAT-052, agent I): edit alert rules — name, condition,
// threshold, exclude chips, enabled toggle — from the phone, and PUT the
// whole set back to /api/rules. Vanilla DOM like the other Ops views. Pure
// helpers (validation, exclude-chip math, threshold hint) are exported for
// tests; rendering needs a DOM.

import { el, skeleton, statusLine } from "./dom";
import { formatCount } from "./format";
import { api, LoginExpiredError, type FetchResult } from "./api";
import { refreshButton } from "./refresh";

// ---- /api/rules contract (built by agent B; consumed here) ----

/** One alert rule as the Ops Worker stores and evaluates it. */
export interface AlertRule {
  name: string;
  condition: string;
  threshold: number;
  exclude?: string[];
  enabled: boolean;
}

export interface RulesPayload {
  rules: AlertRule[];
  /** Condition ids the server understands — the condition <select> options. */
  conditions: string[];
  /** True when the account still runs the uncustomised starter set. */
  starter: boolean;
}

/** The D1 rows-read condition — its thresholds are formatted as counts. */
export const D1_ROWS_READ = "d1-rows-read";

/**
 * Validate one draft rule against the server's condition list. Returns the
 * error messages, empty when the draft is valid. The trimmed-name check
 * comes first so a whitespace-only name reads as missing.
 */
export function validateRuleDraft(rule: AlertRule, conditions: readonly string[]): string[] {
  const errors: string[] = [];
  if (rule.name.trim() === "") errors.push("Name is required.");
  if (!conditions.includes(rule.condition)) errors.push("Choose a condition from the list.");
  if (!Number.isFinite(rule.threshold) || rule.threshold < 0) errors.push("Threshold must be a number of 0 or more.");
  return errors;
}

/**
 * Add one exclude value (trimmed) unless it is empty or already on the
 * list. Match is exact after trimming — "cf-state" and "CF-State" are
 * distinct names on Cloudflare. Returns a new array; the input is untouched.
 */
export function addExclude(list: readonly string[], raw: string): string[] {
  const value = raw.trim();
  if (value === "") return [...list];
  if (list.includes(value)) return [...list];
  return [...list, value];
}

/** Remove every exact match of `value` (trimmed). Returns a new array. */
export function removeExclude(list: readonly string[], value: string): string[] {
  const needle = value.trim();
  return list.filter((v) => v !== needle);
}

/**
 * Human hint under the threshold input. Only the D1 rows-read condition has
 * one today: "1.0B" for 1e9, so the phone user does not count zeros.
 */
export function thresholdHint(condition: string, threshold: number): string | null {
  if (condition !== D1_ROWS_READ) return null;
  if (!Number.isFinite(threshold) || threshold <= 0) return null;
  return formatCount(threshold);
}

function errorState(root: HTMLElement, err: unknown): void {
  if (err instanceof LoginExpiredError) {
    root.replaceChildren(el("p", "cf-empty cf-login-expired", err.message));
    return;
  }
  const box = el("section", "cf-card cf-level-warning");
  box.append(
    el("h2", undefined, "Could not load rules"),
    el("p", "cf-row-detail", err instanceof Error ? err.message : String(err)),
  );
  const retry = el("button", "cf-btn", "Retry");
  retry.addEventListener("click", () => void renderRules(root));
  box.append(retry);
  root.replaceChildren(box);
}

/** Render the Rules view into `root`: skeleton while loading, cards after. */
export async function renderRules(root: HTMLElement): Promise<void> {
  root.replaceChildren(skeleton());
  try {
    const res = await api.fetchJson<RulesPayload>("api/rules");
    renderRulesInto(root, res);
  } catch (err) {
    errorState(root, err);
  }
}

// renderRulesInto paints one RulesPayload as an editable draft set. All
// edits land in `drafts` first; Save PUTs the drafts and repaints from the
// server's echo. Structural changes (add, delete, chip add/remove) re-render
// the card list from drafts, so the DOM never drifts from state.
function renderRulesInto(root: HTMLElement, res: FetchResult<RulesPayload>): void {
  const payload = res.data;
  const drafts: AlertRule[] = payload.rules.map((r) => ({
    ...r,
    exclude: r.exclude ? [...r.exclude] : undefined,
  }));
  let highlightIndex = -1;

  // ---- Card list ----

  const list = el("div", "cf-rules-list");

  // removeChipButton builds the × control for one exclude chip. The label
  // names the value so screen readers announce which chip the button kills.
  function removeChipButton(value: string, onRemove: () => void): HTMLButtonElement {
    const rm = el("button", "cf-chip-remove", "×");
    rm.type = "button";
    rm.setAttribute("aria-label", `Remove ${value}`);
    rm.addEventListener("click", onRemove);
    return rm;
  }

  function ruleCard(draft: AlertRule, index: number): HTMLElement {
    const card = el("section", "cf-card cf-rule");
    card.dataset.index = String(index);
    if (index === highlightIndex) card.classList.add("cf-rule-invalid");
    if (!draft.enabled) card.classList.add("cf-rule-off");

    // Name + enabled toggle share the first row.
    const head = el("div", "cf-rule-head");
    const name = document.createElement("input");
    name.type = "text";
    name.className = "cf-input cf-rule-name";
    name.value = draft.name;
    name.placeholder = "Rule name";
    name.setAttribute("aria-label", `Rule ${index + 1} name`);
    name.addEventListener("input", () => {
      draft.name = name.value;
    });

    const toggle = el("button", "cf-toggle", draft.enabled ? "On" : "Off");
    toggle.type = "button";
    toggle.setAttribute("aria-pressed", String(draft.enabled));
    toggle.setAttribute("aria-label", `Rule ${index + 1} enabled`);
    toggle.addEventListener("click", () => {
      draft.enabled = !draft.enabled;
      toggle.textContent = draft.enabled ? "On" : "Off";
      toggle.setAttribute("aria-pressed", String(draft.enabled));
      card.classList.toggle("cf-rule-off", !draft.enabled);
    });
    head.append(name, toggle);

    // Condition select from the server's list.
    const condField = el("div", "cf-field");
    condField.append(el("label", "cf-field-label", "Condition"));
    const cond = document.createElement("select");
    cond.className = "cf-input cf-select";
    cond.setAttribute("aria-label", `Rule ${index + 1} condition`);
    for (const c of payload.conditions) {
      const opt = document.createElement("option");
      opt.value = c;
      opt.textContent = c;
      cond.append(opt);
    }
    cond.value = payload.conditions.includes(draft.condition) ? draft.condition : payload.conditions[0] ?? "";
    cond.addEventListener("change", () => {
      draft.condition = cond.value;
      paintHint();
    });
    condField.append(cond);

    // Threshold number input + formatted hint (D1 rows-read).
    const threshField = el("div", "cf-field");
    threshField.append(el("label", "cf-field-label", "Threshold"));
    const hint = el("span", "cf-rule-hint");
    function paintHint(): void {
      const h = thresholdHint(draft.condition, draft.threshold);
      hint.textContent = h ?? "";
      hint.hidden = h === null;
    }
    const thresh = document.createElement("input");
    thresh.type = "number";
    thresh.className = "cf-input";
    thresh.min = "0";
    thresh.step = "any";
    thresh.value = Number.isFinite(draft.threshold) ? String(draft.threshold) : "";
    thresh.setAttribute("aria-label", `Rule ${index + 1} threshold`);
    thresh.addEventListener("input", () => {
      draft.threshold = thresh.value === "" ? NaN : Number(thresh.value);
      paintHint();
    });
    paintHint();
    const threshRow = el("div", "cf-field-row");
    threshRow.append(thresh, hint);
    threshField.append(threshRow);

    // Excludes: removable chips + add input.
    const exField = el("div", "cf-field cf-exclude-field");
    exField.append(el("label", "cf-field-label", "Exclude — names this rule ignores"));
    const chips = el("div", "cf-chips");
    function paintChips(): void {
      chips.replaceChildren();
      const exclude = draft.exclude ?? [];
      for (const value of exclude) {
        const chip = el("span", "cf-chip");
        chip.append(
          el("span", "cf-chip-value", value),
          removeChipButton(value, () => {
            draft.exclude = removeExclude(draft.exclude ?? [], value);
            paintChips();
          }),
        );
        chips.append(chip);
      }
      exField.classList.toggle("is-empty", exclude.length === 0);
    }
    const addInput = document.createElement("input");
    addInput.type = "text";
    addInput.className = "cf-input cf-chip-add";
    addInput.placeholder = "Add name to exclude";
    addInput.setAttribute("aria-label", `Rule ${index + 1} add exclude`);
    const addBtn = el("button", "cf-btn cf-btn-ghost", "Add");
    addBtn.type = "button";
    addBtn.addEventListener("click", () => {
      draft.exclude = addExclude(draft.exclude ?? [], addInput.value);
      addInput.value = "";
      paintChips();
    });
    const addRow = el("div", "cf-field-row");
    addRow.append(addInput, addBtn);
    paintChips();
    exField.append(chips, addRow);

    // Per-card footer: Delete. Validation errors surface at the action bar
    // (saveError) with the offending card highlighted.
    const del = el("button", "cf-btn cf-btn-ghost cf-btn-danger", "Delete");
    del.type = "button";
    del.addEventListener("click", () => {
      drafts.splice(index, 1);
      highlightIndex = -1;
      paintList();
    });

    card.append(head, condField, threshField, exField, del);
    return card;
  }

  function paintList(): void {
    list.replaceChildren();
    if (drafts.length === 0) {
      list.append(el("p", "cf-empty", "No rules. Add one to start alerting."));
      return;
    }
    drafts.forEach((draft, i) => list.append(ruleCard(draft, i)));
  }

  // ---- Action bar, starter note, save feedback ----

  const bar = el("div", "cf-dash-bar");
  const line = statusLine(res.ageSec, { demo: res.demo });
  const refresh = refreshButton(() => renderRules(root));
  bar.append(line, refresh);

  const note = payload.starter
    ? el("p", "cf-rule-note", "Using starter rules — save to customise")
    : el("p", "cf-rule-note");
  if (!payload.starter) note.hidden = true;

  const actions = el("div", "cf-rules-actions");
  const add = el("button", "cf-btn", "Add rule");
  add.type = "button";
  add.addEventListener("click", () => {
    drafts.push({
      name: "",
      condition: payload.conditions[0] ?? "",
      threshold: 0,
      exclude: [],
      enabled: true,
    });
    highlightIndex = -1;
    paintList();
  });
  const save = el("button", "cf-btn cf-btn-primary", "Save");
  save.type = "button";
  actions.append(add, save);

  const saveStatus = el("p", "cf-save-status", "");
  saveStatus.hidden = true;
  const saveError = el("p", "cf-save-error cf-level-critical-text", "");
  saveError.hidden = true;

  function flashStatus(): void {
    saveStatus.textContent = "Saved";
    saveStatus.hidden = false;
    window.setTimeout(() => {
      saveStatus.hidden = true;
      saveStatus.textContent = "";
    }, 2500);
  }

  save.addEventListener("click", () => {
    save.disabled = true;
    saveError.hidden = true;

    // Client-side validation first: highlight the first bad card.
    highlightIndex = -1;
    for (let i = 0; i < drafts.length; i++) {
      const problems = validateRuleDraft(drafts[i], payload.conditions);
      if (problems.length > 0) {
        highlightIndex = i;
        saveError.textContent = `Rule ${i + 1}: ${problems.join(" ")}`;
        saveError.hidden = false;
        paintList();
        save.disabled = false;
        return;
      }
    }

    void putRules(drafts)
      .then((echo) => {
        // Adopt the server's echo so the next paint matches what is stored.
        drafts.splice(0, drafts.length, ...echo.rules.map((r) => ({
          ...r,
          exclude: r.exclude ? [...r.exclude] : undefined,
        })));
        payload.starter = false;
        highlightIndex = -1;
        flashStatus();
        paintList();
      })
      .catch((err: unknown) => {
        if (err instanceof RulesHttpError) {
          highlightIndex = err.index ?? -1;
          saveError.textContent = err.index === undefined
            ? err.message
            : `Rule ${err.index + 1}: ${err.message}`;
        } else {
          highlightIndex = -1;
          saveError.textContent = err instanceof Error ? err.message : String(err);
        }
        saveError.hidden = false;
        paintList();
      })
      .finally(() => {
        save.disabled = false;
      });
  });

  paintList();
  root.replaceChildren(bar, note, list, saveError, saveStatus, actions);
}

// ---- PUT /api/rules ----

/** Thrown when the server answers 400 — carries the offending rule index. */
export class RulesHttpError extends Error {
  readonly index?: number;
  constructor(message: string, index?: number) {
    super(message);
    this.name = "RulesHttpError";
    this.index = index;
  }
}

// putRules PUTs the full draft set. 200 echoes `{ rules }`; 400 carries
// `{ error, index? }`. An expired Access session surfaces as
// LoginExpiredError exactly like the GET path (opaqueredirect/403).
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
