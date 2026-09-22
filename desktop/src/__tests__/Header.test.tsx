import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { Header, resolveInitialTheme } from "../components/Header";

describe("Header", () => {
  it("renders both health indicators with their online state", () => {
    render(<Header systemsOnline={true} cloudflareOnline={false} accounts={[]} />);
    expect(screen.getByTestId("health-systems")).toHaveAttribute("data-online", "true");
    expect(screen.getByTestId("health-cloudflare")).toHaveAttribute("data-online", "false");
  });

  it("renders the account switcher with the provided profiles", () => {
    render(
      <Header
        systemsOnline={true}
        cloudflareOnline={true}
        accounts={[{ name: "work" }, { name: "personal" }]}
      />
    );
    const select = screen.getByTestId("account-switcher") as HTMLSelectElement;
    expect(select.options.length).toBe(2);
    expect(select.options[0].textContent).toBe("work");
  });

  it("shows a static label when only one account exists", () => {
    render(
      <Header systemsOnline={true} cloudflareOnline={true} accounts={[{ name: "default" }]} />
    );
    expect(() => screen.getByTestId("account-switcher")).toThrow();
    expect(screen.getByTestId("account-label").textContent).toBe("default");
  });

  it("reflects a healthy state with both dots online", () => {
    render(<Header systemsOnline={true} cloudflareOnline={true} accounts={[]} />);
    expect(screen.getByTestId("health-systems")).toHaveAttribute("data-online", "true");
    expect(screen.getByTestId("health-cloudflare")).toHaveAttribute("data-online", "true");
  });

  // BUG-037: the online/offline state must be announced. `data-online` never
  // enters the accessibility tree and `title` never computes into the
  // accessible name, so the outer status span carries an aria-label.
  it("announces health state through aria-label (BUG-037)", () => {
    render(<Header systemsOnline={true} cloudflareOnline={false} accounts={[]} />);
    expect(screen.getByTestId("health-systems")).toHaveAttribute(
      "aria-label",
      "Systems: online"
    );
    expect(screen.getByTestId("health-cloudflare")).toHaveAttribute(
      "aria-label",
      "Cloudflare: offline"
    );
  });

  it("keeps a visible non-color label beside the dot mark", () => {
    render(<Header systemsOnline={true} cloudflareOnline={false} accounts={[]} />);
    expect(screen.getByTestId("health-systems")).toHaveTextContent("Systems");
    expect(screen.getByTestId("health-cloudflare")).toHaveTextContent("Cloudflare");
  });

  it("renders the brand as the page's level-one heading", () => {
    render(<Header systemsOnline={true} cloudflareOnline={true} accounts={[]} />);
    expect(
      screen.getByRole("heading", { level: 1, name: /cosmoflare/i })
    ).toBeInTheDocument();
  });
});

// --- theme toggle (FEAT-041) ---

describe("Header theme toggle (FEAT-041)", () => {
  afterEach(() => {
    localStorage.removeItem("cosmoflare-theme");
    delete document.documentElement.dataset.theme;
  });

  it("flips data-theme on the document root and persists the choice", () => {
    render(<Header systemsOnline={true} cloudflareOnline={true} accounts={[]} />);
    // Initial state (no stored preference, no matchMedia in jsdom): dark.
    expect(document.documentElement.dataset.theme).toBe("dark");

    fireEvent.click(screen.getByTestId("theme-toggle"));
    expect(document.documentElement.dataset.theme).toBe("light");
    expect(localStorage.getItem("cosmoflare-theme")).toBe("light");

    fireEvent.click(screen.getByTestId("theme-toggle"));
    expect(document.documentElement.dataset.theme).toBe("dark");
    expect(localStorage.getItem("cosmoflare-theme")).toBe("dark");
  });

  it("announces the target theme and exposes pressed state", () => {
    render(<Header systemsOnline={true} cloudflareOnline={true} accounts={[]} />);
    const toggle = screen.getByTestId("theme-toggle");
    expect(toggle).toHaveAttribute("aria-label", "Switch to light theme");
    expect(toggle).toHaveAttribute("aria-pressed", "false");
    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute("aria-label", "Switch to dark theme");
    expect(toggle).toHaveAttribute("aria-pressed", "true");
  });

  it("restores the stored theme on mount", () => {
    localStorage.setItem("cosmoflare-theme", "light");
    render(<Header systemsOnline={true} cloudflareOnline={true} accounts={[]} />);
    expect(document.documentElement.dataset.theme).toBe("light");
    expect(screen.getByTestId("theme-toggle")).toHaveAttribute(
      "aria-label",
      "Switch to dark theme"
    );
  });

  it("resolveInitialTheme ignores invalid stored values", () => {
    localStorage.setItem("cosmoflare-theme", "neon");
    expect(resolveInitialTheme()).toBe("dark");
  });
});
