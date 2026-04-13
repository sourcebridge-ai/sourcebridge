import { NextRequest } from "next/server";
import { z } from "zod";
import {
  createCheckoutSession,
  handleWebhook,
  getSubscriptionStatus,
} from "@/services/billing-service";
import { requireAuth } from "@/api/middleware/auth";
import { validateBody } from "@/api/middleware/validate";
import { errorResponse } from "@/lib/errors";
import { env } from "@/lib/env";

const CheckoutInput = z.object({
  teamId: z.string().uuid(),
  interval: z.enum(["monthly", "yearly"]),
});

export async function handleCreateCheckout(req: NextRequest) {
  try {
    const session = requireAuth(req);
    const input = await validateBody(req, CheckoutInput);
    const returnUrl = `${env().NEXT_PUBLIC_APP_URL}/settings/billing`;
    const url = await createCheckoutSession(
      input.teamId,
      input.interval,
      returnUrl,
    );
    return Response.json({ url });
  } catch (error) {
    return errorResponse(error);
  }
}

export async function handleStripeWebhook(req: NextRequest) {
  try {
    const payload = await req.text();
    const signature = req.headers.get("stripe-signature");
    if (!signature) {
      return Response.json(
        { error: { code: "MISSING_SIGNATURE", message: "Missing Stripe signature" } },
        { status: 400 },
      );
    }
    await handleWebhook(payload, signature);
    return Response.json({ received: true });
  } catch (error) {
    return errorResponse(error);
  }
}

export async function handleGetSubscription(
  req: NextRequest,
  teamId: string,
) {
  try {
    requireAuth(req);
    const status = await getSubscriptionStatus(teamId);
    return Response.json(status);
  } catch (error) {
    return errorResponse(error);
  }
}
