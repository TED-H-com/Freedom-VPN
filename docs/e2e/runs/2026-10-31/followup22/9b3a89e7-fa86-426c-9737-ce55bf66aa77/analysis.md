# Final campaign: native IPv6 MTU and five initial SYNs

TEST ONLY, Linux amd64. Run `9b3a89e7-fa86-426c-9737-ce55bf66aa77`, captured
with source `a1796b0d2be7f0ef068f66347fccb11d55ed58a4`. Post-run header-only
WireGuard decoding and monotonic analysis use `bf82cbe4`; raw pcap SHA/size
receipts in the scenario records are unchanged. No payload, key or ciphertext
is exported. The process exits **1**, with **13 pass / 0 blocked / 2 fail**:
`initial-selection` and `ipv6-asymmetric` fail their original predicates.

## F8-02: explicit selected-sender constraint

The original members mutation order is IPv4/1500 → IPv6/1500 → dual/1500 →
dual/1280, with the original reconnect/settle behavior. All four pass this run;
its last row selects IPv4. The preceding campaign's last row selected IPv6
and passed with smaller reverse packets. Neither replaces the historical
members 11 IPv6 failure or devstack 20 `cce1a688`.

| Row | Selected family | Validated bulk bytes | Elapsed ms |
|---|---|---:|---:|
| matrix4 | ipv4 | 64,064 | 1 |
| matrix6 | ipv6 | 64,064 | 1 |
| matrixDual | ipv4 | 64,064 | 1 |
| matrix1280 | ipv4 | 64,064 | 2 |
| ipv6-asymmetric | ipv6 | 16,016 | 12,011 |
| ipv6-explicit1280 | ipv6 | 64,064 | 1 |
| ipv6-explicit-repeat1 | ipv6 | 64,064 | 1 |
| ipv6-explicit-repeat2 | ipv6 | 64,064 | 1 |

The supplemental asymmetric IPv6 row still lowers **only member** eth0/eth1
MTU to 1280. Node underlay remains 1500 and both tunnel interfaces 1280.
It receives **16,016 / 64,064 bytes in 12,011 ms**. Every one of the **15**
reverse outer **1,360-byte** datagrams leaves node eth2 but is absent at member
eth1. Every smaller selected-endpoint packet arrives, with zero unmatched
receiver datagrams. First missing TCP range **[3005476689,3005477917)** is
1,228 bytes; it appears at origin and node uplink/tunnel, not member tunnel.
All **52** upload fragment pairs (1,280 + 136 bytes, second offset 1,232) arrive.
Member/node/origin counts **668 / 604 / 148** exactly match decoded records,
with zero kernel drops, truncation or decode errors. The first campaign
independently lost all 23 oversized reverse packets with the same mechanism.

The selected sender control **adds a fixture-only /128 route, MTU 1280** to
its member peer, then removes exactly that route. Before/after/final readbacks
prove its presence and cleanup. This is a known path constraint, chosen to
isolate sender sizing without adding a router or changing global MTU/MSS.
There is no reconnect between asymmetric and explicit rows. All three explicit
controls receive **64,064 bytes in 1 ms** each. In `ipv6-explicit1280`, all
**7** reverse 1,280 + 136 fragment pairs arrive. In `ipv6-explicit-repeat2`,
all **44** first fragments (1,280 bytes), **43** second fragments (136 bytes)
and one second fragment (88 bytes) arrive by exact full-IP hash. The current
bulk ports are respectively **58402** and **47948**; only those reverse TCP
streams occur in their command windows. Unlike the first campaign's single
old-flow retransmission, these captures show fragmentation accompanying the
current successful bulk. Byte coverage is complete despite GRO/segmentation
differences. No unfragmented oversized reverse datagram appears in these
explicit-path windows.

**No ICMPv6 PTB or IPv4 fragmentation-needed appears at any of the three
namespaces in any of the nine captured rows.** Node/tunnel therefore never
observes PTB in this experiment. The sender fragments with its explicit route
metric; this is not measured reactive PMTU discovery. The original topology
is an **asymmetric receive-link fixture artifact**, without a router or sender
path signal. Receive-MTU rejection is the location/size-based inference, not
a captured kernel drop reason. No real member-platform link is certified.
If a real known-1280 receiver is established, peer packetization belongs to
pinned NetBird/WireGuard integration, not an enforcement MSS clamp.

## F11-01: five SYNs reproduced and outer interval narrowed

