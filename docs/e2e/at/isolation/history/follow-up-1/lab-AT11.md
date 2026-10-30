# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-10-30T00:05:04Z

Build: 4737a6ec3533f230dddb61a5b5e2a18b7dddd580

Image digests:

- fvpn-w4-isolation-lab:lab sha256:3d04195846457be0a66281213c93c446308563faa8c0372a9a08acf039a5ac64

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
