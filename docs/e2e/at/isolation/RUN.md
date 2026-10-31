# Isolation run procedure — Follow-up 7

```sh
make -f mk/at-isolation.mk at-isolation-lint
make -f mk/at-isolation.mk at-isolation
python3 tests/e2e/at/isolation/review_followup7.py
make go-lint
go vet ./...
```

The binding is `w4-isolation` / port base `36000`; Docker objects are limited
to `fvpn-w4-isolation-*`. The target runs the mechanism lab, removes it, builds
the assembled stack, runs the owned overlay against the shared driver, then
tears down. A blocked or failed product case returns make exit 2 after all
four records are written. A mechanism pass never substitutes for a product
pass (§19.2). Check `pgrep -af gates.sh` before starting; wait if two integration
suites are active. Commands run in the foreground; Go scratch is on disk under
`.agent/tmp`. Host networking is untouched.

## D-087 receiver recipe

Before enrollment, the harness installs its TEST ONLY transparent receiver in
the node's existing loopback mTLS transport. It validates the same client CA,
uses the enrolled client certificate upstream, and forwards bodies and
responses unchanged. The agent config sets `heartbeat_seconds=1`,
`poll_seconds=1`, and `observation_seconds=1`. The receiver is running before
the first lease renewal, including empty-lease issuance. The source entrypoint
starts it again across the ordinary restart and the clone fixture's source
restart. It records no request headers or credential responses.

Every `POST /api/v1/agent/observations` body and acceptance response enters a
queue drained to disk every **100 ms**. Renewal transactions and successful
signed desired-state GET responses are retained for correlation. This is not
a latest-state snapshot. `receiver-capture.json` holds the transaction stream;
`lease-history.json` includes central accepted report IDs so missing accepted
observations can be detected. Repeated POST retries are kept. The separate
250 ms `convergence-timeline.json` remains auxiliary kernel/database evidence.

`review_followup7.py` requires an accepted report with all of these:

- `report_id`, `observed_at`, and `lease.lease_id` retained verbatim;
- nonempty signed desired bindings, matching desired/applied/lease generation,
  empty lease members, and observation inside that lease's validity interval;
- `member_forwarding_enabled=false`, `provisioning_state=ready`,
  `safety_status=passed`, and only `awaiting_member_grants` as block reason;
- lease, qos and source_binding checks coded `AWAITING_MEMBER_GRANTS`, and
  exactly one error with that code at stage `lease`.

Expired leases, empty desired bindings, mismatched generations and other
safety failures are excluded. The script relates each qualifying report to
the next populated lease on its generation. It writes derived evidence only;
it does not silently promote AT10. The AT10 check also needs an actual counted
member control. Comparators are follow-up 6's **4,767 ms** assisted recovery and
usage follow-up 3's **5,021 / 5,025 ms**. A window not observed stays blocked.

## ISO-F6-03 continuous expiry capture

The existing fleet API sequence sets the TEST ONLY offline ceiling to 60 s,
forces renewal through a capacity generation, stops control/worker, reads the
node-held signed lease, and later restores the ceiling with its audit trail.
The original strict case predicate remains: counted forwarding near expiry,
no early baseline, baseline within one configured agent tick, and post-deadline
member ingress denied with flat admission counters (§16.3).

Ten seconds before the signed deadline, continuous header-only tcpdump starts
on the node at both member and uplink boundaries. A member makes TCP connect attempts every
50 ms plus connect time (150 ms timeout), retaining the start/end/result of
every attempt. A separate reader brackets nft/heartbeat reads at a 100 ms target
cadence. Capture lasts 25 s, spanning the deadline; kernel-drop summaries must
be checked. Linux packet timestamps and driver/container UTC share one host
clock. `whole-node-probes.json`, `whole-node-continuous-reads.json`,
`whole-node-bracketed-reads.json`, and the two capture text files preserve the
raw timing. `expiry-timing-analysis.json` derives the last successful connect,
first failed connect, baseline bracket and expiry-reporting heartbeat. A
connection timeout is a missing-response bracket, not an exact drop timestamp;
use the packet headers to locate ingress and response. Conservative monotonic
expiry is allowed by §16.3, but the retained case is not redefined to hide an
early loss of its positive control.

## D-090 containment and D-096 recovery

After the ordinary AT sequence, obtain another counted member control. Start
continuous UDP/9 probes and captures at member fvnb0, node interfaces and the
controlled origin. Inject a foreign clsact ingress flower/mirred redirect on
node fvmb0 toward fvup0. Record injection brackets, accepted observations,
heartbeat, links, addresses, member routes and tc filter read-back. Confirm
`TC_FOREIGN_FILTER`, non-serving, member links administratively down, uplink
and control links retained, and no bypass packets after quarantine.

Remove the foreign filter and request a newer capacity generation for a full
Apply. The harness deliberately removes the saved member routes to expose the
D-096 hold, then restores its saved addresses/routes inside the lab node. It
records who performed restoration and retains each recovery read. No topology
expectation, applied-model row or observation is seeded. D-100 permits refreshing
expectations only through a clean verified Apply of a newer published generation.
A harness restoration or extra recovery makes the assembled run **assisted**;
it is never an unattended clean pass. `foreign-redirect.json` and the
capture files contain the deciding observations, including any failure to
recover.

## Freeze latch

Last, `StartClonedIdentityFixture` stops the source for a private consistent
snapshot, starts a second presentation of the enrolled identity on the clone
lab network, and resumes the source. This is the same copied incarnation,
not a new enrollment. No key material enters committed evidence. Correlate
`identity_frozen: true` in the source receiver's actual acceptance response
with non-serving heartbeat, immediate renewal and omitted generation ACKs.
Clone detection ownership remains ENR-02 / ENR-03; `identity-freeze.json`
records the available fixture result, and the receiver analysis records its
consumer effect. A fixture/setup failure is a blocker, not a latch pass.

## Archives and provenance

`history/follow-up-6/` is the complete 44-file current follow-up 6 snapshot,
byte-identical to the files before this run. It includes the final assisted
run `bf671f3a-c6ac-4824-9107-425c4c5a5f02`, not the older diagnostic or
bootstrap-only snapshots. `history/follow-up-5/` is untouched; the handover
follow-up 13 test remains pinned there under D-098. Pass claims for current
AT10–AT13 come only from the final full run identified by the current records.

## Follow-up 7 capture corrections

The full run used harness `d1a90ea`. Its node `any` captures contain both fvmb0
and fvup0, including direction. The extra origin-container captures failed
because that image has no tcpdump; they are retained as errors. After the run,
the helper was corrected to use an explicit second node-uplink capture. No
new live outcome is attributed to that correction.

The shared `lab.docker` truncates diagnostic stdout. The full receiver disk file
was recovered through a task-owned network-disabled reader and matched to the
source boot IDs in the central report index. The exporter now uses unbounded
`exec.CommandContext(...).Output()` for this specific capture. Keep the full
receiver artifact even when the derived provisional check is blocked.

The extra assist actually used at 06:31:25Z is preserved as an explicit command:

```sh
python3 tests/e2e/at/isolation/assist.py
```

Run only against the still-running assigned lab after capturing the failed
recovery. It performs an authenticated capacity PATCH to 208000000 down_bps
with ETag/Idempotency-Key, then restores the saved lab IPv4 addresses/routes;
it writes `recovery-action.json`. It does not make the assembled run unattended
and does not establish success merely because the API returned 202.
