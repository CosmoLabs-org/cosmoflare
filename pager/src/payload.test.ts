import { describe, expect, it } from "vitest";
import { parsePayload, SEVERITIES, type Payload } from "./payload";

const validEnvelope: Record<string, unknown> = {
  id: "alert-123",
  severity: "warning",
  service: "api-gateway",
  title: "Error rate high",
  detail: "5xx rate above 5% for 10m",
  fired_at: "2026-10-07T13:20:00Z",
};

// omit returns a shallow copy of env without key (avoids unused-var
// destructuring under noUnusedLocals).
function omit(env: Record<string, unknown>, key: string): Record<string, unknown> {
  const copy = { ...env };
  delete copy[key];
  return copy;
}

describe("parsePayload", () => {
  it("parses a valid envelope into the exact wire schema", () => {
    const p = parsePayload(validEnvelope);
    const want: Payload = {
      id: "alert-123",
      severity: "warning",
      service: "api-gateway",
      title: "Error rate high",
      detail: "5xx rate above 5% for 10m",
      fired_at: "2026-10-07T13:20:00Z",
    };
    expect(p).toEqual(want);
    expect(Object.keys(p as object).sort()).toEqual(
      ["detail", "fired_at", "id", "severity", "service", "title"].sort()
    );
  });

  it("parses every severity in the vocabulary", () => {
    for (const severity of SEVERITIES) {
      expect(parsePayload({ ...validEnvelope, severity })).toMatchObject({ severity });
    }
  });

  it("rejects a severity outside the vocabulary", () => {
    expect(parsePayload({ ...validEnvelope, severity: "bogus" })).toBeNull();
    expect(parsePayload({ ...validEnvelope, severity: "" })).toBeNull();
    expect(parsePayload({ ...validEnvelope, severity: 3 })).toBeNull();
  });

  it("rejects a missing or empty id", () => {
    expect(parsePayload(omit(validEnvelope, "id"))).toBeNull();
    expect(parsePayload({ ...validEnvelope, id: "" })).toBeNull();
  });

  it("rejects a missing or empty service", () => {
    expect(parsePayload(omit(validEnvelope, "service"))).toBeNull();
    expect(parsePayload({ ...validEnvelope, service: "" })).toBeNull();
  });

  it("rejects a missing or empty title", () => {
    expect(parsePayload(omit(validEnvelope, "title"))).toBeNull();
    expect(parsePayload({ ...validEnvelope, title: "" })).toBeNull();
  });

  it("accepts a missing or empty detail (optional in the Go schema)", () => {
    expect(parsePayload(omit(validEnvelope, "detail"))).toMatchObject({ detail: "" });
    expect(parsePayload({ ...validEnvelope, detail: "" })).toMatchObject({ detail: "" });
    expect(parsePayload({ ...validEnvelope, detail: 42 })).toBeNull();
  });

  it("accepts a non-RFC3339-but-parseable fired_at and rejects garbage", () => {
    expect(parsePayload({ ...validEnvelope, fired_at: "not-a-date" })).toBeNull();
    expect(parsePayload({ ...validEnvelope, fired_at: 1728300000 })).toBeNull();
    expect(parsePayload(omit(validEnvelope, "fired_at"))).toBeNull();
  });

  it("rejects non-object envelopes", () => {
    expect(parsePayload(null)).toBeNull();
    expect(parsePayload(undefined)).toBeNull();
    expect(parsePayload("alert-123")).toBeNull();
    expect(parsePayload(42)).toBeNull();
    expect(parsePayload([validEnvelope])).toBeNull();
  });
});
