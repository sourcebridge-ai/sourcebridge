/**
 * Seeds the database with sample data for development.
 *
 * Usage: npx tsx src/db/seed.ts
 */
import { createClient } from "@supabase/supabase-js";
import bcrypt from "bcryptjs";

const SUPABASE_URL = process.env.NEXT_PUBLIC_SUPABASE_URL!;
const SERVICE_KEY = process.env.SUPABASE_SERVICE_ROLE_KEY!;

async function seed() {
  const supabase = createClient(SUPABASE_URL, SERVICE_KEY);
  const hash = await bcrypt.hash("password123", 12);

  console.log("Seeding database...");

  const { data: alice } = await supabase
    .from("users")
    .upsert({ email: "alice@acme.dev", name: "Alice Chen", password_hash: hash })
    .select()
    .single();

  const { data: bob } = await supabase
    .from("users")
    .upsert({ email: "bob@acme.dev", name: "Bob Park", password_hash: hash })
    .select()
    .single();

  const { data: team } = await supabase
    .from("teams")
    .upsert({ name: "Acme Corp", slug: "acme", plan: "pro", usage_limit: 10000 })
    .select()
    .single();

  if (alice && team) {
    await supabase
      .from("team_memberships")
      .upsert({ user_id: alice.id, team_id: team.id, role: "owner" });
  }
  if (bob && team) {
    await supabase
      .from("team_memberships")
      .upsert({ user_id: bob.id, team_id: team.id, role: "member" });
  }

  console.log("Seed complete: 2 users, 1 team.");
}

seed();
