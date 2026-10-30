/**
 * Generated OpenAPI contract type exports and schema aliases (`contracts/openapi.yaml` -> `contracts/ts/schema.d.ts`).
 *
 * IMPORTANT:
 * - This module exports ONLY types derived from `@contracts/schema`.
 * - Never add hand-written API wire types here.
 * - Browser session bootstrap types (`/auth/*`, outside `/api/v1`) live in `./auth`.
 * - Portal-only UI view models and projections live in `./view-models`.
 */

import type { components, operations, paths } from '@contracts/schema';

export type { components, operations, paths };
export type SchemaComponents = components['schemas'];

// Error envelope & codes (§13.11, §14.6)
export type ErrorEnvelope = components['schemas']['ErrorEnvelope'];
export type ApiErrorEnvelope = components['schemas']['ErrorEnvelope'];
export type ErrorBody = components['schemas']['ErrorBody'];
export type ApiErrorObject = components['schemas']['ErrorBody'];
export type FieldError = components['schemas']['FieldError'];
export type ErrorCode = components['schemas']['ErrorCode'];
export type ApiErrorCode = components['schemas']['ErrorCode'];

// Common primitives & references (§10.4, §14.1)
export type PeriodId = components['schemas']['PeriodId'];
export type LocalizedText = components['schemas']['LocalizedText'];
export type ResourceType = components['schemas']['ResourceType'];
export type ResourceRef = components['schemas']['ResourceRef'];
export type ReasonRequest = components['schemas']['ReasonRequest'];
export type Signature = components['schemas']['Signature'];
export type Architecture = components['schemas']['Architecture'];
export type RatePair = components['schemas']['RatePair'];

// Node & Fleet states (§5, §13)
export type LifecycleState = components['schemas']['LifecycleState'];
export type ProvisioningState = components['schemas']['ProvisioningState'];
export type PublicationIntent = components['schemas']['PublicationIntent'];
export type PublicationObserved = components['schemas']['PublicationObserved'];
export type ControlStatus = components['schemas']['ControlStatus'];
export type ControlPlaneStatus = components['schemas']['ControlStatus'];
export type DataHealth = components['schemas']['DataHealth'];
export type SafetyStatus = components['schemas']['SafetyStatus'];
export type UpdateJobState = components['schemas']['UpdateJobState'];
export type UpdateResultState = components['schemas']['UpdateResultState'];
export type UpdateBatchState = components['schemas']['UpdateBatchState'];
export type HealthCheckStatus = components['schemas']['HealthCheckStatus'];
export type EntitlementState = components['schemas']['EntitlementState'];
export type PolicyType = components['schemas']['PolicyType'];

// Operations (§14.5, §14.8)
export type OperationStatus = components['schemas']['OperationStatus'];
export type OperationStepStatus = components['schemas']['OperationStepStatus'];
export type OperationKind = components['schemas']['OperationKind'];
export type Operation = components['schemas']['Operation'];
export type OperationResource = components['schemas']['Operation'];
export type OperationStep = components['schemas']['OperationStep'];

