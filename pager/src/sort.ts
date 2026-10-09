// Sort comparators for the sortable list views (FEAT-052). Pure functions —
// no DOM.

export type SortDir = "asc" | "desc";

// cmpRows builds a row comparator from a numeric extractor. Null/undefined
// values always sort last regardless of direction (a dash renders in their
// place, and "no data" should never masquerade as the best or worst value).
export function cmpRows<T>(
  value: (row: T) => number | null | undefined,
  dir: SortDir,
): (a: T, b: T) => number {
  const sign = dir === "asc" ? 1 : -1;
  return (a, b) => {
    const va = value(a);
    const vb = value(b);
    if (va == null && vb == null) return 0;
    if (va == null) return 1;
    if (vb == null) return -1;
    return (va - vb) * sign;
  };
}

// cmpText sorts rows by a string extractor, ascending or descending, with
// locale-aware ordering.
export function cmpText<T>(value: (row: T) => string, dir: SortDir): (a: T, b: T) => number {
  const sign = dir === "asc" ? 1 : -1;
  return (a, b) => value(a).localeCompare(value(b)) * sign;
}
