# Current guard: guarantees, limits and measured cost

Design input, 2026-11-03. All code line numbers refer to task-start main
`cc0ed4ae713f6c8242171f923bdc799a0c07a74c`. Links open the corresponding repository
file; ranges in the prose identify the reviewed code. No new implementation or
live measurement is represented here.

## Contract and vocabulary

[Project Spec](../../spec/PROJECT_SPEC.zh-TW.md) §8.1 requires member/host/internal
isolation, with narrowly scoped DNS and control exceptions. §8.2 requires atomic
rule updates, no allow-all window, revocation before established/related accepts,
and closure when the isolation/baseline/generation check fails. §16.3 additionally
requires autonomous kernel expiry, elapsed-time accounting and an independent
watchdog. §17.1 starts the normal online revocation target at **60 seconds**;
it is not a 60-second allowance after a node has already installed a revocation.
AT10 and AT13 are the reload/isolation and existing-flow revocation cases (§19).

Here, a deadline bounds **admission at the forwarding hook**. A packet admitted
just before expiry can leave a downstream tc queue afterwards. No claim that all
wire egress finishes by the deadline follows from these rules. Suspended machines,
VM restoration, unsupported kernels and hostile root remain platform/owner
acceptance matters, not properties established by this design.

## Why the guard exists

Let `R` be the final monotonic render sample, `B = 100 ms`, `q = 1 ms`, and `D`
the earliest permitted deadline for a binding. The renderer emits
`T = floor_q(D - R - B)` when positive. A relative timeout begins at kernel
installation, not at `R`. If its start `C <= InstalledBy <= R+B`, its expiry
`E = C+T <= D`. Without a guard, an arbitrarily delayed installation with
`C > R+B` can admit beyond `D`, even if the command subsequently returns an
error or the agent dies before checking elapsed time.

The conservative proof treats timeout start as late as commit; individual kernel
versions may initialize it earlier during transaction preparation. It needs only
the upper bracket, not an invented exact commit timestamp. Main explicitly
documents the late-commit/death hazard in
[deadline_install.go:40–46](../../../node/enforce/deadline_install.go).
The guard and candidate are one `nft -f` transaction. A forward base chain at
priority **−151** drops both `iifname MEMBER` and `oifname MEMBER`, ahead of the
ordinary **−150** forward chain. Its `policy accept` does not exempt either
member direction. Only a successful elapsed-budget check permits a second
transaction to delete it. That deletion does not restart the candidate's timers.

The deadline calculation also caps service end by node lease end, signed server
time plus offline limit, binding validity, optional timeout ceiling and anchored
monotonic deadline. Missing required inputs and nonpositive remaining allowances
omit admission. See [deadline.go:11–61](../../../node/enforce/deadline.go),
[model.go:89–114](../../../node/enforce/model.go), and the agent's signed-to-
monotonic model construction in [decide.go:619–695](../../../node/agent/decide.go).
The published D-134 transition window is **[D−101 ms,D]**, not a promise of zero
pre-D loss; see the [follow-up 21 assessment](../G2-followup21-2026-10-31.md).

## Guarantee ledger

