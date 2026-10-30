# Follow-up 11 verification — 2026-10-30

Source `1b2f358` on product base `5555772` includes control #174. The lab uses
only `e2e` / `18000`; no product, contract, migration or module change is made.
The prior merged-tree run is preserved under `runs/2026-10-30/followup10-merged/`.

## Complete fresh-stack run

| Run ID | Source | Pass / blocked / fail | Go / make exit | Go seconds | Down exit |
|---|---|---|---|---:|---:|
| `76f0725c-2264-4021-a553-60f6c921e32b` | `1b2f358` | 26 / 1 / 0 | 1 / 2 | 1412.324 | 0 |

All 27 dated root JSON/Markdown pairs belong to this one complete run. The
blocked scenario intentionally keeps acceptance nonzero. The previous run's
route-discovery timing gap is resolved; no failed scenario is hidden as blocked.

| Scenario | Status | Record / blocked owner |
|---|---|---|
| `admin-login` | pass | [record](runs/2026-10-30/admin-login.md); — |
| `backup` | pass | [record](runs/2026-10-30/backup.md); — |
| `billing` | pass | [record](runs/2026-10-30/billing.md); — |
| `control-metrics` | pass | [record](runs/2026-10-30/control-metrics.md); — |
| `desired-state` | pass | [record](runs/2026-10-30/desired-state.md); — |
| `device-revoke` | blocked | [record](runs/2026-10-30/device-revoke.md); w3-devstack-e2e with w1-netlab: independent post-withdrawal ingress fixture |
| `enroll-bootstrap` | pass | [record](runs/2026-10-30/enroll-bootstrap.md); — |
| `forwarding` | pass | [record](runs/2026-10-30/forwarding.md); — |
| `gateway-principals` | pass | [record](runs/2026-10-30/gateway-principals.md); — |
| `gateway-privacy` | pass | [record](runs/2026-10-30/gateway-privacy.md); — |
| `lease-expiry` | pass | [record](runs/2026-10-30/lease-expiry.md); — |
| `member-traffic` | pass | [record](runs/2026-10-30/member-traffic.md); — |
| `native-scale-hook` | pass | [record](runs/2026-10-30/native-scale-hook.md); — |
| `node-restart` | pass | [record](runs/2026-10-30/node-restart.md); — |
| `portal-billing-return` | pass | [record](runs/2026-10-30/portal-billing-return.md); — |
| `publish` | pass | [record](runs/2026-10-30/publish.md); — |
| `release-register` | pass | [record](runs/2026-10-30/release-register.md); — |
| `revoke` | pass | [record](runs/2026-10-30/revoke.md); — |
| `second-exit` | pass | [record](runs/2026-10-30/second-exit.md); — |
| `serving-catalogue` | pass | [record](runs/2026-10-30/serving-catalogue.md); — |
| `stack-health` | pass | [record](runs/2026-10-30/stack-health.md); — |
| `threat-feed-central` | pass | [record](runs/2026-10-30/threat-feed-central.md); — |
| `threat-feed-node` | pass | [record](runs/2026-10-30/threat-feed-node.md); — |
| `threat-feed-tamper` | pass | [record](runs/2026-10-30/threat-feed-tamper.md); — |
| `update-bad` | pass | [record](runs/2026-10-30/update-bad.md); — |
| `update-good` | pass | [record](runs/2026-10-30/update-good.md); — |
| `usage` | pass | [record](runs/2026-10-30/usage.md); — |

## Gateway and limiter evidence

The rendered control configuration trusts exactly `172.28.30.21/32`; nginx
appends `X-Forwarded-For`. Two real member sessions send requests from
`172.28.30.128` and `.129`. The product's browser rate is unchanged at 10/s,
burst 30, with ingress 20/s, burst 60. No limiter table is read.

