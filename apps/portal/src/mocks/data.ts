/**
 * Contract-sourced mock state derived directly from `contracts/examples/valid/*.json`
 * (Deliverable 5, §14, §18).
 *
 * Every response body served by MSW handlers starts from the canonical valid JSON
 * examples in `contracts/examples/valid/` so mocks cannot drift from `contracts/openapi.yaml`.
 */

import auditEventListJson from '../../../../contracts/examples/valid/AuditEventList.json';
import billingPortalSessionJson from '../../../../contracts/examples/valid/BillingPortalSession.json';
import checkoutSessionJson from '../../../../contracts/examples/valid/CheckoutSession.json';
import dnsPolicyEvidenceJson from '../../../../contracts/examples/valid/DnsPolicyEvidence.json';
import enrollmentJson from '../../../../contracts/examples/valid/Enrollment.json';
import enrollmentCreatedJson from '../../../../contracts/examples/valid/EnrollmentCreated.json';
import enrollmentCreatedReplayJson from '../../../../contracts/examples/valid/EnrollmentCreated.replay.json';
import errorEnvelopeJson from '../../../../contracts/examples/valid/ErrorEnvelope.json';
import errorEnvelopeValidationJson from '../../../../contracts/examples/valid/ErrorEnvelope.validation.json';
import memberDetailJson from '../../../../contracts/examples/valid/MemberDetail.json';
import memberDeviceListJson from '../../../../contracts/examples/valid/MemberDeviceList.json';
import memberExitListJson from '../../../../contracts/examples/valid/MemberExitList.json';
import memberListJson from '../../../../contracts/examples/valid/MemberList.json';
import memberSelfJson from '../../../../contracts/examples/valid/MemberSelf.json';
import memberUsageJson from '../../../../contracts/examples/valid/MemberUsage.json';
import nodeDetailJson from '../../../../contracts/examples/valid/NodeDetail.json';
import nodeDetailDnsEvidenceJson from '../../../../contracts/examples/valid/NodeDetail.dns-evidence.json';
import nodeDetailDnsEvidenceNullJson from '../../../../contracts/examples/valid/NodeDetail.dns-evidence-null.json';
import nodeListJson from '../../../../contracts/examples/valid/NodeList.json';
import operationJson from '../../../../contracts/examples/valid/Operation.json';
import policyDefinitionListJson from '../../../../contracts/examples/valid/PolicyDefinitionList.json';
import policyRevisionJson from '../../../../contracts/examples/valid/PolicyRevision.json';
import policyRevisionCreateDnsProfileJson from '../../../../contracts/examples/valid/PolicyRevisionCreate.dns-profile.json';
import policyRevisionCreateDomainRuleSetJson from '../../../../contracts/examples/valid/PolicyRevisionCreate.domain-rule-set.json';
import policyRevisionCreateFairUseJson from '../../../../contracts/examples/valid/PolicyRevisionCreate.fair-use.json';
import policyRevisionCreatePlanJson from '../../../../contracts/examples/valid/PolicyRevisionCreate.plan.json';
import policyRevisionCreateReleaseSupportJson from '../../../../contracts/examples/valid/PolicyRevisionCreate.release-support.json';
import policyRevisionCreateThreatFeedJson from '../../../../contracts/examples/valid/PolicyRevisionCreate.threat-feed.json';
import releaseJson from '../../../../contracts/examples/valid/Release.json';
import releaseListJson from '../../../../contracts/examples/valid/ReleaseList.json';
import updateBatchJson from '../../../../contracts/examples/valid/UpdateBatch.json';
import updateBatchAcceptedJson from '../../../../contracts/examples/valid/UpdateBatchAccepted.json';
import updateResultNullCandidateHealthJson from '../../../../contracts/examples/valid/UpdateResult.null-candidate-health.json';
import updateResultRolledBackJson from '../../../../contracts/examples/valid/UpdateResult.rolled-back.json';

