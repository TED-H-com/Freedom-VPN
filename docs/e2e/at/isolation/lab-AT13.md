# lab-AT13

Mechanism: revocation, agent SIGKILL and baseline

Status: pass

Spec: §19 AT13, ISO02, ENT02, SEC01

UTC: 2026-10-29T23:40:58Z

Build: c1ac4393f6c206783f06034d73f2915d675bd917

Image digests:

- fvpn-w4-at-isolation:lab sha256:fb7de817d87b65403435999fd8676cf58e6acb3769a2023d6da0b2fdd7120d46

Policy revision: generation 4000-4100

Node incarnation: mechanism-lab

Expected: An established flow dies on revoke, conntrack for that source is removed, SIGKILL leaves the ruleset, and baseline keeps member forwarding off.

Actual: Revoke stopped the live flow and cleared its conntrack. SIGKILL left generation 4100. Baseline returned generation 0 and denied both members while 127.0.0.1:2222 stayed up. NetBird restart, node reboot and the updater switch are not in this container.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-at-isolation make -f mk/netlab.mk netlab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-at-isolation FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- conntrack: 100.64.1.1 removed after revoke
- sigkill: generation 4100 remained
- baseline: generation 0, member forwarding denied
- not_exercised: NetBird restart, node reboot, updater switch
- scope: mechanism lab; product status is AT13.json