This fresh first-selection experiment uses the original fixed source port
**20001**, two-second settle, **5-second dial / 6-second HTTP** bound, and
runs before the ordinary counted-flow retry. It does not replay all members
11 prehistory and cannot reconstruct those historical packet identities.
It reproduces their **five-SYN, zero-forwarding** pattern: sequence
**1392714787**, destination 443, all five on member `fvnb0` egress; **none**
on node `fvmb0` / `fvup0` or origin. HTTP fails after **5,004 ms**, origin
requests = 0, node device upload/download = **0 → 0** at generation **5**.
Heartbeat samples remain serving=true. No repair or health substitution runs.

| Member packet | Unix wall timestamp | Empirical monotonic interval, ns |
|---:|---|---|
| 8 | 1793485082.974273 | [68620490639114, 68620490645887] |
| 144 | 1793485084.020789 | [68621537155114, 68621537161887] |
| 155 | 1793485085.044641 | [68622561007114, 68622561013887] |
| 160 | 1793485086.069663 | [68623586029114, 68623586035887] |
| 163 | 1793485087.092662 | [68624609028114, 68624609034887] |

All packets, including absent-hop checks, retain the same-boot clock joins in
[analysis.json](headers/analysis.json) and role `*.bounds.jsonl` files. The
measured offset spans are **5,773 / 1,422 / 1,148 ns** for member/node/origin;
packet bounds add 1,000 ns for pcap quantization and assume no intervening wall
step. These are empirical bounds, not hardware timestamps or exact route
commit times. Captures decode **167 / 172 / 30** records, with **zero** drops,
truncations or errors. All three link/origin-route readbacks succeed; both
native interfaces report the kernel implementation, and public WireGuard
endpoint reads succeed. This closes the first campaign's missing-tool gaps.

Before first transfer, WireGuard targets a local IPv4 endpoint (route `lo`)
and native peer status is pending. During the first SYN's bracket the member
WireGuard endpoint is briefly absent. By the next sample (member
22:18:03.219560996–22:18:03.234765342Z), the selected peer reports connected
IPv6 P2P and its endpoint route is eth1; node route is eth2. Those readbacks
remain connected through the failed request. They do not prove a usable
WireGuard data session.

The optional fixed public [WireGuard protocol headers](https://www.wireguard.com/protocol/)
join the handshake across changed outer IP headers without exposing keys or
ciphertext. Structural classification does not authenticate messages.

| Hop | Packet / Unix timestamp | Public header evidence |
|---|---|---|
| Member lo | 9 / 1793485082.974376 | Initiation, sender index 2673567615, message hash `da8dccfa…` |
| Node lo | 133 / 1793485083.176940 | Same initiation/hash/index; IPv6 destination classified loopback |
| Node eth2 egress | 134 / 1793485083.177064 | Response to index 2673567615, message hash `ef5ecd0d…`, IPv6 **source classified loopback** |
| Member eth1 ingress | 137 / 1793485083.177072 | Same response/hash/index and loopback source |

The response source token `addr-eff8e7ca506627fe` differs from the selected
node IPv6 endpoint `addr-837a7935ba5539b1`. Its loopback classification is
derived from the captured IP header. Thus handshake traffic reaches node
and returns to member underlay; the response alone does not prove member
kernel acceptance. **No structurally valid WireGuard type-4 transport-data
message is observed at member (including lo) or node during this request.**
The later four SYN retries remain visible only as member inner tunnel egress.
The first unconfirmed data step in this reproduction is therefore **member
inner tunnel egress → observable WireGuard transport-data emission**, before
node inner ingress. No node enforcement fault is established. A stalled or
replaced handshake/session during local-endpoint → IPv6 P2P transition is an
inference; neither kernel queue/drop reason nor root cause was traced. The
loopback-source response is a concrete anomaly for the transport owner.

Node packets 167–170 show another direct IPv6 handshake at 22:18:08.309Z,
after the HTTP timeout and after member capture stopped (confirmed by
22:18:08.292001625Z). Those tail packets cannot establish member-side loss or
subsequent recovery. No reconnect or transport repair was used to make this
first attempt pass. The first campaign's successful one-SYN sample remains
independent evidence, not a reason to dismiss this failed sample.

## Ownership and retained scope

Members 12 owns adoption of the exported capture/readback contract in
`tests/e2e/at/members/forwarding_test.go`; that file is unchanged. The native
binary is NetBird **0.82.0**, pinned in `deploy/devstack/Dockerfile.node`;
`node/agent/probe.go` reads native status but does not implement this handshake.
Ask the native NetBird/WireGuard integration owner to investigate the local
proxy → IPv6 P2P transition and loopback-source response. No product transport,
health, enforcement or global MTU/MSS code changed.

All **27** final raw pcaps remain ignored and mode 0600 under
`.agent/tmp/followup22/<run>/<row>/`. Every packet count and SHA receipt is
verified. Public pins target only immutable run files. The DNS third-node /
installer origin `.24` overlap remains documented as separate reset stacks.
