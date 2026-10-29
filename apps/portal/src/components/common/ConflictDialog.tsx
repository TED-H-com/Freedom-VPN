import React from 'react';
import type { ApiError } from '../../api';
import { useI18n } from '../../i18n';

interface ConflictDialogProps {
  conflictError: ApiError | null;
  onReloadAndDismiss: () => void;
}

export function ConflictDialog({
  conflictError,
  onReloadAndDismiss,
}: ConflictDialogProps): React.ReactElement | null {
  const { t } = useI18n();

  if (!conflictError) {
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
        <h2 id="conflict-dialog-title" style={{ marginTop: 0, color: 'var(--danger)' }}>
          {t('conflict.dialog.title')}
        </h2>
        <p>{t('conflict.dialog.description')}</p>
        {conflictError.currentResourceVersion && (
          <p>
            <strong>
              {t('conflict.dialog.currentVersion', {
                version: conflictError.currentResourceVersion,
              })}
            </strong>
          </p>
        )}
        <p style={{ fontSize: '0.84rem', color: 'var(--text-secondary)' }}>
          {t('op.requestIdLabel')}: <code>{conflictError.request_id}</code>
        </p>
        <div className="btn-row" style={{ marginTop: '18px' }}>
          <button
            type="button"
            className="btn btn-primary"
            onClick={onReloadAndDismiss}
            data-testid="conflict-reload-btn"
          >
            {t('conflict.dialog.reloadBtn')}
          </button>
        </div>
      </div>
    </div>
  );
}
