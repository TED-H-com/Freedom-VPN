## members / netlab — 2026-10-31 (follow-up 22, F11-01, D-102)

> **2026-10-31 — members 12 and native transport owner, F11-01:**

`StartNativeCapture`, `Sample`, `Stop` and selected-peer readbacks are available
through `labstack` / `MemberTransportStack`. The [export contract](scenarios/harness-exports.md#follow-up-22--selected-native-path-capture-test-only)
specifies ready/stop and per-command monotonic bounds, route/link/ICE/relay/
WireGuard endpoint readbacks, private pcap receipts and payload-free joins.
Start before the existing route selection; sample during selection and the
original transfer. It does not reconnect, select an exit, wait for serving or
repair transport. Members 12 owns adoption in
`tests/e2e/at/members/forwarding_test.go`; that file is unchanged.

The [final fresh reproduction](runs/2026-10-31/followup22/9b3a89e7-fa86-426c-9737-ce55bf66aa77/analysis.md)
has **five SYNs**, sequence **1392714787**, source port 20001: member tunnel
packets **8, 144, 155, 160, 163**. None reaches node inner ingress or origin.
The HTTP attempt fails after **5,004 ms**; origin requests = 0, node upload /
download remain **0 → 0**, generation **5**, heartbeat serving=true. It runs
before the ordinary counted-flow retry. No node health repair is applied.

The first SYN's WireGuard initiation (member lo packet 9) has the same message
hash/index as node lo packet 133. Node eth2 packet 134 responds; the identical
response reaches member eth1 packet 137. Its IPv6 **source is loopback**, token
`addr-eff8e7ca506627fe`, distinct from the selected node endpoint. No structurally
valid WireGuard type-4 transport-data message appears at member (including lo)
or node during the request. Readbacks transition from local endpoint/pending
to IPv6 P2P connected while all subsequent inner SYNs still fail. The first
unconfirmed data step in this reproduction is **member inner tunnel egress →
observable WireGuard transport-data emission**, before node inner ingress.
The handshake response reaching member underlay does not prove kernel/session
acceptance. This is not evidence of an enforcement fault.

All three captures are complete (167 / 172 / 30 records), zero drops/truncation;
metadata reads now succeed. Every packet has an empirical monotonic interval
with explicit wall-clock stability assumptions. Node's later handshake tail
is after the member capture stopped and cannot prove receiver loss/recovery.
The loopback-source response and endpoint transition are handed to the
**native NetBird/WireGuard integration owner** (NetBird 0.82.0 binary pin in
`deploy/devstack/Dockerfile.node`; `node/agent/probe.go` only observes status).
Please trace handshake/session acceptance before assigning a fault or repair.

The first campaign's one-SYN success and missing origin/WireGuard readbacks
remain retained as history. The final run reproduces the failure pattern on
a fresh stack; it does not replay all members 11 history or identify those
historical packet identities. Members 12 should compare its own original
sequence using these exports before transferring a root-cause conclusion.
