# Wishlist 0.3.0 event grounded design

## Evidence

The handoff requires admins to own app creation, feature publication, lifecycle choices, and private suggestion review. A visitor can propose demand, but only an admin can publish it or move a feature. The model already represented these operations as domain use cases; the missing boundary was an executable HTTP-shaped contract.

## Design

- Separate `/v1/admin/*` requests from visitor OTP identity.
- Require admin authorization on every route.
- Keep pending suggestions private; expose only the moderation fields an admin needs.
- Accept suggestions as new Voting features, reject them privately, or merge them only into a feature from the same app.
- Let admins edit/deactivate apps, publish/edit features, and advance features one lifecycle stage at a time.
- Preserve feature votes and delivery history as status changes.
- Never derive a lifecycle decision from vote rank.

## Executable evidence

`AdminAdapter` tests cover denied access, app and feature creation, pending suggestion inspection, and same-app merge. The runtime scenario now performs setup, suggestion publication, and lifecycle changes through the public, visitor, and admin adapters. The full deterministic simulation test suite and monitored runtime scenario pass.

## Open boundaries

The simulation does not prove bearer-token storage, HTTP middleware, SQLite concurrency, or browser authorization. Those remain application-owned implementation concerns. This release also does not claim production OTP delivery or deployment readiness.
