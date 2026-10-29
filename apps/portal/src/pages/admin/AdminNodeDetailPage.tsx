import React, { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  apiClient,
  ApiError,
  deriveNodeSummaryPhrase,
  type OperationResource,
} from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { ConflictDialog } from '../../components/common/ConflictDialog';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n, type TranslationKey } from '../../i18n';
import { formatBpsMbps } from '../../lib/decimal';

export function AdminNodeDetailPage(): React.ReactElement {
  const { nodeId = '' } = useParams<{ nodeId: string }>();
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const [activeOp, setActiveOp] = useState<OperationResource | null>(null);
  const [conflictError, setConflictError] = useState<ApiError | null>(null);
  const [actionError, setActionError] = useState<unknown>(null);
  const [savedNotice, setSavedNotice] = useState(false);

  const [name, setName] = useState('');
  const [region, setRegion] = useState('');
  const [groups, setGroups] = useState('');
  const [upBps, setUpBps] = useState('');
  const [downBps, setDownBps] = useState('');
  const [clearCapacity, setClearCapacity] = useState(false);
  const [notes, setNotes] = useState('');

  const nodeQuery = useQuery({
    queryKey: ['admin', 'node', nodeId],
    queryFn: () => apiClient.getNode(nodeId),
    enabled: Boolean(nodeId),
  });

  const node = nodeQuery.data?.data;
  const etag = nodeQuery.data?.etag ?? null;

  useEffect(() => {
    if (node) {
      setName(node.name ?? '');
      setRegion(node.region ?? '');
      setGroups(node.member_group_ids.join(', '));
      setUpBps(String(node.capacity?.up_bps ?? 1000000000));
      setDownBps(String(node.capacity?.down_bps ?? 1000000000));
      setClearCapacity(false);
      setNotes(node.notes ?? '');
    }
  }, [node]);

  if (nodeQuery.error) {
    return (
      <div data-testid="admin-node-detail-page">
        <ApiErrorAlert error={nodeQuery.error} testId="node-detail-error" />
      </div>
    );
  }

  if (nodeQuery.isLoading || !node) {
    return (
      <div className="card" role="status">
        Loading node detail...
      </div>
    );
  }

  const handlePublish = async () => {
    setActionError(null);
    try {
      const op = await apiClient.publishNode(
        node.node_id,
        etag ?? undefined,
        `publish:${node.node_id}:${node.resource_version}`,
      );
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        setConflictError(err);
      } else {
        setActionError(err);
      }
    }
  };

  const handleUnpublish = async () => {
    setActionError(null);
    try {
      const op = await apiClient.unpublishNode(
        node.node_id,
        etag ?? undefined,
        `unpublish:${node.node_id}:${node.resource_version}`,
      );
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        setConflictError(err);
      } else {
        setActionError(err);
      }
    }
  };

  const handleDisable = async () => {
    setActionError(null);
    try {
      const op = await apiClient.disableNode(
        node.node_id,
        etag ?? undefined,
        `disable:${node.node_id}:${node.resource_version}`,
      );
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        setConflictError(err);
      } else {
        setActionError(err);
      }
    }
  };

  const handleRevoke = async () => {
    setActionError(null);
    try {
      const op = await apiClient.revokeNode(
        node.node_id,
        'Admin revoked identity from portal',
        etag ?? undefined,
        `revoke:${node.node_id}:${node.resource_version}`,
      );
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        setConflictError(err);
      } else {
        setActionError(err);
      }
    }
  };

  const handleRetire = async () => {
    setActionError(null);
    try {
      const op = await apiClient.retireNode(
        node.node_id,
        etag ?? undefined,
        `retire:${node.node_id}:${node.resource_version}`,
      );
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        setConflictError(err);
      } else {
        setActionError(err);
      }
    }
  };

  const handleSavePatch = async (overrideEtag?: string) => {
    setActionError(null);
    setSavedNotice(false);
    try {
      const parsedUp = Number.parseInt(upBps.trim(), 10);
      const parsedDown = Number.parseInt(downBps.trim(), 10);
      await apiClient.updateNode(
        node.node_id,
        clearCapacity
          ? {
              name,
              region,
              member_group_ids: groups
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean),
              clear_capacity: true,
              notes,
            }
          : {
              name,
              region,
              member_group_ids: groups
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean),
              capacity: {
                up_bps: Number.isNaN(parsedUp) ? -1 : parsedUp,
                down_bps: Number.isNaN(parsedDown) ? -1 : parsedDown,
                member_up_bps_cap: node.capacity?.member_up_bps_cap ?? null,
                member_down_bps_cap: node.capacity?.member_down_bps_cap ?? null,
              },
              notes,
            },
        overrideEtag ?? etag ?? undefined,
        `patch:${node.node_id}:${overrideEtag ?? etag ?? 'none'}:${Date.now()}`,
      );
      setSavedNotice(true);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        setConflictError(err);
      } else {
        setActionError(err);
      }
    }
  };

  const handleReloadConflict = async () => {
    setConflictError(null);
    await queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
  };

  const phraseCode = deriveNodeSummaryPhrase(node);
  const summaryKey = `node.summary.${phraseCode}` as TranslationKey;
  const isRevokedOrRetired =
    node.lifecycle_state === 'revoked' || node.lifecycle_state === 'retired';
  const latestError = node.recent_errors[0];

  return (
    <div data-testid="admin-node-detail-page">
      <div style={{ marginBottom: '0.75rem' }}>
        <Link to="/admin/nodes">{t('admin.nodeDetail.back')}</Link>
      </div>

      <div className="page-header">
        <div>
          <h1>{t('admin.nodeDetail.title', { name: node.name ?? node.node_id })}</h1>
          <p className="page-subtitle mono" data-testid="node-detail-meta">
            {t('admin.nodeDetail.meta', {
              nodeId: node.node_id,
              incarnationId: node.current_incarnation_id ?? '-',
              etag: etag ?? `"${node.resource_version}"`,
            })}
          </p>
        </div>
        <span className="badge badge-info" data-testid="node-detail-summary-badge">
          {t(summaryKey)}
        </span>
      </div>

      <ConflictDialog error={conflictError} onReload={handleReloadConflict} />

      <OperationBanner
        operation={activeOp}
        onDismiss={() => setActiveOp(null)}
        onTerminal={() => {
          void queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
        }}
      />

      {savedNotice && (
        <div
          className="alert alert-info"
          role="status"
          data-testid="saved-pending-apply-banner"
        >
          <strong>{t('op.savedPendingApply')}</strong> — Desired config_generation:{' '}
          <span className="mono">{node.config_generation.desired}</span> / Observed
          applied_config_generation:{' '}
          <span className="mono">{node.config_generation.applied ?? '-'}</span>
        </div>
      )}

      <ApiErrorAlert error={actionError} testId="node-action-error" />

      {/* Desired vs Observed comparison (§18.1, §18.3) */}
      <section className="card" data-testid="desired-vs-observed-section">
        <h2>{t('admin.nodeDetail.desiredVsObserved.title')}</h2>
        <p className="page-subtitle">
          {t('admin.nodeDetail.lastAckAt', {
            time: node.last_seen_at ?? 'Never',
          })}
        </p>

        <div className="table-wrapper" style={{ marginTop: '0.75rem' }}>
          <table className="data-table">
            <thead>
              <tr>
                <th>{t('admin.nodeDetail.field.setting')}</th>
                <th>{t('admin.nodeDetail.field.desired')}</th>
                <th>{t('admin.nodeDetail.field.observed')}</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>{t('admin.nodeDetail.field.publication')}</td>
                <td>
                  <span className="badge badge-info" data-testid="desired-publication">
                    {node.publication_intent}
                  </span>
                </td>
                <td>
                  <span
                    className={`badge ${
                      node.publication_observed === 'active'
                        ? 'badge-success'
                        : 'badge-warning'
                    }`}
                    data-testid="observed-publication"
                  >
                    {node.publication_observed} (peer: {node.netbird_peer_id ?? 'unknown'}
                    )
                  </span>
                </td>
              </tr>
              <tr>
                <td>{t('admin.nodeDetail.field.configGen')}</td>
                <td className="mono" data-testid="desired-config-gen">
                  {node.config_generation.desired}
                </td>
                <td className="mono" data-testid="observed-config-gen">
                  {node.config_generation.applied ?? '-'}
                </td>
              </tr>
              <tr>
                <td>{t('admin.nodeDetail.field.release')}</td>
                <td className="mono">{node.releases.desired_release_id ?? '-'}</td>
                <td className="mono">
                  current: {node.releases.current_release_id ?? '-'} / last_good:{' '}
                  {node.releases.last_good_release_id ?? '-'}
                </td>
              </tr>
              <tr>
                <td>{t('admin.nodeDetail.field.memberGroups')}</td>
                <td className="mono">{node.member_group_ids.join(', ')}</td>
                <td className="mono">
                  forwarding: {node.member_forwarding_observed ? 'true' : 'false'}
                </td>
              </tr>
              <tr>
                <td>{t('admin.nodeDetail.field.capacity')}</td>
                <td className="mono">
                  {node.capacity ? (
                    <>
                      Up: {formatBpsMbps(String(node.capacity.up_bps))} / Down:{' '}
                      {formatBpsMbps(String(node.capacity.down_bps))}
                    </>
                  ) : (
                    '-'
                  )}
                </td>
                <td className="mono">
                  Safety: {node.safety_status} · Exit IP:{' '}
                  {node.observed_exit_ipv4 ?? node.observed_exit_ipv6 ?? '-'}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Publication vs Revoke Machine Identity actions (§5.2, §18.3) */}
      <section className="card">
        <h2>{t('admin.nodeDetail.pubActions.title')}</h2>
        <div
          style={{
            display: 'flex',
            gap: '0.75rem',
            flexWrap: 'wrap',
            marginTop: '0.75rem',
            marginBottom: '1.25rem',
          }}
        >
          <button
            type="button"
            className="btn btn-primary"
            data-testid="publish-node-btn"
            onClick={() => void handlePublish()}
            disabled={isRevokedOrRetired}
          >
            {t('admin.nodeDetail.pubActions.publishBtn')}
          </button>
          <button
            type="button"
            className="btn btn-secondary"
            data-testid="unpublish-node-btn"
            onClick={() => void handleUnpublish()}
            disabled={isRevokedOrRetired}
          >
            {t('admin.nodeDetail.pubActions.unpublishBtn')}
          </button>
          <button
            type="button"
            className="btn btn-secondary"
            data-testid="disable-node-btn"
            onClick={() => void handleDisable()}
            disabled={isRevokedOrRetired}
          >
            {t('admin.nodeDetail.pubActions.disableBtn')}
          </button>
        </div>

        <div
          style={{
            borderTop: '1px solid var(--color-border)',
            paddingTop: '1rem',
          }}
        >
          <h3 style={{ color: 'var(--color-danger)' }}>
            {t('admin.nodeDetail.pubActions.revokeSectionTitle')}
          </h3>
          <p
            style={{
              fontSize: '0.85rem',
              color: 'var(--color-text-muted)',
              marginTop: '0.35rem',
              marginBottom: '0.75rem',
            }}
          >
            {t('admin.nodeDetail.pubActions.revokeDistinctNote')}
          </p>
          <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap' }}>
            <button
              type="button"
              className="btn btn-danger"
              data-testid="revoke-node-identity-btn"
              onClick={() => void handleRevoke()}
              disabled={isRevokedOrRetired}
            >
              {t('admin.nodeDetail.pubActions.revokeBtn')}
            </button>
            <button
              type="button"
              className="btn btn-danger"
              data-testid="retire-node-btn"
              onClick={() => void handleRetire()}
              disabled={node.lifecycle_state === 'retired'}
            >
              {t('admin.nodeDetail.pubActions.retireBtn')}
            </button>
          </div>
        </div>
      </section>

      {/* Edit Node Form with If-Match ETag */}
      <section className="card">
        <h2>{t('admin.nodeDetail.editForm.title')}</h2>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void handleSavePatch();
          }}
          style={{ marginTop: '0.75rem' }}
        >
          <div className="form-grid">
            <div className="form-group">
              <label htmlFor="edit-node-name">
                {t('admin.nodeDetail.editForm.name')}
              </label>
              <input
                id="edit-node-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label htmlFor="edit-node-region">
                {t('admin.nodeDetail.editForm.region')}
              </label>
              <input
                id="edit-node-region"
                value={region}
                onChange={(e) => setRegion(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label htmlFor="edit-node-groups">
                {t('admin.nodeDetail.editForm.groups')}
              </label>
              <input
                id="edit-node-groups"
                value={groups}
                onChange={(e) => setGroups(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label htmlFor="edit-node-up">{t('admin.nodeDetail.editForm.upBps')}</label>
              <input
                id="edit-node-up"
                className="mono"
                value={upBps}
                disabled={clearCapacity}
                onChange={(e) => setUpBps(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label htmlFor="edit-node-down">
                {t('admin.nodeDetail.editForm.downBps')}
              </label>
              <input
                id="edit-node-down"
                className="mono"
                value={downBps}
                disabled={clearCapacity}
                onChange={(e) => setDownBps(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label htmlFor="edit-node-notes">
                {t('admin.nodeDetail.editForm.notes')}
              </label>
              <input
                id="edit-node-notes"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
              />
            </div>
          </div>

          <div className="form-group" style={{ marginBottom: '0.75rem' }}>
            <label style={{ display: 'inline-flex', alignItems: 'center', gap: '0.5rem' }}>
              <input
                type="checkbox"
                data-testid="clear-capacity-checkbox"
                checked={clearCapacity}
                onChange={(e) => setClearCapacity(e.target.checked)}
              />
              {t('admin.nodeDetail.editForm.clearCapacity')}
            </label>
          </div>

          <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap' }}>
            <button
              type="submit"
              className="btn btn-primary"
              data-testid="save-node-patch-btn"
            >
              {t('admin.nodeDetail.editForm.saveBtn')}
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              data-testid="simulate-conflict-btn"
              onClick={() => void handleSavePatch('"0"')}
            >
              {t('admin.nodeDetail.editForm.simulateConflictBtn')}
            </button>
          </div>
        </form>
      </section>

      {/* Recent Errors & Diagnostics */}
      <section className="card">
        <h2>{t('admin.nodeDetail.errors.title')}</h2>
        {!latestError ? (
          <p style={{ color: 'var(--color-text-muted)', marginTop: '0.5rem' }}>
            {t('admin.nodeDetail.errors.empty')}
          </p>
        ) : (
          <div style={{ marginTop: '0.5rem' }}>
            <details style={{ marginBottom: '0.5rem' }}>
              <summary style={{ cursor: 'pointer', fontWeight: 600 }}>
                <span className="badge badge-danger" style={{ marginRight: '0.5rem' }}>
                  {latestError.code}
                </span>
                {latestError.message}
              </summary>
              <div
                className="mono"
                style={{
                  fontSize: '0.8rem',
                  padding: '0.5rem',
                  background: '#f8fafc',
                  marginTop: '0.35rem',
                }}
              >
                checks:{' '}
                {node.checks.map((c) => `${c.name}=${c.status}`).join(', ') || 'none'}
              </div>
            </details>
          </div>
        )}
      </section>
    </div>
  );
}
