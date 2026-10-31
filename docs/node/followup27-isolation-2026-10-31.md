# Agent hand-off to isolation 9 / devstack 19 — 2026-10-31

## D-087 fixed-generation transition

The agent already emits the qualifying observation. Isolation 8's 544 POSTs
with zero qualifying reports did not exercise the clean fixed-generation
transition; they do not demonstrate a missing producer emission. This follow-up
adds an ordinary-renewal unit proof and an assembled agent/kernel/TLS
fake-control proof, without changing observation or admission behavior.

Use an unfrozen, published, baseline-safe node with desired bindings, a passing
probe and a fixed generation. On the fake control call
`SetMemberGrantsReady(false)` **before issuance of an empty lease**. Merely
changing the fixture cannot revoke a previously issued populated lease. Wait
for the ordinary renewal (or start a fresh process), then retain every
observation body at the receiver. Once the qualifying report is accepted,
call `SetMemberGrantsReady(true)`. This changes only future grants; no generation
bump, unpublish, projector-row edit or agent state mutation is part of the
trigger. Devstack 19 supplies the corresponding assembled harness transition.

The in-container TEST ONLY labcontrol exports the same fixture through
`POST /member-grants {"ready":false|true}` and all distinct accepted bodies
through `GET /observations`. These are labcontrol admin endpoints, not new
product APIs. `POST /lease-window {"seconds":60}` selects the bounded test
lease window. The agent uses one-second heartbeat/poll; the assembled test uses
`observation_seconds=3600` to prove immediate semantic transition reporting.

A qualifying report carries:

- `provisioning_state=ready`, `safety_status=passed`,
  `member_forwarding_enabled=false`, and a non-null `lease.lease_id`;
- exactly `awaiting_member_grants` in this clean fixture, with no
  `lease_expired`, and failed `lease` / `qos` / `source_binding` checks whose
  codes are `AWAITING_MEMBER_GRANTS`;
- one stage=lease error with code `AWAITING_MEMBER_GRANTS` and message
  `Serving lease has no member grants yet.`;
- the same applied generation as the later serving report. A retained QoS
  read detail is diagnostic and need not equal a static sentence.

`observe` / `observationState` recognize the reason/check transition even
before the periodic interval. `noteProvisionalSchedule` / `nextStepDelay`
renew at the next grant-less slot (first wait one heartbeat, doubling up to
60 seconds, still subject to I/O and Retry-After). The matching populated
lease then requires full Apply and live verification before a new serving
report. The harness must distinguish this clean predicate from D-126's valid
mixed safety/provisional report; see the [members hand-off](followup27-members-2026-10-31.md).

Proofs: `TestD087_AT10_FixedGenerationProvisionalToGranted` drives a natural
renewal and the exact false/true fixture, preserves desired bindings and
generation, forbids an early grant request/admission, and checks both delivered
reports. `TestLabD087_D104_GrantRenewalExpiryCorrelation` uses the real process,
real nft/tc and meter, independent TCP denial/allowance, and an increasing
admission byte counter after grants. These are producer/trigger proofs, not a
claim that the entire assembled AT10 row or device matrix has passed.

## D-104 local timing collection

Consumed the hand-off at its actual location:
[netlab 19](../poc/G2-followup19-netlab-agent-2026-10-31.md).
With `test_mode=true`, set `test_timing_path` to an absolute file path in a
harness-owned directory. Default is disabled. Every actual Engine Apply,
live VerifyServing and blocked audit records `ServingEvidence.TimingEvidence()`
immediately after return, including errors. Each append is a separate JSONL
snapshot, outside the operation/guard, with a process run UUID, boot UUID,
contiguous decimal sequence, wall sample bracketed by CLOCK_MONOTONIC, and the
matching D-106 digest. No snapshot overwrites another. Each record also has
the currently accepted lease's signed times, request anchor and absolute
whole-node deadline. Renewal records bracket the synchronous lease transaction
and identify whether it accepted a new lease; failed/replayed requests retain
the current lease without presenting it as newly issued.

The timing object is unchanged: operation start/end, final render, exact
install/release command brackets and flags, all verifier invocations (including
both quarantine-recovery reads), nft ruleset read, millisecond precision text
read and aggregate tc read. All nanoseconds are decimal JSON strings. `source`
distinguishes apply/live/blocked audit; a clean deny-all verifier is not a
forwarding claim. Nothing is added to ObservationReport or D-106. Failed
measurement writes log a fixed message; sequence gaps invalidate a trace.
The harness owns file retention/removal and must use a system monotonic clock
in the same boot/time namespace as its packet recorder.

`tests/netlab/agent/timing-analysis.py` validates ordering, positive samples,
command flags, contiguous sequences and matching digests. It reports median
and worst producer brackets, and lists each renewal loss's send→80ms-completion
window overlap with the install command, captured ingress inside the install,
outer guard, and definitely guarded inner interval separately. Packet timestamps
are read observations, not exact kernel commit times. Missing/ambiguous
correlation remains unresolved. D-120 supplies **no** loss tolerance here.
The new run cannot retrospectively assign timestamps to isolation run
`7779d83c`'s historical 17 losses; those remain in their original evidence.

