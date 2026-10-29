import React, { useEffect, useMemo, useState } from 'react';
import { portalApi, type AuditEventItem } from '../../api';
import { useI18n } from '../../i18n';

export function AdminAuditPage(): React.ReactElement {
  const { t } = useI18n();
  const [events, setEvents] = useState<AuditEventItem[]>([]);
  const [resourceFilter, setResourceFilter] = useState('all');
  const [search, setSearch] = useState('');

  useEffect(() => {
    void portalApi.getAuditEvents().then((res) => setEvents(res.data.items));
  }, []);

  const filtered = useMemo(() => {
    return events.filter((ev) => {
      if (resourceFilter !== 'all' && ev.resource_type !== resourceFilter) {
        return false;
      }
      if (search.trim()) {
        const q = search.trim().toLowerCase();
        if (
          !ev.request_id.toLowerCase().includes(q) &&
          !ev.actor_email.toLowerCase().includes(q) &&
          !ev.action.toLowerCase().includes(q)
        ) {
          return false;
        }
      }
      return true;
    });
  }, [events, resourceFilter, search]);

  return (
    <div>
      <header className="page-header">
        <h1>{t('admin.audit.title')}</h1>
        <p className="page-subtitle">{t('admin.audit.subtitle')}</p>
      </header>

      <section className="card" aria-label="Audit filters">
        <div className="form-grid" style={{ marginBottom: 0 }}>
          <div className="form-field">
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
              <option value="update_batch">update_batch</option>
              <option value="member">member</option>
              <option value="policy">policy</option>
            </select>
          </div>
          <div className="form-field">
            <label htmlFor="audit-search-input">{t('admin.audit.filter.search')}</label>
            <input
              id="audit-search-input"
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
        </div>
      </section>

      <section className="card table-wrapper">
        <table className="data-table" data-testid="audit-table">
          <thead>
            <tr>
              <th scope="col">{t('admin.audit.col.time')}</th>
              <th scope="col">{t('admin.audit.col.actor')}</th>
              <th scope="col">{t('admin.audit.col.action')}</th>
              <th scope="col">{t('admin.audit.col.trace')}</th>
              <th scope="col">{t('admin.audit.col.result')}</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((ev) => (
              <tr key={ev.audit_id}>
                <td>
                  <code>{ev.occurred_at}</code>
                </td>
                <td>
                  <strong>{ev.actor_email}</strong> ({ev.actor_role})
                </td>
                <td>
                  <code>{ev.action}</code>
                  <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                    {ev.resource_type}: <code>{ev.resource_id}</code>
                  </div>
                </td>
                <td>
                  <div>
                    req: <code>{ev.request_id}</code>
                  </div>
                  {ev.operation_id && (
                    <div style={{ fontSize: '0.8rem' }}>
                      op: <code>{ev.operation_id}</code>
                    </div>
                  )}
                </td>
                <td>
                  <span className="badge badge-info">{ev.result}</span>
                  <div style={{ fontSize: '0.8rem', marginTop: 4 }}>
                    <code>{JSON.stringify(ev.sanitized_diff)}</code>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
