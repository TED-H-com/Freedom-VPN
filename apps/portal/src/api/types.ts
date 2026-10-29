// TEMPORARY: replace with contracts/ts when available

/**
 * API contract types aligned with `docs/spec/PROJECT_SPEC.zh-TW.md` (§5, §9, §10, §11, §12, §13, §14, §15, §18).
 * Large integers (bytes, seq, generation, revision, resource_version) are JSON decimal strings (§10.4, §14.1).
 */

export type UserRole = 'admin' | 'member';

export interface SessionInfo {
  authenticated: boolean;
  principal_id: string;
  email: string;
  display_name: string;
  role: UserRole;
  mfa_verified: boolean;
  member_id?: string;
}

export type ApiErrorCode =
  | 'MALFORMED_REQUEST'
  | 'UNAUTHENTICATED'
  | 'FORBIDDEN'
  | 'RESOURCE_NOT_FOUND'
  | 'IDEMPOTENCY_CONFLICT'
  | 'STATE_CONFLICT'
  | 'RESOURCE_VERSION_CONFLICT'
  | 'PRECONDITION_REQUIRED'
  | 'VALIDATION_FAILED'
  | 'NODE_NOT_SERVICEABLE'
  | 'RATE_LIMITED'
  | 'DEPENDENCY_UNAVAILABLE'
  | 'RELEASE_NOT_APPROVED'
  | 'UNSUPPORTED_PLATFORM'
  | 'CONTROL_API_INCOMPATIBLE'
  | 'SCHEMA_INCOMPATIBLE'
  | 'SIGNATURE_INVALID'
  | 'KEY_UNTRUSTED'
  | 'HASH_MISMATCH'
  | 'SIZE_MISMATCH'
  | 'DISK_SPACE_LOW'
  | 'DOWNLOAD_TIMEOUT'
  | 'STORE_UNAVAILABLE'
  | 'UPDATE_BUSY'
  | 'AUTHORIZATION_EXPIRED'
  | 'STALE_GENERATION'
  | 'IDENTITY_REVOKED'
  | 'INCARNATION_MISMATCH'
  | 'CHECKPOINT_FAILED'
  | 'MIGRATION_FAILED'
  | 'START_FAILED'
  | 'HEALTH_CHECK_FAILED'
  | 'MINIMUM_VERSION_VIOLATION'
  | 'ROLLBACK_FAILED'
  | 'ACK_CONFLICT'
  | 'USAGE_EVENT_CONFLICT'
  | 'METER_PATH_INVALID'
  | (string & {});

export interface ApiErrorPayload {
  code: ApiErrorCode;
  message: string;
  request_id: string;
  retryable: boolean;
  details?: Record<string, unknown>;
}

export interface ApiErrorEnvelope {
  error: ApiErrorPayload;
}

export type OperationStatus =
  | 'pending'
  | 'running'
  | 'succeeded'
  | 'partially_succeeded'
  | 'failed'
  | 'superseded';

export type OperationStepStatus =
  | 'pending'
  | 'running'
  | 'succeeded'
  | 'failed'
  | 'skipped'
  | 'not_required';

export interface OperationStep {
  name: string;
  status: OperationStepStatus;
  detail?: string;
  error_code?: string;
}

export interface Operation {
  operation_id: string;
  kind: string;
  status: OperationStatus;
  resource_version?: string;
  desired_generation?: string;
  observed_generation?: string;
  steps: OperationStep[];
  summary_code?: string;
  error_code?: string;
  error_message?: string;
  created_at: string;
  updated_at?: string;
  completed_at?: string;
}

// §5 Node states
export type NodeLifecycleState = 'enrolling' | 'registered' | 'revoked' | 'retired';
export type NodeProvisioningState = 'pending' | 'applying' | 'ready' | 'failed';
export type PublicationIntent = 'unpublished' | 'published' | 'disabled';
export type PublicationObserved =
  | 'absent'
  | 'applying'
  | 'active'
  | 'withdrawn'
  | 'error';
