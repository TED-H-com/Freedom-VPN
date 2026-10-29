import React, { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  apiClient,
  buildDnsPolicyViewModel,
  type OperationResource,
  type PolicyRevisionCreate,
} from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { OperationBanner } from '../../components/common/OperationBanner';
import { PolicyActivationCard } from '../../components/common/PolicyActivationCard';
import { useI18n, type TranslationKey } from '../../i18n';
import { normalizeAndValidateDomainInput, previewDnsDomainMatch } from '../../lib/dns';

export function AdminDnsPolicyPage(): React.ReactElement {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const [domainInput, setDomainInput] = useState('');
  const [action, setAction] = useState<'block' | 'allow'>('block');
  const [includeSubdomains, setIncludeSubdomains] = useState(true);
  const [reason, setReason] = useState('Reported phishing domain');
  const [domainErrorKey, setDomainErrorKey] = useState<string | null>(null);
  const [apiError, setApiError] = useState<unknown>(null);
  const [activeOp, setActiveOp] = useState<OperationResource | null>(null);

  const [previewQuery, setPreviewQuery] = useState('sub.example.com');

  const [extraSchemaType, setExtraSchemaType] = useState<
    'dns_profile' | 'threat_feed' | 'release_support'
  >('dns_profile');
  const [extraReason, setExtraReason] = useState(
    'Operator created policy revision from portal',
  );

  const dnsQuery = useQuery({
    queryKey: ['admin', 'policies'],
    queryFn: () => apiClient.listPolicies(),
  });

  if (dnsQuery.error) {
    return (
      <div data-testid="admin-dns-policy-page">
        <ApiErrorAlert error={dnsQuery.error} testId="admin-dns-error" />
      </div>
    );
  }

  const dns = dnsQuery.data ? buildDnsPolicyViewModel(dnsQuery.data) : null;

  if (dnsQuery.isLoading || !dns || !dnsQuery.data) {
    return (
      <div className="card" role="status">
        Loading DNS policy...
      </div>
    );
  }

  const handleAddRule = async (e: React.FormEvent) => {
    e.preventDefault();
    setDomainErrorKey(null);
    setApiError(null);

    const validation = normalizeAndValidateDomainInput(domainInput);
    if (!validation.valid) {
      setDomainErrorKey(`dns.error.${validation.errorCode}`);
      return;
    }

    try {
      const ruleSetDef = dnsQuery.data.items.find((d) => d.type === 'domain_rule_set');
      const nextRules = [
        {
          rule_id: `ca60a69d-4247-4f93-9de4-${String(dns.rules.length + 20).padStart(12, '0')}`,
          profile_id: 'standard',
          domain: validation.canonicalDomain,
          include_subdomains: includeSubdomains,
          action,
          reason:
            action === 'allow' ? ('false_positive' as const) : ('admin_rule' as const),
          revision: String(BigInt(dns.desired_revision) + 1n),
        },
        ...dns.rules,
      ];

      await apiClient.createPolicyRevision({
        ...(ruleSetDef
          ? { definition_id: ruleSetDef.definition_id }
          : { code: 'standard-domain-rules' }),
        type: 'domain_rule_set',
        schema_version: 1,
        reason,
        payload: {
          rules: nextRules,
        },
      });
      setDomainInput('');
      await queryClient.invalidateQueries({ queryKey: ['admin', 'policies'] });
    } catch (err) {
      setApiError(err);
    }
  };

  const handlePublishBundle = async () => {
    setApiError(null);
    try {
      const ruleSetDef = dnsQuery.data?.items.find(
        (d) => d.definition_id === dns.rule_set_definition_id,
      );
      const targetRevId =
        ruleSetDef?.latest_revision?.revision_id ??
        dns.active_revision_id ??
        dns.rule_set_definition_id;
      const op = await apiClient.activatePolicyRevision(
        dns.rule_set_definition_id,
        {
          revision_id: targetRevId,
          reason: 'Activate compiled DNS rule bundle',
        },
        `"${dns.rule_set_resource_version}"`,
      );
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'policies'] });
    } catch (err) {
      setApiError(err);
    }
  };

  const handleRollback = async () => {
    setApiError(null);
    try {
      const rollbackRevId =
        dns.previous_revision_id ?? dns.active_revision_id ?? dns.rule_set_definition_id;
      const op = await apiClient.activatePolicyRevision(
        dns.rule_set_definition_id,
        {
          revision_id: rollbackRevId,
          reason: 'Rollback to last_good DNS revision',
        },
        `"${dns.rule_set_resource_version}"`,
      );
      setActiveOp(op);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'policies'] });
    } catch (err) {
      setApiError(err);
    }
  };

  const handleCreateExtraPolicyRevision = async (e: React.FormEvent) => {
    e.preventDefault();
    setApiError(null);
    try {
      const existingDef = dnsQuery.data?.items.find((d) => d.type === extraSchemaType);
      let req: PolicyRevisionCreate;
      if (extraSchemaType === 'dns_profile') {
        const ruleSetDef = dnsQuery.data?.items.find((d) => d.type === 'domain_rule_set');
        const feedDef = dnsQuery.data?.items.find((d) => d.type === 'threat_feed');
        req = {
          ...(existingDef
            ? { definition_id: existingDef.definition_id }
            : { code: 'standard' }),
          type: 'dns_profile',
          schema_version: 1,
          reason: extraReason,
          payload: {
            profile_code: 'standard',
            display_name: {
              'zh-TW': '標準 DNS 防護設定檔',
              en: 'Standard DNS Protection Profile',
            },
            domain_rule_set_revision_ids: ruleSetDef?.active_revision_id
              ? [ruleSetDef.active_revision_id]
              : [],
            threat_feed_revision_ids: feedDef?.active_revision_id
              ? [feedDef.active_revision_id]
              : [],
            block_response: 'nxdomain',
            blocked_ttl_seconds: 60,
          },
        };
      } else if (extraSchemaType === 'threat_feed') {
        req = {
          ...(existingDef
            ? { definition_id: existingDef.definition_id }
            : { code: 'urlhaus-malware' }),
          type: 'threat_feed',
          schema_version: 1,
          reason: extraReason,
          payload: {
            provider: 'abuse-ch-mirror',
            feed_code: 'urlhaus-malware',
            license_name: 'CC0-1.0',
            license_url: 'https://creativecommons.org/publicdomain/zero/1.0/',
            commercial_use_basis: 'Public domain dedication (CC0 1.0)',
            attribution: 'URLhaus / abuse.ch',
            source_url: 'https://feeds.example.com/threat/urlhaus-domains.txt',
            format: 'domain_list',
            categories: ['malware', 'phishing'],
            content_sha256:
              'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
            max_size_bytes: '10485760',
            valid_from: '2026-10-01T00:00:00Z',
            valid_until: '2026-12-31T23:59:59Z',
            update_interval_seconds: 86400,
          },
        };
      } else {
        const currentPayload =
          existingDef?.active_revision?.type === 'release_support'
            ? existingDef.active_revision.payload
            : {
                approved_release_ids: ['0195f100-0000-7000-8000-000000000010'],
                revoked_release_ids: [],
                minimum_serving_version: {
                  serving_release_ids: ['0195f100-0000-7000-8000-000000000010'],
                },
              };
        req = {
          ...(existingDef
            ? { definition_id: existingDef.definition_id }
            : { code: 'fleet-release-support' }),
          type: 'release_support',
          schema_version: 1,
          reason: extraReason,
          payload: currentPayload,
        };
      }
      await apiClient.createPolicyRevision(req);
      await queryClient.invalidateQueries({ queryKey: ['admin', 'policies'] });
    } catch (err) {
      setApiError(err);
    }
  };

  const previewResult = previewDnsDomainMatch(
    previewQuery,
    dns.rules,
    dns.threat_feeds.some((f) => f.enabled),
  );

  return (
    <div data-testid="admin-dns-policy-page">
      <div className="page-header">
        <div>
          <h1>{t('admin.dns.title')}</h1>
          <p className="page-subtitle">{t('admin.dns.subtitle')}</p>
        </div>
        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
          <button
            type="button"
            className="btn btn-primary"
            data-testid="dns-publish-bundle-btn"
            onClick={() => void handlePublishBundle()}
          >
            {t('admin.dns.publishBtn')}
          </button>
          <button
            type="button"
            className="btn btn-secondary"
            data-testid="dns-rollback-btn"
            onClick={() => void handleRollback()}
          >
            {t('admin.dns.rollbackBtn')} ({dns.last_good_revision})
          </button>
        </div>
      </div>

      <OperationBanner
        operation={activeOp}
        onDismiss={() => setActiveOp(null)}
        onTerminal={() => {
          void queryClient.invalidateQueries({ queryKey: ['admin', 'policies'] });
        }}
      />

      <ApiErrorAlert error={apiError} testId="dns-api-error" />

      {/* Resolver status */}
      <section className="card">
        <h2>{t('admin.dns.resolversTitle')}</h2>
        <div className="stat-grid" style={{ marginTop: '0.75rem' }}>
          {dns.resolvers.map((res) => (
            <div key={res.resolver_id} className="stat-card">
              <div style={{ fontWeight: 600 }}>{res.name}</div>
              <div className="mono" style={{ fontSize: '0.8rem' }}>
                Applied: {res.applied_revision} / Last Good: {res.last_good_revision}
              </div>
              <div>
                <span className="badge badge-success">{res.status}</span>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Add Domain Rule Form */}
      <section className="card">
        <h2>{t('admin.dns.addRuleTitle')}</h2>
        {domainErrorKey && (
          <div
            className="alert alert-danger"
            role="alert"
            data-testid="dns-domain-validation-error"
            style={{ marginTop: '0.5rem' }}
          >
            {t(domainErrorKey as TranslationKey)}
          </div>
        )}

        <form onSubmit={(e) => void handleAddRule(e)} style={{ marginTop: '0.75rem' }}>
          <div className="form-grid">
            <div className="form-group">
              <label htmlFor="dns-domain-input">{t('admin.dns.domainInput')}</label>
              <input
                id="dns-domain-input"
                data-testid="dns-domain-input"
                className="mono"
                value={domainInput}
                onChange={(e) => setDomainInput(e.target.value)}
                placeholder="bad-tracker.example.com"
              />
            </div>

            <div className="form-group">
              <label htmlFor="dns-action-select">{t('admin.dns.actionInput')}</label>
              <select
                id="dns-action-select"
                value={action}
                onChange={(e) => setAction(e.target.value as 'block' | 'allow')}
              >
                <option value="block">{t('admin.dns.action.block')}</option>
                <option value="allow">{t('admin.dns.action.allow')}</option>
              </select>
            </div>

            <div className="form-group">
              <label htmlFor="dns-reason-input">{t('admin.dns.reasonInput')}</label>
              <input
                id="dns-reason-input"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
              />
            </div>
          </div>

          <div className="form-group">
            <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <input
                type="checkbox"
                checked={includeSubdomains}
                onChange={(e) => setIncludeSubdomains(e.target.checked)}
                style={{ width: 'auto' }}
              />
              {t('admin.dns.includeSubdomains')}
            </label>
          </div>

          <button
            type="submit"
            className="btn btn-primary"
            data-testid="dns-add-rule-btn"
          >
            {t('admin.dns.addRuleSubmit')}
          </button>
        </form>
      </section>

      {/* Match Preview Tool (§12.4) */}
      <section className="card" data-testid="dns-preview-section">
        <h2>{t('admin.dns.previewTitle')}</h2>
        <div className="form-group" style={{ marginTop: '0.75rem' }}>
          <label htmlFor="dns-preview-input">{t('admin.dns.previewInput')}</label>
          <input
            id="dns-preview-input"
            data-testid="dns-preview-input"
            className="mono"
            value={previewQuery}
            onChange={(e) => setPreviewQuery(e.target.value)}
          />
        </div>

        {previewResult && (
          <div
            className={`alert ${
              previewResult.outcome.startsWith('blocked')
                ? 'alert-danger'
                : 'alert-success'
            }`}
            data-testid="dns-preview-result"
          >
            <div style={{ fontWeight: 700 }}>
              {t(`admin.dns.previewOutcome.${previewResult.outcome}` as TranslationKey)}
            </div>
            <div className="mono" style={{ fontSize: '0.8rem', marginTop: '0.25rem' }}>
              Normalized query: {previewResult.canonicalQuery} · Source:{' '}
              {previewResult.matchedRule?.domain ?? previewResult.outcome}
            </div>
          </div>
        )}

        <div className="alert alert-warning" style={{ marginBottom: 0 }}>
          {t('admin.dns.previewIsolationWarning')}
        </div>
      </section>

      {/* Current Rules List */}
      <section className="card">
        <h2>Active & Draft Rules ({dns.rules.length})</h2>
        <div className="table-wrapper" style={{ marginTop: '0.75rem' }}>
          <table className="data-table" data-testid="dns-rules-table">
            <thead>
              <tr>
                <th>Domain (Normalized)</th>
                <th>Action</th>
                <th>include_subdomains</th>
                <th>Reason</th>
              </tr>
            </thead>
            <tbody>
              {dns.rules.map((r) => (
                <tr key={r.rule_id}>
                  <td className="mono">{r.domain}</td>
                  <td>
                    <span
                      className={`badge ${
                        r.action === 'block' ? 'badge-danger' : 'badge-success'
                      }`}
                    >
                      {r.action}
                    </span>
                  </td>
                  <td className="mono">{String(r.include_subdomains)}</td>
                  <td>{r.reason}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* Create Revision for dns_profile / threat_feed / release_support */}
      <section className="card" data-testid="extra-policy-creator-card">
        <h2>Create DNS Profile / Threat Feed / Release Support Revision</h2>
        <form
          onSubmit={(e) => void handleCreateExtraPolicyRevision(e)}
          style={{ marginTop: '0.75rem' }}
        >
          <div className="form-grid">
            <div className="form-group">
              <label htmlFor="extra-policy-schema-select">
                {t('admin.policies.schemaSelectLabel')}
              </label>
              <select
                id="extra-policy-schema-select"
                data-testid="extra-policy-schema-select"
                value={extraSchemaType}
                onChange={(e) =>
                  setExtraSchemaType(
                    e.target.value as 'dns_profile' | 'threat_feed' | 'release_support',
                  )
                }
              >
                <option value="dns_profile">dns_profile</option>
                <option value="threat_feed">threat_feed</option>
                <option value="release_support">release_support</option>
              </select>
            </div>
            <div className="form-group">
              <label htmlFor="extra-policy-reason-input">Reason</label>
              <input
                id="extra-policy-reason-input"
                value={extraReason}
                onChange={(e) => setExtraReason(e.target.value)}
              />
            </div>
          </div>
          <button
            type="submit"
            className="btn btn-secondary"
            data-testid="create-extra-policy-revision-btn"
          >
            {t('admin.policies.createRevisionTitle')}
          </button>
        </form>
      </section>

      <PolicyActivationCard
        definitions={dnsQuery.data.items}
        allowedTypes={[
          'domain_rule_set',
          'dns_profile',
          'threat_feed',
          'release_support',
        ]}
      />
    </div>
  );
}
