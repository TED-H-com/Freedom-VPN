# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: 49799a38-1980-4c0f-a527-725646230eee; started UTC: 2026-10-30T19:42:39Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-10-30T19:42:42Z

Build: 4d84f0a651e60ae84f168a8a33248503a50cc94a

Image digests:

- fvpn-w4-isolation-lab:lab sha256:80410e74e89cadda5972661fdd70d2a3a7b7afb79c2f20d7dbbb48c6f429106d

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
