# Wishlist privacy and retention candidate

Status: **candidate pending application PR validation and infra-platform promotion**, 2026-10-03. This note records the behavior intended for the candidate; it is not evidence of production deployment or an approved public privacy notice.

## Changes

- Add a `/privacy` visitor page for recently verified visitors to access their linked email, votes, and private suggestions, correct an email after verifying the new address, or erase live Wishlist records.
- Add authorized operator retrieval, email correction, and erasure actions in `/admin`. Operator actions require a private rights case reference; the reference is not stored in Wishlist, so the external restricted case record must be maintained.
- Erasure removes the live identity, sessions, OTP data, votes, and private suggestion sources transactionally. Counts fall with the deleted votes and feature lifecycle does not change. Published feature text linked to an erased suggestion is queued for human review.
- Add the retention migration and scheduled cleanup behavior for expired OTP/session records, visitor votes, identities, and suggestions. The 12-month vote window can reduce public vote totals as dormant identities expire.
- Link the visitor rights path from the public board and OTP dialog.

## Data and operations

The API migrations add `identities.last_verified_at` and `privacy_public_reviews`. Startup also applies the existing hourly retention worker. Deploying this candidate requires the migration-capable API image and the existing PostgreSQL database. No irreversible schema rewrite is included. Review deployed migration state and backup/restore gates before promoting.

GDPR applicability is deferred by the product owner. The service has no geographic restriction, so this note does not resolve jurisdiction or legal applicability. Legal review and the full notice remain release gates. Live-store erasure does not by itself delete provider records, logs, legacy SQLite files, exports, or backups.

## Validation

- Local PostgreSQL-backed Go API tests passed for visitor and operator privacy flows, authorization, freshness, correction, erasure, vote counts, suggestion review, and re-registration.
- Web TypeScript typecheck and production build passed.
- CI quality and built-image integration are required on the review PR. Candidate images, immutable digests, and promotion handoff do not exist until the main-branch candidate workflow succeeds.

## Related work

- WNT-87: visitor retention and cleanup.
- WNT-88: verified access, correction, and erasure.
- WNT-85/WNT-86: privacy release decision and full notice.
- WNT-89/WNT-102: processor/logging review and backup/restore gate.
- WNT-106: candidate validation and production promotion.
