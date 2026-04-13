import { describe, it, expect } from "vitest";
import {
  AppError,
  AuthenticationError,
  AuthorizationError,
  NotFoundError,
  RateLimitError,
  ValidationError,
  errorResponse,
} from "@/lib/errors";

describe("Error classes", () => {
  it("creates an AppError with code and status", () => {
    const err = new AppError("test", "TEST_ERR", 400);
    expect(err.message).toBe("test");
    expect(err.code).toBe("TEST_ERR");
    expect(err.status).toBe(400);
  });

  it("AuthenticationError defaults to 401", () => {
    const err = new AuthenticationError();
    expect(err.status).toBe(401);
    expect(err.code).toBe("UNAUTHENTICATED");
  });

  it("AuthorizationError defaults to 403", () => {
    const err = new AuthorizationError();
    expect(err.status).toBe(403);
    expect(err.code).toBe("FORBIDDEN");
  });

  it("NotFoundError includes resource name", () => {
    const err = new NotFoundError("Team");
    expect(err.message).toBe("Team not found");
    expect(err.status).toBe(404);
  });

  it("RateLimitError includes retry info", () => {
    const err = new RateLimitError(30);
    expect(err.message).toContain("30s");
    expect(err.status).toBe(429);
  });

  it("ValidationError includes field details", () => {
    const err = new ValidationError("Bad input", { email: "required" });
    expect(err.fields.email).toBe("required");
    expect(err.status).toBe(422);
  });
});

describe("errorResponse", () => {
  it("returns structured JSON for AppError", async () => {
    const res = errorResponse(new NotFoundError("User"));
    expect(res.status).toBe(404);
    const body = await res.json();
    expect(body.error.code).toBe("NOT_FOUND");
  });

  it("returns 500 for unknown errors", async () => {
    const res = errorResponse(new Error("boom"));
    expect(res.status).toBe(500);
    const body = await res.json();
    expect(body.error.code).toBe("INTERNAL");
  });
});
