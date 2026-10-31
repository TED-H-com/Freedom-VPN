# Enrollment acceptance runbook

From the repository root:

```sh
make -f mk/at-enrollment.mk at-enrollment-lint
make -f mk/at-enrollment.mk at-enrollment
make go-lint
```

The live target binds `FVPN_E2E_PREFIX=w4-enroll` and
`FVPN_E2E_PORT_BASE=12000`, always tears down the owned prefix, requests a fresh
stack with `E2E_RESET=1`, runs the API probes if startup succeeds, and tears down
again before final records are published. It exits nonzero unless both complete
ATs pass. ENR-01 is closed by devstack #182 and the orchestrator correction.
The merged table explicitly assigns 12000 to w4-enroll. The reserved 6000 slot
is never used. Failed checks retain their individual failure and block the AT.

Each invocation writes `AT06.json`, `AT06.md`, `AT07.json`, `AT07.md` and
`summary.md` before the first external action, then finalizes them and an
immutable `history/<run UUID>/` copy before returning nonzero. Private command
logs live only under `.agent/at-enrollment/<run UUID>/`. Each AT carries UTC
start/end times, command exit codes, source commit, dirty state, source-tree
SHA-256, runtime image IDs and test-binary digest when available. Unstarted
images and unbuilt live binaries are explicitly unavailable, not fabricated.

The tree hash uses sorted `git ls-files --cached --others --exclude-standard`
paths, excluding generated `docs/e2e/at/enrollment/` evidence. For each regular
file it hashes UTF-8 relative path, NUL and the binary SHA-256 of the contents.
The aggregate SHA-256 describes the exact source, including uncommitted driver
fixes. Product code is unchanged. Live release artifact/manifest hashes come
from `PrepareServingBaseline` when reached.

AT06 creates in-memory P-256 keys, PKCS#10 CSRs and contract JCS proofs. A
transport fault consumes the first HTTP 201 and returns an error without a
response to the client; its observer retains only public identity/certificate
digests. The identical body is retried and must return 200 with the same
certificate and incarnation. Two other holders start on a shared barrier and
race a fresh token; one must receive 201 and the other 409
`ENROLLMENT_KEY_MISMATCH`. Natural expiry must give 401
`ENROLLMENT_TOKEN_EXPIRED`; API revocation must give 401
`ENROLLMENT_TOKEN_REVOKED`. Missing/invalid proofs must give 422
`VALIDATION_FAILED`; token-only bootstrap on the mTLS listener must receive
the certificate-required TLS alert or explicit 401 `UNAUTHENTICATED`.

Admin node/enrollment/audit responses are reduced to public IDs, digests,
states and counts. The harness computes SHA-256 over the exact plaintext token
bytes and compares that digest with `Enrollment.token_sha256` on create, get
and revoke, and with `enrollment.created` `changes.after.token_sha256`. The
lowercase digest is an allowed scan hit. Plaintext and every other token-derived
encoding are negative patterns. `listNodeIncarnations` is called for every Node
the harness creates. A second incarnation is created only by `revokeNode`, then
`createEnrollment` with that `node_id`, then agent enroll with a fresh key.

The acceptance stack keeps `node` and `node2` for clean `Enroll` / `EnrollAt`
first boots, `MachineIdentityFingerprints`, and `StartClonedIdentityFixture`.
`JoinEnrolledIdentity` needs `node2` while that slot is still waiting, so the
runner brings up a second fresh stack for `EnrollmentPeerInventory`. Follow-up 15 #266 supplies usage and installer wrappers, but their slots differ:
usage hold/snapshot/transport use `node`, `node2`, `node-clone`; installation uses
only `installer-node`. The run checks this boundary and the real install result.
The regular node's endpoint-scope/snapshot proof is separate from counted usage
across reinstall. The current installer failure and exact remaining asks are in
[requests.md](requests.md); no service override, fake daemon, copied ledger or
alternate enrollment path substitutes for those prerequisites.

Control, worker and both available nodes' console streams plus node agent,
watchdog, resolver proxy and NetBird streams are scanned in memory for every
AT06 token and generated key's PEM base64 lines, DER encodings and raw scalar
encodings. Only byte/needle/match counts escape. Missing streams are blocked,
never zero hits. CSR/proof/request digests are stored; raw tokens, private
keys, full CSRs, proofs, session cookies and raw logs never enter public records.

Every previous check is retained with its run ID in `prior_checks`; historical
attempts are not overwritten. Any current failed subcheck determines the blocked AT observation. A
prior failed check prevents a later full pass until the orchestrator explicitly
reviews its resolution. The historical process failures are reviewed by name against explicit current
process exits, current AT06 audit/reinstall/history evidence, and D-078 behavior.
Under control #252, browser and read 429s are acceptable when Retry-After is
present and the next attempt succeeds; join must record zero. Class is inferred
from the current operation mapping, not a nonexistent response header. Historical
class labels stay as measured on their original tree. The AT07 `bebfe230` and
`35e542f8` process failures came from the shared AT06 invocation. They and the
`1d5c02a4` wget-body parser failure remain in `prior_checks`.

PR CI runs only formatting, vet, pinned staticcheck, offline harness tests and
amd64/arm64 compilation. Manual workflow dispatch runs Docker acceptance, as
in `at-updates.yml`, and uploads public records even on failure. Physical arm64
execution and real hardware/machine-image certification remain unmeasured.

