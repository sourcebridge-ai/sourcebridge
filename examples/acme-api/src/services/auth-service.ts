import { findUserByEmail, createUser } from "@/db/queries/users";
import { verifyPassword } from "@/auth/password";
import { signToken } from "@/auth/jwt";
import { setSession, clearSession } from "@/auth/session";
import { createMagicLink, consumeMagicLink } from "@/auth/magic-link";
import { sendMagicLinkEmail } from "./email-service";
import { listUserTeams } from "@/db/queries/teams";
import { AppError, AuthenticationError } from "@/lib/errors";
import type { CreateUserInput } from "@/db/models/user";

export async function signUp(input: CreateUserInput) {
  const existing = await findUserByEmail(input.email);
  if (existing) {
    throw new AppError("Email already registered", "EMAIL_EXISTS", 409);
  }

  const user = await createUser(input);
  const teams = await listUserTeams(user.id);
  const token = signToken({
    sub: user.id,
    email: user.email,
    teamId: teams[0]?.id ?? "",
    role: "owner",
  });

  await setSession(token);
  return { user, token };
}

export async function signIn(email: string, password: string) {
  const user = await findUserByEmail(email);
  if (!user) throw new AuthenticationError("Invalid credentials");

  const valid = await verifyPassword(password, user.password_hash);
  if (!valid) throw new AuthenticationError("Invalid credentials");

  const teams = await listUserTeams(user.id);
  const firstTeam = teams[0];

  const token = signToken({
    sub: user.id,
    email: user.email,
    teamId: firstTeam?.id ?? "",
    role: "owner",
  });

  await setSession(token);
  return { user, token };
}

export async function signOut() {
  await clearSession();
}

export async function requestMagicLink(email: string) {
  const token = await createMagicLink(email);
  await sendMagicLinkEmail(email, token);
}

export async function redeemMagicLink(token: string) {
  const email = await consumeMagicLink(token);
  if (!email) throw new AuthenticationError("Invalid or expired magic link");

  let user = await findUserByEmail(email);
  if (!user) {
    user = await createUser({ email, name: email.split("@")[0], password: "" });
  }

  const teams = await listUserTeams(user.id);
  const sessionToken = signToken({
    sub: user.id,
    email: user.email,
    teamId: teams[0]?.id ?? "",
    role: "owner",
  });

  await setSession(sessionToken);
  return { user, token: sessionToken };
}
