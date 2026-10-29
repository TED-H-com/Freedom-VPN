import React, { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { useI18n } from '../../i18n';

export function AdminEnrollmentDetailPage(): React.ReactElement {
  const { enrollmentId = '' } = useParams<{ enrollmentId: string }>();
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [actionError, setActionError] = useState<unknown>(null);

  const enrollmentQuery = useQuery({
    queryKey: ['admin', 'enrollment', enrollmentId],
    queryFn: () => apiClient.getEnrollment(enrollmentId),
    enabled: Boolean(enrollmentId),
  });

  if (enrollmentQuery.error) {
    return (
      <div data-testid="admin-enrollment-detail-page">
        <ApiErrorAlert error={enrollmentQuery.error} testId="enrollment-detail-error" />
      </div>
    );
  }

  if (enrollmentQuery.isLoading || !enrollmentQuery.data) {
    return (
      <div className="card" role="status">
        Loading enrollment status...
      </div>
    );
  }

  const record = enrollmentQuery.data;

  const handleRevoke = async () => {
    setActionError(null);
    try {
      await apiClient.revokeEnrollment(record.enrollment_id);
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'enrollment', enrollmentId],
      });
    } catch (err) {
      setActionError(err);
    }
  };

  return (
    <div data-testid="admin-enrollment-detail-page">
      <div style={{ marginBottom: '0.75rem' }}>
        <Link to="/admin/nodes">{t('admin.nodeDetail.back')}</Link>
      </div>

      <div className="page-header">
        <div>
          <h1>{t('admin.enrollment.detail.title')}</h1>
          <p className="page-subtitle mono">ID: {record.enrollment_id}</p>
        </div>
        <span className="badge badge-info" data-testid="enrollment-state-badge">
          {t('admin.enrollment.detail.stateLabel', { state: record.state })}
        </span>
      </div>

      <div
        className="alert alert-warning"
        role="status"
        data-testid="enrollment-secret-gone-notice"
      >
        {t('admin.enrollment.detail.secretGone')}
      </div>

      <ApiErrorAlert error={actionError} testId="enrollment-revoke-error" />

      <div className="card">
        <dl className="form-grid">
          <div>
            <dt style={{ fontWeight: 600 }}>Pre-classified Name</dt>
            <dd>{record.name ?? '-'}</dd>
          </div>
          <div>
            <dt style={{ fontWeight: 600 }}>Pre-classified Region</dt>
            <dd>{record.region ?? '-'}</dd>
          </div>
          <div>
            <dt style={{ fontWeight: 600 }}>Expires At (UTC)</dt>
            <dd className="mono">{record.expires_at}</dd>
          </div>
          <div>
            <dt style={{ fontWeight: 600 }}>Secret Available</dt>
            <dd className="mono" data-testid="secret-available-flag">
              false
            </dd>
          </div>
        </dl>

        {record.state === 'pending' && (
          <button
            type="button"
            className="btn btn-danger"
            data-testid="revoke-enrollment-btn"
            onClick={() => void handleRevoke()}
          >
            {t('admin.enrollment.detail.revokeBtn')}
          </button>
        )}
      </div>
    </div>
  );
}
