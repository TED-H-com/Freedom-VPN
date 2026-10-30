# Follow-up 10 verification — 2026-10-30

Three complete fresh stacks use `e2e` / `18000` and product base `27d9229` (follow-up 9, PR #153). Each starts with teardown and a fresh prepare/build. Runtime images are pinned to immutable IDs. No product, contract, migration, module or workflow change is included. The original three sets are archived under `followup10-run1/`, `followup10-run2/` and `followup10-run3/`. The merged-tree set described below is archived under `followup10-merged/`; the dated root contains the latest follow-up. Every set has 26 matching JSON/Markdown pairs and one run/source identity.

| Run | Source | Run ID | Pass / blocked / fail | Go / make exit | Go seconds |
|---|---|---|---|---|---:|
| 1 | `2eeda9d` | `cfda038f-dc18-4371-8047-a4d34954066f` | 25 / 1 / 0 | 1 / 2 | 1404.951 |
| 2 | `2eeda9d` | `273496b3-9897-43b0-b289-a571d1d14561` | 25 / 1 / 0 | 1 / 2 | 1407.660 |
| 3 | `2eeda9d` | `14231a83-130d-40fb-a9b4-0d3851520b67` | 25 / 1 / 0 | 1 / 2 | 1404.999 |

The blocked scenario keeps acceptance non-zero: Go exits 1 and make exits 2
in each run. These are complete runs with no failed scenarios, not green
acceptance commands.

## Scenarios

| Scenario | Run 1 | Run 2 | Run 3 | Record / owner for blocked result |
|---|---|---|---|---|
| `admin-login` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/admin-login.md); — |
| `backup` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/backup.md); — |
| `billing` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/billing.md); — |
| `control-metrics` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/control-metrics.md); — |
| `desired-state` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/desired-state.md); — |
| `device-revoke` | blocked | blocked | blocked | [record](runs/2026-10-30/followup10-run3/device-revoke.md); w3-devstack-e2e: revoked native route may withdraw before node ingress proof; no product defect is inferred |
| `enroll-bootstrap` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/enroll-bootstrap.md); — |
| `forwarding` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/forwarding.md); — |
| `gateway-privacy` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/gateway-privacy.md); — |
| `lease-expiry` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/lease-expiry.md); — |
| `member-traffic` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/member-traffic.md); — |
| `native-scale-hook` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/native-scale-hook.md); — |
| `node-restart` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/node-restart.md); — |
| `portal-billing-return` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/portal-billing-return.md); — |
| `publish` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/publish.md); — |
| `release-register` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/release-register.md); — |
| `revoke` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/revoke.md); — |
| `second-exit` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/second-exit.md); — |
| `serving-catalogue` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/serving-catalogue.md); — |
| `stack-health` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/stack-health.md); — |
| `threat-feed-central` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/threat-feed-central.md); — |
| `threat-feed-node` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/threat-feed-node.md); — |
| `threat-feed-tamper` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/threat-feed-tamper.md); — |
| `update-bad` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/update-bad.md); — |
| `update-good` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/update-good.md); — |
| `usage` | pass | pass | pass | [record](runs/2026-10-30/followup10-run3/usage.md); — |

## Revocation and replacement observations

- Run 1: DELETE `202`; packaged executable `/usr/sbin/conntrack`; new generation `true`; same agent PID `true`; sibling counted `true`; replacement counted `true`. Timing: `not observed; route withdrawal stopped traffic before node ingress`.
- Run 2: DELETE `202`; packaged executable `/usr/sbin/conntrack`; new generation `true`; same agent PID `true`; sibling counted `true`; replacement counted `true`. Timing: `not observed; route withdrawal stopped traffic before node ingress`.
- Run 3: DELETE `202`; packaged executable `/usr/sbin/conntrack`; new generation `true`; same agent PID `true`; sibling counted `true`; replacement counted `true`. Timing: `not observed; route withdrawal stopped traffic before node ingress`.

