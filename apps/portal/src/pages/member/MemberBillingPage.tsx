import React, { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { apiClient, deriveMemberServiceReason } from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { useSession } from '../../context/SessionContext';
import { useI18n } from '../../i18n';
import { formatLocaleWithUtc } from '../../lib/datetime';

export function MemberBillingPage(): React.ReactElement {
  const { t, locale } = useI18n();
  const { persona } = useSession();
  const [searchParams, setSearchParams] = useSearchParams();

  const [redirectUrl, setRedirectUrl] = useState<string | null>(null);
  const [actionError, setActionError] = useState<unknown>(null);
  const [checkoutActionId, setCheckoutActionId] = useState<string>(
    () => `checkout-${Date.now()}`,
  );

  const isCheckoutReturn = searchParams.get('checkout') === 'return';

  const selfQuery = useQuery({
    queryKey: ['member', 'me', persona],
    queryFn: () => apiClient.getMemberSelf(),
    refetchInterval: (query) => {
      const state = query.state.data?.entitlement.state;
      if (isCheckoutReturn && state && state !== 'active' && state !== 'cancel_at_period_end') {
        return 1000;
      }
      return false;
    },
  });

  if (selfQuery.error) {
    return (
      <div data-testid="member-billing-page">
        <ApiErrorAlert error={selfQuery.error} testId="member-billing-error" />
      </div>
    );
  }

  if (selfQuery.isLoading || !selfQuery.data) {
    return (
      <div className="card" role="status">
        Loading billing status...
      </div>
    );
  }

  const me = selfQuery.data;
  const ent = me.entitlement;
  const showPendingConfirm =
    isCheckoutReturn && ent.state !== 'active' && ent.state !== 'cancel_at_period_end';

  const handleCheckout = async () => {
    setActionError(null);
    try {
      const res = await apiClient.createCheckoutSession(undefined, checkoutActionId);
      if ('checkout_url' in res) {
        setRedirectUrl(res.checkout_url);
        setCheckoutActionId(`checkout-${Date.now()}`);
      }
    } catch (err) {
      setActionError(err);
    }
  };

  const handlePortal = async () => {
    setActionError(null);
    try {
      const res = await apiClient.createBillingPortalSession();
      setRedirectUrl(res.portal_url);
    } catch (err) {
      setActionError(err);
    }
  };

  const handleSimulateReturn = () => {
    setSearchParams({ checkout: 'return' });
  };

  return (
    <div data-testid="member-billing-page">
      <div className="page-header">
        <div>
          <h1>{t('member.billing.title')}</h1>
        </div>
        {showPendingConfirm ? (
          <span
            className="badge badge-warning"
            data-testid="billing-return-pending-badge"
          >
            {t('member.billing.returnPendingBadge')}
          </span>
        ) : (
          <span
            className={`badge ${
              ent.state === 'active' ? 'badge-success' : 'badge-warning'
            }`}
          >
            {ent.state}
          </span>
        )}
      </div>

      {/* Mandatory No Card Collection Guarantee (§18.4) */}
      <div
        className="alert alert-info"
        role="note"
        data-testid="no-card-collection-notice"
      >
        {t('member.billing.noCardNotice')}
      </div>

      <ApiErrorAlert error={actionError} testId="billing-action-error" />

      {/* AT03: Checkout return redirect without verified webhook shows "確認中" */}
      {showPendingConfirm && (
        <div
          className="alert alert-warning"
          role="status"
          data-testid="checkout-return-pending-banner"
        >
          <div style={{ fontWeight: 700 }}>{t('member.billing.returnPendingBadge')}</div>
          <div style={{ marginTop: '0.25rem' }}>
            {t('member.billing.returnPendingBody')}
          </div>
        </div>
      )}

      {redirectUrl && (
        <div
          className="alert alert-success"
          role="status"
          data-testid="hosted-redirect-banner"
        >
          Hosted Provider Session Ready:{' '}
          <a
            href={redirectUrl}
            className="mono"
            data-testid="hosted-redirect-url"
            style={{ fontWeight: 600 }}
          >
            {redirectUrl}
          </a>
        </div>
      )}

      <div className="card">
        <dl className="form-grid" style={{ marginBottom: '1rem' }}>
          <div>
            <dt style={{ fontWeight: 600 }}>{t('member.service.stateLabel')}</dt>
            <dd data-testid="billing-entitlement-state">{ent.state}</dd>
          </div>
          <div>
            <dt style={{ fontWeight: 600 }}>{t('member.service.paidThroughLabel')}</dt>
            <dd className="mono">
              {ent.paid_through ? formatLocaleWithUtc(ent.paid_through, locale) : 'None'}
            </dd>
          </div>
          <div>
            <dt style={{ fontWeight: 600 }}>{t('member.service.effectiveEndLabel')}</dt>
            <dd className="mono">
              {ent.effective_service_end
                ? formatLocaleWithUtc(ent.effective_service_end, locale)
                : 'None'}
            </dd>
          </div>
          <div>
            <dt style={{ fontWeight: 600 }}>{t('member.service.reasonLabel')}</dt>
            <dd>{deriveMemberServiceReason(me, locale)}</dd>
          </div>
        </dl>

        <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap' }}>
          <button
            type="button"
            className="btn btn-primary"
            data-testid="start-hosted-checkout-btn"
            onClick={() => void handleCheckout()}
          >
            {t('member.billing.checkoutBtn')}
          </button>

          <button
            type="button"
            className="btn btn-secondary"
            data-testid="open-billing-portal-btn"
            onClick={() => void handlePortal()}
          >
            {t('member.billing.portalBtn')}
          </button>

          <button
            type="button"
            className="btn btn-secondary"
            data-testid="simulate-checkout-return-btn"
            onClick={handleSimulateReturn}
          >
            {t('member.billing.simulateReturnBtn')}
          </button>
        </div>
      </div>
    </div>
  );
}
