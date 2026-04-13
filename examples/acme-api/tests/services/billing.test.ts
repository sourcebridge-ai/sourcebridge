import { describe, it, expect } from "vitest";
import { PLAN_LIMITS } from "@/db/models/team";

describe("Plan limits", () => {
  it("defines free plan limits", () => {
    expect(PLAN_LIMITS.free.usage).toBe(100);
    expect(PLAN_LIMITS.free.members).toBe(3);
  });

  it("defines pro plan limits", () => {
    expect(PLAN_LIMITS.pro.usage).toBe(10_000);
    expect(PLAN_LIMITS.pro.members).toBe(25);
  });

  it("defines enterprise plan as unlimited", () => {
    expect(PLAN_LIMITS.enterprise.usage).toBe(Infinity);
    expect(PLAN_LIMITS.enterprise.members).toBe(Infinity);
  });

  it("has all expected tiers", () => {
    const tiers = Object.keys(PLAN_LIMITS);
    expect(tiers).toEqual(["free", "pro", "enterprise"]);
  });
});
