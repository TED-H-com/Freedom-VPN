/**
 * Operation-typed HTTP API client for `/api/v1` (§14.1, §14.5, §14.6, §14.8, §16.1).
 *
 * Every request is typed against `paths` and `operations` from `@contracts/schema`:
 * - Wrong path, method, path parameter, query parameter, or request body is a compile error.
 * - Attaches `credentials: 'include'` for HttpOnly session cookie auth.
 * - Reads `__Host-fvpn-csrf` from `document.cookie` at request time and attaches `X-CSRF-Token`
 *   on unsafe requests (`POST`, `PATCH`, `PUT`, `DELETE`), never on `GET`.
 * - Generates and reuses `Idempotency-Key` per logical user action across retries.
 * - Captures `ETag` headers and sends `If-Match` on versioned mutations (412 / 428).
 */

import {
  CSRF_HEADER_NAME,
  getAuthSession,
  logoutAuthSession,
  readCsrfToken,
  type AuthSession,
} from './auth';
import { ApiError, parseApiError } from './errors';
import type {
  AuditEventListResponse,
  BillingPortalSessionRequest,
  BillingPortalSessionResponse,
  CheckoutSessionRequest,
  CheckoutSessionResponse,
  EnrollmentCreateRequest,
  EnrollmentCreateResponse,
  EnrollmentRecord,
  GrantCreateRequest,
  MemberDetail,
  MemberDeviceListResponse,
  MemberExitListResponse,
  MemberListResponse,
  MemberPatchRequest,
  MemberSelf,
  MemberUsageResponse,
  NodeDetail,
  NodeListResponse,
  NodePatchRequest,
  OperationResource,
  PolicyActivateRequest,
  PolicyDefinitionListResponse,
  PolicyRevision,
  PolicyRevisionCreate,
  PublicationRequest,
  ReasonRequest,
  ReleaseItem,
  ReleaseListResponse,
  ReleaseRegisterRequest,
  UpdateBatchAccepted,
  UpdateBatchCreateRequest,
  UpdateBatchDetail,
  UpdateBatchRetryRequest,
  operations,
  paths,
} from './types';
import type { WithEtag } from './view-models';

export const DEFAULT_API_BASE_URL = '/api/v1';

export function getApiBaseUrl(override?: string): string {
  const raw = override ?? import.meta.env.VITE_API_BASE_URL ?? DEFAULT_API_BASE_URL;
  const trimmed = raw.trim().replace(/\/+$/, '');
  return trimmed || DEFAULT_API_BASE_URL;
}

export type HttpMethod = 'get' | 'post' | 'put' | 'patch' | 'delete';
const UNSAFE_HTTP_METHODS = new Set<string>(['POST', 'PATCH', 'PUT', 'DELETE']);

export type PathsForMethod<M extends HttpMethod> = {
  [P in keyof paths]: M extends keyof paths[P]
    ? NonNullable<paths[P][M]> extends never
      ? never
      : P
    : never;
}[keyof paths];

export type OperationFor<
  P extends keyof paths,
  M extends HttpMethod,
> = M extends keyof paths[P] ? NonNullable<paths[P][M]> : never;

export type PathParamsOf<Op> = Op extends { parameters: { path: infer PP } }
  ? PP
  : undefined;

export type QueryParamsOf<Op> = Op extends { parameters: { query?: infer QP } }
  ? QP
  : undefined;

export type RequestBodyOf<Op> = Op extends {
  requestBody: { content: { 'application/json': infer B } };
}
  ? B
  : Op extends { requestBody?: { content: { 'application/json': infer B } } }
    ? B | undefined
    : undefined;

type JsonContent<R> = R extends { content: { 'application/json': infer C } } ? C : never;

export type SuccessResponseOf<Op> = Op extends { responses: infer R }
  ? {
      [K in Extract<keyof R, 200 | 201 | 202>]: JsonContent<R[K]>;
    }[Extract<keyof R, 200 | 201 | 202>]
  : never;

export type TypedOperationRequest<
  M extends HttpMethod,
  P extends PathsForMethod<M>,
  Op = OperationFor<P, M>,
