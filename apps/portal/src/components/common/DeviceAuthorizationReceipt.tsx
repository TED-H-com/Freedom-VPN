import { useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  apiClient,
  isDeviceAuthorizationUnconfirmed,
  type AdminDevice,
  type OperationResource,
} from '../../api';
import { useI18n } from '../../i18n';
import { ApiErrorAlert } from './ApiErrorAlert';
import { OperationBanner } from './OperationBanner';

function isConfirmed(device: AdminDevice | null | undefined): boolean {
  return device?.status === 'authorized' && !isDeviceAuthorizationUnconfirmed(device);
}

export interface DeviceAuthorizationReceiptData {
  device: AdminDevice;
  operation: OperationResource;
  actionId: string;
}

export function DeviceAuthorizationReceipt({
  receipt,
  onDismiss,
}: {
  receipt: DeviceAuthorizationReceiptData;
  onDismiss: () => void;
}): React.ReactElement {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  // A fenced grant can return an old, already-succeeded operation. Read the
  // device independently, including when it has left the operator's filter.
  const readback = useQuery({
    queryKey: ['device-authorization-readback', receipt.actionId],
    queryFn: async () => {
      let cursor: string | undefined;
      do {
        const page = await apiClient.listAdminDevices({
          member_id: receipt.device.member_id,
          limit: 100,
          cursor,
        });
        const device = page.items.find((d) => d.device_id === receipt.device.device_id);
        if (device) return device;
        cursor = page.next_cursor ?? undefined;
      } while (cursor);
      return null;
    },
    refetchInterval: (query) => (isConfirmed(query.state.data) ? false : 2000),
  });
  const confirmed = isConfirmed(readback.data);

  useEffect(() => {
    if (confirmed) {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'devices'] });
      void queryClient.invalidateQueries({
        queryKey: ['admin', 'member', receipt.device.member_id],
      });
      void queryClient.invalidateQueries({ queryKey: ['member', 'devices'] });
    }
  }, [confirmed, queryClient, receipt.device.member_id]);

  return (
    <div data-testid="reauthorize-operation-receipt">
      <OperationBanner operation={receipt.operation} receiptOnly onDismiss={onDismiss} />
      <p role="status" data-testid="reauthorize-device-state">
        <strong>{receipt.device.name}: </strong>
        {t(
          confirmed ? 'admin.devices.readbackConfirmed' : 'admin.devices.readbackPending',
        )}
      </p>
      <ApiErrorAlert error={readback.error} testId="admin-device-readback-error" />
    </div>
  );
}
