# Isolation run procedure — Follow-up 6

```sh
make -f mk/at-isolation.mk at-isolation-lint
make -f mk/at-isolation.mk at-isolation
make go-lint
go vet ./...
make go-build
```

Use `FVPN_E2E_PREFIX=w4-isolation`, `FVPN_E2E_PORT_BASE=36000` and only
`fvpn-w4-isolation-*` Docker objects. The mechanism container is
`fvpn-w4-isolation-lab`. The target runs that lab first, removes it, builds a
fresh assembled stack, runs the shared driver through the owned Go overlay,
and tears down. The assigned prefixes are sequential. Host networking is
untouched; host test scratch is disk-backed under `.agent/tmp`.

A blocked or failed product case makes the aggregate exit 2 after all four records are
written. A mechanism pass does not promote a blocked product case (§19.2).
The final inventory includes containers, networks, volumes and image tags,
including the task-owned cache tags. Follow-up 5's teardown fix (#246) removes
the standalone isolation target before shared compose teardown, then removes
cache tags. No sibling prefix is inspected internally or changed.

Bootstrap uses `PrepareServingBaseline`, API-created DNS rules, groups, plans
and finite grants, native admission, and publication read-back. The runner
reuses shared `countedFlow`, `selectExit`, `tracedFetch`, and
`Stack.ObserveMemberIngress` through `driver.py`; SQL reads observations and
signed documents only. It neither writes projector-owned rows nor bypasses a
serving gate.

## Provisional sampling contract

