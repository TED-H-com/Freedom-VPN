import { describe, expect, it } from 'vitest';
import { computeBatchState, sumBatchBuckets, summarizeBatch } from '../batch';
import { matchesDomainRule, normalizeAndValidateDomain, previewDnsMatch } from '../dns';
import { validateFupDraft } from '../fup';
import { INITIAL_BATCH, INITIAL_DNS_POLICY, INITIAL_FUP_POLICY } from '../../mocks/data';

describe('src/lib/batch.ts (§13.12 Fleet Update Batch Invariants)', () => {
  it('TestUPD01_UPD05_AT23_BatchBucketsSumToTargetCountAndRolledBackNotCountedAsSuccess', () => {
    const summary = summarizeBatch(INITIAL_BATCH.target_count, INITIAL_BATCH.counts);

    expect(summary.isSumValid).toBe(true);
    expect(summary.bucketSum).toBe(INITIAL_BATCH.target_count);
    expect(summary.bucketSum).toBe(12);

    // Only succeeded (1) + already_current (1) count as verified success = 2.
    // rolled_back (1), superseded (1), cancelled (1) are NEVER counted as success.
    expect(summary.verifiedSuccessCount).toBe(2);
    expect(summary.rolledBackCount).toBe(1);
    expect(summary.supersededCount).toBe(1);
    expect(summary.cancelledCount).toBe(1);
    expect(summary.failedCount).toBe(1);
    expect(summary.computedState).toBe('running');
  });

  it('TestUPD02_BatchStateTransitionsFollowSection13_12', () => {
    const baseZero = {
      queued: 0,
      waiting_online: 0,
      downloading: 0,
      verifying: 0,
      applying: 0,
      checking: 0,
      succeeded: 0,
      already_current: 0,
      failed: 0,
      rolled_back: 0,
      superseded: 0,
      cancelled: 0,
    };

    // Waiting online -> 'waiting'
    expect(computeBatchState(5, { ...baseZero, succeeded: 4, waiting_online: 1 })).toBe(
      'waiting',
    );

    // All succeeded or already_current -> 'completed'
    expect(computeBatchState(5, { ...baseZero, succeeded: 3, already_current: 2 })).toBe(
      'completed',
    );

    // Terminal with rolled_back -> 'completed_with_exceptions' (never 'completed')
    expect(computeBatchState(5, { ...baseZero, succeeded: 4, rolled_back: 1 })).toBe(
      'completed_with_exceptions',
    );

    // Mismatched bucket sum throws RangeError
    expect(() => computeBatchState(5, { ...baseZero, succeeded: 3 })).toThrow(RangeError);
    expect(() => sumBatchBuckets({ ...baseZero, succeeded: -1 })).toThrow(RangeError);
  });
});