## Initial attempts — 2026-10-30

Both invocations used source commit
`e4215bc151742774bbf957d2305246c514207de8`, source-tree SHA-256
`5e40bea111e14698dc95a939aa71097deb6cc495586110e393567fa5b72ee50d`.
Attempt 1 had a clean worktree; attempt 2's dirty flag reflects attempt 1's
new evidence files. The identical tree hashes confirm no driver/product source
change between attempts. The later evidence commit changes documentation only.

| Attempt | UTC interval | Run ID / immutable records | Result |
|---|---|---|---|
| 1 | 17:02:13Z–17:02:16Z | [79b89e47-b23f-45f8-a683-28e0557dc1ba](history/79b89e47-b23f-45f8-a683-28e0557dc1ba/AT06.json) | Startup 2; runner 1; make 2; both ATs blocked |
| 2 | 17:02:23Z–17:02:24Z | [9db51ea3-d807-449e-a812-cfad739db675](history/9db51ea3-d807-449e-a812-cfad739db675/AT06.json) | Startup 2; runner 1; make 2; both ATs blocked |

Actual startup diagnostic, identical on both attempts (path-bearing make
directory/recipe lines omitted from this public extract):

```text
fvpn-e2e-prepare: port base slot is out of range
make[1]: *** [mk/e2e.mk:42: e2e-up] Error 2
```

Actual second run tail:

```text
fresh_teardown: exit 0
fresh_startup: exit 2
final_teardown: exit 0
AT06: blocked: ENR-01: shared prepare rejected assigned port base 66000: port base slot is out of range
AT07: blocked: ENR-01: shared prepare rejected assigned port base 66000: port base slot is out of range
records: docs/e2e/at/enrollment; run 9db51ea3-d807-449e-a812-cfad739db675
make: *** [mk/at-enrollment.mk:12: at-enrollment] Error 1
```

The initial AT06 retained 27 prior blocked checks; AT07 retained 13. There are
no product passes/failures in these initial attempts because the lab never started.
Live images, live test-binary hashes, token/CSR API observations and log-scan
counts are unavailable. In particular, zero secret hits is **not** claimed.

Actual `at-enrollment-lint` tail, exit 0:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/enrollment	0.007s
....
----------------------------------------------------------------------
Ran 4 tests in 0.001s

OK
at-enrollment-lint: gofmt, vet, pinned staticcheck, offline checks and amd64/arm64 compilation passed
```

`make go-lint` exited 0 with empty stdout/stderr. It ran root gofmt, vet and
staticcheck v0.8.2. The area gate additionally compiled and vetted the e2e-tagged
driver, exercised offline Go/Python tests and cross-compiled its amd64/arm64
test binaries. No dependency changed.

Both final teardowns exited 0. Each attempt's final inventory was:

```json
{"containers": 0, "networks": 0, "volumes": 0, "images": 0}
```

## Follow-up 1 — corrected binding

Binding: `w4-enroll / 12000`, as assigned in shared RUN.md and #182.
The initial 66000 records above and their archives remain unchanged historical
evidence. This follow-up completed on the corrected slot.

ENR-02 and ENR-03 were reviewed against contract revision 9 before requesting
changes. See [API surface review](api-surface-review.md). The current missing
exports are assigned to devstack follow-up 12 (ENR-04/07) and follow-up 13
(ENR-05/06); no host-side replacement fixture is constructed.

Run: `75dbb974-04d2-4e45-a6bd-cfe0d923bd2c`, UTC
`2026-10-30T18:07:10Z`–`2026-10-30T18:09:56Z`.
Source: `8a3964e5bdbc397cc5322c58acce9c91d36ab357` (clean at invocation), tree
SHA-256 `0fef194960b8f9f7ef98d22e30ae086e182781c5c29222ccc4b1df7f30cd385e`.
The evidence commit adds documentation/results only. Runtime image IDs:

| Component | Immutable image ID |
|---|---|
| control | `sha256:14d606b2a2dabee749832f603179662161ebb9bf7804466817fe552ef9401592` |
| worker | `sha256:057f3531233dc26c2bb6006ddba4f99c30f08fc166f128b76ecdf1c1ea8437b7` |
| node and node2 | `sha256:228645164949bf932fb43eab692855229dad9755f63b27e7703df67c6a81398b` |

Live test-binary SHA-256:
`be16c1df64fddac681bd581257512fd2f5a283ddf0eda49bcc9a1296b493cdd2`.
Both architecture artifact digests and manifest/signed-manifest digests are in
`AT06.json`'s `serving_baseline` observation.

Actual live Go probe output (exit 0):

```text
=== RUN   TestAT06_AT07_Live
=== RUN   TestAT06_AT07_Live/AT06
=== RUN   TestAT06_AT07_Live/AT07
--- PASS: TestAT06_AT07_Live (89.77s)
    --- PASS: TestAT06_AT07_Live/AT06 (61.07s)
    --- PASS: TestAT06_AT07_Live/AT07 (25.21s)
