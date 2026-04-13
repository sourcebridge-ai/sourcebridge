import { createAdminClient } from "@/lib/supabase/server";
import type { Team, CreateTeamInput, UpdateTeamInput } from "../models/team";

export async function findTeamById(id: string): Promise<Team | null> {
  const supabase = await createAdminClient();
  const { data } = await supabase
    .from("teams")
    .select("*")
    .eq("id", id)
    .single<Team>();
  return data;
}

export async function findTeamBySlug(slug: string): Promise<Team | null> {
  const supabase = await createAdminClient();
  const { data } = await supabase
    .from("teams")
    .select("*")
    .eq("slug", slug)
    .single<Team>();
  return data;
}

export async function createTeam(
  input: CreateTeamInput,
  ownerId: string,
): Promise<Team> {
  const supabase = await createAdminClient();

  const { data: team, error } = await supabase
    .from("teams")
    .insert({
      name: input.name,
      slug: input.slug,
      plan: "free",
      usage_count: 0,
      usage_limit: 100,
    })
    .select()
    .single<Team>();

  if (error) throw error;

  await supabase.from("team_memberships").insert({
    user_id: ownerId,
    team_id: team.id,
    role: "owner",
  });

  return team;
}

export async function updateTeam(
  id: string,
  input: UpdateTeamInput,
): Promise<Team> {
  const supabase = await createAdminClient();
  const { data, error } = await supabase
    .from("teams")
    .update({ ...input, updated_at: new Date().toISOString() })
    .eq("id", id)
    .select()
    .single<Team>();

  if (error) throw error;
  return data;
}

export async function incrementUsage(teamId: string): Promise<number> {
  const supabase = await createAdminClient();
  const { data, error } = await supabase.rpc("increment_team_usage", {
    p_team_id: teamId,
  });
  if (error) throw error;
  return data as number;
}

export async function listUserTeams(userId: string): Promise<Team[]> {
  const supabase = await createAdminClient();
  const { data, error } = await supabase
    .from("team_memberships")
    .select("teams(*)")
    .eq("user_id", userId);

  if (error) throw error;
  return (data ?? []).map((row: any) => row.teams);
}
