# Probe issuance inspection — follow-up 5

Run `743c186d-5765-43e9-8b40-7839be178e13`, source `dd72ed4e43e2be21a852e6741d8caa06a8f6ad48`, at `2026-10-30T08:35:19Z`.
Exact safe config fields read in the node, any captured fresh challenge, and
fixed diagnostic classes are in [probe-issuance.json](probe-issuance.json).

| Scenario | probe_peer | Default agent config | Omission diagnostic | Result |
|---|---|---|---|---|
| AT23 | absent | `/etc/vpn-node/agent.json` absent | none | no probe peer in lab |
| AT25 | absent | `/etc/vpn-node/agent.json` absent | none | no probe peer in lab; pre-health hash rejection issues no fresh challenge |

This main includes devstack follow-up 9 (#153), not follow-up 10. Its generator
writes `/etc/vpn-node/updater-recover.json` without probe_peer/agent_config and
runs the agent using `/etc/vpn-node/agent-run.json`. We preserved the lab inputs
and did not create a probe or alias agent.json. With no peer, updater f10 does
not read these address inputs, so `agent_config_unavailable` was **not an observed
log class**. It is the expected refusal if a peer is added without supplying
the real agent_config path. See request F5-02 for the devstack owner.
