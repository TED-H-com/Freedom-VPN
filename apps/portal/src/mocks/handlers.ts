/**
 * MSW v2 request handlers for `/auth/*` and `/api/v1/*` backed by
 * `contracts/examples/valid/*.json` (Deliverables 3, 4, 5).
 */

import { http, HttpResponse } from 'msw';
import type {
  EnrollmentCreateRequest,
  GrantCreateRequest,
  MemberGroupCreate,
  MemberPatchRequest,
  NodePatchRequest,
  NodeSummaryItem,
  OperationResource,
  PolicyActivateRequest,
  PolicyRevision,
  PolicyRevisionCreate,
  PublicationRequest,
  ReasonRequest,
  UpdateBatchAccepted,
  UpdateBatchCreateRequest,
} from '../api/types';
import type { UpdateTargetWithResult } from '../api/view-models';
import { normalizeAndValidateDomain } from '../lib/dns';
import {
  buildMemberSelfForPersona,
  buildMemberUsageForPersona,
  buildPersonaSession,
  CONTRACT_EXAMPLES_USED_BY_MOCKS,
  createInitialMockState,
  type MockPersona,
  type MockState,
} from './data';

export type { MockPersona };

let state: MockState = createInitialMockState('admin');
let opCounter = 500;

export function resetMockState(persona: MockPersona = 'admin'): void {
  state = createInitialMockState(persona);
  opCounter = 500;
}

export function setMockRolledBackCandidateHealth(mode: 'populated' | 'null'): void {
  const target = state.updateBatch.targets.find((t) => t.state === 'rolled_back') as
    | UpdateTargetWithResult
    | undefined;
  if (!target) return;
  const source = structuredClone(CONTRACT_EXAMPLES_USED_BY_MOCKS.UpdateResultRolledBack);
  if (mode === 'null') {
    source.candidate_health =
      CONTRACT_EXAMPLES_USED_BY_MOCKS.UpdateResultNullCandidateHealth.candidate_health ??
      null;
  }
  target.previous_release_id = source.previous_release_id;
  target.health = source.health;
  target.candidate_health = source.candidate_health ?? null;
  target.last_result = source;
}

export function setMockPersona(persona: MockPersona): void {
  state.persona = persona;
  state.forceSession401 = false;
}

export function getMockPersona(): MockPersona {
  return state.persona;
}

export function triggerMockSessionExpired(): void {
  state.forceSession401 = true;
}

function nextOperationId(): string {
  opCounter += 1;
  const suffix = String(opCounter).padStart(12, '0');
  return `8089f821-cd9b-4c3f-a526-${suffix}`;
}

function makeErrorResponse(
  status: number,
  code: string,
  message: string,
  retryable = false,
  details: Record<string, unknown> = {},
  extraHeaders: Record<string, string> = {},
) {
  const baseEnvelope = structuredClone(CONTRACT_EXAMPLES_USED_BY_MOCKS.ErrorEnvelope);
  return HttpResponse.json(
    {
      error: {
        ...baseEnvelope.error,
        code,
        message,
        retryable,
        details,
      },
    },
    { status, headers: extraHeaders },
  );
}

function checkSessionGuard(
  request: Request,
  requireAdmin: boolean,
): ReturnType<typeof makeErrorResponse> | null {
  if (state.forceSession401) {
    return makeErrorResponse(401, 'SESSION_EXPIRED', 'Session has expired.');
  }
  const session = buildPersonaSession(state.persona);
  if (!session.authenticated) {
    return makeErrorResponse(401, 'UNAUTHENTICATED', 'Authentication required.');
  }
  if (requireAdmin) {
    if (session.role !== 'admin') {
      return makeErrorResponse(
        403,
        'FORBIDDEN',
        'Member role is not permitted to access /api/v1/admin/* endpoints.',
      );
    }
    if (!session.mfa_verified) {
      return makeErrorResponse(
        403,
        'MFA_REQUIRED',
        'Administrator MFA verification is required.',
      );
    }
  }
  const method = request.method.toUpperCase();
  if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
    const idemKey = request.headers.get('Idempotency-Key');
    if (!idemKey || idemKey.trim().length === 0) {
      return makeErrorResponse(
        422,
        'VALIDATION_ERROR',
        'Missing required Idempotency-Key header on mutating request.',
      );
    }
  }
  return null;
}

function createTrackedOperation(
  kind: OperationResource['kind'],
  resourceType: OperationResource['resource']['type'],
  resourceId: string,
  steps: OperationResource['steps'],
  desiredGen = '129',
  observedGen = '128',
  resourceVersion: string | null = '43',
): OperationResource {
  const op = structuredClone(CONTRACT_EXAMPLES_USED_BY_MOCKS.Operation);
  const opId = nextOperationId();
  op.operation_id = opId;
  op.kind = kind;
  op.status = 'running';
  op.resource = {
    type: resourceType,
    id: resourceId,
  };
  op.resource_version = resourceVersion;
  op.desired_generation = desiredGen;
  op.observed_generation = observedGen;
  op.steps = steps;
  op.error = null;
  op.completed_at = null;
  state.operations[opId] = op;
  state.operationPolls[opId] = 0;
  return op;
}

