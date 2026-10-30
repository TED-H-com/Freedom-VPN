# Updates follow-up 6

Run `a83d5d64-505c-488c-b560-5a777566702e`, source `1c8997d91a2c57a445fd25c62a0e4561fe5e6e8f`, evidence `2026-10-30T17:10:45Z`. Prefix `w4-updates`, port base 42000.

Fresh-stack make exit **2** (Go test 1). Every record was written before
failure and teardown. All five original update scenarios and follow-up 4 checks
pass. The extended suite remains blocked on two new observations:

| AT | Extended status | Decisive observation |
|---|---|---|
| [AT23](AT23.md) | blocked | Real candidate succeeds and confirms; shared lab supplies no probe peer (F5-02 reopened) |
| [AT24](AT24.md) | blocked | Fresh case c passes; recorded c2 cannot seed over accepted release evidence (F6-02) |
| [AT25](AT25.md) | blocked | Tampered archive never executes; no configured peer for requested probe inspection (F5-02) |
| [AT26](AT26.md) | pass | Failed candidate probe, healthy rollback, current ACK, confirmed target and revision 7 fields |
| [AT27](AT27.md) | blocked | Real interrupted job recovers; recorded c3 emits no unexpected-release audit after prior acceptance (F6-02) |

F5-01 is **resolved** by the fresh-null expectation: boot is unrecorded and
receives one bounded audit, not zero. Fleet follow-up 8 (#167), already on this
main, additionally requires agreeing last-good before seeding current. All
three fresh injections therefore preserve null columns. Twelve recorded-case
reports preserve the real accepted 1.0.1 values; the later genuine 1.0.2 success
establishes last-good. No accepted success result or projection was fabricated.

Devstack #170 fixes agent_config to the existing `/etc/vpn-node/agent-run.json`.
Its peer setup lives in an unexported acceptance-test helper, not the shared
baseline path. No peer or alias config was created. No probe refusal class was
emitted; `agent_config_unavailable` is **not** an observed finding.

See [raw heartbeat/batch/audit evidence](pre-result.json), [probe inspection](probe-issuance.json),
[requests](requests.md), [run/digests](run.json), and [tested main](source-main.json).
[Follow-up 5](history/follow-up-5/README.md) remains unchanged as history.
The [first follow-up 6 run](history/follow-up-6-first-run/README.md) is also
preserved. Its host reader could not open the root-owned journal; the final
run captures the challenge inside the node without altering permissions.
The product source commit is recorded above; `harness_source.files_sha256`
records the exact final capture harness, which differed from that commit and
is committed with these records.
These labels cover the requested follow-up scenarios, not the complete original
§19 matrix or physical arm64 execution.
