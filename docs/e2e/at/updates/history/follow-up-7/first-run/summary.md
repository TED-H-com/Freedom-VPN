# Updates follow-up 7

Run: 008a155b-a94f-4c36-8cbb-a574258c678b

Source: c8d4a00cf1bf594b20016263c2cfc3c164d92bc9

These labels follow the orchestrator's follow-up scenario list. They do not redefine the broader §19 cases. AT23/AT25 retain their follow-up 6 evidence and are not rerun. Physical arm64 execution and the full original §19 population remain outside this run. Both architectures contain compiled real binaries.

| Scenario | Result | Observation |
|---|---|---|
| [AT23](AT23.md) | blocked | Retained follow-up 6 evidence; not rerun in follow-up 7. Blocked on F5-02; owner: w3-devstack-e2e follow-up 12 (probe-peer export) |
| [AT24](AT24.md) | pass | Offline target caught up through a real success result; a later new node inherited 1.0.2 through its own successful first-sync job; original snapshot unchanged |
| [AT25](AT25.md) | blocked | Retained follow-up 6 evidence; not rerun in follow-up 7. Blocked on F5-02; owner: w3-devstack-e2e follow-up 12 (probe-peer export) |
| [AT26](AT26.md) | pass | Candidate tunnel_probe failed; healthy last-good restored; one terminal rollback received a current ACK; central rolled_back, needs_attention=true and unconfirmed=false |
| [AT27](AT27.md) | blocked | Fresh unexpected release audited once; accepted releases never seeded or audited by that path; mismatches withhold confirmation; stale ACK replay preserves the newer job; required follow-up 7 check incomplete: stale_ack |

See [current requests](requests.md) and the [follow-up 2 AT26 history](history/follow-up-2/README.md).
