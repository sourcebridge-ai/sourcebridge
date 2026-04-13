/**
 * Database migration runner.
 *
 * Creates the core tables: users, teams, team_memberships, invitations,
 * magic_links, and the increment_team_usage RPC function.
 *
 * Usage: npx tsx src/db/migrate.ts
 */
import { createClient } from "@supabase/supabase-js";

const SUPABASE_URL = process.env.NEXT_PUBLIC_SUPABASE_URL!;
const SERVICE_KEY = process.env.SUPABASE_SERVICE_ROLE_KEY!;

async function migrate() {
  const supabase = createClient(SUPABASE_URL, SERVICE_KEY);

  console.log("Running migrations...");

  const { error } = await supabase.rpc("exec_sql", {
    query: `
      CREATE TABLE IF NOT EXISTS users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        email TEXT UNIQUE NOT NULL,
        name TEXT NOT NULL,
        avatar_url TEXT,
        password_hash TEXT NOT NULL,
        created_at TIMESTAMPTZ DEFAULT now(),
        updated_at TIMESTAMPTZ DEFAULT now()
      );

      CREATE TABLE IF NOT EXISTS teams (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        name TEXT NOT NULL,
        slug TEXT UNIQUE NOT NULL,
        plan TEXT NOT NULL DEFAULT 'free',
        stripe_customer_id TEXT,
        stripe_subscription_id TEXT,
        usage_count INTEGER NOT NULL DEFAULT 0,
        usage_limit INTEGER NOT NULL DEFAULT 100,
        created_at TIMESTAMPTZ DEFAULT now(),
        updated_at TIMESTAMPTZ DEFAULT now()
      );

      CREATE TABLE IF NOT EXISTS team_memberships (
        user_id UUID REFERENCES users(id) ON DELETE CASCADE,
        team_id UUID REFERENCES teams(id) ON DELETE CASCADE,
        role TEXT NOT NULL DEFAULT 'member',
        joined_at TIMESTAMPTZ DEFAULT now(),
        PRIMARY KEY (user_id, team_id)
      );

      CREATE TABLE IF NOT EXISTS invitations (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        team_id UUID REFERENCES teams(id) ON DELETE CASCADE,
        email TEXT NOT NULL,
        role TEXT NOT NULL DEFAULT 'member',
        token TEXT UNIQUE NOT NULL,
        invited_by UUID REFERENCES users(id),
        expires_at TIMESTAMPTZ NOT NULL,
        accepted_at TIMESTAMPTZ,
        created_at TIMESTAMPTZ DEFAULT now()
      );

      CREATE TABLE IF NOT EXISTS magic_links (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        token TEXT UNIQUE NOT NULL,
        email TEXT NOT NULL,
        expires_at TIMESTAMPTZ NOT NULL,
        used BOOLEAN DEFAULT false,
        created_at TIMESTAMPTZ DEFAULT now()
      );

      CREATE OR REPLACE FUNCTION increment_team_usage(p_team_id UUID)
      RETURNS INTEGER AS $$
      DECLARE new_count INTEGER;
      BEGIN
        UPDATE teams
          SET usage_count = usage_count + 1
          WHERE id = p_team_id
          RETURNING usage_count INTO new_count;
        RETURN new_count;
      END;
      $$ LANGUAGE plpgsql;
    `,
  });

  if (error) {
    console.error("Migration failed:", error.message);
    process.exit(1);
  }

  console.log("Migrations complete.");
}

migrate();
