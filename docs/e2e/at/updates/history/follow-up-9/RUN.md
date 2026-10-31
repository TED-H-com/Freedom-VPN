# Running updates follow-up 9

Use the dedicated `w4-updates` stack and port base `42000`:

```sh
make -f mk/at-updates.mk at-updates-lint
make -f mk/at-updates.mk at-updates
make go-lint
go vet ./...
git diff --check
```

The live target resets the owned stack, runs in the foreground, writes evidence before its acceptance gate, and tears down every `fvpn-w4-updates-*` container, image, network and volume even on failure. `AT_UPDATES_REUSE=1` is only for a newly prepared stack before any enrollment. Scratch is on disk under `.agent/tmp`, never tmpfs.

The current run and product SHA are in [summary.md](summary.md) and [source-main.json](source-main.json). [Follow-up 8](history/follow-up-8/README.md) is historical. The [first follow-up 9 diagnostic run](history/follow-up-9-first-run/README.md) exposed the devstack boot-ID mismatch; its labels do not supply current pass claims.

`TestUpdatesFollowup` first builds the shared real signed baseline. Its independent installer record runs before a fleet default exists:

1. `PrepareInstallerNode` creates the dedicated Ubuntu 24.04 systemd container. The harness repacks shared `BuildRelease` binaries at the installer layout, adds the unchanged packaged units and real installer binary for both architectures, signs with the generated lab release key, registers through `RegisterRelease`, and activates support through the API. Sequences are 10 (installed), 11 (upgrade) and 9 (downgrade).
2. A runtime systemd condition holds the agent's first poll, so neither a desired-state file nor an update job exists during the fresh-node check. `RerunInstaller` calls the real installer; packaged NetBird failures remain failures. The signed initial archive is retained for offline seed. Both standalone seed and seed with the packaged `umask 077` are recorded; no tree or journal repair is performed.
3. The updater oneshot and a second foreground `run` exercise missing desired state. `check --json`, latch absence, seeded status, the kernel boot ID and lock metadata are captured. A separate config with an intentionally different fallback boot ID tests compatibility and kernel precedence. The first agent poll is then released to obtain real forwarding reasons.
4. A runtime `ConditionPathExists` delays automatic updater execution while `upgrade --json` stages sequence 11. Pending bytes, status, exit and `current` are recorded. Foreground `apply --pending` uses the product stop/start commands and health logic. No meter checkpoint or health result is supplied by this harness. The generated config is recorded before adding the lab result endpoint, server CA and exact allowed origins.
5. Product `status`, quiescent read-only journal copies, `flush`, central receipts and `GET /api/v1/admin/nodes/{id}` distinguish local outcomes from acceptance. An outbox result still present after flush is submitted and replayed with the same HTTP idempotency key, retaining its original bytes, to capture any rejection details.
6. A freshly staged sequence-11 archive is byte-tampered and must fail before switching. Sequence 9 must be refused by `upgrade` with `DOWNGRADE`; an unexpectedly staged downgrade is preserved and withheld from apply. The installer node is retired through the API with an ETag and read-back before regular all-node batch snapshots.

The regular lab then runs all five requested scenarios unchanged in substance:

- **AT23:** good candidate, challenge-bound health, exported D-063/D-064 probe, central success and confirmation.
- **AT25:** byte-tampered archive, preflight `HASH_MISMATCH`, no candidate execution, accepted failed result.
- **AT26:** genuine closed-port candidate probe failure, healthy last-good rollback, current terminal ACK, target evidence and bounded conflicts.
- **AT24:** naturally offline target, catch-up success, and a later node's own first-sync completion without changing the original snapshot.
- **AT27:** pre-result and accepted-result authority probes, stale ACK replay and newer-job preservation.

Before enrollment, the owned adapter copies each regular container's kernel boot ID into the devstack agent's boot-ID input. The updater config omits `boot_id`. The devstack entrypoint's random UUID otherwise disagrees with D-080's kernel identity. This adapter, installer release-layout support and capture barriers are requested as shared devstack follow-up 16 exports.

`AT23.json`–`AT27.json` and `installer-handoff.json` share one run ID. A regular AT24/AT26 pass does not make the installer extension pass. The make target remains nonzero if that extension is incomplete. Full original §19 populations and physical arm64 execution remain outside this focused run; both architectures are cross-built into the signed artifacts.
