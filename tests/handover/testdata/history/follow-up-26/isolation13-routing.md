## Requests routed by the orchestrator

1. AT11 native platform matrix and AT13 full fault / platform matrix → carried operator rows (needs Ted's real resources); not re-asked.
2. Orchestrator observation: `docker stop` of the node containers ended in SIGKILL / exit 137 after 10 s (ordinary) and 20 s (clone fixture), so container lifecycle rows never exercise a graceful agent stop → devstack's next follow-up (harness signal forwarding, alongside devstack 25's agent exit-status readback).
3. The stale-lease alarm (3 Steps or 30 s) was not exercised; only a one-Step acknowledged hold was retained → isolation's next follow-up, with wire-nodes 34's central hold assertions.
