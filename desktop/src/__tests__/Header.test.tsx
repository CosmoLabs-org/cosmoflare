import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Header } from "../components/Header";

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
