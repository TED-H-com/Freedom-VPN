# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: d69f5652-aed6-4fdd-9e7c-edface751237; started UTC: 2026-11-05T10:12:45Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-11-05T10:12:48Z

Build: 37b3c25f60194e0d2261cf9aca30dfe8fc5d93be

Image digests:

- fvpn-w4-isolation-lab:lab sha256:9b1cb51e2743eab7a4d89e7561680a234a54e54bc8b05e6bd7c4c3195cf5bb0d

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
