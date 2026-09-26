# Public board API mapping

The route behavior synchronizes `sandboxes/simulation/docs/adapters/public-board.md`.

- `GET /healthz` returns `{ "status": "ok" }`.
- `GET /v1/apps` returns active apps as `{ "apps": [...] }`.
- `GET /v1/features` accepts `view` (`voting` by default) and optional app slug `app`, returning `{ "features": [...] }`.
- Invalid view or app slug returns HTTP 400 with error code `invalid_request`.
- Responses are anonymous and `Cache-Control: no-store`.

This first implementation uses an in-memory sample read model. It is not the persistence design for vote or suggestion writes.
