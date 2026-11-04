# Follow-up 47 validation — 2026-11-03

Final non-timing checks passed after one preserved full-module failure and recheck.
The ordinary uninstrumented main lab passed 13/13.
The reserved dry A leg is separately a failed scenario run (12/13), fully
retained in the dry evidence; it is not represented as a passing gate.
Admission required exact free before every make, Docker and Go invocation.
All gates ran outside reservations, after the dry release.
The first plain go test ./... failed the unchanged R23 checkpoint-read/restart
subtest at followup40_test.go:228 with a boot heartbeat context deadline. It
had passed the earlier review gate; a targeted recheck and one plain full-module
retry passed without code changes. The initial exit 1 and raw log remain
recorded. No root cause is inferred from the successful rechecks. No refused
admission was treated as a test result. The JSON retains gate commands, source
identities, output tails, log hashes and admission wait counts.

Root go-lint used its pinned staticcheck and the already set disk environment;
its go-with-disk-tmp prefix was overridden to `set -eu;` to avoid a shell cleanup
trap. No other lint command or rule changed.

| Gate | Exit | Seconds |
| --- | --- | --- |
| node-lint-build | 0 | 3.838 |
| node-test | 0 | 149.039 |
| node-integration | 0 | 136.882 |
| reviews-r16-r19 | 0 | 261.313 |
| reviews-r20-r26 | 0 | 44.177 |
| handover | 0 | 3.900 |
| go-lint | 0 | 2.543 |
| vet | 0 | 0.488 |
| evidence | 0 | 3.057 |
| fixture-hygiene | 0 | 8.959 |
| final-all-tests | 1 | 173.397 |
| r23-recheck | 0 | 1.740 |
| final-all-tests-retry | 0 | 146.662 |
| ordinary main lab (13/13) | 0 | 228.962 |
| synthetic tests (97 root; 32 other directory) | 0 | — |

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
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local go test ./node/agent/... ./node/watchdog/... ./node/internal/agentstore/... ./node/internal/controlclient/... ./node/internal/timeanchor/... ./node/internal/fakecontrol/... ./node/internal/nodelock/... ./node/internal/usagestore/... ./node/internal/meter/...
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	147.489s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.293s
ok  	github.com/TED-H-com/Freedom-VPN/node/watchdog	0.111s
?   	github.com/TED-H-com/Freedom-VPN/node/watchdog/cmd/fvpn-watchdog	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/agentstore	0.170s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/controlclient	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol	0.027s
?   	github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol/cmd/fvpn-labcontrol	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/node/internal/nodelock	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/usagestore	1.004s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/meter	0.665s

node-integration:
TMPDIR='<worktree>/.agent/tmp' GOTMPDIR='<worktree>/.agent/tmp' CGO_ENABLED=0 GOTOOLCHAIN=local go test -count=1 -tags integration ./node/agent/...
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	136.223s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.290s

reviews-r16-r19:
=== RUN   TestReviewR18_DisabledLifecycleDoesNotAllocate
--- PASS: TestReviewR18_DisabledLifecycleDoesNotAllocate (0.00s)
=== RUN   TestReviewR18_DisabledRunCommandMatchesRun
--- PASS: TestReviewR18_DisabledRunCommandMatchesRun (0.01s)
=== RUN   TestReviewR18_RenderCaptureStaysInMemory
--- PASS: TestReviewR18_RenderCaptureStaysInMemory (0.00s)
=== RUN   TestReviewR18_EnabledSampleIgnoresInstallDeadline
    review_r18_lifecycle_test.go:107: finished install context skipped resource sampling (elapsed 672.088µs, total syscr +11)
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
ok  	github.com/TED-H-com/Freedom-VPN/node/enforce	0.023s

handover:
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.542s

go-lint:


vet:


evidence:
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab	2.799s

fixture-hygiene:
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.676s

final-all-tests:
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.024s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.091s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.005s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.789s
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab	2.874s
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/at	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab/cmd/fvpn-netlab	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/dns/cmd/cache-authority	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/dns/internal/cachefixture	[no test files]
FAIL

