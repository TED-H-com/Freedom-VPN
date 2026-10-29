/**
 * MSW request handlers for `/api/v1/*` covering:
 * - Role & session switching
 * - Idempotency-Key deduplication & 409 IDEMPOTENCY_CONFLICT detection
 * - ETag & If-Match precondition verification (428 missing, 412 conflict)
 * - 202 Accepted multi-step operation polling with `Retry-After`
 * - One-time enrollment token (`Cache-Control: no-store` on POST, never returned on GET)
 * - Fleet update batches with all 12 mutually exclusive buckets
 * - Member endpoints & checkout sessions
 */

import { http, HttpResponse } from 'msw';
import type {
  CreateEnrollmentRequest,
  CreateMemberGrantRequest,
  CreateUpdateBatchRequest,
  DnsPolicyState,
  DnsRule,
  EnrollmentCreatedResponse,
  EnrollmentDetail,
  FairUsePolicyDraft,
  FairUsePolicyState,
  MemberItem,
  NodeItem,
  Operation,
  PatchMemberRequest,
  UpdateBatchDetail,
  UpdateNodePatchRequest,
  UpdateNodePublicationRequest,
} from '../api/types';
import {
  buildOverviewSummary,
  createMultiStepOperation,
  getPersonaSession,
  INITIAL_AUDIT_EVENTS,
  INITIAL_BATCH,
  INITIAL_DNS_POLICY,
  INITIAL_ENROLLMENTS,
  INITIAL_FUP_POLICY,
  INITIAL_MEMBERS,
  INITIAL_NODES,
  INITIAL_RELEASES_RESPONSE,
  MEMBER_EXITS,
  type MockPersonaKey,
} from './data';

interface StoredOperation {
  polls: Operation[];
  cursor: number;
}

interface MockServerState {
  persona: MockPersonaKey;
  forceSessionExpired: boolean;
  nodes: NodeItem[];
  enrollments: EnrollmentDetail[];
  members: MemberItem[];
  fupPolicy: FairUsePolicyState;
  dnsPolicy: DnsPolicyState;
  activeBatch: UpdateBatchDetail;
  operations: Map<string, StoredOperation>;
  idempotencyRecords: Map<
    string,
    { payloadJson: string; status: number; responseBody: unknown }
  >;
  reqCounter: number;
}

function clone<T>(val: T): T {
  return JSON.parse(JSON.stringify(val)) as T;
}

function createInitialState(): MockServerState {
  return {
    persona: 'admin',
    forceSessionExpired: false,
    nodes: clone(INITIAL_NODES),
    enrollments: clone(INITIAL_ENROLLMENTS),
    members: clone(INITIAL_MEMBERS),
    fupPolicy: clone(INITIAL_FUP_POLICY),
    dnsPolicy: clone(INITIAL_DNS_POLICY),
    activeBatch: clone(INITIAL_BATCH),
    operations: new Map(),
    idempotencyRecords: new Map(),
    reqCounter: 100,
  };
}

let state: MockServerState = createInitialState();

export function resetMockState(persona: MockPersonaKey = 'admin'): void {
  state = createInitialState();
  state.persona = persona;
}

export function setMockPersona(persona: MockPersonaKey): void {
  state.persona = persona;
  state.forceSessionExpired = false;
}

export function getMockPersona(): MockPersonaKey {
  return state.persona;
}

export function triggerMockSessionExpired(): void {
  state.forceSessionExpired = true;
}

function nextRequestId(): string {
  state.reqCounter += 1;
  const suffix = state.reqCounter.toString(16).padStart(12, '0');
  return `a12d4141-4f3e-487b-8638-${suffix}`;
}

function nextUuid(prefixHex: string): string {
  state.reqCounter += 1;
  const suffix = state.reqCounter.toString(16).padStart(12, '0');
  return `${prefixHex}-cd9b-4c3f-a526-${suffix}`;
}

function checkAuth(_request: Request, requireAdmin = false): Response | null {
  const reqId = nextRequestId();
  if (state.forceSessionExpired) {
    return HttpResponse.json(
      {
        error: {
          code: 'UNAUTHENTICATED',
          message: '登入階段已過期，請重新登入。',
          request_id: reqId,
          retryable: false,
        },
      },
      { status: 401, headers: { 'X-Request-ID': reqId } },
    );
  }
  const session = getPersonaSession(state.persona);
  if (requireAdmin && session.role !== 'admin') {
    return HttpResponse.json(
      {
        error: {
          code: 'FORBIDDEN',
          message: '您的帳號角色無權存取管理員端點。',
          request_id: reqId,
          retryable: false,
        },
      },
      {
        status: 403,
        headers: {
          'X-Request-ID': reqId,
          ' detail-header': reqId,
        },
      },
    );
  }
  return null;
}

