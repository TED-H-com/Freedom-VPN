# Installer follow-up 5 — 2026-10-31

The complete local journaled upgrade passes with the real agent meter checkpoint
and candidate health stage. The final privileged suite passed **13/13 tests** in
134.103 s; make exited 0. `TestD089_RealSeedThenJournaledUpgrade` passed in
7.65 s, including the successful apply, read-only installer check, downgrade
refusal, exact replay, changed-request refusal and independent tamper case.

Base: `ce81d1da` (includes agent-core 24 / #305 and updater 15 / #307).
The [follow-up 4 note](install-followup4-2026-10-31.md) remains unchanged history.

## Real health boundary

The installer-owned control command now has `--upgrade-health`. It calls the
unchanged `node/internal/fakecontrol.UpgradeHealthFixture` and serves signed
unpublished desired-state and lease responses over the fixture's runtime CA and
mTLS identity. The harness explicitly installs its desired-state trust bundle.
All signing keys are generated at runtime. Enrollment-only mode remains in the
socket, marker and ordinary lifecycle scenarios.

The signed egress target is a real TCP listener on the isolated container's
loopback interface. The candidate performs the dial itself. As in agent-core
24's fixture regression, NetBird management/relay status is explicitly TEST ONLY;
the fixture status version is the pinned version, not a value taken from the
health challenge. The separate F9-01 scenario still exercises the real daemon,
socket and packaged strict units.

No checkpoint or health attestation is fabricated. The product agent closes its
real meter epoch, writes the durable checkpoint, restarts from the candidate
tree, validates signed control documents, checks live isolation and writes the
challenge-bound health attestation. The observed successful status was:

```json
{
  "state": "succeeded",
  "stage": "report",
  "journal_stage": "locally_succeeded",
  "current_release_id": "019a0000-0000-7000-8000-0000000000c2",
  "last_good_release_id": "019a0000-0000-7000-8000-0000000000c2",
  "recovery_required": false,
  "health": {
    "dns_policy": "passed",
    "identity_and_state": "passed",
    "internet_only": "passed",
    "processes": "passed",
    "tunnel_probe": "passed"
  }
}
```

This is the local upgrade of a quiet, unpublished node. It does not prove native
NetBird management/member traffic, central result acceptance, held nonzero usage,
or a live serving-node meter-barrier handshake. Arm64 cross-compiles; physical
arm64 execution remains unmeasured.

`TestUPD05_StoppedProbeFailsRealHealth` independently closes only the TCP listener
while signed control remains available. Candidate process, isolation and DNS
checks pass; `tunnel_probe` fails. Restored-release health also fails, so the
updater seals `failed/rollback`, `ROLLBACK_FAILED`, exit 1, with the old release
current and `recovery_required=true`. The test passed in 64.91 s. The original
success assertion still requires apply exit 0 and the requested release current.

Go's internal import boundary is handled entirely in owned tooling:
`tests/installer/control-build.py` creates a disposable harness module under
`.agent/tmp`, with a `node/` import path and a local replacement for the unchanged
root module. It copies only the installer command and the root's pinned module
metadata. Build, vet and the same pinned staticcheck run against that source;
the generated module is deleted afterward. No node source, module dependency,
public facade or other owner's file changes.

## F9-02, F9-03, F9-05 and D-121

- **F9-02:** actual standalone seed under both umasks 022 and 077 passes with
  unchanged baseline identity, tree hash and timestamp. Extraction explicitly
  sets directories to 0755 and files to archive permission bits (zero defaults
  to 0644), matching the merged extractor. The matching-umask workaround is gone.
- **F9-03:** sealed failure returns exit 1 with outcome/code. Exit 2 means only
  the forwarding-blocked sentinel. Both packaged updater oneshots still accept
  0 and 2; this is service acceptance, not proof of update success. `check` can
  report a consistent installation after a failed attempt, but a failed updater
  unit or recovery-required state is drift. Current operator wording is in
  [install.md](install.md#updater-extraction-and-terminal-results).
- **F9-05:** `upgrade` compares decimal-string sequences under the shared lock
  before marker reconciliation, extraction or staging. Lower sequences return
  `DOWNGRADE` / exit 1, including attempts reusing the installed ID. A different
  release at an equal sequence is also refused, matching `install`. The active
  journal's digest-checked envelope supplies the sequence after an updater
  switch; the install-time record stays unchanged. Missing/malformed evidence,
  journal/current disagreement and recovery-required state fail closed.
  CLI tests prove release/staging bytes, modes, mtimes, pending record, current
  and marker unchanged, both after initial install and a journaled upgrade.
- **D-121:** `packaging/installer/handoff_test.go` is unchanged from #307. Its
  real-consumer unit test preserves failed replay's job ID, result ID, payload,
  hash and receipt cardinality. The container test now agrees with D-110:
  changing only request ID is refused for an existing release/manifest identity.
  The tamper case has a separate third signed release. Exact successful replay
  keeps the same succeeded job, with exit 2 because this unpublished fixture
  remains forwarding-blocked. Tampering also reports `HASH_MISMATCH` together
  with that sentinel after recovery. Both assertions require the specific
  sentinel; they do not accept arbitrary nonzero exits.

## Final privileged run output

The full output is retained in the worktree's ignored
`.agent/tmp/followup5-container.log`. These are its actual result lines:

```text
--- PASS: TestMissingTool (0.77s)
--- PASS: TestTokenArgvRefused (0.30s)
--- PASS: TestFaultRollback (2.27s)
--- PASS: TestInstallLifecycle (14.11s)
--- PASS: TestRemovePurge (6.33s)
--- PASS: TestImagePrepareTwoBoots (16.67s)
--- PASS: TestBootstrap (0.15s)
--- PASS: TestD089_MergedUpdaterSeedProbe (0.78s)
--- PASS: TestD089_RealSeedThenJournaledUpgrade (7.65s)
--- PASS: TestUPD05_StoppedProbeFailsRealHealth (64.91s)
--- PASS: TestD089_SeedRefusalKeepsInstalledTreeAndJournal (7.19s)
--- PASS: TestAT07_D116InstallerMarkerAndAgentRules (5.45s)
--- PASS: TestF905_CLIDowngradeHasNoStagingEffects (4.60s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/installer	134.103s
```

F9-01's real pinned socket lifecycle and D-116's exact four-chain verifier plus
legacy-marker migration still pass. Before each container invocation,
`pgrep -af 'gates.sh|mk/at-'` found no other integration suites. No inotify failure,
host network change or sysctl change occurred. Installer-prefixed Docker objects
are removed by the harness.

To reproduce from the repository root, use disk scratch:

```sh
mkdir -p .agent/tmp
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
make -f mk/installer.mk installer-lint installer-unit installer-build
pgrep -af 'gates.sh|mk/at-'
# Wait while two or more other integration suites are active.
make -f mk/installer.mk installer-test
go test -count=1 ./packaging/installer/... ./tests/handover/...
make go-lint go-with-disk-tmp=
test -z "$(gofmt -l .)"
git diff --check
go test -count=1 ./tests/e2e -run TestCommittedFixtureEvidence_ValidAndHygieneClean -v
```

The `go-with-disk-tmp=` override avoids the make helper's shell cleanup trap;
TMPDIR/GOTMPDIR still point at disk. The recorded run used the existing private
Python gate wrapper, with unique disk scratch and cleanup on exit.

## Follow-up 6 boundary

At the starting base, `docs/e2e/RESPONSES.md` still contained devstack follow-up
17's ENR-05 marker blocker; devstack 18's result was absent. Follow-up 6 must
consume that result, including an actual installer migration of the follow-up 3
marker, native traffic, held usage and service restoration. The successful
unpublished upgrade here does not replace that acceptance evidence.
