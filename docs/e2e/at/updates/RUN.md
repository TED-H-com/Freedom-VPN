# Follow-up 10 run procedure

Use branch `w4-at-updates`, prefix `w4-updates` and port base `42000`.
Before starting the stack, run `pgrep -af 'gates.sh|mk/at-'` and wait while two
other integration suites are active. Run foreground commands only. The Python
runner owns disk scratch under `.agent/tmp` and calls the shared `e2e-down`
harness in `finally`; it uses no shell cleanup trap.

```sh
make -f mk/at-updates.mk at-updates-lint
make -f mk/at-updates.mk at-updates
```

The live target builds the current checkout, runs all five regular AT scenarios
and the installer extension in one Go test, writes every record, and fails if
any assigned case or extension check is incomplete. It tears down only the
`fvpn-w4-updates-*` stack. A zero command exit alone never supplies a pass.

The systemd extension uses devstack's `PrepareInstallerNode`,
`RegisterInstallerReleases`, `BuildInstallerRelease`, `RerunInstaller`,
`ProbeInstallerExit`, `HoldInstallerFirstPoll`, `HoldInstallerApply`,
`ConfigureInstallerServing` and `RestoreInstallerServices`. It uses the shared
`upgrade-observe`, `journal-copy` and `ExportContainerFile` evidence tools.
Updater invocations, read-only status and negative stimuli run from the test;
there is no interactive container repair, product edit or journal write.

The initial no-work observation holds only the authenticated desired-state
poll. Bootstrap is allowed to finish so the real NetBird daemon can join through
`/run/fvpn-netbird/netbird.sock`. Seed verifies the same material under the
process default umask and 077. The first stopped-agent attempt establishes a
real final-sample failure and exercises D-111's null-central-baseline receipt.
Its exact pending bytes are restored to the original path for sealed replay;
changing the pending path or request UUID is not a valid replay test.

The next higher release runs with the real agent active. The shared observer
records the updater's persisted wait, boot and latch identity and the agent's
five-field checkpoint. A pass requires byte-equal UUIDs within the original
window, matching receipt accounting, apply exit 0, a real pointer/executable
switch, unit restarts and all five health checks. The run uses a quiet,
unpublished fixture. The serving withdrawal and nonzero durable watermark
measurement is consumed from [devstack 19](../../runs/2026-10-31/followup19/README.md)
and its [D-102 response](../../RESPONSES.md#updater--installer--2026-10-31-follow-up-19-measured-result).

Central checks use real mTLS HTTP: recomputed D-110 UUIDv5 identity, accepted
failure baseline and success promotion, exact keyed and new-key replay,
modified replay conflict, wrong-job refusal, receipt cardinality and node
resource versions. The D-127/r16 M-2 rollback correction is an explicitly
labeled **TEST ONLY HTTP contract stimulus**, derived from the genuine success
receipt. It does not claim a local rollback; regular AT26 measures that path.
The receipt history must retain both payloads and both stored ACKs, and reject
a third outcome. A separate higher staged artifact is byte-tampered; a lower
sequence is refused before writing pending.

The installer's generated result configuration is recorded before supplying
the owned lab result URL, CA and exact allowed origins. This adapter does not
claim that the installer configures reporting itself.

```sh
mkdir -p .agent/tmp
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
make go-lint go-with-disk-tmp=
go vet ./...
go test ./...
go test ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$' -count=1 -v
git diff --check
```

The root lint override removes only its shell cleanup trap; its gofmt, vet and
pinned staticcheck commands remain unchanged. These are engineering acceptance
fixtures with generated TEST ONLY identities and signing material.

Prior root records and the unchanged first run are pinned in the
[follow-up 9 archive](history/follow-up-9/ARCHIVE.md) (D-098). Current claims
must name the new run ID; historical failures stay historical failures.
