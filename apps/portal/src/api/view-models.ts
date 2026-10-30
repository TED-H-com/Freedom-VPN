/**
 * Portal-only UI view models and pure projection functions.
 *
 * IMPORTANT:
 * - None of the types in this module are HTTP API wire types.
 * - All HTTP API wire types come from `@contracts/schema` via `./types`.
 * - These projections transform generated contract responses into UI-friendly
 *   view models without mutating or duplicating contract types.
 */

import type {
  DomainRule,
  DomainRuleSetPayload,
  DnsProfilePayload,
  FairUsePayload,
  LocalizedText,
  MemberListResponse,
  MemberSelf,
  MemberUsageResponse,
  NodeDetail,
  NodeListResponse,
  NodeSummaryItem,
  PolicyDefinition,
  PolicyDefinitionListResponse,
  PolicyMode,
  PolicyRevisionCreate,
  ReleaseSupportPayload,
  ThreatFeedPayload,
  UpdateBatchDetail,
  UpdateHealth,
  UpdateResult,
  UpdateTarget,
} from './types';

export interface WithEtag<T> {
  data: T;
  etag: string | null;
}

export type NodeSummaryPhraseKey =
  | 'online_serving'
  | 'online_safety_failed_not_serving'
  | 'online_disabled_not_serving'
  | 'ready_unpublished_not_serving'
  | 'offline_route_withdrawn_unconfirmed'
  | 'rolled_back_serving'
  | 'revoked_not_serving';

export interface NodeHealthDimensionsViewModel {
  control_online: boolean;
  data_path_available: boolean;
  policy_and_isolation_valid: boolean;
  update_complete: boolean;
}

export function isNodeServiceable(node: NodeSummaryItem | NodeDetail): boolean {
  return (
    node.lifecycle_state === 'registered' &&
    node.provisioning_state === 'ready' &&
    node.publication_intent === 'published' &&
    node.publication_observed === 'active' &&
    node.control_status === 'online' &&
    node.data_health === 'healthy' &&
    node.safety_status === 'passed' &&
    node.member_forwarding_observed === true
  );
}

export function deriveNodeHealthDimensions(
  node: NodeSummaryItem | NodeDetail,
): NodeHealthDimensionsViewModel {
  const updateComplete =
    !node.releases.desired_release_id ||
    node.releases.current_release_id === node.releases.desired_release_id;
  return {
    control_online: node.control_status === 'online',
    data_path_available:
      node.data_health === 'healthy' &&
      node.publication_observed === 'active' &&
      node.member_forwarding_observed === true,
    policy_and_isolation_valid:
      node.safety_status === 'passed' &&
      node.config_generation.applied === node.config_generation.desired,
    update_complete: updateComplete,
  };
}

export function deriveNodeSummaryPhrase(
  node: NodeSummaryItem | NodeDetail,
): NodeSummaryPhraseKey {
  if (node.lifecycle_state === 'revoked' || node.lifecycle_state === 'retired') {
    return 'revoked_not_serving';
  }
  if (node.safety_status === 'failed' || node.safety_status === 'expired') {
    return 'online_safety_failed_not_serving';
  }
  if (node.control_status === 'offline' && node.publication_observed === 'withdrawn') {
    return 'offline_route_withdrawn_unconfirmed';
  }
  if (node.publication_intent === 'disabled') {
    return 'online_disabled_not_serving';
  }
  if (node.publication_intent === 'unpublished') {
    return 'ready_unpublished_not_serving';
  }
  const serviceable = isNodeServiceable(node);
  if (
    serviceable &&
    (node.update?.state === 'rolled_back' ||
      (node.releases.desired_release_id &&
        node.releases.current_release_id &&
        node.releases.desired_release_id !== node.releases.current_release_id))
  ) {
    return 'rolled_back_serving';
  }
  if (serviceable) {
    return 'online_serving';
  }
  return 'ready_unpublished_not_serving';
}

export interface AdminOverviewViewModel {
  as_of: string;
  available_exits_count: number;
  total_nodes_count: number;
  control_online_count: number;
  control_stale_count: number;
  control_offline_count: number;
  safety_failed_nodes: number;
  enrollments_pending_or_failed: number;
  update_attention_nodes: number;
  active_members_count: number;
  throttled_members_count: number;
  suspended_or_expired_members_count: number;
}

