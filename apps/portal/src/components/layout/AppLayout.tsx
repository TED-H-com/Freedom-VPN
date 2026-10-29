import React from 'react';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useSession } from '../../context/SessionContext';
import { useI18n, type Locale } from '../../i18n';
import type { MockPersonaKey } from '../../mocks/data';

export function AppLayout(): React.ReactElement {
  const {
    session,
    sessionExpired,
    persona,
    switchPersona,
    simulateSessionExpired,
    recoverSession,
  } = useSession();
  const { locale, setLocale, t } = useI18n();
  const navigate = useNavigate();

  const handlePersonaChange = async (e: React.ChangeEvent<HTMLSelectElement>) => {
    const val = e.target.value as MockPersonaKey;
    await switchPersona(val);
    if (val === 'admin') {
      void navigate('/admin');
    } else {
      void navigate('/member');
    }
  };

  return (
    <div className="app-shell">
      <a href="#main-content" className="skip-link">
        {t('app.skipToMain')}
      </a>

      <header className="app-header">
        <div className="app-header-brand">
          <span>Freedom VPN</span>
          {session && (
            <span className="badge badge-info" data-testid="active-role-badge">
              {session.role === 'admin' ? t('app.role.admin') : t('app.role.member')} —{' '}
              {session.display_name}
            </span>
          )}
        </div>

        <div className="app-header-controls">
          <label htmlFor="mock-persona-select">
            <span>{t('app.mockSwitcher.label')}</span>
            <select
              id="mock-persona-select"
              value={persona}
              onChange={(e) => void handlePersonaChange(e)}
              data-testid="mock-persona-select"
            >
              <option value="admin">{t('app.mockSwitcher.admin')}</option>
              <option value="member_active">{t('app.mockSwitcher.memberActive')}</option>
              <option value="member_throttled">
                {t('app.mockSwitcher.memberThrottled')}
              </option>
              <option value="member_pending">
                {t('app.mockSwitcher.memberPending')}
              </option>
              <option value="member_cancel_end">
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
          </label>

          <button
            type="button"
            onClick={simulateSessionExpired}
            data-testid="simulate-401-btn"
          >
            {t('app.mockSwitcher.trigger401')}
          </button>

          <label htmlFor="locale-select">
            <span>{t('app.language.label')}</span>
            <select
              id="locale-select"
              value={locale}
              onChange={(e) => setLocale(e.target.value as Locale)}
              data-testid="locale-select"
            >
              <option value="zh-TW">{t('app.language.zhTW')}</option>
              <option value="en">{t('app.language.en')}</option>
            </select>
          </label>
        </div>
      </header>

      <div className="app-body">
        <nav className="app-sidebar" aria-label="Primary Navigation">
          {session?.role === 'admin' && (
            <>
              <div className="nav-section-title">{t('nav.admin.heading')}</div>
              <ul className="nav-list">
                <li>
                  <NavLink
                    to="/admin"
                    end
                    className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
                  >
                    {t('nav.admin.overview')}
                  </NavLink>
                </li>
                <li>
                  <NavLink
                    to="/admin/nodes"
                    end
                    className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
                  >
                    {t('nav.admin.nodes')}
                  </NavLink>
                </li>
                <li>
                  <NavLink
                    to="/admin/nodes/new"
                    className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
                  >
                    {t('nav.admin.addNode')}
                  </NavLink>
                </li>
                <li>
                  <NavLink
                    to="/admin/updates"
                    className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
                  >
                    {t('nav.admin.updates')}
                  </NavLink>
                </li>
                <li>
                  <NavLink
                    to="/admin/members"
                    className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
                  >
                    {t('nav.admin.members')}
                  </NavLink>
                </li>
                <li>
                  <NavLink
                    to="/admin/policies/fup"
                    className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
                  >
                    {t('nav.admin.fup')}
                  </NavLink>
                </li>
                <li>
                  <NavLink
                    to="/admin/policies/dns"
                    className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
                  >
                    {t('nav.admin.dns')}
                  </NavLink>
                </li>
                <li>
                  <NavLink
                    to="/admin/audit"
                    className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
                  >
                    {t('nav.admin.audit')}
                  </NavLink>
                </li>
              </ul>
            </>
          )}

          <div className="nav-section-title">{t('nav.member.heading')}</div>
          <ul className="nav-list">
            <li>
              <NavLink
                to="/member"
                end
                className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
              >
                {t('nav.member.service')}
              </NavLink>
            </li>
            <li>
              <NavLink
                to="/member/getting-started"
                className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
              >
                {t('nav.member.gettingStarted')}
              </NavLink>
            </li>
            <li>
              <NavLink
                to="/member/exits"
                className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
              >
                {t('nav.member.exits')}
              </NavLink>
            </li>
            <li>
              <NavLink
                to="/member/devices"
                className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
              >
                {t('nav.member.devices')}
              </NavLink>
            </li>
            <li>
              <NavLink
                to="/member/usage"
                className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
              >
                {t('nav.member.usage')}
              </NavLink>
            </li>
            <li>
              <NavLink
                to="/member/billing"
                className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
              >
                {t('nav.member.billing')}
              </NavLink>
            </li>
            <li>
              <NavLink
                to="/member/dns-protection"
                className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
              >
                {t('nav.member.dnsProtection')}
              </NavLink>
            </li>
          </ul>
        </nav>

        <main id="main-content" className="app-main" tabIndex={-1}>
          {sessionExpired ? (
            <section
              className="card alert-danger"
              role="alert"
              data-testid="session-expired-banner"
            >
              <h1>{t('auth.sessionExpired.title')}</h1>
              <p>{t('auth.sessionExpired.description')}</p>
              <button
                type="button"
                className="btn btn-primary"
                onClick={() => void recoverSession()}
                data-testid="session-relogin-btn"
              >
                {t('auth.sessionExpired.reloginBtn')}
              </button>
            </section>
          ) : (
            <Outlet />
          )}
        </main>
      </div>

      <footer className="app-footer">{t('app.footer.securityNote')}</footer>
    </div>
  );
}