Run 3 additionally records `transition_ctunku_before=0` and
`transition_ctunku_after=2` while requests were in flight across DELETE.
The later captured request has `ctunku_packets_before=2` and
`ctunku_packets_after=2`, with zero matching packets captured. The two
transition drops have no captured source attribution or first-packet
timestamp, so they do not establish the requested revoked-source timing.

The strict node-ingress assertion remains separate from successful revocation. A revoked native client can lose its advertised exit before another packet reaches `fvmb0`. Frozen device counters, zero conntrack entries and no origin receipt prove withdrawal on that path; they do not prove a measured node-side drop. The recorded timing is an observation, never a pass criterion. No route, group, readiness or projector row is repaired.

## Dedicated updater probe observations

- Run 1: probe setup `true`; non-member `true`; traversal `true`; updater result `true`.
  Window: `{"samples":950,"grant_samples":3,"source_elements":1,"destination_elements":1,"max_timeout_seconds":44,"probe_up_bytes":1463,"probe_down_bytes":1463,"member_bytes_before":18353,"member_bytes_after":18353,"elements_after":0}`. Traffic: `{"attempts":1309,"authenticated_udp_replies":19}`.
- Run 2: probe setup `true`; non-member `true`; traversal `true`; updater result `true`.
  Window: `{"samples":888,"grant_samples":3,"source_elements":1,"destination_elements":1,"max_timeout_seconds":44,"probe_up_bytes":1232,"probe_down_bytes":1232,"member_bytes_before":20982,"member_bytes_after":20982,"elements_after":0}`. Traffic: `{"attempts":1308,"authenticated_udp_replies":16}`.
- Run 3: probe setup `true`; non-member `true`; traversal `true`; updater result `true`.
  Window: `{"samples":673,"grant_samples":3,"source_elements":1,"destination_elements":1,"max_timeout_seconds":44,"probe_up_bytes":1078,"probe_down_bytes":1078,"member_bytes_before":15044,"member_bytes_after":15044,"elements_after":0}`. Traffic: `{"attempts":944,"authenticated_udp_replies":14}`.

The probe sends authenticated 49-byte UDP requests to the isolated origin on port 4443 and validates role-separated HMAC replies. It uses its own one-off setup key, group, transport policy and narrow origin route, without a member grant/device binding. The updater and agent retain their existing validators and challenge TTL. Only the product writes the exception and attestation. The observer requires bounded source/destination elements, real origin receipt, nonzero diagnostic up/down counters, unchanged member totals, and empty sets after the answer; health success alone is insufficient.

## Gateway, protected logs and second exit

- Run 1: cohort `{"Users":20,"Peers":60,"IdentityMatches":60,"Authorized":60,"ScannedLogs":60,"ProtectedLogs":60,"SimultaneousAuthorized":60}`; gateway fault HTTP `502`, sentinel/query matches `0` / `0`, budget-error lines `0`; second node distinct `true`, provisioning `"ready"`, safety `"passed"`.
- Run 2: cohort `{"Users":20,"Peers":60,"IdentityMatches":60,"Authorized":60,"ScannedLogs":60,"ProtectedLogs":60,"SimultaneousAuthorized":60}`; gateway fault HTTP `502`, sentinel/query matches `0` / `0`, budget-error lines `0`; second node distinct `true`, provisioning `"ready"`, safety `"passed"`.
- Run 3: cohort `{"Users":20,"Peers":60,"IdentityMatches":60,"Authorized":60,"ScannedLogs":60,"ProtectedLogs":60,"SimultaneousAuthorized":60}`; gateway fault HTTP `502`, sentinel/query matches `0` / `0`, budget-error lines `0`; second node distinct `true`, provisioning `"ready"`, safety `"passed"`.

