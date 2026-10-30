# Requests to orchestrator — follow-up 3

Run `339fcbc0-a3e0-4923-afc3-c2d6e12e9d1a`, source `b936aa71d57e03dd9263b9747e2b863839dbe4b7`, completed `2026-10-30T05:17:32Z`.
All five requested scenarios pass; the complete fresh-stack make target exits
zero after writing the records and tearing down the owned stack.

## Open requests

None for the assigned follow-up scenarios. The portal request to project
`health`, `candidate_health` and `previous_release_id` onto `UpdateTarget`
belongs to contracts r9 and fleet-update follow-up 5. This harness uses the
immutable result receipt for those fields and does not depend on that work.

## Resolved: w2-fleet-update direct terminal rollback — #129

Fleet-update follow-up 4 (#129) resolves the follow-up 2 request. A validated
terminal rollback now closes the queued current job without synthetic progress.
`internal/fleetupdate/results.go:358` includes `rolled_back` among the terminal
transitions allowed from a nonterminal state. The regression at
`internal/fleetupdate/direct_terminal_integration_test.go:70` uses our captured
result via `internal/fleetupdate/testdata/at26-direct-rollback.json`.

| Evidence | Follow-up 2 | Follow-up 3 |
|---|---|---|
| Run | `9c862efb-3595-44e1-a6e9-b9a1f3f6cd67` | `339fcbc0-a3e0-4923-afc3-c2d6e12e9d1a` |
| ACK accepted | true | true |
| ACK accepted_as_history | true | false |
| Target state | queued | rolled_back |
| needs_attention | false | true |
| unconfirmed after later observation | false | false |
| Batch state | full batch response not captured | completed_with_exceptions |

The candidate performed a real failed loopback TCP probe and reported
`tunnel_probe=failed`. The updater restored the real last-good process and
obtained all five passing checks from that process. Its single terminal result
contains separate candidate and restored health. Central accepted it as current
under an empty rollback allowlist; the batch target carries
`HEALTH_CHECK_FAILED`, the restored current release and attention. A later
authenticated observation clears `unconfirmed`. No applying/checking result,
journal entry or projector-owned row was injected by the harness.

- [Current AT26 record](AT26.json) and [readable evidence](AT26.md) include the
  exact result, original ACK, full batch API JSON, journal, read-only updater
  status and the subsequent observation timestamps.
- [Exact result](AT26-result.json), [original ACK](AT26-response.json) and
  [batch API response](AT26-batch.json) are also provided separately. These are
  the same JSON values captured in AT26; the result's canonical payload hash
  was independently verified.
- The [follow-up 2 blocked capture](history/follow-up-2/README.md) retains its
  original run ID and four byte-for-byte copies, including its history ACK.

The returned ACK's `applied_to_current_observation` is
`false`. This field reflects
ordering against observations; the current ACK is established by
`accepted_as_history=false`. Target completion and later confirmation are
checked separately.

## Earlier requests still resolved

- AT23 confirms the real good candidate after its D-053 success.
- AT24 shows unconfirmed before apply and needs_attention on the failed target.
- AT25 accepts the honest null-health preflight failure; the byte-tampered
  candidate never installs or executes.
- AT27 recovers the interrupted durable transaction and centrally confirms it.

This live run does not separately inject permanent HTTP rejection or Retry-After
across restart; those remain updater-owned regression coverage.

## Scope and shared pipeline

The driver continues to import `tests/e2e.PrepareServingBaseline` for real
0.1.0, 1.0.1 and 1.0.2 builds, signing, registration and API support activation.
No product source was changed. Both architectures build; live execution is
amd64 on an unpublished node with an accepted baseline. Physical arm64,
published member traffic and the remaining original §19 matrices remain open
as mapped in [follow-up 1](follow-up-1.md).
