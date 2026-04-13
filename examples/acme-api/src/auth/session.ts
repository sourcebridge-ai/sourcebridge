import { cookies } from "next/headers";
import { verifyToken, type TokenPayload } from "./jwt";
import { AuthenticationError } from "@/lib/errors";

const SESSION_COOKIE = "acme-session";
const SESSION_MAX_AGE = 7 * 24 * 60 * 60; // 7 days (REQ-AUTH-002)

export async function getSession(): Promise<TokenPayload> {
  const cookieStore = await cookies();
  const token = cookieStore.get(SESSION_COOKIE)?.value;
  if (!token) throw new AuthenticationError("No session cookie");

  try {
    return verifyToken(token);
  } catch {
    throw new AuthenticationError("Invalid or expired session");
  }
}

export async function setSession(token: string): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.set(SESSION_COOKIE, token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    maxAge: SESSION_MAX_AGE,
    path: "/",
  });
}

export async function clearSession(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete(SESSION_COOKIE);
}

export async function optionalSession(): Promise<TokenPayload | null> {
  try {
    return await getSession();
  } catch {
    return null;
  }
}
