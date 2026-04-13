import { NextRequest, NextResponse } from "next/server";
import { verifyToken, type TokenPayload } from "@/auth/jwt";

const PUBLIC_PATHS = [
  "/api/auth/sign-in",
  "/api/auth/sign-up",
  "/api/auth/magic-link",
  "/api/webhooks/stripe",
  "/api/health",
];

/**
 * Extracts and verifies the JWT from the Authorization header or session cookie.
 * Returns null for public paths. Rejects with 401 for invalid tokens.
 */
export function authenticate(req: NextRequest): TokenPayload | null {
  const path = req.nextUrl.pathname;
  if (PUBLIC_PATHS.some((p) => path.startsWith(p))) return null;

  const authHeader = req.headers.get("authorization");
  const token = authHeader?.startsWith("Bearer ")
    ? authHeader.slice(7)
    : req.cookies.get("acme-session")?.value;

  if (!token) return null;

  try {
    return verifyToken(token);
  } catch {
    return null;
  }
}

export function requireAuth(req: NextRequest): TokenPayload {
  const session = authenticate(req);
  if (!session) {
    throw new Response(
      JSON.stringify({ error: { code: "UNAUTHENTICATED", message: "Authentication required" } }),
      { status: 401, headers: { "Content-Type": "application/json" } },
    );
  }
  return session;
}
