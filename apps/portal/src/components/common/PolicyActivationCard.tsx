import React, { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { components } from '@contracts/schema';
import {
  ApiError,
  apiClient,
  type OperationResource,
  type PolicyDefinition,
  type PolicyRevision,
  type PolicyType,
} from '../../api';
import { useI18n } from '../../i18n';
import { ApiErrorAlert } from './ApiErrorAlert';
import { ConflictDialog } from './ConflictDialog';
import { OperationBanner } from './OperationBanner';

type PolicyRevisionSummary = components['schemas']['PolicyRevisionSummary'];

interface PolicyActivationCardProps {
  definitions: PolicyDefinition[];
  allowedTypes?: PolicyType[];
}

interface PendingActivation {
  definition: PolicyDefinition;
  targetSummary: PolicyRevisionSummary;
  targetRevision: PolicyRevision | null;
  currentRevision: PolicyRevision | null;
}

export function PolicyActivationCard({
  definitions,
  allowedTypes,
}: PolicyActivationCardProps): React.ReactElement {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const [pending, setPending] = useState<PendingActivation | null>(null);
  const [reason, setReason] = useState('Operator confirmed activation after diff review');
  const [activeOp, setActiveOp] = useState<OperationResource | null>(null);
  const [conflictError, setConflictError] = useState<ApiError | null>(null);

  const filteredDefs = allowedTypes
    ? definitions.filter((d) => allowedTypes.includes(d.type))
    : definitions;

  const activateMutation = useMutation({
    mutationFn: (opts: {
      definitionId: string;
      revisionId: string;
      reason: string;
      resourceVersion: string;
    }) =>
      apiClient.activatePolicyRevision(
        opts.definitionId,
        {
          revision_id: opts.revisionId,
          reason: opts.reason,
        },
        `"${opts.resourceVersion}"`,
        `activate-policy-${opts.definitionId}-${opts.revisionId}-${Date.now()}`,
      ),
    onSuccess: (op) => {
      setActiveOp(op);
      setPending(null);
      void queryClient.invalidateQueries({ queryKey: ['admin', 'policies'] });
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 412) {
        setConflictError(err);
      }
    },
  });

  const nonConflictError =
    activateMutation.error &&
    !(activateMutation.error instanceof ApiError && activateMutation.error.status === 412)
      ? activateMutation.error
      : null;

  return (
    <div className="card" data-testid="policy-revisions-activation-card">
      <h2 style={{ marginTop: 0 }}>{t('admin.policies.listTitle')}</h2>

      <ApiErrorAlert error={nonConflictError} testId="policy-activate-error" />

      <OperationBanner
        operation={activeOp}
        onDismiss={() => setActiveOp(null)}
        onTerminal={() => {
          void queryClient.invalidateQueries({ queryKey: ['admin', 'policies'] });
        }}
      />

      <ConflictDialog
        error={conflictError}
        onReload={() => {
          setConflictError(null);
          setPending(null);
          void queryClient.invalidateQueries({ queryKey: ['admin', 'policies'] });
        }}
      />

      {pending && (
        <div
          className="modal-backdrop"
          role="dialog"
          aria-modal="true"
          aria-labelledby="policy-activate-confirm-title"
          data-testid="policy-activate-confirm-modal"
        >
          <div className="modal-dialog" style={{ maxWidth: '720px' }}>
            <h3 id="policy-activate-confirm-title" style={{ marginTop: 0 }}>
              {t('admin.policies.confirmDialogTitle')}
            </h3>
            <div className="mono" style={{ fontSize: '0.82rem', marginBottom: '10px' }}>
              Definition: {pending.definition.code} ({pending.definition.type}) ｜ ETag: &quot;
              {pending.definition.resource_version}&quot; ｜ Active Revision:{' '}
              {pending.definition.active_revision?.revision ?? '—'} → Target Revision: #
              {pending.targetSummary.revision} ({pending.targetSummary.revision_id})
            </div>

            <div
              className="form-grid"
              data-testid="policy-activate-diff"
              style={{ marginBottom: '12px' }}
            >
              <div>
                <div style={{ fontWeight: 600, fontSize: '0.82rem', marginBottom: '4px' }}>
                  Current Active Revision (
                  {pending.definition.active_revision
                    ? `#${pending.definition.active_revision.revision}`
                    : 'none'}
                  )
                </div>
                <pre
                  className="mono"
                  style={{
                    fontSize: '0.75rem',
                    maxHeight: '220px',
                    overflow: 'auto',
                    background: '#f8fafc',
                    padding: '8px',
                    borderRadius: '4px',
                    border: '1px solid var(--color-border)',
                  }}
                >
                  {pending.currentRevision
                    ? JSON.stringify(pending.currentRevision.payload, null, 2)
                    : '—'}
                </pre>
              </div>
              <div>
                <div style={{ fontWeight: 600, fontSize: '0.82rem', marginBottom: '4px' }}>
                  Target Revision (#{pending.targetSummary.revision})
                </div>
                <pre
                  className="mono"
                  style={{
                    fontSize: '0.75rem',
                    maxHeight: '220px',
                    overflow: 'auto',
                    background: '#f0fdf4',
                    padding: '8px',
                    borderRadius: '4px',
                    border: '1px solid var(--color-border)',
                  }}
                >
                  {pending.targetRevision
                    ? JSON.stringify(pending.targetRevision.payload, null, 2)
                    : JSON.stringify(pending.targetSummary, null, 2)}
                </pre>
              </div>
            </div>

            <div className="form-group">
              <label htmlFor="policy-activate-reason-input">Activation Reason</label>
              <input
                id="policy-activate-reason-input"
                data-testid="policy-activate-reason-input"
                type="text"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
              />
            </div>

            <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
              <button
                type="button"
                className="btn btn-secondary"
                data-testid="policy-activate-cancel-btn"
                onClick={() => setPending(null)}
              >
                {t('admin.policies.cancelActivateBtn')}
              </button>
              <button
                type="button"
                className="btn btn-primary"
                data-testid="policy-activate-confirm-btn"
                disabled={activateMutation.isPending || !reason.trim()}
                onClick={() =>
                  activateMutation.mutate({
                    definitionId: pending.definition.definition_id,
                    revisionId: pending.targetSummary.revision_id,
                    reason: reason.trim(),
                    resourceVersion: pending.definition.resource_version,
                  })
                }
              >
                {t('admin.policies.confirmActivateBtn')}
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="table-wrapper">
        <table className="data-table" data-testid="policy-definitions-table">
          <thead>
            <tr>
              <th>Definition / Schema</th>
              <th>Active / Generation</th>
              <th>Revisions</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {filteredDefs.map((def) => {
              const revisions = def.revisions ?? [];
              const currentRev = def.active_revision ?? null;
              return (
                <tr
                  key={def.definition_id}
                  data-testid={`policy-def-row-${def.definition_id}`}
                >
                  <td>
                    <div style={{ fontWeight: 600 }}>{def.code}</div>
                    <div
                      className="mono"
                      style={{ fontSize: '0.75rem', color: 'var(--color-text-muted)' }}
                    >
                      {def.type} · {def.definition_id} · rv={def.resource_version}
                    </div>
                  </td>
                  <td className="mono" style={{ fontSize: '0.82rem' }}>
                    active: #{def.active_revision?.revision ?? '—'} (gen {def.generation})
                  </td>
                  <td className="mono" style={{ fontSize: '0.8rem' }}>
                    {revisions.map((r) => `#${r.revision}`).join(', ') || '—'}
                  </td>
                  <td>
                    <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
                      {revisions.map((rev) => {
                        const fullRev =
                          def.latest_revision?.revision_id === rev.revision_id
                            ? def.latest_revision
                            : def.active_revision?.revision_id === rev.revision_id
                              ? def.active_revision
                              : null;
                        return (
                          <button
                            key={rev.revision_id}
                            type="button"
                            className="btn btn-secondary"
                            data-testid={`prepare-activate-${def.definition_id}-${rev.revision}`}
                            onClick={() =>
                              setPending({
                                definition: def,
                                targetSummary: rev,
                                targetRevision: fullRev,
                                currentRevision: currentRev,
                              })
                            }
                          >
                            {t('admin.policies.activateBtn')} #{rev.revision}
                          </button>
                        );
                      })}
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
