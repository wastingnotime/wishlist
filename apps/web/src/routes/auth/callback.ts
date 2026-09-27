import type { APIEvent } from "@solidjs/start/server";
import { adminAuthMode, clearedAuthCookies, clearedTransactionCookie, completeAuthorization } from "../../lib/admin-auth";

export async function GET(event: APIEvent) {
  if (adminAuthMode() !== "casdoor") return Response.redirect("/admin", 302);
  try {
    const sessionCookie = await completeAuthorization(event.request);
    const headers = new Headers({ location: "/admin", "cache-control": "no-store" });
    headers.append("set-cookie", clearedTransactionCookie());
    headers.append("set-cookie", sessionCookie);
    return new Response(null, { status: 302, headers });
  } catch (error) {
    const reason = error instanceof Error && error.message === "admin subject is not allowed" ? "forbidden" : "authentication_failed";
    const headers = new Headers({ location: `/admin?error=${reason}`, "cache-control": "no-store" });
    for (const value of clearedAuthCookies()) headers.append("set-cookie", value);
    return new Response(null, { status: 302, headers });
  }
}
