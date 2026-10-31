# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: 041e9e48-79a6-45dd-a19d-8b5a63290655; started UTC: 2026-10-31T06:15:34Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-10-31T06:15:37Z

Build: 15ce5f60a1c39d194ef9544a29a02fe40575ab84

Image digests:

- fvpn-w4-isolation-lab:lab sha256:20ab0acdd306acdf25186f86060bbba4a29c8d4c2a79ae8ed7d20dfa6aadce7e

Policy revision: generation 11

Node incarnation: mechanism-lab

Expected: IPv6 mode block drops member IPv6, including ULA, link-local, loopback and IPv4-mapped. Metadata stays unreachable. There is no route mode in v1.

Actual: mode block, ctv6 +3, special-address probes denied, target captures empty. Route mode was not claimed (§8.3).

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- ipv6_mode: block
- ctv6_delta: 3
- route_mode: not implemented in v1
- scope: mechanism lab; product status is AT11.json
