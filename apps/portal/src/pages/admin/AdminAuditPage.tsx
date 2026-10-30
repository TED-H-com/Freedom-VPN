import React, { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { apiClient, type ResourceType } from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { useI18n } from '../../i18n';

export function AdminAuditPage(): React.ReactElement {
  const { t } = useI18n();
  const [searchParams, setSearchParams] = useSearchParams();
  const initialResourceFilter = searchParams.get('resource_type') || 'all';
  const [resourceFilter, setResourceFilter] = useState(initialResourceFilter);
  const [search, setSearch] = useState('');
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const [cursorHistory, setCursorHistory] = useState<Array<string | undefined>>([]);

  useEffect(() => {
    const paramFilter = searchParams.get('resource_type') || 'all';
    if (paramFilter !== resourceFilter) {
      setResourceFilter(paramFilter);
      setCursor(undefined);
      setCursorHistory([]);
    }
  }, [searchParams, resourceFilter]);

  const auditQuery = useQuery({
    queryKey: ['admin', 'audit', resourceFilter, cursor],
    queryFn: () =>
      apiClient.listAuditEvents({
        resource_type:
          resourceFilter !== 'all' ? (resourceFilter as ResourceType) : undefined,
        cursor,
      }),
  });

  const allItems = auditQuery.data?.items ?? [];
  const nextCursor = auditQuery.data?.next_cursor ?? null;
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

      <ApiErrorAlert error={auditQuery.error} testId="admin-audit-error" />

      <div className="card">
        <div className="form-grid" style={{ marginBottom: '1rem' }}>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label htmlFor="audit-resource-filter">
              {t('admin.audit.filter.resource')}
            </label>
            <select
              id="audit-resource-filter"
              data-testid="audit-resource-filter"
              value={resourceFilter}
              onChange={(e) => {
                const next = e.target.value;
                setResourceFilter(next);
                setCursor(undefined);
                setCursorHistory([]);
                if (next === 'all') {
                  setSearchParams({});
                } else {
                  setSearchParams({ resource_type: next });
                }
              }}
            >
              <option value="all">{t('admin.nodes.filter.all')}</option>
              <option value="node">node</option>
              <option value="enrollment">enrollment</option>
              <option value="exit">exit</option>
              <option value="member">member</option>
              <option value="member_device">member_device</option>
              <option value="entitlement_grant">entitlement_grant</option>
              <option value="policy">policy</option>
              <option value="policy_definition">policy_definition</option>
              <option value="policy_revision">policy_revision</option>
              <option value="release">release</option>
              <option value="update_batch">update_batch</option>
              <option value="fleet_config">fleet_config</option>
              <option value="member_group">member_group</option>
              <option value="admin_identity">admin_identity</option>
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

        {(cursorHistory.length > 0 || nextCursor) && (
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginTop: '1rem',
            }}
            data-testid="audit-pagination"
          >
            <button
              type="button"
              className="btn btn-secondary"
              data-testid="audit-prev-page-btn"
              disabled={cursorHistory.length === 0}
              onClick={() => {
                const prev = cursorHistory[cursorHistory.length - 1];
                setCursorHistory((h) => h.slice(0, -1));
                setCursor(prev);
              }}
            >
              {t('common.pagination.prev')}
            </button>
            <span className="mono" style={{ fontSize: '0.8rem' }}>
              cursor: {cursor ?? 'first'}
            </span>
            <button
              type="button"
              className="btn btn-secondary"
              data-testid="audit-next-page-btn"
              disabled={!nextCursor}
              onClick={() => {
                if (nextCursor) {
                  setCursorHistory((h) => [...h, cursor]);
                  setCursor(nextCursor);
                }
              }}
            >
              {t('common.pagination.next')}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
