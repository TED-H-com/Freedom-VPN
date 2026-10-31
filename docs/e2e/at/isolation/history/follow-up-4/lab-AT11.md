# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: fa0074cc-3b1a-403d-b8c2-3a3439fe2b7e; started UTC: 2026-10-31T00:26:10Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-10-31T00:26:12Z

Build: 491a307dd505e33b14dc0e993aa7e86b28e82f68

Image digests:

- fvpn-w4-isolation-lab:lab sha256:bca785405a902525083ea32e0e34f11ea9ce9575484a26ed4dbc9a5f232eb66a

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
