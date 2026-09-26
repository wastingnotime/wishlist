# Wishlist 0.2.0 model release decision

Date: 2026-09-26

## Decision

**Accepted: Wishlist domain simulation model 0.2.0.** It extends the released public board model with executable visitor identity and write routes. Evidence: nine passing deterministic tests and the 34-observation acceptance scenario with all monitored invariants passing.

## Released behavior

- Generic OTP request and verification outcomes, with normalized identity reuse and an opaque session establishment effect.
- A verified session query returns only that identity's active feature ids.
- Feature vote toggling, one active vote per identity/feature, no new votes outside Voting, and removal after transitions.
- Private suggestion submission; public board routes do not expose suggestion content.

## Explicit non-claims

This release does not prove real email delivery, IP-level abuse controls, production cookie attributes or browser timing, durable storage, concurrent database uniqueness, production admin authentication, or deployment readiness.

## Synchronization gate

The API and web projects may synchronize the visitor adapter contract. Persistent storage must enforce `UNIQUE(identity_id, feature_id)`. Email sender, session cookie, CSRF, throttling, and OTP secret storage require technology-owned implementations and tests before deployment. Pending moderation remains admin-only.
