# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: 9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04; started UTC: 2026-11-03T14:05:57Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-11-03T14:06:00Z

Build: c248db966d684d58b4781349f71f8394fcb64d9b

Image digests:

- fvpn-w4-isolation-lab:lab sha256:034bc4f2a43a1c67dc54cd6f98d5c9207d7995ff85c083b1be5f95b7c1b85b32

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
