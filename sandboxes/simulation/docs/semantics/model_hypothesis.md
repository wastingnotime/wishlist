# Model hypothesis

## Governing rule

Wishlist senses public demand for WNT products. Votes never select, schedule, or promise work. Only an admin publishes suggestions or changes a public feature's lifecycle.

## Vocabulary and boundaries

- **App:** WNT product available for feature requests; inactive apps and their features are hidden from public reads and cannot receive new visitor suggestions. Their history remains private.
- **Feature:** published, public item in exactly one of `voting`, `producing`, or `delivered`. Historical votes remain attached throughout its lifecycle.
- **Suggestion:** private visitor submission in `pending_review`, `accepted`, `rejected`, or `merged`. It is distinct from a public feature.
- **Verified identity:** normalized email proven through a short-lived challenge. A visitor session represents that proof for 30 days unless ended sooner; no user profile exists.
- **Vote:** one active endorsement by one verified identity for one feature. Only `voting` features accept new votes.
- **Admin:** separately authenticated authority for publishing and lifecycle decisions.

The simulation event history is private internal evidence. Public queries are explicit projections that omit email, OTP material, sessions, and private suggestions. A verified visitor adapter may return only the current identity's active feature ids to restore vote state; the web transport stores the opaque session in a cookie rather than exposing the session id in a response body.

## State transitions

```text
pending_review suggestion -> accepted (new voting feature)
                   -> merged (existing feature)
                   -> rejected

voting feature -> producing -> delivered
```

Every published feature starts in Voting and lifecycle advances one stage at a time. Delivered requires an HTTPS delivery URL. The handoff does not define rollback, so corrections to prior statuses remain an open product question. Vote removal stays available after status transitions, while new votes are accepted only in Voting. Admins may refine a suggestion's title and description while accepting it; there is no separate pending edit action in the app.

## Candidate slices

1. Apps, admin-published features, public board, app filter.
2. Admin lifecycle and historical vote preservation.
3. Email challenge and verified visitor session.
4. Vote uniqueness, removal, and ranking.
5. Private suggestions and admin moderation.
6. Public delivery links and channel adapter behavior.

The first runnable model covers these domain behaviors together so the handoff's acceptance path can be exercised. Browser timing, persistence schema, real email delivery, and production admin authentication remain technology project decisions.

## Open questions

- Should votes be removable after a feature leaves Voting? Current hypothesis: yes, for historical correction, but no new votes are accepted then.
- Should a deactivated app still show its previously published features? Current materialized behavior: no; private history is retained.
- Can an admin publish directly into Producing or Delivered? Current materialized behavior: no; all features start in Voting.
- Should a published feature ever move backward for correction? Current hypothesis: no; the handoff only describes forward progression.
