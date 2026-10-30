# Follow-up 8 verification — 2026-10-30

Three complete fresh-stack runs use prefix `e2e`, port base `18000`, and
committed tree-built binaries on product base `0a01a88` (#130, including #127). The final dated set contains all 23 scenarios
from run 3 only. Earlier complete sets remain in ignored task state; their
manifest digests are retained here. Run 3 changes only the recorded forwarding
field names and documentation; runtime lab behavior is unchanged from runs 1–2.
A blocked scenario keeps Go test / make
nonzero; these are not green full acceptance runs.

| Run | Source | Run ID | Pass / blocked / fail | Seconds | Go / make exit |
|---|---|---|---|---:|---|
| 1 | `abbd9c6` | `70b9ed41-4d42-4c28-8ba5-259f43cdc281` | 17 / 6 / 0 | 524.753 | 1 / 2 |
| 2 | `abbd9c6` | `7ffab11c-f163-42c6-8c31-7183d0ee8863` | 17 / 6 / 0 | 538.313 | 1 / 2 |
| 3 | `05d70d6` | `742a2b14-355f-462e-be89-52ffb2c1993b` | 17 / 6 / 0 | 544.951 | 1 / 2 |

## Requested observables

| Item | Observable | Result / limit |
|---|---|---|
| 1. Restart | Token absent before/after; status `enrolled`; nft readable `true`; new accepted observation with same incarnation `true`; unchanged enrollment rows `true` | `pass`, 6349 ms including Docker stop |
| 2. D-061 resolver | Node detail 200, provisioning `ready`; signed document matches node-mTLS body `true` | `pass`; credentials uid 0 / 0600, synthetic block and disabled query logging required |
| 3. Landing | Authenticated `GET /` 200; built Portal billing return also renders | Walker rejects 302/401/404/500 and callback errors; unit tests pass |
| 4. Scenario split | `threat-feed-node` = `pass`; `threat-feed-tamper` = `blocked` | Negative case records fault-injected=false; no node corruption hook exists yet; owner w3-devstack-e2e |
| 5. Payment hooks | `customer.subscription.created`, `invoice.payment_failed` | Production HMAC verification and event parser link both to a real checkout in unit tests; changed signed bytes are rejected |
| 6. Runtime DNS record | Exact `agent-run.json` `dns` object below | Matches agent follow-up 6 Requests; enrollment has no DNS credentials/config |

```json
{
  "resolver_credentials_file": "/etc/vpn-node/resolver-credentials.json",
  "resolvers": [
    {
      "admin_url": "http://127.0.0.1:8080",
      "dns_addr": "127.0.0.1:53",
      "id": "lab"
    }
  ]
}
```

The DNS configuration, root-owned credential handling and loopback applier
were already present in the recreated branch from follow-up 7. Fresh runtime
proof passes the separate delivery gates in all three complete runs;
no empty policy, fabricated group, signed-state substitute or projection seed
is used. The password is never copied into a run record.

The diagnostic stack exposed a second restart issue: AdGuard stayed in the
old node network namespace and its loopback admin API became unreachable.
`node-netns` now holds the isolated namespace for both containers; it runs only
`sleep infinity`. Restart preserves the resolver listeners, while the watchdog
and agent resume with the existing durable identity and a new boot ID. Publication
polls the latest accepted baseline rather than any historical successful report.

## Product gates and final scenarios

Before the last run, origin/main was fetched and checked:

```text
origin/main=dd09a79aba5d2cc273788955fdb38e4c93ea5c28
check: git log --oneline origin/main | grep 'w2-agent-core follow-up 9'
No match; follow-up 9 absent (grep exit 1). Run 3 retains the existing product base.
```

Publication response: `202`. Operation: `running`.

```json
{
  "step_agent_policy": "succeeded",
  "step_data_path_probe": "failed NODE_NOT_SERVICEABLE member_forwarding_unconfirmed",
  "step_netbird_route_readback": "succeeded"
}
```

The publication operation belongs to w3-wire-nodes. Its
`member_forwarding_unconfirmed` prerequisite depends on w2-agent-core's serving
lease; fresh DNS setup is already proven separately.
The member's exact stored lease and desired/applied generations are:

```json
{
  "desired_generation": "5",
  "applied_generation": "5",
  "latest_lease": {
    "issued_at": "2026-10-30T05:48:51.111308+00:00",
    "expires_at": "2026-10-31T05:48:51.111308+00:00",
    "generation": "1",
    "member_count": 0
  },
  "lease_count": 2
}
```

Accepted observation: `member_forwarding_enabled=false`,
`forwarding_block_reasons=["lease_expired"]`.
Gate (a) is **not closed**: latest lease generation equals applied generation =
`false`; counted native flow = `false`.
Gate (b)'s positive D-056 delivery result is `pass`.
The recorded heartbeat is:

```text
v1
boot_id c14a667b-d336-4bc4-8119-1c74b6ee174b
mono_ns 694751037991892
wall_unix_ns 1793339711841170746
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 5
reason lease_expired
```

Neither a route nor an authorized device projection proves forwarding. See the
member record for heartbeat, checks and counter values. The tamper case remains
an explicitly separate lab gap: no response bytes were corrupted, so no failed
node pin or last-good retention is claimed from central feed rejection.

| Scenario | Run 1 | Run 2 | Run 3 | Final blocker owner / gate |
|---|---|---|---|---|
| `admin-login` | pass | pass | pass | — |
| `backup` | pass | pass | pass | — |
| `billing` | pass | pass | pass | — |
| `control-metrics` | pass | pass | pass | — |
| `desired-state` | pass | pass | pass | — |
| `enroll-bootstrap` | pass | pass | pass | — |
| `forwarding` | blocked | blocked | blocked | w2-agent-core: current member lease and applied serving policy |
| `lease-expiry` | blocked | blocked | blocked | w2-agent-core: AT29 needs active forwarding before control loss |
| `member-traffic` | blocked | blocked | blocked | w2-agent-core: lease refresh after publication/member changes |
| `native-scale-hook` | pass | pass | pass | — |
| `node-restart` | pass | pass | pass | — |
| `portal-billing-return` | pass | pass | pass | — |
| `publish` | blocked | blocked | blocked | w2-agent-core serving/lease prerequisite; w3-wire-nodes publication operation |
| `release-register` | pass | pass | pass | — |
| `revoke` | pass | pass | pass | — |
| `serving-catalogue` | pass | pass | pass | — |
| `stack-health` | pass | pass | pass | — |
| `threat-feed-central` | pass | pass | pass | — |
| `threat-feed-node` | pass | pass | pass | — |
| `threat-feed-tamper` | blocked | blocked | blocked | w3-devstack-e2e: node mTLS response corruption hook required; central wrong-pin evidence does not test node tampering or last-good retention |
| `update-bad` | pass | pass | pass | — |
| `update-good` | pass | pass | pass | — |
| `usage` | blocked | blocked | blocked | w2-agent-core: positive counted-flow prerequisite; then w3-wire-usage sampler acceptance |

## Release digests

Both architectures contain the real source-tree agent/updater/watchdog. The
NetBird component remains exactly 0.82.0. Only amd64 executes on this host.
The final release-register record also retains both architecture artifact hashes.

| Run | Version | Manifest envelope SHA-256 | Signed object SHA-256 |
|---|---|---|---|
| 1 | 0.1.0 | `d06ff3fdb8b64d3ab02a246ab110b8ab2d93522a7a474ba41defccd3f81ab1d3` | `e81fe95afbeeb1ca7c9645a50cfb92045a5ac035a41356877a06fd0a1522bf2b` |
| 1 | 1.0.1 | `561d2309ebccc6bc019d6bde7d07a6039d06c64d7eab77385139171a4e0312cc` | `415f192efaf2e4ddd45604f1c53b9426a688eff148f5ddeb7d7501d62ec63487` |
| 1 | 1.0.2 | `b55fd2a1376fa8a014b4129ab1bb5135f5d8e7773f36a7b7d62d862b9e6682d8` | `2921e2176eeb27e064e25b42a81adebace24035703222ecee8705e3caf6183ad` |
| 2 | 0.1.0 | `7fbdce70b82ea6a6a170e457a6f20dcf49ec6ac2c560382bb57d8febce6df8c5` | `72726afcbdbb1bcc624251e43b7fd29797fab7990ba6667ae43b409dffa60938` |
| 2 | 1.0.1 | `3e74f655c8bb99df2036e4a1a5b90df8dc15aaa99bdd00186a76e4d79c8fe019` | `9045c08180c111edd8f5d5a680d44375aa139f10acbe1f25ebb8c9fc037ba9ee` |
| 2 | 1.0.2 | `83e5252b48112a3d7c659a4e3c93115241b62f22fa7853b96553a18161a20fe8` | `ea94a8c17a825ca31e5098b2edbc9b3b1e32f5eecb7c740e6d39b9298e634b3d` |
| 3 | 0.1.0 | `89384dd94d7363a8c93d208ef43b12235208c3f9694b983d012972389281c4d3` | `99fb7ac7796f8c1b435cb52d0d358b31314aef69396f16f51fb417a189864617` |
| 3 | 1.0.1 | `4633b4bce173e382b56984d3bffbc310e81aee5b923e17be22f542e92842eb72` | `ee3f29b0f4a020f43dd579d581a5fe9cb6231ec8ee7e2b656f7155ae12f40168` |
| 3 | 1.0.2 | `35db5ebe6e6ca99b6a99cd7306b976f72bce2b46501738f06a4c57ccacaeb8c2` | `61d55745b1a020b15e2322564d5e6c4bb094e9e158f4452337adbb18192353b3` |

## Self-check and cleanup

The requested e2e lint, root Go lint, tagged vet and root cross-architecture
build checks pass; unit tests run three times. `e2e-config` validates Compose.
No `e2e-verify` or e2e drift target is defined. No Python helper is owned by this
task; root unittest discovery reports zero tests (exit 5, not a test failure).
Final teardown and repository hygiene are recorded in the hand-off report.