describe('src/lib/fup.ts (§11.1 Fair Use Policy Validation)', () => {
  it('TestFUP01_ValidDraftPassesValidation', () => {
    const res = validateFupDraft(INITIAL_FUP_POLICY.current);
    expect(res.valid).toBe(true);
    expect(res.errors).toEqual([]);
  });

  it('TestFUP01_RejectsNonZeroStartNonIncreasingThresholdsAndInvalidRates', () => {
    const brokenDraft = {
      ...INITIAL_FUP_POLICY.current,
      effective_at: '2026-11-01 00:00:00', // missing Z
      minimum_rate_bps: '5000000',
      tiers: [
        {
          tier_index: 0,
          name_zh: 'Tier 0',
          name_en: 'Tier 0',
          threshold_bytes: '100', // Must start at 0!
          down_bps: '100000000',
          up_bps: '100000000',
          priority: 1,
        },
        {
          tier_index: 1,
          name_zh: '', // Empty name
          name_en: 'Tier 1',
          threshold_bytes: '100', // Not strictly increasing!
          down_bps: '1000000', // Below minimum_rate_bps (5000000)
          up_bps: '', // Missing up_bps!
          priority: 15, // Out of range 1..10
        },
      ],
    };

    const res = validateFupDraft(brokenDraft);
    expect(res.valid).toBe(false);
    const codes = res.errors.map((e) => e.code);
    expect(codes).toContain('INVALID_EFFECTIVE_AT');
    expect(codes).toContain('MISSING_ZERO_START');
    expect(codes).toContain('EMPTY_TIER_NAME');
    expect(codes).toContain('NON_INCREASING_THRESHOLD');
    expect(codes).toContain('MISSING_DIRECTION_RATE');
    expect(codes).toContain('INVALID_PRIORITY');
  });

  it('TestFUP01_RejectsEmptyTiersAndRateBelowMinimum', () => {
    const emptyRes = validateFupDraft({
      ...INITIAL_FUP_POLICY.current,
      minimum_rate_bps: '0',
      tiers: [],
    });
    expect(emptyRes.valid).toBe(false);
    expect(emptyRes.errors.map((e) => e.code)).toContain('EMPTY_TIERS');
    expect(emptyRes.errors.map((e) => e.code)).toContain('INVALID_MINIMUM_RATE');

    const lowRateRes = validateFupDraft({
      ...INITIAL_FUP_POLICY.current,
      minimum_rate_bps: '10000000',
      tiers: [
        {
          tier_index: 0,
          name_zh: 'T0',
          name_en: 'T0',
          threshold_bytes: '0',
          down_bps: '5000000',
          up_bps: '10000000',
          priority: 1,
        },
      ],
    });
    expect(lowRateRes.valid).toBe(false);
    expect(lowRateRes.errors.map((e) => e.code)).toContain('RATE_BELOW_MINIMUM');
  });
});

describe('src/lib/dns.ts (§12.4 DNS Domain Rules & Priority Preview)', () => {
  it('TestDNS02_AT20_NormalizesDomainAndEnforcesLabelBoundaries', () => {
    expect(normalizeAndValidateDomain('  Sub.Example.COM. ')).toEqual({
      valid: true,
      canonicalDomain: 'sub.example.com',
    });

    // Rejects URL, path, port, wildcard
    expect(normalizeAndValidateDomain('https://example.com/path').errorCode).toBe(
      'URL_OR_PATH_NOT_ALLOWED',
    );
    expect(normalizeAndValidateDomain('example.com:8080').errorCode).toBe(
      'URL_OR_PATH_NOT_ALLOWED',
    );
    expect(normalizeAndValidateDomain('*.example.com').errorCode).toBe(
      'WILDCARD_NOT_ALLOWED',
    );
    expect(normalizeAndValidateDomain('').errorCode).toBe('EMPTY_DOMAIN');
    expect(normalizeAndValidateDomain('localhost').errorCode).toBe('INVALID_LABEL');

    // Label boundary check: example.com must NOT match notexample.com
    expect(matchesDomainRule('example.com', 'example.com', true)).toBe(true);
    expect(matchesDomainRule('a.b.example.com', 'example.com', true)).toBe(true);
    expect(matchesDomainRule('notexample.com', 'example.com', true)).toBe(false);
    expect(matchesDomainRule('sub.example.com', 'example.com', false)).toBe(false);

    // Priority preview check
    const blockPreview = previewDnsMatch(
      'api.tracker.example.com',
      INITIAL_DNS_POLICY.rules,
    );
    expect(blockPreview.outcome).toBe('blocked_admin');
    expect(blockPreview.priorityLevel).toBe(1);

    const notMatchedPreview = previewDnsMatch(
      'nottracker.example.com',
      INITIAL_DNS_POLICY.rules,
    );
    expect(notMatchedPreview.outcome).toBe('resolved_normal');
    expect(notMatchedPreview.priorityLevel).toBe(4);

    const allowPreview = previewDnsMatch(
      'cdn-safe.example.com',
      INITIAL_DNS_POLICY.rules,
    );
    expect(allowPreview.outcome).toBe('allowed_admin');
    expect(allowPreview.priorityLevel).toBe(2);

    const threatPreview = previewDnsMatch(
      'login.phishing-login.example.invalid',
      INITIAL_DNS_POLICY.rules,
    );
    expect(threatPreview.outcome).toBe('blocked_threat_feed');
    expect(threatPreview.priorityLevel).toBe(3);
  });
});
