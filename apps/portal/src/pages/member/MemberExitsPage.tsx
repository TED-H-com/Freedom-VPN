import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { useI18n } from '../../i18n';

export function MemberExitsPage(): React.ReactElement {
  const { t, locale } = useI18n();

  const exitsQuery = useQuery({
    queryKey: ['member', 'exits'],
    queryFn: () => apiClient.listMemberExits(),
  });

  const items = exitsQuery.data?.items ?? [];

  return (
    <div data-testid="member-exits-page">
      <div className="page-header">
        <div>
          <h1>{t('member.exits.title')}</h1>
        </div>
      </div>

      {/* Mandatory Manual App Exit Selection Disclosure (EXIT02, §18.4, §18.5) */}
      <div
        className="alert alert-info"
        role="note"
        data-testid="manual-exit-selection-notice"
      >
        {t('member.exits.manualSelectionNotice')}
      </div>

      <ApiErrorAlert error={exitsQuery.error} testId="member-exits-error" />

      <div className="card">
        <div className="table-wrapper">
          <table className="data-table" data-testid="member-exits-table">
            <thead>
              <tr>
                <th>{t('member.exits.col.name')}</th>
                <th>{t('member.exits.col.region')}</th>
                <th>{t('member.exits.col.status')}</th>
                <th>{t('member.exits.col.notes')}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((exit) => (
                <tr key={exit.exit_id} data-testid={`member-exit-row-${exit.exit_id}`}>
                  <td style={{ fontWeight: 600 }}>{exit.display_name}</td>
                  <td className="mono">{exit.region}</td>
                  <td>
                    <span
                      className={`badge ${
                        exit.availability === 'available'
                          ? 'badge-success'
                          : 'badge-warning'
                      }`}
                    >
                      {exit.availability}
                    </span>
                  </td>
                  <td>{exit.note ? (exit.note[locale] ?? exit.note['zh-TW']) : '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
