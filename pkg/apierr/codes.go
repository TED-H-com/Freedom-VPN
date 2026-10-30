package apierr

import "net/http"

// Code is a stable, machine-readable error code. Codes never change meaning;
// new codes may be added.
type Code string

// Spec describes one error code.
type Spec struct {
	Code Code
	// HTTPStatus is zero for report-only codes, which must not be HTTP responses.
	HTTPStatus int
	// Retryable is the default value of the envelope "retryable" flag: true
	// when the same request (with the same Idempotency-Key) may succeed
	// later without the caller changing anything.
	Retryable bool
	// Section cites the Project Spec section that defines the code.
	Section string
	// NodeLocal is true for codes that are primarily reported inside agent
	// payloads (update results §13.10.3, observations) rather than returned
	// by the central API. Legacy codes retain their HTTP mappings. Report-only
	// codes have HTTPStatus zero. All appear in the OpenAPI ErrorCode enum.
	NodeLocal bool
	// Description is a short English description for documentation.
	Description string
}

// General API codes (§14.6) and transport codes.
const (
	MalformedRequest        Code = "MALFORMED_REQUEST"
	Unauthenticated         Code = "UNAUTHENTICATED"
	Forbidden               Code = "FORBIDDEN"
	CSRFTokenInvalid        Code = "CSRF_TOKEN_INVALID"
	ResourceNotFound        Code = "RESOURCE_NOT_FOUND"
	NotFound                Code = "NOT_FOUND"
	MethodNotAllowed        Code = "METHOD_NOT_ALLOWED"
	IdempotencyConflict     Code = "IDEMPOTENCY_CONFLICT"
	StateConflict           Code = "STATE_CONFLICT"
	ResourceVersionConflict Code = "RESOURCE_VERSION_CONFLICT"
	PayloadTooLarge         Code = "PAYLOAD_TOO_LARGE"
	UnsupportedMediaType    Code = "UNSUPPORTED_MEDIA_TYPE"
	ValidationFailed        Code = "VALIDATION_FAILED"
	NodeNotServiceable      Code = "NODE_NOT_SERVICEABLE"
	PreconditionRequired    Code = "PRECONDITION_REQUIRED"
	RateLimited             Code = "RATE_LIMITED"
	InternalError           Code = "INTERNAL_ERROR"
	NotImplemented          Code = "NOT_IMPLEMENTED"
	DependencyUnavailable   Code = "DEPENDENCY_UNAVAILABLE"
)

// Release and update codes (§13.11).
const (
	ReleaseNotApproved      Code = "RELEASE_NOT_APPROVED"
	UnsupportedPlatform     Code = "UNSUPPORTED_PLATFORM"
	ControlAPIIncompatible  Code = "CONTROL_API_INCOMPATIBLE"
	SchemaIncompatible      Code = "SCHEMA_INCOMPATIBLE"
	SignatureInvalid        Code = "SIGNATURE_INVALID"
	KeyUntrusted            Code = "KEY_UNTRUSTED"
	HashMismatch            Code = "HASH_MISMATCH"
	SizeMismatch            Code = "SIZE_MISMATCH"
	ArtifactInvalid         Code = "ARTIFACT_INVALID"
	DiskSpaceLow            Code = "DISK_SPACE_LOW"
	DownloadTimeout         Code = "DOWNLOAD_TIMEOUT"
	StoreUnavailable        Code = "STORE_UNAVAILABLE"
	UpdateBusy              Code = "UPDATE_BUSY"
	AuthorizationExpired    Code = "AUTHORIZATION_EXPIRED"
	StaleGeneration         Code = "STALE_GENERATION"
	IdentityRevoked         Code = "IDENTITY_REVOKED"
	IncarnationMismatch     Code = "INCARNATION_MISMATCH"
	CheckpointFailed        Code = "CHECKPOINT_FAILED"
	MigrationFailed         Code = "MIGRATION_FAILED"
	StartFailed             Code = "START_FAILED"
	HealthCheckFailed       Code = "HEALTH_CHECK_FAILED"
	MinimumVersionViolation Code = "MINIMUM_VERSION_VIOLATION"
	RollbackFailed          Code = "ROLLBACK_FAILED"
	AckConflict             Code = "ACK_CONFLICT"
)

