# Wishlist 0.1.0 model release decision

Date: 2026-09-26

## Decision

**Accepted: Wishlist domain simulation model 0.1.0.** The current model is coherent for deterministic domain behavior and the anonymous public board read contract. Evidence: eight passing simulation tests and the 34-observation acceptance scenario with all monitored invariants passing.

## Released behavior

- Public app and feature reads across Voting, Producing, and Delivered, consistently filtered by app.
- Public ranking by active vote count with deterministic publication and id tie-breaks; rankings do not set lifecycle state.
- Verified email identity as the gate for voting and suggesting, with one active vote per identity and feature.
- Private suggestions with admin edit, accept, reject, and merge decisions.
- Admin-controlled forward feature lifecycle with historical votes retained and optional delivery URL.
- Public projections that omit identity and moderation data.
- Executable anonymous public board adapter contract at `/v1/apps` and `/v1/features`.

## Explicit non-claims

This is not a production application, persistent store, production email provider, production admin authentication, secure browser session, database concurrency proof, privacy certification, browser timing/accessibility validation, or deployment decision. The model does not settle rollback policy.

## Synchronization gate

The first technology slice may materialize the public board routes and browser views. Keep API authority separate from browser rendering. Before public deployment, implement durable vote uniqueness, OTP and session security, admin authorization, input throttling, CSRF protections where needed, and actual browser/integration validation. Extend simulation adapter contracts before synchronizing write flows.
