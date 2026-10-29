import React from 'react';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useSession, type MockPersona } from '../../context/SessionContext';
import { useI18n, type Locale } from '../../i18n';

export function AppLayout(): React.ReactElement {
  const { t, locale, setLocale } = useI18n();
  const {
    session,
    persona,
    switchPersona,
    sessionExpired,
    simulateSessionExpired,
    recoverSession,
    logout,
  } = useSession();
  const navigate = useNavigate();

  const handlePersonaChange = async (e: React.ChangeEvent<HTMLSelectElement>) => {
    const next = e.target.value as MockPersona;
    await switchPersona(next);
    if (next === 'admin' || next === 'admin_no_mfa') {
      navigate('/admin');
    } else if (next !== 'anonymous') {
      navigate('/member');
    }
  };

  const isAdmin = Boolean(session?.authenticated && session.role === 'admin');

  return (
    <div className="app-shell">
      <a href="#main-content" className="skip-link">
        {t('app.skipToMain')}
      </a>

      <header className="top-bar">
        <div className="top-bar-brand">
          <span>Freedom VPN</span>
          {session?.authenticated && session.role && (
            <span
              className={`badge ${isAdmin ? 'badge-info' : 'badge-success'}`}
              data-testid="session-role-badge"
            >
              {isAdmin ? t('app.role.admin') : t('app.role.member')}
            </span>
          )}
          {session?.authenticated && session.display_label && (
            <span
              className="mono"
              style={{ fontSize: '0.8rem', fontWeight: 400, color: '#cbd5e1' }}
              data-testid="session-display-label"
            >
              {session.display_label}
            </span>
          )}
        </div>

        <div className="top-bar-controls">
          {!import.meta.env.PROD &&
            (import.meta.env.VITE_API_MOCK === '1' ||
              import.meta.env.MODE === 'test') && (
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <label htmlFor="persona-select" style={{ fontSize: '0.8rem' }}>
                {t('app.mockSwitcher.label')}
              </label>
              <select
                id="persona-select"
                data-testid="persona-switcher"
                value={persona}
                onChange={(e) => void handlePersonaChange(e)}
                style={{ width: 'auto', padding: '0.25rem 0.5rem', fontSize: '0.8rem' }}
              >
                <option value="admin">{t('app.mockSwitcher.admin')}</option>
                <option value="admin_no_mfa">{t('app.mockSwitcher.adminNoMfa')}</option>
                <option value="anonymous">{t('app.mockSwitcher.anonymous')}</option>
                <option value="member_active">
                  {t('app.mockSwitcher.memberActive')}
                </option>
                <option value="member_throttled">
                  {t('app.mockSwitcher.memberThrottled')}
                </option>
                <option value="member_pending">
                  {t('app.mockSwitcher.memberPending')}
                </option>
                <option value="member_cancel_at_period_end">
                  {t('app.mockSwitcher.memberCancelEnd')}
                </option>
                <option value="member_grace">{t('app.mockSwitcher.memberGrace')}</option>
                <option value="member_expired">
                  {t('app.mockSwitcher.memberExpired')}
                </option>
                <option value="member_suspended">
                  {t('app.mockSwitcher.memberSuspended')}
                </option>
              </select>
              <button
                type="button"
                className="btn btn-secondary"
                data-testid="simulate-401-btn"
                onClick={simulateSessionExpired}
                style={{ padding: '0.25rem 0.5rem', fontSize: '0.75rem' }}
              >
                {t('app.mockSwitcher.trigger401')}
              </button>
            </div>
          )}

          <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
            <label htmlFor="locale-select" style={{ fontSize: '0.8rem' }}>
              {t('app.language.label')}
            </label>
            <select
              id="locale-select"
              data-testid="locale-switcher"
              value={locale}
              onChange={(e) => setLocale(e.target.value as Locale)}
              style={{ width: 'auto', padding: '0.25rem 0.5rem', fontSize: '0.8rem' }}
            >
              <option value="zh-TW">{t('app.language.zhTW')}</option>
              <option value="en">{t('app.language.en')}</option>
            </select>
          </div>

          {session?.authenticated && (
            <button
              type="button"
              className="btn btn-secondary"
              data-testid="logout-btn"
              onClick={() => void logout()}
              style={{ padding: '0.25rem 0.6rem', fontSize: '0.8rem' }}
            >
              {t('app.logout')}
            </button>
          )}
        </div>
      </header>

      {sessionExpired && (
        <div
          className="alert alert-danger"
          role="alert"
          data-testid="session-expired-banner"
          style={{ margin: '1rem 1.5rem 0' }}
        >
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              flexWrap: 'wrap',
              gap: '0.75rem',
            }}
          >
            <div>
              <strong>{t('auth.sessionExpired.title')}</strong>
              <div>{t('auth.sessionExpired.description')}</div>
            </div>
            <button
              type="button"
              className="btn btn-primary"
              data-testid="relogin-btn"
              onClick={() => void recoverSession()}
            >
              {t('auth.sessionExpired.reloginBtn')}
            </button>
          </div>
        </div>
      )}

      <div className="app-body">
        <nav className="sidebar" aria-label="Primary Navigation">
          {isAdmin && (
            <>
              <div className="nav-section-title">{t('nav.admin.heading')}</div>
              <ul className="nav-list">
                <li>
                  <NavLink to="/admin" end>
                    {t('nav.admin.overview')}
                  </NavLink>
                </li>
                <li>
                  <NavLink to="/admin/nodes" end>
                    {t('nav.admin.nodes')}
                  </NavLink>
                </li>
                <li>
                  <NavLink to="/admin/nodes/new">{t('nav.admin.addNode')}</NavLink>
                </li>
                <li>
                  <NavLink to="/admin/updates">{t('nav.admin.updates')}</NavLink>
                </li>
                <li>
                  <NavLink to="/admin/members">{t('nav.admin.members')}</NavLink>
                </li>
                <li>
                  <NavLink to="/admin/policies/fup">{t('nav.admin.fup')}</NavLink>
                </li>
                <li>
                  <NavLink to="/admin/policies/dns">{t('nav.admin.dns')}</NavLink>
                </li>
                <li>
                  <NavLink to="/admin/audit">{t('nav.admin.audit')}</NavLink>
                </li>
              </ul>
            </>
          )}

          <div className="nav-section-title">{t('nav.member.heading')}</div>
          <ul className="nav-list">
            <li>
              <NavLink to="/member" end>
                {t('nav.member.service')}
              </NavLink>
            </li>
            <li>
              <NavLink to="/member/getting-started">
                {t('nav.member.gettingStarted')}
              </NavLink>
            </li>
            <li>
              <NavLink to="/member/exits">{t('nav.member.exits')}</NavLink>
            </li>
            <li>
              <NavLink to="/member/devices">{t('nav.member.devices')}</NavLink>
            </li>
            <li>
              <NavLink to="/member/usage">{t('nav.member.usage')}</NavLink>
            </li>
            <li>
              <NavLink to="/member/billing">{t('nav.member.billing')}</NavLink>
            </li>
            <li>
              <NavLink to="/member/dns-protection">
                {t('nav.member.dnsProtection')}
              </NavLink>
            </li>
          </ul>
        </nav>

        <main id="main-content" className="main-content" tabIndex={-1}>
          <Outlet />
        </main>
      </div>

      <footer className="app-footer">{t('app.footer.securityNote')}</footer>
    </div>
  );
}
