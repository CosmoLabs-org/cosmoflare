import { describe, expect, it } from "vitest";
import { urlBase64ToUint8Array } from "./vapid";

describe("urlBase64ToUint8Array", () => {
  it("decodes an unpadded base64url VAPID key (65-byte uncompressed P-256 point)", () => {
    // 65 bytes: 0x04 followed by 64 x 0xAB, base64url without padding
    const bytes = new Uint8Array(65).fill(0xab);
    bytes[0] = 0x04;
    const b64url = btoa(String.fromCharCode(...bytes)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
    const out = urlBase64ToUint8Array(b64url);
    expect(out.length).toBe(65);
    expect(out[0]).toBe(0x04);
    expect(out[64]).toBe(0xab);
  });
  it("handles - and _ characters", () => {
    expect(Array.from(urlBase64ToUint8Array("-_8"))).toEqual([0xfb, 0xff]);
  });
});
