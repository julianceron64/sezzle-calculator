import { describe, expect, it } from 'vitest';
import { SERVER_ERROR_CODES, type ApiError } from '../../api/types';
import { errorMessage } from './errorMessages';

describe('errorMessage', () => {
  it.each<[ApiError, RegExp]>([
    [{ code: 'DIVISION_BY_ZERO', status: 422 }, /divide by zero/i],
    [{ code: 'NEGATIVE_SQUARE_ROOT', status: 422 }, /square root.*negative/i],
    [{ code: 'RESULT_OUT_OF_RANGE', status: 422 }, /too large|out of range/i],
    [{ code: 'UNDEFINED_RESULT', status: 422 }, /undefined/i],
    [{ code: 'NETWORK_ERROR' }, /reach the server|connection/i],
    [{ code: 'TIMEOUT' }, /too long|time/i],
  ])('has specific copy for %j', (error, expected) => {
    expect(errorMessage(error)).toMatch(expected);
  });

  it('falls back to generic copy for every other code', () => {
    const specific = new Set([
      'DIVISION_BY_ZERO',
      'NEGATIVE_SQUARE_ROOT',
      'RESULT_OUT_OF_RANGE',
      'UNDEFINED_RESULT',
    ]);
    const others: ApiError[] = [
      ...SERVER_ERROR_CODES.filter((code) => !specific.has(code)).map(
        (code): ApiError => ({ code, status: 500 }),
      ),
      { code: 'UNKNOWN_ERROR', status: 502 },
      { code: 'INVALID_RESPONSE' },
      { code: 'ABORTED' },
    ];
    for (const error of others) {
      expect(errorMessage(error)).toBe(
        'Something went wrong. Please try again.',
      );
    }
  });
});
