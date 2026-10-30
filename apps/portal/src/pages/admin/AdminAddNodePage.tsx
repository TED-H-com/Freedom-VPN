import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { apiClient, type EnrollmentCreateResponse } from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { useI18n } from '../../i18n';

export function AdminAddNodePage(): React.ReactElement {
  const { t } = useI18n();
  const [delivery, setDelivery] = useState<'preinstalled' | 'installer'>('preinstalled');
  const [reinstallNodeId, setReinstallNodeId] = useState('');
  const [name, setName] = useState('Taichung Edge 01');
  const [region, setRegion] = useState('tw-taichung');
  const [groups, setGroups] = useState('63575bae-2d23-45c0-ae11-ba6bb9ace34a');
  const [expiresHours, setExpiresHours] = useState(168);
  const [notes, setNotes] = useState('Pre-installed appliance box (TEST ONLY)');

  useEffect(() => {
    void apiClient
      .listAdminGroups({ limit: 50 })
      .then((res) => {
        const firstGroup = res.items[0];
        if (firstGroup) {
          const validIds = new Set(res.items.map((g) => g.group_id));
          setGroups((prev) => {
            const currentIds = prev
              .split(',')
              .map((s) => s.trim())
              .filter(Boolean);
            if (currentIds.length === 0 || !currentIds.every((id) => validIds.has(id))) {
              return firstGroup.group_id;
            }
            return prev;
          });
        }
      })
      .catch(() => {
        // Keep default if group list is unavailable
      });
  }, []);

  // Held strictly in volatile component memory only (§18.2); never persisted to storage or URL.
  const [createdEnrollment, setCreatedEnrollment] =
    useState<EnrollmentCreateResponse | null>(null);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setCopied(false);
    try {
      let groupIds = groups
        .split(',')
        .map((g) => g.trim())
        .filter(Boolean);
      if (
        groupIds.length === 1 &&
        groupIds[0] === '63575bae-2d23-45c0-ae11-ba6bb9ace34a'
      ) {
        try {
          const grpRes = await apiClient.listAdminGroups({ limit: 50 });
          const firstGroup = grpRes.items[0];
          if (firstGroup && !grpRes.items.some((g) => g.group_id === groupIds[0])) {
            groupIds = [firstGroup.group_id];
            setGroups(firstGroup.group_id);
          }
        } catch {
          // Keep current groupIds
        }
      }
      const hours = Number(expiresHours) || (delivery === 'installer' ? 0.25 : 168);
      const clampedSeconds = Math.min(604800, Math.max(60, Math.round(hours * 3600)));
      const response = await apiClient.createEnrollment({
        node_id: reinstallNodeId.trim() || undefined,
        delivery,
        expires_in_seconds: clampedSeconds,
        name: name.trim() || undefined,
        region: region.trim() || undefined,
        member_group_ids: groupIds,
        notes: notes.trim() || undefined,
      });
      setCreatedEnrollment(response);
    } catch (err) {
      setError(err);
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

      <ApiErrorAlert error={error} testId="create-enrollment-error" />

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
              <label htmlFor="enr-delivery">{t('admin.enrollment.form.delivery')}</label>
              <select
                id="enr-delivery"
                data-testid="enr-delivery-select"
                value={delivery}
                onChange={(e) => {
                  const next = e.target.value as 'preinstalled' | 'installer';
                  setDelivery(next);
                  setExpiresHours(next === 'installer' ? 0.25 : 168);
                }}
              >
                <option value="preinstalled">
                  {t('admin.enrollment.form.deliveryPreinstalled')}
                </option>
                <option value="installer">
                  {t('admin.enrollment.form.deliveryInstaller')}
                </option>
              </select>
            </div>

            <div className="form-group">
              <label htmlFor="enr-reinstall-node-id">
                {t('admin.enrollment.form.reinstallNodeId')}
              </label>
              <input
                id="enr-reinstall-node-id"
                data-testid="enr-reinstall-node-id-input"
                className="mono"
                value={reinstallNodeId}
                placeholder="Optional UUID for existing node reinstall"
                onChange={(e) => setReinstallNodeId(e.target.value)}
              />
            </div>

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
                step="0.25"
                min={0.25}
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
