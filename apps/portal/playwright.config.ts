import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { defineConfig } from '@playwright/test';

const baseURL = process.env.PORTAL_E2E_BASE_URL || 'http://127.0.0.1:4173';
const parsedUrl = new URL(baseURL);
const devPort = parsedUrl.port || '4173';

function findCachedChromium(): string | undefined {
  if (process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH) {
    return process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH;
  }
  const cacheRoot = path.join(os.homedir(), '.cache', 'ms-playwright');
  if (!fs.existsSync(cacheRoot)) return undefined;
  const entries = fs
    .readdirSync(cacheRoot)
    .filter((d) => d.startsWith('chromium-'))
    .sort()
    .reverse();
  for (const dir of entries) {
    const candidate = path.join(cacheRoot, dir, 'chrome-linux64', 'chrome');
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }
  return undefined;
}

const executablePath = findCachedChromium();

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.spec.ts',
  timeout: 45_000,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: 'list',
  use: {
    baseURL,
    trace: 'retain-on-failure',
    headless: true,
    launchOptions: executablePath ? { executablePath } : undefined,
  },
  webServer: process.env.PORTAL_E2E_SKIP_WEBSERVER
    ? undefined
    : {
        command: `PORTAL_DEV_PORT=${devPort} node e2e/dev-stack.mjs`,
        url: `${baseURL}/auth/session`,
        timeout: 60_000,
        // Let dev-stack drop its isolated database and remove generated test keys.
        gracefulShutdown: { signal: 'SIGTERM', timeout: 10_000 },
        reuseExistingServer: !process.env.CI,
      },
});
