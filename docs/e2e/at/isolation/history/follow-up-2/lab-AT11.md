# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: 422cdd6d-090b-4bb7-b92b-d4883259ac87; started UTC: 2026-10-30T17:28:21Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-10-30T17:28:24Z

Build: 11698fc4d7205a5c902592660ccb346d7f4db750

Image digests:

- fvpn-w4-isolation-lab:lab sha256:fbab877be24698a159f2db45530aa0943ac229100ba8d311ec52f0b9148bad2c

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
