# Installer follow-up 6 — 2026-10-31

This follow-up consumes devstack 18's scoped ENR-05 serving acceptance and the
combined updater/agent source through #323 (base `6c129f44`). Local evidence uses
Ubuntu 24.04 amd64 systemd containers with no external network. It does not
replace the serving-node upgrade rerun assigned to devstack 19.

## ENR-05 serving evidence consumed

[Devstack 18](../e2e/runs/2026-10-31/followup18/README.md), PR #313, owns the
serving evidence. Run `09cb1198-435f-4609-9d0b-d53e6c1ed441`, source `84b6a342`,
finished **13 pass / 0 blocked / 1 fail**, exit **1**. Its ENR-05 rows pass:

- [Marker migration](../e2e/runs/2026-10-31/followup18/09cb1198-435f-4609-9d0b-d53e6c1ed441/installer-marker-migration.md):
  the fixture supplies the exact old follow-up 3 marker on a stopped node;
  the installer migrates it. The four rendered chains remain in `inet fvpn`,
  and the empty hookless ownership chain is in `inet fvpn_install`.
- [Serving read-back](../e2e/runs/2026-10-31/followup18/09cb1198-435f-4609-9d0b-d53e6c1ed441/installer-serving-readback.md):
  strict serving verification, healthy installer check, fresh heartbeat,
  active services and native member traffic on the real D-116 installer node.
- [Held usage and restoration](../e2e/runs/2026-10-31/followup18/09cb1198-435f-4609-9d0b-d53e6c1ed441/installer-usage.md):
  sequence 2 / ACK 1 with one held event (1,376 upload / 1,001 download bytes),
  identical before/after ledger hashes and CLI bytes, restored services and
  fresh serving heartbeat, resumed persistent TCP/UDP traffic. Releasing the
  503 hold reaches sequence/ACK 3 with no unacknowledged events.

The remaining [serving upgrade failure](../e2e/runs/2026-10-31/followup18/09cb1198-435f-4609-9d0b-d53e6c1ed441/installer-journaled-upgrade.md)
is retained: absent checkpoint, `failed/final_sample/rolled_back`, null health,
unchanged current, erroneous apply exit 0. That source predates #307's exit
handling and #310's bounded handshake. Neither this note nor ENR-05 promotes
that failed attempt to success. [Follow-up 5](install-followup5-2026-10-31.md)
remains isolated installer-side journal/switch/health evidence only.

## Seed diagnosis and consuming sequence

The candidate run `8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179` is also **13/0/1**,
exit **1**: its candidate passed all five health checks and sealed
`succeeded/locally_succeeded`, but revoke/re-enroll exited 1 `baseline_failed`.
Its [read-only diagnostic](../e2e/runs/2026-10-31/followup18/8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179/post-upgrade-reenroll-diagnostic.md)
records `HASH_MISMATCH: seed tree differs from the signed artifact` with ledger
and retired identity preserved. No live-tree repair was made.

