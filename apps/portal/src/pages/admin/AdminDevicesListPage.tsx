import React, { useEffect, useRef, useState } from 'react';
import { Link, useLocation, useSearchParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  apiClient,
  createUuidV4,
  isDeviceAuthorizationUnconfirmed,
  type AdminDevice,
  type DeviceStatus,
} from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import {
  DeviceAuthorizationReceipt,
  type DeviceAuthorizationReceiptData,
} from '../../components/common/DeviceAuthorizationReceipt';
import { useI18n } from '../../i18n';
import { formatLocaleWithUtc } from '../../lib/datetime';

type UnconfirmedFilterValue = 'all' | 'true' | 'false';

function parseInitialUnconfirmedFilter(raw: string | null): UnconfirmedFilterValue {
  if (raw === 'true') {
    return 'true';
  }
  if (raw === 'false') {
    return 'false';
  }
  return 'all';
}

function parseInitialStatusFilter(raw: string | null): 'all' | DeviceStatus {
  if (
    raw === 'pending' ||
    raw === 'authorized' ||
    raw === 'revoked' ||
    raw === 'rejected_over_limit'
  ) {
    return raw;
  }
  return 'all';
}

export function AdminDevicesListPage(): React.ReactElement {
  const { t, locale } = useI18n();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();

  const location = useLocation();
  const unconfirmedFilter = parseInitialUnconfirmedFilter(
    searchParams.get('authorization_unconfirmed'),
  );
  const statusFilter = parseInitialStatusFilter(searchParams.get('status'));
  const memberIdFilter = searchParams.get('member_id') ?? '';
  const [memberIdDraft, setMemberIdDraft] = useState(memberIdFilter);
  const [paging, setPaging] = useState<{
    locationKey: string;
    cursor?: string;
    history: Array<string | undefined>;
  }>({ locationKey: location.key, history: [] });
  // A cursor belongs to exactly one set of filters, including browser navigation.
  const cursor = paging.locationKey === location.key ? paging.cursor : undefined;
  const cursorHistory = paging.locationKey === location.key ? paging.history : [];
  const [confirmDevice, setConfirmDevice] = useState<AdminDevice | null>(null);
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [receipt, setReceipt] = useState<DeviceAuthorizationReceiptData | null>(null);
  const [actionError, setActionError] = useState<unknown>(null);

  useEffect(() => {
    setMemberIdDraft(memberIdFilter);
  }, [memberIdFilter]);
  useEffect(() => {
    if (confirmDevice) dialogRef.current?.showModal();
  }, [confirmDevice]);

  const syncUrlParams = (
    nextUnconfirmed: UnconfirmedFilterValue,
    nextStatus: 'all' | DeviceStatus,
    nextMember: string,
  ) => {
    const next = new URLSearchParams();
    if (nextUnconfirmed !== 'all') {
      next.set('authorization_unconfirmed', nextUnconfirmed);
    }
    if (nextStatus !== 'all') {
      next.set('status', nextStatus);
    }
    if (nextMember.trim().length > 0) {
      next.set('member_id', nextMember.trim());
    }
    setSearchParams(next, { replace: true });
  };

  const trimmedMemberId = memberIdFilter.trim();
  const queryAuthorizationUnconfirmed =
    unconfirmedFilter === 'true'
      ? true
      : unconfirmedFilter === 'false'
        ? false
        : undefined;
  const queryStatus = statusFilter !== 'all' ? statusFilter : undefined;

  const devicesQuery = useQuery({
    queryKey: [
      'admin',
      'devices',
      unconfirmedFilter,
      statusFilter,
      trimmedMemberId,
      cursor,
    ],
    queryFn: () =>
      apiClient.listAdminDevices({
        authorization_unconfirmed: queryAuthorizationUnconfirmed,
        status: queryStatus,
        member_id: trimmedMemberId.length > 0 ? trimmedMemberId : undefined,
        cursor,
      }),
  });

  const items = devicesQuery.data?.items ?? [];
  const nextCursor = devicesQuery.data?.next_cursor ?? null;

  const closeConfirmation = () => {
    dialogRef.current?.close();
    setConfirmDevice(null);
  };

  const handleConfirmReauthorize = async () => {
    if (!confirmDevice || isSubmitting) {
      return;
    }
    const target = confirmDevice;
    setIsSubmitting(true);
    setActionError(null);
    // Fresh Idempotency-Key per click (D-077)
    const freshActionId = `admin-reauthorize:${target.member_id}:${target.device_id}:${createUuidV4()}`;
    try {
      const op = await apiClient.authorizeMemberDevice(
        target.member_id,
        target.device_id,
        freshActionId,
      );
      closeConfirmation();
      setReceipt({ device: target, operation: op, actionId: freshActionId });
      await queryClient.invalidateQueries({ queryKey: ['admin', 'devices'] });
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'member', target.member_id],
      });
      await queryClient.invalidateQueries({ queryKey: ['member', 'devices'] });
    } catch (err) {
      closeConfirmation();
      setActionError(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div data-testid="admin-devices-list-page">
      <div className="page-header">
        <div>
          <h1>{t('admin.devices.title')}</h1>
          <p className="page-subtitle">{t('admin.devices.subtitle')}</p>
        </div>
      </div>

      {receipt && (
        <DeviceAuthorizationReceipt
          receipt={receipt}
          onDismiss={() => setReceipt(null)}
        />
      )}

      <ApiErrorAlert error={actionError} testId="admin-device-authorize-error" />

      <ApiErrorAlert error={devicesQuery.error} testId="admin-devices-query-error" />

      {confirmDevice && (
        <dialog
          ref={dialogRef}
          className="modal-dialog authorization-dialog"
          onCancel={(event) => {
            event.preventDefault();
            if (!isSubmitting) closeConfirmation();
          }}
          aria-modal="true"
          aria-labelledby="reauthorize-confirm-title"
          data-testid="reauthorize-confirm-dialog"
        >
          <h2 id="reauthorize-confirm-title">{t('admin.devices.confirmDialog.title')}</h2>
          <p style={{ marginTop: '0.5rem' }}>
            {t('admin.devices.confirmDialog.description', {
              memberId: confirmDevice.member_id,
              deviceName: confirmDevice.name,
              deviceId: confirmDevice.device_id,
            })}
          </p>
          <div
            style={{
              display: 'flex',
              gap: '0.75rem',
              justifyContent: 'flex-end',
              marginTop: '1.25rem',
            }}
          >
            <button
              type="button"
              className="btn btn-secondary"
              data-testid="cancel-reauthorize-btn"
              disabled={isSubmitting}
              onClick={closeConfirmation}
            >
              {t('admin.devices.confirmDialog.cancelBtn')}
            </button>
            <button
              type="button"
              className="btn btn-primary"
              data-testid="confirm-reauthorize-btn"
              disabled={isSubmitting}
              onClick={() => void handleConfirmReauthorize()}
            >
              {t('admin.devices.confirmDialog.confirmBtn')}
            </button>
          </div>
        </dialog>
      )}

      <div className="card">
        <form
          className="form-grid device-filters"
          onSubmit={(event) => {
            event.preventDefault();
            syncUrlParams(unconfirmedFilter, statusFilter, memberIdDraft);
          }}
        >
          <div className="form-field">
            <label htmlFor="admin-devices-unconfirmed-filter">
              {t('admin.devices.filter.authorizationUnconfirmed')}
            </label>
            <select
              id="admin-devices-unconfirmed-filter"
              data-testid="admin-devices-unconfirmed-filter"
              value={unconfirmedFilter}
              onChange={(e) => {
                const next = e.target.value as UnconfirmedFilterValue;
                syncUrlParams(next, statusFilter, memberIdFilter);
              }}
            >
              <option value="all">{t('admin.devices.filter.unconfirmedAll')}</option>
              <option value="true">{t('admin.devices.filter.unconfirmedTrue')}</option>
              <option value="false">{t('admin.devices.filter.unconfirmedFalse')}</option>
            </select>
          </div>

          <div className="form-field">
            <label htmlFor="admin-devices-status-filter">
              {t('admin.devices.filter.status')}
            </label>
            <select
              id="admin-devices-status-filter"
              data-testid="admin-devices-status-filter"
              value={statusFilter}
              onChange={(e) => {
                const next = e.target.value as 'all' | DeviceStatus;
                syncUrlParams(unconfirmedFilter, next, memberIdFilter);
              }}
            >
              <option value="all">{t('admin.devices.filter.unconfirmedAll')}</option>
              <option value="pending">pending</option>
              <option value="authorized">authorized</option>
              <option value="revoked">revoked</option>
              <option value="rejected_over_limit">rejected_over_limit</option>
            </select>
          </div>

          <div className="form-field">
            <label htmlFor="admin-devices-member-filter">
              {t('admin.devices.filter.memberId')}
            </label>
            <input
              id="admin-devices-member-filter"
              data-testid="admin-devices-member-filter"
              className="mono"
              value={memberIdDraft}
              placeholder="UUID"
              onChange={(e) => {
                const next = e.target.value;
                setMemberIdDraft(next);
              }}
            />
          </div>
          <div className="btn-row">
            <button
              className="btn btn-secondary"
              type="submit"
              data-testid="admin-devices-apply-member-filter"
            >
              {t('admin.devices.filter.apply')}
            </button>

            {(unconfirmedFilter !== 'all' ||
              statusFilter !== 'all' ||
              memberIdFilter.trim().length > 0) && (
              <button
                type="button"
                className="btn btn-secondary"
                data-testid="admin-devices-clear-filters-btn"
                onClick={() => {
                  setMemberIdDraft('');
                  syncUrlParams('all', 'all', '');
                }}
              >
                {t('admin.devices.filter.clearFilters')}
              </button>
            )}
          </div>
        </form>

        <div className="table-wrapper">
          <table className="data-table device-table" data-testid="admin-devices-table">
            <thead>
              <tr>
                <th>{t('admin.devices.col.member')}</th>
                <th>{t('admin.devices.col.device')}</th>
                <th>{t('admin.devices.col.status')}</th>
                <th>{t('admin.devices.col.unconfirmedSince')}</th>
                <th>{t('admin.devices.col.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr>
                  <td
                    colSpan={5}
                    data-testid="admin-devices-empty-row"
                    style={{
                      textAlign: 'center',
                      color: 'var(--text-muted)',
                    }}
                  >
                    {t(
                      devicesQuery.isPending
                        ? 'admin.devices.loading'
                        : devicesQuery.isError
                          ? 'admin.devices.loadFailed'
                          : 'admin.devices.empty',
                    )}
                  </td>
                </tr>
              ) : (
                items.map((dev) => {
                  const unconfirmed = isDeviceAuthorizationUnconfirmed(dev);
                  return (
                    <tr
                      key={dev.device_id}
                      data-testid={`admin-device-row-${dev.device_id}`}
                    >
                      <td className="mono" style={{ fontSize: '0.8rem' }}>
                        <Link
                          to={`/admin/members/${dev.member_id}`}
                          data-testid={`admin-device-member-link-${dev.device_id}`}
                        >
                          {dev.member_id}
                        </Link>
                      </td>
                      <td>
                        <div style={{ fontWeight: 600 }}>{dev.name}</div>
                        <div
                          className="mono"
                          style={{
                            fontSize: '0.75rem',
                            color: 'var(--text-muted)',
                          }}
                        >
                          {dev.device_id} · {dev.platform ?? '-'}
                        </div>
                      </td>
                      <td>
                        <div
                          style={{
                            display: 'flex',
                            flexWrap: 'wrap',
                            gap: '0.35rem',
                            alignItems: 'center',
                          }}
                        >
                          <span
                            className={`badge ${
                              dev.status === 'authorized'
                                ? 'badge-success'
                                : dev.status === 'pending'
                                  ? 'badge-warning'
                                  : 'badge-danger'
                            }`}
                            data-testid={`admin-device-status-${dev.device_id}`}
                          >
                            {dev.status}
                          </span>
                          {unconfirmed && (
                            <span
                              className="badge badge-warning"
                              data-testid={`admin-device-unconfirmed-badge-${dev.device_id}`}
                            >
                              {t('member.devices.status.authorizationUnconfirmedBadge')}
                            </span>
                          )}
                        </div>
                      </td>
                      <td
                        className="mono"
                        style={{ fontSize: '0.8rem' }}
                        data-testid={`admin-device-unconfirmed-since-${dev.device_id}`}
                      >
                        {dev.authorization_unconfirmed_since
                          ? formatLocaleWithUtc(
                              dev.authorization_unconfirmed_since,
                              locale,
                            )
                          : '—'}
                      </td>
                      <td>
                        <button
                          type="button"
                          className="btn btn-secondary"
                          disabled={isSubmitting}
                          data-testid={`admin-reauthorize-device-btn-${dev.device_id}`}
                          onClick={() => {
                            setActionError(null);
                            setConfirmDevice(dev);
                          }}
                        >
                          {t('admin.devices.reauthorizeBtn')}
                        </button>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>

        {(cursorHistory.length > 0 || nextCursor) && (
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginTop: '1rem',
            }}
            data-testid="admin-devices-pagination"
          >
            <button
              type="button"
              className="btn btn-secondary"
              data-testid="admin-devices-prev-page-btn"
              disabled={cursorHistory.length === 0}
              onClick={() => {
                const prev = cursorHistory[cursorHistory.length - 1];
                setPaging({
                  locationKey: location.key,
                  cursor: prev,
                  history: cursorHistory.slice(0, -1),
                });
              }}
            >
              {t('common.pagination.prev')}
            </button>

            <button
              type="button"
              className="btn btn-secondary"
              data-testid="admin-devices-next-page-btn"
              disabled={!nextCursor}
              onClick={() => {
                if (nextCursor) {
                  setPaging({
                    locationKey: location.key,
                    cursor: nextCursor,
                    history: [...cursorHistory, cursor],
                  });
                }
              }}
            >
              {t('common.pagination.next')}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
