import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiClient, deriveUsageQualitySummary } from '../../api';
import { useSession } from '../../context/SessionContext';
import { useI18n } from '../../i18n';
import { formatBpsMbps, formatBytesDual } from '../../lib/decimal';

export function MemberUsagePage(): React.ReactElement {
  const { t, locale } = useI18n();
  const { persona } = useSession();

  const usageQuery = useQuery({
    queryKey: ['member', 'usage', persona],
    queryFn: () => apiClient.getMemberUsage(),
  });

  if (usageQuery.isLoading || !usageQuery.data) {
    return (
      <div className="card" role="status">
        Loading usage ledger...
      </div>
    );
  }

  const u = usageQuery.data;
  const qualitySummary = deriveUsageQualitySummary(u, locale);
  const tierLabel = u.fair_use.tier_name
    ? (u.fair_use.tier_name[locale] ?? u.fair_use.tier_name['zh-TW'])
    : 'Normal speed';

  return (
    <div data-testid="member-usage-page">
      <div className="page-header">
        <div>
          <h1>{t('member.usage.title')}</h1>
          <p className="page-subtitle mono">
            {t('member.usage.periodLabel', {
              period: `${u.period_start} → ${u.period_end}`,
            })}
          </p>
        </div>
        <span className="badge badge-neutral mono" data-testid="usage-as-of-badge">
          {t('member.usage.asOfLabel', { asOf: u.aggregate_as_of ?? '-' })}
        </span>
      </div>

      {/* Honest metering disclosure (§10.1, §18.4) */}
      <div className="alert alert-info" role="note" data-testid="honest-metering-banner">
        {t('member.usage.honestyBanner')}
      </div>

      {/* Gap / Estimated warning when quality !== ok (§10.7, §18.4) */}
      {qualitySummary.badge !== 'ok' && (
        <div
          className="alert alert-warning"
          role="status"
          data-testid="usage-gap-warning-banner"
        >
          <div style={{ fontWeight: 700 }}>
            {t('member.usage.gapWarningTitle', { quality: qualitySummary.badge })}
          </div>
          <div style={{ marginTop: '0.25rem' }}>
            {t('member.usage.gapWarningBody', {
              note: qualitySummary.note ?? '',
            })}
          </div>
        </div>
      )}

      {/* Exact required FUP throttling phrase when throttled (§11.6, §18.4) */}
      {u.fair_use.state !== 'normal' && (
        <div
          className="alert alert-warning"
          role="status"
          data-testid="usage-fup-throttled-banner"
        >
          {t('member.service.fupThrottledBanner', {
            resetAt: u.fair_use.resets_at ?? '-',
          })}
        </div>
      )}

      <div className="stat-grid">
        <div className="stat-card" data-testid="usage-total-card">
          <span className="stat-label">{t('member.usage.totalLabel')}</span>
          <span className="stat-value mono" style={{ fontSize: '1.35rem' }}>
            {formatBytesDual(u.totals.total_bytes)}
          </span>
          <span className="stat-subtext mono">raw: {u.totals.total_bytes} B</span>
        </div>

        <div className="stat-card">
          <span className="stat-label">{t('member.usage.uploadLabel')}</span>
          <span className="stat-value mono" style={{ fontSize: '1.25rem' }}>
            {formatBytesDual(u.totals.upload_bytes)}
          </span>
          <span className="stat-subtext mono">raw: {u.totals.upload_bytes} B</span>
        </div>

        <div className="stat-card">
          <span className="stat-label">{t('member.usage.downloadLabel')}</span>
          <span className="stat-value mono" style={{ fontSize: '1.25rem' }}>
            {formatBytesDual(u.totals.download_bytes)}
          </span>
          <span className="stat-subtext mono">raw: {u.totals.download_bytes} B</span>
        </div>

        <div className="stat-card" data-testid="usage-speed-status-card">
          <span className="stat-label">{t('member.usage.speedStatusLabel')}</span>
          <span
            className="stat-value"
            style={{
              fontSize: '1.25rem',
              color:
                u.fair_use.state !== 'normal'
                  ? 'var(--color-warning)'
                  : 'var(--color-success)',
            }}
          >
            {tierLabel}
          </span>
          <span className="stat-subtext mono">
            {t('member.usage.rateCapsLabel', {
              down: u.fair_use.rate
                ? formatBpsMbps(String(u.fair_use.rate.down_bps))
                : '-',
              up: u.fair_use.rate ? formatBpsMbps(String(u.fair_use.rate.up_bps)) : '-',
            })}
          </span>
          <span className="stat-subtext mono">
            {t('member.usage.nextResetLabel', { resetAt: u.fair_use.resets_at ?? '-' })}
          </span>
        </div>
      </div>

      <section className="card">
        <h2>Daily UTC Breakdown ({u.days.length} days)</h2>
        <div className="table-wrapper" style={{ marginTop: '0.75rem' }}>
          <table className="data-table" data-testid="usage-daily-table">
            <thead>
              <tr>
                <th>UTC Date</th>
                <th>Upload</th>
                <th>Download</th>
                <th>Total</th>
                <th>Quality</th>
              </tr>
            </thead>
            <tbody>
              {u.days.map((d) => {
                const dayQual = d.quality.gap
                  ? 'gap'
                  : d.quality.reconciliation_in_progress
                    ? 'reconciling'
                    : d.quality.estimated
                      ? 'estimated'
                      : 'ok';
                return (
                  <tr key={d.date}>
                    <td className="mono">{d.date}</td>
                    <td className="mono">{formatBytesDual(d.upload_bytes)}</td>
                    <td className="mono">{formatBytesDual(d.download_bytes)}</td>
                    <td className="mono">{formatBytesDual(d.total_bytes)}</td>
                    <td>
                      <span
                        className={`badge ${
                          dayQual === 'ok' ? 'badge-success' : 'badge-warning'
                        }`}
                      >
                        {dayQual}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