PASS
```

This is successful execution of all available probes, not a full AT pass.
The record gate requires the missing evidence too. Actual target tail
(runner exit 1, make exit 2):

```text
fresh_teardown: exit 0
fresh_startup: exit 0
compile_probe: exit 0
live_probe: exit 0
final_teardown: exit 0
AT06: blocked: {'api_hash_matches': False, 'field': 'changes.after.token_hash', 'operation': 'listAuditEvents', 'owner': 'pki-enroll', 'request': 'ENR-02', 'resource_id': '01a18dff-1633-7ee4-a0c8-57652881a146'}
AT07: blocked: ENR-07 / devstack follow-up 12: MachineIdentityFingerprints is not exported; API identity comparison alone is partial
records: docs/e2e/at/enrollment; run 75dbb974-04d2-4e45-a6bd-cfe0d923bd2c
make: *** [mk/at-enrollment.mk:12: at-enrollment] Error 1
```

The consumed/retry certificate hashes match and the incarnation is unchanged.
The concurrent race returned one 201 and one 409 `ENROLLMENT_KEY_MISMATCH`.
Revocation returned 401 `ENROLLMENT_TOKEN_REVOKED`; after 61 seconds natural
expiry returned 401 `ENROLLMENT_TOKEN_EXPIRED`. Missing/false proof returned
422 `VALIDATION_FAILED`, and token-only bootstrap failed with the client
certificate-required TLS alert. For each replay/race resource, the admin audit
API returned one committed consumption, with only `reason` in its change fields.
Peer uniqueness for these own-key requests is unmeasured until ENR-04 is resolved.

AT07's real `node` and `node2` joined using the documented shared hooks, with
identical image IDs and distinct API Node/incarnation IDs. Both had bound peers.
No key-fingerprint, cloned-identity, unACKed counted usage, installer/reinstall,
old-certificate or preserved-usage assertion is inferred from those first boots.

The 64 in-memory token/key patterns found zero matches in 14 log streams
(31939 bytes), serialized current probe evidence (20040 bytes), and 30 existing
public evidence files. Empty-but-present streams are recorded with 0 bytes;
active agent/NetBird streams were also scanned. Missing files would have blocked
rather than been counted as zero matches. Only aggregate scan counts escape.

Actual final area gate tail, exit 0:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/enrollment	0.007s
....
----------------------------------------------------------------------
Ran 4 tests in 0.002s

OK
at-enrollment-lint: gofmt, vet, pinned staticcheck, offline checks and amd64/arm64 compilation passed
```

`make go-lint` exited 0 with empty stdout/stderr. Both Go build targets and the
real serving baseline compile amd64/arm64; physical arm64 execution is not claimed.
Final teardown exit 0; verified owned inventory:

```json
{"containers": 0, "networks": 0, "volumes": 0, "images": 0}
```

## Follow-up 2 — revision 10 and devstack follow-up 12

`git log origin/main` at measurement time contained
`52dc770 roadmap: w3-devstack-e2e follow-up 12 merged (#204)` and did not
contain a follow-up 13 merge. The branch was fast-forwarded to `efba7fb` so
the harness could call those exports. ENR-05 and ENR-06 stay blocked on the
absent names `HoldUsageAcknowledgements`, `UsageLedgerSnapshot`,
`CountedMemberTraffic`, `RerunInstaller` and `ControlledReinstall`.

The runner now uses two fresh stacks. The first keeps `node` and `node2` for
clean first boots, fingerprints and the clone fixture. The second uses waiting
`node2` for `JoinEnrolledIdentity`. `E2E_RESET=1` rebuilds each stack, so the
image IDs differ between cycles. Within the acceptance stack, `node` and
`node2` share one image.

Run: `7615d23d-d572-4db1-a903-9610dd3bcef3`, UTC
`2026-10-30T20:49:33Z`–`2026-10-30T20:55:13Z`.
Source: `efba7fba96e9e354900e9360b6ef4cfebf578ea7` (dirty at invocation), tree
SHA-256 `79cc1e81c8cc85f9044c425e3505608d51714bf91a90d0f444e0ed560899fc0b`.
The evidence commit adds documentation and the harness that produced this run.
Immutable predecessors stay in `history/`, including
`9bb66c4e-e429-450b-8671-521d33926ede` (revision 10 measurement on `746651e`)
and `bebfe230-26d4-4e3d-b052-81d258d04e07` (HTTP 429 before the harness honored
`Retry-After`).

| Cycle | control | worker | node |
|---|---|---|---|
| acceptance | `sha256:098c8246b6573ddb65136615a83035fbbd1b06bac593bbe5efa605d53e2a36ac` | `sha256:9b7488569ae7d110fd60ed7e97e397fa9fa966706b32c48f26b7cf6951fbed00` | `sha256:5d950b704594632e67993c167b8b4d889850dd0bd13800bb81639b445c983e27` |
| join | `sha256:d654b03b32fc271f9f0fe8da18be3591965d02fae9b978a1604729087e3dcbfc` | `sha256:0945b425115a97b07d6f30ee86785e2e74118b78f7f32a4f6a6b5945aebec75e` | `sha256:51ce29b488858de763341d43950628de94a8e72d33a304742d0d074758133d7f` |

Live test-binary SHA-256:
`955f92e2442196807f46d20c8f4679411500e686a9c82cda180bd29e6328bba8`.

