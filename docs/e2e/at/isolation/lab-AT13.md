# lab-AT13

Mechanism: revocation, agent SIGKILL and baseline

Status: pass

Run: 2be2797b-7a28-4e9f-a1fe-5fffa4c1b047; started UTC: 2026-10-31T14:53:56Z; source dirty: false

Spec: §19 AT13, ISO02, ENT02, SEC01

UTC: 2026-10-31T14:54:01Z

Build: e496d826c04054bd718186de078b8f972353fb94

Image digests:

- fvpn-w4-isolation-lab:lab sha256:e65f01e7e28952df01847101c1279adee4d7cc89771c4299b8daee3f597ef08d

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
