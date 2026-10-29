import React, { useCallback, useEffect, useState } from 'react';
import {
  portalApi,
  useOperationPoller,
  type MemberDeviceItem,
  type MemberItem,
} from '../../api';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useSession } from '../../context/SessionContext';
import { useI18n } from '../../i18n';

export function MemberDevicesPage(): React.ReactElement {
  const { t } = useI18n();
  const { persona } = useSession();
  const [member, setMember] = useState<MemberItem | null>(null);
  const [devices, setDevices] = useState<MemberDeviceItem[]>([]);

  const loadDevices = useCallback(async () => {
    const [meRes, devRes] = await Promise.all([
      portalApi.getMemberMe(),
      portalApi.getMemberDevices(),
    ]);
    setMember(meRes.data.member);
    setDevices(devRes.data.items);
  }, []);

  const poller = useOperationPoller({
    scope: 'member',
    onTerminal: () => {
      void loadDevices();
    },
  });

  useEffect(() => {
    void loadDevices();
  }, [loadDevices, persona]);

  const handleRevoke = async (deviceId: string) => {
    const opRes = await portalApi.deleteMemberDevice(deviceId);
    poller.startTracking(opRes);
  };

  return (
    <div>
      <header className="page-header">
        <h1>{t('member.devices.title')}</h1>
        {member && (
          <p className="page-subtitle" data-testid="member-device-quota">
            <strong>
              {t('member.devices.quotaSummary', {
                used: member.authorized_device_count,
                limit: member.device_limit,
              })}
            </strong>
          </p>
        )}
      </header>

      <div className="alert alert-info">{t('member.devices.quotaHelp')}</div>

      <OperationBanner
        operation={poller.operation}
        requestId={poller.requestId}
        isPolling={poller.isPolling}
        onDismiss={poller.reset}
      />

      <section className="card table-wrapper">
        <table className="data-table" data-testid="member-devices-table">
          <thead>
            <tr>
              <th scope="col">{t('member.devices.col.name')}</th>
              <th scope="col">{t('member.devices.col.overlayIp')}</th>
              <th scope="col">{t('member.devices.col.status')}</th>
              <th scope="col">{t('member.devices.col.lastSeen')}</th>
              <th scope="col">{t('member.devices.col.action')}</th>
            </tr>
          </thead>
          <tbody>
            {devices.map((dev) => (
              <tr key={dev.device_id} data-testid={`member-device-row-${dev.name}`}>
                <td>
                  <strong>{dev.name}</strong> ({dev.platform})
                </td>
                <td>
                  <code>{dev.overlay_ipv4}</code> · <code>{dev.netbird_peer_id}</code>
                </td>
                <td>
                  <span
                    className={
                      dev.status === 'authorized'
                        ? 'badge badge-success'
                        : 'badge badge-danger'
                    }
                  >
                    {dev.status}
                  </span>
                </td>
                <td>
                  {dev.last_seen_at}
                  <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
                    auth: {dev.authorized_at}
                  </div>
                </td>
                <td>
                  {dev.status === 'authorized' && (
                    <button
                      type="button"
                      className="btn btn-danger"
                      onClick={() => void handleRevoke(dev.device_id)}
                      data-testid={`revoke-own-device-${dev.name}`}
                    >
                      {t('member.devices.revokeBtn')}
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