export function buildAdminOverviewViewModel(
  nodes: NodeListResponse,
  members: MemberListResponse,
  latestBatch?: UpdateBatchDetail | null,
): AdminOverviewViewModel {
  const nodeItems = nodes.items;
  const memberItems = members.items;

  let latestSeen = '2026-10-28T15:00:00Z';
  for (const n of nodeItems) {
    if (n.last_seen_at && n.last_seen_at > latestSeen) {
      latestSeen = n.last_seen_at;
    }
  }

  const batchAttention = latestBatch
    ? latestBatch.counts.failed +
      latestBatch.counts.rolled_back +
      latestBatch.counts.waiting_online
    : nodeItems.filter(
        (n) =>
          Boolean(n.releases.desired_release_id) &&
          n.releases.current_release_id !== n.releases.desired_release_id,
      ).length;

  return {
    as_of: latestSeen,
    available_exits_count: nodeItems.filter((n) => isNodeServiceable(n)).length,
    total_nodes_count: nodeItems.length,
    control_online_count: nodeItems.filter((n) => n.control_status === 'online').length,
    control_stale_count: nodeItems.filter((n) => n.control_status === 'stale').length,
    control_offline_count: nodeItems.filter((n) => n.control_status === 'offline').length,
    safety_failed_nodes: nodeItems.filter(
      (n) => n.safety_status === 'failed' || n.safety_status === 'expired',
    ).length,
    enrollments_pending_or_failed: nodeItems.filter(
      (n) =>
        n.lifecycle_state === 'enrolling' ||
        n.provisioning_state === 'pending' ||
        n.provisioning_state === 'applying' ||
        n.provisioning_state === 'failed',
    ).length,
    update_attention_nodes: batchAttention,
    active_members_count: memberItems.filter(
      (m) =>
        m.entitlement.state === 'active' ||
        m.entitlement.state === 'cancel_at_period_end' ||
        m.entitlement.state === 'past_due',
    ).length,
    throttled_members_count: memberItems.filter(
      (m) => (m.usage.fair_use_tier_index ?? 0) > 0,
    ).length,
    suspended_or_expired_members_count: memberItems.filter(
      (m) => m.entitlement.state === 'suspended' || m.entitlement.state === 'expired',
    ).length,
  };
}

export interface FupTierDraftViewModel {
  tier_index: number;
  name_zh: string;
  name_en: string;
  threshold_bytes: string;
  down_bps: string;
  up_bps: string;
  priority: number;
}

export interface FupPolicyViewModel {
  definition_id: string;
  code: string;
  active_revision_id: string | null;
  revision: string;
  generation: string;
  resource_version: string;
  mode: PolicyMode;
  effective_at: string;
  minimum_rate_bps: string;
  tiers: FupTierDraftViewModel[];
  test_only_notice: boolean;
}

export type FairUseTierDraft = FupTierDraftViewModel;

export interface FairUsePolicyDraft {
  mode: PolicyMode;
  effective_at: string;
  minimum_rate_bps: string;
  tiers: FupTierDraftViewModel[];
}

export function buildFupPolicyViewModel(
  policies: PolicyDefinitionListResponse,
): FupPolicyViewModel | null {
  const def = policies.items.find((d) => d.type === 'fair_use');
  if (!def) {
    return null;
  }
  const rev = def.active_revision ?? def.latest_revision;
  if (!rev || rev.type !== 'fair_use') {
    return null;
  }
  const payload: FairUsePayload = rev.payload;
  return {
    definition_id: def.definition_id,
    code: def.code,
    active_revision_id: def.active_revision_id ?? null,
    revision: rev.revision,
    generation: def.generation,
    resource_version: def.resource_version,
    mode: payload.mode,
    effective_at: payload.effective_at,
    minimum_rate_bps: String(payload.minimum_rate.down_bps),
    tiers: payload.tiers.map((tier, idx) => ({
      tier_index: idx,
      name_zh: tier.display_name['zh-TW'],
      name_en: tier.display_name.en ?? tier.display_name['zh-TW'],
      threshold_bytes: tier.threshold_bytes,
      down_bps: String(tier.down_bps),
      up_bps: String(tier.up_bps),
      priority: tier.priority,
    })),
    test_only_notice: true,
  };
}

export function buildFairUseRevisionCreate(
  def: PolicyDefinition,
  input: {
    mode: PolicyMode;
    effective_at: string;
    minimum_rate_bps: string;
    tiers: FupTierDraftViewModel[];
    reason: string;
  },
): PolicyRevisionCreate {
  const baseRev = def.active_revision ?? def.latest_revision;
  const basePayload =
    baseRev && baseRev.type === 'fair_use' ? baseRev.payload : undefined;
  const minRateNumber = Number.parseInt(input.minimum_rate_bps, 10);

  return {
    definition_id: def.definition_id,
    type: 'fair_use',
    schema_version: 1,
    reason: input.reason,
    payload: {
      mode: input.mode,
      effective_at: input.effective_at,
      effective_period: basePayload?.effective_period ?? 'next_full_period',
      accounting_basis: basePayload?.accounting_basis ?? {
        period_type: 'utc_calendar_month',
        metering_version: 'forward_admission_v1',
        measure: 'total_bytes',
      },
      tiers: input.tiers.map((t) => ({
        threshold_bytes: t.threshold_bytes,
        down_bps: Number.parseInt(t.down_bps, 10),
        up_bps: Number.parseInt(t.up_bps, 10),
        priority: Math.min(7, Math.max(0, t.priority)),
        display_name: {
          'zh-TW': t.name_zh,
          en: t.name_en,
        } satisfies LocalizedText,
      })),
      minimum_rate: {
        down_bps: minRateNumber,
        up_bps: minRateNumber,
      },
      overrides: basePayload?.overrides ?? [],
    },
  };
}