import type { AuthSession } from '../api/auth';
import type {
  ApiErrorEnvelope,
  AuditEventItem,
  AuditEventListResponse,
  BillingPortalSessionResponse,
  CheckoutSessionResponse,
  DnsPolicyEvidence,
  EnrollmentCreateResponse,
  EnrollmentRecord,
  MemberDetail,
  MemberDeviceListResponse,
  MemberExitListResponse,
  MemberGroup,
  MemberListResponse,
  MemberSelf,
  MemberUsageResponse,
  NodeDetail,
  NodeListResponse,
  OperationResource,
  PolicyDefinition,
  PolicyDefinitionListResponse,
  PolicyRevision,
  ReleaseListResponse,
  UpdateBatchAccepted,
  UpdateBatchDetail,
  UpdateResult,
} from '../api/types';
import {
  buildDnsPolicyViewModel,
  buildFupPolicyViewModel,
  type UpdateTargetWithResult,
} from '../api/view-models';

export const CONTRACT_EXAMPLES_USED_BY_MOCKS = {
  AuditEventList: auditEventListJson as unknown as AuditEventListResponse,
  BillingPortalSession:
    billingPortalSessionJson as unknown as BillingPortalSessionResponse,
  CheckoutSession: checkoutSessionJson as unknown as CheckoutSessionResponse,
  DnsPolicyEvidence: dnsPolicyEvidenceJson as unknown as DnsPolicyEvidence,
  Enrollment: enrollmentJson as unknown as EnrollmentRecord,
  EnrollmentCreated: enrollmentCreatedJson as unknown as EnrollmentCreateResponse,
  EnrollmentCreatedReplay:
    enrollmentCreatedReplayJson as unknown as EnrollmentCreateResponse,
  ErrorEnvelope: errorEnvelopeJson as unknown as ApiErrorEnvelope,
  ErrorEnvelopeValidation: errorEnvelopeValidationJson as unknown as ApiErrorEnvelope,
  MemberDetail: memberDetailJson as unknown as MemberDetail,
  MemberDeviceList: memberDeviceListJson as unknown as MemberDeviceListResponse,
  MemberExitList: memberExitListJson as unknown as MemberExitListResponse,
  MemberList: memberListJson as unknown as MemberListResponse,
  MemberSelf: memberSelfJson as unknown as MemberSelf,
  MemberUsage: memberUsageJson as unknown as MemberUsageResponse,
  NodeDetail: nodeDetailJson as unknown as NodeDetail,
  NodeDetailDnsEvidence: nodeDetailDnsEvidenceJson as unknown as NodeDetail,
  NodeDetailDnsEvidenceNull: nodeDetailDnsEvidenceNullJson as unknown as NodeDetail,
  NodeList: nodeListJson as unknown as NodeListResponse,
  Operation: operationJson as unknown as OperationResource,
  PolicyDefinitionList:
    policyDefinitionListJson as unknown as PolicyDefinitionListResponse,
  PolicyRevision: policyRevisionJson as unknown as PolicyRevision,
  PolicyRevisionCreateDnsProfile: policyRevisionCreateDnsProfileJson,
  PolicyRevisionCreateDomainRuleSet: policyRevisionCreateDomainRuleSetJson,
  PolicyRevisionCreateFairUse: policyRevisionCreateFairUseJson,
  PolicyRevisionCreatePlan: policyRevisionCreatePlanJson,
  PolicyRevisionCreateReleaseSupport: policyRevisionCreateReleaseSupportJson,
  PolicyRevisionCreateThreatFeed: policyRevisionCreateThreatFeedJson,
  Release: releaseJson,
  ReleaseList: releaseListJson as unknown as ReleaseListResponse,
  UpdateBatch: updateBatchJson as unknown as UpdateBatchDetail,
  UpdateBatchAccepted: updateBatchAcceptedJson as unknown as UpdateBatchAccepted,
  UpdateResultNullCandidateHealth:
    updateResultNullCandidateHealthJson as unknown as UpdateResult,
  UpdateResultRolledBack: updateResultRolledBackJson as unknown as UpdateResult,
} as const;

