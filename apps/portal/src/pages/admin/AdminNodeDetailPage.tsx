import React, { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  ApiError,
  portalApi,
  toOperationResponse,
  useOperationPoller,
  type NodeItem,
  type Operation,
} from '../../api';
import { ConflictDialog } from '../../components/common/ConflictDialog';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n } from '../../i18n';
import { formatBpsMbps } from '../../lib/decimal';

export function AdminNodeDetailPage(): React.ReactElement {
  const { nodeId = '' } = useParams<{ nodeId: string }>();
  const { t } = useI18n();

  const [node, setNode] = useState<NodeItem | null>(null);
  const [nameInput, setNameInput] = useState('');
  const [regionInput, setRegionInput] = useState('');
  const [groupsInput, setGroupsInput] = useState('');
  const [upBpsInput, setUpBpsInput] = useState('');
  const [downBpsInput, setDownBpsInput] = useState('');
  const [notesInput, setNotesInput] = useState('');
  const [conflictError, setConflictError] = useState<ApiError | null>(null);
  const [loadError, setLoadError] = useState<ApiError | null>(null);

  const populateForm = useCallback((data: NodeItem) => {
    setNode(data);
    setLoadError(null);
    setNameInput(data.name);
    setRegionInput(data.region);
    setGroupsInput(data.member_groups.join(', '));
    setUpBpsInput(data.capacity.desired_up_bps);
    setDownBpsInput(data.capacity.desired_down_bps);
    setNotesInput(data.notes);
  }, []);

  const reloadNode = useCallback(async () => {
    if (!nodeId) return;
    try {
      const res = await portalApi.getAdminNode(nodeId);
      populateForm(res.data);
    } catch (err) {
      if (err instanceof ApiError) {
        setLoadError(err);
      }
    }
  }, [nodeId, populateForm]);

  const poller = useOperationPoller({
    scope: 'admin',
    onTerminal: () => {
      void reloadNode();
    },
  });

  useEffect(() => {
    void reloadNode();
  }, [reloadNode]);

  if (loadError) {
    return (
      <div className="alert alert-danger" role="alert">
        {loadError.message} ({loadError.code})
      </div>
    );
  }

  if (!node) {
    return <div className="card">Loading node detail...</div>;
  }

  const handleSaveSettings = async (simulateConflict = false) => {
    try {
      const res = await portalApi.patchAdminNode(
        node.node_id,
        {
          name: nameInput.trim(),
          region: regionInput.trim(),
          member_groups: groupsInput
            .split(',')
            .map((s) => s.trim())
            .filter(Boolean),
          desired_up_bps: upBpsInput.trim(),
          desired_down_bps: downBpsInput.trim(),
          notes: notesInput,
          simulate_conflict: simulateConflict,
        },
        { ifMatch: node.etag },
      );

      if (res.status === 202) {
        poller.startTracking(
          toOperationResponse({
            ...res,
            data: res.data as Operation,
          }),
        );
      } else {
        await reloadNode();
      }
    } catch (err) {
      if (err instanceof ApiError && err.isConflict) {
        setConflictError(err);
      }
    }
  };

  const handlePublicationChange = async (intent: 'published' | 'disabled') => {
    const opRes = await portalApi.putNodePublication(node.node_id, {
      publication_intent: intent,
    });
    poller.startTracking(opRes);
  };

  const handleRevokeIdentity = async () => {
    const opRes = await portalApi.revokeNode(
      node.node_id,
      'Admin triggered permanent machine identity revocation',
    );
    poller.startTracking(opRes);
  };

  const handleReloadAfterConflict = async () => {
    setConflictError(null);
    await reloadNode();
  };

  return (
    <div>
      <ConflictDialog
        conflictError={conflictError}
        onReloadAndDismiss={() => void handleReloadAfterConflict()}
      />

      <div style={{ marginBottom: 12 }}>
        <Link to="/admin/nodes">{t('admin.nodeDetail.back')}</Link>
      </div>

      <header className="page-header">
        <h1>{t('admin.nodeDetail.title', { name: node.name })}</h1>
        <p className="page-subtitle" data-testid="node-detail-meta">
          {t('admin.nodeDetail.meta', {
            nodeId: node.node_id,
            incarnationId: node.node_incarnation_id,
            etag: node.etag,
          })}
        </p>
      </header>

      <OperationBanner
        operation={poller.operation}
        requestId={poller.requestId}
        isPolling={poller.isPolling}
        onDismiss={poller.reset}
      />

      <section className="card" aria-labelledby="desired-observed-heading">
        <h2 id="desired-observed-heading">
          {t('admin.nodeDetail.desiredVsObserved.title')}
        </h2>
        <p style={{ fontSize: '0.88rem', color: 'var(--text-secondary)' }}>
          {t('admin.nodeDetail.lastAckAt', {
            time: node.config_generation.last_ack_at ?? '—',
          })}
        </p>
        <div className="table-wrapper">
          <table className="data-table" data-testid="desired-observed-table">
            <thead>
              <tr>
                <th scope="col">{t('admin.nodeDetail.field.setting')}</th>
                <th scope="col">{t('admin.nodeDetail.field.desired')}</th>
                <th scope="col">{t('admin.nodeDetail.field.observed')}</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>{t('admin.nodeDetail.field.publication')}</td>
                <td data-testid="desired-publication">{node.publication_intent}</td>
                <td data-testid="observed-publication">{node.publication_observed}</td>
              </tr>
              <tr>
                <td>{t('admin.nodeDetail.field.configGen')}</td>
                <td data-testid="desired-generation">{node.config_generation.desired}</td>
                <td data-testid="observed-generation">
                  {node.config_generation.applied}
                </td>
              </tr>
              <tr>
                <td>{t('admin.nodeDetail.field.release')}</td>
                <td>
                  {node.release.desired_version} (
                  <code>{node.release.desired_release_id}</code>)
                </td>
                <td>
                  {node.release.current_version} (last_good:{' '}
                  <code>{node.release.last_good_release_id}</code>, min:{' '}
                  <code>{node.release.minimum_serving_version}</code>)
                </td>
              </tr>
              <tr>
                <td>{t('admin.nodeDetail.field.memberGroups')}</td>
                <td>{node.member_groups.join(', ') || '—'}</td>
                <td>{node.observed_member_groups.join(', ') || '—'}</td>
              </tr>
              <tr>
                <td>{t('admin.nodeDetail.field.capacity')}</td>
                <td>
                  Up {formatBpsMbps(node.capacity.desired_up_bps)} / Down{' '}
                  {formatBpsMbps(node.capacity.desired_down_bps)}
                </td>
                <td>
                  Up {formatBpsMbps(node.capacity.applied_up_bps)} / Down{' '}
                  {formatBpsMbps(node.capacity.applied_down_bps)}
                </td>
              </tr>
              <tr>
                <td>{t('admin.nodeDetail.field.safetyAndIp')}</td>
                <td>internet-only-v1 (fixed mandatory)</td>
                <td>
                  safety: <strong>{node.safety_status}</strong> · exit IPv4:{' '}
                  <code>{node.observed_exit_ipv4}</code>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section className="card" aria-labelledby="pub-actions-heading">
        <h2 id="pub-actions-heading">{t('admin.nodeDetail.pubActions.title')}</h2>
        <div className="btn-row" style={{ marginBottom: 16 }}>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => void handlePublicationChange('published')}
            disabled={poller.isPolling || node.lifecycle_state === 'revoked'}
            data-testid="publish-node-btn"
          >
            {t('admin.nodeDetail.pubActions.publishBtn')}
          </button>
          <button
            type="button"
            className="btn"
            onClick={() => void handlePublicationChange('disabled')}
            disabled={poller.isPolling || node.lifecycle_state === 'revoked'}
            data-testid="disable-node-btn"
          >
            {t('admin.nodeDetail.pubActions.disableBtn')}
          </button>
        </div>

        <div className="alert alert-warning">
          <h3 style={{ margin: '0 0 6px', fontSize: '1rem' }}>
            {t('admin.nodeDetail.pubActions.revokeSectionTitle')}
          </h3>
          <p style={{ margin: '0 0 10px', fontSize: '0.9rem' }}>
            {t('admin.nodeDetail.pubActions.revokeDistinctNote')}
          </p>
          <button
            type="button"
            className="btn btn-danger"
            onClick={() => void handleRevokeIdentity()}
            disabled={poller.isPolling || node.lifecycle_state === 'revoked'}
            data-testid="revoke-node-identity-btn"
          >
            {t('admin.nodeDetail.pubActions.revokeBtn')}
          </button>
        </div>
      </section>

      <section className="card" aria-labelledby="edit-node-heading">
        <h2 id="edit-node-heading">{t('admin.nodeDetail.editForm.title')}</h2>
        <div className="form-grid">
          <div className="form-field">
            <label htmlFor="node-edit-name">{t('admin.nodeDetail.editForm.name')}</label>
            <input
              id="node-edit-name"
              type="text"
              value={nameInput}
              onChange={(e) => setNameInput(e.target.value)}
            />
          </div>
          <div className="form-field">
            <label htmlFor="node-edit-region">
              {t('admin.nodeDetail.editForm.region')}
            </label>
            <input
              id="node-edit-region"
              type="text"
              value={regionInput}
              onChange={(e) => setRegionInput(e.target.value)}
            />
          </div>
          <div className="form-field">
            <label htmlFor="node-edit-groups">
              {t('admin.nodeDetail.editForm.groups')}
            </label>
            <input
              id="node-edit-groups"
              type="text"
              value={groupsInput}
              onChange={(e) => setGroupsInput(e.target.value)}
            />
          </div>
          <div className="form-field">
            <label htmlFor="node-edit-up">{t('admin.nodeDetail.editForm.upBps')}</label>
            <input
              id="node-edit-up"
              type="text"
              value={upBpsInput}
              onChange={(e) => setUpBpsInput(e.target.value)}
            />
          </div>
          <div className="form-field">
            <label htmlFor="node-edit-down">
              {t('admin.nodeDetail.editForm.downBps')}
            </label>
            <input
              id="node-edit-down"
              type="text"
              value={downBpsInput}
              onChange={(e) => setDownBpsInput(e.target.value)}
            />
          </div>
          <div className="form-field">
            <label htmlFor="node-edit-notes">
              {t('admin.nodeDetail.editForm.notes')}
            </label>
            <input
              id="node-edit-notes"
              type="text"
              value={notesInput}
              onChange={(e) => setNotesInput(e.target.value)}
            />
          </div>
        </div>
        <div className="btn-row">
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => void handleSaveSettings(false)}
            data-testid="save-node-settings-btn"
          >
            {t('admin.nodeDetail.editForm.saveBtn')}
          </button>
          <button
            type="button"
            className="btn"
            onClick={() => void handleSaveSettings(true)}
            data-testid="simulate-412-btn"
          >
            {t('admin.nodeDetail.editForm.simulateConflictBtn')}
          </button>
        </div>
      </section>

      <section className="card" aria-labelledby="node-errors-heading">
        <h2 id="node-errors-heading">{t('admin.nodeDetail.errors.title')}</h2>
        {node.recent_errors.length === 0 ? (
          <p>{t('admin.nodeDetail.errors.empty')}</p>
        ) : (
          node.recent_errors.map((err) => (
            <details
              key={err.error_id}
              style={{ marginBottom: 10 }}
              open
              data-testid={`node-error-${err.error_code}`}
            >
              <summary style={{ cursor: 'pointer', fontWeight: 600 }}>
                <span className="badge badge-danger">{err.error_code}</span> ·{' '}
                {err.occurred_at} · stage: <code>{err.stage}</code>
              </summary>
              <div style={{ padding: '10px 12px', background: 'var(--bg-subtle)' }}>
                <p style={{ margin: '0 0 6px' }}>{err.message}</p>
                <div style={{ fontSize: '0.82rem' }}>
                  operation_id: <code>{err.operation_id}</code> ｜ request_id:{' '}
                  <code>{err.request_id}</code>
                </div>
              </div>
            </details>
          ))
        )}
      </section>

      <section className="card" aria-labelledby="node-history-heading">
        <h2 id="node-history-heading">{t('admin.nodeDetail.history.title')}</h2>
        <div className="table-wrapper">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Job ID / Batch ID</th>
                <th scope="col">Generation / Attempt</th>
                <th scope="col">State</th>
                <th scope="col">Releases (Prev → Current / Requested)</th>
                <th scope="col">Error Code</th>
                <th scope="col">Started / Finished</th>
              </tr>
            </thead>
            <tbody>
              {node.update_history.map((item) => (
                <tr key={item.job_id}>
                  <td>
                    <code>{item.job_id}</code>
                    <div style={{ fontSize: '0.78rem' }}>
                      batch: <code>{item.batch_id}</code>
                    </div>
                  </td>
                  <td>
                    gen {item.generation} (attempt #{item.attempt})
                  </td>
                  <td>
                    <span
                      className={
                        item.state === 'succeeded'
                          ? 'badge badge-success'
                          : 'badge badge-danger'
                      }
                    >
                      {item.state}
                    </span>
                  </td>
                  <td>
                    <code>{item.previous_release_id}</code> →{' '}
                    <code>{item.current_release_id}</code>
                  </td>
                  <td>{item.error_code ?? '—'}</td>
                  <td>
                    {item.started_at} → {item.finished_at ?? '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
