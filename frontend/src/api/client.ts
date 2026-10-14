import {
  OPERATION_NAMES,
  SERVER_ERROR_CODES,
  type ApiError,
  type CalculateResponse,
  type OperationName,
  type ProblemDetails,
  type Result,
  type ServerErrorCode,
} from './types';

const DEFAULT_TIMEOUT_MS = 5000;

export interface CalculateOptions {
  signal?: AbortSignal;
  timeoutMs?: number;
}

const fail = (error: ApiError): Result<never, ApiError> => ({
  ok: false,
  error,
});

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

const isFiniteNumber = (value: unknown): value is number =>
  typeof value === 'number' && Number.isFinite(value);

const isOperationName = (value: unknown): value is OperationName =>
  OPERATION_NAMES.some((name) => name === value);

const isServerErrorCode = (value: string): value is ServerErrorCode =>
  SERVER_ERROR_CODES.some((code) => code === value);

function isCalculateResponse(value: unknown): value is CalculateResponse {
  return (
    isRecord(value) &&
    isOperationName(value.operation) &&
    isFiniteNumber(value.a) &&
    (value.b === undefined || isFiniteNumber(value.b)) &&
    isFiniteNumber(value.result)
  );
}

function isProblemDetails(value: unknown): value is ProblemDetails {
  return (
    isRecord(value) &&
    typeof value.type === 'string' &&
    typeof value.title === 'string' &&
    typeof value.status === 'number' &&
    typeof value.detail === 'string' &&
    typeof value.code === 'string'
  );
}

async function readJson(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return undefined; // not JSON: callers treat it as an invalid shape
  }
}

async function interpret(
  response: Response,
): Promise<Result<CalculateResponse, ApiError>> {
  if (response.ok) {
    const body = await readJson(response);
    return isCalculateResponse(body)
      ? { ok: true, value: body }
      : fail({ code: 'INVALID_RESPONSE' });
  }

  const { status } = response;
  const isProblem = response.headers
    .get('Content-Type')
    ?.includes('application/problem+json');
  if (isProblem) {
    const body = await readJson(response);
    if (isProblemDetails(body) && isServerErrorCode(body.code)) {
      return fail({ code: body.code, status });
    }
  }
  return fail({ code: 'UNKNOWN_ERROR', status });
}

/**
 * Calls POST /api/v1/{operation}. Never throws for HTTP, network, timeout or
 * abort failures: they are returned as an ApiError (ADR-005, ADR-008).
 */
export async function calculate(
  operation: OperationName,
  operands: readonly number[],
  { signal, timeoutMs = DEFAULT_TIMEOUT_MS }: CalculateOptions = {},
): Promise<Result<CalculateResponse, ApiError>> {
  const timeout = AbortSignal.timeout(timeoutMs);
  const combined = signal ? AbortSignal.any([signal, timeout]) : timeout;
  const [a, b] = operands;
  const baseUrl = import.meta.env.VITE_API_BASE_URL ?? '';

  try {
    const response = await fetch(`${baseUrl}/api/v1/${operation}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ a, b }), // b is dropped when undefined (unary)
      signal: combined,
    });
    return await interpret(response);
  } catch {
    if (signal?.aborted) return fail({ code: 'ABORTED' });
    if (timeout.aborted) return fail({ code: 'TIMEOUT' });
    return fail({ code: 'NETWORK_ERROR' });
  }
}
