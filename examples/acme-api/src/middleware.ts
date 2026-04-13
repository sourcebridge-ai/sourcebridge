import { NextRequest, NextResponse } from "next/server";
import { authenticate } from "@/api/middleware/auth";
import { checkRateLimit } from "@/api/middleware/rate-limit";

/**
 * Next.js Edge Middleware.
 * Runs on every API request to enforce authentication and rate limiting.
 */
export function middleware(req: NextRequest) {
  // Only apply to API routes
  if (!req.nextUrl.pathname.startsWith("/api")) {
    return NextResponse.next();
  }

  // Rate limit by user ID or IP
  const session = authenticate(req);
  const rateLimitKey = session?.sub ?? req.headers.get("x-forwarded-for") ?? "anonymous";

  try {
    checkRateLimit(rateLimitKey);
  } catch {
    return NextResponse.json(
      { error: { code: "RATE_LIMITED", message: "Too many requests" } },
      { status: 429 },
    );
  }

  const response = NextResponse.next();
  if (session) {
    response.headers.set("x-user-id", session.sub);
    response.headers.set("x-team-id", session.teamId);
  }

  return response;
}

export const config = {
  matcher: ["/api/:path*"],
};
