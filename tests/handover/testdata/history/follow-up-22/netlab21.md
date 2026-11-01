# Follow-up 21 — transport attribution and enforcement assessment, 2026-10-31

**The IPv6-underlay lab reproduces reverse large-datagram loss:** 10,010 of
64,064 validated bytes in 12,012.403 ms; 19 outer IPv6 datagrams of 1,360 bytes
leave the tunnel endpoint and none arrives at the member. All 36 of 1,152
bytes arrive. No ICMPv6 Packet Too Big (PTB) appears. This supplies a common
mechanism for the two native failures; the uncaptured members row remains an
inference. No production enforcement, tunnel MTU or MSS change is made.

The 100 ms expiry reserve is intentional. A failed Apply can replace forwarding
for every member with baseline, including an admitted sibling; that outage is
not bounded by a completed guard. The historical 52.596424 ms outer maximum
is an observation, not a demonstrated ceiling. Every loss remains counted.

Scope: EXIT02 / APP01 / NFR01, ISO02 / ENT02 / SEC01, SEC02;
§8.2, §8.3, §16.3 and §19 AT01/AT02/AT09/AT10/AT13/AT29.
This is Linux amd64 mechanism evidence. Linux arm64 cross-build passes; no
physical arm64 or native-platform acceptance is claimed.

## D-098 retention and source identity

Before new evidence, commit `db835bd4` archived **82 files byte-identically**
under [G2-followup20-evidence](G2-followup20-evidence/archive.json), with lengths
and SHA-256 pins. Follow-up 20 had no new runtime run; this preserves its
corrected source pins, attribution notes and all retained follow-up 19 data.
Its nested source pins still name immutable history/run archives and verify.
No original pin-tool or digest was changed.

New [input snapshots](G2-followup21-evidence/inputs/archive.json) preserve
14 consulted files byte-identically from main `50834d996b8d636dfa8a0765567a759ef0903edb`.
Actual pins resolve to these owned immutable copies. `source_path` identifies
provenance; it does not make a living file a pin target. Any actual pin outside
`docs/poc/` must name the existing immutable history/run convention.

The final live harness is commit `ece60010`; production `node/enforce/*` is
unchanged from that main. Full public data is under
[G2-followup21-evidence](G2-followup21-evidence/retention-pins.json).
[Private pcap receipts](G2-followup21-evidence/raw-capture-receipts.json) identify
original disk files; they are not retained-tree pins. Raw pcaps stay in this
worktree's ignored `.agent/f21-private/`; no keys, packet payloads or native
peer IDs are published. Public exports retain headers and opaque packet hashes.

## F8-02: interpretation before measurement

Devstack 20's [IPv4 capture](G2-followup21-evidence/inputs/devstack-README.md)
returns 10,010 / 64,064 bytes in 12,011 ms. Missing TCP range
`[1784148912,1784150140)` is retained at the origin and node tunnel egress,
including retransmissions. The member ACKs that gap and SACKs later bytes.
All 19 reverse outer IPv4 datagrams of length 1,340 are absent at the member;
all 17 of 1,132 arrive. Neither fragmentation-needed nor PTB appears.
Per-I/O instrumentation did not remove the failure.

Members 11's [IPv6-selected row](G2-followup21-evidence/inputs/members-F11-F8-02-mtu1280.json)
returns 13,013 / 64,064 bytes in 12,010 ms, direct P2P IPv6 before and after,
with **IPv4 payload**. Write/read are 64,108 / 13,081, 13 complete lines,
zero partial bytes, upload TCP bytes ACKed 64,109, notsent/retrans zero;
device deltas are +66,216 / +38,241. All three 1500 rows pass.
These client send-side counters prove upload progress, not delivery of the
reverse TCP stream. They do not require a second failure mechanism.

Holding the encrypted payload constant, IPv6's header adds 20 bytes:
**1,340 → 1,360**, exceeding the 1,280 link by **80 bytes**.
The smaller IPv4 packet maps **1,132 → 1,152**, below the link MTU.
Thus reverse large-datagram loss explains both observations consistently.
No per-datagram capture exists for that members row, so consistency is not
proof of its exact dropped packets, size distribution or kernel drop reason.
IPv6Block applies to inner IPv6 forwarding; it does not explain IPv4 carried
over an IPv6 underlay. Neither production ctv6 rule was changed.

## F8-02: new IPv6 capture