| Request series | HTTP 200 | HTTP 429 | Address refusals | Subject refusals |
|---|---:|---:|---:|---:|
| A through gateway, 120 requests | 31 | 89 | 58 | 31 |
| A through gateway, rotating forged XFF | 1 | 119 | 117 | 2 |
| B through gateway | 1 | 0 | 0 | 0 |
| A directly to control, rotating forged XFF | 2 | 118 | 117 | 1 |
| B directly to control, same forged value | 1 | 0 | 0 | 0 |

Every 429 carries `RATE_LIMITED` and Retry-After. Metrics move from address 0 /
subject 0 to 175 / 33 after gateway bursts, then 292 / 34 after the direct burst;
the direct address increase exactly matches its 117 address refusals. B still
gets 200. The varying untrusted header cannot mint fresh principals. These are
pre-authentication address and authenticated browser-subject checks, not an
inference from an API/device row. The agent mTLS path stays direct and ignores
forwarded headers. Nginx worker connections remain a separate limit.

[Full gateway record](runs/2026-10-30/gateway-principals.md).

## Finite grace and lower port slots

The CLI smoke command (no containers or live configuration changed):

```text
fvpn-e2e-prepare -state <disk-temp>/state -prefix e2e -port-base 18000 -grace-seconds 90
exit=0
grace_seconds=90
grace_reason=TEST ONLY finite payment grace
members.json mode=0600
trusted_proxies=["172.28.30.21/32"]
```

