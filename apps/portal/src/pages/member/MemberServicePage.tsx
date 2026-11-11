import React from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  apiClient,
  deriveMemberServiceReason,
  deriveMemberServiceState,
  type NextAction,
} from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { useSession } from '../../context/SessionContext';
import { useI18n, type TranslationKey } from '../../i18n';
import { formatLocaleWithUtc } from '../../lib/datetime';

const nextSteps: Record<NextAction, { label: TranslationKey; to?: string }> = {
  complete_payment: {
    label: 'member.service.next.complete_payment',
    to: '/member/billing',
  },
  wait_for_payment_confirmation: {
    label: 'member.service.next.wait_for_payment_confirmation',
  },
  install_app: {
    label: 'member.service.next.install_app',
    to: '/member/getting-started',
  },
  register_device: {
    label: 'member.service.next.register_device',
    to: '/member/devices',
  },
  free_device_slot: {
    label: 'member.service.next.free_device_slot',
    to: '/member/devices',
  },
  renew_subscription: {
    label: 'member.service.next.renew_subscription',
    to: '/member/billing',
  },
  contact_support: { label: 'member.service.next.contact_support' },
};

export function MemberServicePage(): React.ReactElement {
  const { t, locale } = useI18n();
  const { persona } = useSession();

  const selfQuery = useQuery({
    queryKey: ['member', 'me', persona],
    queryFn: () => apiClient.getMemberSelf(),
  });

  const usageQuery = useQuery({
    queryKey: ['member', 'usage', persona],
    queryFn: () => apiClient.getMemberUsage(),
  });

  if (selfQuery.error) {
    return (
      <div data-testid="member-service-page">
        <ApiErrorAlert error={selfQuery.error} testId="member-service-error" />
      </div>
    );
  }

  if (selfQuery.isLoading || !selfQuery.data) {
    return (
      <div className="card" role="status" data-testid="member-service-loading">
        {t('member.service.loading')}
      </div>
    );
  }

  const me = selfQuery.data;
  const ent = me.entitlement;
  const usage = usageQuery.data;
  const stateView = deriveMemberServiceState(me);
  const isClosed = ent.state === 'closed';
  const actions = me.next_actions.length ? me.next_actions : [stateView.fallbackAction];
  const isFupAdjusted = usage?.fair_use.state === 'adjusted';
  const resetAt = me.fair_use_reset_at ?? usage?.fair_use.resets_at ?? '-';
  const planName =
    ent.plan?.display_name[locale] ??
    ent.plan?.display_name['zh-TW'] ??
    ent.plan?.code ??
    '-';
  const reasonText = deriveMemberServiceReason(me, locale);

  return (
    <div data-testid="member-service-page">
      <div className="page-header">
        <div>
          <h1>{t('member.service.title')}</h1>
          <p className="page-subtitle mono">
            {me.display_name} · {me.email ?? '-'}
          </p>
        </div>
        <span
          className={`badge ${stateView.usable ? 'badge-success' : 'badge-danger'}`}
          data-testid="member-entitlement-state-badge"
          data-state={ent.state}
        >
          {t(stateView.label)}
        </span>
      </div>

      {isClosed && (
        <div
          className="alert alert-danger"
          role="alert"
          data-testid="member-account-closed-banner"
        >
          {t('member.service.closedBanner')}
        </div>
      )}

      {(ent.state === 'suspended' || ent.state === 'expired') && (
        <div
          className="alert alert-danger"
          role="status"
          data-testid={`member-${ent.state}-notice`}
        >
          {reasonText}
        </div>
      )}

      {/* Exact required FUP throttling phrase (§11.6, §18.4) */}
      {isFupAdjusted && (
        <div
          className="alert alert-warning"
          role="status"
          data-testid="member-fup-throttled-banner"
        >
          {t('member.service.fupThrottledBanner', {
            resetAt,
          })}
        </div>
      )}

      {usageQuery.error && (
        <ApiErrorAlert error={usageQuery.error} testId="member-service-usage-error" />
      )}

      <div className="metrics-grid">
        <div className="metric-card" data-testid="member-state-card">
          <span className="stat-label">{t('member.service.stateLabel')}</span>
          <span
            className="metric-value"
            style={{
              fontSize: '1.35rem',
              color: stateView.usable ? 'var(--success)' : 'var(--danger)',
            }}
          >
            {t(stateView.label)}
          </span>
          <span className="metric-desc" data-testid="member-service-usable-now">
            {t(stateView.usable ? 'member.service.usable' : 'member.service.unusable')}
          </span>
          <span className="metric-desc" data-testid="member-state-reason">
            {reasonText}
          </span>
        </div>

        <div className="metric-card">
          <span className="stat-label">{t('member.service.planLabel')}</span>
          <span className="metric-value" style={{ fontSize: '1.35rem' }}>
            {planName}
          </span>
          <span className="metric-desc mono" data-testid="member-device-quota">
            {me.limits.device_limit === null
              ? t('member.devices.authorizedCount', {
                  used: me.limits.authorized_device_count,
                })
              : t('member.devices.quotaSummary', {
                  used: me.limits.authorized_device_count,
                  limit: me.limits.device_limit,
                })}
          </span>
        </div>

        <div className="metric-card">
          <span className="stat-label">{t('member.service.effectiveEndLabel')}</span>
          <span
            className="metric-value mono"
            style={{ fontSize: '1.05rem' }}
            data-testid="member-effective-end"
          >
            {ent.effective_service_end
              ? formatLocaleWithUtc(ent.effective_service_end, locale)
              : t('common.none')}
          </span>
          <span className="metric-desc mono">
            {t('member.service.paidThroughLabel')}:{' '}
            <span data-testid="member-paid-through">
              {ent.paid_through
                ? formatLocaleWithUtc(ent.paid_through, locale)
                : t('common.none')}
            </span>
          </span>
        </div>

        <div className="metric-card">
          <span className="stat-label">{t('member.service.nextResetLabel')}</span>
          <span className="metric-value mono" style={{ fontSize: '1.05rem' }}>
            {resetAt !== '-' ? formatLocaleWithUtc(resetAt, locale) : t('common.none')}
          </span>
          <span className="metric-desc" data-testid="member-service-speed">
            {usage?.fair_use.tier_name
              ? (usage.fair_use.tier_name[locale] ?? usage.fair_use.tier_name['zh-TW'])
              : t('member.service.normalSpeed')}
          </span>
        </div>
      </div>

      <section
        className="card"
        aria-labelledby="member-next-step-title"
        data-testid="member-service-next-steps"
      >
        <h2 id="member-next-step-title">{t('member.service.nextLabel')}</h2>
        {actions.map((action) => {
          const step = nextSteps[action];
          if (!step) return null;
          return (
            <p key={action} data-testid={`member-next-${action}`}>
              {step.to ? <Link to={step.to}>{t(step.label)}</Link> : t(step.label)}
            </p>
          );
        })}
      </section>

      <div className="card" style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap' }}>
        <Link to="/member/getting-started" className="btn btn-primary">
          {t('nav.member.gettingStarted')}
        </Link>
        <Link to="/member/exits" className="btn btn-secondary">
          {t('nav.member.exits')}
        </Link>
        <Link to="/member/devices" className="btn btn-secondary">
          {t('nav.member.devices')}
        </Link>
        <Link to="/member/usage" className="btn btn-secondary">
          {t('nav.member.usage')}
        </Link>
        <Link to="/member/billing" className="btn btn-secondary">
          {t('nav.member.billing')}
        </Link>
      </div>
    </div>
  );
}
