import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import {
  apiClient,
  ApiError,
  buildLoginUrl,
  CSRF_COOKIE_NAME,
  CSRF_HEADER_NAME,
  FreedomVpnApiClient,
  isTerminalOperationStatus,
  parseApiError,
  readCsrfToken,
  sanitizeReturnTo,
} from '../index';
import { PRIMARY_NODE_ID } from '../../mocks/data';
import { setMockPersona } from '../../mocks/handlers';
import { server } from '../../mocks/server';

describe('src/api (Typed Client, Auth Bootstrap, CSRF, Idempotency, ETag, Operations)', () => {
  it('TestCSRF_DoubleSubmitHeaderPresentOnUnsafeMethodsAndAbsentOnGet', async () => {
    const observedRequests: Array<{
      method: string;
      pathname: string;
      csrfHeader: string | null;
      credentials: RequestCredentials;
    }> = [];

    server.use(
      http.all('*', ({ request }) => {
        const url = new URL(request.url);
        observedRequests.push({
          method: request.method.toUpperCase(),
          pathname: url.pathname,
          csrfHeader: request.headers.get(CSRF_HEADER_NAME),
          credentials: request.credentials,
        });
        if (url.pathname === '/auth/session') {
          return HttpResponse.json(
            {
              authenticated: true,
              role: 'admin',
              display_label: 'ops-admin@example.com',
              mfa_verified: true,
            },
            { headers: { 'Cache-Control': 'no-store' } },
          );
        }
        if (url.pathname === '/auth/logout') {
          return new HttpResponse(null, { status: 204 });
        }
        return HttpResponse.json(
          { items: [], next_cursor: null },
          { headers: { ETag: '"10"' } },
        );
      }),
    );

    // 1. Set __Host-fvpn-csrf cookie in document.cookie
    document.cookie = `${CSRF_COOKIE_NAME}=csrf-secret-token-9988; Secure; Path=/`;
    expect(readCsrfToken()).toBe('csrf-secret-token-9988');

    // Safe GET requests must NEVER carry X-CSRF-Token
    await apiClient.getSession();
    await apiClient.listNodes();

    // Unsafe requests (POST, PATCH, PUT, DELETE, including POST /auth/logout) MUST carry X-CSRF-Token
    await apiClient.createEnrollment({ delivery: 'preinstalled' }, 'csrf-post-test');
    await apiClient.updateNode(
      PRIMARY_NODE_ID,
      { notes: 'csrf patch' },
      '"10"',
      'csrf-patch-test',
    );
    await apiClient.setNodePublication(
      PRIMARY_NODE_ID,
      { publication_intent: 'published', reason: 'Publish node' },
      '"10"',
      'csrf-put-test',
    );
    await apiClient.revokeMemberDevice(
      '2df6c7df-e7e8-4bd4-9f2c-a5836dd0d8ea',
      'csrf-delete-test',
    );
    await apiClient.logout();

    const getReqs = observedRequests.filter((r) => r.method === 'GET');
    const unsafeReqs = observedRequests.filter((r) => r.method !== 'GET');

    expect(getReqs.length).toBe(2);
    for (const req of getReqs) {
      expect(req.csrfHeader).toBeNull();
      expect(req.credentials).toBe('include');
    }

    expect(unsafeReqs.map((r) => r.method)).toEqual([
      'POST',
      'PATCH',
      'PUT',
      'DELETE',
      'POST',
    ]);
    for (const req of unsafeReqs) {
      expect(req.csrfHeader).toBe('csrf-secret-token-9988');
      expect(req.credentials).toBe('include');
    }

    // 2. When __Host-fvpn-csrf cookie is absent, send request anyway without header and let server decide
    document.cookie = `${CSRF_COOKIE_NAME}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; Secure; Path=/`;
    expect(readCsrfToken()).toBeNull();

    await apiClient.createEnrollment({ delivery: 'installer' }, 'csrf-absent-test');
    const lastReq = observedRequests[observedRequests.length - 1];
    expect(lastReq?.method).toBe('POST');
    expect(lastReq?.csrfHeader).toBeNull();
  });

  it('TestAuthBootstrap_SessionStatesAndRelativeLoginUrl', async () => {
    // 1. Anonymous state (200 with authenticated=false)
    setMockPersona('anonymous');
    const anonSession = await apiClient.getSession();
    expect(anonSession).toEqual({ authenticated: false });

    // 2. Member state
    setMockPersona('member_active');
    const memberSession = await apiClient.getSession();
    expect(memberSession.authenticated).toBe(true);
    expect(memberSession.role).toBe('member');
    expect(memberSession.mfa_verified).toBe(false);
    expect(memberSession.member_id).toBe('dc01a74b-9e6f-41ac-81cd-03291d0a88c0');

    // 3. Admin without MFA state
    setMockPersona('admin_no_mfa');
    const adminNoMfa = await apiClient.getSession();
    expect(adminNoMfa).toEqual({
      authenticated: true,
      role: 'admin',
      display_label: 'ops-admin@example.com',
      mfa_verified: false,
    });

    // 4. Admin with MFA state
    setMockPersona('admin');
    const adminWithMfa = await apiClient.getSession();
    expect(adminWithMfa).toEqual({
      authenticated: true,
      role: 'admin',
      display_label: 'ops-admin@example.com',
      mfa_verified: true,
    });

    // 5. Logout transitions session to anonymous
    await apiClient.logout();
    const afterLogout = await apiClient.getSession();
    expect(afterLogout).toEqual({ authenticated: false });

    // 6. Login URL always uses a relative return_to path and rejects absolute URLs
    expect(buildLoginUrl('/admin/nodes?tab=active')).toBe(
      '/auth/login?return_to=%2Fadmin%2Fnodes%3Ftab%3Dactive',
    );
    expect(sanitizeReturnTo('https://evil.example.com/phish')).toBe('/');
    expect(sanitizeReturnTo('//evil.example.com/phish')).toBe('/');
    expect(sanitizeReturnTo('javascript:alert(1)')).toBe('/');
    expect(buildLoginUrl('https://evil.example.com/phish')).toBe(
      '/auth/login?return_to=%2F',
    );
  });

  it('TestTypedClient_CompileTimeRejectsWrongPathAndWrongBody', () => {
    // Compile-time verification of operation-typed client (Deliverable 2 & 8):
    // The branches below are inside `if (false)` so they never execute at runtime,
    // but TypeScript's compiler strictly typechecks them and verifies `@ts-expect-error`.
    const checkTypesOnly = false;
    if (checkTypesOnly) {
      // @ts-expect-error Wrong path not in OpenAPI `paths` must fail typecheck
      void apiClient.callOperation({ method: 'get', path: '/admin/nonexistent-route' });

      void apiClient.callOperation({
        method: 'post',
        path: '/admin/enrollments',
        // @ts-expect-error Wrong request body shape must fail typecheck
        body: { invalid_property: 123 },
      });

      void apiClient.callOperation({
        method: 'put',
        path: '/admin/nodes/{node_id}/publication',
        pathParams: { node_id: PRIMARY_NODE_ID },
        // @ts-expect-error Invalid enum value for `publication_intent` must fail typecheck
        body: { publication_intent: 'invalid_intent' },
      });
    }
    expect(typeof apiClient.callOperation).toBe('function');
  });

  it('TestAPI02_IdempotencyKeyReusedAcrossRetryOfSameLogicalAction', async () => {
    const receivedKeys: string[] = [];

    server.use(
      http.put('/api/v1/admin/nodes/:nodeId/publication', ({ request }) => {
        const key = request.headers.get('Idempotency-Key');
        if (key) receivedKeys.push(key);
        return HttpResponse.json(
          {
            operation_id: '8089f821-cd9b-4c3f-a526-20ee33e68ad5',
            kind: 'node.publish',
            status: 'accepted',
            resource: { type: 'node', id: PRIMARY_NODE_ID },
            resource_version: '42',
            desired_generation: '128',
            observed_generation: '127',
            steps: [],
            error: null,
            request_id: 'req-idem-1',
            created_at: '2026-10-28T15:00:00Z',
            updated_at: '2026-10-28T15:00:00Z',
            completed_at: null,
          },
          { status: 202 },
        );
      }),
    );

    const actionId = `publish:${PRIMARY_NODE_ID}:attempt`;
    await apiClient.publishNode(PRIMARY_NODE_ID, '"42"', actionId);
    await apiClient.publishNode(PRIMARY_NODE_ID, '"42"', actionId);

    expect(receivedKeys).toHaveLength(2);
    expect(receivedKeys[0]).toBeTruthy();
    expect(receivedKeys[0]).toBe(receivedKeys[1]);

    await apiClient.publishNode(PRIMARY_NODE_ID, '"42"', 'publish:different-action');
    expect(receivedKeys[2]).not.toBe(receivedKeys[0]);
  });

  it('TestAPI03_EtagCapturedAndIfMatchSentAnd412ConflictNormalized', async () => {
    const detail = await apiClient.getNode(PRIMARY_NODE_ID);
    expect(detail.etag).toBe('"42"');

    const updated = await apiClient.updateNode(
      PRIMARY_NODE_ID,
      { notes: 'Updated note with matching ETag' },
      detail.etag ?? undefined,
    );
    expect(updated.etag).toBe('"43"');

    await expect(
      apiClient.updateNode(PRIMARY_NODE_ID, { notes: 'Stale write attempt' }, '"42"'),
    ).rejects.toMatchObject({
      status: 412,
      code: 'RESOURCE_VERSION_CONFLICT',
      details: {
        current_resource_version: '43',
      },
    });
  });

  it('TestAPI04_202AcceptedOperationLifecycleTransitionsToTerminal', async () => {
    const detail = await apiClient.getNode(PRIMARY_NODE_ID);
    const opInitial = await apiClient.publishNode(
      PRIMARY_NODE_ID,
      detail.etag ?? undefined,
    );
    expect(opInitial.status).toBe('running');
    expect(isTerminalOperationStatus(opInitial.status)).toBe(false);

    const opPolled = await apiClient.getOperation(opInitial.operation_id);
    expect(opPolled.status).toBe('succeeded');
    expect(isTerminalOperationStatus(opPolled.status)).toBe(true);
    expect(opPolled.observed_generation).toBe(opPolled.desired_generation);
  });

  it('TestSEC01_ErrorEnvelopeSanitizesSecretsSqlAndStackTracesAndParsesRetryAfter', () => {
    const rawErrorBody = {
      error: {
        code: 'RATE_LIMITED',
        message:
          'SELECT * FROM nodes WHERE token = "fvpn_enr_999" goroutine 1 [running]: BEGIN PRIVATE KEY',
        request_id: 'req-sec-99',
        retryable: true,
        details: {
          // TEST ONLY - NOT A SECRET
          private_key: '-----BEGIN PRIVATE KEY----- TEST ONLY - NOT A SECRET',
          safe_field: 'allowed-value',
          leaked_sql: 'INSERT INTO enrollments VALUES (1)',
        },
      },
    };

    const err = parseApiError(429, rawErrorBody, '15');
    expect(err).toBeInstanceOf(ApiError);
    expect(err.status).toBe(429);
    expect(err.code).toBe('RATE_LIMITED');
    expect(err.retryAfterSeconds).toBe(15);
    expect(err.requestId).toBe('req-sec-99');

    expect(err.serverMessage).toBe('[REDACTED]');
    expect(err.details.private_key).toBe('[REDACTED]');
    expect(err.details.leaked_sql).toBe('[REDACTED]');
    expect(err.details.safe_field).toBe('allowed-value');
    expect(err.messageForLocale('zh-TW')).not.toContain('SELECT');
    expect(err.messageForLocale('en')).not.toContain('PRIVATE KEY');

    // Unknown error code still renders generic fallback message (Deliverable 6)
    const unknownCodeErr = parseApiError(500, {
      error: {
        code: 'FUTURE_UNKNOWN_ERROR_CODE',
        message: 'Unexpected future code',
        request_id: 'req-unknown-1',
        retryable: false,
      },
    });
    expect(unknownCodeErr.code).toBe('FUTURE_UNKNOWN_ERROR_CODE');
    expect(unknownCodeErr.messageForLocale('en')).toContain(
      'The request could not be completed',
    );
  });

  it('TestAUTH01_401UnauthorizedTriggersSessionExpiredHandler', async () => {
    let unauthorizedTriggered = false;
    const customClient = new FreedomVpnApiClient({
      onUnauthorized: () => {
        unauthorizedTriggered = true;
      },
    });

    server.use(
      http.get('/api/v1/member/me', () =>
        HttpResponse.json(
          {
            error: {
              code: 'SESSION_EXPIRED',
              message: 'OIDC session expired',
              request_id: 'req-401',
              retryable: false,
            },
          },
          { status: 401 },
        ),
      ),
    );

    await expect(customClient.getMemberSelf()).rejects.toMatchObject({
      status: 401,
      code: 'SESSION_EXPIRED',
    });
    expect(unauthorizedTriggered).toBe(true);
  });
});
