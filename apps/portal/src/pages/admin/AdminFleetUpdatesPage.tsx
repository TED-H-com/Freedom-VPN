import React, { useCallback, useEffect, useState } from 'react';
import { portalApi, useOperationPoller, type FleetReleasesResponse } from '../../api';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n } from '../../i18n';
import { summarizeBatch, UPDATE_BUCKET_KEYS } from '../../lib/batch';

export function AdminFleetUpdatesPage(): React.ReactElement {
  const { t } = useI18n();
  const [data, setData] = useState<FleetReleasesResponse | null>(null);
  const [selectedReleaseId, setSelectedReleaseId] = useState<string>('');
  const [justAccepted, setJustAccepted] = useState<boolean>(false);

  const loadData = useCallback(async () => {
    const res = await portalApi.getReleasesAndBatch();
    setData(res.data);
    setSelectedReleaseId((prev) => prev || res.data.fleet_default_release_id);
  }, []);

  const poller = useOperationPoller({
    scope: 'admin',
    onTerminal: () => {
      void loadData();
    },
  });

  useEffect(() => {
    void loadData();
  }, [loadData]);

  if (!data) {
    return <div className="card">Loading releases and update batch...</div>;
  }

  const selectedRelease =
    data.releases.find((r) => r.release_id === selectedReleaseId) ?? data.releases[0]!;
  const batch = data.active_batch;
  const batchSummary = summarizeBatch(batch.target_count, batch.counts);

  const handleUpdateAll = async () => {
    const res = await portalApi.createUpdateBatch({
      release_id: selectedRelease.release_id,
      scope: 'all_nodes',
      expected_generation: data.fleet_generation,
      set_fleet_default: true,
      reason: `Admin triggered fleet update to ${selectedRelease.product_version}`,
    });
    if (res.status === 202) {
      setJustAccepted(true);
      setData((prev) =>
        prev
          ? {
              ...prev,
              fleet_default_release_id: res.data.fleet_default_release_id,
              active_batch: res.data,
            }
          : prev,
      );
    }
  };

  const handleRetryFailed = async () => {
    const opRes = await portalApi.retryUpdateBatch(batch.batch_id);
    poller.startTracking(opRes);
  };

  return (
    <div>
      <header className="page-header">
        <h1>{t('admin.updates.title')}</h1>
        <p className="page-subtitle">{t('admin.updates.subtitle')}</p>
        <p style={{ fontSize: '0.88rem', marginTop: 6 }}>
          <strong>
            {t('admin.updates.currentFleetDefault', {
              releaseId: data.fleet_default_release_id,
            })}
          </strong>{' '}
          ｜{' '}
          <strong>
            {t('admin.updates.minServingVersion', {
              version: data.minimum_serving_version,
            })}
          </strong>
        </p>
      </header>

      <OperationBanner
        operation={poller.operation}
        requestId={poller.requestId}
        isPolling={poller.isPolling}
        onDismiss={poller.reset}
      />

      {justAccepted && (
        <div
          className="alert alert-info"
          role="status"
          data-testid="update-batch-accepted-banner"
        >
          <strong>{t('admin.updates.acceptedBanner')}</strong>
        </div>
      )}

      <section className="card" aria-labelledby="select-release-heading">
        <h2 id="select-release-heading">{t('admin.updates.selectReleaseLabel')}</h2>
        <div className="form-grid">
          <div className="form-field">
            <label htmlFor="target-release-select">
              {t('admin.updates.selectReleaseLabel')}
            </label>
            <select
              id="target-release-select"
              value={selectedRelease.release_id}
              onChange={(e) => setSelectedReleaseId(e.target.value)}
              data-testid="target-release-select"
            >
              {data.releases.map((rel) => (
                <option key={rel.release_id} value={rel.release_id}>
                  v{rel.product_version} ({rel.release_id}) — seq #{rel.release_sequence}{' '}
                  [{rel.architectures.join(', ')}]
                </option>
              ))}
            </select>
          </div>
        </div>

        <div
          className={
            selectedRelease.requires_service_restart
              ? 'alert alert-warning'
              : 'alert alert-info'
          }
          data-testid="release-restart-indicator"
        >
          {selectedRelease.requires_service_restart
            ? t('admin.updates.restartRequiredYes')
            : t('admin.updates.restartRequiredNo')}
        </div>

        <div className="btn-row">
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => void handleUpdateAll()}
            data-testid="run-update-all-btn"
          >
            {t('admin.updates.runAllBtn')}
          </button>
        </div>
      </section>

      <section className="card" aria-labelledby="batch-progress-heading">
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
            <h2 id="batch-progress-heading" style={{ marginBottom: 4 }}>
              {t('admin.updates.batchTitle', { batchId: batch.batch_id })}
            </h2>
            <p style={{ margin: 0, fontSize: '0.9rem' }}>
              <span data-testid="batch-computed-state">
                {t('admin.updates.batchStateLabel', {
                  state: batchSummary.computedState,
                })}
              </span>{' '}
              ｜{' '}
              <span data-testid="batch-target-count">
                {t('admin.updates.targetCountLabel', { count: batch.target_count })}
              </span>{' '}
              ｜{' '}
              <span data-testid="batch-bucket-sum">
                {t('admin.updates.bucketSumCheck', {
                  sum: batchSummary.bucketSum,
                  targetCount: batch.target_count,
                })}
              </span>
            </p>
          </div>
          <button
            type="button"
            className="btn"
            onClick={() => void handleRetryFailed()}
            disabled={poller.isPolling}
            data-testid="retry-failed-targets-btn"
          >
            {t('admin.updates.retryFailedBtn')}
          </button>
        </div>

        <div
          className="alert alert-info"
          style={{ marginTop: 12 }}
          data-testid="batch-verified-success-banner"
        >
          {t('admin.updates.verifiedSuccessLabel', {
            count: batchSummary.verifiedSuccessCount,
          })}
        </div>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))',
            gap: 10,
            marginBottom: 18,
          }}
          data-testid="batch-buckets-grid"
        >
          {UPDATE_BUCKET_KEYS.map((bucketKey) => (
            <div
              key={bucketKey}
              style={{
                padding: '10px 12px',
                border: '1px solid var(--border-color)',
                borderRadius: 6,
                background: 'var(--bg-subtle)',
              }}
              data-testid={`bucket-${bucketKey}`}
            >
              <div style={{ fontSize: '0.78rem', fontWeight: 700 }}>{bucketKey}</div>
              <div style={{ fontSize: '1.35rem', fontWeight: 700 }}>
                {batch.counts[bucketKey]}
              </div>
            </div>
          ))}
        </div>

        <div className="table-wrapper">
          <table className="data-table" data-testid="batch-targets-table">
            <thead>
              <tr>
                <th scope="col">{t('admin.updates.col.node')}</th>
                <th scope="col">{t('admin.updates.col.pubIntent')}</th>
                <th scope="col">{t('admin.updates.col.versions')}</th>
                <th scope="col">{t('admin.updates.col.bucket')}</th>
                <th scope="col">{t('admin.updates.col.error')}</th>
              </tr>
            </thead>
            <tbody>
              {batch.targets.map((target) => (
                <tr key={target.target_id} data-testid={`target-row-${target.node_name}`}>
                  <td>
                    <strong>{target.node_name}</strong>
                    <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                      {target.region} · <code>{target.arch}</code>
                    </div>
                  </td>
                  <td>
                    <span className="badge badge-neutral">
                      {target.publication_intent}
                    </span>
                  </td>
                  <td>
                    <code>{target.current_version}</code> →{' '}
                    <code>{target.requested_version}</code> (last_good:{' '}
                    <code>{target.last_good_version}</code>)
                  </td>
                  <td>
                    <span
                      className={
                        target.bucket === 'succeeded' ||
                        target.bucket === 'already_current'
                          ? 'badge badge-success'
                          : target.bucket === 'failed' ||
                              target.bucket === 'rolled_back' ||
                              target.bucket === 'cancelled'
                            ? 'badge badge-danger'
                            : 'badge badge-info'
                      }
                    >
                      {target.bucket}
                    </span>
                  </td>
                  <td>
                    {target.error_code ? (
                      <code style={{ color: 'var(--danger)' }}>{target.error_code}</code>
                    ) : (
                      '—'
                    )}{' '}
                    <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                      ({target.last_reported_at})
                    </span>
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
