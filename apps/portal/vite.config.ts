import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig(({ mode }) => {
  const apiMockEnv = process.env.VITE_API_MOCK;
  const mockEnabled = apiMockEnv ? apiMockEnv === '1' : mode !== 'production';

  return {
    plugins: [react()],
    define: {
      'import.meta.env.VITE_API_MOCK': JSON.stringify(mockEnabled ? '1' : '0'),
    },
    server: {
      port: 5173,
      proxy: mockEnabled
        ? undefined
        : {
            '/api/v1': {
              target: process.env.VITE_API_PROXY_TARGET ?? 'http://127.0.0.1:8080',
              changeOrigin: true,
            },
          },
    },
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/test/setup.ts'],
      coverage: {
        provider: 'v8',
        reporter: ['text', 'html'],
        include: ['src/api/**/*.ts', 'src/lib/**/*.ts'],
        exclude: ['src/**/*.test.ts', 'src/**/*.test.tsx', 'src/api/types.ts'],
        thresholds: {
          lines: 80,
        },
      },
    },
  };
});
