/**
 * Asynchronous 202 operation polling hook (`useOperationPoller`) (§14.5, §14.8, §18.3).
 *
 * Honesty contract:
 * - A 202 Accepted response is NEVER treated as completed.
 * - Polls `/api/v1/admin/operations/{id}` (or `/api/v1/member/operations/{id}`)
 *   respecting the server's `Retry-After` header (in seconds).
 * - Exposes each step's status (`pending`, `running`, `succeeded`, `failed`, etc.)
 *   and only marks `isDone = true` when the operation reaches a terminal state.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { isOperationTerminal, portalApi, type OperationResponse } from './client';
import type { Operation } from './types';

export interface UseOperationPollerOptions {
  scope?: 'admin' | 'member';
  onTerminal?: (op: Operation) => void;
  /**
   * Multiplier applied to `Retry-After` seconds (default 1000 for ms; tests can pass smaller values).
   */
  timeUnitMs?: number;
}

export interface UseOperationPollerState {
  operation: Operation | null;
  requestId: string | null;
  retryAfterSeconds: number;
  isPolling: boolean;
  isDone: boolean;
  isSuccess: boolean;
  isPartialSuccess: boolean;
  isFailed: boolean;
  pollError: unknown;
  startTracking: (initial: OperationResponse) => void;
  reset: () => void;
}

export function useOperationPoller(
  options: UseOperationPollerOptions = {},
): UseOperationPollerState {
  const scope = options.scope ?? 'admin';
  const timeUnitMs = options.timeUnitMs ?? 1000;
  const onTerminalRef = useRef(options.onTerminal);
  onTerminalRef.current = options.onTerminal;

  const [operation, setOperation] = useState<Operation | null>(null);
  const [requestId, setRequestId] = useState<string | null>(null);
  const [retryAfterSeconds, setRetryAfterSeconds] = useState<number>(1);
  const [isPolling, setIsPolling] = useState<boolean>(false);
  const [pollError, setPollError] = useState<unknown>(null);

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearTimer = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const startTracking = useCallback(
    (initial: OperationResponse) => {
      clearTimer();
      setPollError(null);
      setOperation(initial.operation);
      setRequestId(initial.requestId);
      const waitSec = Math.max(1, initial.retryAfterSeconds || 1);
      setRetryAfterSeconds(waitSec);

      if (isOperationTerminal(initial.operation.status)) {
        setIsPolling(false);
        onTerminalRef.current?.(initial.operation);
      } else {
        setIsPolling(true);
      }
    },
    [clearTimer],
  );

  const reset = useCallback(() => {
    clearTimer();
    setOperation(null);
    setRequestId(null);
    setIsPolling(false);
    setPollError(null);
  }, [clearTimer]);

  useEffect(() => {
    if (!isPolling || !operation) {
      return;
    }
    if (isOperationTerminal(operation.status)) {
      setIsPolling(false);
      return;
    }

    const opId = operation.operation_id;
    const delayMs = Math.max(50, retryAfterSeconds * timeUnitMs);

    timerRef.current = setTimeout(async () => {
      try {
        const res =
          scope === 'member'
            ? await portalApi.getMemberOperation(opId)
            : await portalApi.getAdminOperation(opId);

        setOperation(res.data);
        setRequestId(res.requestId);
        if (res.retryAfterSeconds !== null && res.retryAfterSeconds > 0) {
          setRetryAfterSeconds(res.retryAfterSeconds);
        }

        if (isOperationTerminal(res.data.status)) {
          setIsPolling(false);
          onTerminalRef.current?.(res.data);
        }
      } catch (err) {
        setPollError(err);
        setIsPolling(false);
      }
    }, delayMs);

    return () => {
      clearTimer();
    };
  }, [isPolling, operation, retryAfterSeconds, scope, timeUnitMs, clearTimer]);

  const isDone = operation !== null && isOperationTerminal(operation.status);
  const isSuccess = operation?.status === 'succeeded';
  const isPartialSuccess = operation?.status === 'partially_succeeded';
  const isFailed = operation?.status === 'failed' || operation?.status === 'superseded';

  return {
    operation,
    requestId,
    retryAfterSeconds,
    isPolling,
    isDone,
    isSuccess,
    isPartialSuccess,
    isFailed,
    pollError,
    startTracking,
    reset,
  };
}
