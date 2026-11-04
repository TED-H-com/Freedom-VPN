package members

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

type grantHTTPResponse struct {
	status int
	body   []byte
	code   string
}

type checkRecorder interface {
	check(name, expected, actual, command string, ok bool)
}

// finiteExpirySteps injects the operations called by finiteExpiry so the exact
// branching and check-recording logic runs both in the live suite and in
// offline unit tests without a running stack.
//
// Grant state and product entitlement rule accepting POST /admin/members/{id}/grants
// (plan_revision_id = l.planRev, valid_until = now + 1h) on each path:
//
//  1. Early return after failed replaceFiniteGrant:
//     Stack.ReplaceFiniteGrant may have failed before revoking the member's
//     prior l.planRev grant (prior l.planRev grant still live; 3 authorized,
//     0 pending), after revoking the prior grant and before creating the 75s
//     grant (no live grant; 0 authorized, 3 pending), or after creating the 75s
//     l.planRev grant when replay/operation/projection verification failed (75s
//     l.planRev grant live until valid_until, or expired if polling exceeded
//     valid_until). In every sub-state, members.Service.Grant
//     (internal/members/service.go:460-466) queries:
//     SELECT EXISTS(SELECT 1 FROM entitlement_grants WHERE member_id=$1
//     AND revoked_at IS NULL AND plan_revision_id<>$2 AND valid_from<$4 AND valid_until>$3)
//     which rejects overlapping active grants (409 STATE_CONFLICT) only when
//     plan_revision_id <> $2 (different plan revisions). Because the restoration
//     POST uses the same plan_revision_id (l.planRev), plan_revision_id <> $2 is
//     false when a prior or 75s l.planRev grant is still live, and no row
//     matches valid_until > now when no grant is live. entitlement.Compute
//     (internal/entitlement/engine.go:194-227) merges overlapping same-plan
//     grants containing now into one active component with EffectiveServiceEnd =
//     now + 1h (or activates the new [now, now+1h) grant alone when no grant was
//     live), and reconcileDevices (internal/entitlement/devices.go:57-99)
//     extends open bindings on already-authorized devices and re-admits any
//     previously authorized pending devices (authorized_at IS NOT NULL, no
//     fence) in created_at ASC, id ASC order back to 3 authorized, 0 pending.
//
//  2. Early return after failed traceForwarding:
//     The 75s l.planRev grant is installed and is either still live until its
//     valid_until (now < valid_until; 3 authorized, 0 pending), or already
//     expired (valid_until <= now; 0 authorized, 3 pending) if forwardingGate /
//     traceForwarding timed out past valid_until (as in run 118271c1). When
//     still live, members.Service.Grant accepts POST /admin/members/{id}/grants
//     under the same-plan overlap rule (plan_revision_id <> $2 is false) and
//     entitlement.Compute merges [valid_from, valid_until) with [now, now+1h)
//     to extend EffectiveServiceEnd to now + 1h. When already expired,
//     members.Service.Grant accepts the new grant because valid_until > now is
//     false on the expired row, and reconcileDevices re-admits all 3 pending
//     devices back to 3 authorized, 0 pending.
//
//  3. Normal path after real valid_until expiry:
//     The 75s l.planRev grant has expired in real UTC time (valid_until <= now;
//     0 authorized, 3 pending). POST /admin/members/{id}/grants creates a fresh
//     active grant on l.planRev for [now, now+1h), and reconcileDevices
//     re-admits all 3 pending devices back to 3 authorized, 0 pending.
type finiteExpirySteps struct {
	until              time.Time
	replaceFiniteGrant func(until time.Time) bool
	forwardingGate     func()
	traceForwarding    func() bool
	runNormalExpiry    func(until time.Time)
	grantRestore       func() grantHTTPResponse
	waitDevices        func() (restored bool, authorized, pending int)
	waitOperation      func(opID uuid.UUID) (completed bool, operation grantHTTPResponse)
}

