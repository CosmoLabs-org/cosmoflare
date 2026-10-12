// Email Routing payload types (FEAT-059, ROAD-107) — mirrors
// apps/ops/src/email.ts's collector shapes plus the cache envelope the
// worker's /api/email endpoint wraps around them (index.ts cached()). The
// collector is operator-gated (Email Routing Read): an errors-only payload
// means the token lacks the scope, and the overview must stay unchanged.

export interface EmailZoneRow {
  zone: string;
  enabled: boolean;
  name?: string;
}

export interface EmailRoutingPayload {
  generatedAt: string;
  zones: EmailZoneRow[];
  errors: string[];
  cache?: { ageSec: number; stale: boolean };
}
