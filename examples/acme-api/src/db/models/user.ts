import { z } from "zod";

export const UserRole = z.enum(["owner", "admin", "member"]);
export type UserRole = z.infer<typeof UserRole>;

export interface User {
  id: string;
  email: string;
  name: string;
  avatar_url: string | null;
  password_hash: string;
  created_at: string;
  updated_at: string;
}

export interface TeamMembership {
  user_id: string;
  team_id: string;
  role: UserRole;
  joined_at: string;
}

export const CreateUserInput = z.object({
  email: z.string().email(),
  name: z.string().min(1).max(100),
  password: z.string().min(8).max(128),
});

export const UpdateUserInput = z.object({
  name: z.string().min(1).max(100).optional(),
  avatar_url: z.string().url().nullable().optional(),
});

export type CreateUserInput = z.infer<typeof CreateUserInput>;
export type UpdateUserInput = z.infer<typeof UpdateUserInput>;
