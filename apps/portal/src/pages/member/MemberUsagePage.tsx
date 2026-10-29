import React, { useEffect, useState } from 'react';
import { portalApi, type MemberItem } from '../../api';
import { useSession } from '../../context/SessionContext';
import { useI18n } from '../../i18n';
import { formatBpsMbps, formatBytesDual } from '../../lib/decimal';

export function MemberUsagePage(): React.ReactElement {
  const { t } = useI18n();
  const { persona } = useSession();
  const [usage, setUsage] = useState<MemberItem | null>(null);

  useEffect(() => {
    void portalApi.getMemberUsage().then((res) => setUsage(res.data));
  }, [persona]);

  if (!usage) {
    return <div className="card">Loading usage...</div>;
  }

  return (
    <div>
      <header className="page-header">
        <h1>{t('member.usage.title')}</h1>
        <p className="page-subtitle" data-testid="member-usage-as-of">
          {t('member.usage.asOfLabel', { asOf: usage.usage_aggregate_as_of })} ｜{' '}
          {t('member.usage.periodLabel', { period: usage.accounting_period })}
        </p>
      </header>

      <div className="alert alert-info" data-testid="member-usage-honesty-banner">
        {t('member.usage.honestyBanner')}
      </div>

      {usage.fup_tier_index > 0 && (
        <div
          className="alert alert-warning"
          role="status"
          data-testid="usage-fup-throttled-banner"
        >
          <strong>
            {t('member.service.fupThrottledBanner', {
              resetAt: usage.next_fup_reset_at,
            })}
          </strong>
        </div>
      )}

      {usage.usage_quality !== 'complete' && (
        <div className="alert alert-warning" role="alert" data-testid="usage-gap-warning">
          <strong>
            {t('member.usage.gapWarningTitle', { quality: usage.usage_quality })}
          </strong>
          <p style={{ margin: '6px 0 0' }}>
            {t('member.usage.gapWarningBody', {
              note: usage.usage_gap_note ?? usage.usage_quality,
            })}
          </p>
        </div>
      )}

      <div className="metrics-grid">
        <article className="metric-card" data-testid="usage-total-card">
          <div>
            <h2>{t('member.usage.totalLabel')}</h2>
            <div className="metric-value" style={{ fontSize: '1.45rem' }}>
              {formatBytesDual(usage.monthly_total_bytes)}
            </div>
            <p className="metric-desc">
              Decimal string: <code>{usage.monthly_total_bytes}</code> bytes
            </p>
          </div>
        </article>

        <article className="metric-card">
          <div>
            <h2>{t('member.usage.uploadLabel')}</h2>
            <div className="metric-value" style={{ fontSize: '1.35rem' }}>
              {formatBytesDual(usage.monthly_upload_bytes)}
            </div>
            <p className="metric-desc">
              Decimal string: <code>{usage.monthly_upload_bytes}</code> bytes
            </p>
          </div>
        </article>

        <article className="metric-card">
          <div>
            <h2>{t('member.usage.downloadLabel')}</h2>
            <div className="metric-value" style={{ fontSize: '1.35rem' }}>
              {formatBytesDual(usage.monthly_download_bytes)}
            </div>
            <p className="metric-desc">
              Decimal string: <code>{usage.monthly_download_bytes}</code> bytes
            </p>
          </div>
        </article>

        <article className="metric-card" data-testid="usage-speed-card">
          <div>
            <h2>{t('member.usage.speedStatusLabel')}</h2>
            <div className="metric-value" style={{ fontSize: '1.3rem' }}>
              {usage.fup_tier_name}
            </div>
            <p className="metric-desc">
              {t('member.usage.rateCapsLabel', {
                down: formatBpsMbps(usage.effective_down_bps),
                up: formatBpsMbps(usage.effective_up_bps),
              })}
            </p>
            <p className="metric-desc">
              {t('member.usage.nextResetLabel', { resetAt: usage.next_fup_reset_at })}
            </p>
          </div>
        </article>
      </div>
    </div>
  );
}
