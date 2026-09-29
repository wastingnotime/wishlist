import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { isSameOriginWrite } from "./request-origin.ts";

const originalEnv = { ...process.env };
afterEach(() => {
  process.env = { ...originalEnv };
});

test("accepts the public HTTPS origin when the production request URL is HTTP", () => {
  process.env.APP_ENV = "production";
  process.env.WISHLIST_OIDC_REDIRECT_URI = "https://wastingnotime.org/wishlist/auth/callback";

  const request = new Request("http://wastingnotime.org/wishlist/api/admin/apps", {
    method: "POST",
    headers: { origin: "https://wastingnotime.org" },
  });

  assert.equal(isSameOriginWrite(request), true);
});

test("rejects a cross-origin production write", () => {
  process.env.APP_ENV = "production";
  process.env.WISHLIST_OIDC_REDIRECT_URI = "https://wastingnotime.org/wishlist/auth/callback";

  const request = new Request("http://wastingnotime.org/wishlist/api/admin/apps", {
    method: "POST",
    headers: { origin: "https://attacker.example" },
  });

  assert.equal(isSameOriginWrite(request), false);
});

test("rejects a production write without an Origin header", () => {
  process.env.APP_ENV = "production";
  process.env.WISHLIST_OIDC_REDIRECT_URI = "https://wastingnotime.org/wishlist/auth/callback";

  const request = new Request("http://wastingnotime.org/wishlist/api/admin/apps", {
    method: "POST",
  });

  assert.equal(isSameOriginWrite(request), false);
});

test("uses the request URL origin in local development", () => {
  delete process.env.APP_ENV;
  delete process.env.WISHLIST_PUBLIC_ORIGIN;

  const request = new Request("http://127.0.0.1:5173/api/suggestions", {
    method: "POST",
    headers: { origin: "http://127.0.0.1:5173" },
  });

  assert.equal(isSameOriginWrite(request), true);
});

test("an explicit public origin overrides the callback origin", () => {
  process.env.APP_ENV = "production";
  process.env.WISHLIST_PUBLIC_ORIGIN = "https://wishlist.example";
  process.env.WISHLIST_OIDC_REDIRECT_URI = "https://identity.example/callback";

  const request = new Request("http://wishlist.example/api/admin/apps", {
    method: "POST",
    headers: { origin: "https://wishlist.example" },
  });

  assert.equal(isSameOriginWrite(request), true);
});
