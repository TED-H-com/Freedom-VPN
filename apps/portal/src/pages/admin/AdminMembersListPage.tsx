import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { apiClient, type EntitlementState } from '../../api';
import { useI18n } from '../../i18n';
import { formatBytesDual } from '../../lib/decimal';

export function AdminMembersListPage(): React.ReactElement {
  const { t } = useI18n();
  const [stateFilter, setStateFilter] = useState('all');

  const membersQuery = useQuery({
    queryKey: ['admin', 'members', stateFilter],
    queryFn: () =>
      apiClient.listMembers({
        entitlement_state:
          stateFilter !== 'all' ? (stateFilter as EntitlementState) : undefined,
      }),
  });

  const items = membersQuery.data?.items ?? [];

  return (
    <div data-testid="admin-members-list-page">
      <div className="page-header">
        <div>
          <h1>{t('admin.members.title')}</h1>
          <p className="page-subtitle">{t('admin.members.subtitle')}</p>
        </div>
      </div>

      <div className="card">
        <div className="form-group" style={{ maxWidth: '300px', marginBottom: '1rem' }}>
          <label htmlFor="member-state-filter">{t('admin.members.filter.state')}</label>
          <select
            id="member-state-filter"
            value={stateFilter}
            onChange={(e) => setStateFilter(e.target.value)}
          >
            <option value="all">{t('admin.nodes.filter.all')}</option>
            <option value="active">active</option>
            <option value="pending_payment">pending_payment</option>
            <option value="cancel_at_period_end">cancel_at_period_end</option>
            <option value="past_due">past_due</option>
            <option value="expired">expired</option>
            <option value="suspended">suspended</option>
          </select>
        </div>

        <div className="table-wrapper">
          <table className="data-table" data-testid="members-table">
            <thead>
              <tr>
                <th>{t('admin.members.col.member')}</th>
                <th>{t('admin.members.col.entitlement')}</th>
                <th>{t('admin.members.col.planAndEnd')}</th>
                <th>{t('admin.members.col.devices')}</th>
                <th>{t('admin.members.col.usage')}</th>
                <th>{t('admin.members.col.fupTier')}</th>
                <th>{t('admin.members.col.payment')}</th>
                <th>{t('admin.members.col.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((m) => (
                <tr key={m.member_id} data-testid={`member-row-${m.member_id}`}>
                  <td>
                    <div style={{ fontWeight: 600 }}>{m.display_name}</div>
                    <div
                      className="mono"
                      style={{ fontSize: '0.75rem', color: 'var(--color-text-muted)' }}
                    >
                      {m.member_id} · {m.email ?? '-'}
                    </div>
                  </td>
                  <td>
                    <span
                      className={`badge ${
                        m.entitlement.state === 'active'
                          ? 'badge-success'
                          : m.entitlement.state === 'suspended' ||
                              m.entitlement.state === 'expired'
                            ? 'badge-danger'
                            : 'badge-warning'
                      }`}
                    >
                      {m.entitlement.state}
                    </span>
                  </td>
                  <td className="mono" style={{ fontSize: '0.8rem' }}>
                    <div>{m.entitlement.plan?.code ?? '-'}</div>
                    <div>{m.entitlement.effective_service_end ?? '-'}</div>
                  </td>
                  <td className="mono">
                    {m.authorized_device_count} / {m.entitlement.device_limit ?? '-'}
                  </td>
                  <td className="mono">{formatBytesDual(m.usage.total_bytes)}</td>
                  <td>
                    <span
                      className={`badge ${
                        (m.usage.fair_use_tier_index ?? 0) > 0
                          ? 'badge-warning'
                          : 'badge-neutral'
                      }`}
                    >
                      {(m.usage.fair_use_tier_index ?? 0) > 0 ? 'adjusted' : 'normal'}
                    </span>
                  </td>
                  <td className="mono" style={{ fontSize: '0.75rem' }}>
                    <div>paid_through: {m.entitlement.paid_through ?? '-'}</div>
                    <div>suspended: {String(m.entitlement.admin_suspended)}</div>
                  </td>
                  <td>
                    <Link
                      to={`/admin/members/${m.member_id}`}
                      className="btn btn-secondary"
                      data-testid={`manage-member-${m.member_id}`}
                    >
                      {t('admin.members.manageBtn')}
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
