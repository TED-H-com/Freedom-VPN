import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '../..');

export default defineConfig(({ mode }) => {
  const apiMockEnv = process.env.VITE_API_MOCK;
  const mockEnabled =
    mode === 'production' ? false : apiMockEnv ? apiMockEnv === '1' : mode === 'test';
  const proxyTarget =
    process.env.VITE_API_PROXY_TARGET ??
    process.env.VITE_DEV_API_ORIGIN ??
    'https://127.0.0.1:8443';

  return {
    plugins: [react()],
    resolve: {
      alias: {
        '@contracts': path.resolve(repoRoot, 'contracts/ts'),
      },
    },
    define: {
      'import.meta.env.VITE_API_MOCK': JSON.stringify(mockEnabled ? '1' : '0'),
    },
    server: {
      port: 5173,
      fs: {
        allow: [repoRoot],
      },
      proxy: mockEnabled
        ? undefined
        : {
            '/api/v1': {
              target: proxyTarget,
              changeOrigin: true,
              secure: false,
              xfwd: false,
            },
            '/auth': {
              target: proxyTarget,
              changeOrigin: true,
              secure: false,
              xfwd: false,
              rewrite: (reqPath) => {
                if (reqPath.startsWith('/auth/login') && !/[?&]role=/.test(reqPath)) {
                  const decoded = decodeURIComponent(reqPath);
                  if (decoded.includes('return_to=/admin')) {
                    return `${reqPath}${reqPath.includes('?') ? '&' : '?'}role=admin`;
                  }
                }
                return reqPath;
              },
            },
          },
    },
    test: {
      globals: true,
      environment: 'jsdom',
      include: ['src/**/*.{test,spec}.{ts,tsx}'],
      exclude: ['e2e/**', 'node_modules/**'],
      environmentOptions: {
        jsdom: {
          url: 'https://portal.example.com',
        },
      },
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
