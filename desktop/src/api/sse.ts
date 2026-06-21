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
  onEvent: (e: DaemonEvent) => void,
  onConnectionChange?: (connected: boolean) => void
): void {
  const cb = useRef(onEvent);
  cb.current = onEvent;
  const connCb = useRef(onConnectionChange);
  connCb.current = onConnectionChange;

  useEffect(() => {
    if (!endpoint) return;
    const url = new URL(endpoint.url + "/events");
    url.searchParams.set("token", endpoint.token);
    const es = new EventSource(url.toString());

    // Surface transport state so the caller can drive the "systems online"
    // health tier (open stream = daemon up and answering; error = unreachable).
    // EventSource natively auto-reconnects on transport drops, reusing the same
    // token. A token *change* (after a daemon respawn) is handled one level up:
    // the caller resolves a fresh endpoint and passes it in, which re-runs this
    // effect — tearing down the stale stream and opening one with the new token.
    es.onopen = () => connCb.current?.(true);
    es.onerror = () => connCb.current?.(false);

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
      es.onopen = null;
      es.onerror = null;
      for (const [channel, handler] of handlers) {
        es.removeEventListener(channel, handler as EventListener);
      }
      es.close();
    };
  }, [endpoint]);
}
