# Follow-up 20 — operator evidence through #344

Verified against main `96ec0630e4834e13a3a14d65553ba8108bb5da55` on 2026-10-31,
through agent-core 28 (#344). Handover ran no new live acceptance. The
[seven roadmap rows](../../tests/handover/testdata/followup20-main.json) and
[37 immutable inputs](../../tests/handover/testdata/followup20-snapshots.json)
pin this cut under D-098 / D-109 / D-132. Counts are pass / blocked / fail unless
stated otherwise. The [decision and findings register](README.md#follow-up-20-decisions-register-2026-10-31-edt)
records current dispositions. Updates 11, isolation 10, members 11 and
dns-publish 9 are running but have no results on main at #344; their subjects
below remain pending. Later results must not be read into this cutoff.

## Usage 6 — #338, D-130 measurement closed

**AT14–AT20 pass** in live run **71b1a160**, with both D-130 halves (#330 / #331)
and D-118 origin retention. The first event of the fresh stream is the production
**planBaseline unknown-continuity gap**: sequence **1**, zero records, interval
`2026-10-31T16:05:33.253516Z` → `2026-10-31T16:05:33.253517Z`, exactly **1 µs**.
It reaches local and central **ACK 1 before counted traffic**; it is neither
skipped nor re-minted. Counted event **2** then reaches ACK 2 with
**1,066 + 1,513 = 2,579 bytes**, exactly equal to the kernel upload/download
direction-counter deltas, with original member/device/binding attribution,
canonical payload retention and stored same-key ACK replay.

This closes the newly minted-event measurement left pending at follow-up 19's
cutoff. Enrollment 9 owns sealed-event recovery; usage 6 measures newly minted
events, without rewriting old events. Follow-up 5 run **170ea3e3** and its three
diagnostics are byte-identically archived under
`docs/e2e/at/usage/history/follow-up-5/` (**59 files**, SHA256SUMS).
[Usage source snapshot](../../tests/handover/testdata/history/follow-up-20/usage6.md).
Usage's AT20 is the retained-usage handoff, not §19's DNS AT20 case.

## Handover 19 and fleet-update 13 — #339 / #340

Handover 19 remains frozen through **#337** (`57a8cc9f`):
`TestHandover_Followup19Claims`, **16 subtests / 13 pinned roadmap rows /
37 immutable inputs**, D-129–D-132 beside D-124–D-128, and migration **00193**
rollout / guarded Down guidance. Follow-up 18 remains frozen too. The
[original register](../../tests/handover/testdata/history/follow-up-20/register19.md)
and original entrypoint excerpts preserve the previous assertions. Deploy
00193 before ingestion resumes; Down refuses equal-endpoint rows. No spool
editing, event skipping or constraint bypass is authorized.

Fleet-update 13 closes the D-132 lane follow-up. The D062 harness tests read
owned copies in `internal/fleetupdate/testdata`, byte-identical to the updates
follow-up-9 archives. Each has a one-line archive-path / SHA-256 comment; empty
precheck results fail explicitly before indexing. `TestD132_EvidenceReferencesUseImmutableHistory`
in `evidence_paths_test.go` rejects mutable `docs/e2e/at` references outside
`history/` across the package, including fixtures and comments.
`docs/updates/central-acceptance.md` records updates 10's measured D-110 / D-127
central acceptance: 202 / stored replay / 409, unchanged release pointers on
checkpoint failure, and a labeled TEST ONLY correction. Physical rollback is
not implied. **No product code, contract, migration or dependency changed** in
#340. [Fleet source snapshot](../../tests/handover/testdata/history/follow-up-20/fleet13.md).

## Installer 7 and updater 18 — #341 / #342

**Updates F10-02 FIXED (#341):** the installer derives `result_url` for
`POST /api/v1/agent/update-results` from the required validated HTTPS control
origin. It uses the enrollment server CA and enrolled mTLS certificate/key;
`allowed_artifact_origins` is **exactly the control origin**. The node-certificate
issuer is not assumed to be the HTTPS server issuer. Loopback requires the
explicit **TEST ONLY** `install --allow-loopback` opt-in. Invalid inputs fail
before host changes; same-release reruns repair reporting fields and server CA
while preserving unrelated settings, identity, usage and updater journal.
The privileged suite passes **13/13**, including generated-config delivery,
distinct server/client-issuer CAs, wrong-trust and unauthenticated negatives.
The dated handoff is `docs/node/install-followup7-2026-10-31.md`;
its [frozen copy](../../tests/handover/testdata/history/follow-up-20/installer7.md)
keeps the measured scope.

**Pending owner decision for updates 11:** a release host distinct from the
control origin would be refused by that exact-origin policy — an owner
decision, not a defect, until updates 11 reports. No policy expansion or
production origin is chosen in this handover.

Updater 18 fixes silent nil-sink delivery: explicit `Engine.Flush` without a
sink fails **VALIDATION_FAILED**. Sealing without reporting configuration logs
**result_url_missing**, keeps the original receipt **pending**, and creates no
ACK. A later configuration repair transmits the **identical receipt once**.
`TestUPD05_MissingResultURLKeepsReceiptPending` and
`TestUPD05_ResultConfigPolicyAndServerTrust` cover that behavior and trust policy.
`TestAT25_ReleaseRootCandidateSwitchAndHealth` aligns fixtures to root
`fvpn-agent` with whole-root integrity. Race/load checks pass **40/40**.

`docs/updates/node-followup18-2026-10-31.md` consumes devstack 19 and updates 10
using **immutable UUID-run pins only** (D-132).
[Frozen handoff](../../tests/handover/testdata/history/follow-up-20/updater18.md).
**F18-01 remains pending with devstack 21:** confirm classification of the
stopped-producer **1 / 1 / 2** apply/replay/recover exits as the intended D-125
negative. The historical 13/0/1 campaign stays failed; serving success is a
separate run.

## Devstack 20 — #343, shared prerequisites delivered

Follow-up 19 was archived first (**168 files**). The D-087 / D-131 seam is now
composed in a **TEST ONLY control**: `FVPN_E2E_GRANT_SEAM=1`, read at
`deploy/devstack/up.sh:122`, exports `SetMemberGrantsReady`,
`MemberGrantEvidence` and `MemberGrantInvariants`. This is a lab composition of
the constructor seam; it adds no production control setting.

Fixed-generation runs **5ab1f11c** and **79e59c20** pass **7 / 0 / 0** each;
true → accepted serving is **3,980 / 4,112 ms**. The final run retains 47 original
wire hashes and 103 contiguous detached timing records. **The D-087 central-seam
blocker is closed**; assembled AT10 belongs to **isolation 10, pending at #344**.

Members F10-02 diagnostics retain **247 every-read-success counter samples**
with same-boot monotonic brackets; the historical unreadable sample was not
reproduced. Members F10-03 diagnostics retain **36 complete per-query DNS traces**;
the historical responder remains unrecoverable. These exports do not close
members 11 / dns-publish 9 re-measurements or rewrite either failed historical row.

**Updates F10-01 shared side FIXED (#343):** regular release-root artifacts and
`state/meter.checkpoint` pass switching **11 / 0 / 0**, with five health checks.
Updates 11 must adopt its own `live_test.go:221/:436` root-path assertions and
measure regular **AT23–AT27**; those results remain pending at this cutoff.
The broader fixture remains **15 / 0 / 1**, retaining an **instrumented
MTU-1280 failure: 10,010 / 64,064 bytes**. No global MTU fix was applied.
**F8-02 is instrumented but still unattributed**, for netlab 21. D-102 answers
are in `docs/e2e/RESPONSES.md`.
[Devstack summary](../../tests/handover/testdata/history/follow-up-20/devstack20.md)
and [D-102 answers](../../tests/handover/testdata/history/follow-up-20/devstack20-responses.md)
are frozen here.

## Agent-core 28 — #344, historical writer still unattributed

The delivery adds signed same-boot kernel adoption across restart and
same-generation repair with the original lease. Control I/O is bounded; serving
verification has an independent bounded context under process cancellation.
The watchdog reads fresh state under the Apply lock, emits baseline reason
logs, and the producer adds explicit **baseline / heartbeat source kinds**.
An ordinary stop checkpoints usage while keeping the meter epoch open; updater
withdrawal still requires finalization.

**ISO-F6-03 remains OPEN.** Operations **955–961** all lie inside process run
**0b191fdd**. Boot was not the initial writer; watchdog decision evidence was
never retained. No writer is attributed from the later restart proof. The dated
`docs/node/followup28-iso-f6-03-2026-10-31.md` handoff supplies isolation 10's new
evidence contract: retain agent/watchdog logs, complete baseline/heartbeat and
Apply/live/blocked producer records across all process runs, and independent
kernel and ingress/egress observations. Count every loss. The restart sections
in `docs/node/README.md` now match the implemented contract; that request is closed.
[Frozen handoff and distributions](../../tests/handover/testdata/history/follow-up-20/agent28.md).

**D-133 was applied to this delivery.** Its first delivery's final live gate
failed: `TestD085_LiveReadNetlab`, Apply install **152.125 ms > 100 ms**, host load
approximately **16**. The same-host **A, B, A, B** comparison at a CPU-quiet host
then passed **all four live gates and four full agent suites**. A is unchanged
main `0df1716d`; B is branch `036a4829`. Quiet is defined by runnable queue and
idle, never by load average: this run required six five-second samples with
**r ≤ 3 / idle ≥ 80%**, discarding the since-boot row. Per-leg host states,
failed CPU samples, all completed-guard p50 / p90 / p99 / max and failed
operations are retained in the handoff.

| Measurement (ms) | A1 / A2 | B1 / B2 |
|---|---|---|
| Live install-command maximum | 25.49 / 24.76 | 29.01 / 27.93 |
| Live outer p99 | 38.07 / 40.34 | 51.67 / 50.46 |
| Assembled outer p99 | 47.89 / 49.75 | 43.60 / 41.24 |

The failed live timing gate is attributed to **host contention**, with the
distributions retained; B's higher live tail is visible and its assembled tail
is lower. No production value or acceptance tolerance changed. Two **measured
limits remain open, not defects of record**: literal zero pre-D loss (two
pre-D expiry drops, first ingress about **D−63.3 ms**, while the unchanged
D-104 predicate passes), and the historical **52.596 ms** outer-guard ceiling
(**B1 57.92 ms**). Both remain unmet across the main/branch comparison; netlab 21
owns the enforcement assessment with both distributions. No one requested a
tolerance. D-133 is a process / test-acceptance decision, decided **12:4x EDT**,
applied **14:0x EDT**, not a product value.

## Open work at #344

- ISO-F6-03 writer and assembled D-087 AT10: isolation 10 on agent-core 28's
  instrumented main; no result on main at this cutoff.
- Regular AT23–AT27 and the distinct artifact-origin question: updates 11.
  F10-01's shared-fixture side and updates F10-02 are closed.
- Members F10-02 / F10-03 re-measurements: members 11 / dns-publish 9. F10-04 is
  closed under D-129; members 11's complete AT09 row is still pending.
- F18-01: devstack 21 confirmation of the intended stopped-producer negative.
- Instrumented F8-02 and the two D-133 measured limits: netlab 21 assessment.
  Native MTU 1280 remains open under D-123; IPv6Block remains the overlay policy.
- AT11 real platform routes; AT13 full continuous fault matrix including
  physical arm64; AT29 / AT31 resources; physical arm64 execution, OS reboot,
  power loss and real platform clients.
- **Dependabot #22 OPEN**: medium, `minimatch < 11.0.1`,
  **GHSA-g3hv-75wg-m249**, in `apps/portal/package-lock.json`; owner **portal**.
  This is a **dependency-hygiene item**, not a product defect until the portal
  lane reports reachability. The [API status capture](../../tests/handover/testdata/history/follow-up-20/dependabot-22.json)
  records the open alert; a main-cut lockfile excerpt separately pins its
  development dependency. No dependency was changed.

The agent-core README restart request and “agent-core 28 in flight” are no
longer open items. Historical acceptance failures remain frozen, and no earlier
AT23–AT27 pass is carried forward.
