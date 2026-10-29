import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import {
  ApiError,
  apiRequest,
  etagCache,
  idempotencyKeys,
  mapErrorToUserMessage,
  onConflictError,
  onForbiddenError,
  onUnauthorizedError,
  portalApi,
  useOperationPoller,
} from '../index';
import { server } from '../../mocks/server';

describe('src/api (§14.1, §14.5, §14.6, §14.8 API Client & Error Envelope)', () => {
  it('TestAPI01_ParsesErrorEnvelopeAndMapsToUserFacingText', async () => {
    server.use(
      http.get('/api/v1/admin/nodes/bad-node', () => {
        return HttpResponse.json(
          {
            error: {
              code: 'NODE_NOT_SERVICEABLE',
              message: '節點隔離檢查未通過',
              request_id: 'req-err-999',
              retryable: false,
              details: { operation_id: 'op-err-111' },
            },
          },
          {
            status: 422,
            headers: {
              'X-Request-ID': 'req-err-999',
              'Retry-After': '5',
            },
          },
        );
      }),
    );

    let caught: unknown = null;
    try {
      await portalApi.getAdminNode('bad-node');
    } catch (err) {
      caught = err;
    }

    expect(caught).toBeInstanceOf(ApiError);
    const apiErr = caught as ApiError;
    expect(apiErr.status).toBe(422);
    expect(apiErr.code).toBe('NODE_NOT_SERVICEABLE');
    expect(apiErr.request_id).toBe('req-err-999');
    expect(apiErr.retryable).toBe(false);
    expect(apiErr.retryAfterSeconds).toBe(5);
    expect(apiErr.operationId).toBe('op-err-111');

    const userMsgZh = mapErrorToUserMessage(apiErr, 'zh-TW');
    expect(userMsgZh.title).toContain('節點未通過服務基線檢查');
    expect(userMsgZh.requestId).toBe('req-err-999');
    expect(userMsgZh.operationId).toBe('op-err-111');

    const userMsgEn = mapErrorToUserMessage(apiErr, 'en');
    expect(userMsgEn.title).toContain('Node Not Serviceable');
  });

  it('TestAPI01_IdempotencyKeyReusedOnRetryAndFreshOnNewAction', async () => {
    const capturedKeys: string[] = [];
    server.use(
      http.put('/api/v1/admin/nodes/:nodeId/publication', ({ request }) => {
        const key = request.headers.get('Idempotency-Key') ?? '';
        capturedKeys.push(key);
        if (capturedKeys.length === 1) {
          return HttpResponse.json(
            {
              error: {
                code: 'DEPENDENCY_UNAVAILABLE',
                message: 'Temporary error',
                request_id: 'req-503-1',
                retryable: true,
              },
            },
            { status: 503, headers: { 'Retry-After': '2' } },
          );
        }
        return HttpResponse.json(
          {
            operation_id: '8089f821-cd9b-4c3f-a526-20ee33e68ad5',
            kind: 'node.publish',
            status: 'running',
            steps: [{ name: 'agent_policy', status: 'running' }],
            created_at: '2026-10-29T12:00:00Z',
          },
          { status: 202, headers: { 'X-Request-ID': 'req-202-2' } },
        );
      }),
    );

    const actionId = 'user-click-publish-tw-tpe-01';

    // First attempt fails with retryable 503
    await expect(
      portalApi.putNodePublication(
        '93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb',
        { publication_intent: 'published' },
        actionId,
      ),
    ).rejects.toThrow(ApiError);

    // Retry of the SAME user action reuses the exact same Idempotency-Key
    const retryRes = await portalApi.putNodePublication(
      '93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb',
      { publication_intent: 'published' },
      actionId,
    );
    expect(retryRes.status).toBe(202);
    expect(capturedKeys).toHaveLength(2);
    expect(capturedKeys[0]).toBeTruthy();
    expect(capturedKeys[0]).toBe(capturedKeys[1]);

    // A brand-new user action gets a different Idempotency-Key
    idempotencyKeys.release(actionId);
    await portalApi.putNodePublication(
      '93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb',
      { publication_intent: 'published' },
      'user-click-publish-second-action',
    );
    expect(capturedKeys[2]).not.toBe(capturedKeys[0]);
  });

  it('TestAPI01_428MissingPreconditionTreatedAsBugAnd412TriggersConflict', async () => {
    etagCache.clear();

    // Missing If-Match throws 428 PRECONDITION_REQUIRED bug error immediately
    let bugErr: ApiError | null = null;
    try {
      await apiRequest('/admin/nodes/93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb', {
        method: 'PATCH',
        body: { name: 'new-name' },
        requireIfMatch: true,
      });
    } catch (e) {
      bugErr = e as ApiError;
    }
    expect(bugErr).not.toBeNull();
    expect(bugErr?.status).toBe(428);
    expect(bugErr?.isPreconditionBug).toBe(true);

    // Stale ETag triggers 412 conflict listener
    let conflictCaught: ApiError | null = null;
    const unsubConflict = onConflictError((err) => {
      conflictCaught = err;
    });

    await expect(
      portalApi.patchAdminNode(
        '93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb',
        { name: 'stale-update' },
        { ifMatch: '"stale-etag-v1"' },
      ),
    ).rejects.toThrow(ApiError);

    unsubConflict();
    expect(conflictCaught).not.toBeNull();
    expect((conflictCaught as unknown as ApiError).isConflict).toBe(true);
    expect((conflictCaught as unknown as ApiError).currentResourceVersion).toBe('42');
  });

  it('TestAPI01_401And403NotifySessionListeners', async () => {
    let saw401 = false;
    let saw403 = false;
    const unsub401 = onUnauthorizedError(() => {
      saw401 = true;
    });
    const unsub403 = onForbiddenError(() => {
      saw403 = true;
    });

    server.use(
      http.get('/api/v1/test-401', () =>
        HttpResponse.json(
          {
            error: {
              code: 'UNAUTHENTICATED',
              message: 'Expired',
              request_id: 'r1',
              retryable: false,
            },
          },
          { status: 401 },
        ),
      ),
      http.get('/api/v1/test-403', () =>
        HttpResponse.json(
          {
            error: {
              code: 'FORBIDDEN',
              message: 'Denied',
              request_id: 'r2',
              retryable: false,
            },
          },
          { status: 403 },
        ),
      ),
    );

    await expect(apiRequest('/test-401')).rejects.toThrow(ApiError);
    await expect(apiRequest('/test-403')).rejects.toThrow(ApiError);

    unsub401();
    unsub403();
    expect(saw401).toBe(true);
    expect(saw403).toBe(true);
  });

  it('TestEXIT01_OperationPollerNeverMarks202DoneUntilTerminalState', async () => {
    const initialOpRes = await portalApi.putNodePublication(
      '93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb',
      { publication_intent: 'published' },
    );

    expect(initialOpRes.status).toBe(202);
    expect(initialOpRes.isTerminal).toBe(false);
    expect(initialOpRes.operation.status).toBe('running');

    const { result } = renderHook(() =>
      useOperationPoller({ scope: 'admin', timeUnitMs: 20 }),
    );

    act(() => {
      result.current.startTracking(initialOpRes);
    });

    // Immediately after 202, isDone MUST be false and isPolling MUST be true
    expect(result.current.isDone).toBe(false);
    expect(result.current.isPolling).toBe(true);

    // Wait for polling to reach terminal 'succeeded' state
    await waitFor(() => {
      expect(result.current.isDone).toBe(true);
    });

    expect(result.current.isSuccess).toBe(true);
    expect(result.current.isPolling).toBe(false);
    expect(result.current.operation?.status).toBe('succeeded');
  });
});
