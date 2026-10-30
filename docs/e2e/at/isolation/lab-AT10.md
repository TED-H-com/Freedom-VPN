# lab-AT10

Mechanism: host and upstream isolation during apply and interface change

Status: pass

Spec: §19 AT10, ISO01, ISO02

UTC: 2026-10-29T23:40:47Z

Build: c1ac4393f6c206783f06034d73f2915d675bd917

Image digests:

- fvpn-w4-at-isolation:lab sha256:fb7de817d87b65403435999fd8676cf58e6acb3769a2023d6da0b2fdd7120d46

Policy revision: generation 3000-3020

Node incarnation: mechanism-lab

Expected: Protected IPv4 destinations stay denied while the ruleset is regenerated and a lab veth appears. The controlled Internet endpoint stays reachable. The probe holds at least ten attempts per second and records no allow.

Actual: 20 regenerations, interface flap, rate 86.5 attempts/s, protected destinations denied, 203.0.113.10:80 allowed, captures empty. This is the mechanism lab, not the assembled stack.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-at-isolation make -f mk/netlab.mk netlab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-at-isolation FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- attempts: 266
- seconds: 3.076
- rate_per_s: 86.48
- regenerations: 20
- flap: veth fv-px 198.18.0.0/30 inside the lab container
- internet: 203.0.113.10:80 allowed
- scope: mechanism lab; product status is AT10.json
