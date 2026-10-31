# 2026-10-31 — D-095 / D-099 SECOPS-21 evidence handover

For **security-ops follow-up 12** and the orchestrator: D-095 holds the first post-restore compilation until a fresh observation from the current incarnation arrives. [PR #277](https://github.com/TED-H-com/Freedom-VPN/pull/277) supplies D-099's HTTP cache fix; see the [wire-nodes handover](../nodes/2026-10-31-secops-21-handover.md). The compiler is unchanged in this follow-up. **SECOPS-21 still needs live first-delivery, node-local store and applied-ACK evidence. The fixture's ACK is simulated.**

The follow-up 7 probe previously returned restored generation **2** after learning retained generation **5** and exited 1. Re-running that saved probe against base `510a4ab` (which includes #277) passes. Its permanent regression is `TestAT30_D095_HTTPRestoreCacheProbe` in [`internal/nodes/httpwire/restore_cache_probe_integration_test.go`](../../internal/nodes/httpwire/restore_cache_probe_integration_test.go). Each subcase uses an isolated `dbtest` database, production compiler and HTTP handlers, and fixture mTLS identity. Snapshot generations are TEST ONLY setup, not an actual backup/WAL restore.

The regression checks frozen and reconciling holds, including an old report replay after freeze. It also completes recovery before the fresh observation in one subcase. Each hold must return retryable **`503 DEPENDENCY_UNAVAILABLE`**, exact message **`pending fresh observation after restore`**, without signing or changing the desired generation, stored rows, envelope or ETag. The first conditional GET after the fresh observation must return 200 with a newly verified signature and ETag and exactly one new central desired-state row. Issuing that offer must leave applied generation at the retained value until the separate simulated observation ACK arrives. An unchanged conditional GET then returns 304 with no signing or writes.

All three regression subcases passed on the D-099 base:

| Fixture | Restored desired / stored maximum | Retained applied | Observed first HTTP offer / simulated ACK |
| --- | --- | --- | --- |
| Original follow-up 7 probe | 2 / 2 | 5 | 6 / 6 |
| SECOPS-21 generations | 11 / 13 | 14 | 15 / 15 |
| Recovery completes before fresh observation | 11 / 13 | 14 | 15 / 15 |

A test-only Go overlay of the pre-#277 HTTP cache files makes this same regression fail: the original probe receives stale **304 / generation 2** instead of 6; both SECOPS-21 cases receive stale **304 / generation 11** instead of 15. No production file was changed for this negative control. This establishes that the regression detects the former cache reuse defect.

The stale pre-#277 cache path reproduced by that overlay is `cachedStateSigner.Sign` in `internal/nodes/httpwire/state_cache.go`.

D-099's accepted interpretation is essential: `CompileTx` always proposes `max(desired, applied, maxStored) + 1`. The HTTP cache may reuse the cached generation or the ordinary one-step proposal only when no higher applied/stored fence exists and no restore observation is outstanding. Otherwise it re-signs and replaces the cache. Strict proposal equality would disable ordinary cache reuse. No new compiler decision API is introduced here.

The live rerun must preserve this ordering and evidence (§14.7, §17.3–§17.5, DATA01 / MTR03, AT30):

1. Stop control and worker, restore the T0 database, then successfully run freeze-ingest (`fvpn-usage-admin ... freeze`) **on the restored database before starting either service**. Follow [restore Step 1](../usage/restore-runbook.md). Capture successful freeze and subsequent startup times; freezing after startup leaves the original race open.
2. Start services and begin recovery. Before delivering any fresh current-incarnation observation, capture `GET /api/v1/agent/desired-state`, including a conditional request with the retained ETag. It must return the exact 503 hold above. A cached 200 or 304 is a failure. Capture the node detail's `desired_state_restore_observation` drift entry and the operation's pending `fresh_observation` step. No generation, envelope or signature may advance during the hold.
3. Record the retained T1 node's local accepted/applied generations and incarnation before releasing its observation. Deliver a fresh current-incarnation report with applied **14**, centrally received strictly after the freeze cutoff. Capture the **first delivered HTTP offer**, not just a later compiler result: restored desired **11** / stored maximum **13** / retained **14** must produce **15**, with a new verified signature and ETag. Preserve every offer; do not hide a stale first offer with later successful ones or adjust generations in the harness.
4. Read back the retained node's local store after delivery. Its accepted generation, ETag and envelope must match the delivered **15**. Then capture successful local application and the node's subsequent observation ACK with applied **15**, and central observed/applied **15**. A completed `fresh_observation` step proves receipt freshness only. Absence of `ErrStaleGeneration` log lines alone is insufficient because the agent can ignore that error silently. An unchanged conditional GET should now return 304 without another generation or signature.
5. Repeat with `complete-recovery` before the fresh report, including a node never visited by the compiler during recovery. It remains held. A report received between freeze and begin remains fresh after completion; duplicate old reports and other-incarnation reports must not release the hold. Existing `TestAT30_D095` compiler tests cover these ordering cases and a fresh receipt whose node clock moved backwards.

Do not close SECOPS-21 from the compiler tests alone. Security-ops follow-up 12 must establish closure from this live rerun.

With the admin test DSN already configured, run the deciding tests using disk scratch:

```bash
mkdir -p .agent/tmp
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
export GOTOOLCHAIN=local CGO_ENABLED=0 GOMAXPROCS=4 GOFLAGS=-p=4
go test -v -count=1 -tags integration ./internal/nodes/httpwire -run '^TestAT30_D095_HTTPRestoreCacheProbe$'
go test -v -count=1 -tags integration ./internal/desiredstate -run TestAT30_D095
```

`dbtest` creates and drops each isolated database. This lane changed only the regression and this handover: no compiler, HTTP production source, contract, migration, dependency or observation-writer change. Route the live rerun through the orchestrator under D-102; new requests stay in this lane's dated report. Do not close live SECOPS-21 or claim measured RPO/RTO from these fixtures.
