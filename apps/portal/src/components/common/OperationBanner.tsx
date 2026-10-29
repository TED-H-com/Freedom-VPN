import React from 'react';
import type { Operation, OperationStep } from '../../api/types';
import { useI18n, type TranslationKey } from '../../i18n';

interface OperationBannerProps {
  operation: Operation | null;
  requestId?: string | null;
  isPolling: boolean;
  onDismiss?: () => void;
}

const STEP_NAME_KEYS: Record<string, TranslationKey> = {
  agent_policy: 'op.step.agent_policy',
  netbird_route_readback: 'op.step.netbird_route_readback',
  data_path_probe: 'op.step.data_path_probe',
  node_allowset_withdraw: 'op.step.node_allowset_withdraw',
  cert_and_peer_revoke: 'op.step.cert_and_peer_revoke',
  policy_outbox_sync: 'op.step.policy_outbox_sync',
  resolver_1_smoke_apply: 'op.step.resolver_1_smoke_apply',
  resolver_2_smoke_apply: 'op.step.resolver_2_smoke_apply',
};

function stepBadgeClass(status: OperationStep['status']): string {
  switch (status) {
    case 'succeeded':
      return 'badge badge-success';
    case 'running':
      return 'badge badge-info';
    case 'failed':
      return 'badge badge-danger';
    default:
      return 'badge badge-neutral';
  }
}

export function OperationBanner({
  operation,
  requestId,
  isPolling,
  onDismiss,
}: OperationBannerProps): React.ReactElement | null {
  const { t } = useI18n();

  if (!operation) {
    return null;
  }

  const isRunningOrPending =
    operation.status === 'pending' || operation.status === 'running' || isPolling;
  const isPartial = operation.status === 'partially_succeeded';
  const isSucceeded = operation.status === 'succeeded' && !isPolling;

  const alertClass = isRunningOrPending
    ? 'alert alert-info'
    : isSucceeded
      ? 'alert alert-success'
      : isPartial
        ? 'alert alert-warning'
        : 'alert alert-danger';

  return (
    <section
      className={alertClass}
      role="status"
      aria-live="polite"
      data-testid="operation-banner"
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          gap: '12px',
        }}
      >
        <div>
          <strong data-testid="operation-status-headline">
            {isRunningOrPending
              ? `${t('op.savedPendingApply')} — ${t('op.acceptedPending')}`
              : isSucceeded
                ? t('op.completedApplied')
                : isPartial
                  ? t('op.partiallySucceeded')
                  : t('op.failed')}
          </strong>
          {operation.summary_code === 'route_withdrawn_device_unconfirmed' && (
            <p
              style={{ margin: '6px 0 0', fontWeight: 700 }}
              data-testid="operation-offline-unpublish-warning"
            >
              {t('op.routeWithdrawnDeviceUnconfirmed')}
            </p>
          )}
          <div style={{ fontSize: '0.82rem', marginTop: '4px' }}>
            <span>
              {t('op.idLabel')}: <code>{operation.operation_id}</code>
            </span>
            {requestId && (
              <span style={{ marginLeft: '12px' }}>
                {t('op.requestIdLabel')}: <code>{requestId}</code>
              </span>
            )}
            {operation.desired_generation && (
              <span style={{ marginLeft: '12px' }}>
                {t('op.generationsLabel', {
                  desired: operation.desired_generation,
                  observed: operation.observed_generation ?? '—',
                })}
              </span>
            )}
          </div>
        </div>
        {onDismiss && !isRunningOrPending && (
          <button type="button" className="btn" onClick={onDismiss}>
            {t('op.dismiss')}
          </button>
        )}
      </div>

      <ul className="step-list" aria-label="Operation steps">
        {operation.steps.map((step) => {
          const labelKey = STEP_NAME_KEYS[step.name];
          const stepTitle = labelKey ? t(labelKey) : step.name;
          return (
            <li
              key={step.name}
              className="step-item"
              data-testid={`op-step-${step.name}`}
            >
              <span className={stepBadgeClass(step.status)}>{step.status}</span>
              <span>{stepTitle}</span>
              {step.detail && (
                <span style={{ fontSize: '0.82rem', color: 'var(--text-secondary)' }}>
                  ({step.detail})
                </span>
              )}
            </li>
          );
        })}
      </ul>
    </section>
  );
}