Scratch state is deleted. Unit tests exercise default 0 and finite 90 through
the product's `LoadMembers`, and reject negative/over-bound values. The complete
stack uses the unchanged default 0. Both control and worker load members
configuration when their domains are built at startup; both must start/restart
after an intentional configuration change. This proves the prepare hook, not a
live paid-to-grace expiry campaign. [RUN.md](RUN.md#finite-payment-grace-secops-04--at29)
lists the provider truth, positive flow, projected deadlines and counter evidence
the secops owner must record. `max_offline_lease_seconds` remains w3-wire-nodes'
D-065 API scope.

`TestENR01_LowerSlotsPreserveExistingBindings` covers all ten slots, checks every
port and subnet for collisions, keeps the original eight mappings, and rejects
66000. New `w4-enroll / 12000` uses `172.28.38.0/24` and `192.0.11.0/24`;
reserved 6000 uses `172.28.39.0/24` and `192.0.12.0/24`. Neither origin is in a
documentation prefix. No stack was started on either new base. The complete
[binding table](RUN.md#assigned-port-slots) includes offsets and owners.

## Counted traffic, stale-client case and resource observations

Bounded native route discovery waited 7.400 seconds. The member's controlled
origin request then increased admission bytes `0 → 2632`; member traffic,
forwarding and real usage projection pass.

The second native client first carried a counted flow normally, then carried
another counted flow **after** `PauseNativeControl` froze only its NetBird
userspace service. DELETE returned 202; generation 8 projected ready/passed with
the exit agent PID unchanged. The stale client retained
`192.0.3.10 dev fvnb0 table netbird`, but the node capture reported:

```text
0 packets captured
0 packets received by filter
0 packets dropped by kernel
```

`ctunku` stayed `0 → 0`; revoked admission bytes froze `5262 → 5262`, origin
receipt was zero and conntrack had zero entries. The sibling and replacement
kept counted traffic. Member accept rules were 6 after DELETE and 12 after
replacement. The native service resumed successfully. DELETE-to-first-node-drop
is **not observed**, not zero and not a passed timing assertion.

A read-only check at `2026-10-30T17:34:10Z`, after the client resumed but during
the same run, found `usesKernelInterface=true` and no node NetBird peer for the
revoked address. This is consistent with upstream transport withdrawal; it is
not a capture of the exact peer-removal time. The evidence does not establish
that the node's forward chain received and dropped the attempted request.

The case is **blocked**, owner **w3-devstack-e2e with w1-netlab**: an independent
controlled tunnel-ingress fixture is still needed after remote peer withdrawal,
with old-device source attribution and ingress/drop counter observation. The
current hook preserves the client's route but cannot preserve the remote
transport. No revoked peer, group, binding or product enforcement is restored.
[Device record](runs/2026-10-30/device-revoke.md) and
[scenario requirements](scenarios/device-revoke.md) retain the exact scope.

Resource samples use read-only Docker exec into the node: cgroup v2 CPU/memory
and namespace `/proc` FD/process/conntrack counts, every five seconds. Full
series, UTC timestamps, configured cadence and read-error counts live beside
the scenario inventory. CPU is percent of one core; memory includes page cache;
observer overhead is included. Separate resolver/control containers are not
part of the node-container totals. These are sampled workload values, not a
production capacity threshold or complete mixed-traffic AT31 certification.

| Workload | Samples | Read errors | Max sampled CPU (% one core) | Max memory bytes | Max open FDs | Max conntrack entries |
|---|---:|---:|---:|---:|---:|---:|
| [member-traffic](runs/2026-10-30/member-traffic.md) | 9 | 0 | 126.494 | 62767104 | 102 | 134 |
| [native-scale-hook](runs/2026-10-30/native-scale-hook.md) | 61 | 0 | 150.407 | 91271168 | 134 | 79 |

The native cohort contains 20 distinct users × 3 peers: 60 identity matches,
authorized devices, protected-log scans and simultaneous authorized peers.
Gateway budget errors and replayed query/sentinel matches are all zero. Helpers
are exported through `tests/e2e/internal/labstack`; see [RUN.md](RUN.md).

## Real-release digests

| Version | Manifest envelope SHA-256 | Signed-object SHA-256 |
|---|---|---|
| 0.1.0 | `8d693c22d33f91501565e671a3bb28f8ade10bf90587af27dabdf0e566ba5dbc` | `d9297ee51b1afdcf575a13dc23458686664d567318ea21b9bf063e8443cbb66f` |
| 1.0.1 | `49c036aed394f409f36b6e77da7b5fc53a2be10b6f976b65b1ca001e1d613ce9` | `050bda6dc4d119915eb321017d2844adc3fb4b9749ddbf255dcd0fcc16258f30` |
| 1.0.2 | `d59caaf7e96c0ec58790567c94e8531c5d9636c353b8ca593ddb2250f6103e0b` | `59223bdb8c2ae4b9d722fb991af26a7ab50e9bb955506a64a3968be7beee7c26` |

All six architecture artifact digests are in the release-register record.
Both architectures build; only amd64 executes here. The update-good record
retains zero probe omissions, 29 authenticated replies, 2233 bytes in each
probe counter, unchanged nonzero member counters, and exception removal.

## Self-check and isolation

| Command | Exit |
|---|---:|
| `go test ./tests/e2e/cmd/fvpn-e2e-prepare` | 0 |
| `go test -tags e2e ./tests/e2e/...` (offline; stack flag unset) | 0 |
| `make -f mk/e2e.mk e2e-lint` | 0 |
| `make go-lint` | 0 |
| `make go-build` (linux/amd64 and linux/arm64) | 0 |
| finite-grace CLI smoke, scratch deleted | 0 |
| `make -f mk/e2e.mk e2e-test` (Go 1; one blocked scenario) | 2 |
| `make -f mk/e2e.mk e2e-down` | 0 |

Acceptance tail:

```text
acceptance summary: pass=26 blocked=1 fail=0 run=76f0725c-2264-4021-a553-60f6c921e32b source=1b2f358880d38ae643865edc2bb6f19756748456
--- FAIL: TestE2E_Acceptance (1412.31s)
FAIL github.com/TED-H-com/Freedom-VPN/tests/e2e 1412.324s
MAKE_EXIT=2
```

The expiry check withdrew member accept rules after the signed policy expired;
heartbeat was `serving 0`, `reason policy_expired`, generation 96. Control then
restarted, operator revocation passed, and the second real exit enrolled ready.
This bounded control-loss case is separate from the finite payment-grace option.

All work uses `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, and disk-backed `.agent/tmp`.
Teardown leaves no own-prefix containers, networks, volumes or runtime image
tags. Shared build-cache images and all other tasks' services are left alone.
Whitespace and §8 hygiene checks are empty. No push, PR, merge or rebase.
