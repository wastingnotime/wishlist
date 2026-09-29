# Wishlist production preparation

Status: preparation in progress; production release is not yet eligible.

## Repository-side preparation

- [x] API and web container builds use separate non-root runtime images.
- [ ] API container cross-compiles arm64 from the native BuildKit platform rather than running the Go compiler under target-architecture emulation (pending Wishlist PR #3).
- [ ] Web production assets build on the native BuildKit platform; the final runtime stage still targets the requested image architecture (pending Wishlist PR #3).
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
- [x] [Wishlist PR #2](https://github.com/wastingnotime/wishlist/pull/2) merged with the production SMTP adapter and manual candidate publishing workflow.
- [x] Infra-platform applied immutable Wishlist ECR repositories and the dedicated main-branch publisher role (2026-09-28).
- [x] The first candidate push exposed the missing `ecr:BatchGetImage` action. Infra-platform PR #664 added the repository-scoped permission, and the updated policy is now applied.
- [x] Configured the Wishlist `INFRA_PLATFORM_PROMOTION_TOKEN` repository secret with Actions write access limited to infra-platform (2026-09-29); candidate publication now dispatches its run ID to the infra-owned promotion intake.
- [ ] Retry candidate publication after the native build optimization is reviewed and merged; preserve its digest handoff for infra-platform promotion.
- [ ] Candidate run 36469623944 attempt 2 published the API image but the arm64-emulated web build ran for over 15 minutes; that attempt was cancelled. Wishlist PR #3 moves both build stages to the native platform.

These checks validate local artifacts only. They do not publish images or establish production readiness in infra-platform.

## Platform email handoff

- [x] Infra-platform selected Mailgun and merged the consumer contract in [PR #659](https://github.com/wastingnotime/infra-platform/pull/659).
- [x] Infra-platform provisioned the SMTP password in Secrets Manager after Terraform created `/wnt/email/smtp-password` (2026-09-28). The value remains in the vault and Secrets Manager only.
- [x] Implemented the production Go SMTP OTP adapter against the platform settings. It requires STARTTLS with certificate validation, reads the mounted Docker secret, and bounds send time.
- [x] Infra-platform materialized the password as Docker secret `wnt-email-smtp-password-d2aacfeaf72a` on the Swarm manager (SSM run `c56e1446-d1b5-4ad6-9a20-3e7c0a6ef0cb`, 2026-09-28).
- [ ] Attach the Docker secret to the reviewed Wishlist API stack after the Wishlist image and runtime contract are accepted.
- [ ] Deploy the reviewed Wishlist candidate and verify end-to-end OTP receipt and verification through the production SMTP path.

## Release blockers and external inputs

- [x] Complete the Wishlist candidate publish permission fix through [infra-platform issue #662](https://github.com/wastingnotime/infra-platform/issues/662) and [infra-platform PR #664](https://github.com/wastingnotime/infra-platform/pull/664).
- [ ] Add the Wishlist application stack, production runtime settings, and reviewed promotion path in infra-platform.
- [ ] Create repository-local candidate integration validation for the API and web images, including a real-browser run against the built web image.
- [ ] Agree with infra-platform on production service names, public route, ports, PostgreSQL placement/credentials, replicas, resources, and rollout/rollback expectations.
- [ ] Run candidate validation and preserve its validation result, promotion handoff, release notes, image digest, and build provenance.
- [ ] Obtain infra-platform's reviewed production promotion decision and observe the deployed digest before recording a production release.

## Required runtime settings

The API needs `APP_ENV=production`, `WISHLIST_DATABASE_URL`, `WISHLIST_OTP_SECRET`, Casdoor discovery URL, issuer and audience, allowed Casdoor subject IDs, `WNT_EMAIL_SMTP_HOST`, `WNT_EMAIL_SMTP_PORT`, `WNT_EMAIL_SMTP_USERNAME`, `WNT_EMAIL_SMTP_PASSWORD_FILE`, and `WNT_EMAIL_FROM`. `WNT_EMAIL_REPLY_TO` is optional. The web service needs `APP_ENV=production`, an internal `WISHLIST_API_URL`, Casdoor discovery URL, issuer, client ID and secret, redirect URI, audience, and a base64 encoded 32-byte `WISHLIST_SESSION_KEY`. Production OIDC endpoints and the public callback must use HTTPS.

Keep all credentials in the deployment secret store. The local PostgreSQL password, demo admin token, and OTP logger are development-only.

## Authority boundary

Wishlist owns source, image metadata, release notes, and candidate validation evidence. `infra-platform` owns ECR authority, deployment manifests, production PostgreSQL and Swarm runtime decisions, and final promotion. Image publication or successful validation does not mean the app is in production.
