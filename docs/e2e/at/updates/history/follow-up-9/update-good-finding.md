# update-good-finding

First install ends recovery_required / ROLLBACK_FAILED

Status: blocked

Spec: §13.9, §13.11, §19 AT23

At: 2026-10-30T00:28:30Z

Expected: A first install with no last-good release records ROLLBACK_FAILED after health checks fail, without a symlink restore.

Actual: error_code=ROLLBACK_FAILED classification=health_failed_without_last_good

Git: 80c2971e3ad964f4cdb097da61aec80cbb08dab6

Images: fvpn-w4-updates-node:local a59961894d3a; fvpn-w4-updates-worker:local 37db66b36e2c; fvpn-w4-updates-control:local 831cbd9da1b1; fvpn-w4-updates-origin:local a284069da6c8; fvpn-w4-updates-idp:local 51e3274256ee; fvpn-w4-updates-payment:local 740e16c55514; fvpn-w4-updates-netbird:0.82.0 7b5b53b2dc8b; fvpn-w4-updates-postgres:17 31ba9c94a03a; fvpn-w4-updates-nginx:1.30.0 55d1fb0944aa; fvpn-w4-updates-adguard:v0.107.59 5d5e3aef39a8; fvpn-w4-updates-coredns:1.12.0 40384aa1f5ea

Policy revision: unset

Node incarnation: 01a18a33-cc39-7585-8deb-a50b5f93b1fc

Commands:

- fvpn-updater run on the primary node, first install

- exit_code: 2
- finished: true
- classification: health_failed_without_last_good
- journal: stage=recovery_required error_code=ROLLBACK_FAILED error_stage=rollback error_detail=ROLLBACK_FAILED job_phase=done health={"dns_policy":"failed","identity_and_state":"passed","internet_only":"failed","processes":"passed","tunnel_probe":"failed"} current=01a18a33-dc7e-75e7-aa9a-f0d506a207a2 last_good= rollback_release= artifact=f47af399eabe3fe03c8b2715e42de60fe00056d508728619d68dfda202d107ca
- events: seq=1 stage=prepared detail=
seq=2 stage=blocked detail=
seq=3 stage=final_sampled detail=
seq=4 stage=stopped detail=
seq=5 stage=migrated detail=
seq=6 stage=switched detail=
seq=7 stage=checking detail=
seq=8 stage=rolling_back detail=
seq=9 stage=recovery_required detail=
- updater_out: (empty stdout/stderr; see exit_code and journal)
- artifact_sha: f47af399eabe3fe03c8b2715e42de60fe00056d508728619d68dfda202d107ca
- amd64_sha: f47af399eabe3fe03c8b2715e42de60fe00056d508728619d68dfda202d107ca
- arm64_sha: eb5d03bcea96fc50155a402270646ba8dba4cc443182e40669319f3235317777
- downloaded_amd64: true
- downloaded_arm64: false
- identity_unchanged: true
- forward_chain: present accepts=0
- cited_run: main:docs/e2e/runs/2026-10-30/update-good.md (comparison only; this record uses its own journal)
- decision: Missing updater health adapter: host.go:151 selects NodeChecks; health.go:59-68 unconditionally fails isolation, DNS and tunnel checks regardless of publication. apply.go:212-215 starts rollback; decide.go:33-34 and apply.go:238-243 produce recovery_required when no safe last-good exists, before SwitchSymlink at apply.go:266. This is correct fail-closed recovery, but publication alone is not a valid health prerequisite. Spec §13.9 requires real baseline and restricted authenticated candidate-path probes while unpublished/disabled. See health-analysis.md, Requests to orchestrator.
