"""Run the causal Wishlist journey and verify its declared observatory coverage."""

import json
import sys

from mrl_simulation_runtime.runner import SimulationRunner

from app.simulation.mrl_runtime_scenario import create_simulation


if __name__ == "__main__":
    scenario = create_simulation()
    assert not scenario.scheduled_actions
    assert sum(actor.behavior is not None for actor in scenario.actors) >= 4
    observations = SimulationRunner().run(scenario).observations
    rows = observations.observations
    node_ids = {node.id for node in scenario.observatory_nodes}
    assert {actor.name for actor in scenario.actors} <= node_ids
    assert all(edge.from_node in node_ids and edge.to_node in node_ids
               for edge in scenario.observatory_edges)
    declared = {node.id for node in scenario.observatory_nodes if node.kind == "use_case"}
    invoked = {row.name for row in rows if row.type == "use_case_invoked"}
    assert invoked == declared, f"Use case coverage drift: {declared ^ invoked}"
    assert {row.name for row in rows if row.type == "domain_event"} <= node_ids
    assert all(row.payload["passed"] for row in rows if row.type == "invariant_result")
    assert rows[-1].type == "scenario_finished"
    if "--summary" in sys.argv:
        print(json.dumps({
            "scenario": scenario.name,
            "actors": [actor.name for actor in scenario.actors],
            "use_cases": sorted(invoked),
            "invocations": sum(row.type == "use_case_invoked" for row in rows),
            "domain_events": sum(row.type == "domain_event" for row in rows),
            "invariants": {row.name: row.payload["passed"] for row in rows if row.type == "invariant_result"},
            "finished_at": rows[-1].sim_time.isoformat(),
        }, sort_keys=True))
    else:
        print(observations.to_jsonl(), end="")
