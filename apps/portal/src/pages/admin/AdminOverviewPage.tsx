import React from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { apiClient, buildAdminOverviewViewModel } from '../../api';
import { useI18n } from '../../i18n';
import { PRIMARY_BATCH_ID } from '../../mocks/data';

export function AdminOverviewPage(): React.ReactElement {
  const { t } = useI18n();
  const nodesQuery = useQuery({
    queryKey: ['admin', 'nodes', 'all', 'all'],
    queryFn: () => apiClient.listNodes(),
  });
  const membersQuery = useQuery({
    queryKey: ['admin', 'members', 'all'],
    queryFn: () => apiClient.listMembers(),
  });
  const batchQuery = useQuery({
    queryKey: ['admin', 'update-batch', PRIMARY_BATCH_ID],
    queryFn: () => apiClient.getUpdateBatch(PRIMARY_BATCH_ID),
  });

  if (
    nodesQuery.isLoading ||
    membersQuery.isLoading ||
    !nodesQuery.data ||
    !membersQuery.data
  ) {
    return (
      <div className="card" role="status">
        Loading overview...
      </div>
    );
  }

  const d = buildAdminOverviewViewModel(
    nodesQuery.data,
    membersQuery.data,
    batchQuery.data ?? null,
  );

  return (
    <div data-testid="admin-overview-page">
      <div className="page-header">
        <div>
          <h1>{t('admin.overview.title')}</h1>
          <p className="page-subtitle">{t('admin.overview.subtitle')}</p>
        </div>
        <span className="badge badge-neutral mono">
          {t('admin.overview.asOf', { asOf: d.as_of })}
        </span>
      </div>

      <div className="stat-grid">
        <div className="stat-card" data-testid="card-available-exits">
          <span className="stat-label">{t('admin.overview.card.availableExits')}</span>
          <span className="stat-value" style={{ color: 'var(--color-success)' }}>
            {d.available_exits_count} / {d.total_nodes_count}
          </span>
          <span className="stat-subtext">
            {t('admin.overview.card.availableExitsDesc')}
          </span>
          <Link to="/admin/nodes" style={{ fontSize: '0.8rem', marginTop: '0.4rem' }}>
            {t('admin.overview.link.goNodes')} →
          </Link>
        </div>

        <div className="stat-card" data-testid="card-control-status">
          <span className="stat-label">{t('admin.overview.card.controlStatus')}</span>
          <span className="stat-value">{d.control_online_count}</span>
          <span className="stat-subtext">
            {t('admin.overview.card.controlBreakdown', {
              online: d.control_online_count,
              stale: d.control_stale_count,
              offline: d.control_offline_count,
            })}
          </span>
        </div>

        <div className="stat-card" data-testid="card-safety-failed">
          <span className="stat-label">{t('admin.overview.card.safetyFailed')}</span>
          <span
            className="stat-value"
            style={{
              color:
                d.safety_failed_nodes > 0
                  ? 'var(--color-danger)'
                  : 'var(--color-success)',
            }}
          >
            {d.safety_failed_nodes}
          </span>
          <span className="stat-subtext">
            {t('admin.overview.card.safetyFailedDesc')}
          </span>
        </div>

        <div className="stat-card" data-testid="card-enrollments-pending">
          <span className="stat-label">
            {t('admin.overview.card.enrollmentsPending')}
          </span>
          <span className="stat-value">{d.enrollments_pending_or_failed}</span>
          <span className="stat-subtext">
            {t('admin.overview.card.enrollmentsPendingDesc')}
          </span>
        </div>

        <div className="stat-card" data-testid="card-update-attention">
          <span className="stat-label">{t('admin.overview.card.updateAttention')}</span>
          <span
            className="stat-value"
            style={{
              color:
                d.update_attention_nodes > 0
                  ? 'var(--color-warning)'
                  : 'var(--color-success)',
            }}
          >
            {d.update_attention_nodes}
          </span>
          <span className="stat-subtext">
            {t('admin.overview.card.updateAttentionDesc')}
          </span>
          <Link to="/admin/updates" style={{ fontSize: '0.8rem', marginTop: '0.4rem' }}>
            {t('admin.overview.link.goUpdates')} →
          </Link>
        </div>

        <div className="stat-card" data-testid="card-members">
          <span className="stat-label">{t('admin.overview.card.members')}</span>
          <span className="stat-value">{d.active_members_count}</span>
          <span className="stat-subtext">
            {t('admin.overview.card.membersBreakdown', {
              active: d.active_members_count,
              throttled: d.throttled_members_count,
              inactive: d.suspended_or_expired_members_count,
            })}
          </span>
          <Link to="/admin/members" style={{ fontSize: '0.8rem', marginTop: '0.4rem' }}>
            {t('admin.overview.link.goMembers')} →
          </Link>
        </div>
      </div>
    </div>
  );
}
