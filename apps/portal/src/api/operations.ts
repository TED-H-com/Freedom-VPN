/**
 * Asynchronous 202 operation status and polling helpers (§14.5, §14.8, §18.3).
 *
 * Honesty contract:
 * - A 202 Accepted response is NEVER treated as completed.
 * - Polls `/api/v1/admin/operations/{id}` (or `/api/v1/member/operations/{id}`).
 * - Exposes each step's status (`pending`, `running`, `succeeded`, `failed`, etc.)
 *   and only marks `isTerminalOperationStatus(status) === true` when the operation reaches a terminal state.
 */

import type { OperationResource, OperationStatus } from './types';

export const TERMINAL_OPERATION_STATUSES: readonly OperationStatus[] = [
  'succeeded',
  'partially_succeeded',
  'failed',
  'superseded',
];

export function isTerminalOperationStatus(status: OperationStatus): boolean {
  return TERMINAL_OPERATION_STATUSES.includes(status);
}

export function isOperationTerminal(status: OperationStatus): boolean {
  return isTerminalOperationStatus(status);
}

export function hasOfflineRouteWithdrawnWarning(op: OperationResource): boolean {
  return op.steps.some(
    (s) =>
      (s.name === 'node_allowset_withdraw' && s.status === 'failed') ||
      Boolean(s.message?.includes('中央已撤銷路由')),
  );
}