// Domain codes used elsewhere in the spec.
const (
	UsageEventConflict       Code = "USAGE_EVENT_CONFLICT"
	LedgerEpochMismatch      Code = "LEDGER_EPOCH_MISMATCH"
	ReplayUnavailable        Code = "REPLAY_UNAVAILABLE"
	MeterPathInvalid         Code = "METER_PATH_INVALID"
	IsolationBaselineMissing Code = "ISOLATION_BASELINE_MISSING"
	MeterEpochChange         Code = "METER_EPOCH_CHANGE"
	IsolationPolicyInvalid   Code = "ISOLATION_POLICY_INVALID"
	GenerationMismatch       Code = "GENERATION_MISMATCH"
	HookPriorityMismatch     Code = "HOOK_PRIORITY_MISMATCH"
	CounterOrderInvalid      Code = "COUNTER_ORDER_INVALID"
	FlowtableBypass          Code = "FLOWTABLE_BYPASS"
	DeviceClassMissing       Code = "DEVICE_CLASS_MISSING"
	QosClassMissing          Code = "QOS_CLASS_MISSING"
	MarkSemanticsViolated    Code = "MARK_SEMANTICS_VIOLATED"
	ProbeUnbounded           Code = "PROBE_UNBOUNDED"
	DeviceLimitReached       Code = "DEVICE_LIMIT_REACHED"
	EnrollmentTokenInvalid   Code = "ENROLLMENT_TOKEN_INVALID"
	EnrollmentTokenExpired   Code = "ENROLLMENT_TOKEN_EXPIRED"
	EnrollmentTokenRevoked   Code = "ENROLLMENT_TOKEN_REVOKED"
	EnrollmentKeyMismatch    Code = "ENROLLMENT_KEY_MISMATCH"
	LeaseExpired             Code = "LEASE_EXPIRED"
	WebhookSignatureBad      Code = "WEBHOOK_SIGNATURE_INVALID"
	BootstrapConsumed        Code = "BOOTSTRAP_CONSUMED"
	BootstrapNotReady        Code = "BOOTSTRAP_NOT_READY"
	BootstrapExpired         Code = "BOOTSTRAP_EXPIRED"
	NodeIncarnationActive    Code = "NODE_INCARNATION_ACTIVE"
	NodeRetired              Code = "NODE_RETIRED"
)

