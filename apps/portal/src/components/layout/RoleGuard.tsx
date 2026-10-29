import React from 'react';
import { Link } from 'react-router-dom';
import type { UserRole } from '../../api/types';
import { useSession } from '../../context/SessionContext';
import { useI18n } from '../../i18n';

/**
 * Route guard enforcing role separation (`/admin/*` vs `/member/*`).
 *
 * Security note (§9.1, §16.1, Prompt Deliverable 4):
 * A member can never reach admin routes. This client-side check is defense in depth only;
 * backend API endpoints enforce role & MFA checks on every request.
 */
export function RoleGuard({
  requiredRole,
  children,
}: {
  requiredRole: UserRole;
  children: React.ReactNode;
}): React.ReactElement {
  const { session, loading } = useSession();
  const { t } = useI18n();

  if (loading && !session) {
    return <div className="card">Loading session...</div>;
  }

  if (requiredRole === 'admin' && session?.role !== 'admin') {
    return (
      <section
        className="card alert-danger"
        role="alert"
        data-testid="role-guard-forbidden"
      >
        <h1>{t('auth.forbidden.title')}</h1>
        <p>{t('auth.forbidden.description')}</p>
        <div className="btn-row" style={{ marginTop: '14px' }}>
          <Link to="/member" className="btn btn-primary">
            {t('auth.forbidden.backToMember')}
          </Link>
        </div>
      </section>
    );
  }

  return <>{children}</>;
}
