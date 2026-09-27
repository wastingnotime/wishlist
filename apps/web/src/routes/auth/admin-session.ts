import type { APIEvent } from "@solidjs/start/server";
import { adminAuthMode, sessionAccessToken } from "../../lib/admin-auth";

export async function GET(event: APIEvent) {
  const mode = adminAuthMode();
  if (mode === "token") return Response.json({ mode, authorized: false }, { headers: { "cache-control": "no-store" } });
  const token = sessionAccessToken(event.request);
  if (!token) return Response.json({ mode, authorized: false }, { headers: { "cache-control": "no-store" } });
  try {
    const response = await fetch(new URL("/v1/admin/session", process.env.WISHLIST_API_URL ?? "http://127.0.0.1:8080"), {
      headers: { authorization: `Bearer ${token}` }, cache: "no-store",
    });
    return Response.json({ mode, authorized: response.ok }, { headers: { "cache-control": "no-store" } });
  } catch {
    return Response.json({ mode, authorized: false }, { status: 503, headers: { "cache-control": "no-store" } });
  }
}
