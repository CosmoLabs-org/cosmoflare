// Durable Objects payload types (FEAT-p3KJAC2): every namespace with its 24h
// usage — requests, errors, error rate. The imperative vanilla renderer that
// used to live here was retired (TASK-pD575FN);
// pager/src/app/views/DurableObjectsView.tsx is the only render path now.

export interface DORow {
  namespaceId: string;
  script: string;
  requests: number;
  errors: number;
  errorPct: number;
}

export interface DOPayload {
  generatedAt: string;
  namespaces: number;
  rows: DORow[];
  errors: string[];
  cache?: { ageSec: number; stale: boolean };
}
