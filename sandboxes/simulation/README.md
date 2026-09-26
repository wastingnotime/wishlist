# Wishlist domain simulation

This is one evolving, deterministic Python event-sourced simulation of the proposed Wishlist app. It keeps product hypotheses separate from a future web/API implementation. Source: [`work/sources/wnt-wishlist-handoff.md`](../../work/sources/wnt-wishlist-handoff.md).

The selected implementation pack is WNT's `mrl-simulation-project` Python event-sourced shape. Domain decisions and state reconstruction live in `src/app`; only `src/app/simulation/mrl_runtime_scenario.py` imports the WNT MRL Runtime. No production framework, database, or email provider is selected here.

Run tests from the repository root:

```bash
PYTHONPATH=sandboxes/simulation/src python3 -m pytest -q sandboxes/simulation/tests
```

Run the complete acceptance scenario and print its semantic observations:

```bash
PYTHONPATH="$HOME/.wnt/runtime/mrl:sandboxes/simulation/src" python3 sandboxes/simulation/tools/run_scenario.py
```

Model version 0.1.0 and its public board read adapter are released. The contract is described in [`docs/adapters/public-board.md`](docs/adapters/public-board.md); EGD and the release decision are recorded under `work/changes/wishlist-model-release-0.1.0/`. App projects must still validate their technology-specific behavior and security boundaries.
