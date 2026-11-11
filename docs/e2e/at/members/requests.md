# Portal delivery for F4-04 — 2026-11-11

**Delivered:** [#512](https://github.com/TED-H-com/Freedom-VPN/pull/512), merge commit `f3bfb4a5c5ae5f1938a85d971dfe14caf8e3b4c9`, supplies rendered expired/suspended member service screens and live checks for F4-04. The [DOM regression tests](../../../../apps/portal/src/pages/__tests__/member_service.test.tsx) cover all eight entitlement states in zh-TW and English; [the portal guide](../../../portal/README.md) describes the checks and their limits.

**Live checks** in [`apps/portal/e2e/portal.spec.ts`](../../../../apps/portal/e2e/portal.spec.ts):

- `TestAT04_F404_SuspendedMemberScreenMatchesServer_Bilingual` creates a fresh member, suspends it through the administrator page, waits for the operation to report observed application and the member API to return `suspended` with `next_actions` `["contact_support"]`, then loads `/member/service` and compares the screen with the API in zh-TW and English.
- `TestAT04_F404_ExpiredMemberScreenMatchesServer_Bilingual` creates a fresh member, gives it a finite administrator grant, observes `active`, waits for the real deadline until the member API returns `expired` with `["renew_subscription"]`, then loads the page and compares it in both languages.
- `TestAT04_F404_DefaultMemberRestoredAfterInactiveScreens` reads the member API as its first action and checks that the default member is served again.

**Run:** `make -f mk/portal.mk portal-e2e` runs the live suite against real control and worker processes, a test OIDC provider and local fakes for NetBird and Stripe.

**CI receipt:** Portal CI [run 43014038790](https://github.com/TED-H-com/Freedom-VPN/actions/runs/43014038790), job “Verify Portal & Live E2E (Node 24 + Go 1.27.3)” on the merge commit above, concluded **success** (2026-11-11T11:41:39Z to 2026-11-11T11:44:53Z).

**Limits:** suspension is reached through the administrator page, with no payment attempted afterward; expiry comes from an administrator grant ending, not a paid period. The page is loaded after the member API reports the state, so an already open page crossing the deadline is not observed. These checks cover no real identity provider, payment provider, client apps, devices, traffic withdrawal, assembled Docker stack or production deployment. This is not a members acceptance run.

**Status:** F4-04's portal evidence is available. The members acceptance result does not change: AT04 stays as recorded until a members acceptance run consumes this evidence and fails closed if it is missing. Earlier sections keep their wording.

# Follow-up 24 — 2026-11-04

### Requests to orchestrator

**2026-11-04:** no new product or shared-harness request. Follow-up 23 request 1 is answered by devstack 36 **#476 / D-162**, and this owner run passes the unchanged freshness predicates. Request 2 is answered by agent-core 52 **#477** and the extracted enqueue/attempt records. F22-AT01's historical packet 158/161 evidence remains with the existing **D-148** owner; F11-01 stays **OPEN/upstream-dependent**. The carried owner-resource gaps and D-067's unexercised provisional path remain unchanged. No outside-owned fix or handover-test change is proposed.

[This run and all retained limits](followup24-2026-11-04.md), [F22-D146 cadence result](F24-observation-windows.json), [F22-AT01 historical packet evidence](F23-retrospective-F22-AT01.json).

The existing groups below retain their original wording and historical status. F4-03, F4-04, F4-05 / F4-07, F4-06 and F4-10 remain owner-resource rows and are not re-asked. Handover claims continue using frozen evidence; no handover-owned change was needed.

# Follow-up 23 — 2026-11-04

### Requests to orchestrator

2026-11-04:

- **F23-D146 / devstack:** the regular-node prepared config omits `observation_seconds` in `tests/e2e/cmd/fvpn-e2e-prepare/main.go:384–408`; `node/agent/config.go:145–146` defaults it to 30 s. The 5 s override at `tests/e2e/installer_fixture.go:119` applies to installer-node only. [Allowlisted retained config](F23-observation-config.json) and [complete row observation windows](F23-observation-windows.json) show a roughly 30 s cadence and the dual-row stale read. Please resolve the fixture/report-cadence mismatch with the owner. No freshness policy, 15 s bound, D-151 qualification or outside-owned file was changed.
- **F23-observation submissions / agent-core:** `node/agent/observe.go:285–365` and `node/internal/controlclient/client.go:248` emit no individual observation-submission timestamp/outcome. Existing exported logs cannot supply the requested attempts. Every row records this as unavailable, with start/end byte coverage and overwrite detection. Owner instrumentation is needed to distinguish scheduling, queueing, transport and rejection for each attempt; this sub-item stops at that missing export evidence.
- **F22-AT01 / existing D-148 owner:** the old node response is present at member underlay ingress, with an IPv6 loopback source label. [Pinned retrospective extraction](F23-retrospective-F22-AT01.json) places the first unconfirmed boundary after ingress and before WireGuard receive accounting. No per-packet kernel drop trace or proof of the response-source mechanism is retained. The native NetBird response path/shared fixture remains owner work; no unsupported causal file attribution or product change is made here. F11-01 stays **OPEN, upstream-dependent (D-148)**.

F4-03, F4-04, F4-05 / F4-07, F4-06 and F4-10 remain the existing owner-resource groups below; they are not re-asked. No handover claim requires changing a handover-owned file.

# Follow-up 22 — 2026-11-04

### Requests to orchestrator

- **2026-11-04 — baseline drift disposition:** the single permitted retry used assigned baseline `55f2765d` with agent-core 49/50; preflight `main` was already `0d944880` (netlab follow-up 35 #462). [Exact commits and affected code paths](F22-baseline-drift.json). This does not meet the literal latest-main requirement. No further live retry is authorized; retain this run and decide whether a separately assigned later-main measurement is needed.
- **2026-11-04 — F22-D146 freshness disposition, original predicates retained:** IPv6 receive-only 1280 indexes 49/51 remain blocked. The new later read identifies only observation age **22.393183 s ≥ 15 s**, with generation/incarnation matching and forwarding true/no reasons. [Original row and later-read classification](F22-native-MTU.json). The instrumentation is complete in this lane's own code. Any change to freshness policy, producer reporting or sampling procedure needs an explicit owner assignment; none is made here. F21's failed conjunct remains unknowable from its retained records.

**Completed assignments:** F21-AT08 now has sanitized selected-peer views before/after every bounded transfer, both node/member sides through existing hooks, and a native first-transfer capture. The new AT08 transfer passes without changing its origin/counter predicate; its old failure's cause remains unproven. F21-D146 now records the later-read failed condition without changing qualification. The G2 alias is supplied, resolving the three frozen run-2 links. These are not outstanding shared-hook requests.

**Carried, not re-asked:** F4-03, F4-04, F4-05/F4-07, F4-06 and F4-10 remain owner-resource groups. F11-01 remains **OPEN/upstream-dependent (D-148)**: this run's AT01 first transfer is uncounted; no cause or merged-change attribution is established. No F48-01 signature appears in retained logs, and no independent closure is claimed. The historical groups below retain their wording and outcomes; earlier requests describe their historical status.


---

# Follow-up 21 — 2026-11-03

### Requests to orchestrator

- **F21-AT08 — devstack/native investigation:** retain the blocked initial publication transfer at 03:54:29Z: route selected, five member-interface packets, no node/origin packet, zero origin requests and unchanged device upload/download counters. [Original forwarding](history/follow-up-21/F21-AT08-forwarding.json). Later success does not repair it. Cause and merged-change attribution remain unproven; no common cause with F11-01/F14-01 is assigned. Shared native fixtures and product paths are read-only here.
- **F21-D146 — review the missing serving-read qualification:** IPv6 receive-only 1280 indexes 49/51 remain blocked. All bytes, MSS rewrites, reverse datagrams and no-PTB checks pass, but the original rule read records `serving_read_ok=false`. Preserve this result; any follow-up diagnostics or predicate decision require an orchestrator assignment. [Original qualification](history/follow-up-21/F21-native-MTU.json). No predicate or product change is made.
- **Archive navigation outside ownership:** add `docs/e2e/at/poc/G2-followup34-2026-11-03.md` as an alias/reference to `docs/poc/G2-followup34-2026-11-03.md`. Three byte-pinned archived documents use `../../../poc/G2-followup34-2026-11-03.md`, which resolves there after relocation. [Working direct link and scope](history/follow-up-18-run2/INDEX.md). This outside-owned item is left to the orchestrator; no pinned byte is changed.

F4-03, F4-04, F4-05/F4-07, F4-06 and F4-10 stay owner-resource rows, not new requests. **F11-01 remains OPEN and upstream-dependent (D-148), with no product change.** The carried groups below keep their wording and historical outcomes. F48-01 remains assigned to agent-core 49; no matching `baseline_unverified`/`GENERATION_MISMATCH` signature is demonstrated by this run's retained restart logs, and no closure is claimed.

---

# Follow-up 19 owner assignments — 2026-11-03

### Requests to orchestrator

**Answered in Follow-up 19 (2026-11-03, after run `50117c68`, commits `6186fd60` / `306a1969`, `306a19693d694a0d4f26565705e471f8f0ee9448`):** the `2026-11-03 — finite-expiry failure cleanup` request below is resolved in `tests/e2e/at/members/finite_grant_test.go`, `tests/e2e/at/members/grant_restore.go`, and `tests/e2e/at/members/grant_restore_test.go` (along with the cross-case early-return restorations in `tests/e2e/at/members/reauthorization_test.go`, `tests/e2e/at/members/populated_hold_test.go`, `tests/e2e/at/members/devices_billing_test.go`, `tests/e2e/at/members/sibling_live_test.go`, and `tests/e2e/at/members/persistent_test.go`).

**2026-11-03 — shared e2e helper early-return cleanup: ANSWERED by devstack 33 (#441).** Both Follow-up 19 requests are resolved: `Stack.ReplaceFiniteGrant` now restores successfully revoked originals through the public API after a later failure, and `Stack.PrepareNativeRouter1280` rolls back completed setup steps in reverse order. See the dated [shared-owner adoption notes](../../RESPONSES.md#2026-11-03--members-19-finite-grant-early-return-recovery-follow-up-33). Follow-up 20 retains the explicitly permitted same-plan fallback, including its possible one-hour extension and additional grant/audit, and keeps all device, worker-operation and fence verification. Normal expiry restoration and the successful downgrade's later return remain necessary. No new live proof is claimed.

**2026-11-03 — Follow-up 20 routing after r27 (already assigned):** agent-core owns H-1 (false restart `TC_FOREIGN_FILTER`), M-1 (closed-recovery renewal delay) and L-1 (repeated quarantine/baseline writes). The members predicates and offline proof are documented in [F20-r27-predicates.md](history/follow-up-18-run2/F20-r27-predicates.md). The new restart foreign-filter check fails on run `50117c68` evidence. The next assembled run remains for the orchestrator to schedule after H-1 and M-1 merge; this follow-up performs none.

---

# Follow-up 18 owner assignments — 2026-11-03

### Requests to orchestrator

**1 new request (answered in Follow-up 19 commits `6186fd60` / `306a1969`); both follow-up 17 requests and all seven carried groups remain tracked below.** F4-03, F4-04, F4-05/F4-07, F4-06 and F4-10 remain owner-resource rows and are not re-asked. F11-01 remains **OPEN and upstream-dependent (D-148)**; no product change is made here.

**2026-11-03 — finite-expiry failure cleanup (ANSWERED in Follow-up 19, commits `6186fd60` / `306a1969` after run `50117c68`):** schedule an acceptance-harness follow-up for `tests/e2e/at/members/finite_grant_test.go` so an early failed-readiness/positive-probe branch reliably performs its normal public-API grant restoration, retaining the failed original assertion. [Run 1's exact member/transition evidence](history/follow-up-18-run1/F18-AT09-precondition.json) proves the restoration was skipped before AT09 lost its admitted-device prerequisite. Orchestrator note 1 authorizes reader compatibility only; this failure branch was not changed. Run 2's actual restoration checks are recorded below. A passing normal path does not exercise cleanup after failure. *Resolution (Follow-up 19, 2026-11-03, commits `6186fd60` / `306a1969`):* `finiteExpiry` now restores member `A` (`3 authorized, 0 pending`) via `POST /api/v1/admin/members/{id}/grants` on both early-return paths (after failed `replaceFiniteGrant` and after failed `traceForwarding`), keeping the original failed check failed, recording `"finite expiry early-return restoration HTTP"` and `"finite expiry early-return restored 3/0"`, and preserving the normal path's single restoration (`"finite expiry restoration HTTP"` and `"finite expiry restored 3/0"`), with offline unit tests in `tests/e2e/at/members/grant_restore_test.go`.

Run 2 restoration checks: `[{"name": "finite expiry restoration HTTP", "status": "pass"}, {"name": "finite expiry restored 3/0", "status": "pass"}]`. [Original current precondition/restore records](history/follow-up-18-run2/F18-AT09-precondition.json).

The run-1 reader request is resolved in commit `4eb5d7d3`: only the missing `retire` phase was added; lifecycle validation remains fail-closed. [Value audit and tests](history/follow-up-18-run2/F18-reader-audit.md). The two authorized assembled runs are complete; no third run is requested or performed in this delivery.

The existing F11-03 / agent-core track receives [run 2's original AT09 checks](history/follow-up-18-run2/F18-AT09-subchecks.md) and [restart records](history/follow-up-18-run2/F18-AT09-restart.json). No timing-only attribution to a particular agent-core 37–43 change is made. Original run-1 failures remain in its archive.

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
