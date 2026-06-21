import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { Dashboard, type DashboardClient } from "../views/Dashboard";

function withClient(ui: ReactNode) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 0 } },
  });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

const stubClient: DashboardClient = {
  async get<T>(path: string): Promise<T> {
    let data: unknown;
    switch (path) {
      case "/zones":
        data = [{ name: "a.com" }, { name: "b.com" }];
        break;
      case "/r2/buckets":
        data = [{ name: "media" }];
        break;
      case "/workers":
        data = [{ id: "w1" }];
        break;
      case "/kv":
        data = [{ id: "ns1" }, { id: "ns2" }, { id: "ns3" }];
        break;
      default:
        data = [];
    }
    return data as T;
  },
};

describe("Dashboard", () => {
  it("renders a count card for each service", async () => {
    withClient(<Dashboard client={stubClient} profile="" />);
    await waitFor(() => {
      expect(screen.getByTestId("card-zones")).toHaveTextContent("2");
      expect(screen.getByTestId("card-r2")).toHaveTextContent("1");
      expect(screen.getByTestId("card-workers")).toHaveTextContent("1");
      expect(screen.getByTestId("card-kv")).toHaveTextContent("3");
    });
  });

  it("passes the selected profile to every request", async () => {
    const seen: string[] = [];
    const tracing: DashboardClient = {
      async get<T>(path: string, params?: Record<string, string>): Promise<T> {
        seen.push(`${path}:${params?.profile ?? ""}`);
        return [] as unknown as T;
      },
    };
    withClient(<Dashboard client={tracing} profile="work" />);
    await screen.findAllByTestId(/^card-/);
    expect(seen.length).toBe(4); // one per service
    expect(seen.every((s) => s.endsWith(":work"))).toBe(true);
  });

  it("shows a loading state before data arrives", () => {
    const slow: DashboardClient = {
      // Never resolves — cards stay in the loading state.
      get: <T,>() => new Promise<T>(() => {}),
    };
    withClient(<Dashboard client={slow} profile="" />);
    expect(
      screen.getAllByTestId(/^card-/).every((c) => /loading/i.test(c.textContent ?? ""))
    ).toBe(true);
  });
});