export const PRIMARY_NODE_ID = nodeDetailJson.node_id;
export const SAFETY_FAILED_NODE_ID = 'caba5aec-4b2b-432c-9512-211a7b3e96db';
export const OFFLINE_NODE_ID = 'b4fe029a-ecbf-4b1a-9ee7-de399791ae6e';
export const PRIMARY_ENROLLMENT_ID = enrollmentJson.enrollment_id;
export const PRIMARY_MEMBER_ID = memberDetailJson.member_id;
export const PRIMARY_DEVICE_ID =
  memberDeviceListJson.items[0]?.device_id ?? '2df6c7df-e7e8-4bd4-9f2c-a5836dd0d8ea';
export const PRIMARY_RELEASE_ID =
  releaseListJson.items[0]?.release_id ?? '6d2797d0-b9a5-4724-96af-9b3039d351b1';
export const FALLBACK_RELEASE_ID =
  releaseListJson.fleet.fleet_default_release_id ??
  'cdb25ffa-5395-400c-ad3e-b32899f14bb8';
export const PRIMARY_BATCH_ID = updateBatchJson.batch_id;
export const PRIMARY_FUP_DEFINITION_ID =
  policyDefinitionListJson.items[0]?.definition_id ??
  '4e1af170-8b6f-4e0e-9117-f61927eb6b07';
export const PRIMARY_DNS_RULESET_DEFINITION_ID = '2398841d-bd48-4a11-97a4-87c59517157b';
export const PRIMARY_GROUP_ID = '63575bae-2d23-45c0-ae11-ba6bb9ace34a';
export const SECONDARY_GROUP_ID = '73575bae-2d23-45c0-ae11-ba6bb9ace34b';

export type MockPersona =
  | 'anonymous'
  | 'admin_no_mfa'
  | 'admin'
  | 'member_active'
  | 'member_throttled'
  | 'member_pending_payment'
  | 'member_cancel_at_period_end'
  | 'member_grace'
  | 'member_expired'
  | 'member_suspended';

function clone<T>(value: T): T {
  return structuredClone(value);
}

export function buildPersonaSession(persona: MockPersona): AuthSession {
  if (persona === 'anonymous') {
    return { authenticated: false };
  }
  if (persona === 'admin_no_mfa') {
    return {
      authenticated: true,
      role: 'admin',
      display_label: 'ops-admin@example.com',
      mfa_verified: false,
    };
  }
  if (persona === 'admin') {
    return {
      authenticated: true,
      role: 'admin',
      display_label: 'ops-admin@example.com',
      mfa_verified: true,
    };
  }
  return {
    authenticated: true,
    role: 'member',
    display_label:
      memberSelfJson.email ?? memberSelfJson.display_name ?? 'member-a@example.com',
    mfa_verified: false,
    member_id: PRIMARY_MEMBER_ID,
  };
}

export function buildMemberSelfForPersona(persona: MockPersona): MemberSelf {
  const base = clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.MemberSelf);
  switch (persona) {
    case 'member_pending_payment':
      base.entitlement.state = 'pending_payment';
      base.entitlement.paid_through = null;
      base.entitlement.effective_service_end = null;
      base.next_actions = ['wait_for_payment_confirmation'];
      break;
    case 'member_cancel_at_period_end':
      base.entitlement.state = 'cancel_at_period_end';
      base.entitlement.cancel_at_period_end = true;
      if (base.subscription) {
        base.subscription.cancel_at_period_end = true;
      }
      break;
    case 'member_grace':
      base.entitlement.state = 'past_due';
      base.entitlement.grace_until = '2026-10-31T00:00:00Z';
      if (base.subscription) {
        base.subscription.provider_status = 'past_due';
      }
      base.next_actions = ['renew_subscription'];
      break;
    case 'member_expired':
      base.entitlement.state = 'expired';
      base.entitlement.effective_service_end = '2026-10-01T00:00:00Z';
      base.entitlement.paid_through = '2026-09-28T00:00:00Z';
      base.next_actions = ['complete_payment'];
      break;
    case 'member_suspended':
      base.entitlement.state = 'suspended';
      base.entitlement.admin_suspended = true;
      base.next_actions = ['contact_support'];
      break;
    default:
      break;
  }
  return base;
}

