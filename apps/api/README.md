# Wishlist API

Go API for the public board, verified visitor writes, and admin management. PostgreSQL stores catalog data, sessions, votes, and private suggestions. A unique database key enforces one vote per identity per feature.

## Run locally

Start the repository's local PostgreSQL container with `docker compose up -d --wait postgres`, then run:

```bash
WISHLIST_DATABASE_URL='postgres://wishlist:local-wishlist-only@127.0.0.1:5439/wishlist?sslmode=disable' go run ./cmd/api
```

The API listens on `127.0.0.1:8080`. `WISHLIST_DATABASE_URL` is required; startup applies versioned schema migrations and seeds only missing sample rows. Set `WISHLIST_API_ADDR`, `WISHLIST_ADMIN_TOKEN`, and `WISHLIST_OTP_SECRET` as needed. Local OTP codes are printed to the API terminal without the recipient address. The local admin token is `local-development-admin-token`; set a private value before using the app with real data.

Production startup intentionally fails until a production email OTP adapter exists. The local code logger and default keys are development-only.

## Routes

| Method | Route | Purpose |
| --- | --- | --- |
| GET | `/healthz`, `/readyz` | Liveness and database readiness |
| GET | `/v1/apps` | Active public apps |
| GET | `/v1/features?view=voting&app=cat-care` | Public features; views are `voting`, `producing`, `delivered` |
| POST | `/v1/otp`, `/v1/otp/verify` | Request and verify a one-time email code |
| GET, DELETE | `/v1/session` | Read or end the HttpOnly visitor session |
| POST | `/v1/features/{id}/vote` | Add or remove the visitor's vote |
| POST | `/v1/suggestions` | Submit a private suggestion |
| GET, POST, PATCH | `/v1/admin/apps`, `/v1/admin/features`, `/v1/admin/suggestions` | Admin app, feature, and moderation operations |

Writes through the browser BFF require a same-origin request. Admin routes also require `Authorization: Bearer <WISHLIST_ADMIN_TOKEN>`. Public responses contain vote aggregates and never contain identity or pending suggestion data.

Suggestions stay private until an admin accepts them. Admin-created and accepted features start in Voting; admins control lifecycle transitions. Existing votes remain attached when a feature changes status.

## Validate

From the repository root, `make test` runs the API and browser suites with a disposable PostgreSQL container. To run the API suite alone, set `WISHLIST_TEST_DATABASE_URL` to a PostgreSQL database whose user can create schemas, then run `go test ./...`. Each API test uses its own temporary schema.
