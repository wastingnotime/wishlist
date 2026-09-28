# Browser API mapping

The browser calls its same-origin BFF at `/api/apps` and `/api/features` locally and `/wishlist/api/apps` and `/wishlist/api/features` in production. The BFF forwards these public GET requests to the Go API without changing query parameters or response DTOs. Upstream location is configured by `WISHLIST_API_URL`. The production `/wishlist` base path also covers assets, auth endpoints, and health checks.

The root URL stores view and app selection as `/?view=<status>&app=<slug>`. Missing view means `voting`; changing either selection updates the URL and fetches the current list. Reload keeps the selection and requests fresh data. The API responses are marked `no-store`.

The browser contract corresponds to the released public board adapter in `sandboxes/simulation/docs/adapters/public-board.md`. E2E evidence records the Chromium run in `runs/wishlist-public-board/browser-e2e.json`.
