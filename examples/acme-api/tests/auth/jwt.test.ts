import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock env before importing jwt module
vi.mock("@/lib/env", () => ({
  env: () => ({
    JWT_SECRET: "test-secret-at-least-sixteen-chars",
  }),
}));

import { signToken, verifyToken, decodeToken } from "@/auth/jwt";

describe("JWT", () => {
  const payload = {
    sub: "user-123",
    email: "alice@acme.dev",
    teamId: "team-456",
    role: "owner" as const,
  };

  it("signs and verifies a valid token", () => {
    const token = signToken(payload);
    const decoded = verifyToken(token);

    expect(decoded.sub).toBe("user-123");
    expect(decoded.email).toBe("alice@acme.dev");
    expect(decoded.teamId).toBe("team-456");
    expect(decoded.role).toBe("owner");
    expect(decoded.exp).toBeGreaterThan(Date.now() / 1000);
  });

  it("rejects a tampered token", () => {
    const token = signToken(payload);
    const tampered = token.slice(0, -5) + "XXXXX";

    expect(() => verifyToken(tampered)).toThrow();
  });

  it("decodes a token without verification", () => {
    const token = signToken(payload);
    const decoded = decodeToken(token);

    expect(decoded?.sub).toBe("user-123");
    expect(decoded?.email).toBe("alice@acme.dev");
  });

  it("returns null for an invalid token string", () => {
    expect(decodeToken("not-a-jwt")).toBeNull();
  });
});
