/**
 * Central HTTP API client (`/api/v1`) (§14.1, §14.5, §14.6, §14.8, §18.3, AGENTS.md §4).
 *
 * Responsibilities:
 * - Uses `/api/v1` base URL with cookie session credentials (`credentials: 'include'`).
 * - Captures `X-Request-ID`, `ETag`, `Retry-After`, `Location`, `Cache-Control` from responses.
 * - Generates and caches `Idempotency-Key` per user action ID so retries of the same action
 *   reuse the exact same key, while new user actions get a fresh key.
 * - Captures `ETag` per resource path and attaches `If-Match` on conditional updates.
 *   Treats missing `If-Match` / HTTP 428 as a client bug (`PRECONDITION_REQUIRED`),
 *   and HTTP 412 (`RESOURCE_VERSION_CONFLICT`) as a conflict that triggers reload without overwrite.
 * - Notifies auth listeners on HTTP 401 (`UNAUTHENTICATED`) and HTTP 403 (`FORBIDDEN`).
 */

import { ApiError, parseApiErrorResponse, parseRetryAfterSeconds } from './errors';
import type {
  AdminOverviewSummary,
  AuditEventItem,
  BillingPortalSessionResponse,
  CheckoutSessionResponse,
  CreateEnrollmentRequest,
  CreateMemberGrantRequest,
  CreateUpdateBatchRequest,
  DnsPolicyState,
  DnsRule,
  EnrollmentCreatedResponse,
  EnrollmentDetail,
  FairUsePolicyDraft,
  FairUsePolicyState,
  FleetReleasesResponse,
  MemberDeviceItem,
  MemberExitItem,
  MemberItem,
  MemberMeResponse,
  NodeItem,
  Operation,
  OperationStatus,
  PatchMemberRequest,
  SessionInfo,
  UpdateBatchDetail,
  UpdateNodePatchRequest,
  UpdateNodePublicationRequest,
} from './types';

export const API_BASE_URL = '/api/v1';

export interface ApiResponse<T> {
  status: number;
  data: T;
  requestId: string;
  etag: string | null;
  retryAfterSeconds: number | null;
  location: string | null;
  cacheControl: string | null;
}

export interface OperationResponse {
  status: 202;
  operation: Operation;
  requestId: string;
  retryAfterSeconds: number;
  location: string | null;
  isTerminal: boolean;
}

export function isOperationTerminal(status: OperationStatus): boolean {
  return (
    status === 'succeeded' ||
    status === 'partially_succeeded' ||
    status === 'failed' ||
    status === 'superseded'
  );
}

function generateUuidV4(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

class IdempotencyKeyStore {
  private readonly keysByActionId = new Map<string, string>();

  public getOrCreate(actionId: string): string {
    const existing = this.keysByActionId.get(actionId);
    if (existing) {
      return existing;
    }
    const fresh = generateUuidV4();
    this.keysByActionId.set(actionId, fresh);
    return fresh;
  }

  public release(actionId: string): void {
    this.keysByActionId.delete(actionId);
  }

  public clearAll(): void {
    this.keysByActionId.clear();
  }
}

class ETagCache {
  private readonly etagsByPath = new Map<string, string>();

  public set(path: string, etag: string): void {
    this.etagsByPath.set(path, etag);
  }

  public get(path: string): string | undefined {
    return this.etagsByPath.get(path);
  }

  public clear(): void {
    this.etagsByPath.clear();
  }
}

export const idempotencyKeys = new IdempotencyKeyStore();
export const etagCache = new ETagCache();

type AuthErrorListener = (error: ApiError) => void;
type ConflictListener = (error: ApiError, resourcePath: string) => void;

const unauthorizedListeners = new Set<AuthErrorListener>();
const forbiddenListeners = new Set<AuthErrorListener>();
const conflictListeners = new Set<ConflictListener>();

export function onUnauthorizedError(listener: AuthErrorListener): () => void {
  unauthorizedListeners.add(listener);
  return () => {
    unauthorizedListeners.delete(listener);
  };
}

export function onForbiddenError(listener: AuthErrorListener): () => void {
  forbiddenListeners.add(listener);
  return () => {
    forbiddenListeners.delete(listener);
  };
}

export function onConflictError(listener: ConflictListener): () => void {
  conflictListeners.add(listener);
  return () => {
    conflictListeners.delete(listener);
  };
}

export interface ApiRequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
  /**
   * Logical user action identifier. When provided on a side-effecting request,
   * the same `Idempotency-Key` is attached across retries of the same `actionId`.
   */
  actionId?: string;
  /**
   * Explicit `Idempotency-Key` override.
   */
  idempotencyKey?: string;
  /**
   * Whether this request updates a mutable resource and requires `If-Match`.
   */
  requireIfMatch?: boolean;
  /**
   * Explicit `ETag` value for `If-Match`. Falls back to `etagCache.get(etagResourcePath ?? path)`.
   */
  ifMatch?: string;
  etagResourcePath?: string;
  headers?: Record<string, string>;
  signal?: AbortSignal;
}

