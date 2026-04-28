"use client";

/**
 * /repositories/[id]/subsystems
 *
 * Redirects to the main repository detail page with the "subsystems" tab
 * selected. The tab system lives in the parent page (repositories/[id]/page.tsx)
 * and uses query-param routing (?tab=subsystems). This sub-route exists so
 * direct links and the plan's file-list entry resolve correctly.
 */

import { useParams, useRouter } from "next/navigation";
import { useEffect } from "react";

export default function SubsystemsRedirectPage() {
  const params = useParams();
  const router = useRouter();

  useEffect(() => {
    const id = Array.isArray(params?.id) ? params.id[0] : params?.id;
    if (id) {
      router.replace(`/repositories/${id}?tab=subsystems`);
    }
  }, [params, router]);

  return null;
}
