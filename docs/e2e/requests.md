# Dev-stack owner requests — follow-up 10

Current evidence: [verification](followup10-verification.md) and three complete
fresh-stack sets on product base `27d9229` (PR #153). The dated root is run 3.

## F3-02 — missing executable resolved; strict ingress measurement remains open

Owner of the remaining measurement: **w3-devstack-e2e**. The fresh image has
`/usr/sbin/conntrack`. Member DELETE returns `202`;
a newer serving generation is applied, the agent PID stays unchanged, the
sibling keeps a counted flow and a replacement native device also carries a
counted flow. The [record](runs/2026-10-30/device-revoke.md) retains the accepted
observation, ready/passed row, counters and conntrack result. The old missing
executable no longer prevents serving apply. Harness owners should rebuild the
current image rather than install a package at runtime or skip conntrack deletion.

The complete `device-revoke` case is **blocked** because
NetBird withdraws the revoked client's advertised route before the captured
post-apply request. The observed route uses `eth0`, the client has no exit network, node
`fvmb0` captures zero matching packets and `ctunku` does not increase. Revoked
upload/download counters freeze, conntrack is empty, origin receipt is zero,
and sibling/replacement traffic succeeds. These are partial withdrawal proofs,
not a measured node-side drop. DELETE-to-first-observed-drop timing is
`not observed; route withdrawal stopped traffic before node ingress`.

Run 3 additionally records `transition_ctunku_before=0` and
`transition_ctunku_after=2` while requests were in flight across DELETE.
The later captured request has `ctunku_packets_before=2` and
`ctunku_packets_after=2`, with zero matching packets captured. The two
transition drops have no captured source attribution or first-packet
timestamp, so they do not establish the requested revoked-source timing.

**Request to orchestrator:** retain a separate controlled stale-client transport
case for the kernel-ingress assertion, or explicitly separate the upstream
withdrawal and kernel-drop assertions. This run does not restore a revoked
route/group, spoof an admitted source or modify enforcement to manufacture that
packet. No agent/enforcement defect is inferred from the earlier route cutoff.

## D-063 / D-064 — dedicated synthetic traversal

The [update-good record](runs/2026-10-30/update-good.md) is
**pass**. A non-member setup-key peer selects a dedicated
origin `/32` route. Its separate transport policy targets only the enrolled node;
no member device, grant, binding or publication row is created for it.
The original updater destination validator remains in force; unit checks accept
the isolated origin and still reject documentation and overlay destinations.

Measured window: `{"samples":673,"grant_samples":3,"source_elements":1,"destination_elements":1,"max_timeout_seconds":44,"probe_up_bytes":1078,"probe_down_bytes":1078,"member_bytes_before":15044,"member_bytes_after":15044,"elements_after":0}`.
Measured requests: `{"attempts":944,"authenticated_udp_replies":14}`.
The origin reports the node's SNAT source. Both diagnostic counters must advance,
member upload/download totals must stay unchanged, and source/destination elements
must be empty after the answer. The nftables JSON timeout unit is seconds;
the check requires at most 45. Only the product installs/removes the exception
and writes the challenge-bound answer. The external lab observer supplies the
traffic proof; an `overlay_up` health reason by itself is not that proof.

## SECOPS-10 and SECOPS-16 — gateway and protected native streams

Full cohort: `{"Users":20,"Peers":60,"IdentityMatches":60,"Authorized":60,"ScannedLogs":60,"ProtectedLogs":60,"SimultaneousAuthorized":60}`. The final simultaneous
read checks the exact admitted device IDs. Nginx uses two workers with 4096
connections each for the documented 60-peer cohort plus services. This is native
admission/gateway evidence, not a full mixed-traffic AT31 hardware certification.
Long runs can provide the optional real-admin-login callback to
`ScaleNativeMembers`; no recent-auth window is extended.

The controlled upstream-error replay returned HTTP
`502` with sentinel/query matches
`0` / `0` and connection-budget
error lines `0`. HTTP diagnostics retain only
server/method/status because nginx native error strings can append raw query
values. Process-level errors remain available. `ScanNativeJoinLog` runs inside
the owned container before deletion and returns only presence, size, owner/mode
and recognized device-flow-key line counts. `NativePeer.LogScan` and the scale
inventory expose those results without any matching secret value.

**Request to secops:** adopt the returned inventory and rerun the broader
AT28/AT31 mixed workload using the current image and gateway configuration.

## F3-03 — second real exit hook available; topology remainder stays open

`StartSecondExit` and `EnrollAt(..., "node2")` are exported through
`tests/e2e/internal/labstack`; see [RUN.md](RUN.md#second-runnable-exit-f3-03).
The [second-exit record](runs/2026-10-30/second-exit.md) is
**pass**, with distinct identity
`true`, provisioning
`"ready"` and safety `"passed"`.
It has independent durable state, resolver and network namespace on the same
lab bridges. Dynamic peers use a separate `.128/25` pool so a new peer cannot
take a stopped static service's IP.

**Request to harness owners:** use the real second node and normal publication
APIs instead of the portal's synthetic `SEEDED_NODE_2_ID`. Two-exit counted traffic,
persistent TCP/UDP timing hooks, other boot sources, native platform/network moves,
forced relay/PMTU, a second filtering resolver, controlled DNS variants and wider
retention fixtures remain the respective harness/operator backlog. Physical
arm64 and physical network/boot flavours remain **needs operator run**.

## Earlier resolved findings and measurement limits

The [follow-up 9 trace](followup9-forwarding-trace.md) remains the explanation
for explicit native exit selection. The resolver corruption hook, per-session
IdP tickets and fresh admin login before revoke remain available. SNAT rules
still lack a counter expression; packet/conntrack/origin evidence establishes
translation without adding rules. The control-loss record is a bounded signed
policy-expiry withdrawal, not the complete AT29 crash/clock/grace campaign.
