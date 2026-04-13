import { NextRequest } from "next/server";
import {
  getTeam,
  getTeamMembers,
  createNewTeam,
  updateExistingTeam,
  inviteTeamMember,
  removeTeamMember,
} from "@/services/team-service";
import { CreateTeamInput, UpdateTeamInput } from "@/db/models/team";
import { CreateInvitationInput } from "@/db/models/invitation";
import { validateBody } from "@/api/middleware/validate";
import { requireAuth } from "@/api/middleware/auth";
import { errorResponse } from "@/lib/errors";

export async function handleGetTeam(req: NextRequest, teamId: string) {
  try {
    const session = requireAuth(req);
    const team = await getTeam(teamId, session);
    return Response.json(team);
  } catch (error) {
    return errorResponse(error);
  }
}

export async function handleCreateTeam(req: NextRequest) {
  try {
    const session = requireAuth(req);
    const input = await validateBody(req, CreateTeamInput);
    const team = await createNewTeam(input, session);
    return Response.json(team, { status: 201 });
  } catch (error) {
    return errorResponse(error);
  }
}

export async function handleUpdateTeam(req: NextRequest, teamId: string) {
  try {
    const session = requireAuth(req);
    const input = await validateBody(req, UpdateTeamInput);
    const team = await updateExistingTeam(teamId, input, session);
    return Response.json(team);
  } catch (error) {
    return errorResponse(error);
  }
}

export async function handleGetMembers(req: NextRequest, teamId: string) {
  try {
    const session = requireAuth(req);
    const members = await getTeamMembers(teamId, session);
    return Response.json(members);
  } catch (error) {
    return errorResponse(error);
  }
}

export async function handleInviteMember(req: NextRequest, teamId: string) {
  try {
    const session = requireAuth(req);
    const input = await validateBody(req, CreateInvitationInput);
    const invitation = await inviteTeamMember(teamId, input, session);
    return Response.json(invitation, { status: 201 });
  } catch (error) {
    return errorResponse(error);
  }
}

export async function handleRemoveMember(
  req: NextRequest,
  teamId: string,
  userId: string,
) {
  try {
    const session = requireAuth(req);
    await removeTeamMember(teamId, userId, session);
    return Response.json({ ok: true });
  } catch (error) {
    return errorResponse(error);
  }
}