func runFiniteExpiry(r checkRecorder, s finiteExpirySteps) {
	if !s.replaceFiniteGrant(s.until) {
		restoreFiniteGrantOnEarlyReturn(r, s)
		return
	}
	s.forwardingGate()
	if !s.traceForwarding() {
		restoreFiniteGrantOnEarlyReturn(r, s)
		return
	}
	s.runNormalExpiry(s.until)
	// The hook deliberately requires a live grant. After expiry, normal public
	// creation restores service; pretending an expired grant is live would weaken it.
	restore := s.grantRestore()
	r.check("finite expiry restoration HTTP", "202", string(restore.body), "POST /admin/members/{id}/grants after expiry", restore.status == 202)
	var op api.Operation
	_ = json.Unmarshal(restore.body, &op)
	restored, _, _ := s.waitDevices()
	completed, operation := s.waitOperation(op.OperationId)
	r.check("finite expiry restored 3/0", "three authorized, zero pending; worker succeeded", fmt.Sprintf("restored=%t; operation=%s", restored, operation.body), "read devices and worker operation after production grant creation", restored && completed)
	s.forwardingGate()
	s.traceForwarding()
}

// Follow-up 20: retain the fallback even if devstack 33 restored the originals.
// The 2026-11-03 adoption note explicitly permits this same-plan POST; it can
// extend validity to our one-hour end and creates another grant/audit. Keeping
// it also covers expiry during later readiness checks. Device/operation checks
// below and the limit restoration's fence checks remain mandatory.
func restoreFiniteGrantOnEarlyReturn(r checkRecorder, s finiteExpirySteps) {
	restore := s.grantRestore()
	r.check("finite expiry early-return restoration HTTP", "202", string(restore.body), "POST /admin/members/{id}/grants on early return", restore.status == 202)
	var op api.Operation
	_ = json.Unmarshal(restore.body, &op)
	restored, authorized, pending := s.waitDevices()
	completed, operation := s.waitOperation(op.OperationId)
	r.check("finite expiry early-return restored 3/0", "three authorized, zero pending; worker succeeded", fmt.Sprintf("restored=%t; authorized=%d; pending=%d; operation=%s", restored, authorized, pending, operation.body), "read devices and worker operation after early-return production grant creation", restored && authorized == 3 && pending == 0 && completed)
}

// reauthorizationLimitSteps injects the steps around the lower-plan downgrade
// and D-077 fence insertion in reauthorizationLimit (AT02).
//
// Grant/device state and product entitlement rule on each early return:
//
//  1. Early return after failed replaceLowerGrant:
//     A persistent flow was started, and ReplaceFiniteGrant attempted to replace
//     l.a's l.planRev (limit 3) grant with lower (limit 2, plan_revision_id <>
//     l.planRev). It may have left l.a with its prior l.planRev grant live, with
//     no live grant (0 authorized, 3 pending), or with the lower grant live
//     (2 authorized, 1 pending). Because members.Service.Grant rejects
//     POST /admin/members/{id}/grants on l.planRev with 409 STATE_CONFLICT if a
//     live grant on a different plan_revision_id (lower) exists, restoration
//     first transitions from any live non-l.planRev grant back to l.planRev via
//     replaceOriginalPlan (when hasLiveOtherPlanGrant is true) or issues
//     POST /admin/members/{id}/grants directly via grantRestore (when no live
//     different-plan grant exists), after stopping the persistent flow.
//
//  2. Early return after failed fenceInsert:
//     l.a is on lower (2 authorized, 1 pending) and id may also have a fence row
//     if fenceInsert failed after/with a present fence. Restoration transitions
//     l.a back to l.planRev, clears any fence on id via
//     POST /admin/members/{id}/devices/{device_id}/authorize, and verifies
//     3 authorized, 0 pending.
type reauthorizationLimitSteps struct {
	startPersistent       func()
	abortPersistent       func()
	replaceLowerGrant     func() bool
	afterLowerGrant       func()
	fenceInsert           func() bool
	afterFenceInsert      func()
	hasLiveOtherPlanGrant func() bool
	replaceOriginalPlan   func() bool
	grantRestore          func() grantHTTPResponse
	fenceCount            func() int
	authorizeDevice       func() grantHTTPResponse
	waitDevices           func() (restored bool, authorized, pending int)
	waitOperation         func(opID uuid.UUID) (completed bool, operation grantHTTPResponse)
}

