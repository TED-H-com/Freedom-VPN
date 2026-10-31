import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '../../mocks/server';
import { AppRoutes } from '../../App';
import { apiClient } from '../../api';
import { SessionProvider } from '../../context/SessionContext';
import { I18nProvider } from '../../i18n';
import {
  CONTRACT_EXAMPLES_USED_BY_MOCKS,
  OFFLINE_NODE_ID,
  PRIMARY_DEVICE_ID,
  PRIMARY_ENROLLMENT_ID,
  PRIMARY_FUP_DEFINITION_ID,
  PRIMARY_GROUP_ID,
  PRIMARY_MEMBER_ID,
  PRIMARY_NODE_ID,
  SAFETY_FAILED_NODE_ID,
  SECONDARY_GROUP_ID,
} from '../../mocks/data';
import {
  FENCED_PENDING_DEVICE_ID,
  FRESH_PENDING_DEVICE_ID,
  OVER_LIMIT_DEVICE_ID,
  REVOKED_DEVICE_ID,
  setMockDeviceStatesForRevision12,
  setMockNodeIdentityConflict,
  setMockNodeLeaseState,
  setMockPersona,
  setMockRolledBackCandidateHealth,
  type MockPersona,
} from '../../mocks/handlers';
import { assertAccessible } from '../../test/setup';

function renderAt(
  initialPath: string,
  persona: MockPersona = 'admin',
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
      <I18nProvider>
        <SessionProvider>
          <MemoryRouter initialEntries={[initialPath]}>
            <AppRoutes />
          </MemoryRouter>
        </SessionProvider>
      </I18nProvider>
    </QueryClientProvider>,
  );
}

