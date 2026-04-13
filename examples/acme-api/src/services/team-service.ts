import {
  findTeamById,
  createTeam,
  updateTeam,
  incrementUsage,
} from "@/db/queries/teams";
import { listTeamMembers } from "@/db/queries/users";
import {
  createInvitation,
  findInvitationByToken,
  acceptInvitation,
} from "@/db/queries/invitations";
import { sendInvitationEmail } from "./email-service";
import { createAdminClient } from "@/lib/supabase/server";
import { PLAN_LIMITS } from "@/db/models/team";
import {
  AuthorizationError,
  NotFoundError,
  AppError,
} from "@/lib/errors";
import type { CreateTeamInput, UpdateTeamInput } from "@/db/models/team";
import type { CreateInvitationInput } from "@/db/models/invitation";
import type { TokenPayload } from "@/auth/jwt";

export async function getTeam(teamId: string, session: TokenPayload) {
  if (session.teamId !== teamId) throw new AuthorizationError();
  const team = await findTeamById(teamId);
  if (!team) throw new NotFoundError("Team");
  return team;
}

export async function getTeamMembers(teamId: string, session: TokenPayload) {
  if (session.teamId !== teamId) throw new AuthorizationError();
  return listTeamMembers(teamId);
}

export async function createNewTeam(
  input: CreateTeamInput,
  session: TokenPayload,
) {
  return createTeam(input, session.sub);
}

export async function updateExistingTeam(
  teamId: string,
  input: UpdateTeamInput,
  session: TokenPayload,
) {
  if (session.teamId !== teamId) throw new AuthorizationError();
  if (session.role !== "owner" && session.role !== "admin") {
    throw new AuthorizationError("Only owners and admins can update teams");
  }
  return updateTeam(teamId, input);
}

export async function inviteTeamMember(
  teamId: string,
  input: CreateInvitationInput,
  session: TokenPayload,
) {
  if (session.teamId !== teamId) throw new AuthorizationError();
  if (session.role !== "owner" && session.role !== "admin") {
    throw new AuthorizationError("Only owners and admins can invite members");
  }

  const team = await findTeamById(teamId);
  if (!team) throw new NotFoundError("Team");

  const members = await listTeamMembers(teamId);
  const limit = PLAN_LIMITS[team.plan].members;
  if (members.length >= limit) {
    throw new AppError(
      `Team member limit reached (${limit}). Upgrade your plan.`,
      "MEMBER_LIMIT",
      403,
    );
  }

  const invitation = await createInvitation(teamId, session.sub, input);
  await sendInvitationEmail(input.email, team.name, invitation.token);
  return invitation;
}

export async function acceptTeamInvitation(token: string, userId: string) {
  const invitation = await findInvitationByToken(token);
  if (!invitation) throw new NotFoundError("Invitation");

  const supabase = await createAdminClient();
  await supabase.from("team_memberships").insert({
    user_id: userId,
    team_id: invitation.team_id,
    role: invitation.role,
  });

  await acceptInvitation(invitation.id);
  return invitation;
}

export async function trackUsage(teamId: string) {
  const team = await findTeamById(teamId);
  if (!team) throw new NotFoundError("Team");

  if (team.usage_count >= team.usage_limit) {
    throw new AppError(
      "Usage limit exceeded. Upgrade your plan for more capacity.",
      "USAGE_LIMIT",
      403,
    );
  }

  return incrementUsage(teamId);
}

export async function removeTeamMember(
  teamId: string,
  userId: string,
  session: TokenPayload,
) {
  if (session.role !== "owner" && session.role !== "admin") {
    throw new AuthorizationError("Only owners and admins can remove members");
  }

  const supabase = await createAdminClient();
  await supabase
    .from("team_memberships")
    .delete()
    .eq("team_id", teamId)
    .eq("user_id", userId);
}
