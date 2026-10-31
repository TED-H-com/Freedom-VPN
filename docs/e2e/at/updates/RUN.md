# Follow-up 12 run procedure

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
process default umask and 077. The initial successful apply exercises D-111 with a null central baseline.
A later negative stops the producer and withdraws its genuine checkpoint, then
requires a sealed final-sample failure without promoting the failed candidate.
Its exact pending bytes are restored to the original path for sealed replay;
changing the pending path or request UUID is not a valid replay test.

The higher release runs with the real agent active. The shared observer
records the updater's persisted wait, boot and latch identity and the agent's
five-field checkpoint. A pass requires byte-equal UUIDs within the original
window, matching receipt accounting, apply exit 0, a real pointer/executable
switch, unit restarts and all five health checks. The run uses a quiet,
unpublished fixture. The serving withdrawal and nonzero durable watermark
measurement is consumed from [devstack 19](../../runs/2026-10-31/followup19/README.md)
and its [D-102 response](../../RESPONSES.md#updater--installer--2026-10-31-follow-up-19-measured-result).

Acknowledged receipts are recovered from central only when their result/job
IDs and canonical hash match the local journal ACK history. This preserves
evidence when the product sender has already drained the outbox.

Central checks use real mTLS HTTP: recomputed D-110 UUIDv5 identity, accepted
failure baseline and success promotion, exact keyed and new-key replay,
modified replay conflict, wrong-job refusal, receipt cardinality and node
resource versions. The D-127/r16 M-2 rollback correction is an explicitly
labeled **TEST ONLY HTTP contract stimulus**, derived from the genuine success
receipt. It does not claim a local rollback; regular AT26 measures that path and AT27 requires its same-run physical rollback witness.
The receipt history must retain both payloads and both stored ACKs, and reject
a third outcome. A separate higher staged artifact is byte-tampered; a lower
sequence is refused before writing pending.

The generated result URL, server CA, client cert/key paths and exact origin
policy are read back and never rewritten. Compare the whole generated config
hash before and after the extension. Devstack 21's shared `installerSpec`
supplies `allow_loopback=true` and exact `control_url=https://127.0.0.1:9444`;
its `installArgs` adds the explicit **TEST ONLY** `--allow-loopback` only for
that pair. `RerunInstaller` is the sole install path. Record the shared spec,
the diagnostic argument vector derived from it, and the actual wrapper outcome.
A nonzero wrapper exit stops the installer item, records a devstack request,
and never triggers a lane-local install fallback. `ProbeInstallerExit` remains
only the existing upgrade-command exit diagnostic, never an install path.

Before any curl contract stimulus, require product ACK history with a matching
sealed hash, central immutable payload/ACK, exactly one local history row and
an empty outbox; repeat product flush and require the admin view to remain
unchanged. The supplementary [transport capture](installer-delivery-transport.json)
counts central route logs through the extension's precise completion timestamp. Only the
installer can produce results then: the regular fixture has not enrolled.
The 11 explicitly recorded curl requests are separated from the two automatic
posts (one success and one stopped-producer failure). No repeated-flush POST
may be inferred from database deduplication alone. Separate route counts directly
before and after the repeat flush must both equal one.

For the separate [uncached origin probe](artifact-origin.json), after the
extension retires the installer fixture but before whole-stack teardown,
resolve current with `readlink -f` (the shared installer may use a relative
symlink) and read its retained signed manifest. Select the amd64
artifact's SHA-256, record updater status and config/cache hashes, and move
only `update-staging/<sha256>.tar.zst` to a held sibling. Invoke the exact `seed`
command recorded in that JSON with the **generated** config and current tree.
The runner invokes `supplemental.py` after the Go sequence, while the owned
stack remains up. Restore the archive in Python `finally` and compare its bytes/hash, generated
config and current. The required result is exit 0 with only a verified seeded
baseline added; it is neither a new apply nor central acceptance. Artifact hosting
and generated origin policy stay unchanged. The shared release hostname maps
to an owned non-private Docker origin; `allowed_artifact_origins` grants
private/CGNAT exceptions rather than restricting all hosts. No private-mirror
acceptance follows from this observation.

```sh
mkdir -p .agent/tmp
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
make go-lint go-with-disk-tmp=
go vet ./...
go test -count=1 ./tests/e2e/at/updates/...
GOOS=linux GOARCH=arm64 go build ./...
go test ./...
go test ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$' -count=1 -v
git diff --check
```

The root lint override removes only its shell cleanup trap; its gofmt, vet and
pinned staticcheck commands remain unchanged. These are engineering acceptance
fixtures with generated TEST ONLY identities and signing material.

All 43 prior root records are pinned in the
[follow-up 11 archive](history/follow-up-11/ARCHIVE.md) (D-098/D-132). The older diagnostic archives remain unchanged. Current claims
must name the new run ID; historical failures stay historical failures.

Use the current `go env GOCACHE`; do not hard-code the cache location. All
scratch stays under `.agent/tmp`. Recheck suite admission after startup. This
run consumes updater 19 (#351), devstack 21 (#352), installer 8 (#353) and
agent-core 29 (#356) on product main `45209609`. It is the first regular
AT23–AT27 measurement on the tree containing agent-core 29's additive TEST ONLY
Apply host samples. This is not a contention-cause or worst-case bound test.

The [first follow-up 12 attempt](history/follow-up-12-first-attempt/ARCHIVE.md)
is retained separately: all regular and installer checks passed, but the new
supplemental probe incorrectly passed a relative symlink as `seed --tree`.
Its make exit 2 is preserved. The corrected probe uses the resolved directory;
final claims must come from a complete fresh run, with no rows carried over.

The corrected measured invocation used two-stage admission: wait for fewer than
two other suites, start the shared stack with `make -f mk/e2e.mk e2e-up E2E_RESET=1`
using the same prefix/port/state bindings, recheck admission, then invoke
`AT_UPDATES_REUSE=1 make -f mk/at-updates.mk at-updates`. This reuses only the
fresh reset stack before its first enrollment, never the failed attempt's state.
The live runner still performs all ATs, supplemental probes and final teardown.
