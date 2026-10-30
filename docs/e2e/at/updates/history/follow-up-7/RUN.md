# Updates acceptance harness — follow-up 7

Run `71078ab8-edf8-4d23-af72-ea247f704011`, source `c8d4a00cf1bf594b20016263c2cfc3c164d92bc9`, evidence `2026-10-30T18:09:36Z`. Prefix `w4-updates`, port base 42000.

```sh
make -f mk/at-updates.mk at-updates-lint
make -f mk/at-updates.mk at-updates
make go-lint
```

The live target fixes prefix `w4-updates` and port base `42000`, recreates the
owned stack, uses disk scratch under `.agent/tmp`, and tears down on every exit.
Never reuse another prefix or the shared PostgreSQL. Expected current exit:
**2**, because AT23/AT25 are explicitly retained blocked on F5-02. The record
writer saves all AT JSON/Markdown before rejecting incomplete acceptance.

The run builds real signed 0.1.0/1.0.1/1.0.2 candidates through
`tests/e2e.PrepareServingBaseline`; a registered 1.0.3 supplies the unrelated
release stimulus. A real accepted 1.0.1 result provides the AT24/AT27 baseline.
AT26 exercises failed candidate health and healthy rollback. AT24 stops the
agent for the product's configured online window, verifies waiting_online,
resumes and completes the authorized update, then starts/enrolls a new node
through shared Stack hooks and completes its inherited first_sync job.
AT27 exercises both observation-authority branches, withheld confirmation and
an exact stale ACK replay. All database inspection is SELECT-only.

AT23/AT25 load their original follow-up 6 records from
`history/follow-up-6/`; their run IDs, times, digests and observations remain
unchanged. Only the retained-status explanation names the current owner:
**w3-devstack-e2e follow-up 12 (probe-peer export)**. They are not run and no
probe setup is improvised. Follow-up 8 will re-enable those scenarios after the
shared export lands.

Outputs: [summary](summary.md), AT23–AT27 JSON/Markdown, [authority evidence](pre-result.json),
[inheritance evidence](inheritance.json), exact AT26 result/ACK/batch JSON,
[run provenance](run.json), and [requests](requests.md). Prior evidence is kept
under [history/follow-up-6](history/follow-up-6/README.md).

Additional checks used for this handoff:

```sh
mkdir -p .agent/tmp
TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp" CGO_ENABLED=0 GOTOOLCHAIN=local go vet -tags e2e ./tests/e2e/...
make go-test
make go-build
```

No physical arm64 execution or full original §19 population claim is made.
