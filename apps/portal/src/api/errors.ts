/**
 * Standard API error envelope parsing, redaction, and localized user messaging (§13.11, §14.6, AGENTS.md §4).
 *
 * `ApiErrorCode` is derived directly from `components['schemas']['ErrorCode']` in the generated contract.
 * Never exposes stack traces, SQL fragments, certificates, or tokens in user-visible messages.
 */

import type { ApiErrorCode, ApiErrorEnvelope, FieldError } from './types';

export type { ApiErrorCode, ApiErrorEnvelope, FieldError };

export interface NormalizedApiError {
  status: number;
  code: ApiErrorCode | (string & {});
  userMessage: Record<'zh-TW' | 'en', string>;
  serverMessage: string;
  requestId: string;
  retryable: boolean;
  retryAfterSeconds: number | null;
  details: Record<string, unknown>;
}

const SENSITIVE_PATTERNS: readonly RegExp[] = [
  /BEGIN\s+(?:RSA\s+|EC\s+|OPENSSH\s+)?PRIVATE\s+KEY/i,
  /\bsk_(?:live|test)_[A-Za-z0-9]+/i,
  /\bgh[po]_[A-Za-z0-9]+/i,
  /\bBearer\s+[A-Za-z0-9._~+/-]+=*/i,
  /\bSELECT\s+.+\s+FROM\s+/i,
  /\bINSERT\s+INTO\s+/i,
  /\bUPDATE\s+\S+\s+SET\s+/i,
  /\bgoroutine\s+\d+\s+\[/i,
  /\bat\s+\S+\s+\([^)]+:\d+:\d+\)/i,
  /\bfvpn_enr_[A-Za-z0-9_-]+/i,
];