describe('Portal Screens & Acceptance Tests (§18, §19 AT02/AT03/AT28/AT32)', () => {
  it('TestAdminOverview_RendersSeparateHealthDimensionsAndNeverCollapsesStatus', async () => {
    const { container } = renderAt('/admin', 'admin');

    expect(await screen.findByTestId('admin-overview-page')).toBeInTheDocument();
    expect(screen.getByTestId('card-available-exits')).toBeInTheDocument();
    expect(screen.getByTestId('card-control-status')).toBeInTheDocument();
    expect(screen.getByTestId('card-safety-failed')).toBeInTheDocument();
    expect(screen.getByTestId('card-update-attention')).toBeInTheDocument();
    expect(screen.getByTestId('card-members')).toBeInTheDocument();

    expect(within(screen.getByTestId('card-safety-failed')).getByText('1')).toBeVisible();

    await assertAccessible(container);
  });

  it('TestAT32_NodesList_DistinguishesOnlineServingFromOnlineSafetyFailedNotServing', async () => {
    const { container } = renderAt('/admin/nodes', 'admin');

    expect(await screen.findByTestId('nodes-table')).toBeInTheDocument();

    // Primary node: online, actively serving
    const primarySummary = await screen.findByTestId(`node-summary-${PRIMARY_NODE_ID}`);
    expect(primarySummary).toHaveTextContent('在線，正常提供出口服務');

    // Safety-failed node: online, safety check failed -> NOT serving
    const safetyFailedSummary = screen.getByTestId(
      `node-summary-${SAFETY_FAILED_NODE_ID}`,
    );
    expect(safetyFailedSummary).toHaveTextContent('在線，隔離檢查失敗，未提供服務');

    // Offline node: route withdrawn, device revocation unconfirmed
    const offlineSummary = screen.getByTestId(`node-summary-${OFFLINE_NODE_ID}`);
    expect(offlineSummary).toHaveTextContent(
      '中央已撤銷路由；設備撤權尚未確認／受租約限制',
    );

    await assertAccessible(container);
  });

  it('TestNodeDetail_DesiredVsObserved_SavedPendingApply_412ConflictAndRevokeFlow', async () => {
    const user = userEvent.setup();
    renderAt(`/admin/nodes/${PRIMARY_NODE_ID}`, 'admin');

    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    expect(screen.getByTestId('desired-vs-observed-section')).toBeInTheDocument();
    expect(screen.getByTestId('desired-config-gen')).toHaveTextContent('128');

    // 1. Save patch -> shows "已儲存，等待套用"
    await user.click(screen.getByTestId('save-node-patch-btn'));
    const savedBanner = await screen.findByTestId('saved-pending-apply-banner');
    expect(savedBanner).toHaveTextContent('已儲存，等待套用');

    // 2. Simulate concurrent edit 412 conflict -> shows ConflictDialog
    await user.click(screen.getByTestId('simulate-conflict-btn'));
    const conflictDialog = await screen.findByTestId('conflict-dialog');
    expect(conflictDialog).toBeInTheDocument();
    expect(conflictDialog).toHaveTextContent('412');

    await user.click(screen.getByTestId('conflict-reload-btn'));
    await waitFor(() => {
      expect(screen.queryByTestId('conflict-dialog')).not.toBeInTheDocument();
    });

    // 3. Publish node -> shows 202 OperationBanner (not immediately completed)
    await user.click(screen.getByTestId('publish-node-btn'));
    expect(await screen.findByTestId('operation-banner')).toBeInTheDocument();

    // 4. Permanently revoke machine identity (distinct from disable)
    await user.click(screen.getByTestId('revoke-node-identity-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('node-detail-summary-badge')).toHaveTextContent(
        '裝置身分已永久撤銷，停止服務',
      );
    });
  });

  it('TestEnrollment_OneTimeTokenDisplayOnlyOnceAndNeverPersistedToStorage', async () => {
    const user = userEvent.setup();
    const { container } = renderAt('/admin/nodes/new', 'admin');

    expect(await screen.findByTestId('admin-add-node-page')).toBeInTheDocument();

    await user.click(screen.getByTestId('create-enrollment-submit-btn'));

    const oneTimeBanner = await screen.findByTestId('one-time-enrollment-banner');
    expect(oneTimeBanner).toBeInTheDocument();
    const tokenValue = screen.getByTestId('one-time-token-value').textContent ?? '';
    expect(tokenValue).toContain('TESTONLYnotarealtoken_');

    // Verify token is NEVER written to localStorage, sessionStorage, or URL
    expect(JSON.stringify(window.localStorage)).not.toContain(tokenValue);
    expect(JSON.stringify(window.sessionStorage)).not.toContain(tokenValue);
    expect(window.location.href).not.toContain(tokenValue);

    await assertAccessible(container);

    // Navigate to enrollment status view -> plaintext token is gone forever
    await user.click(screen.getByTestId('navigate-enrollment-status-link'));
    expect(await screen.findByTestId('admin-enrollment-detail-page')).toBeInTheDocument();
    expect(screen.getByTestId('enrollment-secret-gone-notice')).toBeInTheDocument();
    expect(screen.getByTestId('secret-available-flag')).toHaveTextContent('false');
    expect(screen.queryByText(tokenValue)).not.toBeInTheDocument();

    // Revoke unused enrollment token
    await user.click(screen.getByTestId('revoke-enrollment-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('enrollment-state-badge')).toHaveTextContent('revoked');
    });
  });

  it('TestFleetUpdates_OneActionUpdateAllNodesAnd12BucketAccounting', async () => {
    const user = userEvent.setup();
    const { container } = renderAt('/admin/updates', 'admin');

    expect(await screen.findByTestId('admin-fleet-updates-page')).toBeInTheDocument();
    expect(screen.getByTestId('fleet-default-release')).toBeInTheDocument();
    expect(screen.getByTestId('min-serving-version')).toBeInTheDocument();

    // Verify bucket sum equals target_count (3 / 3)
    expect(screen.getByTestId('bucket-sum-check')).toHaveTextContent('3 / 3');

    // Verified success count only counts succeeded (1) + already_current (0) = 1, never rolled_back (1)
    expect(screen.getByTestId('verified-success-summary')).toHaveTextContent('1');
    expect(
      screen.getByTestId(`target-bucket-${SAFETY_FAILED_NODE_ID}`),
    ).toHaveTextContent('rolled_back');

    // D-048 target policy evidence: reported target vs not-reported target
    expect(screen.getByTestId(`target-policy-gen-${PRIMARY_NODE_ID}`)).toHaveTextContent(
      '128',
    );
    expect(screen.getByTestId(`target-policy-refs-${PRIMARY_NODE_ID}`)).toHaveTextContent(
      PRIMARY_FUP_DEFINITION_ID,
    );
    expect(screen.getByTestId(`target-policy-gen-${OFFLINE_NODE_ID}`)).toHaveTextContent(
      '尚未回報 (not yet reported)',
    );
    expect(screen.getByTestId(`target-policy-refs-${OFFLINE_NODE_ID}`)).toHaveTextContent(
      '尚未回報 (not yet reported)',
    );

    // Target detail panel defaults to first target (PRIMARY_NODE_ID) and switches on Inspect Target click
    expect(screen.getByTestId('batch-target-detail-panel')).toBeInTheDocument();
    expect(screen.getByTestId('target-detail-policy-gen')).toHaveTextContent('128');
    expect(screen.getByTestId('target-detail-policy-refs')).toHaveTextContent(
      PRIMARY_FUP_DEFINITION_ID,
    );

    await user.click(screen.getByTestId(`inspect-target-${OFFLINE_NODE_ID}`));
    expect(screen.getByTestId('target-detail-policy-gen')).toHaveTextContent(
      '尚未回報 (not yet reported)',
    );
    expect(screen.getByTestId('target-detail-policy-refs')).toHaveTextContent(
      '尚未回報 (not yet reported)',
    );

    // Run one-action Update All Nodes -> 202 accepted banner
    await user.click(screen.getByTestId('run-all-nodes-update-btn'));
    expect(await screen.findByTestId('update-batch-accepted-banner')).toBeInTheDocument();

    await assertAccessible(container);
  });

  it('TestAdminMembers_ListAndDetailSuspendResumeAndGrant', async () => {
    const user = userEvent.setup();
    renderAt('/admin/members', 'admin');

    expect(await screen.findByTestId('admin-members-list-page')).toBeInTheDocument();
    expect(
      await screen.findByTestId(`member-row-${PRIMARY_MEMBER_ID}`),
    ).toBeInTheDocument();

    renderAt(`/admin/members/${PRIMARY_MEMBER_ID}`, 'admin');
    expect(await screen.findByTestId('admin-member-detail-page')).toBeInTheDocument();

    // Suspend member
    await user.click(screen.getByTestId('suspend-member-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('member-detail-entitlement-badge')).toHaveTextContent(
        'suspended',
      );
    });

    // Create finite grant
    await user.click(screen.getByTestId('create-grant-btn'));
    expect(await screen.findByTestId('operation-banner')).toBeInTheDocument();
  });

  it('TestAdminFupPolicy_TestOnlyBannerAndClientValidation', async () => {
    const user = userEvent.setup();
    const { container } = renderAt('/admin/policies/fup', 'admin');

    expect(await screen.findByTestId('admin-fup-policy-page')).toBeInTheDocument();
    expect(screen.getByTestId('fup-test-only-banner')).toHaveTextContent('TEST ONLY');

    // Valid submit
    await user.click(screen.getByTestId('fup-submit-btn'));
    expect(await screen.findByTestId('fup-validation-passed')).toBeInTheDocument();

    // Break Tier 0 start threshold -> validation error
    const tier0Threshold = screen.getByTestId('fup-tier-threshold-0');
    await user.clear(tier0Threshold);
    await user.type(tier0Threshold, '500');
    await user.click(screen.getByTestId('fup-submit-btn'));

    expect(await screen.findByTestId('fup-validation-errors')).toBeInTheDocument();

    await assertAccessible(container);
  });

  it('TestAdminDnsPolicy_DomainNormalizationPreviewAndRollback', async () => {
    const user = userEvent.setup();
    const { container } = renderAt('/admin/policies/dns', 'admin');

    expect(await screen.findByTestId('admin-dns-policy-page')).toBeInTheDocument();

    // Preview tool matches subdomain of blocked example.com
    expect(screen.getByTestId('dns-preview-result')).toHaveTextContent('阻擋');

    // Reject URL with path in domain input
    await user.type(
      screen.getByTestId('dns-domain-input'),
      'https://bad.example.com/login',
    );
    await user.click(screen.getByTestId('dns-add-rule-btn'));
    expect(await screen.findByTestId('dns-domain-validation-error')).toBeInTheDocument();

    // Trigger rollback -> 202 OperationBanner
    await user.click(screen.getByTestId('dns-rollback-btn'));
    expect(await screen.findByTestId('operation-banner')).toBeInTheDocument();

    await assertAccessible(container);
  });

  it('TestAdminAudit_ReadOnlyLogWithNoDeleteControls', async () => {
    const { container } = renderAt('/admin/audit', 'admin');

    expect(await screen.findByTestId('admin-audit-page')).toBeInTheDocument();
    expect(screen.getByTestId('audit-table')).toBeInTheDocument();

    expect(
      screen.queryByRole('button', { name: /delete|刪除|clear/i }),
    ).not.toBeInTheDocument();

    await assertAccessible(container);
  });

  it('TestAT28_RoleGuard_SessionBootstrapStatesAnonymousMemberAdminWithoutMfaAndAdmin', async () => {
    // 1. Anonymous bootstrap state -> prompts sign-in with relative /auth/login?return_to=...
    renderAt('/admin/nodes?filter=online', 'anonymous');
    const anonPrompt = await screen.findByTestId('auth-anonymous-prompt');
    expect(anonPrompt).toBeInTheDocument();
    const loginLink = screen.getByTestId('login-redirect-btn');
    expect(loginLink).toHaveAttribute(
      'href',
      '/auth/login?return_to=%2Fadmin%2Fnodes%3Ffilter%3Donline',
    );

    // 2. Member bootstrap state on /admin/* -> 403 Forbidden
    const { container } = renderAt('/admin/nodes', 'member_active');
    const forbiddenCard = await screen.findByTestId('role-guard-forbidden');
    expect(forbiddenCard).toBeInTheDocument();
    expect(forbiddenCard).toHaveTextContent('403');
    expect(screen.queryByTestId('nodes-table')).not.toBeInTheDocument();
    await assertAccessible(container);

    // 3. Admin without MFA (mfa_verified=false) on /admin/* -> MFA required guard (§16.1)
    renderAt('/admin/nodes', 'admin_no_mfa');
    const mfaCard = await screen.findByTestId('role-guard-mfa-required');
    expect(mfaCard).toBeInTheDocument();
    expect(screen.getByTestId('mfa-login-btn')).toHaveAttribute(
      'href',
      '/auth/login?return_to=%2Fadmin%2Fnodes',
    );
    expect(screen.queryByTestId('nodes-table')).not.toBeInTheDocument();

    // 4. Admin with MFA (mfa_verified=true) on /admin/* -> allowed
    renderAt('/admin/nodes', 'admin');
    expect(await screen.findByTestId('nodes-table')).toBeInTheDocument();
  });

  it('TestMemberScreens_AllEntitlementStatesAndDisclosures_AT02_AT03', async () => {
    const user = userEvent.setup();

    // 1. Member Service page (throttled persona shows exact FUP phrase)
    renderAt('/member', 'member_throttled');
    expect(await screen.findByTestId('member-service-page')).toBeInTheDocument();
    expect(await screen.findByTestId('member-fup-throttled-banner')).toHaveTextContent(
      '本期大量使用，速度已依公平使用政策調整；下次重置為……',
    );

    // 2. Member Getting Started (no kill-switch disclosure + Android Private DNS note)
    renderAt('/member/getting-started', 'member_active');
    expect(await screen.findByTestId('member-getting-started-page')).toBeInTheDocument();
    expect(screen.getByTestId('no-kill-switch-disclosure')).toBeInTheDocument();
    expect(screen.getByTestId('android-private-dns-warning')).toBeInTheDocument();

    // 3. Member Exits (manual selection in NetBird app disclosure)
    renderAt('/member/exits', 'member_active');
    expect(await screen.findByTestId('member-exits-page')).toBeInTheDocument();
    expect(screen.getByTestId('manual-exit-selection-notice')).toBeInTheDocument();

    // 4. AT02: Member Devices quota & self-service revoke (and DOC12 Device ID column label)
    renderAt('/member/devices', 'member_active');
    expect(await screen.findByTestId('member-devices-page')).toBeInTheDocument();
    expect(screen.getByTestId('member-devices-table')).toHaveTextContent(
      '裝置識別碼 (Device ID)',
    );
    expect(screen.getByTestId('member-devices-table')).not.toHaveTextContent(
      '分配位址 ／ Peer ID',
    );
    expect(screen.getByTestId('device-quota-badge')).toHaveTextContent('2 / 3');
    await user.click(screen.getByTestId(`revoke-own-device-${PRIMARY_DEVICE_ID}`));
    expect(await screen.findByTestId('operation-banner')).toBeInTheDocument();

    // 5. Member Usage (honest metering + gap warning + > 2^53 decimal total)
    renderAt('/member/usage', 'member_throttled');
    expect(await screen.findByTestId('member-usage-page')).toBeInTheDocument();
    expect(screen.getByTestId('honest-metering-banner')).toBeInTheDocument();
    expect(screen.getByTestId('usage-gap-warning-banner')).toBeInTheDocument();
    expect(screen.getByTestId('usage-total-card')).toHaveTextContent('9007199255728647');

    // 6. AT03: Member Billing return redirect (?checkout=return) while pending_payment shows "確認中"
    renderAt('/member/billing?checkout=return', 'member_pending_payment');
    expect(await screen.findByTestId('member-billing-page')).toBeInTheDocument();
    expect(screen.getByTestId('no-card-collection-notice')).toBeInTheDocument();
    expect(screen.getByTestId('billing-return-pending-badge')).toHaveTextContent(
      '確認中',
    );
    expect(screen.getByTestId('checkout-return-pending-banner')).toBeInTheDocument();

    // 7. Member DNS Protection scope & Private DNS / DoH limitations
    renderAt('/member/dns-protection', 'member_active');
    expect(await screen.findByTestId('member-dns-protection-page')).toBeInTheDocument();
    expect(screen.getByTestId('dns-scope-card')).toBeInTheDocument();
    expect(screen.getByTestId('dns-limitations-card')).toBeInTheDocument();
  });

  it('TestEnrollmentDetailDirectRoute_ShowsPendingEnrollmentWithoutPlaintextToken', async () => {
    renderAt(`/admin/enrollments/${PRIMARY_ENROLLMENT_ID}`, 'admin');
    expect(await screen.findByTestId('admin-enrollment-detail-page')).toBeInTheDocument();
    expect(screen.getByTestId('secret-available-flag')).toHaveTextContent('false');
  });

  it('TestD057_AdminGroupsPage_ListCreateConflict409AndValidation422', async () => {
    const user = userEvent.setup();
    const { container } = renderAt('/admin/groups', 'admin');

    expect(await screen.findByTestId('admin-groups-page')).toBeInTheDocument();
    expect(await screen.findByTestId('group-row-standard-exits')).toBeInTheDocument();
    expect(screen.getByTestId('group-exit-count-standard-exits')).toHaveTextContent('2');
    expect(screen.getByTestId('group-row-default-members')).toBeInTheDocument();

    // 422 validation failure on invalid code
    await user.type(screen.getByTestId('group-code-input'), 'A');
    await user.type(screen.getByTestId('group-name-input'), 'Invalid Code Group');
    await user.click(screen.getByTestId('group-create-submit-btn'));
    expect(await screen.findByTestId('group-validation-detail')).toHaveTextContent(
      '/code: invalid',
    );
    expect(screen.getByTestId('group-code-field-error')).toHaveTextContent('/code');

    // 409 conflict on duplicate code
    await user.clear(screen.getByTestId('group-code-input'));
    await user.clear(screen.getByTestId('group-name-input'));
    await user.type(screen.getByTestId('group-code-input'), 'standard-exits');
    await user.type(screen.getByTestId('group-name-input'), 'Unique Name');
    await user.click(screen.getByTestId('group-create-submit-btn'));
    expect(await screen.findByTestId('group-conflict-field-error')).toHaveTextContent(
      '409',
    );

    // 201 create valid group
    await user.clear(screen.getByTestId('group-code-input'));
    await user.clear(screen.getByTestId('group-name-input'));
    await user.type(screen.getByTestId('group-code-input'), 'asia-premium');
    await user.type(screen.getByTestId('group-name-input'), 'Asia Premium Exits');
    await user.click(screen.getByTestId('group-create-submit-btn'));
    expect(await screen.findByTestId('group-row-asia-premium')).toBeInTheDocument();
    expect(screen.getByTestId('group-exit-count-asia-premium')).toHaveTextContent('0');

    await assertAccessible(container);
  });

  it('TestD057_D058_AdminNodeDetail_AttachedGroupsReplacement412AndCheckDetail', async () => {
    const user = userEvent.setup();
    renderAt(`/admin/nodes/${PRIMARY_NODE_ID}`, 'admin');

    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    expect(await screen.findByTestId('node-attached-groups')).toHaveTextContent(
      'standard-exits (Standard Member Exits)',
    );

    // D-058: NodeCheck.detail rendered in checks table
    expect(screen.getByTestId('node-checks-table')).toBeInTheDocument();
    expect(screen.getByTestId('node-check-detail-internet_only')).toHaveTextContent(
      'inet fvpn table verified; RFC1918 egress blocked',
    );
    expect(screen.getByTestId('node-check-detail-meter_path')).toHaveTextContent(
      'forward_admission_v1 counters active on wg-nb0',
    );

    // Replace attached groups together with publication intent via PUT .../publication
    const defaultMembersCheckbox = await screen.findByTestId(
      'pub-group-checkbox-default-members',
    );
    await user.click(defaultMembersCheckbox);
    await user.click(screen.getByTestId('publish-node-btn'));

    expect(await screen.findByTestId('operation-banner')).toBeInTheDocument();
    expect(await screen.findByTestId('node-attached-groups')).toHaveTextContent(
      'default-members (Default Members Group)',
    );

    // 412 conflict on PUT .../publication with stale If-Match
    await user.click(screen.getByTestId('simulate-pub-conflict-btn'));
    expect(await screen.findByTestId('conflict-dialog')).toBeInTheDocument();
    await user.click(screen.getByTestId('conflict-reload-btn'));
    expect(screen.queryByTestId('conflict-dialog')).not.toBeInTheDocument();
  });

  it('TestD057_MemberViewsNeverExposeGroupIds', async () => {
    const memberRoutes = [
      { path: '/member', testId: 'member-service-page' },
      { path: '/member/exits', testId: 'member-exits-page' },
      { path: '/member/devices', testId: 'member-devices-page' },
      { path: '/member/usage', testId: 'member-usage-page' },
      { path: '/member/billing', testId: 'member-billing-page' },
      { path: '/member/dns-protection', testId: 'member-dns-protection-page' },
      { path: '/member/getting-started', testId: 'member-getting-started-page' },
    ];

    for (const route of memberRoutes) {
      const { container, unmount } = renderAt(route.path, 'member_active');
      expect(await screen.findByTestId(route.testId)).toBeInTheDocument();
      const html = container.innerHTML;
      expect(html).not.toContain(PRIMARY_GROUP_ID);
      expect(html).not.toContain(SECONDARY_GROUP_ID);
      expect(html).not.toContain('standard-exits');
      expect(html).not.toContain('default-members');
      unmount();
    }
  });

  it('TestD058_AdminDnsPolicy_ThreatFeedStatusAndUpdateIntervalValidation', async () => {
    const user = userEvent.setup();
    renderAt('/admin/policies/dns', 'admin');

    expect(await screen.findByTestId('admin-dns-policy-page')).toBeInTheDocument();
    expect(screen.getByTestId('threat-feed-status-section')).toBeInTheDocument();

    // zh-TW wording for bootstrap-empty, last-good, and failing-retaining-last-good
    expect(screen.getByTestId('threat-feed-state-phishing-bootstrap')).toHaveTextContent(
      '初始空白狀態',
    );
    expect(screen.getByTestId('threat-feed-state-standard')).toHaveTextContent(
      '正常運作：使用最新驗證通過之最後良好版本',
    );
    expect(screen.getByTestId('threat-feed-state-botnet-c2-feed')).toHaveTextContent(
      '重新整理失敗：持續保留並使用上一版已驗證之最後良好版本',
    );

    // Switch to English and verify en wording
    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');
    expect(screen.getByTestId('threat-feed-state-phishing-bootstrap')).toHaveTextContent(
      'Bootstrap empty',
    );
    expect(screen.getByTestId('threat-feed-state-standard')).toHaveTextContent(
      'Healthy: serving verified last-good content',
    );
    expect(screen.getByTestId('threat-feed-state-botnet-c2-feed')).toHaveTextContent(
      'Refresh failing: retaining and serving verified last-good content',
    );

    // ThreatFeedPayload.update_interval_seconds default 21600 + client bounds 300-604800 + server 422 field detail
    await user.selectOptions(
      screen.getByTestId('extra-policy-schema-select'),
      'threat_feed',
    );
    const intervalInput = screen.getByTestId(
      'threat-feed-update-interval-input',
    ) as HTMLInputElement;
    expect(intervalInput.value).toBe('21600');

    // Client-side bounds enforcement (< 300)
    await user.clear(intervalInput);
    await user.type(intervalInput, '120');
    await user.click(screen.getByTestId('create-extra-policy-revision-btn'));
    expect(
      await screen.findByTestId('threat-feed-interval-field-error'),
    ).toHaveTextContent('300');

    // Server-side 422 field error detail (> 604800 with client check bypassed)
    await user.click(screen.getByTestId('threat-feed-bypass-client-check'));
    await user.clear(intervalInput);
    await user.type(intervalInput, '700000');
    await user.click(screen.getByTestId('create-extra-policy-revision-btn'));
    expect(
      await screen.findByTestId('threat-feed-interval-field-error'),
    ).toHaveTextContent('/payload/update_interval_seconds: schema_invalid');
    expect(await screen.findByTestId('dns-api-error-fields')).toHaveTextContent(
      '/payload/update_interval_seconds',
    );
  });

  it('TestD056_AdminNodeDetail_TypedDnsPolicyEvidencePopulatedAndNull', async () => {
    const user = userEvent.setup();

    // 1. Populated DnsPolicyEvidence on PRIMARY_NODE_ID in zh-TW and en
    renderAt(`/admin/nodes/${PRIMARY_NODE_ID}`, 'admin');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    expect(screen.getByTestId('node-dns-policy-evidence-section')).toBeInTheDocument();
    expect(screen.getByTestId('node-dns-evidence-disclaimer')).toHaveTextContent(
      '絕非中央驗證已套用之證明',
    );
    expect(screen.getByTestId('node-dns-evidence-document-id')).toHaveTextContent(
      '01900000-0000-7000-8000-000000000056',
    );
    expect(screen.getByTestId('node-dns-evidence-content-sha256')).toHaveTextContent(
      '2bed2c3a6225090d221f86c8d597c2782a44149ad9bafa7f02783c4cbaf5f20d',
    );
    expect(screen.getByTestId('node-dns-evidence-generation')).toHaveTextContent(
      '9007199254740993',
    );
    expect(screen.getByTestId('node-dns-evidence-outcome')).toHaveTextContent('applied');
    expect(
      screen.getByTestId('node-dns-evidence-feed-malware-domains-default'),
    ).toHaveTextContent('rules=1420');

    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');
    expect(screen.getByTestId('node-dns-evidence-disclaimer')).toHaveTextContent(
      'reported by the node and are never central proof of apply',
    );

    // 2. Null DnsPolicyEvidence on OFFLINE_NODE_ID shows clear "not yet reported" state in zh-TW and en
    renderAt(`/admin/nodes/${OFFLINE_NODE_ID}`, 'admin');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    expect(screen.getByTestId('node-dns-evidence-not-reported')).toHaveTextContent(
      'Not yet reported',
    );
    await user.selectOptions(screen.getByTestId('locale-switcher'), 'zh-TW');
    expect(screen.getByTestId('node-dns-evidence-not-reported')).toHaveTextContent(
      '尚未回報 (not yet reported)',
    );
  });

  it('TestD062_AdminFleetUpdates_RolledBackTargetCandidateHealthPopulatedAndNull', async () => {
    const user = userEvent.setup();

    // 1. Populated candidate_health on rolled_back target (side by side with restored health and previous_release_id)
    setMockRolledBackCandidateHealth('populated');
    renderAt('/admin/updates', 'admin');
    expect(await screen.findByTestId('admin-fleet-updates-page')).toBeInTheDocument();

    await user.click(
      await screen.findByTestId(`inspect-target-${SAFETY_FAILED_NODE_ID}`),
    );
    expect(await screen.findByTestId('batch-target-detail-panel')).toBeInTheDocument();
    expect(
      screen.getByTestId('target-detail-rollback-health-section'),
    ).toBeInTheDocument();

    const restoredCard = screen.getByTestId('target-detail-restored-health');
    const candidateCard = screen.getByTestId('target-detail-candidate-health');
    expect(restoredCard).toBeInTheDocument();
    expect(candidateCard).toBeInTheDocument();

    // Candidate named as previous_release_id
    expect(screen.getByTestId('target-detail-previous-release-id')).toHaveTextContent(
      '6d2797d0-b9a5-4724-96af-9b3039d351b1',
    );
    expect(
      within(restoredCard).getByTestId('restored-health-check-tunnel_probe'),
    ).toHaveTextContent('passed');
    expect(
      within(candidateCard).getByTestId('candidate-health-check-tunnel_probe'),
    ).toHaveTextContent('failed');
    expect(
      within(candidateCard).getByTestId('candidate-health-check-processes'),
    ).toHaveTextContent('passed');

    // 2. Null candidate_health on rolled_back target (e.g. rolled back before candidate health check ran)
    setMockRolledBackCandidateHealth('null');
    renderAt('/admin/updates', 'admin');
    expect(await screen.findByTestId('admin-fleet-updates-page')).toBeInTheDocument();

    await user.click(
      await screen.findByTestId(`inspect-target-${SAFETY_FAILED_NODE_ID}`),
    );
    expect(
      await screen.findByTestId('target-detail-rollback-health-section'),
    ).toBeInTheDocument();
    expect(screen.getByTestId('target-detail-previous-release-id')).toHaveTextContent(
      '6d2797d0-b9a5-4724-96af-9b3039d351b1',
    );
    expect(screen.getByTestId('restored-health-check-tunnel_probe')).toHaveTextContent(
      'passed',
    );
    expect(screen.getByTestId('target-detail-candidate-health-null')).toHaveTextContent(
      'candidate_health: null',
    );

    // 3. Target with no current receipt (waiting_online) shows explicit "no evidence" state rather than blanks
    await user.click(await screen.findByTestId(`inspect-target-${OFFLINE_NODE_ID}`));
    expect(
      await screen.findByTestId('target-detail-no-receipt-evidence'),
    ).toBeInTheDocument();
    expect(screen.getByTestId('target-detail-previous-release-id')).toHaveTextContent(
      /尚未回報|Not yet reported/,
    );
    expect(screen.getByTestId('target-detail-restored-health-null')).toHaveTextContent(
      /尚未回報|Not yet reported/,
    );
    expect(screen.getByTestId('target-detail-candidate-health-null')).toHaveTextContent(
      /尚未回報|Not yet reported/,
    );
  });

  it('TestD065_AdminSettings_FleetConfigUpdateSub900Warning412Conflict422FieldErrorAndAudit', async () => {
    const user = userEvent.setup();
    const { container } = renderAt('/admin/settings', 'admin');

    expect(await screen.findByTestId('admin-settings-page')).toBeInTheDocument();
    await assertAccessible(container);

    // 1. Shows current max_offline_lease_seconds, default 86400, updated_at, resource_version, and §16.3 copy
    expect(screen.getByTestId('fleet-config-current-seconds')).toHaveTextContent(
      '86400 s',
    );
    expect(screen.getByTestId('fleet-config-default-seconds')).toHaveTextContent(
      '86400 s',
    );
    expect(screen.getByTestId('fleet-config-resource-version')).toHaveTextContent('1');
    expect(screen.getByTestId('fleet-config-etag')).toHaveTextContent('ETag: "1"');
    expect(screen.getByTestId('fleet-config-semantics-copy')).toHaveTextContent('86400');

    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');
    expect(screen.getByTestId('fleet-config-semantics-copy')).toHaveTextContent(
      'applies only to serving leases issued or renewed after the change',
    );
    expect(screen.getByTestId('fleet-config-semantics-copy')).toHaveTextContent(
      'Already issued leases keep their existing value until renewal',
    );
    expect(screen.getByTestId('fleet-config-semantics-copy')).toHaveTextContent(
      '86400 seconds (24 hours) is the §16.3 ceiling, not a recommendation',
    );
    await user.selectOptions(screen.getByTestId('locale-switcher'), 'zh-TW');

    // 2. Typing < 900 s shows sub-900 warning visibly without blocking submission
    expect(screen.queryByTestId('fleet-config-sub-900-warning')).toBeNull();
    const secondsInput = screen.getByTestId('fleet-config-lease-seconds-input');
    const reasonInput = screen.getByTestId('fleet-config-reason-input');

    await user.clear(secondsInput);
    await user.type(secondsInput, '600');
    expect(screen.getByTestId('fleet-config-sub-900-warning')).toBeInTheDocument();
    expect(screen.getByTestId('fleet-config-submit-btn')).not.toBeDisabled();

    // 3. 422 VALIDATION_FAILED out of range (< 60) shows as a field error
    await user.clear(secondsInput);
    await user.type(secondsInput, '30');
    await user.clear(reasonInput);
    await user.type(reasonInput, 'Test out of range');
    await user.click(screen.getByTestId('fleet-config-submit-btn'));
    expect(await screen.findByTestId('fleet-config-field-error')).toHaveTextContent(
      'max_offline_lease_seconds',
    );
    expect(screen.getByTestId('fleet-config-seconds-field-error')).toBeInTheDocument();

    // 3b. 428 PRECONDITION_REQUIRED (missing If-Match) and 409 IDEMPOTENCY_CONFLICT (reused key with different payload) map through existing error map
    await expect(
      apiClient.updateFleetConfig(
        { max_offline_lease_seconds: 3600, reason: 'Missing If-Match' },
        null,
        'missing-if-match-action',
      ),
    ).rejects.toMatchObject({
      status: 428,
      code: 'PRECONDITION_REQUIRED',
    });

    // 4. Update to 3600 with reason succeeds (200), updates ETag to "2", and links to /admin/audit?resource_type=fleet_config
    await user.clear(secondsInput);
    await user.type(secondsInput, '3600');
    await user.clear(reasonInput);
    await user.type(reasonInput, 'Shorten new leases for the AT29 lab.');
    await user.click(screen.getByTestId('fleet-config-submit-btn'));

    expect(await screen.findByTestId('fleet-config-saved-banner')).toHaveTextContent(
      '3600',
    );
    expect(screen.getByTestId('fleet-config-current-seconds')).toHaveTextContent(
      '3600 s',
    );
    expect(screen.getByTestId('fleet-config-etag')).toHaveTextContent('ETag: "2"');

    // 5. Concurrent update advances server resource_version to "3"; submitting the page form with stale ETag "2" triggers 412 with current_resource_version: "3"
    await apiClient.updateFleetConfig(
      { max_offline_lease_seconds: 1800, reason: 'Concurrent operator update' },
      '"2"',
      'concurrent-op-action',
    );
    await user.clear(secondsInput);
    await user.type(secondsInput, '7200');
    await user.clear(reasonInput);
    await user.type(reasonInput, 'Stale ETag submission');
    await user.click(screen.getByTestId('fleet-config-submit-btn'));

    expect(await screen.findByTestId('fleet-config-conflict-banner')).toBeInTheDocument();
    expect(screen.getByTestId('fleet-config-conflict-version')).toHaveTextContent('3');
    expect(screen.getByTestId('conflict-dialog')).toBeInTheDocument();

    // Reload from conflict dialog loads 1800 and ETag "3"
    await user.click(screen.getByTestId('conflict-reload-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('fleet-config-current-seconds')).toHaveTextContent(
        '1800 s',
      );
      expect(screen.getByTestId('fleet-config-etag')).toHaveTextContent('ETag: "3"');
    });

    // 6. Save 3600 again and click the audit link to verify /admin/audit filtered to fleet_config shows fleet_config.update with before/after values
    await user.clear(secondsInput);
    await user.type(secondsInput, '3600');
    await user.clear(reasonInput);
    await user.type(reasonInput, 'Restore 3600 after reload');
    await user.click(screen.getByTestId('fleet-config-submit-btn'));
    const auditLink = await screen.findByTestId('fleet-config-audit-link');
    expect(auditLink).toHaveAttribute('href', '/admin/audit?resource_type=fleet_config');
    await user.click(auditLink);

    expect(await screen.findByTestId('admin-audit-page')).toBeInTheDocument();
    expect(screen.getByTestId('audit-resource-filter')).toHaveValue('fleet_config');
    expect(screen.getByTestId('audit-table')).toHaveTextContent('fleet_config.update');
    expect(screen.getByTestId('audit-table')).toHaveTextContent('1800');
    expect(screen.getByTestId('audit-table')).toHaveTextContent('3600');
  });

  it('TestD072_AdminNodeDetail_IdentityConflictThreeStatesAndRevokeLink', async () => {
    const user = userEvent.setup();

    // 1. {mapping_frozen: false, conflict_count: 0} -> quiet "no identity conflict" line in zh-TW and en
    renderAt(`/admin/nodes/${PRIMARY_NODE_ID}`, 'admin');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    expect(screen.getByTestId('node-identity-conflict-section')).toBeInTheDocument();
    expect(screen.getByTestId('node-identity-conflict-none')).toHaveTextContent(
      '無身分映射衝突 (no identity conflict)',
    );
    expect(screen.queryByTestId('node-identity-conflict-frozen-badge')).toBeNull();

    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');
    expect(screen.getByTestId('node-identity-conflict-none')).toHaveTextContent(
      'No identity conflict',
    );
    await user.selectOptions(screen.getByTestId('locale-switcher'), 'zh-TW');

    // 2. {mapping_frozen: true, conflict_count: 2, last_conflict_at: '2026-10-30T18:00:00Z'} -> prominent badge, count, locale timestamp, resolution text linking to existing revoke button
    const { container } = renderAt(`/admin/nodes/${SAFETY_FAILED_NODE_ID}`, 'admin');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    expect(screen.getByTestId('node-identity-conflict-frozen-badge')).toHaveTextContent(
      '身分映射已凍結',
    );
    expect(screen.getByTestId('node-identity-conflict-count')).toHaveTextContent('2');
    expect(screen.getByTestId('node-identity-conflict-last-at')).toHaveTextContent(
      '2026-10-30T18:00:00Z',
    );
    const revokeLink = screen.getByTestId('node-identity-conflict-revoke-link');
    expect(revokeLink).toHaveTextContent('請撤銷此節點並重新註冊；系統不會自動解除凍結');
    expect(revokeLink).toHaveAttribute('href', '#revoke-node-identity-btn');
    const revokeBtn = screen.getByTestId('revoke-node-identity-btn');
    expect(revokeBtn).toHaveAttribute('id', 'revoke-node-identity-btn');
    // Never show unfreeze button or endpoint hashes/addresses
    expect(container.innerHTML).not.toContain('endpoint_hash');
    expect(screen.queryByRole('button', { name: /unfreeze|解除凍結/i })).toBeNull();

    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');
    expect(screen.getByTestId('node-identity-conflict-frozen-badge')).toHaveTextContent(
      'Identity mapping frozen',
    );
    expect(screen.getByTestId('node-identity-conflict-count')).toHaveTextContent(
      'Conflict count (conflict_count): 2',
    );
    expect(screen.getByTestId('node-identity-conflict-last-at')).toHaveTextContent(
      '2026-10-30T18:00:00Z',
    );
    expect(screen.getByTestId('node-identity-conflict-revoke-link')).toHaveTextContent(
      'Revoke this node and enrol it again; there is no automatic unfreeze',
    );
    await assertAccessible(container);

    // 3. null or absent -> nothing rendered (no placeholder text)
    renderAt(`/admin/nodes/${OFFLINE_NODE_ID}`, 'admin');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    expect(screen.queryByTestId('node-identity-conflict-section')).toBeNull();
    expect(screen.queryByTestId('node-identity-conflict-none')).toBeNull();
    expect(screen.queryByTestId('node-identity-conflict-frozen-badge')).toBeNull();

    setMockNodeIdentityConflict(OFFLINE_NODE_ID, undefined);
    renderAt(`/admin/nodes/${OFFLINE_NODE_ID}`, 'admin');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    expect(screen.queryByTestId('node-identity-conflict-section')).toBeNull();
  });

  it('TestD067_D071_AdminNodeDetail_ProvisionalLeaseAwaitingMemberGrantsWaitingCopyVsLeaseExpiredFailureCopy', async () => {
    const user = userEvent.setup();

    // 1. Provisional lease waiting state: awaiting_member_grants + lease/qos/source_binding carrying AWAITING_MEMBER_GRANTS
    setMockNodeLeaseState(PRIMARY_NODE_ID, 'awaiting_member_grants');
    renderAt(`/admin/nodes/${PRIMARY_NODE_ID}`, 'admin');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();

    // Renders as waiting state, not failure, in zh-TW
    expect(
      screen.getByTestId('node-provisional-lease-waiting-banner'),
    ).toBeInTheDocument();
    expect(screen.getByTestId('node-provisional-lease-waiting-copy')).toHaveTextContent(
      '等待會員授權；節點會自動續租（最長 60 秒）',
    );
    expect(
      screen.getByTestId('forwarding-block-reason-awaiting_member_grants'),
    ).toHaveTextContent('等待會員授權；節點會自動續租（最長 60 秒）');
    expect(
      screen.getByTestId('forwarding-block-reason-awaiting_member_grants'),
    ).toHaveClass('badge-warning');
    expect(screen.queryByTestId('node-lease-expired-failure-banner')).toBeNull();

    for (const checkName of ['lease', 'qos', 'source_binding'] as const) {
      const statusBadge = screen.getByTestId(`node-check-status-${checkName}`);
      expect(statusBadge).toHaveClass('badge-warning');
      expect(statusBadge).not.toHaveClass('badge-danger');
      expect(statusBadge).toHaveTextContent('等待中 (waiting)');
      expect(statusBadge).not.toHaveTextContent('failed');
      expect(screen.getByTestId(`node-check-error-code-${checkName}`)).toHaveTextContent(
        'AWAITING_MEMBER_GRANTS',
      );
      expect(screen.getByTestId(`node-check-detail-${checkName}`)).toHaveTextContent(
        '等待會員授權；節點會自動續租（最長 60 秒）',
      );
    }
    expect(screen.getByTestId('node-latest-error-badge')).toHaveClass('badge-warning');
    expect(screen.getByTestId('node-latest-error-summary')).toHaveTextContent(
      '等待會員授權；節點會自動續租（最長 60 秒）',
    );

    // Switch to English and verify en waiting copy
    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');
    expect(screen.getByTestId('node-provisional-lease-waiting-copy')).toHaveTextContent(
      'Waiting for member grants; the node renews automatically (within 60 s)',
    );
    for (const checkName of ['lease', 'qos', 'source_binding'] as const) {
      const statusBadge = screen.getByTestId(`node-check-status-${checkName}`);
      expect(statusBadge).toHaveClass('badge-warning');
      expect(statusBadge).toHaveTextContent('waiting');
      expect(screen.getByTestId(`node-check-detail-${checkName}`)).toHaveTextContent(
        'Waiting for member grants; the node renews automatically (within 60 s)',
      );
    }
    expect(screen.getByTestId('node-latest-error-summary')).toHaveTextContent(
      'Waiting for member grants; the node renews automatically (within 60 s)',
    );

    // 2. lease_expired / LEASE_EXPIRED keeps its failure copy in both languages
    setMockNodeLeaseState(PRIMARY_NODE_ID, 'lease_expired');
    renderAt(`/admin/nodes/${PRIMARY_NODE_ID}`, 'admin');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    expect(screen.queryByTestId('node-provisional-lease-waiting-banner')).toBeNull();
    expect(screen.getByTestId('node-lease-expired-failure-banner')).toBeInTheDocument();
    expect(screen.getByTestId('node-lease-expired-failure-copy')).toHaveTextContent(
      'Node serving lease has expired; member forwarding has stopped.',
    );
    for (const checkName of ['lease', 'qos', 'source_binding'] as const) {
      const statusBadge = screen.getByTestId(`node-check-status-${checkName}`);
      expect(statusBadge).toHaveClass('badge-danger');
      expect(statusBadge).toHaveTextContent('failed');
      expect(screen.getByTestId(`node-check-error-code-${checkName}`)).toHaveTextContent(
        'LEASE_EXPIRED',
      );
      expect(screen.getByTestId(`node-check-detail-${checkName}`)).toHaveTextContent(
        'Node serving lease has expired; member forwarding has stopped.',
      );
    }

    await user.selectOptions(screen.getByTestId('locale-switcher'), 'zh-TW');
    expect(screen.getByTestId('node-lease-expired-failure-copy')).toHaveTextContent(
      '節點服務租約 (serving lease) 已過期，已停止接受會員流量。',
    );
    expect(screen.getByTestId('node-check-detail-lease')).toHaveTextContent(
      '節點服務租約 (serving lease) 已過期，已停止接受會員流量。',
    );
  });

  it('TestD076_MemberDevices_AuthorizationUnconfirmedVsFreshPendingAndAuthorized', async () => {
    const user = userEvent.setup();

    // Seed 3 states: authorized, fenced pending (authorization_unconfirmed: true), fresh pending (authorization_unconfirmed: false)
    setMockDeviceStatesForRevision12({ omitAbsentOnFreshPending: false });
    const { container } = renderAt('/member/devices', 'member_active');
    expect(await screen.findByTestId('member-devices-page')).toBeInTheDocument();

    // 1. Authorized device
    expect(screen.getByTestId(`device-status-${PRIMARY_DEVICE_ID}`)).toHaveTextContent(
      'authorized',
    );
    expect(
      screen.queryByTestId(`device-auth-unconfirmed-badge-${PRIMARY_DEVICE_ID}`),
    ).toBeNull();
    expect(
      screen.getByTestId(`revoke-own-device-${PRIMARY_DEVICE_ID}`),
    ).toBeInTheDocument();

    // 2. Fenced pending device (authorization_unconfirmed: true)
    expect(
      screen.getByTestId(`device-status-${FENCED_PENDING_DEVICE_ID}`),
    ).toHaveTextContent('pending');
    expect(
      screen.getByTestId(`device-auth-unconfirmed-badge-${FENCED_PENDING_DEVICE_ID}`),
    ).toHaveTextContent('待管理員重新授權');
    expect(
      screen.getByTestId(`device-auth-unconfirmed-note-${FENCED_PENDING_DEVICE_ID}`),
    ).toHaveTextContent('服務還原後需由管理員重新確認此裝置授權');
    expect(
      screen.queryByTestId(`device-pending-waiting-${FENCED_PENDING_DEVICE_ID}`),
    ).toBeNull();
    // No new member action on fenced pending device
    expect(
      screen.queryByTestId(`revoke-own-device-${FENCED_PENDING_DEVICE_ID}`),
    ).toBeNull();

    // 3. Fresh pending device (authorization_unconfirmed: false)
    expect(
      screen.getByTestId(`device-status-${FRESH_PENDING_DEVICE_ID}`),
    ).toHaveTextContent('pending');
    expect(
      screen.queryByTestId(`device-auth-unconfirmed-badge-${FRESH_PENDING_DEVICE_ID}`),
    ).toBeNull();
    expect(
      screen.queryByTestId(`device-auth-unconfirmed-note-${FRESH_PENDING_DEVICE_ID}`),
    ).toBeNull();
    expect(
      screen.getByTestId(`device-pending-waiting-${FRESH_PENDING_DEVICE_ID}`),
    ).toHaveTextContent('等待授權中');

    // Switch to English and verify en copy
    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');
    expect(
      screen.getByTestId(`device-auth-unconfirmed-badge-${FENCED_PENDING_DEVICE_ID}`),
    ).toHaveTextContent('Needs operator re-authorization');
    expect(
      screen.getByTestId(`device-auth-unconfirmed-note-${FENCED_PENDING_DEVICE_ID}`),
    ).toHaveTextContent(
      'An operator must re-authorize this device after a service restore. You cannot clear this state yourself; contact support.',
    );
    expect(
      screen.getByTestId(`device-pending-waiting-${FRESH_PENDING_DEVICE_ID}`),
    ).toHaveTextContent('Waiting for authorization');
    await assertAccessible(container);

    // 4. Absent authorization_unconfirmed renders identically to false
    setMockDeviceStatesForRevision12({ omitAbsentOnFreshPending: true });
    renderAt('/member/devices', 'member_active');
    expect(await screen.findByTestId('member-devices-page')).toBeInTheDocument();
    expect(
      screen.queryByTestId(`device-auth-unconfirmed-badge-${FRESH_PENDING_DEVICE_ID}`),
    ).toBeNull();
    expect(
      screen.getByTestId(`device-pending-waiting-${FRESH_PENDING_DEVICE_ID}`),
    ).toBeInTheDocument();
  });

  it('TestD077_D081_AdminDevicesList_FiltersReauthorizeAnd409Errors', async () => {
    const user = userEvent.setup();
    setMockDeviceStatesForRevision12({ deviceLimit: 2 });

    // Open with ?authorization_unconfirmed=true (restore runbook link) -> filter defaults to 'true'
    const { container } = renderAt(
      '/admin/devices?authorization_unconfirmed=true',
      'admin',
    );
    expect(await screen.findByTestId('admin-devices-list-page')).toBeInTheDocument();
    const unconfirmedSelect = screen.getByTestId(
      'admin-devices-unconfirmed-filter',
    ) as HTMLSelectElement;
    expect(unconfirmedSelect.value).toBe('true');

    // Only the fenced device is shown when authorization_unconfirmed=true
    expect(
      await screen.findByTestId(`admin-device-row-${FENCED_PENDING_DEVICE_ID}`),
    ).toBeInTheDocument();
    expect(screen.queryByTestId(`admin-device-row-${PRIMARY_DEVICE_ID}`)).toBeNull();
    expect(
      screen.getByTestId(`admin-device-member-link-${FENCED_PENDING_DEVICE_ID}`),
    ).toHaveAttribute('href', `/admin/members/${PRIMARY_MEMBER_ID}`);
    expect(
      screen.getByTestId(`admin-device-unconfirmed-since-${FENCED_PENDING_DEVICE_ID}`),
    ).toHaveTextContent('2026-10-30T08:30:00Z');

    // Re-authorize the fenced device via confirmation dialog -> 202 operation receipt -> leaves authorization_unconfirmed=true list
    await user.click(
      screen.getByTestId(`admin-reauthorize-device-btn-${FENCED_PENDING_DEVICE_ID}`),
    );
    expect(screen.getByTestId('reauthorize-confirm-dialog')).toBeInTheDocument();
    await user.click(screen.getByTestId('confirm-reauthorize-btn'));

    expect(
      await screen.findByTestId('reauthorize-operation-receipt'),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(
        screen.queryByTestId(`admin-device-row-${FENCED_PENDING_DEVICE_ID}`),
      ).toBeNull();
    });
    expect(screen.getByTestId('admin-devices-empty-row')).toBeInTheDocument();

    // Switch filter to 'false' (or 'all') -> see authorized and revoked/over-limit devices
    await user.selectOptions(unconfirmedSelect, 'all');
    expect(
      await screen.findByTestId(`admin-device-row-${FENCED_PENDING_DEVICE_ID}`),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(`admin-device-status-${FENCED_PENDING_DEVICE_ID}`),
    ).toHaveTextContent('authorized');
    expect(
      screen.queryByTestId(`admin-device-unconfirmed-badge-${FENCED_PENDING_DEVICE_ID}`),
    ).toBeNull();

    // Attempt Re-authorize on a revoked device -> 409 STATE_CONFLICT inline error copy from errors.ts
    await user.click(
      screen.getByTestId(`admin-reauthorize-device-btn-${REVOKED_DEVICE_ID}`),
    );
    await user.click(screen.getByTestId('confirm-reauthorize-btn'));
    const errorAlert = await screen.findByTestId('admin-device-authorize-error');
    expect(errorAlert).toHaveTextContent('STATE_CONFLICT');
    expect(errorAlert).toHaveTextContent(
      '資源目前狀態不允許此操作，請重新載入最新狀態後再試。',
    );

    // Attempt Re-authorize on an over-limit device -> 409 DEVICE_LIMIT_REACHED inline error copy keyed on error.code (D-081)
    await user.click(
      screen.getByTestId(`admin-reauthorize-device-btn-${OVER_LIMIT_DEVICE_ID}`),
    );
    await user.click(screen.getByTestId('confirm-reauthorize-btn'));
    const limitAlert = await screen.findByTestId('admin-device-authorize-error');
    expect(limitAlert).toHaveTextContent('DEVICE_LIMIT_REACHED');
    expect(limitAlert).toHaveTextContent(
      '已達方案允許的裝置上限（預設 3 台），請先撤銷舊裝置後再重試。',
    );

    // Filter by status=revoked -> only revoked device appears
    await user.selectOptions(
      screen.getByTestId('admin-devices-status-filter'),
      'revoked',
    );
    expect(
      await screen.findByTestId(`admin-device-row-${REVOKED_DEVICE_ID}`),
    ).toBeInTheDocument();
    expect(screen.queryByTestId(`admin-device-row-${PRIMARY_DEVICE_ID}`)).toBeNull();

    // Clear filters -> all devices restored
    await user.click(screen.getByTestId('admin-devices-clear-filters-btn'));
    expect(
      await screen.findByTestId(`admin-device-row-${PRIMARY_DEVICE_ID}`),
    ).toBeInTheDocument();

    await assertAccessible(container);
  });

  it('TestD077_ReusedSuccessfulReceiptWaitsForDeviceReadbackOnALaterPage', async () => {
    const user = userEvent.setup();
    setMockDeviceStatesForRevision12();
    const devices = await apiClient.listAdminDevices();
    const fenced = devices.items.find((d) => d.device_id === FENCED_PENDING_DEVICE_ID)!;
    const op = {
      ...CONTRACT_EXAMPLES_USED_BY_MOCKS.Operation,
      kind: 'member.device_authorize',
      status: 'succeeded',
    };
    let phase = 0;
    const observedPhases: number[] = [];
    const cursors: Array<string | null> = [];
    server.use(
      http.post('/api/v1/admin/members/:memberId/devices/:deviceId/authorize', () =>
        HttpResponse.json(op, { status: 202 }),
      ),
      http.get('/api/v1/admin/devices', ({ request }) => {
        const params = new URL(request.url).searchParams;
        if (params.get('member_id')) {
          cursors.push(params.get('cursor'));
          observedPhases.push(phase);
          return HttpResponse.json(
            params.has('cursor')
              ? {
                  items: [
                    {
                      ...fenced,
                      status: phase === 2 ? 'authorized' : 'pending',
                      authorization_unconfirmed: phase === 0,
                    },
                  ],
                  next_cursor: null,
                }
              : { items: [devices.items[0]], next_cursor: 'second-page' },
          );
        }
        return HttpResponse.json({ items: phase > 0 ? [] : [fenced], next_cursor: null });
      }),
    );
    renderAt('/admin/devices?authorization_unconfirmed=true');
    await user.click(
      await screen.findByTestId(
        `admin-reauthorize-device-btn-${FENCED_PENDING_DEVICE_ID}`,
      ),
    );
    await user.click(screen.getByTestId('cancel-reauthorize-btn'));
    expect(screen.queryByTestId('reauthorize-operation-receipt')).toBeNull();
    await user.click(
      screen.getByTestId(`admin-reauthorize-device-btn-${FENCED_PENDING_DEVICE_ID}`),
    );
    await user.click(screen.getByTestId('confirm-reauthorize-btn'));
    const state = await screen.findByTestId('reauthorize-device-state');
    expect(state).toHaveTextContent('等待裝置授權確認中');
    expect(screen.getByTestId('operation-banner')).toHaveAttribute(
      'data-operation-status',
      'succeeded',
    );
    expect(screen.getByTestId('operation-status-headline')).toHaveTextContent(
      '此操作紀錄不代表裝置已完成授權',
    );
    expect(
      within(screen.getByTestId('operation-banner')).getByRole('link'),
    ).toHaveAttribute('href', `/api/v1/admin/operations/${op.operation_id}`);
    await waitFor(() => expect(cursors).toContain('second-page'));
    // Even after the fence clears, pending status is insufficient evidence.
    phase = 1;
    await waitFor(() => expect(observedPhases).toContain(1), { timeout: 3000 });
    expect(state).toHaveTextContent('等待裝置授權確認中');
    phase = 2;
    await waitFor(() => expect(state).toHaveTextContent('已確認裝置授權'), {
      timeout: 4000,
    });
    await waitFor(() =>
      expect(
        screen.queryByTestId(`admin-device-row-${FENCED_PENDING_DEVICE_ID}`),
      ).toBeNull(),
    );
  }, 10_000);

  it('TestD077_AdminDevicesCursorResetStatusMemberFiltersAndNavigationLink', async () => {
    const user = userEvent.setup();
    setMockDeviceStatesForRevision12();
    const devices = (await apiClient.listAdminDevices()).items;
    const requests: URLSearchParams[] = [];
    server.use(
      http.get('/api/v1/admin/devices', ({ request }) => {
        const params = new URL(request.url).searchParams;
        requests.push(params);
        const second = params.has('cursor');
        return HttpResponse.json({
          items: [devices[second ? 1 : 0]],
          next_cursor: second ? null : 'opaque-filter-bound-cursor',
        });
      }),
    );
    renderAt('/admin/devices');
    await user.click(await screen.findByTestId('admin-devices-next-page-btn'));
    await waitFor(() =>
      expect(requests.at(-1)?.get('cursor')).toBe('opaque-filter-bound-cursor'),
    );
    await user.click(screen.getByTestId('admin-devices-prev-page-btn'));
    await waitFor(() => expect(requests.at(-1)?.has('cursor')).toBe(false));
    await user.click(screen.getByTestId('admin-devices-next-page-btn'));
    await waitFor(() => expect(requests.at(-1)?.has('cursor')).toBe(true));
    await user.selectOptions(
      screen.getByTestId('admin-devices-status-filter'),
      'pending',
    );
    await waitFor(() => expect(requests.at(-1)?.get('status')).toBe('pending'));
    expect(requests.at(-1)?.has('cursor')).toBe(false);
    await user.type(screen.getByTestId('admin-devices-member-filter'), PRIMARY_MEMBER_ID);
    expect(requests.at(-1)?.has('member_id')).toBe(false);
    await user.click(screen.getByTestId('admin-devices-apply-member-filter'));
    await waitFor(() =>
      expect(requests.at(-1)?.get('member_id')).toBe(PRIMARY_MEMBER_ID),
    );
    expect(requests.at(-1)?.has('cursor')).toBe(false);
    await user.selectOptions(
      screen.getByTestId('admin-devices-unconfirmed-filter'),
      'false',
    );
    await waitFor(() =>
      expect(requests.at(-1)?.get('authorization_unconfirmed')).toBe('false'),
    );
    await user.click(screen.getByTestId('admin-devices-clear-filters-btn'));
    await waitFor(() => expect(requests.at(-1)?.toString()).toBe(''));
    renderAt(`/admin/members/${PRIMARY_MEMBER_ID}`);
    expect(await screen.findByTestId('member-detail-view-devices-link')).toHaveAttribute(
      'href',
      `/admin/devices?member_id=${PRIMARY_MEMBER_ID}`,
    );
  });

  it.each([409, 422])('TestD081_DeviceLimitCopyUsesCodeAtHTTP%s', async (status) => {
    const user = userEvent.setup();
    setMockDeviceStatesForRevision12();
    server.use(
      http.post('/api/v1/admin/members/:memberId/devices/:deviceId/authorize', () =>
        HttpResponse.json(
          {
            error: {
              code: 'DEVICE_LIMIT_REACHED',
              message: 'TEST ONLY limit',
              request_id: 'test-limit',
              retryable: false,
              details: {},
            },
          },
          { status },
        ),
      ),
    );
    renderAt('/admin/devices?authorization_unconfirmed=true');
    await user.click(
      await screen.findByTestId(
        `admin-reauthorize-device-btn-${FENCED_PENDING_DEVICE_ID}`,
      ),
    );
    await user.click(screen.getByTestId('confirm-reauthorize-btn'));
    expect(await screen.findByTestId('admin-device-authorize-error')).toHaveTextContent(
      '已達方案允許的裝置上限',
    );
    expect(
      screen.getByTestId(`admin-device-row-${FENCED_PENDING_DEVICE_ID}`),
    ).toBeInTheDocument();
    expect(screen.queryByTestId('reauthorize-operation-receipt')).toBeNull();
  });

  it('TestSECOPS19_D077_AdminNodeDetail_UsageOutboxThreeStatesInformationalOnly', async () => {
    const user = userEvent.setup();

    // State 1: Object (PRIMARY_NODE_ID) — values rendered, relative age from observed_at, no alarm classes
    const { container } = renderAt(`/admin/nodes/${PRIMARY_NODE_ID}`, 'admin');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    const outboxSection = screen.getByTestId('node-usage-outbox-section');
    expect(outboxSection).toBeInTheDocument();
    expect(screen.getByTestId('node-usage-outbox-values')).toBeInTheDocument();
    expect(screen.getByTestId('node-usage-outbox-unacked')).toHaveTextContent(
      '2 / 1840 B',
    );
    expect(screen.getByTestId('node-usage-outbox-acked-through')).toHaveTextContent(
      '10 / 12',
    );
    expect(screen.getByTestId('node-usage-outbox-oldest-age')).toHaveTextContent(
      '30 秒 (2026-10-28T14:06:00Z)',
    );
    expect(screen.getByTestId('node-usage-outbox-archive-bytes')).toHaveTextContent(
      '4096 B',
    );
    expect(screen.getByTestId('node-usage-outbox-observed-at')).toHaveTextContent(
      '2026-10-28T14:06:30Z',
    );
    // Informational only: no danger badge or danger alert inside usage outbox section
    expect(outboxSection.querySelector('.badge-danger')).toBeNull();
    expect(outboxSection.querySelector('.alert-danger')).toBeNull();

    // Switch to English and verify relative age formatting
    await user.selectOptions(screen.getByTestId('locale-switcher'), 'en');
    expect(screen.getByTestId('node-usage-outbox-oldest-age')).toHaveTextContent(
      '30s (2026-10-28T14:06:00Z)',
    );
    await assertAccessible(container);

    // State 2: null (SAFETY_FAILED_NODE_ID) — "no report yet"
    renderAt(`/admin/nodes/${SAFETY_FAILED_NODE_ID}`, 'admin');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    expect(screen.getByTestId('node-usage-outbox-null')).toHaveTextContent(
      'No report yet',
    );

    // State 3: field absent (OFFLINE_NODE_ID) — "not projected by this server"
    renderAt(`/admin/nodes/${OFFLINE_NODE_ID}`, 'admin');
    expect(await screen.findByTestId('admin-node-detail-page')).toBeInTheDocument();
    expect(screen.getByTestId('node-usage-outbox-absent')).toHaveTextContent(
      'Not projected by this server',
    );
  });
});
