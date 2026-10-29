import React, { useEffect, useState } from 'react';
import { portalApi, type MemberExitItem } from '../../api';
import { useI18n } from '../../i18n';

export function MemberExitsPage(): React.ReactElement {
  const { locale, t } = useI18n();
  const [exits, setExits] = useState<MemberExitItem[]>([]);

  useEffect(() => {
    void portalApi.getMemberExits().then((res) => setExits(res.data.items));
  }, []);

  return (
    <div>
      <header className="page-header">
        <h1>{t('member.exits.title')}</h1>
      </header>

      <div
        className="alert alert-info"
        role="note"
        data-testid="manual-exit-selection-notice"
      >
        <strong>{t('member.exits.manualSelectionNotice')}</strong>
      </div>

      <section className="card table-wrapper">
        <table className="data-table" data-testid="member-exits-table">
          <thead>
            <tr>
              <th scope="col">{t('member.exits.col.name')}</th>
              <th scope="col">{t('member.exits.col.region')}</th>
              <th scope="col">{t('member.exits.col.status')}</th>
              <th scope="col">{t('member.exits.col.notes')}</th>
            </tr>
          </thead>
          <tbody>
            {exits.map((exit) => (
              <tr key={exit.exit_id}>
                <td>
                  <strong>{exit.name}</strong>
                </td>
                <td>{exit.region}</td>
                <td>
                  <span
                    className={
                      exit.status === 'available'
                        ? 'badge badge-success'
                        : 'badge badge-danger'
                    }
                  >
                    {exit.status}
                  </span>
                </td>
                <td>{locale === 'en' ? exit.note_en : exit.note_zh}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
