import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import userEvent from '@testing-library/user-event';
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
    reasons: ['您的服務資格有效。', 'Your service entitlement is active.'],
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
      '續費待確認，寬限期限為 {graceUntil}；有效服務期限至 {end}。',
      'Renewal is pending; the grace deadline is {graceUntil}. Your service entitlement runs until {end}.',
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

const steps = {
  complete_payment: {
    text: [
      '前往付款與訂閱頁面完成付款。',
      'Go to Billing & Subscription to complete payment.',
    ],
    to: '/member/billing',
    page: 'member-billing-page',
  },
  wait_for_payment_confirmation: {
    text: [
      '等待付款確認；確認前無法使用出口。',
      'Wait for payment confirmation; exit access is unavailable until confirmed.',
    ],
  },
  install_app: {
    text: ['前往開始使用頁面安裝 App。', 'Go to Getting Started to install the app.'],
    to: '/member/getting-started',
    page: 'member-getting-started-page',
  },
  register_device: {
    text: [
      '前往我的裝置頁面查看裝置授權。',
      'Go to My Devices to check device authorization.',
    ],
    to: '/member/devices',
    page: 'member-devices-page',
  },
  free_device_slot: {
    text: ['前往我的裝置頁面釋出裝置名額。', 'Go to My Devices to free a device slot.'],
    to: '/member/devices',
    page: 'member-devices-page',
  },
  renew_subscription: {
    text: [
      '前往付款與訂閱頁面續訂服務。',
      'Go to Billing & Subscription to renew service.',
    ],
    to: '/member/billing',
    page: 'member-billing-page',
  },
  contact_support: { text: ['請聯絡客服。', 'Contact support.'] },
} satisfies Record<
  MemberSelf['next_actions'][number],
  { text: [string, string]; to?: string; page?: string }
>;

function assertStep(action: keyof typeof steps, locale: Locale) {
  const step = steps[action];
  const element = screen.getByTestId(`member-next-${action}`);
  const index = locale === 'zh-TW' ? 0 : 1;
  expect(element.textContent).toBe(step.text[index]);
  expect(element).not.toHaveTextContent(step.text[1 - index]!);
  if ('to' in step)
    expect(within(element).getByRole('link')).toHaveAttribute('href', step.to);
  else expect(within(element).queryByRole('link')).not.toBeInTheDocument();
  return element;
}

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
        for (const action of row.actions) assertStep(action, locale);
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

    for (const action of Object.keys(steps) as Array<keyof typeof steps>) {
      it(`NextActionExactCopyAndRealRoute_${action}`, async () => {
        renderMember(member('active', [action]), locale);
        await screen.findByTestId('member-entitlement-state-badge');
        const element = assertStep(action, locale);
        const step = steps[action];
        if ('to' in step) {
          await userEvent.click(within(element).getByRole('link'));
          expect(await screen.findByTestId(step.page)).toBeVisible();
          expect(screen.queryByTestId('member-service-page')).not.toBeInTheDocument();
        }
      });
    }

    for (const variant of [
      'grant_without_subscription',
      'grant_with_past_due_subscription',
      'paid_subscription',
    ] as const) {
      for (const path of ['/member/service', '/member/billing']) {
        it(`ActiveReasonOnlyClaimsEntitlement_${variant}_${path}`, async () => {
          const me = buildMemberSelfForPersona('member_active');
          if (variant === 'grant_without_subscription') {
            me.subscription = null;
            me.entitlement.paid_through = null;
          }
          if (variant === 'grant_with_past_due_subscription') {
            me.subscription!.provider_status = 'past_due';
            me.subscription!.paid_through = '2026-09-28T00:00:00Z';
            me.entitlement.paid_through = me.subscription!.paid_through;
          }
          const expected = cases[2]!.reasons[i];
          renderMember(me, locale, path);
          const reason = await screen.findByText(expected, { exact: true });
          expect(reason).toBeVisible();
          expect(reason).not.toHaveTextContent(/訂閱|付款|subscription|payment|billing/i);
          expect(
            screen.queryByText(cases[2]!.reasons[j], { exact: true }),
          ).not.toBeInTheDocument();
          expect(
            screen.queryByText(
              /訂閱狀態正常|subscription is in order|billing is in order/i,
            ),
          ).not.toBeInTheDocument();
        });
      }
    }

    for (const unknown of ['future_action', '__proto__', 'constructor', 'toString']) {
      it(`UnknownActionDoesNotInventFallback_${unknown}`, async () => {
        const { container } = renderMember(
          member('expired', [unknown as MemberSelf['next_actions'][number]]),
          locale,
        );
        const unavailable = await screen.findByTestId('member-next-unavailable');
        const texts = [
          '目前無法顯示下一步，請重新載入頁面或聯絡客服。',
          'The next step cannot be shown. Reload the page or contact support.',
        ];
        expect(unavailable.textContent).toBe(texts[i]);
        expect(unavailable).not.toHaveTextContent(texts[j]!);
        expect(
          screen.queryByTestId('member-next-renew_subscription'),
        ).not.toBeInTheDocument();
        expect(
          within(screen.getByTestId('member-service-next-steps')).queryByRole('link'),
        ).not.toBeInTheDocument();
        if (unknown === 'future_action') await assertAccessible(container);
      });
    }

    it('MixedActionsShowOnlyKnownServerActions', async () => {
      renderMember(
        member('expired', [
          'constructor',
          'install_app',
          'future_action',
        ] as MemberSelf['next_actions']),
        locale,
      );
      await screen.findByTestId('member-entitlement-state-badge');
      assertStep('install_app', locale);
      expect(screen.queryByTestId('member-next-unavailable')).not.toBeInTheDocument();
      expect(
        screen.queryByTestId('member-next-renew_subscription'),
      ).not.toBeInTheDocument();
      expect(
        screen.getByTestId('member-service-next-steps').querySelectorAll('p'),
      ).toHaveLength(1);
    });

    it('EmptyActionsUseStateFallback', async () => {
      renderMember(member('expired', []), locale);
      await screen.findByTestId('member-entitlement-state-badge');
      assertStep('renew_subscription', locale);
      expect(screen.queryByTestId('member-next-unavailable')).not.toBeInTheDocument();
    });

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

    it('GraceNamesDeadlineAndExtendedEffectiveServiceEnd', async () => {
      // §9.2 Compute extends the connected interval through a future grant.
      const me = buildMemberSelfForPersona('member_grace');
      me.entitlement.effective_service_end = '2026-11-04T00:00:00Z';
      const deadline = formatLocaleWithUtc(me.entitlement.grace_until!, locale);
      const effectiveEnd = formatLocaleWithUtc(
        me.entitlement.effective_service_end,
        locale,
      );
      const expected =
        locale === 'zh-TW'
          ? `續費待確認，寬限期限為 ${deadline}；有效服務期限至 ${effectiveEnd}。`
          : `Renewal is pending; the grace deadline is ${deadline}. Your service entitlement runs until ${effectiveEnd}.`;
      renderMember(me, locale);
      expect((await screen.findByTestId('member-state-reason')).textContent).toBe(
        expected,
      );
      expect(deriveMemberServiceReason(me, locale)).toBe(expected);
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