Nginx is sized for 60 native peers plus lab services: two workers, 4096 connections each. HTTP diagnostics expose only server/method/status; native HTTP errors that could contain raw URIs are suppressed. The exported pre-delete scanner returns aggregate inventory only. Dynamic peers use `.128/25`, separate from static service IPs, so control-loss faults cannot let a peer take control’s address. The second exit has a real independent identity/resolver/namespace; simultaneous two-exit traffic, other boot flavours, physical networks and arm64 execution remain separate operator/harness work.

## Release digests

| Run | Version | Manifest envelope SHA-256 | Signed-object SHA-256 |
|---|---|---|---|
| 1 | 0.1.0 | `764b953d6b15d50d713a26a05f5dc5b9f07a1cd9a55af73ac8683a884d85a153` | `0981e6441566523cde5892381647fc67ed4fd92bd513147127398096384564af` |
| 1 | 1.0.1 | `2f9014626670b4fcff5e0d2beb8427ad25f5a6b095ece7f35c1aa5543ad77076` | `0769237686afd7fb734fafc3d9e75ed95c515c714b379b89abcca73d445538bc` |
| 1 | 1.0.2 | `fdf9014a4b15594244d255a5ddcfef91058f3c0c8894d295487be3f82c4a1d2e` | `d594f6a1645fb728cb8216b02953aef40cc70b67bc99b6e690f12a1ae0797cb9` |
| 2 | 0.1.0 | `e4e1b62fa6f1eb68c7cc1a8466dab40813f046ca2632f48a16092627b5730d12` | `aa1f3912fee872b3c92713fef717b647aa6bb8de212ed5d43860dc7a6894bfc5` |
| 2 | 1.0.1 | `fbe600eb5ac8cf48d0f9ccc1448b82d093a5ffb9daec703ed264e7c8c4b00312` | `7e986c2a684ca44e45710224505a165f4c734a611eebf1fef5e3a9d6b492696b` |
| 2 | 1.0.2 | `3fdd48ff43ebaa0def349ee893ef9f85c064a9711986016cd79c7353463d15e2` | `eb653091c26b78117225bab84245139ebd1cf8b3d5a585345c4277f044de2362` |
| 3 | 0.1.0 | `eb8caa1a84f3711f389e50dc72bee02a8e0783738da57979307775b1bd82f452` | `6868a0769ec7dbafcc85ede5dd6bfb48a3bcf67cd7488b7334fdf5c4abb691cb` |
| 3 | 1.0.1 | `27f3914f756fb470750707f5dccbfba297bbc137d2638f72dbcb95d5b9dcf34e` | `8d39322358ec8d784ad1e9eb08de3c2a1086e6aa4e001668ffe6acb501483256` |
| 3 | 1.0.2 | `54eca061edef567957e525ef4d369c5ea4f6c601a0aebb0b7ddb511b67c3a0c8` | `c0aa153de178cbe61e1f1b4b3b0553cb3d857656f11d893d99ec975f0629ec24` |

Both architectures’ artifact digests are retained in each release-register record. Arm64 bundles are cross-compiled, not executed on this host.

## Diagnostics excluded from the final series

Exploratory diagnostics found a stale admin session after 48 sequential native admissions (`403 FORBIDDEN`), corrected by an optional real-login refresh callback without extending the 900-second security window. They also corrected the probe’s incarnation peer lookup and added its dedicated transport policy. Starting an extra diagnostic peer during control loss exposed static-IP reuse; the separate dynamic pool fixes that lab fault. An incomplete pre-pool attempt was stopped before collecting final evidence. A subsequent diagnostic proved probe traversal but exposed a timeout field labeled milliseconds while nftables reports seconds; the observer and bound check were corrected before the final series. A completed TLS diagnostic then observed bidirectional probe bytes but no completed HTTPS request in the short grant; the final probe uses an authenticated UDP round trip instead. These mutable-tree diagnostic attempts are not counted or relabeled as final runs.

## Self-check

