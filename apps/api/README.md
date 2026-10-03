# Wishlist API

Go API for the public board, verified visitor writes, and admin management. PostgreSQL stores catalog data, sessions, votes, and private suggestions. A unique database key enforces one vote per identity per feature.

## Run locally

Start the repository's local PostgreSQL container with `docker compose up -d --wait postgres`, then run:

```bash
WISHLIST_DATABASE_URL='postgres://wishlist:local-wishlist-only@127.0.0.1:5439/wishlist?sslmode=disable' go run ./cmd/api
```

The API listens on `127.0.0.1:8080`. `WISHLIST_DATABASE_URL` is required; startup applies versioned schema migrations and seeds only missing sample rows. Set `WISHLIST_API_ADDR`, `WISHLIST_ADMIN_TOKEN`, and `WISHLIST_OTP_SECRET` as needed. Local OTP codes are printed to the API terminal without the recipient address. The local admin token is `local-development-admin-token`; set a private value before using the app with real data.

Production admin access uses Casdoor OIDC. Configure `WISHLIST_OIDC_DISCOVERY_URL`, `WISHLIST_OIDC_ISSUER`, `WISHLIST_OIDC_AUDIENCE`, and a comma-separated `WISHLIST_ADMIN_SUBJECTS` list of Casdoor subject IDs. The API verifies the token signature, issuer, audience, and expiry, then checks the subject for every admin request. The local bearer token is ignored when `APP_ENV=production` or `WISHLIST_ADMIN_AUTH_MODE=casdoor`. `GET /v1/admin/session` lets the web callback confirm admin authority. An ordinary Casdoor login does not grant admin access.

Production startup requires the platform SMTP contract: `WNT_EMAIL_SMTP_HOST`, `WNT_EMAIL_SMTP_PORT`, `WNT_EMAIL_SMTP_USERNAME`, `WNT_EMAIL_SMTP_PASSWORD_FILE`, and `WNT_EMAIL_FROM`; `WNT_EMAIL_REPLY_TO` is optional. The password file is mounted by infra-platform as `/run/secrets/wnt_email_smtp_password`. The API requires STARTTLS with certificate validation and uses a bounded send timeout. Production also requires `WISHLIST_OTP_SECRET`; startup fails if the SMTP settings, mounted password, or OTP digest secret are missing. Production startup omits the local demo catalog; an admin must create production apps after the first launch. See the repository's [production readiness record](../../work/changes/wishlist-production-readiness/production_readiness.md) for remaining candidate and infrastructure gates.

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
| GET, DELETE | `/v1/privacy/data` | Recently verified visitor: retrieve linked records or erase live data |
| POST | `/v1/privacy/email/request`, `/v1/privacy/email/verify` | Recently verified visitor: verify a new email and correct the identity |
| POST | `/v1/admin/privacy/data`, `/v1/admin/privacy/correct-email`, `/v1/admin/privacy/erase` | Authorized operator: retrieve, correct, or erase by email with a private case reference |
| GET, POST | `/v1/admin/privacy/reviews`, `/v1/admin/privacy/reviews/{id}/resolve` | Review published feature text after source erasure |
| GET, POST, PATCH | `/v1/admin/apps`, `/v1/admin/features`, `/v1/admin/suggestions` | Admin app, feature, and moderation operations |

### Visitor email OTP limits

Any syntactically valid email address can request a sign-in code; Wishlist does
not maintain a recipient allowlist. Delivery depends on the configured SMTP
provider accepting and delivering to that address. Addresses are trimmed and
normalized to lowercase.

The API allows at most three OTP requests per email address per rolling hour.
Each code expires after 10 minutes, and verification is limited to five
attempts per code. After five incorrect attempts, request a new code.

### Visitor retention

On startup and every hour, the API removes expired OTP challenges and visitor
sessions, OTP request history within 24 hours of creation (after its one-hour
throttle use), private suggestions pending
for more than 180 days, and reviewed private suggestions one year after review.
Votes expire one year after the identity's last successful email verification;
the public vote count then decreases without changing the feature lifecycle.
An identity with no retained vote or suggestion is removed 90 days after its
30-day session validity ends. These periods are product-selected and still need
privacy review before they are published as a user-facing retention promise.
The cleanup writes only aggregate row counts to the API log. It does not cover
provider records, runtime logs, backups, or legacy SQLite copies. Visitor rights
requests use a separate recent-OTP workflow; see the [rights record](../../work/changes/wishlist-privacy-lgpd-baseline/rights-requests.md) and [retention record](../../work/changes/wishlist-privacy-lgpd-baseline/retention-and-deletion.md).

Writes through the browser BFF require a same-origin request. Admin routes require a bearer credential: the local development token in local mode, or a verified Casdoor token from an allowed subject in Casdoor mode. Public responses contain vote aggregates and never contain identity or pending suggestion data.

Suggestions stay private until an admin accepts them. Admin-created and accepted features start in Voting; admins control lifecycle transitions. Existing votes remain attached when a feature changes status.

## Validate

From the repository root, `make test` runs the API and browser suites with a disposable PostgreSQL container. To run the API suite alone, set `WISHLIST_TEST_DATABASE_URL` to a PostgreSQL database whose user can create schemas, then run `go test ./...`. Each API test uses its own temporary schema.
