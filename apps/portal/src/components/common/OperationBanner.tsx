import React, { useEffect, useState } from 'react';
import {
  apiClient,
  hasOfflineRouteWithdrawnWarning,
  isTerminalOperationStatus,
  type OperationResource,
  type OperationStep,
} from '../../api';
import { useI18n, type TranslationKey } from '../../i18n';

interface OperationBannerProps {
  operation: OperationResource | null;
  scope?: 'admin' | 'member';
  isPolling?: boolean;
  onDismiss?: () => void;
  onTerminal?: (op: OperationResource) => void;
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
  scope = 'admin',
  isPolling = false,
  onDismiss,
  onTerminal,
}: OperationBannerProps): React.ReactElement | null {
  const { t } = useI18n();
  const [polledOp, setPolledOp] = useState<OperationResource | null>(operation);
  const [pollAttempt, setPollAttempt] = useState(0);

  useEffect(() => {
    setPolledOp(operation);
    setPollAttempt(0);
  }, [operation]);

  useEffect(() => {
    if (!polledOp || isTerminalOperationStatus(polledOp.status)) {
      return;
    }
    let cancelled = false;
    const delayMs =
      pollAttempt === 0 ? 60 : Math.min(2000, 120 * Math.pow(2, pollAttempt - 1));
    const timer = setTimeout(async () => {
      try {
        const updated =
          scope === 'member'
            ? await apiClient.getMemberOperation(polledOp.operation_id)
            : await apiClient.getOperation(polledOp.operation_id);
        if (!cancelled) {
          setPolledOp(updated);
          if (isTerminalOperationStatus(updated.status)) {
            if (onTerminal) {
              onTerminal(updated);
            }
          } else {
            setPollAttempt((prev) => prev + 1);
          }
        }
      } catch {
        if (!cancelled) {
          setPollAttempt((prev) => prev + 1);
        }
      }
    }, delayMs);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [polledOp, pollAttempt, scope, onTerminal]);

  const current = polledOp ?? operation;
  if (!current) {
    return null;
  }

  const isRunningOrPending =
    current.status === 'pending' || current.status === 'running' || isPolling;
  const isPartial = current.status === 'partially_succeeded';
  const isSuperseded = current.status === 'superseded';
  const isSucceeded = current.status === 'succeeded' && !isPolling;

  const alertClass = isRunningOrPending
    ? 'alert alert-info'
    : isSucceeded
      ? 'alert alert-success'
      : isPartial || isSuperseded
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
                  : isSuperseded
                    ? t('op.superseded')
                    : t('op.failed')}
          </strong>
          {hasOfflineRouteWithdrawnWarning(current) && (
            <p
              style={{ margin: '6px 0 0', fontWeight: 700 }}
              data-testid="operation-offline-unpublish-warning"
            >
              {t('op.routeWithdrawnDeviceUnconfirmed')}
            </p>
          )}
          <div style={{ fontSize: '0.82rem', marginTop: '4px' }}>
            <span>
              {t('op.idLabel')}: <code>{current.operation_id}</code>
            </span>
            <span style={{ marginLeft: '12px' }}>
              {t('op.requestIdLabel')}: <code>{current.request_id}</code>
            </span>
            {current.desired_generation && (
              <span style={{ marginLeft: '12px' }}>
                {t('op.generationsLabel', {
                  desired: current.desired_generation,
                  observed: current.observed_generation ?? '—',
                })}
              </span>
            )}
          </div>
        </div>
        {onDismiss && (
          <button type="button" className="btn btn-secondary" onClick={onDismiss}>
            {t('op.dismiss')}
          </button>
        )}
      </div>

      <ul className="step-list" aria-label="Operation steps">
        {current.steps.map((step) => {
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
              {step.message && (
                <span style={{ fontSize: '0.82rem', color: 'var(--color-text-muted)' }}>
                  ({step.message})
                </span>
              )}
            </li>
          );
        })}
      </ul>
    </section>
  );
}
