import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from 'react';
import { onUnauthorizedError, portalApi, type SessionInfo } from '../api';
import {
  getMockPersona,
  setMockPersona,
  triggerMockSessionExpired,
} from '../mocks/handlers';
import type { MockPersonaKey } from '../mocks/data';

interface SessionContextValue {
  session: SessionInfo | null;
  loading: boolean;
  sessionExpired: boolean;
  persona: MockPersonaKey;
  switchPersona: (nextPersona: MockPersonaKey) => Promise<void>;
  simulateSessionExpired: () => void;
  recoverSession: () => Promise<void>;
}

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({
  children,
  initialPersona,
}: {
  children: React.ReactNode;
  initialPersona?: MockPersonaKey;
}): React.ReactElement {
  const [persona, setPersonaState] = useState<MockPersonaKey>(() => {
    if (initialPersona) {
      setMockPersona(initialPersona);
      return initialPersona;
    }
    if (typeof window !== 'undefined') {
      const urlPersona = new URLSearchParams(window.location.search).get(
        'persona',
      ) as MockPersonaKey | null;
      if (
        urlPersona &&
        [
          'admin',
          'member_active',
          'member_throttled',
          'member_pending',
          'member_suspended',
        ].includes(urlPersona)
      ) {
        setMockPersona(urlPersona);
        return urlPersona;
      }
    }
    return getMockPersona();
  });
  const [session, setSession] = useState<SessionInfo | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [sessionExpired, setSessionExpired] = useState<boolean>(false);

  const loadSession = useCallback(async () => {
    setLoading(true);
    try {
      const res = await portalApi.getSession();
      setSession(res.data);
      setSessionExpired(false);
    } catch {
      setSessionExpired(true);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadSession();
  }, [loadSession]);

  useEffect(() => {
    return onUnauthorizedError(() => {
      setSessionExpired(true);
    });
  }, []);

  const switchPersona = useCallback(
    async (nextPersona: MockPersonaKey) => {
      setMockPersona(nextPersona);
      setPersonaState(nextPersona);
      await loadSession();
    },
    [loadSession],
  );

  const simulateSessionExpired = useCallback(() => {
    triggerMockSessionExpired();
    setSessionExpired(true);
  }, []);

  const recoverSession = useCallback(async () => {
    setMockPersona(persona);
    setSessionExpired(false);
    await loadSession();
  }, [persona, loadSession]);

  return (
    <SessionContext.Provider
      value={{
        session,
        loading,
        sessionExpired,
        persona,
        switchPersona,
        simulateSessionExpired,
        recoverSession,
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
