# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: 5e9e4041-d942-4562-9e99-489bf735636a; started UTC: 2026-10-31T18:42:07Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-10-31T18:42:10Z

Build: d9b770e06a3510880236eed57fe229465947951e

Image digests:

- fvpn-w4-isolation-lab:lab sha256:2c2b1647185ba2d99aa6fbd5a12f89939621fba5a3e11cf1709a8bb63223b7c8

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
