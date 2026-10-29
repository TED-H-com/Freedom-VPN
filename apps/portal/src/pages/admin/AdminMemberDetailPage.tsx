import React, { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, ApiError, type OperationResource } from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { ConflictDialog } from '../../components/common/ConflictDialog';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n } from '../../i18n';

export function AdminMemberDetailPage(): React.ReactElement {
  const { memberId = '' } = useParams<{ memberId: string }>();
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const [suspendReason, setSuspendReason] = useState(
    'Abuse ticket #991: terms violation review',
  );
  const [grantValidUntil, setGrantValidUntil] = useState('2026-12-31T00:00:00Z');
  const [grantReason, setGrantReason] = useState('Migration parallel transition grant');

  const [displayName, setDisplayName] = useState<string>('');
  const [memberGroups, setMemberGroups] = useState<string>('');
  const [notes, setNotes] = useState<string>('');

  const [activeOp, setActiveOp] = useState<OperationResource | null>(null);
  const [conflictError, setConflictError] = useState<ApiError | null>(null);
  const [error, setError] = useState<unknown>(null);

  const memberQuery = useQuery({
    queryKey: ['admin', 'member', memberId],
    queryFn: () => apiClient.getMember(memberId),
    enabled: Boolean(memberId),
  });

  const m = memberQuery.data?.data;
  const etag = memberQuery.data?.etag ?? undefined;

  useEffect(() => {
    if (m) {
      setDisplayName(m.display_name ?? '');
      setMemberGroups(m.member_group_ids.join(', '));
      setNotes(m.notes ?? '');
    }
  }, [m]);

  if (memberQuery.error) {
    return (
      <div data-testid="admin-member-detail-page">
        <ApiErrorAlert error={memberQuery.error} testId="member-detail-error" />
      </div>
    );
  }

  if (memberQuery.isLoading || !m) {
    return (
      <div className="card" role="status">
        Loading member detail...
      </div>
    );
  }

  const handleSuspend = async () => {
    setError(null);
    try {
      const res = await apiClient.suspendMember(m.member_id, suspendReason, etag);
      if ('operation_id' in res.data) {
        setActiveOp(res.data);
      }
      await queryClient.invalidateQueries({ queryKey: ['admin', 'member', memberId] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) setConflictError(err);
      else setError(err);
    }
  };

  const handleResume = async () => {
    setError(null);
    try {
      const res = await apiClient.resumeMember(
        m.member_id,
        'Review cleared by admin',
        etag,
      );
      if ('operation_id' in res.data) {
        setActiveOp(res.data);
      }
      await queryClient.invalidateQueries({ queryKey: ['admin', 'member', memberId] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) setConflictError(err);
      else setError(err);
    }
  };

  const handleSavePatch = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    try {
      await apiClient.updateMember(
        m.member_id,
        {
          display_name: displayName.trim() || undefined,
          member_group_ids: memberGroups
            .split(',')
            .map((g) => g.trim())
            .filter(Boolean),
          notes: notes.trim() || undefined,
        },
        etag,
        `patch-member:${m.member_id}:${m.resource_version}:${Date.now()}`,
      );
      await queryClient.invalidateQueries({ queryKey: ['admin', 'member', memberId] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) setConflictError(err);
      else setError(err);
    }
  };

  const handleCreateGrant = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    try {
      const op = await apiClient.createMemberGrant(m.member_id, {
        source: 'admin_grant',
        valid_until: grantValidUntil.trim(),
        reason: grantReason.trim(),
      });
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'member', memberId] });
    } catch (err) {
      setError(err);
    }
  };

  return (
    <div data-testid="admin-member-detail-page">
      <div style={{ marginBottom: '0.75rem' }}>
        <Link to="/admin/members">{t('admin.memberDetail.back')}</Link>
      </div>

      <div className="page-header">
        <div>
          <h1>
            {t('admin.memberDetail.title', { name: m.display_name ?? m.member_id })}
          </h1>
          <p className="page-subtitle mono">
            {m.member_id} · {m.email ?? '-'} · ETag: {etag ?? `"${m.resource_version}"`}
          </p>
        </div>
        <span
          className={`badge ${
            m.entitlement.state === 'active' ? 'badge-success' : 'badge-danger'
          }`}
          data-testid="member-detail-entitlement-badge"
        >
          {m.entitlement.state}
        </span>
      </div>

      <ConflictDialog
        error={conflictError}
        onReload={() => {
          setConflictError(null);
          void queryClient.invalidateQueries({ queryKey: ['admin', 'member', memberId] });
        }}
      />

      <OperationBanner
        operation={activeOp}
        onDismiss={() => setActiveOp(null)}
        onTerminal={() => {
          void queryClient.invalidateQueries({ queryKey: ['admin', 'member', memberId] });
        }}
      />

      <ApiErrorAlert error={error} testId="member-detail-error" />

      {/* Entitlement & Deadline Breakdown (§9.2, §15.2) */}
      <section className="card">
        <h2>{t('admin.memberDetail.entitlementBox')}</h2>
        <div className="form-grid" style={{ marginTop: '0.75rem' }}>
          <div className="mono">
            {t('admin.memberDetail.paidThrough', {
              time: m.entitlement.paid_through ?? 'None',
            })}
          </div>
          <div className="mono" data-testid="member-effective-service-end">
            {t('admin.memberDetail.effectiveEnd', {
              time: m.entitlement.effective_service_end ?? 'None',
            })}
          </div>
          <div className="mono">
            {t('admin.memberDetail.reconciledAt', {
              time: m.subscription?.last_reconciled_at ?? '-',
            })}
          </div>
        </div>
      </section>

      {/* Admin Suspend / Resume */}
      <section className="card">
        <h2>{t('admin.memberDetail.suspendSection')}</h2>
        <div className="form-group" style={{ marginTop: '0.75rem' }}>
          <label htmlFor="suspend-reason-input">
            {t('admin.memberDetail.suspendReasonLabel')}
          </label>
          <input
            id="suspend-reason-input"
            value={suspendReason}
            onChange={(e) => setSuspendReason(e.target.value)}
          />
        </div>
        <div style={{ display: 'flex', gap: '0.75rem' }}>
          <button
            type="button"
            className="btn btn-danger"
            data-testid="suspend-member-btn"
            onClick={() => void handleSuspend()}
          >
            {t('admin.memberDetail.suspendBtn')}
          </button>
          <button
            type="button"
            className="btn btn-secondary"
            data-testid="resume-member-btn"
            onClick={() => void handleResume()}
          >
            {t('admin.memberDetail.resumeBtn')}
          </button>
        </div>
      </section>

      {/* Member Profile, Groups & Notes Patch */}
      <section className="card">
        <h2>{t('admin.memberDetail.patchSection')}</h2>
        <div className="mono" style={{ fontSize: '0.8rem', marginTop: '0.35rem' }}>
          Authorized Devices: {m.authorized_device_count} /{' '}
          {m.entitlement.device_limit ?? '-'}
        </div>
        <form onSubmit={(e) => void handleSavePatch(e)} style={{ marginTop: '0.75rem' }}>
          <div className="form-grid">
            <div className="form-group">
              <label htmlFor="member-display-name-input">
                {t('admin.memberDetail.deviceLimitLabel')}
              </label>
              <input
                id="member-display-name-input"
                data-testid="member-display-name-input"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label htmlFor="member-groups-input">
                {t('admin.memberDetail.clearDeviceLimit')}
              </label>
              <input
                id="member-groups-input"
                data-testid="member-groups-input"
                className="mono"
                value={memberGroups}
                onChange={(e) => setMemberGroups(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label htmlFor="member-notes-input">
                {t('admin.memberDetail.notesLabel')}
              </label>
              <input
                id="member-notes-input"
                data-testid="member-notes-input"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
              />
            </div>
          </div>
          <button
            type="submit"
            className="btn btn-primary"
            data-testid="save-member-patch-btn"
          >
            {t('admin.memberDetail.savePatchBtn')}
          </button>
        </form>
      </section>

      {/* Finite Entitlement Grant */}
      <section className="card">
        <h2>{t('admin.memberDetail.grantSection')}</h2>
        <form
          onSubmit={(e) => void handleCreateGrant(e)}
          style={{ marginTop: '0.75rem' }}
        >
          <div className="form-grid">
            <div className="form-group">
              <label htmlFor="grant-valid-until">
                {t('admin.memberDetail.grantValidUntil')}
              </label>
              <input
                id="grant-valid-until"
                className="mono"
                value={grantValidUntil}
                onChange={(e) => setGrantValidUntil(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label htmlFor="grant-reason">{t('admin.memberDetail.grantReason')}</label>
              <input
                id="grant-reason"
                value={grantReason}
                onChange={(e) => setGrantReason(e.target.value)}
              />
            </div>
          </div>
          <button
            type="submit"
            className="btn btn-primary"
            data-testid="create-grant-btn"
          >
            {t('admin.memberDetail.grantSubmitBtn')}
          </button>
        </form>
      </section>

      {/* Member Entitlement Grants */}
      <section className="card">
        <h2>{t('admin.memberDetail.devicesSection')}</h2>
        <div className="table-wrapper" style={{ marginTop: '0.75rem' }}>
          <table className="data-table">
            <thead>
              <tr>
                <th>Grant ID</th>
                <th>Source</th>
                <th>Reason</th>
                <th>Valid Window</th>
              </tr>
            </thead>
            <tbody>
              {m.grants.map((grant) => (
                <tr
                  key={grant.grant_id}
                  data-testid={`admin-grant-row-${grant.grant_id}`}
                >
                  <td className="mono" style={{ fontSize: '0.75rem' }}>
                    {grant.grant_id}
                  </td>
                  <td>
                    <span className="badge badge-info">{grant.source}</span>
                  </td>
                  <td>{grant.reason}</td>
                  <td className="mono" style={{ fontSize: '0.75rem' }}>
                    {grant.valid_from} → {grant.valid_until}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