export function buildMemberUsageForPersona(persona: MockPersona): MemberUsageResponse {
  const base = clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.MemberUsage);
  if (persona === 'member_throttled') {
    base.quality.gap = true;
    base.quality.estimated = true;
    base.fair_use.state = 'adjusted';
    return base;
  }
  base.totals = {
    upload_bytes: '40000000000',
    download_bytes: '80000000000',
    total_bytes: '120000000000',
  };
  base.quality = {
    estimated: false,
    gap: false,
    reconciliation_in_progress: false,
  };
  base.fair_use = {
    ...base.fair_use,
    state: 'normal',
    tier_index: 0,
    tier_name: {
      'zh-TW': '正常速度',
      en: 'Normal speed',
    },
    rate: {
      down_bps: 200000000,
      up_bps: 100000000,
    },
    reason: null,
  };
  return base;
}

function buildInitialNodeDetails(): Record<string, NodeDetail> {
  const primary = clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.NodeDetail);
  primary.publication_intent = 'published';
  primary.publication_observed = 'active';
  primary.member_forwarding_observed = true;
  primary.group_ids = [PRIMARY_GROUP_ID];
  primary.member_group_ids = [PRIMARY_GROUP_ID];
  primary.config_generation = {
    desired: '128',
    applied: '128',
  };
  primary.releases = {
    current_release_id: PRIMARY_RELEASE_ID,
    desired_release_id: PRIMARY_RELEASE_ID,
    last_good_release_id: FALLBACK_RELEASE_ID,
  };
  primary.update = null;
  primary.drift = [];
  primary.checks = [
    {
      name: 'internet_only',
      status: 'passed',
      observed_at: '2026-10-28T14:06:30Z',
      error_code: null,
      detail: 'inet fvpn table verified; RFC1918 egress blocked',
    },
    {
      name: 'meter_path',
      status: 'passed',
      observed_at: '2026-10-28T14:06:30Z',
      error_code: null,
      detail: 'forward_admission_v1 counters active on wg-nb0',
    },
  ];
  primary.dns_policy_evidence = {
    ...clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.DnsPolicyEvidence),
    feeds: [
      {
        feed_code: 'malware-domains-default',
        content_sha256:
          '7777777777777777777777777777777777777777777777777777777777777777',
        rule_count: 1420,
      },
    ],
  };

  const safetyFailed = clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.NodeDetail);
  safetyFailed.node_id = SAFETY_FAILED_NODE_ID;
  safetyFailed.name = 'Taipei 2';
  safetyFailed.region = 'tw-taipei';
  safetyFailed.publication_intent = 'published';
  safetyFailed.publication_observed = 'withdrawn';
  safetyFailed.safety_status = 'failed';
  safetyFailed.member_forwarding_observed = false;
  safetyFailed.group_ids = [PRIMARY_GROUP_ID];
  safetyFailed.member_group_ids = [PRIMARY_GROUP_ID];
  safetyFailed.resource_version = '19';
  safetyFailed.dns_policy_evidence = null;
  safetyFailed.checks = [
    {
      name: 'internet_only',
      status: 'failed',
      observed_at: '2026-10-28T14:06:30Z',
      error_code: 'NODE_NOT_SERVICEABLE',
      detail: 'RFC1918 route leak detected on 10.0.0.0/8',
    },
    {
      name: 'meter_path',
      status: 'passed',
      observed_at: '2026-10-28T14:06:30Z',
      error_code: null,
      detail: 'forward_admission_v1 counters active on wg-nb0',
    },
  ];
  safetyFailed.recent_errors = [
    {
      code: 'NODE_NOT_SERVICEABLE',
      stage: 'enforce',
      message:
        'RFC1918 isolation probe detected unexpected route leak; forwarding disabled fail-closed.',
      observed_at: '2026-10-28T14:06:30Z',
    },
  ];

  const offlineNode = clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.NodeDetail);
  offlineNode.node_id = OFFLINE_NODE_ID;
  offlineNode.name = 'Singapore 3';
  offlineNode.region = 'sg-central';
  offlineNode.publication_intent = 'disabled';
  offlineNode.publication_observed = 'withdrawn';
  offlineNode.control_status = 'offline';
  offlineNode.data_health = 'unknown';
  offlineNode.safety_status = 'unknown';
  offlineNode.member_forwarding_observed = false;
  offlineNode.group_ids = [];
  offlineNode.member_group_ids = [];
  offlineNode.resource_version = '7';
  offlineNode.dns_policy_evidence = null;

  return {
    [primary.node_id]: primary,
    [safetyFailed.node_id]: safetyFailed,
    [offlineNode.node_id]: offlineNode,
  };
}

