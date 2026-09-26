from mrl_simulation_runtime.runner import SimulationRunner

from app.simulation.mrl_runtime_scenario import create_simulation


if __name__ == "__main__":
    print(SimulationRunner().run(create_simulation()).observations.to_jsonl(), end="")
