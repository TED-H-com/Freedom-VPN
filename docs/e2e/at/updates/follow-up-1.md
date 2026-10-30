# Updates follow-up 1: real candidates and D-053

Run the suite with `make -f mk/at-updates.mk at-updates`. It creates only the
`w4-updates` stack on port base `42000`, writes all five records, then exits
nonzero for any blocked/failed scenario and tears the stack down. To reuse a
healthy, freshly prepared stack before first enrollment, add
`AT_UPDATES_REUSE=1`. Run `make -f mk/at-updates.mk at-updates-lint` and
`make go-test` for the area checks and offline tests.

The uppercase [AT23](AT23.md), [AT24](AT24.md), [AT25](AT25.md),
[AT26](AT26.md), [AT27](AT27.md), and [summary](summary.md) are the current
follow-up evidence. Lowercase records and `update-good-finding.*` are the
historical first pass. They are preserved as evidence of the original blockers.

## Scope and spec mapping

The orchestrator's follow-up labels name narrower scenarios than the original
§19 ATs. A follow-up pass proves only the scenario below. It does not clear the
full acceptance matrix.

| Follow-up label | Requested scenario | Product contract | Remaining original §19 coverage |
|---|---|---|---|
| AT23 | Real good candidate, local success, central confirmation | UPD01/UPD02, §13.4–§13.10, part of §19 AT23 | Physical arm64, three join sources, mixed publication states, all-node population |
| AT24 | Unconfirmed and needs_attention API evidence | UPD02, §13.12 | Offline/enrolling snapshots and later enrollment inheritance (§19 AT24) |
| AT25 | Byte-tampered real archive never executes | UPD02, §13.4, part of §19 AT25 | Wrong signatures/architecture, migration, low disk, interrupted download, usage continuity |
| AT26 | Genuine failed health causes last-good rollback | UPD02/UPD05, §13.9, part of §19 AT25 | Original §19 AT26 idempotency, race and stale-ACK matrix |
| AT27 | Interrupted apply resumes the durable transaction | UPD02/UPD03, §13.5–§13.6, part of §19 AT25 | Full reboot/power cut, original §19 AT27 minimum-serving and disabled-node matrix |

## Implementation and stimuli

`tests/e2e/at/updates/run.py` overlays the owned `live_test.go.txt` into the
shared `tests/e2e` package without writing a source file there. It invokes the
existing `runner.releases` builder, signer and product registration path for
real 0.1.0, 1.0.1 and 1.0.2 bundles on amd64 and arm64. No second release
pipeline is maintained. Staticcheck does not load virtual overlay sources,
so lint checks an ephemeral copy of that package under the owned harness path.
The old marker-bundle driver was removed; its broader historical observations
are not claimed as newly exercised.

The lab runs an unpublished node with a product-observed ready baseline.
The manifest pins the exact NetBird daemon token `0.82.0`. Runtime configs
use five-second heartbeat and poll periods and the existing real
`agent-service.sh` restart hook. Executable checks read the running process's
`/proc` link. Every record includes the run ID, source commit, signed-manifest
and envelope digests, and both artifact hashes. Only selected journal/API
fields are published; raw local support evidence stays in ignored scratch.

After establishing a real last-good (1.0.1 initially, then 1.0.2 after recovery),
the test sets only the 0.1.0 candidate's
`egress_probe_target` to its own closed loopback TCP port. This is a **TEST ONLY
lab stimulus**, not an injected health result. The candidate performs a real
failed dial and answers the challenge with `tunnel_probe=failed`. The restored
last-good config remains healthy and must answer a separate rollback challenge.
The config is restored after the experiment. The rollback case runs last so a
rejected rollback result cannot obstruct the earlier cases in the product outbox.
After normal runs the harness invokes the product `flush` command to deliver
preflight-failure results, since `run` can return before flushing those. No health attestation, transaction
stage or projector-owned row is written by the harness.

For tampering, the harness flips one byte in the real 1.0.2 archive, keeping its
size. It requires `HASH_MISMATCH`, no transaction, no installed candidate
executable and the previous real agent still running. Original bytes are then
restored through the same lab origin API.

For interrupted apply, a new batch authorizes those original 1.0.2 bytes. A
**TEST ONLY** stop-command wrapper stops the actual agent and sends SIGKILL to
its updater parent. The journal must still show `final_sampled`, forwarding
must remain blocked, and production `fvpn-updater recover` must complete the
same transaction with fresh D-053 health. This is process interruption on the
same boot, not a claim of physical power-loss durability.

## D-053 interpretation

An unpublished node can pass baseline health without opening member
forwarding. The candidate's own NetBird status, isolation and DNS decisions
answer the challenge. A configured unreachable egress probe must fail;
`not_run` also fails. No config value or prewritten health file is accepted as
proof. A good update still requires a later central observation to clear
`unconfirmed`; accepting a terminal result alone is not enough.

The current default unpublished tunnel probe checks the NetBird management and
relay status unless an egress target is configured. It does not prove member
traffic through a published exit. Physical forwarding and metering coverage
remain with the broader acceptance tasks.
