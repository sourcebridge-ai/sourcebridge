import { NextRequest } from "next/server";
import { z } from "zod";
import { signUp, signIn, signOut, requestMagicLink, redeemMagicLink } from "@/services/auth-service";
import { validateBody } from "@/api/middleware/validate";
import { errorResponse } from "@/lib/errors";

const SignInInput = z.object({
  email: z.string().email(),
  password: z.string().min(1),
});

const SignUpInput = z.object({
  email: z.string().email(),
  name: z.string().min(1).max(100),
  password: z.string().min(8).max(128),
});

const MagicLinkInput = z.object({
  email: z.string().email(),
});

export async function handleSignIn(req: NextRequest) {
  try {
    const input = await validateBody(req, SignInInput);
    const result = await signIn(input.email, input.password);
    return Response.json({
      user: { id: result.user.id, email: result.user.email, name: result.user.name },
      token: result.token,
    });
  } catch (error) {
    return errorResponse(error);
  }
}

export async function handleSignUp(req: NextRequest) {
  try {
    const input = await validateBody(req, SignUpInput);
    const result = await signUp(input);
    return Response.json(
      { user: { id: result.user.id, email: result.user.email, name: result.user.name }, token: result.token },
      { status: 201 },
    );
  } catch (error) {
    return errorResponse(error);
  }
}

export async function handleSignOut() {
  try {
    await signOut();
    return Response.json({ ok: true });
  } catch (error) {
    return errorResponse(error);
  }
}

export async function handleRequestMagicLink(req: NextRequest) {
  try {
    const input = await validateBody(req, MagicLinkInput);
    await requestMagicLink(input.email);
    return Response.json({ ok: true });
  } catch (error) {
    return errorResponse(error);
  }
}

export async function handleRedeemMagicLink(req: NextRequest) {
  try {
    const token = req.nextUrl.searchParams.get("token");
    if (!token) return Response.json({ error: { code: "MISSING_TOKEN", message: "Token required" } }, { status: 400 });
    const result = await redeemMagicLink(token);
    return Response.json({
      user: { id: result.user.id, email: result.user.email, name: result.user.name },
      token: result.token,
    });
  } catch (error) {
    return errorResponse(error);
  }
}
