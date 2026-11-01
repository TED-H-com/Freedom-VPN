# Follow-up 24 — F11-01 packet loss boundary and F13-01 evidence retention

2026-11-01. TEST ONLY, owned container namespaces. NetBird remains **0.82.0**,
image digest unchanged. `node/agent/probe.go` is unchanged and only observes.
All 154 [follow-up 23 pins](../../../followup23-captures.sha256) verified before
work; their directory is unchanged. This archive is pinned separately by
[followup24-captures.sha256](../../../followup24-captures.sha256).

## What was reproduced

Every row is a fresh reset stack. There is one bounded original request per
completed attempt, no successful replacement sample, no prerequisite counted
flow, no reconnect or forced health result. Debug/observer overhead affects
scheduling; these runs do not establish a natural reproduction rate.

| Attempt / immutable record | Input | Result | Distinguishing evidence |
|---|---|---|---|
| [95b1285f](95b1285f-cf5b-456a-843b-be4fe1849d2c/initial-selection.json) | dual | Instrumentation failed before selection | Invalid netdev expression; no first request. Retained, corrected in the next source revision. |
| [e201f325](e201f325-d0ef-4bbe-8d91-16cdab9ab8bc/kernel-path-analysis.json) | dual, selected IPv4 | One SYN, 106 ms, origin 1; counters 0→2409 / 0→2462 | Type-4 present; no `::1` response. |
| [d556673b](d556673b-8164-4b5a-92c4-9333049097e3/kernel-path-analysis.json) | dual, selected IPv4 | Five SYNs, 5004 ms timeout, origin 0, counters 0→0 | Six type-4 records on each peer, including loopback traffic; no `::1` response. A different retained failure, not the old IPv6/no-type-4 outcome. |
| [cacad79c](cacad79c-d64b-4692-8b08-290df23e7a6a/kernel-path-analysis.json) | separate IPv6 constraint | One SYN, 9 ms, origin 1; counters 0→2409 / 0→2463 | Type-4 precedes a `::1`-source response dropped before member IPv6 prerouting. |
| [951bd882](951bd882-8106-4514-a36a-a1d82ca71c76/kernel-path-analysis.json) | separate IPv6 constraint | One SYN, 109 ms, origin 1; counters 0→2265 / 0→2412 | Type-4 present; no `::1` response within the captured window. |

The original devstack 22/23 **five-SYN/no-type-4 IPv6** outcome was **not
reproduced** in these four completed attempts. It remains in its immutable
archive. The third completed attempt reproduces members 13's important ordering:
a successful first request/type-4 session **before** the loopback-source response.
The two IPv6 constraints are explicitly separate diagnostics, not original
members dual-stack runs. A two-second passive tail was added after e201f325;
it sends nothing and never changes the measured request result.

The IPv4 timeout has a separate native session/forwarding gap. Its loopback
transport data alone does not prove successful node inner forwarding. Member
WireGuard snapshots after the handover could not match the selected peer using
the old exact allowed-IP lookup; they are unavailable, not proof of an empty
kernel endpoint. The observer now joins the selected native status public key
**in memory only**, then exports endpoint/handshake metadata. Both later IPv6
runs match the selected peer in every sample. No earlier artifact was rewritten.

## The `::1` response: measured boundary

In **cacad79c**, all three pcaps are complete: member **253**, node **209**,
origin **50** records; zero kernel drops, no decoder errors/truncated records.
Raw pcaps, nftrace and debug logs remain private, with exact SHA/size receipts.
The derived record verifies those receipts before interpreting public headers.

1. Member initiation **35** traverses the retained relay/proxy and appears as
   node loopback initiation **186**, with the member's IPv6 endpoint as source
   and `::1` as destination. The WireGuard message hashes match exactly.
2. Node response **187**, source `::1`, leaves `eth2`. Member **219** receives
   the same IP/WireGuard datagram on `eth1`; response hash
   `8724b1944161f7c962221397e5a8d26534359941b81594157d1289b6f692658d`.
   Both are type 2, WireGuard length 92 / UDP length 100, ports 51820.