export async function apiRequest<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<ApiResponse<T>> {
  const method = options.method ?? 'GET';
  const url = path.startsWith('http') ? path : `${API_BASE_URL}${path}`;
  const reqHeaders = new Headers(options.headers);

  if (options.body !== undefined) {
    reqHeaders.set('Content-Type', 'application/json');
  }

  // Attach Idempotency-Key for side-effecting POST/PUT/DELETE/PATCH when actionId or idempotencyKey is given
  if (options.idempotencyKey) {
    reqHeaders.set('Idempotency-Key', options.idempotencyKey);
  } else if (options.actionId) {
    reqHeaders.set('Idempotency-Key', idempotencyKeys.getOrCreate(options.actionId));
  } else if (method === 'POST') {
    reqHeaders.set('Idempotency-Key', generateUuidV4());
  }

  const resourceKey = options.etagResourcePath ?? path;
  if (options.requireIfMatch) {
    const etagToUse = options.ifMatch ?? etagCache.get(resourceKey);
    if (!etagToUse) {
      throw new ApiError({
        status: 428,
        code: 'PRECONDITION_REQUIRED',
        message: 'Missing If-Match ETag precondition on mutable resource update',
        request_id: 'client-precondition-check',
        retryable: false,
      });
    }
    reqHeaders.set('If-Match', etagToUse);
  } else if (options.ifMatch) {
    reqHeaders.set('If-Match', options.ifMatch);
  }

  const response = await fetch(url, {
    method,
    headers: reqHeaders,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    credentials: 'include',
    signal: options.signal,
  });

  const requestId = response.headers.get('X-Request-ID') ?? 'req-unknown';
  const etag = response.headers.get('ETag');
  const retryAfterSeconds = parseRetryAfterSeconds(response.headers.get('Retry-After'));
  const location = response.headers.get('Location');
  const cacheControl = response.headers.get('Cache-Control');

  let parsedBody: unknown = null;
  const contentType = response.headers.get('Content-Type') ?? '';
  if (response.status !== 204 && contentType.includes('application/json')) {
    parsedBody = await response.json();
  }

  if (!response.ok) {
    const apiErr = parseApiErrorResponse(response.status, response.headers, parsedBody);
    if (apiErr.isSessionExpired) {
      unauthorizedListeners.forEach((listener) => listener(apiErr));
    } else if (apiErr.isForbidden) {
      forbiddenListeners.forEach((listener) => listener(apiErr));
    } else if (apiErr.isConflict) {
      conflictListeners.forEach((listener) => listener(apiErr, resourceKey));
    }
    throw apiErr;
  }

  if (etag) {
    etagCache.set(resourceKey, etag);
  } else if (
    typeof parsedBody === 'object' &&
    parsedBody !== null &&
    'etag' in parsedBody &&
    typeof (parsedBody as { etag: unknown }).etag === 'string'
  ) {
    etagCache.set(resourceKey, (parsedBody as { etag: string }).etag);
  }

  return {
    status: response.status,
    data: parsedBody as T,
    requestId,
    etag:
      etag ??
      (typeof parsedBody === 'object' &&
      parsedBody !== null &&
      'etag' in parsedBody &&
      typeof (parsedBody as { etag: unknown }).etag === 'string'
        ? (parsedBody as { etag: string }).etag
        : null),
    retryAfterSeconds,
    location,
    cacheControl,
  };
}

