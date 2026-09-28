import { clearedAuthCookies } from "../../lib/admin-auth";
import { appPath } from "../../lib/paths";

export function GET() {
  const headers = new Headers({ location: appPath("/admin"), "cache-control": "no-store" });
  for (const value of clearedAuthCookies()) headers.append("set-cookie", value);
  return new Response(null, { status: 302, headers });
}
