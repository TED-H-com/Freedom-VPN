/**
 * Typed API Error parsing and user-facing message mapping (§13.11, §14.6, §14.8, AGENTS.md §4).
 * Never exposes stack traces, SQL, certificates, or secrets.
 */

import type { ApiErrorCode, ApiErrorEnvelope, ApiErrorPayload } from './types';

export class ApiError extends Error {
  public readonly status: number;
  public readonly code: ApiErrorCode;
  public readonly request_id: string;
  public readonly retryable: boolean;
  public readonly details?: Record<string, unknown>;
  public readonly retryAfterSeconds: number | null;

  constructor(params: {
    status: number;
    code: ApiErrorCode;
    message: string;
    request_id: string;
    retryable: boolean;
    details?: Record<string, unknown>;
    retryAfterSeconds?: number | null;
  }) {
    super(params.message);
    this.name = 'ApiError';
    this.status = params.status;
    this.code = params.code;
    this.request_id = params.request_id;
    this.retryable = params.retryable;
    this.details = params.details;
    this.retryAfterSeconds = params.retryAfterSeconds ?? null;
  }

  public get isConflict(): boolean {
    return this.status === 412 || this.code === 'RESOURCE_VERSION_CONFLICT';
  }

  public get isPreconditionBug(): boolean {
    return this.status === 428 || this.code === 'PRECONDITION_REQUIRED';
  }

  public get isSessionExpired(): boolean {
    return this.status === 401 || this.code === 'UNAUTHENTICATED';
  }

  public get isForbidden(): boolean {
    return this.status === 403 || this.code === 'FORBIDDEN';
  }

  public get operationId(): string | undefined {
    const opId = this.details?.['operation_id'];
    return typeof opId === 'string' ? opId : undefined;
  }

  public get currentResourceVersion(): string | undefined {
    const ver = this.details?.['current_resource_version'];
    return typeof ver === 'string' ? ver : undefined;
  }
}

function isErrorEnvelope(val: unknown): val is ApiErrorEnvelope {
  if (typeof val !== 'object' || val === null || !('error' in val)) {
    return false;
  }
  const errObj = (val as { error: unknown }).error;
  if (typeof errObj !== 'object' || errObj === null) {
    return false;
  }
  const payload = errObj as Partial<ApiErrorPayload>;
  return typeof payload.code === 'string' && typeof payload.message === 'string';
}

export function parseRetryAfterSeconds(headerVal: string | null): number | null {
  if (!headerVal) return null;
  const parsed = Number.parseInt(headerVal.trim(), 10);
  if (Number.isFinite(parsed) && parsed >= 0) {
    return parsed;
  }
  return null;
}

export function parseApiErrorResponse(
  status: number,
  headers: Headers,
  body: unknown,
): ApiError {
  const headerReqId = headers.get('X-Request-ID') ?? 'req-unknown';
  const retryAfterSeconds = parseRetryAfterSeconds(headers.get('Retry-After'));

  if (isErrorEnvelope(body)) {
    const { code, message, request_id, retryable, details } = body.error;
    return new ApiError({
      status,
      code,
      message,
      request_id: request_id || headerReqId,
      retryable: Boolean(retryable),
      details,
      retryAfterSeconds,
    });
  }

  const defaultCodeByStatus: Record<number, ApiErrorCode> = {
    400: 'MALFORMED_REQUEST',
    401: 'UNAUTHENTICATED',
    403: 'FORBIDDEN',
    404: 'RESOURCE_NOT_FOUND',
    409: 'STATE_CONFLICT',
    412: 'RESOURCE_VERSION_CONFLICT',
    422: 'VALIDATION_FAILED',
    428: 'PRECONDITION_REQUIRED',
    429: 'RATE_LIMITED',
    503: 'DEPENDENCY_UNAVAILABLE',
  };

  return new ApiError({
    status,
    code: defaultCodeByStatus[status] ?? 'DEPENDENCY_UNAVAILABLE',
    message: `HTTP ${status}`,
    request_id: headerReqId,
    retryable: status === 429 || status === 503,
    retryAfterSeconds,
  });
}

export interface UserFacingErrorInfo {
  title: string;
  description: string;
  code: string;
  requestId: string;
  operationId?: string;
  retryable: boolean;
  isConflict: boolean;
  isPreconditionBug: boolean;
}

