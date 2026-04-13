import { describe, it, expect } from "vitest";
import { z } from "zod";
import { validateBody } from "@/api/middleware/validate";

const TestSchema = z.object({
  name: z.string().min(1),
  age: z.number().int().positive(),
});

function makeRequest(body: unknown): Request {
  return new Request("http://localhost/test", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

describe("validateBody", () => {
  it("parses a valid body", async () => {
    const req = makeRequest({ name: "Alice", age: 30 });
    const result = await validateBody(req, TestSchema);

    expect(result.name).toBe("Alice");
    expect(result.age).toBe(30);
  });

  it("rejects an invalid body with field errors", async () => {
    const req = makeRequest({ name: "", age: -1 });

    await expect(validateBody(req, TestSchema)).rejects.toThrow("Validation failed");
  });

  it("rejects non-JSON body", async () => {
    const req = new Request("http://localhost/test", {
      method: "POST",
      body: "not json",
    });

    await expect(validateBody(req, TestSchema)).rejects.toThrow("Invalid JSON");
  });
});
