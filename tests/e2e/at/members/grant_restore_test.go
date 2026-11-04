package members

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type stepEntry struct {
	Name     string
	Status   string
	Expected string
	Actual   string
	Command  string
}

type stepLog struct {
	failed bool
	steps  []stepEntry
}

func (l *stepLog) check(name, expected, actual, command string, ok bool) {
	status := "pass"
	if !ok {
		status = "fail"
		l.failed = true
	}
	l.steps = append(l.steps, stepEntry{
		Name:     name,
		Status:   status,
		Expected: expected,
		Actual:   actual,
		Command:  command,
	})
}

func TestAT02_FiniteExpiryRestoresOnEveryEarlyReturnAndOnceOnNormalPath(t *testing.T) {
	until := time.Date(2026, 11, 3, 22, 1, 15, 0, time.UTC)
	opID := uuid.MustParse("019a4b00-0000-7000-8000-000000000001")
	createBody := []byte(`{"operation_id":"` + opID.String() + `","kind":"member.grant","status":"pending"}`)
	succeededBody := []byte(`{"operation_id":"` + opID.String() + `","kind":"member.grant","status":"succeeded"}`)

	t.Run("early_return_after_failed_replaceFiniteGrant", func(t *testing.T) {
		for _, subState := range []struct {
			name              string
			initialAuthorized int
			initialPending    int
		}{
			{"prior_grant_still_live", 3, 0},
			{"shared_helper_restored_originals_same_plan_fallback_retained", 3, 0},
			{"revoked_without_replacement", 0, 3},
			{"finite_grant_installed_projection_timed_out", 3, 0},
		} {
			t.Run(subState.name, func(t *testing.T) {
				rec := &stepLog{}
				authorized, pending := subState.initialAuthorized, subState.initialPending
				var gateCalls, traceCalls, normalExpiryCalls, restoreCalls, waitDevCalls, waitOpCalls int

				runFiniteExpiry(rec, finiteExpirySteps{
					until: until,
					replaceFiniteGrant: func(gotUntil time.Time) bool {
						if !gotUntil.Equal(until) {
							t.Fatalf("until = %v, want %v", gotUntil, until)
						}
						rec.check(
							"finite grant production transition",
							"accepted, replayed and worker projected; finite deadline; export partial result on error",
							`{"error":"finite grant: worker operation/projection deadline"}`,
							"Stack.ReplaceFiniteGrant: authenticated domain revoke + public grant POST + worker operation GET; no projection writes",
							false,
						)
						return false
					},
					forwardingGate: func() {
						gateCalls++
					},
					traceForwarding: func() bool {
						traceCalls++
						return true
					},
					runNormalExpiry: func(time.Time) {
						normalExpiryCalls++
					},
					grantRestore: func() grantHTTPResponse {
						restoreCalls++
						authorized, pending = 3, 0
						return grantHTTPResponse{status: 202, body: createBody}
					},
					waitDevices: func() (bool, int, int) {
						waitDevCalls++
						return authorized == 3 && pending == 0, authorized, pending
					},
					waitOperation: func(gotID uuid.UUID) (bool, grantHTTPResponse) {
						waitOpCalls++
						if gotID != opID {
							t.Fatalf("opID = %v, want %v", gotID, opID)
						}
						return true, grantHTTPResponse{status: 200, body: succeededBody}
					},
				})

				if !rec.failed {
					t.Fatal("record must remain failed when replaceFiniteGrant fails")
				}
				if restoreCalls != 1 || waitDevCalls != 1 || waitOpCalls != 1 {
					t.Fatalf("restore/waitDev/waitOp = %d/%d/%d, want 1/1/1", restoreCalls, waitDevCalls, waitOpCalls)
				}
				if gateCalls != 0 || traceCalls != 0 || normalExpiryCalls != 0 {
					t.Fatalf("gate/trace/normalExpiry = %d/%d/%d, want 0/0/0", gateCalls, traceCalls, normalExpiryCalls)
				}
				if authorized != 3 || pending != 0 {
					t.Fatalf("final devices = %d/%d, want 3/0", authorized, pending)
				}
				if len(rec.steps) != 3 {
					t.Fatalf("steps = %d, want 3: %+v", len(rec.steps), rec.steps)
				}
				orig := rec.steps[0]
				if orig.Name != "finite grant production transition" || orig.Status != "fail" ||
					orig.Expected != "accepted, replayed and worker projected; finite deadline; export partial result on error" ||
					orig.Actual != `{"error":"finite grant: worker operation/projection deadline"}` {
					t.Fatalf("original failed check altered: %+v", orig)
				}
				httpStep := rec.steps[1]
				if httpStep.Name != "finite expiry early-return restoration HTTP" || httpStep.Status != "pass" ||
					httpStep.Expected != "202" || httpStep.Actual != string(createBody) {
					t.Fatalf("unexpected early-return HTTP check: %+v", httpStep)
				}
				restoredStep := rec.steps[2]
				if restoredStep.Name != "finite expiry early-return restored 3/0" || restoredStep.Status != "pass" ||
					restoredStep.Expected != "three authorized, zero pending; worker succeeded" ||
					!strings.Contains(restoredStep.Actual, "restored=true; authorized=3; pending=0; operation="+string(succeededBody)) {
					t.Fatalf("unexpected early-return restored check: %+v", restoredStep)
				}
			})
		}
	})

	t.Run("early_return_after_failed_traceForwarding", func(t *testing.T) {
		for _, subState := range []struct {
			name              string
			grantExpired      bool
			initialAuthorized int
			initialPending    int
		}{
			{"grant_still_live_until_valid_until", false, 3, 0},
			{"grant_expired_during_readiness_timeout", true, 0, 3},
		} {
			t.Run(subState.name, func(t *testing.T) {
				rec := &stepLog{}
				authorized, pending := subState.initialAuthorized, subState.initialPending
				var gateCalls, traceCalls, normalExpiryCalls, restoreCalls, waitDevCalls, waitOpCalls int

				runFiniteExpiry(rec, finiteExpirySteps{
					until: until,
					replaceFiniteGrant: func(time.Time) bool {
						rec.check(
							"finite grant production transition",
							"accepted, replayed and worker projected; finite deadline; export partial result on error",
							`{"before_authorized":3,"before_pending":0}`,
							"Stack.ReplaceFiniteGrant: authenticated domain revoke + public grant POST + worker operation GET; no projection writes",
							true,
						)
						return true
					},
					forwardingGate: func() {
						gateCalls++
					},
					traceForwarding: func() bool {
						traceCalls++
						rec.check(
							"populated-grant serving readiness",
							"fresh accepted serving observation; desired=applied=observed lease generation; unexpired populated lease, live intended binding and same-boot producer serving row",
							`{"ready":false,"last_stage":"producer_stream_export","reader_error":"producer_stream_unavailable"}`,
							"read-only SQL + local desired state + streamed ExportContainerFile; stop positive probe when unavailable",
							false,
						)
						return false
					},
					runNormalExpiry: func(time.Time) {
						normalExpiryCalls++
					},
					grantRestore: func() grantHTTPResponse {
						restoreCalls++
						authorized, pending = 3, 0
						return grantHTTPResponse{status: 202, body: createBody}
					},
					waitDevices: func() (bool, int, int) {
						waitDevCalls++
						return authorized == 3 && pending == 0, authorized, pending
					},
					waitOperation: func(gotID uuid.UUID) (bool, grantHTTPResponse) {
						waitOpCalls++
						if gotID != opID {
							t.Fatalf("opID = %v, want %v", gotID, opID)
						}
						return true, grantHTTPResponse{status: 200, body: succeededBody}
					},
				})

				if !rec.failed {
					t.Fatal("record must remain failed when traceForwarding fails")
				}
				if gateCalls != 1 || traceCalls != 1 || normalExpiryCalls != 0 {
					t.Fatalf("gate/trace/normalExpiry = %d/%d/%d, want 1/1/0", gateCalls, traceCalls, normalExpiryCalls)
				}
				if restoreCalls != 1 || waitDevCalls != 1 || waitOpCalls != 1 {
					t.Fatalf("restore/waitDev/waitOp = %d/%d/%d, want 1/1/1", restoreCalls, waitDevCalls, waitOpCalls)
				}
				if authorized != 3 || pending != 0 {
					t.Fatalf("final devices = %d/%d, want 3/0", authorized, pending)
				}
				if len(rec.steps) != 4 {
					t.Fatalf("steps = %d, want 4: %+v", len(rec.steps), rec.steps)
				}
				failedStep := rec.steps[1]
				if failedStep.Name != "populated-grant serving readiness" || failedStep.Status != "fail" ||
					failedStep.Expected != "fresh accepted serving observation; desired=applied=observed lease generation; unexpired populated lease, live intended binding and same-boot producer serving row" ||
					failedStep.Actual != `{"ready":false,"last_stage":"producer_stream_export","reader_error":"producer_stream_unavailable"}` {
					t.Fatalf("original failed readiness check altered: %+v", failedStep)
				}
				if rec.steps[2].Name != "finite expiry early-return restoration HTTP" || rec.steps[2].Status != "pass" {
					t.Fatalf("unexpected step[2]: %+v", rec.steps[2])
				}
				if rec.steps[3].Name != "finite expiry early-return restored 3/0" || rec.steps[3].Status != "pass" ||
					!strings.Contains(rec.steps[3].Actual, "restored=true; authorized=3; pending=0; operation="+string(succeededBody)) {
					t.Fatalf("unexpected step[3]: %+v", rec.steps[3])
				}
			})
		}
	})

	t.Run("normal_path_restores_once_with_unchanged_checks", func(t *testing.T) {
		rec := &stepLog{}
		authorized, pending := 3, 0
		var gateCalls, traceCalls, normalExpiryCalls, restoreCalls, waitDevCalls, waitOpCalls int

		runFiniteExpiry(rec, finiteExpirySteps{
			until: until,
			replaceFiniteGrant: func(time.Time) bool {
				rec.check(
					"finite grant production transition",
					"accepted, replayed and worker projected; finite deadline; export partial result on error",
					`{"before_authorized":3,"before_pending":0}`,
					"Stack.ReplaceFiniteGrant: authenticated domain revoke + public grant POST + worker operation GET; no projection writes",
					true,
				)
				return true
			},
			forwardingGate: func() {
				gateCalls++
			},
			traceForwarding: func() bool {
				traceCalls++
				return true
			},
			runNormalExpiry: func(gotUntil time.Time) {
				normalExpiryCalls++
				authorized, pending = 0, 3
				rec.check(
					"real finite valid_until expiry",
					"real UTC expiry releases all three slots without a fallback plan",
					`{"expired":true,"authorized":0,"pending":3}`,
					"poll production entitlement and device projection; no clock or row mutation",
					true,
				)
				rec.check(
					"real finite expiry audited withdrawal",
					"three authorized-to-pending audit events",
					`[]`,
					"read expiry device.revoke audit rows; no synthetic audit input",
					true,
				)
			},
			grantRestore: func() grantHTTPResponse {
				restoreCalls++
				authorized, pending = 3, 0
				return grantHTTPResponse{status: 202, body: createBody}
			},
			waitDevices: func() (bool, int, int) {
				waitDevCalls++
				return authorized == 3 && pending == 0, authorized, pending
			},
			waitOperation: func(gotID uuid.UUID) (bool, grantHTTPResponse) {
				waitOpCalls++
				if gotID != opID {
					t.Fatalf("opID = %v, want %v", gotID, opID)
				}
				return true, grantHTTPResponse{status: 200, body: succeededBody}
			},
		})

		if rec.failed {
			t.Fatalf("normal path unexpectedly failed: %+v", rec.steps)
		}
		if restoreCalls != 1 || waitDevCalls != 1 || waitOpCalls != 1 || normalExpiryCalls != 1 {
			t.Fatalf("restore/waitDev/waitOp/normalExpiry = %d/%d/%d/%d, want 1/1/1/1", restoreCalls, waitDevCalls, waitOpCalls, normalExpiryCalls)
		}
		if gateCalls != 2 || traceCalls != 2 {
			t.Fatalf("gate/trace = %d/%d, want 2/2", gateCalls, traceCalls)
		}
		if len(rec.steps) != 5 {
			t.Fatalf("steps = %d, want 5: %+v", len(rec.steps), rec.steps)
		}
		if rec.steps[3].Name != "finite expiry restoration HTTP" ||
			rec.steps[3].Expected != "202" ||
			rec.steps[3].Actual != string(createBody) ||
			rec.steps[3].Command != "POST /admin/members/{id}/grants after expiry" {
			t.Fatalf("normal restoration HTTP check changed: %+v", rec.steps[3])
		}
		if rec.steps[4].Name != "finite expiry restored 3/0" ||
			rec.steps[4].Expected != "three authorized, zero pending; worker succeeded" ||
			rec.steps[4].Actual != "restored=true; operation="+string(succeededBody) ||
			rec.steps[4].Command != "read devices and worker operation after production grant creation" {
			t.Fatalf("normal restored 3/0 check changed: %+v", rec.steps[4])
		}
	})
}