func runReauthorizationLimit(r checkRecorder, s reauthorizationLimitSteps) {
	s.startPersistent()
	if !s.replaceLowerGrant() {
		s.abortPersistent()
		r.check("D-081 setup", "real plan downgrade", "failed", "production grant transition", false)
		restoreLimitGrantOnEarlyReturn(r, s)
		return
	}
	s.afterLowerGrant()
	if !s.fenceInsert() {
		restoreLimitGrantOnEarlyReturn(r, s)
		return
	}
	s.afterFenceInsert()
}

func restoreLimitGrantOnEarlyReturn(r checkRecorder, s reauthorizationLimitSteps) {
	var completed bool
	var operation grantHTTPResponse
	if s.hasLiveOtherPlanGrant != nil && s.hasLiveOtherPlanGrant() {
		replaced := s.replaceOriginalPlan()
		r.check("D-081 early-return restoration HTTP", "true", fmt.Sprintf("replaced=%t", replaced), "Stack.ReplaceFiniteGrant back to original plan on early return", replaced)
		completed = replaced
	} else {
		restore := s.grantRestore()
		r.check("D-081 early-return restoration HTTP", "202", string(restore.body), "POST /admin/members/{id}/grants on D-081 early return", restore.status == 202)
		var op api.Operation
		_ = json.Unmarshal(restore.body, &op)
		completed, operation = s.waitOperation(op.OperationId)
	}
	if s.fenceCount != nil && s.fenceCount() > 0 && s.authorizeDevice != nil {
		auth := s.authorizeDevice()
		r.check("D-081 early-return fence authorization", "202", string(auth.body), "POST /admin/members/{id}/devices/{device_id}/authorize on early return", auth.status == 202)
	}
	restored, authorized, pending := s.waitDevices()
	r.check("D-081 early-return restored 3/0", "three authorized, zero pending; worker succeeded", fmt.Sprintf("restored=%t; authorized=%d; pending=%d; operation=%s", restored, authorized, pending, operation.body), "read devices and worker operation after D-081 early-return restoration", restored && authorized == 3 && pending == 0 && completed)
}

type reauthorizationSteps struct {
	fenceInsert      func() bool
	afterFenceInsert func()
	fenceCount       func() int
	authorizeDevice  func() grantHTTPResponse
	waitAuthorized   func() bool
}

func runReauthorization(r checkRecorder, s reauthorizationSteps) {
	if !s.fenceInsert() {
		if s.fenceCount != nil && s.fenceCount() > 0 && s.authorizeDevice != nil {
			auth := s.authorizeDevice()
			ready := true
			if s.waitAuthorized != nil {
				ready = s.waitAuthorized()
			}
			r.check("D-077 early-return fence restoration", "202 and device authorized with zero fences", fmt.Sprintf("status=%d; ready=%t; fence_after=%d; body=%s", auth.status, ready, s.fenceCount(), auth.body), "POST /admin/members/{id}/devices/{device_id}/authorize on early return", auth.status == 202 && ready && s.fenceCount() == 0)
		}
		return
	}
	s.afterFenceInsert()
}

type populatedHoldCleanupSteps struct {
	restoreHoldMember   func()
	deviceNeedsRestore  func() bool
	authorizeHoldDevice func() grantHTTPResponse
	publishGroupA       func()
}

func runPopulatedHoldCleanup(r checkRecorder, s populatedHoldCleanupSteps) {
	s.restoreHoldMember()
	if s.deviceNeedsRestore != nil && s.deviceNeedsRestore() && s.authorizeHoldDevice != nil {
		got := s.authorizeHoldDevice()
		r.check("populated hold early-return restore original B binding", "202 explicit device authorization", fmt.Sprintf("HTTP %d %s", got.status, got.code), "POST admin member device authorize in hold cleanup; no device or binding row edits", got.status == 202)
	}
	s.publishGroupA()
}

func runEntitlementsPrecheck(hasPaid, hasSecondExit bool, disableSecondExit func(), block func(string)) bool {
	if !hasPaid {
		if hasSecondExit && disableSecondExit != nil {
			disableSecondExit()
		}
		if block != nil {
			block("AT03 paid session unavailable")
		}
		return false
	}
	return true
}

func stopSiblingEarly(finished *bool, stopSamples func(), stopNative func()) {
	if finished != nil && *finished {
		return
	}
	if finished != nil {
		*finished = true
	}
	if stopSamples != nil {
		stopSamples()
	}
	if stopNative != nil {
		stopNative()
	}
}