| ID | Actual guarantee and its limit | Code evidence |
|---|---|---|
| G1 deadline | Before budget approval the candidate is inaccessible to every member. After approval its countdown cannot exceed `D` under the supported timeout semantics. Established flows have no expiry exemption. Wall-clock changes cannot refresh the elapsed allowance. | [apply.go:155–209](../../../node/enforce/apply.go); [deadline_install.go:13–21,40–50](../../../node/enforce/deadline_install.go); [render_nft.go:132–139,181–190](../../../node/enforce/render_nft.go) |
| G2 revocation | At the guarded candidate's atomic commit, all member admissions stop. After release, an address absent from the new allowset is still dropped in both directions before established/related accepts. Conntrack deletion is cleanup, not the authorization gate. | [render_nft.go:96,132–139,181–190](../../../node/enforce/render_nft.go); [apply.go:236–274](../../../node/enforce/apply.go) |
| G3 baseline | Boot with no adoptable verified same-boot generation installs generation 0: empty member allowset, forward policy drop, unconditional member-direction drops, member input drops, control output retained. A separately bounded synthetic probe may be the only forward exception; ordinary rollback uses the bare baseline. | [engine.go:415–457](../../../node/agent/engine.go); [render_nft.go:256–294,335–394](../../../node/enforce/render_nft.go); [apply.go:277–315](../../../node/enforce/apply.go) |
| G4 rejection | Invalid datapath, load-state error and initial Render rejection can return before any closure attempt. Later failures attempt baseline or quarantine; the agent rejects serving and withdraws on a non-verified-conntrack error. Thus `Apply(error)` alone is **not** proof of installed closure. | [apply.go:78–147,151–175,191–234](../../../node/enforce/apply.go); [engine.go:1103–1121](../../../node/agent/engine.go); [serving.go:127–155](../../../node/agent/serving.go) |
| G5 kill/crash | Once the guarded install commits and before guard deletion commits, agent death leaves **every member closed**, independently of watchdog scheduling or the candidate deadline. Before that commit, the old kernel state can remain. After deletion, the candidate's already-safe timers remain. | [deadline_install.go:40–50](../../../node/enforce/deadline_install.go); [apply.go:177–205](../../../node/enforce/apply.go) |
| G6 late commit | Timeout/error/negative elapsed sample or render-to-installed above 100 ms prevents release and attempts baseline. A committed late candidate remains behind the unconditional guard even if cleanup never runs. Killing `nft` is not treated as proof the transaction did not commit. | [apply.go:181–203](../../../node/enforce/apply.go); [executor.go:44–53,87–96](../../../node/enforce/executor.go) |
| G7 verification | Main releases the guard **before** live verification. Rules, generation, admission lifetimes, exact isolation-set coverage, counters, class assignments, hooks, foreign rules/offload and tc are checked. Failure then closes via baseline/quarantine; it is not a promise that an unverified candidate never forwarded. | [apply.go:198–234](../../../node/enforce/apply.go); [verify.go:14–120,579–599,832–865](../../../node/enforce/verify.go); [verify_tc.go:130–176](../../../node/enforce/verify_tc.go); [deadline_read.go:112–160](../../../node/enforce/deadline_read.go) |
| G8 rollback | Rollback installs the node-wide baseline, never the old policy. It removes the guard in the **same** transaction as closure, preserves named counters and only removes owned tc roots when foreign-filter checks permit. Rollback can fail; several callers discard its error. No finite recovery/outage bound or infallible cleanup is established. | [apply.go:301–349](../../../node/enforce/apply.go); [render_nft.go:275–294,335–394](../../../node/enforce/render_nft.go); [engine.go:1286–1294](../../../node/agent/engine.go) |
| G9 ISO01/ISO02 | One complete old/new nft transaction; no intermediate flush-visible allow-all state. During the committed guard interval both member directions drop. Otherwise current source admission, deny/host/resolver rules, IPv6 blocking, input drops and exact interface bindings remain. Foreign tc can bypass nft, so it requires link quarantine, not confidence in the guard alone. | [render_nft.go:28–30,125–225](../../../node/enforce/render_nft.go); [quarantine.go:49–104](../../../node/enforce/quarantine.go) |
| G10 accounting/QoS/control | Named counters are retained; denied traffic is not ordinary forwarded usage. tc HTB/fq_codel is prepared before candidate installation. tc is a sequential batch, **not atomic with nft or across commands**; an early crash can leave partial shaping. Host control remains independent of the member interface. | [render_nft.go:55–75,141–179,191–225](../../../node/enforce/render_nft.go); [render_tc.go:28–80](../../../node/enforce/render_tc.go); [executor.go:44–53](../../../node/enforce/executor.go); [apply.go:149–154](../../../node/enforce/apply.go) |

For G2, a successfully budgeted invocation places the local cutoff no later than
`InstalledBy <= R+100 ms`; this is conditional on reaching that stage, **not** a
bound from API receipt, Apply start or central revocation. If the process dies
before installation, the hard fallback is the previous kernel expiry `E_old <=
D_old`. With a running watchdog, successful closure may happen earlier. Its
stale threshold defaults to 30 seconds and configured check interval to 10
seconds, but lock contention, scheduling and failed commands prevent a universal
40-second bound. [watch.go:83–90,120–208,225–242](../../../node/watchdog/watch.go)
uses a try-lock, a five-second context and retries; a context is not a kernel
commit deadline. End-to-end 60-second online convergence still needs central and
agent evidence. Offline revocation remains lease-limited (§16.3).

