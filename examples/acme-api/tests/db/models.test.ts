import { describe, it, expect } from "vitest";
import { CreateUserInput, UpdateUserInput, UserRole } from "@/db/models/user";
import { CreateTeamInput } from "@/db/models/team";
import { CreateInvitationInput, invitationExpiresAt } from "@/db/models/invitation";

describe("User schemas", () => {
  it("validates a valid user creation input", () => {
    const result = CreateUserInput.safeParse({
      email: "test@example.com",
      name: "Test User",
      password: "securepass123",
    });
    expect(result.success).toBe(true);
  });

  it("rejects invalid email", () => {
    const result = CreateUserInput.safeParse({
      email: "not-an-email",
      name: "Test",
      password: "securepass123",
    });
    expect(result.success).toBe(false);
  });

  it("rejects short password", () => {
    const result = CreateUserInput.safeParse({
      email: "test@example.com",
      name: "Test",
      password: "short",
    });
    expect(result.success).toBe(false);
  });

  it("validates update input with optional fields", () => {
    expect(UpdateUserInput.safeParse({}).success).toBe(true);
    expect(UpdateUserInput.safeParse({ name: "New Name" }).success).toBe(true);
  });
});

describe("Team schemas", () => {
  it("validates a valid team creation input", () => {
    const result = CreateTeamInput.safeParse({
      name: "My Team",
      slug: "my-team",
    });
    expect(result.success).toBe(true);
  });

  it("rejects invalid slug characters", () => {
    const result = CreateTeamInput.safeParse({
      name: "My Team",
      slug: "My Team!",
    });
    expect(result.success).toBe(false);
  });
});

describe("Invitation schemas", () => {
  it("validates a valid invitation input", () => {
    const result = CreateInvitationInput.safeParse({
      email: "invite@example.com",
      role: "member",
    });
    expect(result.success).toBe(true);
  });

  it("rejects owner role in invitations", () => {
    const result = CreateInvitationInput.safeParse({
      email: "invite@example.com",
      role: "owner",
    });
    expect(result.success).toBe(false);
  });

  it("generates a future expiration date", () => {
    const expires = new Date(invitationExpiresAt());
    expect(expires.getTime()).toBeGreaterThan(Date.now());
  });
});
