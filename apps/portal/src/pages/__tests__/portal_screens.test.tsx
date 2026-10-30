import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { AppRoutes } from '../../App';
import { SessionProvider } from '../../context/SessionContext';
import { I18nProvider } from '../../i18n';
import {
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
});