The agent calls `SystemPlane.Apply` → `enforce.Apply`; it does **not** call
`releaseDeadlineGuard` independently. It verifies again before acknowledging and
persists the actual applied model separately from desired state. See
[plane.go:92–114](../../../node/agent/plane.go),
[engine.go:1078–1161](../../../node/agent/engine.go), and periodic verification /
expiry withdrawal in [serving.go:24–48,90–122](../../../node/agent/serving.go).
A verified conntrack-delete error deliberately retains the new restrictive
ruleset and queues retries; exhausting the agent's retry bound withdraws serving
([engine.go:1162–1175](../../../node/agent/engine.go)).

## Crash cuts on the current path

| Cut, including a killed command with ambiguous acknowledgement | Kernel state that may remain; recovery obligation |
|---|---|
| Before validation/load/initial render finishes | Previous verified generation or baseline. No fresh timeout was installed. Agent/watchdog withdrawal may not yet have run. |
| During each tc command; after shape, before final render | Previous nft policy with partial/new tc. Existing deadlines still count down; no universal atomic-QoS promise. A surviving caller attempts baseline; foreign attachments use quarantine. |
| Final render, before candidate transaction commits | Previous policy. Rejected transaction has no partial nft effects. |
| Candidate transaction commits; process dies before return/budget check | Candidate plus unconditional guard: all member traffic closed, even if its relative TTL is overlong. |
| Budget accepted, before/during release transaction | Guard still closes until deletion commits. A lost acknowledgement can mean guard or candidate serving; timeout remains safe in either state. |
| Release commits; death before/during verifier | New candidate may serve, with safe deadlines; no verified ACK. Watchdog/restart must examine evidence, not infer success. |
| Verifier fails; death before/during rollback | If baseline commits, all closed. If it fails or never executes, candidate may still serve until existing expiry/independent closure; main does not guarantee magical cleanup. |
| Baseline commits, during tc cleanup | Generation 0, no member forwarding; residual tc is not authorization. Never restore old admission to regain availability. |
| Verification succeeds, during conntrack cleanup / durable ACK | New rules enforce revocation even with old conntrack entries. Same-boot adoption requires full live checks; a reboot or lost anchor requires closure/fresh authority. |

These are code-level limits, not newly filed vulnerabilities or new acceptance
waivers. The [D-135 explanation](../../node/followup29-install-contention-2026-10-31.md)
already distinguishes attempted rollback from proven closure.

## Follow-up 30 cost, with its qualification retained

The [outcome](../G2-followup30-evidence/guard-outcome.md) and
[drop ledger](../G2-followup30-evidence/drop-ledger.md) record 100 reloads,
101 verified Applies, 118 successful TCP reads and 381 correlated TCP segments.
Seven missing node egresses have seven unique guard-drop matches, all member →
responder inside install-command brackets: five original data segments, one
retry and one ACK, generations 1002/1018/1019/1041/1069/1070/1093. No reverse
probe TCP guard drop was observed, although the code drops that direction too.

| Observed bracket, 101 Applies | p50 ms | p95 ms | max ms |
|---|---:|---:|---:|
| Install command | 13.684706 | 24.470368 | 69.025183 |
| Release command | 13.153761 | 22.889335 | 35.959668 |
| Install start → release end | 28.471576 | 43.438670 | 94.665010 |
| Installed-by → release start | 0.005087 | 0.008232 | 0.014273 |
| Producer outer bracket | 28.493114 | 43.459978 | 94.684882 |

Actual all-member closure is bounded between install-end→release-start and
install-start→release-end, not equal to either. Loss recovery can be much longer:
cycle 26 loses its original and +208.154 ms retry, succeeds at +416.030 ms, and
reads at **416.739885 ms**. The +138 script bytes per Apply, 18 NFLOG records,
1,044 IP bytes / 9,975 derived bytes, 62.928116/57.258627 ms collector start/stop,
and 5.367775832 seconds of outside-Apply snapshots are observed diagnostic cost,
not isolated incremental overhead. No production logging is proposed.

The guard window failed foreign-CPU qualification at 2.478498 cores; the full
suite failed it in eleven intervals, maximum 4.310365. Functional passes remain
distinct from qualification. [PR #418's orchestrator reading](https://github.com/TED-H-com/Freedom-VPN/pull/418)
establishes the guard mechanism; B1's **1,501.407973 ms** failed read still has no
packet/timer trace and remains FAIL and unexplained. Neither this design nor a
future successful comparison may rewrite that result.
