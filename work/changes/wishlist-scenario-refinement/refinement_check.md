# Causal scenario refinement check

Date: 2026-09-26

## Reason

The previous runtime scenario had two passive actor names, six fixed scheduled steps, and no explicit observatory graph. The supervision UI therefore displayed a generic Use Case and Scheduler instead of the Wishlist workflow.

## Refinement

- Four named human actors schedule their own first intentions; the fake OTP provider contributes delivery observations.
- Follow-up actions are scheduled after their causal prerequisite succeeds.
- The graph declares 12 named use cases, adapters, a domain node, event store, and public projection.
- Observations link intentions, use cases, outcomes, and private event names without exposing event payloads.
- The journey includes voting by three visitors, all suggestion review outcomes, a lower-ranked selection, delivery, historical vote removal, inactive app visibility, logout, and expiry.
- A field-update bug found by the journey was fixed in the simulation application service.

## Check

The 14 deterministic tests and `run_scenario.py --summary` pass. The run contains 38 use case invocations, 27 domain events, and three passing invariants. Its JSONL evidence is retained under `runs/wishlist-causal-journey/`. The runner asserts that every declared use case was invoked, all graph edges reference declared nodes, all invariants passed, and the run finished. This refines evaluation evidence; it does not change the released public adapter contracts or introduce a new domain release.