Token digest equalities (plaintext withheld). Replay enrollment
`01a18e93-b9fe-7b24-9cc7-82091c90bfe0`, digest
`50435538cb9a48656dfd0df1d462f8cf0d56dbb90c2c074b058fd998514d673e`, matched on
create, pending get, consumed get, and `enrollment.created` only. Revoked
enrollment `01a18e93-bcf0-72b6-9d38-d72af92b5636`, digest
`86664b1c9de1260254e2d60916b5fcfb36b7fb7aba159ece097d6bbb6e075b35`, also matched
the revoke response. Reinstall on node `01a18e93-b9fe-7b25-9ced-0a7211195a0a`:
count 2, newest `2815de1f-1728-4c3b-afaa-73cd45945654` current, previous
`0a8192ac-6bf7-421e-9cc8-db98ad7df608` retired with reason `reinstall` after
`revocation`, usage unchanged `"0"`/`"0"` with `through` null, cursor round-trip
without duplicates. Join enrollment `01a18e97-7339-7310-a690-01af5e615720`:
discarded race HTTP 201 and 409 `ENROLLMENT_KEY_MISMATCH`, replay enroll 200,
bootstrap 200, inventory peers 1, setup keys 1, uses 1, second read still one
peer.

Acceptance-stack secret scan: 77 in-memory patterns, 14 streams, 88535 bytes,
0 plaintext or other token-derived matches, 0 lowercase-digest occurrences in
logs. Admin reads: 48 bodies, 40938 bytes, 16 digest occurrences, 0 plaintext.
The join stack scanned its own control/worker logs (10041 and 3683 bytes, 1
needle) and one admin body (502 bytes, 1 digest); plaintext matches were 0.

Actual target tail (runner exit 1, make exit 2):

```text
fresh_teardown: exit 0
fresh_startup: exit 0
compile_probe: exit 0
live_probe: exit 0
join_teardown: exit 0
join_startup: exit 0
live_probe_join: exit 0
final_teardown: exit 0
AT06: blocked: Prior failed check requires explicit review
AT07: blocked: StartClonedIdentityFixture.Observe same_identity true, conflicts 0, mapping_frozen false (ENR-08)
records: docs/e2e/at/enrollment; run 7615d23d-d572-4db1-a903-9610dd3bcef3
make: *** [mk/at-enrollment.mk:12: at-enrollment] Error 1
```

The AT07 line above names the decisive fields. The process printed the full
observation object; it is stored as AT07 `actual`. Both Go probes exited 0.
AT06's blocked status is the retained `bebfe230` 429 failures, not a current
subcheck. Current counts: AT06 205 passing / 0 blocked / 0 failed; AT07 15
passing / 7 blocked / 0 failed.

Actual area-lint tail, exit 0:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/enrollment	0.007s
....
----------------------------------------------------------------------
Ran 4 tests in 0.002s

OK
at-enrollment-lint: gofmt, vet, pinned staticcheck, offline checks and amd64/arm64 compilation passed
```

`make go-lint` and `go vet ./...` exited 0. Final teardown exit 0:

```json
{"containers": 0, "networks": 0, "volumes": 0, "images": 0}
```

## Follow-up 3 — production caller, second /24, retained 429s

HEAD at measurement was `99ab668` (`#219` follow-up 13, `#217` identity projection, `#211` `ObserveEndpointTx`). ENR-05 and ENR-06 stay blocked: those five symbols are not in `docs/e2e/RUN.md` or `tests/e2e`, matching the follow-up 13 recheck. The harness did not emulate them.

`StartClonedIdentityFixture` still has no network argument. On base 12000 the running lab confirmed source `172.28.38.40` / `192.0.11.20` and clone `172.28.38.129` / `192.0.11.23`. Both pairs are one IPv4 `/24`. Trusted proxy remains `172.28.38.21/32`, and the agent dials control through the local resolver proxy, so `X-Forwarded-For` is not the compatibility domain. ENR-07 stays blocked for devstack follow-up 14. ENR-08's missing-caller gap is closed by `#211` and `#217`; this run did not measure a conflict row.

Attempt `35e542f8-730e-4be0-8214-e4a9dd818c40` saw the same two browser 429s and failed the Go process because the review check used status `fail`. That directory is unchanged. The gate now records a recurring 429 as `blocked`, so the process can exit 0 and the deciding observation names the limiter.

Run: `60a7102e-b824-4927-a576-62033f458157`, UTC
`2026-10-30T23:05:38Z`–`2026-10-30T23:10:54Z`.
Source: `99ab668e686712d5d0ad4e0c6b79d332fc6f9d25` (dirty), tree SHA-256
`62e86bf6b54985af168edc6d49193a272c4442007b03074df8686e07f37be406`.
Live test-binary SHA-256:
`2884a259187ad5ecd78c400e6cb905ecde344dbaf7cf024194f700b638d4ba25`.

| Cycle | control | worker | node |
|---|---|---|---|
| acceptance | `sha256:d2f3a7faa0c87483f5170778540105ab86ce35ace89a3f75a1852258423ca3ef` | `sha256:8a8db70d406b7aa4a316196d30bb8e32218f04bea94ff42416483d76306876b9` | `sha256:041966e57aeb3c66fa8c99fdb91e72acdc94aa3804003ba6ee87808835552cc7` |
| join | `sha256:1f0db9fb88c7cb665af7cdfddc30e12898927125490c2eb9fdb0fb27027e6611` | `sha256:f029174cfd8ea0f00ec85b42ed80a4b15cbee0f4d1a9c3c7c0e5446461f26f46` | `sha256:add5aef7045e2cb5ef86efe96b5a1e097e8bfbef98d9a6a07965ece0a5dada86` |

