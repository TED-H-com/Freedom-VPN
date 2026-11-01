# lab-AT10

Mechanism: host and upstream isolation during apply and interface change

Status: pass

Run: 38074149-1137-4782-80fd-8669c36c6fcb; started UTC: 2026-11-01T11:05:46Z; source dirty: false

Spec: §19 AT10, ISO01, ISO02

UTC: 2026-11-01T11:05:52Z

Build: c521cb0018844eac1de24ee243c858518bb1a5e3

Image digests:

- fvpn-w4-isolation-lab:lab sha256:988bff2032573870968c8eeca7f816edda35c15b5643f5c1616fe70049694f47

Policy revision: generation 3000-3020

Node incarnation: mechanism-lab

Expected: Protected IPv4 destinations stay denied while the ruleset is regenerated and a lab veth appears. The controlled Internet endpoint stays reachable. The probe holds at least ten attempts per second and records no allow.

Actual: 20 regenerations, interface flap, rate 86.5 attempts/s, protected destinations denied, 203.0.113.10:80 allowed, captures empty. This is the mechanism lab, not the assembled stack.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- tc_readback: [{"interface":"fv-fh","attachment":"qdisc","objects":[{"kind":"fq_codel","handle":"2f:","parent":"1:2f"},{"kind":"fq_codel","handle":"30:","parent":"1:30"},{"kind":"htb","handle":"1:","root":true}]},{"interface":"fv-ih","attachment":"qdisc","objects":[{"kind":"fq_codel","handle":"2f:","parent":"1:2f"},{"kind":"fq_codel","handle":"30:","parent":"1:30"},{"kind":"htb","handle":"1:","root":true}]},{"interface":"fv-fh","attachment":"1:1","objects":[]},{"interface":"fv-fh","attachment":"1:2f","objects":[]},{"interface":"fv-fh","attachment":"1:30","objects":[]},{"interface":"fv-fh","attachment":"1:fffe","objects":[]},{"interface":"fv-fh","attachment":"2f:","objects":[]},{"interface":"fv-fh","attachment":"30:","objects":[]},{"interface":"fv-fh","attachment":"egress","objects":[]},{"interface":"fv-fh","attachment":"ingress","objects":[]},{"interface":"fv-fh","attachment":"root","objects":[]},{"interface":"fv-ih","attachment":"1:1","objects":[]},{"interface":"fv-ih","attachment":"1:2f","objects":[]},{"interface":"fv-ih","attachment":"1:30","objects":[]},{"interface":"fv-ih","attachment":"1:fffe","objects":[]},{"interface":"fv-ih","attachment":"2f:","objects":[]}]
- attempts: 364
- seconds: 4.207
- rate_per_s: 86.52
- regenerations: 20
- flap: veth fv-px 198.18.0.0/30 inside the lab container
- internet: 203.0.113.10:80 allowed
- scope: mechanism lab; product status is AT10.json
