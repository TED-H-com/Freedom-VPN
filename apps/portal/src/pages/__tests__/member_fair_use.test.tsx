import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';
import { AppRoutes } from '../../App';
import { type EntitlementState, type MemberSelf } from '../../api';
import { SessionProvider } from '../../context/SessionContext';
import { I18nProvider, type Locale } from '../../i18n';
import { formatLocaleWithUtc } from '../../lib/datetime';
import { buildMemberSelfForPersona, buildMemberUsageForPersona } from '../../mocks/data';
import { setMockPersona } from '../../mocks/handlers';
import { server } from '../../mocks/server';
import { assertAccessible } from '../../test/setup';

const periodEnd = '2026-11-01T00:00:00Z';
const overrideEnd = '2026-10-20T12:30:00Z';
const paths = ['/member/service', '/member/usage'] as const;
const noticeId = (path: string) =>
  path === '/member/service'
    ? 'member-fup-throttled-banner'
    : 'usage-fup-throttled-banner';

function renderPage(
  path: string,
  locale: Locale,
  usage = buildMemberUsageForPersona('member_throttled'),
  me = buildMemberSelfForPersona('member_active'),
) {
  setMockPersona('member_active');
  server.use(
    http.get('/api/v1/member/me', () => HttpResponse.json(me)),
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

// Independent copy expectations; dates use the existing locale + UTC formatter.
const notices = [
  {
    reason: 'fair_use_tier',
    dated: [
      '本期大量使用，速度已依公平使用政策調整；下次重置為 {date}',
      'High usage this period: speed has been adjusted under the Fair Use Policy; next reset is {date}',
    ],
    undated: [
      '本期大量使用，速度已依公平使用政策調整。',
      'High usage this period: speed has been adjusted under the Fair Use Policy.',
    ],
  },
  {
    reason: 'admin_override',
    dated: [
      '您的速度已由管理員暫時調整；將於 {date} 重新計算。',
      'An administrator has temporarily adjusted your speed; it will be recalculated at {date}.',
    ],
    undated: [
      '您的速度已由管理員暫時調整。',
      'An administrator has temporarily adjusted your speed.',
    ],
  },
  {
    reason: null,
    dated: [
      '您的速度目前已調整；將於 {date} 重新計算。',
      'Your speed is currently adjusted; it will be recalculated at {date}.',
    ],
    undated: ['您的速度目前已調整。', 'Your speed is currently adjusted.'],
  },
] as const;

const billingStates: Array<{
  state: EntitlementState;
  actions: MemberSelf['next_actions'];
  labels: [string, string];
}> = [
  {
    state: 'pending_payment',
    actions: ['complete_payment'],
    labels: ['待付款', 'Awaiting payment'],
  },
  {
    state: 'pending_payment',
    actions: ['wait_for_payment_confirmation'],
    labels: ['開通中', 'Confirmation in progress'],
  },
  { state: 'active', actions: [], labels: ['有效', 'Active'] },
  {
    state: 'cancel_at_period_end',
    actions: [],
    labels: ['有效（期末取消）', 'Active (ends at period end)'],
  },
  { state: 'grace', actions: [], labels: ['寬限期', 'Grace period'] },
  { state: 'past_due', actions: [], labels: ['付款逾期', 'Payment overdue'] },
  { state: 'expired', actions: [], labels: ['到期', 'Expired'] },
  { state: 'suspended', actions: [], labels: ['停權', 'Suspended'] },
  { state: 'closed', actions: [], labels: ['帳號已關閉', 'Account closed'] },
];

for (const [i, locale] of (['zh-TW', 'en'] as const).entries()) {
  describe(`TestAT19_FUP03_MemberFairUse_${locale}`, () => {
    for (const row of notices) {
      for (const dated of [true, false]) {
        it(`ReasonAndDate_${row.reason}_${dated ? 'dated' : 'undated'}_BothPages`, async () => {
          const usage = buildMemberUsageForPersona('member_throttled');
          usage.fair_use.reason = row.reason;
          usage.fair_use.resets_at = dated ? overrideEnd : null;
          const expected = (dated ? row.dated : row.undated)[i]!.replace(
            '{date}',
            formatLocaleWithUtc(overrideEnd, locale),
          );
          const rendered: string[] = [];
          for (const path of paths) {
            const view = renderPage(path, locale, usage);
            const notice = await screen.findByTestId(noticeId(path));
            expect(notice.textContent).toBe(expected);
            expect(notice).toHaveAttribute('role', 'status');
            expect(notice).not.toHaveTextContent(/……|\{date\}|\{resetAt\}/);
            if (!dated) expect(notice).not.toHaveTextContent(/2026|—| -/);
            if (row.reason !== 'fair_use_tier') {
              expect(notice).not.toHaveTextContent(
                /大量使用|High usage|公平使用政策|Fair Use Policy/,
              );
            }
            if (path === '/member/usage' && row.reason === 'fair_use_tier') {
              const speedCard = screen.getByTestId('usage-speed-status-card');
              expect(
                within(speedCard).getByText(
                  i === 0 ? '公平使用調整中' : 'Fair-use adjusted',
                  { exact: true },
                ),
              ).toBeVisible();
              expect(speedCard).not.toHaveTextContent(
                /管理員暫時調整|Temporarily adjusted by an administrator|速度已調整|Speed adjusted/,
              );
              expect(speedCard.querySelector('.badge')).toBeNull();
            }
            rendered.push(notice.textContent!);
            if (!dated) await assertAccessible(view.container);
            view.unmount();
          }
          expect(rendered[0]).toBe(rendered[1]);
        });
      }
    }

    for (const dated of [true, false]) {
      it(`UnknownReason_${dated ? 'dated' : 'undated'}_BothPages`, async () => {
        const usage = buildMemberUsageForPersona('member_throttled');
        usage.fair_use.reason =
          'reason_not_known_to_portal' as typeof usage.fair_use.reason;
        usage.fair_use.resets_at = dated ? overrideEnd : null;
        const expected = (dated ? notices[2].dated : notices[2].undated)[i]!.replace(
          '{date}',
          formatLocaleWithUtc(overrideEnd, locale),
        );
        const label = ['速度已調整', 'Speed adjusted'][i]!;
        const rendered: string[] = [];
        for (const path of paths) {
          const view = renderPage(path, locale, usage);
          const notice = await screen.findByTestId(noticeId(path));
          expect(notice.textContent).toBe(expected);
          expect(notice).toHaveAttribute('role', 'status');
          expect(notice).not.toHaveTextContent(/……|\{date\}|\{resetAt\}/);
          if (!dated) expect(notice).not.toHaveTextContent(/2026|—| -/);
          expect(notice).not.toHaveTextContent(
            /大量使用|High usage|公平使用政策|Fair Use Policy|管理員|administrator|暫時調整|temporarily adjusted/,
          );
          if (path === '/member/usage') {
            const card = within(screen.getByTestId('usage-speed-status-card'));
            const tier = card.getByText(
              i === 0 ? '公平使用調整中' : 'Fair-use adjusted',
              { exact: true },
            );
            const reasonLabel = card.getByText(label, { exact: true });
            expect(tier.nextElementSibling).toBe(reasonLabel);
            expect(reasonLabel).toBeVisible();
          }
          rendered.push(notice.textContent!);
          view.unmount();
        }
        expect(rendered[0]).toBe(rendered[1]);
      });
    }

    for (const reason of ['admin_override', null] as const) {
      it(`TierZeroReason_${reason}_BothPages`, async () => {
        const usage = buildMemberUsageForPersona('member_throttled');
        usage.fair_use.reason = reason;
        usage.fair_use.tier_index = 0;
        usage.fair_use.tier_name = { 'zh-TW': '正常速度', en: 'Normal speed' };
        usage.fair_use.resets_at = null;
        const label =
          reason === 'admin_override'
            ? ['管理員暫時調整', 'Temporarily adjusted by an administrator'][i]!
            : ['速度已調整', 'Speed adjusted'][i]!;
        for (const path of paths) {
          const view = renderPage(path, locale, usage);
          const notice = await screen.findByTestId(noticeId(path));
          expect(notice.textContent).toBe(
            notices[reason === 'admin_override' ? 1 : 2].undated[i],
          );
          expect(notice).not.toHaveTextContent(/大量使用|High usage/);
          if (path === '/member/usage') {
            const card = within(screen.getByTestId('usage-speed-status-card'));
            const tier = card.getByText(i === 0 ? '正常速度' : 'Normal speed', {
              exact: true,
            });
            const reasonLabel = card.getByText(label, { exact: true });
            expect(tier.nextElementSibling).toBe(reasonLabel);
            expect(reasonLabel).toBeVisible();
          }
          view.unmount();
        }
      });
    }

    it('OverrideResetUsesUsageAndAccountingCardUsesSelf', async () => {
      const usage = buildMemberUsageForPersona('member_throttled');
      usage.fair_use.reason = 'admin_override';
      usage.fair_use.resets_at = overrideEnd;
      const me = buildMemberSelfForPersona('member_active');
      me.fair_use_reset_at = periodEnd;
      renderPage('/member/service', locale, usage, me);
      const notice = await screen.findByTestId(noticeId('/member/service'));
      expect(notice.textContent).toBe(
        notices[1].dated[i]!.replace('{date}', formatLocaleWithUtc(overrideEnd, locale)),
      );
      expect(notice).not.toHaveTextContent(periodEnd);
      const card = screen.getByText(
        i === 0 ? '下一次公平使用 (FUP) 重置時間' : 'Next Fair-Use (FUP) Reset Time',
      ).parentElement!;
      expect(
        within(card).getByText(formatLocaleWithUtc(periodEnd, locale), {
          exact: true,
          normalizer: (text) => text,
        }),
      ).toBeVisible();
      expect(card).not.toHaveTextContent(overrideEnd);
    });

    it('AbsentAccountingResetDoesNotBorrowUsageReset', async () => {
      const usage = buildMemberUsageForPersona('member_throttled');
      usage.fair_use.resets_at = overrideEnd;
      const me = buildMemberSelfForPersona('member_active');
      me.fair_use_reset_at = null;
      renderPage('/member/service', locale, usage, me);
      const notice = await screen.findByTestId(noticeId('/member/service'));
      expect(notice.textContent).toBe(
        notices[0].dated[i]!.replace('{date}', formatLocaleWithUtc(overrideEnd, locale)),
      );
      const card = screen.getByText(
        i === 0 ? '下一次公平使用 (FUP) 重置時間' : 'Next Fair-Use (FUP) Reset Time',
      ).parentElement!;
      expect(
        within(card).getByText(i === 0 ? '無' : 'None', { exact: true }),
      ).toBeVisible();
      expect(card).not.toHaveTextContent(overrideEnd);
    });

    for (const state of ['normal', 'adjusted'] as const) {
      for (const applying of [true, false]) {
        for (const reason of ['fair_use_tier', 'admin_override', null] as const) {
          it(`Applying_${state}_${applying}_${reason}_BothPages`, async () => {
            const usage = buildMemberUsageForPersona('member_throttled');
            Object.assign(usage.fair_use, {
              state,
              applying,
              reason,
              tier_name: { 'zh-TW': '測試級距', en: 'Test tier' },
            });
            const applyingText =
              i === 0
                ? '速度設定正在套用到出口節點，部分節點可能尚未生效。'
                : 'Your speed setting is still being applied; some exit nodes may not have it yet.';
            for (const path of paths) {
              const view = renderPage(path, locale, usage);
              await screen.findByText(i === 0 ? '測試級距' : 'Test tier', {
                exact: true,
              });
              if (state === 'normal')
                expect(screen.queryByTestId(noticeId(path))).not.toBeInTheDocument();
              else expect(screen.getByTestId(noticeId(path))).toBeVisible();
              if (path === '/member/usage' && state === 'normal') {
                const speedCard = screen.getByTestId('usage-speed-status-card');
                expect(
                  within(speedCard).getByText(i === 0 ? '測試級距' : 'Test tier', {
                    exact: true,
                  }),
                ).toBeVisible();
                expect(speedCard).not.toHaveTextContent(
                  /管理員暫時調整|Temporarily adjusted by an administrator|速度已調整|Speed adjusted/,
                );
                expect(speedCard.querySelector('.badge')).toBeNull();
              }
              const applyingNotice = screen.queryByText(applyingText, { exact: true });
              if (applying) {
                expect(applyingNotice).toBeVisible();
                expect(applyingNotice).toHaveAttribute('role', 'status');
                if (state === 'adjusted')
                  expect(screen.getByTestId(noticeId(path)).nextElementSibling).toBe(
                    applyingNotice,
                  );
                if (reason === 'admin_override') await assertAccessible(view.container);
              } else expect(applyingNotice).not.toBeInTheDocument();
              view.unmount();
            }
          });
        }
      }
    }

    it('UsageFallbackNameAndAbsentCapsAndReset', async () => {
      const usage = buildMemberUsageForPersona('member_active');
      Object.assign(usage.fair_use, { tier_name: null, rate: null, resets_at: null });
      renderPage('/member/usage', locale, usage);
      const card = await screen.findByTestId('usage-speed-status-card');
      expect(
        within(card).getByText(i === 0 ? '正常速度' : 'Normal speed', { exact: true }),
      ).toBeVisible();
      expect(
        within(card).getByText(
          i === 0
            ? '目前同一出口合計速率上限：下行 無 ／ 上行 無'
            : 'Current Per-Exit Combined Rate Cap: Down None / Up None',
          { exact: true },
        ),
      ).toBeVisible();
      expect(
        within(card).getByText(i === 0 ? '下次重置時間：無' : 'Next Reset Time: None', {
          exact: true,
        }),
      ).toBeVisible();
      expect(card).not.toHaveTextContent(' -');
    });
  });

  describe(`TestAT03_EntitlementBilling_${locale}`, () => {
    it('CheckoutReturnDoesNotReplaceEntitlementLabel', async () => {
      const me = buildMemberSelfForPersona('member_pending_payment');
      me.next_actions = ['complete_payment'];
      renderPage('/member/billing?checkout=return', locale, undefined, me);
      const badge = await screen.findByTestId('billing-entitlement-state-badge');
      expect(badge.textContent).toBe(i === 0 ? '待付款' : 'Awaiting payment');
      expect(badge).toHaveAttribute('data-state', 'pending_payment');
      expect(screen.getByTestId('billing-entitlement-state').textContent).toBe(
        badge.textContent,
      );
      expect(screen.getByTestId('billing-return-pending-badge').textContent).toBe(
        i === 0 ? '確認中' : 'Verifying (確認中)',
      );
      expect(screen.getByTestId('checkout-return-pending-banner')).toBeVisible();
    });

    for (const row of billingStates) {
      it(`TranslatedState_${row.state}_${row.actions[0] ?? 'default'}`, async () => {
        const me = buildMemberSelfForPersona('member_active');
        me.entitlement.state = row.state;
        me.next_actions = row.actions;
        const billing = renderPage('/member/billing', locale, undefined, me);
        const stateRow = await screen.findByTestId('billing-entitlement-state');
        expect(stateRow.textContent).toBe(row.labels[i]);
        const badge = screen.getByTestId('billing-entitlement-state-badge');
        expect(badge.textContent).toBe(row.labels[i]);
        for (const element of [stateRow, badge])
          expect(element).toHaveAttribute('data-state', row.state);
        if (row.actions.includes('wait_for_payment_confirmation')) {
          expect(screen.getByTestId('billing-return-pending-badge').textContent).toBe(
            i === 0 ? '確認中' : 'Verifying (確認中)',
          );
        }
        await assertAccessible(billing.container);
        billing.unmount();
        renderPage('/member/service', locale, undefined, me);
        expect(
          (await screen.findByTestId('member-entitlement-state-badge')).textContent,
        ).toBe(row.labels[i]);
      });
    }

    for (const present of [true, false]) {
      it(`Dates_${present ? 'present' : 'absent'}`, async () => {
        const me = buildMemberSelfForPersona('member_active');
        me.entitlement.paid_through = present ? overrideEnd : null;
        me.entitlement.effective_service_end = present ? periodEnd : null;
        renderPage('/member/billing', locale, undefined, me);
        await screen.findByTestId('billing-entitlement-state');
        for (const [label, date] of [
          [
            i === 0
              ? '已付費期間截止 (paid_through)'
              : 'Billing Paid Through (paid_through)',
            overrideEnd,
          ],
          [
            i === 0
              ? '有效服務到期日 (effective_service_end)'
              : 'Effective Service End (effective_service_end)',
            periodEnd,
          ],
        ]) {
          const value = screen.getByText(label!, { exact: true }).nextElementSibling!;
          expect(value.textContent).toBe(
            present ? formatLocaleWithUtc(date, locale) : i === 0 ? '無' : 'None',
          );
        }
      });
    }
  });
}
