# Follow-up 7 — assembled isolation evidence

Final full run **041e9e48-79a6-45dd-a19d-8b5a63290655**, started
2026-10-31T06:16:20Z; final case records written at 06:31:52Z. Harness build
`d1a90ea60abd853f404e1df571dd3a5871d89b5f`, clean at launch. Product baseline
is current-main `c6e3eca` (includes #267/#272); immutable image IDs are in each
AT JSON. No product code changed. The initial startup diagnostic was terminated
before its AT matrix because its driver had started before the final observer
update; none of its outcomes is used here.

**This was an assisted run, not an unattended clean pass.** The scripted
member-route restoration and the extra API generation in `recovery-action.json`
did not restore serving before the clone test. All current pass claims below
come from this one full run. Mechanism AT10–AT13 passed separately.

| Case | Result | Checks: pass / blocked / fail | Deciding observation |
|---|---|---|---|
| AT10 / ISO01, ISO02 | blocked | 7 / 2 / 0 | Six protected destinations and the interface/generation transition passed. Foreign-tc containment withdrew serving, but clean Apply/D-096 recovery was not established. No qualifying provisional report. |
| AT11 / ISO01, ISO03 | blocked | 2 / 1 / 0 | Metadata and deprecated-6to4 passed; real IPv6/platform paths remain unavailable. |
| AT12 / ISO01, ISO04, DNS03 | pass | 10 / 0 / 0 | Four peer denies, three management-port denies, designated UDP/TCP DNS, forged-source attribution and group withdrawal passed with their counted controls. |
| AT13 / ISO02, ENT02, SEC01 | fail | 5 / 1 / 1 | Whole-node expiry failed the unchanged baseline-within-one-tick predicate. Post-deadline denial passed; the full continuous crash/NetBird/reboot/updater matrix remains blocked. |

## Foreign ingress redirect and recovery

All times below use the shared host UTC clock. `foreign-redirect.json` retains
bracketed kernel reads and accepted product observations;
`foreign-redirect-analysis.json` derives the intervals. The node `any` capture
identifies **both fvmb0 ingress and fvup0 egress**. The member-side capture has
553 packets and the node capture 12; both completed with zero kernel drops.
The separate origin capture failed because that image has no tcpdump; it is
retained as an error, never described as an empty target capture.

- Injection: **06:29:29.204706725–06:29:29.278990163Z**, clsact ingress flower
  `pref 49152`, UDP/9 to the controlled origin, mirred redirect to fvup0.
- An unsnatted packet left fvup0 at **06:29:29.305982538Z**, before quarantine.
  This is a measured redirect bypass before withdrawal; there is no blanket
  zero-packet claim for the entire injection window.
- `TC_FOREIGN_FILTER` report `01a190a6-22f8-74d6-8178-96f3551b3f4c` was observed
  at **06:29:29.464313239Z** and accepted at **06:29:29.490009445Z**. It includes
  `tc_evidence`, `initial_quarantine`, `quarantine` and `apply_quarantine` errors,
  with the actual flower/mirred attachment and retained link/baseline evidence.
- First confirmed member-down read: **06:29:29.805935672–06:29:29.832949690Z**,
  at most **628.243 ms** after injection began. fvmb0 is administratively down;
  fvup0 and control eth0 stay up. Accepted observations continue. After this
  read, **538 member probes and zero node-uplink packets** were recorded.
- Filter removed and member routes deliberately withheld:
  **06:29:32.865020378–06:29:32.889058275Z**. Capacity PATCH returned 202 and
  signed generation 32 was received at **06:29:33.290828482Z**.
- Harness restoration began **06:29:39.259867752Z**, **6,370.809 ms** after
  removal completed. It restored the saved lab IPv4 address/route state; the
  commands also reported already-existing objects. NetBird was running, but
  **the harness performed the recorded restoration**.
- No `member_path_unconfigured` report was observed. Reads instead retained
  `TC_FOREIGN_FILTER`, `policy_missing` and non-serving state. Thus the phase
  labeled “D-096 hold” is an attempted measurement, **not proof that a clean
  Apply reached the D-096 check**. Recovery remains blocked.
- Extra assist at **06:31:25.419525174Z**: API PATCH accepted generation 33,
  plus another restoration of the saved member path. The foreign case's final
  counted recovery control was false; no recovery pass is claimed.

D-100 is respected: neither expected topology hashes nor product state rows
were edited. The ordinary earlier interface-change test passed on generation
16; the follow-up 6 unexpected rejection did not recur at that point. The
known injected attachment is separately identified in this run.

## D-087 provisional capture

The receiver was installed before enrollment/empty-lease issuance, with
heartbeat/poll/observation intervals all **1 s** and a **100 ms** drain.
`receiver-capture.json` retains **4,482 transactions**, including **3,629
observation POST bodies** and 581 successful observation responses. The
boot-indexed central snapshot has 581 source reports and 11 clone reports;
**zero source report IDs are missing**. Clone requests use its own proxy and
are not mislabeled as source receiver traffic. The receiver continued until
the source stopped during teardown.

The first export hit the shared driver's diagnostic output limit. The complete
20 MB disk capture was recovered through a task-owned, network-disabled reader,
then checked against the central report index. The reader container/image were
removed. The owned exporter now bypasses that diagnostic limit for future runs.

**Zero reports meet the provisional definition.** Thirty distinct reports
mention `awaiting_member_grants`, but the pre-freeze candidates also have a
failed tunnel probe/safety status, and later candidates have the identity latch;
they are not the requested ready/passed single-error state. Rejected 429
reports and mismatched/expired/empty-binding states are excluded. Therefore
there is **no new ready-to-populated duration** to compare with follow-up 6's
assisted **4,767 ms** or usage follow-up 3's **5,021 / 5,025 ms**. Those numbers
remain historical comparators, not this run's measurements.

The stream also contains **3,031 HTTP 429 responses**. The rate-limited retry
stream and observation backlog are retained for agent-core/wire-nodes; absence
of a qualifying accepted report is not inferred from a latest-state snapshot.

## ISO-F6-03 whole-node deadline

Signed node deadline: **06:29:02.925490361Z**, generation 31, 60-second lease.
The original case predicate was not changed. The near-deadline counted request
succeeded (99,079 → 103,011 bytes), but the baseline was not seen within the
configured **1 s** tick, so the check and AT13 remain **fail** (§16.3).

The continuous node capture has **2,389 packets, zero kernel drops**, tagged by
fvmb0/fvup0 and direction. All **251 probes** have start/end timestamps; **189
connected**, and none connected after the deadline. The independent origin
capture failed (no tcpdump); node ingress and uplink evidence remains available.

- Last successful SYN-ACK to the member: **06:29:02.696232187Z**, **229.258 ms
  before** the signed deadline. Last successful connect completed at
  **06:29:02.696688147Z**.
- First failed connect: **06:29:02.746857075–06:29:02.897114892Z**. Its SYN was
  captured on fvmb0 at **06:29:02.748690751Z**, **176.800 ms before** expiry,
  with no corresponding fvup0 packet. The observed stop lies between that
  failed ingress and the last successful response; it is not a post-deadline leak.
- First baseline read bracket: **06:29:04.228093040–06:29:04.250071427Z**,
  **1,302.603–1,324.581 ms after** expiry.
- First expiry-heartbeat read bracket:
  **06:29:10.466713660–06:29:10.496189725Z**, **7,541.223–7,570.699 ms after**
  expiry. Earlier reads still held a serving heartbeat from 06:28:35.295935274Z.
- Post-deadline ingress was denied and admission counters stayed
  **126,231 → 126,231**. The fleet ceiling was restored to 86,400 through the
  API; both audit records are in AT13.

Conservative monotonic expiry is required by §16.3. The roughly 177–229 ms early
packet cutoff is evidence for calibration, not by itself a claim of a spec
violation. The strict one-tick baseline result remains failed. Raw packet
headers, probes and bracketed reads accompany `expiry-timing-analysis.json`.

## Identity-freeze consumer

The real clone fixture was reachable. It presents a copied enrolled identity
and incarnation from a second lab network; clone detection remains ENR-02/03's
ownership. Source acceptance of report `01a190a8-1b90-7a45-95af-4cf4146ee529`
returned **`identity_frozen: true` at 06:31:48.795212863Z**. Its queued body had
been observed at 06:31:38.640670555Z; acceptance timing is used for the latch.

Renewal started at **06:31:48.818693730Z**, **23.481 ms** later, with null
`applied_config_generation`. The first captured frozen heartbeat has wall time
**06:31:48.979081056Z**, **183.868 ms** after acceptance, `serving 0` and reason
`identity_frozen`. Subsequent captured reports omit generation ACKs and report
no forwarding. However, the source was already non-serving with empty grants
before acceptance. **Withdrawal of live grants within one heartbeat is not
proved by this run.** Restart persistence after the freeze was not tested here.

## Archives, checks and remaining scope

`history/follow-up-6/` contains all 44 former current files byte-identically;
`history/follow-up-5/` is untouched. Handover follow-up 13 already reads the
follow-up 5 archive, so no handover test edit was needed. Obsolete follow-up 6
recovery artifacts were removed from the current root after archiving.

Final owned inventory is **0 / 0 / 0 / 0** (containers/networks/volumes/images).
`validation-results.json` records actual command exits and output tails.
No physical arm64 or member-platform run occurred; cross-build evidence is
separate. No push, PR, main rewrite, product code or contract edit was made.
