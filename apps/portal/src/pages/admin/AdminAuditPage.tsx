import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiClient, type ResourceType } from '../../api';
import { useI18n } from '../../i18n';

export function AdminAuditPage(): React.ReactElement {
  const { t } = useI18n();
  const [resourceFilter, setResourceFilter] = useState('all');
  const [search, setSearch] = useState('');

  const auditQuery = useQuery({
    queryKey: ['admin', 'audit', resourceFilter],
    queryFn: () =>
      apiClient.listAuditEvents({
        resource_type:
          resourceFilter !== 'all' ? (resourceFilter as ResourceType) : undefined,
      }),
  });

  const allItems = auditQuery.data?.items ?? [];
  const filtered = allItems.filter((evt) => {
    if (!search.trim()) return true;
    const q = search.trim().toLowerCase();
    return (
      (evt.request_id ?? '').toLowerCase().includes(q) ||
      (evt.actor.id ?? '').toLowerCase().includes(q) ||
      evt.action.toLowerCase().includes(q) ||
      evt.resource.id.toLowerCase().includes(q)
    );
  });

  return (
    <div data-testid="admin-audit-page">
      <div className="page-header">
        <div>
          <h1>{t('admin.audit.title')}</h1>
          <p className="page-subtitle">{t('admin.audit.subtitle')}</p>
        </div>
      </div>

      <div className="card">
        <div className="form-grid" style={{ marginBottom: '1rem' }}>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label htmlFor="audit-resource-filter">
              {t('admin.audit.filter.resource')}
            </label>
            <select
              id="audit-resource-filter"
              value={resourceFilter}
              onChange={(e) => setResourceFilter(e.target.value)}
            >
              <option value="all">{t('admin.nodes.filter.all')}</option>
              <option value="node">node</option>
              <option value="enrollment">enrollment</option>
              <option value="member">member</option>
              <option value="policy">policy</option>
              <option value="update_batch">update_batch</option>
            </select>
          </div>

          <div className="form-group" style={{ marginBottom: 0 }}>
            <label htmlFor="audit-search">{t('admin.audit.filter.search')}</label>
            <input
              id="audit-search"
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
        </div>

        <div className="table-wrapper">
          <table className="data-table" data-testid="audit-table">
            <thead>
              <tr>
                <th>{t('admin.audit.col.time')}</th>
                <th>{t('admin.audit.col.actor')}</th>
                <th>{t('admin.audit.col.action')}</th>
                <th>{t('admin.audit.col.trace')}</th>
                <th>{t('admin.audit.col.result')}</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((evt) => (
                <tr
                  key={evt.audit_event_id}
                  data-testid={`audit-row-${evt.audit_event_id}`}
                >
                  <td className="mono" style={{ fontSize: '0.8rem' }}>
                    {evt.occurred_at}
                  </td>
                  <td>
                    <div className="mono">{evt.actor.id}</div>
                    <span className="badge badge-neutral">{evt.actor.type}</span>
                  </td>
                  <td>
                    <div style={{ fontWeight: 600 }}>{evt.action}</div>
                    <div className="mono" style={{ fontSize: '0.75rem' }}>
                      {evt.resource.type}:{evt.resource.id}
                    </div>
                  </td>
                  <td className="mono" style={{ fontSize: '0.75rem' }}>
                    <div>req: {evt.request_id}</div>
                    <div>op: {evt.operation_id ?? '-'}</div>
                  </td>
                  <td>
                    <span
                      className={`badge ${
                        evt.result === 'succeeded' ? 'badge-success' : 'badge-danger'
                      }`}
                    >
                      {evt.result}
                    </span>
                    <div
                      className="mono"
                      style={{
                        fontSize: '0.75rem',
                        marginTop: '0.25rem',
                        color: 'var(--color-text-muted)',
                      }}
                    >
                      {JSON.stringify(evt.changes?.before ?? null)} →{' '}
                      {JSON.stringify(evt.changes?.after ?? null)}
                    </div>
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
