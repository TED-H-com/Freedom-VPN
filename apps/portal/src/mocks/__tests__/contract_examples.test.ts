import { describe, expect, it } from 'vitest';
import { KNOWN_API_ERROR_CODES } from '../../api/errors';
import type {
  Architecture,
  ControlPlaneStatus,
  EnrollmentState,
  EntitlementState,
  ExitAvailability,
  LifecycleState,
  OperationStatus,
  OperationStepStatus,
  PolicyMode,
  PolicyType,
  ProvisioningState,
  PublicationIntent,
  PublicationObserved,
  ReleaseSupportState,
  ReleaseVerificationState,
  SafetyStatus,
  UpdateBatchState,
  UpdateJobState,
} from '../../api/types';
import { isValidDecimalString } from '../../lib/decimal';
import { CONTRACT_EXAMPLES_USED_BY_MOCKS } from '../data';

const DECIMAL_KEY_NAMES = new Set([
  'resource_version',
  'generation',
  'desired',
  'applied',
  'desired_generation',
  'observed_generation',
  'expected_generation',
  'revision',
  'release_sequence',
  'fleet_generation',
  'upload_bytes',
  'download_bytes',
  'total_bytes',
  'threshold_bytes',
  'max_size_bytes',
  'installed_size_bytes',
  'size_bytes',
  'current_resource_version',
]);

const TIMESTAMP_KEY_NAMES = new Set([
  'created_at',
  'updated_at',
  'completed_at',
  'observed_at',
  'occurred_at',
  'expires_at',
  'consumed_at',
  'revoked_at',
  'retired_at',
  'last_seen_at',
  'manifest_created_at',
  'registered_at',
  'started_at',
  'finished_at',
  'last_result_at',
  'last_observed_at',
  'valid_from',
  'valid_until',
  'effective_service_end',
  'paid_through',
  'grace_until',
  'computed_at',
  'last_reconciled_at',
  'fair_use_reset_at',
  'authorized_at',
  'period_start',
  'period_end',
  'aggregate_as_of',
  'resets_at',
  'as_of',
  'effective_at',
]);

const ARCHITECTURES: readonly Architecture[] = ['amd64', 'arm64'];
const LIFECYCLE_STATES: readonly LifecycleState[] = [
  'enrolling',
  'registered',
  'revoked',
  'retired',
];
const PUBLICATION_INTENTS: readonly PublicationIntent[] = [
  'unpublished',
  'published',
  'disabled',
];
const PUBLICATION_OBSERVED: readonly PublicationObserved[] = [
  'absent',
  'applying',
  'active',
  'withdrawn',
  'error',
];
const PROVISIONING_STATES: readonly ProvisioningState[] = [
  'pending',
  'applying',
  'ready',
  'failed',
];
const CONTROL_STATUSES: readonly ControlPlaneStatus[] = ['online', 'stale', 'offline'];
const SAFETY_STATUSES: readonly SafetyStatus[] = [
  'unknown',
  'passed',
  'failed',
  'expired',
];
const ENROLLMENT_STATES: readonly EnrollmentState[] = [
  'pending',
  'consumed',
  'expired',
  'revoked',
];
const ENTITLEMENT_STATES: readonly EntitlementState[] = [
  'pending_payment',
  'active',
  'cancel_at_period_end',
  'past_due',
  'grace',
  'expired',
  'suspended',
  'closed',
];
const EXIT_AVAILABILITIES: readonly ExitAvailability[] = [
  'available',
  'temporarily_unavailable',
];
const OPERATION_STATUSES: readonly OperationStatus[] = [
  'pending',
  'running',
  'succeeded',
  'partially_succeeded',
  'failed',
  'superseded',
];
const OPERATION_STEP_STATUSES: readonly OperationStepStatus[] = [
  'pending',
  'running',
  'succeeded',
  'failed',
  'not_required',
  'cancelled',
];
const POLICY_TYPES: readonly PolicyType[] = [
  'fair_use',
  'plan',
  'dns_profile',
  'domain_rule_set',
  'threat_feed',
  'release_support',
];
const POLICY_MODES: readonly PolicyMode[] = ['shadow', 'enforce'];
const RELEASE_VERIFICATION_STATES: readonly ReleaseVerificationState[] = [
  'verifying',
  'verified',
  'rejected',
];
const RELEASE_SUPPORT_STATES: readonly ReleaseSupportState[] = ['approved', 'revoked'];
const UPDATE_BATCH_STATES: readonly UpdateBatchState[] = [
  'accepted',
  'waiting',
  'running',
  'completed',
  'completed_with_exceptions',
];
const UPDATE_JOB_STATES: readonly UpdateJobState[] = [
  'queued',
  'waiting_online',
  'downloading',
  'verifying',
  'applying',
  'checking',
  'succeeded',
  'already_current',
  'failed',
  'rolled_back',
  'superseded',
  'superseded_incarnation',
  'cancelled_revoked',
  'cancelled_retired',
];

