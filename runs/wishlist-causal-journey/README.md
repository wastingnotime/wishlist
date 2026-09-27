# Causal Wishlist journey run

This deterministic observation log records the refined shared scenario. Four human actors and the fake OTP provider exercise 12 named use cases through 38 invocations and 27 domain events. All three monitored invariants passed. Simulated time runs from 2026-01-01 12:00 UTC to 2026-01-31 12:15 UTC so the visitor session expiry is observed.

Regenerate and verify from the repository root:

```bash
PYTHONPATH="$HOME/.wnt/runtime/mrl:sandboxes/simulation/src" python3 sandboxes/simulation/tools/run_scenario.py > runs/wishlist-causal-journey/observations.jsonl
PYTHONPATH="$HOME/.wnt/runtime/mrl:sandboxes/simulation/src" python3 sandboxes/simulation/tools/run_scenario.py --summary
```

The log includes event names and outcomes, without private event payloads, email addresses, OTP codes, or session ids.
