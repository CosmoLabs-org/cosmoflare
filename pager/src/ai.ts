// AI Gateway payload types (BR-03, ROAD-112 phase 2) — mirrors
// apps/ops/src/ai_gateway.ts's collector shapes plus the cache envelope the
// worker's /api/ai endpoint wraps around them (index.ts cached()). The
// dollar figures are gateway-side estimates (costBasis "gateway-estimate"):
// views must label every dollar value "est." — never presented as real bill
// numbers (real-numbers rule).

export interface AIModelRow {
  model: string;
  provider: string;
  requests: number;
  tokensIn?: number;
  tokensOut?: number;
  costUsd?: number;
}

export interface AIGatewayRow {
  id: string;
  name: string;
  requests24h: number;
  requestsMtd: number;
  models: AIModelRow[];
}

export interface AIGatewaysPayload {
  generatedAt: string;
  gateways: AIGatewayRow[];
  costBasis: "gateway-estimate";
  notes: string[];
  errors: string[];
  cache?: { ageSec: number; stale: boolean };
}
