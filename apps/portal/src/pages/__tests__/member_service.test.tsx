import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';
import { AppRoutes } from '../../App';
import {
  deriveMemberServiceReason,
  deriveMemberServiceState,
  type EntitlementState,
  type MemberSelf,
} from '../../api';
import { SessionProvider } from '../../context/SessionContext';
import { I18nProvider, type Locale } from '../../i18n';
import { formatLocaleWithUtc } from '../../lib/datetime';
import { buildMemberSelfForPersona, buildMemberUsageForPersona } from '../../mocks/data';
import { setMockPersona } from '../../mocks/handlers';
import { server } from '../../mocks/server';
import { assertAccessible } from '../../test/setup';

const end = '2026-10-01T00:00:00Z';
const graceEnd = '2026-10-03T00:00:00Z';
const locales = ['zh-TW', 'en'] as const;
const usableText = [
  '目前服務資格允許使用出口。',
  'Your entitlement currently allows exit access.',
] as const;
const unusableText = [
  '目前服務資格不允許使用出口。',
  'Your entitlement currently does not allow exit access.',
] as const;

// Independent expected copy: do not derive expectations from the dictionaries or view model.
const cases: Array<{
  name: string;
  state: EntitlementState;
  actions: MemberSelf['next_actions'];
  usable: boolean;
  labels: [string, string];
  reasons: [string, string];
}> = [
  {
    name: 'pending_payment_complete',
    state: 'pending_payment',
    actions: ['complete_payment'],
    usable: false,
    labels: ['待付款', 'Awaiting payment'],
    reasons: [
      '尚未確認付款，服務尚未開通。',
      'Payment has not been confirmed; service is not yet available.',
    ],
  },
  {
    name: 'pending_payment_confirming',
    state: 'pending_payment',
    actions: ['wait_for_payment_confirmation'],
    usable: false,
    labels: ['開通中', 'Confirmation in progress'],
    reasons: [
      '尚未確認付款，服務尚未開通。',
      'Payment has not been confirmed; service is not yet available.',
    ],
  },
  {
    name: 'active',
    state: 'active',
    actions: ['install_app', 'register_device'],
    usable: true,
    labels: ['有效', 'Active'],
    reasons: [
      '訂閱狀態正常，服務資格有效。',
      'Your subscription is in order and your service entitlement is active.',
    ],
  },
  {
    name: 'cancel_at_period_end',
    state: 'cancel_at_period_end',
    actions: ['install_app', 'register_device'],
    usable: true,
    labels: ['有效（期末取消）', 'Active (ends at period end)'],
    reasons: [
      '已設定期末取消，服務可繼續使用至 {end}。',
      'Cancellation is scheduled for period end; service continues until {end}.',
    ],
  },
  {
    name: 'grace',
    state: 'grace',
    actions: ['install_app', 'register_device'],
    usable: true,
    labels: ['寬限期', 'Grace period'],
    reasons: [
      '續費待確認，服務可繼續使用至寬限期限 {graceUntil}。',
      'Renewal is pending; service continues until the grace deadline of {graceUntil}.',
    ],
  },
  {
    name: 'past_due',
    state: 'past_due',
    actions: ['renew_subscription'],
    usable: false,
    labels: ['付款逾期', 'Payment overdue'],
    reasons: [
      '付款已逾期，目前無法使用服務。',
      'Payment is overdue and service is unavailable.',
    ],
  },
  {
    name: 'expired',
    state: 'expired',
    actions: ['renew_subscription'],
    usable: false,
    labels: ['到期', 'Expired'],
    reasons: [
      '服務期間已於 {end} 結束，無法再使用出口。',
      'Your service period ended at {end}; exit access is no longer available.',
    ],
  },
  {
    name: 'suspended',
    state: 'suspended',
    actions: ['contact_support'],
    usable: false,
    labels: ['停權', 'Suspended'],
    reasons: [
      '管理員已暫停您的服務；付款或續費不會解除停權。',
      'An administrator has suspended your service; payment or renewal does not lift the suspension.',
    ],
  },
  {
    name: 'closed',
    state: 'closed',
    actions: ['contact_support'],
    usable: false,
    labels: ['帳號已關閉', 'Account closed'],
    reasons: [
      '您的帳號已關閉，無法使用服務。',
      'Your account is closed and service is unavailable.',
    ],
  },
];

function member(
  state: EntitlementState,
  actions: MemberSelf['next_actions'],
): MemberSelf {
  const me = buildMemberSelfForPersona('member_active');
  me.entitlement = {
    ...me.entitlement,
    state,
    effective_service_end: end,
    grace_until: graceEnd,
    admin_suspended: state === 'suspended' || state === 'closed',
  };
  me.next_actions = actions;
  return me;
}

