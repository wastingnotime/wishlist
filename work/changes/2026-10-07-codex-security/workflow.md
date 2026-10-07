# Wishlist Codex Security patch workflow

## Setup status

Requested on 2026-10-07: use the existing `wishlist` Codex cloud environment
for Workflow 2 (patch proposals as PRs) from
https://www.gend.co/blog/codex-security-ai-appsec-agent.

The environment is listed for `wastingnotime/wishlist`. Codex Security Cloud
was installed, but opening it reports “Your plan doesn’t support Security
Cloud.” No scan, monitoring configuration, generated threat model, or patch
PR has been created. Account/workspace access must be resolved first.

## Cloud configuration when access is available

Select `wastingnotime/wishlist`, branch `main`, and the existing `wishlist`
cloud environment. Start with a Repository scan to establish a baseline.
Review the generated threat model using the context below. Enable Commit
changes monitoring after reviewing the baseline and validation coverage.
These settings are intended configuration, not a record of enabled settings.

## Threat-model context

Wishlist is an internet-facing public demand board. Votes inform WNT judgment
and never automatically change feature lifecycle state or promise delivery.

- `apps/web/` owns the SolidStart browser experience and same-origin BFF;
  `apps/api/` owns authoritative Go HTTP behavior and PostgreSQL persistence.
  `sandboxes/simulation/` is the behavioral source for synchronized changes.
- Anonymous visitors can read published apps/features and request email OTPs.
  Verified visitor sessions authorize votes and private suggestions. Privacy
  access, correction, and erasure require recent verification; identity
  isolation must hold across these operations.
- Casdoor OIDC authenticates administrators, and the API checks an explicit
  subject allowlist. Login alone must not authorize admin operations. Inspect
  signature, issuer, audience, expiry, state/PKCE, encrypted HttpOnly admin
  cookies, visitor cookies, and origin validation on browser writes.
- Public responses must never disclose email addresses, OTPs, session material,
  private suggestions, or moderation data. Protect credentials and personal
  data in logs, errors, server-rendered output, exports, and privacy workflows.
- Examine OTP throttling, attempt limits, expiry, replay, session termination,
  vote uniqueness, admin moderation, stored input rendering, SQL handling,
  retention cleanup, and published-text handling following erasure.
- Production requires HTTPS OIDC, secure cookies, SMTP STARTTLS with certificate
  validation, configured secrets, and no local admin-token authorization.
  Local OTP logging and development credentials are deliberate test adapters;
  evaluate whether production can accidentally select them.
- Infra-platform owns production deployment, secrets, backups, and promotion.
  Validate against disposable local data and test adapters. Do not use live
  visitor data, production credentials, real email delivery, or deployed
  services for reproduction.

Use the API/web READMEs, production readiness record, and privacy records as
supporting context. Treat documented controls as intended behavior and verify
their implementation rather than assuming they prevent an attack.

## Patch instructions

For an accepted finding, propose one focused remediation PR per independently
reviewable issue. Follow repository instructions, preserve project boundaries,
and avoid unrelated cleanup. Preserve simulation/application consistency when
a fix changes domain behavior.

The PR description must explain the affected route or flow, attacker
preconditions, reduced risk, finding reference, and validation evidence. State
whether the original behavior was reproduced and identify any validation
limitations. Keep vulnerability details within the repository's authorized
review audience and omit credentials and personal data.

Add a regression check that exercises the attack condition and intended
authorized behavior where practical. Use the existing checks appropriate to
the change: `make test` for API/browser behavior, and `make integration` for
built-image, routing, or runtime changes. If Docker or another prerequisite is
unavailable, report the missing check explicitly; do not claim it passed.

Review the proposed patch and evidence before opening its PR. Use normal PR
review and existing CI; do not auto-merge or deploy security patches. Image
publication remains a candidate, with production promotion owned by
infra-platform.

## References

- https://developers.openai.com/blog/scaling-cyber-defenders-with-daybreak
- https://www.gend.co/blog/codex-security-ai-appsec-agent
