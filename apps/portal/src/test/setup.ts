import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import axe from 'axe-core';
import { afterAll, afterEach, beforeAll, expect } from 'vitest';
import { etagCache, idempotencyKeys } from '../api/client';
import { resetMockState } from '../mocks/handlers';
import { server } from '../mocks/server';

// jsdom has no modal top layer; real focus/Escape behavior is covered by Playwright.
HTMLDialogElement.prototype.showModal = function () {
  this.setAttribute('open', '');
};

HTMLDialogElement.prototype.close = function () {
  this.removeAttribute('open');
};

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
  if (typeof document !== 'undefined' && document.cookie) {
    for (const part of document.cookie.split(';')) {
      const eq = part.indexOf('=');
      const name = (eq === -1 ? part : part.slice(0, eq)).trim();
      if (name) {
        document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
      }
    }
  }
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
