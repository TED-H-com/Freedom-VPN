# lab-AT10

Mechanism: host and upstream isolation during apply and interface change

Status: pass

Run: 49799a38-1980-4c0f-a527-725646230eee; started UTC: 2026-10-30T19:42:34Z; source dirty: false

Spec: §19 AT10, ISO01, ISO02

UTC: 2026-10-30T19:42:39Z

Build: 4d84f0a651e60ae84f168a8a33248503a50cc94a

Image digests:

- fvpn-w4-isolation-lab:lab sha256:80410e74e89cadda5972661fdd70d2a3a7b7afb79c2f20d7dbbb48c6f429106d

Policy revision: generation 3000-3020

Node incarnation: mechanism-lab

Expected: Protected IPv4 destinations stay denied while the ruleset is regenerated and a lab veth appears. The controlled Internet endpoint stays reachable. The probe holds at least ten attempts per second and records no allow.

Actual: 20 regenerations, interface flap, rate 86.7 attempts/s, protected destinations denied, 203.0.113.10:80 allowed, captures empty. This is the mechanism lab, not the assembled stack.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- attempts: 280
- seconds: 3.231
- rate_per_s: 86.66
- regenerations: 20
- flap: veth fv-px 198.18.0.0/30 inside the lab container
- internet: 203.0.113.10:80 allowed
- scope: mechanism lab; product status is AT10.json
