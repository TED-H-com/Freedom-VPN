# Follow-up 46 validation — 2026-11-03

No production or node/enforce change. All commands used admission; no timing
reservation was requested. Exit 125 from an admission wait was never counted
as a test result. Node tests waited for the orchestrator reservation to release.
The gate wrapper checked exact free again before every make, Docker and Go
invocation. Disk scratch, local toolchain and the cache identity below apply.
The first local scope helper returned 1 because it used a two-point main
comparison after another lane merged. The corrected helper uses the required
main...HEAD plus working changes, and the merge-base for production equality;
it passes without editing another area or merging newer main into this lane.
Root go-lint used its pinned staticcheck with the disk environment already set;
its go-with-disk-tmp recipe prefix was overridden to `set -eu;` to avoid a shell
cleanup trap. No other lint command or rule changed.

| Gate | Exit | Seconds |
| --- | --- | --- |
| node-lint-build | 0 | 3.604 |
| node-test | 0 | 261.855 |
| node-integration | 0 | 138.598 |
| reviews-r16-r19 | 0 | 20.039 |
| reviews-r20-r26 | 0 | 44.282 |
| handover | 0 | 3.879 |
| go-lint | 0 | 1.720 |
| vet | 0 | 0.462 |
| evidence | 0 | 3.010 |
| fixture-hygiene | 0 | 8.927 |
| final-all-tests | 0 | 181.392 |
| ordinary main lab (13/13) | 0 | 232.009 |
| synthetic tests (65 root; 24 other directory) | 0 | — |

Real output tails (private locations redacted):

