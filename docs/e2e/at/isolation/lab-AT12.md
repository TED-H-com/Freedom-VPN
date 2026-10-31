# lab-AT12

Mechanism: cross-member, management plane and DNS exceptions

Status: pass

Run: c0106e5d-d8e8-400b-abe0-181d6428fc55; started UTC: 2026-10-31T01:57:45Z; source dirty: false

Spec: §19 AT12, ISO01, ISO04, DNS03

UTC: 2026-10-31T01:57:50Z

Build: 7fa0443c532b1c9a395cd93620152fb6902584ae

Image digests:

- fvpn-w4-isolation-lab:lab sha256:7c4d3eabd27e2cb8d996dd644e48526fb3aeace6e2050c03783974cb4abb2475

Policy revision: generation 12

Node incarnation: mechanism-lab

Expected: Only TCP and UDP 53 to the designated resolver succeed. Peer, SSH, agent API, arbitrary overlay and a spoofed source are denied.

Actual: Two members and two devices of member A: peer, management, overlay, rogue and spoofed source denied. TCP and UDP 53 to 198.51.100.53 allowed. Host 127.0.0.1:2222 stayed up. Mechanism lab only.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- capture_direction: a2 inout equals out (a2-to-a1 SYN); a1/a2/b1/lan/inet ingress empty; three peer ingress positive controls observed; see lab-AT12-captures.txt
- resolver: 198.51.100.53 tcp/53 udp/53 allow; tcp/8080 deny
- spoof_source: 100.64.7.7 denied
- members: fv-a1 fv-a2 fv-b1
- scope: mechanism lab; product status is AT12.json
