# Wishlist admin write origin validation

## Problem

The production edge terminates viewer TLS and forwards HTTP to the Wishlist web
service. The BFF previously compared the browser's HTTPS `Origin` header with
the internal HTTP request URL, rejecting legitimate same-origin writes before
admin authorization.

## Change

Non-GET BFF requests are checked against the explicitly configured
`WISHLIST_PUBLIC_ORIGIN`. When it is absent in production, the BFF uses the
origin of the already-required public OIDC redirect URI. Local development
continues to compare against the request URL. Missing and cross-origin values
remain rejected.

This check only admits a request to the BFF. Admin authorization remains the
API's responsibility, and Wishlist demand never changes feature lifecycle
state automatically.

## Validation

- A production HTTPS browser origin is accepted when the internal request URL
  uses HTTP.
- A different browser origin is rejected.
- Local same-origin writes retain their existing behavior.
- An explicit public origin takes precedence over the OIDC redirect URI.
