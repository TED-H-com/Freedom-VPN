/**
 * Fleet update batch counting and state rules (§13.10.4, §13.12).
 *
 * Invariants (§13.12):
 * - Every target sits in exactly one mutually exclusive count bucket.
 * - All bucket counts sum to the immutable `target_count`.
 * - `succeeded` + `already_current` count as verified success.
 * - `rolled_back`, `superseded`, and `cancelled` are NEVER counted as success.
 * - Batch state:
 *   - `running` when any target is queued, downloading, verifying, applying, or checking;
 *   - `waiting` when no target is actively running on-node and at least one is `waiting_online`;
 *   - `completed` when all targets are in `succeeded` or `already_current`;
 *   - `completed_with_exceptions` when all targets are terminal and at least one is
 *     `failed`, `rolled_back`, `superseded`, or `cancelled`.
 */

import type {
  UpdateBatchCounts,
  UpdateBatchState,
  UpdateTargetBucket,
} from '../api/types';

export const UPDATE_BUCKET_KEYS: readonly UpdateTargetBucket[] = [
  'queued',
  'waiting_online',
  'downloading',
  'verifying',
  'applying',
  'checking',
  'succeeded',
  'already_current',
  'failed',
  'rolled_back',
  'superseded',
  'cancelled',
] as const;

export const IN_PROGRESS_BUCKETS: readonly UpdateTargetBucket[] = [
  'queued',
  'downloading',
  'verifying',
  'applying',
  'checking',
] as const;

export const SUCCESS_BUCKETS: readonly UpdateTargetBucket[] = [
  'succeeded',
  'already_current',
] as const;

export const EXCEPTION_TERMINAL_BUCKETS: readonly UpdateTargetBucket[] = [
  'failed',
  'rolled_back',
  'superseded',
  'cancelled',
] as const;

export interface BatchSummary {
  targetCount: number;
  bucketSum: number;
  isSumValid: boolean;
  verifiedSuccessCount: number;
  inProgressCount: number;
  waitingOnlineCount: number;
  failedCount: number;
  rolledBackCount: number;
  supersededCount: number;
  cancelledCount: number;
  exceptionTotalCount: number;
  computedState: UpdateBatchState;
}

export function sumBatchBuckets(counts: UpdateBatchCounts): number {
  let total = 0;
  for (const key of UPDATE_BUCKET_KEYS) {
    const val = counts[key];
    if (!Number.isInteger(val) || val < 0) {
      throw new RangeError(`Invalid bucket count for ${key}: ${String(val)}`);
    }
    total += val;
  }
  return total;
}

export function computeBatchState(
  targetCount: number,
  counts: UpdateBatchCounts,
): UpdateBatchState {
  const sum = sumBatchBuckets(counts);
  if (sum !== targetCount) {
    throw new RangeError(
      `Batch bucket counts sum (${sum}) does not equal immutable target_count (${targetCount})`,
    );
  }

  let inProgress = 0;
  for (const key of IN_PROGRESS_BUCKETS) {
    inProgress += counts[key];
  }
  if (inProgress > 0) {
    return 'running';
  }

  if (counts.waiting_online > 0) {
    return 'waiting';
  }

  let exceptions = 0;
  for (const key of EXCEPTION_TERMINAL_BUCKETS) {
    exceptions += counts[key];
  }
  if (exceptions > 0) {
    return 'completed_with_exceptions';
  }

  return 'completed';
}

export function summarizeBatch(
  targetCount: number,
  counts: UpdateBatchCounts,
): BatchSummary {
  const bucketSum = sumBatchBuckets(counts);
  const isSumValid = bucketSum === targetCount;
  const verifiedSuccessCount = counts.succeeded + counts.already_current;
  const inProgressCount =
    counts.queued +
    counts.downloading +
    counts.verifying +
    counts.applying +
    counts.checking;
  const waitingOnlineCount = counts.waiting_online;
  const failedCount = counts.failed;
  const rolledBackCount = counts.rolled_back;
  const supersededCount = counts.superseded;
  const cancelledCount = counts.cancelled;
  const exceptionTotalCount =
    failedCount + rolledBackCount + supersededCount + cancelledCount;

  const computedState = isSumValid ? computeBatchState(targetCount, counts) : 'running';

  return {
    targetCount,
    bucketSum,
    isSumValid,
    verifiedSuccessCount,
    inProgressCount,
    waitingOnlineCount,
    failedCount,
    rolledBackCount,
    supersededCount,
    cancelledCount,
    exceptionTotalCount,
    computedState,
  };
}