3. Member nft trace **79e95677**, lines **218–221**, sees the matching header
   shape at netdev `ingress_eth1`, continues through the observation rules,
   and records the chain's accept policy. It has **no later inet prerouting
   or input event**. Normal IPv6 UDP traces do traverse those later hooks.
   The observation table counts **one** loopback-source ingress and **zero**
   loopback-source prerouting/input packets. No traced rule drops this packet.
4. Both namespace `/proc/net/snmp6` and absolute `nstat -asz` show
   **Ip6InHdrErrors 0→1**, after the request and before the end of the passive
   tail. `Ip6InAddrErrors`, `Udp6InErrors` and `Udp6InCsumErrors` remain zero;
   `Ip6InDiscards` remains 23. Conntrack reads succeed and contain no IPv6
   loopback tuple before/after. Missing hook traversal, not conntrack absence
   alone, locates the loss before normal IPv6 conntrack/local-input handling.
5. Member type-4 traffic already starts at **09:33:52.703797Z** (packet **96**),
   before the response at **09:33:52.903737Z**. Its handshake changes from zero
   to **1793525632**; its sampled endpoint remains the IPv4 loopback proxy
   (`addr-060f0ef186d1db5b:3128`) through the passive tail. Node's endpoint
   becomes the member IPv6 address/51820, with handshake **1793525632**.
   Neither peer is observed roaming to an IPv6 `::1` endpoint.

The hard observation is therefore **after member netdev ingress and before
IPv6 prerouting**, with one IPv6 header rejection. The response does not reach
WireGuard input in this observed path. It is not an observed nft policy drop
or a demonstrated WireGuard roam to `::1`.