`make -f mk/e2e.mk e2e-lint`:

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...

EXIT=0
```

`make go-lint`:

```text
EXIT=0
```

`go vet -tags e2e ./tests/e2e/...`:

```text
EXIT=0
```

`go test -count=3 ./tests/e2e/... ./tests/e2e/cmd/... ./tests/e2e/internal/...`:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare	0.010s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.006s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.003s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	0.001s

EXIT=0
```

`make go-build`:

```text
building github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-origin for linux/arm64
building github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-payment for linux/arm64
building github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare for linux/arm64
building github.com/TED-H-com/Freedom-VPN/tests/netlab/cmd/fvpn-netlab for linux/arm64

EXIT=0
```

`make -f mk/e2e.mk e2e-config`:

```text
FVPN_E2E_STATE=<worktree>/.agent/e2e-config-placeholder docker compose -p fvpn-e2e -f <worktree>/deploy/devstack/compose.yaml config -q

EXIT=0
```

No shipped Python helper or dependency changed; the workflow is unchanged. `git diff --check` and the §8 hygiene grep are empty. Each run’s `e2e-down` exits 0 and the own-prefix container/network/volume/runtime-image inventories are empty. No push, merge, rebase or tag was made.

## Merged-tree run — send-back f10b

One fresh complete stack on `79259af` includes the orchestrator's merge of
#154–#163. Product code is unchanged. The record's `source_commit` is that
merge commit; the lab-only test/config/observer fixes in this hand-off were
present during the run and are committed together with its evidence. Their
`git diff --binary HEAD -- tests/e2e` SHA-256 at run start was
`60064bf48d3b44c1d696964d518f3ca3538d19038bcc7e713b37ed6ea560ff40`.

| Run ID | Product source | Pass / blocked / fail | Go / make exit | Go seconds | Down exit |
|---|---|---|---|---:|---:|
| `3a129919-af3f-4bc9-b54a-8075eae8af45` | `79259af` | 21 / 5 / 0 | 1 / 2 | 796.728 | 0 |

All 26 archived `followup10-merged/` JSON/Markdown pairs belong to this run. Four statuses changed
from pass to blocked: `member-traffic`, `usage`, `forwarding`, `lease-expiry`.
`device-revoke` remains blocked, now at its positive-flow prerequisite.
The acceptance command is nonzero because blocked scenarios fail its gate.

