import type { ApiError } from '../../api/types';

const GENERIC = 'Something went wrong. Please try again.';

/** User-facing copy, chosen by error code only (never by backend `detail` text). */
export function errorMessage(error: ApiError): string {
  switch (error.code) {
    case 'DIVISION_BY_ZERO':
      return 'Cannot divide by zero.';
    case 'NEGATIVE_SQUARE_ROOT':
      return 'Cannot take the square root of a negative number.';
    case 'RESULT_OUT_OF_RANGE':
      return 'The result is too large to calculate.';
    case 'UNDEFINED_RESULT':
      return 'The result is undefined.';
    case 'NETWORK_ERROR':
      return 'Could not reach the server. Check your connection and try again.';
    case 'TIMEOUT':
      return 'The server took too long to respond. Please try again.';
    default:
      return GENERIC;
  }
}