function advanceAndGetOperation(operationId: string): OperationResource | null {
  const op = state.operations[operationId];
  if (!op) return null;

  const polls = (state.operationPolls[operationId] ?? 0) + 1;
  state.operationPolls[operationId] = polls;

  if (polls >= 1 && (op.status === 'pending' || op.status === 'running')) {
    const hasFailedStep = op.steps.some((s) => s.status === 'failed');
    if (hasFailedStep) {
      op.status = 'partially_succeeded';
    } else {
      op.status = 'succeeded';
      op.observed_generation = op.desired_generation;
      op.steps = op.steps.map((s) => ({
        ...s,
        status: 'succeeded',
        observed_at: '2026-10-28T15:05:10Z',
      }));
    }
    op.updated_at = '2026-10-28T15:05:10Z';
    op.completed_at = '2026-10-28T15:05:10Z';
  }
  return op;
}

function toNodeSummary(detail: (typeof state.nodes)[string]): NodeSummaryItem {
  return {
    node_id: detail.node_id,
    exit_id: detail.exit_id,
    name: detail.name,
    region: detail.region,
    member_group_ids: detail.member_group_ids,
    lifecycle_state: detail.lifecycle_state,
    provisioning_state: detail.provisioning_state,
    publication_intent: detail.publication_intent,
    publication_observed: detail.publication_observed,
    control_status: detail.control_status,
    data_health: detail.data_health,
    safety_status: detail.safety_status,
    member_forwarding_observed: detail.member_forwarding_observed,
    config_generation: detail.config_generation,
    releases: detail.releases,
    update: detail.update,
    last_seen_at: detail.last_seen_at,
    resource_version: detail.resource_version,
    created_at: detail.created_at,
  };
}

