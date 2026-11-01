# First-selection forwarding observation — Follow-up 13

- **F11-01:** first selection succeeds with one SYN, node/origin receipt and counted traffic. All three captures are complete (269 / 292 / 77 records, zero drops). Type-4 transport data appears before a loopback-source IPv6 response arrives after the successful request. This differs from devstack 22's five-SYN/no-type-4 reproduction; neither historical cause nor a fix is inferred. The original prehistory and sleep/wake branch remain. [Own capture and comparison](F13-F11-01-forwarding.json).

The capture begins before the original route selection, samples during selection
and before/after the original bounded transfer, then stops with a bounded cleanup
context. It changes no route-selection logic, reconnect, serving state or traffic
precondition. Export startup/build/copy overhead is observable and can affect
first-use timing. Readbacks include routes, links, ICE/relay and WireGuard endpoints.
Public headers contain no payloads or native keys. Historical five-SYN packet
identities cannot be reconstructed from this fresh successful sample.

The initial selection's pending ICE candidate readbacks and loopback WireGuard
endpoints do not establish whether that request used a direct or relayed
transport. Later matrix P2P classifications are separate observations.
