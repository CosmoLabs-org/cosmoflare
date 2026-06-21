// SSE client hook for the daemon's multiplexed /events stream. Uses the native
// EventSource, which auto-reconnects on disconnect (the plan's
// "SSE disconnect -> client auto-reconnects with backoff" requirement). The
// token rides the query string because EventSource cannot set headers; the
// daemon's authMiddleware accepts ?token= for this (localhost desktop only).

import { useEffect, useRef } from "react";
import type { DaemonEndpoint } from "./client";

export type DaemonChannel = "metrics" | "notifications" | "status";

export interface DaemonEvent<T = unknown> {
  channel: DaemonChannel;
  data: T;
}

export interface StatusPayload {
  systems_online?: boolean;
  cloudflare_online?: boolean;
}

const CHANNELS: DaemonChannel[] = ["metrics", "notifications", "status"];

/**
 * Subscribe to the daemon SSE stream. `onEvent` is kept in a ref so the effect
 * does not resubscribe on every render. Pass `endpoint=null` before the daemon
 * is ready; the hook no-ops until an endpoint appears.
 */
export function useDaemonSSE(
  endpoint: DaemonEndpoint | null,
  onEvent: (e: DaemonEvent) => void
): void {
  const cb = useRef(onEvent);
  cb.current = onEvent;

  useEffect(() => {
    if (!endpoint) return;
    const url = new URL(endpoint.url + "/events");
    url.searchParams.set("token", endpoint.token);
    const es = new EventSource(url.toString());

    const handlers: Array<[DaemonChannel, (e: MessageEvent) => void]> = [];
    for (const channel of CHANNELS) {
      const handler = (e: MessageEvent) => {
        let data: unknown;
        try {
          data = JSON.parse(e.data);
        } catch {
          return; // ignore malformed frame
        }
        cb.current({ channel, data });
      };
      es.addEventListener(channel, handler as EventListener);
      handlers.push([channel, handler]);
    }

    return () => {
      for (const [channel, handler] of handlers) {
        es.removeEventListener(channel, handler as EventListener);
      }
      es.close();
    };
  }, [endpoint]);
}
