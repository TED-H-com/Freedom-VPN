import React, { useEffect, useState } from 'react';
import { portalApi } from '../../api';
import { useI18n } from '../../i18n';

type OsPlatform = 'windows' | 'macos' | 'ios' | 'android';

export function MemberGettingStartedPage(): React.ReactElement {
  const { t } = useI18n();
  const [managementUrl, setManagementUrl] = useState('https://netbird.vpn.example.com');
  const [selectedOs, setSelectedOs] = useState<OsPlatform>('macos');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    void portalApi.getMemberMe().then((res) => setManagementUrl(res.data.management_url));
  }, []);

  const handleCopy = async () => {
    if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
      await navigator.clipboard.writeText(managementUrl);
    }
    setCopied(true);
  };

  return (
    <div>
      <header className="page-header">
        <h1>{t('member.gettingStarted.title')}</h1>
        <p className="page-subtitle">{t('member.gettingStarted.subtitle')}</p>
      </header>

      <section className="card" aria-labelledby="mgmt-url-heading">
        <h2 id="mgmt-url-heading">{t('member.gettingStarted.mgmtUrlLabel')}</h2>
        <div className="code-box" data-testid="self-hosted-mgmt-url">
          {managementUrl}
        </div>
        <div className="btn-row" style={{ marginTop: 12 }}>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => void handleCopy()}
          >
            {copied ? 'Copied!' : t('member.gettingStarted.copyUrlBtn')}
          </button>
        </div>
      </section>

      <section className="card" aria-labelledby="os-steps-heading">
        <h2 id="os-steps-heading">{t('member.gettingStarted.osSelectLabel')}</h2>
        <div className="form-field" style={{ maxWidth: 280, marginBottom: 14 }}>
          <label htmlFor="os-platform-select">
            {t('member.gettingStarted.osSelectLabel')}
          </label>
          <select
            id="os-platform-select"
            value={selectedOs}
            onChange={(e) => setSelectedOs(e.target.value as OsPlatform)}
            data-testid="os-platform-select"
          >
            <option value="windows">Windows</option>
            <option value="macos">macOS</option>
            <option value="ios">iOS</option>
            <option value="android">Android</option>
          </select>
        </div>

        <ol style={{ paddingLeft: 20, lineHeight: 1.8 }}>
          <li>
            <strong>[{selectedOs.toUpperCase()}]</strong>{' '}
            {t('member.gettingStarted.step1')}
          </li>
          <li>{t('member.gettingStarted.step2')}</li>
          <li>{t('member.gettingStarted.step3')}</li>
          <li>{t('member.gettingStarted.step4')}</li>
        </ol>

        {selectedOs === 'android' && (
          <div
            className="alert alert-warning"
            style={{ marginTop: 14 }}
            data-testid="android-private-dns-warning"
          >
            <strong>{t('member.gettingStarted.androidPrivateDnsWarning')}</strong>
          </div>
        )}
      </section>

      <section
        className="card alert-warning"
        aria-labelledby="kill-switch-limitation-heading"
        data-testid="kill-switch-limitation-card"
      >
        <h2 id="kill-switch-limitation-heading">
          {t('member.gettingStarted.killSwitchWarningTitle')}
        </h2>
        <p style={{ margin: 0 }}>{t('member.gettingStarted.killSwitchWarningBody')}</p>
      </section>
    </div>
  );
}
