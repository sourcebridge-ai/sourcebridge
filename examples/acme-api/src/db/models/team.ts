import { z } from "zod";

export type PlanTier = "free" | "pro" | "enterprise";

export interface Team {
  id: string;
  name: string;
  slug: string;
  plan: PlanTier;
  stripe_customer_id: string | null;
  stripe_subscription_id: string | null;
  usage_count: number;
  usage_limit: number;
  created_at: string;
  updated_at: string;
}

export const CreateTeamInput = z.object({
  name: z.string().min(1).max(100),
  slug: z.string().min(2).max(50).regex(/^[a-z0-9-]+$/),
});

export const UpdateTeamInput = z.object({
  name: z.string().min(1).max(100).optional(),
});

export type CreateTeamInput = z.infer<typeof CreateTeamInput>;
export type UpdateTeamInput = z.infer<typeof UpdateTeamInput>;

export const PLAN_LIMITS: Record<PlanTier, { usage: number; members: number }> = {
  free: { usage: 100, members: 3 },
  pro: { usage: 10_000, members: 25 },
  enterprise: { usage: Infinity, members: Infinity },
};
