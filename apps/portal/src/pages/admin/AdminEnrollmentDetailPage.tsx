import React, { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { portalApi, type EnrollmentDetail } from '../../api';
import { useI18n } from '../../i18n';

export function AdminEnrollmentDetailPage(): React.ReactElement {
  const { enrollmentId = '' } = useParams<{ enrollmentId: string }>();
  const { t } = useI18n();
  const [detail, setDetail] = useState<EnrollmentDetail | null>(null);

  const loadDetail = useCallback(async () => {
    if (!enrollmentId) return;
    const res = await portalApi.getEnrollment(enrollmentId);
    setDetail(res.data);
  }, [enrollmentId]);

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  if (!detail) {
    return <div className="card">Loading enrollment status...</div>;
  }

  const handleRevoke = async () => {
    const res = await portalApi.revokeEnrollment(detail.enrollment_id);
    setDetail(res.data);
  };

  return (
    <div>
      <header className="page-header">
        <h1>{t('admin.enrollment.detail.title')}</h1>
      </header>

      <section className="card" data-testid="enrollment-status-card">
        <div className="alert alert-info" data-testid="enrollment-secret-gone-notice">
          {t('admin.enrollment.detail.secretGone')}
        </div>

        <p data-testid="enrollment-state-display">
          <strong>
            {t('admin.enrollment.detail.stateLabel', { state: detail.state })}
          </strong>
        </p>
        <p>
          Enrollment ID: <code>{detail.enrollment_id}</code> ｜ Reserved Node ID:{' '}
          <code>{detail.node_id}</code>
        </p>
        <p>
          Name: <strong>{detail.name}</strong> ｜ Region: <strong>{detail.region}</strong>{' '}
          ｜ Expires: <code>{detail.expires_at}</code>
        </p>

        <div className="btn-row" style={{ marginTop: 14 }}>
          {detail.state === 'unused' && (
            <button
              type="button"
              className="btn btn-danger"
              onClick={() => void handleRevoke()}
              data-testid="revoke-enrollment-btn"
            >
              {t('admin.enrollment.detail.revokeBtn')}
            </button>
          )}
          <Link to="/admin/nodes/new" className="btn">
            {t('nav.admin.addNode')}
          </Link>
        </div>
      </section>
    </div>
  );
}
