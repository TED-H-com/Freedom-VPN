import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  apiClient,
  buildLoginUrl,
  redirectToLogin,
  type AuthSession,
  type SessionRole,
} from '../api';

export type MockPersona =
  | 'anonymous'
  | 'admin_no_mfa'
  | 'admin'
  | 'member_active'
  | 'member_throttled'
  | 'member_pending_payment'
  | 'member_cancel_at_period_end'
  | 'member_grace'
  | 'member_expired'
  | 'member_suspended';

interface SessionContextValue {
  session: AuthSession | null;
  isLoading: boolean;
  sessionExpired: boolean;
  persona: MockPersona;
  switchPersona: (next: MockPersona) => Promise<void>;
  simulateSessionExpired: () => void;
  recoverSession: () => Promise<void>;
  loginUrlForPath: (returnTo?: string, role?: SessionRole) => string;
  startLogin: (returnTo?: string, role?: SessionRole) => string;
  logout: () => Promise<void>;
}

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({
  children,
}: {
  children: React.ReactNode;
}): React.ReactElement {
  const queryClient = useQueryClient();
  const [persona, setPersonaState] = useState<MockPersona>('admin');
  const [sessionExpired, setSessionExpired] = useState(false);

  useEffect(() => {
    apiClient.setUnauthorizedHandler(() => {
      setSessionExpired(true);
    });
    return () => {
      apiClient.setUnauthorizedHandler(undefined);
    };
  }, []);

  const sessionQuery = useQuery({
    queryKey: ['session', persona],
    queryFn: () => apiClient.getSession(),
    retry: false,
  });

  const switchPersona = useCallback(
    async (next: MockPersona) => {
      if (!import.meta.env.PROD) {
        if (import.meta.env.VITE_API_MOCK === '1' || import.meta.env.MODE === 'test') {
          const mod = await import('../mocks/handlers');
          mod.setMockPersona(next);
        }
      }
      setPersonaState(next);
      setSessionExpired(false);
      await queryClient.invalidateQueries();
    },
    [queryClient],
  );

  const simulateSessionExpired = useCallback(() => {
    if (!import.meta.env.PROD) {
      if (import.meta.env.VITE_API_MOCK === '1' || import.meta.env.MODE === 'test') {
        void import('../mocks/handlers').then((mod) => {
          mod.triggerMockSessionExpired();
        });
      }
    }
    setSessionExpired(true);
  }, []);

  const recoverSession = useCallback(async () => {
    if (!import.meta.env.PROD) {
      if (import.meta.env.VITE_API_MOCK === '1' || import.meta.env.MODE === 'test') {
        const mod = await import('../mocks/handlers');
        mod.setMockPersona(persona);
        setSessionExpired(false);
        await queryClient.invalidateQueries();
        return;
      }
    }
    redirectToLogin();
  }, [persona, queryClient]);

  const loginUrlForPath = useCallback((returnTo?: string, role?: SessionRole) => {
    return buildLoginUrl(returnTo, undefined, role);
  }, []);

  const startLogin = useCallback((returnTo?: string, role?: SessionRole) => {
    return redirectToLogin(returnTo, undefined, undefined, role);
  }, []);

  const logout = useCallback(async () => {
    await apiClient.logout(sessionQuery.data?.role);
    setPersonaState('anonymous');
    setSessionExpired(false);
    await queryClient.invalidateQueries();
  }, [queryClient, sessionQuery.data?.role]);

  return (
    <SessionContext.Provider
      value={{
        session: sessionQuery.data ?? null,
        isLoading: sessionQuery.isLoading,
        sessionExpired,
        persona,
        switchPersona,
        simulateSessionExpired,
        recoverSession,
        loginUrlForPath,
        startLogin,
        logout,
      }}
    >
      {children}
    </SessionContext.Provider>
  );
}

export function useSession(): SessionContextValue {
  const ctx = useContext(SessionContext);
  if (!ctx) {
    throw new Error('useSession must be used within a SessionProvider');
  }
  return ctx;
}
