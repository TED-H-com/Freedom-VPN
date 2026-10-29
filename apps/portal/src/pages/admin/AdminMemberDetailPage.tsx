import React, { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { portalApi, useOperationPoller, type MemberItem } from '../../api';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n } from '../../i18n';
import { formatBytesDual } from '../../lib/decimal';

export function AdminMemberDetailPage(): React.ReactElement {
  const { memberId = '' } = useParams<{ memberId: string }>();
  const { t } = useI18n();
  const [member, setMember] = useState<MemberItem | null>(null);
  const [suspendReason, setSuspendReason] = useState('管理員安全調查停權');
  const [grantUntil, setGrantUntil] = useState('2026-12-05T00:00:00Z');
  const [grantReason, setGrantReason] = useState(
    '既有手動 WireGuard 會員移轉並行期有限授權 (§20.3)',
  );

  const loadMember = useCallback(async () => {
    if (!memberId) return;
    const res = await portalApi.getAdminMember(memberId);
    setMember(res.data);
  }, [memberId]);

  const poller = useOperationPoller({
    scope: 'admin',
    onTerminal: () => {
      void loadMember();
    },
  });

  useEffect(() => {
    void loadMember();
  }, [loadMember]);

  if (!member) {
    return <div className="card">Loading member detail...</div>;
  }

  const handleToggleSuspend = async (suspend: boolean) => {
    const opRes = await portalApi.patchAdminMember(
      member.member_id,
      {
        admin_suspended: suspend,
        admin_suspension_reason: suspend ? suspendReason : undefined,
      },
      { ifMatch: member.etag },
    );
    poller.startTracking(opRes);
  };

  const handleCreateGrant = async (e: React.FormEvent) => {
    e.preventDefault();
    const opRes = await portalApi.createMemberGrant(member.member_id, {
      valid_until: grantUntil.trim(),
      reason: grantReason.trim(),
    });
    poller.startTracking(opRes);
  };

  const handleRevokeDevice = async (deviceId: string) => {
    const opRes = await portalApi.revokeMemberDeviceAsAdmin(member.member_id, deviceId);
    poller.startTracking(opRes);
  };

  return (
    <div>
      <div style={{ marginBottom: 12 }}>
        <Link to="/admin/members">{t('admin.memberDetail.back')}</Link>
      </div>

      <header className="page-header">
        <h1>{t('admin.memberDetail.title', { name: member.display_name })}</h1>
        <p className="page-subtitle">
          ID: <code>{member.member_id}</code> ｜ Email: <code>{member.email}</code> ｜
          ETag: <code>{member.etag}</code>
        </p>
      </header>

      <OperationBanner
        operation={poller.operation}
        requestId={poller.requestId}
        isPolling={poller.isPolling}
        onDismiss={poller.reset}
      />

      <section className="card" aria-labelledby="entitlement-breakdown-heading">
        <h2 id="entitlement-breakdown-heading">
          {t('admin.memberDetail.entitlementBox')}
        </h2>
        <p>
          Entitlement State: <strong>{member.entitlement_state}</strong> —{' '}
          {member.entitlement_reason}
        </p>
        <p>
          {t('admin.memberDetail.paidThrough', { time: member.paid_through })} ｜{' '}
          {t('admin.memberDetail.effectiveEnd', { time: member.effective_service_end })}{' '}
          ｜ {t('admin.memberDetail.reconciledAt', { time: member.last_reconciled_at })}
        </p>
        <p>
          Monthly Usage ({member.accounting_period}):{' '}
          <strong>{formatBytesDual(member.monthly_total_bytes)}</strong> (raw:{' '}
          <code>{member.monthly_total_bytes}</code> bytes, quality: {member.usage_quality}
          )
        </p>
      </section>

      <section className="card" aria-labelledby="suspend-member-heading">
        <h2 id="suspend-member-heading">{t('admin.memberDetail.suspendSection')}</h2>
        <div className="form-grid">
          <div className="form-field">
            <label htmlFor="suspend-reason-input">
              {t('admin.memberDetail.suspendReasonLabel')}
            </label>
            <input
              id="suspend-reason-input"
              type="text"
              value={suspendReason}
              onChange={(e) => setSuspendReason(e.target.value)}
            />
          </div>
        </div>
        <div className="btn-row">
          {!member.admin_suspended ? (
            <button
              type="button"
              className="btn btn-danger"
              onClick={() => void handleToggleSuspend(true)}
              data-testid="suspend-member-btn"
            >
              {t('admin.memberDetail.suspendBtn')}
            </button>
          ) : (
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => void handleToggleSuspend(false)}
              data-testid="resume-member-btn"
            >
              {t('admin.memberDetail.resumeBtn')}
            </button>
          )}
        </div>
      </section>

      <form
        className="card"
        aria-labelledby="grant-member-heading"
        onSubmit={(e) => void handleCreateGrant(e)}
      >
        <h2 id="grant-member-heading">{t('admin.memberDetail.grantSection')}</h2>
        <div className="form-grid">
          <div className="form-field">
            <label htmlFor="grant-valid-until">
              {t('admin.memberDetail.grantValidUntil')}
            </label>
            <input
              id="grant-valid-until"
              type="text"
              value={grantUntil}
              onChange={(e) => setGrantUntil(e.target.value)}
            />
          </div>
          <div className="form-field">
            <label htmlFor="grant-reason">{t('admin.memberDetail.grantReason')}</label>
            <input
              id="grant-reason"
              type="text"
              value={grantReason}
              onChange={(e) => setGrantReason(e.target.value)}
            />
          </div>
        </div>
        <div className="btn-row">
          <button
            type="submit"
            className="btn btn-primary"
            data-testid="create-grant-btn"
          >
            {t('admin.memberDetail.grantSubmitBtn')}
          </button>
        </div>
      </form>

      <section className="card" aria-labelledby="member-devices-heading">
        <h2 id="member-devices-heading">{t('admin.memberDetail.devicesSection')}</h2>
        <div className="table-wrapper">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Device / Platform</th>
                <th scope="col">Overlay IPv4 / Peer</th>
                <th scope="col">Status</th>
                <th scope="col">Action</th>
              </tr>
            </thead>
            <tbody>
              {member.devices.map((dev) => (
                <tr key={dev.device_id}>
                  <td>
                    <strong>{dev.name}</strong> ({dev.platform})
                  </td>
                  <td>
                    <code>{dev.overlay_ipv4}</code> · <code>{dev.netbird_peer_id}</code>
                  </td>
                  <td>{dev.status}</td>
                  <td>
                    {dev.status === 'authorized' && (
                      <button
                        type="button"
                        className="btn btn-danger"
                        onClick={() => void handleRevokeDevice(dev.device_id)}
                        data-testid={`admin-revoke-device-${dev.device_id}`}
                      >
                        {t('admin.memberDetail.revokeDeviceBtn')}
                      </button>
                    )}
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
