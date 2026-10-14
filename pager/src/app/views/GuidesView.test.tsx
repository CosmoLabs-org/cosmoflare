// @vitest-environment jsdom
// GuidesView (FEAT-pDQEFQV + FEAT-pAGY842): the two article cards, article
// navigation to "#/guides/permissions" (title + a kv term visible), and
// the quiet not-found card for an unknown id. No network — the data
// module is static.
import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { GUIDES } from "../../guides";
import GuidesView from "./GuidesView";

afterEach(() => {
  cleanup();
  window.location.hash = "";
});

/** Show a hash and let the view's hashchange listener react. */
function showHash(hash: string): void {
  window.location.hash = hash;
  act(() => {
    window.dispatchEvent(new Event("hashchange"));
  });
}

/** The first kv term in article 1's body — the article view must surface
 *  the cf-domain-field rows. */
function firstKvTerm(guide: (typeof GUIDES)[number]): string {
  for (const block of guide.body) {
    if (block.kind === "kv") return block.term;
  }
  return "";
}

describe("GuidesView", () => {
  it("renders the two article cards", () => {
    render(<GuidesView />);
    for (const guide of GUIDES) {
      expect(screen.getByText(guide.title)).toBeTruthy();
      // Both articles share the updated date — assert presence, not uniqueness.
      expect(screen.getAllByText(`Updated ${guide.updated}`).length).toBeGreaterThan(0);
    }
  });

  it("navigating to article 1 renders its title and a kv term", () => {
    render(<GuidesView />);
    showHash("#/guides/permissions");
    expect(screen.getByText(GUIDES[0].title)).toBeTruthy();
    const term = firstKvTerm(GUIDES[0]);
    expect(term.length).toBeGreaterThan(0);
    expect(screen.getByText(term)).toBeTruthy();
  });

  it("unknown id renders the quiet not-found card", () => {
    render(<GuidesView />);
    showHash("#/guides/no-such-guide");
    expect(screen.getByText("No guide with that id")).toBeTruthy();
    expect(screen.queryByText(GUIDES[0].title)).toBeNull();
  });
});
