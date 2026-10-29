import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';
import { AppRoutes } from '../../App';
import { SessionProvider } from '../../context/SessionContext';
import { I18nProvider } from '../../i18n';
import type { MockPersonaKey } from '../../mocks/data';
import { assertAccessible } from '../../test/setup';

function renderPortalAt(initialEntry: string, persona: MockPersonaKey = 'admin') {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <I18nProvider initialLocale="zh-TW">
        <SessionProvider initialPersona={persona}>
          <MemoryRouter initialEntries={[initialEntry]}>
            <AppRoutes />
          </MemoryRouter>
        </SessionProvider>
      </I18nProvider>
    </QueryClientProvider>,
  );
}

describe('Portal Screens & Acceptance Flows (§18, AT01..AT04, AT08, AT23)', () => {
  it('TestID01_AT28_MemberRouteGuardBlocksAdminRoutes', async () => {
    const { container } = renderPortalAt('/admin/nodes', 'member_active');

    await waitFor(() => {
      expect(screen.getByTestId('role-guard-forbidden')).toBeInTheDocument();
    });

    expect(screen.getByTestId('role-guard-forbidden')).toHaveTextContent(
      '存取遭拒：權限不足 (403)',
    );
    await assertAccessible(container);
  });

  it('TestENR01_AT06_OneTimeEnrollmentTokenShownOnceAndGoneAfterNavigation', async () => {
    const user = userEvent.setup();
    const { container } = renderPortalAt('/admin/nodes/new', 'admin');

    await waitFor(() => {
      expect(screen.getByTestId('create-enrollment-submit-btn')).toBeInTheDocument();
    });
    await assertAccessible(container);

    // Submit enrollment creation
    await user.click(screen.getByTestId('create-enrollment-submit-btn'));

    await waitFor(() => {
      expect(screen.getByTestId('one-time-token-panel')).toBeInTheDocument();
    });

    const tokenText = screen.getByTestId('one-time-token-value').textContent ?? '';
    expect(tokenText).toMatch(/^fvpn_enr_test_only_/);

    // Verify token is NEVER stored in localStorage or sessionStorage
    expect(JSON.stringify(window.localStorage)).not.toContain(tokenText);
    expect(JSON.stringify(window.sessionStorage)).not.toContain(tokenText);

    // Navigate to the enrollment detail status page (simulating re-entering)
    await user.click(screen.getByTestId('reenter-enrollment-status-btn'));

    await waitFor(() => {
      expect(screen.getByTestId('enrollment-status-card')).toBeInTheDocument();
    });

    // Plaintext token is gone; only state (`unused`) is shown
    expect(screen.queryByTestId('one-time-token-value')).not.toBeInTheDocument();
    expect(document.body.textContent).not.toContain(tokenText);
    expect(screen.getByTestId('enrollment-state-display')).toHaveTextContent('unused');
    await assertAccessible(container);
  });

  it('TestAPI01_412ConflictDialogReloadsResourceAndDoesNotOverwrite', async () => {
    const user = userEvent.setup();
    const { container } = renderPortalAt(
      '/admin/nodes/93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb',
      'admin',
    );

    await waitFor(() => {
      expect(screen.getByTestId('simulate-412-btn')).toBeInTheDocument();
    });
    expect(screen.getByTestId('node-detail-meta')).toHaveTextContent('v42');

    // Trigger a concurrent edit conflict (412)
    await user.click(screen.getByTestId('simulate-412-btn'));

    await waitFor(() => {
      expect(screen.getByTestId('conflict-dialog')).toBeInTheDocument();
    });
    expect(screen.getByTestId('conflict-dialog')).toHaveTextContent('43');

    // Reloading from conflict dialog fetches latest server version (v43) without overwriting
    await user.click(screen.getByTestId('conflict-reload-btn'));

    await waitFor(() => {
      expect(screen.queryByTestId('conflict-dialog')).not.toBeInTheDocument();
    });
    expect(screen.getByTestId('node-detail-meta')).toHaveTextContent('v43');
    await assertAccessible(container);
  });

  it('TestEXIT01_AT08_PublishAndOfflineUnpublish202ProgressAndSpecWording', async () => {
    const user = userEvent.setup();
    // Offline node de-fra-01: disabling shows partial completion & exact lease wording
    renderPortalAt('/admin/nodes/a240f510-77b8-4c02-8d99-2238d40a5105', 'admin');

    await waitFor(() => {
      expect(screen.getByTestId('disable-node-btn')).toBeInTheDocument();
    });

    await user.click(screen.getByTestId('disable-node-btn'));

    // Immediately upon 202, banner shows 「已儲存，等待套用」 (never done on 202!)
    await waitFor(() => {
      expect(screen.getByTestId('operation-banner')).toBeInTheDocument();
    });
    expect(screen.getByTestId('operation-status-headline')).toHaveTextContent(
      '已儲存，等待套用',
    );

    // Wait for operation poll to finish -> shows exact spec phrase for offline unpublish
    await waitFor(
      () => {
        expect(
          screen.getByTestId('operation-offline-unpublish-warning'),
        ).toHaveTextContent('中央已撤銷路由；設備撤權尚未確認／受租約限制');
      },
      { timeout: 3000 },
    );
  });

  it('TestUPD01_UPD05_AT23_FleetUpdateScreenAndNodesListAccessible', async () => {
    const user = userEvent.setup();
    const { container } = renderPortalAt('/admin/updates', 'admin');

    await waitFor(() => {
      expect(screen.getByTestId('batch-buckets-grid')).toBeInTheDocument();
    });

    expect(screen.getByTestId('batch-target-count')).toHaveTextContent('12');
    expect(screen.getByTestId('batch-bucket-sum')).toHaveTextContent('12 / 12');
    expect(screen.getByTestId('batch-verified-success-banner')).toHaveTextContent('2');
    expect(screen.getByTestId('release-restart-indicator')).toHaveTextContent(
      'requires_service_restart: true',
    );

    await user.click(screen.getByTestId('run-update-all-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('update-batch-accepted-banner')).toHaveTextContent(
        '已受理',
      );
    });

    await assertAccessible(container);
  });

  it('TestNODE02_NodesListShowsFourSeparateBadgesAndExactIsolationSummary', async () => {
    const { container } = renderPortalAt('/admin/nodes', 'admin');

    await waitFor(() => {
      expect(screen.getByTestId('nodes-table')).toBeInTheDocument();
    });

    expect(screen.getByTestId('node-summary-jp-tyo-01')).toHaveTextContent(
      '在線，隔離檢查失敗，未提供服務',
    );
    expect(screen.getByTestId('node-summary-de-fra-01')).toHaveTextContent(
      '中央已撤銷路由；設備撤權尚未確認／受租約限制',
    );
    await assertAccessible(container);
  });

  it('TestMEM01_FUP03_AT03_MemberScreensHonestyAndAccessibility', async () => {
    // 1. Throttled member on Usage & Speed page shows exact FUP phrase and gap warning
    const { container, unmount } = renderPortalAt('/member/usage', 'member_throttled');

    await waitFor(() => {
      expect(screen.getByTestId('usage-fup-throttled-banner')).toBeInTheDocument();
    });
    expect(screen.getByTestId('usage-fup-throttled-banner')).toHaveTextContent(
      '本期大量使用，速度已依公平使用政策調整；下次重置為……',
    );
    expect(screen.getByTestId('usage-gap-warning')).toBeInTheDocument();
    expect(screen.getByTestId('member-usage-honesty-banner')).toHaveTextContent(
      '並非即時跳動的流量儀表',
    );
    await assertAccessible(container);
    unmount();

    // 2. Billing checkout return page (?checkout=return) shows 「確認中」 (AT03)
    const billingRender = renderPortalAt(
      '/member/billing?checkout=return',
      'member_pending',
    );
    await waitFor(() => {
      expect(screen.getByTestId('checkout-return-verifying-banner')).toBeInTheDocument();
    });
    expect(screen.getByTestId('checkout-return-verifying-banner')).toHaveTextContent(
      '確認中',
    );
    await assertAccessible(billingRender.container);
    billingRender.unmount();

    // 3. Getting Started shows §7.3 kill-switch limitation and §12.6 Android Private DNS warning
    const user = userEvent.setup();
    const gsRender = renderPortalAt('/member/getting-started', 'member_active');
    await waitFor(() => {
      expect(screen.getByTestId('kill-switch-limitation-card')).toBeInTheDocument();
    });
    await user.selectOptions(screen.getByTestId('os-platform-select'), 'android');
    expect(screen.getByTestId('android-private-dns-warning')).toBeInTheDocument();
    await assertAccessible(gsRender.container);
    gsRender.unmount();
  });

  it('TestAllRemainingScreens_AdminOverviewMembersFupDnsAuditAndMemberPagesAccessible', async () => {
    const user = userEvent.setup();

    // Admin Overview
    const ov = renderPortalAt('/admin', 'admin');
    await waitFor(() => {
      expect(screen.getByTestId('metric-available-exits')).toBeInTheDocument();
    });
    await assertAccessible(ov.container);
    ov.unmount();

    // Admin Members List & Detail
    const memList = renderPortalAt('/admin/members', 'admin');
    await waitFor(() => {
      expect(screen.getByTestId('members-table')).toBeInTheDocument();
    });
    await assertAccessible(memList.container);
    memList.unmount();

    const memDetail = renderPortalAt(
      '/admin/members/11111111-2222-4333-8444-555555555501',
      'admin',
    );
    await waitFor(() => {
      expect(screen.getByTestId('suspend-member-btn')).toBeInTheDocument();
    });
    await user.click(screen.getByTestId('create-grant-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('operation-banner')).toBeInTheDocument();
    });
    await assertAccessible(memDetail.container);
    memDetail.unmount();

    // Admin FUP Policy Page
    const fupPage = renderPortalAt('/admin/policies/fup', 'admin');
    await waitFor(() => {
      expect(screen.getByTestId('fup-test-only-banner')).toBeInTheDocument();
    });
    await user.click(screen.getByTestId('publish-fup-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('fup-validation-passed')).toBeInTheDocument();
    });
    await assertAccessible(fupPage.container);
    fupPage.unmount();

    // Admin DNS Policy Page
    const dnsPage = renderPortalAt('/admin/policies/dns', 'admin');
    await waitFor(() => {
      expect(screen.getByTestId('dns-preview-result')).toBeInTheDocument();
    });
    await user.type(screen.getByTestId('dns-domain-input'), 'https://invalid.url/path');
    await user.click(screen.getByTestId('add-dns-rule-btn'));
    expect(screen.getByTestId('dns-rule-error')).toBeInTheDocument();
    await user.clear(screen.getByTestId('dns-domain-input'));
    await user.type(screen.getByTestId('dns-domain-input'), 'malicious.example.com');
    await user.click(screen.getByTestId('add-dns-rule-btn'));
    await user.click(screen.getByTestId('publish-dns-btn'));
    await assertAccessible(dnsPage.container);
    dnsPage.unmount();

    // Admin Audit Page
    const auditPage = renderPortalAt('/admin/audit', 'admin');
    await waitFor(() => {
      expect(screen.getByTestId('audit-table')).toBeInTheDocument();
    });
    await assertAccessible(auditPage.container);
    auditPage.unmount();

    // Member Exits, Devices, DNS Protection, and 401 Session Expired flow
    const exitsPage = renderPortalAt('/member/exits', 'member_active');
    await waitFor(() => {
      expect(screen.getByTestId('manual-exit-selection-notice')).toBeInTheDocument();
    });
    await assertAccessible(exitsPage.container);
    exitsPage.unmount();

    const devPage = renderPortalAt('/member/devices', 'member_active');
    await waitFor(() => {
      expect(screen.getByTestId('member-devices-table')).toBeInTheDocument();
    });
    await user.click(screen.getByTestId('revoke-own-device-macbook-air-m3'));
    await waitFor(() => {
      expect(screen.getByTestId('operation-banner')).toBeInTheDocument();
    });
    await assertAccessible(devPage.container);
    devPage.unmount();

    const dnsInfoPage = renderPortalAt('/member/dns-protection', 'member_active');
    await waitFor(() => {
      expect(screen.getByTestId('dns-protection-scope-card')).toBeInTheDocument();
    });
    await user.click(screen.getByTestId('simulate-401-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('session-expired-banner')).toBeInTheDocument();
    });
    await user.click(screen.getByTestId('session-relogin-btn'));
    await waitFor(() => {
      expect(screen.queryByTestId('session-expired-banner')).not.toBeInTheDocument();
    });
    await assertAccessible(dnsInfoPage.container);
    dnsInfoPage.unmount();
  });
});
