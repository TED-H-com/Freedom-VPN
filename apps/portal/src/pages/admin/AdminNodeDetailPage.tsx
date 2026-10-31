import React, { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  apiClient,
  ApiError,
  deriveNodeSummaryPhrase,
  isNodeAwaitingMemberGrants,
  isNodeLeaseExpired,
  isProvisionalLeaseWaitingCheck,
  type NodeDetailWithForwardingReasons,
  type OperationResource,
} from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { ConflictDialog } from '../../components/common/ConflictDialog';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n, type TranslationKey } from '../../i18n';
import { formatLocaleWithUtc } from '../../lib/datetime';
import { formatBpsMbps } from '../../lib/decimal';

export function AdminNodeDetailPage(): React.ReactElement {
  const { nodeId = '' } = useParams<{ nodeId: string }>();
  const { t, locale } = useI18n();
  const queryClient = useQueryClient();

  const [activeOp, setActiveOp] = useState<OperationResource | null>(null);
  const [conflictError, setConflictError] = useState<ApiError | null>(null);
  const [actionError, setActionError] = useState<unknown>(null);
  const [savedSnapshot, setSavedSnapshot] = useState<{
    desired: string;
    applied: string | null;
  } | null>(null);

  const [name, setName] = useState('');
  const [region, setRegion] = useState('');
  const [groups, setGroups] = useState('');
  const [upBps, setUpBps] = useState('');
  const [downBps, setDownBps] = useState('');
  const [clearCapacity, setClearCapacity] = useState(false);
  const [notes, setNotes] = useState('');
  const [selectedGroupIds, setSelectedGroupIds] = useState<string[]>([]);

  const nodeQuery = useQuery({
    queryKey: ['admin', 'node', nodeId],
    queryFn: () => apiClient.getNode(nodeId),
    enabled: Boolean(nodeId),
  });

  const adminGroupsQuery = useQuery({
    queryKey: ['admin', 'groups'],
    queryFn: () => apiClient.listAdminGroups({ limit: 100 }),
  });
  const availableGroups = adminGroupsQuery.data?.items ?? [];

  const node = nodeQuery.data?.data;
  const etag = nodeQuery.data?.etag ?? null;
  const nodeBatchId = node?.update?.batch_id ?? null;

  const nodeBatchQuery = useQuery({
    queryKey: ['admin', 'update-batch', nodeBatchId],
    queryFn: () => apiClient.getUpdateBatch(nodeBatchId!),
    enabled: Boolean(nodeBatchId),
  });
  const nodeUpdateTarget =
    nodeBatchQuery.data?.targets.find((target) => target.node_id === node?.node_id) ??
    null;

  const [syncedNodeKey, setSyncedNodeKey] = useState<string>('');
  const groupsKey = availableGroups.map((g) => g.group_id).join(',');
  const currentNodeKey = node
    ? `${node.node_id}:${node.resource_version}:${(node.group_ids ?? node.member_group_ids).join(',')}:${groupsKey}`
    : '';
  if (node && syncedNodeKey !== currentNodeKey) {
    setSyncedNodeKey(currentNodeKey);
    setName(node.name ?? '');
    setRegion(node.region ?? '');
    setGroups(node.member_group_ids.join(', '));
    const rawAttached = node.group_ids ?? node.member_group_ids ?? [];
    const normalizedIds = rawAttached.map((idOrCode) => {
      const byCode = availableGroups.find((g) => g.code === idOrCode);
      return byCode ? byCode.group_id : idOrCode;
    });
    setSelectedGroupIds(normalizedIds);
    setUpBps(String(node.capacity?.up_bps ?? 1000000000));
    setDownBps(String(node.capacity?.down_bps ?? 1000000000));
    setClearCapacity(false);
    setNotes(node.notes ?? '');
  }

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

  const resolveGroupLabel = (idOrCode: string): string => {
    const matched =
      availableGroups.find((g) => g.group_id === idOrCode) ??
      availableGroups.find((g) => g.code === idOrCode);
    if (matched) {
      return `${matched.code} (${matched.name})`;
    }
    return idOrCode;
  };

  const attachedGroupIds = node.group_ids ?? node.member_group_ids ?? [];

  const toggleGroupSelection = (groupId: string) => {
    setSelectedGroupIds((prev) =>
      prev.includes(groupId) ? prev.filter((id) => id !== groupId) : [...prev, groupId],
    );
  };

  const handlePublish = async (overrideEtag?: string) => {
    setActionError(null);
    try {
      const effectiveEtag = overrideEtag ?? etag ?? undefined;
      const op = await apiClient.publishNode(
        node.node_id,
        effectiveEtag,
        `publish:${node.node_id}:${effectiveEtag ?? node.resource_version}:${selectedGroupIds.join(',')}`,
        'Publish node to member service',
        selectedGroupIds,
      );
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
      await queryClient.invalidateQueries({ queryKey: ['admin', 'groups'] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        setConflictError(err);
      } else {
        setActionError(err);
      }
    }
  };

  const handleUnpublish = async (overrideEtag?: string) => {
    setActionError(null);
    try {
      const effectiveEtag = overrideEtag ?? etag ?? undefined;
      const op = await apiClient.unpublishNode(
        node.node_id,
        effectiveEtag,
        `unpublish:${node.node_id}:${effectiveEtag ?? node.resource_version}:${selectedGroupIds.join(',')}`,
        'Unpublish node from member service',
        selectedGroupIds,
      );
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
      await queryClient.invalidateQueries({ queryKey: ['admin', 'groups'] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        setConflictError(err);
      } else {
        setActionError(err);
      }
    }
  };

  const handleDisable = async (overrideEtag?: string) => {
    setActionError(null);
    try {
      const effectiveEtag = overrideEtag ?? etag ?? undefined;
      const op = await apiClient.disableNode(
        node.node_id,
        effectiveEtag,
        `disable:${node.node_id}:${effectiveEtag ?? node.resource_version}:${selectedGroupIds.join(',')}`,
        'Disable node from member service',
        selectedGroupIds,
      );
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
      await queryClient.invalidateQueries({ queryKey: ['admin', 'groups'] });
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
    setSavedSnapshot(null);
    try {
      const parsedUp = Number.parseInt(upBps.trim(), 10);
      const parsedDown = Number.parseInt(downBps.trim(), 10);
      const updated = await apiClient.updateNode(
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
      const nextDesired =
        'config_generation' in updated.data
          ? updated.data.config_generation.desired
          : (updated.data.desired_generation ?? node.config_generation.desired);
      const nextApplied =
        'config_generation' in updated.data
          ? (updated.data.config_generation.applied ?? null)
          : (node.config_generation.applied ?? null);
      if ('config_generation' in updated.data) {
        queryClient.setQueryData(['admin', 'node', nodeId], {
          data: updated.data,
          etag: updated.etag,
        });
      } else if (updated.etag) {
        queryClient.setQueryData(
          ['admin', 'node', nodeId],
          (prev: { data: typeof node; etag: string | null } | undefined) =>
            prev ? { ...prev, etag: updated.etag } : prev,
        );
      }
      await queryClient.invalidateQueries({ queryKey: ['admin', 'node', nodeId] });
      setSavedSnapshot({
        desired: nextDesired,
        applied: nextApplied,
      });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        setConflictError(err);
      } else {
        setActionError(err);
      }
    }
  };

  const handleReloadConflict = async () => {
    await queryClient.refetchQueries({ queryKey: ['admin', 'node', nodeId] });
    setConflictError(null);
    setSavedSnapshot(null);
  };

  const phraseCode = deriveNodeSummaryPhrase(node);
  const summaryKey = `node.summary.${phraseCode}` as TranslationKey;
  const isRevokedOrRetired =
    node.lifecycle_state === 'revoked' || node.lifecycle_state === 'retired';
  const latestError = node.recent_errors[0];
  const nodeWithReasons = node as NodeDetailWithForwardingReasons;
  const awaitingMemberGrants = isNodeAwaitingMemberGrants(nodeWithReasons);
  const leaseExpired = isNodeLeaseExpired(nodeWithReasons);

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
          {node.identity_conflict == null ? null : !node.identity_conflict
              .mapping_frozen && node.identity_conflict.conflict_count === 0 ? (
            <div
              style={{ marginTop: '0.25rem' }}
              data-testid="node-identity-conflict-section"
            >
              <span
                className="text-muted"
                style={{ fontSize: '0.82rem' }}
                data-testid="node-identity-conflict-none"
              >
                {t('admin.nodeDetail.identityConflict.none')}
              </span>
            </div>
          ) : (
            <div
              className="alert alert-danger"
              style={{ marginTop: '0.5rem', marginBottom: 0, padding: '0.6rem 0.75rem' }}
              data-testid="node-identity-conflict-section"
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  flexWrap: 'wrap',
                  gap: '0.5rem',
                }}
                data-testid="node-identity-conflict-frozen"
              >
                <span
                  className="badge badge-danger"
                  data-testid="node-identity-conflict-frozen-badge"
                >
                  {t('admin.nodeDetail.identityConflict.frozenBadge')}
                </span>
                <span
                  className="mono"
                  style={{ fontSize: '0.82rem' }}
                  data-testid="node-identity-conflict-count"
                >
                  {t('admin.nodeDetail.identityConflict.countLabel', {
                    count: String(node.identity_conflict.conflict_count),
                  })}
                </span>
                <span
                  className="mono"
                  style={{ fontSize: '0.82rem' }}
                  data-testid="node-identity-conflict-last-at"
                >
                  {t('admin.nodeDetail.identityConflict.lastAtLabel', {
                    time: formatLocaleWithUtc(
                      node.identity_conflict.last_conflict_at,
                      locale,
                    ),
                  })}
                </span>
              </div>
              <div
                style={{ marginTop: '0.35rem', fontSize: '0.85rem' }}
                data-testid="node-identity-conflict-resolution"
              >
                <a
                  href="#revoke-node-identity-btn"
                  data-testid="node-identity-conflict-revoke-link"
                  onClick={(e) => {
                    const el = document.getElementById('revoke-node-identity-btn');
                    if (el && typeof el.scrollIntoView === 'function') {
                      e.preventDefault();
                      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
                      el.focus();
                    }
                  }}
                  style={{
                    fontWeight: 600,
                    color: 'var(--color-danger)',
                    textDecoration: 'underline',
                  }}
                >
                  {t('admin.nodeDetail.identityConflict.resolutionText')}
                </a>
              </div>
            </div>
          )}
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

      {savedSnapshot && (
        <div
          className="alert alert-info"
          role="status"
          data-testid="saved-pending-apply-banner"
        >
          <strong>{t('op.savedPendingApply')}</strong> — Desired config_generation:{' '}
          <span className="mono">{savedSnapshot.desired}</span> / Observed
          applied_config_generation:{' '}
          <span className="mono">{savedSnapshot.applied ?? '-'}</span>
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

        {awaitingMemberGrants && (
          <div
            className="banner banner-info"
            role="status"
            style={{ marginTop: '0.75rem' }}
            data-testid="node-provisional-lease-waiting-banner"
          >
            <span
              className="badge badge-warning mono"
              style={{ marginRight: '0.5rem' }}
              data-testid="node-provisional-lease-waiting-badge"
            >
              awaiting_member_grants
            </span>
            <span data-testid="node-provisional-lease-waiting-copy">
              {t('admin.nodeDetail.provisionalLease.waitingCopy')}
            </span>
          </div>
        )}

        {leaseExpired && (
          <div
            className="alert alert-danger"
            role="alert"
            style={{ marginTop: '0.75rem' }}
            data-testid="node-lease-expired-failure-banner"
          >
            <span
              className="badge badge-danger mono"
              style={{ marginRight: '0.5rem' }}
              data-testid="node-lease-expired-failure-badge"
            >
              lease_expired
            </span>
            <span data-testid="node-lease-expired-failure-copy">
              {t('admin.nodeDetail.leaseExpired.failureCopy')}
            </span>
          </div>
        )}

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
                <td data-testid="node-attached-groups">
                  {attachedGroupIds.length === 0 ? (
                    <span className="text-muted">
                      {t('admin.nodeDetail.attachedGroups.none')}
                    </span>
                  ) : (
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.35rem' }}>
                      {attachedGroupIds.map((gid) => (
                        <span
                          key={gid}
                          className="badge badge-info mono"
                          data-testid={`node-attached-group-badge-${gid}`}
                        >
                          {resolveGroupLabel(gid)}
                        </span>
                      ))}
                    </div>
                  )}
                </td>
                <td className="mono" data-testid="observed-member-forwarding">
                  <div>
                    forwarding: {node.member_forwarding_observed ? 'true' : 'false'}
                  </div>
                  {nodeWithReasons.forwarding_block_reasons &&
                    nodeWithReasons.forwarding_block_reasons.length > 0 && (
                      <div
                        style={{
                          display: 'flex',
                          flexWrap: 'wrap',
                          gap: '0.35rem',
                          marginTop: '0.35rem',
                        }}
                        data-testid="node-forwarding-block-reasons"
                      >
                        {nodeWithReasons.forwarding_block_reasons.map((reason) =>
                          reason === 'awaiting_member_grants' ? (
                            <span
                              key={reason}
                              className="badge badge-warning"
                              data-testid="forwarding-block-reason-awaiting_member_grants"
                            >
                              {reason}:{' '}
                              {t('admin.nodeDetail.provisionalLease.waitingCopy')}
                            </span>
                          ) : reason === 'lease_expired' ? (
                            <span
                              key={reason}
                              className="badge badge-danger"
                              data-testid="forwarding-block-reason-lease_expired"
                            >
                              {reason}: {t('admin.nodeDetail.leaseExpired.failureCopy')}
                            </span>
                          ) : (
                            <span
                              key={reason}
                              className="badge badge-danger mono"
                              data-testid={`forwarding-block-reason-${reason}`}
                            >
                              {reason}
                            </span>
                          ),
                        )}
                      </div>
                    )}
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

      {/* Node Update Batch Target & Policy Evidence (§13.12, D-048) */}
      {(node.update || nodeUpdateTarget) && (
        <section className="card" data-testid="node-update-target-evidence-section">
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              flexWrap: 'wrap',
              gap: '0.5rem',
            }}
          >
            <div>
              <h2>{t('admin.nodeDetail.updateEvidenceTitle')}</h2>
              <p className="page-subtitle mono" style={{ fontSize: '0.8rem' }}>
                batch_id: {node.update?.batch_id ?? '-'} · generation:{' '}
                {nodeUpdateTarget?.generation ?? node.update?.generation ?? '0'} · state:{' '}
                {nodeUpdateTarget?.state ?? node.update?.state ?? '-'}
              </p>
            </div>
            {node.update?.batch_id && (
              <Link
                to={`/admin/updates?batch_id=${encodeURIComponent(node.update.batch_id)}`}
                className="btn btn-secondary"
                style={{ fontSize: '0.8rem' }}
                data-testid="node-detail-batch-link"
              >
                {node.update.batch_id} →
              </Link>
            )}
          </div>

          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))',
              gap: '0.75rem',
              marginTop: '0.75rem',
            }}
          >
            <div
              style={{
                padding: '0.6rem 0.75rem',
                background: '#f8fafc',
                borderRadius: 'var(--radius-sm)',
                border: '1px solid var(--color-border)',
              }}
              data-testid="node-detail-target-policy-gen"
            >
              <div
                style={{
                  fontSize: '0.75rem',
                  color: 'var(--color-text-muted)',
                  marginBottom: '0.25rem',
                }}
              >
                {t('admin.updates.targetDetailPolicyGenLabel')}
              </div>
              <div className="mono">
                {nodeUpdateTarget?.policy_generation != null ? (
                  <span className="badge badge-info">
                    {nodeUpdateTarget.policy_generation}
                  </span>
                ) : (
                  <span className="badge badge-warning">
                    {t('admin.updates.evidenceNotReported')}
                  </span>
                )}
              </div>
            </div>

            <div
              style={{
                padding: '0.6rem 0.75rem',
                background: '#f8fafc',
                borderRadius: 'var(--radius-sm)',
                border: '1px solid var(--color-border)',
              }}
              data-testid="node-detail-target-policy-refs"
            >
              <div
                style={{
                  fontSize: '0.75rem',
                  color: 'var(--color-text-muted)',
                  marginBottom: '0.25rem',
                }}
              >
                {t('admin.updates.targetDetailPolicyRefsLabel')}
              </div>
              <div className="mono" style={{ wordBreak: 'break-all' }}>
                {nodeUpdateTarget?.policy_refs != null ? (
                  <span className="badge badge-info">
                    {nodeUpdateTarget.policy_refs.length > 0
                      ? nodeUpdateTarget.policy_refs.join(', ')
                      : 'none (0)'}
                  </span>
                ) : (
                  <span className="badge badge-warning">
                    {t('admin.updates.evidenceNotReported')}
                  </span>
                )}
              </div>
            </div>
          </div>
        </section>
      )}

      {/* Publication vs Revoke Machine Identity actions (§5.2, §18.3, D-057) */}
      <section className="card">
        <h2>{t('admin.nodeDetail.pubActions.title')}</h2>

        <div
          style={{
            marginTop: '0.75rem',
            padding: '0.75rem',
            background: '#f8fafc',
            borderRadius: 'var(--radius-sm)',
            border: '1px solid var(--color-border)',
          }}
          data-testid="node-publication-groups-editor"
        >
          <div style={{ fontWeight: 600, fontSize: '0.9rem', marginBottom: '0.35rem' }}>
            {t('admin.nodeDetail.attachedGroups.title')}
          </div>
          <div
            style={{
              fontSize: '0.82rem',
              color: 'var(--color-text-muted)',
              marginBottom: '0.5rem',
            }}
          >
            {t('admin.nodeDetail.attachedGroups.selectLabel')}
          </div>
          {availableGroups.length === 0 ? (
            <div className="text-muted" style={{ fontSize: '0.85rem' }}>
              {t('admin.nodeDetail.attachedGroups.none')}
            </div>
          ) : (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.75rem' }}>
              {availableGroups.map((g) => (
                <label
                  key={g.group_id}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '0.4rem',
                    fontSize: '0.85rem',
                    cursor: 'pointer',
                  }}
                >
                  <input
                    type="checkbox"
                    data-testid={`pub-group-checkbox-${g.code}`}
                    checked={selectedGroupIds.includes(g.group_id)}
                    onChange={() => toggleGroupSelection(g.group_id)}
                    disabled={isRevokedOrRetired}
                  />
                  <span className="mono" style={{ fontWeight: 600 }}>
                    {g.code}
                  </span>
                  <span>({g.name})</span>
                </label>
              ))}
            </div>
          )}
        </div>

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
          <button
            type="button"
            className="btn btn-secondary"
            data-testid="simulate-pub-conflict-btn"
            onClick={() => void handlePublish('"0"')}
            disabled={isRevokedOrRetired}
          >
            {t('admin.nodeDetail.attachedGroups.simulateConflictBtn')}
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
              id="revoke-node-identity-btn"
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
            <label
              style={{ display: 'inline-flex', alignItems: 'center', gap: '0.5rem' }}
            >
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

      {/* Node Checks Detail Table (D-058) */}
      <section className="card" data-testid="node-checks-section">
        <h2>{t('admin.nodeDetail.checks.title')}</h2>
        <div className="table-wrapper" style={{ marginTop: '0.75rem' }}>
          <table className="data-table" data-testid="node-checks-table">
            <thead>
              <tr>
                <th>{t('admin.nodeDetail.checks.col.name')}</th>
                <th>{t('admin.nodeDetail.checks.col.status')}</th>
                <th>{t('admin.nodeDetail.checks.col.errorCode')}</th>
                <th>{t('admin.nodeDetail.checks.col.detail')}</th>
                <th>{t('admin.nodeDetail.checks.col.observedAt')}</th>
              </tr>
            </thead>
            <tbody>
              {node.checks.length === 0 ? (
                <tr>
                  <td colSpan={5} className="text-muted">
                    {t('admin.nodeDetail.checks.empty')}
                  </td>
                </tr>
              ) : (
                node.checks.map((c) => {
                  const isWaitingGrants = isProvisionalLeaseWaitingCheck(
                    c,
                    nodeWithReasons.forwarding_block_reasons,
                  );
                  const isLeaseExpiredCheck = c.error_code === 'LEASE_EXPIRED';
                  return (
                    <tr key={c.name} data-testid={`node-check-row-${c.name}`}>
                      <td className="mono" style={{ fontWeight: 600 }}>
                        {c.name}
                      </td>
                      <td>
                        <span
                          className={`badge ${
                            isWaitingGrants
                              ? 'badge-warning'
                              : c.status === 'passed'
                                ? 'badge-success'
                                : c.status === 'failed'
                                  ? 'badge-danger'
                                  : 'badge-warning'
                          }`}
                          data-testid={`node-check-status-${c.name}`}
                        >
                          {isWaitingGrants
                            ? t('admin.nodeDetail.provisionalLease.waitingStatus')
                            : c.status}
                        </span>
                      </td>
                      <td
                        className="mono"
                        data-testid={`node-check-error-code-${c.name}`}
                      >
                        {c.error_code ?? '—'}
                      </td>
                      <td
                        className="mono"
                        style={{ fontSize: '0.82rem', wordBreak: 'break-word' }}
                        data-testid={`node-check-detail-${c.name}`}
                      >
                        {isWaitingGrants
                          ? `${t('admin.nodeDetail.provisionalLease.waitingCopy')}${
                              c.detail ? ` — ${c.detail}` : ''
                            }`
                          : isLeaseExpiredCheck
                            ? `${t('admin.nodeDetail.leaseExpired.failureCopy')}${
                                c.detail ? ` — ${c.detail}` : ''
                              }`
                            : (c.detail ?? '—')}
                      </td>
                      <td className="mono" style={{ fontSize: '0.8rem' }}>
                        {c.observed_at}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </section>

      {/* Typed DNS Policy Evidence reported by the node (D-056) */}
      <section className="card" data-testid="node-dns-policy-evidence-section">
        <h2>{t('admin.nodeDetail.dnsEvidence.title')}</h2>
        <div
          className="banner banner-info"
          style={{ marginTop: '0.5rem', marginBottom: '0.75rem', fontSize: '0.85rem' }}
          data-testid="node-dns-evidence-disclaimer"
        >
          {t('admin.nodeDetail.dnsEvidence.disclaimer')}
        </div>
        {node.dns_policy_evidence == null ? (
          <div
            style={{ marginTop: '0.5rem' }}
            data-testid="node-dns-evidence-not-reported"
          >
            <span className="badge badge-warning">
              {t('admin.nodeDetail.dnsEvidence.notReported')}
            </span>
          </div>
        ) : (
          <div>
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
                gap: '0.75rem',
                fontSize: '0.85rem',
              }}
            >
              <div
                style={{
                  padding: '0.6rem 0.75rem',
                  background: '#f8fafc',
                  borderRadius: 'var(--radius-sm)',
                  border: '1px solid var(--color-border)',
                }}
              >
                <div
                  style={{
                    fontSize: '0.75rem',
                    color: 'var(--color-text-muted)',
                    marginBottom: '0.25rem',
                  }}
                >
                  {t('admin.nodeDetail.dnsEvidence.documentId')}
                </div>
                <div
                  className="mono"
                  style={{ wordBreak: 'break-all' }}
                  data-testid="node-dns-evidence-document-id"
                >
                  {node.dns_policy_evidence.document_id}
                </div>
              </div>

              <div
                style={{
                  padding: '0.6rem 0.75rem',
                  background: '#f8fafc',
                  borderRadius: 'var(--radius-sm)',
                  border: '1px solid var(--color-border)',
                }}
              >
                <div
                  style={{
                    fontSize: '0.75rem',
                    color: 'var(--color-text-muted)',
                    marginBottom: '0.25rem',
                  }}
                >
                  {t('admin.nodeDetail.dnsEvidence.contentSha256')}
                </div>
                <div
                  className="mono"
                  style={{ wordBreak: 'break-all' }}
                  data-testid="node-dns-evidence-content-sha256"
                >
                  {node.dns_policy_evidence.content_sha256}
                </div>
              </div>

              <div
                style={{
                  padding: '0.6rem 0.75rem',
                  background: '#f8fafc',
                  borderRadius: 'var(--radius-sm)',
                  border: '1px solid var(--color-border)',
                }}
              >
                <div
                  style={{
                    fontSize: '0.75rem',
                    color: 'var(--color-text-muted)',
                    marginBottom: '0.25rem',
                  }}
                >
                  {t('admin.nodeDetail.dnsEvidence.generation')}
                </div>
                <div className="mono" data-testid="node-dns-evidence-generation">
                  {node.dns_policy_evidence.generation}
                </div>
              </div>

              <div
                style={{
                  padding: '0.6rem 0.75rem',
                  background: '#f8fafc',
                  borderRadius: 'var(--radius-sm)',
                  border: '1px solid var(--color-border)',
                }}
              >
                <div
                  style={{
                    fontSize: '0.75rem',
                    color: 'var(--color-text-muted)',
                    marginBottom: '0.25rem',
                  }}
                >
                  {t('admin.nodeDetail.dnsEvidence.outcome')}
                </div>
                <div>
                  <span
                    className={`badge ${
                      node.dns_policy_evidence.outcome === 'applied'
                        ? 'badge-success'
                        : node.dns_policy_evidence.outcome === 'failed'
                          ? 'badge-danger'
                          : 'badge-warning'
                    }`}
                    data-testid="node-dns-evidence-outcome"
                  >
                    {node.dns_policy_evidence.outcome}
                  </span>
                </div>
              </div>

              <div
                style={{
                  padding: '0.6rem 0.75rem',
                  background: '#f8fafc',
                  borderRadius: 'var(--radius-sm)',
                  border: '1px solid var(--color-border)',
                }}
              >
                <div
                  style={{
                    fontSize: '0.75rem',
                    color: 'var(--color-text-muted)',
                    marginBottom: '0.25rem',
                  }}
                >
                  {t('admin.nodeDetail.dnsEvidence.appliedAt')}
                </div>
                <div className="mono" data-testid="node-dns-evidence-applied-at">
                  {node.dns_policy_evidence.applied_at ?? '—'}
                </div>
              </div>

              <div
                style={{
                  padding: '0.6rem 0.75rem',
                  background: '#f8fafc',
                  borderRadius: 'var(--radius-sm)',
                  border: '1px solid var(--color-border)',
                }}
              >
                <div
                  style={{
                    fontSize: '0.75rem',
                    color: 'var(--color-text-muted)',
                    marginBottom: '0.25rem',
                  }}
                >
                  {t('admin.nodeDetail.dnsEvidence.errorCode')}
                </div>
                <div className="mono" data-testid="node-dns-evidence-error-code">
                  {node.dns_policy_evidence.error_code ?? '—'}
                </div>
              </div>
            </div>

            <div style={{ marginTop: '0.75rem' }} data-testid="node-dns-evidence-feeds">
              <div
                style={{
                  fontSize: '0.82rem',
                  fontWeight: 600,
                  marginBottom: '0.35rem',
                }}
              >
                {t('admin.nodeDetail.dnsEvidence.feedsTitle')}
              </div>
              {node.dns_policy_evidence.feeds.length === 0 ? (
                <div className="text-muted" style={{ fontSize: '0.82rem' }}>
                  {t('admin.nodeDetail.dnsEvidence.feedsEmpty')}
                </div>
              ) : (
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
                  {node.dns_policy_evidence.feeds.map((f) => (
                    <span
                      key={f.feed_code}
                      className="badge badge-neutral mono"
                      data-testid={`node-dns-evidence-feed-${f.feed_code}`}
                    >
                      {f.feed_code} (rules={f.rule_count}, sha256=
                      {f.content_sha256.slice(0, 12)}…)
                    </span>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </section>

      {/* Recent Errors & Diagnostics */}
      <section className="card" data-testid="node-recent-errors-section">
        <h2>{t('admin.nodeDetail.errors.title')}</h2>
        {!latestError ? (
          <p style={{ color: 'var(--color-text-muted)', marginTop: '0.5rem' }}>
            {t('admin.nodeDetail.errors.empty')}
          </p>
        ) : (
          <div style={{ marginTop: '0.5rem' }}>
            <details style={{ marginBottom: '0.5rem' }}>
              <summary
                style={{ cursor: 'pointer', fontWeight: 600 }}
                data-testid="node-latest-error-summary"
              >
                <span
                  className={`badge ${
                    latestError.code === 'AWAITING_MEMBER_GRANTS'
                      ? 'badge-warning'
                      : 'badge-danger'
                  }`}
                  style={{ marginRight: '0.5rem' }}
                  data-testid="node-latest-error-badge"
                >
                  {latestError.code}
                </span>
                {latestError.code === 'AWAITING_MEMBER_GRANTS'
                  ? `${t('admin.nodeDetail.provisionalLease.waitingCopy')}${
                      latestError.message ? ` (${latestError.message})` : ''
                    }`
                  : latestError.code === 'LEASE_EXPIRED'
                    ? `${t('admin.nodeDetail.leaseExpired.failureCopy')}${
                        latestError.message ? ` (${latestError.message})` : ''
                      }`
                    : latestError.message}
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
                {node.checks
                  .map(
                    (c) =>
                      `${c.name}=${
                        isProvisionalLeaseWaitingCheck(
                          c,
                          nodeWithReasons.forwarding_block_reasons,
                        )
                          ? 'waiting'
                          : c.status
                      }`,
                  )
                  .join(', ') || 'none'}
              </div>
            </details>
          </div>
        )}
      </section>
    </div>
  );
}
