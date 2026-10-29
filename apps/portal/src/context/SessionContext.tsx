import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, buildLoginUrl, redirectToLogin, type AuthSession } from '../api';
import {
  getMockPersona,
  setMockPersona,
  triggerMockSessionExpired,
  type MockPersona,
} from '../mocks/handlers';

interface SessionContextValue {
  session: AuthSession | null;
  isLoading: boolean;
  sessionExpired: boolean;
  persona: MockPersona;
  switchPersona: (next: MockPersona) => Promise<void>;
  simulateSessionExpired: () => void;
  recoverSession: () => Promise<void>;
  loginUrlForPath: (returnTo?: string) => string;
  startLogin: (returnTo?: string) => string;
  logout: () => Promise<void>;
}

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({
  children,
}: {
  children: React.ReactNode;
}): React.ReactElement {
  const queryClient = useQueryClient();
  const [persona, setPersonaState] = useState<MockPersona>(() => getMockPersona());
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
      setMockPersona(next);
      setPersonaState(next);
      setSessionExpired(false);
      await queryClient.invalidateQueries();
    },
    [queryClient],
  );

  const simulateSessionExpired = useCallback(() => {
    triggerMockSessionExpired();
    setSessionExpired(true);
  }, []);

  const recoverSession = useCallback(async () => {
    setMockPersona(persona);
    setSessionExpired(false);
    await queryClient.invalidateQueries();
  }, [persona, queryClient]);

  const loginUrlForPath = useCallback((returnTo?: string) => {
    return buildLoginUrl(returnTo);
  }, []);

  const startLogin = useCallback((returnTo?: string) => {
    return redirectToLogin(returnTo);
  }, []);

  const logout = useCallback(async () => {
    await apiClient.logout();
    setPersonaState(getMockPersona());
    setSessionExpired(false);
    await queryClient.invalidateQueries();
  }, [queryClient]);

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
