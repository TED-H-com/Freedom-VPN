# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: 2be2797b-7a28-4e9f-a1fe-5fffa4c1b047; started UTC: 2026-10-31T14:53:48Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-10-31T14:53:51Z

Build: e496d826c04054bd718186de078b8f972353fb94

Image digests:

- fvpn-w4-isolation-lab:lab sha256:e65f01e7e28952df01847101c1279adee4d7cc89771c4299b8daee3f597ef08d

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
