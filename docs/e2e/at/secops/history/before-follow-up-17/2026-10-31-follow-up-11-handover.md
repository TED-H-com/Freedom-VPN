# 2026-10-31 — Follow-up 11 handover (live execution pending)

D-098 is complete: `history/follow-up-10/` contains all 104 root JSON/Markdown
records exactly as merged in PR #265, commit `2f1f939`. Each byte and SHA-256
is checked against that commit. Follow-up 10 run
`0e8ff9a4-4fa1-4154-ae35-847a5be4d985` is under
[restore-reconcile.json](history/follow-up-10/restore-reconcile.json), key
`measurement_comparison.follow_up_10`.

| Archived Follow-up 10 measurement | Value |
|---|---|
| Pre-replay database RPO | 95.507504 s |
| Post-replay service usage RPO | 0 s |
| Protocol RTO | 93.532554563 s |
| Total outage RTO | 106.868537734 s |

`history/follow-up-9/restore-reconcile.json` is unchanged. The existing
`TestHandover_Followup13Claims` already reads that snapshot and passes; no
handover test change is needed. Current root measurements remain Follow-up 10.

| Follow-up 11 sub-case | Current result | Reason / owner |
|---|---|---|
| AT29 established TCP/UDP finite grant expiration | blocked, not run | Docker namespace clarification pending / orchestrator |
| AT29 established TCP/UDP lease expiration with control loss and agent SIGKILL | blocked, not run | Docker namespace clarification pending / orchestrator |
| AT29 established TCP/UDP device revoke | blocked, not run | Docker namespace clarification pending / orchestrator |
| AT29 explicit manual grant withdrawal | blocked | No revokeMemberGrant HTTP operation in revision 13 / contracts + w3-wire-members |
| AT29 guest clock rollback | blocked | No independently clocked disposable guest / w3-devstack-e2e + operator |
| AT31 bounded relay then direct restoration | blocked, not run | Docker namespace clarification pending / orchestrator |
| AT31 complete capacity case | blocked | Selected candidate hardware and 60-peer mixed relay load remain unmeasured / operator + acceptance lane |
| AT30 regression | blocked, not run | Docker namespace clarification pending / orchestrator |
| SECOPS-21 | open — D-099 pending | No wire-nodes follow-up 29 roadmap row on base `c6e3eca`; no new first-offer measurement |

The brief simultaneously assigns the existing `w4-secops` stack (which creates
`fvpn-w4-secops-*` objects and the `fvpn-w4-secops` network) and restricts Docker
objects to `fvpn-w4-at-security-ops-*`. A clarification is pending before any
Docker mutation. No substitute stack, port allocation or resource renaming was
invented. The preparation uses only owned harness paths.

Prepared live commands after clarification:

```sh
make -f mk/at-secops.mk at-secops-followup11-transports
FVPN_SECOPS_FOLLOWUP11=restore make -f mk/at-secops.mk at-secops-followup10
python3 tests/e2e/at/secops/followup11_records.py
python3 tests/e2e/at/secops/followup11_guards.py
```

Check `pgrep -af gates.sh` before each run and wait if two suites are active.
The restore target retains the full Follow-up 10 drill and writes Follow-up 11
records. It freezes before services start, verifies the exact D-095 HTTP/node
hold while the agent is stopped, then resumes the retained agent. No production
row or generation is repaired. D-099 eligibility is recorded from the measured
main ancestor. First compiled rows, the HTTP probe and local final state remain
distinct evidence; without proof of the first delivered offer's local store and
ACK, SECOPS-21 cannot close even after D-099 merges.

Export limits are from `docs/e2e/RUN.md`: one helper per member; original
TCP/9446 and authenticated UDP/9445 sockets; independent attempt counters;
20-minute helper lifetime; `Present=false` counters are not zero-byte proof;
relay match expires after 180 seconds and each path wait is bounded at 90
seconds. Native route withdrawal and exit enforcement are not independently
attributed by these exports. No measured transport pass is claimed here.
