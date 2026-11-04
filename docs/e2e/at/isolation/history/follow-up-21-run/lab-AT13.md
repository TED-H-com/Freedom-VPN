# lab-AT13

Mechanism: revocation, agent SIGKILL and baseline

Status: pass

Run: 7599576e-339c-49c9-8bf5-0f710818cbda; started UTC: 2026-11-04T02:52:10Z; source dirty: false

Spec: §19 AT13, ISO02, ENT02, SEC01

UTC: 2026-11-04T02:52:17Z

Build: fe2c68ecbf1ef1481d744c998be95710d28be954

Image digests:

- fvpn-w4-isolation-lab:lab sha256:df384dcae1b09e3eb7af86ee262965e02ccd35cc19be6f0d555814df2763d72f

Policy revision: generation 4000-4100

Node incarnation: mechanism-lab

Expected: An established flow dies on revoke, conntrack for that source is removed, SIGKILL leaves the ruleset, and baseline keeps member forwarding off.

Actual: Revoke stopped the live flow and cleared its conntrack. SIGKILL left generation 4100. Baseline returned generation 0 and denied both members while 127.0.0.1:2222 stayed up. NetBird restart, node reboot and the updater switch are not in this container.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- conntrack: 100.64.1.1 removed after revoke
- sigkill: generation 4100 remained
- baseline: generation 0, member forwarding denied
- not_exercised: NetBird restart, node reboot, updater switch
- scope: mechanism lab; product status is AT13.json