`Observe` before the clone finished booting did not return a row (`read_ok` false). After both stages were `enrolled`, and again after the short settle: `endpoint_rows` 1, `conflicts` 0, `mapping_frozen` false, same three fingerprints. `getNode` HTTP 200 projected `identity_conflict` `{mapping_frozen: false, conflict_count: 0, last_conflict_at: null}` before the fixture and after the settle. Provisioning was still `applying`, safety `failed`, forwarding false; that is not a freeze. Counts: AT06 254 passing / 1 blocked / 0 failed; AT07 15 passing / 7 blocked / 0 failed.

```text
fresh_teardown: exit 0
fresh_startup: exit 0
compile_probe: exit 0
live_probe: exit 0
join_teardown: exit 0
join_startup: exit 0
live_probe_join: exit 0
final_teardown: exit 0
AT06: blocked: HTTP 429 recurred (2) on limiter class browser, bucket BrowserRate 10/s burst 30. Owner: control. The bebfe230 review stays unresolved.
AT07: blocked: ENR-07 / devstack follow-up 14: the lab has no second /24 for the clone.
records: docs/e2e/at/enrollment; run 60a7102e-b824-4927-a576-62033f458157
```

Runner exit 1. `make -f mk/at-enrollment.mk at-enrollment` exit 2. Final inventory `{containers:0,networks:0,volumes:0,images:0}`.

## Follow-up 4 — D-078 and the clone-presentation network

HEAD at measurement was `9e493e8` (devstack follow-up 14, `#240`). The harness was dirty: D-078 scoring and the ENR-07 recovery path were uncommitted. The JSON `source.dirty` field stays true. Tree SHA-256 excludes `docs/e2e/at/enrollment/`.

`StartClonedIdentityFixture(ctx, "node")` is unchanged. `AttachClonePresentationNetwork` is called through `tests/e2e/internal/labstack`. On base 12000 the presentation network was `fvpn-w4-enroll-node-clone-net`, `172.29.38.0/24`, control `172.29.38.30`, clone netns `172.29.38.23`. The process container rewrote `lab-hosts` and did not pass `--add-host`. `same_lab_slash24` and `same_origin_slash24` stayed true because the clone netns remains on `172.28.38.0/24` (`172.28.38.129`) and `192.0.11.0/24` (`192.0.11.23`). The agent dials `agent.example.invalid`, which `lab-hosts` resolves to the presentation control address, so the compatibility domain is `172.29.38.0/24`.

ENR-05 and ENR-06 stay blocked, waiting on devstack f15. Expected exports: `HoldUsageAcknowledgements(ctx, slot, hold)`, `UsageLedgerSnapshot(ctx, slot)`, `CountedMemberTraffic`, `RerunInstaller(ctx, slot)`, `ControlledReinstall(ctx, slot, enrollment)`. `fvpn-agent usage snapshot [-json]` is merged and is not the usage-hold hooks. `fvpn-node-install` is on main (`#239`) and is not the installer wrappers. The entrypoint re-enroll used for the clone conflict is not ENR-06. The harness did not emulate those hooks.

Attempt `1d5c02a4-b254-4a9b-b687-b1535428fb3f` is retained. Its probe exited 1 because wget omitted `--content-on-error` and the 403 body was empty. The conflict observations in that directory stay. The `IDENTITY_REVOKED` pass is claimed from `ec2e6279` only.

Run: `ec2e6279-0649-4d47-b31a-cf13aeba7303`, UTC
`2026-10-31T02:18:00Z`–`2026-10-31T02:23:47Z`.
Source: `9e493e8f6407dd7f091963fb7e133a11123305c3` (dirty), tree SHA-256
`d57359f925f2860a4efab951ed1c6d10025bfe916d3ff4474b4a57203437d2d5`.
Live test-binary SHA-256:
`93b735ac8059ca086f429e560d32d20cf9f07e1798de2ca8a47944fc9ab0f6f7`.

| Cycle | control | worker | node |
|---|---|---|---|
| acceptance | `sha256:223129268f08d40ebc5b8dfa252b0ce750923b88fd7f5921f5ff18acefef56ed` | `sha256:52aaf09e25f23bd7c6c991e835a07ecba5f8c20477fc39fd01b4080bd0447dd7` | `sha256:3eb87e3696629ddffe62cb9922cfdf3b3f6d2cb42095dad62ae4bfbea28c4e0a` |
| join | `sha256:43187b2434a362aa814145215dc7f4e0790206ab3336686b91aea8648ba8e7b0` | `sha256:bd3a46c7c33796db1828b0f5b0972c5f859475856d1a36844a7c5eae1c590c37` | `sha256:a82e73d0d00a964035f29e07bc09fcdded9294f79ee764b9321c795887b46642` |

Full-phase `rate_limit_review`: `behaviour` pass, `http_429` 2, both class `browser`, `retry_after` true, `retry_succeeded` true. Join-phase `rate_limit_review`: `behaviour` pass, `http_429` 0, `samples` empty. Follow-up 3 run `60a7102e` is re-scored in `prior_failure_review.rescored_follow_up_3` with `header_value_recorded` false. AT06 disposition `resolved`. Counts: AT06 305 passing / 0 blocked / 0 failed; AT07 16 passing / 6 blocked / 0 failed.