| Scenario | Original run 3 | Merged tree | Record / blocked owner |
|---|---|---|---|
| `admin-login` | pass | pass | [record](runs/2026-10-30/followup10-merged/admin-login.md); — |
| `backup` | pass | pass | [record](runs/2026-10-30/followup10-merged/backup.md); — |
| `billing` | pass | pass | [record](runs/2026-10-30/followup10-merged/billing.md); — |
| `control-metrics` | pass | pass | [record](runs/2026-10-30/followup10-merged/control-metrics.md); — |
| `desired-state` | pass | pass | [record](runs/2026-10-30/followup10-merged/desired-state.md); — |
| `device-revoke` | blocked | blocked | [record](runs/2026-10-30/followup10-merged/device-revoke.md); w3-devstack-e2e: member route discovery timing; no positive counted flow |
| `enroll-bootstrap` | pass | pass | [record](runs/2026-10-30/followup10-merged/enroll-bootstrap.md); — |
| `forwarding` | pass | blocked | [record](runs/2026-10-30/followup10-merged/forwarding.md); w3-devstack-e2e: member route discovery timing; no positive counted flow |
| `gateway-privacy` | pass | pass | [record](runs/2026-10-30/followup10-merged/gateway-privacy.md); — |
| `lease-expiry` | pass | blocked | [record](runs/2026-10-30/followup10-merged/lease-expiry.md); w3-devstack-e2e: member route discovery timing; no positive counted flow |
| `member-traffic` | pass | blocked | [record](runs/2026-10-30/followup10-merged/member-traffic.md); w3-devstack-e2e: member route discovery timing; no positive counted flow |
| `native-scale-hook` | pass | pass | [record](runs/2026-10-30/followup10-merged/native-scale-hook.md); — |
| `node-restart` | pass | pass | [record](runs/2026-10-30/followup10-merged/node-restart.md); — |
| `portal-billing-return` | pass | pass | [record](runs/2026-10-30/followup10-merged/portal-billing-return.md); — |
| `publish` | pass | pass | [record](runs/2026-10-30/followup10-merged/publish.md); — |
| `release-register` | pass | pass | [record](runs/2026-10-30/followup10-merged/release-register.md); — |
| `revoke` | pass | pass | [record](runs/2026-10-30/followup10-merged/revoke.md); — |
| `second-exit` | pass | pass | [record](runs/2026-10-30/followup10-merged/second-exit.md); — |
| `serving-catalogue` | pass | pass | [record](runs/2026-10-30/followup10-merged/serving-catalogue.md); — |
| `stack-health` | pass | pass | [record](runs/2026-10-30/followup10-merged/stack-health.md); — |
| `threat-feed-central` | pass | pass | [record](runs/2026-10-30/followup10-merged/threat-feed-central.md); — |
| `threat-feed-node` | pass | pass | [record](runs/2026-10-30/followup10-merged/threat-feed-node.md); — |
| `threat-feed-tamper` | pass | pass | [record](runs/2026-10-30/followup10-merged/threat-feed-tamper.md); — |
| `update-bad` | pass | pass | [record](runs/2026-10-30/followup10-merged/update-bad.md); — |
| `update-good` | pass | pass | [record](runs/2026-10-30/followup10-merged/update-good.md); — |
| `usage` | pass | blocked | [record](runs/2026-10-30/followup10-merged/usage.md); w3-devstack-e2e: member route discovery timing; no positive counted flow |

### Member-flow prerequisite and owner

At `2026-10-30T16:26:40Z`, member traffic records
`netbird_networks_before_selection="No networks available."` and
`exit_selection="expected exactly one API-published exit"`. The controlled-origin
route still uses `eth0`. The node's stored applied/desired/lease generations
are all `5`, and its heartbeat says `serving 1`; the earlier admin detail in the
same record still had applied `3` and publication observed `withdrawn`.
This is a sequence of snapshots, not one atomic projection read.

A read-only check at `2026-10-30T16:29:28Z` during this same run returned exit 0:

```text
Available Networks:
  - ID: fvpn-exit-efe234da22858407926c372f
    Network: 0.0.0.0/0
    Status: Not Selected
```

The lab checks route availability only once before selecting it. This run
therefore identifies a lab route-discovery timing gap, owned by
**w3-devstack-e2e**; it does not establish an agent forwarding defect or a counted
member flow. `usage` submits no synthetic sample, `device-revoke` performs no
DELETE, and `lease-expiry` injects no fault without that prerequisite. The
historical revoked-source drop/timing gap remains open separately. No route or
product projection was repaired during this run.

### #154 issuer validation and actual exception

`updater.LoadConfig` still validates `probe_peer`, as do engine construction and
challenge issuance. The corrected unit test supplies a temporary agent config
and literal control/result/resolver endpoints, and selects a non-local source
using the actual interface inventory. The isolated origin is accepted;
documentation/CGNAT destinations, an unavailable agent config, and
control/resolver collisions remain refused. No network traffic or validator
exception is used by that test.

Both boot recovery and the updater-run config explicitly set
`agent_config=/etc/vpn-node/agent-run.json`. In the stack, that file specifies
control `https://127.0.0.1:9444`, resolver admin `http://127.0.0.1:8080` and DNS
`127.0.0.1:53`. The node's IPv4 addresses were `127.0.0.1`, `172.28.30.40`,
`192.0.3.20` and `100.121.145.172`. The real synthetic peer source was
`100.121.131.241/32`; its only destination was TEST ONLY isolated origin
`192.0.3.10/32`, UDP 4443. The result endpoint remained
`https://agent.example.invalid:9443/api/v1/agent/update-results`.

