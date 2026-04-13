import { createAdminClient } from "@/lib/supabase/server";
import type { User, CreateUserInput, UpdateUserInput } from "../models/user";
import { hashPassword } from "@/auth/password";

export async function findUserByEmail(email: string): Promise<User | null> {
  const supabase = await createAdminClient();
  const { data } = await supabase
    .from("users")
    .select("*")
    .eq("email", email)
    .single<User>();
  return data;
}

export async function findUserById(id: string): Promise<User | null> {
  const supabase = await createAdminClient();
  const { data } = await supabase
    .from("users")
    .select("*")
    .eq("id", id)
    .single<User>();
  return data;
}

export async function createUser(input: CreateUserInput): Promise<User> {
  const supabase = await createAdminClient();
  const passwordHash = await hashPassword(input.password);

  const { data, error } = await supabase
    .from("users")
    .insert({
      email: input.email,
      name: input.name,
      password_hash: passwordHash,
    })
    .select()
    .single<User>();

  if (error) throw error;
  return data;
}

export async function updateUser(
  id: string,
  input: UpdateUserInput,
): Promise<User> {
  const supabase = await createAdminClient();
  const { data, error } = await supabase
    .from("users")
    .update({ ...input, updated_at: new Date().toISOString() })
    .eq("id", id)
    .select()
    .single<User>();

  if (error) throw error;
  return data;
}

export async function listTeamMembers(
  teamId: string,
): Promise<(User & { role: string })[]> {
  const supabase = await createAdminClient();
  const { data, error } = await supabase
    .from("team_memberships")
    .select("role, users(*)")
    .eq("team_id", teamId);

  if (error) throw error;
  return (data ?? []).map((row: any) => ({
    ...row.users,
    role: row.role,
  }));
}