export type NodeControlStatus = 'online' | 'stale' | 'offline';
export type NodeDataHealth = 'unknown' | 'healthy' | 'degraded' | 'failed';
export type NodeSafetyStatus = 'unknown' | 'passed' | 'failed' | 'expired';

export type NodeUpdateState =
  | 'queued'
  | 'waiting_online'
  | 'downloading'
  | 'verifying'
  | 'applying'
  | 'checking'
  | 'succeeded'
  | 'already_current'
  | 'failed'
  | 'rolled_back'
  | 'superseded'
  | 'cancelled';

export interface NodeErrorRecord {
  error_id: string;
  error_code: ApiErrorCode;
  stage: string;
  message: string;
  operation_id: string;
  request_id: string;
  occurred_at: string;
}

export interface NodeUpdateHistoryItem {
  job_id: string;
  batch_id: string;
  attempt: number;
  generation: string;
  state: NodeUpdateState;
  requested_release_id: string;
  current_release_id: string;
  previous_release_id: string;
  error_code: string | null;
  started_at: string;
  finished_at: string | null;
}

export interface NodeItem {
  node_id: string;
  node_incarnation_id: string;
  name: string;
  region: string;
  arch: 'amd64' | 'arm64';
  os_version: string;
  member_groups: string[];
  observed_member_groups: string[];
  lifecycle_state: NodeLifecycleState;
  provisioning_state: NodeProvisioningState;
  publication_intent: PublicationIntent;
  publication_observed: PublicationObserved;
  control_status: NodeControlStatus;
  data_health: NodeDataHealth;
  safety_status: NodeSafetyStatus;
  serving_allowed: boolean;
  summary_code:
    | 'online_serving'
    | 'online_safety_failed_not_serving'
    | 'online_disabled_not_serving'
    | 'ready_unpublished_not_serving'
    | 'offline_route_withdrawn_unconfirmed'
    | 'rolled_back_serving'
    | 'revoked_not_serving';
  config_generation: {
    desired: string;
    applied: string;
    last_ack_at: string | null;
  };
  release: {
    current_release_id: string;
    current_version: string;
    desired_release_id: string;
    desired_version: string;
    last_good_release_id: string;
    minimum_serving_version: string;
  };
  update_state: NodeUpdateState;
  capacity: {
    desired_up_bps: string;
    desired_down_bps: string;
    applied_up_bps: string;
    applied_down_bps: string;
  };
  observed_exit_ipv4: string;
  notes: string;
  resource_version: string;
  etag: string;
  last_observed_at: string;
  recent_errors: NodeErrorRecord[];
  update_history: NodeUpdateHistoryItem[];
}

export interface AdminOverviewSummary {
  available_exits_count: number;
  total_nodes_count: number;
  online_nodes_count: number;
  stale_nodes_count: number;
  offline_nodes_count: number;
  safety_failed_nodes_count: number;
  pending_or_failed_enrollments_count: number;
  update_attention_nodes_count: number;
  active_members_count: number;
  throttled_members_count: number;
  suspended_or_expired_members_count: number;
  as_of: string;
}

// §6 & §18.2 Enrollment
export type EnrollmentState = 'unused' | 'used' | 'expired' | 'revoked';

export interface CreateEnrollmentRequest {
  name?: string;
  region?: string;
  member_groups: string[];
  expires_in_hours: number;
  notes?: string;
}

export interface EnrollmentCreatedResponse {
  enrollment_id: string;
  node_id: string;
  state: EnrollmentState;
  enrollment_token: string;
  secret_available: true;
  name: string;
  region: string;
  member_groups: string[];
  notes: string;
  expires_at: string;
  created_at: string;
}

export interface EnrollmentDetail {
  enrollment_id: string;
  node_id: string;
  state: EnrollmentState;
  secret_available: false;
  name: string;
  region: string;
  member_groups: string[];
  notes: string;
  expires_at: string;
  created_at: string;
  consumed_at: string | null;
  revoked_at: string | null;
}

export interface UpdateNodePatchRequest {
  name?: string;
  region?: string;
  member_groups?: string[];
  desired_up_bps?: string;
  desired_down_bps?: string;
  notes?: string;
  simulate_conflict?: boolean;
}

