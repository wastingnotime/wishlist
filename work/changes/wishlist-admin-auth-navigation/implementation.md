# Admin authentication navigation

## Finding

On the production `/wishlist` deployment, Solid Router intercepted the admin's same-origin sign-in and sign-out anchors as client-side route changes. These endpoints are server redirects: sign-in sets the OIDC transaction cookie and redirects to Casdoor, while sign-out clears the admin cookies and redirects back to `/wishlist/admin`. Fetching either endpoint as a client-side navigation does not perform the browser document redirect, leaving a blank page until reload.

## Change

Set `target="_self"` on the two admin authentication anchors. Solid Router skips anchors with a target, so the browser performs a full navigation and follows the server redirect. This keeps the existing OIDC and session behavior while making the navigation work on the deployed base path.

## Validation

- `make test` passes: Go tests, TypeScript typecheck, OIDC/request-origin tests, production build, and five Playwright browser flows.
- The new Playwright flow stubs an unauthenticated Casdoor session and redirect, clicks the admin sign-in link, and verifies that a new document loaded and the redirect returned to `/admin`.
- The production issue was reproduced before the change by clicking sign-in in the active browser: it remained on the blank `/wishlist/auth/login` response until a reload.
