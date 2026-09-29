function configuredPublicOrigin(): string | undefined {
  const configured = process.env.WISHLIST_PUBLIC_ORIGIN?.trim();
  if (configured) return new URL(configured).origin;

  if (process.env.APP_ENV === "production") {
    const redirectURI = process.env.WISHLIST_OIDC_REDIRECT_URI?.trim();
    if (!redirectURI) throw new Error("WISHLIST_PUBLIC_ORIGIN or WISHLIST_OIDC_REDIRECT_URI is required in production");
    return new URL(redirectURI).origin;
  }
}

export function isSameOriginWrite(request: Request): boolean {
  const received = request.headers.get("origin");
  if (!received) return false;

  const expected = configuredPublicOrigin() ?? new URL(request.url).origin;
  return received === expected;
}