> = {
  method: M;
  path: P;
  actionId?: string;
  ifMatch?: string;
  signal?: AbortSignal;
} & (PathParamsOf<Op> extends undefined
  ? { pathParams?: undefined }
  : { pathParams: PathParamsOf<Op> }) &
  (QueryParamsOf<Op> extends undefined
    ? { query?: undefined }
    : { query?: QueryParamsOf<Op> }) &
  (undefined extends RequestBodyOf<Op>
    ? { body?: RequestBodyOf<Op> }
    : { body: RequestBodyOf<Op> });

export function createUuidV4(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  const bytes = new Uint8Array(16);
  if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
    crypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < 16; i += 1) {
      bytes[i] = (i * 37 + 11) & 0xff;
    }
  }
  bytes[6] = ((bytes[6] ?? 0) & 0x0f) | 0x40;
  bytes[8] = ((bytes[8] ?? 0) & 0x3f) | 0x80;
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20, 32)}`;
}

/**
 * Manages Idempotency-Key values keyed by logical action identifier so that
 * retrying the same user action reuses the exact same key (§14.5).
 */
export class IdempotencyKeyStore {
  private readonly keys = new Map<string, string>();

  getOrCreate(actionId: string): string {
    const existing = this.keys.get(actionId);
    if (existing) {
      return existing;
    }
    const created = createUuidV4();
    this.keys.set(actionId, created);
    return created;
  }

  clear(actionId: string): void {
    this.keys.delete(actionId);
  }

  clearAll(): void {
    this.keys.clear();
  }
}

/**
 * Stores observed ETag values per resource path for optimistic concurrency (`If-Match`, §14.5).
 */
export class EtagStore {
  private readonly etags = new Map<string, string>();

  set(resourcePath: string, etag: string | null): void {
    if (etag && etag.trim().length > 0) {
      this.etags.set(resourcePath, etag.trim());
    }
  }

  get(resourcePath: string): string | undefined {
    return this.etags.get(resourcePath);
  }

  clear(): void {
    this.etags.clear();
  }
}

export interface ApiClientConfig {
  baseUrl?: string;
  authBaseUrl?: string;
  fetchImpl?: typeof fetch;
  idempotencyStore?: IdempotencyKeyStore;
  etagStore?: EtagStore;
  onUnauthorized?: (err: ApiError) => void;
}

export interface RawRequestOptions {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  path: string;
  body?: unknown;
  actionId?: string;
  ifMatch?: string;
  etagKey?: string;
  signal?: AbortSignal;
}

function interpolatePath(
  pathTemplate: string,
  pathParams?: Record<string, string>,
): string {
  if (!pathParams) {
    return pathTemplate;
  }
  return pathTemplate.replace(/\{([^}]+)\}/g, (_match, key: string) => {
    const value = pathParams[key];
    if (value === undefined) {
      throw new Error(`Missing required path parameter: ${key}`);
    }
    return encodeURIComponent(String(value));
  });
}

function appendQueryParams(
  resolvedPath: string,
  query?: Record<string, unknown>,
): string {
  if (!query) {
    return resolvedPath;
  }
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined && v !== null && v !== '') {
      params.set(k, String(v));
    }
  }
  const qs = params.toString();
  return qs ? `${resolvedPath}?${qs}` : resolvedPath;
}

function deriveEtagFromData(data: unknown, headerEtag: string | null): string | null {
  if (headerEtag && headerEtag.trim().length > 0) {
    return headerEtag.trim();
  }
  if (data && typeof data === 'object' && 'resource_version' in data) {
    const rv = (data as { resource_version?: unknown }).resource_version;
    if (typeof rv === 'string' && rv.length > 0) {
      return `"${rv}"`;
    }
  }
  return null;
}

export class FreedomVpnApiClient {
  private readonly baseUrl: string;
  private readonly authBaseUrl?: string;
  private readonly fetchImpl: typeof fetch;
  readonly idempotencyStore: IdempotencyKeyStore;
  readonly etagStore: EtagStore;
  private onUnauthorized?: (err: ApiError) => void;

  constructor(config: ApiClientConfig = {}) {
    this.baseUrl = getApiBaseUrl(config.baseUrl);
    this.authBaseUrl = config.authBaseUrl;
    this.fetchImpl = config.fetchImpl ?? ((input, init) => fetch(input, init));
    this.idempotencyStore = config.idempotencyStore ?? new IdempotencyKeyStore();
    this.etagStore = config.etagStore ?? new EtagStore();
    this.onUnauthorized = config.onUnauthorized;
  }

  setUnauthorizedHandler(handler: ((err: ApiError) => void) | undefined): void {
    this.onUnauthorized = handler;
  }

  async requestWithEtag<T>(options: RawRequestOptions): Promise<WithEtag<T>> {
    const headers = new Headers({
      Accept: 'application/json',
    });

    if (options.body !== undefined) {
      headers.set('Content-Type', 'application/json');
    }

    const upperMethod = options.method.toUpperCase();
    const isMutating = UNSAFE_HTTP_METHODS.has(upperMethod);

    if (isMutating) {
      const actionKey = options.actionId ?? `${upperMethod}:${options.path}`;
      const idempotencyKey = this.idempotencyStore.getOrCreate(actionKey);
      headers.set('Idempotency-Key', idempotencyKey);

      const csrfToken = readCsrfToken();
      if (csrfToken) {
        headers.set(CSRF_HEADER_NAME, csrfToken);
      }
    }

    const etagLookupKey = options.etagKey ?? options.path;
    const effectiveIfMatch = options.ifMatch ?? this.etagStore.get(etagLookupKey);
    if (
      effectiveIfMatch &&
      (upperMethod === 'PATCH' ||
        upperMethod === 'PUT' ||
        upperMethod === 'DELETE' ||
        options.ifMatch !== undefined ||
        options.etagKey !== undefined)
    ) {
      headers.set('If-Match', effectiveIfMatch);
    }

    const url = `${this.baseUrl}${options.path}`;
    const response = await this.fetchImpl(url, {
      method: upperMethod,
      credentials: 'include',
      headers,
      body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
      signal: options.signal,
    });

    const rawHeaderEtag = response.headers.get('ETag') ?? response.headers.get('etag');

    let parsedBody: unknown = null;
    const text = await response.text();
    if (text.trim().length > 0) {
      try {
        parsedBody = JSON.parse(text);
      } catch {
        parsedBody = null;
      }
    }

    if (!response.ok) {
      const apiErr = parseApiError(
        response.status,
        parsedBody,
        response.headers.get('Retry-After'),
      );
      if (apiErr.status === 401 && this.onUnauthorized) {
        this.onUnauthorized(apiErr);
      }
      throw apiErr;
    }

    const etag = deriveEtagFromData(parsedBody, rawHeaderEtag);
    if (etag) {
      this.etagStore.set(etagLookupKey, etag);
    }

    return {
      data: parsedBody as T,
      etag,
    };
  }

  async request<T>(options: RawRequestOptions): Promise<T> {
    const res = await this.requestWithEtag<T>(options);
    return res.data;
  }

  /**
   * Executes an OpenAPI operation strictly typed by `paths` and `operations` from `@contracts/schema`.
   * Passing an unknown path, wrong HTTP method, missing path parameter, or invalid request body
   * fails at TypeScript compile time.
   */
  async callOperation<M extends HttpMethod, P extends PathsForMethod<M>>(
    req: TypedOperationRequest<M, P>,
  ): Promise<SuccessResponseOf<OperationFor<P, M>>> {
    const res = await this.callOperationWithEtag(req);
    return res.data;
  }

  async callOperationWithEtag<M extends HttpMethod, P extends PathsForMethod<M>>(
    req: TypedOperationRequest<M, P>,
  ): Promise<WithEtag<SuccessResponseOf<OperationFor<P, M>>>> {
    const resolvedPath = interpolatePath(
      req.path as string,
      req.pathParams as Record<string, string> | undefined,
    );
    const fullPath = appendQueryParams(
      resolvedPath,
      req.query as Record<string, unknown> | undefined,
    );
    return this.requestWithEtag<SuccessResponseOf<OperationFor<P, M>>>({
      method: req.method.toUpperCase() as RawRequestOptions['method'],
      path: fullPath,
      body: req.body,
      actionId: req.actionId,
      ifMatch: req.ifMatch,
      etagKey: resolvedPath,
      signal: req.signal,
    });
  }

  // --- Browser Auth Session Routes (outside /api/v1, defined in ./auth) ---

  getSession(): Promise<AuthSession> {
    return getAuthSession(this.fetchImpl, this.authBaseUrl);
  }

  logout(): Promise<void> {
    return logoutAuthSession(this.fetchImpl, this.authBaseUrl);
  }

  // --- Admin Nodes & Enrollments (typed via @contracts/schema operations) ---

  listNodes(
    query?: operations['listNodes']['parameters']['query'],
  ): Promise<NodeListResponse> {
    return this.callOperation({
      method: 'get',
      path: '/admin/nodes',
      query,
    });
  }

  async getNode(nodeId: string): Promise<WithEtag<NodeDetail>> {
    return this.callOperationWithEtag({
      method: 'get',
      path: '/admin/nodes/{node_id}',
      pathParams: { node_id: nodeId },
    });
  }

  updateNode(
    nodeId: string,
    patch: NodePatchRequest,
    ifMatch?: string,
    actionId?: string,
  ): Promise<WithEtag<NodeDetail | OperationResource>> {
    return this.callOperationWithEtag({
      method: 'patch',
      path: '/admin/nodes/{node_id}',
      pathParams: { node_id: nodeId },
      body: patch,
      ifMatch,
      actionId,
    });
  }

  retireNode(
    nodeId: string,
    ifMatch?: string,
    actionId?: string,
  ): Promise<OperationResource> {
    const etag =
      ifMatch ?? this.etagStore.get(`/admin/nodes/${encodeURIComponent(nodeId)}`);
    return this.callOperation({
      method: 'delete',
      path: '/admin/nodes/{node_id}',
      pathParams: { node_id: nodeId },
      ifMatch: etag,
      actionId,
    });
  }

  setNodePublication(
    nodeId: string,
    body: PublicationRequest,
    ifMatch?: string,
    actionId?: string,
  ): Promise<OperationResource> {
    const etag =
      ifMatch ?? this.etagStore.get(`/admin/nodes/${encodeURIComponent(nodeId)}`);
    return this.callOperation({
      method: 'put',
      path: '/admin/nodes/{node_id}/publication',
      pathParams: { node_id: nodeId },
      body,
      ifMatch: etag,
      actionId,
    });
  }

  publishNode(
    nodeId: string,
    ifMatch?: string,
    actionId?: string,
    reason = 'Publish node to member service',
  ): Promise<OperationResource> {
    return this.setNodePublication(
      nodeId,
      { publication_intent: 'published', reason },
      ifMatch,
      actionId,
    );
  }

  disableNode(
    nodeId: string,
    ifMatch?: string,
    actionId?: string,
    reason = 'Disable node from member service',
  ): Promise<OperationResource> {
    return this.setNodePublication(
      nodeId,
      { publication_intent: 'disabled', reason },
      ifMatch,
      actionId,
    );
  }

  revokeNode(
    nodeId: string,
    reason: string,
    ifMatch?: string,
    actionId?: string,
  ): Promise<OperationResource> {
    const etag =
      ifMatch ?? this.etagStore.get(`/admin/nodes/${encodeURIComponent(nodeId)}`);
    const body: ReasonRequest = { reason };
    return this.callOperation({
      method: 'post',
      path: '/admin/nodes/{node_id}/revoke',
      pathParams: { node_id: nodeId },
      body,
      ifMatch: etag,
      actionId,
    });
  }

  createEnrollment(
    body: EnrollmentCreateRequest,
    actionId?: string,
  ): Promise<EnrollmentCreateResponse> {
    return this.callOperation({
      method: 'post',
      path: '/admin/enrollments',
      body,
      actionId,
    });
  }

  getEnrollment(enrollmentId: string): Promise<EnrollmentRecord> {
    return this.callOperation({
      method: 'get',
      path: '/admin/enrollments/{enrollment_id}',
      pathParams: { enrollment_id: enrollmentId },
    });
  }

  revokeEnrollment(
    enrollmentId: string,
    reason = 'Revoked by administrator',
    actionId?: string,
  ): Promise<EnrollmentRecord> {
    return this.callOperation({
      method: 'post',
      path: '/admin/enrollments/{enrollment_id}/revoke',
      pathParams: { enrollment_id: enrollmentId },
      body: { reason },
      actionId,
    });
  }

  // --- Admin Releases & Fleet Updates ---

  listReleases(
    query?: operations['listReleases']['parameters']['query'],
  ): Promise<ReleaseListResponse> {
    return this.callOperation({
      method: 'get',
      path: '/admin/releases',
      query,
    });
  }

  registerRelease(
    body: ReleaseRegisterRequest,
    actionId?: string,
  ): Promise<ReleaseItem | OperationResource> {
    return this.callOperation({
      method: 'post',
      path: '/admin/releases',
      body,
      actionId,
    });
  }

  createUpdateBatch(
    body: UpdateBatchCreateRequest,
    actionId?: string,
  ): Promise<UpdateBatchAccepted> {
    return this.callOperation({
      method: 'post',
      path: '/admin/update-batches',
      body,
      actionId,
    });
  }

  getUpdateBatch(
    batchId: string,
    query?: operations['getUpdateBatch']['parameters']['query'],
  ): Promise<UpdateBatchDetail> {
    return this.callOperation({
      method: 'get',
      path: '/admin/update-batches/{batch_id}',
      pathParams: { batch_id: batchId },
      query,
    });
  }

  retryUpdateBatch(
    batchId: string,
    body: UpdateBatchRetryRequest = { reason: 'Retry failed targets' },
    actionId?: string,
  ): Promise<OperationResource> {
    return this.callOperation({
      method: 'post',
      path: '/admin/update-batches/{batch_id}/retry',
      pathParams: { batch_id: batchId },
      body,
      actionId,
    });
  }

  // --- Admin Members & Entitlements ---

  listMembers(
    query?: operations['listMembers']['parameters']['query'],
  ): Promise<MemberListResponse> {
    return this.callOperation({
      method: 'get',
      path: '/admin/members',
      query,
    });
  }

  getMember(memberId: string): Promise<WithEtag<MemberDetail>> {
    return this.callOperationWithEtag({
      method: 'get',
      path: '/admin/members/{member_id}',
      pathParams: { member_id: memberId },
    });
  }

  updateMember(
    memberId: string,
    patch: MemberPatchRequest,
    ifMatch?: string,
    actionId?: string,
  ): Promise<WithEtag<MemberDetail | OperationResource>> {
    const etag =
      ifMatch ?? this.etagStore.get(`/admin/members/${encodeURIComponent(memberId)}`);
    return this.callOperationWithEtag({
      method: 'patch',
      path: '/admin/members/{member_id}',
      pathParams: { member_id: memberId },
      body: patch,
      ifMatch: etag,
      actionId,
    });
  }

  suspendMember(
    memberId: string,
    reason: string,
    ifMatch?: string,
    actionId?: string,
  ): Promise<WithEtag<MemberDetail | OperationResource>> {
    return this.updateMember(
      memberId,
      { admin_suspended: true, suspension_reason: reason },
      ifMatch,
      actionId,
    );
  }

  resumeMember(
    memberId: string,
    reason: string,
    ifMatch?: string,
    actionId?: string,
  ): Promise<WithEtag<MemberDetail | OperationResource>> {
    return this.updateMember(
      memberId,
      { admin_suspended: false, suspension_reason: reason },
      ifMatch,
      actionId,
    );
  }

  createMemberGrant(
    memberId: string,
    body: GrantCreateRequest,
    actionId?: string,
  ): Promise<OperationResource> {
    return this.callOperation({
      method: 'post',
      path: '/admin/members/{member_id}/grants',
      pathParams: { member_id: memberId },
      body,
      actionId,
    });
  }

  // --- Admin Policies (Fair Use, DNS, Threat Feeds, Plans, Release Support) ---

  async listPolicies(
    query?: operations['listPolicies']['parameters']['query'],
  ): Promise<PolicyDefinitionListResponse> {
    const res = await this.callOperation({
      method: 'get',
      path: '/admin/policies',
      query,
    });
    for (const def of res.items) {
      this.etagStore.set(
        `/admin/policies/${encodeURIComponent(def.definition_id)}/activate`,
        `"${def.resource_version}"`,
      );
    }
    return res;
  }

  createPolicyRevision(
    body: PolicyRevisionCreate,
    actionId?: string,
  ): Promise<PolicyRevision> {
    return this.callOperation({
      method: 'post',
      path: '/admin/policies',
      body,
      actionId,
    });
  }

  activatePolicyRevision(
    definitionId: string,
    body: PolicyActivateRequest,
    ifMatch?: string,
    actionId?: string,
  ): Promise<OperationResource> {
    const activatePath = `/admin/policies/${encodeURIComponent(definitionId)}/activate`;
    const etag = ifMatch ?? this.etagStore.get(activatePath);
    return this.callOperation({
      method: 'post',
      path: '/admin/policies/{definition_id}/activate',
      pathParams: { definition_id: definitionId },
      body,
      ifMatch: etag,
      actionId,
    });
  }

  // --- Admin Operations & Audit ---

  getOperation(operationId: string): Promise<OperationResource> {
    return this.callOperation({
      method: 'get',
      path: '/admin/operations/{operation_id}',
      pathParams: { operation_id: operationId },
    });
  }

  getMemberOperation(operationId: string): Promise<OperationResource> {
    return this.callOperation({
      method: 'get',
      path: '/member/operations/{operation_id}',
      pathParams: { operation_id: operationId },
    });
  }

  listAuditEvents(
    query?: operations['listAuditEvents']['parameters']['query'],
  ): Promise<AuditEventListResponse> {
    return this.callOperation({
      method: 'get',
      path: '/admin/audit-events',
      query,
    });
  }

  // --- Member Endpoints ---

  getMemberSelf(): Promise<MemberSelf> {
    return this.callOperation({
      method: 'get',
      path: '/member/me',
    });
  }

  listMemberExits(
    query?: operations['listMemberExits']['parameters']['query'],
  ): Promise<MemberExitListResponse> {
    return this.callOperation({
      method: 'get',
      path: '/member/exits',
      query,
    });
  }

  listMemberDevices(
    query?: operations['listMemberDevices']['parameters']['query'],
  ): Promise<MemberDeviceListResponse> {
    return this.callOperation({
      method: 'get',
      path: '/member/devices',
      query,
    });
  }

  revokeMemberDevice(deviceId: string, actionId?: string): Promise<OperationResource> {
    return this.callOperation({
      method: 'delete',
      path: '/member/devices/{device_id}',
      pathParams: { device_id: deviceId },
      actionId,
    });
  }

  getMemberUsage(
    query?: operations['getMemberUsage']['parameters']['query'],
  ): Promise<MemberUsageResponse> {
    return this.callOperation({
      method: 'get',
      path: '/member/usage',
      query,
    });
  }

  createCheckoutSession(
    body: CheckoutSessionRequest = {
      plan_id: 'b4fe029a-ecbf-4b1a-9ee7-de399791ae6e',
      return_url: 'https://portal.example.com/member/billing?checkout=return',
    },
    actionId?: string,
  ): Promise<CheckoutSessionResponse | OperationResource> {
    return this.callOperation({
      method: 'post',
      path: '/member/checkout-sessions',
      body,
      actionId,
    });
  }

  createBillingPortalSession(
    body: BillingPortalSessionRequest = {
      return_url: 'https://portal.example.com/member/billing',
    },
    actionId?: string,
  ): Promise<BillingPortalSessionResponse> {
    return this.callOperation({
      method: 'post',
      path: '/member/billing-portal-sessions',
      body,
      actionId,
    });
  }
}

export const idempotencyKeys = new IdempotencyKeyStore();
export const etagCache = new EtagStore();
export const apiClient = new FreedomVpnApiClient({
  idempotencyStore: idempotencyKeys,
  etagStore: etagCache,
});
