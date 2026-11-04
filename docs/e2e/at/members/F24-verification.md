# Follow-up 24 verification

[Complete machine-readable receipts](F24-gates.json). All required non-live gates pass. Gates ran in the requested foreground order, with admission captured separately. The live acceptance executed once, after the sole permitted pre-check retry. Both attempts are retained in F24-attempts.json. Its exit 2 preserves the blocked cases. No acceptance retry.

All admission waits and timed-section coverage are recorded separately. [Every check and phase](F24-admission.json).

| Gate | Exit | Elapsed seconds |
|---|---:|---:|
| 01-area-lint | 0 | 1.459 |
| 02-e2e-lint | 0 | 1.884 |
| 03a-members | 0 | 43.951 |
| 03b-members-e2e | 0 | 49.415 |
| 04-assembled | 2 | 3563.221 |
| 05-handover | 0 | 5.207 |
| 06-root-lint | 0 | 1024.991 |
| 07a-vet | 0 | 0.469 |
| 07b-e2e-vet | 0 | 0.222 |
| 08-gofmt | 0 | 0.123 |
| 09-diff | 0 | 0.024 |
| 10-hygiene | 0 | 37.139 |
| 11-handover-final | 0 | 4.681 |

## Archive, links and scope

[Archive/source byte audit](F24-final-audit.json): 145/145 originals match the starting baseline bytes; all earlier archives and dated notes unchanged; owned paths only. Final handover includes the Markdown link check after the new records were added. New large artifacts retain gzip and uncompressed SHA-256/byte pins.

## Real output tails

Worktree/home paths are labeled. Markdown trims trailing whitespace from Docker progress lines; JSON retains the original tails apart from path labels.

### 01-area-lint

Command: `make -f mk/at-members.mk at-members-lint`

```text
(no stdout/stderr)
```

### 02-e2e-lint

Command: `make -f mk/e2e.mk e2e-lint`

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

### 03a-members

Command: `go test -count=1 ./tests/e2e/at/members/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	43.612s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
```

### 03b-members-e2e

Command: `go test -count=1 -tags e2e ./tests/e2e/at/members/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	48.877s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
```

### 04-assembled

Command: `python3 -B .agent/tmp/f24/live.py`

```text
 Container fvpn-w4-members-netbird Stopping
 Container fvpn-w4-members-netbird Stopped
 Container fvpn-w4-members-netbird Removing
 Container fvpn-w4-members-netbird Removed
 Volume fvpn-w4-members-pgbackrest-repo Removing
 Volume fvpn-w4-members-nb Removing
 Network fvpn-w4-members-origin Removing
 Volume fvpn-w4-members-pg Removing
 Network fvpn-w4-members Removing
 Network fvpn-w4-members Resource is still in use
 Network fvpn-w4-members-origin Resource is still in use
 Volume fvpn-w4-members-pgbackrest-repo Removed
 Volume fvpn-w4-members-pg Removed
 Volume fvpn-w4-members-nb Removed
e2e stack is down
Owned teardown exit: 0 inventory: {"containers": {"exit": 0, "owned": []}, "networks": {"exit": 0, "owned": []}, "volumes": {"exit": 0, "owned": []}, "images": {"exit": 0, "owned": []}}
```

### 05-handover

Command: `go test -count=1 ./tests/handover/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	4.315s
```

### 06-root-lint

Command: `make go-lint go-with-disk-tmp=`

```text
(no stdout/stderr)
```

### 07a-vet

Command: `go vet ./...`

```text
(no stdout/stderr)
```

### 07b-e2e-vet

Command: `go vet -tags e2e ./tests/e2e/...`

```text
(no stdout/stderr)
```

### 08-gofmt

Command: `bash -c test -z "$(gofmt -l $(git ls-files '*.go'))"`

```text
(no stdout/stderr)
```

### 09-diff

Command: `git diff --check`

```text
(no stdout/stderr)
```

### 10-hygiene

Command: `go test -count=1 -run ^TestCommittedEvidenceHygiene$ -v ./tests/e2e/at/members/...`

```text
=== RUN   TestCommittedEvidenceHygiene
    evidence_test.go:65: scanned 330 acceptance JSON records plus 820 Markdown documents
--- PASS: TestCommittedEvidenceHygiene (36.81s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	36.814s
testing: warning: no tests to run
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s [no tests to run]
```

### 11-handover-final

Command: `go test -count=1 ./tests/handover/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	4.332s
```
