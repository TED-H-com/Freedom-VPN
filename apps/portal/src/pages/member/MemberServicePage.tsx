import React, { useEffect, useState } from 'react';
import { portalApi, type MemberItem } from '../../api';
import { useSession } from '../../context/SessionContext';
import { useI18n } from '../../i18n';

export function MemberServicePage(): React.ReactElement {
  const { t } = useI18n();
  const { persona } = useSession();
  const [member, setMember] = useState<MemberItem | null>(null);

  useEffect(() => {
    void portalApi.getMemberMe().then((res) => setMember(res.data.member));
  }, [persona]);

  if (!member) {
    return <div className="card">Loading service status...</div>;
  }

  return (
    <div>
      <header className="page-header">
        <h1>{t('member.service.title')}</h1>
      </header>

      {member.fup_tier_index > 0 && (
        <div
          className="alert alert-warning"
          role="status"
          data-testid="member-fup-throttled-banner"
        >
          <strong>
            {t('member.service.fupThrottledBanner', {
              resetAt: member.next_fup_reset_at,
            })}
          </strong>
        </div>
      )}

      <div className="metrics-grid">
        <article className="metric-card" data-testid="member-entitlement-card">
          <div>
            <h2>{t('member.service.stateLabel')}</h2>
            <div className="metric-value">
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
                style={{ fontSize: '1.1rem', padding: '6px 12px' }}
                data-testid="member-entitlement-state-badge"
              >
                {member.entitlement_state}
              </span>
            </div>
            <p className="metric-desc" data-testid="member-entitlement-reason">
              <strong>{t('member.service.reasonLabel')}:</strong>{' '}
              {member.entitlement_reason}
            </p>
          </div>
        </article>

        <article className="metric-card">
          <div>
            <h2>{t('member.service.planLabel')}</h2>
            <div className="metric-value" style={{ fontSize: '1.25rem' }}>
              {member.plan_name}
            </div>
            <p className="metric-desc">
              Devices: {member.authorized_device_count} / {member.device_limit}
            </p>
          </div>
        </article>

        <article className="metric-card">
          <div>
            <h2>{t('member.service.effectiveEndLabel')}</h2>
            <div className="metric-value" style={{ fontSize: '1.2rem' }}>
              <code>{member.effective_service_end}</code>
            </div>
            <p className="metric-desc">
              {t('member.service.paidThroughLabel')}: <code>{member.paid_through}</code>
            </p>
          </div>
        </article>

        <article className="metric-card">
          <div>
            <h2>{t('member.service.nextResetLabel')}</h2>
            <div className="metric-value" style={{ fontSize: '1.2rem' }}>
              <code>{member.next_fup_reset_at}</code>
            </div>
            <p className="metric-desc">FUP Tier: {member.fup_tier_name}</p>
          </div>
        </article>
      </div>
    </div>
  );
}