The [update-good record](runs/2026-10-30/followup10-merged/update-good.md) reports
`updater_log_readable=true`, `probe_omission_count=0`,
`invalid_probe_peer=false`, `probe_traversal_verified=true`,
`running_candidate_executable=true`, `locally_succeeded=true`, updater exit `0`.
This proves issuance and observed traffic, not merely successful config loading.

Probe window:

```json
{"samples":843,"grant_samples":3,"source_elements":1,"destination_elements":1,"max_timeout_seconds":44,"probe_up_bytes":1848,"probe_down_bytes":1848,"member_bytes_before":0,"member_bytes_after":0,"elements_after":0}
```

Authenticated peer traffic:

```json
{"attempts":1311,"authenticated_udp_replies":24}
```

The origin received 24 requests from the node's SNAT address `192.0.3.20`.
The source/destination sets were present in three samples, with one element each
and at most 44 seconds remaining; both counters reached 1,848 bytes and the sets
were empty after the answer. Member bytes were `0 → 0`; this confirms no member
counter change during the exception, but does not replace the missing positive
member-flow proof. No challenge nonce, raw subject, key or token is copied.
[update-bad](runs/2026-10-30/followup10-merged/update-bad.md) also passes: updater exit `1`,
`HASH_MISMATCH=true`, tampered real bytes rejected.

### Merged-run release digests

| Version | Manifest envelope SHA-256 | Signed-object SHA-256 |
|---|---|---|
| 0.1.0 | `3723f5d100f170f78da485a43562284e3b923ab856df0edee17703bcd5e82464` | `0a044e8632b786a5d2ba664e2c78772c2c777c2a33f49628d5515cce32813922` |
| 1.0.1 | `3c345b7501565b4ed9596ef44bb7b1da2d7abf680edb55cc045234767d432b76` | `d3891203abfbde92fa12c32a0547f66ec975238cc502fd9b6457492a2ae7ba1e` |
| 1.0.2 | `6f970df2c17abc54bd73ee8e58efe33dcf8cc09f640c085c3403c0bf3b9b74a6` | `b59adbf4b6d5a819314107b8aeddab0738885d63b3860f94f2021a136abf1c03` |

The release-register record retains all six artifact digests. Both architectures
were cross-compiled; arm64 execution is not claimed.

### Merged-tree self-check and teardown

| Command | Exit |
|---|---:|
| `go vet ./tests/e2e/...` | 0 |
| `go test ./tests/e2e/cmd/...` | 0 |
| `make go-lint` | 0 |
| `make -f mk/e2e.mk e2e-lint` | 0 |
| `make go-build` (linux/amd64 and linux/arm64) | 0 |
| `make -f mk/e2e.mk e2e-test` (Go 1; 21 pass / 5 blocked / 0 fail) | 2 |
| `make -f mk/e2e.mk e2e-down` | 0 |

Acceptance tail:

```text
acceptance summary: pass=21 blocked=5 fail=0 run=3a129919-af3f-4bc9-b54a-8075eae8af45 source=79259af1facc1faa798fbb83cc20aa4e74539d1f
--- FAIL: TestE2E_Acceptance (796.71s)
FAIL github.com/TED-H-com/Freedom-VPN/tests/e2e 796.728s
MAKE_EXIT=2
```

An earlier startup attempt could not bind port 18444 and never entered Go
acceptance. It is excluded from the run table; make exited 2 and teardown 0.
The unrelated listener was left untouched. The complete run began only after
the port was free. Final own-prefix container/network/volume/runtime-image
inventories are empty; shared services were left alone. No additional merge,
push, rebase, tag or PR was made for this send-back.
