# First simulation validation receipt

Run from repository root:

```bash
PYTHONPATH=sandboxes/simulation/src python3 -m pytest -q sandboxes/simulation/tests
PYTHONPATH="$HOME/.wnt/runtime/mrl:sandboxes/simulation/src" python3 sandboxes/simulation/tools/run_scenario.py > runs/wishlist-domain-simulation/observations.jsonl
```

The scenario produces semantic JSONL observations without email addresses or OTP material. It is deterministic with seed 1 and initial time 2026-01-01T12:00:00Z. The latest validation completed with 34 observations, including OTP verification, vote, private suggestion submission and edit, plus all invariant checks passing. The domain suite has 9 passing tests, including the public board and visitor adapters.
