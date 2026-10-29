import React, { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, type OperationResource } from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n } from '../../i18n';
import { formatLocaleWithUtc } from '../../lib/datetime';

export function MemberDevicesPage(): React.ReactElement {
  const { t, locale } = useI18n();
  const queryClient = useQueryClient();

  const [activeOp, setActiveOp] = useState<OperationResource | null>(null);
  const [error, setError] = useState<unknown>(null);

  const devicesQuery = useQuery({
    queryKey: ['member', 'devices'],
    queryFn: () => apiClient.listMemberDevices(),
  });

  if (devicesQuery.error) {
    return (
      <div data-testid="member-devices-page">
        <ApiErrorAlert error={devicesQuery.error} testId="member-devices-error" />
      </div>
    );
  }

  if (devicesQuery.isLoading || !devicesQuery.data) {
    return (
      <div className="card" role="status">
        Loading devices...
      </div>
    );
  }

  const { items, device_limit, authorized_count } = devicesQuery.data;

  const handleRevoke = async (deviceId: string) => {
    setError(null);
    try {
      const op = await apiClient.revokeMemberDevice(
        deviceId,
        `member-revoke:${deviceId}`,
      );
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['member', 'devices'] });
      await queryClient.invalidateQueries({ queryKey: ['member', 'me'] });
    } catch (err) {
      setError(err);
    }
  };

  return (
    <div data-testid="member-devices-page">
      <div className="page-header">
        <div>
          <h1>{t('member.devices.title')}</h1>
          <p className="page-subtitle">{t('member.devices.quotaHelp')}</p>
        </div>
        <span className="badge badge-info mono" data-testid="device-quota-badge">
          {t('member.devices.quotaSummary', {
            used: authorized_count,
            limit: device_limit ?? '-',
          })}
        </span>
      </div>

      <div
        className="alert alert-info"
        role="note"
        data-testid="member-devices-add-guide"
      >
        {t('member.devices.addGuide')}
      </div>

      {device_limit !== null && authorized_count >= device_limit && (
        <div
          className="alert alert-warning"
          role="status"
          data-testid="device-limit-warning"
        >
          {t('member.devices.quotaSummary', {
            used: authorized_count,
            limit: device_limit,
          })}
        </div>
      )}

      <OperationBanner
        operation={activeOp}
        scope="member"
        onDismiss={() => setActiveOp(null)}
        onTerminal={() => {
          void queryClient.invalidateQueries({ queryKey: ['member', 'devices'] });
          void queryClient.invalidateQueries({ queryKey: ['member', 'me'] });
        }}
      />

      <ApiErrorAlert error={error} testId="member-device-action-error" />

      <div className="card">
        <div className="table-wrapper">
          <table className="data-table" data-testid="member-devices-table">
            <thead>
              <tr>
                <th>{t('member.devices.col.name')}</th>
                <th>{t('member.devices.col.overlayIp')}</th>
                <th>{t('member.devices.col.status')}</th>
                <th>{t('member.devices.col.lastSeen')}</th>
                <th>{t('member.devices.col.action')}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((dev) => (
                <tr
                  key={dev.device_id}
                  data-testid={`member-device-row-${dev.device_id}`}
                >
                  <td>
                    <div style={{ fontWeight: 600 }}>{dev.name}</div>
                    <div
                      style={{ fontSize: '0.75rem', color: 'var(--color-text-muted)' }}
                    >
                      {dev.platform ?? '-'}
                    </div>
                  </td>
                  <td className="mono" style={{ fontSize: '0.8rem' }}>
                    <div>{dev.device_id}</div>
                  </td>
                  <td>
                    <span
                      className={`badge ${
                        dev.status === 'authorized' ? 'badge-success' : 'badge-danger'
                      }`}
                      data-testid={`device-status-${dev.device_id}`}
                    >
                      {dev.status}
                    </span>
                  </td>
                  <td className="mono" style={{ fontSize: '0.75rem' }}>
                    <div>seen: {formatLocaleWithUtc(dev.last_seen_at, locale)}</div>
                    <div>auth: {formatLocaleWithUtc(dev.authorized_at, locale)}</div>
                  </td>
                  <td>
                    {dev.status === 'authorized' && (
                      <button
                        type="button"
                        className="btn btn-danger"
                        data-testid={`revoke-own-device-${dev.device_id}`}
                        onClick={() => void handleRevoke(dev.device_id)}
                      >
                        {t('member.devices.revokeBtn')}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
