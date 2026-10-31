# Running the updates acceptance suite

Run `71f2aa37-b0b9-4568-abe0-2ba7665377e4`, source `bc93ce430ebe8a50e5dfdaf4121352047fe89249` (on `main` `52dc77026cb168df5684ae2ccd5cbcadb4dbaeff`), evidence `2026-10-30T20:59:15Z`.

The suite uses only stack prefix `w4-updates` (`fvpn-w4-updates-*`) and port
base `42000`. It never touches `fvpn-e2e-*`, any other task's prefix, host
network state, or the shared integration test database.

## Offline lint and unit gate

```sh
make -f mk/at-updates.mk at-updates-lint
make go-lint
```

`at-updates-lint` checks `gofmt`, `go vet` (default and `-tags e2e`), pinned
`staticcheck` (default and `-tags e2e`), and `TestUpdatesFollowupEvidence`
(which verifies the follow-up 4, follow-up 5, and follow-up 8 evidence gates,
D-068 policy quote matching against `docs/updates/central-lifecycle.md`, and
`peer_id` redaction).

## Full live run

```sh
make -f mk/at-updates.mk at-updates
```

The target tears down any stale `fvpn-w4-updates-*` objects, brings up a fresh
stack with `E2E_RESET=1`, runs `TestUpdatesFollowup` in the foreground, writes
`docs/e2e/at/updates/{AT23..AT27,at23..at27}.{json,md}` and `summary.md`
before exiting, and always tears down `fvpn-w4-updates-*` in its `EXIT` trap
(including `fvpn-w4-updates-node-probe` and `fvpn-w4-updates-node2`).

All five scenarios (`AT23`–`AT27`) run in one suite:

1. **AT23 + pre-result injection**: Enrolls `node`, sets up the exported
   dedicated non-member probe peer via `stack.SetupUpdaterProbe(ctx, "node", ...)`,
   runs fresh-incarnation pre-result heartbeat injections (`a`, `b`, `c`, and
   `agreeing_null_last_good`) under D-068, then runs `fvpn-updater` to apply
   release `1.0.1` (`good`) while observing D-063 challenge issuance and D-064
   bounded exception traversal via both `UpdaterProbe.Observe(ctx)` and the
   in-node read-only journal stream.
2. **AT25**: Uploads a midpoint-byte-tampered `1.0.2` archive to the lab origin,
   triggers `fvpn-updater` with the exported probe peer configured, verifies
   preflight `HASH_MISMATCH` rejection before any transaction, candidate
   executable, or health challenge (`fresh_challenges=[]`, `grant_samples=0`,
   `elements_after=0`), confirms central `failed` (`ARTIFACT_INVALID`,
   `needs_attention=true`, `unconfirmed=false`), and restores the valid `1.0.2`
   archive.
3. **AT26 + follow-up 4 checks**: Configures candidate `0.1.0` (`boot`) with
   closed loopback `egress_probe_target` (`127.0.0.1:1`), verifies candidate
   `tunnel_probe=failed`, automatic rollback to healthy `1.0.1` (`last_good`),
   current `rolled_back` ACK (`accepted_as_history=false`), revision 7 target
   evidence fields, observation-authority bounds, 64 KiB conflict bounds, and
   `failed_requires_cause` (422 `VALIDATION_FAILED`).
4. **AT24 + AT27**: Stops the primary agent and waits for the configured 180s
   online window to expire naturally; creates the `1.0.2` catch-up batch
   (`online_at_snapshot=0`, target `waiting_online`); replays the genuine
   accepted `1.0.1` result (HTTP 202, identical original ACK, newer job and
   generations untouched); resumes the agent to receive the signed `1.0.2`
   intent; runs recorded-incarnation heartbeat injections (`c1`, `c2`, `c3` × 4)
   verifying D-068 accepted-result authority (`1.0.1` preserved, 0 seeding
   audits, `last_observed_at=null` and `unconfirmed=true` on mismatches);
   completes the `1.0.2` update on `node`; and starts/enrolls `node2` via
   `stack.StartSecondExit` + `stack.EnrollAt`, verifying `node2` completes its
   own `first_sync` batch for `1.0.2` while the original batch snapshot stays
   at `target_count=1`.
