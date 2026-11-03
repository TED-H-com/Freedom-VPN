# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: e5bdeacd-a2e4-427f-9c6e-594e75c4e701; started UTC: 2026-11-03T20:56:06Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-11-03T20:56:09Z

Build: 9493950a6308c86a729ddc7fff0fe0ff97bb370f

Image digests:

- fvpn-w4-isolation-lab:lab sha256:fc54a3d39b52f06e91e5da325f2c9824e4eab8db52c4c2c64eb2b1d30df456a5

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