export const handlers = [
  // Session
  http.get('/api/v1/session', ({ request }) => {
    const authErr = checkAuth(request, false);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(getPersonaSession(state.persona), {
      status: 200,
      headers: {
        'X-Request-ID': reqId,
        'Cache-Control': 'no-store',
      },
    });
  }),

  // Admin Overview
  http.get('/api/v1/admin/overview', ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(buildOverviewSummary(state.nodes, state.members), {
      status: 200,
      headers: { 'X-Request-ID': reqId },
    });
  }),

  // Admin Nodes List
  http.get('/api/v1/admin/nodes', ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(
      { items: state.nodes },
      { status: 200, headers: { 'X-Request-ID': reqId } },
    );
  }),

  // Admin Node Detail
  http.get('/api/v1/admin/nodes/:nodeId', ({ request, params }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const node = state.nodes.find((n) => n.node_id === params['nodeId']);
    if (!node) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的節點。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }
    return HttpResponse.json(node, {
      status: 200,
      headers: {
        'X-Request-ID': reqId,
        ETag: node.etag,
      },
    });
  }),

  // Admin Node PATCH (ETag + 412 conflict + 202 operation)
  http.patch('/api/v1/admin/nodes/:nodeId', async ({ request, params }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const node = state.nodes.find((n) => n.node_id === params['nodeId']);
    if (!node) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的節點。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }

    const ifMatch = request.headers.get('If-Match');
    if (!ifMatch) {
      return HttpResponse.json(
        {
          error: {
            code: 'PRECONDITION_REQUIRED',
            message: '修改既有資源必須提供 If-Match ETag 標頭。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 428, headers: { 'X-Request-ID': reqId } },
      );
    }

    const body = (await request.json()) as UpdateNodePatchRequest;

    if (body.simulate_conflict || ifMatch !== node.etag) {
      // Simulate another admin having incremented the version on the server
      if (body.simulate_conflict) {
        const nextVer = String(BigInt(node.resource_version) + 1n);
        node.resource_version = nextVer;
        node.etag = `"node-${node.node_id.slice(0, 8)}-v${nextVer}"`;
        node.notes = `${node.notes} [Concurrent update v${nextVer}]`;
      }
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_VERSION_CONFLICT',
            message: '節點設定已有更新，請重新讀取後再儲存。',
            request_id: reqId,
            retryable: false,
            details: { current_resource_version: node.resource_version },
          },
        },
        { status: 412, headers: { 'X-Request-ID': reqId } },
      );
    }

    const nextVer = String(BigInt(node.resource_version) + 1n);
    const nextGen = String(BigInt(node.config_generation.desired) + 1n);
    node.resource_version = nextVer;
    node.etag = `"node-${node.node_id.slice(0, 8)}-v${nextVer}"`;
    if (body.name !== undefined) node.name = body.name;
    if (body.region !== undefined) node.region = body.region;
    if (body.member_groups !== undefined) node.member_groups = body.member_groups;
    if (body.desired_up_bps !== undefined) {
      node.capacity.desired_up_bps = body.desired_up_bps;
    }
    if (body.desired_down_bps !== undefined) {
      node.capacity.desired_down_bps = body.desired_down_bps;
    }
    if (body.notes !== undefined) node.notes = body.notes;
    node.config_generation.desired = nextGen;

    const opId = nextUuid('8089f821');
    const createdOp = createMultiStepOperation({
      operationId: opId,
      kind: 'node.configure',
      resourceVersion: nextVer,
      desiredGeneration: nextGen,
      observedGeneration: node.config_generation.applied,
      finalStatus: 'succeeded',
    });

    state.operations.set(opId, { polls: createdOp.polls, cursor: 0 });

    return HttpResponse.json(createdOp.polls[0], {
      status: 202,
      headers: {
        'X-Request-ID': reqId,
        ETag: node.etag,
        Location: `/api/v1/admin/operations/${opId}`,
        'Retry-After': '1',
      },
    });
  }),

  // Admin Node Publication (PUT -> 202 with step progress)
  http.put('/api/v1/admin/nodes/:nodeId/publication', async ({ request, params }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const node = state.nodes.find((n) => n.node_id === params['nodeId']);
    if (!node) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的節點。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }

    const body = (await request.json()) as UpdateNodePublicationRequest;
    const idemKey = request.headers.get('Idempotency-Key');
    const bodyJson = JSON.stringify(body);
    if (idemKey) {
      const scopeKey = `PUT:/admin/nodes/${node.node_id}/publication:${idemKey}`;
      const existing = state.idempotencyRecords.get(scopeKey);
      if (existing) {
        if (existing.payloadJson !== bodyJson) {
          return HttpResponse.json(
            {
              error: {
                code: 'IDEMPOTENCY_CONFLICT',
                message: '相同的 Idempotency-Key 搭配了不同的請求內容。',
                request_id: reqId,
                retryable: false,
              },
            },
            { status: 409, headers: { 'X-Request-ID': reqId } },
          );
        }
        return HttpResponse.json(existing.responseBody as Operation, {
          status: 202,
          headers: { 'X-Request-ID': reqId, 'Retry-After': '1' },
        });
      }
    }

    const nextVer = String(BigInt(node.resource_version) + 1n);
    const nextGen = String(BigInt(node.config_generation.desired) + 1n);
    node.publication_intent = body.publication_intent;
    node.publication_observed = 'applying';
    node.resource_version = nextVer;
    node.etag = `"node-${node.node_id.slice(0, 8)}-v${nextVer}"`;
    node.config_generation.desired = nextGen;

    const isOfflineUnpublish =
      node.control_status === 'offline' && body.publication_intent !== 'published';

    const opId = nextUuid('8089f821');
    const createdOp = createMultiStepOperation({
      operationId: opId,
      kind: body.publication_intent === 'published' ? 'node.publish' : 'node.unpublish',
      resourceVersion: nextVer,
      desiredGeneration: nextGen,
      observedGeneration: node.config_generation.applied,
      finalStatus: isOfflineUnpublish ? 'partially_succeeded' : 'succeeded',
      summaryCode: isOfflineUnpublish ? 'route_withdrawn_device_unconfirmed' : undefined,
      stepNames:
        body.publication_intent === 'published'
          ? ['agent_policy', 'netbird_route_readback', 'data_path_probe']
          : ['netbird_route_readback', 'node_allowset_withdraw'],
    });

    state.operations.set(opId, { polls: createdOp.polls, cursor: 0 });

    if (idemKey) {
      const scopeKey = `PUT:/admin/nodes/${node.node_id}/publication:${idemKey}`;
      state.idempotencyRecords.set(scopeKey, {
        payloadJson: bodyJson,
        status: 202,
        responseBody: createdOp.polls[0],
      });
    }

    return HttpResponse.json(createdOp.polls[0], {
      status: 202,
      headers: {
        'X-Request-ID': reqId,
        ETag: node.etag,
        Location: `/api/v1/admin/operations/${opId}`,
        'Retry-After': '1',
      },
    });
  }),

  // Admin Node Revoke (POST -> 202)
  http.post('/api/v1/admin/nodes/:nodeId/revoke', ({ request, params }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const node = state.nodes.find((n) => n.node_id === params['nodeId']);
    if (!node) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的節點。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }

    node.lifecycle_state = 'revoked';
    node.publication_intent = 'disabled';
    node.publication_observed = 'withdrawn';
    node.serving_allowed = false;
    node.summary_code = 'revoked_not_serving';

    const opId = nextUuid('8089f821');
    const createdOp = createMultiStepOperation({
      operationId: opId,
      kind: 'node.revoke',
      resourceVersion: node.resource_version,
      desiredGeneration: node.config_generation.desired,
      observedGeneration: node.config_generation.applied,
      finalStatus: 'succeeded',
      stepNames: [
        'netbird_route_readback',
        'cert_and_peer_revoke',
        'node_allowset_withdraw',
      ],
    });
    state.operations.set(opId, { polls: createdOp.polls, cursor: 0 });

    return HttpResponse.json(createdOp.polls[0], {
      status: 202,
      headers: {
        'X-Request-ID': reqId,
        Location: `/api/v1/admin/operations/${opId}`,
        'Retry-After': '1',
      },
    });
  }),

  // Admin Create Enrollment (POST -> 201 with one-time token and Cache-Control: no-store)
  http.post('/api/v1/admin/enrollments', async ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const body = (await request.json()) as CreateEnrollmentRequest;

    const enrollmentId = nextUuid('550e8400');
    const nodeId = nextUuid('93bc18d9');
    const nowIso = '2026-10-29T12:00:00Z';
    const expiresIso = '2026-11-05T12:00:00Z';

    const detailRecord: EnrollmentDetail = {
      enrollment_id: enrollmentId,
      node_id: nodeId,
      state: 'unused',
      secret_available: false,
      name: body.name?.trim() || 'pending-node',
      region: body.region?.trim() || 'UNASSIGNED',
      member_groups: body.member_groups.length > 0 ? body.member_groups : ['standard'],
      notes: body.notes?.trim() || '',
      expires_at: expiresIso,
      created_at: nowIso,
      consumed_at: null,
      revoked_at: null,
    };
    state.enrollments.unshift(detailRecord);

    const createdResponse: EnrollmentCreatedResponse = {
      enrollment_id: enrollmentId,
      node_id: nodeId,
      state: 'unused',
      // TEST ONLY one-time token generated per request, never stored in detailRecord
      enrollment_token: `fvpn_enr_test_only_${enrollmentId.replace(/-/g, '')}`,
      secret_available: true,
      name: detailRecord.name,
      region: detailRecord.region,
      member_groups: detailRecord.member_groups,
      notes: detailRecord.notes,
      expires_at: expiresIso,
      created_at: nowIso,
    };

    return HttpResponse.json(createdResponse, {
      status: 201,
      headers: {
        'X-Request-ID': reqId,
        'Cache-Control': 'no-store',
      },
    });
  }),

  // Admin Get Enrollment (NEVER returns the raw token)
  http.get('/api/v1/admin/enrollments/:enrollmentId', ({ request, params }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const record = state.enrollments.find(
      (e) => e.enrollment_id === params['enrollmentId'],
    );
    if (!record) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的啟用碼紀錄。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }
    return HttpResponse.json(record, {
      status: 200,
      headers: {
        'X-Request-ID': reqId,
        'Cache-Control': 'no-store',
      },
    });
  }),

  // Admin Revoke Enrollment
  http.post('/api/v1/admin/enrollments/:enrollmentId/revoke', ({ request, params }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const record = state.enrollments.find(
      (e) => e.enrollment_id === params['enrollmentId'],
    );
    if (!record) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的啟用碼紀錄。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }
    record.state = 'revoked';
    record.revoked_at = '2026-10-29T12:05:00Z';
    return HttpResponse.json(record, {
      status: 200,
      headers: { 'X-Request-ID': reqId },
    });
  }),

  // Admin Releases & Active Update Batch
  http.get('/api/v1/admin/releases', ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(
      {
        ...INITIAL_RELEASES_RESPONSE,
        fleet_default_release_id: state.activeBatch.fleet_default_release_id,
        active_batch: state.activeBatch,
      },
      { status: 200, headers: { 'X-Request-ID': reqId } },
    );
  }),

  // Admin Create Update Batch (202 Accepted)
  http.post('/api/v1/admin/update-batches', async ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const body = (await request.json()) as CreateUpdateBatchRequest;
    const idemKey = request.headers.get('Idempotency-Key');
    const bodyJson = JSON.stringify(body);

    if (idemKey) {
      const scopeKey = `POST:/admin/update-batches:${idemKey}`;
      const existing = state.idempotencyRecords.get(scopeKey);
      if (existing) {
        if (existing.payloadJson !== bodyJson) {
          return HttpResponse.json(
            {
              error: {
                code: 'IDEMPOTENCY_CONFLICT',
                message: '相同的 Idempotency-Key 搭配了不同的請求內容。',
                request_id: reqId,
                retryable: false,
              },
            },
            { status: 409, headers: { 'X-Request-ID': reqId } },
          );
        }
        return HttpResponse.json(existing.responseBody as UpdateBatchDetail, {
          status: 202,
          headers: { 'X-Request-ID': reqId },
        });
      }
    }

    const rel = INITIAL_RELEASES_RESPONSE.releases.find(
      (r) => r.release_id === body.release_id,
    );
    if (!rel || !rel.approved) {
      return HttpResponse.json(
        {
          error: {
            code: 'RELEASE_NOT_APPROVED',
            message: '所選發行版本未核准，不可建立批次更新。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 422, headers: { 'X-Request-ID': reqId } },
      );
    }

    state.activeBatch = {
      ...state.activeBatch,
      release_id: rel.release_id,
      product_version: rel.product_version,
      requires_service_restart: rel.requires_service_restart,
      fleet_default_release_id: rel.release_id,
      updated_at: '2026-10-29T12:00:00Z',
    };

    if (idemKey) {
      const scopeKey = `POST:/admin/update-batches:${idemKey}`;
      state.idempotencyRecords.set(scopeKey, {
        payloadJson: bodyJson,
        status: 202,
        responseBody: state.activeBatch,
      });
    }

    return HttpResponse.json(state.activeBatch, {
      status: 202,
      headers: { 'X-Request-ID': reqId },
    });
  }),

  // Admin Get Update Batch
  http.get('/api/v1/admin/update-batches/:batchId', ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(state.activeBatch, {
      status: 200,
      headers: { 'X-Request-ID': reqId },
    });
  }),

  // Admin Retry Failed Targets in Batch (202 Accepted)
  http.post('/api/v1/admin/update-batches/:batchId/retry', ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();

    const opId = nextUuid('9012f821');
    const createdOp = createMultiStepOperation({
      operationId: opId,
      kind: 'update_batch.retry_failed',
      resourceVersion: '43',
      desiredGeneration: '8',
      observedGeneration: '7',
      finalStatus: 'succeeded',
      stepNames: ['agent_policy', 'data_path_probe'],
    });
    state.operations.set(opId, { polls: createdOp.polls, cursor: 0 });

    return HttpResponse.json(createdOp.polls[0], {
      status: 202,
      headers: {
        'X-Request-ID': reqId,
        Location: `/api/v1/admin/operations/${opId}`,
        'Retry-After': '1',
      },
    });
  }),

  // Admin Members List & Detail
  http.get('/api/v1/admin/members', ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(
      { items: state.members },
      { status: 200, headers: { 'X-Request-ID': reqId } },
    );
  }),

  http.get('/api/v1/admin/members/:memberId', ({ request, params }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const member = state.members.find((m) => m.member_id === params['memberId']);
    if (!member) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的會員。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }
    return HttpResponse.json(member, {
      status: 200,
      headers: {
        'X-Request-ID': reqId,
        ETag: member.etag,
      },
    });
  }),

  http.patch('/api/v1/admin/members/:memberId', async ({ request, params }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const member = state.members.find((m) => m.member_id === params['memberId']);
    if (!member) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的會員。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }

    const ifMatch = request.headers.get('If-Match');
    if (!ifMatch) {
      return HttpResponse.json(
        {
          error: {
            code: 'PRECONDITION_REQUIRED',
            message: '修改會員資料必須提供 If-Match ETag。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 428, headers: { 'X-Request-ID': reqId } },
      );
    }

    if (ifMatch !== member.etag) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_VERSION_CONFLICT',
            message: '會員狀態已被更新，請重新載入後再試。',
            request_id: reqId,
            retryable: false,
            details: { current_resource_version: member.resource_version },
          },
        },
        { status: 412, headers: { 'X-Request-ID': reqId } },
      );
    }

    const body = (await request.json()) as PatchMemberRequest;
    const nextVer = String(BigInt(member.resource_version) + 1n);
    member.resource_version = nextVer;
    member.etag = `"member-${member.member_id.slice(-4)}-v${nextVer}"`;

    if (typeof body.admin_suspended === 'boolean') {
      member.admin_suspended = body.admin_suspended;
      member.admin_suspension_reason = body.admin_suspended
        ? (body.admin_suspension_reason ?? '管理員執行人工停權')
        : null;
      member.entitlement_state = body.admin_suspended ? 'suspended' : 'active';
      member.entitlement_reason = body.admin_suspended
        ? '管理員已執行人工停權（優先於付款狀態，所有裝置立即停止轉送，§9.2）'
        : '人工停權已解除，依有效服務期限恢復轉送資格';
    }

    const opId = nextUuid('7012f821');
    const createdOp = createMultiStepOperation({
      operationId: opId,
      kind: body.admin_suspended ? 'member.suspend' : 'member.resume',
      resourceVersion: nextVer,
      desiredGeneration: '53',
      observedGeneration: '52',
      finalStatus: 'succeeded',
      stepNames: ['policy_outbox_sync', 'node_allowset_withdraw'],
    });
    state.operations.set(opId, { polls: createdOp.polls, cursor: 0 });

    return HttpResponse.json(createdOp.polls[0], {
      status: 202,
      headers: {
        'X-Request-ID': reqId,
        ETag: member.etag,
        Location: `/api/v1/admin/operations/${opId}`,
        'Retry-After': '1',
      },
    });
  }),

  http.post('/api/v1/admin/members/:memberId/grants', async ({ request, params }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const member = state.members.find((m) => m.member_id === params['memberId']);
    if (!member) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的會員。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }

    const body = (await request.json()) as CreateMemberGrantRequest;
    if (!body.valid_until || !body.reason.trim()) {
      return HttpResponse.json(
        {
          error: {
            code: 'VALIDATION_FAILED',
            message: '有限期人工授權必須提供截止時間與理由（§14.2）。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 422, headers: { 'X-Request-ID': reqId } },
      );
    }

    member.grants.unshift({
      grant_id: nextUuid('660e8400'),
      member_id: member.member_id,
      source: 'admin_manual_grant',
      valid_from: '2026-10-29T12:00:00Z',
      valid_until: body.valid_until,
      reason: body.reason,
      created_by: 'admin-01@example.com',
      revoked_at: null,
    });
    member.effective_service_end = body.valid_until;

    const opId = nextUuid('7012f821');
    const createdOp = createMultiStepOperation({
      operationId: opId,
      kind: 'member.grant.create',
      resourceVersion: member.resource_version,
      desiredGeneration: '54',
      observedGeneration: '53',
      finalStatus: 'succeeded',
      stepNames: ['policy_outbox_sync', 'agent_policy'],
    });
    state.operations.set(opId, { polls: createdOp.polls, cursor: 0 });

    return HttpResponse.json(createdOp.polls[0], {
      status: 202,
      headers: {
        'X-Request-ID': reqId,
        Location: `/api/v1/admin/operations/${opId}`,
        'Retry-After': '1',
      },
    });
  }),

  http.post(
    '/api/v1/admin/members/:memberId/devices/:deviceId/revoke',
    ({ request, params }) => {
      const authErr = checkAuth(request, true);
      if (authErr) return authErr;
      const reqId = nextRequestId();
      const member = state.members.find((m) => m.member_id === params['memberId']);
      const dev = member?.devices.find((d) => d.device_id === params['deviceId']);
      if (dev && member) {
        dev.status = 'revoked';
        dev.revoked_at = '2026-10-29T12:00:00Z';
        member.authorized_device_count = member.devices.filter(
          (d) => d.status === 'authorized',
        ).length;
      }

      const opId = nextUuid('7012f821');
      const createdOp = createMultiStepOperation({
        operationId: opId,
        kind: 'device.revoke',
        resourceVersion: member?.resource_version ?? '1',
        desiredGeneration: '55',
        observedGeneration: '54',
        finalStatus: 'succeeded',
        stepNames: ['node_allowset_withdraw', 'cert_and_peer_revoke'],
      });
      state.operations.set(opId, { polls: createdOp.polls, cursor: 0 });

      return HttpResponse.json(createdOp.polls[0], {
        status: 202,
        headers: {
          'X-Request-ID': reqId,
          Location: `/api/v1/admin/operations/${opId}`,
          'Retry-After': '1',
        },
      });
    },
  ),

  // Admin FUP Policy
  http.get('/api/v1/admin/policies/fair_use', ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(state.fupPolicy, {
      status: 200,
      headers: {
        'X-Request-ID': reqId,
        ETag: state.fupPolicy.etag,
      },
    });
  }),

  http.post('/api/v1/admin/policies/fair_use/activate', async ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const draft = (await request.json()) as FairUsePolicyDraft;

    const nextRev = String(BigInt(state.fupPolicy.active_revision_number) + 1n);
    state.fupPolicy = {
      ...state.fupPolicy,
      active_revision_number: nextRev,
      resource_version: nextRev,
      etag: `"policy-fup-v${nextRev}"`,
      current: draft,
    };

    const opId = nextUuid('8089f821');
    const createdOp = createMultiStepOperation({
      operationId: opId,
      kind: 'policy.fair_use.activate',
      resourceVersion: nextRev,
      desiredGeneration: nextRev,
      observedGeneration: String(BigInt(nextRev) - 1n),
      finalStatus: 'succeeded',
      stepNames: ['policy_outbox_sync', 'agent_policy'],
    });
    state.operations.set(opId, { polls: createdOp.polls, cursor: 0 });

    return HttpResponse.json(createdOp.polls[0], {
      status: 202,
      headers: {
        'X-Request-ID': reqId,
        ETag: state.fupPolicy.etag,
        Location: `/api/v1/admin/operations/${opId}`,
        'Retry-After': '1',
      },
    });
  }),

  // Admin DNS Policy
  http.get('/api/v1/admin/policies/dns', ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(state.dnsPolicy, {
      status: 200,
      headers: {
        'X-Request-ID': reqId,
        ETag: state.dnsPolicy.etag,
      },
    });
  }),

  http.post('/api/v1/admin/policies/dns/rules', async ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const body = (await request.json()) as Omit<DnsRule, 'rule_id' | 'revision'>;

    const nextVer = String(BigInt(state.dnsPolicy.resource_version) + 1n);
    const newRule: DnsRule = {
      ...body,
      rule_id: nextUuid('8e210b2c'),
      revision: nextVer,
    };
    state.dnsPolicy = {
      ...state.dnsPolicy,
      desired_revision: nextVer,
      resource_version: nextVer,
      etag: `"policy-dns-v${nextVer}"`,
      rules: [newRule, ...state.dnsPolicy.rules],
    };

    return HttpResponse.json(state.dnsPolicy, {
      status: 201,
      headers: {
        'X-Request-ID': reqId,
        ETag: state.dnsPolicy.etag,
      },
    });
  }),

  http.post('/api/v1/admin/policies/dns/activate', ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();

    const opId = nextUuid('8089f821');
    const createdOp = createMultiStepOperation({
      operationId: opId,
      kind: 'policy.dns.activate',
      resourceVersion: state.dnsPolicy.resource_version,
      desiredGeneration: state.dnsPolicy.desired_revision,
      observedGeneration: state.dnsPolicy.active_revision,
      finalStatus: 'succeeded',
      stepNames: ['resolver_1_smoke_apply', 'resolver_2_smoke_apply'],
    });
    state.operations.set(opId, { polls: createdOp.polls, cursor: 0 });
    state.dnsPolicy.active_revision = state.dnsPolicy.desired_revision;

    return HttpResponse.json(createdOp.polls[0], {
      status: 202,
      headers: {
        'X-Request-ID': reqId,
        Location: `/api/v1/admin/operations/${opId}`,
        'Retry-After': '1',
      },
    });
  }),

  http.post('/api/v1/admin/policies/dns/rollback', ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();

    const opId = nextUuid('8089f821');
    const createdOp = createMultiStepOperation({
      operationId: opId,
      kind: 'policy.dns.rollback',
      resourceVersion: state.dnsPolicy.resource_version,
      desiredGeneration: state.dnsPolicy.active_revision,
      observedGeneration: state.dnsPolicy.active_revision,
      finalStatus: 'succeeded',
      stepNames: ['resolver_1_smoke_apply', 'resolver_2_smoke_apply'],
    });
    state.operations.set(opId, { polls: createdOp.polls, cursor: 0 });

    return HttpResponse.json(createdOp.polls[0], {
      status: 202,
      headers: {
        'X-Request-ID': reqId,
        Location: `/api/v1/admin/operations/${opId}`,
        'Retry-After': '1',
      },
    });
  }),

  // Admin Audit Events
  http.get('/api/v1/admin/audit-events', ({ request }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(
      { items: INITIAL_AUDIT_EVENTS },
      { status: 200, headers: { 'X-Request-ID': reqId } },
    );
  }),

  // Admin & Member Operation Polling
  http.get('/api/v1/admin/operations/:operationId', ({ request, params }) => {
    const authErr = checkAuth(request, true);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const opId = String(params['operationId']);
    const stored = state.operations.get(opId);
    if (!stored) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的操作紀錄。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }
    const idx = Math.min(stored.cursor, stored.polls.length - 1);
    const currentOp = stored.polls[idx]!;
    if (stored.cursor < stored.polls.length - 1) {
      stored.cursor += 1;
    }
    return HttpResponse.json(currentOp, {
      status: 200,
      headers: {
        'X-Request-ID': reqId,
        'Retry-After': '1',
      },
    });
  }),

  http.get('/api/v1/member/operations/:operationId', ({ request, params }) => {
    const authErr = checkAuth(request, false);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const opId = String(params['operationId']);
    const stored = state.operations.get(opId);
    if (!stored) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的操作紀錄。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }
    const idx = Math.min(stored.cursor, stored.polls.length - 1);
    const currentOp = stored.polls[idx]!;
    if (stored.cursor < stored.polls.length - 1) {
      stored.cursor += 1;
    }
    return HttpResponse.json(currentOp, {
      status: 200,
      headers: {
        'X-Request-ID': reqId,
        'Retry-After': '1',
      },
    });
  }),

  // Member Endpoints
  http.get('/api/v1/member/me', ({ request }) => {
    const authErr = checkAuth(request, false);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const session = getPersonaSession(state.persona);
    const member =
      state.members.find((m) => m.member_id === session.member_id) ?? state.members[0]!;
    return HttpResponse.json(
      {
        member,
        management_url: 'https://netbird.vpn.example.com',
      },
      {
        status: 200,
        headers: {
          'X-Request-ID': reqId,
          'Cache-Control': 'no-store',
        },
      },
    );
  }),

  http.get('/api/v1/member/devices', ({ request }) => {
    const authErr = checkAuth(request, false);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const session = getPersonaSession(state.persona);
    const member =
      state.members.find((m) => m.member_id === session.member_id) ?? state.members[0]!;
    return HttpResponse.json(
      { items: member.devices },
      { status: 200, headers: { 'X-Request-ID': reqId } },
    );
  }),

  http.delete('/api/v1/member/devices/:deviceId', ({ request, params }) => {
    const authErr = checkAuth(request, false);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const session = getPersonaSession(state.persona);
    const member =
      state.members.find((m) => m.member_id === session.member_id) ?? state.members[0]!;
    const dev = member.devices.find((d) => d.device_id === params['deviceId']);
    if (!dev) {
      return HttpResponse.json(
        {
          error: {
            code: 'RESOURCE_NOT_FOUND',
            message: '找不到指定的裝置。',
            request_id: reqId,
            retryable: false,
          },
        },
        { status: 404, headers: { 'X-Request-ID': reqId } },
      );
    }

    dev.status = 'revoked';
    dev.revoked_at = '2026-10-29T12:00:00Z';
    member.authorized_device_count = member.devices.filter(
      (d) => d.status === 'authorized',
    ).length;

    const opId = nextUuid('7012f821');
    const createdOp = createMultiStepOperation({
      operationId: opId,
      kind: 'member.device.revoke',
      resourceVersion: dev.resource_version,
      desiredGeneration: '60',
      observedGeneration: '59',
      finalStatus: 'succeeded',
      stepNames: ['node_allowset_withdraw', 'cert_and_peer_revoke'],
    });
    state.operations.set(opId, { polls: createdOp.polls, cursor: 0 });

    return HttpResponse.json(createdOp.polls[0], {
      status: 202,
      headers: {
        'X-Request-ID': reqId,
        Location: `/api/v1/member/operations/${opId}`,
        'Retry-After': '1',
      },
    });
  }),

  http.get('/api/v1/member/exits', ({ request }) => {
    const authErr = checkAuth(request, false);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(
      { items: MEMBER_EXITS },
      { status: 200, headers: { 'X-Request-ID': reqId } },
    );
  }),

  http.get('/api/v1/member/usage', ({ request }) => {
    const authErr = checkAuth(request, false);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    const session = getPersonaSession(state.persona);
    const member =
      state.members.find((m) => m.member_id === session.member_id) ?? state.members[0]!;
    return HttpResponse.json(member, {
      status: 200,
      headers: { 'X-Request-ID': reqId },
    });
  }),

  http.post('/api/v1/member/checkout-sessions', ({ request }) => {
    const authErr = checkAuth(request, false);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(
      {
        checkout_session_id: 'cs_test_9911223344',
        hosted_url:
          'https://checkout.payment-provider.example.com/pay/cs_test_9911223344',
        expires_at: '2026-10-29T13:00:00Z',
      },
      {
        status: 201,
        headers: {
          'X-Request-ID': reqId,
          'Cache-Control': 'no-store',
        },
      },
    );
  }),

  http.post('/api/v1/member/billing-portal-sessions', ({ request }) => {
    const authErr = checkAuth(request, false);
    if (authErr) return authErr;
    const reqId = nextRequestId();
    return HttpResponse.json(
      {
        portal_session_id: 'bps_test_55667788',
        hosted_url: 'https://billing.payment-provider.example.com/p/bps_test_55667788',
      },
      {
        status: 201,
        headers: {
          'X-Request-ID': reqId,
          'Cache-Control': 'no-store',
        },
      },
    );
  }),
];
