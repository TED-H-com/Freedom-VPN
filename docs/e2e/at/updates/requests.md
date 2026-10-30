# Requests to orchestrator — follow-up 2

Run `9c862efb-3595-44e1-a6e9-b9a1f3f6cd67`, source
`5cb36d03d6e07e6e90a34d4ab2fe65511d63b313`, completed
2026-10-30T04:41:07Z. AT23, AT24, AT25 and AT27 pass. AT26 remains blocked;
the make target exits nonzero after writing all records.

## w2-fleet-update: accepted rollback leaves the current target queued

Coordinate with **w2-updater** on the product progress/terminal-result path.
The candidate's real TCP probe failed. The updater restored the real 1.0.2
last-good process, checked it successfully and sent `state=rolled_back` with
all five restored `health` checks passed and
`candidate_health.tunnel_probe=failed`. The signed candidate's rollback
allowlist was empty. No journal, health or projection values were injected.

The exact [result JSON](AT26-result.json) and original stored
[central ACK](AT26-response.json) were read from the immutable
`update_results` receipt for this batch. The payload's canonical SHA-256,
excluding `payload_hash`, was independently verified as
`1d4511e98fd64bc19375a1832319350d6502093ce19fcbcad61cb7d19b5aa88e`.
The ACK says `accepted=true`, `accepted_as_history=true`,
`applied_to_current_observation=false`, `observed_update_generation="4"`.
The product sender retired the result; its outbox is empty. This is an
accepted history result, not another HTTP rejection or health failure.

After the 90-second target poll, the product batch API still showed:

```json
{
  "state": "queued",
  "unconfirmed": false,
  "needs_attention": false,
  "current_release": "01a18b18-82f2-717d-be44-07141b97ac90",
  "error_code": ""
}
```

[AT26.json](AT26.json) preserves the batch ID, real journal, both health
attestations, empty allowlist, process read-back, exact result/ACK and the
read-only `fvpn-updater status` snapshot. All manifest and artifact digests
are in the same record. `AT26.md` is its readable counterpart.

Code explains the observed outcome:

- `internal/fleetupdate/results.go:346` only permits `rolled_back` from
  `applying` or `checking`. A queued job cannot advance directly to it.
- `internal/fleetupdate/results.go:249` therefore classifies the result as
  history; lines 252–260 skip the terminal ledger transition and update only
  `last_result_at`. Line 286 produces the observed accepted-history ACK.
- `node/agent/observe.go:143` reads the real updater snapshot, including state
  at line 155. `internal/fleetupdate/serving.go:190` adapts the report to
  version/generation observations; line 111 updates the job's observation
  time, without promoting its state to applying/checking. A real checking
  heartbeat therefore does not satisfy the rollback transition prerequisite.
- `node/updater/apply.go:414` builds the terminal result with the restored and
  candidate health profiles, then enqueues and flushes it at lines 433–441.
  The current live path delivered one rollback result and no earlier progress
  result for this batch.
- `internal/fleetupdate/d062_integration_test.go:115` explicitly submits an
  `applying` result before its rollback assertion. That setup misses the live
  queued-to-rollback delivery path.

Please reconcile the producer and central transition contract, add regression
coverage for the unmodified real producer path, and rerun AT26. Preserve the
rules for stale/superseded results and restored-health validation. The harness
will not post synthetic applying/checking results or seed a job state to make
this pass. No product source was changed here.

## Previous requests resolved in this run

- **w2-agent-core / w2-fleet-update:** AT23 and AT27 now receive subsequent
  authenticated update observations and clear `unconfirmed`.
- **w2-updater / w2-fleet-update:** AT25's real `HASH_MISMATCH` remains local;
  the wire `ARTIFACT_INVALID` download failure has `health=null` and is accepted.
  AT24 observes the failed target with `needs_attention=true`.
- **w2-updater / w2-fleet-update:** AT26 delivers separate candidate and
  restored health and passes central validation with an empty rollback
  allowlist. The remaining blocker is the terminal state transition above.

AT27 now delivers and confirms recovery after the preceding preflight failure.
That failure is accepted under D-062, so this live run does not independently
exercise skipping a permanently rejected HTTP result or Retry-After across a
restart; those remain updater-owned regression coverage.

## Shared helper and remaining scope

The overlay and `run.py` are retired. The ordinary tagged Go test imports
`tests/e2e.PrepareServingBaseline`, documented in `docs/e2e/RUN.md`, for all
real builds, signing, registration and API support-policy activation. The
owned adapter retains transport, enrollment, service-hook configuration and
read-only evidence access because those helpers remain private. No second
release pipeline or database writes were added.

The passes cover the requested follow-up scenarios. Physical arm64 runs,
multi-node populations, published dataplane traffic and the remaining original
§19 matrices are still open as mapped in [follow-up 1](follow-up-1.md).
