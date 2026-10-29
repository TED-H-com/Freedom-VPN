import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AppRoutes } from './App';
import { SessionProvider } from './context/SessionContext';
import { I18nProvider } from './i18n';
import './styles.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
    },
  },
});

async function enableMocking(): Promise<void> {
  const mockEnv = import.meta.env.VITE_API_MOCK;
  const shouldMock = mockEnv === '1' || (import.meta.env.DEV && mockEnv !== '0');
  if (!shouldMock) {
    return;
  }
  const { worker } = await import('./mocks/browser');
  await worker.start({
    onUnhandledRequest: 'bypass',
  });
}

void enableMocking().then(() => {
  const rootEl = document.getElementById('root');
  if (!rootEl) return;

  ReactDOM.createRoot(rootEl).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <I18nProvider>
          <SessionProvider>
            <BrowserRouter>
              <AppRoutes />
            </BrowserRouter>
          </SessionProvider>
        </I18nProvider>
      </QueryClientProvider>
    </React.StrictMode>,
  );
});
