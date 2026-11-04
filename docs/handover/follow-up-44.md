# Follow-up 44 — #457–#463; lock wait merged, candidate proof still open

Verified against `main` **5c21e670596f5b0b7029f6a02648305a24f0f22a**, the
final roadmap commit through **#463**, on **2026-11-04 EST**. Follow-up 43's
cutoff was **#456 / 04d31ea5**. This range contains exactly **#457, #458, #459,
#460, #461, #462 and #463**: **seven merges and seven final roadmap rows**.
No unbriefed PR merged inside the range. Local main and the read-only public
PR listing ended at #463 when checked at start.

**Running at briefing, status only:** secops **18** (SECOPS-F17-01 under
D-160, one live AT29 run); members **22** (D-098 archive of run **5715939e**,
F21-AT08, F21-D146, and one assembled run measuring AT02 on **#461**);
netlab **36** (D-161 in the D-120 analyzer, launched on netlab 35's merge);
devstack **35** (F30-01's candidate path on updater 25's lock wait, launched
on #463). No delivered or merged PR for these follow-ups was present at
start. Any later delivery is **after the cutoff**; no later result enters
this cut. No date is promised.

**D-098 / D-109 / D-132:** `TestHandover_Followup44Claims` binds the full
narrative, seven final roadmap rows and PR excerpts, **25 frozen inputs**
(nine #457 handover inputs and sixteen producer records), **30 Go test
names on main**, and **16 frozen source readings**. All lane-document quotes
and citations below use committed frozen copies. No claim reads a living
members, isolation, secops or usage evidence root. All **37 earlier claims
functions** retain their assertions; follow-up 43's rewritten inputs now use
the frozen #457 bytes.

## Handover 43 (#457): verified counts and routing correction

[PR #457](https://github.com/TED-H-com/Freedom-VPN/pull/457) records the
previous **#446–#456 cut, up to 04d31ea5**. Its actual claims test and manifests
pin **16 subtests**, **eleven final roadmap rows and PR excerpts**, **35 frozen
inputs**, **60 Go test declarations**, **16 archived code readings**, and
review **r29's 180-line raw archive with two byte-identical imported proof
files**. Its **36 earlier claims functions** preserve their assertions with
seven input-expression redirects. The register has **60 decisions and 124
findings rows**: **11 updated, 11 added, 102 byte-identical**. Those are
#457's counts, not this cut's.

The session account is attributed to #457's body: **codex gpt-6-astra**,
launched **01:02 EST** on **2026-11-04**, delivered **01:32**; **6 commits,
69 owned files, +74,808/−328**. Docs, claims, snapshots, r29 and digest values
only; no reservation or Docker.

**Routing correction recorded now:** members F21-AT08 and F21-D146 were
routed to devstack at that cut. After its cutoff, the orchestrator re-routed
both to **members 22**, because the code that measures them is members' own.
Both register rows now say members; their evidence and open status are unchanged.

## F30-01: devstack 34 (#458), then updater 25 (#463)

[PR #458](https://github.com/TED-H-com/Freedom-VPN/pull/458) and the
[frozen analysis](../../tests/handover/testdata/history/follow-up-44/docs_e2e_runs_2026-11-04_followup34_analysis.md.txt)
record acceptance run **af4c130b: 31 pass, 2 blocked, 0 fail**, one attempt.
`lease-expiry` passed and left the nightly list; `member-ipv6`'s entry stays.
`update-good` was newly blocked: **`fvpn-updater run` exited 3 (`ExitBusy`)
before it started a candidate**. Lock samples show the running agent holding
`update.lock` for **0.43 to 0.86 s** at each new generation, **every 5 s**;
the updater gave up after **80 ms**. The failed acquisition's instant was not
captured. The candidate's deferred boot recovery under D-155 was never
exercised. The run is **confounded under D-133**; no timing comparison is made.

**r29 L-2 is answered:** its stale citations left with the removed
`lease-expiry` entry. The `publisherTrace32` comment now describes the API
after **#443**, ordinary 412 with `details.current_resource_version`; the
historical two-shape predicate is unchanged. The nightly verdict on this run
still fails on unexpected blocked `update-good`, with no new quarantine entry.
`TestD144_CurrentExpectedBlockedListAfterFollowup34` pins the remaining list.

[PR #463](https://github.com/TED-H-com/Freedom-VPN/pull/463) and the
[frozen updater note](../../tests/handover/testdata/history/follow-up-44/docs_updates_node-followup25-2026-11-04.md.txt)
record updater 25 under **D-158 as revised**:

- `run`, `seed`, `apply` and standalone `flush` wait for a live holder within
  **one 10 s budget**, polling every **20 ms**. Past the budget: **exit 3**,
  unchanged line `fvpn-updater: updater: UPDATE_BUSY: lock`.
- `recover` and standalone `Retain` keep **five attempts 20 ms apart**.
  Nested calls reuse ownership. Stale details retain **exit 1 after five
  attempts counted from the first stale answer**, including after prior live
  contention; unreadable metadata stays a live holder. Common lock consumers
  are unchanged. Cancellation ends the wait.
- One JSON record, **`update lock wait`**, only after a missed first attempt,
  with `command`, `waited_ms`, `acquired` and `budget_ms`. An uncontended
  command adds none; acquisition does not prove command or candidate success.
- Real second-holder proofs: **600 ms** contention yielded **exit 3 at about
  81 ms before**, and **exit 0 at about 617 ms after**, for all four commands
  in the PR's summary. The dated table gives run/seed/apply/flush precisely
  as **617.335 / 617.479 / 664.315 / 615.934 ms**. The CLI's real default
  budget gives **exit 3 at 10.006 s**, never early, with the exact error line.
  `TestD158_CommandsProceedAfter600msHolder`,
  `TestD158_StaleWindowStartsAtFirstStaleAnswer` and
  `TestD158_CLIWaitRecordAndExactErrorLine` are on main.

**F30-01 stays open.** The orchestrator refined its closing condition after
#458: a `lease-expiry` pass without a real candidate update cannot close it.
One acceptance run must show **`update-good` starts the candidate; its boot
recovery defers while its parent holds `update.lock`, then clears; the
candidate serves members; `lease-expiry` passes afterward**. Devstack 35 runs
for this at cutoff. The merged code and unit proofs do not supply that run.

## Isolation 22 (#459): member window answered; F22-01 opened

[PR #459](https://github.com/TED-H-com/Freedom-VPN/pull/459) and the
[frozen isolation note](../../tests/handover/testdata/history/follow-up-44/docs_e2e_at_isolation_followup22-2026-11-04.md.txt)
answer **Isolation F21-01** with a **fresh-renewal anchor**. The schedule
proofs are offline: original-time replay, jitter and early-renewal cases reach
the window, while a missing window still blocks. The existing bounds, five
probes and six serving reads remain. Main has
`TestAT13_MemberScheduleFollowup21RenewalReplay` and
`TestAT13_MemberScheduleMissingWindowStillBlocks`.

Assembled run **7b0db5f2: 27 pass, 2 blocked, 0 fail**. **r28 M-1's first live
pass:** the member deadline came **26.3 s before** the node's accepted lease
ended; all **five requests after it timed out under a serving node**. The
short whole-node expiry again **passes D-140 and qualifies under D-142**.
The original run 21 stays blocked; run 20's historical pass remains unqualified
as serving-node expiry evidence. The two operator matrices remain blocked.

**Node writes: 15 commands in 16 attempts.** One publication PUT got **412**;
its retry with the fresh ETag and same key/body got **202**, with read-back
retained. **F22-01 is new and open, owned by netlab:** the included
`TestAT10_AT13_ReloadRevokeBaseline` failed D-120 clause (iv), one TCP read
timeout at **sequence 101**. Everything else in the campaign held: all **101
Applies verified**, no UDP loss and no install over 100 ms. It was routed to
netlab 35; the historical failure stays failed.

## Secops 17 (#460): fresh control-loss lease; AT29 still failed

[PR #460](https://github.com/TED-H-com/Freedom-VPN/pull/460) and the
[frozen secops note](../../tests/handover/testdata/history/follow-up-44/docs_e2e_at_secops_2026-11-04-follow-up-17.md.txt)
answer **SECOPS-F16-01**: the fleet cap is now set **before admission**, so
control loss gets a fresh **60 s lease by construction**. Offline
`TestAT29_PostCapLeaseSetup17` retains rejection of the old lease and rejects
invalid setup. In **b2de46c8**, the case passed on lease **bef669a3**, with
**one agent identity throughout** and **both transports continuous to about
D − 1 s** (last observed replies D − 0.94 s). The run, passing rows included,
is **confounded under D-133**.

**AT29 stays failed: SECOPS-F17-01 is new and open.** In the old killed-agent
`lease_expiry` case, **two 60 s admission holds inside the case** delayed the
fault **60.7 s past the recorded deadline**. The node renewed meanwhile, so
the cutoff predicate failed. This is **the harness's timing, not a product
deadline violation**. Secops 18 runs for it at cutoff under D-160. The other
sub-cases pass; `guest_clock_rollback` stays blocked as the operator's. No
historical failure is relabeled.

## Agent-core 50 (#461): D-157 merged; assembled AT02 measurement pending

[PR #461](https://github.com/TED-H-com/Freedom-VPN/pull/461) and the
[frozen agent note](../../tests/handover/testdata/history/follow-up-44/docs_node_followup50-2026-11-04.md.txt)
record **one renewal right after central accepts a verified closed recovery's
clean report**, instead of D-067's next slot. It happens **at most once per
closed recovery**, is **consumed on dispatch**, and a **baseline or replacement
Apply discards it**. D-067's cadence is unchanged otherwise, including after
an empty answer. One request per Step and Retry-After remain. No serving
before a populated matching lease and verified Apply is authorized.

**The orchestrator approved F48's proof change:** the intervals after the
closed recovery's request go from **5, 10 and 20 s** to **about 0, 10 and 20 s**.
Every other F48 invariant stays, including deny-all, no redundant baseline,
no serving before a populated lease, doubling and Retry-After. Main has
`TestF48_ClosedRecoveryWaitKeepsDenyAllAndRenewalRate` and
`TestD157_AT02_ClosedRecoveryRenewsAfterAcceptedReadyReport`.

The ordinary lab passed **13 of 13**; its fake issuer does not model central's
readiness transaction. **AT02's deny-all after a closed recovery stays open**
until members 22's assembled run measures it on #461. No live run has measured
the change yet at this cutoff; members 21's **4.968 s and 241 losses** remain
historical evidence, with no zero-loss or date promise.

## Netlab 35 (#462): timing-compatible, not proven

[PR #462](https://github.com/TED-H-com/Freedom-VPN/pull/462) and the
[frozen netlab note](../../tests/handover/testdata/history/follow-up-44/docs_poc_G2-followup35-2026-11-04.md.txt)
record F22-01's and F27-01/B1's diagnosis as **timing-compatible, not proven**.
Measured retry offsets, transplanted onto those historical sends, meet later
install brackets. **Neither run retained TCP packets**, so attribution stays
open. **D-159's condition is false: no positive minimum Apply spacing** exists.
Its three paths are **binding-deadline wake-ups**, **zero remaining Step delay**,
and **a closed recovery's second Apply in the same Step**. The campaign's
cadence is unchanged.

The three collectors now retain **combined TCP/UDP header captures** at origin,
node and responder, with **checked UDP projections** for the unchanged analyzer.
TCP echoes record **match, EOF, stale or mismatch**. **Five standalone
attempts** stay recorded: the first failed on the new splitter's handling of a
trailing blank separator and stays failed; four passed with **zero TCP
timeouts**, **TCP p95 205 to 209 ms** (rounded). **Attempts 3 to 5 are
confounded**. The full netlab gate and **all 13 agent scenarios pass**; these
are functional results, with no historical attribution or timing-comparison
claim. The orchestrator accepted attempt 1 among the five and required no
sixth run. The [operator limitation](operator-guide.md#tcp-reload-guard) is
quoted exactly from netlab's hand-off.

## Decisions and findings at this cut

The [decisions register](README.md#follow-up-44-decisions-register-2026-11-04-est)
has **64 decisions**, from **60**, adding only **D-158, D-159, D-160 and D-161**.
All four are **the orchestrator's provisional decisions; the owner may
overturn them**. Main had no row after D-157, and **every earlier row is
byte-identical**, including its historical status wording.

- **D-158 — 01:4x EST, revised 02:2x, 2026-11-04:** the four commands' 10 s /
  20 ms wait, prompt recover/Retain, nested reuse, first-stale five attempts,
  unreadable-as-live and one missed-first-attempt record. Other common lock
  consumers are unchanged. The agent's ordinary hold is bounded by its Step's
  local deadline, about **10 s**; the original **2 s** was unsupported by code.
- **D-159 — 01:5x EST:** slow D-120 reloads to the agent's minimum spacing only
  if cadence resonates with TCP retries and that minimum is at least about
  **1 s**. Otherwise nothing changes. #462 found no minimum; condition false.
- **D-160 — 02:2x EST:** no host-admission wait within a timed sub-case, from
  reading the deadline it measures through its last cutoff sample. Check
  admission before it. Overlap that still happens is D-133 confounded, never
  delayed. Source: SECOPS-F17-01.
- **D-161 — 03:0x EST:** D-120's **“zero TCP timeouts”** becomes **“zero TCP
  timeouts not attributed to the guard”**. A timeout passes as guard loss only
  when complete, checked same-run captures prove **every lost transmission of
  the stalled bytes** entered the node **inside, or on the boundary of, exactly
  one verified outer guard bracket**, **within both 100 ms bounds**, and never
  left it; later delivery is **in order on the same connection**, which was
  **never reset or closed**. Guard-attributed timeouts count toward **“≤ 5 losses
  per 100 reloads”**. Every campaign reports **its longest TCP stall and its
  guard-attributed TCP drops**. All other D-120 predicates stand; historical
  failures stay failed. D-145 keeps the guard; TCP retransmission can hit a
  later guard; **D-120 is a test-acceptance definition, not a product guarantee**.
  Netlab 36 is implementing it, running at cutoff; **nothing is merged**.

The [findings register](README.md#findings-register-at-the-follow-up-44-cutoff)
has **126 rows: 10 updated, 2 added, 114 byte-identical**. The new rows are
**Isolation F22-01 (netlab, open)** and **SECOPS-F17-01 (secops, open)**.
F30-01 and AT02 remain open with the precise live closing conditions above;
all other open rows and every historical failure retain their status.

## Operator and member entrypoints

Seven entrypoints point here. The operator guide keeps F30-01 and AT02's
anchors, adds updater 25's exact verified sentence immediately after updater
24's exit-status paragraph, and gives the TCP guard limitation its own anchor.
Nothing else in this cut changes what an operator or member sees. Scope-review
changes only digest values for changed owned documents; no product, roadmap,
contract or other owner's file changes.

### Requests to orchestrator

**2026-11-04 EST:** carry devstack 35's full F30-01 candidate-path acceptance;
members 22's run-5715939e archive, F21-AT08, F21-D146 and assembled AT02
measurement on #461; secops 18's D-160 harness correction and one live AT29
run; netlab 36's D-161 analyzer work. These are status-only hand-offs, with no
later results imported. No new outside-owned pin needs changing; no item is
stopped. All other open operator/resource requests remain unchanged.