func TestAT02_AuditedEarlyReturnRestorations(t *testing.T) {
	opID := uuid.MustParse("019a4b00-0000-7000-8000-000000000002")
	createBody := []byte(`{"operation_id":"` + opID.String() + `","kind":"member.grant","status":"pending"}`)
	succeededBody := []byte(`{"operation_id":"` + opID.String() + `","kind":"member.grant","status":"succeeded"}`)

	t.Run("reauthorizationLimit_early_return_after_failed_lower_grant", func(t *testing.T) {
		rec := &stepLog{}
		authorized, pending := 0, 3
		var started, aborted, restoredCalls, afterLower, afterFence int

		runReauthorizationLimit(rec, reauthorizationLimitSteps{
			startPersistent:   func() { started++ },
			abortPersistent:   func() { aborted++ },
			replaceLowerGrant: func() bool { return false },
			afterLowerGrant:   func() { afterLower++ },
			fenceInsert:       func() bool { return true },
			afterFenceInsert:  func() { afterFence++ },
			hasLiveOtherPlanGrant: func() bool {
				return false
			},
			grantRestore: func() grantHTTPResponse {
				restoredCalls++
				authorized, pending = 3, 0
				return grantHTTPResponse{status: 202, body: createBody}
			},
			waitDevices: func() (bool, int, int) {
				return authorized == 3 && pending == 0, authorized, pending
			},
			waitOperation: func(uuid.UUID) (bool, grantHTTPResponse) {
				return true, grantHTTPResponse{status: 200, body: succeededBody}
			},
		})

		if !rec.failed || started != 1 || aborted != 1 || restoredCalls != 1 || afterLower != 0 || afterFence != 0 {
			t.Fatalf("unexpected state: failed=%t started=%d aborted=%d restored=%d afterLower=%d afterFence=%d", rec.failed, started, aborted, restoredCalls, afterLower, afterFence)
		}
		if rec.steps[0].Name != "D-081 setup" || rec.steps[0].Status != "fail" || rec.steps[0].Expected != "real plan downgrade" || rec.steps[0].Actual != "failed" {
			t.Fatalf("original D-081 setup check altered: %+v", rec.steps[0])
		}
		if rec.steps[1].Name != "D-081 early-return restoration HTTP" || rec.steps[2].Name != "D-081 early-return restored 3/0" {
			t.Fatalf("missing D-081 early-return restoration checks: %+v", rec.steps)
		}
	})

	t.Run("reauthorizationLimit_early_return_after_failed_fenceInsert_with_lower_plan_and_fence", func(t *testing.T) {
		rec := &stepLog{}
		authorized, pending, fences := 2, 1, 1
		var replaceOrigCalls, authCalls, afterFence int

		runReauthorizationLimit(rec, reauthorizationLimitSteps{
			startPersistent:   func() {},
			abortPersistent:   func() {},
			replaceLowerGrant: func() bool { return true },
			afterLowerGrant:   func() {},
			fenceInsert: func() bool {
				rec.check("D-077 restore fence input", "0 -> 1 fence; no device/binding/observation edits", "before=1; after=1; inserted=false", "lab Postgres INSERT device_authorization_fences per migration 00188 / restore-drill §6.3", false)
				return false
			},
			afterFenceInsert:      func() { afterFence++ },
			hasLiveOtherPlanGrant: func() bool { return true },
			replaceOriginalPlan: func() bool {
				replaceOrigCalls++
				return true
			},
			fenceCount: func() int { return fences },
			authorizeDevice: func() grantHTTPResponse {
				authCalls++
				fences = 0
				authorized, pending = 3, 0
				return grantHTTPResponse{status: 202, body: []byte(`{"status":"pending"}`)}
			},
			waitDevices: func() (bool, int, int) {
				return authorized == 3 && pending == 0, authorized, pending
			},
		})

		if !rec.failed || replaceOrigCalls != 1 || authCalls != 1 || afterFence != 0 || authorized != 3 || pending != 0 || fences != 0 {
			t.Fatalf("unexpected state: failed=%t replaceOrig=%d auth=%d afterFence=%d authz=%d pending=%d fences=%d", rec.failed, replaceOrigCalls, authCalls, afterFence, authorized, pending, fences)
		}
		if rec.steps[0].Name != "D-077 restore fence input" || rec.steps[0].Status != "fail" {
			t.Fatalf("original fenceInsert check altered: %+v", rec.steps[0])
		}
		if rec.steps[1].Name != "D-081 early-return restoration HTTP" || rec.steps[2].Name != "D-081 early-return fence authorization" || rec.steps[3].Name != "D-081 early-return restored 3/0" {
			t.Fatalf("unexpected D-081 fence-failure restoration steps: %+v", rec.steps)
		}
	})

	t.Run("reauthorizationLimit_normal_path_runs_once", func(t *testing.T) {
		rec := &stepLog{}
		var normalRan, earlyRestore int
		runReauthorizationLimit(rec, reauthorizationLimitSteps{
			startPersistent:   func() {},
			abortPersistent:   func() { earlyRestore++ },
			replaceLowerGrant: func() bool { return true },
			afterLowerGrant:   func() {},
			fenceInsert:       func() bool { return true },
			afterFenceInsert:  func() { normalRan++ },
			grantRestore:      func() grantHTTPResponse { earlyRestore++; return grantHTTPResponse{status: 202} },
		})
		if normalRan != 1 || earlyRestore != 0 || len(rec.steps) != 0 {
			t.Fatalf("normalRan=%d earlyRestore=%d steps=%+v", normalRan, earlyRestore, rec.steps)
		}
	})

	t.Run("reauthorization_early_return_clears_fence_and_normal_path_runs_once", func(t *testing.T) {
		rec := &stepLog{}
		fences := 1
		var authCalls, afterFence int
		runReauthorization(rec, reauthorizationSteps{
			fenceInsert: func() bool {
				rec.check("D-077 restore fence input", "0 -> 1 fence; no device/binding/observation edits", "before=1; after=1; inserted=false", "lab Postgres INSERT device_authorization_fences per migration 00188 / restore-drill §6.3", false)
				return false
			},
			afterFenceInsert: func() { afterFence++ },
			fenceCount:       func() int { return fences },
			authorizeDevice: func() grantHTTPResponse {
				authCalls++
				fences = 0
				return grantHTTPResponse{status: 202, body: []byte(`{"kind":"member.device_authorize"}`)}
			},
			waitAuthorized: func() bool { return true },
		})
		if !rec.failed || authCalls != 1 || afterFence != 0 || fences != 0 || len(rec.steps) != 2 || rec.steps[1].Name != "D-077 early-return fence restoration" {
			t.Fatalf("unexpected reauthorization early-return state: authCalls=%d fences=%d steps=%+v", authCalls, fences, rec.steps)
		}

		normalRec := &stepLog{}
		runReauthorization(normalRec, reauthorizationSteps{
			fenceInsert:      func() bool { return true },
			afterFenceInsert: func() { afterFence++ },
			fenceCount:       func() int { return 1 },
			authorizeDevice:  func() grantHTTPResponse { authCalls++; return grantHTTPResponse{status: 202} },
		})
		if afterFence != 1 || authCalls != 1 || len(normalRec.steps) != 0 {
			t.Fatalf("normal reauthorization ran early restore: afterFence=%d authCalls=%d steps=%+v", afterFence, authCalls, normalRec.steps)
		}
	})

	t.Run("populatedHoldCleanup_and_entitlementsPrecheck_and_stopSiblingEarly", func(t *testing.T) {
		rec := &stepLog{}
		var memberRestored, deviceAuthorized, published int
		runPopulatedHoldCleanup(rec, populatedHoldCleanupSteps{
			restoreHoldMember:  func() { memberRestored++ },
			deviceNeedsRestore: func() bool { return true },
			authorizeHoldDevice: func() grantHTTPResponse {
				deviceAuthorized++
				return grantHTTPResponse{status: 202}
			},
			publishGroupA: func() { published++ },
		})
		if memberRestored != 1 || deviceAuthorized != 1 || published != 1 || len(rec.steps) != 1 || rec.steps[0].Name != "populated hold early-return restore original B binding" {
			t.Fatalf("unexpected populatedHoldCleanup early-return result: %d/%d/%d steps=%+v", memberRestored, deviceAuthorized, published, rec.steps)
		}

		normalRec := &stepLog{}
		runPopulatedHoldCleanup(normalRec, populatedHoldCleanupSteps{
			restoreHoldMember:   func() { memberRestored++ },
			deviceNeedsRestore:  func() bool { return false },
			authorizeHoldDevice: func() grantHTTPResponse { deviceAuthorized++; return grantHTTPResponse{status: 202} },
			publishGroupA:       func() { published++ },
		})
		if memberRestored != 2 || deviceAuthorized != 1 || published != 2 || len(normalRec.steps) != 0 {
			t.Fatalf("normal populatedHoldCleanup should not re-authorize already-authorized device: %d/%d/%d steps=%+v", memberRestored, deviceAuthorized, published, normalRec.steps)
		}

		var disabled int
		var blocked string
		if runEntitlementsPrecheck(false, true, func() { disabled++ }, func(msg string) { blocked = msg }) {
			t.Fatal("expected false when paid session is nil")
		}
		if disabled != 1 || blocked != "AT03 paid session unavailable" {
			t.Fatalf("disabled=%d blocked=%q", disabled, blocked)
		}
		if !runEntitlementsPrecheck(true, true, func() { disabled++ }, func(msg string) { blocked = msg }) || disabled != 1 {
			t.Fatalf("normal entitlements precheck should not disable second exit early: disabled=%d", disabled)
		}

		finished := false
		var samplesStopped, nativeStopped int
		stopSiblingEarly(&finished, func() { samplesStopped++ }, func() { nativeStopped++ })
		stopSiblingEarly(&finished, func() { samplesStopped++ }, func() { nativeStopped++ })
		if !finished || samplesStopped != 1 || nativeStopped != 1 {
			t.Fatalf("stopSiblingEarly = finished:%t samples:%d native:%d", finished, samplesStopped, nativeStopped)
		}
	})
}
