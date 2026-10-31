# Requests to orchestrator — follow-up 8

Run `71f2aa37-b0b9-4568-abe0-2ba7665377e4`, source `bc93ce430ebe8a50e5dfdaf4121352047fe89249` (on `main` `52dc77026cb168df5684ae2ccd5cbcadb4dbaeff`), evidence `2026-10-30T20:59:15Z`. Prefix `w4-updates`, port base 42000.

All five scenarios (`AT23`–`AT27`) **pass** in a single fresh-stack run. Make exit **0**.

## Resolved: F5-02 — probe-peer export (#204)

**Closed by `w3-devstack-e2e` follow-up 12 (PR #204).** `Stack.SetupUpdaterProbe` and `UpdaterProbe.Observe` are now exported by `tests/e2e` and wired into `tests/e2e/at/updates/{lab_test.go,probe_test.go,live_test.go}`:

- **AT23** observes D-063 issuance both through the in-node read-only journal stream (`configured_probe_issued=true`, 0 omission diagnostics, no host read of the `0700` journal, raw `peer_id` redacted in published evidence) and through `UpdaterProbe.Observe(ctx)` (`verified=true`, `authenticated_udp_replies=14`, origin receipt `requests=14` from node SNAT `192.0.7.20`, `grant_samples=4`, `max_timeout_seconds=44`, `probe_up_bytes=1078`, `probe_down_bytes=1078`, `member_bytes_before=0`, `member_bytes_after=0`, `elements_after=0`).
- **AT25** runs with `probe_peer` and `agent_config` configured in `updater.json` and both observers active. The byte-tampered archive fails preflight with `HASH_MISMATCH` (`stage=""`) before `stageCheck`, so `fresh_challenges=[]` and `probe_observer` records `grant_samples=0`, `authenticated_udp_replies=0`, `elements_after=0`. As recorded in [probe-issuance.md](probe-issuance.md) and [AT25.md](AT25.md), the D-063/D-064 probe scope does not apply to an archive that never passes preflight.

## Resolved: F6-02 — corrected recorded c2/c3 expectations (#186, D-068 #189)

Closed in follow-up 7 (**#186**) and re-verified on current `main` with fleet-update follow-up 9 (**#189**, **D-068**). Under D-068, pre-result heartbeat seeding fills only a null `nodes.current_release_id` on the node's current non-superseded job when `last_good_release_id` agrees, a non-null `current_release_id` advances only through an accepted result, and once any result is accepted, seeding closes for later jobs in that incarnation. All fresh (`a`, `b`, `c`, `agreeing_null_last_good`) and recorded (`c1`, `c2`, `c3`) assertions in `preresult_test.go` and `restated_test.go` hold unchanged; the verbatim policy quotes and line citations in `restated_test.go` and `preresult_test.go` were updated to match `docs/updates/central-lifecycle.md:425-506`. See [pre-result.md](pre-result.md) and [pre-result.json](pre-result.json).

## Remaining full-§19 scope items outside this focused follow-up

No blocker remains for the follow-up 8 scenario set (`AT23`–`AT27` all pass, exit 0). The broader §19 items noted in earlier follow-ups remain outside this harness scope:

- **Owner: `w3-devstack-e2e` / `w2-fleet-update` (broader §19 AT24 snapshot population):** `TestUpdatesFollowup` proves offline target catch-up (`online_at_snapshot=0 -> waiting_online -> succeeded`) and post-batch `node2` inheritance through its own `first_sync` job while `original_target_count=1` is unchanged. A third pre-existing `enrolling` node at snapshot time (to exercise `skipped_enrolling` in the same batch snapshot alongside an online and an offline ready node) is not part of this focused follow-up.
- **Owner: operator hardware evidence (`docs/e2e/at/updates/operator-run.md`):** Both `linux/amd64` and `linux/arm64` release artifacts are compiled, signed, and registered by `PrepareServingBaseline` in every run; executing `fvpn-updater` and `fvpn-agent` on a physical or VM `arm64` Ubuntu 24.04 host remains an operator run item (`at23-arm64.md`).

## Devstack follow-up 15 response (2026-10-31)

F5-02 remains closed through `SetupUpdaterProbe` / `Observe`; the new member UDP helper uses another port and never counts updater traffic. F6-02 remains the product D-068 contract. The follow-up 8 request on origin/main for a third pre-existing `enrolling` snapshot target remains open with fleet-update/devstack: `StartSecondExit` is a fixed second slot and no third enrolling-target fixture is claimed here. A dedicated `installer-node` is now available for ENR-06, but it is not automatically a ready fleet target or a replacement for that snapshot population proof. Physical arm64 execution remains operator evidence; final cross-build results are in the hand-off report.