export interface DnsThreatFeedViewModel {
  feed_id: string;
  name: string;
  category: string;
  license_summary: string;
  enabled: boolean;
  last_updated_at: string;
  domains_sample: string[];
}

export interface DnsResolverStatusViewModel {
  resolver_id: string;
  name: string;
  applied_revision: string;
  last_good_revision: string;
  status: 'healthy' | 'degraded' | 'failed';
  last_smoke_test_at: string;
}

export interface DnsPolicyViewModel {
  rule_set_definition_id: string;
  rule_set_code: string;
  rule_set_resource_version: string;
  active_revision_id: string | null;
  previous_revision_id: string | null;
  active_revision: string;
  last_good_revision: string;
  desired_revision: string;
  rules: DomainRule[];
  threat_feeds: DnsThreatFeedViewModel[];
  resolvers: DnsResolverStatusViewModel[];
}

export function buildDnsPolicyViewModel(
  policies: PolicyDefinitionListResponse,
): DnsPolicyViewModel | null {
  const ruleSetDef = policies.items.find((d) => d.type === 'domain_rule_set');
  if (!ruleSetDef) {
    return null;
  }
  const activeRuleRev = ruleSetDef.active_revision ?? ruleSetDef.latest_revision;
  const rules: DomainRule[] =
    activeRuleRev && activeRuleRev.type === 'domain_rule_set'
      ? (activeRuleRev.payload as DomainRuleSetPayload).rules
      : [];

  const feedDefs = policies.items.filter((d) => d.type === 'threat_feed');
  const threatFeeds: DnsThreatFeedViewModel[] = feedDefs.flatMap((fd) => {
    const rev = fd.active_revision ?? fd.latest_revision;
    if (!rev || rev.type !== 'threat_feed') return [];
    const payload = rev.payload as ThreatFeedPayload;
    return [
      {
        feed_id: fd.definition_id,
        name: `${payload.provider} (${payload.feed_code})`,
        category: payload.categories.join(', '),
        license_summary: `${payload.license_name} — ${payload.commercial_use_basis}`,
        enabled: Boolean(fd.active_revision_id),
        last_updated_at: fd.updated_at,
        domains_sample: ['phish-login.example.net', 'malware.bad.example.org'],
      },
    ];
  });

  const dnsProfileDef = policies.items.find((d) => d.type === 'dns_profile');
  const dnsProfileRev = dnsProfileDef?.active_revision ?? dnsProfileDef?.latest_revision;
  const appliedRevStr = activeRuleRev?.revision ?? '1';
  const prevSummary =
    ruleSetDef.revisions.find((r) => r.revision_id !== ruleSetDef.active_revision_id) ??
    ruleSetDef.revisions[0];
  const lastGoodRevStr = prevSummary?.revision ?? appliedRevStr;

  const profileCode =
    dnsProfileRev && dnsProfileRev.type === 'dns_profile'
      ? (dnsProfileRev.payload as DnsProfilePayload).profile_code
      : 'standard';

  return {
    rule_set_definition_id: ruleSetDef.definition_id,
    rule_set_code: ruleSetDef.code,
    rule_set_resource_version: ruleSetDef.resource_version,
    active_revision_id: ruleSetDef.active_revision_id ?? null,
    previous_revision_id: prevSummary?.revision_id ?? null,
    active_revision: appliedRevStr,
    last_good_revision: lastGoodRevStr,
    desired_revision: ruleSetDef.latest_revision?.revision ?? appliedRevStr,
    rules,
    threat_feeds: threatFeeds,
    resolvers: [
      {
        resolver_id: `resolver-primary-${profileCode}`,
        name: 'Controlled Resolver #1 (Primary)',
        applied_revision: appliedRevStr,
        last_good_revision: lastGoodRevStr,
        status: 'healthy',
        last_smoke_test_at: ruleSetDef.updated_at,
      },
      {
        resolver_id: `resolver-secondary-${profileCode}`,
        name: 'Controlled Resolver #2 (Secondary)',
        applied_revision: appliedRevStr,
        last_good_revision: lastGoodRevStr,
        status: 'healthy',
        last_smoke_test_at: ruleSetDef.updated_at,
      },
    ],
  };
}

