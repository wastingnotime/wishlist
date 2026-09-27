# Wishlist 0.4.0 event grounded design

## Evidence

The materialized Go API and PostgreSQL store publish features in Voting, require an HTTPS delivery link for Delivered, hide inactive apps and their features from public reads, use `pending_review` for private submissions, and expire visitor sessions after 30 days. The API also ends sessions on logout and exposes an admin authorization check. These behaviors had drifted from the released simulation.

## Model refinement

- Publish all features in Voting, then advance one stage at a time. Require an HTTPS URL for Delivered.
- Hide inactive apps and their features from public projections while retaining private event history.
- Track visitor session expiry and explicit logout as replayable model state.
- Use `pending_review` for suggestions and allow title or description changes only as part of acceptance.
- Match materialized app limits for app names and descriptions, slug syntax, and HTTPS public URLs.
- Expose the admin session check and the pending suggestion fields used by the app.

PostgreSQL and Casdoor remain technology choices outside the pure model. Casdoor only supplies admin authority to the production adapter.

## Executable evidence

The deterministic tests cover lifecycle constraints, inactive app visibility, expired and ended sessions, HTTPS URLs, suggestion state, and admin DTO fields. The runtime scenario still exercises private suggestion publication and an admin selected delivery path. Both checks passed for this release.
