export const OPERATION_NAMES = [
  'add',
  'subtract',
  'multiply',
  'divide',
  'power',
  'sqrt',
  'percentage',
] as const;
export type OperationName = (typeof OPERATION_NAMES)[number];

export interface CalculateResponse {
  operation: OperationName;
  a: number;
  b?: number;
  result: number;
}

/** RFC 9457 problem details as sent by the backend (ADR-005). */
export interface ProblemDetails {
  type: string;
  title: string;
  status: number;
  detail: string;
  code: string;
}

/** Stable backend codes. The UI maps errors by code only, never by detail text. */
export const SERVER_ERROR_CODES = [
  'UNSUPPORTED_MEDIA_TYPE',
  'PAYLOAD_TOO_LARGE',
  'INVALID_JSON',
  'UNKNOWN_FIELD',
  'INVALID_OPERAND',
  'MISSING_OPERAND',
  'UNEXPECTED_OPERAND',
  'UNKNOWN_OPERATION',
  'NOT_FOUND',
  'METHOD_NOT_ALLOWED',
  'DIVISION_BY_ZERO',
  'NEGATIVE_SQUARE_ROOT',
  'UNDEFINED_RESULT',
  'RESULT_OUT_OF_RANGE',
  'INTERNAL_ERROR',
] as const;
export type ServerErrorCode = (typeof SERVER_ERROR_CODES)[number];

/** Discriminated by `code`. Client-side failures carry no HTTP status. */
export type ApiError =
  | { code: ServerErrorCode; status: number }
  | { code: 'UNKNOWN_ERROR'; status: number }
  | { code: 'NETWORK_ERROR' | 'TIMEOUT' | 'ABORTED' | 'INVALID_RESPONSE' };

export type ApiErrorCode = ApiError['code'];

export type Result<T, E> = { ok: true; value: T } | { ok: false; error: E };
