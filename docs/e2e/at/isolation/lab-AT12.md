# lab-AT12

Mechanism: cross-member, management plane and DNS exceptions

Status: pass

Run: df9dbad7-049d-4eaa-ab98-5490906cc3e9; started UTC: 2026-11-04T14:06:46Z; source dirty: false

Spec: §19 AT12, ISO01, ISO04, DNS03

UTC: 2026-11-04T14:06:51Z

Build: 56d814113913ddc6177d47d4a5160a65c93add25

Image digests:

- fvpn-w4-isolation-lab:lab sha256:9b53681fbed522d7b3a619f5991cfc66abb8fb4d9d82847bdb8fa39c73fc98b5

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
