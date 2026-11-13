import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { AppRoutes } from '../../App';
import {
  deriveNodeSummaryPhrase,
  isNodeServiceable,
  type ControlPlaneStatus,
  type DataHealth,
  type LifecycleState,
  type NodeSummaryItem,
  type NodeUpdateSummary,
  type ProvisioningState,
  type PublicationIntent,
  type PublicationObserved,
  type SafetyStatus,
} from '../../api';
import { SessionProvider } from '../../context/SessionContext';
import { en, I18nProvider, zhTW, type TranslationKey } from '../../i18n';
import {
  CONTRACT_EXAMPLES_USED_BY_MOCKS,
  FALLBACK_RELEASE_ID,
  OFFLINE_NODE_ID,
  PRIMARY_GROUP_ID,
  PRIMARY_NODE_ID,
  PRIMARY_RELEASE_ID,
  SAFETY_FAILED_NODE_ID,
} from '../../mocks/data';
import {
  resetMockState,
  setMockNodeOverride,
  setMockPersona,
  type MockPersona,
} from '../../mocks/handlers';
import { assertAccessible } from '../../test/setup';

const STALE_PUBLISHED_NODE_ID = '11111111-26c2-4ec4-bf5b-0c70df89b001';
const OFFLINE_ACTIVE_NODE_ID = '22222222-26c2-4ec4-bf5b-0c70df89b002';
const DEGRADED_PUBLISHED_NODE_ID = '33333333-26c2-4ec4-bf5b-0c70df89b003';

function renderAt(
  initialPath: string,
  persona: MockPersona = 'admin',
  initialLocale: 'zh-TW' | 'en' = 'zh-TW',
): ReturnType<typeof render> {
  cleanup();
  setMockPersona(persona);
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        refetchOnWindowFocus: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <I18nProvider initialLocale={initialLocale}>
        <SessionProvider>
          <MemoryRouter initialEntries={[initialPath]}>
            <AppRoutes />
          </MemoryRouter>
        </SessionProvider>
      </I18nProvider>
    </QueryClientProvider>,
  );
}

const SAMPLE_UPDATE_TEMPLATE: NodeUpdateSummary = structuredClone(
  CONTRACT_EXAMPLES_USED_BY_MOCKS.NodeDetail.update!,
);

function makeServiceableNodeSummary(
  overrides: Partial<NodeSummaryItem> = {},
): NodeSummaryItem {
  return {
    node_id: PRIMARY_NODE_ID,
    exit_id: 'caba5aec-4b2b-432c-9512-211a7b3e96db',
    name: 'Tokyo 1',
    region: 'jp-tokyo',
    member_group_ids: [PRIMARY_GROUP_ID],
    lifecycle_state: 'registered',
    provisioning_state: 'ready',
    publication_intent: 'published',
    publication_observed: 'active',
    control_status: 'online',
    data_health: 'healthy',
    safety_status: 'passed',
    member_forwarding_observed: true,
    config_generation: {
      desired: '128',
      applied: '128',
    },
    releases: {
      current_release_id: PRIMARY_RELEASE_ID,
      desired_release_id: PRIMARY_RELEASE_ID,
      last_good_release_id: FALLBACK_RELEASE_ID,
    },
    update: null,
    last_seen_at: '2026-10-28T14:06:30Z',
    resource_version: '42',
    created_at: '2026-10-01T09:00:00Z',
    ...overrides,
  };
}

function makeUpdateWithState(state: NodeUpdateSummary['state']): NodeUpdateSummary {
  return {
    ...SAMPLE_UPDATE_TEMPLATE,
    state,
  };
}

