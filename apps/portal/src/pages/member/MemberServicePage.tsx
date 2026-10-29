import React from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { apiClient, deriveMemberServiceReason } from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { useSession } from '../../context/SessionContext';
import { useI18n } from '../../i18n';
import { formatLocaleWithUtc } from '../../lib/datetime';

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
      <div className="card" role="status">
        Loading service status...
      </div>
    );
  }

  const me = selfQuery.data;
  const ent = me.entitlement;
  const usage = usageQuery.data;
  const isActive =
    ent.state === 'active' ||
    ent.state === 'cancel_at_period_end' ||
    ent.state === 'past_due';
  const isClosed = ent.state === 'closed' || me.next_actions.includes('contact_support');
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
          className={`badge ${
            ent.state === 'active'
              ? 'badge-success'
              : ent.state === 'suspended' ||
                  ent.state === 'expired' ||
                  ent.state === 'closed'
                ? 'badge-danger'
                : 'badge-warning'
          }`}
          data-testid="member-entitlement-state-badge"
        >
          {ent.state}
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

      <div className="stat-grid">
        <div className="stat-card" data-testid="member-state-card">
          <span className="stat-label">{t('member.service.stateLabel')}</span>
          <span
            className="stat-value"
            style={{
              fontSize: '1.35rem',
              color: isActive ? 'var(--color-success)' : 'var(--color-danger)',
            }}
          >
            {ent.state}
          </span>
          <span className="stat-subtext" data-testid="member-state-reason">
            {reasonText}
          </span>
        </div>

        <div className="stat-card">
          <span className="stat-label">{t('member.service.planLabel')}</span>
          <span className="stat-value" style={{ fontSize: '1.35rem' }}>
            {planName}
          </span>
          <span className="stat-subtext mono">
            Devices: {me.limits.authorized_device_count} / {me.limits.device_limit}
          </span>
        </div>

        <div className="stat-card">
          <span className="stat-label">{t('member.service.effectiveEndLabel')}</span>
          <span
            className="stat-value mono"
            style={{ fontSize: '1.05rem' }}
            data-testid="member-effective-end"
          >
            {ent.effective_service_end
              ? formatLocaleWithUtc(ent.effective_service_end, locale)
              : 'None'}
          </span>
          <span className="stat-subtext mono">
            {t('member.service.paidThroughLabel')}:{' '}
            {ent.paid_through ? formatLocaleWithUtc(ent.paid_through, locale) : 'None'}
          </span>
        </div>

        <div className="stat-card">
          <span className="stat-label">{t('member.service.nextResetLabel')}</span>
          <span className="stat-value mono" style={{ fontSize: '1.05rem' }}>
            {resetAt !== '-' ? formatLocaleWithUtc(resetAt, locale) : '-'}
          </span>
          <span className="stat-subtext">
            {usage?.fair_use.tier_name
              ? (usage.fair_use.tier_name[locale] ?? usage.fair_use.tier_name['zh-TW'])
              : 'Normal speed'}
          </span>
        </div>
      </div>

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
