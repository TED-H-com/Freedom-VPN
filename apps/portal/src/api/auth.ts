/**
 * Browser session bootstrap routes: `GET /auth/session`, `GET /auth/login?return_to=<relative path>`,
 * `/auth/callback`, and `POST /auth/logout`.
 *
 * IMPORTANT: These browser session routes live outside `/api/v1` and are not in the
 * generated OpenAPI schema (`contracts/ts/schema.d.ts`). They are defined in this single
 * small hand-written module per the contract boundary (§14.1, §16.1).
 */

import { parseApiError } from './errors';

export type SessionRole = 'admin' | 'member';

/**
 * Response shape of `GET /auth/session` (returned with `Cache-Control: no-store`).
 * Returns HTTP 200 with `authenticated: false` when there is no active session.
 */
export interface AuthSession {
  authenticated: boolean;
  role?: SessionRole;
  display_label?: string;
  mfa_verified?: boolean;
  member_id?: string;
  admin_id?: string;
}

export const DEFAULT_AUTH_BASE_URL = '/auth';
export const CSRF_COOKIE_NAME = '__Host-fvpn-csrf';
export const CSRF_HEADER_NAME = 'X-CSRF-Token';

let sessionCsrfToken: string | null = null;

export function setSessionCsrfToken(token: string | null): void {
  sessionCsrfToken = token && token.trim().length > 0 ? token.trim() : null;
}

export function inferRoleFromPath(pathname?: string): SessionRole | undefined {
  const path =
    pathname ?? (typeof window !== 'undefined' ? window.location.pathname : '');
  if (path === '/admin' || path.startsWith('/admin/')) {
    return 'admin';
  }
  if (path === '/member' || path.startsWith('/member/')) {
    return 'member';
  }
  return undefined;
}

export function getAuthBaseUrl(override?: string): string {
  const raw = override ?? import.meta.env.VITE_AUTH_BASE_URL ?? DEFAULT_AUTH_BASE_URL;
  const trimmed = raw.trim().replace(/\/+$/, '');
  return trimmed || DEFAULT_AUTH_BASE_URL;
}

/**
 * Reads the readable (non-HttpOnly) `__Host-fvpn-csrf` cookie from `document.cookie`
 * or the `X-CSRF-Token` response header captured from `GET /auth/session` (§16.1).
 */
export function readCsrfToken(cookieSource?: string): string | null {
  const effectiveCookies =
    cookieSource !== undefined
      ? cookieSource
      : typeof document !== 'undefined'
        ? document.cookie
        : undefined;

  if (effectiveCookies) {
    const cookies = effectiveCookies.split(';');
    for (const entry of cookies) {
      const trimmed = entry.trim();
      if (!trimmed) continue;
      const eqIndex = trimmed.indexOf('=');
      if (eqIndex === -1) continue;
      const name = trimmed.slice(0, eqIndex).trim();
      if (name === CSRF_COOKIE_NAME) {
        const value = trimmed.slice(eqIndex + 1).trim();
        if (!value) break;
        try {
          return decodeURIComponent(value);
        } catch {
          return value;
        }
      }
    }
  }

  if (cookieSource === undefined && sessionCsrfToken) {
    return sessionCsrfToken;
  }
  return null;
}

/**
 * Ensures `return_to` is strictly a relative path starting with a single `/`
 * and never an absolute URL, protocol-relative URL (`//`), or dangerous scheme.
 */
