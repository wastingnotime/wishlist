import type { APIEvent } from "@solidjs/start/server";

const routes = new Map<string, Set<string>>([
  ["apps", new Set(["GET"])],
  ["features", new Set(["GET"])],
  ["session", new Set(["GET", "DELETE"])],
  ["otp", new Set(["POST"])],
  ["otp/verify", new Set(["POST"])],
  ["suggestions", new Set(["POST"])],
  ["admin/suggestions", new Set(["GET"])],
  ["admin/apps", new Set(["GET", "POST"])],
  ["admin/features", new Set(["POST"])],
]);

async function proxy(event: APIEvent) {
  const path = (event.params.path ?? "").replace(/^\/+|\/+$/g, "");
  const methods = routes.get(path)
    ?? (/^features\/[^/]+\/vote$/.test(path) ? new Set(["POST"])
    : /^admin\/apps\/[^/]+$/.test(path) || /^admin\/features\/[^/]+$/.test(path) ? new Set(["PATCH"])
    : /^admin\/suggestions\/[^/]+\/(accept|reject|merge)$/.test(path) ? new Set(["POST"]) : undefined);
  if (!methods) return Response.json({ error: { code: "not_found" } }, { status: 404 });
  if (!methods.has(event.request.method)) return Response.json({ error: { code: "method_not_allowed" } }, { status: 405 });

  const processEnv = (globalThis as typeof globalThis & { process?: { env?: Record<string, string | undefined> } }).process?.env;
  const upstreamBase = processEnv?.WISHLIST_API_URL ?? "http://127.0.0.1:8080";
  const incoming = new URL(event.request.url);
  if (event.request.method !== "GET" && event.request.headers.get("origin") !== incoming.origin) {
    return Response.json({ error: { code: "forbidden" } }, { status: 403 });
  }
  const upstream = new URL(`/v1/${path}${incoming.search}`, upstreamBase);
  try {
    const headers = new Headers();
    for (const name of ["content-type", "cookie", "authorization"]) {
      const value = event.request.headers.get(name);
      if (value) headers.set(name, value);
    }
    if (event.request.method !== "GET") headers.set("x-wishlist-same-origin", "1");
    const response = await fetch(upstream, {
      method: event.request.method,
      headers,
      body: event.request.method === "GET" || event.request.method === "DELETE"
        ? undefined
        : await event.request.arrayBuffer(),
      cache: "no-store",
    });
    const outgoing = new Headers({
      "content-type": response.headers.get("content-type") ?? "application/json",
      "cache-control": "no-store",
    });
    const cookie = response.headers.get("set-cookie");
    if (cookie) outgoing.set("set-cookie", cookie);
    return new Response(response.body, { status: response.status, headers: outgoing });
  } catch {
    return Response.json({ error: { code: "api_unavailable" } }, { status: 503 });
  }
}

export const GET = proxy;
export const POST = proxy;
export const DELETE = proxy;
export const PATCH = proxy;