[Updater 17's PR diagnosis](https://github.com/TED-H-com/Freedom-VPN/pull/323)
and [dated implementation note](../updates/node-updater.md#post-upgrade-seed-diagnosis--2026-10-31)
reproduce that diagnostic with the historical pre-#307 extractor only. The
explicit-mode extraction already merged in #307 resolves the reproduced
022/077 divergence. The old live run lacks per-entry inventories, so exact
attribution to its mismatching entry remains an inference.

Inspection of `packaging/installer/baseline.go` confirms that it resolves
`current` to `releases/<id>` and supplies the matching retained envelope to
the installed updater. It does not implement a second hash. Updater seed/apply
use the same extractor and sorted relative-path tree hash over file bytes,
file permission bits and subdirectory modes. The current link, previous release,
release-root entry and external state sidecars are outside that inventory.
No installer or updater product change is needed in this follow-up.

`TestD089_RealSeedThenJournaledUpgrade` now consumes #307/#310/#320/#323:

1. Install and seed the real release; standalone seed under 022 and 077 keeps
   the baseline hash, identity and timestamp unchanged.
2. Stage the signed candidate with the updater service temporarily held by a
   TEST ONLY ExecCondition. Assert staging did not move `current`.
3. Direct real updater apply under explicit umask 022; assert a real candidate
   switch, terminal local success and all five real candidate health results.
4. Revoke the old mTLS identity at fake control and require 403
   `IDENTITY_REVOKED`. Re-enroll through the real installer (umask 077), including
   seed of the resolved candidate and its matching signed envelope.
5. Require seed exit 0, healthy installer check, unchanged release file hashes,
   modes and mtimes (including the retained previous release), unchanged ledger
   database hash, a new identity/incarnation, and old key/cert/node hashes
   preserved in a mode-0700 retired directory. The old certificate still fails.

The shared fake-control fixture's single consumed bootstrap slot is reset for
the new TEST ONLY identity when revoked. This supplies its next setup credential;
it does not alter a release, checkpoint, ledger or journal. It enrolls a new
fixture node identity; same-node central authorization remains devstack evidence.
On failure, the test retains live and privately extracted signed-reference
file/mode inventories in its log before container cleanup. It never repairs
the live tree or weakens seed verification.

## Real correlated FileMeter path

The fixture uses the real agent and updater binaries, their systemd units,
SQLite stores, kernel isolation verifier, signed `fakecontrol.UpgradeHealthFixture`
desired state/lease, candidate executable identity and a real TCP listener.
Only NetBird management/relay status is TEST ONLY; native peer traffic is not
claimed by this local test.

There is no checkpoint or updater latch before apply. The real agent then
publishes `barrier_id` byte-equal to the updater's UUID after its barrier. The
test reads the sealed receipt through read-only SQLite with the live WAL,
checks its canonical payload hash, and compares its closed epoch, durable
sequence and ACK against the checkpoint and real usage snapshot. It also
requires a distinct journaled new-epoch ID. The quiet fixture's watermarks are
0/0; this is real post-block accounting without fabricated nonzero traffic.
The unpublished fixture keeps member forwarding blocked after local success.

The stopped-listener negative still requires candidate tunnel health failure,
old current restored, recovery required and exit 1. F9-05 still requires
`DOWNGRADE` / exit 1 with no staging effects, including after the switch.
Exact successful replay keeps its job and returns the blocked sentinel exit 2;
a changed pending record is refused. Updater 17's CLI regression in the required
consumer gate separately proves an already sealed nondestructive final-sample
failure: initial/replayed apply **1**, recover **2**, read-only status **0**, with
unchanged sealed receipt and current. Destructive recovery-required failures
retain exit 2. Systemd acceptance and a layout check never substitute for the
durable outcome.

## Devstack 19 serving-node rerun hand-off

Run after this lane merges, with #307, #310, #320 and #323 in the actual node
image/binaries. Preserve the source identity and failed attempts. Required proof:

1. Start from a genuinely serving D-116 installer node with native member traffic
   and real held nonzero usage. Keep the marker migration/strict verifier,
   same-incarnation ledger preservation and service-restoration evidence.
2. Observe the updater's latch and verified member withdrawal while the control
   channel remains available. Retain the real agent checkpoint whose `barrier_id`
   exactly matches the persisted wait identity within its original boot/deadline.
   Do not inject a checkpoint or reuse a quiet shutdown as serving evidence.
3. Prove final-sample accounting in the sealed receipt, the actual candidate
   symlink/executable switch, all candidate health checks, `locally_succeeded`,
   `state=succeeded` and initial apply exit 0. Retain publication/forwarding state
   separately; record central result acceptance separately if tested.
4. Preserve exit behavior on any failure: pre-stop final-sample failure and
   sealed replay apply exit 1, recover exit 2, unchanged current and receipt,
   blocked gate; destructive recovery-required failures retain exit 2.
5. The installer must show a verified staged manifest/request, no staging-time
   pointer switch, post-switch `check` using `updater_journal` while retaining
   install history, and successful post-upgrade revoke/re-enroll seed of the
   candidate. Retain ledger/WAL and retired identity evidence, baseline exits,
   service restoration and old-certificate refusal. If seed mismatch recurs,
   preserve signed-reference versus resolved-current bytes/modes before any
   remediation and route the evidence to its owner.

This is an acceptance hand-off, not a claim that devstack 19 has run.

## Remaining unclaimed

- Successful serving-node correlated upgrade on the combined source; central
  acceptance of this local upgrade receipt; D-105 retired-stream delivery.
- Physical arm64 execution (cross-compilation only), production signing and
  distribution, fleet rollout and real production service restoration.
- Boot-source provenance, physical cold boot/power-loss recovery, real-device
  kernel/throughput bounds and a measured checkpoint-wait SLO (30 s / 100 ms
  remain engineering bounds, 待實測).
- Full AT07/AT09/AT25/AT26 or SECOPS-21 closure; devstack 18's native MTU failure
  and original high-generation namespace-restart question.

## Validation

Full privileged suite: **13/13 PASS**, package time **186.289 s**, make exit 0.
The correlated upgrade plus revoke/re-enroll test passed in **29.07 s**; the
stopped-probe negative passed in **77.94 s**; F9-05 passed in **3.75 s**.
Installer lint/unit/build passed, including linux/amd64 and linux/arm64 builds.
No inotify error occurred. Admission checks found one other integration gate
before each focused run and none before the full suite; no third suite was run.

The initial focused attempt passed the correlated upgrade but failed before
seed at `starting fvpn-node-netbird-up.service failed`: the fake control's
single bootstrap slot was already consumed. After provisioning the next
TEST ONLY credential through its existing API, the focused retry passed
(29.51 s, package 34.795 s), followed by the full suite above. That initial
failure is retained in the private gate log with file/mode inventories; no
seed HASH_MISMATCH occurred in either attempt.

Required gates use disk scratch under `.agent/tmp`, CGO disabled and the local
Go toolchain. Run foreground commands; check shared-host admission before the
privileged suite and wait while two other integration suites are active.

```sh
mkdir -p .agent/tmp
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
export CGO_ENABLED=0 GOTOOLCHAIN=local
make -f mk/installer.mk installer-lint installer-unit installer-build
pgrep -af 'gates.sh|mk/at-'
make -f mk/installer.mk installer-test
go test -count=1 ./packaging/installer/... ./node/updater/... ./tests/handover/...
make go-lint go-with-disk-tmp=
test -z "$(gofmt -l .)"
git diff --check
go test -count=1 ./tests/e2e -run TestCommittedFixtureEvidence_ValidAndHygieneClean -v
```

The root lint override removes its shell cleanup trap only; the outer runner
owns disk scratch cleanup and retains the same gofmt/vet/pinned-staticcheck
checks. Final measured results are recorded in the lane's hand-off report.
