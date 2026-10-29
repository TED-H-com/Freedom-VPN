import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import axe from 'axe-core';
import { afterAll, afterEach, beforeAll, expect } from 'vitest';
import { etagCache, idempotencyKeys } from '../api/client';
import { resetMockState } from '../mocks/handlers';
import { server } from '../mocks/server';

beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' });
});

afterEach(() => {
  cleanup();
  server.resetHandlers();
  resetMockState('admin');
  idempotencyKeys.clearAll();
  etagCache.clear();
  window.localStorage.clear();
  window.sessionStorage.clear();
});

afterAll(() => {
  server.close();
});

export async function assertAccessible(container: HTMLElement): Promise<void> {
  const results = await axe.run(container, {
    rules: {
      // jsdom does not implement layout/canvas for computed color contrast
      'color-contrast': { enabled: false },
    },
  });
  expect(results.violations).toEqual([]);
}