function renderMember(
  me: MemberSelf,
  locale: Locale,
  path = '/member/service',
  selfReady?: Promise<void>,
) {
  setMockPersona('member_active');
  const usage = buildMemberUsageForPersona('member_active');
  usage.fair_use.tier_name = null; // Exercise the localized fallback, not fixture copy.
  server.use(
    http.get('/api/v1/member/me', async () => {
      await selfReady;
      return HttpResponse.json(me);
    }),
    http.get('/api/v1/member/usage', () => HttpResponse.json(usage)),
  );
  return render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <I18nProvider initialLocale={locale}>
        <SessionProvider>
          <MemoryRouter initialEntries={[path]}>
            <AppRoutes />
          </MemoryRouter>
        </SessionProvider>
      </I18nProvider>
    </QueryClientProvider>,
  );
}

function expectedReason(row: (typeof cases)[number], locale: Locale): string {
  return row.reasons[locale === 'zh-TW' ? 0 : 1]
    .replace('{end}', formatLocaleWithUtc(end, locale))
    .replace('{graceUntil}', formatLocaleWithUtc(graceEnd, locale));
}

for (const locale of locales) {
  const i = locale === 'zh-TW' ? 0 : 1;
  const j = i === 0 ? 1 : 0;
  const other = locales[j];
  describe(`TestAT04_F404_MemberServiceScreen_${locale}`, () => {
    for (const row of cases) {
      it(row.name, async () => {
        const { container } = renderMember(member(row.state, row.actions), locale);
        const badge = await screen.findByTestId('member-entitlement-state-badge');
        const service = screen.getByTestId('member-service-page');
        if (locale === 'zh-TW') {
          expect(service).not.toHaveTextContent(
            /Devices:|None|Normal speed|Loading service status\.\.\./,
          );
        }
        expect(screen.getByTestId('member-device-quota').textContent).toBe(
          locale === 'zh-TW'
            ? '已授權裝置名額使用量：2 / 3 台'
            : 'Authorized Device Quota: 2 / 3 devices',
        );
        expect(screen.getByTestId('member-service-speed').textContent).toBe(
          locale === 'zh-TW' ? '正常速度' : 'Normal speed',
        );
        expect(screen.getByTestId('member-service-speed')).not.toHaveTextContent(
          locale === 'zh-TW' ? 'Normal speed' : '正常速度',
        );
        expect(badge).toHaveTextContent(row.labels[i]);
        expect(badge.textContent).toBe(row.labels[i]);
        expect(badge).not.toHaveTextContent(row.labels[j]);
        expect(badge).toHaveAttribute('data-state', row.state);
        expect(badge).toHaveClass(row.usable ? 'badge-success' : 'badge-danger');
        expect(
          within(screen.getByTestId('member-state-card')).getByText(row.labels[i], {
            exact: true,
          }),
        ).toBeVisible();
        const usable = screen.getByTestId('member-service-usable-now');
        expect(usable.textContent).toBe((row.usable ? usableText : unusableText)[i]);
        expect(usable).not.toHaveTextContent((row.usable ? usableText : unusableText)[j]);
        const reason = screen.getByTestId('member-state-reason');
        expect(reason.textContent).toBe(expectedReason(row, locale));
        expect(reason).not.toHaveTextContent(expectedReason(row, other));
        for (const [id, present] of [
          ['member-account-closed-banner', row.state === 'closed'],
          ['member-suspended-notice', row.state === 'suspended'],
          ['member-expired-notice', row.state === 'expired'],
        ] as const) {
          if (present) expect(screen.getByTestId(id)).toBeVisible();
          else expect(screen.queryByTestId(id)).not.toBeInTheDocument();
        }
        for (const action of row.actions)
          expect(screen.getByTestId(`member-next-${action}`)).toBeVisible();
        if (row.state === 'expired' || row.state === 'suspended') {
          const notice = screen.getByTestId(`member-${row.state}-notice`);
          expect(notice.textContent).toBe(expectedReason(row, locale));
          expect(notice).not.toHaveTextContent(expectedReason(row, other));
          const next = screen.getByTestId(`member-next-${row.actions[0]}`);
          const nextText =
            row.state === 'expired'
              ? [
                  '前往付款與訂閱頁面續訂服務。',
                  'Go to Billing & Subscription to renew service.',
                ]
              : (['請聯絡客服。', 'Contact support.'] as const);
          expect(next.textContent).toBe(nextText[i]);
          expect(next).not.toHaveTextContent(nextText[j]!);
          if (row.state === 'expired') {
            expect(screen.getByTestId('member-effective-end').textContent).toBe(
              formatLocaleWithUtc(end, locale),
            );
            expect(within(next).getByRole('link')).toHaveAttribute(
              'href',
              '/member/billing',
            );
          }
          await assertAccessible(container);
        }
      });
      it(`reason_${row.name}`, () => {
        expect(deriveMemberServiceReason(member(row.state, row.actions), locale)).toBe(
          expectedReason(row, locale),
        );
      });
    }

    it('NoDeviceLimitShowsCountAndConsistentAbsentDates', async () => {
      const me = member('suspended', ['contact_support']);
      me.limits.device_limit = null;
      me.limits.authorized_device_count = 0;
      me.entitlement.effective_service_end = null;
      me.entitlement.paid_through = null;
      renderMember(me, locale);
      await screen.findByTestId('member-entitlement-state-badge');
      const quota = screen.getByTestId('member-device-quota');
      expect(quota.textContent).toBe(
        locale === 'zh-TW' ? '已授權裝置：0 台' : 'Authorized devices: 0',
      );
      expect(quota).not.toHaveTextContent('/');
      for (const id of ['member-effective-end', 'member-paid-through']) {
        expect(screen.getByTestId(id).textContent).toBe(
          locale === 'zh-TW' ? '無' : 'None',
        );
        expect(screen.getByTestId(id)).not.toHaveTextContent(
          locale === 'zh-TW' ? 'None' : '無',
        );
      }
      if (locale === 'zh-TW')
        expect(screen.getByTestId('member-service-page')).not.toHaveTextContent(
          /Devices:|None|Normal speed|Loading service status\.\.\./,
        );
    });

    it('DeviceLimitUsesExistingTranslatedQuota', async () => {
      const me = member('active', ['free_device_slot']);
      me.limits.device_limit = 3;
      me.limits.authorized_device_count = 3;
      renderMember(me, locale);
      await screen.findByTestId('member-entitlement-state-badge');
      expect(screen.getByTestId('member-device-quota').textContent).toBe(
        locale === 'zh-TW'
          ? '已授權裝置名額使用量：3 / 3 台'
          : 'Authorized Device Quota: 3 / 3 devices',
      );
    });

    it('LoadingServiceStatusIsLocalized', async () => {
      let release = () => {};
      const ready = new Promise<void>((resolve) => {
        release = resolve;
      });
      renderMember(member('active', []), locale, '/member/service', ready);
      try {
        const loading = await screen.findByTestId('member-service-loading');
        expect(loading.textContent).toBe(
          locale === 'zh-TW' ? '正在載入服務狀態…' : 'Loading service status...',
        );
        expect(loading).not.toHaveTextContent(
          locale === 'zh-TW' ? 'Loading service status...' : '正在載入服務狀態…',
        );
      } finally {
        await act(async () => {
          release();
        });
      }
      await screen.findByTestId('member-entitlement-state-badge');
    });

    it('ContactSupportDoesNotMeanClosed_regression', async () => {
      renderMember(buildMemberSelfForPersona('member_suspended'), locale);
      await screen.findByTestId('member-entitlement-state-badge');
      expect(
        screen.queryByTestId('member-account-closed-banner'),
      ).not.toBeInTheDocument();
      expect(screen.getByTestId('member-suspended-notice')).toHaveTextContent(
        cases[7]!.reasons[i],
      );
      expect(screen.getByTestId('member-service-page')).not.toHaveTextContent(
        /所有裝置授權已撤銷|All device authorizations are revoked/,
      );
    });

    it('GraceIsUsable_regression', async () => {
      renderMember(buildMemberSelfForPersona('member_grace'), locale);
      expect(await screen.findByTestId('member-entitlement-state-badge')).toHaveClass(
        'badge-success',
      );
      expect(screen.getByTestId('member-service-usable-now').textContent).toBe(
        usableText[i],
      );
    });

    it('BillingUsesCorrectedSharedReason', async () => {
      const row = cases[5]!;
      renderMember(member(row.state, row.actions), locale, '/member/billing');
      expect(
        await screen.findByText(expectedReason(row, locale), { exact: true }),
      ).toBeVisible();
    });

    it('UnknownStateFailsClosed', async () => {
      // Deliberately simulate a newer server; the production Record stays exhaustive.
      const me = member('future_state' as EntitlementState, []);
      expect(deriveMemberServiceState(me).usable).toBe(false);
      renderMember(me, locale);
      expect(await screen.findByTestId('member-entitlement-state-badge')).toHaveClass(
        'badge-danger',
      );
      expect(screen.getByTestId('member-service-usable-now').textContent).toBe(
        unusableText[i],
      );
      expect(screen.getByTestId('member-entitlement-state-badge')).not.toHaveTextContent(
        'future_state',
      );
      expect(screen.getByTestId('member-state-reason').textContent).toBe(
        i === 0
          ? '目前無法確認服務資格，請重新載入或聯絡客服。'
          : 'Your entitlement could not be confirmed. Reload or contact support.',
      );
    });
  });
}

it.each(['expired', 'suspended', 'pending_payment'] as const)(
  'TestAT04_F404_EmptyActionsFallback_%s',
  async (state) => {
    renderMember(member(state, []), 'en');
    await screen.findByTestId('member-entitlement-state-badge');
    expect(
      screen.getByTestId(
        `member-next-${state === 'expired' ? 'renew_subscription' : state === 'suspended' ? 'contact_support' : 'complete_payment'}`,
      ),
    ).toBeVisible();
  },
);

it('TestAT04_F404_ServerActionsTakePrecedenceWithoutChangingState', async () => {
  renderMember(member('expired', ['contact_support']), 'en');
  expect(await screen.findByTestId('member-entitlement-state-badge')).toHaveAttribute(
    'data-state',
    'expired',
  );
  expect(screen.getByTestId('member-next-contact_support')).toBeVisible();
  expect(screen.queryByTestId('member-next-renew_subscription')).not.toBeInTheDocument();
  expect(screen.queryByTestId('member-account-closed-banner')).not.toBeInTheDocument();
});