The fresh isolated kernel WireGuard fixture uses a direct IPv6 P2P path:
`fv-fwd/fv-fa1` (sender MTU 1500) ↔ `fv-a1/fv-a1u` (member MTU 1280).
Both tunnel interfaces are 1280; node transit, exit and origin links remain
1500. This models the native fixture's asymmetric receive-link mutation,
not a routed bottleneck with a router emitting PTB. Only these container
namespaces are changed. There is no NetBird ICE negotiation in this fixture.

The exact application criterion is 64 complete validated 1,001-byte lines in
12 seconds, with `MEMBER`/`OK` greeting and `ECHO ` framing. The origin uses
TCP_NODELAY and one write per echo. The selected endpoint is IPv6; all inner
TCP is IPv4. A 1500 control precedes lowering only the member underlay, with
no extra reconnect after lowering it, as in the broader native capture.
Bound: **three 1280 attempts, stop at first failure**. Attempt **1** failed;
attempts 2–3 were not run.

| Row | Validated / expected | Elapsed ms | Client stream read / write | Complete / partial lines |
|---|---:|---:|---:|---:|
| IPv6 underlay, member MTU 1500 | 64,064 / 64,064 | 4.479 | 64,387 / 64,108 | 64 / 0 bytes |
| IPv6 underlay, member MTU 1280 | 10,010 / 64,064 | 12,012.403 | 10,063 / 64,108 | 10 / 0 bytes |

In this lab the origin application reads all 64,108 and writes all 64,387
stream bytes in both rows. Successful writes mean kernel acceptance, not
remote receipt. This differs from devstack's blocked origin write without
changing the observed reverse-path loss. Neither row is an exact native replay.

[Datagram matching](G2-followup21-evidence/lab/TestF21_AT09_IPv6Underlay/attempt-1-datagrams.json)
compares full captured outer IPv6 packet SHA-256 values as multisets across
the direct link, including fragments. There are no unmatched receiver packets.

| Direction / outer IP length | Sent | Arrived at opposite underlay capture | Absent |
|---|---:|---:|---:|
| Reverse 1360 | 19 | 0 | **19** |
| Reverse 1152 | 36 | 36 | 0 |
| Reverse 928 | 1 | 1 | 0 |
| Reverse 144 | 11 | 11 | 0 |
| Forward 1280, first IPv6 fragment | 52 | 52 | 0 |
| Forward 136, last IPv6 fragment | 52 | 52 | 0 |
| Forward 144 / 176 / 624 | 17 / 1 / 1 | 17 / 1 / 1 | 0 |

At 1500, all nine reverse 1360-byte datagrams and all 53 of 1152 bytes arrive;
the upload also has zero absent datagrams. In the failed row the missing
reverse packets are unfragmented. The successful upload has 52 source-fragment
pairs. Both endpoints therefore observe fragmentation in one direction but
no successful size adaptation in the other. **PTB count = 0 at all four points.**

The first missing TCP range is **`[2773444484,2773445712)`**, 1,228 bytes.
Origin, enforcement-node return egress and tunnel egress retain that range
and its retransmissions; member tunnel ingress never has it. The final member
ACK stays at 2773444484 with SACK `[2773445712,2773472652)`.
[Analysis](G2-followup21-evidence/analysis.json) retains the actual matching
header records; the numbered missing outer packets are in the datagram file.
This distinguishes network loss from a reader that received every complete echo.

Failure localization: **after tunnel-endpoint underlay egress, before member
underlay ingress**. The enforcement node already forwarded the missing data.
The direct veth pair has asymmetric MTUs, making receive-link rejection a
supported inference; no kernel drop-reason trace was taken. This result does
not attribute loss to nft, tc, the central service, or an IPv6 overlay block.

| Capture (failed row) | Decoded / tcpdump captured | Kernel drops |
|---|---:|---:|
| Member: underlay + tunnel | 295 / 295 | 0 |
| Tunnel endpoint: underlay + tunnel + transit | 431 / 431 | 0 |
| Enforcement node: transit + uplink | 279 / 279 | 0 |
| Origin | 138 / 138 | 0 |

All records parse without truncation. Per-interface indices and MTUs are
retained in `attempt-1-links.json`. Capture readiness precedes the client;
SIGINT/flush follows its completion by 200 ms. Pcap uses UTC microseconds;
these observations are not hardware timestamps. Offload settings come from
the existing topology; no capture length alone is treated as wire loss.