Read [agent-core #250](https://github.com/TED-H-com/Freedom-VPN/pull/250)
and its Follow-up 20 report before choosing the cadence. Its routing section
names the approximately **5 s** lab window, referring to usage follow-up 3's
**5021 / 5025 ms**. The report's deterministic **55 s from ready / 60 s from
request** proof uses a 90 s heartbeat and is not this lab's expected window.
The historical 80.622 s first-ever-ready metric includes pre-publication
setup and is not an isolated renewal-cadence measurement.

The sampler now starts immediately after enrollment, before publication,
and runs at a **250 ms target cadence** until the first counted member byte.
Repeated reads are retained. Each sample has the desired/lease/observation
generations, grant count, lease ID and issuance time, observation ID and time,
all checks, all errors, forwarding state, and a separately timestamped live
forward-chain read. These are read-only snapshots; database and nft reads are
sequential, not atomic.

`EvaluateProvisional` requires at least three distinct samples inside one
same-generation grant-less interval with desired bindings, an accepted
ready/passed observation using that lease, `AWAITING_MEMBER_GRANTS` on
lease/qos/source_binding, exactly one error at stage lease, and no member
forward accept. It relates the ready timestamp to the populated lease's
issuance on that generation. The AT10 check additionally requires the real
member's later counted origin control. `convergence-timeline.json` retains
all reads, including transitions before a coherent observation is available;
those stale transitions are not scored as ready reports.

### Read-only recovery sampling used on this run

The ordinary bootstrap did not enter the qualifying condition. During the
final run's later tc failure, a separate observer ran against the same stack:

```sh
python3 tests/e2e/at/isolation/driver.py sample-recovery
docker restart fvpn-w4-isolation-node
# After the assembled target finishes and writes all four records:
FVPN_AT_REVIEW_RECOVERY_DIR="$PWD/docs/e2e/at/isolation" go test -count=1 -v -run '^TestAT10_ReviewRecoveryWindow$' ./tests/e2e/at/isolation
```

Start the observer before the recovery restart (in a second terminal); the
observer itself performs no mutation. Use the same assigned prefix, port base
and state-directory environment as the assembled driver. This was an extra
recovery after `TC_FOREIGN_FILTER`, not the later ordinary AT13 restart. The
exact action and time are in `recovery-action.json`. The observer retains a
rolling pre-window buffer, every in-window poll, and the first admitted
same-generation sample. Its executable source is recorded separately from
the stack's image source. It does not submit any agent observation.

The explicit offline review verifies run/build identity, re-evaluates all
qualifying samples, and requires a counted AT12 `node-management-22` control
whose serving heartbeat has the same generation after lease population. It
preserves the original bootstrap-only record under
`history/follow-up-6-bootstrap-only/`, then updates only AT10's provisional
check. The original interface failure and four blocked peer controls remain.
The positive control's `at` marks its attempt start; the post-transfer
heartbeat timestamps the successful counted recovery.

## Serving read-back

The first counted positive triggers `nft -j list ruleset` and the latest
accepted product observation read. The AT10 record retains both. D-079
source presence is established from the module root; attribution comes from
the observed foreign rules, not the presence of the patch alone.

[Netlab #256](https://github.com/TED-H-com/Freedom-VPN/pull/256) adds live tc
reads to `VerifyServing`. The mechanism AT10 control preserves the returned
`evidence.tc` **verbatim** in `lab-AT10-tc.json` and its `tc_readback`
observation. No foreign tc state is planted for this read. The assembled
record separately states whether the product observation exposes tc; lab
proof and product observation transport are distinct evidence surfaces.

## Existing subset controls

Each negative embeds that device's counted, CA-verified origin request.
Destination checks prove a listening target from the node and capture at
node/target ingress. The interface-change case maintains at least ten
protected-target attempts per second across a disposable dummy interface and
an API capacity change. Routes/interfaces are confined to owned containers.

Publication excludes B while retaining A, then disables and restores the
exit. Revocation observes TCP/UDP fixtures, conntrack deletion, a subsequent
serving generation and an admitted sibling. `Stack.ObserveMemberIngress`
measures revoked and forged `100.64.7.7` UDP/9 sources on `fvmb0`: `ctunku`
+1, member counters unchanged and no uplink/origin packet; the admitted
sibling moves its own upload counter by one. This existing hook uses
`tun-mirred`, link type 65534, source port 40000, with temporary injection
objects removed by the shared harness. It runs after the initial tc evidence.

Finite-member expiry uses the signed member deadline with control and worker
stopped, retaining the route and checking five ingress attempts, zero uplink
packets and frozen admission bytes. Whole-node expiry then uses the admin
fleet-config API, strong ETag/If-Match and Idempotency-Key to set the ceiling
to 60 seconds. A capacity patch forces a new lease. Control and worker stop
before its renewal lead; forwarding must hold near expiry and return to the
baseline within one configured agent tick. The ceiling is restored to 86400
through the same API and both audit events are retained. The original
TEST ONLY audit-reason wording remains unchanged. No host clock is changed.

## Records and archives

`AT10.json`–`AT13.json` and their Markdown files are the product records.
`lab-AT*.json` and sibling captures/matrices are mechanism evidence.
`summary.md` gives this run's measured outcomes; `requests.md` names remaining
owners. `cleanup-inventory.json` is the owned-prefix inventory after full
teardown. The source commit, immutable image IDs, run ID, incarnation and UTC
times are in the case records.

Follow-up 5's 33 records are byte-identical under `history/follow-up-5/`.
Earlier histories retain their original run identity. The live AT13 evaluator
SHA256 is `d4cd4dfd0957cace3ef9cd63168a4653c75ac1d0ba07d5c5ca1af00603f38d53`
(concatenated `capture.go` and `capture_review_test.go`); neither file changed.
The offline `at-isolation-review-captures` target rewrites records in its
selected directory. Use a copy of an archived run for offline review; never
replace current live records with that output.

## Follow-up 6 run identity and attribution

Final run `bf671f3a-c6ac-4824-9107-425c4c5a5f02`, harness `e711ebd822eda2ea1b5ba10b187818fda6777a74`, product origin/main baseline `5d1e5ce8727ca824fa24197ce2568bd513bac7f3`; UTC `2026-10-31T04:37:28Z`–`2026-10-31T05:01:32Z`. The foreign-rule read-back is `[]`: Docker chains are absent, so #240 explains the serving pass. D-079 #241 is present but no `loopback_scoped` entry was exercised. The initial diagnostic run, owned partial-operation wait correction, final-run tc rejection, extra real restart and 19-sample recovery proof are documented in `summary.md`. The final run was assisted; interface and four peer checks remain blocked, and whole-node expiry is failed on near-deadline continuity. Ready-observed → populated was 4767 ms; ready-received → populated was 4760 ms, beside usage follow-up 3’s 5021 / 5025 ms. Final inventory: `{"containers":0,"networks":0,"volumes":0,"images":0}`.
