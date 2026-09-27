# Causal Wishlist journey

The runtime scenario is one shared simulation environment with four people and one fake mail provider:

| Actor | Intention |
| --- | --- |
| Curator | Publish a catalog, review suggestions, choose work, deliver it, and control app visibility. |
| Maya | Propose medication reminders, support a lower-ranked feature, then remove her historical vote and log out. |
| Leo | Support the popular feature, propose a duplicate, and later encounter session expiry. |
| Noa | Support the popular feature and submit a suggestion the curator rejects. |
| OTP mail provider | Confirm that challenge delivery was accepted without exposing an address or code in observations. |

Actor behaviors schedule their first intention. Each result schedules the next meaningful action, so later work depends on the catalog, verified sessions, submissions, and moderation outcomes. The runtime scheduler only advances simulated time and orders actions; it is not a domain actor.

The observatory declares named use cases for catalog, identity, demand, moderation, and lifecycle. Each invocation emits an intention, a named use case, a result, and any domain events. The event payloads stay private. The run checks that every declared use case was invoked, every invariant passed, and the scenario finished.

The journey exercises three review decisions (accept, merge, reject), rank-independent selection of a lower-ranked feature, delivery with an HTTPS link, vote removal after delivery, rejection of a new vote after delivery, app deactivation/reactivation, logout, and 30-day visitor session expiry.

Run a concise verified summary:

```bash
PYTHONPATH="$HOME/.wnt/runtime/mrl:sandboxes/simulation/src" python3 sandboxes/simulation/tools/run_scenario.py --summary
```

Omit `--summary` for the full JSONL observation log. Use `mrl-simulation supervise` to inspect the declared observatory graph interactively.
