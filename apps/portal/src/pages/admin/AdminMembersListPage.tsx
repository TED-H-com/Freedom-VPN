import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { portalApi, type MemberItem } from '../../api';
import { useI18n } from '../../i18n';
import { formatBytesDual } from '../../lib/decimal';

export function AdminMembersListPage(): React.ReactElement {
  const { t } = useI18n();
  const [members, setMembers] = useState<MemberItem[]>([]);
  const [stateFilter, setStateFilter] = useState<string>('all');

  useEffect(() => {
    void portalApi.getAdminMembers().then((res) => setMembers(res.data.items));
  }, []);

  const filtered = useMemo(() => {
    if (stateFilter === 'all') return members;
    return members.filter((m) => m.entitlement_state === stateFilter);
  }, [members, stateFilter]);

  return (
    <div>
      <header className="page-header">
        <h1>{t('admin.members.title')}</h1>
        <p className="page-subtitle">{t('admin.members.subtitle')}</p>
      </header>

      <section className="card" aria-label="Member filters">
        <div className="form-grid" style={{ marginBottom: 0 }}>
          <div className="form-field">
            <label htmlFor="member-state-filter">{t('admin.members.filter.state')}</label>
            <select
              id="member-state-filter"
              value={stateFilter}
              onChange={(e) => setStateFilter(e.target.value)}
            >
              <option value="all">{t('admin.nodes.filter.all')}</option>
              <option value="active">active</option>
              <option value="pending_payment">pending_payment</option>
              <option value="cancel_at_period_end">cancel_at_period_end</option>
              <option value="grace">grace / past_due</option>
              <option value="expired">expired</option>
              <option value="suspended">suspended</option>
              <option value="closed">closed</option>
            </select>
          </div>
        </div>
      </section>

      <section className="card table-wrapper">
        <table className="data-table" data-testid="members-table">
          <thead>
            <tr>
              <th scope="col">{t('admin.members.col.member')}</th>
              <th scope="col">{t('admin.members.col.entitlement')}</th>
              <th scope="col">{t('admin.members.col.planAndEnd')}</th>
              <th scope="col">{t('admin.members.col.devices')}</th>
              <th scope="col">{t('admin.members.col.usage')}</th>
              <th scope="col">{t('admin.members.col.fupTier')}</th>
              <th scope="col">{t('admin.members.col.payment')}</th>
              <th scope="col">{t('admin.members.col.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((member) => (
              <tr key={member.member_id} data-testid={`member-row-${member.email}`}>
                <td>
                  <strong>{member.display_name}</strong>
                  <div style={{ fontSize: '0.82rem', color: 'var(--text-secondary)' }}>
                    {member.email}
                  </div>
                </td>
                <td>
                  <span
                    className={
                      member.entitlement_state === 'active'
                        ? 'badge badge-success'
                        : member.entitlement_state === 'suspended' ||
                            member.entitlement_state === 'expired' ||
                            member.entitlement_state === 'closed'
                          ? 'badge badge-danger'
                          : 'badge badge-warning'
                    }
                  >
                    {member.entitlement_state}
                  </span>
                </td>
                <td>
                  <div>{member.plan_name}</div>
                  <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                    End: <code>{member.effective_service_end}</code>
                  </div>
                </td>
                <td>
                  {member.authorized_device_count} / {member.device_limit}
                </td>
                <td>
                  <div>{formatBytesDual(member.monthly_total_bytes)}</div>
                  <div style={{ fontSize: '0.78rem', color: 'var(--text-muted)' }}>
                    raw: <code>{member.monthly_total_bytes}</code> B (
                    {member.usage_quality})
                  </div>
                </td>
                <td>
                  <span
                    className={
                      member.fup_tier_index === 0
                        ? 'badge badge-success'
                        : 'badge badge-warning'
                    }
                  >
                    {member.fup_tier_name}
                  </span>
                </td>
                <td>
                  <div>
                    {member.payment_provider}:{' '}
                    <strong>{member.provider_subscription_status}</strong>
                  </div>
                  <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
                    Reconciled: {member.last_reconciled_at}
                  </div>
                </td>
                <td>
                  <Link
                    to={`/admin/members/${member.member_id}`}
                    className="btn"
                    data-testid={`manage-member-${member.email}`}
                  >
                    {t('admin.members.manageBtn')}
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
