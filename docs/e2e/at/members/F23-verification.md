# Follow-up 23 verification

[Complete machine-readable receipts](F23-gates.json). All required non-live gates pass. Gates ran in the requested foreground order, with admission captured separately. The assembled run executed once. Its exit 2 preserves the blocked cases. No acceptance retry.

Post-run hygiene admission waited one 60-second interval (60.079 s including checks), outside every live/timed section. Assembled-run admission wait remains 0 s. [Every check and phase](F23-admission.json).

| Gate | Exit | Elapsed seconds |
|---|---:|---:|
| 01-area-lint | 0 | 1.766 |
| 02-e2e-lint | 0 | 1.904 |
| 03a-members | 0 | 36.222 |
| 01b-area-lint | 0 | 2.209 |
| 02b-e2e-lint | 0 | 3.484 |
| 03c-members | 0 | 36.217 |
| 03d-members-e2e | 0 | 39.67 |
| 04-assembled | 2 | 2977.673 |
| 05-handover | 0 | 4.185 |
| 06-root-lint | 0 | 1.926 |
| 07a-vet | 0 | 0.465 |
| 07b-tagged-vet | 0 | 0.223 |
| 08-gofmt | 0 | 0.132 |
| 09-diff | 0 | 0.485 |
| 10-hygiene | 0 | 92.028 |
| 11-final-handover | 0 | 4.268 |
| 12-final-diff | 0 | 0.754 |
| 13-final-diff | 0 | 0.744 |

## Archive, links and scope

[Archive/source byte audit](F23-final-audit.json): 122/122 originals match the starting baseline bytes; all earlier archives and dated notes unchanged; owned paths only. Final handover includes the Markdown link check after the new records were added. New large artifacts retain gzip and uncompressed SHA-256/byte pins.

## Real output tails

Worktree/home paths are labeled. Markdown trims trailing whitespace from Docker progress lines; JSON retains the original tails apart from path labels. A final staged whitespace check caught those Docker line endings after the earlier diff gate; they were removed without changing evidence bytes or dated notes.

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
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	35.880s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
```

### 01b-area-lint

Command: `make -f mk/at-members.mk at-members-lint`

```text
(no stdout/stderr)
```

### 02b-e2e-lint

Command: `make -f mk/e2e.mk e2e-lint`

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

### 03c-members

Command: `go test -count=1 ./tests/e2e/at/members/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	35.879s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
```

### 03d-members-e2e

Command: `go test -count=1 -tags e2e ./tests/e2e/at/members/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	39.168s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
```

### 04-assembled

Command: `python3 -B .agent/tmp/f23/live.py`

```text
 Container fvpn-w4-members-netbird Stopping
 Container fvpn-w4-members-netbird Stopped
 Container fvpn-w4-members-netbird Removing
 Container fvpn-w4-members-netbird Removed
 Volume fvpn-w4-members-nb Removing
 Volume fvpn-w4-members-pg Removing
 Network fvpn-w4-members Removing
 Network fvpn-w4-members-origin Removing
 Volume fvpn-w4-members-pgbackrest-repo Removing
 Network fvpn-w4-members Resource is still in use
 Network fvpn-w4-members-origin Resource is still in use
 Volume fvpn-w4-members-nb Removed
 Volume fvpn-w4-members-pg Removed
 Volume fvpn-w4-members-pgbackrest-repo Removed
e2e stack is down
Owned teardown exit: 0 inventory: {"containers": {"exit": 0, "owned": []}, "networks": {"exit": 0, "owned": []}, "volumes": {"exit": 0, "owned": []}, "images": {"exit": 0, "owned": []}}
```

### 05-handover

Command: `go test -count=1 ./tests/handover/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.820s
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

### 07b-tagged-vet

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
    evidence_test.go:65: scanned 300 acceptance JSON records plus 750 Markdown documents
--- PASS: TestCommittedEvidenceHygiene (31.65s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	31.657s
testing: warning: no tests to run
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s [no tests to run]
```

### 11-final-handover

Command: `go test -count=1 ./tests/handover/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.906s
```

### 12-final-diff

Command: `git diff --check main`

```text
(no stdout/stderr)
```

### 13-final-diff

Command: `git diff --check main`

```text
(no stdout/stderr)
```
