import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '../..');

export default defineConfig(({ mode }) => {
  const apiMockEnv = process.env.VITE_API_MOCK;
  const mockEnabled = apiMockEnv ? apiMockEnv === '1' : mode !== 'production';
  const proxyTarget = process.env.VITE_API_PROXY_TARGET ?? 'http://127.0.0.1:8080';

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
            },
            '/auth': {
              target: proxyTarget,
              changeOrigin: true,
            },
          },
    },
    test: {
      globals: true,
      environment: 'jsdom',
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
