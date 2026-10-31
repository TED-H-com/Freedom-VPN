# Installer follow-up 7 — 2026-10-31

F10-02 is fixed in the installer: rendered updater reporting uses the supplied,
validated HTTPS control origin, the API's existing update-result route, the
server CA that authenticates enrollment, and the enrolled mTLS client identity.
No contract, migration, updater implementation or production endpoint is added.
Base: main `57a8cc9f` (updates 10 / #337). The configuration and migration behavior
are documented in [install.md](install.md#authenticated-updater-reporting-f10-02-136-13103).

## Follow-up 6 request 1: accepted external serving rerun

Devstack 19 (#332), source `a9ad1c47`, run
`d6aaa798-6c39-49f1-8fa9-5f84f696ac91`, records **14 pass / 0 blocked / 0 fail**,
initial apply exit **0**. This closes installer follow-up 6's request for the
combined-source serving-node upgrade rerun, within the following evidence scope.
Earlier failed attempts in [follow-up 6](install-followup6-2026-10-31.md) remain
failed; this is a later acceptance, not a relabeling of those runs.

| Follow-up 6 claim/request | Devstack 19 evidence and disposition |
|---|---|
| ENR-05 marker migration plus strict serving verification | [Marker migration][marker] and [serving read-back][serving]: hookless installer owner in `inet fvpn_install`; four agent-rendered chains in `inet fvpn`; healthy installer check, active services, authenticated central observation and native member traffic. Accepted for this serving fixture. |
| Same-incarnation held usage, idempotent wrapper and restored traffic | [Usage][usage] and [wrappers][wrappers]: held real nonzero usage, byte-equal quiescent snapshot across wrapper, services restored, native traffic resumes and held events drain. This is not retired-stream delivery acceptance. |
| Verified staging without early pointer switch | [Journaled upgrade][upgrade]: verified retained manifest/request and unchanged current after staging; subsequent candidate pointer and live executable switch are separately recorded. |
| Member withdrawal, control availability and D-125 correlation | [Journaled upgrade][upgrade]: member allow-set populated/control HTTP 200 before the latch; allow-set empty/control HTTP 200 after the barrier. Checkpoint `barrier_id` equals persisted wait identity `b455cf8d-3a77-4909-a8a7-c60f9ed89be2` byte-for-byte within the original boot/deadline. No injected checkpoint. The sampled stale heartbeat is retained separately from nft withdrawal evidence. |
| Durable finalization, sealed accounting and candidate health | [Journaled upgrade][upgrade]: epoch closed with `close_reason=final_sample`; canonical last durable event sequence **4**, ACK **3**, matching the sealed receipt and a distinct new epoch. The final sample is quiet: no new event/bytes are claimed after withdrawal. All five health checks pass, journal `locally_succeeded`, state `succeeded`, initial apply **0**. Publication remains disabled and member forwarding blocked. |
| Post-switch installer check preserves install history | [Journaled upgrade][upgrade]: current candidate comes from `release_source=updater_journal`; the original install record remains separate. |
| Candidate revoke → re-enrollment → seed, without tree repair | [Re-enrollment][reenroll]: central revoke, same-node new incarnation, candidate seed exit **0**, no `HASH_MISMATCH`, no tree repair. Retired identity and bound state remain, canonical origin events are byte-identical after D-118 adoption, services restore, old certificate gets **403** directly and through proxy. The intentionally stopped seed check reports inactive-unit drift; that is distinct from the successful seed subprocess and later restoration. |
| Failure stays failed; exact sealed replay/recovery does not mint receipts | [Stopped-agent candidate][stopped], run `04de29d3-e56b-4a32-b678-21462845ddb6`, source `fb5b8fce`: **13/0/1**, suite exit **1**. Stopped producer cannot answer the fresh latch: `CHECKPOINT_FAILED` at `final_sample`, null health/usage, no switch, no recovery-required destructive failure. Initial and sealed replay apply **1**, recover **2**, latch retained. Coherent journals before/after are byte-identical, as is the sealed receipt. Its later seed verifies the unchanged initial tree, not the candidate. |

[marker]: ../e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-marker-migration.json
[serving]: ../e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-serving-readback.json
[usage]: ../e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-usage.json
[wrappers]: ../e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-wrappers.json
[upgrade]: ../e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-journaled-upgrade.json
[reenroll]: ../e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-reenroll.json
[stopped]: ../e2e/runs/2026-10-31/followup19/04de29d3-e56b-4a32-b678-21462845ddb6/installer-quiescent-candidate.json

### Immutable evidence digests (D-132)

Only the UUID-directory JSON files linked above are pinned here. SHA-256:

| Evidence | SHA-256 |
|---|---|
| [Marker migration][marker] | `d7d689a7e9fde5148bfb96df8ff7202be8745faa64e533b46831f1cd200e5064` |
| [Serving read-back][serving] | `d80db255a42b07eb5e445374f9e790b3e472153e420bf1ec97bf9d4927b4a3fa` |
| [Usage][usage] | `c3369a5576d8c9de1b480011bd98753d622afa729b9f749d576bf95167b19b30` |
| [Wrappers][wrappers] | `f7ebeca8d1f216e09f1c684a1342e96302f48b4f73daf64eb98863cc258886da` |
| [Journaled upgrade][upgrade] | `83f56c2e808bbdbfa1555408b570d0d89b05cbf8a6a32802c2c47e0e8006e873` |
| [Candidate re-enrollment][reenroll] | `6bb99daf1f9ed5fac5d63794ff0f4b2582fea42f280e6823f794eeb0925e5a74` |
| [Stopped-agent candidate][stopped] | `665042c306439ac0cb36c021d9f1a32dffa26ce13c9949684b3da81b901c9cdc` |

Private SQLite copies are described by the owner evidence; this lane does not
read them or pin inaccessible files. No living AT-root evidence digest is added.

## Updates 10 extension consumed

Updates 10 (#337), run `695f977c-b587-4768-88f1-0514f3a4f1ac`, harness
`0df8599dc84d4b85e48182f20aa4b0fc741bc72a`, product main
`fb5e002549f79e8ef61bebcf42cce1678ab65cb1`: **23/23 mandatory extension checks**.
Sources: [installer-handoff.json](../e2e/at/updates/installer-handoff.json),
[readable handoff](../e2e/at/updates/installer-handoff.md), and
[follow-up 10](../e2e/at/updates/follow-up-10.md). These are living publication
paths on this base: cited by run identity, without file-digest pins or test reads.

| Installer-owned handoff / consumer operation | Measured result |
|---|---|
| Wrapper `install --json` | **0**, installed, agent and pinned NetBird active with the packaged socket. Central starts with null current/last-good. |
| Seed default umask and exact 077 replay | **0/0**, same sequence-10 baseline, tree hash and timestamp. No central promotion from seed. |
| Fresh updater oneshot / `run` | **0/0**, no desired/pending work, no block created. |
| Installer `upgrade --json` to sequence 11 | **0**, staged; verified pending record; current remains 10 until apply. |
| Real updater `apply --pending` with live producer | **0**, correlated FileMeter checkpoint, durable closed epoch, sequence 11 switch, units restart, five checks, sealed `source=installer` result. The quiet unpublished fixture is not a new serving-traffic measurement. |
| Result to production central over mTLS | **202**, current=last-good=11, source installer/batch null, D-110 job recomputed. This run overrides reporting URL/CA/origins: it did not prove the installer's generated reporting config. |
| Exact receipt replay with same/new keys | **202/202**, stored ACK and stable ETag `"8"`. Modified receipt/wrong job ID return **409/409**, with no pointer/version change. |
| Withdraw genuine checkpoint, stage sequence 12 | Stage **0**, apply **1**, `CHECKPOINT_FAILED`, sequence 11 retained, forwarding blocked. Failure evidence accepted **202** without candidate promotion. |
| Restore exact pending bytes / sealed apply / recover | Apply **1**, recover **2**; original receipt/hash and current retained, no duplicate receipt. These are required non-success exits, not a successful wrapper result. |
| Tamper archive for sequence 13 | Apply **2**, `HASH_MISMATCH` before switch, current remains 11. |
| Installer downgrade to sequence 10 | **1**, `DOWNGRADE`, no pending record written. |
| Boot provenance within this running kernel | Agent, seed, job, result and lock holder equal the kernel boot ID; this is not physical cold-boot evidence. |

The D-127/r16 later rollback correction is **TEST ONLY HTTP stimulus**, accepted
202 with ETag advancing from `"9"` to `"10"`; exact first/correction replays are
202 and a third changed outcome is 409. It is not a physically executed rollback.

F10-02 is installer-owned and addressed here. F10-01 is owned by devstack with
agent-core consultation: regular release archives/services use `bin/fvpn-agent`
while the agent verifies the release-root executable; legacy shared FileMeter
paths also need alignment. The ordinary AT23–AT27 gate in updates 10 stays
blocked/failing (live make **2**, Go test **1**), despite the passing extension.
No out-of-scope layout or FileMeter file is changed in this lane.

## Generated-config delivery regression

`TestD089_RealSeedThenJournaledUpgrade` installs real binaries in an isolated
Ubuntu 24.04 amd64 systemd container, verifies seed/upgrade/switch/health, then
uses the installed updater's `flush` command with the original rendered config.
The owned test central extends the unchanged shared enrollment/signed-state
fixture with an mTLS result endpoint. Its HTTPS server CA is generated separately
from the enrolled node issuer. It verifies the client chain, current identity /
revocation fence, node/incarnation, canonical payload hash, D-110 job and
Idempotency-Key before storing the exact receipt and returning **202**.
The test reads back that receipt and ACK, requires byte-equal sealed content,
repeats flush without changing the ACK, and checks that the config hash is
unchanged. No test adapter inserts URL, CA or origins after install.

The test central initially returns a TEST ONLY 503 during apply’s automatic
flush so the durable seal can be inspected before acknowledgment. After the
negative trust checks, the test releases only that server-side hold.
The same test refuses a missing CA at engine construction, refuses the wrong
(node-issuer) CA at delivery with no accepted receipt, and requires **401** for
an unauthenticated result POST. It restores only its test trust-file stimulus
before the successful flush. All test keys are generated at runtime.
The production central's full eligibility, PostgreSQL and observation semantics
are covered by the external updates 10 evidence above; this local receiver does
not claim to reimplement or retest that domain service.

Unit tests cover missing/malformed URL and CA before host commands, exact
configured origins, explicit loopback opt-in, and repair of an old empty
reporting config on `already_current` while preserving identity, unrelated
settings and absence of a pinned boot fallback. The existing no-network consumer
test explicitly disables its sender; it retains sealed receipts locally and is
not counted as reporting evidence.

## D-132 audit

Audited every `packaging/installer/**/*_test.go` and `tests/installer/**`, including
fixture-build scripts, for `docs/e2e`, split path construction and filesystem
reads. There are **zero AT-evidence reads**, living or historical, and no path
needs repointing. Tests construct runtime fixtures. No acceptance test is added
that depends on a living evidence document. Future AT fixtures must be embedded
or read only at `docs/e2e/at/<lane>/history/<follow-up>/`.

## Requirement and AT coverage

- PKG01 / ENR03 / ENR04, §6.1–§6.3; AT05/AT07: authenticated inputs, distinct
  node identity, rerun repair and post-upgrade re-enrollment. No automatic
  publication or blanket AT07 closure.
- UPD02 / UPD03 / UPD05, §13.5–§13.10.3; AT23/AT25/AT26/AT27: staged-versus-applied
  separation, preserved durable accounting and identity, failure exits, actual
  sealed-result delivery and central acceptance scoped as above.
- SEC01 / SEC02, §16.2: mTLS server/client verification, runtime-only test keys,
  no tokens/key bytes in generated config or delivery logs. Public CA paths and
  exact origins are configuration, not new signing-key material.

## Validation and remaining measurements

Full privileged suite: **13/13 PASS**, package **168.107 s**, make exit **0**.
The real generated-config delivery / correlated upgrade / revoke-re-enrollment
scenario passed in **22.25 s**; the stopped-probe negative passed in **73.83 s**.
Installer lint/unit/build, the installer/updater/handover consumer gate, root
`make go-lint` (same checks with its shell cleanup trap disabled), and the
committed-evidence hygiene test pass. Installer builds cover linux/amd64 and
linux/arm64. Disk scratch is under `.agent/tmp`.

Earlier complete attempts are retained in the worktree report: 10/13 at
178.869 s (two old missing-CA fixture prerequisites and a receipt reader after
automatic ACK removed its outbox payload), then 12/13 at 209.078 s (curl rejected
the synthetic TLS certificate's empty issuer name). The fixture corrections
provide a valid CA, hold acceptance at 503 until the seal is read, and name the
TEST ONLY certificate subject. They do not weaken product checks or alter
journals. An isolated curl probe independently verified the corrected TLS cert
and unauthenticated 401 before the final full run.

Three partial attempts were stopped for shared-host admission and supply no
pass claim: two premature restarts during changing external gates, then a new
external gate racing a check that saw only one. Owned containers were removed
between attempts. The final guard recorded one external suite and the complete
run passed. No inotify-limit error occurred; no host settings were changed.
Physical arm64, physical cold boot, power-loss/fsync recovery on target media,
production signing/distribution/rollout and real-device kernel/throughput remain
**待實測**. arm64 builds are not arm64 execution. The **30 s wait / 100 ms poll**
remain configured engineering bounds, not verified latency SLOs under load.
No D-105 retired-stream delivery, full AT07/AT09/AT25/AT26, SECOPS-21 or unrelated
MTU/reconnect investigation is closed by this record.

### Requests to orchestrator

2026-10-31:

1. Consume this F10-02 fix in the next updates extension run and remove its
   reporting URL/CA/origin override in `tests/e2e/at/updates/installer_test.go`,
   retaining the generated-config readback and
   real production central ACK. Installer 6 request 1 is closed by devstack 19;
   this follow-up requests no further serving rerun for that item.
2. Retain F10-01 with devstack / agent-core: align `tests/e2e/serving.go`,
   `deploy/devstack/agent-service.sh`, shared assertions and FileMeter inputs in
   `tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go` / `tests/e2e/scenarios_test.go`, then
   rerun regular AT23–AT27. Preserve `node/agent/release.go` validation. These
   paths are outside installer ownership; no changes were made here.
3. Before adding a regression that consumes updates 10's current extension,
   its AT owner must archive the run under `docs/e2e/at/updates/history/` (D-132).
   The current doc citations carry no living-file digest or executable dependency.