// Nodes & Enrollments (§5, §6, §14.2)
export type ConfigGeneration = components['schemas']['ConfigGeneration'];
export type NodeReleases = components['schemas']['NodeReleases'];
export type NodeUpdateSummary = components['schemas']['NodeUpdateSummary'];
export type NodeCapacity = components['schemas']['NodeCapacity'];
export type NodePlatform = components['schemas']['NodePlatform'];
export type NodeSummary = components['schemas']['NodeSummary'];
export type NodeSummaryItem = components['schemas']['NodeSummary'];
export type NodeDetail = components['schemas']['NodeDetail'];
export type NodeCheck = components['schemas']['NodeCheck'];
export type DnsPolicyEvidence = components['schemas']['DnsPolicyEvidence'];
export type DnsFeedEvidence = components['schemas']['DnsFeedEvidence'];
export type NodeError = components['schemas']['NodeError'];
export type StateDrift = components['schemas']['StateDrift'];
export type NodeList = components['schemas']['NodeList'];
export type NodeListResponse = components['schemas']['NodeList'];
export type NodePatch = components['schemas']['NodePatch'];
export type NodePatchRequest = components['schemas']['NodePatch'];
export type PublicationRequest = components['schemas']['PublicationRequest'];
export type EnrollmentDelivery = components['schemas']['EnrollmentDelivery'];
export type EnrollmentState = components['schemas']['EnrollmentState'];
export type EnrollmentCreateRequest = components['schemas']['EnrollmentCreateRequest'];
export type Enrollment = components['schemas']['Enrollment'];
export type EnrollmentRecord = components['schemas']['Enrollment'];
export type EnrollmentCreated = components['schemas']['EnrollmentCreated'];
export type EnrollmentCreateResponse = components['schemas']['EnrollmentCreated'];
export type MemberGroup = components['schemas']['MemberGroup'];
export type MemberGroupCreate = components['schemas']['MemberGroupCreate'];
export type MemberGroupList = components['schemas']['MemberGroupList'];
export type MemberGroupListResponse = components['schemas']['MemberGroupList'];

// Members & Entitlements (§9, §10, §14.2)
export type PlanRef = components['schemas']['PlanRef'];
export type EntitlementSummary = components['schemas']['EntitlementSummary'];
export type SubscriptionSummary = components['schemas']['SubscriptionSummary'];
export type UsageQualityFlags = components['schemas']['UsageQualityFlags'];
export type MemberUsageSummary = components['schemas']['MemberUsageSummary'];
export type MemberSummary = components['schemas']['MemberSummary'];
export type MemberDetail = components['schemas']['MemberDetail'];
export type MemberList = components['schemas']['MemberList'];
export type MemberListResponse = components['schemas']['MemberList'];
export type MemberPatch = components['schemas']['MemberPatch'];
export type MemberPatchRequest = components['schemas']['MemberPatch'];
export type GrantSource = components['schemas']['GrantSource'];
export type GrantCreateRequest = components['schemas']['GrantCreateRequest'];
export type EntitlementGrant = components['schemas']['EntitlementGrant'];

// Policies (§11, §12, §13.2, §14.2)
export type PolicyRevisionSummary = components['schemas']['PolicyRevisionSummary'];
export type PolicyDefinition = components['schemas']['PolicyDefinition'];
export type PolicyDefinitionList = components['schemas']['PolicyDefinitionList'];
export type PolicyDefinitionListResponse = components['schemas']['PolicyDefinitionList'];
export type PolicyActivateRequest = components['schemas']['PolicyActivateRequest'];
export type PolicyRevisionCreate = components['schemas']['PolicyRevisionCreate'];
export type PolicyRevision = components['schemas']['PolicyRevision'];
export type PlanPayload = components['schemas']['PlanPayload'];
export type PlanPrice = components['schemas']['PlanPrice'];
export type FairUsePayload = components['schemas']['FairUsePayload'];
export type PolicyMode = components['schemas']['FairUsePayload']['mode'];
export type FairUseTier = components['schemas']['FairUseTier'];
export type FairUseOverride = components['schemas']['FairUseOverride'];
export type DnsProfilePayload = components['schemas']['DnsProfilePayload'];
export type DomainRuleSetPayload = components['schemas']['DomainRuleSetPayload'];
export type DomainRule = components['schemas']['DomainRule'];
export type DnsRule = components['schemas']['DomainRule'];
export type ThreatFeedPayload = components['schemas']['ThreatFeedPayload'];
export type ThreatFeedStatus = components['schemas']['ThreatFeedStatus'];
export type ReleaseSupportPayload = components['schemas']['ReleaseSupportPayload'];

