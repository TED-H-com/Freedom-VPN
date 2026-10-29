import React, { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  apiClient,
  ApiError,
  extractMinimumServingVersionLabel,
  type OperationResource,
} from '../../api';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n } from '../../i18n';
import { computeVerifiedSuccessCount, sumBatchBuckets } from '../../lib/batch';
import { PRIMARY_BATCH_ID, PRIMARY_RELEASE_ID } from '../../mocks/data';

export function AdminFleetUpdatesPage(): React.ReactElement {
  const { t, locale } = useI18n();
  const queryClient = useQueryClient();

  const [selectedReleaseId, setSelectedReleaseId] = useState(PRIMARY_RELEASE_ID);
  const [acceptedNotice, setAcceptedNotice] = useState(false);
  const [retryOperation, setRetryOperation] = useState<OperationResource | null>(null);
  const [error, setError] = useState<ApiError | null>(null);

  const releasesQuery = useQuery({
    queryKey: ['admin', 'releases'],
    queryFn: () => apiClient.listReleases(),
  });

  const policiesQuery = useQuery({
    queryKey: ['admin', 'policies', 'release_support'],
    queryFn: () => apiClient.listPolicies({ type: 'release_support' }),
  });

  const batchQuery = useQuery({
    queryKey: ['admin', 'update-batch', PRIMARY_BATCH_ID],
    queryFn: () => apiClient.getUpdateBatch(PRIMARY_BATCH_ID),
  });

  if (
    releasesQuery.isLoading ||
    batchQuery.isLoading ||
    !releasesQuery.data ||
    !batchQuery.data
  ) {
    return (
      <div className="card" role="status">
        Loading releases and fleet update batch...
      </div>
    );
  }

  const releases = releasesQuery.data.items;
  const fleet = releasesQuery.data.fleet;
  const batch = batchQuery.data;

  const selectedRelease =
    releases.find((r) => r.release_id === selectedReleaseId) ?? releases[0];
  const minServingLabel = extractMinimumServingVersionLabel(
    policiesQuery.data,
    fleet.serving_release_ids,
  );

  const bucketSum = sumBatchBuckets(batch.counts);
  const verifiedSuccess = computeVerifiedSuccessCount(batch.counts);

  const handleRunAllUpdate = async () => {
    setError(null);
    try {
      await apiClient.createUpdateBatch({
        release_id: selectedReleaseId,
        scope: 'all_nodes',
        expected_generation: batch.targets[0]?.generation ?? '7',
        set_fleet_default: true,
        reason: `Fleet update all nodes to ${selectedRelease?.product_version ?? selectedReleaseId}`,
      });
      setAcceptedNotice(true);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'releases'] });
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'update-batch', PRIMARY_BATCH_ID],
      });
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err);
      }
    }
  };

  const handleRetryFailed = async () => {
    setError(null);
    try {
      const op = await apiClient.retryUpdateBatch(batch.batch_id);
      setRetryOperation(op);
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'update-batch', PRIMARY_BATCH_ID],
      });
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err);
      }
    }
  };

  return (
    <div data-testid="admin-fleet-updates-page">
      <div className="page-header">
        <div>
          <h1>{t('admin.updates.title')}</h1>
          <p className="page-subtitle">{t('admin.updates.subtitle')}</p>
        </div>
        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
          <span className="badge badge-info mono" data-testid="fleet-default-release">
            {t('admin.updates.currentFleetDefault', {
              releaseId: fleet.fleet_default_release_id ?? '-',
            })}
          </span>
          <span className="badge badge-neutral mono" data-testid="min-serving-version">
            {t('admin.updates.minServingVersion', {
              version: minServingLabel,
            })}
          </span>
        </div>
      </div>

      {error && (
        <div className="alert alert-danger" role="alert">
          <strong>{error.code}</strong>: {error.messageForLocale(locale)}
        </div>
      )}

      <OperationBanner
        operation={retryOperation}
        onDismiss={() => setRetryOperation(null)}
      />

      {acceptedNotice && (
        <div
          className="alert alert-info"
          role="status"
          data-testid="update-batch-accepted-banner"
        >
          {t('admin.updates.acceptedBanner')}
        </div>
      )}

      {/* One-action Update All Nodes control (§13.12, UPD01) */}
      <section className="card">
        <div className="form-grid" style={{ alignItems: 'end' }}>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label htmlFor="target-release-select">
              {t('admin.updates.selectReleaseLabel')}
            </label>
            <select
              id="target-release-select"
              data-testid="target-release-select"
              value={selectedReleaseId}
              onChange={(e) => setSelectedReleaseId(e.target.value)}
            >
              {releases
                .filter((r) => r.support_state === 'approved')
                .map((rel) => (
                  <option key={rel.release_id} value={rel.release_id}>
                    {rel.product_version} (seq {rel.release_sequence} ·{' '}
                    {rel.architectures.join(', ')})
                  </option>
                ))}
            </select>
          </div>

          <div>
            {selectedRelease?.compatibility.requires_service_restart ? (
              <span className="badge badge-warning">
                {t('admin.updates.restartRequiredYes')}
              </span>
            ) : (
              <span className="badge badge-success">
                {t('admin.updates.restartRequiredNo')}
              </span>
            )}
          </div>

          <div>
            <button
              type="button"
              className="btn btn-primary"
              data-testid="run-all-nodes-update-btn"
              onClick={() => void handleRunAllUpdate()}
            >
              {t('admin.updates.runAllBtn')}
            </button>
          </div>
        </div>
      </section>

      {/* Batch Progress & 12 Mutually Exclusive Buckets (§13.12) */}
      <section className="card" data-testid="update-batch-progress-card">
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: '0.75rem',
            marginBottom: '1rem',
          }}
        >
          <div>
            <h2>{t('admin.updates.batchTitle', { batchId: batch.batch_id })}</h2>
            <div style={{ fontSize: '0.85rem', color: 'var(--color-text-muted)' }}>
              {t('admin.updates.batchStateLabel', { state: batch.state })} ·{' '}
              {t('admin.updates.targetCountLabel', { count: batch.target_count })} ·{' '}
              <span className="mono" data-testid="bucket-sum-check">
                {t('admin.updates.bucketSumCheck', {
                  sum: bucketSum,
                  targetCount: batch.target_count,
                })}
              </span>
            </div>
          </div>

          <button
            type="button"
            className="btn btn-secondary"
            data-testid="retry-failed-targets-btn"
            onClick={() => void handleRetryFailed()}
          >
            {t('admin.updates.retryFailedBtn')}
          </button>
        </div>

        <div
          className="alert alert-success"
          data-testid="verified-success-summary"
          style={{ marginBottom: '1rem' }}
        >
          {t('admin.updates.verifiedSuccessLabel', { count: verifiedSuccess })}
        </div>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))',
            gap: '0.5rem',
            marginBottom: '1.25rem',
          }}
          data-testid="batch-bucket-grid"
        >
          {Object.entries(batch.counts).map(([bucketName, count]) => (
            <div
              key={bucketName}
              style={{
                padding: '0.5rem 0.75rem',
                border: '1px solid var(--color-border)',
                borderRadius: 'var(--radius-sm)',
                background: count > 0 ? '#f8fafc' : '#ffffff',
              }}
              data-testid={`bucket-${bucketName}`}
            >
              <div
                className="mono"
                style={{ fontSize: '0.75rem', color: 'var(--color-text-muted)' }}
              >
                {bucketName}
              </div>
              <div style={{ fontSize: '1.15rem', fontWeight: 700 }}>{count}</div>
            </div>
          ))}
        </div>

        <div className="table-wrapper">
          <table className="data-table" data-testid="batch-targets-table">
            <thead>
              <tr>
                <th>{t('admin.updates.col.node')}</th>
                <th>{t('admin.updates.col.pubIntent')}</th>
                <th>{t('admin.updates.col.versions')}</th>
                <th>{t('admin.updates.col.bucket')}</th>
                <th>{t('admin.updates.col.error')}</th>
              </tr>
            </thead>
            <tbody>
              {batch.targets.map((target) => (
                <tr key={target.node_id} data-testid={`batch-target-${target.node_id}`}>
                  <td>
                    <div style={{ fontWeight: 600 }}>{target.node_name}</div>
                    <div
                      className="mono"
                      style={{ fontSize: '0.75rem', color: 'var(--color-text-muted)' }}
                    >
                      {target.node_id}
                    </div>
                  </td>
                  <td>
                    <span className="badge badge-info">
                      online_at_snapshot: {String(target.online_at_snapshot)}
                    </span>
                  </td>
                  <td className="mono" style={{ fontSize: '0.8rem' }}>
                    {target.current_release_id ?? '-'} → {batch.release_id}
                  </td>
                  <td>
                    <span
                      className={`badge ${
                        target.state === 'succeeded' || target.state === 'already_current'
                          ? 'badge-success'
                          : target.state === 'rolled_back' || target.state === 'failed'
                            ? 'badge-danger'
                            : 'badge-warning'
                      }`}
                      data-testid={`target-bucket-${target.node_id}`}
                    >
                      {target.state}
                    </span>
                  </td>
                  <td className="mono" style={{ fontSize: '0.75rem' }}>
                    {target.current_job?.error_code ?? '-'} (
                    {target.last_result_at ?? target.last_observed_at ?? 'pending'})
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