`cloned_identity_conflict` passed. `Observe` frozen and stable: `conflicts` 1, `mapping_frozen` true, `endpoint_rows` 1, one `node.identity.conflict` audit before and after the 8s wait. `projection_frozen`: HTTP 200, object, `conflict_count` 1, `last_conflict_at` present, `provisioning_state` `failed`, `member_forwarding_observed` false, `safety_status` `failed`. Revoke HTTP 202. Replacement incarnation `01a18fc3-386b-716e-960f-fa63faeb507a` (old `01a18fc1-4a36-76d7-b0de-b274b6d4486c`). Projection after re-enroll and after the clone presented again: object `{mapping_frozen: false, conflict_count: 0, last_conflict_at: null}`, `provisioning_state` `applying`. Clone desired-state: HTTP 403 `IDENTITY_REVOKED`, `clone_denied` true. `Observe` after re-enroll on the old incarnation: `conflicts` 1, `mapping_frozen` true, `same_identity` false.

```text
fresh_teardown: exit 0
fresh_startup: exit 0
compile_probe: exit 0
live_probe: exit 0
join_teardown: exit 0
join_startup: exit 0
live_probe_join: exit 0
final_teardown: exit 0
AT06: pass
AT07: blocked: ENR-05 stays blocked, waiting on devstack f15.
records: docs/e2e/at/enrollment; run ec2e6279-0649-4d47-b31a-cf13aeba7303
```

Runner exit 1 because AT07 is blocked. `make -f mk/at-enrollment.mk at-enrollment` exit 2 by design. Final inventory `{containers:0,networks:0,volumes:0,images:0}`.

## Follow-up 5 implementation and prerequisites

Follow-up 4's root records are preserved in [history/follow-up-4](history/follow-up-4/README.md).
This run invokes `PrepareInstallerNode`, API token creation, `StageInstallerToken`
and initial `RerunInstaller` against `fvpn-w4-enroll-installer-node`. The result
records actual exit/status/code, identity IDs, image digest, before/after hashes,
UTC invocation times, and boolean journal diagnostics. Raw journals, outcome
messages and tokens are not published. Successful rerun/reinstall assertions
require a working initial installation and same-slot counted-usage support.

`usage_hold_endpoint_scope` stops the regular node agent through `Service`,
records the CLI snapshot, holds only the usage POST route, verifies the retryable
503 while desired-state still responds, releases the hold, checks central's
malformed-JSON validation response, and compares exact snapshot stdout. Cleanup
releases the marker and resumes the agent. This probe has no member traffic and
cannot satisfy `counted_unacked_usage` or historical usage retention. No monthly
total, held-byte or central-once assertion is claimed from it.

The shipped wrappers leave agent/watchdog/updater stopped after hash sampling;
forwarding restoration needs a devstack export. `Reenroll` retains old unACKed
usage but explicitly has no authorized upload path after the identity changes.
These limitations are owned requests, not skipped evidence represented as pass.

### Follow-up 5 measured result — `4011dd32-73ef-4f49-b78b-6cf1ab8496c9`

UTC **2026-10-31T06:10:11Z–2026-10-31T06:15:49Z**. Source
`200c027144ecf523c647c0954659467ca873ae5f`, clean at invocation; tree SHA-256
`04e361cfab859e45662593cf6bd439100f28019e1a6af4399fa9ec595438ccf9`. Test binary SHA-256
`9ded592b18155b5490d617efc49e00d0ecb9b678fe0b6ad9da4ef93e83092878`. The final evidence commit also corrects
one legacy reporting sentence to say browser/read under #252 and removes an
unnecessary Python string escape; neither changes the measured predicates.
The immutable records retain the exact wording emitted by the measured source.

AT06 **344 / 0 / 0** (pass / blocked / fail); AT07 **19 / 6 / 0**.
Both `prior_failure_review` dispositions are `resolved`. AT07 explicitly names
`bebfe230` and `35e542f8` as shared AT06 process failures, plus the `1d5c02a4`
403-body parser miss. The current processes each exited 0, and the supporting
AT06 audit/reinstall/history and D-078 checks passed. Original rows remain.

Full and join phases each recorded `http_429: 0`. `rate_limit_review` includes
#233/#252 defaults and method/path classification provenance. No class-specific
rejection or new saturation limit was observed in this run.

| Measurement | Actual result |
|---|---|
| Hold mechanism | Network-side proxy marker on `node`; only usage POST withheld before control; no ledger-epoch rotation |
| Hold requested / applied | `2026-10-31T06:12:05.707281842Z` / `2026-10-31T06:12:05.728533112Z` |
| Release requested / applied | `2026-10-31T06:12:05.829922214Z` / `2026-10-31T06:12:05.850705481Z` |
| Held / other / released responses | 503 `DEPENDENCY_UNAVAILABLE`, retryable true / desired-state 200 / usage 422 `VALIDATION_FAILED` for an empty JSON body |
| Ledger | Exact paused stdout unchanged before/during/after; event sequence and ACK `"0"`, unACKed count 0; no member traffic; not a retention/drain proof |
| Cleanup | Hold released, agent resumed |
| Installer invocation | `2026-10-31T06:13:19.306946818Z`–`2026-10-31T06:13:42.72750912Z`; `PrepareInstallerNode` → API token → `StageInstallerToken` → `RerunInstaller` |
| Initial install | Exit `1`, code `failed`, status empty, wrapper error false; real enrollment succeeded, service startup failed |
| Installer identity | Node `01a19097-5517-7833-bb6e-7a0fb2533c95`, incarnation `01a19097-5a5e-7a9d-be44-03ed39315faf`; API has one active/current row; safety unknown, forwarding null |
| Installer diagnostics | `netbird_socket_readonly: true`, `agent_heartbeat_readonly: true`; journal scan 0 secret matches |
| Same-slot support | Hold, snapshot and selected member transport each returned `unknown node slot` for `installer-node` |
| Still unreached | Counted backlog, successful rerun, controlled reinstall, mode-0700 retirement, old certificate on both runtime paths, nonzero historical attribution after new traffic |

