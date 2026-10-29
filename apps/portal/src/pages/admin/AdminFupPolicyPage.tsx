import React, { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  apiClient,
  buildFairUseRevisionCreate,
  buildFupPolicyViewModel,
  type FupTierDraftViewModel,
  type OperationResource,
  type PolicyMode,
} from '../../api';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n, type TranslationKey } from '../../i18n';
import { formatBpsMbps, formatBytesDual } from '../../lib/decimal';
import { validateFupPolicyDraft, type FupValidationError } from '../../lib/fup';

export function AdminFupPolicyPage(): React.ReactElement {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const fupQuery = useQuery({
    queryKey: ['admin', 'policy', 'fair_use'],
    queryFn: () => apiClient.listPolicies({ type: 'fair_use' }),
  });

  const [mode, setMode] = useState<PolicyMode>('shadow');
  const [effectiveAt, setEffectiveAt] = useState('2026-11-01T00:00:00Z');
  const [minimumRateBps, setMinimumRateBps] = useState('5000000');
  const [reason, setReason] = useState('Adjust fair-use tiers for next period');
  const [tiers, setTiers] = useState<FupTierDraftViewModel[]>([]);
  const [validationErrors, setValidationErrors] = useState<FupValidationError[]>([]);
  const [validationPassed, setValidationPassed] = useState(false);
  const [activeOp, setActiveOp] = useState<OperationResource | null>(null);

  const fupVm = fupQuery.data ? buildFupPolicyViewModel(fupQuery.data) : null;
  const fupDef = fupQuery.data?.items.find((d) => d.type === 'fair_use') ?? null;

  useEffect(() => {
    if (fupVm) {
      setMode(fupVm.mode);
      setEffectiveAt(fupVm.effective_at);
      setMinimumRateBps(fupVm.minimum_rate_bps);
      setTiers(fupVm.tiers);
    }
  }, [fupQuery.data]); // eslint-disable-line react-hooks/exhaustive-deps

  if (fupQuery.isLoading || !fupVm || !fupDef) {
    return (
      <div className="card" role="status">
        Loading Fair Use Policy...
      </div>
    );
  }

  const updateTier = (index: number, patch: Partial<FupTierDraftViewModel>) => {
    setTiers((prev) =>
      prev.map((tier, idx) => (idx === index ? { ...tier, ...patch } : tier)),
    );
  };

  const handleAddTier = () => {
    setTiers((prev) => [
      ...prev,
      {
        tier_index: prev.length,
        name_zh: '新公平使用級距',
        name_en: 'New Fair Use Tier',
        threshold_bytes: '2500000000000',
        down_bps: '10000000',
        up_bps: '5000000',
        priority: Math.min(7, prev.length + 1),
      },
    ]);
  };

  const handleRemoveTier = (index: number) => {
    setTiers((prev) =>
      prev.filter((_, idx) => idx !== index).map((t, i) => ({ ...t, tier_index: i })),
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setValidationPassed(false);

    const validation = validateFupPolicyDraft({
      mode,
      effective_at: effectiveAt,
      minimum_rate_bps: minimumRateBps,
      tiers,
    });

    if (!validation.valid) {
      setValidationErrors(validation.errors);
      return;
    }

    setValidationErrors([]);
    setValidationPassed(true);

    const createPayload = buildFairUseRevisionCreate(fupDef, {
      mode,
      effective_at: effectiveAt,
      minimum_rate_bps: minimumRateBps,
      tiers,
      reason,
    });

    const createdRev = await apiClient.createPolicyRevision(createPayload);
    const op = await apiClient.activatePolicyRevision(
      fupDef.definition_id,
      {
        revision_id: createdRev.revision_id,
        reason,
      },
      `"${fupDef.resource_version}"`,
    );
    setActiveOp(op);
    await queryClient.invalidateQueries({ queryKey: ['admin', 'policy', 'fair_use'] });
  };

  return (
    <div data-testid="admin-fup-policy-page">
      <div className="page-header">
        <div>
          <h1>{t('admin.fup.title')}</h1>
          <p className="page-subtitle mono">
            Active Revision: {fupVm.revision} (Generation {fupVm.generation})
          </p>
        </div>
      </div>

      {/* Mandatory TEST ONLY notice (AGENTS.md §1) */}
      <div className="alert alert-warning" role="note" data-testid="fup-test-only-banner">
        {t('admin.fup.testOnlyBanner')}
      </div>

      <OperationBanner operation={activeOp} onDismiss={() => setActiveOp(null)} />

      {validationErrors.length > 0 && (
        <div
          className="alert alert-danger"
          role="alert"
          data-testid="fup-validation-errors"
        >
          <ul style={{ margin: 0, paddingLeft: '1.25rem' }}>
            {validationErrors.map((err, idx) => (
              <li key={`${err.field}-${idx}`}>
                <span className="mono">{err.field}</span>:{' '}
                {t(err.messageKey as TranslationKey)}
              </li>
            ))}
          </ul>
        </div>
      )}

      {validationPassed && (
        <div
          className="alert alert-success"
          role="status"
          data-testid="fup-validation-passed"
        >
          {t('admin.fup.validationPassed')}
        </div>
      )}

      <form className="card" onSubmit={(e) => void handleSubmit(e)}>
        <div className="form-grid">
          <div className="form-group">
            <label htmlFor="fup-mode">{t('admin.fup.modeLabel')}</label>
            <select
              id="fup-mode"
              data-testid="fup-mode-select"
              value={mode}
              onChange={(e) => setMode(e.target.value as PolicyMode)}
            >
              <option value="shadow">{t('admin.fup.mode.shadow')}</option>
              <option value="enforce">{t('admin.fup.mode.enforce')}</option>
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="fup-effective-at">{t('admin.fup.effectiveAtLabel')}</label>
            <input
              id="fup-effective-at"
              className="mono"
              value={effectiveAt}
              onChange={(e) => setEffectiveAt(e.target.value)}
            />
          </div>

          <div className="form-group">
            <label htmlFor="fup-min-rate">{t('admin.fup.minRateLabel')}</label>
            <input
              id="fup-min-rate"
              data-testid="fup-min-rate-input"
              className="mono"
              value={minimumRateBps}
              onChange={(e) => setMinimumRateBps(e.target.value)}
            />
          </div>

          <div className="form-group">
            <label htmlFor="fup-reason">{t('admin.fup.reasonLabel')}</label>
            <input
              id="fup-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
          </div>
        </div>

        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginTop: '1rem',
            marginBottom: '0.75rem',
          }}
        >
          <h2>{t('admin.fup.tiersTitle')}</h2>
          <button
            type="button"
            className="btn btn-secondary"
            data-testid="fup-add-tier-btn"
            onClick={handleAddTier}
          >
            {t('admin.fup.addTierBtn')}
          </button>
        </div>

        <div className="table-wrapper">
          <table className="data-table" data-testid="fup-tiers-table">
            <thead>
              <tr>
                <th>Tier</th>
                <th>Name (zh-TW / en)</th>
                <th>threshold_bytes (Decimal)</th>
                <th>down_bps</th>
                <th>up_bps</th>
                <th>priority</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {tiers.map((tier, idx) => (
                <tr key={idx} data-testid={`fup-tier-row-${idx}`}>
                  <td className="mono">#{idx}</td>
                  <td>
                    <input
                      aria-label={`Tier ${idx} zh-TW name`}
                      value={tier.name_zh}
                      onChange={(e) => updateTier(idx, { name_zh: e.target.value })}
                      style={{ marginBottom: '0.25rem' }}
                    />
                    <input
                      aria-label={`Tier ${idx} en name`}
                      value={tier.name_en}
                      onChange={(e) => updateTier(idx, { name_en: e.target.value })}
                    />
                  </td>
                  <td>
                    <input
                      aria-label={`Tier ${idx} threshold_bytes`}
                      data-testid={`fup-tier-threshold-${idx}`}
                      className="mono"
                      value={tier.threshold_bytes}
                      onChange={(e) =>
                        updateTier(idx, { threshold_bytes: e.target.value })
                      }
                    />
                    <div style={{ fontSize: '0.7rem', color: 'var(--color-text-muted)' }}>
                      {(() => {
                        try {
                          return formatBytesDual(tier.threshold_bytes);
                        } catch {
                          return 'invalid';
                        }
                      })()}
                    </div>
                  </td>
                  <td>
                    <input
                      aria-label={`Tier ${idx} down_bps`}
                      data-testid={`fup-tier-down-${idx}`}
                      className="mono"
                      value={tier.down_bps}
                      onChange={(e) => updateTier(idx, { down_bps: e.target.value })}
                    />
                    <div style={{ fontSize: '0.7rem', color: 'var(--color-text-muted)' }}>
                      {(() => {
                        try {
                          return formatBpsMbps(tier.down_bps);
                        } catch {
                          return 'invalid';
                        }
                      })()}
                    </div>
                  </td>
                  <td>
                    <input
                      aria-label={`Tier ${idx} up_bps`}
                      data-testid={`fup-tier-up-${idx}`}
                      className="mono"
                      value={tier.up_bps}
                      onChange={(e) => updateTier(idx, { up_bps: e.target.value })}
                    />
                  </td>
                  <td>
                    <input
                      aria-label={`Tier ${idx} priority`}
                      type="number"
                      min={0}
                      max={7}
                      value={tier.priority}
                      onChange={(e) =>
                        updateTier(idx, { priority: Number(e.target.value) })
                      }
                      style={{ width: '70px' }}
                    />
                  </td>
                  <td>
                    {idx > 0 && (
                      <button
                        type="button"
                        className="btn btn-danger"
                        onClick={() => handleRemoveTier(idx)}
                      >
                        {t('admin.fup.removeTierBtn')}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div style={{ marginTop: '1rem' }}>
          <button type="submit" className="btn btn-primary" data-testid="fup-submit-btn">
            {t('admin.fup.validateAndPublishBtn')}
          </button>
        </div>
      </form>
    </div>
  );
}
