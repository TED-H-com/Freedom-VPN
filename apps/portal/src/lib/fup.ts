/**
 * Fair Use Policy (FUP) draft validation (§11.1).
 *
 * Rules from §11.1:
 * - Thresholds (`threshold_bytes`) must be valid decimal strings, starting at "0" for Tier 0,
 *   and strictly increasing across tiers (`0 = T0 < T1 < T2 < ...`).
 * - Complete tiers: no gaps; Tier 0 must start at "0".
 * - Both directions (`down_bps` and `up_bps`) must be present on every tier.
 * - Rates must be non-negative decimal strings (`>= 0`) and at least `minimum_rate_bps`
 *   (where `minimum_rate_bps > 0`).
 * - Mode must be `shadow` or `enforce`, and switching/activating requires a valid UTC RFC 3339
 *   `effective_at` timestamp ending in `Z`.
 */

import type { FairUsePolicyDraft, FairUseTierDraft } from '../api/view-models';
import {
  compareDecimalStrings,
  isValidDecimalString,
  parseDecimalString,
} from './decimal';

export interface FupValidationError {
  field: string;
  code:
    | 'EMPTY_TIERS'
    | 'MISSING_ZERO_START'
    | 'INVALID_DECIMAL'
    | 'NON_INCREASING_THRESHOLD'
    | 'MISSING_DIRECTION_RATE'
    | 'RATE_BELOW_MINIMUM'
    | 'INVALID_MINIMUM_RATE'
    | 'INVALID_PRIORITY'
    | 'EMPTY_TIER_NAME'
    | 'INVALID_EFFECTIVE_AT';
  messageKey: string;
  tierIndex?: number;
}

export interface FupValidationResult {
  valid: boolean;
  errors: FupValidationError[];
}

const UTC_RFC3339_REGEX = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$/;

export function validateFupDraft(draft: FairUsePolicyDraft): FupValidationResult {
  const errors: FupValidationError[] = [];

  if (!UTC_RFC3339_REGEX.test(draft.effective_at)) {
    errors.push({
      field: 'effective_at',
      code: 'INVALID_EFFECTIVE_AT',
      messageKey: 'fup.validation.invalidEffectiveAt',
    });
  }

  let minRateBigInt = 0n;
  if (!isValidDecimalString(draft.minimum_rate_bps)) {
    errors.push({
      field: 'minimum_rate_bps',
      code: 'INVALID_MINIMUM_RATE',
      messageKey: 'fup.validation.invalidMinimumRate',
    });
  } else {
    minRateBigInt = parseDecimalString(draft.minimum_rate_bps);
    if (minRateBigInt <= 0n) {
      errors.push({
        field: 'minimum_rate_bps',
        code: 'INVALID_MINIMUM_RATE',
        messageKey: 'fup.validation.invalidMinimumRate',
      });
    }
  }

  if (!Array.isArray(draft.tiers) || draft.tiers.length === 0) {
    errors.push({
      field: 'tiers',
      code: 'EMPTY_TIERS',
      messageKey: 'fup.validation.emptyTiers',
    });
    return { valid: false, errors };
  }

  const firstTier: FairUseTierDraft | undefined = draft.tiers[0];
  if (!firstTier || firstTier.threshold_bytes !== '0') {
    errors.push({
      field: 'tiers[0].threshold_bytes',
      code: 'MISSING_ZERO_START',
      messageKey: 'fup.validation.missingZeroStart',
      tierIndex: 0,
    });
  }

  for (let i = 0; i < draft.tiers.length; i++) {
    const tier = draft.tiers[i];
    if (!tier) continue;

    if (!tier.name_zh.trim() || !tier.name_en.trim()) {
      errors.push({
        field: `tiers[${i}].name`,
        code: 'EMPTY_TIER_NAME',
        messageKey: 'fup.validation.emptyTierName',
        tierIndex: i,
      });
    }

    if (!isValidDecimalString(tier.threshold_bytes)) {
      errors.push({
        field: `tiers[${i}].threshold_bytes`,
        code: 'INVALID_DECIMAL',
        messageKey: 'fup.validation.invalidThreshold',
        tierIndex: i,
      });
    } else if (i > 0) {
      const prevTier = draft.tiers[i - 1];
      if (
        prevTier &&
        isValidDecimalString(prevTier.threshold_bytes) &&
        compareDecimalStrings(tier.threshold_bytes, prevTier.threshold_bytes) <= 0
      ) {
        errors.push({
          field: `tiers[${i}].threshold_bytes`,
          code: 'NON_INCREASING_THRESHOLD',
          messageKey: 'fup.validation.nonIncreasingThreshold',
          tierIndex: i,
        });
      }
    }

    if (!tier.down_bps || !tier.up_bps) {
      errors.push({
        field: `tiers[${i}].rates`,
        code: 'MISSING_DIRECTION_RATE',
        messageKey: 'fup.validation.missingDirectionRate',
        tierIndex: i,
      });
    } else if (
      !isValidDecimalString(tier.down_bps) ||
      !isValidDecimalString(tier.up_bps)
    ) {
      errors.push({
        field: `tiers[${i}].rates`,
        code: 'INVALID_DECIMAL',
        messageKey: 'fup.validation.invalidRateDecimal',
        tierIndex: i,
      });
    } else {
      const down = parseDecimalString(tier.down_bps);
      const up = parseDecimalString(tier.up_bps);
      if (minRateBigInt > 0n && (down < minRateBigInt || up < minRateBigInt)) {
        errors.push({
          field: `tiers[${i}].rates`,
          code: 'RATE_BELOW_MINIMUM',
          messageKey: 'fup.validation.rateBelowMinimum',
          tierIndex: i,
        });
      }
    }

    if (!Number.isInteger(tier.priority) || tier.priority < 0 || tier.priority > 10) {
      errors.push({
        field: `tiers[${i}].priority`,
        code: 'INVALID_PRIORITY',
        messageKey: 'fup.validation.invalidPriority',
        tierIndex: i,
      });
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

export const validateFupPolicyDraft = validateFupDraft;
