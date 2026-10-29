import React, { useCallback, useEffect, useState } from 'react';
import {
  portalApi,
  useOperationPoller,
  type FairUsePolicyDraft,
  type FairUsePolicyState,
  type FairUseTier,
} from '../../api';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n, type TranslationKey } from '../../i18n';
import { formatBpsMbps, formatBytesGB, isValidDecimalString } from '../../lib/decimal';
import { validateFupDraft, type FupValidationResult } from '../../lib/fup';

export function AdminFupPolicyPage(): React.ReactElement {
  const { t } = useI18n();
  const [policyState, setPolicyState] = useState<FairUsePolicyState | null>(null);
  const [draft, setDraft] = useState<FairUsePolicyDraft | null>(null);
  const [validation, setValidation] = useState<FupValidationResult | null>(null);

  const loadPolicy = useCallback(async () => {
    const res = await portalApi.getFupPolicy();
    setPolicyState(res.data);
    setDraft(JSON.parse(JSON.stringify(res.data.current)) as FairUsePolicyDraft);
  }, []);

  const poller = useOperationPoller({
    scope: 'admin',
    onTerminal: () => {
      void loadPolicy();
    },
  });

  useEffect(() => {
    void loadPolicy();
  }, [loadPolicy]);

  if (!policyState || !draft) {
    return <div className="card">Loading FUP policy...</div>;
  }

  const updateTier = (idx: number, patch: Partial<FairUseTier>) => {
    const nextTiers = draft.tiers.map((tier, i) =>
      i === idx ? { ...tier, ...patch } : tier,
    );
    const nextDraft = { ...draft, tiers: nextTiers };
    setDraft(nextDraft);
    setValidation(validateFupDraft(nextDraft));
  };

  const handlePublish = async (e: React.FormEvent) => {
    e.preventDefault();
    const check = validateFupDraft(draft);
    setValidation(check);
    if (!check.valid) {
      return;
    }
    const opRes = await portalApi.publishFupPolicy(draft, {
      ifMatch: policyState.etag,
    });
    poller.startTracking(opRes);
  };

  return (
    <div>
      <header className="page-header">
        <h1>{t('admin.fup.title')}</h1>
        <p className="page-subtitle">
          Active Revision: <code>#{policyState.active_revision_number}</code> ｜ ETag:{' '}
          <code>{policyState.etag}</code> ｜ Nodes Applied:{' '}
          {policyState.node_applied_summary.applied_current_generation} /{' '}
          {policyState.node_applied_summary.total_nodes}
        </p>
      </header>

      <div className="alert alert-warning" data-testid="fup-test-only-banner">
        <strong>{t('admin.fup.testOnlyBanner')}</strong>
      </div>

      <OperationBanner
        operation={poller.operation}
        requestId={poller.requestId}
        isPolling={poller.isPolling}
        onDismiss={poller.reset}
      />

      {validation && !validation.valid && (
        <div
          className="alert alert-danger"
          role="alert"
          data-testid="fup-validation-errors"
        >
          <ul style={{ margin: 0, paddingLeft: 18 }}>
            {validation.errors.map((err, idx) => (
              <li key={`${err.field}-${idx}`}>
                <code>{err.field}</code>: {t(err.messageKey as TranslationKey)}
              </li>
            ))}
          </ul>
        </div>
      )}

      {validation && validation.valid && (
        <div
          className="alert alert-success"
          role="status"
          data-testid="fup-validation-passed"
        >
          {t('admin.fup.validationPassed')}
        </div>
      )}

      <form className="card" onSubmit={(e) => void handlePublish(e)}>
        <div className="form-grid">
          <div className="form-field">
            <label htmlFor="fup-mode-select">{t('admin.fup.modeLabel')}</label>
            <select
              id="fup-mode-select"
              value={draft.mode}
              onChange={(e) =>
                setDraft({ ...draft, mode: e.target.value as 'shadow' | 'enforce' })
              }
              data-testid="fup-mode-select"
            >
              <option value="shadow">{t('admin.fup.mode.shadow')}</option>
              <option value="enforce">{t('admin.fup.mode.enforce')}</option>
            </select>
          </div>

          <div className="form-field">
            <label htmlFor="fup-effective-at">{t('admin.fup.effectiveAtLabel')}</label>
            <input
              id="fup-effective-at"
              type="text"
              value={draft.effective_at}
              onChange={(e) => setDraft({ ...draft, effective_at: e.target.value })}
            />
          </div>

          <div className="form-field">
            <label htmlFor="fup-min-rate">{t('admin.fup.minRateLabel')}</label>
            <input
              id="fup-min-rate"
              type="text"
              value={draft.minimum_rate_bps}
              onChange={(e) => setDraft({ ...draft, minimum_rate_bps: e.target.value })}
            />
          </div>

          <div className="form-field">
            <label htmlFor="fup-reason">{t('admin.fup.reasonLabel')}</label>
            <input
              id="fup-reason"
              type="text"
              value={draft.reason}
              onChange={(e) => setDraft({ ...draft, reason: e.target.value })}
            />
          </div>
        </div>

        <h2>{t('admin.fup.tiersTitle')}</h2>
        <div className="table-wrapper">
          <table className="data-table" data-testid="fup-tiers-table">
            <thead>
              <tr>
                <th scope="col">Tier</th>
                <th scope="col">Display Name (zh-TW)</th>
                <th scope="col">threshold_bytes (Decimal String)</th>
                <th scope="col">down_bps (Decimal String)</th>
                <th scope="col">up_bps (Decimal String)</th>
                <th scope="col">Priority (1..10)</th>
              </tr>
            </thead>
            <tbody>
              {draft.tiers.map((tier, idx) => (
                <tr key={tier.tier_index} data-testid={`fup-tier-row-${idx}`}>
                  <td>
                    <strong>Tier {idx}</strong>
                  </td>
                  <td>
                    <input
                      aria-label={`Tier ${idx} Name zh-TW`}
                      type="text"
                      value={tier.name_zh}
                      onChange={(e) => updateTier(idx, { name_zh: e.target.value })}
                    />
                  </td>
                  <td>
                    <input
                      aria-label={`Tier ${idx} threshold_bytes`}
                      type="text"
                      value={tier.threshold_bytes}
                      onChange={(e) =>
                        updateTier(idx, { threshold_bytes: e.target.value })
                      }
                      data-testid={`fup-tier-threshold-${idx}`}
                    />
                    <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
                      {isValidDecimalString(tier.threshold_bytes)
                        ? formatBytesGB(tier.threshold_bytes)
                        : 'Invalid decimal'}
                    </div>
                  </td>
                  <td>
                    <input
                      aria-label={`Tier ${idx} down_bps`}
                      type="text"
                      value={tier.down_bps}
                      onChange={(e) => updateTier(idx, { down_bps: e.target.value })}
                      data-testid={`fup-tier-down-${idx}`}
                    />
                    <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
                      {isValidDecimalString(tier.down_bps)
                        ? formatBpsMbps(tier.down_bps)
                        : 'Invalid decimal'}
                    </div>
                  </td>
                  <td>
                    <input
                      aria-label={`Tier ${idx} up_bps`}
                      type="text"
                      value={tier.up_bps}
                      onChange={(e) => updateTier(idx, { up_bps: e.target.value })}
                      data-testid={`fup-tier-up-${idx}`}
                    />
                    <div style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
                      {isValidDecimalString(tier.up_bps)
                        ? formatBpsMbps(tier.up_bps)
                        : 'Invalid decimal'}
                    </div>
                  </td>
                  <td>
                    <input
                      aria-label={`Tier ${idx} priority`}
                      type="number"
                      min={1}
                      max={10}
                      value={tier.priority}
                      onChange={(e) =>
                        updateTier(idx, {
                          priority: Number.parseInt(e.target.value, 10) || 1,
                        })
                      }
                      style={{ width: 70 }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="btn-row" style={{ marginTop: 16 }}>
          <button type="submit" className="btn btn-primary" data-testid="publish-fup-btn">
            {t('admin.fup.validateAndPublishBtn')}
          </button>
        </div>
      </form>
    </div>
  );
}
