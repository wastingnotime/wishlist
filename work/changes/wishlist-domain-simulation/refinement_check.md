# First refinement check

The shared simulation implements the handoff's acceptance path with an append-only private event history, deterministic fake OTP delivery, explicit public projections, and a WNT MRL Runtime scenario adapter.

The first run exposed an observation serialization error: an actor intention carried a Python function. The adapter now emits a stable string intention. The in-memory event store also rejects duplicate active votes as a behavioral stand-in for the future database uniqueness constraint.

The test suite exercises public filtering and ranking, private moderation, OTP expiry and single use, vote uniqueness and removal, admin-only transitions, vote preservation, delivered links, and public privacy. The scenario receipt is at `runs/wishlist-domain-simulation/observations.jsonl`.

Remaining model questions are recorded in `sandboxes/simulation/docs/semantics/model_hypothesis.md`. Browser timing, production persistence constraints, production authentication, and actual email delivery require later adapter contracts and validation. This check is not model EGD or a release decision.
