import { describe, it, expect, beforeEach } from "vitest";
import { checkRateLimit, cleanupExpiredEntries } from "@/api/middleware/rate-limit";

describe("Rate limiter", () => {
  beforeEach(() => {
    cleanupExpiredEntries();
  });

  it("allows requests under the limit", () => {
    for (let i = 0; i < 100; i++) {
      expect(() => checkRateLimit("user-1")).not.toThrow();
    }
  });

  it("blocks requests over the limit", () => {
    for (let i = 0; i < 100; i++) {
      checkRateLimit("user-2");
    }
    expect(() => checkRateLimit("user-2")).toThrow(/Rate limit exceeded/);
  });

  it("tracks limits per key independently", () => {
    for (let i = 0; i < 100; i++) {
      checkRateLimit("user-3");
    }
    // Different user should not be affected
    expect(() => checkRateLimit("user-4")).not.toThrow();
  });
});
