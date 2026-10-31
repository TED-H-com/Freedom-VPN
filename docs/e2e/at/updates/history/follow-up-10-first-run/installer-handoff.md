# installer-handoff

D-110/D-111/D-125/D-127 installer hand-off

Status: fail

Spec: UPD02, UPD03, UPD05, §13.5, §13.6, §13.9, D-080, D-098, D-110, D-111, D-125, D-127

At: 2026-10-31T14:59:58Z

Run: 8d9eb051-b47c-47f9-9b6f-cc5c57f4b14a

Scope: Follow-up 10 systemd hand-off and central receipt acceptance; serving traffic proof cited from devstack 19

Lab: w4-updates, port base 42000

Expected: Install, seed, stage and real correlated apply; central baseline/promotion/replay/correction; tamper, checkpoint and downgrade refusals

Actual: create release_support: 422 VALIDATION_FAILED VALIDATION_FAILED Request does not match the API schema.; incomplete checks: install, seed, fresh_no_work, fresh_guard_clear, boot_kernel, legacy_boot_config, checkpoint_failure, failure_replay, failure_baseline, staged, correlated, apply, restarts, central, job_id, replay, conflict, wrong_job, correction, tamper, downgrade

Git: 6a72a634979b2f3aec475b023c16d5221a2edb6f

Commands:

- make -f mk/at-updates.mk at-updates

- consumed_devstack19: "docs/e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-journaled-upgrade.md: serving withdrawal, five-field correlation and nonzero durable watermark; this run uses a quiet unpublished agent and separately tests central acceptance"
- release_10: {"architectures":["amd64","arm64"],"manifest_sha256":"d3051235092e4e72b5def058c493a4cd5429d416cfdd010ae28b4c5958618263","release_id":"01a19279-433d-712e-b0b6-b563aa6eac19"}
- release_11: {"architectures":["amd64","arm64"],"manifest_sha256":"50aefd3e80896e37a21c605919258ecbf37cbc6be499d6d6b58dcdf01137ba44","release_id":"01a19279-4d90-72ba-b52b-06bd3b131ddf"}
- release_12: {"architectures":["amd64","arm64"],"manifest_sha256":"817e579855a1f5141490c8de779d2db624365e4678ffec3032360706bdd9743f","release_id":"01a19279-6762-7b3e-b46b-a8cb59837c4e"}
- release_13: {"architectures":["amd64","arm64"],"manifest_sha256":"45625e4fddfeb54e31a11be44110ea78f39f66762bcef7eeeb001d6b83c94301","release_id":"01a19279-77f8-7814-ab1f-b904fd914830"}
