# Dev-stack owner requests — follow-up 10

Current evidence: [merged-tree verification](followup10-verification.md#merged-tree-run--send-back-f10b),
run `3a129919-af3f-4bc9-b54a-8075eae8af45` on merged product tree `79259af`:
**21 pass / 5 blocked / 0 fail**, Go exit 1 / make exit 2, teardown 0.
The dated root contains this complete run. `followup10-run1/`,
`followup10-run2/` and `followup10-run3/` preserve the original three runs.

## Merged-tree route discovery timing — lab owner

Owner: **w3-devstack-e2e**. [Member traffic](runs/2026-10-30/member-traffic.md)
records `No networks available.` at the immediate exit-selection check, followed
by `expected exactly one API-published exit`; its origin route still uses `eth0`.
Stored applied/desired/lease generations are all 5 with heartbeat `serving 1`.
At `16:29:28Z`, a later read-only check on the same client shows the default exit
available but `Not Selected`. The single availability check ran before the
route appeared; this evidence does not establish a product forwarding defect.

**Request:** add bounded route-discovery polling before explicit selection and
re-run the positive flow. In this run `member-traffic`, `usage`, `forwarding`,
`device-revoke` and `lease-expiry` remain blocked at that prerequisite. No
revocation DELETE or control-stop fault is injected without a counted flow.
The existing device-revoke owner string describes its later historical gate;
the current record's actual gate is `positive member flow prerequisite absent`.
No route/group/projector row was repaired during this send-back.

## F3-02 — missing executable resolved; strict ingress measurement remains open

The original three runs below remain historical proof; the merged run did not
reach this scenario beyond its positive-flow prerequisite.

Owner of the remaining measurement: **w3-devstack-e2e**. The fresh image has
`/usr/sbin/conntrack`. Member DELETE returns `202`;
a newer serving generation is applied, the agent PID stays unchanged, the
sibling keeps a counted flow and a replacement native device also carries a
counted flow. The [original run 3 record](runs/2026-10-30/followup10-run3/device-revoke.md) retains the accepted
observation, ready/passed row, counters and conntrack result. The old missing
executable no longer prevents serving apply. Harness owners should rebuild the
current image rather than install a package at runtime or skip conntrack deletion.

In the original three runs, the complete `device-revoke` case is **blocked**
because NetBird withdraws the revoked client's advertised route before the captured
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
The #154 issuer validator remains in force. The lab explicitly supplies
`agent_config=/etc/vpn-node/agent-run.json`; unit checks exercise the actual
LoadConfig contract with control/resolver endpoints and local interface inventory.
The isolated origin is accepted; documentation/CGNAT destinations, missing agent
configuration and control/resolver collisions are refused. The live updater
reports `probe_omission_count=0`, `invalid_probe_peer=false`, and a successful
candidate executable with updater exit 0.

Measured window: `{"samples":843,"grant_samples":3,"source_elements":1,"destination_elements":1,"max_timeout_seconds":44,"probe_up_bytes":1848,"probe_down_bytes":1848,"member_bytes_before":0,"member_bytes_after":0,"elements_after":0}`.
Measured requests: `{"attempts":1311,"authenticated_udp_replies":24}`.
The origin records 24 authenticated requests from the node's SNAT source.
Member totals were 0 → 0; this establishes no member accounting change during
the probe, not a positive member-flow proof. Both diagnostic counters must advance,
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
