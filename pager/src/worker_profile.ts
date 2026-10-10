// Worker profile helpers (UI-3, operator request 2026-10-10): the pure
// formatting the React WorkersView/WorkerProfileView share. The imperative
// vanilla renderer that used to live here was retired (TASK-pD575FN);
// pager/src/app/views/WorkerProfileView.tsx is the only render path now.

/** "12.5ms" from a CPU percentile, "—" when the analytics row has no data.
 *  Exported (P-05a) so the React WorkersView/WorkerProfileView format CPU
 *  percentiles identically — never re-implemented. */
export function cpuMs(v: number | null): string {
  return v === null ? "—" : `${v.toFixed(1)}ms`;
}
