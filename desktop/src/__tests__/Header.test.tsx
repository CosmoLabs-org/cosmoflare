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
});