const ERROR_MESSAGES_ZH: Record<string, { title: string; description: string }> = {
  MALFORMED_REQUEST: {
    title: '請求格式錯誤',
    description: '送出的欄位或標頭格式不正確，請檢查輸入內容後再試。',
  },
  UNAUTHENTICATED: {
    title: '登入階段已過期',
    description: '您的登入憑證已失效或過期，請重新登入後繼續操作。',
  },
  FORBIDDEN: {
    title: '權限不足',
    description: '您目前的帳號角色無權執行此操作或存取此資源。',
  },
  RESOURCE_NOT_FOUND: {
    title: '找不到指定資源',
    description: '目標資源不存在，或不在您的帳號可存取範圍內。',
  },
  IDEMPOTENCY_CONFLICT: {
    title: '重複請求內容衝突 (409)',
    description:
      '相同的冪等識別碼 (Idempotency-Key) 被用於不同的請求內容，系統已拒絕覆寫。',
  },
  STATE_CONFLICT: {
    title: '資源狀態衝突 (409)',
    description: '目前資源狀態不允許此變更，請先重新讀取最新狀態。',
  },
  RESOURCE_VERSION_CONFLICT: {
    title: '設定版本衝突 (412)',
    description:
      '此資源剛被其他管理員或背景程序更新，請重新載入最新資料後再決定是否修改，避免覆蓋他人變更。',
  },
  PRECONDITION_REQUIRED: {
    title: '前端程式異常：缺少版本條件 (428)',
    description:
      '更新請求缺少必要的 If-Match ETag 標頭。這屬於前端程式錯誤，系統已阻止寫入。',
  },
  VALIDATION_FAILED: {
    title: '資料驗證未通過 (422)',
    description: '一或多個欄位不符合規格限制，請修正標示的欄位。',
  },
  NODE_NOT_SERVICEABLE: {
    title: '節點未通過服務基線檢查 (422)',
    description: '節點尚未完成配置或隔離檢查未通過，無法上架為出口。',
  },
  RATE_LIMITED: {
    title: '請求過於頻繁 (429)',
    description: '已達速率限制，請稍候依建議時間重試。',
  },
  DEPENDENCY_UNAVAILABLE: {
    title: '服務暫時無法寫入 (503)',
    description: '控制服務或相依元件暫時忙碌，可使用原操作安全重試。',
  },
  RELEASE_NOT_APPROVED: {
    title: '發行版本未核准或已撤銷',
    description: '所選 Release 尚未通過簽章與雙架構檢核，不可作為更新目標。',
  },
  START_FAILED: {
    title: '節點新版本啟動失敗',
    description:
      '候選版本啟動或健康檢查未通過，節點已依規則回復至最後良好版本 (last_good)。',
  },
  HEALTH_CHECK_FAILED: {
    title: '節點健康與隔離檢查失敗',
    description: '資料路徑或 Internet-only 隔離檢查未通過，會員轉送維持關閉。',
  },
};

const ERROR_MESSAGES_EN: Record<string, { title: string; description: string }> = {
  MALFORMED_REQUEST: {
    title: 'Malformed Request',
    description: 'The request payload or headers are invalid. Please review your input.',
  },
  UNAUTHENTICATED: {
    title: 'Session Expired',
    description: 'Your session is no longer valid. Please sign in again to continue.',
  },
  FORBIDDEN: {
    title: 'Access Denied',
    description: 'Your account role does not have permission to access this resource.',
  },
  RESOURCE_NOT_FOUND: {
    title: 'Resource Not Found',
    description:
      'The requested resource does not exist or is not accessible to your account.',
  },
  IDEMPOTENCY_CONFLICT: {
    title: 'Idempotency Key Conflict (409)',
    description:
      'The same Idempotency-Key was sent with a different payload. Overwrite was prevented.',
  },
  STATE_CONFLICT: {
    title: 'Resource State Conflict (409)',
    description:
      'The current state of the resource does not allow this action. Reload the latest state first.',
  },
  RESOURCE_VERSION_CONFLICT: {
    title: 'Edit Conflict Detected (412)',
    description:
      'This resource was modified by another admin or worker. Reload the resource before saving so you do not overwrite recent changes.',
  },
  PRECONDITION_REQUIRED: {
    title: 'Client Bug: Missing Precondition (428)',
    description:
      'A mutable resource update was sent without an If-Match ETag header. This is a client bug.',
  },
  VALIDATION_FAILED: {
    title: 'Validation Failed (422)',
    description:
      'One or more fields failed contract validation. Please fix the highlighted fields.',
  },
  NODE_NOT_SERVICEABLE: {
    title: 'Node Not Serviceable (422)',
    description:
      'Node provisioning or isolation check has not passed; it cannot be published as an exit.',
  },
  RATE_LIMITED: {
    title: 'Rate Limited (429)',
    description:
      'Too many requests. Please wait for the Retry-After window before retrying.',
  },
  DEPENDENCY_UNAVAILABLE: {
    title: 'Service Temporarily Unavailable (503)',
    description:
      'The control service is temporarily unavailable. You can safely retry this action.',
  },
  RELEASE_NOT_APPROVED: {
    title: 'Release Not Approved',
    description:
      'The selected release is not approved or has been revoked and cannot be targeted.',
  },
  START_FAILED: {
    title: 'Candidate Start Failed',
    description:
      'Candidate release failed to start or pass health checks and rolled back to last_good.',
  },
  HEALTH_CHECK_FAILED: {
    title: 'Health & Isolation Check Failed',
    description:
      'Data path or Internet-only isolation check failed; member forwarding stays closed.',
  },
};

export function mapErrorToUserMessage(
  err: unknown,
  locale: 'zh-TW' | 'en' = 'zh-TW',
): UserFacingErrorInfo {
  const dict = locale === 'en' ? ERROR_MESSAGES_EN : ERROR_MESSAGES_ZH;

  if (err instanceof ApiError) {
    const mapped = dict[err.code];
    return {
      title:
        mapped?.title ??
        (locale === 'en' ? `Operation Failed (${err.code})` : `操作失敗 (${err.code})`),
      description: mapped?.description ?? err.message,
      code: err.code,
      requestId: err.request_id,
      operationId: err.operationId,
      retryable: err.retryable,
      isConflict: err.isConflict,
      isPreconditionBug: err.isPreconditionBug,
    };
  }

  return {
    title: locale === 'en' ? 'Network or Unexpected Error' : '網路或非預期錯誤',
    description:
      locale === 'en'
        ? 'Could not reach the control API. Check your connection and retry.'
        : '無法連線至控制服務 API，請檢查網路連線後重試。',
    code: 'NETWORK_ERROR',
    requestId: 'n/a',
    retryable: true,
    isConflict: false,
    isPreconditionBug: false,
  };
}