const ERROR_CODE_MESSAGES: Record<ApiErrorCode, Record<'zh-TW' | 'en', string>> = {
  MALFORMED_REQUEST: {
    'zh-TW': '請求格式無法解析，請檢查輸入內容。',
    en: 'The request body or parameters are malformed.',
  },
  UNAUTHENTICATED: {
    'zh-TW': '尚未登入或登入階段已失效，請重新登入。',
    en: 'You are not signed in or your session has expired. Please sign in again.',
  },
  FORBIDDEN: {
    'zh-TW': '權限不足，您無權執行此操作或存取此頁面。',
    en: 'Access denied. You do not have permission to perform this action.',
  },
  CSRF_TOKEN_INVALID: {
    'zh-TW': '跨站請求保護驗證 (CSRF Token) 失敗，請重新整理頁面後再試。',
    en: 'CSRF token verification failed. Please refresh the page and try again.',
  },
  RESOURCE_NOT_FOUND: {
    'zh-TW': '找不到指定的資源。',
    en: 'The requested resource was not found.',
  },
  NOT_FOUND: {
    'zh-TW': '找不到指定的資源。',
    en: 'The requested resource was not found.',
  },
  METHOD_NOT_ALLOWED: {
    'zh-TW': '不支援的 HTTP 請求方法。',
    en: 'HTTP method not allowed for this endpoint.',
  },
  IDEMPOTENCY_CONFLICT: {
    'zh-TW': '相同的 Idempotency-Key 已被用於不同請求內容，或相同操作正在處理中。',
    en: 'Idempotency conflict: this key is in progress or was used with a different payload.',
  },
  STATE_CONFLICT: {
    'zh-TW': '資源目前狀態不允許此操作，請重新載入最新狀態後再試。',
    en: 'Current resource state conflicts with this action. Reload and try again.',
  },
  RESOURCE_VERSION_CONFLICT: {
    'zh-TW':
      '資源版本已被其他操作更新 (412 Conflict)，請重新載入最新狀態再決定是否重試。',
    en: 'Resource version was modified concurrently (412 Conflict). Reload the latest state before retrying.',
  },
  PAYLOAD_TOO_LARGE: {
    'zh-TW': '請求內容超過允許的大小上限。',
    en: 'Request payload exceeds the maximum allowed size.',
  },
  UNSUPPORTED_MEDIA_TYPE: {
    'zh-TW': '不支援的內容格式，請使用 application/json。',
    en: 'Unsupported Content-Type; application/json is required.',
  },
  VALIDATION_FAILED: {
    'zh-TW': '輸入欄位格式或條件不符合規範，請檢查後重試。',
    en: 'One or more fields failed validation. Please review your input and try again.',
  },
  NODE_NOT_SERVICEABLE: {
    'zh-TW': '節點目前未通過服務可用性或安全檢查，無法提供轉送服務。',
    en: 'Node is currently not serviceable and cannot forward member traffic.',
  },
  PRECONDITION_REQUIRED: {
    'zh-TW': '缺少版本前置條件 (If-Match ETag, 428)，請先重新讀取資源。',
    en: 'Missing precondition header (If-Match ETag, 428). Reload the resource first.',
  },
  RATE_LIMITED: {
    'zh-TW': '請求頻率過高，請依建議等待時間後再試。',
    en: 'Too many requests. Please wait for the suggested retry window before trying again.',
  },
  INTERNAL_ERROR: {
    'zh-TW': '系統發生內部錯誤，請附上請求追蹤碼 (request_id) 聯繫管理員。',
    en: 'An internal error occurred. Please contact support with the request_id.',
  },
  NOT_IMPLEMENTED: {
    'zh-TW': '此操作尚未實作。',
    en: 'This operation is not implemented.',
  },
  DEPENDENCY_UNAVAILABLE: {
    'zh-TW': '外部相依服務暫時無法連線，系統將稍後重試。',
    en: 'An external dependency is temporarily unavailable. Please retry shortly.',
  },
  RELEASE_NOT_APPROVED: {
    'zh-TW': '此發行版本尚未核准或已被撤銷，禁止部署。',
    en: 'This release is not approved or has been revoked and cannot be deployed.',
  },
  UNSUPPORTED_PLATFORM: {
    'zh-TW': '不支援的作業系統或硬體架構。',
    en: 'Unsupported operating system or CPU architecture.',
  },
  CONTROL_API_INCOMPATIBLE: {
    'zh-TW': '節點程式與控制面 API 版本不相容。',
    en: 'Node software is incompatible with the control API major version.',
  },
  SCHEMA_INCOMPATIBLE: {
    'zh-TW': '政策或狀態 Schema 版本不相容。',
    en: 'Policy or local state schema version is incompatible.',
  },
  SIGNATURE_INVALID: {
    'zh-TW': 'Ed25519 簽章驗證失敗，已拒絕套用。',
    en: 'Ed25519 signature verification failed.',
  },
  KEY_UNTRUSTED: {
    'zh-TW': '簽章金鑰不受信任、不在有效期間或用途不符。',
    en: 'Signing key is untrusted, outside its validity window, or has the wrong purpose.',
  },
  HASH_MISMATCH: {
    'zh-TW': 'SHA-256 雜湊驗證不符，已中止操作。',
    en: 'SHA-256 digest mismatch detected; operation aborted.',
  },
  SIZE_MISMATCH: {
    'zh-TW': '套件檔案大小與 Manifest 宣告不符。',
    en: 'Artifact byte size does not match the manifest declaration.',
  },
  ARTIFACT_INVALID: {
    'zh-TW': '更新套件格式或內容不安全，已拒絕安裝。',
    en: 'The release archive is malformed or unsafe; installation was refused.',
  },
  DISK_SPACE_LOW: {
    'zh-TW': '節點磁碟可用空間不足，無法下載並保留最後良好版本 (last_good)。',
    en: 'Insufficient node disk space to stage release while preserving last_good.',
  },
  DOWNLOAD_TIMEOUT: {
    'zh-TW': '下載發行版套件逾時。',
    en: 'Artifact download timed out.',
  },
  STORE_UNAVAILABLE: {
    'zh-TW': '發行版儲存庫或本地儲存無法存取。',
    en: 'Release store or local state store is unavailable.',
  },
  UPDATE_BUSY: {
    'zh-TW': '節點正在執行另一項更新工作。',
    en: 'Node updater is currently busy with another update job.',
  },
  AUTHORIZATION_EXPIRED: {
    'zh-TW': '授權或期望狀態已過期，節點已依 Fail-Closed 停止轉送。',
    en: 'Authorization or desired-state envelope has expired (fail-closed).',
  },
  STALE_GENERATION: {
    'zh-TW': '世代序號 (generation) 過舊，已被拒絕。',
    en: 'Stale generation rejected.',
  },
  IDENTITY_REVOKED: {
    'zh-TW': '此節點機器身分已被永久撤銷，必須使用新啟用碼重新註冊。',
    en: 'This node identity has been permanently revoked and must re-enroll.',
  },
  INCARNATION_MISMATCH: {
    'zh-TW': '節點實例識別 (incarnation_id) 不一致，請重新同步節點狀態。',
    en: 'Node incarnation mismatch detected. Please re-synchronize node state.',
  },
  CHECKPOINT_FAILED: {
    'zh-TW': '更新前最終計量取樣或存檔失敗。',
    en: 'Pre-update final usage sample or checkpoint failed.',
  },
  MIGRATION_FAILED: {
    'zh-TW': '本地狀態資料遷移失敗。',
    en: 'Local state schema migration failed.',
  },
  START_FAILED: {
    'zh-TW': '節點服務啟動失敗。',
    en: 'Node service failed to start during update switch.',
  },
  HEALTH_CHECK_FAILED: {
    'zh-TW': '更新後本地健康檢查未通過。',
    en: 'Post-update local health check failed.',
  },
  MINIMUM_VERSION_VIOLATION: {
    'zh-TW': '節點版本低於最低允許服務版本 (minimum_serving_version)。',
    en: 'Node release is not in the minimum allowed serving version set.',
  },
  ROLLBACK_FAILED: {
    'zh-TW': '自動回復至 last_good 失敗，節點已停止轉送並等待人工修復 (Fail-Closed)。',
    en: 'Automatic rollback to last_good failed; node stopped forwarding (fail-closed).',
  },
  ACK_CONFLICT: {
    'zh-TW': '回報雜湊與既有紀錄衝突 (ACK_CONFLICT)。',
    en: 'Result acknowledgment conflicts with an existing report hash.',
  },
  USAGE_EVENT_CONFLICT: {
    'zh-TW': '用量事件序列或雜湊與帳本紀錄衝突。',
    en: 'Usage event sequence or payload hash conflicts with the ledger.',
  },
  LEDGER_EPOCH_MISMATCH: {
    'zh-TW': '節點用量帳本世代 (ledger_epoch) 不一致，需管理員執行異常復原。',
    en: 'Node usage ledger epoch mismatch; administrator recovery is required.',
  },
  REPLAY_UNAVAILABLE: {
    'zh-TW': '節點本地重送緩衝區已無此區間事件。',
    en: 'Requested usage sequence range is no longer available for replay.',
  },
  METER_PATH_INVALID: {
    'zh-TW': '節點計量鏈路異常，已依 Fail-Closed 停止會員轉送。',
    en: 'Node metering path is invalid; member forwarding disabled (fail-closed).',
  },
  ISOLATION_BASELINE_MISSING: {
    'zh-TW': '節點缺少隔離基線，已停止會員轉送。',
    en: 'The isolation baseline is missing; member forwarding is disabled.',
  },
  METER_EPOCH_CHANGE: {
    'zh-TW': '節點計量世代已變更，用量事件序號持續遞增。',
    en: 'The meter epoch changed; usage event sequencing continues.',
  },
  ISOLATION_POLICY_INVALID: {
    'zh-TW': '節點隔離允許或拒絕規則與期望政策不符。',
    en: 'The node allow/deny isolation policy does not match the expected ruleset.',
  },
  GENERATION_MISMATCH: {
    'zh-TW': '核心規則集世代與期望世代不一致。',
    en: 'The kernel ruleset does not match the requested generation.',
  },
  HOOK_PRIORITY_MISMATCH: {
    'zh-TW': '基礎鏈的掛載點 (hook) 或優先順序發生偏移。',
    en: 'A base chain hook or priority has drifted from the expected baseline.',
  },
  COUNTER_ORDER_INVALID: {
    'zh-TW': '規則順序允許流量在租約檢查或裝置計數器前被放行。',
    en: 'An accept rule can bypass the lease check or per-device counters.',
  },
  FLOWTABLE_BYPASS: {
    'zh-TW': 'Flowtable 或硬體卸載可能繞過流量計數器。',
    en: 'A flowtable or offload rule could bypass the accounting counters.',
  },
  DEVICE_CLASS_MISSING: {
    'zh-TW': '裝置綁定缺少對應的流量計數器或分類規則。',
    en: 'A device binding is missing its accounting counter or traffic class.',
  },
  QOS_CLASS_MISSING: {
    'zh-TW': 'HTB 父類別或會員流量控制類別缺失或設定不符。',
    en: 'The HTB parent class or a member QoS class is missing or misconfigured.',
  },
  MARK_SEMANTICS_VIOLATED: {
    'zh-TW': '規則違反封包標記規範，改寫了保留的 skb 或 conntrack mark。',
    en: 'A rule writes to reserved skb mark or conntrack mark ranges.',
  },
  PROBE_UNBOUNDED: {
    'zh-TW': '探測例外集合元素缺少核心逾時設定，可能超過健康檢查期間。',
    en: 'A synthetic-probe set element has no kernel timeout and would outlive the health check.',
  },
  DEVICE_LIMIT_REACHED: {
    'zh-TW': '已達方案允許的裝置上限（預設 3 台），請先撤銷舊裝置後再重試。',
    en: 'Device limit reached for your plan (default 3). Revoke an existing device first.',
  },
  ENROLLMENT_TOKEN_INVALID: {
    'zh-TW': '節點啟用碼無效或已被使用。',
    en: 'The node enrollment token is invalid or has already been consumed.',
  },
  ENROLLMENT_TOKEN_EXPIRED: {
    'zh-TW': '節點啟用碼已過期，請建立新的單次啟用碼。',
    en: 'The node enrollment token has expired. Create a new one-time token.',
  },
  ENROLLMENT_TOKEN_REVOKED: {
    'zh-TW': '此節點啟用碼已被管理員撤銷。',
    en: 'This enrollment token has been revoked by an administrator.',
  },
  ENROLLMENT_KEY_MISMATCH: {
    'zh-TW': '節點啟用重試的公鑰與首次綁定不一致。',
    en: 'Enrollment retry public key does not match the bound key.',
  },
  LEASE_EXPIRED: {
    'zh-TW': '節點服務租約 (serving lease) 已過期，已停止接受會員流量。',
    en: 'Node serving lease has expired; member forwarding has stopped.',
  },
  AWAITING_MEMBER_GRANTS: {
    'zh-TW': '正在等待會員授權。這不是會員可以自行處理的錯誤。',
    en: 'Waiting for member grants. This is not an error the member can act on.',
  },
  BOOTSTRAP_CONSUMED: {
    'zh-TW': '節點啟動憑證已領取，無法再次領取。',
    en: 'The node bootstrap credential has already been consumed.',
  },
  BOOTSTRAP_NOT_READY: {
    'zh-TW': '節點啟動憑證尚未準備完成，請稍後再試。',
    en: 'The node bootstrap credential is not ready. Please retry shortly.',
  },
  BOOTSTRAP_EXPIRED: {
    'zh-TW': '節點啟動憑證的領取期限已過。',
    en: 'The node bootstrap credential delivery deadline has expired.',
  },
  NODE_INCARNATION_ACTIVE: {
    'zh-TW': '節點目前的實例仍有效，請先撤銷再重新註冊。',
    en: 'The current node incarnation must be revoked before re-enrollment.',
  },
  NODE_RETIRED: {
    'zh-TW': '此節點已退役，無法重新註冊。',
    en: 'This node is retired and cannot re-enroll.',
  },
  WEBHOOK_SIGNATURE_INVALID: {
    'zh-TW': '支付供應商 Webhook 簽章驗證失敗。',
    en: 'Payment provider webhook signature verification failed.',
  },
};

