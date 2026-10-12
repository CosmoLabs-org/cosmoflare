import { describe, expect, it } from "vitest";
import { formatAgeText, periodMarkup } from "./statusstrip";

// UI-1 (operator request 2026-10-10): the bottom status strip's pure
// formatting helpers — honest "updated X ago" and the billing-period day
// counter (IMP-pSZEMZP, operator request 2026-10-12).

describe("formatAgeText", () => {
  it("says just now under a minute", () => {
    expect(formatAgeText(0)).toBe("just now");
    expect(formatAgeText(59)).toBe("just now");
  });

  it("switches to minutes at 60s", () => {
    expect(formatAgeText(60)).toBe("1m ago");
  });

  it("stays in minutes under an hour", () => {
    expect(formatAgeText(3599)).toBe("59m ago");
  });

  it("switches to hours at 3600s", () => {
    expect(formatAgeText(3600)).toBe("1h ago");
  });

  it("switches to days at 86400s", () => {
    expect(formatAgeText(86400)).toBe("1d ago");
  });
});

describe("periodMarkup", () => {
  it("returns null when the period is unknown", () => {
    expect(periodMarkup()).toBeNull();
    expect(periodMarkup(undefined, 30)).toBeNull();
    expect(periodMarkup(3, undefined)).toBeNull();
  });

  it("renders the 24/30 counter lockup", () => {
    expect(periodMarkup(3, 30)).toBe(
      "<span class='cf-cycle-day'>3</span><span class='cf-cycle-total'>/30</span>",
    );
    expect(periodMarkup(24, 30)).toBe(
      "<span class='cf-cycle-day'>24</span><span class='cf-cycle-total'>/30</span>",
    );
  });
});
