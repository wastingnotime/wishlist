import type { APIEvent } from "@solidjs/start/server";
import { adminAuthMode, authorizationRedirect } from "../../lib/admin-auth";

export async function GET(_event: APIEvent) {
  if (adminAuthMode() !== "casdoor") return Response.redirect("/admin", 302);
  try {
    const result = await authorizationRedirect();
    return new Response(null, { status: 302, headers: { location: result.location, "set-cookie": result.cookie, "cache-control": "no-store" } });
  } catch {
    return Response.json({ error: { code: "authentication_unavailable" } }, { status: 503 });
  }
}