const GENERIC_MESSAGES: Record<'zh-TW' | 'en', string> = {
  'zh-TW': '操作無法完成，請稍後再試或提供追蹤碼聯繫支援。',
  en: 'The request could not be completed. Please try again later or contact support with the request ID.',
};

export function containsSensitiveLeak(text: string): boolean {
  return SENSITIVE_PATTERNS.some((pattern) => pattern.test(text));
}

export function sanitizeText(text: string): string {
  if (!text || containsSensitiveLeak(text)) {
    return '[REDACTED]';
  }
  return text;
}

export function sanitizeDetails(
  details: Record<string, unknown> | undefined | null,
): Record<string, unknown> {
  if (!details || typeof details !== 'object') {
    return {};
  }
  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(details)) {
    if (/secret|token|password|private_key|authorization|cookie/i.test(key)) {
      result[key] = '[REDACTED]';
    } else if (typeof value === 'string') {
      result[key] = sanitizeText(value);
    } else if (
      typeof value === 'number' ||
      typeof value === 'boolean' ||
      value === null
    ) {
      result[key] = value;
    } else if (Array.isArray(value)) {
      result[key] = value.map((item) =>
        typeof item === 'string' ? sanitizeText(item) : item,
      );
    }
  }
  return result;
}

export const KNOWN_API_ERROR_CODES = Object.keys(ERROR_CODE_MESSAGES) as ApiErrorCode[];

