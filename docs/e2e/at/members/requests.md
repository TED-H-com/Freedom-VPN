# Follow-up 17 owner assignments — 2026-11-03

### Requests to orchestrator

**1 new request; seven carried groups.** The five F4 resource groups below are unchanged and are not re-asked. Devstack 30 supplied the prior survivor-continuation request; this harness now consumes that API, with its own live qualification recorded in [the survivor row](F17-populated-holds.md#sibling-removal).

**2026-11-03 — devstack, separate source-address-fault hold:** the exported D-102 seam controls matching lease delivery, and supplies no independent source-address fault with deterministic restoration. Please supply a bounded TEST ONLY source-binding/address fault hook that preserves the control channel and records its injection/removal independently of TC failure. The shared export surface `tests/e2e/stack_exports.go` and shared node-side fault helpers are read-only for this lane. The separate source-address row is **not exercised**; TC errors in source_binding do not substitute for it.

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
