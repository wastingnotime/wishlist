import type { APIEvent } from "@solidjs/start/server";

const allowed = new Set(["apps", "features"]);

export async function GET(event: APIEvent) {
  const path = event.params.path ?? "";
  if (!allowed.has(path)) return Response.json({ error: { code: "not_found" } }, { status: 404 });

  const processEnv = (globalThis as typeof globalThis & { process?: { env?: Record<string, string | undefined> } }).process?.env;
  const upstreamBase = processEnv?.WISHLIST_API_URL ?? "http://127.0.0.1:8080";
  const incoming = new URL(event.request.url);
  const upstream = new URL(`/v1/${path}${incoming.search}`, upstreamBase);
  try {
    const response = await fetch(upstream, { cache: "no-store" });
    return new Response(response.body, {
      status: response.status,
      headers: {
        "content-type": response.headers.get("content-type") ?? "application/json",
        "cache-control": "no-store",
      },
    });
  } catch {
    return Response.json({ error: { code: "api_unavailable" } }, { status: 503 });
  }
}