export function sanitizeReturnTo(returnTo?: string | null): string {
  if (!returnTo) {
    return '/';
  }
  const trimmed = returnTo.trim();
  if (!trimmed.startsWith('/') || trimmed.startsWith('//') || trimmed.startsWith('/\\')) {
    return '/';
  }
  const pathPart = trimmed.split(/[?#]/, 1)[0] ?? '';
  if (pathPart.includes('://') || pathPart.includes('\\')) {
    return '/';
  }
  return trimmed;
}

/**
 * Builds the login URL `/auth/login?return_to=<relative path>`.
 * Never accepts or emits an absolute `return_to` URL.
 */
export function buildLoginUrl(
  returnTo?: string,
  authBaseUrl?: string,
  role?: SessionRole,
): string {
  const base = getAuthBaseUrl(authBaseUrl);
  const currentRelative =
    returnTo ??
    (typeof window !== 'undefined'
      ? `${window.location.pathname}${window.location.search}${window.location.hash}`
      : '/');
  const safeReturnTo = sanitizeReturnTo(currentRelative);
  const roleParam = role ? `&role=${encodeURIComponent(role)}` : '';
  return `${base}/login?return_to=${encodeURIComponent(safeReturnTo)}${roleParam}`;
}

/**
 * Performs a full-page browser redirect to `/auth/login?return_to=<relative path>`.
 */
export function redirectToLogin(
  returnTo?: string,
  navigateFn?: (url: string) => void,
  authBaseUrl?: string,
  role?: SessionRole,
): string {
  const url = buildLoginUrl(returnTo, authBaseUrl, role);
  if (navigateFn) {
    navigateFn(url);
  } else if (
    typeof window !== 'undefined' &&
    typeof window.location?.assign === 'function'
  ) {
    window.location.assign(url);
  }
  return url;
}

/**
 * Bootstraps browser session state via `GET /auth/session`.
 * Unsafe CSRF header is never sent on GET.
 */
export async function getAuthSession(
  fetchImpl: typeof fetch = fetch,
  authBaseUrl?: string,
  role?: SessionRole,
): Promise<AuthSession> {
  const base = getAuthBaseUrl(authBaseUrl);
  const effectiveRole = role ?? inferRoleFromPath();
  const query = effectiveRole ? `?role=${encodeURIComponent(effectiveRole)}` : '';
  const response = await fetchImpl(`${base}/session${query}`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  });

  const text = await response.text();
  let parsed: unknown = null;
  if (text.trim().length > 0) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = null;
    }
  }

  if (!response.ok) {
    setSessionCsrfToken(null);
    throw parseApiError(response.status, parsed);
  }

  const headerCsrf = response.headers.get(CSRF_HEADER_NAME);

  if (parsed && typeof parsed === 'object' && 'authenticated' in parsed) {
    const obj = parsed as Record<string, unknown>;
    if (obj.authenticated === false) {
      setSessionCsrfToken(null);
      return { authenticated: false };
    }
    if (headerCsrf) {
      setSessionCsrfToken(headerCsrf);
    }
    const resolvedRole =
      obj.role === 'admin' || obj.role === 'member' ? obj.role : undefined;
    const memberId = typeof obj.member_id === 'string' ? obj.member_id : undefined;
    const adminId = typeof obj.admin_id === 'string' ? obj.admin_id : undefined;
    const displayLabel =
      typeof obj.display_label === 'string'
        ? obj.display_label
        : (adminId ?? memberId ?? resolvedRole);
    return {
      authenticated: Boolean(obj.authenticated),
      role: resolvedRole,
      display_label: displayLabel,
      mfa_verified: typeof obj.mfa_verified === 'boolean' ? obj.mfa_verified : false,
      member_id: memberId,
      admin_id: adminId,
    };
  }

  setSessionCsrfToken(null);
  return { authenticated: false };
}

/**
 * Logs out the current browser session via `POST /auth/logout`.
 * Includes `X-CSRF-Token` from `__Host-fvpn-csrf` or `X-CSRF-Token` session header.
 */
export async function logoutAuthSession(
  fetchImpl: typeof fetch = fetch,
  authBaseUrl?: string,
  role?: SessionRole,
): Promise<void> {
  const base = getAuthBaseUrl(authBaseUrl);
  const effectiveRole = role ?? inferRoleFromPath();
  const query = effectiveRole ? `?role=${encodeURIComponent(effectiveRole)}` : '';
  const headers = new Headers({
    Accept: 'application/json',
  });
  const csrfToken = readCsrfToken();
  if (csrfToken) {
    headers.set(CSRF_HEADER_NAME, csrfToken);
  }

  const response = await fetchImpl(`${base}/logout${query}`, {
    method: 'POST',
    credentials: 'include',
    headers,
  });

  if (!response.ok) {
    const text = await response.text();
    let parsed: unknown = null;
    if (text.trim().length > 0) {
      try {
        parsed = JSON.parse(text);
      } catch {
        parsed = null;
      }
    }
    throw parseApiError(response.status, parsed);
  }
  setSessionCsrfToken(null);
}
