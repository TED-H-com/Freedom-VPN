# Isolation acceptance follow-up 1

The product gate now fails after writing all AT10–AT13 records when any case is not `pass`. `make -f mk/at-isolation.mk at-isolation` returned **2** on this run: mechanism lab exit 0, product tests exit 1. A recorded `blocked` result cannot make the acceptance target green (§19.2). The regression test exercises blocked and failed child test processes and verifies that all four JSON/Markdown records replace earlier passes.

## AT12 capture diagnosis

The original `dst port 8080` filter captured both directions on `fv-a2`, which also originated the a2-to-a1 test. The reproduced failure captured an outgoing SYN from `100.64.1.2` to `100.64.1.1:8080`; it did not show a packet arriving at a2. See [the original capture](lab-AT12-before-capture.txt).

The corrected run recorded simultaneous `inout`, `out` and `in` captures. The a2 `inout` and `out` packet headers match; a2 ingress is empty. a1, b1, LAN and simulated-Internet ingress are also empty during the deny matrix. Before those probes, separate host-originated UDP controls reached all three peer ingress captures, proving that ingress collection works. Capture startup, successful completion and zero kernel capture drops are required. Per-socket timestamps are omitted from the corrected captures because simultaneous sockets can timestamp the same packet a few microseconds apart.

[Capture evidence](lab-AT12-captures.txt) and the [14-row matrix](lab-AT12-matrix.txt) show nine denies, a UDP DNS reply, a TCP connection to DNS port 53, and allowed Internet TCP connections for all three devices. The host listener remained available. **The original failure was a capture-filter artefact. No target-ingress bypass was observed, so no `node/enforce` change or request is warranted.** These are mechanism results; they do not establish product acceptance or a DNS transaction over TCP.

## Fresh assembled stack

Source build `4737a6ec3533f230dddb61a5b5e2a18b7dddd580` includes main through `350e57c` and wire-nodes follow-up 7 (#62). Current main `408d2da` was subsequently merged; its only added change was roadmap tracking, so the tested product source is unchanged. The stack used fresh secrets/state and a fresh database (`E2E_RESET=1`), with binding `w4-isolation` / `36000`. No projector-owned rows were hand-written (D-050).

At `2026-10-30T00:05:53Z`:

| Observation | Value |
|---|---|
| Node | `01a18a20-6449-7a97-a553-facb665b7600` |
| Incarnation | `01a18a20-6816-7b31-8533-3d15704d446c` |
| Control / NetBird | `online` / `netbird_up` |
| Provisioning / safety / data health | `applying` / `failed` / `unknown` |
| Desired generation / publication intent | `1` / `unpublished` |
| Device bindings / resolver count | `0` / `1` |
| Publication observed | `absent` |
| Forward generation / policy / member accepts | `0` / `drop` / `0` |

Both idempotent publication requests returned `422 NODE_NOT_SERVICEABLE`, before an operation was created. The exact failed checks were `["provisioning_state=applying (want ready)","safety_status=failed"]`. All four product ATs remain **blocked**; none newly passes. The product topology still has one member container and no global member IPv6 address. [AT10.json](AT10.json) through [AT13.json](AT13.json) retain the response and node observations. All four refreshed `lab-AT*.json` mechanism records pass independently.

## Run and cleanup

The lab prefix is now **`fvpn-w4-isolation-lab`**, superseding the older prefix in the original README. Build and test on the unprivileged host first; only the isolated Docker topology is privileged. The task runner places Go temp files and lab scratch under disk-backed `.agent/`, never `/tmp` or `/dev/shm`, and names cleanup containers explicitly.

```sh
make -f mk/at-isolation.mk at-isolation-lint
make -f mk/at-isolation.mk at-isolation
```

The aggregate resets the assembled stack, writes evidence, tears down both environments, and preserves a nonzero result. For an isolated mechanism rerun:

```sh
make -f mk/at-isolation.mk at-isolation-lab-up
mkdir -p .agent/tmp
at_tmp=$(mktemp -d "$PWD/.agent/tmp/isolation.XXXXXX")
export TMPDIR="$at_tmp" GOTMPDIR="$at_tmp" GOTOOLCHAIN=local CGO_ENABLED=0
FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab \
  go test -tags netlab -count=1 -run '^TestAT12_CrossMemberDNS$' ./tests/netlab/at
make -f mk/at-isolation.mk at-isolation-lab-down
rmdir "$at_tmp"
```

The same disk temp setup supports offline gate regression testing with `FVPN_E2E=0 go test -tags e2e ./tests/e2e/at/isolation`.

Root and area lint, the full Go suite, all binary builds for linux/amd64 and linux/arm64, and both tagged acceptance-test cross-compiles passed. No physical arm64 or member-platform run occurred. The task's containers, networks, volumes and image tags were absent after teardown; generated stack state and its local databases were also removed.

The remaining work belongs to the serving-baseline and member-admission follow-ups, followed by another product probe run. Real restart/reboot/update continuous isolation and member-platform IPv6 acceptance remain unproved; see [operator-run.md](operator-run.md).