export function buildInitialPolicyDefinitions(): PolicyDefinitionListResponse {
  const baseList = clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.PolicyDefinitionList);
  const ruleSetCreate = clone(
    CONTRACT_EXAMPLES_USED_BY_MOCKS.PolicyRevisionCreateDomainRuleSet,
  );
  const threatFeedCreate = clone(
    CONTRACT_EXAMPLES_USED_BY_MOCKS.PolicyRevisionCreateThreatFeed,
  );
  const dnsProfileCreate = clone(
    CONTRACT_EXAMPLES_USED_BY_MOCKS.PolicyRevisionCreateDnsProfile,
  );
  const releaseSupportCreate = clone(
    CONTRACT_EXAMPLES_USED_BY_MOCKS.PolicyRevisionCreateReleaseSupport,
  );

  const ruleSetRevId = '2398841d-bd48-4a11-97a4-87c59517157b';
  const prevRuleSetRevId = '1198841d-bd48-4a11-97a4-87c59517157a';
  const ruleSetDef: PolicyDefinition = {
    definition_id: PRIMARY_DNS_RULESET_DEFINITION_ID,
    type: 'domain_rule_set',
    code: ruleSetCreate.code ?? 'standard',
    active_revision_id: ruleSetRevId,
    generation: '12',
    resource_version: '12',
    active_revision: {
      type: 'domain_rule_set',
      revision_id: ruleSetRevId,
      definition_id: PRIMARY_DNS_RULESET_DEFINITION_ID,
      revision: '12',
      schema_version: 1,
      payload: ruleSetCreate.payload as PolicyDefinition['active_revision'] extends {
        payload: infer P;
      }
        ? P
        : never,
      payload_sha256: '6666666666666666666666666666666666666666666666666666666666666666',
      reason: ruleSetCreate.reason,
      created_by: '6a7d5389-5347-41a4-9cd7-768e07072254',
      created_at: '2026-10-28T15:00:00Z',
    } as NonNullable<PolicyDefinition['active_revision']>,
    latest_revision: undefined,
    revisions: [
      {
        revision_id: ruleSetRevId,
        revision: '12',
        schema_version: 1,
        payload_sha256:
          '6666666666666666666666666666666666666666666666666666666666666666',
        reason: ruleSetCreate.reason,
        created_by: '6a7d5389-5347-41a4-9cd7-768e07072254',
        created_at: '2026-10-28T15:00:00Z',
      },
      {
        revision_id: prevRuleSetRevId,
        revision: '11',
        schema_version: 1,
        payload_sha256:
          '1111111111111111111111111111111111111111111111111111111111111111',
        reason: 'Previous known-good DNS ruleset.',
        created_by: '6a7d5389-5347-41a4-9cd7-768e07072254',
        created_at: '2026-10-27T15:00:00Z',
      },
    ],
    created_at: '2026-10-01T00:00:00Z',
    updated_at: '2026-10-28T15:00:00Z',
  };
  ruleSetDef.latest_revision = clone(ruleSetDef.active_revision);

  const threatFeedDefId = 'a5e85d12-74ef-4b99-a88b-b6842fe8945f';
  const threatFeedDef: PolicyDefinition = {
    definition_id: threatFeedDefId,
    type: 'threat_feed',
    code: threatFeedCreate.code ?? 'standard',
    active_revision_id: threatFeedDefId,
    generation: '4',
    resource_version: '4',
    active_revision: {
      type: 'threat_feed',
      revision_id: threatFeedDefId,
      definition_id: threatFeedDefId,
      revision: '4',
      schema_version: 1,
      payload: threatFeedCreate.payload,
      payload_sha256: '7777777777777777777777777777777777777777777777777777777777777777',
      reason: threatFeedCreate.reason,
      created_by: '6a7d5389-5347-41a4-9cd7-768e07072254',
      created_at: '2026-10-28T15:00:00Z',
    } as NonNullable<PolicyDefinition['active_revision']>,
    latest_revision: undefined,
    revisions: [
      {
        revision_id: threatFeedDefId,
        revision: '4',
        schema_version: 1,
        payload_sha256:
          '7777777777777777777777777777777777777777777777777777777777777777',
        reason: threatFeedCreate.reason,
        created_by: '6a7d5389-5347-41a4-9cd7-768e07072254',
        created_at: '2026-10-28T15:00:00Z',
      },
    ],
    threat_feed_status: {
      approved_content_sha256: threatFeedCreate.payload.content_sha256,
      last_good_content_sha256: threatFeedCreate.payload.content_sha256,
      last_good_rule_count: 1420,
      last_good_at: '2026-10-28T14:00:00Z',
      last_good_age_seconds: '3600',
      last_attempt_at: '2026-10-28T14:00:00Z',
      last_result: 'success',
      consecutive_failures: 0,
      last_error_code: null,
      last_error_message: null,
      next_due_at: '2026-10-28T20:00:00Z',
    },
    created_at: '2026-10-01T00:00:00Z',
    updated_at: '2026-10-28T15:00:00Z',
  };
  threatFeedDef.latest_revision = clone(threatFeedDef.active_revision);

  const threatFeedBootstrapDefId = 'b5e85d12-74ef-4b99-a88b-b6842fe89460';
  const threatFeedBootstrapDef: PolicyDefinition = {
    ...clone(threatFeedDef),
    definition_id: threatFeedBootstrapDefId,
    code: 'phishing-bootstrap',
    active_revision_id: threatFeedBootstrapDefId,
    threat_feed_status: {
      approved_content_sha256:
        'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
      last_good_content_sha256: null,
      last_good_rule_count: null,
      last_good_at: null,
      last_good_age_seconds: null,
      last_attempt_at: null,
      last_result: 'never',
      consecutive_failures: 0,
      last_error_code: null,
      last_error_message: null,
      next_due_at: null,
    },
  };

  const threatFeedFailingDefId = 'c5e85d12-74ef-4b99-a88b-b6842fe89461';
  const threatFeedFailingDef: PolicyDefinition = {
    ...clone(threatFeedDef),
    definition_id: threatFeedFailingDefId,
    code: 'botnet-c2-feed',
    active_revision_id: threatFeedFailingDefId,
    threat_feed_status: {
      approved_content_sha256:
        'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
      last_good_content_sha256:
        'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc',
      last_good_rule_count: 850,
      last_good_at: '2026-10-28T10:00:00Z',
      last_good_age_seconds: '18000',
      last_attempt_at: '2026-10-28T15:00:00Z',
      last_result: 'failed',
      consecutive_failures: 3,
      last_error_code: 'HASH_MISMATCH',
      last_error_message: 'Feed refresh failed; retaining last-good content.',
      next_due_at: '2026-10-28T15:15:00Z',
    },
  };

  const dnsProfileDefId = '3dac5c85-4d8c-4424-9a8d-0b15a37f3495';
  const dnsProfileDef: PolicyDefinition = {
    definition_id: dnsProfileDefId,
    type: 'dns_profile',
    code: dnsProfileCreate.code ?? 'standard',
    active_revision_id: dnsProfileDefId,
    generation: '6',
    resource_version: '6',
    active_revision: {
      type: 'dns_profile',
      revision_id: dnsProfileDefId,
      definition_id: dnsProfileDefId,
      revision: '6',
      schema_version: 1,
      payload: dnsProfileCreate.payload,
      payload_sha256: '9999999999999999999999999999999999999999999999999999999999999999',
      reason: dnsProfileCreate.reason,
      created_by: '6a7d5389-5347-41a4-9cd7-768e07072254',
      created_at: '2026-10-28T15:00:00Z',
    } as NonNullable<PolicyDefinition['active_revision']>,
    latest_revision: undefined,
    revisions: [
      {
        revision_id: dnsProfileDefId,
        revision: '6',
        schema_version: 1,
        payload_sha256:
          '9999999999999999999999999999999999999999999999999999999999999999',
        reason: dnsProfileCreate.reason,
        created_by: '6a7d5389-5347-41a4-9cd7-768e07072254',
        created_at: '2026-10-28T15:00:00Z',
      },
    ],
    created_at: '2026-10-01T00:00:00Z',
    updated_at: '2026-10-28T15:00:00Z',
  };
  dnsProfileDef.latest_revision = clone(dnsProfileDef.active_revision);

  const relSupDefId = '8e210b2c-f844-4ad9-88f6-7e6fa0cbbe68';
  const relSupDef: PolicyDefinition = {
    definition_id: relSupDefId,
    type: 'release_support',
    code: releaseSupportCreate.code ?? 'fleet',
    active_revision_id: relSupDefId,
    generation: '2',
    resource_version: '2',
    active_revision: {
      type: 'release_support',
      revision_id: relSupDefId,
      definition_id: relSupDefId,
      revision: '2',
      schema_version: 1,
      payload: releaseSupportCreate.payload,
      payload_sha256: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
      reason: releaseSupportCreate.reason,
      created_by: '6a7d5389-5347-41a4-9cd7-768e07072254',
      created_at: '2026-10-28T15:00:00Z',
    } as NonNullable<PolicyDefinition['active_revision']>,
    latest_revision: undefined,
    revisions: [
      {
        revision_id: relSupDefId,
        revision: '2',
        schema_version: 1,
        payload_sha256:
          'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        reason: releaseSupportCreate.reason,
        created_by: '6a7d5389-5347-41a4-9cd7-768e07072254',
        created_at: '2026-10-28T15:00:00Z',
      },
    ],
    created_at: '2026-10-01T00:00:00Z',
    updated_at: '2026-10-28T15:00:00Z',
  };
  relSupDef.latest_revision = clone(relSupDef.active_revision);

  baseList.items.push(
    ruleSetDef,
    threatFeedDef,
    threatFeedBootstrapDef,
    threatFeedFailingDef,
    dnsProfileDef,
    relSupDef,
  );
  return baseList;
}