export const handlers = [
  // --- Browser Auth Session Routes (outside /api/v1) ---
  http.get('/auth/session', () => {
    if (state.forceSession401) {
      return makeErrorResponse(401, 'SESSION_EXPIRED', 'Session expired.');
    }
    const session = buildPersonaSession(state.persona);
    return HttpResponse.json(session, {
      status: 200,
      headers: {
        'Cache-Control': 'no-store',
      },
    });
  }),

  http.post('/auth/logout', () => {
    state.persona = 'anonymous';
    state.forceSession401 = false;
    return new HttpResponse(null, {
      status: 204,
      headers: {
        'Cache-Control': 'no-store',
      },
    });
  }),

  // --- Admin Groups (D-057) ---
  http.get('/api/v1/admin/groups', ({ request }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    return HttpResponse.json({
      items: state.groups,
      next_cursor: null,
    });
  }),

  http.post('/api/v1/admin/groups', async ({ request }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const body = (await request.json()) as MemberGroupCreate;
    const code = (body.code ?? '').trim();
    const name = (body.name ?? '').trim();

    const codeValid = /^[a-z0-9][a-z0-9-]{1,63}$/.test(code);
    if (!codeValid) {
      return makeErrorResponse(
        422,
        'VALIDATION_FAILED',
        'Request validation failed.',
        false,
        {
          fields: [{ field: '/code', reason: 'invalid' }],
        },
      );
    }
    if (!name || name.length > 120) {
      return makeErrorResponse(
        422,
        'VALIDATION_FAILED',
        'Request validation failed.',
        false,
        {
          fields: [{ field: '/name', reason: 'invalid' }],
        },
      );
    }

    const dup = state.groups.find((g) => g.code === code || g.name === name);
    if (dup) {
      return makeErrorResponse(
        409,
        'STATE_CONFLICT',
        'Group code or name already exists.',
      );
    }

    const created = {
      group_id: `83575bae-2d23-45c0-ae11-${String(state.groups.length + 10).padStart(12, '0')}`,
      code,
      name,
      exit_count: 0,
    };
    state.groups.push(created);
    return HttpResponse.json(created, { status: 201 });
  }),

  // --- Admin Nodes ---
  http.get('/api/v1/admin/nodes', ({ request }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const url = new URL(request.url);
    const pubFilter = url.searchParams.get('publication_intent');
    const controlFilter = url.searchParams.get('control_status');

    let items = Object.values(state.nodes).map(toNodeSummary);
    if (pubFilter) {
      items = items.filter((n) => n.publication_intent === pubFilter);
    }
    if (controlFilter) {
      items = items.filter((n) => n.control_status === controlFilter);
    }

    return HttpResponse.json({
      items,
      next_cursor: null,
    });
  }),

  http.get('/api/v1/admin/nodes/:nodeId', ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const nodeId = String(params.nodeId);
    const node = state.nodes[nodeId];
    if (!node) {
      return makeErrorResponse(404, 'NOT_FOUND', `Node ${nodeId} not found.`);
    }

    return HttpResponse.json(node, {
      headers: {
        ETag: `"${node.resource_version}"`,
      },
    });
  }),

  http.patch('/api/v1/admin/nodes/:nodeId', async ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const nodeId = String(params.nodeId);
    const node = state.nodes[nodeId];
    if (!node) {
      return makeErrorResponse(404, 'NOT_FOUND', `Node ${nodeId} not found.`);
    }

    const currentEtag = `"${node.resource_version}"`;
    const ifMatch = request.headers.get('If-Match');
    if (!ifMatch) {
      return makeErrorResponse(
        428,
        'PRECONDITION_REQUIRED',
        'If-Match header is required when updating node configuration.',
      );
    }
    if (ifMatch !== currentEtag) {
      return makeErrorResponse(
        412,
        'RESOURCE_VERSION_CONFLICT',
        'Node configuration was modified by another process.',
        false,
        {
          current_resource_version: node.resource_version,
        },
      );
    }

    const body = (await request.json()) as NodePatchRequest;
    if (body.capacity && (body.capacity.up_bps <= 0 || body.capacity.down_bps <= 0)) {
      return makeErrorResponse(
        422,
        'VALIDATION_ERROR',
        'Capacity bps values must be positive integers.',
      );
    }

    if (body.name !== undefined) node.name = body.name;
    if (body.region !== undefined) node.region = body.region;
    if (body.member_group_ids !== undefined) {
      node.member_group_ids = body.member_group_ids;
    }
    if (body.clear_capacity) {
      node.capacity = null;
    } else if (body.capacity !== undefined) {
      node.capacity = body.capacity;
    }
    if (body.notes !== undefined) {
      node.notes = body.notes;
    }

    const nextVer = String(BigInt(node.resource_version) + 1n);
    const nextGen = String(BigInt(node.config_generation.desired) + 1n);
    node.resource_version = nextVer;
    node.config_generation.desired = nextGen;

    return HttpResponse.json(node, {
      status: 200,
      headers: {
        ETag: `"${nextVer}"`,
      },
    });
  }),

  http.delete('/api/v1/admin/nodes/:nodeId', ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const nodeId = String(params.nodeId);
    const node = state.nodes[nodeId];
    if (!node) {
      return makeErrorResponse(404, 'NOT_FOUND', `Node ${nodeId} not found.`);
    }
    const ifMatch = request.headers.get('If-Match');
    if (!ifMatch) {
      return makeErrorResponse(
        428,
        'PRECONDITION_REQUIRED',
        'If-Match header is required when retiring a node.',
      );
    }
    if (ifMatch !== `"${node.resource_version}"`) {
      return makeErrorResponse(
        412,
        'RESOURCE_VERSION_CONFLICT',
        'Node version conflict.',
        false,
        { current_resource_version: node.resource_version },
      );
    }

    node.lifecycle_state = 'retired';
    node.publication_intent = 'disabled';
    node.publication_observed = 'withdrawn';
    node.member_forwarding_observed = false;
    node.retired_at = '2026-10-28T15:05:00Z';
    const nextVer = String(BigInt(node.resource_version) + 1n);
    node.resource_version = nextVer;

    const op = createTrackedOperation(
      'node.retire',
      'node',
      nodeId,
      [
        {
          name: 'netbird_route_readback',
          status: 'succeeded',
          required: true,
          observed_at: '2026-10-28T15:05:01Z',
          error_code: null,
          message: null,
        },
      ],
      node.config_generation.desired,
      node.config_generation.applied ?? '0',
      nextVer,
    );
    return HttpResponse.json(op, {
      status: 202,
      headers: { ETag: `"${nextVer}"` },
    });
  }),

  http.put('/api/v1/admin/nodes/:nodeId/publication', async ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const nodeId = String(params.nodeId);
    const node = state.nodes[nodeId];
    if (!node) {
      return makeErrorResponse(404, 'NOT_FOUND', `Node ${nodeId} not found.`);
    }

    const ifMatch = request.headers.get('If-Match');
    if (!ifMatch) {
      return makeErrorResponse(
        428,
        'PRECONDITION_REQUIRED',
        'If-Match header is required when changing node publication intent.',
      );
    }
    if (ifMatch !== `"${node.resource_version}"`) {
      return makeErrorResponse(
        412,
        'RESOURCE_VERSION_CONFLICT',
        'Node version conflict.',
        false,
        { current_resource_version: node.resource_version },
      );
    }

    const body = (await request.json()) as PublicationRequest;
    if (body.group_ids !== undefined) {
      const knownIds = new Set(state.groups.map((g) => g.group_id));
      for (const gid of body.group_ids) {
        if (!knownIds.has(gid)) {
          return makeErrorResponse(
            422,
            'VALIDATION_FAILED',
            `Unknown group_id: ${gid}`,
            false,
            {
              fields: [{ field: '/group_ids', reason: 'unknown_group' }],
            },
          );
        }
      }
      node.group_ids = [...body.group_ids];
      node.member_group_ids = [...body.group_ids];
      for (const grp of state.groups) {
        grp.exit_count = Object.values(state.nodes).filter((n) =>
          (n.group_ids ?? n.member_group_ids ?? []).includes(grp.group_id),
        ).length;
      }
    }
    const nextGen = String(BigInt(node.config_generation.desired) + 1n);
    const nextVer = String(BigInt(node.resource_version) + 1n);
    node.publication_intent = body.publication_intent;
    node.config_generation.desired = nextGen;
    node.resource_version = nextVer;

    if (body.publication_intent === 'published') {
      node.publication_observed = 'applying';
      const op = createTrackedOperation(
        'node.publish',
        'node',
        nodeId,
        [
          {
            name: 'agent_policy',
            status: 'succeeded',
            required: true,
            observed_at: '2026-10-28T15:05:01Z',
            error_code: null,
            message: null,
          },
          {
            name: 'netbird_route_readback',
            status: 'running',
            required: true,
            observed_at: null,
            error_code: null,
            message: null,
          },
          {
            name: 'data_path_probe',
            status: 'pending',
            required: true,
            observed_at: null,
            error_code: null,
            message: null,
          },
        ],
        nextGen,
        node.config_generation.applied ?? '0',
        nextVer,
      );
      return HttpResponse.json(op, {
        status: 202,
        headers: { ETag: `"${nextVer}"` },
      });
    }

    node.publication_observed = 'withdrawn';
    node.member_forwarding_observed = false;
    const isOffline = node.control_status === 'offline';
    const op = createTrackedOperation(
      'node.unpublish',
      'node',
      nodeId,
      [
        {
          name: 'netbird_route_readback',
          status: 'succeeded',
          required: true,
          observed_at: '2026-10-28T15:05:01Z',
          error_code: null,
          message: null,
        },
        {
          name: 'node_allowset_withdraw',
          status: isOffline ? 'failed' : 'running',
          required: true,
          observed_at: isOffline ? '2026-10-28T15:05:02Z' : null,
          error_code: isOffline ? 'DEPENDENCY_UNAVAILABLE' : null,
          message: isOffline ? '中央已撤銷路由；設備撤權尚未確認／受租約限制' : null,
        },
      ],
      nextGen,
      node.config_generation.applied ?? '0',
      nextVer,
    );
    return HttpResponse.json(op, {
      status: 202,
      headers: { ETag: `"${nextVer}"` },
    });
  }),

  http.post('/api/v1/admin/nodes/:nodeId/revoke', async ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const nodeId = String(params.nodeId);
    const node = state.nodes[nodeId];
    if (!node) {
      return makeErrorResponse(404, 'NOT_FOUND', `Node ${nodeId} not found.`);
    }

    const ifMatch = request.headers.get('If-Match');
    if (!ifMatch) {
      return makeErrorResponse(
        428,
        'PRECONDITION_REQUIRED',
        'If-Match header is required when revoking node identity.',
      );
    }
    if (ifMatch !== `"${node.resource_version}"`) {
      return makeErrorResponse(
        412,
        'RESOURCE_VERSION_CONFLICT',
        'Node version conflict.',
        false,
        { current_resource_version: node.resource_version },
      );
    }

    const body = (await request.json()) as ReasonRequest;
    const nextGen = String(BigInt(node.config_generation.desired) + 1n);
    const nextVer = String(BigInt(node.resource_version) + 1n);
    node.lifecycle_state = 'revoked';
    node.publication_intent = 'disabled';
    node.publication_observed = 'withdrawn';
    node.member_forwarding_observed = false;
    node.config_generation.desired = nextGen;
    node.resource_version = nextVer;
    node.notes = `${node.notes ?? ''} [Revoked: ${body.reason}]`.trim();

    const op = createTrackedOperation(
      'node.revoke',
      'node',
      nodeId,
      [
        {
          name: 'cert_and_peer_revoke',
          status: 'succeeded',
          required: true,
          observed_at: '2026-10-28T15:05:01Z',
          error_code: null,
          message: null,
        },
        {
          name: 'netbird_route_readback',
          status: 'succeeded',
          required: true,
          observed_at: '2026-10-28T15:05:02Z',
          error_code: null,
          message: null,
        },
      ],
      nextGen,
      node.config_generation.applied ?? '0',
      nextVer,
    );
    return HttpResponse.json(op, {
      status: 202,
      headers: { ETag: `"${nextVer}"` },
    });
  }),

  // --- Admin Enrollments (§14.5, §18.2) ---
  http.post('/api/v1/admin/enrollments', async ({ request }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const idemKey = request.headers.get('Idempotency-Key') ?? '';
    const existing = state.idempotencyRecords[`enrollment:${idemKey}`];
    if (existing) {
      const replay = structuredClone(
        CONTRACT_EXAMPLES_USED_BY_MOCKS.EnrollmentCreatedReplay,
      );
      return HttpResponse.json(replay, {
        status: 201,
        headers: { 'Cache-Control': 'no-store' },
      });
    }

    const body = (await request.json()) as EnrollmentCreateRequest;
    const created = structuredClone(CONTRACT_EXAMPLES_USED_BY_MOCKS.EnrollmentCreated);
    created.enrollment.delivery = body.delivery;
    if (body.name !== undefined) {
      created.enrollment.name = body.name;
    }
    if (body.region !== undefined) {
      created.enrollment.region = body.region;
    }
    if (body.member_group_ids !== undefined) {
      created.enrollment.member_group_ids = body.member_group_ids;
    }
    if (body.notes !== undefined) {
      created.enrollment.notes = body.notes;
    }

    // Store record without secret for subsequent GETs
    const storedRecord = structuredClone(created.enrollment);
    state.enrollments[storedRecord.enrollment_id] = storedRecord;
    state.idempotencyRecords[`enrollment:${idemKey}`] = {
      status: 201,
      body: created,
    };

    return HttpResponse.json(created, {
      status: 201,
      headers: {
        'Cache-Control': 'no-store',
      },
    });
  }),

  http.get('/api/v1/admin/enrollments/:enrollmentId', ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const enrollmentId = String(params.enrollmentId);
    const record = state.enrollments[enrollmentId];
    if (!record) {
      return makeErrorResponse(404, 'NOT_FOUND', `Enrollment ${enrollmentId} not found.`);
    }
    return HttpResponse.json(record, {
      headers: {
        'Cache-Control': 'no-store',
      },
    });
  }),

  http.post('/api/v1/admin/enrollments/:enrollmentId/revoke', ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const enrollmentId = String(params.enrollmentId);
    const record = state.enrollments[enrollmentId];
    if (!record) {
      return makeErrorResponse(404, 'NOT_FOUND', `Enrollment ${enrollmentId} not found.`);
    }
    record.state = 'revoked';
    record.revoked_at = '2026-10-28T15:06:00Z';
    return HttpResponse.json(record);
  }),

  // --- Admin Releases & Fleet Updates (§13.12) ---
  http.get('/api/v1/admin/releases', ({ request }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;
    return HttpResponse.json(state.releaseList);
  }),

  http.post('/api/v1/admin/update-batches', async ({ request }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const body = (await request.json()) as UpdateBatchCreateRequest;
    const rel = state.releaseList.items.find((r) => r.release_id === body.release_id);
    if (!rel) {
      return makeErrorResponse(404, 'NOT_FOUND', `Release ${body.release_id} not found.`);
    }
    if (rel.support_state === 'revoked') {
      return makeErrorResponse(
        409,
        'RELEASE_REVOKED',
        'Cannot deploy a revoked release.',
      );
    }

    state.releaseList.fleet.fleet_default_release_id = rel.release_id;
    state.updateBatch.release_id = rel.release_id;
    state.updateBatch.reason = body.reason;
    state.updateBatch.requires_service_restart =
      rel.compatibility.requires_service_restart;
    state.updateBatch.state = 'running';

    const accepted: UpdateBatchAccepted = {
      ...structuredClone(CONTRACT_EXAMPLES_USED_BY_MOCKS.UpdateBatchAccepted),
      batch_id: state.updateBatch.batch_id,
      release_id: rel.release_id,
      target_count: state.updateBatch.target_count,
      fleet_default_release_id: rel.release_id,
    };

    return HttpResponse.json(accepted, { status: 202 });
  }),

  http.get('/api/v1/admin/update-batches/:batchId', ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const batchId = String(params.batchId);
    if (batchId !== state.updateBatch.batch_id) {
      return makeErrorResponse(404, 'NOT_FOUND', `Update batch ${batchId} not found.`);
    }
    return HttpResponse.json(state.updateBatch);
  }),

  http.post('/api/v1/admin/update-batches/:batchId/retry', ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const batchId = String(params.batchId);
    if (batchId !== state.updateBatch.batch_id) {
      return makeErrorResponse(404, 'NOT_FOUND', `Update batch ${batchId} not found.`);
    }

    const op = createTrackedOperation('update_batch.retry', 'update_batch', batchId, [
      {
        name: 'agent_policy',
        status: 'running',
        required: true,
        observed_at: null,
        error_code: null,
        message: null,
      },
    ]);
    return HttpResponse.json(op, { status: 202 });
  }),

  // --- Admin Members & Entitlements (§9.2, §14.2) ---
  http.get('/api/v1/admin/members', ({ request }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const url = new URL(request.url);
    const stateFilter = url.searchParams.get('entitlement_state');
    let items = state.memberList.items;
    if (stateFilter) {
      items = items.filter((m) => m.entitlement.state === stateFilter);
    }
    return HttpResponse.json({
      items,
      next_cursor: null,
    });
  }),

  http.get('/api/v1/admin/members/:memberId', ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const memberId = String(params.memberId);
    if (memberId !== state.memberDetail.member_id) {
      return makeErrorResponse(404, 'NOT_FOUND', `Member ${memberId} not found.`);
    }
    return HttpResponse.json(state.memberDetail, {
      headers: {
        ETag: `"${state.memberDetail.resource_version}"`,
      },
    });
  }),

  http.patch('/api/v1/admin/members/:memberId', async ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const memberId = String(params.memberId);
    if (memberId !== state.memberDetail.member_id) {
      return makeErrorResponse(404, 'NOT_FOUND', `Member ${memberId} not found.`);
    }

    const ifMatch = request.headers.get('If-Match');
    if (!ifMatch) {
      return makeErrorResponse(
        428,
        'PRECONDITION_REQUIRED',
        'If-Match header is required when updating member state.',
      );
    }
    if (ifMatch !== `"${state.memberDetail.resource_version}"`) {
      return makeErrorResponse(
        412,
        'RESOURCE_VERSION_CONFLICT',
        'Member resource version conflict.',
        false,
        { current_resource_version: state.memberDetail.resource_version },
      );
    }

    const body = (await request.json()) as MemberPatchRequest;
    const nextVer = String(BigInt(state.memberDetail.resource_version) + 1n);
    const nextRev = String(BigInt(state.memberDetail.entitlement.revision) + 1n);
    state.memberDetail.resource_version = nextVer;
    state.memberDetail.entitlement.revision = nextRev;

    const suspended = body.admin_suspended ?? state.memberDetail.suspension.suspended;
    const reason = body.suspension_reason ?? state.memberDetail.suspension.reason;
    state.memberDetail.entitlement.admin_suspended = suspended;
    state.memberDetail.entitlement.state = suspended ? 'suspended' : 'active';
    state.memberDetail.suspension = {
      suspended,
      reason: suspended ? reason : null,
      suspended_at: suspended ? '2026-10-28T15:05:00Z' : null,
      suspended_by: suspended ? '6a7d5389-5347-41a4-9cd7-768e07072254' : null,
    };

    const summary = state.memberList.items.find((m) => m.member_id === memberId);
    if (summary) {
      summary.entitlement.admin_suspended = suspended;
      summary.entitlement.state = state.memberDetail.entitlement.state;
      summary.resource_version = nextVer;
    }

    const op = createTrackedOperation(
      suspended ? 'member.suspend' : 'member.unsuspend',
      'member',
      memberId,
      [
        {
          name: 'policy_outbox_sync',
          status: 'running',
          required: true,
          observed_at: null,
          error_code: null,
          message: null,
        },
      ],
      nextRev,
      String(BigInt(nextRev) - 1n),
      nextVer,
    );
    return HttpResponse.json(op, {
      status: 202,
      headers: { ETag: `"${nextVer}"` },
    });
  }),

  http.post('/api/v1/admin/members/:memberId/grants', async ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const memberId = String(params.memberId);
    const body = (await request.json()) as GrantCreateRequest;
    if (
      !body.valid_until ||
      !body.valid_until.endsWith('Z') ||
      Number.isNaN(Date.parse(body.valid_until)) ||
      !body.reason?.trim()
    ) {
      return makeErrorResponse(
        422,
        'GRANT_INVALID_WINDOW',
        'Grant requires a finite UTC RFC 3339 valid_until timestamp and a non-empty reason.',
      );
    }

    state.memberDetail.grants.push({
      grant_id: `3bc6d106-3081-443d-b9f4-${String(state.memberDetail.grants.length + 10).padStart(12, '0')}`,
      member_id: memberId,
      source: body.source,
      valid_from: body.valid_from ?? '2026-10-28T15:05:00Z',
      valid_until: body.valid_until,
      reason: body.reason,
      plan_revision_id: body.plan_revision_id ?? null,
      created_by: '6a7d5389-5347-41a4-9cd7-768e07072254',
      created_at: '2026-10-28T15:05:00Z',
      revoked_at: null,
    });
    if (
      state.memberDetail.entitlement.effective_service_end &&
      body.valid_until > state.memberDetail.entitlement.effective_service_end
    ) {
      state.memberDetail.entitlement.effective_service_end = body.valid_until;
    }

    const op = createTrackedOperation('member.grant', 'member', memberId, [
      {
        name: 'policy_outbox_sync',
        status: 'running',
        required: true,
        observed_at: null,
        error_code: null,
        message: null,
      },
    ]);
    return HttpResponse.json(op, { status: 202 });
  }),

  // --- Admin Policies (§11, §12, §14.2) ---
  http.get('/api/v1/admin/policies', ({ request }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const url = new URL(request.url);
    const typeFilter = url.searchParams.get('type');
    const items = typeFilter
      ? state.policyDefinitions.items.filter((d) => d.type === typeFilter)
      : state.policyDefinitions.items;

    return HttpResponse.json({
      items,
      next_cursor: null,
    });
  }),

  http.post('/api/v1/admin/policies', async ({ request }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const body = (await request.json()) as PolicyRevisionCreate;
    if (body.type === 'domain_rule_set') {
      for (const r of body.payload.rules) {
        const norm = normalizeAndValidateDomain(r.domain);
        if (!norm.valid) {
          return makeErrorResponse(
            422,
            'DOMAIN_RULE_INVALID',
            `Invalid domain rule: ${r.domain}`,
          );
        }
      }
    }
    if (body.type === 'threat_feed') {
      const interval = body.payload.update_interval_seconds;
      if (
        typeof interval !== 'number' ||
        !Number.isInteger(interval) ||
        interval < 300 ||
        interval > 604800
      ) {
        return makeErrorResponse(
          422,
          'VALIDATION_FAILED',
          'payload.update_interval_seconds must be in 300..604800',
          false,
          {
            reason: 'schema_invalid',
            fields: [
              {
                field: '/payload/update_interval_seconds',
                reason: 'schema_invalid',
              },
            ],
          },
        );
      }
    }

    const def = state.policyDefinitions.items.find(
      (d) =>
        (body.definition_id && d.definition_id === body.definition_id) ||
        (body.code && d.type === body.type && d.code === body.code) ||
        d.type === body.type,
    );

    const baseRev = structuredClone(CONTRACT_EXAMPLES_USED_BY_MOCKS.PolicyRevision);
    const nextRevNum = def ? String(BigInt(def.generation) + 1n) : '4';
    const newRevId = `8d3b9d9a-d489-4583-a054-${nextRevNum.padStart(12, '0')}`;
    const createdRev: PolicyRevision = {
      ...baseRev,
      type: body.type,
      revision_id: newRevId,
      definition_id: def?.definition_id ?? baseRev.definition_id,
      revision: nextRevNum,
      schema_version: body.schema_version,
      payload: body.payload,
      reason: body.reason,
      created_at: '2026-10-28T15:05:00Z',
    } as PolicyRevision;

    if (def) {
      def.latest_revision = createdRev;
      def.revisions.unshift({
        revision_id: createdRev.revision_id,
        revision: createdRev.revision,
        schema_version: createdRev.schema_version,
        payload_sha256: createdRev.payload_sha256,
        reason: createdRev.reason,
        created_by: createdRev.created_by,
        created_at: createdRev.created_at,
      });
    }

    return HttpResponse.json(createdRev, { status: 201 });
  }),

  http.post(
    '/api/v1/admin/policies/:definitionId/activate',
    async ({ request, params }) => {
      const guard = checkSessionGuard(request, true);
      if (guard) return guard;

      const definitionId = String(params.definitionId);
      const def = state.policyDefinitions.items.find(
        (d) => d.definition_id === definitionId,
      );
      if (!def) {
        return makeErrorResponse(
          404,
          'NOT_FOUND',
          `Policy definition ${definitionId} not found.`,
        );
      }

      const ifMatch = request.headers.get('If-Match');
      if (!ifMatch) {
        return makeErrorResponse(
          428,
          'PRECONDITION_REQUIRED',
          'If-Match header is required when activating a policy revision.',
        );
      }
      if (ifMatch !== `"${def.resource_version}"`) {
        return makeErrorResponse(
          412,
          'RESOURCE_VERSION_CONFLICT',
          'Policy definition version conflict.',
          false,
          { current_resource_version: def.resource_version },
        );
      }

      const body = (await request.json()) as PolicyActivateRequest;
      const nextGen = String(BigInt(def.generation) + 1n);
      const nextVer = String(BigInt(def.resource_version) + 1n);
      def.active_revision_id = body.revision_id;
      if (def.latest_revision && def.latest_revision.revision_id === body.revision_id) {
        def.active_revision = structuredClone(def.latest_revision);
      } else if (def.active_revision) {
        const summary = def.revisions.find((r) => r.revision_id === body.revision_id);
        if (summary) {
          def.active_revision.revision_id = summary.revision_id;
          def.active_revision.revision = summary.revision;
        }
      }
      def.generation = nextGen;
      def.resource_version = nextVer;

      const steps: OperationResource['steps'] =
        def.type === 'domain_rule_set' || def.type === 'dns_profile'
          ? [
              {
                name: 'resolver_1_smoke_apply',
                status: 'succeeded',
                required: true,
                observed_at: '2026-10-28T15:05:01Z',
                error_code: null,
                message: null,
              },
              {
                name: 'resolver_2_smoke_apply',
                status: 'running',
                required: true,
                observed_at: null,
                error_code: null,
                message: null,
              },
            ]
          : [
              {
                name: 'policy_outbox_sync',
                status: 'running',
                required: true,
                observed_at: null,
                error_code: null,
                message: null,
              },
            ];

      const op = createTrackedOperation(
        'policy.activate',
        'policy_definition',
        definitionId,
        steps,
        nextGen,
        String(BigInt(nextGen) - 1n),
        nextVer,
      );
      return HttpResponse.json(op, {
        status: 202,
        headers: { ETag: `"${nextVer}"` },
      });
    },
  ),

  // --- Admin Operations & Audit ---
  http.get('/api/v1/admin/operations/:operationId', ({ request, params }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const opId = String(params.operationId);
    const op = advanceAndGetOperation(opId);
    if (!op) {
      return makeErrorResponse(404, 'NOT_FOUND', `Operation ${opId} not found.`);
    }
    return HttpResponse.json(op);
  }),

  http.get('/api/v1/member/operations/:operationId', ({ request, params }) => {
    const guard = checkSessionGuard(request, false);
    if (guard) return guard;

    const opId = String(params.operationId);
    const op = advanceAndGetOperation(opId);
    if (!op) {
      return makeErrorResponse(404, 'NOT_FOUND', `Operation ${opId} not found.`);
    }
    return HttpResponse.json(op);
  }),

  http.get('/api/v1/admin/audit-events', ({ request }) => {
    const guard = checkSessionGuard(request, true);
    if (guard) return guard;

    const url = new URL(request.url);
    const resourceType = url.searchParams.get('resource_type');
    let items = state.auditEvents;
    if (resourceType) {
      items = items.filter((e) => e.resource.type === resourceType);
    }
    return HttpResponse.json({
      items,
      next_cursor: null,
    });
  }),

  // --- Member Endpoints (§14.3, §18.4) ---
  http.get('/api/v1/member/me', ({ request }) => {
    const guard = checkSessionGuard(request, false);
    if (guard) return guard;
    return HttpResponse.json(buildMemberSelfForPersona(state.persona));
  }),

  http.get('/api/v1/member/exits', ({ request }) => {
    const guard = checkSessionGuard(request, false);
    if (guard) return guard;
    return HttpResponse.json(state.memberExits);
  }),

  http.get('/api/v1/member/devices', ({ request }) => {
    const guard = checkSessionGuard(request, false);
    if (guard) return guard;
    return HttpResponse.json(state.memberDevices);
  }),

  http.delete('/api/v1/member/devices/:deviceId', ({ request, params }) => {
    const guard = checkSessionGuard(request, false);
    if (guard) return guard;

    const deviceId = String(params.deviceId);
    const dev = state.memberDevices.items.find((d) => d.device_id === deviceId);
    if (!dev) {
      return makeErrorResponse(404, 'NOT_FOUND', `Device ${deviceId} not found.`);
    }
    dev.status = 'revoked';
    dev.revoked_at = '2026-10-28T15:05:00Z';
    state.memberDevices.authorized_count = state.memberDevices.items.filter(
      (d) => d.status === 'authorized',
    ).length;

    const op = createTrackedOperation('member.device_revoke', 'member_device', deviceId, [
      {
        name: 'cert_and_peer_revoke',
        status: 'running',
        required: true,
        observed_at: null,
        error_code: null,
        message: null,
      },
    ]);
    return HttpResponse.json(op, { status: 202 });
  }),

  http.get('/api/v1/member/usage', ({ request }) => {
    const guard = checkSessionGuard(request, false);
    if (guard) return guard;
    return HttpResponse.json(buildMemberUsageForPersona(state.persona));
  }),

  http.post('/api/v1/member/checkout-sessions', ({ request }) => {
    const guard = checkSessionGuard(request, false);
    if (guard) return guard;
    return HttpResponse.json(
      structuredClone(CONTRACT_EXAMPLES_USED_BY_MOCKS.CheckoutSession),
      { status: 201 },
    );
  }),

  http.post('/api/v1/member/billing-portal-sessions', ({ request }) => {
    const guard = checkSessionGuard(request, false);
    if (guard) return guard;
    return HttpResponse.json(
      structuredClone(CONTRACT_EXAMPLES_USED_BY_MOCKS.BillingPortalSession),
      { status: 201 },
    );
  }),
];