The **initial capture attempt is retained** under `initial-capture-rejected/`.
Its 1500 control passed, but kernel capture drops (member 87, tunnel 34,
node 14, origin 14) prevented attribution and the 1280 attempt was never reached.
The rollback capture also failed completeness. A fresh topology then used
32 MiB tcpdump buffers and a 65,535-byte snap length. No application, enforcement
or MTU policy changed between attempts. Zero-drop checks stayed mandatory.

## Assessment (a): expiry reserve and proposed acceptance wording

Code references:

- [`model.go:89`](../../node/enforce/model.go#L89): `DeadlineGranularity = 1ms`;
  [`model.go:94`](../../node/enforce/model.go#L94): `DeadlineInstallBound = 100ms`,
  explicitly an enforced budget, **not a scheduler SLA**.
- [`deadline.go:36`](../../node/enforce/deadline.go#L36): take the earliest
  service/node/offline/binding/monotonic ceiling, subtract the reserve,
  truncate toward earlier to 1 ms; no positive representable remainder denies.
- [`apply.go:153`](../../node/enforce/apply.go#L153): resample monotonic time
  after ownership reads and tc, advance the original model without resampling
  wall time, reserve 100 ms, then render the final candidate.
- [`deadline_install.go:42`](../../node/enforce/deadline_install.go#L42): candidate
  and drop guard are installed in one atomic transaction. The guard drops both
  directions on member ingress at filter priority −151 (before ordinary −150).
- [`apply.go:179`](../../node/enforce/apply.go#L179): bound the command and check
  render→installed ≤100 ms before release. Release does not restart timeouts;
  verification clears the reserve to avoid subtracting it twice.

Let R be the final render sample, I the actual kernel insertion instant,
D the effective absolute deadline, B=100 ms, q=1 ms. For an admitted element,
`timeout = floor((D − R − B)/q) × q`, and `expiry = I + timeout`.
Because a released installation establishes `R ≤ I ≤ installed_by ≤ R+B`,
the designed conservative cutoff lies within **[D−101 ms, D]**. The early
amount is about `100 ms − (I−R)` plus less than 1 ms truncation. It is not a
new 100 ms grace after D. An almost-expired element may be omitted outright.
The reserve protects against unknown insertion time, scheduling delay or
process death before the budget check; removing it can extend a relative
kernel timeout beyond the signed deadline.

Agent-core A/B/A/B all retain 97 positives then 15 drops and two pre-D drops;
B2's last positive is D−100.373132 ms and first dropped ingress D−63.512896 ms.
The corrected independent isolation 10 sample is **161 / 41**, last positive
**D−99.655342 ms**, first dropped ingress **D−63.564818 ms**, **one** pre-D
loss. Earlier prompt approximations are superseded by these retained numbers.
Current netlab's 100 ms cadence records **26 / 7**, last positive
D−100.729554 ms and first dropped ingress D−0.795 ms: one pre-D loss.
That coarse cadence does not locate the exact expiry instant within the
101 ms conservative window.
All pass unchanged D-104; none proves literal zero pre-D loss.

**Proposal for the orchestrator to record, not an adopted predicate change:**

> AT13/AT29 expiry: retain the original effective absolute deadline D. Require
> independent kernel denial at/after D and no resurrection after the first
> expiry loss without a newly valid authorization. With a completed bounded
> installation, the conservative expiry transition may start in [D−101 ms,D]
> because of the documented 100 ms insertion reserve and 1 ms truncation.
> Require qualifying positives through the D−101 ms boundary, then continuous
> loss; an earlier unexplained loss, later resumed positive, or admission
> at/after D fails expiry/continuity. Retain every lost ingress, its send/ingress
> bracket and its completed-guard overlap. Losses in the conservative window
> still count as pre-D losses and fail a separately requested literal zero
> pre-D-loss assertion. Do not subtract them from renewal/availability counts.

The existing D-104 diagnostic's first-drop observation bound D+100 ms remains
an observation bound; it never permits positives after D. Its existing
sample-count/order checks and strict no-post-D-egress test are unchanged.
No tolerance, D-120 extension or new acceptance pass is introduced.

## Assessment (b): guard cost and installation overrun

Nearest-rank **p50 / p90 / p99 / max**, milliseconds; completed guards only:

| Population | n | Outer guard | Install command |
|---|---:|---|---|
| Isolation 10 current | 968 | 31.085 / 40.529 / 48.851 / 52.928 | 16.110 / 22.319 / 27.699 / 30.620 |
| Isolation 10 second attempt | 931 | 30.750 / 38.828 / 46.306 / 49.845 | 15.779 / 21.513 / 26.902 / 30.439 |
| Isolation 10 first attempt | 678 | 31.635 / 41.828 / 51.362 / 57.630 | 16.460 / 22.524 / 28.102 / 30.792 |
| Follow-up 21 reload | 101 | 32.480 / 41.400 / 45.396 / 52.675 | 16.698 / 21.783 / 28.473 / 30.818 |
| Follow-up 21 rollback diagnostic | 7 | 30.168 / 52.168 / 52.168 / 52.168 | 19.276 / 30.301 / 30.301 / 30.301 |

Raw per-Apply brackets are retained, including the three diagnostic failed
verifiers whose guards completed before baseline. Isolation's 49 failed
installs are **not** completed guards and are not included in its percentiles.
Current reload passes unchanged D-120 with two retained boundary UDP losses
(packets 14 and 33); both next datagrams succeed. It does not erase
isolation's 30 / 1,396 renewal losses, all overlapping completed guards.

The controlled agent-core D-133 A/B/A/B remains the causal comparison; this
new diagnostic is not another A/B. A's live maxima 40.355 / 41.614 ms and B's
57.924611 / 52.020393 ms remain visible. B1 Apply 137's outer bracket is
29.006529 ms install + 0.002777 ms gap + 28.915305 ms release. There is no
agent work hidden inside that gap. Current reload's maximum inter-command gap
is 0.007671 ms. The original 52.596424 ms maximum was already exceeded by the
independent isolation first attempt (57.630034 ms), as well as B1.

**Assessment: retain the current mechanism and accept these distributions as
its measured cost; do not assert a fixed 52.596424 ms ceiling.** A guaranteed
ceiling at that historical sample is unsupported. This recommendation does
not authorize lost-packet tolerance or certify a future maximum.

The release command already consists of one `delete chain` operation. Install
is one atomic candidate+guard batch. Merging release into that batch would
remove the post-commit elapsed-time decision: a late commit or agent death
could expose an overlong allowance. Installing the guard in an earlier separate
batch increases the closed period. Opening between stages forfeits the same
safety property. A persistent netlink executor or prebuilt shadow-object design
might reduce command launch/parse overhead, but is an architectural change
requiring measured benefit and crash/late-commit proof. There is no demonstrated
safe local command shortening to apply in this task.

For an overrun, `apply.go:189` **never calls release**. It attempts baseline and
returns failure. Baseline atomically replaces serving rules with generation 0
and may remove the now-redundant guard in that same closed transaction
(`render_nft.go:262`); this is not an opening release. If baseline cannot be
installed, an installed guard remains the fail-closed barrier. There is no
claim that a 100 ms context forcibly bounds kernel scheduling latency.
The bound remains intentional under CPU contention: availability is sacrificed
rather than extending authorization. Isolation first-attempt Apply 712's
215.890796 ms render→installed (population max 340.350 ms), no release and
`apply_rollback` baseline are consistent with this design. Agent-core 29 owns
its D-133 contention attribution; the earlier ownership-only escalation was
withdrawn. No production change means a new D-133 A/B/A/B is not triggered here.

## F11-03: admitted sibling during rollback

**Yes.** Verification failure invokes baseline for the node, not just the
revoked device (`apply.go:209–221`, `installBaselineModel`). The admitted
sibling's allowset/rules are withdrawn too. They stay closed until a subsequent
successful Apply; there is no automatic 100 ms or one-guard recovery ceiling.
A post-verify conntrack-delete failure is deliberately different: it retains
the verified serving rules (`apply.go:234`), so do not conflate these paths.

The fixture holds A1 admitted while A2 is absent, runs 5 ms A1 probes, injects
one post-release verifier read failure per trial, and executes real baseline
and recovery transactions. The fault is TEST ONLY; it is not a fabricated
historical cause. A **200 ms explicit baseline observation hold** precedes
retry; the rest is actual command/read/recovery work.

| Trial | Failed Apply completed outer guard ms | Baseline→recovery outer bracket ms | Definitely closed interval ms | Lost ingresses within baseline/recovery bracket |
|---|---:|---:|---:|---:|
| 1 | 34.435027 | 304.815081 | 272.051181 | 56 |
| 2 | 28.329661 | 280.395264 | 253.222256 | 54 |
| 3 | 24.020724 | 285.296508 | 257.972529 | 55 |

Outer outage brackets run from baseline command start to recovery guard-release
end. The definitely-closed interval runs from baseline command end to recovery
release start. Neither endpoint is an invented kernel commit time.
[Exact brackets and losses](G2-followup21-evidence/lab/TestF21_AT02_RollbackSibling/summary.json)
record **397 ingresses / 224 egresses / 173 losses**, **150 outside** all
completed guard brackets. All three captures have zero kernel drops. These
are counted failures during deliberate fault injection; the diagnostic test
passes by proving the failure mode and subsequent verified recovery.

The [members source](G2-followup21-evidence/inputs/members-F11-03-capture.json)
shows generation-13 failed Applies 411/417/423/429/435/441/447/453 with
**no release**, followed by `apply_rollback` records 412/418/424/430/436/442/448/454.
Apply 411's install command spans 238.739 ms; it is not a completed guard.
Apply 459 finally completes release. That ordering supports a prolonged
baseline outage, not the assertion that every loss fits a successful guard.
The source explicitly lacks an independent monotonic bracket for the older
member injection, so this consult does not assign an exact historical packet
loss instant or claim the lab's 280–305 ms as its duration. Agent-core 29 owns
that cause/retry correlation and any Engine repair.

## F11-01: initial selected route

The [AT01 source](G2-followup21-evidence/inputs/members-F11-01-forwarding.json)
retains five member `fvnb0` packets, zero node/origin receipts, generation 5
and unchanged zero device counters after route selection. Selection establishes
local intent, not receipt by the selected peer. The first unconfirmed interval
is after member tunnel egress and before node ingress.

The new instrument locates **its own** inner/outer packets at member, tunnel
endpoint, enforcement node and origin, as above. The AT01 NetBird-selected
route, ICE endpoint and first-use route installation are outside this kernel
netlab fixture. No captured local packet is represented as one of those five
native packets. Devstack 21 owns that fixture investigation.

### Requests to orchestrator

1. **2026-10-31 — devstack/native transport owner, F8-02:** in
   `tests/e2e/native_underlay.go` and `tests/e2e/followup18_native_test.go`, retain
   the asymmetric receive-MTU failure and add the matching **IPv6** endpoint
   captures to the original members mutation sequence. The loss is beyond
   enforcement and before member underlay receipt. `ApplyNativeInputs` currently
   lowers only member eth0/eth1. For a fixture intended to represent a 1280
   *path*, make the selected peer's sender egress constraint explicit too
   (a fixture-only route MTU for that P2P endpoint, or a controlled routed
   bottleneck with PTB), and prove reverse source fragmentation/PMTU adaptation
   and 64,064-byte completion. Do not replace the failed row with that control.
   If the asymmetric path is intentional, assign the native NetBird/WireGuard
   transport owner to handle peer/path MTU changes and reverse packetization
   on that selected path; the required behavior is to avoid unfragmented
   1360-byte packets toward a known 1280 receiver. No global tunnel MTU/MSS
   setting or `node/enforce` clamp is justified. These files are outside this lane.
2. **2026-10-31 — orchestrator/acceptance owner:** record the proposed expiry
   wording and the outer-guard assessment above before changing any AT13/AT29
   predicate or documented ceiling. Keep D-104/D-120 unchanged and every
   pre-D/renewal loss counted. The design reserves 100 ms plus <1 ms rounding;
   literal zero pre-D loss and a guaranteed historical 52.596424 ms ceiling
   are not established. Spec/acceptance edits are outside this task's scope.
3. **2026-10-31 — agent-core 29, F11-03:** consume the completed-guard,
   baseline and explicit retry brackets above when reviewing
   `node/agent/engine.go` Apply/rollback and members' 411–459 stream. Preserve
   the failed install population separately; join error cause, baseline write,
   next successful Apply and sibling ingress on the same boot. Recovery duration
   is caller-dependent. No bypass of fail-closed baseline is proposed; any
   agent change and historical D-133 attribution remain in that lane.
4. **2026-10-31 — devstack 21, F11-01:** extend the initial route experiment in
   `tests/e2e/at/members/forwarding_test.go` (coordinate with members)
   and `tests/e2e/native_underlay.go` with simultaneous member tunnel/selected
   underlay, selected node underlay/tunnel, and origin captures plus peer endpoint
   and route readbacks. Join those five SYNs/retries to outer packets before
   assigning a drop to the node. This AT01 route is outside the netlab fixture;
   no native forwarding fix is claimed here.
