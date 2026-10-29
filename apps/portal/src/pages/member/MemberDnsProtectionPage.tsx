import React from 'react';
import { useI18n } from '../../i18n';

export function MemberDnsProtectionPage(): React.ReactElement {
  const { t } = useI18n();

  return (
    <div data-testid="member-dns-protection-page">
      <header className="page-header">
        <h1>{t('member.dns.title')}</h1>
      </header>

      <section
        className="card"
        aria-labelledby="dns-scope-heading"
        data-testid="dns-scope-card"
      >
        <h2 id="dns-scope-heading">{t('member.dns.scopeTitle')}</h2>
        <p>{t('member.dns.scopeBody')}</p>
      </section>

      <section
        className="card alert-warning"
        aria-labelledby="dns-limitations-heading"
        data-testid="dns-limitations-card"
      >
        <h2 id="dns-limitations-heading">{t('member.dns.limitationsTitle')}</h2>
        <p>{t('member.dns.limitationsBody')}</p>
      </section>

      <section className="card" aria-labelledby="dns-false-positive-heading">
        <h2 id="dns-false-positive-heading">{t('member.dns.falsePositiveTitle')}</h2>
        <p>{t('member.dns.falsePositiveBody')}</p>
      </section>
    </div>
  );
}
