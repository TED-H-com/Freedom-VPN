import { describe, expect, it } from 'vitest';
import {
  addDecimalStrings,
  compareDecimalStrings,
  DecimalValidationError,
  formatBpsMbps,
  formatBytesDual,
  formatBytesGB,
  formatBytesGiB,
  isValidDecimalString,
  MAX_UINT64,
  parseDecimalString,
  subDecimalStrings,
  sumDecimalStrings,
} from '../decimal';

describe('src/lib/decimal.ts (§10.4, §14.1, AGENTS.md §4)', () => {
  it('TestMTR01_AT17_Parse2Pow53PlusOneAndMaxUint64WithoutPrecisionLoss', () => {
    // 2^53 + 1 = 9007199254740993 (unsafe in IEEE 754 float64)
    const unsafePlusOne = '9007199254740993';
    const unsafeEven = '9007199254740992';
    expect(parseDecimalString(unsafePlusOne)).toBe(9007199254740993n);
    expect(parseDecimalString(unsafePlusOne)).not.toBe(parseDecimalString(unsafeEven));
    expect(compareDecimalStrings(unsafePlusOne, unsafeEven)).toBe(1);
    expect(compareDecimalStrings(unsafeEven, unsafePlusOne)).toBe(-1);
    expect(compareDecimalStrings(unsafePlusOne, unsafePlusOne)).toBe(0);

    // Maximum uint64: 18446744073709551615
    const maxU64Str = '18446744073709551615';
    expect(parseDecimalString(maxU64Str)).toBe(MAX_UINT64);
    expect(parseDecimalString('0')).toBe(0n);
    expect(isValidDecimalString(maxU64Str)).toBe(true);
  });

  it('TestAPI01_RejectInvalidDecimalFormsAndUint64Overflow', () => {
    const invalidInputs: unknown[] = [
      '',
      '00',
      '01',
      '-1',
      '+1',
      '1.5',
      '1e5',
      '1_000',
      ' 10',
      '10 ',
      'NaN',
      'Infinity',
      'abc',
      '18446744073709551616', // MAX_UINT64 + 1
      '100000000000000000000', // 21 digits
      12345,
      null,
      undefined,
    ];

    for (const invalid of invalidInputs) {
      expect(isValidDecimalString(invalid)).toBe(false);
      expect(() => parseDecimalString(invalid)).toThrow(DecimalValidationError);
    }
  });

  it('TestMTR02_AddSumAndSubtractWithoutNumberConversion', () => {
    expect(addDecimalStrings('9007199254740993', '2')).toBe('9007199254740995');
    expect(sumDecimalStrings(['0', '9007199254740993', '10'])).toBe('9007199254741003');
    expect(subDecimalStrings('9007199254740995', '2')).toBe('9007199254740993');

    // Overflow beyond MAX_UINT64 is rejected
    expect(() => addDecimalStrings('18446744073709551615', '1')).toThrow(
      DecimalValidationError,
    );
    expect(() => sumDecimalStrings(['18446744073709551610', '10'])).toThrow(
      DecimalValidationError,
    );
    // Negative subtraction is rejected
    expect(() => subDecimalStrings('10', '11')).toThrow(DecimalValidationError);
  });

  it('TestMTR01_FormatBytesInDecimalGBAndExplicitGiB', () => {
    // 1 GB = 1,000,000,000 bytes (§10.4)
    expect(formatBytesGB('1000000000', 2)).toBe('1.00 GB');
    expect(formatBytesGB('1500000000', 2)).toBe('1.50 GB');
    expect(formatBytesGB('1000000000', 0)).toBe('1 GB');

    // 1 GiB = 1,073,741,824 bytes, explicitly labeled GiB
    expect(formatBytesGiB('1073741824', 2)).toBe('1.00 GiB');
    expect(formatBytesGiB('1000000000', 2)).toBe('0.93 GiB');

    expect(formatBytesDual('100000000000', 2)).toBe('100.00 GB (93.13 GiB)');
    expect(formatBpsMbps('100000000')).toBe('100 Mbps');

    expect(() => formatBytesGB('1000', -1)).toThrow(RangeError);
  });
});
