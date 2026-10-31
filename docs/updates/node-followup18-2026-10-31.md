# Updater follow-up 18 — 2026-10-31

Devstack 19 (#332) and updates 10 (#337) close the two live hand-offs from
[follow-up 17](node-updater.md#filegate--filemeter-handshake--2026-10-31-follow-up-17-d-125)
within their recorded scope. The serving-node campaign is **14/0/0**, initial
apply **0**. The installer → updater → central extension is **23/23**.
Neither count means the regular AT23–AT27 campaign passed: updates 10's regular
AT23 failed to start the candidate and AT24–AT27 remained blocked (F10-01).
This audit starts from main `57a8cc9f`, including #323, #332 and #337.

## Evidence and D-132 citation policy

- **S19:** [serving upgrade JSON](../e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-journaled-upgrade.json)
  and [readable record](../e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-journaled-upgrade.md),
  run `d6aaa798-6c39-49f1-8fa9-5f84f696ac91`, source `a9ad1c47`.
- **R19:** [candidate re-enrollment JSON](../e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-reenroll.json)
  and [readable record](../e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-reenroll.md), same run.
- **N19:** [stopped-agent JSON](../e2e/runs/2026-10-31/followup19/04de29d3-e56b-4a32-b678-21462845ddb6/installer-quiescent-candidate.json)
  and [readable record](../e2e/runs/2026-10-31/followup19/04de29d3-e56b-4a32-b678-21462845ddb6/installer-quiescent-candidate.md),
  run `04de29d3-e56b-4a32-b678-21462845ddb6`, source `fb5b8fce`, campaign **13/0/1**.
- **U10:** [installer-handoff.json](../e2e/at/updates/installer-handoff.json),
  [installer-handoff.md](../e2e/at/updates/installer-handoff.md), and
  [follow-up-10.md](../e2e/at/updates/follow-up-10.md), run
  `695f977c-b587-4768-88f1-0514f3a4f1ac`, harness `0df8599d`, product main
  `fb5e0025`. These are living root paths, cited as observed on 2026-10-31;
  **no file digest is pinned and no updater test reads them**. A later archive
  must be supplied by their owner before a digest-dependent test may consume them.

Only the immutable UUID-run JSON files above are SHA-256 pinned here:

| Evidence file | SHA-256 |
|---|---|
| S19 `installer-journaled-upgrade.json` | `83f56c2e808bbdbfa1555408b570d0d89b05cbf8a6a32802c2c47e0e8006e873` |
| R19 `installer-reenroll.json` | `6bb99daf1f9ed5fac5d63794ff0f4b2582fea42f280e6823f794eeb0925e5a74` |
| N19 `installer-quiescent-candidate.json` | `665042c306439ac0cb36c021d9f1a32dffa26ce13c9949684b3da81b901c9cdc` |

## Follow-up 17 claim reconciliation

The requirement mapping is UPD02/UPD03/UPD05, MTR03 and API01;
[spec §13.5–§13.9](../spec/PROJECT_SPEC.zh-TW.md#135-updater-隔離目錄與本機交易)
and §13.10.3 govern the transaction and receipt. AT25/AT26/AT27 are the main
acceptance cases, with AT16 accounting continuity and AT23 platform scope.

| Claim | External evidence and measured result | Remaining limit |
|---|---|---|
| D-125 `barrier_id` byte-equals the persisted block identity | S19 `handshake_trace`, `correlated_checkpoint`, `correlated_with_original_boot_deadline`: exact match after live member allow-set withdrawal; authenticated control still 200. U10 `correlated_checkpoint`, `correlated_wait`, `check_correlated`: exact match with the real agent alive. | Equivalent UUID spellings, foreign/malformed identities and legacy four-member evidence are component-tested, not live fault injections. |
| Fresh correlated FileMeter evidence permits apply | S19 absent-before → new checkpoint → candidate pointer and actual executable switch → health 5/5 → sealed `succeeded`, exit 0. U10 `check_apply`, `check_restarts`, `check_central`: sequence 10 → 11, five health checks, `source=installer`, `batch=null`, central 202 with current=last-good=11 and the D-110 job ID recomputed. | U10 is quiet/unpublished; serving withdrawal comes from S19. Neither result grants publication. |
| Late, within-bound correlated publication permits apply | S19 original wait starts 14:07:31.266673717Z, deadline 14:08:01.266673717Z; correlated checkpoint observed 14:07:34.708073795Z. The real delayed response is within that original window and applies successfully. `TestD125_RuleCorrelatedLateWithinBound` separately covers a deliberately delayed producer. | Natural latency is measured; a controlled near-deadline live injection and worst-case scheduler/filesystem bounds remain 待實測. |
| Finalization preserves genuine epoch/watermarks and avoids fabricated usage | S19 epoch closes `final_sample` at 14:07:34.685593870Z; durable sequence 4 / ACK 3 and closed epoch match the sealed receipt, with a distinct new epoch. U10 finalization and receipt agree at 0 / 0. | S19 final sample is quiet and adds no new event; `final_sample_event_verified` proves closure plus the hash-verified prior watermark event. The component producer proves one 123-byte event; do not claim those bytes were measured in S19. |
| Invalid/missing checkpoint fails closed before service stop/switch | N19 has no checkpoint before/after the wait: sealed `failed/final_sample/CHECKPOINT_FAILED`, null health/usage, baseline retained. U10 stops the producer and withdraws its checkpoint before apply: same failure, sequence 11 retained, forwarding disabled, central accepts failed evidence 202. | Malformed JSON, special files, oversized snapshots, invalid epochs/watermarks and ACK-ahead cases remain component coverage (`TestUPD03_FileMeterRejectsInvalidSnapshot`), not live measurements. |
| Foreign or other-boot evidence never authorizes switch | Follow-up 17 `TestD125_RulePreviousBlockIdentity`, `RuleByteEqualUUIDSpelling`, `RuleCorrelatedPriorBootRejected`, plus restart tests retain the baseline and null usage receipt. U10 confirms matching kernel/agent/seed/job/lock boot identity on the positive path. | S19/N19/U10 do not inject a foreign identity or reboot mid-wait. Those negative live claims remain 待實測. |
| Withdrawal discards pending/previously sampled evidence | U10's `withdraw_checkpoint` and `check_checkpoint_failure` measure checkpoint absence before apply. Follow-up 17 `RuleWithdrawnBeforeCorrelatedPublication` and `RuleFinalSampledRestartDiscardsWithdrawnCheckpoint` cover latch withdrawal; `meter_barrier_test.go` covers unblock/reblock. | Removing a checkpoint before apply is not withdrawing the latch after an accepted sample. The latter returns `STATE_CONFLICT/meter_barrier_withdrawn` in components and remains 待實測 live. |
| Mid-handshake restart preserves the original deadline and rereads evidence; identical republication accounts once | `TestD125_RuleRestartRechecksCorrelatedFile`, `RuleFinalSampledRestartRechecksFile`, `RuleIdenticalRepublicationAccountsOnce` and `RuleCorrelatedAfterBound` prove these component paths, including refusal to reopen a timed-out job. N19/U10 measure sealed terminal replay only. | A crash at `blocked`/`final_sampled`, same-identity producer republication, and valid publication after deadline are not externally injected by these runs; live measurement remains open. |
| Sealed nondestructive failure exits apply 1 / replay 1 / recover 2 | N19 `apply_exit`, `failed_apply_replay_exit`, `failed_recover_exit`: **1/1/2**, no switch, blocked latch, `recovery_required=false`. U10 restores exact pending bytes and measures **1/2** replay/recover with the same failure. | Matches follow-up 17. This is not destructive recovery-required injection. Local journal stage `rolled_back` describes a nondestructive completed transaction; wire outcome remains `failed`. |
| Sealed replay does not duplicate or alter the receipt | N19 `failed_recovery_seal_byte_identical`, unchanged coherent journal snapshots; U10 `check_failure_replay`, receipt cardinality and central read-back. Successful same-key/new-key exact replays are **202/202**, ETag stays **"8"**; modified receipt is **409 ACK_CONFLICT**, wrong job **409 STATE_CONFLICT/job_id_mismatch**. | HTTP 202 is durable receipt acceptance; it is not new execution or a new forwarding claim. |
| D-127/r16 M-2 permits one later rollback correction while preserving first receipt | U10 `correction_scope`, `rollback_correction`, `central_installer_history`: **TEST ONLY HTTP stimulus** from real success accepted 202; central current=last-good moves 11 → 10, `rolled_back`, ETag **"9" → "10"**. First/correction replays **202/202**, third changed outcome **409 ACK_CONFLICT**, correction ETag stable. | Central correction handling is externally measured. This is not a physical rollback: local current remains sequence 11. Regular AT26 rollback was blocked by F10-01. |
| Post-upgrade seed has no extractor-induced drift | R19 real revoke/re-enrollment seed exits 0 without `HASH_MISMATCH` or tree repair. U10 default-umask seed and exact replay under 077 both exit 0, same tree hash/timestamp. | Resolves the pre-#307 seed hand-off within these runs; seed still refuses actual drift. |

U10 additionally measures tampered sequence-13 archive `HASH_MISMATCH` before
switch (apply **2**), sequence-10 downgrade refusal (`DOWNGRADE`, upgrade **1**),
and central D-110 job recomputation. The tamper refusal's exit 2 is a different
path from the sealed nondestructive final-sample failure's exit 1; no discrepancy
with the follow-up 17 claim was found.

## Findings and ownership — 2026-10-31

| Finding | Expected versus measured | Owner / disposition |
|---|---|---|
| F18-01: stopped-agent legacy scenario cannot succeed | N19's old candidate-success scenario is still recorded as a failure (13/0/1). Its actual 1/1/2 exits and sealed failed receipt agree with D-125 and follow-up 17. A stopped producer cannot answer a new identity; do not rewrite the historical row as a successful upgrade. | Devstack owns scenario classification/expectation. Updater and agent-core retain fail-closed behavior. Serving success is separately closed by S19. |
| F10-01: regular AT candidate layout | U10 regular AT23 expected candidate startup, but got `agent: active release identity is inconsistent`, `recovery_required/ROLLBACK_FAILED`, no health challenge, and no verified last-good; AT24–AT27 blocked. | Devstack 20 owns the shared fixture/service layout; updates owns the subsequent regular acceptance rerun. Outside this lane. |
| F10-02 / F18-02: generated reporting absent; library delivery silently skipped | U10 generated `result_url` was empty; its explicit endpoint/CA/origin reporting adapter proved central acceptance. Before this change, CLI `flush` already exited 1, but `Engine.Flush` returned nil for a nil sink and sealing skipped delivery silently. | Installer 7 owns rendered settings. Updater fixes the silent library path here: `Flush` returns `VALIDATION_FAILED` / exit 1; sealing logs `update result pending`, reason `result_url_missing`. Receipt bytes/status remain pending; later configuration repair delivers once. |

No other discrepancy between follow-up 17's scoped exit/receipt expectations and
S19/N19/U10 was found. The former devstack 18 exit-0 failed receipt remains
historical evidence from older code; these new runs measure the corrected path.

## Release-root audit

The executable contract is `<release>/fvpn-agent`, checked by
[`agent.loadActiveRelease`](../../node/agent/release.go) (line 40 on this base),
and used by [`installer.Layout.agentBin`](../../packaging/installer/layout.go).
No production updater candidate/switch/health path assumes `bin/fvpn-agent` or
adds another level below the release root:

| Path inspected | Layout behavior |
|---|---|
| [`layout.go`](../../node/updater/layout.go), `releaseDir`, `relativeRelease` | Uses releases/release-ID and points current directly at that directory. |
| [`engine.go`](../../node/updater/engine.go), `installTree`; [`pending.go`](../../node/updater/pending.go), installer staging validation | Verifies the signed artifact and immutable whole tree; extracts into the release root. Installer staging cannot supply a replacement execution path. |
| [`extract.go`](../../node/updater/extract.go), `ExtractArchive`; [`seed.go`](../../node/updater/seed.go), `verifySeedTree` | Preserves safe archive-relative paths and compares whole-tree hashes; no bin-prefix insertion, fallback lookup or tree repair. Extraction is generic and does not itself certify that an archive contains a runnable agent. |
| [`apply.go`](../../node/updater/apply.go), `stepSwitch`, `stepStart`, rollback | Fsyncs and atomically switches the release directory; gives that same root to `Services.Start` on candidate and rollback. |
| [`host.go`](../../node/updater/host.go), `CommandServices.Start` | Appends the root to the configured start command. Commands/systemd units determine execution; the updater does not invent a binary location. |
| [`health.go`](../../node/updater/health.go), `viewFor`, `checkProcesses`; [`attest.go`](../../node/updater/attest.go), `NodeChecks` | Hashes the root tree and requires challenge-bound agent/component evidence in production. The actual running-agent identity check remains agent-owned. |
| [`packaging/release`](../../packaging/release), `PackageDir` | Packages the supplied tree without relocating executable paths. |

The updater's own shared `writeMarker` fixture and exact-installed-size archive
test did use `bin/fvpn-agent`. They now use the product root layout.
`TestAT25_ReleaseRootCandidateSwitchAndHealth` checks a staged candidate, the
current-root executable, service start root, and health-hash failure after root
binary drift. Service/candidate attestation adapters remain fixtures; S19 supplies
the real process/health proof. Generic extraction tests intentionally keep `bin/`
examples for path traversal/link rejection; they do not define product layout.
No installer or devstack file was changed.

## Reporting settings installer 7 must render

[`Config.Engine` and `HTTPSink`](../../node/updater/host.go) use the following
JSON fields. `AllowedOrigins` is an internal Go field, **not** a JSON config key;
unknown config keys are rejected by `LoadConfig`.

| Rendered field | Required meaning for real delivery | Absent / invalid behavior |
|---|---|---|
| `result_url` | Full central endpoint, normally `https://control.example.com/api/v1/agent/update-results`, for both fleet and installer receipts. Do not use only the origin or append the path twice. | Empty creates no sink. Local apply/seed/recovery remain possible; local success never means delivered. Sealing warns with `result_url_missing`; explicit CLI/library flush fails, leaves the original pending receipt and creates no ACK history. Invalid nonempty URLs fail engine construction. |
| `allowed_artifact_origins` | Exact `scheme://host[:port]` origins when the central result endpoint resolves to private/CGNAT addresses. The same list goes to `HTTPSink.AllowedOrigins` and artifact/manifest downloads. Include the result origin when distinct from the artifact origin. | Empty retains the private/CGNAT guard. No wildcard/subnet match; no path, userinfo, query, fragment or trailing slash. Match is scheme + hostname + effective port (443/80 defaults). The list does not supply CA trust. |
| `allow_loopback` | `false` in production; `true` only for TEST ONLY local fixtures. | False rejects loopback even when the origin is listed. True permits loopback destinations and permits HTTP only for loopback names; it does not admit arbitrary private addresses or HTTP for a private origin. |
| `mtls_cert`, `mtls_key` | Paths to this incarnation's issued client certificate and node-held private key. Real central reporting requires the authenticated node identity; neither key bytes nor tokens belong in `result_url`. | A half-pair or unreadable/invalid pair fails engine construction. Both empty supplies no client certificate; an authenticated endpoint must reject that client. No fake acceptance is produced. |
| `mtls_ca` | PEM CA bundle that verifies the **result endpoint's server** certificate/hostname. It must cover the actual central listener; do not assume the node-certificate issuer also signs that server. The same client is used for artifact/manifest retrieval. | Empty uses system roots, not insecure TLS. An explicit path builds a new root pool from that PEM (replacing the system pool); missing/unparseable PEM fails engine construction. Untrusted server/hostname/handshake failure leaves delivery pending. |

[`checkURLPolicy`, `dialPolicy`, `validateOrigins`](../../node/updater/verify.go)
enforce the policy at configuration/submission and at DNS resolution/connection;
[`clientWithDial`](../../node/updater/engine.go) also checks redirects. Metadata
hosts, link-local, multicast and unspecified addresses stay denied even with a
listed origin. The guard pins a permitted resolved address for dialing.
`NewMTLSClient` uses normal certificate/hostname verification, TLS 1.2 minimum
for its configured TLS transport, and a 30-second client timeout. It never sets
`InsecureSkipVerify` or installs release/desired-state signing keys as TLS CAs.

`HTTPSink` sends POST JSON with `Idempotency-Key` equal to `result_id`.
Transport/TLS failures never call `NoteAck`; the receipt stays pending.
Authentication 4xx responses are retained as rejections, 409 as conflict (the
D-110 job mismatch is a rejection), and 429/5xx remain retryable. Only a parsed
202 ACK reaches local acceptance history. Correcting missing endpoint settings
must flush the existing sealed payload, not rebuild the job or receipt.
`TestUPD05_MissingResultURLKeepsReceiptPending` proves this repair sends once;
`TestUPD05_ResultConfigPolicyAndServerTrust` checks endpoint wiring, explicit CA
trust, system-root refusal of the local test CA, absent CA file, and loopback
policy. Real client-certificate authentication is evidenced by U10, not by that
local TLS fixture.

## Load check and D-132 test audit

Agent-core 27's requested negative subtest is distinct from the successful
correlated-late test: `TestAT25_FileMeterWaitFailsClosedWithoutCorrelation/late_within_bound`
publishes an **uncorrelated** checkpoint during a 250 ms wait and must refuse it.
The initial `-count=20 -race` run passed **20/20**, zero failures/race reports,
package time **9.428 s**. Concurrent `go test -count=1 ./internal/...` passed
**33 packages**, 13 packages had no tests, zero failures. Both process exits were
0. No sleep/deadline change was needed; `FileMeter.Wait` and its fail-closed
checks are unchanged. The final repeat after root-fixture alignment also passed
**20/20**, zero failures/race reports, in **9.405 s**; the concurrent internal
suite again passed **33 packages**, with 13 no-test packages and zero failures.
Both exits were 0: **40/40 iterations across the two runs**, with no fixture
schedule or production wait change.

```sh
mkdir -p .agent/tmp
export TMPDIR=$PWD/.agent/tmp GOTMPDIR=$PWD/.agent/tmp GOMAXPROCS=4 GOFLAGS=-p=4
pgrep -af 'gates.sh|mk/at-' # wait while two other suites are active
CGO_ENABLED=0 go test -count=1 ./internal/... > .agent/tmp/load-internal.log 2>&1 &
load_pid=$!
CGO_ENABLED=1 go test -v -count=20 -race \
  -run '^TestAT25_FileMeterWaitFailsClosedWithoutCorrelation$/^late_within_bound$' \
  ./node/updater
meter_rc=$?
wait "$load_pid"
load_rc=$?
printf 'meter_exit=%s internal_exit=%s\n' "$meter_rc" "$load_rc"
```

The only background process is the requested concurrent load in the same shell,
which is waited before that shell finishes. Scratch is disk-backed in the worktree;
there are no cleanup traps or containers in this lane's run.

Recursive audit of `node/updater/**/*_test.go` (36 files including this follow-up)
and `node/internal/updatejournal/**/*_test.go` (8 files) found **zero `docs/e2e/`
reads**, including constructed `docs`/`e2e` path fragments. File-read and path-join
sites use generated fixtures, temporary stores or contract vectors. In particular,
`TestF903_Devstack18SealedServingFailureCLI` synthesizes the old receipt shape with
generated identities; its comment cites a run but it does not read that run.
No test pin or evidence path needed changing. Future AT evidence reads must use
`docs/e2e/at/<lane>/history/<follow-up>/`, never living AT root files (D-132).

## Gates and remaining measurement

All required gates passed; the worktree hand-off report retains the real output
tails and logs under `.agent/tmp/f18/`:

| Gate | Result |
|---|---|
| `make -f mk/updater.mk updater-lint updater-test updater-integration updater-build` | Exit 0; pinned staticcheck, local tests, real central interoperability, linux/amd64 + linux/arm64 updater/release builds. |
| `go test -count=1 ./node/updater/... ./node/nodelock/... ./packaging/installer/... ./internal/installerjob/...` | Exit 0. |
| `go test -count=1 -run 'TestMTR03_UPD05_DurableFileMeterBarrier' ./node/agent/...` | Exit 0; unchanged agent test, 0.131 s. |
| `make go-lint 'go-with-disk-tmp=set -eu;'` | Exit 0; whole-module gofmt/vet/pinned staticcheck. The helper override uses the already exported disk scratch without the cleanup trap in unowned `mk/go.mk`; the checks are unchanged. |
| `go vet ./...` | Exit 0. |
| Final load repeat above | 20/20; concurrent internal suite 33 packages passed; both exits 0. |
| gofmt, `git diff --check`, owned-path/public-hygiene audit, local doc links/digests | Clean; no forbidden hygiene matches; 3/3 immutable digests agree. |

Integration used `dbtest` per-test databases from the provided DSN (sweeping
unset); the shared PostgreSQL was not reconfigured. No dependency, contract,
migration or unowned product path changed.

**待實測:** the 30 s default wait and 100 ms polling cadence as operational bounds
under worst-case CPU/storage load; near-deadline live injection; physical arm64
execution; filesystem/power-loss durability; physical D-127 rollback; the remaining
live negative-correlation cases above; and real fleet rollout/mixed-population
AT23–AT27. One observed 30-second timeout and one successful delayed response do
not establish those timing bounds. Cross-building arm64 does not establish real
arm64 forwarding/metering/updating. No production value or capability is inferred
from these TEST ONLY runs.

### Requests to orchestrator

- **2026-10-31 — installer 7 / F10-02:** render the endpoint, exact result-origin
  policy and endpoint-server CA/client identity described above in
  `packaging/installer/config.go`; then measure installer-generated reporting
  without the U10 reporting adapter. That path is outside this lane and was not
  edited. Consume F18-02's explicit pending diagnostic/library flush error.
- **2026-10-31 — devstack 20 + updates / F10-01:** align the shared release fixture
  and service manager to root `fvpn-agent`, then rerun regular AT23–AT27. Their
  existing blocked rows are retained; the installer extension does not replace them.
- **2026-10-31 — devstack / F18-01:** keep the stopped-agent failure evidence and
  treat its 1/1/2 sealed failure as the expected negative D-125 behavior in any
  future scenario. Do not report that historical campaign as passing an upgrade.
- **2026-10-31 — acceptance owners:** retain the remaining measurement items above.
  Archive U10 under its `history/` tree before requesting a digest-dependent
  consumer test; this lane has no mutable AT evidence dependency to repair.