export interface UpdateNodePublicationRequest {
  publication_intent: PublicationIntent;
  reason?: string;
}

// §13 Fleet updates & releases
export interface ReleaseItem {
  release_id: string;
  product_version: string;
  release_sequence: string;
  created_at: string;
  approved: boolean;
  requires_service_restart: boolean;
  minimum_updater_version: string;
  architectures: ('amd64' | 'arm64')[];
  manifest_sha256: string;
  components: {
    node_agent: string;
    node_updater: string;
    netbird: string;
    enforcement: string;
  };
}

export type UpdateTargetBucket =
  | 'queued'
  | 'waiting_online'
  | 'downloading'
  | 'verifying'
  | 'applying'
  | 'checking'
  | 'succeeded'
  | 'already_current'
  | 'failed'
  | 'rolled_back'
  | 'superseded'
  | 'cancelled';

export type UpdateBatchState =
  | 'accepted'
  | 'running'
  | 'waiting'
  | 'completed'
  | 'completed_with_exceptions';

export interface UpdateBatchCounts {
  queued: number;
  waiting_online: number;
  downloading: number;
  verifying: number;
  applying: number;
  checking: number;
  succeeded: number;
  already_current: number;
  failed: number;
  rolled_back: number;
  superseded: number;
  cancelled: number;
}

export interface UpdateBatchTargetItem {
  target_id: string;
  node_id: string;
  node_name: string;
  region: string;
  arch: 'amd64' | 'arm64';
  publication_intent: PublicationIntent;
  attempt: number;
  generation: string;
  bucket: UpdateTargetBucket;
  current_release_id: string;
  current_version: string;
  requested_release_id: string;
  requested_version: string;
  last_good_version: string;
  error_code: string | null;
  last_reported_at: string;
}

export interface UpdateBatchDetail {
  batch_id: string;
  operation_id: string;
  release_id: string;
  product_version: string;
  requires_service_restart: boolean;
  state: UpdateBatchState;
  target_count: number;
  online_at_snapshot: number;
  offline_at_snapshot: number;
  fleet_default_release_id: string;
  counts: UpdateBatchCounts;
  targets: UpdateBatchTargetItem[];
  created_at: string;
  updated_at: string;
}

export interface FleetReleasesResponse {
  fleet_default_release_id: string;
  minimum_serving_version: string;
  fleet_generation: string;
  releases: ReleaseItem[];
  active_batch: UpdateBatchDetail;
}

export interface CreateUpdateBatchRequest {
  release_id: string;
  scope: 'all_nodes';
  expected_generation: string;
  set_fleet_default: true;
  reason: string;
}

// §9.2 Members & Entitlements
export type EntitlementState =
  | 'pending_payment'
  | 'active'
  | 'cancel_at_period_end'
  | 'past_due'
  | 'grace'
  | 'expired'
  | 'suspended'
  | 'closed';

export type UsageQualityFlag =
  | 'complete'
  | 'gap'
  | 'boundary_estimated'
  | 'reconciliation_in_progress';

export interface MemberDeviceItem {
  device_id: string;
  member_id: string;
  name: string;
  platform: 'windows' | 'macos' | 'ios' | 'android' | 'linux';
  netbird_peer_id: string;
  overlay_ipv4: string;
  status: 'authorized' | 'pending_quota' | 'revoked';
  authorized_at: string;
  last_seen_at: string;
  revoked_at: string | null;
  resource_version: string;
}

export interface EntitlementGrantItem {
  grant_id: string;
  member_id: string;
  source: string;
  valid_from: string;
  valid_until: string;
  reason: string;
  created_by: string;
  revoked_at: string | null;
}

