import React from 'react';
import type { ApiError } from '../../api';
import { useI18n } from '../../i18n';

interface ConflictDialogProps {
  error?: ApiError | null;
  conflictError?: ApiError | null;
  onReload?: () => void;
  onReloadAndDismiss?: () => void;
}

export function ConflictDialog({
  error,
  conflictError,
  onReload,
  onReloadAndDismiss,
}: ConflictDialogProps): React.ReactElement | null {
  const { t } = useI18n();
  const activeError = error ?? conflictError ?? null;
  const handleReload = onReload ?? onReloadAndDismiss;

  if (!activeError) {
    return null;
  }

  return (
    <div
      className="modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="conflict-dialog-title"
      data-testid="conflict-dialog"
    >
      <div className="modal-dialog">
        <h2
          id="conflict-dialog-title"
          style={{ marginTop: 0, color: 'var(--color-danger)' }}
        >
          {t('conflict.dialog.title')}
        </h2>
        <p>{t('conflict.dialog.description')}</p>
        {activeError.currentResourceVersion && (
          <p>
            <strong>
              {t('conflict.dialog.currentVersion', {
                version: activeError.currentResourceVersion,
              })}
            </strong>
          </p>
        )}
        <p style={{ fontSize: '0.84rem', color: 'var(--color-text-muted)' }}>
          {t('op.requestIdLabel')}: <code>{activeError.requestId}</code>
        </p>
        <div className="btn-row" style={{ marginTop: '18px' }}>
          <button
            type="button"
            className="btn btn-primary"
            onClick={handleReload}
            data-testid="conflict-reload-btn"
          >
            {t('conflict.dialog.reloadBtn')}
          </button>
        </div>
      </div>
    </div>
  );
}
