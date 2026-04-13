import Stripe from "stripe";
import { env } from "@/lib/env";
import { findTeamById } from "@/db/queries/teams";
import { createAdminClient } from "@/lib/supabase/server";
import { NotFoundError, AppError } from "@/lib/errors";
import type { PlanTier } from "@/db/models/team";
import { PLAN_LIMITS } from "@/db/models/team";

let stripeClient: Stripe | null = null;

function stripe(): Stripe {
  if (!stripeClient) {
    stripeClient = new Stripe(env().STRIPE_SECRET_KEY, {
      apiVersion: "2025-03-31.basil",
    });
  }
  return stripeClient;
}

export async function createCheckoutSession(
  teamId: string,
  interval: "monthly" | "yearly",
  returnUrl: string,
): Promise<string> {
  const team = await findTeamById(teamId);
  if (!team) throw new NotFoundError("Team");

  const priceId =
    interval === "monthly"
      ? env().STRIPE_PRICE_PRO_MONTHLY
      : env().STRIPE_PRICE_PRO_YEARLY;

  let customerId = team.stripe_customer_id;
  if (!customerId) {
    const customer = await stripe().customers.create({
      metadata: { teamId: team.id, teamSlug: team.slug },
    });
    customerId = customer.id;

    const supabase = await createAdminClient();
    await supabase
      .from("teams")
      .update({ stripe_customer_id: customerId })
      .eq("id", teamId);
  }

  const session = await stripe().checkout.sessions.create({
    customer: customerId,
    mode: "subscription",
    line_items: [{ price: priceId, quantity: 1 }],
    success_url: `${returnUrl}?checkout=success`,
    cancel_url: `${returnUrl}?checkout=cancel`,
    metadata: { teamId },
  });

  return session.url!;
}

export async function handleWebhook(
  payload: string,
  signature: string,
): Promise<void> {
  const event = stripe().webhooks.constructEvent(
    payload,
    signature,
    env().STRIPE_WEBHOOK_SECRET,
  );

  switch (event.type) {
    case "checkout.session.completed": {
      const session = event.data.object as Stripe.Checkout.Session;
      await activateSubscription(
        session.metadata!.teamId,
        session.subscription as string,
      );
      break;
    }
    case "customer.subscription.deleted": {
      const sub = event.data.object as Stripe.Subscription;
      const teamId = sub.metadata.teamId;
      if (teamId) await downgradeToPlan(teamId, "free");
      break;
    }
    case "invoice.payment_failed": {
      const invoice = event.data.object as Stripe.Invoice;
      console.warn(`Payment failed for customer ${invoice.customer}`);
      break;
    }
  }
}

async function activateSubscription(
  teamId: string,
  subscriptionId: string,
): Promise<void> {
  const supabase = await createAdminClient();
  await supabase
    .from("teams")
    .update({
      plan: "pro" as PlanTier,
      stripe_subscription_id: subscriptionId,
      usage_limit: PLAN_LIMITS.pro.usage,
      updated_at: new Date().toISOString(),
    })
    .eq("id", teamId);
}

async function downgradeToPlan(
  teamId: string,
  plan: PlanTier,
): Promise<void> {
  const supabase = await createAdminClient();
  await supabase
    .from("teams")
    .update({
      plan,
      stripe_subscription_id: null,
      usage_limit: PLAN_LIMITS[plan].usage,
      updated_at: new Date().toISOString(),
    })
    .eq("id", teamId);
}

export async function getSubscriptionStatus(teamId: string) {
  const team = await findTeamById(teamId);
  if (!team) throw new NotFoundError("Team");

  if (!team.stripe_subscription_id) {
    return { active: false, plan: team.plan };
  }

  const subscription = await stripe().subscriptions.retrieve(
    team.stripe_subscription_id,
  );

  return {
    active: subscription.status === "active",
    plan: team.plan,
    currentPeriodEnd: new Date(
      subscription.current_period_end * 1000,
    ).toISOString(),
    cancelAtPeriodEnd: subscription.cancel_at_period_end,
  };
}