function buildInitialUpdateBatch(): UpdateBatchDetail {
  const batch = clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.UpdateBatch);
  const templateTarget = batch.targets[0];
  if (templateTarget) {
    templateTarget.policy_generation = '128';
    templateTarget.policy_refs = [
      PRIMARY_FUP_DEFINITION_ID,
      PRIMARY_DNS_RULESET_DEFINITION_ID,
    ];

    const rolledBackResult = clone(
      CONTRACT_EXAMPLES_USED_BY_MOCKS.UpdateResultRolledBack,
    );
    const rolledBackTarget = clone(templateTarget) as UpdateTargetWithResult;
    rolledBackTarget.target_id = '9648dcdb-c82e-4611-8608-06895697e0b1';
    rolledBackTarget.node_id = SAFETY_FAILED_NODE_ID;
    rolledBackTarget.node_name = 'Taipei 2';
    rolledBackTarget.state = 'rolled_back';
    rolledBackTarget.current_release_id = FALLBACK_RELEASE_ID;
    rolledBackTarget.previous_release_id = rolledBackResult.previous_release_id;
    rolledBackTarget.health = rolledBackResult.health;
    rolledBackTarget.candidate_health = rolledBackResult.candidate_health ?? null;
    rolledBackTarget.last_result = rolledBackResult;
    rolledBackTarget.retryable = true;
    rolledBackTarget.needs_attention = true;
    rolledBackTarget.unconfirmed = false;
    rolledBackTarget.policy_generation = '127';
    rolledBackTarget.policy_refs = [];
    if (rolledBackTarget.current_job) {
      rolledBackTarget.current_job.state = 'rolled_back';
      rolledBackTarget.current_job.error_code = 'HEALTH_CHECK_FAILED';
    }

    const waitingOfflineTarget = clone(templateTarget);
    waitingOfflineTarget.target_id = 'a648dcdb-c82e-4611-8608-06895697e0b2';
    waitingOfflineTarget.node_id = OFFLINE_NODE_ID;
    waitingOfflineTarget.node_name = 'Singapore 3';
    waitingOfflineTarget.online_at_snapshot = false;
    waitingOfflineTarget.state = 'waiting_online';
    waitingOfflineTarget.attempts = 0;
    waitingOfflineTarget.current_job = null;
    waitingOfflineTarget.current_release_id = FALLBACK_RELEASE_ID;
    waitingOfflineTarget.retryable = true;
    waitingOfflineTarget.needs_attention = false;
    waitingOfflineTarget.unconfirmed = true;
    waitingOfflineTarget.policy_generation = null;
    waitingOfflineTarget.policy_refs = null;

    batch.targets = [templateTarget, rolledBackTarget, waitingOfflineTarget];
    batch.target_count = 3;
    batch.counts = {
      queued: 0,
      waiting_online: 1,
      downloading: 0,
      verifying: 0,
      applying: 0,
      checking: 0,
      succeeded: 1,
      already_current: 0,
      failed: 0,
      rolled_back: 1,
      superseded: 0,
      cancelled: 0,
    };
  }
  return batch;
}

