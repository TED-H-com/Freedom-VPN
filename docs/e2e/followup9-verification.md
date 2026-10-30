# Follow-up 9 verification — 2026-10-30

Three complete fresh stacks use `e2e` / `18000`, agent follow-up 9 (#146),
wire-nodes follow-up 15 (#148), and product base `9b17127`. The task branch was
advanced from its clean base before edits; no merge, rebase, push or tag was made.
All three final runs use the same runtime configuration and scenario behavior.
After run 1 compiled, a lint-only edit removed the unused old-session parameter
from the revoke helper; each run records its own exact source commit.
The dated root contains run 3 only; [run 1](runs/2026-10-30/followup9-run1/member-traffic.json) and [run 2](runs/2026-10-30/followup9-run2/member-traffic.json) preserve
the earlier complete JSON/Markdown pairs. A separately labeled complete
authentication diagnostic is retained; incomplete diagnostic attempts are excluded.

| Run | Source | Run ID | Pass / blocked / fail | Go / make exit | Seconds |
|---|---|---|---|---|---:|
| 1 | `943e2a1` | `8fb30ba4-b012-42d9-ba79-4ecb46adff53` | 23 / 0 / 0 | 0 / 0 | 1004.880 |
| 2 | `43176b8` | `65712989-0b05-4117-9a50-689499cd31be` | 23 / 0 / 0 | 0 / 0 | 1004.468 |
| 3 | `43176b8` | `22c14cd9-39fc-444b-a7c5-c04cf3ffbff6` | 23 / 0 / 0 | 0 / 0 | 997.351 |

| Scenario | Run 1 | Run 2 | Run 3 | Owner / limit |
|---|---|---|---|---|
| `admin-login` | pass | pass | pass | — |
| `backup` | pass | pass | pass | — |
| `billing` | pass | pass | pass | — |
| `control-metrics` | pass | pass | pass | — |
| `desired-state` | pass | pass | pass | — |
| `enroll-bootstrap` | pass | pass | pass | — |
| `forwarding` | pass | pass | pass | — |
| `lease-expiry` | pass | pass | pass | — |
| `member-traffic` | pass | pass | pass | — |
| `native-scale-hook` | pass | pass | pass | — |
| `node-restart` | pass | pass | pass | — |
| `portal-billing-return` | pass | pass | pass | — |
| `publish` | pass | pass | pass | — |
| `release-register` | pass | pass | pass | — |
| `revoke` | pass | pass | pass | — |
| `serving-catalogue` | pass | pass | pass | — |
| `stack-health` | pass | pass | pass | — |
| `threat-feed-central` | pass | pass | pass | — |
| `threat-feed-node` | pass | pass | pass | — |
| `threat-feed-tamper` | pass | pass | pass | — |
| `update-bad` | pass | pass | pass | — |
| `update-good` | pass | pass | pass | — |
| `usage` | pass | pass | pass | — |

## Release digests

| Run | Version | Manifest envelope SHA-256 | Signed-object SHA-256 |
|---|---|---|---|
| 1 | 0.1.0 | `eeff658d245cb6b7b95f02dd6af8bf931e24047171cd2a220e0f3ff6c3d2afd3` | `52ed0ea463a33e56fb7387cda830237d97bd85ede636abba17933d8a032d6a7e` |
| 1 | 1.0.1 | `1c891006197d400e8f626bdb5cb27122886ef9adc28c1b5646f834ed347572d9` | `a5884dd44d0f8ebaa28440bcfa8f66c370e70da4f34ae3e75d78805431753eb4` |
| 1 | 1.0.2 | `1b584b5175d7b3fdc72be13344ee20c497223b79c3ffb7ca5cc4ed889eb05090` | `950b86cc6ee226470142a90763b3cf345eff6a2caf9d9249ae5e9845191ee453` |
| 2 | 0.1.0 | `9f1d5ac3de304490963c74adac6d047277996ff31332a496b09020eae775da6b` | `3922c5410f44aa5e92177cfb02da2b195ea9911d798173a989c57e5cb2233c97` |
| 2 | 1.0.1 | `07086f20cd6874c9c3b87467f1501d958a1443c60d4afb235784d659eb301566` | `93f7b06a0a860f6f3526a234efe7beb14c58a10fcd8ef1499ea06afd59bd739b` |
| 2 | 1.0.2 | `60e9e5f0cc0e6a13989f007eed95d864bc14c78e81774f204f4bcee09089b5e0` | `4464c47e368b8f66e4b5f295ecad0ad3acca462c545c7d8dc3a2745dd159aab8` |
| 3 | 0.1.0 | `4dbaf398ad377405a1df72dc4ee1b0801d120b38dd0ccc80295228c34275be37` | `54219dc118d3d00049ee2e05379fb3d6d38ce564d910d90f83554af65444b2d9` |
| 3 | 1.0.1 | `eafaf839875e48066b00a77f07e32483394f5632f04b89aec00f13b8695bafcc` | `dd470aba8820b15ba1e9d98bb788bf1567b79f7d66665d83d99f021c9197b9eb` |
| 3 | 1.0.2 | `b2666ff035e4b615c23947dab0579bec5da292d4a795cfbc77aef8e47f9f15c8` | `6c97d080c6cb5cebb670ccf13d8769d2d2331dd75c4d1a274df88e0def640607` |

## Completed diagnostic run (excluded from the three final runs)

Run `376e5f84-535e-4ddd-902b-985852d4691b`, source `724ab19`, finished in 995.259 s: **22 pass / 0 blocked / 1 fail**, Go 1 / make 2. All forwarding and DNS proofs passed; the final revoke was `403 FORBIDDEN`; the initial admin login was older than the unchanged 900-second recent-auth window. The harness now performs a fresh OIDC admin login before revoke. Its complete original JSON records remain in `runs/2026-10-30/followup9-diagnostic-auth/`; Markdown trailing whitespace alone is normalized. Earlier incomplete bring-up/cadence diagnostic attempts are excluded.

## Observed behavior and limits

- Run 1: admission bytes 0 → 2633; origin `{"probe_id":"b0e12105-d4ee-495a-bbab-70415d83852b","requests":1,"source_address":"192.0.3.20"}`; coherent stored lease/applied/desired generation `true`.
- Run 1: resolver fault injected `true`, last-good unchanged `true`; failed/recovered document evidence is in that run's tamper record.
- Run 1: control-loss member accept rules 30 → 0; signed desired deadline `{"issued_at":"2026-10-30T07:13:34.550334465Z","expires_at":"2026-10-30T07:23:34.550334465Z","config_generation":"18"}`; wait ended `2026-10-30T07:23:39Z`.
- Run 2: admission bytes 0 → 2631; origin `{"probe_id":"d1f188d7-ad56-4752-9483-2350f50eb5f8","requests":1,"source_address":"192.0.3.20"}`; coherent stored lease/applied/desired generation `true`.
- Run 2: resolver fault injected `true`, last-good unchanged `true`; failed/recovered document evidence is in that run's tamper record.
- Run 2: control-loss member accept rules 30 → 0; signed desired deadline `{"issued_at":"2026-10-30T07:31:34.335052682Z","expires_at":"2026-10-30T07:41:34.335052682Z","config_generation":"18"}`; wait ended `2026-10-30T07:41:39Z`.
- Run 3: admission bytes 0 → 2631; origin `{"probe_id":"ef9c25a6-39c0-40c6-a863-08b226040d2f","requests":1,"source_address":"192.0.3.20"}`; coherent stored lease/applied/desired generation `true`.
- Run 3: resolver fault injected `true`, last-good unchanged `true`; failed/recovered document evidence is in that run's tamper record.
- Run 3: control-loss member accept rules 30 → 0; signed desired deadline `{"issued_at":"2026-10-30T07:49:28.885132045Z","expires_at":"2026-10-30T07:59:28.885132045Z","config_generation":"18"}`; wait ended `2026-10-30T07:59:30Z`.

The lease-expiry record measures control-loss withdrawal, bounded by the actual signed desired deadline. It does not claim a 24-hour node lease elapsed, agent-crash independence, clock rollback, or the full AT29 campaign. Physical arm64 execution and 20 × 3 scale remain unmeasured. The renderer's SNAT rule has no counter; independent uplink, conntrack and origin evidence proves translation. See [requests](requests.md).

## Self-check

`e2e-lint`:

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
EXIT=0
```

`golint`:

```text
EXIT=0
```

`vet`:

```text
EXIT=0
```

`unit`:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-origin	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-payment	0.022s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare	0.011s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.006s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.003s
EXIT=0
```

`build`:

```text
building github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-nodeconfig for linux/arm64
building github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-origin for linux/arm64
building github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-payment for linux/arm64
building github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare for linux/arm64
building github.com/TED-H-com/Freedom-VPN/tests/netlab/cmd/fvpn-netlab for linux/arm64
EXIT=0
```

`config`:

```text
FVPN_E2E_STATE=<worktree>/.agent/e2e-config-placeholder docker compose -p fvpn-e2e -f <worktree>/deploy/devstack/compose.yaml config -q
EXIT=0
SHELL_EXIT=0
```


## Teardown and final audit

`make -f mk/e2e.mk e2e-down` exited 0. Prefix inventory after its fallback
cleanup was empty:

```text
containers=0
networks=0
volumes=0
runtime_images=0
```

Each retained set has 23 matching JSON/Markdown record pairs, one run ID and
one source commit. Raw subject fields remain redacted. The §8 hygiene grep is
empty. No area verify/drift target or shipped Python helper is present; Compose
config and shell syntax checks pass. Toolchain: Go 1.27.3, linux/amd64,
`CGO_ENABLED=0`, `GOTOOLCHAIN=local`, with builds for amd64 and arm64. Shared
neutral build-cache images are retained; other prefixes were not touched.