export function isKnownErrorCode(code: string): code is ApiErrorCode {
  return Object.prototype.hasOwnProperty.call(ERROR_CODE_MESSAGES, code);
}

export function getUserErrorMessage(
  code: string,
  locale: 'zh-TW' | 'en' = 'zh-TW',
): string {
  if (isKnownErrorCode(code)) {
    return ERROR_CODE_MESSAGES[code][locale];
  }
  return GENERIC_MESSAGES[locale];
}

export class ApiError extends Error implements NormalizedApiError {
  readonly status: number;
  readonly code: ApiErrorCode | (string & {});
  readonly userMessage: Record<'zh-TW' | 'en', string>;
  readonly serverMessage: string;
  readonly requestId: string;
  readonly retryable: boolean;
  readonly retryAfterSeconds: number | null;
  readonly details: Record<string, unknown>;

  constructor(init: NormalizedApiError) {
    super(init.userMessage['zh-TW']);
    this.name = 'ApiError';
    this.status = init.status;
    this.code = init.code;
    this.userMessage = init.userMessage;
    this.serverMessage = init.serverMessage;
    this.requestId = init.requestId;
    this.retryable = init.retryable;
    this.retryAfterSeconds = init.retryAfterSeconds;
    this.details = init.details;
  }

  get request_id(): string {
    return this.requestId;
  }