// Releases & Update Batches (§13, §14.2)
export type ReleaseManifestEnvelope = components['schemas']['ReleaseManifestEnvelope'];
export type ReleaseManifest = components['schemas']['ReleaseManifest'];
export type ComponentVersion = components['schemas']['ComponentVersion'];
export type ManifestComponents = components['schemas']['ManifestComponents'];
export type ManifestCompatibility = components['schemas']['ManifestCompatibility'];
export type ManifestArtifact = components['schemas']['ManifestArtifact'];
export type ReleaseRegisterRequest = components['schemas']['ReleaseRegisterRequest'];
export type Release = components['schemas']['Release'];
export type ReleaseItem = components['schemas']['Release'];
export type ReleaseVerificationState =
  components['schemas']['Release']['verification_state'];
export type ReleaseSupportState = components['schemas']['Release']['support_state'];
export type FleetReleaseState = components['schemas']['FleetReleaseState'];
export type ReleaseList = components['schemas']['ReleaseList'];
export type ReleaseListResponse = components['schemas']['ReleaseList'];
export type UpdateBatchCreateRequest = components['schemas']['UpdateBatchCreateRequest'];
export type UpdateBatchAccepted = components['schemas']['UpdateBatchAccepted'];
export type UpdateBatchCounts = components['schemas']['UpdateBatchCounts'];
export type UpdateTargetBucket = keyof components['schemas']['UpdateBatchCounts'];
export type UpdateBatch = components['schemas']['UpdateBatch'];
export type UpdateBatchDetail = components['schemas']['UpdateBatch'];
export type UpdateTarget = components['schemas']['UpdateTarget'];
export type UpdateTargetItem = components['schemas']['UpdateTarget'];
export type UpdateJobSummary = components['schemas']['UpdateJobSummary'];
export type UpdateStage = components['schemas']['UpdateStage'];
export type UpdateHealth = components['schemas']['UpdateHealth'];
export type UpdateResult = components['schemas']['UpdateResult'];
export type UpdateUsageCheckpoint = components['schemas']['UpdateUsageCheckpoint'];
export type UpdateBatchRetryRequest = components['schemas']['UpdateBatchRetryRequest'];

// Audit Events (§14.2, §18.1)
export type AuditEvent = components['schemas']['AuditEvent'];
export type AuditEventItem = components['schemas']['AuditEvent'];
export type AuditEventList = components['schemas']['AuditEventList'];
export type AuditEventListResponse = components['schemas']['AuditEventList'];

// Member Self-Service (§14.3, §18.4)
export type NextAction = components['schemas']['NextAction'];
export type MemberSelf = components['schemas']['MemberSelf'];
export type DeviceStatus = components['schemas']['DeviceStatus'];
export type MemberDevice = components['schemas']['MemberDevice'];
export type MemberDeviceList = components['schemas']['MemberDeviceList'];
export type MemberDeviceListResponse = components['schemas']['MemberDeviceList'];
export type MemberExit = components['schemas']['MemberExit'];
export type MemberExitItem = components['schemas']['MemberExit'];
export type ExitAvailability = components['schemas']['MemberExit']['availability'];
export type MemberExitList = components['schemas']['MemberExitList'];
export type MemberExitListResponse = components['schemas']['MemberExitList'];
export type UsageTotals = components['schemas']['UsageTotals'];
export type UsageDay = components['schemas']['UsageDay'];
export type FairUseStatus = components['schemas']['FairUseStatus'];
export type MemberUsage = components['schemas']['MemberUsage'];
export type MemberUsageResponse = components['schemas']['MemberUsage'];
export type CheckoutSessionRequest = components['schemas']['CheckoutSessionRequest'];
export type CheckoutSession = components['schemas']['CheckoutSession'];
export type CheckoutSessionResponse = components['schemas']['CheckoutSession'];
export type BillingPortalSessionRequest =
  components['schemas']['BillingPortalSessionRequest'];
export type BillingPortalSession = components['schemas']['BillingPortalSession'];
export type BillingPortalSessionResponse = components['schemas']['BillingPortalSession'];
