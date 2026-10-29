import React, { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { portalApi, type MemberItem } from '../../api';
import { useSession } from '../../context/SessionContext';
import { useI18n } from '../../i18n';

export function MemberBillingPage(): React.ReactElement {
  const { t } = useI18n();
  const { persona } = useSession();
  const [searchParams, setSearchParams] = useSearchParams();
  const [member, setMember] = useState<MemberItem | null>(null);
  const [hostedUrl, setHostedUrl] = useState<string | null>(null);

  const isReturnFromCheckout = searchParams.get('checkout') === 'return';

  useEffect(() => {
    void portalApi.getMemberMe().then((res) => setMember(res.data.member));
  }, [persona]);

  if (!member) {
    return <div className="card">Loading billing info...</div>;
  }

  const handleCheckout = async () => {
    const res = await portalApi.createCheckoutSession(member.plan_code);
    setHostedUrl(res.data.hosted_url);
  };

  const handleBillingPortal = async () => {
    const res = await portalApi.createBillingPortalSession();
    setHostedUrl(res.data.hosted_url);
  };

  const handleSimulateReturn = () => {
    setSearchParams({ checkout: 'return' });
  };

  return (
    <div>
      <header className="page-header">
        <h1>{t('member.billing.title')}</h1>
      </header>

      <div className="alert alert-info" data-testid="no-card-notice">
        {t('member.billing.noCardNotice')}
      </div>

      {isReturnFromCheckout && (
        <section
          className="card alert-warning"
          role="status"
          aria-live="polite"
          data-testid="checkout-return-verifying-banner"
        >
          <h2>
            <span className="badge badge-warning" style={{ fontSize: '1rem' }}>
              {t('member.billing.returnPendingBadge')}
            </span>
          </h2>
          <p style={{ margin: '8px 0 0' }}>{t('member.billing.returnPendingBody')}</p>
        </section>
      )}

      <section className="card">
        <p>
          Plan: <strong>{member.plan_name}</strong> ｜ Provider Status:{' '}
          <strong>{member.provider_subscription_status}</strong>
        </p>
        <p>
          Paid Through (<code>paid_through</code>): <code>{member.paid_through}</code> ｜
          Cancel at Period End: <strong>{String(member.cancel_at_period_end)}</strong>
        </p>
        <p>
          Last Reconciled: <code>{member.last_reconciled_at}</code>
        </p>

        <div className="btn-row" style={{ marginTop: 14 }}>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => void handleCheckout()}
            data-testid="start-hosted-checkout-btn"
          >
            {t('member.billing.checkoutBtn')}
          </button>
          <button
            type="button"
            className="btn"
            onClick={() => void handleBillingPortal()}
            data-testid="open-billing-portal-btn"
          >
            {t('member.billing.portalBtn')}
          </button>
          <button
            type="button"
            className="btn"
            onClick={handleSimulateReturn}
            data-testid="simulate-checkout-return-btn"
          >
            {t('member.billing.simulateReturnBtn')}
          </button>
        </div>

        {hostedUrl && (
          <div className="alert alert-success" style={{ marginTop: 14 }}>
            Hosted Provider Session URL: <code>{hostedUrl}</code>
          </div>
        )}
      </section>
    </div>
  );
}
