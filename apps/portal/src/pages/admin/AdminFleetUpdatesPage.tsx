import React, { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  apiClient,
  extractMinimumServingVersionLabel,
  type OperationResource,
  type ReleaseRegisterRequest,
} from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n } from '../../i18n';
import { computeVerifiedSuccessCount, sumBatchBuckets } from '../../lib/batch';

const IS_MOCK_ENV =
  !import.meta.env.PROD &&
  (import.meta.env.VITE_API_MOCK === '1' || import.meta.env.MODE === 'test');

const MOCK_FALLBACK_BATCH_ID = IS_MOCK_ENV ? '85ef720a-31a0-4250-90d2-4b92c58695bc' : '';
const MOCK_FALLBACK_RELEASE_ID = IS_MOCK_ENV
  ? '6d2797d0-b9a5-4724-96af-9b3039d351b1'
  : '';

export function AdminFleetUpdatesPage(): React.ReactElement {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();

  const initialBatchId = searchParams.get('batch_id') ?? MOCK_FALLBACK_BATCH_ID;
  const [activeBatchId, setActiveBatchId] = useState<string>(initialBatchId);
  const [batchLookupInput, setBatchLookupInput] = useState<string>(initialBatchId);
  const [targetsCursor, setTargetsCursor] = useState<string | undefined>(undefined);
  const [targetsCursorHistory, setTargetsCursorHistory] = useState<
    Array<string | undefined>
  >([]);

  const [selectedReleaseId, setSelectedReleaseId] = useState(MOCK_FALLBACK_RELEASE_ID);
  const [expectedGeneration, setExpectedGeneration] = useState<string>('');
  const [acceptedNotice, setAcceptedNotice] = useState(false);
  const [retryOperation, setRetryOperation] = useState<OperationResource | null>(null);
  const [error, setError] = useState<unknown>(null);

  const [manifestUrl, setManifestUrl] = useState(
    'https://releases.example.com/v1.4.2/manifest.json',
  );
  const [manifestEnvelopeJson, setManifestEnvelopeJson] = useState('');
  const [registerSuccess, setRegisterSuccess] = useState<string | null>(null);

  const releasesQuery = useQuery({
    queryKey: ['admin', 'releases'],
    queryFn: () => apiClient.listReleases(),
  });

  const policiesQuery = useQuery({
    queryKey: ['admin', 'policies', 'release_support'],
    queryFn: () => apiClient.listPolicies({ type: 'release_support' }),
  });

  const batchQuery = useQuery({
    queryKey: ['admin', 'update-batch', activeBatchId, targetsCursor],
    queryFn: () => apiClient.getUpdateBatch(activeBatchId, { cursor: targetsCursor }),
    enabled: Boolean(activeBatchId),
  });

  useEffect(() => {
    const items = releasesQuery.data?.items ?? [];
    if (items.length > 0 && !selectedReleaseId) {
      const approved = items.find((r) => r.support_state === 'approved') ?? items[0];
      if (approved) {
        setSelectedReleaseId(approved.release_id);
      }
    }
  }, [releasesQuery.data, selectedReleaseId]);

  if (releasesQuery.error) {
    return (
      <div data-testid="admin-fleet-updates-page">
        <ApiErrorAlert error={releasesQuery.error} testId="admin-updates-error" />
      </div>
    );
  }

  if (
    releasesQuery.isLoading ||
    !releasesQuery.data ||
    (Boolean(activeBatchId) && batchQuery.isLoading)
  ) {
    return (
      <div className="card" role="status">
        Loading releases and fleet update batch...
      </div>
    );
  }

  const releases = releasesQuery.data.items;
  const fleet = releasesQuery.data.fleet;
  const batch = batchQuery.data ?? null;

  const selectedRelease =
    releases.find((r) => r.release_id === selectedReleaseId) ?? releases[0];
  const minServingLabel = extractMinimumServingVersionLabel(
    policiesQuery.data,
    fleet.serving_release_ids,
  );

  const bucketSum = batch ? sumBatchBuckets(batch.counts) : 0;
  const verifiedSuccess = batch ? computeVerifiedSuccessCount(batch.counts) : 0;

  const handleRunAllUpdate = async () => {
    setError(null);
    try {
      const targetGen = expectedGeneration.trim() || batch?.targets[0]?.generation || '0';
      const created = await apiClient.createUpdateBatch({
        release_id: selectedRelease?.release_id ?? selectedReleaseId,
        scope: 'all_nodes',
        expected_generation: targetGen,
        set_fleet_default: true,
        reason: `Fleet update all nodes to ${selectedRelease?.product_version ?? selectedReleaseId}`,
      });
      setAcceptedNotice(true);
      if (created?.batch_id) {
        setActiveBatchId(created.batch_id);
        setBatchLookupInput(created.batch_id);
        setTargetsCursor(undefined);
        setTargetsCursorHistory([]);
      }
      await queryClient.invalidateQueries({ queryKey: ['admin', 'releases'] });
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'update-batch'],
      });
    } catch (err) {
      setError(err);
    }
  };

  const handleRetryFailed = async () => {
    if (!batch) return;
    setError(null);
    try {
      const op = await apiClient.retryUpdateBatch(batch.batch_id);
      setRetryOperation(op);
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'update-batch', batch.batch_id],
      });
    } catch (err) {
      setError(err);
    }
  };

  const handleRegisterRelease = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setRegisterSuccess(null);
    try {
      const parsedManifest = JSON.parse(
        manifestEnvelopeJson,
      ) as ReleaseRegisterRequest['manifest'];
      const registered = await apiClient.registerRelease({
        manifest_url: manifestUrl.trim(),
        manifest: parsedManifest,
      });
      setRegisterSuccess(
        'release_id' in registered ? registered.release_id : registered.operation_id,
      );
      setManifestEnvelopeJson('');
      await queryClient.invalidateQueries({ queryKey: ['admin', 'releases'] });
    } catch (err) {
      setError(err);
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

      <ApiErrorAlert
        error={error ?? batchQuery.error}
        testId="admin-updates-action-error"
      />

      <OperationBanner
        operation={retryOperation}
        onDismiss={() => setRetryOperation(null)}
        onTerminal={() => {
          void queryClient.invalidateQueries({ queryKey: ['admin', 'update-batch'] });
        }}
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
              value={selectedRelease?.release_id ?? selectedReleaseId}
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

          <div className="form-group" style={{ marginBottom: 0 }}>
            <label htmlFor="expected-gen-input">Expected Fleet Generation</label>
            <input
              id="expected-gen-input"
              data-testid="expected-generation-input"
              className="mono"
              value={expectedGeneration}
              placeholder={batch?.targets[0]?.generation ?? '0'}
              onChange={(e) => setExpectedGeneration(e.target.value)}
            />
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

        <div
          className="form-grid"
          style={{
            alignItems: 'end',
            marginTop: '1rem',
            paddingTop: '0.75rem',
            borderTop: '1px solid var(--color-border)',
          }}
        >
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label htmlFor="batch-id-lookup-input">Inspect Update Batch ID</label>
            <input
              id="batch-id-lookup-input"
              data-testid="batch-id-lookup-input"
              className="mono"
              value={batchLookupInput}
              placeholder="UUID of update batch"
              onChange={(e) => setBatchLookupInput(e.target.value)}
            />
          </div>
          <div>
            <button
              type="button"
              className="btn btn-secondary"
              data-testid="load-batch-btn"
              onClick={() => {
                const next = batchLookupInput.trim();
                setActiveBatchId(next);
                setTargetsCursor(undefined);
                setTargetsCursorHistory([]);
                if (next) {
                  setSearchParams({ batch_id: next });
                } else {
                  setSearchParams({});
                }
              }}
            >
              Load Batch
            </button>
          </div>
        </div>
      </section>

      {/* Batch Progress & 12 Mutually Exclusive Buckets (§13.12) */}
      {batch && (
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
                  <th>{t('admin.updates.col.policyEvidence')}</th>
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
                      <div style={{ display: 'flex', gap: '0.35rem', flexWrap: 'wrap' }}>
                        <span
                          className={`badge ${
                            target.state === 'succeeded' ||
                            target.state === 'already_current'
                              ? 'badge-success'
                              : target.state === 'rolled_back' ||
                                  target.state === 'failed'
                                ? 'badge-danger'
                                : 'badge-warning'
                          }`}
                          data-testid={`target-bucket-${target.node_id}`}
                        >
                          {target.state}
                        </span>
                        {target.needs_attention && (
                          <span
                            className="badge badge-danger"
                            data-testid={`target-needs-attention-${target.node_id}`}
                          >
                            {t('admin.updates.needsAttentionBadge')}
                          </span>
                        )}
                        {target.unconfirmed && (
                          <span
                            className="badge badge-warning"
                            data-testid={`target-unconfirmed-${target.node_id}`}
                          >
                            {t('admin.updates.unconfirmedBadge')}
                          </span>
                        )}
                      </div>
                    </td>
                    <td
                      className="mono"
                      style={{ fontSize: '0.75rem' }}
                      data-testid={`target-policy-evidence-${target.node_id}`}
                    >
                      <div
                        style={{
                          display: 'flex',
                          flexDirection: 'column',
                          gap: '0.25rem',
                        }}
                      >
                        <div data-testid={`target-policy-gen-${target.node_id}`}>
                          <span style={{ color: 'var(--color-text-muted)' }}>
                            policy_generation:{' '}
                          </span>
                          {target.policy_generation != null ? (
                            <span className="badge badge-info">
                              {target.policy_generation}
                            </span>
                          ) : (
                            <span className="badge badge-warning">
                              {t('admin.updates.evidenceNotReported')}
                            </span>
                          )}
                        </div>
                        <div data-testid={`target-policy-refs-${target.node_id}`}>
                          <span style={{ color: 'var(--color-text-muted)' }}>
                            policy_refs:{' '}
                          </span>
                          {target.policy_refs != null ? (
                            <span className="badge badge-info">
                              {target.policy_refs.length > 0
                                ? target.policy_refs.join(', ')
                                : 'none (0)'}
                            </span>
                          ) : (
                            <span className="badge badge-warning">
                              {t('admin.updates.evidenceNotReported')}
                            </span>
                          )}
                        </div>
                      </div>
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

          {(targetsCursorHistory.length > 0 || batch.next_cursor) && (
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginTop: '1rem',
              }}
              data-testid="batch-targets-pagination"
            >
              <button
                type="button"
                className="btn btn-secondary"
                data-testid="batch-targets-prev-btn"
                disabled={targetsCursorHistory.length === 0}
                onClick={() => {
                  const prev = targetsCursorHistory[targetsCursorHistory.length - 1];
                  setTargetsCursorHistory((h) => h.slice(0, -1));
                  setTargetsCursor(prev);
                }}
              >
                {t('common.pagination.prev')}
              </button>
              <span className="mono" style={{ fontSize: '0.8rem' }}>
                cursor: {targetsCursor ?? 'first'}
              </span>
              <button
                type="button"
                className="btn btn-secondary"
                data-testid="batch-targets-next-btn"
                disabled={!batch.next_cursor}
                onClick={() => {
                  if (batch.next_cursor) {
                    setTargetsCursorHistory((h) => [...h, targetsCursor]);
                    setTargetsCursor(batch.next_cursor);
                  }
                }}
              >
                {t('common.pagination.next')}
              </button>
            </div>
          )}
        </section>
      )}

      {/* Register Signed Release Manifest (§13.2, §13.3) */}
      <section className="card" data-testid="register-release-card">
        <h2>{t('admin.updates.registerReleaseTitle')}</h2>
        {registerSuccess && (
          <div className="alert alert-success" data-testid="register-release-success">
            Registered release: <span className="mono">{registerSuccess}</span>
          </div>
        )}
        <form
          onSubmit={(e) => void handleRegisterRelease(e)}
          style={{ marginTop: '0.75rem' }}
        >
          <div className="form-group">
            <label htmlFor="register-manifest-url">
              {t('admin.updates.manifestUrlLabel')}
            </label>
            <input
              id="register-manifest-url"
              data-testid="register-manifest-url-input"
              className="mono"
              value={manifestUrl}
              onChange={(e) => setManifestUrl(e.target.value)}
            />
          </div>
          <div className="form-group">
            <label htmlFor="register-manifest-json">
              {t('admin.updates.manifestJsonLabel')}
            </label>
            <textarea
              id="register-manifest-json"
              data-testid="register-manifest-json-input"
              className="mono"
              rows={4}
              value={manifestEnvelopeJson}
              placeholder='{"signed": {...}, "signatures": [...]}'
              onChange={(e) => setManifestEnvelopeJson(e.target.value)}
            />
          </div>
          <button
            type="submit"
            className="btn btn-secondary"
            data-testid="register-release-submit-btn"
            disabled={!manifestEnvelopeJson.trim()}
          >
            {t('admin.updates.registerReleaseBtn')}
          </button>
        </form>
      </section>
    </div>
  );
}
