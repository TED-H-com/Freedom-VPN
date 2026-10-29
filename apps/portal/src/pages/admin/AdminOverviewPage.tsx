import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { portalApi, type AdminOverviewSummary } from '../../api';
import { useI18n } from '../../i18n';

export function AdminOverviewPage(): React.ReactElement {
  const { t } = useI18n();
  const [summary, setSummary] = useState<AdminOverviewSummary | null>(null);

  useEffect(() => {
    void portalApi.getAdminOverview().then((res) => setSummary(res.data));
  }, []);

  if (!summary) {
    return <div className="card">Loading overview...</div>;
  }

  return (
    <div>
      <header className="page-header">
        <h1>{t('admin.overview.title')}</h1>
        <p className="page-subtitle">{t('admin.overview.subtitle')}</p>
        <p style={{ fontSize: '0.86rem', color: 'var(--text-secondary)', marginTop: 6 }}>
          <strong data-testid="overview-as-of">
            {t('admin.overview.asOf', { asOf: summary.as_of })}
          </strong>
        </p>
      </header>

      <div className="metrics-grid">
        <article className="metric-card" data-testid="metric-available-exits">
          <div>
            <h2>{t('admin.overview.card.availableExits')}</h2>
            <div className="metric-value">
              {summary.available_exits_count} / {summary.total_nodes_count}
            </div>
            <p className="metric-desc">{t('admin.overview.card.availableExitsDesc')}</p>
          </div>
          <Link to="/admin/nodes" className="btn">
            {t('admin.overview.link.goNodes')}
          </Link>
        </article>

        <article className="metric-card" data-testid="metric-control-status">
          <div>
            <h2>{t('admin.overview.card.controlStatus')}</h2>
            <div className="metric-value">{summary.online_nodes_count}</div>
            <p className="metric-desc">
              {t('admin.overview.card.controlBreakdown', {
                online: summary.online_nodes_count,
                stale: summary.stale_nodes_count,
                offline: summary.offline_nodes_count,
              })}
            </p>
          </div>
          <Link to="/admin/nodes" className="btn">
            {t('admin.overview.link.goNodes')}
          </Link>
        </article>

        <article className="metric-card" data-testid="metric-safety-failed">
          <div>
            <h2>{t('admin.overview.card.safetyFailed')}</h2>
            <div
              className="metric-value"
              style={{
                color:
                  summary.safety_failed_nodes_count > 0
                    ? 'var(--danger)'
                    : 'var(--text-primary)',
              }}
            >
              {summary.safety_failed_nodes_count}
            </div>
            <p className="metric-desc">{t('admin.overview.card.safetyFailedDesc')}</p>
          </div>
          <Link to="/admin/nodes" className="btn">
            {t('admin.overview.link.goNodes')}
          </Link>
        </article>

        <article className="metric-card" data-testid="metric-enrollments">
          <div>
            <h2>{t('admin.overview.card.enrollmentsPending')}</h2>
            <div className="metric-value">
              {summary.pending_or_failed_enrollments_count}
            </div>
            <p className="metric-desc">
              {t('admin.overview.card.enrollmentsPendingDesc')}
            </p>
          </div>
          <Link to="/admin/nodes/new" className="btn">
            {t('nav.admin.addNode')}
          </Link>
        </article>

        <article className="metric-card" data-testid="metric-update-attention">
          <div>
            <h2>{t('admin.overview.card.updateAttention')}</h2>
            <div
              className="metric-value"
              style={{
                color:
                  summary.update_attention_nodes_count > 0
                    ? 'var(--warning)'
                    : 'var(--text-primary)',
              }}
            >
              {summary.update_attention_nodes_count}
            </div>
            <p className="metric-desc">{t('admin.overview.card.updateAttentionDesc')}</p>
          </div>
          <Link to="/admin/updates" className="btn">
            {t('admin.overview.link.goUpdates')}
          </Link>
        </article>

        <article className="metric-card" data-testid="metric-members">
          <div>
            <h2>{t('admin.overview.card.members')}</h2>
            <div className="metric-value">{summary.active_members_count}</div>
            <p className="metric-desc">
              {t('admin.overview.card.membersBreakdown', {
                active: summary.active_members_count,
                throttled: summary.throttled_members_count,
                inactive: summary.suspended_or_expired_members_count,
              })}
            </p>
          </div>
          <Link to="/admin/members" className="btn">
            {t('admin.overview.link.goMembers')}
          </Link>
        </article>
      </div>
    </div>
  );
}
