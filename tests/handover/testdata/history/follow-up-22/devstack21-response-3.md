## members / agent-core / netlab — 2026-10-31 (follow-up 21, F11-01–03, D-102)

F11-01's [fixture analysis](scenarios/followup21-members.md#f11-01-the-first-unobserved-segment)
places the first unobserved segment after member inner `fvnb0` egress and
before node observable inner ingress. Five member packets, zero matching
node/origin packets, generation-5 zero admission/diagnostic counters and no
origin request are retained. The fixture's inner TCP filter cannot observe the
encrypted outer path, handshake, relay or link drops; it cannot assign the
missing transport sub-hop. This is not a generation-0 baseline or proven node
enforcement drop. Netlab 21's transport ownership remains; AT01 stays blocked.

F11-03's [original input/timing extract](followup21-members.json) and
[health-check analysis](scenarios/followup21-members.md#f11-03-what-the-failed-health-result-actually-asserts)
retain the exact two injection inputs and twelve generation-13 Applies for
agent-core 29. All eight failed Applies have `install_ok=false`, 184–304 ms
render/install brackets against the unchanged 100 ms bound, unreleased guard
and no verification records. `HEALTH_CHECK_FAILED/stage=apply` is the sanitized
Apply error, not an observed failed tunnel challenge. Capture/injection work
can add load, but historical command error/exit, resource pressure and injection
monotonic bounds were not retained; causation is not established. The fixture
cannot recreate the original full model/transaction from a later stack. No
health criterion, fixture timing or positive sibling assertion is changed;
the correct revoked-source drop does not pass the failed sibling control.

F11-02 is a members readiness prerequisite: a current-generation unexpired
lease with zero grants and `AWAITING_MEMBER_GRANTS` is not serving. Future D-077
positive probes must wait for an accepted serving report joined to a populated
current-generation lease and the intended device's binding, then independently
assert counted forwarding. Keep the original generation-10 failed transfer.
The shared hooks do not manufacture grants or force generations. Members owns
its `forwardingGate` in `tests/e2e/at/members/forwarding_test.go`; no recipient
harness change or substitute passing rerun is made here. F10-02/F10-03 historical
failures remain unattributed, while members 11's fresh diagnostics remain passes.
