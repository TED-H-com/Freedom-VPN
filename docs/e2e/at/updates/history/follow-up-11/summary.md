# Updates follow-up 11

Run: daddad24-b6f7-4c32-ba8e-1c7d24f76ea2

Source: 3ac13f7e115edf22e2ec9d3a6450733ec1fb7b6a

These labels follow the orchestrator's follow-up scenario list. They do not redefine the broader §19 cases. Physical arm64 execution and the full original §19 population remain outside this run. Both architectures contain compiled real binaries.

| Scenario | Result | Observation |
|---|---|---|
| [AT23](AT23.md) | pass | Real 1.0.1 candidate answered D-053 with exported D-063 probe peer issued in health-challenge.json and bounded D-064 exception verified; locally_succeeded, then central succeeded and confirmed |
| [AT24](AT24.md) | pass | Offline target caught up through a real success result; a later new node inherited 1.0.2 through its own successful first-sync job; original snapshot unchanged |
| [AT25](AT25.md) | pass | Byte-tampered archive produced HASH_MISMATCH at preflight before any transaction, candidate executable, or health challenge; null-health failed result accepted; probe scope does not apply to an archive that never passes preflight |
| [AT26](AT26.md) | pass | Candidate tunnel_probe failed; healthy last-good restored; one terminal rollback received a current ACK; central rolled_back, needs_attention=true and unconfirmed=false |
| [AT27](AT27.md) | pass | Fresh unexpected release audited once; accepted releases never seeded or audited by that path; mismatches withhold confirmation; stale ACK replay preserves the newer job |

See [current requests](requests.md) and the [follow-up 10 archive](history/follow-up-10/ARCHIVE.md).
