import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { portalApi, type NodeItem } from '../../api';
import { useI18n, type TranslationKey } from '../../i18n';

const SUMMARY_KEYS: Record<NodeItem['summary_code'], TranslationKey> = {
  online_serving: 'node.summary.online_serving',
  online_safety_failed_not_serving: 'node.summary.online_safety_failed_not_serving',
  online_disabled_not_serving: 'node.summary.online_disabled_not_serving',
  ready_unpublished_not_serving: 'node.summary.ready_unpublished_not_serving',
  offline_route_withdrawn_unconfirmed: 'node.summary.offline_route_withdrawn_unconfirmed',
  rolled_back_serving: 'node.summary.rolled_back_serving',
  revoked_not_serving: 'node.summary.revoked_not_serving',
};

export function AdminNodesListPage(): React.ReactElement {
  const { t } = useI18n();
  const [nodes, setNodes] = useState<NodeItem[]>([]);
  const [search, setSearch] = useState('');
  const [pubFilter, setPubFilter] = useState<string>('all');
  const [controlFilter, setControlFilter] = useState<string>('all');

  useEffect(() => {
    void portalApi.getAdminNodes().then((res) => setNodes(res.data.items));
  }, []);

  const filteredNodes = useMemo(() => {
    return nodes.filter((node) => {
      if (pubFilter !== 'all' && node.publication_intent !== pubFilter) {
        return false;
      }
      if (controlFilter !== 'all' && node.control_status !== controlFilter) {
        return false;
      }
      if (search.trim()) {
        const q = search.trim().toLowerCase();
        if (
          !node.name.toLowerCase().includes(q) &&
          !node.region.toLowerCase().includes(q) &&
          !node.node_id.toLowerCase().includes(q)
        ) {
          return false;
        }
      }
      return true;
    });
  }, [nodes, search, pubFilter, controlFilter]);

  return (
    <div>
      <header className="page-header">
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: 12,
          }}
        >
          <div>
            <h1>{t('admin.nodes.title')}</h1>
            <p className="page-subtitle">{t('admin.nodes.subtitle')}</p>
          </div>
          <div className="btn-row">
            <Link to="/admin/nodes/new" className="btn btn-primary">
              {t('admin.nodes.addBtn')}
            </Link>
            <Link to="/admin/updates" className="btn">
              {t('admin.nodes.fleetUpdateBtn')}
            </Link>
          </div>
        </div>
      </header>

      <section className="card" aria-label="Node filters">
        <div className="form-grid" style={{ marginBottom: 0 }}>
          <div className="form-field">
            <label htmlFor="node-search-input">{t('admin.nodes.filter.search')}</label>
            <input
              id="node-search-input"
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <div className="form-field">
            <label htmlFor="node-pub-filter">{t('admin.nodes.filter.publication')}</label>
            <select
              id="node-pub-filter"
              value={pubFilter}
              onChange={(e) => setPubFilter(e.target.value)}
            >
              <option value="all">{t('admin.nodes.filter.all')}</option>
              <option value="published">published</option>
              <option value="unpublished">unpublished</option>
              <option value="disabled">disabled</option>
            </select>
          </div>
          <div className="form-field">
            <label htmlFor="node-control-filter">{t('admin.nodes.filter.control')}</label>
            <select
              id="node-control-filter"
              value={controlFilter}
              onChange={(e) => setControlFilter(e.target.value)}
            >
              <option value="all">{t('admin.nodes.filter.all')}</option>
              <option value="online">online</option>
              <option value="stale">stale</option>
              <option value="offline">offline</option>
            </select>
          </div>
        </div>
      </section>

      <section className="card table-wrapper">
        <table className="data-table" data-testid="nodes-table">
          <thead>
            <tr>
              <th scope="col">{t('admin.nodes.col.node')}</th>
              <th scope="col">{t('admin.nodes.col.publication')}</th>
              <th scope="col">{t('admin.nodes.col.controlOnline')}</th>
              <th scope="col">{t('admin.nodes.col.dataAvailable')}</th>
              <th scope="col">{t('admin.nodes.col.policyValid')}</th>
              <th scope="col">{t('admin.nodes.col.updateComplete')}</th>
              <th scope="col">{t('admin.nodes.col.summary')}</th>
              <th scope="col">{t('admin.nodes.col.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {filteredNodes.map((node) => (
              <tr key={node.node_id} data-testid={`node-row-${node.name}`}>
                <td>
                  <strong>{node.name}</strong>
                  <div style={{ fontSize: '0.82rem', color: 'var(--text-secondary)' }}>
                    {node.region} · <code>{node.arch}</code>
                  </div>
                  <div style={{ fontSize: '0.78rem', color: 'var(--text-muted)' }}>
                    {node.notes}
                  </div>
                </td>
                <td>
                  <div>
                    Desired:{' '}
                    <span className="badge badge-info">{node.publication_intent}</span>
                  </div>
                  <div style={{ marginTop: 4 }}>
                    Observed:{' '}
                    <span
                      className={
                        node.publication_observed === 'active'
                          ? 'badge badge-success'
                          : node.publication_observed === 'applying'
                            ? 'badge badge-warning'
                            : 'badge badge-neutral'
                      }
                    >
                      {node.publication_observed}
                    </span>
                  </div>
                </td>
                <td>
                  <span
                    className={
                      node.control_status === 'online'
                        ? 'badge badge-success'
                        : node.control_status === 'stale'
                          ? 'badge badge-warning'
                          : 'badge badge-danger'
                    }
                  >
                    {node.control_status}
                  </span>
                </td>
                <td>
                  <span
                    className={
                      node.data_health === 'healthy' && node.serving_allowed
                        ? 'badge badge-success'
                        : node.data_health === 'failed'
                          ? 'badge badge-danger'
                          : 'badge badge-warning'
                    }
                  >
                    {node.data_health} (serving: {node.serving_allowed ? 'yes' : 'no'})
                  </span>
                </td>
                <td>
                  <span
                    className={
                      node.safety_status === 'passed'
                        ? 'badge badge-success'
                        : 'badge badge-danger'
                    }
                  >
                    {node.safety_status}
                  </span>
                  <div style={{ fontSize: '0.78rem', marginTop: 4 }}>
                    Gen: {node.config_generation.applied} /{' '}
                    {node.config_generation.desired}
                  </div>
                </td>
                <td>
                  <span
                    className={
                      node.update_state === 'succeeded' ||
                      node.update_state === 'already_current'
                        ? 'badge badge-success'
                        : node.update_state === 'rolled_back' ||
                            node.update_state === 'failed'
                          ? 'badge badge-danger'
                          : 'badge badge-warning'
                    }
                  >
                    {node.update_state}
                  </span>
                  <div style={{ fontSize: '0.78rem', marginTop: 4 }}>
                    Current: <code>{node.release.current_version}</code> / Target:{' '}
                    <code>{node.release.desired_version}</code>
                  </div>
                </td>
                <td data-testid={`node-summary-${node.name}`}>
                  {t(SUMMARY_KEYS[node.summary_code])}
                </td>
                <td>
                  <Link
                    to={`/admin/nodes/${node.node_id}`}
                    className="btn"
                    data-testid={`manage-node-${node.name}`}
                  >
                    {t('admin.nodes.viewDetail')}
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
