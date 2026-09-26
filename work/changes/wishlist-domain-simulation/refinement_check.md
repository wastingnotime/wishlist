# First refinement check

The shared simulation implements the handoff's acceptance path with an append-only private event history, deterministic fake OTP delivery, explicit public projections, and a WNT MRL Runtime scenario adapter.

The first run exposed an observation serialization error: an actor intention carried a Python function. The adapter now emits a stable string intention. The in-memory event store also rejects duplicate active votes as a behavioral stand-in for the future database uniqueness constraint.

The test suite exercises public filtering and ranking, private moderation, OTP expiry and single use, vote uniqueness and removal, admin-only transitions, vote preservation, delivered links, and public privacy. The scenario receipt is at `runs/wishlist-domain-simulation/observations.jsonl`.

Remaining model questions are recorded in `sandboxes/simulation/docs/semantics/model_hypothesis.md`. Browser timing, production persistence constraints, production authentication, and actual email delivery require later adapter contracts and validation. This check is not model EGD or a release decision.

## Refinement 2

Review against the lifecycle diagram and moderation operations found two gaps. The model had allowed arbitrary status reversal and had only supported edits inline with acceptance. Published features now advance one lifecycle stage at a time; admins can still create a feature directly in any public stage, as the handoff permits. The model records direct Delivered features without inventing a Producing timestamp. Pending suggestions now have an explicit admin edit event/use case, and acceptance carries the reviewed content into the published feature.

The acceptance scenario now exercises editing before publication. Validation passes with 7 domain tests and 34 runtime observations; the `SuggestionEdited` event is present, monitored invariants pass, and the observation log contains no visitor email. Rollback remains an explicit product question rather than an unsupported correction behavior.
