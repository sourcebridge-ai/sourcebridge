import { describe, it, expect } from "vitest";
import { hashPassword, verifyPassword } from "@/auth/password";

describe("Password hashing", () => {
  it("hashes a password and verifies it", async () => {
    const plain = "my-secure-password";
    const hashed = await hashPassword(plain);

    expect(hashed).not.toBe(plain);
    expect(await verifyPassword(plain, hashed)).toBe(true);
  });

  it("rejects an incorrect password", async () => {
    const hashed = await hashPassword("correct-password");
    expect(await verifyPassword("wrong-password", hashed)).toBe(false);
  });

  it("produces different hashes for the same input", async () => {
    const hash1 = await hashPassword("same-password");
    const hash2 = await hashPassword("same-password");
    expect(hash1).not.toBe(hash2); // different salts
  });
});
