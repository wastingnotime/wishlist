import { clearedAuthCookies } from "../../lib/admin-auth";

export function GET() {
  const headers = new Headers({ location: "/admin", "cache-control": "no-store" });
  for (const value of clearedAuthCookies()) headers.append("set-cookie", value);
  return new Response(null, { status: 302, headers });
}
