import React, { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import type { components } from '@contracts/schema';
import {
  apiClient,
  buildFairUseRevisionCreate,
  buildFupPolicyViewModel,
  type FupTierDraftViewModel,
  type OperationResource,
  type PolicyMode,
  type PolicyRevisionCreate,
} from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { OperationBanner } from '../../components/common/OperationBanner';
import { PolicyActivationCard } from '../../components/common/PolicyActivationCard';
import { useI18n, type TranslationKey } from '../../i18n';
import { formatBpsMbps, formatBytesDual } from '../../lib/decimal';
import { validateFupPolicyDraft, type FupValidationError } from '../../lib/fup';

type FairUseOverride = components['schemas']['FairUseOverride'];

const DEFAULT_FALLBACK_TIERS: FupTierDraftViewModel[] = [
  {
    tier_index: 0,
    name_zh: '標準速率',
    name_en: 'Normal Speed',
    threshold_bytes: '0',
    down_bps: '300000000',
    up_bps: '300000000',
    priority: 1,
  },
  {
    tier_index: 1,
    name_zh: '公平使用調整級距 1',
    name_en: 'Fair Use Adjusted Tier 1',
    threshold_bytes: '1000000000000',
    down_bps: '25000000',
    up_bps: '10000000',
    priority: 3,
  },
];

export function AdminFupPolicyPage(): React.ReactElement {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const fupQuery = useQuery({
    queryKey: ['admin', 'policies'],
    queryFn: () => apiClient.listPolicies(),
  });

  const [mode, setMode] = useState<PolicyMode>('shadow');
  const [effectiveAt, setEffectiveAt] = useState('2026-11-01T00:00:00Z');
  const [minimumRateBps, setMinimumRateBps] = useState('5000000');
  const [reason, setReason] = useState('Adjust fair-use tiers for next period');
  const [tiers, setTiers] = useState<FupTierDraftViewModel[]>(DEFAULT_FALLBACK_TIERS);
  const [overrides, setOverrides] = useState<FairUseOverride[]>([]);

  const [newOverrideMemberId, setNewOverrideMemberId] = useState('');
  const [newOverrideDownBps, setNewOverrideDownBps] = useState('100000000');
  const [newOverrideUpBps, setNewOverrideUpBps] = useState('100000000');
  const [newOverrideValidUntil, setNewOverrideValidUntil] =
    useState('2026-12-31T00:00:00Z');
  const [newOverrideReason, setNewOverrideReason] = useState(
    'Temporary support override',
  );

  const [planCode, setPlanCode] = useState('standard-monthly');
  const [planDeviceLimit, setPlanDeviceLimit] = useState(3);
  const [planPriceMinor, setPlanPriceMinor] = useState(19900);
  const [planCurrency, setPlanCurrency] = useState('TWD');
  const [planReason, setPlanReason] = useState('Update plan revision');

  const [validationErrors, setValidationErrors] = useState<FupValidationError[]>([]);
  const [validationPassed, setValidationPassed] = useState(false);
  const [activeOp, setActiveOp] = useState<OperationResource | null>(null);
  const [apiError, setApiError] = useState<unknown>(null);

  const fupVm = fupQuery.data ? buildFupPolicyViewModel(fupQuery.data) : null;
  const fupDef = fupQuery.data?.items.find((d) => d.type === 'fair_use') ?? null;
  const planDef = fupQuery.data?.items.find((d) => d.type === 'plan') ?? null;
  const dnsProfileDef =
    fupQuery.data?.items.find((d) => d.type === 'dns_profile') ?? null;

  useEffect(() => {
    if (fupVm) {
      setMode(fupVm.mode);
      setEffectiveAt(fupVm.effective_at);
      setMinimumRateBps(fupVm.minimum_rate_bps);
      setTiers(fupVm.tiers);
    }
    const rev = fupDef?.latest_revision ?? fupDef?.active_revision;
    if (rev && rev.type === 'fair_use') {
      setOverrides(rev.payload.overrides ?? []);
    }
  }, [fupQuery.data]); // eslint-disable-line react-hooks/exhaustive-deps

  if (fupQuery.error) {
    return (
      <div data-testid="admin-fup-policy-page">
        <ApiErrorAlert error={fupQuery.error} testId="admin-fup-error" />
      </div>
    );
  }

  if (fupQuery.isLoading || !fupQuery.data) {
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

  const handleAddOverride = () => {
    if (!newOverrideMemberId.trim()) return;
    setOverrides((prev) => [
      ...prev.filter((o) => o.member_id !== newOverrideMemberId.trim()),
      {
        member_id: newOverrideMemberId.trim(),
        down_bps: Number.parseInt(newOverrideDownBps, 10) || 100000000,
        up_bps: Number.parseInt(newOverrideUpBps, 10) || 100000000,
        priority: 1,
        valid_until: newOverrideValidUntil.trim(),
        reason: newOverrideReason.trim() || 'Admin override',
        actor_id: '0195f100-0000-7000-8000-000000000099',
      },
    ]);
    setNewOverrideMemberId('');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setValidationPassed(false);
    setApiError(null);

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

    try {
      const createPayload: PolicyRevisionCreate = fupDef
        ? buildFairUseRevisionCreate(fupDef, {
            mode,
            effective_at: effectiveAt,
            minimum_rate_bps: minimumRateBps,
            tiers,
            reason,
          })
        : {
            code: 'standard-fup',
            type: 'fair_use',
            schema_version: 1,
            reason,
            payload: {
              mode,
              effective_at: effectiveAt,
              effective_period: 'next_full_period',
              accounting_basis: {
                period_type: 'utc_calendar_month',
                metering_version: 'forward_admission_v1',
                measure: 'total_bytes',
              },
              tiers: tiers.map((tier) => ({
                threshold_bytes: tier.threshold_bytes,
                down_bps: Number.parseInt(tier.down_bps, 10),
                up_bps: Number.parseInt(tier.up_bps, 10),
                priority: tier.priority,
                display_name: {
                  'zh-TW': tier.name_zh,
                  en: tier.name_en,
                },
              })),
              minimum_rate: {
                down_bps: Number.parseInt(minimumRateBps, 10),
                up_bps: Number.parseInt(minimumRateBps, 10),
              },
              overrides,
            },
          };
      if (createPayload.type === 'fair_use') {
        createPayload.payload.overrides = overrides;
      }

      const createdRev = await apiClient.createPolicyRevision(createPayload);
      if (fupDef) {
        const op = await apiClient.activatePolicyRevision(
          fupDef.definition_id,
          {
            revision_id: createdRev.revision_id,
            reason,
          },
          `"${fupDef.resource_version}"`,
        );
        setActiveOp(op);
      }
      await queryClient.invalidateQueries({ queryKey: ['admin', 'policies'] });
      await queryClient.invalidateQueries({ queryKey: ['admin', 'policy', 'fair_use'] });
    } catch (err) {
      setApiError(err);
    }
  };

  const handleCreatePlanRevision = async (e: React.FormEvent) => {
    e.preventDefault();
    setApiError(null);
    try {
      const fupRevId =
        fupDef?.active_revision_id ??
        fupDef?.latest_revision?.revision_id ??
        '0195f100-0000-7000-8000-000000000051';
      const dnsRevId =
        dnsProfileDef?.active_revision_id ??
        dnsProfileDef?.latest_revision?.revision_id ??
        '0195f100-0000-7000-8000-000000000055';
      await apiClient.createPolicyRevision({
        ...(planDef ? { definition_id: planDef.definition_id } : { code: planCode }),
        type: 'plan',
        schema_version: 1,
        reason: planReason,
        payload: {
          display_name: {
            'zh-TW': 'Freedom VPN 標準方案 (TEST ONLY)',
            en: 'Freedom VPN Standard Plan (TEST ONLY)',
          },
          device_limit: Number(planDeviceLimit) || 3,
          exit_group_ids: ['63575bae-2d23-45c0-ae11-ba6bb9ace34a'],
          price: {
            amount_minor: Number(planPriceMinor) || 19900,
            currency: planCurrency.trim() || 'TWD',
            billing_interval: 'month',
            provider: 'stripe',
            provider_price_ref: 'price_test_only_standard_monthly',
          },
          rate_limit: null,
          fair_use_revision_id: fupRevId,
          dns_profile_revision_id: dnsRevId,
        },
      });
      await queryClient.invalidateQueries({ queryKey: ['admin', 'policies'] });
    } catch (err) {
      setApiError(err);
    }
  };

  return (
    <div data-testid="admin-fup-policy-page">
      <div className="page-header">
        <div>
          <h1>{t('admin.fup.title')}</h1>
          <p className="page-subtitle mono">
            Active Revision: {fupVm?.revision ?? '0'} (Generation{' '}
            {fupVm?.generation ?? '0'})
          </p>
        </div>
      </div>

      {/* Mandatory TEST ONLY notice (AGENTS.md §1) */}
      <div className="alert alert-warning" role="note" data-testid="fup-test-only-banner">
        {t('admin.fup.testOnlyBanner')}
      </div>

      <ApiErrorAlert error={apiError} testId="fup-api-error" />

      <OperationBanner
        operation={activeOp}
        onDismiss={() => setActiveOp(null)}
        onTerminal={() => {
          void queryClient.invalidateQueries({ queryKey: ['admin', 'policies'] });
        }}
      />

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

        {/* Per-member Fair-Use Overrides (§11.1) */}
        <div style={{ marginTop: '1.25rem' }}>
          <h3>{t('admin.fup.overridesTitle')}</h3>
          <div className="form-grid" style={{ alignItems: 'end', marginTop: '0.5rem' }}>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label htmlFor="fup-override-member-id">
                {t('admin.fup.overrideMemberId')}
              </label>
              <input
                id="fup-override-member-id"
                data-testid="fup-override-member-id-input"
                className="mono"
                value={newOverrideMemberId}
                placeholder="Member UUID"
                onChange={(e) => setNewOverrideMemberId(e.target.value)}
              />
            </div>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label htmlFor="fup-override-down">down_bps</label>
              <input
                id="fup-override-down"
                className="mono"
                value={newOverrideDownBps}
                onChange={(e) => setNewOverrideDownBps(e.target.value)}
              />
            </div>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label htmlFor="fup-override-up">up_bps</label>
              <input
                id="fup-override-up"
                className="mono"
                value={newOverrideUpBps}
                onChange={(e) => setNewOverrideUpBps(e.target.value)}
              />
            </div>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label htmlFor="fup-override-until">
                {t('admin.fup.overrideValidUntil')}
              </label>
              <input
                id="fup-override-until"
                className="mono"
                value={newOverrideValidUntil}
                onChange={(e) => setNewOverrideValidUntil(e.target.value)}
              />
            </div>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label htmlFor="fup-override-reason">Reason</label>
              <input
                id="fup-override-reason"
                value={newOverrideReason}
                onChange={(e) => setNewOverrideReason(e.target.value)}
              />
            </div>
            <div>
              <button
                type="button"
                className="btn btn-secondary"
                data-testid="fup-add-override-btn"
                onClick={handleAddOverride}
              >
                {t('admin.fup.addOverrideBtn')}
              </button>
            </div>
          </div>

          {overrides.length > 0 && (
            <div className="table-wrapper" style={{ marginTop: '0.75rem' }}>
              <table className="data-table" data-testid="fup-overrides-table">
                <thead>
                  <tr>
                    <th>member_id</th>
                    <th>down_bps / up_bps</th>
                    <th>valid_until</th>
                    <th>reason</th>
                    <th>Action</th>
                  </tr>
                </thead>
                <tbody>
                  {overrides.map((ov) => (
                    <tr
                      key={ov.member_id}
                      data-testid={`fup-override-row-${ov.member_id}`}
                    >
                      <td className="mono">{ov.member_id}</td>
                      <td className="mono">
                        {ov.down_bps} / {ov.up_bps}
                      </td>
                      <td className="mono">{ov.valid_until}</td>
                      <td>{ov.reason}</td>
                      <td>
                        <button
                          type="button"
                          className="btn btn-danger"
                          onClick={() =>
                            setOverrides((prev) =>
                              prev.filter((x) => x.member_id !== ov.member_id),
                            )
                          }
                        >
                          {t('admin.fup.removeTierBtn')}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        <div style={{ marginTop: '1rem' }}>
          <button type="submit" className="btn btn-primary" data-testid="fup-submit-btn">
            {t('admin.fup.validateAndPublishBtn')}
          </button>
        </div>
      </form>

      {/* Plan Policy Revision Creator (schema: plan) */}
      <section className="card" data-testid="plan-policy-creator-card">
        <h2>Create Plan Policy Revision (TEST ONLY)</h2>
        <form
          onSubmit={(e) => void handleCreatePlanRevision(e)}
          style={{ marginTop: '0.75rem' }}
        >
          <div className="form-grid">
            <div className="form-group">
              <label htmlFor="plan-code-input">Plan Code</label>
              <input
                id="plan-code-input"
                className="mono"
                value={planCode}
                onChange={(e) => setPlanCode(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label htmlFor="plan-device-limit-input">Device Limit</label>
              <input
                id="plan-device-limit-input"
                type="number"
                min={1}
                max={20}
                value={planDeviceLimit}
                onChange={(e) => setPlanDeviceLimit(Number(e.target.value))}
              />
            </div>
            <div className="form-group">
              <label htmlFor="plan-price-minor-input">
                Price Minor Units (TEST ONLY)
              </label>
              <input
                id="plan-price-minor-input"
                type="number"
                value={planPriceMinor}
                onChange={(e) => setPlanPriceMinor(Number(e.target.value))}
              />
            </div>
            <div className="form-group">
              <label htmlFor="plan-currency-input">Currency</label>
              <input
                id="plan-currency-input"
                className="mono"
                value={planCurrency}
                onChange={(e) => setPlanCurrency(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label htmlFor="plan-reason-input">Reason</label>
              <input
                id="plan-reason-input"
                value={planReason}
                onChange={(e) => setPlanReason(e.target.value)}
              />
            </div>
          </div>
          <button
            type="submit"
            className="btn btn-secondary"
            data-testid="create-plan-revision-btn"
          >
            Create Plan Revision
          </button>
        </form>
      </section>

      <PolicyActivationCard
        definitions={fupQuery.data.items}
        allowedTypes={['fair_use', 'plan']}
      />
    </div>
  );
}
