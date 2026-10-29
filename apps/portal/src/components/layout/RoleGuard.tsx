import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import type { SessionRole } from '../../api';
import { useSession } from '../../context/SessionContext';
import { useI18n } from '../../i18n';

interface RoleGuardProps {
  requiredRole: SessionRole;
  children: React.ReactNode;
}

export function RoleGuard({
  requiredRole,
  children,
}: RoleGuardProps): React.ReactElement {
  const { session, isLoading, loginUrlForPath } = useSession();
  const { t } = useI18n();
  const location = useLocation();
  const currentRelativePath = `${location.pathname}${location.search}${location.hash}`;

  if (isLoading) {
    return (
      <div className="card" role="status" aria-live="polite">
        Loading session...
      </div>
    );
  }

  if (!session || !session.authenticated) {
    const loginHref = loginUrlForPath(currentRelativePath);
    return (
      <section
        className="card"
        aria-labelledby="anonymous-heading"
        data-testid="auth-anonymous-prompt"
      >
        <h1 id="anonymous-heading">{t('auth.anonymous.title')}</h1>
        <p style={{ marginTop: '0.5rem', marginBottom: '1rem' }}>
          {t('auth.anonymous.description')}
        </p>
        <a href={loginHref} className="btn btn-primary" data-testid="login-redirect-btn">
          {t('auth.anonymous.loginBtn')}
        </a>
      </section>
    );
  }

  if (requiredRole === 'admin' && session.role !== 'admin') {
    return (
      <section
        className="card"
        role="alert"
        aria-labelledby="forbidden-heading"
        data-testid="role-guard-forbidden"
      >
        <h1 id="forbidden-heading" style={{ color: 'var(--color-danger)' }}>
          {t('auth.forbidden.title')}
        </h1>
        <p style={{ marginTop: '0.5rem', marginBottom: '1rem' }}>
          {t('auth.forbidden.description')}
        </p>
        <Link to="/member" className="btn btn-primary">
          {t('auth.forbidden.backToMember')}
        </Link>
      </section>
    );
  }

  if (requiredRole === 'admin' && !session.mfa_verified) {
    const mfaHref = loginUrlForPath(currentRelativePath);
    return (
      <section
        className="card"
        role="alert"
        aria-labelledby="mfa-required-heading"
        data-testid="role-guard-mfa-required"
      >
        <h1 id="mfa-required-heading" style={{ color: 'var(--color-warning)' }}>
          {t('auth.mfaRequired.title')}
        </h1>
        <p style={{ marginTop: '0.5rem', marginBottom: '1rem' }}>
          {t('auth.mfaRequired.description')}
        </p>
        <a href={mfaHref} className="btn btn-primary" data-testid="mfa-login-btn">
          {t('auth.mfaRequired.verifyBtn')}
        </a>
      </section>
    );
  }

  return <>{children}</>;
}