export function toOperationResponse(res: ApiResponse<Operation>): OperationResponse {
  return {
    status: 202,
    operation: res.data,
    requestId: res.requestId,
    retryAfterSeconds: res.retryAfterSeconds ?? 1,
    location: res.location,
    isTerminal: isOperationTerminal(res.data.status),
  };
}

/**
 * Typed domain endpoints for the Portal.
 */
export const portalApi = {
  // Session
  getSession: () => apiRequest<SessionInfo>('/session'),

  // Admin Overview & Nodes
  getAdminOverview: () => apiRequest<AdminOverviewSummary>('/admin/overview'),
  getAdminNodes: () => apiRequest<{ items: NodeItem[] }>('/admin/nodes'),
  getAdminNode: (nodeId: string) => apiRequest<NodeItem>(`/admin/nodes/${nodeId}`),
  patchAdminNode: (
    nodeId: string,
    payload: UpdateNodePatchRequest,
    options: { ifMatch?: string; actionId?: string } = {},
  ) =>
    apiRequest<Operation | NodeItem>(`/admin/nodes/${nodeId}`, {
      method: 'PATCH',
      body: payload,
      requireIfMatch: true,
      ifMatch: options.ifMatch,
      etagResourcePath: `/admin/nodes/${nodeId}`,
      actionId: options.actionId,
    }),
  putNodePublication: async (
    nodeId: string,
    payload: UpdateNodePublicationRequest,
    actionId?: string,
  ): Promise<OperationResponse> => {
    const res = await apiRequest<Operation>(`/admin/nodes/${nodeId}/publication`, {
      method: 'PUT',
      body: payload,
      actionId,
    });
    return toOperationResponse(res);
  },
  revokeNode: async (
    nodeId: string,
    reason: string,
    actionId?: string,
  ): Promise<OperationResponse> => {
    const res = await apiRequest<Operation>(`/admin/nodes/${nodeId}/revoke`, {
      method: 'POST',
      body: { reason },
      actionId,
    });
    return toOperationResponse(res);
  },

  // Admin Enrollments (§18.2)
  createEnrollment: (payload: CreateEnrollmentRequest, actionId?: string) =>
    apiRequest<EnrollmentCreatedResponse>('/admin/enrollments', {
      method: 'POST',
      body: payload,
      actionId,
    }),
  getEnrollment: (enrollmentId: string) =>
    apiRequest<EnrollmentDetail>(`/admin/enrollments/${enrollmentId}`),
  revokeEnrollment: (enrollmentId: string, actionId?: string) =>
    apiRequest<EnrollmentDetail>(`/admin/enrollments/${enrollmentId}/revoke`, {
      method: 'POST',
      actionId,
    }),

  // Admin Fleet Updates (§13.10, §13.12)
  getReleasesAndBatch: () => apiRequest<FleetReleasesResponse>('/admin/releases'),
  createUpdateBatch: async (
    payload: CreateUpdateBatchRequest,
    actionId?: string,
  ): Promise<ApiResponse<UpdateBatchDetail>> =>
    apiRequest<UpdateBatchDetail>('/admin/update-batches', {
      method: 'POST',
      body: payload,
      actionId,
    }),
  getUpdateBatch: (batchId: string) =>
    apiRequest<UpdateBatchDetail>(`/admin/update-batches/${batchId}`),
  retryUpdateBatch: async (
    batchId: string,
    actionId?: string,
  ): Promise<OperationResponse> => {
    const res = await apiRequest<Operation>(`/admin/update-batches/${batchId}/retry`, {
      method: 'POST',
      actionId,
    });
    return toOperationResponse(res);
  },

  // Admin Members (§9.2, §18.1)
  getAdminMembers: () => apiRequest<{ items: MemberItem[] }>('/admin/members'),
  getAdminMember: (memberId: string) =>
    apiRequest<MemberItem>(`/admin/members/${memberId}`),
  patchAdminMember: async (
    memberId: string,
    payload: PatchMemberRequest,
    options: { ifMatch?: string; actionId?: string } = {},
  ): Promise<OperationResponse> => {
    const res = await apiRequest<Operation>(`/admin/members/${memberId}`, {
      method: 'PATCH',
      body: payload,
      requireIfMatch: true,
      ifMatch: options.ifMatch,
      etagResourcePath: `/admin/members/${memberId}`,
      actionId: options.actionId,
    });
    return toOperationResponse(res);
  },
  createMemberGrant: async (
    memberId: string,
    payload: CreateMemberGrantRequest,
    actionId?: string,
  ): Promise<OperationResponse> => {
    const res = await apiRequest<Operation>(`/admin/members/${memberId}/grants`, {
      method: 'POST',
      body: payload,
      actionId,
    });
    return toOperationResponse(res);
  },
  revokeMemberDeviceAsAdmin: async (
    memberId: string,
    deviceId: string,
    actionId?: string,
  ): Promise<OperationResponse> => {
    const res = await apiRequest<Operation>(
      `/admin/members/${memberId}/devices/${deviceId}/revoke`,
      {
        method: 'POST',
        actionId,
      },
    );
    return toOperationResponse(res);
  },

  // Admin Policies: FUP & DNS (§11.1, §12.4, §12.5)
  getFupPolicy: () => apiRequest<FairUsePolicyState>('/admin/policies/fair_use'),
  publishFupPolicy: async (
    draft: FairUsePolicyDraft,
    options: { ifMatch?: string; actionId?: string } = {},
  ): Promise<OperationResponse> => {
    const res = await apiRequest<Operation>('/admin/policies/fair_use/activate', {
      method: 'POST',
      body: draft,
      requireIfMatch: true,
      ifMatch: options.ifMatch,
      etagResourcePath: '/admin/policies/fair_use',
      actionId: options.actionId,
    });
    return toOperationResponse(res);
  },
  getDnsPolicy: () => apiRequest<DnsPolicyState>('/admin/policies/dns'),
  addDnsRule: (
    rule: Omit<DnsRule, 'rule_id' | 'revision'>,
    options: { ifMatch?: string; actionId?: string } = {},
  ) =>
    apiRequest<DnsPolicyState>('/admin/policies/dns/rules', {
      method: 'POST',
      body: rule,
      requireIfMatch: true,
      ifMatch: options.ifMatch,
      etagResourcePath: '/admin/policies/dns',
      actionId: options.actionId,
    }),
  publishDnsPolicy: async (actionId?: string): Promise<OperationResponse> => {
    const res = await apiRequest<Operation>('/admin/policies/dns/activate', {
      method: 'POST',
      actionId,
    });
    return toOperationResponse(res);
  },
  rollbackDnsPolicy: async (actionId?: string): Promise<OperationResponse> => {
    const res = await apiRequest<Operation>('/admin/policies/dns/rollback', {
      method: 'POST',
      actionId,
    });
    return toOperationResponse(res);
  },

  // Admin Audit & Operations
  getAuditEvents: () => apiRequest<{ items: AuditEventItem[] }>('/admin/audit-events'),
  getAdminOperation: (operationId: string) =>
    apiRequest<Operation>(`/admin/operations/${operationId}`),

  // Member Portal (§14.3, §18.4)
  getMemberMe: () => apiRequest<MemberMeResponse>('/member/me'),
  getMemberDevices: () => apiRequest<{ items: MemberDeviceItem[] }>('/member/devices'),
  deleteMemberDevice: async (
    deviceId: string,
    actionId?: string,
  ): Promise<OperationResponse> => {
    const res = await apiRequest<Operation>(`/member/devices/${deviceId}`, {
      method: 'DELETE',
      actionId,
    });
    return toOperationResponse(res);
  },
  getMemberExits: () => apiRequest<{ items: MemberExitItem[] }>('/member/exits'),
  getMemberUsage: () => apiRequest<MemberItem>('/member/usage'),
  createCheckoutSession: (planCode: string, actionId?: string) =>
    apiRequest<CheckoutSessionResponse>('/member/checkout-sessions', {
      method: 'POST',
      body: { plan_code: planCode },
      actionId,
    }),
  createBillingPortalSession: (actionId?: string) =>
    apiRequest<BillingPortalSessionResponse>('/member/billing-portal-sessions', {
      method: 'POST',
      actionId,
    }),
  getMemberOperation: (operationId: string) =>
    apiRequest<Operation>(`/member/operations/${operationId}`),
};
