# Wishlist production preparation

Status: preparation in progress; production release is not yet eligible.

## Repository-side preparation

- [x] API and web container builds use separate non-root runtime images.
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

These checks validate local artifacts only. They do not publish images or establish production readiness in infra-platform.

## Platform email handoff

- [x] Infra-platform selected Mailgun and merged the consumer contract in [PR #659](https://github.com/wastingnotime/infra-platform/pull/659).
- [x] Infra-platform provisioned the SMTP password in Secrets Manager after Terraform created `/wnt/email/smtp-password` (2026-09-28). The value remains in the vault and Secrets Manager only.
- [x] Implemented the production Go SMTP OTP adapter against the platform settings. It requires STARTTLS with certificate validation, reads the mounted Docker secret, and bounds send time.
- [ ] Infra-platform materializes the password as a Docker secret on the Swarm manager and attaches it to the reviewed Wishlist API stack.
- [ ] Deploy the reviewed Wishlist candidate and verify end-to-end OTP receipt and verification through the production SMTP path.

## Release blockers and external inputs

- [ ] Publish and promote a reviewed Wishlist API/web candidate through infra-platform; no Wishlist image-publishing role or production ECR repositories are recorded in the current product contract.
- [ ] Establish the production ECR publisher role/repositories and candidate image naming with infra-platform. No Wishlist image-publishing role or ECR repository contract exists in the current infra-platform surface.
- [ ] Create repository-local candidate integration validation for the API and web images, including a real-browser run against the built web image.
- [ ] Agree with infra-platform on production service names, public route, ports, PostgreSQL placement/credentials, replicas, resources, and rollout/rollback expectations.
- [ ] Run candidate validation and preserve its validation result, promotion handoff, release notes, image digest, and build provenance.
- [ ] Obtain infra-platform's reviewed production promotion decision and observe the deployed digest before recording a production release.

## Required runtime settings

The API needs `APP_ENV=production`, `WISHLIST_DATABASE_URL`, `WISHLIST_OTP_SECRET`, Casdoor discovery URL, issuer and audience, and the allowed Casdoor subject IDs. Once the OTP provider is selected, its sender-specific settings must also be supplied as secrets. The web service needs `APP_ENV=production`, an internal `WISHLIST_API_URL`, Casdoor discovery URL, issuer, client id and secret, redirect URI, audience, and a base64 encoded 32-byte `WISHLIST_SESSION_KEY`. Production OIDC endpoints and the public callback must use HTTPS.

Keep all credentials in the deployment secret store. The local PostgreSQL password, demo admin token, and OTP logger are development-only.

## Authority boundary

Wishlist owns source, image metadata, release notes, and candidate validation evidence. `infra-platform` owns ECR authority, deployment manifests, production PostgreSQL and Swarm runtime decisions, and final promotion. Image publication or successful validation does not mean the app is in production.
