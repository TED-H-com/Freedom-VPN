# Follow-up 13 run procedure

Use branch `w4-at-updates`, prefix `w4-updates`. Port base defaults to `42000`; this run uses
`AT_UPDATES_PORT_BASE=54000` because an unrelated host listener occupies 42443.
Only the owned harness slot changes; no host listener or other suite is modified.
Before starting the stack, run `pgrep -af 'gates.sh|mk/at-'` and wait while two
other integration suites are active. Run foreground commands only. The Python
runner owns disk scratch under `.agent/tmp` and calls the shared `e2e-down`
harness in `finally`; it uses no shell cleanup trap.

```sh
make -f mk/at-updates.mk at-updates-lint
AT_UPDATES_PORT_BASE=54000 make -f mk/at-updates.mk at-updates
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

All 44 prior root records are pinned in the
[follow-up 12 archive](history/follow-up-12/ARCHIVE.md) (D-098/D-132).
Every earlier archive, including the failed first attempt, remains unchanged.
Use the actual `go env GOCACHE`; scratch stays under `.agent/tmp`.

Follow-up 13 consumes agent-core 31 (#369) and netlab 22 (#365) on product main
`5bd751026073444087b32ac2a7115f8566104f8b`. The enforcement lifecycle hook remains
opt-in; this run does not enable its timing gate. The agent's safe-boundary
shutdown may finish bounded local work after SIGTERM while control I/O cancels.
This is a regression observation, not a worst-case shutdown-bound experiment.

The lane compiles its TEST ONLY stop observer and installs it only inside its
owned containers. The regular service script delegates public commands through
the observer to a byte-identical original; supervisor calls still exec that
original directly. The installer fixture uses a PATH shim forwarding to the
original `/usr/bin/systemctl`. Generated updater JSON, argv, service timeouts,
and product binaries are unchanged. Monotonic duration surrounds the delegated
command, excluding SQLite readback. Only updater-parent stop/start commands are
recorded. A 1 ms read-only `/proc` sampler captures the prior agent's wait status;
systemd also exposes its retained exit status and stop timeout. Unobserved exits
remain explicit gaps. Lease row presence/identity/expiry is read before/after
commands without publishing envelopes or keys.

`stops.py` collects every command and transaction stage before teardown. It
includes the repeated `stepMigrate` stop, checks inventory, flags command errors,
killed agents and lost stored leases, and pairs all stops with the archived run.
The old journal's stage brackets include other work and are not direct timings.
See [agent-stops.json](agent-stops.json) and [the comparison](agent-stops.md).

The [startup failure](history/follow-up-13-startup/startup.json) happened before
any Go AT or installer test: port 42443 was occupied. Its original note proposed
waiting for that port; the final admission decision instead selected the free,
shared-harness-supported 54000 slot. The failed startup remains exit 2. There is
only one measured AT invocation; no failed predicate is rerun or weakened.
