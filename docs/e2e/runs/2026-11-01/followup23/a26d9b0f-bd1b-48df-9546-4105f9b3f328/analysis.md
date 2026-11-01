# Follow-up 23 native path evidence — 2026-11-01

Run `a26d9b0f-bd1b-48df-9546-4105f9b3f328`, runner source `c26303b57317c893ca169a87a662b0d0bd238daa`:
**14 pass / 0 blocked / 3 fail**, process exit **1**. These are fixture records,
not a complete AT09 or platform qualification. No production MTU/MSS setting changed.

## D-137: a sender-learnable path

The original four mutation rows run first, then the unchanged asymmetric IPv6
limitation and three known-route-MTU controls. Only then is a separate router
added. Node/router ingress links have MTU 1500; router egress/member link has
MTU 1280. Both endpoints' links are read back in every capture. The Docker
bridge sets the same MTU on the member-facing link. Forwarding is enabled only
inside the owned router container, with no ICMP filter or NAT there. Peer host
routes have no configured MTU. No node interface is added under a serving agent.

| Row | Received / required | Duration | ICMP at router / sender | Sender route after |
|---|---:|---:|---|---|
| Routed IPv6 | 64,064 / 64,064 | 1 ms | 1 PTB / same 1 PTB, MTU 1280 | learned MTU 1280 |
| Routed IPv4 | 64,064 / 64,064 | 1 ms | 0 frag-needed / 0 | no learned MTU metric; link remains 1500 |
| Receive-only IPv4 1280 | 10,010 / 64,064 | 12,011 ms | 0 | unchanged limitation |
| Receive-only IPv6 1280 | 10,010 / 64,064 | 12,006 ms | 0 | unchanged limitation |

[IPv6 analysis](router-analysis-ipv6.json) joins router PTB packet **558** to
node packet **547** by exact IP hash. The sender route has no MTU metric before
the transfer and MTU 1280 afterwards. One oversized IPv6 datagram reaches the
router; 84 outgoing IPv6 fragments arrive at the member, with no unmatched
router egress fragment. This is reactive PMTU evidence, separate from the
known-route-MTU controls.

[IPv4 analysis](router-analysis-ipv4.json) records **34** oversized incoming
outer UDP packets, all **DF=false**. Each becomes two fragments at router egress:
**68** fragments, all matched at member ingress by exact IP hash. The sender
receives no frag-needed and learns no reduced PMTU. This is transit fragmentation,
not proof of IPv4 DF/ICMP discovery. Inner TCP byte-range coverage is complete in
both router rows. Virtual capture/GSO/GRO observations are not physical frame counts.

The three known-MTU IPv6 controls complete; [original rows and stream gaps](headers/analysis.json)
retain both receive-only failures. Historical F8-02 remains failed. Under
provisional D-137, these routed rows support the tested sender-learnable scope;
QUIC, relay and physical/member platform coverage remain separate requirements.

## F11-01: pinned proxy → IPv6 P2P transition

[Initial-selection record](initial-selection.json) reproduces five SYNs and
**5,003 ms** timeout, origin requests **0**, generation-5 device counters
**0 → 0** in both directions. The member capture has no WireGuard type-4 message.
All 214 member / 224 node / 51 origin records are complete, with zero drops or
truncation. Later node transport data follows the request deadline and member
capture stop; it cannot prove delivery during that failed request.

[Allowlisted daemon events](native-debug-events.json) record node local relay
proxy creation at **08:18:15.094Z**, selected IPv6 endpoint configuration at
**08:18:15.195Z**, then relay-packet redirection at **08:18:15.296Z**. Capture:

1. Member loopback packet **8**, **08:18:15.092478Z**: initiation, sender index
   **3306059932**.
2. Node loopback packet **164**, **08:18:15.296795Z**: the identical initiation
   message, now source=member selected IPv6 and destination=IPv6 loopback.
3. Node underlay packet **167**, **08:18:15.297068Z**: response to that index,
   source=IPv6 loopback, destination=member selected IPv6.
4. Member underlay packet **178**, **08:18:15.297075Z**: identical response hash;
   no resulting member type-4 transport data before the timeout.

The image still pins NetBird **0.82.0**, digest
`sha256:eca5b274c635bfbb04ec6724a0712d1146190dd450eea199eeb83916fdd94965`.
The corresponding upstream commit is
`3c00a97878e6af537d28c3bdd4f42cb4286aa883`. Its
[peer transition](https://github.com/netbirdio/netbird/blob/3c00a97878e6af537d28c3bdd4f42cb4286aa883/client/internal/peer/conn.go#L472)
configures the direct endpoint, then calls `RedirectAs` on the retained relay
proxy. The
[loopback proxy](https://github.com/netbirdio/netbird/blob/3c00a97878e6af537d28c3bdd4f42cb4286aa883/client/iface/wgproxy/loopback/wrapper.go#L44)
constructs IPv6 packets with remote endpoint as source and `::1` as destination,
and injects them into local WireGuard. The log source lines and packets support
this specific mechanism; the response's selected loopback source is observed,
not inferred from mere Connected status.

**Failing path / owner:** NetBird 0.82.0 Linux kernel-WireGuard loopback proxy's
retained-relay injection during the IPv6 P2P transition; native NetBird/WireGuard
integration and upstream NetBird own investigation. Kernel input rejection and
handshake acceptance are not directly instrumented. No kernel drop reason,
enforcement defect, health repair or product fix is claimed. `node/agent/probe.go`
only consumes status and is not changed. No upstream or product source was changed.

Header/clock derivations use the repository decoders; raw logs and pcaps remain
private under the ignored run directory with SHA/size receipts. Exact pins live
in the follow-up 23 manifest. Earlier follow-up 22 evidence was verified before
the run and remains byte-identical.
