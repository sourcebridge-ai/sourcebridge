import jwt from "jsonwebtoken";
import { env } from "@/lib/env";

export interface TokenPayload {
  sub: string;
  email: string;
  teamId: string;
  role: "owner" | "admin" | "member";
  iat: number;
  exp: number;
}

const TOKEN_TTL = "7d";

export function signToken(payload: Omit<TokenPayload, "iat" | "exp">): string {
  return jwt.sign(payload, env().JWT_SECRET, { expiresIn: TOKEN_TTL });
}

export function verifyToken(token: string): TokenPayload {
  return jwt.verify(token, env().JWT_SECRET) as TokenPayload;
}

export function decodeToken(token: string): TokenPayload | null {
  const decoded = jwt.decode(token);
  if (!decoded || typeof decoded === "string") return null;
  return decoded as TokenPayload;
}
