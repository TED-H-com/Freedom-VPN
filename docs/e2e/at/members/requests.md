# Follow-up 19 owner assignments — 2026-11-03

### Requests to orchestrator

**Answered in Follow-up 19 (2026-11-03, after run `50117c68`, commits `6186fd60` / `306a1969`, `306a19693d694a0d4f26565705e471f8f0ee9448`):** the `2026-11-03 — finite-expiry failure cleanup` request below is resolved in `tests/e2e/at/members/finite_grant_test.go`, `tests/e2e/at/members/grant_restore.go`, and `tests/e2e/at/members/grant_restore_test.go` (along with the cross-case early-return restorations in `tests/e2e/at/members/reauthorization_test.go`, `tests/e2e/at/members/populated_hold_test.go`, `tests/e2e/at/members/devices_billing_test.go`, `tests/e2e/at/members/sibling_live_test.go`, and `tests/e2e/at/members/persistent_test.go`).

**2026-11-03 — shared e2e helper early-return cleanup (outside owned paths):** the Follow-up 19 cross-case early-return audit identified two helper sites outside `w4-at-members` owned paths (`tests/e2e/grant_fixture.go` lines 130–185 `Stack.ReplaceFiniteGrant`, which revokes live grants before creating/verifying the replacement grant and returns on create/replay/projection error without restoring a live grant; and `tests/e2e/native_underlay.go` lines 264–281 `Stack.PrepareNativeRouter1280`, which executes 10 sequential Docker network/container/route mutations and returns early on step `i` error without rolling back steps `0..i-1`). Callers in `tests/e2e/at/members/` now restore member grants via the public API after a failed `ReplaceFiniteGrant`; please assign helper-level rollback in `tests/e2e/grant_fixture.go` and `tests/e2e/native_underlay.go` to the devstack/shared-e2e owner if shared-helper rollback is desired.

---

# Follow-up 18 owner assignments — 2026-11-03

### Requests to orchestrator

**1 new request (answered in Follow-up 19 commits `6186fd60` / `306a1969`); both follow-up 17 requests and all seven carried groups remain tracked below.** F4-03, F4-04, F4-05/F4-07, F4-06 and F4-10 remain owner-resource rows and are not re-asked. F11-01 remains **OPEN and upstream-dependent (D-148)**; no product change is made here.

**2026-11-03 — finite-expiry failure cleanup (ANSWERED in Follow-up 19, commits `6186fd60` / `306a1969` after run `50117c68`):** schedule an acceptance-harness follow-up for `tests/e2e/at/members/finite_grant_test.go` so an early failed-readiness/positive-probe branch reliably performs its normal public-API grant restoration, retaining the failed original assertion. [Run 1's exact member/transition evidence](history/follow-up-18-run1/F18-AT09-precondition.json) proves the restoration was skipped before AT09 lost its admitted-device prerequisite. Orchestrator note 1 authorizes reader compatibility only; this failure branch was not changed. Run 2's actual restoration checks are recorded below. A passing normal path does not exercise cleanup after failure. *Resolution (Follow-up 19, 2026-11-03, commits `6186fd60` / `306a1969`):* `finiteExpiry` now restores member `A` (`3 authorized, 0 pending`) via `POST /api/v1/admin/members/{id}/grants` on both early-return paths (after failed `replaceFiniteGrant` and after failed `traceForwarding`), keeping the original failed check failed, recording `"finite expiry early-return restoration HTTP"` and `"finite expiry early-return restored 3/0"`, and preserving the normal path's single restoration (`"finite expiry restoration HTTP"` and `"finite expiry restored 3/0"`), with offline unit tests in `tests/e2e/at/members/grant_restore_test.go`.

Run 2 restoration checks: `[{"name": "finite expiry restoration HTTP", "status": "pass"}, {"name": "finite expiry restored 3/0", "status": "pass"}]`. [Original current precondition/restore records](F18-AT09-precondition.json).

The run-1 reader request is resolved in commit `4eb5d7d3`: only the missing `retire` phase was added; lifecycle validation remains fail-closed. [Value audit and tests](F18-reader-audit.md). The two authorized assembled runs are complete; no third run is requested or performed in this delivery.

The existing F11-03 / agent-core track receives [run 2's original AT09 checks](F18-AT09-subchecks.md) and [restart records](F18-AT09-restart.json). No timing-only attribution to a particular agent-core 37–43 change is made. Original run-1 failures remain in its archive.

The follow-up 17 independent source-address/binding-fault request stays open with devstack. Its archive-navigation request remains tracked below; the target reference is present in the baseline and needed no outside-owned edit here. Historical wording and counts below are retained as carried groups; their measurements describe earlier runs.

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

[Results](history/follow-up-17/summary.md), [gates](history/follow-up-17/RUN.md), [prior assignments](history/follow-up-16/requests.md).
