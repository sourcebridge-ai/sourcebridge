import { z } from "zod";
import { ValidationError } from "@/lib/errors";

/**
 * Parses and validates a request body against a Zod schema.
 * Throws ValidationError with per-field messages on failure.
 */
export async function validateBody<T extends z.ZodSchema>(
  request: Request,
  schema: T,
): Promise<z.infer<T>> {
  let body: unknown;
  try {
    body = await request.json();
  } catch {
    throw new ValidationError("Invalid JSON body", { body: "Expected valid JSON" });
  }

  const result = schema.safeParse(body);
  if (!result.success) {
    const fields: Record<string, string> = {};
    for (const issue of result.error.issues) {
      fields[issue.path.join(".")] = issue.message;
    }
    throw new ValidationError("Validation failed", fields);
  }

  return result.data;
}
