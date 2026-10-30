# Requests to orchestrator — follow-up 4

Run `9cb542dd-53af-4566-97e8-4365260847c5`, source `136bd15d20359c64f2144cd3359f63ac28825988`, completed `2026-10-30T06:20:24Z`.
The complete fresh-stack make target exits **0** after all five assigned
scenarios pass and its teardown completes. Exact run metadata and manifest /
artifact digests are in [run.json](run.json).

## Open requests

None for the assigned follow-up 4 scenarios. No product source, contract or
migration was changed. The broader acceptance limitations remain in the
[scope mapping](follow-up-1.md).

## Verified: revision 7 target evidence — #136

Before the rollback job has any accepted receipt, all three keys are present
and null on the admin batch GET: [raw response](AT26-before-result-batch.json).
After the one accepted terminal rollback, `previous_release_id` equals the
failed candidate, `health` contains all five restored checks passed, and
`candidate_health.tunnel_probe` is failed. The values exactly equal the
[accepted result](AT26-result.json): [raw batch response](AT26-batch.json),
[original ACK](AT26-response.json), [full AT26 evidence](AT26.md).

## Verified: accepted-result authority and bounded receipts — #140

[AT27](AT27.md) includes the exact controlled observation fields, HTTP responses,
raw batch GETs and read-only projection snapshots. The disagreeing heartbeat
returns 202 but leaves the target rolled_back on the restored release,
`recovery_required=false`, `unconfirmed=true`. The agreeing heartbeat clears
unconfirmed. A report with generation 104 and null job ID returns 202 and
records arrival, leaving high-water 4 and the serving release unchanged.
A final agreeing report restores confirmation before the real agent resumes.

[AT26](AT26.md) records two distinct conflicting payloads under the accepted
result ID and three additional replays of the second. All five return 409
`ACK_CONFLICT`; SELECT-only reads count one conflict row and one
`update_result.conflict` audit for each distinct hash. The original receipt
is unchanged. [AT25](AT25.md) records a failed result at stage check with null
error_code: HTTP 422 `VALIDATION_FAILED`, message "Failed result requires an
error code", with the new job still queued and needs_attention=false.

The [initial run](history/follow-up-4-rate-limit/README.md) had one 429 from
an unpaced replay burst; its make exit was 2. The configured default is 5/s
for agent requests and 1/s for heartbeats (`internal/config/config.go:81`,
classification at `internal/httpapi/middleware.go:472`). The harness now waits
two seconds before each probe. No limiter was disabled, no failed response
was retried into a pass, and no product receipt-bound defect is inferred from
that initial rate-limited request.

## Still resolved: direct terminal rollback — #129

The original [follow-up 2 blocked capture](history/follow-up-2/README.md)
remains unchanged. The [follow-up 3 passing capture](history/follow-up-3/README.md)
is also preserved. Current AT26 again delivers one real terminal rolled_back
result from queued without synthetic progress, obtains accepted_as_history=false,
and closes the batch with attention and HEALTH_CHECK_FAILED. A later
observation confirms. The empty rollback allowlist remains unrestricted.

## Earlier requests still resolved

AT23 confirms real D-053 candidate success. AT24 shows pending unconfirmed and
failed needs_attention. AT25 rejects the tampered archive before execution and
accepts honest null-health preflight evidence. AT27 recovers the same durable
transaction after SIGKILL and confirms it centrally. Real releases still use
`tests/e2e.PrepareServingBaseline`; there is no overlay or second release pipeline.

Live execution remains amd64 on an unpublished accepted baseline; both
architectures build. Physical arm64, published member traffic and the complete
original §19 acceptance matrix are not claimed here.
