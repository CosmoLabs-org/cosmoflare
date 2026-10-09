// Refresh icon button (IMP-pPFDXJA): circular-arrow icon button with loading
// spin, success check and reduced-motion fallback. Shared by every data view
// (dashboard, billing, tables, rules) so refresh behaves identically everywhere.

import { el } from "./dom";

// Duration constants (ms).
const CHECK_MS = 900;
const SHAKE_MS = 240;
// fadeSwap fade duration; keep in sync with .cf-fade-swap in styles.css.
const FADE_MS = 150;

// prefersReducedMotion reports the OS-level reduced-motion preference.
// Guarded: vitest runs with environment "node", where matchMedia is absent.
export function prefersReducedMotion(): boolean {
  try {
    return globalThis.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
  } catch {
    return false;
  }
}

// Icon markup: static constants (no untrusted input), 18px, stroke follows
// the button text color via currentColor — same approach as logo.ts.
const REFRESH_SVG =
  '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' +
  '<polyline points="23 4 23 10 17 10"></polyline>' +
  '<path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>' +
  '</svg>';

const CHECK_SVG =
  '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' +
  '<polyline points="20 6 9 17 4 12"></polyline>' +
  '</svg>';

// setIcon swaps the glyph inside the icon span.
function setIcon(icon: HTMLElement, markup: string): void {
  icon.textContent = "";
  icon.innerHTML = markup;
}

// setGlyph shows a plain text glyph (the reduced-motion "…").
function setGlyph(icon: HTMLElement, glyph: string): void {
  icon.innerHTML = "";
  icon.textContent = glyph;
}

// rearm clears the busy state so the button accepts clicks again.
function rearm(btn: HTMLButtonElement): void {
  btn.removeAttribute("aria-busy");
  btn.disabled = false;
}

// refreshButton builds the 44x44 icon-only Refresh button used by every data
// view. While onRefresh runs the button reads busy to assistive tech
// (aria-busy + disabled + rotating arrow). On success the arrow swaps to a
// check for a beat; on failure the icon shakes and the tooltip carries the
// error. With prefers-reduced-motion the spin and shake are replaced by a
// static "…" glyph and no animation classes are applied.
export function refreshButton(onRefresh: () => Promise<void>): HTMLButtonElement {
  const btn = el("button", "cf-btn cf-refresh-btn");
  btn.type = "button";
  btn.setAttribute("aria-label", "Refresh data");
  btn.title = "Refresh";

  const icon = el("span", "cf-refresh-icon");
  icon.setAttribute("aria-hidden", "true");
  setIcon(icon, REFRESH_SVG);
  btn.append(icon);

  let busy = false;
  btn.addEventListener("click", () => {
    if (busy) return;
    busy = true;
    btn.disabled = true;
    btn.setAttribute("aria-busy", "true");
    const reduced = prefersReducedMotion();
    if (reduced) {
      setGlyph(icon, "…");
    } else {
      btn.classList.add("cf-refresh-busy");
    }

    function finish(ok: boolean, message: string): void {
      btn.classList.remove("cf-refresh-busy");
      if (reduced) setIcon(icon, REFRESH_SVG); // restore the arrow from "…"
      if (ok && !reduced) {
        // Success beat: arrow → check → arrow, then the button re-arms.
        setIcon(icon, CHECK_SVG);
        setTimeout(() => {
          setIcon(icon, REFRESH_SVG);
          rearm(btn);
        }, CHECK_MS);
        return;
      }
      if (!ok) {
        btn.title = `Refresh failed — ${message}`;
        if (!reduced) {
          btn.classList.add("cf-refresh-shake");
          setTimeout(() => {
            btn.classList.remove("cf-refresh-shake");
            rearm(btn);
          }, SHAKE_MS);
          return;
        }
      }
      rearm(btn);
    }

    void Promise.resolve().then(onRefresh).then(
      () => finish(true, ""),
      (err: unknown) => finish(false, err instanceof Error ? err.message : String(err)),
    );
  });

  return btn;
}

// fadeSwap swaps oldNode's content for newNode with a 150ms ease-out fade of
// the new content in. oldNode keeps its DOM identity (closures holding it —
// refresh buttons, the router — stay valid); because the swap is a single
// layout pass and the wrapper-less host keeps its box, there is no layout
// shift. newNode must be a single element; views wrap multi-node paints in a
// plain <div>, whose margins collapse through so spacing is unchanged.
export function fadeSwap(oldNode: HTMLElement, newNode: HTMLElement): void {
  oldNode.replaceChildren(newNode);
  newNode.classList.add("cf-fade-swap");
  const style = newNode.style;
  style.opacity = "0";
  void (newNode as HTMLElement).offsetWidth; // style flush so 0→1 transitions
  style.opacity = "1";
  setTimeout(() => {
    newNode.classList.remove("cf-fade-swap");
    style.opacity = "";
  }, FADE_MS + 50);
}
