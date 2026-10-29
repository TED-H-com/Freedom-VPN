import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  apiClient,
  deriveNodeHealthDimensions,
  deriveNodeSummaryPhrase,
  type ControlPlaneStatus,
  type NodeSummaryItem,
  type PublicationIntent,
} from '../../api';
import { useI18n, type TranslationKey } from '../../i18n';

export function AdminNodesListPage(): React.ReactElement {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [pubFilter, setPubFilter] = useState<string>('all');
  const [controlFilter, setControlFilter] = useState<string>('all');

  const nodesQuery = useQuery({
    queryKey: ['admin', 'nodes', pubFilter, controlFilter],
    queryFn: () =>
      apiClient.listNodes({
        publication_intent:
          pubFilter !== 'all' ? (pubFilter as PublicationIntent) : undefined,
        control_status:
          controlFilter !== 'all' ? (controlFilter as ControlPlaneStatus) : undefined,
      }),
  });

  const allItems = nodesQuery.data?.items ?? [];
  const filteredItems = allItems.filter((node) => {
    if (!search.trim()) return true;
    const q = search.trim().toLowerCase();
    return (
      (node.name ?? '').toLowerCase().includes(q) ||
      (node.region ?? '').toLowerCase().includes(q) ||
      node.node_id.toLowerCase().includes(q)
    );
  });

  const renderDimensionBadge = (ok: boolean, labelTrue: string, labelFalse: string) => (
    <span className={`badge ${ok ? 'badge-success' : 'badge-danger'}`}>
      {ok ? labelTrue : labelFalse}
    </span>
  );

  const renderSummaryPhrase = (node: NodeSummaryItem) => {
    const phraseCode = deriveNodeSummaryPhrase(node);
    const key = `node.summary.${phraseCode}` as TranslationKey;
    const isServing =
      phraseCode === 'online_serving' || phraseCode === 'rolled_back_serving';
    const isWarning = phraseCode === 'rolled_back_serving';
    return (
      <span
        className={`badge ${
          isWarning ? 'badge-warning' : isServing ? 'badge-success' : 'badge-danger'
        }`}
        data-testid={`node-summary-${node.node_id}`}
      >
        {t(key)}
      </span>
    );
  };

  return (
    <div data-testid="admin-nodes-list-page">
      <div className="page-header">
        <div>
          <h1>{t('admin.nodes.title')}</h1>
          <p className="page-subtitle">{t('admin.nodes.subtitle')}</p>
        </div>
        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
          <Link to="/admin/nodes/new" className="btn btn-primary">
            + {t('admin.nodes.addBtn')}
          </Link>
          <Link to="/admin/updates" className="btn btn-secondary">
            {t('admin.nodes.fleetUpdateBtn')}
          </Link>
        </div>
      </div>

      <div className="card">
        <div className="form-grid" style={{ marginBottom: '1rem' }}>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label htmlFor="node-search">{t('admin.nodes.filter.search')}</label>
            <input
              id="node-search"
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Tokyo / tw-taipei / node_id"
            />
          </div>

          <div className="form-group" style={{ marginBottom: 0 }}>
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

          <div className="form-group" style={{ marginBottom: 0 }}>
            <label htmlFor="node-ctrl-filter">{t('admin.nodes.filter.control')}</label>
            <select
              id="node-ctrl-filter"
              value={controlFilter}
              onChange={(e) => setControlFilter(e.target.value)}
            >
              <option value="all">{t('admin.nodes.filter.all')}</option>
              <option value="online">online</option>
              <option value="stale">stale</option>
              <option value="offline">offline</option>
              <option value="revoked">revoked</option>
            </select>
          </div>
        </div>

        <div className="table-wrapper">
          <table className="data-table" data-testid="nodes-table">
            <thead>
              <tr>
                <th>{t('admin.nodes.col.node')}</th>
                <th>{t('admin.nodes.col.publication')}</th>
                <th>{t('admin.nodes.col.controlOnline')}</th>
                <th>{t('admin.nodes.col.dataAvailable')}</th>
                <th>{t('admin.nodes.col.policyValid')}</th>
                <th>{t('admin.nodes.col.updateComplete')}</th>
                <th>{t('admin.nodes.col.summary')}</th>
                <th>{t('admin.nodes.col.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {filteredItems.map((node) => {
                const health = deriveNodeHealthDimensions(node);
                return (
                  <tr key={node.node_id} data-testid={`node-row-${node.node_id}`}>
                    <td>
                      <div style={{ fontWeight: 600 }}>{node.name ?? node.node_id}</div>
                      <div
                        className="mono"
                        style={{ fontSize: '0.75rem', color: 'var(--color-text-muted)' }}
                      >
                        {node.node_id} · {node.region ?? '-'} · {node.lifecycle_state}
                      </div>
                    </td>
                    <td>
                      <div style={{ fontSize: '0.8rem' }}>
                        Desired:{' '}
                        <span className="badge badge-info">
                          {node.publication_intent}
                        </span>
                      </div>
                      <div style={{ fontSize: '0.8rem', marginTop: '0.2rem' }}>
                        Observed:{' '}
                        <span
                          className={`badge ${
                            node.publication_observed === 'active'
                              ? 'badge-success'
                              : 'badge-warning'
                          }`}
                        >
                          {node.publication_observed}
                        </span>
                      </div>
                    </td>
                    <td>
                      {renderDimensionBadge(
                        health.control_online,
                        `online (${node.control_status})`,
                        node.control_status,
                      )}
                    </td>
                    <td>
                      {renderDimensionBadge(
                        health.data_path_available,
                        'forwarding',
                        'stopped',
                      )}
                    </td>
                    <td>
                      {renderDimensionBadge(
                        health.policy_and_isolation_valid,
                        'safety: ok',
                        `safety: ${node.safety_status}`,
                      )}
                    </td>
                    <td>
                      <div>
                        {renderDimensionBadge(
                          health.update_complete,
                          'up-to-date',
                          'attention',
                        )}
                      </div>
                      <div
                        className="mono"
                        style={{ fontSize: '0.75rem', marginTop: '0.2rem' }}
                      >
                        {node.releases.current_release_id ?? '-'} →{' '}
                        {node.releases.desired_release_id ?? '-'}
                      </div>
                    </td>
                    <td>{renderSummaryPhrase(node)}</td>
                    <td>
                      <Link
                        to={`/admin/nodes/${node.node_id}`}
                        className="btn btn-secondary"
                        data-testid={`manage-node-${node.node_id}`}
                      >
                        {t('admin.nodes.viewDetail')}
                      </Link>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
