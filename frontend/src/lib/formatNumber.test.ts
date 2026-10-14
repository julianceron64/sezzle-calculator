import { describe, expect, it } from 'vitest';
import { formatNumber } from './formatNumber';

// FE-04: expectations are exactly the table in spec 003.
describe('formatNumber', () => {
  const basic: [string, number, string][] = [
    ['zero', 0, '0'],
    ['negative zero', -0, '0'],
    ['0.1 + 0.2', 0.1 + 0.2, '0.3'],
    ['2/3', 2 / 3, '0.666666666667'],
    ['negative decimal', -1.5, '-1.5'],
    ['1e11 stays plain', 1e11, '100000000000'],
    ['1e12 goes exponential', 1e12, '1e+12'],
    ['15-digit integer', 123456789012345, '1.23456789e+14'],
    ['negative 15-digit integer', -123456789012345, '-1.23456789e+14'],
    ['1e15', 1e15, '1e+15'],
    ['-1e15', -1e15, '-1e+15'],
    ['20-digit integer', Number('12345678901234567890'), '1.23456789e+19'],
    ['1e-6', 1e-6, '0.000001'],
    ['1e-7', 1e-7, '0.0000001'],
    ['1e-9 stays plain', 1e-9, '0.000000001'],
    ['1e-10 goes exponential', 1e-10, '1e-10'],
    ['-1e-10', -1e-10, '-1e-10'],
  ];

  // The threshold is decided on the rounded value, not the raw one.
  const roundingCrossing: [string, number, string][] = [
    ['999999999999.4 (control, stays plain)', 999999999999.4, '999999999999'],
    ['999999999999.9 rounds up to 1e12', 999999999999.9, '1e+12'],
    ['-999999999999.9', -999999999999.9, '-1e+12'],
    ['999999999999999.9', 999999999999999.9, '1e+15'],
    [
      '9.9999999999999e-10 rounds up to 1e-9',
      9.9999999999999e-10,
      '0.000000001',
    ],
    ['9.99999e-10 (control)', 9.99999e-10, '9.99999e-10'],
  ];

  it.each([...basic, ...roundingCrossing])('%s', (_name, input, expected) => {
    expect(formatNumber(input)).toBe(expected);
  });
});