function buildInitialReleaseList(): ReleaseListResponse {
  const relList = clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.ReleaseList);
  const firstRelease = relList.items[0];
  if (firstRelease) {
    const prevRelease = clone(firstRelease);
    prevRelease.release_id = FALLBACK_RELEASE_ID;
    prevRelease.product_version = '1.0.9';
    prevRelease.release_sequence = '41';
    prevRelease.is_fleet_default = true;
    if (prevRelease.compatibility) {
      prevRelease.compatibility.requires_service_restart = false;
    }
    relList.items.push(prevRelease);
  }
  return relList;
}

const initialPolicies = buildInitialPolicyDefinitions();

export const INITIAL_BATCH: Pick<UpdateBatchDetail, 'target_count' | 'counts'> = {
  target_count: 12,
  counts: {
    queued: 1,
    waiting_online: 1,
    downloading: 1,
    verifying: 1,
    applying: 1,
    checking: 1,
    succeeded: 1,
    already_current: 1,
    failed: 1,
    rolled_back: 1,
    superseded: 1,
    cancelled: 1,
  },
};

export const INITIAL_FUP_POLICY = {
  current: buildFupPolicyViewModel(initialPolicies)!,
};

export const INITIAL_DNS_POLICY = buildDnsPolicyViewModel(initialPolicies)!;

