# Probe issuance — follow-up 6

Run `a83d5d64-505c-488c-b560-5a777566702e`, source `1c8997d91a2c57a445fd25c62a0e4561fe5e6e8f`, evidence `2026-10-30T17:10:45Z`. Prefix `w4-updates`, port base 42000.

[Raw configurations, captured challenges/attestations and fixed diagnostics](probe-issuance.json).

| Scenario | probe_peer | agent_config | Fresh challenges captured | Omission/refusal |
|---|---|---|---|---|
| AT23 | absent/null | `/etc/vpn-node/agent-run.json`, exists | 1 | none |
| AT25 | absent/null | `/etc/vpn-node/agent-run.json`, exists | 0 | none; HASH_MISMATCH before health |

Issued probe members (null means absent in the actual captured challenge):

```json
[
  {
    "scenario": "AT23",
    "issued_probe": null,
    "probe_omitted": null
  }
]
```

Devstack follow-up 10 (#170) configures the real agent path, but its dedicated
peer is created only by `runner.prepareProbe` in `tests/e2e/probe_scenario_test.go:45`,
called by the private update-good scenario at `tests/e2e/scenarios_test.go:550`.
`PrepareServingBaseline` and `Stack` do not expose that setup.
`tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:56-65` writes agent_config but no peer.
F5-02 remains open for an importable shared peer setup hook.

There is no issued peer, /32, destination socket or resolver flag to report.
D-064 installation/refusal is not exercised. The running agent answers the real
D-053 challenge without an exception; `node/agent/probeexc.go:43-45` handles the
absent probe directly. Lines 66-71 and 125-127 require a matching challenge and
another lock holder before installation. Attestation alone does not prove an
exception window. No `health challenge probe omitted` event or `probeRefusal`
class was emitted. F6-01 (`agent_config_unavailable`) does not apply: the
configured file exists. No file, peer or journal-health value was fabricated.
