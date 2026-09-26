# Wishlist 0.1.0 model expectation-gap detection

Date: 2026-09-26

## Compared evidence

- Preserved product handoff at `work/sources/wnt-wishlist-handoff.md`.
- Current model hypothesis and domain background knowledge.
- Six candidate slices and the combined demand-signal slice contract.
- Event-sourced model, public projection, and executable public board adapter.
- Eight passing deterministic simulation tests, including public adapter routes.
- Replayable 34-observation acceptance scenario at `runs/wishlist-domain-simulation/observations.jsonl`.
- Refinement receipts covering lifecycle, moderation edits, event privacy, and scenario instrumentation.

## Coherence findings

1. Public listing has executable route behavior for active app listing and the three feature views, with consistent app filtering, deterministic vote ordering, and explicit error responses.
2. OTP verification, vote uniqueness/removal, private suggestions, admin moderation, and lifecycle changes are exercised in use-case tests; the acceptance scenario traverses verification, vote, private suggestion, publication, selection, and delivery.
3. Public projections and runtime observations omit emails, OTP material, sessions, and private suggestion content.
4. Votes remain on features through status changes; ranking does not promote features.
5. The lifecycle is forward-only one stage at a time after publication. Admin creation may start in any public stage. The handoff does not define rollback; the hypothesis is recorded openly.

## Gaps and dispositions

| Gap | Evidence | Disposition |
| --- | --- | --- |
| The public board has an executable HTTP-shaped adapter, but no actual web client/browser run. | Simulation adapter and no `apps/web` project yet. | Release the model contract; validate URL state, refresh, accessibility, and timing in the first browser slice. |
| Event store, OTP delivery, IDs, and admin credential are deterministic fakes. | Simulation infrastructure. | Explicit non-claims; technology projects must choose durable storage, real email, session security, and admin authentication. |
| Suggestion approval and feature publication are separate in-memory event appends. | Use-case implementation. | Sequential local behavior is covered; production API must persist acceptance atomically or recover safely. |
| IP-level OTP throttling, database uniqueness under concurrency, CSRF, and session-cookie properties are not proven. | No production HTTP or database adapter. | Technology security gates, not model claims. Preserve the unique vote invariant with a database constraint. |
| Whether admins may move a published feature backward is unresolved. | Handoff diagram only specifies forward progression. | Keep forward-only behavior as a bounded hypothesis; revisit if product owners require correction transitions. |

## EGD result

No remaining gap contradicts the bounded domain release intent. Recommend model release 0.1.0 with the public board adapter as the first synchronization boundary. Do not treat simulation fakes or the executable route adapter as production persistence, authentication, browser validation, or deployment evidence.
