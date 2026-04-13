import { createAdminClient } from "@/lib/supabase/server";

export async function handleHealthCheck() {
  const checks: Record<string, "ok" | "error"> = {};

  try {
    const supabase = await createAdminClient();
    const { error } = await supabase.from("users").select("id").limit(1);
    checks.database = error ? "error" : "ok";
  } catch {
    checks.database = "error";
  }

  const allHealthy = Object.values(checks).every((v) => v === "ok");

  return Response.json(
    { status: allHealthy ? "healthy" : "degraded", checks },
    { status: allHealthy ? 200 : 503 },
  );
}