```text
node-lint-build:
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./node/agent/... ./node/watchdog/... ./node/internal/agentstore/... ./node/internal/controlclient/... ./node/internal/timeanchor/... ./node/internal/fakecontrol/... ./node/internal/nodelock/... ./node/internal/usagestore/... ./node/internal/meter/...
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local go vet -tags netlab ./node/agent/... ./tests/netlab/agent/...
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./node/agent/... ./node/watchdog/... ./node/internal/agentstore/... ./node/internal/controlclient/... ./node/internal/timeanchor/... ./node/internal/fakecontrol/... ./node/internal/nodelock/... ./node/internal/usagestore/... ./node/internal/meter/...
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags netlab ./node/agent/... ./tests/netlab/agent/...
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 go build -trimpath -o bin/linux_amd64/fvpn-agent ./node/agent/cmd/fvpn-agent
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=arm64 go build -trimpath -o bin/linux_arm64/fvpn-agent ./node/agent/cmd/fvpn-agent
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 go build -trimpath -o bin/linux_amd64/fvpn-watchdog ./node/watchdog/cmd/fvpn-watchdog
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=arm64 go build -trimpath -o bin/linux_arm64/fvpn-watchdog ./node/watchdog/cmd/fvpn-watchdog

node-test:
Waiting for host admission: reserved for orchestrator until 19:10:34 (orchestrator verify devstack f32)
Waiting for host admission: reserved for orchestrator until 19:10:34 (orchestrator verify devstack f32)
Waiting for host admission: reserved for orchestrator until 19:10:34 (orchestrator verify devstack f32)
Waiting for host admission: reserved for orchestrator until 19:10:34 (orchestrator verify devstack f32)
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local go test ./node/agent/... ./node/watchdog/... ./node/internal/agentstore/... ./node/internal/controlclient/... ./node/internal/timeanchor/... ./node/internal/fakecontrol/... ./node/internal/nodelock/... ./node/internal/usagestore/... ./node/internal/meter/...
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	140.230s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.291s
ok  	github.com/TED-H-com/Freedom-VPN/node/watchdog	0.105s
?   	github.com/TED-H-com/Freedom-VPN/node/watchdog/cmd/fvpn-watchdog	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/agentstore	0.164s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/controlclient	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol	0.030s
?   	github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol/cmd/fvpn-labcontrol	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/node/internal/nodelock	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/usagestore	0.980s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/meter	0.620s

node-integration:
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local go test -count=1 -tags integration ./node/agent/...
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	137.949s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.291s

reviews-r16-r19:
=== RUN   TestReviewR18_DisabledLifecycleDoesNotAllocate
--- PASS: TestReviewR18_DisabledLifecycleDoesNotAllocate (0.00s)
=== RUN   TestReviewR18_DisabledRunCommandMatchesRun
--- PASS: TestReviewR18_DisabledRunCommandMatchesRun (0.01s)
=== RUN   TestReviewR18_RenderCaptureStaysInMemory
--- PASS: TestReviewR18_RenderCaptureStaysInMemory (0.00s)
=== RUN   TestReviewR18_EnabledSampleIgnoresInstallDeadline
    review_r18_lifecycle_test.go:107: finished install context skipped resource sampling (elapsed 622.098µs, total syscr +11)
--- PASS: TestReviewR18_EnabledSampleIgnoresInstallDeadline (0.12s)
=== RUN   TestReviewR18_EnabledCancelReapsChild
--- PASS: TestReviewR18_EnabledCancelReapsChild (0.05s)
=== RUN   TestReviewR19_Q4_CurrentRenderDoesNotRestrictEstablishedDownload
    review_r19_intersection_test.go:68: current destination denial precedes the established download accept
--- PASS: TestReviewR19_Q4_CurrentRenderDoesNotRestrictEstablishedDownload (0.00s)
=== RUN   TestReviewR19_Q4_ExistingCounterIsNotRecreated
--- PASS: TestReviewR19_Q4_ExistingCounterIsNotRecreated (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/enforce	0.222s

reviews-r20-r26:
    --- PASS: TestR20_H2_VerifierIgnoresEmptiedDenyAndHostSets/serving_emptied_and_timed_deny_and_host_sets (0.00s)
    --- PASS: TestR20_H2_VerifierIgnoresEmptiedDenyAndHostSets/probe_baseline_emptied_deny_host_and_nonempty_allowed_v4 (0.00s)
=== RUN   TestR20_Held_VerifyServingCatchesResolverBindingQoSAndSNATDrift
--- PASS: TestR20_Held_VerifyServingCatchesResolverBindingQoSAndSNATDrift (0.01s)
=== RUN   TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen
=== RUN   TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen/cold_start_rollback_with_failed_loadState_omits_delete_chain_and_flush_set
=== RUN   TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen/expired_ctx_during_verify_aborts_rollback_after_guard_release
--- PASS: TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen (0.00s)
    --- PASS: TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen/cold_start_rollback_with_failed_loadState_omits_delete_chain_and_flush_set (0.00s)
    --- PASS: TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen/expired_ctx_during_verify_aborts_rollback_after_guard_release (0.00s)
=== RUN   TestR24_H2_OrphanMemberTCClassDeletedBeforeNftBatch
--- PASS: TestR24_H2_OrphanMemberTCClassDeletedBeforeNftBatch (0.00s)
=== RUN   TestR24_M2_PostShapeClockCheckSkipsBaselineInApply
--- PASS: TestR24_M2_PostShapeClockCheckSkipsBaselineInApply (0.00s)
=== RUN   TestR24_Held_Q1_GuardInstallBoundLifetimeCeilingAndCounterContinuity
--- PASS: TestR24_Held_Q1_GuardInstallBoundLifetimeCeilingAndCounterContinuity (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/enforce	0.022s

handover:
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.484s

go-lint:


vet:


evidence:
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab	2.754s

fixture-hygiene:
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.665s

final-all-tests:
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-stream	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.037s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.094s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.103s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.776s
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab	2.830s
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/at	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab/cmd/fvpn-netlab	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/dns/cmd/cache-authority	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/dns/internal/cachefixture	[no test files]

ordinary-main-lab:
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab/agent	218.202s
make[1]: Entering directory '<worktree>/.agent/tmp/f46-production'
docker rm -f fvpn-w2-agent-core-f46 >/dev/null 2>&1 || true
docker network rm fvpn-w2-agent-core-f46-dns >/dev/null 2>&1 || true
<worktree>/.agent/f46/bin/make -f mk/netlab.mk netlab-wipe-scratch FVPN_LAB_PREFIX='fvpn-w2-agent-core-f46'
make[2]: Entering directory '<worktree>/.agent/tmp/f46-production'
mkdir -p <worktree>/.agent/tmp/f46-production/.agent/tmp/fvpn-w2-agent-core-f46-scratch
docker rm -f fvpn-w2-agent-core-f46-wipe >/dev/null 2>&1 || true
docker run --rm --name fvpn-w2-agent-core-f46-wipe --network none \
	-v <worktree>/.agent/tmp/f46-production/.agent/tmp/fvpn-w2-agent-core-f46-scratch:/scratch fvpn-w2-agent-core-f46:lab \
	find /scratch -mindepth 1 -delete
make[2]: Leaving directory '<worktree>/.agent/tmp/f46-production'
rmdir <worktree>/.agent/tmp/f46-production/.agent/tmp/fvpn-w2-agent-core-f46-scratch 2>/dev/null || true
make[1]: Leaving directory '<worktree>/.agent/tmp/f46-production'
```

Environment: CGO_ENABLED=0, GOTOOLCHAIN=local, GOMAXPROCS=4, TMPDIR/GOTMPDIR=.agent/tmp, GOCACHE=~/.cache/go-build.
Python used -B. Node builds passed for linux/amd64 and linux/arm64; no physical
arm64 or field claim. Gofmt is empty, all git diff --check modes pass, owned
scope and evidence hygiene pass, prior runner/test/record hashes are unchanged,
and final owned Docker inventory is empty. Plain go test ./... passed before
the final commit. The JSON retains exact gate commands, exit codes, output tails,
log hashes, admission wait counts and source identities.