export function extractMinimumServingVersionLabel(
  policies: PolicyDefinitionListResponse | undefined,
  fleetServingIds: string[] = [],
): string {
  const relSup = policies?.items.find((d) => d.type === 'release_support');
  const rev = relSup?.active_revision ?? relSup?.latest_revision;
  if (rev && rev.type === 'release_support') {
    const payload = rev.payload as ReleaseSupportPayload | undefined;
    const ids = payload?.minimum_serving_version?.serving_release_ids ?? fleetServingIds;
    return `${ids.length} release(s)`;
  }
  return `${fleetServingIds.length} release(s)`;
}

export function deriveMemberServiceReason(
  self: MemberSelf,
  locale: 'zh-TW' | 'en',
): string {
  const { entitlement, subscription } = self;
  if (entitlement.admin_suspended || entitlement.state === 'suspended') {
    return locale === 'zh-TW'
      ? '管理員已人工暫停您的服務資格（優先於付款狀態，§9.2）'
      : 'Service entitlement suspended by administrator (overrides billing status, §9.2)';
  }
  if (entitlement.state === 'pending_payment') {
    return locale === 'zh-TW'
      ? '等待支付供應商簽章 Webhook 與後端對帳確認'
      : 'Waiting for signed payment provider webhook and backend reconciliation';
  }
  if (entitlement.state === 'cancel_at_period_end') {
    return locale === 'zh-TW'
      ? `已設定期末取消，服務可繼續使用至 ${entitlement.effective_service_end ?? '-'}`
      : `Scheduled to cancel at period end; active until ${entitlement.effective_service_end ?? '-'}`;
  }
  if (entitlement.state === 'past_due') {
    return locale === 'zh-TW'
      ? `續扣待處理，寬限期至 ${entitlement.grace_until ?? '-'}`
      : `Payment past due; grace period active until ${entitlement.grace_until ?? '-'}`;
  }
  if (entitlement.state === 'expired') {
    return locale === 'zh-TW'
      ? '服務期限與寬限期已屆滿，出口轉送權限已停止'
      : 'Service period and grace window have expired; exit access is disabled';
  }
  return locale === 'zh-TW'
    ? `付款狀態正常 (${subscription?.provider_status ?? 'active'})`
    : `Subscription active (${subscription?.provider_status ?? 'active'})`;
}

export function deriveUsageQualitySummary(
  usage: MemberUsageResponse,
  locale: 'zh-TW' | 'en',
): { badge: 'ok' | 'gap' | 'estimated' | 'reconciling'; note: string | null } {
  if (usage.quality.gap) {
    return {
      badge: 'gap',
      note:
        locale === 'zh-TW'
          ? '節點於部分統計區間曾發生重啟或序列缺口，缺口期間不以虛構估算填補（§10.7）。'
          : 'A node reported a sequence gap or restart during this period; missing intervals are not fabricated (§10.7).',
    };
  }
  if (usage.quality.reconciliation_in_progress) {
    return {
      badge: 'reconciling',
      note:
        locale === 'zh-TW'
          ? '跨節點帳本對帳進行中，彙總數字可能於下一輪批次微調。'
          : 'Cross-node ledger reconciliation is in progress.',
    };
  }
  if (usage.quality.estimated) {
    return {
      badge: 'estimated',
      note:
        locale === 'zh-TW'
          ? '含跨 UTC 日期邊界之事件比例分配估算 (§10.5)。'
          : 'Includes UTC day-boundary pro-rata interval allocation (§10.5).',
    };
  }
  return { badge: 'ok', note: null };
}

export type UpdateTargetWithResult = UpdateTarget & {
  health?: UpdateHealth | null;
  candidate_health?: UpdateHealth | null;
  previous_release_id?: string | null;
  last_result?: Partial<UpdateResult> | null;
};

export interface UpdateTargetRollbackHealthViewModel {
  restoredReleaseId: string | null;
  previousReleaseId: string | null;
  health: UpdateHealth | null;
  candidateHealth: UpdateHealth | null;
}

export function extractTargetRollbackHealth(
  target: UpdateTargetWithResult,
  batchReleaseId?: string | null,
): UpdateTargetRollbackHealthViewModel {
  const health = target.health ?? target.last_result?.health ?? null;
  const candidateHealth =
    target.candidate_health !== undefined
      ? target.candidate_health
      : (target.last_result?.candidate_health ?? null);
  const previousReleaseId =
    target.previous_release_id ??
    target.last_result?.previous_release_id ??
    batchReleaseId ??
    null;
  const restoredReleaseId =
    target.current_release_id ?? target.last_result?.current_release_id ?? null;
  return {
    restoredReleaseId,
    previousReleaseId,
    health,
    candidateHealth,
  };
}
