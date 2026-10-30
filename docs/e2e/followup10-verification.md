# Follow-up 10 verification — 2026-10-30

Three complete fresh stacks use `e2e` / `18000` and product base `27d9229` (follow-up 9, PR #153). Each starts with teardown and a fresh prepare/build. Runtime images are pinned to immutable IDs. No product, contract, migration, module or workflow change is included. The root dated records contain run 3 only; `followup10-run1/` and `followup10-run2/` retain the earlier complete sets. Every set has 26 matching JSON/Markdown pairs and one run/source identity.

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
| `admin-login` | pass | pass | pass | [record](runs/2026-10-30/admin-login.md); — |
| `backup` | pass | pass | pass | [record](runs/2026-10-30/backup.md); — |
| `billing` | pass | pass | pass | [record](runs/2026-10-30/billing.md); — |
| `control-metrics` | pass | pass | pass | [record](runs/2026-10-30/control-metrics.md); — |
| `desired-state` | pass | pass | pass | [record](runs/2026-10-30/desired-state.md); — |
| `device-revoke` | blocked | blocked | blocked | [record](runs/2026-10-30/device-revoke.md); w3-devstack-e2e: revoked native route may withdraw before node ingress proof; no product defect is inferred |
| `enroll-bootstrap` | pass | pass | pass | [record](runs/2026-10-30/enroll-bootstrap.md); — |
| `forwarding` | pass | pass | pass | [record](runs/2026-10-30/forwarding.md); — |
| `gateway-privacy` | pass | pass | pass | [record](runs/2026-10-30/gateway-privacy.md); — |
| `lease-expiry` | pass | pass | pass | [record](runs/2026-10-30/lease-expiry.md); — |
| `member-traffic` | pass | pass | pass | [record](runs/2026-10-30/member-traffic.md); — |
| `native-scale-hook` | pass | pass | pass | [record](runs/2026-10-30/native-scale-hook.md); — |
| `node-restart` | pass | pass | pass | [record](runs/2026-10-30/node-restart.md); — |
| `portal-billing-return` | pass | pass | pass | [record](runs/2026-10-30/portal-billing-return.md); — |
| `publish` | pass | pass | pass | [record](runs/2026-10-30/publish.md); — |
| `release-register` | pass | pass | pass | [record](runs/2026-10-30/release-register.md); — |
| `revoke` | pass | pass | pass | [record](runs/2026-10-30/revoke.md); — |
| `second-exit` | pass | pass | pass | [record](runs/2026-10-30/second-exit.md); — |
| `serving-catalogue` | pass | pass | pass | [record](runs/2026-10-30/serving-catalogue.md); — |
| `stack-health` | pass | pass | pass | [record](runs/2026-10-30/stack-health.md); — |
| `threat-feed-central` | pass | pass | pass | [record](runs/2026-10-30/threat-feed-central.md); — |
| `threat-feed-node` | pass | pass | pass | [record](runs/2026-10-30/threat-feed-node.md); — |
| `threat-feed-tamper` | pass | pass | pass | [record](runs/2026-10-30/threat-feed-tamper.md); — |
| `update-bad` | pass | pass | pass | [record](runs/2026-10-30/update-bad.md); — |
| `update-good` | pass | pass | pass | [record](runs/2026-10-30/update-good.md); — |
| `usage` | pass | pass | pass | [record](runs/2026-10-30/usage.md); — |

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
