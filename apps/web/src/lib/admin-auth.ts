import { createCipheriv, createDecipheriv, createHash, randomBytes, timingSafeEqual } from "node:crypto";

const transactionCookie = "wishlist-admin-transaction";
const sessionCookie = "wishlist-admin-session";
type Transaction = { state: string; verifier: string; createdAt: number };
type Session = { accessToken: string; expiresAt: number };
type Discovery = { issuer: string; authorization_endpoint: string; token_endpoint: string };

export function adminAuthMode(): "casdoor" | "token" {
  return process.env.APP_ENV === "production" || process.env.WISHLIST_ADMIN_AUTH_MODE === "casdoor" ? "casdoor" : "token";
}

function required(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

function key(): Buffer {
  const value = Buffer.from(required("WISHLIST_SESSION_KEY"), "base64");
  if (value.length !== 32) throw new Error("WISHLIST_SESSION_KEY must decode to 32 bytes");
  return value;
}

function seal(value: object): string {
  const iv = randomBytes(12);
  const cipher = createCipheriv("aes-256-gcm", key(), iv);
  const encrypted = Buffer.concat([cipher.update(JSON.stringify(value), "utf8"), cipher.final()]);
  return Buffer.concat([iv, cipher.getAuthTag(), encrypted]).toString("base64url");
}

function unseal<T>(value: string): T | undefined {
  try {
    const payload = Buffer.from(value, "base64url");
    if (payload.length < 29) return;
    const decipher = createDecipheriv("aes-256-gcm", key(), payload.subarray(0, 12));
    decipher.setAuthTag(payload.subarray(12, 28));
    return JSON.parse(Buffer.concat([decipher.update(payload.subarray(28)), decipher.final()]).toString("utf8")) as T;
  } catch { return; }
}

function cookies(request: Request): Record<string, string> {
  return Object.fromEntries((request.headers.get("cookie") ?? "").split(";").map(value => value.trim()).filter(Boolean).map(value => {
    const separator = value.indexOf("=");
    return separator < 0 ? [value, ""] : [value.slice(0, separator), value.slice(separator + 1)];
  }));
}

function cookie(name: string, value: string, maxAge: number): string {
  const secure = process.env.APP_ENV === "production" || process.env.WISHLIST_COOKIE_SECURE !== "false" ? "; Secure" : "";
  return `${name}=${value}; Path=/; Max-Age=${maxAge}${secure}; SameSite=Lax; HttpOnly`;
}

async function discovery(): Promise<Discovery> {
  const discoveryURL = required("WISHLIST_OIDC_DISCOVERY_URL");
  if (process.env.APP_ENV === "production" && ![discoveryURL, required("WISHLIST_OIDC_ISSUER"), required("WISHLIST_OIDC_REDIRECT_URI")].every(isHTTPS)) {
    throw new Error("production OIDC URLs must use HTTPS");
  }
  const response = await fetch(discoveryURL);
  if (!response.ok) throw new Error(`OIDC discovery returned ${response.status}`);
  const metadata = await response.json() as Discovery;
  if (metadata.issuer !== required("WISHLIST_OIDC_ISSUER") || !metadata.authorization_endpoint || !metadata.token_endpoint) {
    throw new Error("OIDC discovery issuer mismatch or missing endpoints");
  }
  if (process.env.APP_ENV === "production" && ![metadata.authorization_endpoint, metadata.token_endpoint].every(isHTTPS)) {
    throw new Error("production OIDC endpoints must use HTTPS");
  }
  return metadata;
}

function isHTTPS(value: string): boolean {
  try { return new URL(value).protocol === "https:"; } catch { return false; }
}

export async function authorizationRedirect(): Promise<{ location: string; cookie: string }> {
  const metadata = await discovery();
  const state = randomBytes(32).toString("base64url");
  const verifier = randomBytes(48).toString("base64url");
  const target = new URL(metadata.authorization_endpoint);
  target.searchParams.set("client_id", required("WISHLIST_OIDC_CLIENT_ID"));
  target.searchParams.set("redirect_uri", required("WISHLIST_OIDC_REDIRECT_URI"));
  target.searchParams.set("response_type", "code");
  target.searchParams.set("scope", "openid profile email");
  target.searchParams.set("resource", required("WISHLIST_OIDC_AUDIENCE"));
  target.searchParams.set("state", state);
  target.searchParams.set("code_challenge_method", "S256");
  target.searchParams.set("code_challenge", createHash("sha256").update(verifier).digest("base64url"));
  return { location: target.toString(), cookie: cookie(transactionCookie, seal({ state, verifier, createdAt: Date.now() } satisfies Transaction), 600) };
}

export async function completeAuthorization(request: Request): Promise<string> {
  const callback = new URL(request.url);
  const transaction = unseal<Transaction>(cookies(request)[transactionCookie] ?? "");
  const code = callback.searchParams.get("code") ?? "";
  const receivedState = Buffer.from(callback.searchParams.get("state") ?? "");
  const expectedState = Buffer.from(transaction?.state ?? "");
  if (!transaction || !code || Date.now() - transaction.createdAt > 600_000 || expectedState.length !== receivedState.length || !timingSafeEqual(expectedState, receivedState)) {
    throw new Error("OIDC authorization transaction is invalid or expired");
  }
  const metadata = await discovery();
  const body = new URLSearchParams({
    grant_type: "authorization_code",
    client_id: required("WISHLIST_OIDC_CLIENT_ID"),
    client_secret: required("WISHLIST_OIDC_CLIENT_SECRET"),
    redirect_uri: required("WISHLIST_OIDC_REDIRECT_URI"),
    resource: required("WISHLIST_OIDC_AUDIENCE"),
    code,
    code_verifier: transaction.verifier,
  });
  const response = await fetch(metadata.token_endpoint, { method: "POST", headers: { "content-type": "application/x-www-form-urlencoded" }, body, cache: "no-store" });
  const result = await response.json() as { access_token?: string; expires_in?: number };
  if (!response.ok || !result.access_token) throw new Error("OIDC token exchange failed");
  const check = await fetch(new URL("/v1/admin/session", process.env.WISHLIST_API_URL ?? "http://127.0.0.1:8080"), {
    headers: { authorization: `Bearer ${result.access_token}` }, cache: "no-store",
  });
  if (check.status === 403) throw new Error("admin subject is not allowed");
  if (!check.ok) throw new Error("OIDC token was not authorized by the Wishlist API");
  const maxAge = Math.max(1, Math.min(Number(result.expires_in) || 3600, 3600));
  return cookie(sessionCookie, seal({ accessToken: result.access_token, expiresAt: Date.now() + maxAge * 1000 } satisfies Session), maxAge);
}

export function sessionAccessToken(request: Request): string {
  const session = unseal<Session>(cookies(request)[sessionCookie] ?? "");
  return session && session.expiresAt > Date.now() ? session.accessToken : "";
}

export function clearedAuthCookies(): string[] {
  return [cookie(transactionCookie, "", 0), cookie(sessionCookie, "", 0)];
}

export function clearedTransactionCookie(): string { return cookie(transactionCookie, "", 0); }
