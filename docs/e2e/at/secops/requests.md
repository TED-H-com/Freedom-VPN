# Requests to orchestrator

No application or shared lab files changed. The corrected base 30000 runs;
the old port blocker is void. Standard HTTP and documented state/IdP hooks
provide live probes without copying the shared stack or its scenario runner.

| ID | Cases blocked | Owner / exact location | Observed reproduction and required behavior |
|---|---|---|---|
| SECOPS-03 | AT29, full AT31 | nodes/devstack: `apps/worker/jobs_nodes.go`, `NodeJobs`; `tests/e2e/scenarios_test.go`, `runner.enroll`, `runner.member` | Run the assigned stack, enroll the real agent, GET its detail, then PUT publication with current ETag. See `live.json` for the refusal and actual provisioning/health states. `nft list chain inet fvpn forward` remains generation 0 with zero member accepts. Establish a serviceable node, execute publication with observed worker read-back, authorize/bind a real member peer, and prove a counted controlled flow. Await wire-nodes follow-up 5 and devstack follow-up 3; no duplicate implementation here. |
| SECOPS-04 | AT29 | node lab: `deploy/devstack/entrypoint-node.sh`, `tests/e2e/harness.go` | Once SECOPS-03 is resolved, supply short lease and finite paid/grace/grant fixtures. Existing pause-agent hook is usable. Observe monotonic deadline, kernel set timeout, successful flow before cut and stop point afterward. Clock rollback requires a disposable VM or injected time source; Docker shares host wall time. |
| SECOPS-05 | full AT30 | DB/devstack: `deploy/devstack/compose.yaml`, `docs/db/README.md`, `docs/e2e/README.md` | Live `SHOW archive_mode` returns `off`. The measured network-none logical restore preserves hashes and deduplication but is not PITR. Provide a base-backup/WAL archive and isolated full-stack checkpoint including current provider/NetBird state, independently retained revocation journal, original node SQLite archive and restored workers. Replay and reconcile before measuring full-service RPO/RTO. |
| SECOPS-06 | full AT31 | devstack: `tests/e2e/cmd/fvpn-e2e-idp/main.go`, `server.assume`/`authorize`; `deploy/devstack/nginx.conf`; serving fixture | The IdP stores a single global `next` identity, consumed by authorize; it cannot safely stage 20 distinct simultaneous sign-ins. Add browser-scoped assumption tickets. The live bounded API sample preserves existing limits and records 429/Retry-After separately. Full mixed/relay load also requires SECOPS-03 and a positive controlled relay path; historical gateway relay 404 is not reclassified as a live throughput result. |
| SECOPS-07 | AT32 | README, portal and member-guide owners | Resolve DOC01–DOC13 in [scope-review.json](scope-review.json): client fallback, nonexistent Authorize button, unsupported platform certification, DNS redirection/bypass claims, latency guarantees, device-ID label and stale spec references. Keep desired/applied and kill-switch caveats. |
| SECOPS-08 | AT29–AT32 physical rows | operator / acceptance orchestrator | Supply physical arm64, second NAT, four member platforms and candidate hardware; run [operator-runs.md](operator-runs.md). Cross-compilation and development-host timings are not substitutes. |

AT32's spec requirements also include NODE01, NODE04 and SCOPE01. The source/model
consistency requirement and the requested public-statement review are both kept.
Per-case JSON is authoritative for actual statuses and measured values.
