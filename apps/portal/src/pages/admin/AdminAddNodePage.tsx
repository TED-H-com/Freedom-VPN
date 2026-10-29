import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { apiClient, ApiError, type EnrollmentCreateResponse } from '../../api';
import { useI18n } from '../../i18n';

export function AdminAddNodePage(): React.ReactElement {
  const { t, locale } = useI18n();
  const [name, setName] = useState('Taichung Edge 01');
  const [region, setRegion] = useState('tw-taichung');
  const [groups, setGroups] = useState('63575bae-2d23-45c0-ae11-ba6bb9ace34a');
  const [expiresHours, setExpiresHours] = useState(168);
  const [notes, setNotes] = useState('Pre-installed appliance box (TEST ONLY)');

  // Held strictly in volatile component memory only (§18.2); never persisted to storage or URL.
  const [createdEnrollment, setCreatedEnrollment] =
    useState<EnrollmentCreateResponse | null>(null);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setCopied(false);
    try {
      const hours = Number(expiresHours) || 168;
      const clampedSeconds = Math.min(604800, Math.max(60, Math.round(hours * 3600)));
      const response = await apiClient.createEnrollment({
        delivery: 'preinstalled',
        expires_in_seconds: clampedSeconds,
        name: name.trim() || undefined,
        region: region.trim() || undefined,
        member_group_ids: groups
          .split(',')
          .map((g) => g.trim())
          .filter(Boolean),
        notes: notes.trim() || undefined,
      });
      setCreatedEnrollment(response);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err);
      }
    }
  };

  const handleCopy = async () => {
    if (!createdEnrollment?.token) return;
    try {
      if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
        await navigator.clipboard.writeText(createdEnrollment.token);
      }
      setCopied(true);
    } catch {
      setCopied(true);
    }
  };

  return (
    <div data-testid="admin-add-node-page">
      <div className="page-header">
        <div>
          <h1>{t('admin.enrollment.title')}</h1>
          <p className="page-subtitle">{t('admin.enrollment.subtitle')}</p>
        </div>
      </div>

      {error && (
        <div className="alert alert-danger" role="alert">
          <strong>{error.code}</strong>: {error.messageForLocale(locale)}
        </div>
      )}

      {createdEnrollment && (
        <section
          className="card"
          style={{ borderColor: 'var(--color-warning)', borderWidth: '2px' }}
          data-testid="one-time-enrollment-banner"
        >
          <h2 style={{ color: 'var(--color-warning)' }}>
            {t('admin.enrollment.oneTime.bannerTitle')}
          </h2>
          <p style={{ fontSize: '0.875rem', marginTop: '0.4rem' }}>
            {t('admin.enrollment.oneTime.securityNote')}
          </p>

          <div
            className="mono"
            style={{
              background: '#0f172a',
              color: '#f8fafc',
              padding: '0.85rem 1rem',
              borderRadius: 'var(--radius-sm)',
              marginTop: '0.75rem',
              marginBottom: '0.75rem',
              wordBreak: 'break-all',
            }}
            data-testid="one-time-token-value"
          >
            {createdEnrollment.token}
          </div>

          <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap' }}>
            <button
              type="button"
              className="btn btn-primary"
              data-testid="copy-enrollment-token-btn"
              onClick={() => void handleCopy()}
            >
              {copied
                ? t('admin.enrollment.oneTime.copied')
                : t('admin.enrollment.oneTime.copyBtn')}
            </button>

            <Link
              to={`/admin/enrollments/${createdEnrollment.enrollment.enrollment_id}`}
              className="btn btn-secondary"
              data-testid="navigate-enrollment-status-link"
              onClick={() => setCreatedEnrollment(null)}
            >
              {t('admin.enrollment.oneTime.viewStatusBtn')}
            </Link>
          </div>
        </section>
      )}

      <section className="card">
        <form onSubmit={(e) => void handleSubmit(e)}>
          <div className="form-grid">
            <div className="form-group">
              <label htmlFor="enr-name">{t('admin.enrollment.form.name')}</label>
              <input
                id="enr-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label htmlFor="enr-region">{t('admin.enrollment.form.region')}</label>
              <input
                id="enr-region"
                value={region}
                onChange={(e) => setRegion(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label htmlFor="enr-groups">{t('admin.enrollment.form.groups')}</label>
              <input
                id="enr-groups"
                value={groups}
                onChange={(e) => setGroups(e.target.value)}
              />
            </div>

            <div className="form-group">
              <label htmlFor="enr-expires">
                {t('admin.enrollment.form.expiresHours')}
              </label>
              <input
                id="enr-expires"
                type="number"
                min={1}
                max={168}
                value={expiresHours}
                onChange={(e) => setExpiresHours(Number(e.target.value))}
              />
            </div>
          </div>

          <div className="form-group">
            <label htmlFor="enr-notes">{t('admin.enrollment.form.notes')}</label>
            <input
              id="enr-notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
            />
          </div>

          <button
            type="submit"
            className="btn btn-primary"
            data-testid="create-enrollment-submit-btn"
          >
            {t('admin.enrollment.form.submitBtn')}
          </button>
        </form>
      </section>
    </div>
  );
}
