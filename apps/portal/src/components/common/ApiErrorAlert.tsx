import React from 'react';
import { ApiError } from '../../api';
import { useI18n } from '../../i18n';

interface ApiErrorAlertProps {
  error: unknown;
  testId?: string;
  onRetry?: () => void;
}

export function ApiErrorAlert({
  error,
  testId = 'api-error-alert',
  onRetry,
}: ApiErrorAlertProps): React.ReactElement | null {
  const { t, locale } = useI18n();

  if (!error) {
    return null;
  }

  const apiErr = error instanceof ApiError ? error : null;
  const code = apiErr?.code ?? 'INTERNAL_ERROR';
  const requestId = apiErr?.requestId ?? 'unknown';
  const humanMessage = apiErr
    ? apiErr.messageForLocale(locale)
    : error instanceof Error
      ? error.message
      : String(error);
  const retryAfterSeconds = apiErr?.retryAfterSeconds ?? null;

  return (
    <div className="alert alert-danger" role="alert" data-testid={testId}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          gap: '12px',
          flexWrap: 'wrap',
        }}
      >
        <div>
          <strong>{t('api.error.title')}: </strong>
          <span data-testid={`${testId}-message`}>{humanMessage}</span>
          {apiErr?.serverMessage && apiErr.serverMessage !== humanMessage && (
            <div style={{ fontSize: '0.85rem', marginTop: '4px' }}>
              {apiErr.serverMessage}
            </div>
          )}
          <div style={{ fontSize: '0.82rem', marginTop: '6px' }}>
            <span data-testid={`${testId}-code`}>
              {t('api.error.codeLabel', { code })}
            </span>
            <span style={{ marginLeft: '12px' }} data-testid={`${testId}-request-id`}>
              {t('api.error.requestIdLabel', { requestId })}
            </span>
          </div>
          {retryAfterSeconds !== null && (
            <div
              style={{ fontSize: '0.85rem', fontWeight: 700, marginTop: '6px' }}
              data-testid={`${testId}-retry-after`}
            >
              {t('api.error.retryAfterNotice', { seconds: retryAfterSeconds })}
            </div>
          )}
        </div>
        {onRetry && (
          <button
            type="button"
            className="btn btn-secondary"
            data-testid={`${testId}-retry-btn`}
            onClick={onRetry}
          >
            Retry
          </button>
        )}
      </div>
    </div>
  );
}
