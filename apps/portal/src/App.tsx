import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { ErrorBoundary } from './components/common/ErrorBoundary';
import { AppLayout } from './components/layout/AppLayout';
import { RoleGuard } from './components/layout/RoleGuard';
import { useSession } from './context/SessionContext';
import { AdminAddNodePage } from './pages/admin/AdminAddNodePage';
import { AdminAuditPage } from './pages/admin/AdminAuditPage';
import { AdminDnsPolicyPage } from './pages/admin/AdminDnsPolicyPage';
import { AdminEnrollmentDetailPage } from './pages/admin/AdminEnrollmentDetailPage';
import { AdminFleetUpdatesPage } from './pages/admin/AdminFleetUpdatesPage';
import { AdminFupPolicyPage } from './pages/admin/AdminFupPolicyPage';
import { AdminMemberDetailPage } from './pages/admin/AdminMemberDetailPage';
import { AdminMembersListPage } from './pages/admin/AdminMembersListPage';
import { AdminNodeDetailPage } from './pages/admin/AdminNodeDetailPage';
import { AdminNodesListPage } from './pages/admin/AdminNodesListPage';
import { AdminOverviewPage } from './pages/admin/AdminOverviewPage';
import { MemberBillingPage } from './pages/member/MemberBillingPage';
import { MemberDevicesPage } from './pages/member/MemberDevicesPage';
import { MemberDnsProtectionPage } from './pages/member/MemberDnsProtectionPage';
import { MemberExitsPage } from './pages/member/MemberExitsPage';
import { MemberGettingStartedPage } from './pages/member/MemberGettingStartedPage';
import { MemberServicePage } from './pages/member/MemberServicePage';
import { MemberUsagePage } from './pages/member/MemberUsagePage';

function RootRedirect(): React.ReactElement {
  const { session } = useSession();
  if (session?.role === 'member') {
    return <Navigate to="/member" replace />;
  }
  return <Navigate to="/admin" replace />;
}

export function AppRoutes(): React.ReactElement {
  return (
    <ErrorBoundary>
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<RootRedirect />} />

          {/* Admin routes guarded by RoleGuard */}
          <Route
            path="admin"
            element={
              <RoleGuard requiredRole="admin">
                <AdminOverviewPage />
              </RoleGuard>
            }
          />
          <Route
            path="admin/nodes"
            element={
              <RoleGuard requiredRole="admin">
                <AdminNodesListPage />
              </RoleGuard>
            }
          />
          <Route
            path="admin/nodes/new"
            element={
              <RoleGuard requiredRole="admin">
                <AdminAddNodePage />
              </RoleGuard>
            }
          />
          <Route
            path="admin/nodes/:nodeId"
            element={
              <RoleGuard requiredRole="admin">
                <AdminNodeDetailPage />
              </RoleGuard>
            }
          />
          <Route
            path="admin/enrollments/:enrollmentId"
            element={
              <RoleGuard requiredRole="admin">
                <AdminEnrollmentDetailPage />
              </RoleGuard>
            }
          />
          <Route
            path="admin/updates"
            element={
              <RoleGuard requiredRole="admin">
                <AdminFleetUpdatesPage />
              </RoleGuard>
            }
          />
          <Route
            path="admin/members"
            element={
              <RoleGuard requiredRole="admin">
                <AdminMembersListPage />
              </RoleGuard>
            }
          />
          <Route
            path="admin/members/:memberId"
            element={
              <RoleGuard requiredRole="admin">
                <AdminMemberDetailPage />
              </RoleGuard>
            }
          />
          <Route
            path="admin/policies/fup"
            element={
              <RoleGuard requiredRole="admin">
                <AdminFupPolicyPage />
              </RoleGuard>
            }
          />
          <Route
            path="admin/policies/dns"
            element={
              <RoleGuard requiredRole="admin">
                <AdminDnsPolicyPage />
              </RoleGuard>
            }
          />
          <Route
            path="admin/audit"
            element={
              <RoleGuard requiredRole="admin">
                <AdminAuditPage />
              </RoleGuard>
            }
          />

          {/* Member routes */}
          <Route
            path="member"
            element={
              <RoleGuard requiredRole="member">
                <MemberServicePage />
              </RoleGuard>
            }
          />
          <Route
            path="member/getting-started"
            element={
              <RoleGuard requiredRole="member">
                <MemberGettingStartedPage />
              </RoleGuard>
            }
          />
          <Route
            path="member/exits"
            element={
              <RoleGuard requiredRole="member">
                <MemberExitsPage />
              </RoleGuard>
            }
          />
          <Route
            path="member/devices"
            element={
              <RoleGuard requiredRole="member">
                <MemberDevicesPage />
              </RoleGuard>
            }
          />
          <Route
            path="member/usage"
            element={
              <RoleGuard requiredRole="member">
                <MemberUsagePage />
              </RoleGuard>
            }
          />
          <Route
            path="member/billing"
            element={
              <RoleGuard requiredRole="member">
                <MemberBillingPage />
              </RoleGuard>
            }
          />
          <Route
            path="member/dns-protection"
            element={
              <RoleGuard requiredRole="member">
                <MemberDnsProtectionPage />
              </RoleGuard>
            }
          />
        </Route>
      </Routes>
    </ErrorBoundary>
  );
}
