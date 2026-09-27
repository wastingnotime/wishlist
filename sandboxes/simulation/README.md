# Wishlist domain simulation

This is one evolving, deterministic Python event-sourced simulation of the proposed Wishlist app. It keeps product hypotheses separate from the materialized web/API implementation. Source: [`work/sources/wnt-wishlist-handoff.md`](../../work/sources/wnt-wishlist-handoff.md).

The selected implementation pack is WNT's `mrl-simulation-project` Python event-sourced shape. Domain decisions and state reconstruction live in `src/app`; only `src/app/simulation/mrl_runtime_scenario.py` imports the WNT MRL Runtime. No production framework, database, or email provider is selected here.

Run tests from the repository root:

```bash
PYTHONPATH=sandboxes/simulation/src python3 -m pytest -q sandboxes/simulation/tests
```

Run the complete acceptance scenario and print its semantic observations:

```bash
PYTHONPATH="$HOME/.wnt/runtime/mrl:sandboxes/simulation/src" python3 sandboxes/simulation/tools/run_scenario.py
```

Add `--summary` for a compact, checked result. The [causal journey](docs/evaluation/causal-journey.md) describes its actors, named use cases, and observatory graph. `mrl-simulation supervise` displays that graph interactively.

Model version 0.1.0 released the public board read adapter, 0.2.0 released visitor identity and writes, 0.3.0 released admin catalog and moderation routes, and 0.4.0 synchronizes the model with materialized app behavior. See the [visitor contract](docs/adapters/visitor-writes.md), [admin contract](docs/adapters/admin-writes.md), and the EGD/release records under `work/changes/wishlist-model-release-0.2.0/` through `work/changes/wishlist-model-release-0.4.0/`. App projects must validate their technology-specific behavior and security boundaries.