describe('Admin Node Summary & Overview (§5, §5.1, §6.1, §18.1, §18.3)', () => {
  it('Test1_NodeSummaryRules_TableDrivenAll13RulesAndEdgeCases', () => {
    const cases: Array<{
      name: string;
      rule: number;
      node: NodeSummaryItem;
      expectedKey: string;
      expectedZh: string;
      expectedEn: string;
    }> = [
      {
        name: 'rule 1a: lifecycle revoked',
        rule: 1,
        node: makeServiceableNodeSummary({ lifecycle_state: 'revoked' }),
        expectedKey: 'revoked_not_serving',
        expectedZh: '裝置身分已永久撤銷，停止服務',
        expectedEn: 'Machine identity permanently revoked, not serving',
      },
      {
        name: 'rule 1b: lifecycle retired',
        rule: 1,
        node: makeServiceableNodeSummary({ lifecycle_state: 'retired' }),
        expectedKey: 'revoked_not_serving',
        expectedZh: '裝置身分已永久撤銷，停止服務',
        expectedEn: 'Machine identity permanently revoked, not serving',
      },
      {
        name: 'rule 2: lifecycle enrolling',
        rule: 2,
        node: makeServiceableNodeSummary({
          lifecycle_state: 'enrolling',
          provisioning_state: 'pending',
          publication_intent: 'unpublished',
          publication_observed: 'absent',
          member_forwarding_observed: false,
        }),
        expectedKey: 'enrolling_not_serving',
        expectedZh: '註冊中，未提供服務',
        expectedEn: 'Enrolling, not serving traffic',
      },
      {
        name: 'rule 3: control offline and observed withdrawn',
        rule: 3,
        node: makeServiceableNodeSummary({
          publication_intent: 'disabled',
          publication_observed: 'withdrawn',
          control_status: 'offline',
          data_health: 'unknown',
          safety_status: 'unknown',
          member_forwarding_observed: false,
        }),
        expectedKey: 'offline_route_withdrawn_unconfirmed',
        expectedZh: '中央已撤銷路由；設備撤權尚未確認／受租約限制',
        expectedEn:
          'Central routes withdrawn; device revocation unconfirmed / bounded by lease',
      },
      {
        name: 'rule 4a: published, active node that would be serviceable but is offline with observed active',
        rule: 4,
        node: makeServiceableNodeSummary({
          control_status: 'offline',
          publication_observed: 'active',
        }),
        expectedKey: 'offline_unconfirmed',
        expectedZh: '離線，目前服務狀態未能確認',
        expectedEn: 'Offline, current serving state unconfirmed',
      },
      {
        name: 'rule 4b: offline node with safety failed',
        rule: 4,
        node: makeServiceableNodeSummary({
          control_status: 'offline',
          publication_observed: 'active',
          safety_status: 'failed',
          member_forwarding_observed: false,
        }),
        expectedKey: 'offline_unconfirmed',
        expectedZh: '離線，目前服務狀態未能確認',
        expectedEn: 'Offline, current serving state unconfirmed',
      },
      {
        name: 'rule 4c: offline node with intent disabled and observed applying',
        rule: 4,
        node: makeServiceableNodeSummary({
          control_status: 'offline',
          publication_intent: 'disabled',
          publication_observed: 'applying',
          member_forwarding_observed: false,
        }),
        expectedKey: 'offline_unconfirmed',
        expectedZh: '離線，目前服務狀態未能確認',
        expectedEn: 'Offline, current serving state unconfirmed',
      },
      {
        name: 'rule 5a: published, active node that would be serviceable but is stale',
        rule: 5,
        node: makeServiceableNodeSummary({
          control_status: 'stale',
        }),
        expectedKey: 'stale_unconfirmed',
        expectedZh: '回報延遲，目前服務狀態未能確認',
        expectedEn: 'Reports late, current serving state unconfirmed',
      },
      {
        name: 'rule 5b: stale node with observed withdrawn',
        rule: 5,
        node: makeServiceableNodeSummary({
          control_status: 'stale',
          publication_observed: 'withdrawn',
          member_forwarding_observed: false,
        }),
        expectedKey: 'stale_unconfirmed',
        expectedZh: '回報延遲，目前服務狀態未能確認',
        expectedEn: 'Reports late, current serving state unconfirmed',
      },
      {
        name: 'rule 6: online node with safety failed',
        rule: 6,
        node: makeServiceableNodeSummary({
          control_status: 'online',
          safety_status: 'failed',
          publication_observed: 'withdrawn',
          member_forwarding_observed: false,
        }),
        expectedKey: 'online_safety_failed_not_serving',
        expectedZh: '在線，隔離檢查失敗，未提供服務',
        expectedEn: 'Online, isolation check failed, not serving traffic',
      },
      {
        name: 'rule 7: online node with safety expired',
        rule: 7,
        node: makeServiceableNodeSummary({
          control_status: 'online',
          safety_status: 'expired',
          member_forwarding_observed: false,
        }),
        expectedKey: 'online_safety_expired_not_serviceable',
        expectedZh: '在線，隔離檢查已過期，服務條件未符合',
        expectedEn: 'Online, isolation check expired, serving conditions not met',
      },
      {
        name: 'rule 8: online node with intent disabled',
        rule: 8,
        node: makeServiceableNodeSummary({
          control_status: 'online',
          publication_intent: 'disabled',
          publication_observed: 'withdrawn',
          member_forwarding_observed: false,
        }),
        expectedKey: 'online_disabled_not_serving',
        expectedZh: '在線，管理員已停用出口，未提供服務',
        expectedEn: 'Online, disabled by admin, not serving traffic',
      },
      {
        name: 'rule 9: online node with intent unpublished and provisioning ready',
        rule: 9,
        node: makeServiceableNodeSummary({
          control_status: 'online',
          provisioning_state: 'ready',
          publication_intent: 'unpublished',
          publication_observed: 'absent',
          member_forwarding_observed: false,
        }),
        expectedKey: 'ready_unpublished_not_serving',
        expectedZh: '已就緒，尚未上架，未提供服務',
        expectedEn: 'Provisioned ready, unpublished, not serving traffic',
      },
      {
        name: 'rule 10: online node with intent unpublished and provisioning not ready',
        rule: 10,
        node: makeServiceableNodeSummary({
          control_status: 'online',
          provisioning_state: 'applying',
          publication_intent: 'unpublished',
          publication_observed: 'absent',
          member_forwarding_observed: false,
        }),
        expectedKey: 'online_unpublished_not_ready_not_serving',
        expectedZh: '在線，本機配置未就緒，尚未上架，未提供服務',
        expectedEn: 'Online, provisioning not ready, unpublished, not serving traffic',
      },
      {
        name: 'rule 11: serviceable node with different release ids and update rolled_back',
        rule: 11,
        node: makeServiceableNodeSummary({
          releases: {
            current_release_id: FALLBACK_RELEASE_ID,
            desired_release_id: PRIMARY_RELEASE_ID,
            last_good_release_id: FALLBACK_RELEASE_ID,
          },
          update: makeUpdateWithState('rolled_back'),
        }),
        expectedKey: 'rolled_back_serving',
        expectedZh: '在線，上次更新已回復至最後良好版本 (last_good)',
        expectedEn: 'Online, update rolled back to verified last_good release',
      },
      {
        name: 'rule 12a: serviceable node with different release ids and update downloading',
        rule: 12,
        node: makeServiceableNodeSummary({
          releases: {
            current_release_id: FALLBACK_RELEASE_ID,
            desired_release_id: PRIMARY_RELEASE_ID,
            last_good_release_id: FALLBACK_RELEASE_ID,
          },
          update: makeUpdateWithState('downloading'),
        }),
        expectedKey: 'online_serving',
        expectedZh: '在線，正常提供出口服務',
        expectedEn: 'Online, actively serving exit traffic',
      },
      {
        name: 'rule 12b: serviceable node with equal release ids and no update',
        rule: 12,
        node: makeServiceableNodeSummary(),
        expectedKey: 'online_serving',
        expectedZh: '在線，正常提供出口服務',
        expectedEn: 'Online, actively serving exit traffic',
      },
      {
        name: 'rule 13: online published node that is not serviceable (data degraded)',
        rule: 13,
        node: makeServiceableNodeSummary({
          data_health: 'degraded',
        }),
        expectedKey: 'online_published_not_serviceable',
        expectedZh: '在線，已上架，服務條件未全部符合',
        expectedEn: 'Online, published, not all serving conditions met',
      },
    ];

    const coveredRules = new Set(cases.map((c) => c.rule));
    expect([...coveredRules].sort((a, b) => a - b)).toEqual([
      1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13,
    ]);

    const actual = cases.map((c) => {
      const key = deriveNodeSummaryPhrase(c.node);
      const dictKey = `node.summary.${key}` as TranslationKey;
      return {
        name: c.name,
        key,
        zh: zhTW[dictKey],
        en: en[dictKey],
      };
    });

    const expected = cases.map((c) => ({
      name: c.name,
      key: c.expectedKey,
      zh: c.expectedZh,
      en: c.expectedEn,
    }));

    expect(actual).toEqual(expected);
  });

  it('Test2_NodeSummaryClaims_All207360States', () => {
    const lifecycles: LifecycleState[] = [
      'enrolling',
      'registered',
      'revoked',
      'retired',
    ];
    const provisionings: ProvisioningState[] = ['pending', 'applying', 'ready', 'failed'];
    const intents: PublicationIntent[] = ['unpublished', 'published', 'disabled'];
    const observedStates: PublicationObserved[] = [
      'absent',
      'applying',
      'active',
      'withdrawn',
      'error',
    ];
    const controls: ControlPlaneStatus[] = ['online', 'stale', 'offline'];
    const dataHealths: DataHealth[] = ['unknown', 'healthy', 'degraded', 'failed'];
    const safeties: SafetyStatus[] = ['unknown', 'passed', 'failed', 'expired'];
    const forwardings: Array<boolean | null> = [true, false, null];
    const updates: Array<NodeUpdateSummary | null> = [
      null,
      makeUpdateWithState('rolled_back'),
      makeUpdateWithState('downloading'),
    ];
    const releaseVariants: Array<{
      current_release_id: string;
      desired_release_id: string;
      last_good_release_id: string;
    }> = [
      {
        current_release_id: PRIMARY_RELEASE_ID,
        desired_release_id: PRIMARY_RELEASE_ID,
        last_good_release_id: FALLBACK_RELEASE_ID,
      },
      {
        current_release_id: FALLBACK_RELEASE_ID,
        desired_release_id: PRIMARY_RELEASE_ID,
        last_good_release_id: FALLBACK_RELEASE_ID,
      },
    ];

    const expectedKeys = [
      'revoked_not_serving',
      'enrolling_not_serving',
      'offline_route_withdrawn_unconfirmed',
      'offline_unconfirmed',
      'stale_unconfirmed',
      'online_safety_failed_not_serving',
      'online_safety_expired_not_serviceable',
      'online_disabled_not_serving',
      'ready_unpublished_not_serving',
      'online_unpublished_not_ready_not_serving',
      'rolled_back_serving',
      'online_serving',
      'online_published_not_serviceable',
    ] as const;

    const counts: Record<string, number> = Object.fromEntries(
      expectedKeys.map((k) => [k, 0]),
    );

    const node = makeServiceableNodeSummary();
    let total = 0;

    for (const lifecycle_state of lifecycles) {
      node.lifecycle_state = lifecycle_state;
      for (const provisioning_state of provisionings) {
        node.provisioning_state = provisioning_state;
        for (const publication_intent of intents) {
          node.publication_intent = publication_intent;
          for (const publication_observed of observedStates) {
            node.publication_observed = publication_observed;
            for (const control_status of controls) {
              node.control_status = control_status;
              for (const data_health of dataHealths) {
                node.data_health = data_health;
                for (const safety_status of safeties) {
                  node.safety_status = safety_status;
                  for (const member_forwarding_observed of forwardings) {
                    node.member_forwarding_observed = member_forwarding_observed;
                    for (const update of updates) {
                      node.update = update;
                      for (const releases of releaseVariants) {
                        node.releases = releases;
                        total += 1;

                        const key = deriveNodeSummaryPhrase(node) as string;
                        if (!(key in counts)) {
                          expect.fail(`Unexpected phrase key: ${key}`);
                        }
                        counts[key] = (counts[key] ?? 0) + 1;

                        const serviceable = isNodeServiceable(node);
                        if (serviceable && node.control_status !== 'online') {
                          expect.fail(
                            `Serviceable node must have control_status === 'online', got ${node.control_status}`,
                          );
                        }

                        let claimHolds = false;
                        switch (key) {
                          case 'revoked_not_serving':
                            claimHolds =
                              node.lifecycle_state === 'revoked' ||
                              node.lifecycle_state === 'retired';
                            break;
                          case 'enrolling_not_serving':
                            claimHolds =
                              node.lifecycle_state === 'enrolling' && !serviceable;
                            break;
                          case 'offline_route_withdrawn_unconfirmed':
                            claimHolds =
                              node.control_status === 'offline' &&
                              node.publication_observed === 'withdrawn';
                            break;
                          case 'offline_unconfirmed':
                            claimHolds = node.control_status === 'offline';
                            break;
                          case 'stale_unconfirmed':
                            claimHolds = node.control_status === 'stale';
                            break;
                          case 'online_safety_failed_not_serving':
                            claimHolds =
                              node.control_status === 'online' &&
                              node.safety_status === 'failed' &&
                              !serviceable;
                            break;
                          case 'online_safety_expired_not_serviceable':
                            claimHolds =
                              node.control_status === 'online' &&
                              node.safety_status === 'expired' &&
                              !serviceable;
                            break;
                          case 'online_disabled_not_serving':
                            claimHolds =
                              node.control_status === 'online' &&
                              node.publication_intent === 'disabled' &&
                              !serviceable;
                            break;
                          case 'ready_unpublished_not_serving':
                            claimHolds =
                              node.provisioning_state === 'ready' &&
                              node.publication_intent === 'unpublished' &&
                              !serviceable;
                            break;
                          case 'online_unpublished_not_ready_not_serving':
                            claimHolds =
                              node.control_status === 'online' &&
                              node.provisioning_state !== 'ready' &&
                              node.publication_intent === 'unpublished' &&
                              !serviceable;
                            break;
                          case 'rolled_back_serving':
                            claimHolds =
                              serviceable && node.update?.state === 'rolled_back';
                            break;
                          case 'online_serving':
                            claimHolds = serviceable;
                            break;
                          case 'online_published_not_serviceable':
                            claimHolds =
                              node.control_status === 'online' &&
                              node.publication_intent === 'published' &&
                              !serviceable;
                            break;
                        }

                        if (!claimHolds) {
                          expect.fail(
                            `Claim violated for key "${key}" on node state: ${JSON.stringify(
                              {
                                lifecycle_state: node.lifecycle_state,
                                provisioning_state: node.provisioning_state,
                                publication_intent: node.publication_intent,
                                publication_observed: node.publication_observed,
                                control_status: node.control_status,
                                data_health: node.data_health,
                                safety_status: node.safety_status,
                                member_forwarding_observed:
                                  node.member_forwarding_observed,
                                update_state: node.update?.state ?? null,
                                releases_equal:
                                  node.releases.current_release_id ===
                                  node.releases.desired_release_id,
                                serviceable,
                              },
                            )}`,
                          );
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }

    expect(total).toBe(207_360);
    for (const key of expectedKeys) {
      expect(counts[key], `Expected key ${key} to occur at least once`).toBeGreaterThan(
        0,
      );
    }
    expect(counts).toEqual({
      revoked_not_serving: 103_680,
      enrolling_not_serving: 51_840,
      offline_route_withdrawn_unconfirmed: 3_456,
      offline_unconfirmed: 13_824,
      stale_unconfirmed: 17_280,
      online_safety_failed_not_serving: 4_320,
      online_safety_expired_not_serviceable: 4_320,
      online_disabled_not_serving: 2_880,
      ready_unpublished_not_serving: 720,
      online_unpublished_not_ready_not_serving: 2_160,
      rolled_back_serving: 2,
      online_serving: 4,
      online_published_not_serviceable: 2_874,
    });
  });

  it('Test3_NodesListPage_SummaryTextsAndBadgeClassesBothLocales', async () => {
    const user = userEvent.setup();

    setMockNodeOverride(STALE_PUBLISHED_NODE_ID, {
      name: 'Tokyo Stale',
      control_status: 'stale',
      publication_intent: 'published',
      publication_observed: 'active',
    });
    setMockNodeOverride(OFFLINE_ACTIVE_NODE_ID, {
      name: 'Tokyo Offline Active',
      control_status: 'offline',
      publication_intent: 'published',
      publication_observed: 'active',
    });
    setMockNodeOverride(DEGRADED_PUBLISHED_NODE_ID, {
      name: 'Tokyo Degraded',
      control_status: 'online',
      publication_intent: 'published',
      publication_observed: 'active',
      data_health: 'degraded',
    });

    const { container } = renderAt('/admin/nodes', 'admin', 'zh-TW');
    expect(await screen.findByTestId('nodes-table')).toBeInTheDocument();
    expect(
      await screen.findByTestId(`node-summary-${PRIMARY_NODE_ID}`),
    ).toBeInTheDocument();

    const expectedRows: Array<{
      nodeId: string;
      badgeClass: 'badge-success' | 'badge-warning' | 'badge-danger';
      zh: string;
      en: string;
    }> = [
      {
        nodeId: PRIMARY_NODE_ID,
        badgeClass: 'badge-success',
        zh: '在線，正常提供出口服務',
        en: 'Online, actively serving exit traffic',
      },
      {
        nodeId: SAFETY_FAILED_NODE_ID,
        badgeClass: 'badge-danger',
        zh: '在線，隔離檢查失敗，未提供服務',
        en: 'Online, isolation check failed, not serving traffic',
      },
      {
        nodeId: OFFLINE_NODE_ID,
        badgeClass: 'badge-danger',
        zh: '中央已撤銷路由；設備撤權尚未確認／受租約限制',
        en: 'Central routes withdrawn; device revocation unconfirmed / bounded by lease',
      },
      {
        nodeId: STALE_PUBLISHED_NODE_ID,
        badgeClass: 'badge-warning',
        zh: '回報延遲，目前服務狀態未能確認',
        en: 'Reports late, current serving state unconfirmed',
      },
      {
        nodeId: OFFLINE_ACTIVE_NODE_ID,
        badgeClass: 'badge-danger',
        zh: '離線，目前服務狀態未能確認',
        en: 'Offline, current serving state unconfirmed',
      },
      {
        nodeId: DEGRADED_PUBLISHED_NODE_ID,
        badgeClass: 'badge-danger',
        zh: '在線，已上架，服務條件未全部符合',
        en: 'Online, published, not all serving conditions met',
      },
    ];

    for (const row of expectedRows) {
      const badge = screen.getByTestId(`node-summary-${row.nodeId}`);
      expect(badge.textContent).toBe(row.zh);
      expect(badge).toHaveClass('badge', row.badgeClass);
      for (const other of ['badge-success', 'badge-warning', 'badge-danger'] as const) {
        if (other !== row.badgeClass) {
          expect(badge).not.toHaveClass(other);
        }
      }
    }

    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');

    for (const row of expectedRows) {
      const badge = screen.getByTestId(`node-summary-${row.nodeId}`);
      expect(badge.textContent).toBe(row.en);
      expect(badge).toHaveClass('badge', row.badgeClass);
    }

    await assertAccessible(container);

    // Verify resetMockState removes the override nodes and restores default mock state
    resetMockState('admin');
    renderAt('/admin/nodes', 'admin', 'zh-TW');
    expect(
      await screen.findByTestId(`node-summary-${PRIMARY_NODE_ID}`),
    ).toHaveTextContent('在線，正常提供出口服務');
    expect(screen.queryByTestId(`node-summary-${STALE_PUBLISHED_NODE_ID}`)).toBeNull();
    expect(screen.queryByTestId(`node-summary-${OFFLINE_ACTIVE_NODE_ID}`)).toBeNull();
    expect(screen.queryByTestId(`node-summary-${DEGRADED_PUBLISHED_NODE_ID}`)).toBeNull();
  });

  it('Test4_NodeDetailPage_StaleNodeSummaryBadgeBothLocales', async () => {
    const user = userEvent.setup();

    setMockNodeOverride(PRIMARY_NODE_ID, {
      control_status: 'stale',
    });

    const { container } = renderAt(`/admin/nodes/${PRIMARY_NODE_ID}`, 'admin', 'zh-TW');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();

    const badge = screen.getByTestId('node-detail-summary-badge');
    expect(badge.textContent).toBe('回報延遲，目前服務狀態未能確認');
    expect(badge).toHaveClass('badge', 'badge-info');

    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');
    expect(badge.textContent).toBe('Reports late, current serving state unconfirmed');
    expect(badge).toHaveClass('badge', 'badge-info');

    await assertAccessible(container);
  });

  it('Test5_AdminOverviewPage_AsOfAndSafetyCardBothLocales', async () => {
    const user = userEvent.setup();

    // 1. Default mocks: all nodes have last_seen_at = 2026-10-28T14:06:30Z
    const { container } = renderAt('/admin', 'admin', 'zh-TW');
    const overview = await screen.findByTestId('admin-overview-page');

    const asOfBadge =
      screen.queryByTestId('admin-overview-as-of') ??
      overview.querySelector('.page-header .badge');
    expect(asOfBadge).not.toBeNull();
    expect(asOfBadge!.textContent).toBe('最近節點回報 (UTC)：2026-10-28T14:06:30Z');
    expect(asOfBadge!.textContent).not.toContain('15:00:00Z');
    expect(screen.getByTestId('admin-overview-as-of')).toBe(asOfBadge);

    const safetyCard = screen.getByTestId('card-safety-failed');
    expect(within(safetyCard).getByText('1')).toBeVisible();
    const safetyDesc =
      screen.queryByTestId('card-safety-failed-desc') ??
      safetyCard.querySelector('.stat-subtext');
    expect(safetyDesc).not.toBeNull();
    expect(safetyDesc!.textContent).toBe('隔離檢查結果為失敗或已過期的節點');
    expect(screen.getByTestId('card-safety-failed-desc')).toBe(safetyDesc);

    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');
    expect(asOfBadge!.textContent).toBe('Latest node report (UTC): 2026-10-28T14:06:30Z');
    expect(asOfBadge!.textContent).not.toContain('15:00:00Z');
    expect(safetyDesc!.textContent).toBe('Nodes whose isolation check failed or expired');

    await assertAccessible(container);

    // 2. When no node has reported (every last_seen_at null)
    setMockNodeOverride(PRIMARY_NODE_ID, { last_seen_at: null });
    setMockNodeOverride(SAFETY_FAILED_NODE_ID, { last_seen_at: null });
    setMockNodeOverride(OFFLINE_NODE_ID, { last_seen_at: null });

    renderAt('/admin', 'admin', 'zh-TW');
    const overviewNone = await screen.findByTestId('admin-overview-page');
    const asOfNoneBadge =
      screen.queryByTestId('admin-overview-as-of') ??
      overviewNone.querySelector('.page-header .badge');
    expect(asOfNoneBadge!.textContent).toBe('尚無節點回報');

    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');
    expect(asOfNoneBadge!.textContent).toBe('No node report yet');
  });
});
