import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../../api';
import { useSession } from '../../context/SessionContext';
import { useI18n } from '../../i18n';

type OsPlatform = 'Windows' | 'macOS' | 'iOS' | 'Android' | 'Linux';

const MANAGEMENT_URL = 'https://netbird.vpn.example.com';

export function MemberGettingStartedPage(): React.ReactElement {
  const { t } = useI18n();
  const { persona } = useSession();
  const [selectedOs, setSelectedOs] = useState<OsPlatform>('Android');
  const [copied, setCopied] = useState(false);

  const selfQuery = useQuery({
    queryKey: ['member', 'me', persona],
    queryFn: () => apiClient.getMemberSelf(),
  });

  if (selfQuery.isLoading || !selfQuery.data) {
    return (
      <div className="card" role="status">
        Loading setup instructions...
      </div>
    );
  }

  const handleCopyUrl = async () => {
    try {
      if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
        await navigator.clipboard.writeText(MANAGEMENT_URL);
      }
      setCopied(true);
    } catch {
      setCopied(true);
    }
  };

  return (
    <div data-testid="member-getting-started-page">
      <div className="page-header">
        <div>
          <h1>{t('member.gettingStarted.title')}</h1>
          <p className="page-subtitle">{t('member.gettingStarted.subtitle')}</p>
        </div>
      </div>

      {/* Mandatory No Always-On / Kill-Switch Disclosure (§7.3) */}
      <div
        className="alert alert-warning"
        role="note"
        data-testid="no-kill-switch-disclosure"
      >
        <div style={{ fontWeight: 700 }}>
          {t('member.gettingStarted.killSwitchWarningTitle')}
        </div>
        <div style={{ marginTop: '0.25rem' }}>
          {t('member.gettingStarted.killSwitchWarningBody')}
        </div>
      </div>

      {/* Self-hosted Management URL */}
      <section className="card">
        <h2>{t('member.gettingStarted.mgmtUrlLabel')}</h2>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '0.75rem',
            flexWrap: 'wrap',
            marginTop: '0.75rem',
          }}
        >
          <code
            className="mono"
            style={{
              background: '#f1f5f9',
              padding: '0.5rem 0.75rem',
              borderRadius: 'var(--radius-sm)',
              fontWeight: 600,
            }}
            data-testid="management-url-code"
          >
            {MANAGEMENT_URL}
          </code>
          <button
            type="button"
            className="btn btn-secondary"
            data-testid="copy-mgmt-url-btn"
            onClick={() => void handleCopyUrl()}
          >
            {copied
              ? t('admin.enrollment.oneTime.copied')
              : t('member.gettingStarted.copyUrlBtn')}
          </button>
        </div>
      </section>

      {/* Per-OS Setup Steps */}
      <section className="card">
        <div className="form-group" style={{ maxWidth: '300px' }}>
          <label htmlFor="os-platform-select">
            {t('member.gettingStarted.osSelectLabel')}
          </label>
          <select
            id="os-platform-select"
            data-testid="os-platform-select"
            value={selectedOs}
            onChange={(e) => setSelectedOs(e.target.value as OsPlatform)}
          >
            <option value="Windows">Windows</option>
            <option value="macOS">macOS</option>
            <option value="iOS">iOS</option>
            <option value="Android">Android</option>
            <option value="Linux">Linux</option>
          </select>
        </div>

        <ol style={{ paddingLeft: '1.25rem', lineHeight: 1.8 }}>
          <li>{t('member.gettingStarted.step1')}</li>
          <li>{t('member.gettingStarted.step2')}</li>
          <li>{t('member.gettingStarted.step3')}</li>
          <li>{t('member.gettingStarted.step4')}</li>
        </ol>

        {selectedOs === 'Android' && (
          <div
            className="alert alert-info"
            style={{ marginTop: '1rem', marginBottom: 0 }}
            data-testid="android-private-dns-warning"
          >
            {t('member.gettingStarted.androidPrivateDnsWarning')}
          </div>
        )}
      </section>
    </div>
  );
}
