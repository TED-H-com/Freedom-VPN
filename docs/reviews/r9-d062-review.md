# Adversarial Review `r9-d062-review`: Update Result Evidence, Login Rate Limiter, and Canonical Identity (`D-062`, `D-060`, `D-052`)

- **Branch:** `r9-d062-review` (commit `fa34463`)
- **Date:** 2026-10-30 (`2026-10-30 01:25:17 -0400`)
- **Reviewing model:** `grok/grok-4.7` (recorded in commit `fa34463` `Agent:` trailer; not stated in `.agent/REVIEW.md` body)
- **Decision / scope reviewed:** `D-062` update result evidence and observation authority (`§13.6`, `§13.10`), `D-060` login rate limiter (`§9.1`, `§14.6`), and `D-052` `F-04` canonical issuer identity (`§9.1`, `§15.2`; `internal/fleetupdate`, `internal/httpapi`, `internal/auth`, `internal/members/issuer`, `node/internal/updatejournal`)
- **Adopted proof files on `main`:**
  - [`internal/fleetupdate/review_r9_d062_integration_test.go`](../../internal/fleetupdate/review_r9_d062_integration_test.go) (adopted in `#140`, alongside [`internal/fleetupdate/r9_observation_integration_test.go`](../../internal/fleetupdate/r9_observation_integration_test.go))
  - [`internal/httpapi/review_r9_d060_test.go`](../../internal/httpapi/review_r9_d060_test.go) (adopted in `#141`)
  - `TestReviewR9_CanonicalIssuerTable` from `review_r9_d052_test.go` ported into [`internal/members/issuer/issuer_test.go`](../../internal/members/issuer/issuer_test.go) (adopted in `#139`; passing held tests on `r9-d062-review` in `review_r9_d052_integration_test.go` and `review_r9_d062_test.go` are covered by [`internal/auth/review_d052_integration_test.go`](../../internal/auth/review_d052_integration_test.go) and [`node/internal/updatejournal/candidate_health_test.go`](../../node/internal/updatejournal/candidate_health_test.go))

## Findings

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | A generation-bearing observation in `internal/fleetupdate/serving.go` (`ObserveTx`, `ObserveReportTx`) and `internal/fleetupdate/progress.go` overwrites `nodes.current_release_id`, `last_good_release_id`, and `recovery_required` after an accepted result, and a report with a null `update.job_id` advances the numeric generation high-water mark. | closed | #140 |
| Medium | Conflicting replays of `SubmitResult` in `internal/fleetupdate/results.go` insert `update_result_conflicts` with `ON CONFLICT DO NOTHING` but append a new `update_result.conflict` audit event on every replay of an already-recorded conflicting hash. | closed | #140 |
| Medium | A `failed` result at stage `check` with `error_code` null in `internal/fleetupdate/results.go` (`preHealthFailure`) is accepted because `beforeHealth` only covers `claim` through `start`, promoting the failed candidate to `CurrentReleaseID`. | closed | #140 |
| Medium | The login and pre-authentication address limiter in `internal/httpapi/middleware.go` keys on the full `net.SplitHostPort(r.RemoteAddr)` host, giving each `/128` inside a single IPv6 `/64` allocation its own burst bucket up to `MaxRatePrincipals`. | closed | #141 |
| Low | The domain observation path `ObserveReportTx` in `internal/fleetupdate/serving.go` and `internal/fleetupdate/progress.go` does not validate `update.state` or `update.stage` against the contract enums when called directly without the HTTP body validator. | closed | #140 |
| Low | A trailing percent-encoded slash (`%2F`) in `internal/auth/store.go` (`canonicalIssuer`) and `internal/members/issuer/issuer.go` (`Canonical`) collapses onto the trimmed decoded path (`https://idp.example/a%2F` -> `https://idp.example/a`). | closed | #137, #139 |

## What Held

- **Result identity and replay (`D-062d`, `TestReviewR9_D062_ResultIdentityReplayAndReceiptBounds`):**
  - Replaying the exact bytes after acceptance returns the stored ACK without appending a second accept audit.
  - A job owned by another node or submitted by a superseded incarnation (after revoke and re-enroll) returns `IncarnationMismatch` and writes no receipt (`authenticate` checks the current incarnation before writing a receipt).
  - An unissued attempt number returns `StaleGeneration` and records one immutable rejection receipt inside the bounded ACK vocabulary; `GetBatch` JSON never leaks `boot_id` or `result_id`.
- **Evidence completeness (`D-062a`–`c`, `TestReviewR9_D062_EvidenceCompleteness`):**
  - `succeeded` with `health: null` is rejected; `rolled_back` whose `current_release_id` equals the candidate is rejected; `rolled_back` with `candidate_health` omitted, explicit `null`, or populated sets `needs_attention = true` and keeps the restored release as `CurrentReleaseID`; extra properties on `UpdateHealth` are rejected by schema validation.
- **Observation generation (`D-062e`, `TestReviewR9_D062_ObservationGenerationAndServingRelease`):**
  - An observation with `update.generation: null` after a stored generation leaves the numeric high-water mark and accepted serving release in place; generation `N+5` with another node's `job_id` returns `StaleGeneration` and rolls back; an observation from a revoked previous incarnation returns `IDENTITY_REVOKED` and never confirms the current job.
- **Node journal atomicity (`TestReviewR9_PendingV2EnrichmentIsAtomicAcrossCrash`):**
  - `PopulateCandidateHealth` (`node/internal/updatejournal/candidate_health.go`) updates `payload` and `payload_hash` in one SQLite `UPDATE`; a mid-update abort leaves the pre-image intact with matching `HashHex` and `result_id`.
