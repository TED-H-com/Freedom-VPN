## Requests routed by the orchestrator

1. Usage (AT14–AT16) → usage follow-up 8 (collected): add `ctdenyd` to the synthetic exclusion input in `tests/e2e/at/usage/at14_16_node_test.go`, and assert that the download source guards precede ordinary `d_` admission while resolver DNS `nd_` replies stay ahead of them. Archived rows stay as they are.
2. Devstack → devstack follow-up 28 (collected): `ctdenyd` coverage in `tests/e2e/internal/counterread/read_test.go`. A nonzero `ctdenyd` must survive parsing and never change the returned usage or presence.
3. Agent-core → agent-core follow-up 35 (collected): add `ctdenyd` to the diagnostic fixtures in `node/agent/engine_test.go` and `node/agent/probeexc_test.go`, and cover the filtering in `node/agent/usage.go` and `meterreset.go`, so the diagnostic can create neither member usage nor a usage epoch.
4. Spec owner → the owner's open questions: should a RELATED ICMP error with a denied, private, host or resolver outer source pass when its quoted connection is allowed? The fail-closed guard stays until an explicit, scoped exception and its kernel evidence are specified.
5. Netlab follow-up 25 (collected, the orchestrator's): the D-120 missing-successor condition (A2 on `main` and the earlier branch attempt) as a measurement-end question, with clause (iii) unchanged and every historical failure kept; large JSONL evidence published as lossless gzip with both SHA-256s, as isolation and members do.
6. Handover follow-up 28 (collected): r19 H-1 fixed here.
