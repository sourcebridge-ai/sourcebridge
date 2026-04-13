import { z } from "zod";
import type { UserRole } from "./user";

export interface Invitation {
  id: string;
  team_id: string;
  email: string;
  role: UserRole;
  token: string;
  invited_by: string;
  expires_at: string;
  accepted_at: string | null;
  created_at: string;
}

export const CreateInvitationInput = z.object({
  email: z.string().email(),
  role: z.enum(["admin", "member"]),
});

export type CreateInvitationInput = z.infer<typeof CreateInvitationInput>;

const INVITATION_TTL_DAYS = 7;

export function invitationExpiresAt(): string {
  return new Date(
    Date.now() + INVITATION_TTL_DAYS * 24 * 60 * 60 * 1000,
  ).toISOString();
}