export interface MockState {
  persona: MockPersona;
  forceSession401: boolean;
  nodes: Record<string, NodeDetail>;
  groups: MemberGroup[];
  enrollments: Record<string, EnrollmentRecord>;
  releaseList: ReleaseListResponse;
  updateBatch: UpdateBatchDetail;
  memberList: MemberListResponse;
  memberDetail: MemberDetail;
  policyDefinitions: PolicyDefinitionListResponse;
  auditEvents: AuditEventItem[];
  memberExits: MemberExitListResponse;
  memberDevices: MemberDeviceListResponse;
  operations: Record<string, OperationResource>;
  operationPolls: Record<string, number>;
  idempotencyRecords: Record<string, { status: number; body: unknown }>;
}

export function createInitialMockState(persona: MockPersona = 'admin'): MockState {
  const baseOp = clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.Operation);
  return {
    persona,
    forceSession401: false,
    nodes: buildInitialNodeDetails(),
    groups: [
      {
        group_id: PRIMARY_GROUP_ID,
        code: 'standard-exits',
        name: 'Standard Member Exits',
        exit_count: 2,
      },
      {
        group_id: SECONDARY_GROUP_ID,
        code: 'default-members',
        name: 'Default Members Group',
        exit_count: 0,
      },
    ],
    enrollments: {
      [PRIMARY_ENROLLMENT_ID]: clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.Enrollment),
    },
    releaseList: buildInitialReleaseList(),
    updateBatch: buildInitialUpdateBatch(),
    memberList: clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.MemberList),
    memberDetail: clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.MemberDetail),
    policyDefinitions: buildInitialPolicyDefinitions(),
    auditEvents: clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.AuditEventList.items),
    memberExits: clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.MemberExitList),
    memberDevices: clone(CONTRACT_EXAMPLES_USED_BY_MOCKS.MemberDeviceList),
    operations: {
      [baseOp.operation_id]: baseOp,
    },
    operationPolls: {},
    idempotencyRecords: {},
  };
}
