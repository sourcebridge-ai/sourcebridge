import crypto from "crypto";
import { createAdminClient } from "@/lib/supabase/server";
import type { Invitation, CreateInvitationInput } from "../models/invitation";
import { invitationExpiresAt } from "../models/invitation";

export async function createInvitation(
  teamId: string,
  invitedBy: string,
  input: CreateInvitationInput,
): Promise<Invitation> {
  const supabase = await createAdminClient();
  const token = crypto.randomBytes(32).toString("hex");

  const { data, error } = await supabase
    .from("invitations")
    .insert({
      team_id: teamId,
      email: input.email,
      role: input.role,
      token,
      invited_by: invitedBy,
      expires_at: invitationExpiresAt(),
    })
    .select()
    .single<Invitation>();

  if (error) throw error;
  return data;
}

export async function findInvitationByToken(
  token: string,
): Promise<Invitation | null> {
  const supabase = await createAdminClient();
  const { data } = await supabase
    .from("invitations")
    .select("*")
    .eq("token", token)
    .is("accepted_at", null)
    .gt("expires_at", new Date().toISOString())
    .single<Invitation>();
  return data;
}

export async function acceptInvitation(id: string): Promise<void> {
  const supabase = await createAdminClient();
  await supabase
    .from("invitations")
    .update({ accepted_at: new Date().toISOString() })
    .eq("id", id);
}

export async function listPendingInvitations(
  teamId: string,
): Promise<Invitation[]> {
  const supabase = await createAdminClient();
  const { data, error } = await supabase
    .from("invitations")
    .select("*")
    .eq("team_id", teamId)
    .is("accepted_at", null)
    .gt("expires_at", new Date().toISOString())
    .order("created_at", { ascending: false });

  if (error) throw error;
  return data ?? [];
}
