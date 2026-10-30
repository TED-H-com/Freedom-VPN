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
states and counts. Current-incarnation equality does not prove historical
incarnation uniqueness; locally hashing a token does not prove its stored hash.
Those missing observations stay blocked (ENR-02/03). No peer-uniqueness pass is
possible until a real join and complete peer inventory are available (ENR-04).

AT07 clean first boots use `Enroll` and `StartSecondExit`/`EnrollAt` with the
same immutable node image. Admin identities and image IDs are compared.
Fingerprinting awaits the documented MachineIdentityFingerprints export; the
harness does not emulate it with identity-file reads. OS machine identity, an
enrolled-identity clone, unacknowledged counted usage, installer rerun,
controlled reinstall, old mTLS identity rejection and historical usage all
remain explicit required checks with named missing hooks (ENR-05..07).

Control, worker and both available nodes' console streams plus node agent,
watchdog, resolver proxy and NetBird streams are scanned in memory for every
AT06 token and generated key's PEM base64 lines, DER encodings and raw scalar
encodings. Only byte/needle/match counts escape. Missing streams are blocked,
never zero hits. CSR/proof/request digests are stored; raw tokens, private
keys, full CSRs, proofs, session cookies and raw logs never enter public records.

Every previous check is retained with its run ID in `prior_checks`; historical
attempts are not overwritten. Any current failed subcheck determines the blocked AT observation. A
prior failed check prevents a later full pass until the orchestrator explicitly
reviews its resolution. A partial readback cannot erase a prior failure.

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
