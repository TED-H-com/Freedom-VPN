import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { portalApi, type EnrollmentCreatedResponse } from '../../api';
import { useI18n } from '../../i18n';

/**
 * Add Node Enrollment screen (§18.2).
 *
 * Security contract:
 * - The enrollment token is returned once on POST (201 Created, `Cache-Control: no-store`).
 * - Stored ONLY in transient component memory (`createdEnrollment`), never in `localStorage`,
 *   `sessionStorage`, URL query parameters, or logs.
 * - Navigating away to `/admin/enrollments/:enrollmentId` discards the in-memory token and
 *   fetches the server status (`secret_available: false`).
 */
export function AdminAddNodePage(): React.ReactElement {
  const { t } = useI18n();
  const navigate = useNavigate();

  const [name, setName] = useState('');
  const [region, setRegion] = useState('');
  const [groups, setGroups] = useState('standard');
  const [expiresHours, setExpiresHours] = useState('168');
  const [notes, setNotes] = useState('');

  const [createdEnrollment, setCreatedEnrollment] =
    useState<EnrollmentCreatedResponse | null>(null);
  const [copied, setCopied] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const res = await portalApi.createEnrollment({
      name: name.trim() || undefined,
      region: region.trim() || undefined,
      member_groups: groups
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      expires_in_hours: Number.parseInt(expiresHours, 10) || 168,
      notes: notes.trim() || undefined,
    });
    setCreatedEnrollment(res.data);
    setCopied(false);
  };

  const handleCopyToken = async () => {
    if (!createdEnrollment) return;
    if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
      await navigator.clipboard.writeText(createdEnrollment.enrollment_token);
    }
    setCopied(true);
  };

  const handleNavigateToStatusPage = () => {
    if (!createdEnrollment) return;
    const id = createdEnrollment.enrollment_id;
    setCreatedEnrollment(null);
    void navigate(`/admin/enrollments/${id}`);
  };

  return (
    <div>
      <header className="page-header">
        <h1>{t('admin.enrollment.title')}</h1>
        <p className="page-subtitle">{t('admin.enrollment.subtitle')}</p>
      </header>

      {createdEnrollment ? (
        <section
          className="card alert-warning"
          aria-labelledby="one-time-token-heading"
          data-testid="one-time-token-panel"
        >
          <h2 id="one-time-token-heading">{t('admin.enrollment.oneTime.bannerTitle')}</h2>
          <p>{t('admin.enrollment.oneTime.securityNote')}</p>

          <div className="code-box" data-testid="one-time-token-value">
            {createdEnrollment.enrollment_token}
          </div>

          <div style={{ marginTop: 12, fontSize: '0.88rem' }}>
            Enrollment ID: <code>{createdEnrollment.enrollment_id}</code> ｜ Reserved Node
            ID: <code>{createdEnrollment.node_id}</code> ｜ Expires:{' '}
            <code>{createdEnrollment.expires_at}</code>
          </div>

          <div className="btn-row" style={{ marginTop: 16 }}>
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => void handleCopyToken()}
              data-testid="copy-enrollment-token-btn"
            >
              {copied
                ? t('admin.enrollment.oneTime.copied')
                : t('admin.enrollment.oneTime.copyBtn')}
            </button>
            <button
              type="button"
              className="btn"
              onClick={handleNavigateToStatusPage}
              data-testid="reenter-enrollment-status-btn"
            >
              {t('admin.enrollment.oneTime.viewStatusBtn')}
            </button>
          </div>
        </section>
      ) : (
        <form className="card" onSubmit={(e) => void handleSubmit(e)}>
          <div className="form-grid">
            <div className="form-field">
              <label htmlFor="enr-name">{t('admin.enrollment.form.name')}</label>
              <input
                id="enr-name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="tw-tpe-02"
              />
            </div>
            <div className="form-field">
              <label htmlFor="enr-region">{t('admin.enrollment.form.region')}</label>
              <input
                id="enr-region"
                type="text"
                value={region}
                onChange={(e) => setRegion(e.target.value)}
                placeholder="TW-Taipei"
              />
            </div>
            <div className="form-field">
              <label htmlFor="enr-groups">{t('admin.enrollment.form.groups')}</label>
              <input
                id="enr-groups"
                type="text"
                value={groups}
                onChange={(e) => setGroups(e.target.value)}
              />
            </div>
            <div className="form-field">
              <label htmlFor="enr-expires">
                {t('admin.enrollment.form.expiresHours')}
              </label>
              <input
                id="enr-expires"
                type="number"
                min={1}
                max={720}
                value={expiresHours}
                onChange={(e) => setExpiresHours(e.target.value)}
              />
            </div>
            <div className="form-field">
              <label htmlFor="enr-notes">{t('admin.enrollment.form.notes')}</label>
              <input
                id="enr-notes"
                type="text"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
              />
            </div>
          </div>
          <div className="btn-row">
            <button
              type="submit"
              className="btn btn-primary"
              data-testid="create-enrollment-submit-btn"
            >
              {t('admin.enrollment.form.submitBtn')}
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
