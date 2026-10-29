/**
 * Decimal-safe integer utilities for bytes, sequence numbers, generations,
 * revisions, and resource versions (§10.4, §14.1, AGENTS.md §3 & §4).
 *
 * Contract:
 * - Decimal strings match `^(0|[1-9][0-9]*)$`, have at most 20 digits,
 *   and are at most 18446744073709551615 (max uint64).
 * - Never converts counters through float64 / JavaScript Number().
 * - Formats bytes in decimal GB (1 GB = 1,000,000,000 bytes, §10.4)
 *   and explicit binary GiB (1 GiB = 1,073,741,824 bytes).
 */

export const DECIMAL_STRING_REGEX = /^(0|[1-9][0-9]*)$/;
export const MAX_DECIMAL_DIGITS = 20;
export const MAX_UINT64 = 18446744073709551615n;

export const BYTES_PER_GB = 1_000_000_000n;
export const BYTES_PER_MB = 1_000_000n;
export const BYTES_PER_GIB = 1_073_741_824n;
export const BPS_PER_MBPS = 1_000_000n;

export class DecimalValidationError extends Error {
  public readonly input: unknown;

  constructor(message: string, input: unknown) {
    super(message);
    this.name = 'DecimalValidationError';
    this.input = input;
  }
}

/**
 * Parses and validates a canonical decimal string into a bigint per AGENTS.md §4.
 * Rejects non-strings, negative values, leading zeros, decimals, exponents,
 * whitespace, strings > 20 digits, and values > MAX_UINT64.
 */
export function parseDecimalString(value: unknown): bigint {
  if (typeof value !== 'string') {
    throw new DecimalValidationError('Expected decimal string', value);
  }
  if (value.length === 0 || value.length > MAX_DECIMAL_DIGITS) {
    throw new DecimalValidationError(
      `Decimal string must be 1..${MAX_DECIMAL_DIGITS} digits`,
      value,
    );
  }
  if (!DECIMAL_STRING_REGEX.test(value)) {
    throw new DecimalValidationError(
      'Decimal string must match ^(0|[1-9][0-9]*)$',
      value,
    );
  }
  const parsed = BigInt(value);
  if (parsed > MAX_UINT64) {
    throw new DecimalValidationError(
      'Decimal string exceeds maximum uint64 (18446744073709551615)',
      value,
    );
  }
  return parsed;
}

/**
 * Type guard checking whether a value is a valid canonical uint64 decimal string.
 */
export function isValidDecimalString(value: unknown): value is string {
  try {
    parseDecimalString(value);
    return true;
  } catch {
    return false;
  }
}

/**
 * Compares two valid decimal strings without using Number().
 * Returns -1 if a < b, 0 if a === b, 1 if a > b.
 */
export function compareDecimalStrings(a: string, b: string): -1 | 0 | 1 {
  const bigA = parseDecimalString(a);
  const bigB = parseDecimalString(b);
  if (bigA < bigB) return -1;
  if (bigA > bigB) return 1;
  return 0;
}

/**
 * Adds two decimal strings using pure BigInt arithmetic and checks for uint64 overflow.
 */
export function addDecimalStrings(a: string, b: string): string {
  const bigA = parseDecimalString(a);
  const bigB = parseDecimalString(b);
  const sum = bigA + bigB;
  if (sum > MAX_UINT64) {
    throw new DecimalValidationError(
      'Sum of decimal strings exceeds maximum uint64',
      `${a} + ${b}`,
    );
  }
  return sum.toString();
}

/**
 * Sums an array of decimal strings using pure BigInt arithmetic.
 */
export function sumDecimalStrings(values: readonly string[]): string {
  let acc = 0n;
  for (const val of values) {
    const parsed = parseDecimalString(val);
    acc += parsed;
    if (acc > MAX_UINT64) {
      throw new DecimalValidationError(
        'Sum of decimal strings exceeds maximum uint64',
        values,
      );
    }
  }
  return acc.toString();
}

/**
 * Subtracts b from a using pure BigInt arithmetic, rejecting negative results.
 */
export function subDecimalStrings(a: string, b: string): string {
  const bigA = parseDecimalString(a);
  const bigB = parseDecimalString(b);
  if (bigB > bigA) {
    throw new DecimalValidationError(
      'Subtraction underflow: result cannot be negative',
      `${a} - ${b}`,
    );
  }
  return (bigA - bigB).toString();
}

/**
 * Internal helper that formats `numerator / unit` with `decimals` fractional digits
 * using pure BigInt division and half-up rounding, never calling Number() on the counter.
 */
function formatFixedBigInt(numerator: bigint, unit: bigint, decimals: number): string {
  if (decimals < 0 || decimals > 6 || !Number.isInteger(decimals)) {
    throw new RangeError('decimals must be an integer between 0 and 6');
  }
  const scale = 10n ** BigInt(decimals);
  const scaled = (numerator * scale + unit / 2n) / unit;
  if (decimals === 0) {
    return scaled.toString();
  }
  const whole = scaled / scale;
  const frac = scaled % scale;
  const fracStr = frac.toString().padStart(decimals, '0');
  return `${whole.toString()}.${fracStr}`;
}

/**
 * Formats a byte counter decimal string in decimal GB (1 GB = 1,000,000,000 bytes, §10.4).
 */
export function formatBytesGB(bytesStr: string, decimals = 2): string {
  const bytes = parseDecimalString(bytesStr);
  return `${formatFixedBigInt(bytes, BYTES_PER_GB, decimals)} GB`;
}

/**
 * Formats a byte counter decimal string in binary GiB (1 GiB = 1,073,741,824 bytes, §10.4),
 * explicitly labeled as GiB.
 */
export function formatBytesGiB(bytesStr: string, decimals = 2): string {
  const bytes = parseDecimalString(bytesStr);
  return `${formatFixedBigInt(bytes, BYTES_PER_GIB, decimals)} GiB`;
}

/**
 * Formats a byte counter showing decimal GB primary with explicit GiB secondary (§10.4).
 */
export function formatBytesDual(bytesStr: string, decimals = 2): string {
  return `${formatBytesGB(bytesStr, decimals)} (${formatBytesGiB(bytesStr, decimals)})`;
}

/**
 * Formats a bits-per-second decimal string into Mbps (1 Mbps = 1,000,000 bps).
 */
export function formatBpsMbps(bpsStr: string, decimals = 0): string {
  const bps = parseDecimalString(bpsStr);
  return `${formatFixedBigInt(bps, BPS_PER_MBPS, decimals)} Mbps`;
}
