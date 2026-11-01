# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: 1c25ee44-1872-4cce-a9f5-d52491d87095; started UTC: 2026-11-01T09:39:24Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-11-01T09:39:27Z

Build: 99e58bff0cf7276e36cb04f35605835208c8638e

Image digests:

- fvpn-w4-isolation-lab:lab sha256:3d3bfa3b2e36ea184825464c02464b8fe4daae00ff6306c78705b11b4406e6ed

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
