# Follow-up 18 owner assignments — 2026-11-03

### Requests to orchestrator

**2 new requests; both follow-up 17 requests and all seven carried groups remain tracked below.** F4-03, F4-04, F4-05/F4-07, F4-06 and F4-10 remain owner-resource rows and are not re-asked. F11-01 remains **OPEN and upstream-dependent (D-148)**; this lane makes no product change.

**2026-11-03 — acceptance harness lifecycle compatibility:** assign a follow-up to reconcile `tests/e2e/at/members/command_lifecycle_test.go` with the production `retire` phase emitted by `node/enforce/apply.go`, retaining lifecycle validation and negative controls. [Original producer and 16 exact export-prefix joins](F18-lifecycle-phase-compatibility.json) prove the measured consumer rejects command 28 at sequence 582. This run's original failed results stay failed. The present task permits only the D-146 downgrade change, so no unrelated validator predicate was changed.

**2026-11-03 — acceptance fixture restoration and unexercised D-146 rows:** assign a follow-up to make the finite-expiry scenario's cleanup reliable on the failed-readiness branch in `tests/e2e/at/members/finite_grant_test.go`, without converting its original failure into success or changing the 1280 byte predicate. [Same-member evidence](F18-AT09-precondition.json) retains the grant deadline, missing restoration, later zero visible exits and admitted-device blocker. After that correction, a separately authorized assembled run is needed for all three receive-only rows; this delivery used its single run and makes no live clamp qualification.

The existing F11-03 / agent-core track also receives [the original restart and readiness failures](F18-AT09-subchecks.md): generation 43 → 44, no counted recovery, and the recorded TC_FOREIGN_FILTER/read/quarantine errors. No specific agent-core 37–43 change is identified by these records. This is an update to the carried investigation, not a new causal claim.

The follow-up 17 source-address-fault request remains open with devstack. Its archive-navigation request is retained below as requested; the target reference is present in this baseline, and this lane needed no outside-owned edit. The historical wording and seven groups below are unchanged; their measurements describe earlier runs, not this one.

---

# Follow-up 17 owner assignments — 2026-11-03

### Requests to orchestrator

**2 new requests; seven carried groups.** The five F4 resource groups below are unchanged and are not re-asked. Devstack 30 supplied the prior survivor-continuation request; this harness now consumes that API, with its own live qualification recorded in [the survivor row](history/follow-up-17/F17-populated-holds.md#sibling-removal).

**2026-11-03 — devstack, separate source-address-fault hold:** the exported D-102 seam controls matching lease delivery, and supplies no independent source-address fault with deterministic restoration. Please supply a bounded TEST ONLY source-binding/address fault hook that preserves the control channel and records its injection/removal independently of TC failure. The shared export surface `tests/e2e/stack_exports.go` and shared node-side fault helpers are read-only for this lane. The separate source-address row is **not exercised**; TC errors in source_binding do not substitute for it.

**2026-11-03 — archive link outside lane ownership:** the byte-pinned follow-up 16 `OPERATORS.md` links `../../../poc/G1-app-checklist.md`, which now resolves to `docs/e2e/at/poc/G1-app-checklist.md`, outside owned paths. Please add a reference there to `docs/poc/G1-app-checklist.md`. All other relocated references have additive links within this lane. The [archive index](history/follow-up-16/INDEX.md) supplies a working direct checklist link; no pinned file or older archive was edited.

The seven entries below retain follow-up 16’s wording and counts; “fresh” in those carried entries refers to run 0ffb355a. Current observations follow the list.

1. **F4-03 — platform/operator:** real platform/QUIC, physical second NAT/roaming and PMTU resources; shared native fixtures/platform integrations are outside this lane. Owner-resource status unchanged.
2. **F4-04 — portal/devstack/operator:** rendered expired/suspended screens (`apps/portal`); API/traffic evidence does not certify UI. Owner-resource status unchanged.
3. **F4-10 — packaging/devstack/operator:** boot-flavour provenance and physical arm64 (`packaging`, shared boot fixtures). Cross-build is separate. Owner-resource status unchanged.
4. **F4-05/F4-07 — native integration/operator:** production nameserver distribution, Private DNS, browser DoH and Unicode UI/platform resources. Owner-resource status unchanged; devstack 23’s positive IDNA wire fixture is now consumed and is no longer a missing helper.
5. **F4-06 — usage/database/backup/business:** representative URL/privacy and remaining raw/rollup/file/backup retention surfaces and commercial feed approval (`internal/usage`, retention/backup components and operator resources). Owner-resource status unchanged.
6. **F11-01 / F14-01 — existing devstack native investigation:** fresh initial selection still fails its original deadline; a longer capture joins late SYNs and a loopback-source response. Three re-grants pass with full-population sessions. [Joined evidence](history/follow-up-16/F16-loopback-and-first-selection.json), [af718646 assessment](history/follow-up-16/F16-F14-01-assessment.md). No historical attribution or closure; no additional campaign is assigned.
7. **F11-03 / D-135 / D-136 / D-139 — existing orchestrator and agent-core track:** all 19 sibling losses count and the primary continuous-stream provisional check fails. The separate populated hold exercises two failed Applies, not a simultaneous continuous sibling fault injection. Intersection serving remains a design track. No outage ceiling or loss waiver is requested.

The primary continuous-stream failed-Apply row is exercised: three failed Applies intersect the original stream, and the unchanged provisional availability predicate fails with measurement_valid=false. All losses count; this is not a passing availability classification. No predicate or historical status changes.

[Results](summary.md), [gates](RUN.md), [prior assignments](history/follow-up-16/requests.md).