export interface MemberItem {
  member_id: string;
  email: string;
  display_name: string;
  entitlement_state: EntitlementState;
  entitlement_reason: string;
  admin_suspended: boolean;
  admin_suspension_reason: string | null;
  plan_code: string;
  plan_name: string;
  device_limit: number;
  authorized_device_count: number;
  effective_service_end: string;
  paid_through: string;
  grace_until: string | null;
  cancel_at_period_end: boolean;
  payment_provider: string;
  provider_subscription_status:
    | 'active'
    | 'incomplete'
    | 'past_due'
    | 'canceled'
    | 'refunded_full'
    | 'refunded_partial';
  last_reconciled_at: string;
  accounting_period: string;
  next_fup_reset_at: string;
  monthly_upload_bytes: string;
  monthly_download_bytes: string;
  monthly_total_bytes: string;
  usage_quality: UsageQualityFlag;
  usage_gap_note: string | null;
  usage_aggregate_as_of: string;
  fup_tier_index: number;
  fup_tier_name: string;
  fup_mode: 'shadow' | 'enforce';
  effective_down_bps: string;
  effective_up_bps: string;
  resource_version: string;
  etag: string;
  devices: MemberDeviceItem[];
  grants: EntitlementGrantItem[];
}

export interface CreateMemberGrantRequest {
  valid_until: string;
  reason: string;
}

export interface PatchMemberRequest {
  admin_suspended?: boolean;
  admin_suspension_reason?: string;
}

// §11 Fair Use Policy & Plans
export interface FairUseTier {
  tier_index: number;
  name_zh: string;
  name_en: string;
  threshold_bytes: string;
  down_bps: string;
  up_bps: string;
  priority: number;
}

export interface FairUsePolicyDraft {
  mode: 'shadow' | 'enforce';
  accounting_basis: 'utc_calendar_month_forward_admission_v1';
  minimum_rate_bps: string;
  effective_at: string;
  reason: string;
  tiers: FairUseTier[];
}

export interface FairUsePolicyState {
  definition_id: string;
  active_revision_id: string;
  active_revision_number: string;
  resource_version: string;
  etag: string;
  current: FairUsePolicyDraft;
  node_applied_summary: {
    total_nodes: number;
    applied_current_generation: number;
    pending_nodes: number;
  };
}

// §12 DNS Policy
export interface DnsRule {
  rule_id: string;
  profile_id: string;
  domain: string;
  include_subdomains: boolean;
  action: 'block' | 'allow';
  reason: string;
  revision: string;
}

export interface DnsThreatFeedSource {
  source_id: string;
  name: string;
  category: 'malware_phishing' | 'ads' | 'family';
  license_name: string;
  commercial_use_verified: boolean;
  last_updated_at: string;
  rule_count: number;
  status: 'healthy' | 'stale_using_last_good';
}

export interface DnsResolverObservation {
  resolver_id: string;
  name: string;
  ipv4: string;
  applied_revision: string;
  status: 'healthy' | 'degraded' | 'failed_retained_last_good';
  last_checked_at: string;
}

export interface DnsPolicyState {
  profile_id: string;
  active_revision: string;
  desired_revision: string;
  deployment_status: 'all_applied' | 'partially_applied' | 'failed_retained_last_good';
  resource_version: string;
  etag: string;
  rules: DnsRule[];
  feeds: DnsThreatFeedSource[];
  resolvers: DnsResolverObservation[];
}

// §14.2 & §18.1 Audit
export interface AuditEventItem {
  audit_id: string;
  actor_id: string;
  actor_email: string;
  actor_role: 'admin' | 'member' | 'system';
  action: string;
  resource_type: 'node' | 'enrollment' | 'member' | 'policy' | 'update_batch' | 'device';
  resource_id: string;
  request_id: string;
  operation_id: string | null;
  result: 'succeeded' | 'accepted' | 'rejected' | 'conflict';
  occurred_at: string;
  sanitized_diff: {
    before?: Record<string, unknown>;
    after?: Record<string, unknown>;
  };
}

// §14.3 & §18.4 Member Portal endpoints
export interface MemberExitItem {
  exit_id: string;
  node_id: string;
  name: string;
  region: string;
  status: 'available' | 'degraded' | 'unavailable';
  note_zh: string;
  note_en: string;
}

export interface MemberMeResponse {
  member: MemberItem;
  management_url: string;
}

export interface CheckoutSessionResponse {
  checkout_session_id: string;
  hosted_url: string;
  expires_at: string;
}

export interface BillingPortalSessionResponse {
  portal_session_id: string;
  hosted_url: string;
}
