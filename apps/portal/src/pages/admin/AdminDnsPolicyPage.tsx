import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { portalApi, useOperationPoller, type DnsPolicyState } from '../../api';
import { OperationBanner } from '../../components/common/OperationBanner';
import { useI18n, type TranslationKey } from '../../i18n';
import { normalizeAndValidateDomain, previewDnsMatch } from '../../lib/dns';

export function AdminDnsPolicyPage(): React.ReactElement {
  const { t } = useI18n();
  const [dnsState, setDnsState] = useState<DnsPolicyState | null>(null);
  const [domainInput, setDomainInput] = useState('');
  const [actionInput, setActionInput] = useState<'block' | 'allow'>('block');
  const [includeSubdomains, setIncludeSubdomains] = useState(true);
  const [reasonInput, setReasonInput] = useState('admin_rule');
  const [ruleErrorKey, setRuleErrorKey] = useState<TranslationKey | null>(null);

  const [previewQuery, setPreviewQuery] = useState('sub.tracker.example.com');

  const loadDns = useCallback(async () => {
    const res = await portalApi.getDnsPolicy();
    setDnsState(res.data);
  }, []);

  const poller = useOperationPoller({
    scope: 'admin',
    onTerminal: () => {
      void loadDns();
    },
  });

  useEffect(() => {
    void loadDns();
  }, [loadDns]);

  const previewResult = useMemo(() => {
    if (!dnsState) return null;
    return previewDnsMatch(previewQuery, dnsState.rules, true);
  }, [dnsState, previewQuery]);

  if (!dnsState) {
    return <div className="card">Loading DNS policy...</div>;
  }

  const handleAddRule = async (e: React.FormEvent) => {
    e.preventDefault();
    const norm = normalizeAndValidateDomain(domainInput);
    if (!norm.valid) {
      setRuleErrorKey(`dns.error.${norm.errorCode ?? 'INVALID_LABEL'}` as TranslationKey);
      return;
    }
    setRuleErrorKey(null);
    const res = await portalApi.addDnsRule(
      {
        profile_id: dnsState.profile_id,
        domain: norm.canonicalDomain,
        include_subdomains: includeSubdomains,
        action: actionInput,
        reason: reasonInput.trim() || 'admin_rule',
      },
      { ifMatch: dnsState.etag },
    );
    setDnsState(res.data);
    setDomainInput('');
  };

  const handlePublish = async () => {
    const opRes = await portalApi.publishDnsPolicy();
    poller.startTracking(opRes);
  };

  const handleRollback = async () => {
    const opRes = await portalApi.rollbackDnsPolicy();
    poller.startTracking(opRes);
  };

  return (
    <div>
      <header className="page-header">
        <h1>{t('admin.dns.title')}</h1>
        <p className="page-subtitle">{t('admin.dns.subtitle')}</p>
      </header>

      <OperationBanner
        operation={poller.operation}
        requestId={poller.requestId}
        isPolling={poller.isPolling}
        onDismiss={poller.reset}
      />

      <section className="card" aria-labelledby="dns-resolvers-heading">
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: 12,
          }}
        >
          <h2 id="dns-resolvers-heading" style={{ margin: 0 }}>
            {t('admin.dns.resolversTitle')} (Active rev #{dnsState.active_revision} /
            Desired rev #{dnsState.desired_revision})
          </h2>
          <div className="btn-row">
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => void handlePublish()}
              data-testid="publish-dns-btn"
            >
              {t('admin.dns.publishBtn')}
            </button>
            <button
              type="button"
              className="btn"
              onClick={() => void handleRollback()}
              data-testid="rollback-dns-btn"
            >
              {t('admin.dns.rollbackBtn')}
            </button>
          </div>
        </div>
        <div className="table-wrapper" style={{ marginTop: 12 }}>
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Resolver</th>
                <th scope="col">Overlay IPv4</th>
                <th scope="col">Applied Revision</th>
                <th scope="col">Status</th>
              </tr>
            </thead>
            <tbody>
              {dnsState.resolvers.map((res) => (
                <tr key={res.resolver_id}>
                  <td>
                    <code>{res.name}</code>
                  </td>
                  <td>
                    <code>{res.ipv4}</code>
                  </td>
                  <td>rev #{res.applied_revision}</td>
                  <td>
                    <span className="badge badge-success">{res.status}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="card" aria-labelledby="dns-preview-heading">
        <h2 id="dns-preview-heading">{t('admin.dns.previewTitle')}</h2>
        <div className="form-field" style={{ marginBottom: 12 }}>
          <label htmlFor="dns-preview-input">{t('admin.dns.previewInput')}</label>
          <input
            id="dns-preview-input"
            type="text"
            value={previewQuery}
            onChange={(e) => setPreviewQuery(e.target.value)}
            data-testid="dns-preview-input"
          />
        </div>
        {previewResult && (
          <div
            className={
              !previewResult.validQuery
                ? 'alert alert-danger'
                : previewResult.outcome.startsWith('blocked')
                  ? 'alert alert-warning'
                  : 'alert alert-success'
            }
            data-testid="dns-preview-result"
          >
            {!previewResult.validQuery ? (
              <strong>
                {t(
                  `dns.error.${previewResult.errorCode ?? 'INVALID_LABEL'}` as TranslationKey,
                )}
              </strong>
            ) : (
              <>
                <div>
                  <strong>
                    {t(
                      `admin.dns.previewOutcome.${previewResult.outcome}` as TranslationKey,
                    )}
                  </strong>{' '}
                  (Canonical: <code>{previewResult.canonicalQuery}</code>)
                </div>
                {previewResult.matchedRule && (
                  <div style={{ fontSize: '0.86rem', marginTop: 4 }}>
                    Matched Rule: <code>{previewResult.matchedRule.domain}</code> (
                    include_subdomains:{' '}
                    {String(previewResult.matchedRule.include_subdomains)}, reason:{' '}
                    {previewResult.matchedRule.reason})
                  </div>
                )}
                <div style={{ fontSize: '0.84rem', marginTop: 6 }}>
                  {t('admin.dns.previewIsolationWarning')}
                </div>
              </>
            )}
          </div>
        )}
      </section>

      <form
        className="card"
        aria-labelledby="add-dns-rule-heading"
        onSubmit={(e) => void handleAddRule(e)}
      >
        <h2 id="add-dns-rule-heading">{t('admin.dns.addRuleTitle')}</h2>
        {ruleErrorKey && (
          <div className="alert alert-danger" role="alert" data-testid="dns-rule-error">
            {t(ruleErrorKey)}
          </div>
        )}
        <div className="form-grid">
          <div className="form-field">
            <label htmlFor="dns-domain-input">{t('admin.dns.domainInput')}</label>
            <input
              id="dns-domain-input"
              type="text"
              value={domainInput}
              onChange={(e) => setDomainInput(e.target.value)}
              placeholder="example.com"
              data-testid="dns-domain-input"
            />
          </div>
          <div className="form-field">
            <label htmlFor="dns-action-select">{t('admin.dns.actionInput')}</label>
            <select
              id="dns-action-select"
              value={actionInput}
              onChange={(e) => setActionInput(e.target.value as 'block' | 'allow')}
            >
              <option value="block">{t('admin.dns.action.block')}</option>
              <option value="allow">{t('admin.dns.action.allow')}</option>
            </select>
          </div>
          <div className="form-field">
            <label htmlFor="dns-reason-input">{t('admin.dns.reasonInput')}</label>
            <input
              id="dns-reason-input"
              type="text"
              value={reasonInput}
              onChange={(e) => setReasonInput(e.target.value)}
            />
          </div>
        </div>
        <div style={{ marginBottom: 14 }}>
          <label
            htmlFor="dns-subdomains-checkbox"
            style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}
          >
            <input
              id="dns-subdomains-checkbox"
              type="checkbox"
              checked={includeSubdomains}
              onChange={(e) => setIncludeSubdomains(e.target.checked)}
            />
            <span>{t('admin.dns.includeSubdomains')}</span>
          </label>
        </div>
        <button type="submit" className="btn btn-primary" data-testid="add-dns-rule-btn">
          {t('admin.dns.addRuleSubmit')}
        </button>

        <div className="table-wrapper" style={{ marginTop: 18 }}>
          <table className="data-table" data-testid="dns-rules-table">
            <thead>
              <tr>
                <th scope="col">Domain</th>
                <th scope="col">Action</th>
                <th scope="col">include_subdomains</th>
                <th scope="col">Reason</th>
                <th scope="col">Revision</th>
              </tr>
            </thead>
            <tbody>
              {dnsState.rules.map((r) => (
                <tr key={r.rule_id}>
                  <td>
                    <code>{r.domain}</code>
                  </td>
                  <td>
                    <span
                      className={
                        r.action === 'block'
                          ? 'badge badge-danger'
                          : 'badge badge-success'
                      }
                    >
                      {r.action}
                    </span>
                  </td>
                  <td>{r.include_subdomains ? 'true' : 'false'}</td>
                  <td>{r.reason}</td>
                  <td>#{r.revision}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </form>
    </div>
  );
}