  get currentResourceVersion(): string | null {
    const val = this.details.current_resource_version;
    return typeof val === 'string' ? val : null;
  }

  get fieldErrors(): FieldError[] {
    const raw = this.details.fields;
    if (!Array.isArray(raw)) {
      return [];
    }
    const items: FieldError[] = [];
    for (const item of raw) {
      if (
        item &&
        typeof item === 'object' &&
        typeof (item as { field?: unknown }).field === 'string' &&
        typeof (item as { reason?: unknown }).reason === 'string'
      ) {
        items.push({
          field: sanitizeText((item as { field: string }).field),
          reason: sanitizeText((item as { reason: string }).reason),
        });
      }
    }
    return items;
  }

  messageForLocale(locale: 'zh-TW' | 'en'): string {
    return this.userMessage[locale];
  }
}

export function parseRetryAfterSeconds(headerValue: string | null): number | null {
  if (!headerValue) {
    return null;
  }
  const trimmed = headerValue.trim();
  if (/^\d+$/.test(trimmed)) {
    const seconds = Number.parseInt(trimmed, 10);
    return Number.isFinite(seconds) && seconds >= 0 ? seconds : null;
  }
  const dateMs = Date.parse(trimmed);
  if (!Number.isNaN(dateMs)) {
    const diffSec = Math.ceil((dateMs - Date.now()) / 1000);
    return Math.max(0, diffSec);
  }
  return null;
}

export function parseApiError(
  status: number,
  body: unknown,
  retryAfterHeader: string | null = null,
): ApiError {
  const retryAfterSeconds = parseRetryAfterSeconds(retryAfterHeader);

  if (
    body &&
    typeof body === 'object' &&
    'error' in body &&
    body.error &&
    typeof body.error === 'object'
  ) {
    const errObj = body.error as Record<string, unknown>;
    const rawCode = typeof errObj.code === 'string' ? errObj.code : 'INTERNAL_ERROR';
    const rawMsg = typeof errObj.message === 'string' ? errObj.message : '';
    const requestId =
      typeof errObj.request_id === 'string' && errObj.request_id.trim().length > 0
        ? sanitizeText(errObj.request_id)
        : 'unknown';
    const retryable = typeof errObj.retryable === 'boolean' ? errObj.retryable : false;
    const rawDetails =
      errObj.details && typeof errObj.details === 'object'
        ? (errObj.details as Record<string, unknown>)
        : {};

    const userMessage = isKnownErrorCode(rawCode)
      ? ERROR_CODE_MESSAGES[rawCode]
      : GENERIC_MESSAGES;

    return new ApiError({
      status,
      code: rawCode,
      userMessage,
      serverMessage: sanitizeText(rawMsg),
      requestId,
      retryable,
      retryAfterSeconds,
      details: sanitizeDetails(rawDetails),
    });
  }

  let fallbackCode: ApiErrorCode = 'INTERNAL_ERROR';
  if (status === 401) fallbackCode = 'UNAUTHENTICATED';
  else if (status === 403) fallbackCode = 'FORBIDDEN';
  else if (status === 404) fallbackCode = 'RESOURCE_NOT_FOUND';
  else if (status === 409) fallbackCode = 'STATE_CONFLICT';
  else if (status === 412) fallbackCode = 'RESOURCE_VERSION_CONFLICT';
  else if (status === 422) fallbackCode = 'VALIDATION_FAILED';
  else if (status === 428) fallbackCode = 'PRECONDITION_REQUIRED';
  else if (status === 429) fallbackCode = 'RATE_LIMITED';
  else if (status === 503) fallbackCode = 'DEPENDENCY_UNAVAILABLE';

  return new ApiError({
    status,
    code: fallbackCode,
    userMessage: ERROR_CODE_MESSAGES[fallbackCode],
    serverMessage: '',
    requestId: 'unknown',
    retryable: status === 429 || status === 503,
    retryAfterSeconds,
    details: {},
  });
}
