# Wishlist production server base path routing

## Finding

The production browser received the Wishlist HTML shell from JSON BFF paths such
as `/wishlist/api/apps`, then failed while parsing the HTML as JSON. The same
behavior reproduced against the built web server locally. SolidStart's server
base URL mounted the handler under `/wishlist`, but its file route matcher
looked up `/api/*` against the still-prefixed request pathname and missed the
API route.

## Change

The server entry now removes the configured `SERVER_BASE_URL` from its internal
request URL before SolidStart matches routes. The browser URLs, asset URLs, and
the `/wishlist` public prefix remain unchanged. The transformation applies only
to paths inside the configured prefix and retains query parameters and the
original request method and headers.

## Validation

- Built the production web server with `WISHLIST_BASE_PATH=/wishlist`.
- Ran the built server locally: `/wishlist/healthz` returned plain `ok`; the
  `/wishlist/api/apps`, `/features`, and `/session` routes returned JSON 503
  responses when the API was intentionally unavailable. This confirms the
  requests reach their expected handlers instead of the HTML page.
- Production rollout and browser verification remain pending a reviewed image
  publication and deployment.
