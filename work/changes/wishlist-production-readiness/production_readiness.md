# Wishlist production preparation

Status: Wishlist production runtime is active; the MVP release and updated candidate remain in progress. A healthy route or an earlier deployed digest does not establish readiness of this candidate or completion of public-voting gates.

## Repository-side preparation

- [x] API and web container builds use separate non-root runtime images.
- [x] API container cross-compiles arm64 from the native BuildKit platform (Wishlist PR #3 merged).
- [x] Web production assets build on the native BuildKit platform; the final runtime stage targets the requested image architecture (Wishlist PR #3 merged).
- [x] Images carry OCI source, revision, and build-time labels.
- [x] API health checks database readiness at `/readyz`; web image exposes `/healthz` without calling the API.
- [x] Production API startup does not insert the local Cat Care / Sliding Tasks demo catalog.
- [x] Production web API requests require an explicit `WISHLIST_API_URL` rather than silently using localhost.
- [x] `make build-images` builds both local images; tags and provenance values can be overridden.

## Local validation

- [x] `make test`: Go API tests, PostgreSQL-backed API integration, web typecheck and build, admin auth tests, and four Playwright browser flows passed.
- [x] Both API and web production images built locally.
- [x] API image smoke check returned healthy `/healthz` and `/readyz` responses with a disposable PostgreSQL database; the image runs as UID/GID `10001:10001`.
- [x] Web image smoke check returned `ok` from `/healthz` and runs as the non-root `node` user.
- [x] Follow-up web typecheck and admin auth tests passed after adding the production API URL guard test.
- [x] Production SMTP adapter configuration and message tests passed with `make test`; the four Playwright browser flows passed.
- [x] API and web production images built locally with `make build-images IMAGE_TAG=otp-review`.
- [x] `make integration` validates the built API and web images at `/wishlist` with a real browser and verifies that a published feature survives API restart (2026-09-29).
- [x] [Wishlist PR #2](https://github.com/wastingnotime/wishlist/pull/2) merged with the production SMTP adapter and manual candidate publishing workflow.
- [x] Infra-platform applied immutable Wishlist ECR repositories and the dedicated main-branch publisher role (2026-09-28).
- [x] The first candidate push exposed the missing `ecr:BatchGetImage` action. Infra-platform PR #664 added the repository-scoped permission, and the updated policy is now applied.
- [x] Configured the Wishlist `INFRA_PLATFORM_PROMOTION_TOKEN` repository secret with Actions write access limited to infra-platform (2026-09-29); candidate publication now dispatches its run ID to the infra-owned promotion intake.
- [x] Native-platform candidate build optimization was merged in Wishlist PR #3. Latest successful candidate publication was 2026-09-29; this change still needs a fresh main-branch candidate, validation artifact, and digest handoff.

These checks validate local artifacts only. They do not publish images or establish candidate production readiness in infra-platform. The existing `/wishlist/healthz` route returned HTTP 200 on 2026-10-03, and infra-platform has recorded prior successful Wishlist deployments; this confirms the existing runtime only, not the candidate in this change.

## Platform email handoff

- [x] Infra-platform selected Mailgun and merged the consumer contract in [PR #659](https://github.com/wastingnotime/infra-platform/pull/659).
- [x] Infra-platform provisioned the SMTP password in Secrets Manager after Terraform created `/wnt/email/smtp-password` (2026-09-28). The value remains in the vault and Secrets Manager only.
- [x] Implemented the production Go SMTP OTP adapter against the platform settings. It requires STARTTLS with certificate validation, reads the mounted Docker secret, and bounds send time.
- [x] Infra-platform materialized the password as Docker secret `wnt-email-smtp-password-d2aacfeaf72a` on the Swarm manager (SSM run `c56e1446-d1b5-4ad6-9a20-3e7c0a6ef0cb`, 2026-09-28).
- [x] Attach the Docker email secret to the Wishlist API stack.
- [ ] Promote this candidate through infra-platform and verify end-to-end OTP receipt and verification through the production SMTP path.

## Release blockers and external inputs

- [ ] Review and promote the visitor-retention API change, then verify the hourly cleanup and 12-month vote expiry behavior in the deployed runtime. Local PostgreSQL tests pass; the new migration and worker are not deployed yet. Track the remaining privacy gate in [WNT-87](https://linear.app/wastingnotime/issue/WNT-87/wishlist-enforce-retention-and-cleanup-of-visitor-data).
- [ ] Review and promote the [verified rights workflow](../wishlist-privacy-lgpd-baseline/rights-requests.md), then validate recent OTP, visitor access/correction/erasure, operator verification/case handling, public-text review, and restore reconciliation in the deployed runtime. Local PostgreSQL tests pass; WNT-88 is not deployed. The full notice and legal review remain separate release gates.
- [ ] Complete the [WNT-102](https://linear.app/wastingnotime/issue/WNT-102/wishlist-establish-and-test-postgresql-backup-and-restore-before) backup gate before public voting: observe a scheduled run and alert delivery, repeat an isolated restore, run current retention cleanup, and prove erasure reconciliation. The infra-platform apply, first manual encrypted backup, and basic restore are evidenced in the [backup receipt](https://github.com/wastingnotime/infra-platform/blob/main/runs/2026-10-03-wishlist-backup-baseline.md); they do not close the remaining checks.
- [x] Complete the Wishlist candidate publish permission fix through [infra-platform issue #662](https://github.com/wastingnotime/infra-platform/issues/662) and [infra-platform PR #664](https://github.com/wastingnotime/infra-platform/pull/664).
- [x] Infra-platform owns the Wishlist production stack and runtime settings; an existing stack and prior deployment are present. Candidate updates require the reviewed digest-pinned promotion path.
- [x] Create repository-local candidate integration validation for the API and web images, including a real-browser run against the built web image.
- [ ] Agree with infra-platform on production service names, public route, ports, PostgreSQL placement/credentials, replicas, resources, and rollout/rollback expectations.
- [ ] Run candidate validation and preserve its validation result, promotion handoff, release notes, image digest, and build provenance.
- [ ] Obtain infra-platform's reviewed production promotion decision and observe the deployed digest before recording a production release.

## Required runtime settings

The API needs `APP_ENV=production`, `WISHLIST_DATABASE_URL`, `WISHLIST_OTP_SECRET`, Casdoor discovery URL, issuer and audience, allowed Casdoor subject IDs, `WNT_EMAIL_SMTP_HOST`, `WNT_EMAIL_SMTP_PORT`, `WNT_EMAIL_SMTP_USERNAME`, `WNT_EMAIL_SMTP_PASSWORD_FILE`, and `WNT_EMAIL_FROM`. `WNT_EMAIL_REPLY_TO` is optional. The web service needs `APP_ENV=production`, an internal `WISHLIST_API_URL`, Casdoor discovery URL, issuer, client ID and secret, redirect URI, audience, and a base64 encoded 32-byte `WISHLIST_SESSION_KEY`. `WISHLIST_PUBLIC_ORIGIN` may explicitly set the trusted browser origin; otherwise production derives it from the redirect URI. Production OIDC endpoints and the public callback must use HTTPS.

Keep all credentials in the deployment secret store. The local PostgreSQL password, demo admin token, and OTP logger are development-only.

## Authority boundary

Wishlist owns source, image metadata, release notes, and candidate validation evidence. `infra-platform` owns ECR authority, deployment manifests, production PostgreSQL and Swarm runtime decisions, and final promotion. Image publication or successful validation does not mean the app is in production.