function walkObject(
  node: unknown,
  path: string,
  visitor: (key: string, value: unknown, fullPath: string) => void,
): void {
  if (!node || typeof node !== 'object') return;
  if (Array.isArray(node)) {
    node.forEach((item, index) => walkObject(item, `${path}[${index}]`, visitor));
    return;
  }
  for (const [k, v] of Object.entries(node as Record<string, unknown>)) {
    const childPath = path ? `${path}.${k}` : k;
    visitor(k, v, childPath);
    walkObject(v, childPath, visitor);
  }
}

describe('contracts/examples/valid/*.json (Deliverable 5 Contract Drift Guard)', () => {
  it('TestContractExamples_RequiredKeysPresentAcrossAllMockedExamples', () => {
    const ex = CONTRACT_EXAMPLES_USED_BY_MOCKS;

    expect(Object.keys(ex).length).toBeGreaterThanOrEqual(24);

    // NodeList & NodeDetail required keys
    expect(ex.NodeList).toHaveProperty('items');
    expect(ex.NodeList).toHaveProperty('next_cursor');
    expect(ex.NodeList.items[0]).toEqual(
      expect.objectContaining({
        node_id: expect.any(String),
        name: expect.any(String),
        region: expect.any(String),
        lifecycle_state: expect.any(String),
        provisioning_state: expect.any(String),
        publication_intent: expect.any(String),
        publication_observed: expect.any(String),
        control_status: expect.any(String),
        data_health: expect.any(String),
        safety_status: expect.any(String),
        config_generation: expect.objectContaining({
          desired: expect.any(String),
        }),
        releases: expect.objectContaining({
          current_release_id: expect.any(String),
        }),
        resource_version: expect.any(String),
      }),
    );

    expect(ex.NodeDetail).toEqual(
      expect.objectContaining({
        node_id: expect.any(String),
        lifecycle_state: expect.any(String),
        provisioning_state: expect.any(String),
        publication_intent: expect.any(String),
        publication_observed: expect.any(String),
        config_generation: expect.objectContaining({
          desired: expect.any(String),
        }),
        member_group_ids: expect.any(Array),
        checks: expect.any(Array),
        recent_errors: expect.any(Array),
        drift: expect.any(Array),
        created_at: expect.any(String),
      }),
    );

    // Enrollment & EnrollmentCreated required keys
    expect(ex.EnrollmentCreated).toEqual(
      expect.objectContaining({
        enrollment: expect.objectContaining({
          enrollment_id: expect.any(String),
          delivery: expect.any(String),
          state: expect.any(String),
        }),
        token: expect.any(String),
        secret_available: true,
      }),
    );
    expect(ex.EnrollmentCreatedReplay.secret_available).toBe(false);
    expect(ex.EnrollmentCreatedReplay.token).toBeNull();
    expect(ex.Enrollment).toEqual(
      expect.objectContaining({
        enrollment_id: expect.any(String),
        delivery: expect.any(String),
        state: expect.any(String),
      }),
    );

    // ReleaseList & UpdateBatch required keys
    expect(ex.ReleaseList).toHaveProperty('items');
    expect(ex.ReleaseList).toHaveProperty('fleet');
    expect(ex.UpdateBatch).toEqual(
      expect.objectContaining({
        batch_id: expect.any(String),
        operation_id: expect.any(String),
        release_id: expect.any(String),
        state: expect.any(String),
        target_count: expect.any(Number),
        counts: expect.any(Object),
        targets_url: expect.any(String),
        targets: expect.any(Array),
      }),
    );

    // MemberList, MemberDetail, MemberSelf, MemberDeviceList, MemberExitList, MemberUsage
    expect(ex.MemberList.items.length).toBeGreaterThan(0);
    expect(ex.MemberDetail).toEqual(
      expect.objectContaining({
        member_id: expect.any(String),
        display_name: expect.any(String),
        entitlement: expect.any(Object),
        grants: expect.any(Array),
        usage: expect.any(Object),
        suspension: expect.any(Object),
        resource_version: expect.any(String),
      }),
    );
    expect(ex.MemberSelf).toEqual(
      expect.objectContaining({
        member_id: expect.any(String),
        display_name: expect.any(String),
        entitlement: expect.any(Object),
        limits: expect.any(Object),
        next_actions: expect.any(Array),
      }),
    );
    expect(ex.MemberDeviceList).toEqual(
      expect.objectContaining({
        items: expect.any(Array),
        device_limit: expect.any(Number),
        authorized_count: expect.any(Number),
      }),
    );
    expect(ex.MemberExitList.items[0]).toEqual(
      expect.objectContaining({
        exit_id: expect.any(String),
        display_name: expect.any(String),
        region: expect.any(String),
        availability: expect.any(String),
      }),
    );
    expect(ex.MemberUsage).toEqual(
      expect.objectContaining({
        period_id: expect.any(String),
        period_start: expect.any(String),
        period_end: expect.any(String),
        granularity: 'day',
        metering_version: 'forward_admission_v1',
        totals: expect.any(Object),
        quality: expect.any(Object),
        aggregate_as_of: expect.any(String),
        days: expect.any(Array),
        fair_use: expect.any(Object),
      }),
    );

    // Operation, AuditEventList, CheckoutSession, BillingPortalSession, ErrorEnvelope
    expect(ex.Operation).toEqual(
      expect.objectContaining({
        operation_id: expect.any(String),
        kind: expect.any(String),
        status: expect.any(String),
        resource: expect.any(Object),
        steps: expect.any(Array),
        request_id: expect.any(String),
      }),
    );
    expect(ex.AuditEventList.items[0]).toEqual(
      expect.objectContaining({
        audit_event_id: expect.any(String),
        occurred_at: expect.any(String),
        actor: expect.any(Object),
        action: expect.any(String),
        resource: expect.any(Object),
        request_id: expect.any(String),
        result: expect.any(String),
      }),
    );
    expect(ex.CheckoutSession).toHaveProperty('checkout_url');
    expect(ex.BillingPortalSession).toHaveProperty('portal_url');
    expect(ex.ErrorEnvelope.error).toEqual(
      expect.objectContaining({
        code: expect.any(String),
        message: expect.any(String),
        request_id: expect.any(String),
        retryable: expect.any(Boolean),
      }),
    );
  });

  it('TestContractExamples_DecimalStringsMatchCanonicalPatternAndTimestampsEndWithZ', () => {
    const decimalRegex = /^(0|[1-9][0-9]*)$/;
    let checkedDecimals = 0;
    let checkedTimestamps = 0;

    for (const [exampleName, exampleObj] of Object.entries(
      CONTRACT_EXAMPLES_USED_BY_MOCKS,
    )) {
      walkObject(exampleObj, exampleName, (key, value, fullPath) => {
        if (DECIMAL_KEY_NAMES.has(key) && value !== null && value !== undefined) {
          expect(
            typeof value === 'string' &&
              decimalRegex.test(value) &&
              isValidDecimalString(value),
            `Expected ${fullPath} to be a valid uint64 decimal string, got ${JSON.stringify(value)}`,
          ).toBe(true);
          checkedDecimals += 1;
        }

        if ((key === 'up_bps' || key === 'down_bps') && typeof value === 'string') {
          expect(
            decimalRegex.test(value) && isValidDecimalString(value),
            `Expected ${fullPath} string bps to match decimal pattern`,
          ).toBe(true);
          checkedDecimals += 1;
        }

        if (TIMESTAMP_KEY_NAMES.has(key) && value !== null && value !== undefined) {
          expect(
            typeof value === 'string' &&
              value.endsWith('Z') &&
              !Number.isNaN(Date.parse(value)),
            `Expected ${fullPath} to be a UTC RFC 3339 timestamp ending in Z, got ${JSON.stringify(value)}`,
          ).toBe(true);
          checkedTimestamps += 1;
        }
      });
    }

    expect(checkedDecimals).toBeGreaterThan(20);
    expect(checkedTimestamps).toBeGreaterThan(30);
  });

  it('TestContractExamples_EnumValuesBelongToGeneratedUnionMembers', () => {
    const ex = CONTRACT_EXAMPLES_USED_BY_MOCKS;

    for (const node of ex.NodeList.items) {
      expect(LIFECYCLE_STATES).toContain(node.lifecycle_state);
      expect(PUBLICATION_INTENTS).toContain(node.publication_intent);
      expect(PUBLICATION_OBSERVED).toContain(node.publication_observed);
      expect(PROVISIONING_STATES).toContain(node.provisioning_state);
      expect(CONTROL_STATUSES).toContain(node.control_status);
      expect(SAFETY_STATUSES).toContain(node.safety_status);
    }

    if (ex.NodeDetail.platform) {
      expect(ARCHITECTURES).toContain(ex.NodeDetail.platform.arch);
    }
    expect(PUBLICATION_INTENTS).toContain(ex.NodeDetail.publication_intent);
    expect(ENROLLMENT_STATES).toContain(ex.Enrollment.state);
    expect(ENROLLMENT_STATES).toContain(ex.EnrollmentCreated.enrollment.state);

    for (const rel of ex.ReleaseList.items) {
      expect(RELEASE_VERIFICATION_STATES).toContain(rel.verification_state);
      expect(RELEASE_SUPPORT_STATES).toContain(rel.support_state);
      for (const arch of rel.architectures) {
        expect(ARCHITECTURES).toContain(arch);
      }
    }

    expect(UPDATE_BATCH_STATES).toContain(ex.UpdateBatch.state);
    expect(UPDATE_BATCH_STATES).toContain(ex.UpdateBatchAccepted.state);
    for (const target of ex.UpdateBatch.targets) {
      expect(UPDATE_JOB_STATES).toContain(target.state);
    }

    for (const m of ex.MemberList.items) {
      expect(ENTITLEMENT_STATES).toContain(m.entitlement.state);
    }
    expect(ENTITLEMENT_STATES).toContain(ex.MemberDetail.entitlement.state);
    expect(ENTITLEMENT_STATES).toContain(ex.MemberSelf.entitlement.state);

    for (const exit of ex.MemberExitList.items) {
      expect(EXIT_AVAILABILITIES).toContain(exit.availability);
    }

    expect(OPERATION_STATUSES).toContain(ex.Operation.status);
    for (const step of ex.Operation.steps) {
      expect(OPERATION_STEP_STATUSES).toContain(step.status);
    }

    for (const pol of ex.PolicyDefinitionList.items) {
      expect(POLICY_TYPES).toContain(pol.type);
      if (pol.active_revision?.type === 'fair_use') {
        expect(POLICY_MODES).toContain(pol.active_revision.payload.mode);
      }
    }

    expect(KNOWN_API_ERROR_CODES).toContain(ex.ErrorEnvelope.error.code);
    expect(KNOWN_API_ERROR_CODES).toContain(ex.ErrorEnvelopeValidation.error.code);
  });
});
