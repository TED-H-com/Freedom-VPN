# Reproduce the Follow-up 27 fixtures

2026-11-01. TEST ONLY, owned `fvpn-e2e-*` containers. The predeclared
[plan](plan.md), [stream amendment](stream-duration-amendment.md), and two
[readback](hold-readback-amendment.md) / [build](hold-build-amendment.md)
amendments retain every attempt. A rerun creates a new UUID evidence directory;
never replace a committed run or its analysis.

Use the repository toolchain and its configured Go cache. Keep temporary data
on disk inside the worktree:

```sh
mkdir -p .agent/tmp/followup27
export TMPDIR="$PWD/.agent/tmp/followup27"
export GOTMPDIR="$TMPDIR"
export GOCACHE="$(go env GOCACHE)"
export GOTOOLCHAIN=local CGO_ENABLED=0 GOMAXPROCS=4 GOFLAGS=-p=4
export PYTHONDONTWRITEBYTECODE=1
pgrep -af 'gates.sh|mk/at-'
```

Wait while two other suites are running; a gate wrapper and its own child make
are one suite. Recheck after startup. Run each fresh stack serially in the
foreground, retaining failed attempts and always running teardown:

```sh
make -f mk/e2e.mk e2e-up E2E_RESET=1
pgrep -af 'gates.sh|mk/at-'
FVPN_E2E_FOLLOWUP27=hold FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'
make -f mk/e2e.mk e2e-down
```

For a native first-selection plus sustained-stream attempt, replace the test
line with:

```sh
FVPN_E2E_FOLLOWUP27=native FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'
```

Use `ipv6` for the other family. The committed campaign declares four fresh
stacks per family, with both shapes in each stack. Capture starts before member
join. After fresh populated serving proof the original exit selection and
two-second members wait precede the first request; there is no preliminary
successful traffic gate. Stream readiness is independent of that request.
The stream runs 150 seconds, followed by a passive tail. A successful sender
row does **not** mean zero loss: `measurement.json` and every CSV interval
retain actual node-ingress loss and delay. No node nft tracer is installed.

Use the printed UUID in these post-processing commands, before pinning a new
run. The private directory is produced by the runner and is never committed:

```sh
python3 tests/e2e/analyze_followup27.py docs/e2e/runs/2026-11-01/followup27/RUN_ID .agent/tmp/followup27/RUN_ID
python3 tests/e2e/summarize_followup27.py docs/e2e/runs/2026-11-01/followup27/RUN_ID
python3 tests/e2e/archive_followup27.py docs/e2e/runs/2026-11-01/followup27/RUN_ID
```

The analyzer verifies all private receipts and capture completeness before
attributing packet absence. A prior measurement, CSV or timeline is immutable.
Large capture JSON, header JSONL and original Markdown are lossless gzip.
`archive.json` records both compressed and original byte counts and SHA-256.
Plain Markdown paths remain explicit navigation documents. They are not
replacement measurements, and the evidence reader rejects them as captures.
`followup27_io.read` reads original bytes with receipt verification; `gzip -dc`
also permits ordinary inspection. Final capture pins point only into this dated
archive, including the original `.gz` files and separate navigation documents.

The hold's lease-path rationale, exact arming sequence and consumer joins are
in [the harness export contract](../../../scenarios/harness-exports.md) and
[the populated proof](hold-fixture.md). The proof's two earlier harness failures
remain failed; separate supplements prove their local lease bytes.

After teardown, run the requested gates in order: `e2e-lint`, untagged e2e
tests, tagged e2e tests, `TestCommittedFixtureEvidence_ValidAndHygieneClean`,
handover tests with `-count=1`, root `make go-lint`, tagged linux/amd64 and
linux/arm64 builds, and `git diff --check`. The campaign gate artifact records
the exact commands, results and real output tails. No product code or NetBird
pin is changed by this fixture.
