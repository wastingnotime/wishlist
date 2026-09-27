import assert from "node:assert/strict";
import { randomBytes } from "node:crypto";
import { createServer } from "node:http";
import { after, test } from "node:test";
import { authorizationRedirect, completeAuthorization, sessionAccessToken } from "./admin-auth.ts";

let providerURL = "";
const provider = createServer((request, response) => {
  if (request.url === "/discovery") {
    response.setHeader("content-type", "application/json");
    response.end(JSON.stringify({ issuer: providerURL, authorization_endpoint: providerURL + "/authorize", token_endpoint: providerURL + "/token" }));
  } else if (request.url === "/token") {
    response.setHeader("content-type", "application/json");
    response.end(JSON.stringify({ access_token: "signed-test-token", expires_in: 300 }));
  } else {
    response.statusCode = 404; response.end();
  }
});
const api = createServer((request, response) => {
  response.statusCode = request.headers.authorization === "Bearer signed-test-token" ? 200 : 403;
  response.end("{}");
});
await new Promise<void>(resolve => provider.listen(0, "127.0.0.1", resolve));
await new Promise<void>(resolve => api.listen(0, "127.0.0.1", resolve));
providerURL = `http://127.0.0.1:${(provider.address() as { port: number }).port}`;
const apiURL = `http://127.0.0.1:${(api.address() as { port: number }).port}`;
after(() => { provider.close(); api.close(); });
process.env.WISHLIST_ADMIN_AUTH_MODE = "casdoor";
process.env.WISHLIST_SESSION_KEY = randomBytes(32).toString("base64");
process.env.WISHLIST_OIDC_DISCOVERY_URL = providerURL + "/discovery";
process.env.WISHLIST_OIDC_ISSUER = providerURL;
process.env.WISHLIST_OIDC_CLIENT_ID = "wishlist";
process.env.WISHLIST_OIDC_CLIENT_SECRET = "test-secret";
process.env.WISHLIST_OIDC_REDIRECT_URI = "https://wishlist.example/auth/callback";
process.env.WISHLIST_OIDC_AUDIENCE = "wishlist-api";
process.env.WISHLIST_API_URL = apiURL;

test("OIDC code flow uses state and PKCE, then stores only a sealed server session", async () => {
  const start = await authorizationRedirect();
  const target = new URL(start.location);
  assert.equal(target.searchParams.get("response_type"), "code");
  assert.equal(target.searchParams.get("code_challenge_method"), "S256");
  assert.ok(target.searchParams.get("code_challenge"));
  assert.ok(start.cookie.includes("HttpOnly"));
  assert.ok(start.cookie.includes("Secure"));
  const transaction = start.cookie.split(";")[0];
  const state = target.searchParams.get("state");
  await assert.rejects(() => completeAuthorization(new Request("https://wishlist.example/auth/callback?code=ok&state=wrong", { headers: { cookie: transaction } })), /invalid or expired/);
  const session = await completeAuthorization(new Request(`https://wishlist.example/auth/callback?code=ok&state=${state}`, { headers: { cookie: transaction } }));
  assert.ok(session.includes("HttpOnly"));
  assert.ok(!session.includes("signed-test-token"));
  assert.equal(sessionAccessToken(new Request("https://wishlist.example/admin", { headers: { cookie: session.split(";")[0] } })), "signed-test-token");
  assert.equal(sessionAccessToken(new Request("https://wishlist.example/admin", { headers: { cookie: session.split(";")[0] + "tampered" } })), "");
});
