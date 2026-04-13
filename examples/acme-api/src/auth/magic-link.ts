import crypto from "crypto";
import { createAdminClient } from "@/lib/supabase/server";

const MAGIC_LINK_TTL_MINUTES = 15;

interface MagicLinkRecord {
  token: string;
  email: string;
  expires_at: string;
  used: boolean;
}

export async function createMagicLink(email: string): Promise<string> {
  const token = crypto.randomBytes(32).toString("hex");
  const expiresAt = new Date(
    Date.now() + MAGIC_LINK_TTL_MINUTES * 60 * 1000,
  ).toISOString();

  const supabase = await createAdminClient();
  await supabase.from("magic_links").insert({
    token,
    email,
    expires_at: expiresAt,
    used: false,
  });

  return token;
}

export async function consumeMagicLink(
  token: string,
): Promise<string | null> {
  const supabase = await createAdminClient();

  const { data } = await supabase
    .from("magic_links")
    .select("*")
    .eq("token", token)
    .eq("used", false)
    .gt("expires_at", new Date().toISOString())
    .single<MagicLinkRecord>();

  if (!data) return null;

  await supabase
    .from("magic_links")
    .update({ used: true })
    .eq("token", token);

  return data.email;
}