Run the suites from the repository root with disk TMPDIR/GOTMPDIR:

```sh
FVPN_LAB_PREFIX=fvpn-w2-agent-core-f27 make -f mk/node.mk node-test-netlab
FVPN_LAB_PREFIX=fvpn-w2-agent-core-f27 make -f mk/node.mk node-test-live-netlab
```

`FVPN_AGENT_EVIDENCE_DIR` optionally selects a local artifact directory.
The process suite writes its operations, renewal/expiry packets, accepted
observations and analysis under `assembled/`. The live suite retains every
operation across restart/recovery and its separate 1ms renewal / phased expiry
traces. Both launchers remove their own prefixed containers and images.

## Measured results

Final process run, 2026-10-31 at 13:04–13:06 UTC: generation 3 stayed fixed.
Grant-ready → counted forwarding took **2.842420 s**; admission counters moved
**1,404 → 12,060 bytes**. The qualifying and serving report IDs differ, and both
were delivered before the one-hour periodic observation interval.

The run retained **403 snapshots** (98 Applies, 299 standalone verifications,
six renewal transactions), including blocked/failed operations. The two
ordinary successful renewals during the measured stream are retained with
request anchors and their accepted lease deadlines. Producer brackets over
the entire provisional→granted→renewal→expiry capture are:

| Producer bracket | Samples | Median | Worst |
| --- | ---: | ---: | ---: |
| Apply operation | 98 | 80.522598 ms | 111.687840 ms |
| Final render → installed-by | 98 | 16.199167 ms | 27.132020 ms |
| Install command | 98 | 16.101975 ms | 27.045925 ms |
| Release command | 98 | 15.969551 ms | 28.742363 ms |
| Outer guard bracket | 98 | 32.271094 ms | 51.114216 ms |
| Complete verifier | 397 | 23.888295 ms | 44.653661 ms |
| nft ruleset read | 397 | 2.601977 ms | 5.977605 ms |
| Millisecond precision text read | 392 | 1.466713 ms | 3.664890 ms |
| Aggregate tc read | 397 | 16.431154 ms | 28.096213 ms |

The inner definitely-guarded interval had median **0.005385 ms**, maximum
**0.010987 ms**. The kernel commits occur somewhere inside the install/release
command brackets. Total Apply duration is not the D-104 100ms budget; the
checked final-render→installed-by interval stayed within that bound.

The independent **80ms dual-phase** renewal diagnostic delivered
**1,506 / 1,518** probes (**12 losses**). All 12 had captured member ingress and
no uplink egress. All 12 ingress read timestamps coincided with outer guard
brackets; **10 loss windows overlapped an install-command bracket**, while
two did not. This is command/capture correlation, not exact kernel drop timing.
Every loss remains counted; strict renewal continuity is not a pass. There were
zero capture drops and no unmatched loss ingress in this run.

Whole-node expiry D=`35537916089829` ns passed the unchanged D-104 packet
predicate: last positive **D−100.581476 ms**, first drop **D−63.888900 ms**,
97 positives / 15 drops, no positive at/after D or after the first expiry drop.
The final trace uses the last actually accepted node deadline, with no renewal
after the harness holds lease responses. No cadence allowance was added.

The preceding process run is also retained, not discarded: 395 snapshots,
1,481/1,518 delivered, 37 losses, all 37 ingress timestamps in outer brackets,
35 loss windows overlapping install. Its expiry offsets were
−100.168232 / −63.890781 ms. Phase/scheduling differences change the loss count;
neither run supports uninterrupted renewal availability.

The separate live trace retained **477 snapshots across two process runs**:
120 Applies, 339 live reads, 11 blocked audits and seven renewal transactions.
It includes **14 failed operations** and **two Apply snapshots with both
quarantine-recovery verifiers**. With 1ms dense probes over two ordinary
renewals and 75 guarded Applies, **74,898 / 76,000** were delivered
(**98.5500%**); all 1,102 loss ingress timestamps coincided with the library's
outer guard brackets. Median/worst outer bracket was **29.063589 / 44.393668 ms**;
final render→installed **14.148121 / 26.049629 ms**. Final phased expiry was
**D−100.671926 ms / D−0.776235 ms**, 27 positives / seven drops, with no later
positive. This extends the historical follow-up 25 trace without replacing it.

Artifacts are retained in the lane's ignored `.agent/f27-evidence/`: the final
process `assembled/`, preceding `assembled-first/`, and live
`operations.jsonl`, `d104-renewals.json`, `d104-expiry.json`,
`live-analysis.json`. Each includes the original integer/string clock evidence;
the process `analysis.json` lists every loss and matching operation sequence.
The live launcher returns private container-created artifacts to the invoking
harness owner before cleaning up its container/image.

Platform: Linux amd64, kernel `7.0.0-38-generic`, nftables `1.0.9`,
iproute2 `6.1.0` / libbpf `1.3.0`. This is an agent/kernel/TLS fake-control lab;
physical arm64, native NetBird and the full crash/reboot/updater/platform matrix
remain 待實測. These measurements do not close the entire AT10 or AT13 matrix.