The code-level interpretation is IPv6's loopback-source validation: the matching
[upstream Linux v7.0 receive code](https://github.com/torvalds/linux/blob/028ef9c96e96197026887c0f092424679298aae8/net/ipv6/ip6_input.c)
rejects loopback source/destination on a non-loopback, non-L3-master interface
before the prerouting hook and increments the header-error counter. Observed
kernel release is **7.0.0-38-generic**. This is a source-supported interpretation
of the measured boundary, **not a captured kernel drop tracepoint or proof that
the distribution's exact binary matches every upstream line**. A separate
reverse-path rule is not evidenced.

On the node, [NetBird's pinned `RedirectAs` headers](https://github.com/netbirdio/netbird/blob/3c00a97878e6af537d28c3bdd4f42cb4286aa883/client/iface/wgproxy/loopback/wrapper.go)
set the injected IPv6 destination to `::1` and source to the peer endpoint.
[Its connection handover](https://github.com/netbirdio/netbird/blob/3c00a97878e6af537d28c3bdd4f42cb4286aa883/client/internal/peer/conn.go)
configures the direct endpoint, then resumes the retained relay using those
headers. [Linux WireGuard's endpoint/source handling](https://github.com/torvalds/linux/blob/028ef9c96e96197026887c0f092424679298aae8/drivers/net/wireguard/socket.c)
retains the received destination as the local IPv6 reply source. This explains
the observed `::1` reply without inferring that the remote endpoint itself
became `::1`. Native debug events corroborate the IPv6 configure/redirect order.
The exact private/source receipts are in [source-review.json](source-review.json).

Thus a malformed late handover response can be discarded while an earlier
relay-established session already completed the request. That is consistent
with members 13 and this success, and does not retroactively prove the cause
of every historical timeout. The IPv4 timeout above must remain separate.

## Repair choices for the owner — none applied

| Option | Evidence and expected scope | Cost and risk / decision needed |
|---|---|---|
| Change the NetBird client pin to a verified fix | The [release list](https://github.com/netbirdio/netbird/releases) available on 2026-11-01 has v0.82.0 as the newest stable entry. [v0.82.0 notes](https://github.com/netbirdio/netbird/releases/tag/v0.82.0) do not identify a repair for this response-source path. No fixed version is established. | Prefer qualifying an upstream source correction, then an exact digest pin. Requires cold first-use/relay→IPv6 handover, reconnect, MTU, enforcement/metering and both architectures. A downgrade is not established as a fix and can regress other changes. Pin remains unchanged. |
| `NB_DISABLE_KERNEL_WG_PROXY=true` (deprecated alias `NB_DISABLE_EBPF_WG_PROXY`) | The [pinned factory](https://github.com/netbirdio/netbird/blob/3c00a97878e6af537d28c3bdd4f42cb4286aa883/client/iface/wgproxy/factory_kernel.go) avoids the shared loopback proxy and uses the UDP proxy. However, [UDP `RedirectAs` raw headers](https://github.com/netbirdio/netbird/blob/3c00a97878e6af537d28c3bdd4f42cb4286aa883/client/iface/wgproxy/udp/rawsocket.go) still use IPv6 destination `::1`. | Small daemon configuration/restart cost, different per-peer socket/resource behavior. **Not an established IPv6 fix**; merely switching factories leaves the risky header shape. Set on daemon, not a one-shot CLI. |
| `NB_WG_KERNEL_DISABLED=true` | [Documented daemon setting](https://docs.netbird.io/client/environment-variables); the pinned [userspace factory](https://github.com/netbirdio/netbird/blob/3c00a97878e6af537d28c3bdd4f42cb4286aa883/client/iface/wgproxy/factory_usp.go) uses a [bind proxy](https://github.com/netbirdio/netbird/blob/3c00a97878e6af537d28c3bdd4f42cb4286aa883/client/iface/wgproxy/bind/proxy.go) instead of this raw kernel injection path. | Plausible way to avoid this specific boundary, untested here. Changes CPU/throughput, interfaces and platform behavior. On nodes it conflicts with the agreed kernel-WireGuard baseline, so it requires an explicit architecture decision and full data-plane qualification. |
| Prefer IPv4 P2P | Avoids the IPv6 `::1` response shape when IPv4 is selected. One fresh IPv4 request passes; another fails with five SYNs despite type-4 data. | Not a general forwarding/readiness repair. Reduces IPv6-only reachability, may increase relay usage or NAT dependence. Current fixture filters **force** a family; they are not a production preference implementation. Owner must define candidate preference and fallback behavior. |
| Node-side source correction | Upstream/node-native integration could inject retained-relay traffic toward the real local IPv6 endpoint, or otherwise avoid retaining loopback as the reply source. The observed outbound response is the boundary to correct. No supported, verified agent-level repair was found. | Requires a NetBird/upstream transport change and careful socket/routing/roaming/checksum review. An exact peer-scoped egress SNAT workaround is only a lab hypothesis: conntrack tuples, changing candidates, namespaces and ownership make it risky. Do not weaken member loopback validation or change the observation-only health probe. No NAT or kernel patch is installed. |

Owner: native NetBird/WireGuard integration, with orchestrator approval of any
pin/daemon/architecture choice. F11-01 remains an evidence hand-off, not repaired.

## F13-01 and D-102

The shared counter reader now keeps exact failed command bytes and parser text
in private files, with public digest/size/line count and a structural subclass.
See the [export contract](../../../scenarios/harness-exports.md#follow-up-24--private-parse-evidence-and-ipv6-input-observation-test-only)
and [dated answers](../../../RESPONSES.md#members-13--2026-11-01-follow-up-24-d-102).
Regression tests use the real enforcement parser with synthetic exit-0 failures,
check exact bytes/text and mode 0600, preserve one read/no retry, and exercise
private export success/failure. Historical F13-01 and F10-02 are still
unattributed; no real failed parser bytes were available from those runs and
no new parser defect is inferred. No `node/enforce` edit.

F12-04 belongs to members' original query/mutation sequence. The existing
`StartNativeDNSCapture`/reply joins are ready for that case; no fresh independent
DNS query was run here. Requests 1 and 5 point to #367's existing routed-hop and
positive-IDNA evidence. Members' request register and acceptance files are intact.