// specs is the authoritative table. The OpenAPI ErrorCode enum must list
// exactly these codes (checked by the contracts tests).
var specs = []Spec{
	// §14.6
	{MalformedRequest, http.StatusBadRequest, false, "§14.6", false, "Malformed JSON, unknown member or missing required header (for example Idempotency-Key)."},
	{Unauthenticated, http.StatusUnauthorized, false, "§14.6", false, "No valid session, client certificate or credential."},
	{Forbidden, http.StatusForbidden, false, "§14.6", false, "Authenticated principal is not allowed to perform this action; also mTLS identity mismatch (§10.6, §14.4)."},
	{CSRFTokenInvalid, http.StatusForbidden, false, "§16.1", false, "Missing or mismatched CSRF token on a state-changing browser request."},
	{ResourceNotFound, http.StatusNotFound, false, "§14.6", false, "Resource does not exist or does not belong to the caller; never reveals other members' resources (§14.3)."},
	{NotFound, http.StatusNotFound, false, "§12.3, D-056", false, "Resolver document does not exist or was not issued to this node; never reveals another node's documents."},
	{MethodNotAllowed, http.StatusMethodNotAllowed, false, "§14.1", false, "HTTP method not supported on this path."},
	{IdempotencyConflict, http.StatusConflict, false, "§14.5", false, "Idempotency-Key reused with a different request payload."},
	{StateConflict, http.StatusConflict, false, "§14.6", false, "Request conflicts with current resource state, or expected_generation does not match (§14.5)."},
	{ResourceVersionConflict, http.StatusPreconditionFailed, false, "§14.6", false, "If-Match does not match the current resource version; details.current_resource_version holds the current value."},
	{PayloadTooLarge, http.StatusRequestEntityTooLarge, false, "§14.6", false, "Request body or batch exceeds the configured limit."},
	{UnsupportedMediaType, http.StatusUnsupportedMediaType, false, "§14.1", false, "Request body is not application/json."},
	{ValidationFailed, http.StatusUnprocessableEntity, false, "§14.6", false, "Request is well-formed but fails schema or semantic validation; details.fields lists the offending fields."},
	{NodeNotServiceable, http.StatusUnprocessableEntity, false, "§14.6", false, "Node does not satisfy the serving baseline (§5.1) for the requested action, for example publishing a node that is not ready."},
	{PreconditionRequired, http.StatusPreconditionRequired, false, "§14.6", false, "If-Match header is required to modify this resource."},
	{RateLimited, http.StatusTooManyRequests, true, "§14.6", false, "Rate limit exceeded; honor Retry-After and back off with jitter."},
	{InternalError, http.StatusInternalServerError, false, "§14.6", false, "Unexpected server error. No internal detail is returned; quote request_id."},
	{NotImplemented, http.StatusNotImplemented, false, "§14.6 (transitional)", false, "Command not implemented; documented endpoints must not ship with this response in a release."},
	{DependencyUnavailable, http.StatusServiceUnavailable, true, "§14.6", false, "Temporarily unable to persist or reach a dependency; retry with the same Idempotency-Key. Events were not acknowledged."},

	// §13.11
	{ReleaseNotApproved, http.StatusUnprocessableEntity, false, "§13.11", false, "Release is not approved or is revoked and cannot be a new update target."},
	{UnsupportedPlatform, http.StatusUnprocessableEntity, false, "§13.11", true, "Release lacks an artifact for the node OS/architecture or a required capability."},
	{ControlAPIIncompatible, http.StatusUnprocessableEntity, false, "§13.11", true, "Target release does not support the current control API major version."},
	{SchemaIncompatible, http.StatusUnprocessableEntity, false, "§13.11", true, "Policy, usage or state schema, or rollback readability, is incompatible."},
	{SignatureInvalid, http.StatusUnprocessableEntity, false, "§13.11", false, "A signature does not verify, uses an unknown algorithm or is malformed."},
	{KeyUntrusted, http.StatusUnprocessableEntity, false, "§13.11", false, "No trusted key signed the document, or a signing key is revoked, expired or lacks the purpose."},
	{HashMismatch, http.StatusUnprocessableEntity, false, "§13.11", true, "Downloaded content SHA-256 differs from the manifest; bounded re-download only."},
	{SizeMismatch, http.StatusUnprocessableEntity, false, "§13.11", true, "Downloaded content size differs from the manifest; bounded re-download only."},
	{ArtifactInvalid, 0, false, "§13.3, §13.4", true, "Hostile or malformed release archive, symlink, or entry/byte cap exceeded; node report only."},
	{DiskSpaceLow, http.StatusInsufficientStorage, false, "§13.11", true, "Not enough disk space to start a destructive apply."},
	{DownloadTimeout, http.StatusGatewayTimeout, true, "§13.11", true, "Artifact download timed out; bounded backoff retry."},
	{StoreUnavailable, http.StatusServiceUnavailable, true, "§13.11", true, "Release store unavailable; bounded backoff retry."},
	{UpdateBusy, http.StatusConflict, true, "§13.11", true, "Local update lock is held by another process."},
	{AuthorizationExpired, http.StatusConflict, false, "§13.11", true, "Update authorization expired before switching; fetch a fresh authorized job."},
	{StaleGeneration, http.StatusConflict, false, "§13.11", false, "Job generation is older than the current desired generation; the job is superseded."},
	{IdentityRevoked, http.StatusForbidden, false, "§13.11", false, "Node identity (certificate or incarnation) is revoked; retrying cannot restore it."},
	{IncarnationMismatch, http.StatusForbidden, false, "§13.11", false, "Body node_id/incarnation does not match the authenticated certificate binding (§14.4)."},
	{CheckpointFailed, http.StatusInternalServerError, false, "§13.11", true, "Final usage sample, journal or outbox could not be persisted; the switch did not start."},
	{MigrationFailed, http.StatusInternalServerError, false, "§13.11", true, "Local state migration failed."},
	{StartFailed, http.StatusInternalServerError, false, "§13.11", true, "Candidate release failed to start."},
	{HealthCheckFailed, http.StatusInternalServerError, false, "§13.11", true, "Candidate release failed the health profile (§13.9)."},
	{MinimumVersionViolation, http.StatusUnprocessableEntity, false, "§13.11", false, "Version is below minimum_serving_version; it may not serve."},
	{RollbackFailed, http.StatusInternalServerError, false, "§13.11", true, "Rollback to last_good failed; recovery_required is true."},
	{AckConflict, http.StatusConflict, false, "§13.11", false, "Same result_id reported with a different payload_hash."},

	// Domain codes
	{UsageEventConflict, http.StatusConflict, false, "§10.6", false, "Same (node_id, node_incarnation_id, event_seq) with a different payload_sha256."},
	{LedgerEpochMismatch, http.StatusConflict, false, "§10.8, §17.3", false, "Batch references a ledger_epoch other than the current one; fetch GET /agent/usage-checkpoint first."},
	{ReplayUnavailable, http.StatusGone, false, "§10.8, §15.4", false, "Requested sequence range is outside the verifiable replay window; a gap is recorded."},
	{MeterPathInvalid, http.StatusUnprocessableEntity, false, "§10.3", true, "Metering hook, interface or binding generation is not valid; member forwarding stops."},
	{IsolationBaselineMissing, 0, false, "§14.4, §16.3", true, "Isolation baseline is missing; member forwarding is off; node report only."},
	{MeterEpochChange, 0, false, "§14.4, §17.3", true, "Meter epoch changed; event sequence continues within the incarnation; node report only."},
	{IsolationPolicyInvalid, 0, false, "§13.11", true, "The allow/deny policy itself does not match; node report only."},
	{GenerationMismatch, 0, false, "§13.11", true, "The kernel ruleset is not the requested generation; node report only."},
	{HookPriorityMismatch, 0, false, "§8.2", true, "A base chain hook or priority drifted; node report only."},
	{CounterOrderInvalid, 0, false, "§10.2, §16.3", true, "An accept can skip the per-device counters, including an established/related accept placed before the lease check; node report only."},
	{FlowtableBypass, 0, false, "§10.3", true, "A flowtable or offload could bypass the counters; node report only."},
	{DeviceClassMissing, 0, false, "§13.11", true, "A device binding has no counter or class; node report only."},
	{QosClassMissing, 0, false, "§11.4, §11.5", true, "The HTB parent or a member class is missing or wrong; node report only."},
	{MarkSemanticsViolated, 0, false, "§11.4", true, "A rule writes skb mark or conntrack mark. NetBird owns 0x1BD00-0x1BDFF; node report only."},
	{ProbeUnbounded, 0, false, "§13.9", true, "A synthetic-probe set element has no kernel timeout, so the exception would outlive the health check; node report only."},
	{DeviceLimitReached, http.StatusConflict, false, "§9.4, MEM02", false, "Authorizing another device would exceed the plan device limit."},
	{EnrollmentTokenInvalid, http.StatusUnauthorized, false, "§6.1, ENR02", false, "Enrollment token is unknown or malformed."},
	{EnrollmentTokenExpired, http.StatusUnauthorized, false, "§6.4, ENR02", false, "Enrollment token has expired."},
	{EnrollmentTokenRevoked, http.StatusUnauthorized, false, "§6.4, ENR02", false, "Enrollment token was revoked."},
	{EnrollmentKeyMismatch, http.StatusConflict, false, "§6.2, ENR02", false, "Token is already bound to a different public key or enrollment_request_id."},
	{LeaseExpired, http.StatusConflict, false, "§16.3", true, "Serving lease expired; member forwarding is off until a new lease is applied."},
	{WebhookSignatureBad, http.StatusUnauthorized, false, "§9.3", false, "Payment provider webhook signature is missing or invalid; entitlement unchanged."},
	{BootstrapConsumed, http.StatusGone, false, "§6.4, §14.7 (D-015)", false, "Bootstrap delivery was consumed; plaintext is never replayed."},
	{BootstrapNotReady, http.StatusConflict, true, "§6.4, §14.7 (D-015)", false, "Worker has not yet prepared the bootstrap delivery; retry the same request."},
	{BootstrapExpired, http.StatusGone, false, "§6.4, §14.7 (D-015)", false, "Unconsumed bootstrap delivery deadline has passed."},
	{NodeIncarnationActive, http.StatusConflict, false, "§6.4, AT07", false, "Reinstallation requires revocation of the existing Node incarnation first."},
	{NodeRetired, http.StatusConflict, false, "§14.2, AT07", false, "A retired Node cannot be enrolled again."},
}

var byCode = func() map[Code]Spec {
	m := make(map[Code]Spec, len(specs))
	for _, s := range specs {
		if _, dup := m[s.Code]; dup {
			panic("apierr: duplicate code " + string(s.Code))
		}
		m[s.Code] = s
	}
	return m
}()

// All returns every defined code specification in table order.
func All() []Spec {
	out := make([]Spec, len(specs))
	copy(out, specs)
	return out
}

// Lookup returns the specification of code.
func Lookup(code Code) (Spec, bool) {
	s, ok := byCode[code]
	return s, ok
}

// Valid reports whether code is defined.
func (c Code) Valid() bool {
	_, ok := byCode[c]
	return ok
}

// HTTPStatus returns the HTTP status, or 500 for unknown/report-only codes.
// Writer renders report-only codes as INTERNAL_ERROR, never as HTTP errors.
func (c Code) HTTPStatus() int {
	if s, ok := byCode[c]; ok && s.HTTPStatus != 0 {
		return s.HTTPStatus
	}
	return http.StatusInternalServerError
}

// DefaultRetryable returns the default retryable flag of code (false for
// unknown codes).
func (c Code) DefaultRetryable() bool {
	return byCode[c].Retryable
}
