# Updates follow-up 1: real candidates and D-053

> Historical implementation and findings before D-062. Follow-up 2 retires
> the overlay and imports `tests/e2e.PrepareServingBaseline`. See the
> [current summary](summary.md) and [current requests](requests.md). Follow-up 5
> adds pre-result heartbeat injection and probe inspection after #155/#154.
> See [pre-result evidence](pre-result.md) and [probe finding](probe-issuance.md).

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

| Follow-up label | Requested scenario | Product contract | Remaining original §19 coverage | Follow-up 5 |
|---|---|---|---|---|
| AT23 | Real good candidate, local success, central confirmation | UPD01/UPD02, §13.4–§13.10, part of §19 AT23 | Physical arm64, three join sources, mixed publication states, all-node population | pass |
| AT24 | Unconfirmed and needs_attention API evidence; pre-result version authority | UPD02, §13.12 | Offline/enrolling snapshots and later enrollment inheritance (§19 AT24) | blocked — recorded boot prerequisite absent |
| AT25 | Byte-tampered real archive never executes; failed result needs a cause | UPD02, §13.4, part of §19 AT25 | Wrong signatures/architecture, migration, low disk, interrupted download, usage continuity | pass |
| AT26 | Genuine health failure, healthy last-good rollback, one current terminal ACK and later confirmation; revision 7 target evidence and bounded conflicting receipts | UPD02/UPD05, §13.9, part of §19 AT25 | Original §19 AT26 idempotency, race and stale-ACK matrix | pass |
| AT27 | Interrupted apply resumes the durable transaction; accepted release authority, job-bound confirmation and pre-result audit bounds | UPD02/UPD03, §13.5–§13.6, part of §19 AT25 | Full reboot/power cut, original §19 AT27 minimum-serving and disabled-node matrix | blocked — boot claim adds an audit without the requested recorded-boot baseline |

## Implementation and stimuli

`tests/e2e/at/updates/live_test.go` imports the shared
`tests/e2e.PrepareServingBaseline` pipeline. The retired follow-up 1 overlay
(`run.py` and `live_test.go.txt`) is no longer used. Real candidates are built,
signed, registered and supported through the shared helper; no second release
pipeline is maintained. Follow-up 4 adds explicitly requested TEST ONLY API
negative probes in `authority_test.go`.

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

## Historical requests to orchestrator

These follow-up 1 requests are resolved in the current run. The later direct
rollback transition request is resolved by #129; see [current requests](requests.md)
and the preserved [follow-up 2 capture](history/follow-up-2/README.md).

1. **w2-agent-core and w3-wire-fleet — carry the real update observation.**
   The good candidate reaches `locally_succeeded`, answers all five checks and
   receives a central `succeeded` result, but later heartbeats leave the target
   `unconfirmed`. `node/agent/observe.go:120` initializes update generation,
   job, stage and state to null. `internal/fleetupdate/serving.go:125` reads the
   null generation as zero; `serving.go:149` ignores that older generation once
   the accepted result has advanced fleet state. The confirmation condition at
   `internal/fleetupdate/progress.go:230` correctly requires a subsequent
   observation. Populate the authenticated observation from durable updater
   state; do not synthesize confirmation in this harness or weaken the central
   ordering check. The records include later node receipts alongside the stale
   fleet receipt and terminal result time.

2. **w4-node-updater and w3-wire-fleet — reconcile rollback result health and
   rollback allowlist semantics.** The failed candidate and restored agent both
   answer real D-053 challenges. Local rollback restores healthy last-good,
   but the result remains pending instead of completing the central target.
   `node/updater/apply.go:305` checks the restored release, while
   `node/internal/updatejournal/txn.go:145` deliberately preserves the first
   candidate health in `health_json`. `node/updater/apply.go:403` sends that
   candidate failure as the rolled-back result's health. Central requires healthy
   rollback evidence at `internal/fleetupdate/results.go:191`. Preserve the
   candidate failure for diagnosis and agree on how the restored release's
   successful checks reach central.

   Separately, the shared real-release builder uses `allowed_release_ids=[]`
   (`tests/e2e/harness.go:385`). The updater explicitly treats an empty list as
   no additional restriction (`node/updater/health.go:144`), while central
   unconditionally requires membership (`internal/fleetupdate/results.go:191`).
   Both conditions independently explain rejection of these rollback results.
   Align the product interpretation and, if needed, parameterize the shared
   release helper; this task did not fork or re-sign its manifests to hide the
   disagreement.

   Preserve the actionable central rejection in the updater's diagnostics:
   `node/updater/host.go:447` maps non-202/non-409 HTTP responses to
   `STORE_UNAVAILABLE`, and `node/updater/result.go:218` leaves the first rejected
   result pending and stops the flush. The live evidence distinguishes the HTTP
   status from this generic local error. This is not evidence of a database
   outage.

3. **w4-node-updater and w3-wire-fleet — accept honest preflight failure
   evidence.** The final make-target run's failed-download outbox payload has
   `health=null`. It records `ARTIFACT_INVALID` on the wire (the intentional
   normalization at `node/updater/revoke.go:291`); its local job retains
   `HASH_MISMATCH`. `node/updater/engine.go:699` constructs this failure without
   health. Central rejects every terminal result with absent health at
   `internal/fleetupdate/results.go:144`, returning HTTP 422. Agree on valid
   preflight-failure evidence without fabricating passed checks. The rejected
   first result stays pending, so later recovery and rollback results cannot
   be delivered by the normal sequential flush. Consequently the central
   target remains queued, `unconfirmed=true`, `needs_attention=false`.

The [delivery payload evidence](result-delivery.md) records the final run's
sanitized outbox fields. The separate [rollback diagnostic](rollback-delivery-finding.md)
comes from the earlier run, where rollback preceded tampering and independently
reached central rejection. This distinguishes the rollback contract mismatch
from the first pending failure's effect on delivery in the final run.

The unpublished node's original unconditional health-adapter blocker is resolved
by D-053. The remaining requests concern real observation and result delivery;
no `node/updater`, agent, central service or contract file was modified here.

## Follow-up 6 mapping

Run `a83d5d64-505c-488c-b560-5a777566702e`, source `1c8997d91a2c57a445fd25c62a0e4561fe5e6e8f`, evidence `2026-10-30T17:10:45Z`. Prefix `w4-updates`, port base 42000.

Fresh case c now passes (F5-01 resolved). Original AT23–AT27 scenarios and follow-up 4 extensions pass. Extended AT23/AT25 remain blocked on shared probe setup (F5-02 reopened); AT24/AT27 on the requested recorded c2/c3 expectations (F6-02). AT26 passes. Make exit 2. See [current summary](summary.md), [raw pre-result evidence](pre-result.json), [probe evidence](probe-issuance.json), and [requests](requests.md).

## Follow-up 7 mapping

Run `71078ab8-edf8-4d23-af72-ea247f704011`, source `c8d4a00cf1bf594b20016263c2cfc3c164d92bc9`, evidence `2026-10-30T18:09:36Z`. Prefix `w4-updates`, port base 42000.

AT24/AT26/AT27 pass under the restated accepted-result authority criteria. AT24 includes offline catch-up and a real post-batch node's own completed first_sync job. AT23/AT25 retain follow-up 6 blocked evidence for F5-02. F6-02 is resolved (#186). Make exit 2. See [follow-up 7 history](history/follow-up-7/README.md).

## Follow-up 8 mapping

The merged PR #212 evidence and its original run mapping are preserved in [history/follow-up-8](history/follow-up-8/README.md). See [the current summary](summary.md) for follow-up 9; historical pass claims do not carry forward.
