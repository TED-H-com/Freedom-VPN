# lab-AT10

Mechanism: host and upstream isolation during apply and interface change

Status: pass

Run: 56d20530-8042-4d0c-8156-de5a47ce8b61; started UTC: 2026-11-01T12:02:05Z; source dirty: false

Spec: §19 AT10, ISO01, ISO02

UTC: 2026-11-01T12:02:11Z

Build: 716a16335cb5335473416740c76a75ce95536796

Image digests:

- fvpn-w4-isolation-lab:lab sha256:2e27f7e3829f7866846d056bd934954c25a578b69d17957ca38d795e7ea29c2f

Policy revision: generation 3000-3020

Node incarnation: mechanism-lab

Expected: Protected IPv4 destinations stay denied while the ruleset is regenerated and a lab veth appears. The controlled Internet endpoint stays reachable. The probe holds at least ten attempts per second and records no allow.

Actual: 20 regenerations, interface flap, rate 86.5 attempts/s, protected destinations denied, 203.0.113.10:80 allowed, captures empty. This is the mechanism lab, not the assembled stack.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- tc_readback: [{"interface":"fv-fh","attachment":"qdisc","objects":[{"kind":"fq_codel","handle":"2f:","parent":"1:2f"},{"kind":"fq_codel","handle":"30:","parent":"1:30"},{"kind":"htb","handle":"1:","root":true}]},{"interface":"fv-ih","attachment":"qdisc","objects":[{"kind":"fq_codel","handle":"2f:","parent":"1:2f"},{"kind":"fq_codel","handle":"30:","parent":"1:30"},{"kind":"htb","handle":"1:","root":true}]},{"interface":"fv-fh","attachment":"1:1","objects":[]},{"interface":"fv-fh","attachment":"1:2f","objects":[]},{"interface":"fv-fh","attachment":"1:30","objects":[]},{"interface":"fv-fh","attachment":"1:fffe","objects":[]},{"interface":"fv-fh","attachment":"2f:","objects":[]},{"interface":"fv-fh","attachment":"30:","objects":[]},{"interface":"fv-fh","attachment":"egress","objects":[]},{"interface":"fv-fh","attachment":"ingress","objects":[]},{"interface":"fv-fh","attachment":"root","objects":[]},{"interface":"fv-ih","attachment":"1:1","objects":[]},{"interface":"fv-ih","attachment":"1:2f","objects":[]},{"interface":"fv-ih","attachment":"1:30","objects":[]},{"interface":"fv-ih","attachment":"1:fffe","objects":[]},{"interface":"fv-ih","attachment":"2f:","objects":[]}]
- attempts: 357
- seconds: 4.128
- rate_per_s: 86.48
- regenerations: 20
- flap: veth fv-px 198.18.0.0/30 inside the lab container
- internet: 203.0.113.10:80 allowed
- scope: mechanism lab; product status is AT10.json
