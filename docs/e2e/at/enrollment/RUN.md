# Enrollment acceptance runbook

From the repository root:

```sh
make -f mk/at-enrollment.mk at-enrollment-lint
make -f mk/at-enrollment.mk at-enrollment
make go-lint
```

The live target binds `FVPN_E2E_PREFIX=w4-enroll` and
`FVPN_E2E_PORT_BASE=66000`, always tears down the owned prefix, requests a fresh
stack with `E2E_RESET=1`, runs the API probes if startup succeeds, and tears down
again before final records are published. It exits nonzero unless both complete
ATs pass. **The assigned base is currently invalid (ENR-01).** No substitute slot
is chosen. A failed startup is recorded as blocked acceptance, never skipped or
presented as product success.

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
same immutable node image. Only the documented public certificate is read
from each identity directory; its SPKI fingerprint and admin identity are
compared. Node private keys stay on their nodes. OS machine identity, an
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
attempts are not overwritten. Any current failure outranks blocked checks. A
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

The current AT06 retains 27 prior blocked checks; AT07 retains 13. There are
no current or prior product passes/failures because the lab never started.
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