r23-recheck:
=== RUN   TestR23_H2_InterruptedCheckpointPreparesNextEpoch/removed_restart_true
=== RUN   TestR23_H2_InterruptedCheckpointPreparesNextEpoch/replaced_restart_false
=== RUN   TestR23_H2_InterruptedCheckpointPreparesNextEpoch/replaced_restart_true
=== RUN   TestR23_H2_InterruptedCheckpointPreparesNextEpoch/checkpoint_read_restart_false
=== RUN   TestR23_H2_InterruptedCheckpointPreparesNextEpoch/checkpoint_read_restart_true
=== RUN   TestR23_H2_InterruptedCheckpointPreparesNextEpoch/checkpoint_write_restart_false
=== RUN   TestR23_H2_InterruptedCheckpointPreparesNextEpoch/checkpoint_write_restart_true
--- PASS: TestR23_H2_InterruptedCheckpointPreparesNextEpoch (1.35s)
    --- PASS: TestR23_H2_InterruptedCheckpointPreparesNextEpoch/removed_restart_false (0.17s)
    --- PASS: TestR23_H2_InterruptedCheckpointPreparesNextEpoch/removed_restart_true (0.19s)
    --- PASS: TestR23_H2_InterruptedCheckpointPreparesNextEpoch/replaced_restart_false (0.15s)
    --- PASS: TestR23_H2_InterruptedCheckpointPreparesNextEpoch/replaced_restart_true (0.17s)
    --- PASS: TestR23_H2_InterruptedCheckpointPreparesNextEpoch/checkpoint_read_restart_false (0.16s)
    --- PASS: TestR23_H2_InterruptedCheckpointPreparesNextEpoch/checkpoint_read_restart_true (0.18s)
    --- PASS: TestR23_H2_InterruptedCheckpointPreparesNextEpoch/checkpoint_write_restart_false (0.15s)
    --- PASS: TestR23_H2_InterruptedCheckpointPreparesNextEpoch/checkpoint_write_restart_true (0.19s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	1.359s

final-all-tests-retry:
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-stream	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.019s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.061s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.005s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/at	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab/cmd/fvpn-netlab	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/dns/cmd/cache-authority	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/netlab/dns/internal/cachefixture	[no test files]

ordinary-main-lab:
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab/agent	214.736s
make[1]: Entering directory '<worktree>/.agent/tmp/f47-production'
docker rm -f fvpn-w2-agent-core-f47 >/dev/null 2>&1 || true
docker network rm fvpn-w2-agent-core-f47-dns >/dev/null 2>&1 || true
<worktree>/.agent/f47/bin/make -f mk/netlab.mk netlab-wipe-scratch FVPN_LAB_PREFIX='fvpn-w2-agent-core-f47'
make[2]: Entering directory '<worktree>/.agent/tmp/f47-production'
mkdir -p <worktree>/.agent/tmp/f47-production/.agent/tmp/fvpn-w2-agent-core-f47-scratch
docker rm -f fvpn-w2-agent-core-f47-wipe >/dev/null 2>&1 || true
docker run --rm --name fvpn-w2-agent-core-f47-wipe --network none \
	-v <worktree>/.agent/tmp/f47-production/.agent/tmp/fvpn-w2-agent-core-f47-scratch:/scratch fvpn-w2-agent-core-f47:lab \
	find /scratch -mindepth 1 -delete
make[2]: Leaving directory '<worktree>/.agent/tmp/f47-production'
rmdir <worktree>/.agent/tmp/f47-production/.agent/tmp/fvpn-w2-agent-core-f47-scratch 2>/dev/null || true
make[1]: Leaving directory '<worktree>/.agent/tmp/f47-production'
```

Environment: CGO_ENABLED=0, GOTOOLCHAIN=local, GOMAXPROCS=4, TMPDIR/GOTMPDIR=.agent/tmp, GOCACHE=~/.cache/go-build.
Python used -B. Node builds passed for linux/amd64 and linux/arm64; physical
arm64 and field networks were not measured. Gofmt is empty, git diff --check
passes, owned scope and evidence hygiene pass, earlier runner/test/record hashes
are unchanged, and final owned Docker inventory is empty. Plain go test ./...
passed before the final commit. No production, node/enforce or dependency edit.

Final records evidence recheck: exit 0, 8.519 seconds.

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/netlab	2.728s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.247s
```