- **Login rate limiter (`D-060`, `TestReviewR9_LoginLimiterAddressIdentityBoundsAndRoutes`):**
  - `X-Forwarded-For`, `X-Real-IP`, `Forwarded`, `Authorization`, and query identity strings are ignored when keying the pre-auth bucket; `/auth/login`, `/auth/login?role=admin`, and `/auth/callback` share the login bucket while `/auth/session` and `/auth/logout` use the `session` class and remain available when the login bucket is exhausted; `429 RATE_LIMITED` sets `Retry-After >= 1`, increments `fvpn_control_rate_limited_total`, and writes no audit row; `Browser.allowSubject` limits post-verification logins per `(issuer, client_id, subject)`.
- **Canonical identity (`D-052 F-04`, `TestReviewR9_CanonicalLockMatchesLookup`, `TestReviewR9_AdminOldSessionSurvivesClientScopeRotation`, `TestReviewR9_DistinctIssuersStayDistinctMembers`):**
  - Eight concurrent mixed-case logins across `portal-a`..`portal-h` serialize on the canonical `(issuer, subject)` advisory lock and produce one member and one admin; admin sessions survive `client_scope` rotation; distinct schemes or hosts stay distinct members.

## Detailed Review Notes (Archived from `.agent/REVIEW.md`)

### [High] A generation-bearing observation overwrites the serving release and recovery flag
- **Where:** `internal/fleetupdate/serving.go:91, 107, 150, 173`, `internal/fleetupdate/progress.go:236`.
- **Scenario:** After an accepted result, a later observation on the same incarnation and generation overwrote `nodes.current_release_id` and `last_good_release_id` with the heartbeat's releases and recomputed `recovery_required` from `revoked_release_ids` alone (clearing `recovery_required: true` set by a failed result). A report with `update.job_id: null` and `generation: N+5` moved the numeric high-water mark and serving release while leaving the job unconfirmed.
- **Proof:** `TestReviewR9_D062_ObservationGenerationAndServingRelease` (`internal/fleetupdate/review_r9_d062_integration_test.go`).
- **Fix (#140):** Heartbeat seeding of `current_release_id` and `last_good_release_id` ends at the first accepted result for the incarnation; after that, only accepted results change serving releases and `recovery_required` (a heartbeat can raise recovery, never clear it). A heartbeat whose releases disagree with the accepted values keeps the accepted values and withholds confirmation (`unconfirmed` stays `true` until an agreeing heartbeat for the exact `job_id` and `generation` arrives). A report without both `update.job_id` and `update.generation` records arrival only.

### [Medium] Conflicting replays append conflict rows and audits without a bound
- **Where:** `internal/fleetupdate/results.go:116-122`.
- **Scenario:** `results.go` inserted `update_result_conflicts(result_id, conflicting_hash) ON CONFLICT DO NOTHING` and then unconditionally appended `update_result.conflict`, so replaying the same conflicting payload hash appended a new audit row on every request.
- **Proof:** `TestReviewR9_D062_ResultIdentityReplayAndReceiptBounds/changed_bytes_conflict_without_overwrite_and_audits_grow` (`internal/fleetupdate/review_r9_d062_integration_test.go`).
- **Fix (#140):** Wrote the `update_result_conflicts` row and the `update_result.conflict` audit once per `(result_id, conflicting_hash)` in a single transaction; replaying a recorded conflicting hash returns `409 ACK_CONFLICT` with no new row or audit.

### [Medium] A `failed` result at stage `check` with a null error code becomes the current release
- **Where:** `internal/fleetupdate/results.go:170, 361`.
- **Scenario:** `preHealthFailure` required `error_code` only when `beforeHealth` (`claim` through `start`) was true, so `state=failed` at `stage=check` with `error_code=null` was accepted and set `CurrentReleaseID` to the failed candidate.
- **Proof:** `TestReviewR9_D062_EvidenceCompleteness/failed_at_check_without_error_code` (`internal/fleetupdate/review_r9_d062_integration_test.go`).
- **Fix (#140):** Required `error_code` on every `failed` result regardless of stage (returning `422 VALIDATION_FAILED` with an immutable rejection receipt when missing).

### [Medium] One IPv6 `/64` receives one login bucket per `/128`
- **Where:** `internal/httpapi/middleware.go:539`.
- **Scenario:** Keying the address limiter on the full host from `net.SplitHostPort(r.RemoteAddr)` gave each `/128` inside `2001:db8:0:1::/64` its own full `login_rate` burst bucket.
- **Proof:** `TestReviewR9_LoginLimiterAddressIdentityBoundsAndRoutes/ipv6_prefix` (`internal/httpapi/review_r9_d060_test.go`).
- **Fix (#141):** Keyed IPv6 connection addresses on their canonical `/64` prefix (keeping IPv4 and IPv4-mapped IPv6 per `/32` address) across the login, ingress, and anonymous limiters, and failed closed with `429 RATE_LIMITED` on unparseable `RemoteAddr` values without inserting them into the principal map.

### [Low] Domain observation enum validation and percent-encoded trailing slash in `issuer.Canonical`
- **Proofs:** `TestReviewR9_D062_ObservationGenerationAndServingRelease/stage_and_state_outside_the_vocabulary` (`internal/fleetupdate/review_r9_d062_integration_test.go`), `TestReviewR9_CanonicalIssuerTable` (`internal/members/issuer/issuer_test.go`).
- **Fix (#137, #139, #140):** Validated `update.state` and `update.stage` in `ObserveReportTx` (`#140`), unified login on `internal/members/issuer.Canonical` (`#137`), and updated `issuer.Canonical` to trim only literal trailing slashes from the escaped path while preserving percent-encoded slashes (`%2F`, `%2f`) (`#139`).