The initial `RerunInstaller` is an installation attempt, not evidence of an
idempotent rerun on an enrolled, forwarding node. Its real exit 1 is preserved
inside each dependent blocked check. It does not create a synthetic failed
`probe_process` row. No unit overrides or NetBird stub were used.

Actual make output tail (exit 2; long result descriptions omitted here, retained
verbatim in the AT records and private command log):

```text
fresh_teardown: exit 0
fresh_startup: exit 0
compile_probe: exit 0
live_probe: exit 0
join_teardown: exit 0
join_startup: exit 0
live_probe_join: exit 0
final_teardown: exit 0
records: docs/e2e/at/enrollment; run 4011dd32-73ef-4f49-b78b-6cf1ab8496c9
make: *** [mk/at-enrollment.mk:12: at-enrollment] Error 1
```

Area lint, final output tail (exit 0):

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/enrollment	0.007s
..................
----------------------------------------------------------------------
Ran 18 tests in 0.004s

OK
at-enrollment-lint: gofmt, vet, pinned staticcheck, offline checks and amd64/arm64 compilation passed
```

`make go-lint` and `go vet ./...` exited 0 with empty stdout/stderr.
`git diff --check` passed. Area lint includes the tagged/offline tests, pinned
staticcheck and linux/amd64 plus linux/arm64 compilation. Live execution was
amd64; physical arm64 remains unmeasured. No dependencies changed.
All fresh/join/final teardowns exited 0 and the final owned-prefix inventory
was `{containers:0, networks:0, volumes:0, images:0}`.

## Follow-up 6 — fixed installer, missing same-slot response

Base `945eff8` includes installer PR #279. Read its PR body and
`docs/node/install-followup3-2026-10-31.md` before invoking the wrappers.
`docs/e2e/RESPONSES.md` is absent on this tree. The follow-up 15 documentation
and implementations still reject `installer-node` for
`HoldUsageAcknowledgements`, `UsageLedgerSnapshot[Bytes]` and
`MemberTransportStack`. `Service` targets the regular supervised node and has
no installer-slot parameter. No undocumented service-restoration command is
used. Missing same-slot traffic and serving evidence stays blocked.

The installer probe uses `PrepareInstallerNode` → API enrollment token →
`StageInstallerToken` → initial `RerunInstaller`. If installation succeeds,
it calls the same-release `RerunInstaller`, then API revocation and a fresh
same-Node enrollment before `ControlledReinstall`. It records wrapper
exit/status/code, timestamps, identity IDs, before/after SHA-256 digests,
retired mode/digests and API incarnation-history/usage projections. A
successful invocation does not satisfy the missing serving/usage prerequisite.
Only both direct mTLS `403 IDENTITY_REVOKED` responses, after successful
reenrollment, can close the separate old-certificate row.

Each reached installer phase captures journal, service status and socket-path
observations before teardown. Raw diagnostic messages stay private under the
run's `.agent/at-enrollment/` directory; public evidence uses selected fields.
The fixed socket `/run/fvpn-netbird/netbird.sock`, old socket absence and mode
of `/run/fvpn-netbird` are observed independently of the CLI exit. No service
overrides, copied identity state, hand-run installer or fabricated meter
checkpoint is used. The journaled higher-release upgrade is not invoked; its
known `final_sample` integration gap remains outside AT07.

Digest fields are `key_sha256`, `cert_sha256`, `node_json_sha256`,
`usage_db_sha256`, `usage_wal_sha256`, `machine_id_sha256`, and `current`.
The `absent` sentinel is retained. Historical files are unchanged; prior rows
copied into the current root records get only these field-name substitutions.
[Follow-up 5 archive](history/follow-up-5/README.md) preserves all eight root
files byte-for-byte and records their SHA-256 checksums (D-098).

The area lint target now uses Python `TemporaryDirectory` on disk. The root
lint helper can be overridden without editing `mk/go.mk`:

```sh
make -f mk/at-enrollment.mk at-enrollment-lint
make -f mk/at-enrollment.mk at-enrollment
make go-lint 'go-with-disk-tmp=python3 tests/e2e/at/enrollment/disktmp.py'
python3 tests/e2e/at/enrollment/disktmp.py go vet ./...
python3 tests/e2e/at/enrollment/disktmp.py go test ./...
git diff --check
```

Before the live run, checked `pgrep -af gates.sh | grep -v grep` and
`docker ps --format '{{.Names}}' | grep -c '^fvpn-w'`. Waited through busy
checks at 07:17:36 and 07:22:43 UTC. At 07:27:51 UTC only one other gate was
active and the container count was zero; the enrollment run then started.

### Follow-up 6 measured result — `1af27741-cbd9-4fdb-aa1c-7d02790e7a46`

UTC **2026-10-31T07:28:01Z–2026-10-31T07:34:14Z**. Source
`3f604e2c2d0fd89ee7e2d91014ab31ff46aed7a8`, clean at invocation. Tree SHA-256
`a7e7159e3f9652344ec56461abd2242de6c1f64cc0425af715a7f84b2aa143e8`; binary SHA-256
`e8b2b9a297b2b98392a9633449ffe090969e731a32da31b311fdfe6cb942c67a`. Final changes after measurement are records only.

AT06 **376 / 0 / 0**; AT07 **26 / 5 / 0** (pass / blocked / fail).
The AT06 increase from 344 is exactly 32 additional evidence-file scan rows:
nine follow-up-5 archive files plus seven new attempt files in both phases.
All previous assertion names/counts remain represented. Both phases record
`http_429: 0`, `behaviour: pass`. Both historical review dispositions are
`resolved`; original failures are preserved.

| Installer measurement | Result |
|---|---|
| Initial invocation | `2026-10-31T07:31:09.346675341Z`–`2026-10-31T07:31:11.930606746Z`; exit 0, status `installed` |
| Same-release rerun | Exit 0, status `already_current`; initial-after = rerun-before = rerun-after across all seven hash/link fields |
| API authorization | Node revoke 202; fresh `node_id`-bound enrollment; token delivered only to ControlledReinstall |
| Controlled reinstall | `2026-10-31T07:31:13.707442342Z`–`2026-10-31T07:31:27.749790797Z`; exit 0, status `reenrolled` |
| Stable Node | `01a190de-9766-70be-b3a3-f1a4e64c7a10` |
| Old incarnation | `01a190de-9d5f-7615-a589-5ad0f19271ec` |
| New incarnation | `01a190de-b351-76b9-9c0f-8300d9496d3a` |
| Retention | Retired mode `700`; key/cert/node.json hashes match; all four bound-state observations match; usage.db unchanged and usage WAL `absent` before/after |
| API history | Two newest-first rows: new active/current, previous retired with reason `reinstall`; old upload/download `"0"`/`"0"` and checkpoint time unchanged; new usage `"0"`/`"0"`, through null |
| Old certificate | Direct mTLS desired-state GET and observations POST each 403 `IDENTITY_REVOKED` |
| Socket | `/run/fvpn-netbird/netbird.sock` present, parent mode `700`; `/var/run/netbird.sock` absent after all three invocations |
| Prior startup errors | NetBird socket and agent heartbeat read-only diagnostics both false in each journal sample |
| After every wrapper | NetBird active/running; agent, watchdog, updater service and updater timer inactive/dead, result success |
| API serving projection | Applying; safety unknown; member forwarding null both after initial install and after reenroll |
| Same-slot exports | Hold, snapshot and member transport all report `unknown node slot`; no documented service-restoration export |

The wrapper's `code` field is `baseline` on each successful invocation;
`exit: 0`, success status and `failing_step: none` are retained separately.
It is not a failure code in this observation. The lifecycle commands complete,
but their wrapper cleanup leaves no observed serving baseline. No counted
traffic, held nonzero event set, historical nonzero attribution, once-only
backlog delivery or full journaled upgrade is claimed.

Remaining blocked rows: `counted_unacked_usage`, `installer_rerun`,
`identity_ledger_preserved`, `controlled_reinstall`, `historical_usage_retained`.
The `old_certificate_denied` row now passes independently on both runtime paths.
Owners and required exports are in the dated D-102 requests section.

Actual live gate output (exit 2, result description lines omitted here; full
strings remain in AT JSON and the private command log):

```text
fresh_teardown: exit 0
fresh_startup: exit 0
compile_probe: exit 0
live_probe: exit 0
join_teardown: exit 0
join_startup: exit 0
live_probe_join: exit 0
final_teardown: exit 0
records: docs/e2e/at/enrollment; run 1af27741-cbd9-4fdb-aa1c-7d02790e7a46
make: *** [mk/at-enrollment.mk:12: at-enrollment] Error 1
```

Actual area lint tail (exit 0):

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/enrollment	0.008s
...................
----------------------------------------------------------------------
Ran 19 tests in 0.004s

OK
at-enrollment-lint: gofmt, vet, pinned staticcheck, offline checks and amd64/arm64 compilation passed
```

Root `make go-lint` with the disk-scratch helper override and `go vet ./...`
each exited 0 with empty output. Untagged `go test ./...` exited 0.
The area gate includes tagged offline tests, pinned staticcheck and both
linux/amd64 and linux/arm64 compilation. Live hardware is amd64; real arm64
execution is not claimed. No dependency changes or tmpfs test scratch.
Final teardown exited 0 and inventory was
`{"containers":0,"networks":0,"volumes":0,"images":0}`.

### D-102 final moving-main recheck (2026-10-31)

While this run was in progress, main advanced to `5ce76e9d2b1d26c49a44d77c9184d7aa73d71f31` and added
`docs/e2e/RESPONSES.md`. Read that file directly from the main commit during
the final audit: it contains members and updates answers only, with no
enrollment/ENR-05 response. The measured worktree remains based on `945eff8`,
where the file is absent. This later documentation does not close the missing
installer-slot exports. No merge, rebase, pin update or product change was made;
all live results retain their exact measured source.
