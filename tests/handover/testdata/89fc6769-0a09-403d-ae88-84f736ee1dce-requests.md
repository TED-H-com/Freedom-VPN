# Requests to orchestrator — follow-up 9

Product main: `5575470587a4e3014bd515e624ba2ad0aacbd140`. Harness: `5d98ea8f613fc415897b0a42940f0af6834ee2ae`. Current live run: `89fc6769-0a09-403d-ae88-84f736ee1dce`, prefix `w4-updates`, port base `42000`. See [installer evidence](installer-handoff.json), [run procedure](RUN.md) and [current AT results](summary.md). Requests below come from this run, except the explicitly identified diagnostic boot mismatch.

The roadmap rows used are installer initial/follow-up 2 (#239/#258), devstack follow-up 15 (#266), updater follow-ups 13/14 (#253/#269), and fleet-update follow-up 10 (#273). Installer follow-up 3 is requested work, not claimed merged. No product paths were edited here.

| Request | Owner | Deciding observation | Requested follow-up |
|---|---|---|---|
| F9-01 | Installer follow-up 3: packaged systemd units | Initial install exits 1. `fvpn-netbird.service` reports `listen unix /var/run/netbird.sock: bind: read-only file system`; `fvpn-node-netbird-up.service` fails dependency/start. | Fix the existing ENR-06 startup blocker and re-run the unmodified units. The harness retained the real failure and continued seed/stage/apply diagnostics. |
| F9-02 | Updater extraction/seed; installer integration | Standalone `seed` exits 1 `HASH_MISMATCH: seed tree differs from the signed artifact`. The same command, envelope and live tree with the packaged `umask 077` exits 0. Installed binaries are mode 0700, units 0600 and nested directories 0700. | Make verified extraction/tree comparison deterministic across installer, oneshot and interactive process umasks. `packaging/installer/cmd/fvpn-node-install/main.go` sets 0077; `node/updater/extract.go` creates modes through umask without restoring them. Preserve fail-closed verification; do not waive the tree check. |
| F9-03 | Installer/updater apply wiring | Config points at `/var/lib/vpn-node/state/meter.checkpoint`; the file is absent. Authentic apply exits 0 but seals `source=installer`, `outcome=failed`, `error_code=CHECKPOINT_FAILED`, `error_stage=final_sample`, transaction `rolled_back`. `current` stays at sequence 10; service start timestamps are unchanged; pending is consumed. | Connect the genuine durable meter barrier and verify switch/restarts/health afterward. Clarify/fix exit 0 for this blocked failed outcome: the next `run` returns 2. The harness did not synthesize a checkpoint or passing health. |
| F9-04 | Installer configuration; fleet-update result acceptance | Generated `updater.json` has empty `result_url`. After explicitly adding the owned lab endpoint/CA/origins, the product sender gets 422 `VALIDATION_FAILED`. The original payload has null health, `CHECKPOINT_FAILED`, unchanged seeded sequence-10 current, and `source=installer`; central has no recorded current release. | Configure reporting on the installed node. Fleet-update should define/handle legitimate pre-health installer failure evidence when the central baseline is null. `acceptInstallerResult` passes `n.current` to the central-job baseline check, which refuses a null baseline. Current run stores zero installer receipts and keeps the node summary `update=null`; no central promotion is claimed. |
| F9-05 | Installer `upgrade` | Sequence 9 against installed sequence 10 returns `status=staged`, exit 0, and writes pending. | Apply the downgrade refusal already present in `install` to `upgrade` before staging. The runtime stage barrier withheld this unexpected downgrade from the updater; it is preserved as `pending-release.downgrade-not-applied.json` during the run. |
| F9-06 | Devstack follow-up 16 | First diagnostic AT23 failed because the entrypoint's random agent boot UUID differed from the updater's kernel boot ID. Aligning the agent input before enrollment restores challenge identity. Shared installer fixtures also use separate keys, an unregistered release and a placeholder other-architecture archive. | Export kernel-aligned boot setup, centrally registered installer-layout real cross-builds, and first-poll/stage capture barriers. The minimum adapters live only in this lane's harness. |

The failure response for F9-04 is:

```text
422 VALIDATION_FAILED
Pre-health failure requires null health, an error and the unchanged attempt release
```

The immutable result is submitted and replayed with one HTTP idempotency key. Both return 422 with that same validation message; request IDs differ. This is not evidence of a broken accepted-result replay. **Wire-nodes follow-up 29** should include the eventual accepted installer-result keyed replay once F9-03/F9-04 are resolved. No accepted installer ACK exists in this run to replay.

Fresh-node D-080 behavior is measured before the first agent poll: missing desired and pending files, oneshot `ExecMainStatus=0`, `Result=success`, second foreground run exit 0, and no forwarding latch. `check` still exits 2 for the stopped/failed units and missing runtime directory, but does not report `fvpn-updater.service` failure. After releasing the first poll, actual serving reasons are `not_published` and `safety_check_failed`; forwarding remains false. This is updater no-work evidence on the installer-created layout, not a healthy installation claim.

Seed with the packaged umask records the verified baseline and kernel boot ID while leaving `last_good` empty and health null. Lock holder and installer job claim use that same kernel ID without `boot_id` in the generated config. A config with a different legacy fallback is accepted, still using the kernel ID.

Physical arm64 execution, the broader original §19 mixed population, and a third pre-existing enrolling target remain later work. Current pass claims, if any, are only those in the current run's AT records.
