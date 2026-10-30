import React, { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { apiClient, ApiError, type FleetConfig } from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { ConflictDialog } from '../../components/common/ConflictDialog';
import { useI18n } from '../../i18n';
import { formatLocaleWithUtc } from '../../lib/datetime';

const SUB_900_WARNING_THRESHOLD_SECONDS = 900;

function extractValidationField(
  err: ApiError,
): 'max_offline_lease_seconds' | 'reason' | 'generic' {
  const directField = err.details.field;
  if (typeof directField === 'string') {
    if (directField.includes('max_offline_lease_seconds')) {
      return 'max_offline_lease_seconds';
    }
    if (directField.includes('reason')) {
      return 'reason';
    }
  }

  const candidateLists = [err.details.errors, err.details.fields];
  for (const list of candidateLists) {
    if (Array.isArray(list)) {
      for (const item of list) {
        if (item && typeof item === 'object') {
          const fieldVal = (item as Record<string, unknown>).field;
          if (typeof fieldVal === 'string') {
            if (fieldVal.includes('max_offline_lease_seconds')) {
              return 'max_offline_lease_seconds';
            }
            if (fieldVal.includes('reason')) {
              return 'reason';
            }
          }
        }
      }
    }
  }

  return 'generic';
}

export const AdminSettingsPage: React.FC = () => {
  const { t, locale } = useI18n();
  const queryClient = useQueryClient();

  const [secondsInput, setSecondsInput] = useState<string>('');
  const [reasonInput, setReasonInput] = useState<string>('');
  const [readEtag, setReadEtag] = useState<string | null>(null);
  const [initializedFromServer, setInitializedFromServer] = useState(false);
  const [conflictError, setConflictError] = useState<ApiError | null>(null);
  const [apiError, setApiError] = useState<ApiError | null>(null);
  const [savedConfig, setSavedConfig] = useState<FleetConfig | null>(null);

  const fleetConfigQuery = useQuery({
    queryKey: ['admin', 'fleetConfig'],
    queryFn: () => apiClient.getFleetConfig(),
  });

  const config = fleetConfigQuery.data?.data;
  const latestEtag =
    fleetConfigQuery.data?.etag ?? (config ? `"${config.resource_version}"` : null);

  useEffect(() => {
    if (config && !initializedFromServer) {
      setSecondsInput(String(config.max_offline_lease_seconds));
      setReadEtag(latestEtag);
      setInitializedFromServer(true);
    }
  }, [config, initializedFromServer, latestEtag]);

  const effectiveIfMatch = readEtag ?? latestEtag ?? undefined;

  const parsedInputNumber =
    secondsInput.trim() === '' ? Number.NaN : Number(secondsInput.trim());
  const isSub900WarningVisible =
    (Number.isFinite(parsedInputNumber) &&
      parsedInputNumber < SUB_900_WARNING_THRESHOLD_SECONDS) ||
    (config !== undefined &&
      config.max_offline_lease_seconds < SUB_900_WARNING_THRESHOLD_SECONDS &&
      !Number.isFinite(parsedInputNumber));

  const updateMutation = useMutation({
    mutationFn: async (params: {
      maxOfflineLeaseSeconds: number;
      reason: string;
      ifMatch?: string;
    }) => {
      const freshSubmissionActionId = `fleet-config:update:${Date.now()}-${Math.random()
        .toString(36)
        .slice(2, 10)}`;
      return apiClient.updateFleetConfig(
        {
          max_offline_lease_seconds: params.maxOfflineLeaseSeconds,
          reason: params.reason,
        },
        params.ifMatch,
        freshSubmissionActionId,
      );
    },
    onSuccess: (res) => {
      setConflictError(null);
      setApiError(null);
      setSavedConfig(res.data);
      const nextEtag = res.etag ?? `"${res.data.resource_version}"`;
      setReadEtag(nextEtag);
      setSecondsInput(String(res.data.max_offline_lease_seconds));
      queryClient.setQueryData(['admin', 'fleetConfig'], res);
      void queryClient.invalidateQueries({ queryKey: ['admin', 'audit'] });
      void queryClient.invalidateQueries({ queryKey: ['admin', 'auditEvents'] });
    },
    onError: (err) => {
      if (err instanceof ApiError) {
        if (err.status === 412 || err.code === 'RESOURCE_VERSION_CONFLICT') {
          setConflictError(err);
          setApiError(null);
          return;
        }
        setApiError(err);
      }
    },
  });

  const handleReloadAfterConflict = async () => {
    const refetched = await fleetConfigQuery.refetch();
    const freshData = refetched.data?.data;
    const freshEtag =
      refetched.data?.etag ?? (freshData ? `"${freshData.resource_version}"` : null);
    if (freshData) {
      setSecondsInput(String(freshData.max_offline_lease_seconds));
      setReadEtag(freshEtag);
    }
    setConflictError(null);
    setApiError(null);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setApiError(null);
    setConflictError(null);
    setSavedConfig(null);

    const numericSeconds =
      secondsInput.trim() === '' ? 0 : Math.trunc(Number(secondsInput.trim()));
    updateMutation.mutate({
      maxOfflineLeaseSeconds: Number.isFinite(numericSeconds) ? numericSeconds : 0,
      reason: reasonInput,
      ifMatch: effectiveIfMatch,
    });
  };

  if (fleetConfigQuery.isLoading) {
    return null;
  }

  if (fleetConfigQuery.error instanceof ApiError && !config) {
    return (
      <ApiErrorAlert error={fleetConfigQuery.error} testId="fleet-config-load-error" />
    );
  }

  if (!config) {
    return null;
  }

  const validationField =
    apiError && (apiError.status === 422 || apiError.code === 'VALIDATION_FAILED')
      ? extractValidationField(apiError)
      : null;

  return (
    <div data-testid="admin-settings-page">
      <div className="page-header">
        <div>
          <h1 className="page-title">{t('admin.settings.title')}</h1>
          <p className="page-subtitle">{t('admin.settings.subtitle')}</p>
        </div>
        <span className="badge badge-neutral" data-testid="fleet-config-etag">
          ETag: {effectiveIfMatch ?? `"${config.resource_version}"`}
        </span>
      </div>

      {conflictError && (
        <div
          className="alert alert-danger"
          role="alert"
          data-testid="fleet-config-conflict-banner"
        >
          <div className="alert-title">
            {t('admin.settings.conflictBanner', {
              version: conflictError.currentResourceVersion ?? 'unknown',
            })}
          </div>
          <div style={{ marginTop: '0.35rem', fontSize: '0.85rem' }}>
            <code>
              current_resource_version:{' '}
              <span data-testid="fleet-config-conflict-version">
                {conflictError.currentResourceVersion ?? 'unknown'}
              </span>
            </code>
          </div>
          <div className="btn-group" style={{ marginTop: '0.5rem' }}>
            <button
              type="button"
              className="btn btn-primary btn-sm"
              onClick={() => void handleReloadAfterConflict()}
              data-testid="fleet-config-reload-btn"
            >
              {t('admin.settings.conflictReloadBtn')}
            </button>
          </div>
        </div>
      )}

      <ConflictDialog
        error={conflictError}
        onReload={() => void handleReloadAfterConflict()}
      />

      {apiError && <ApiErrorAlert error={apiError} testId="fleet-config-api-error" />}

      {savedConfig && (
        <div
          className="alert alert-info"
          role="status"
          data-testid="fleet-config-saved-banner"
        >
          <div className="alert-title">
            {t('admin.settings.savedBanner', {
              seconds: savedConfig.max_offline_lease_seconds,
              version: savedConfig.resource_version,
            })}
          </div>
          <div style={{ marginTop: '0.35rem' }}>
            <Link
              to="/admin/audit?resource_type=fleet_config"
              data-testid="fleet-config-audit-link"
            >
              {t('admin.settings.auditLinkText')}
            </Link>
          </div>
        </div>
      )}

      <div
        className="card"
        style={{ marginBottom: '1rem' }}
        data-testid="fleet-config-current-card"
      >
        <h2 className="card-title">{t('admin.settings.currentCardTitle')}</h2>
        <div className="grid-3">
          <div>
            <div style={{ fontSize: '0.82rem', color: 'var(--color-text-muted)' }}>
              {t('admin.settings.currentSecondsLabel')}
            </div>
            <div
              style={{ fontSize: '1.25rem', fontWeight: 700, marginTop: '0.2rem' }}
              data-testid="fleet-config-current-seconds"
            >
              {config.max_offline_lease_seconds} s
            </div>
          </div>

          <div>
            <div style={{ fontSize: '0.82rem', color: 'var(--color-text-muted)' }}>
              {t('admin.settings.defaultSecondsLabel')}
            </div>
            <div
              style={{ fontSize: '1.25rem', fontWeight: 700, marginTop: '0.2rem' }}
              data-testid="fleet-config-default-seconds"
            >
              {config.default_max_offline_lease_seconds} s
            </div>
          </div>

          <div>
            <div style={{ fontSize: '0.82rem', color: 'var(--color-text-muted)' }}>
              {t('admin.settings.updatedAtLabel')}
            </div>
            <div
              style={{ fontSize: '0.95rem', fontWeight: 600, marginTop: '0.2rem' }}
              data-testid="fleet-config-updated-at"
            >
              {formatLocaleWithUtc(config.updated_at, locale)}
            </div>
            <div style={{ fontSize: '0.78rem', color: 'var(--color-text-muted)' }}>
              {t('admin.settings.resourceVersionLabel')}:{' '}
              <code data-testid="fleet-config-resource-version">
                {config.resource_version}
              </code>
            </div>
          </div>
        </div>

        <div
          className="alert alert-info"
          style={{ marginTop: '1rem', marginBottom: 0 }}
          data-testid="fleet-config-semantics-copy"
        >
          <div className="alert-title">{t('admin.settings.semanticsTitle')}</div>
          <ul style={{ margin: '0.4rem 0 0 1.2rem', padding: 0 }}>
            <li>{t('admin.settings.semanticsNewLeasesOnly')}</li>
            <li>{t('admin.settings.semanticsExistingLeases')}</li>
            <li>{t('admin.settings.semanticsCeilingNote')}</li>
          </ul>
        </div>
      </div>

      {isSub900WarningVisible && (
        <div
          className="alert alert-warning"
          role="status"
          data-testid="fleet-config-sub-900-warning"
        >
          <div className="alert-title">{t('admin.settings.sub900Warning')}</div>
        </div>
      )}

      <div className="card" data-testid="fleet-config-form-card">
        <h2 className="card-title">{t('admin.settings.formTitle')}</h2>

        {validationField && (
          <div
            className="alert alert-danger"
            role="alert"
            data-testid="fleet-config-field-error"
          >
            {validationField === 'max_offline_lease_seconds'
              ? t('admin.settings.fieldErrorSeconds')
              : validationField === 'reason'
                ? t('admin.settings.fieldErrorReason')
                : t('admin.settings.fieldErrorGeneric')}
            {apiError?.serverMessage && (
              <div style={{ marginTop: '0.25rem', fontSize: '0.82rem' }}>
                <code>{apiError.serverMessage}</code>
              </div>
            )}
          </div>
        )}

        <form onSubmit={handleSubmit} noValidate data-testid="fleet-config-form">
          <div className="grid-2">
            <div className="form-group">
              <label className="form-label" htmlFor="fleet-config-lease-seconds-input">
                {t('admin.settings.leaseSecondsInputLabel')}
              </label>
              <input
                id="fleet-config-lease-seconds-input"
                type="number"
                className="form-input"
                value={secondsInput}
                onChange={(e) => setSecondsInput(e.target.value)}
                data-testid="fleet-config-lease-seconds-input"
              />
              {validationField === 'max_offline_lease_seconds' && (
                <div
                  style={{
                    color: 'var(--color-danger)',
                    fontSize: '0.82rem',
                    marginTop: '0.25rem',
                  }}
                  data-testid="fleet-config-seconds-field-error"
                >
                  {t('admin.settings.fieldErrorSeconds')}
                </div>
              )}
            </div>

            <div className="form-group">
              <label className="form-label" htmlFor="fleet-config-reason-input">
                {t('admin.settings.reasonInputLabel')}
              </label>
              <input
                id="fleet-config-reason-input"
                type="text"
                className="form-input"
                value={reasonInput}
                onChange={(e) => setReasonInput(e.target.value)}
                placeholder={t('admin.settings.reasonPlaceholder')}
                data-testid="fleet-config-reason-input"
              />
              {validationField === 'reason' && (
                <div
                  style={{
                    color: 'var(--color-danger)',
                    fontSize: '0.82rem',
                    marginTop: '0.25rem',
                  }}
                  data-testid="fleet-config-reason-field-error"
                >
                  {t('admin.settings.fieldErrorReason')}
                </div>
              )}
            </div>
          </div>

          <div className="btn-group">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={updateMutation.isPending}
              data-testid="fleet-config-submit-btn"
            >
              {t('admin.settings.submitBtn')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
