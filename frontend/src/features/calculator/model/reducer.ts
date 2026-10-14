import type { ApiError, OperationName } from '../../../api/types';
import { assertNever } from '../../../lib/assertNever';
import { formatNumber } from '../../../lib/formatNumber';
import type {
  Action,
  BinaryOp,
  Continuation,
  Digit,
  Pending,
  State,
} from './types';

const MAX_DIGITS = 15;

export const initialState: State = {
  status: 'editing',
  input: '0',
  pending: null,
  nextId: 1,
};

const SYMBOLS: Record<BinaryOp, string> = {
  add: '+',
  subtract: '−',
  multiply: '×',
  divide: '÷',
  power: '^',
  percentage: '% of',
};

const countDigits = (input: string) => input.replace(/[-.]/g, '').length;

/** Main display line: the input as typed while editing, otherwise a formatted value. */
export function getDisplay(state: State): string {
  switch (state.status) {
    case 'editing':
      return state.input ?? formatNumber(state.pending?.left ?? 0);
    case 'result':
      return formatNumber(state.value);
    case 'evaluating':
      return state.shown;
    case 'error':
      return 'Error';
    default:
      return assertNever(state);
  }
}

/** Secondary display line: the pending operation, e.g. "2 +" or "20 % of". */
export function getExpression(state: State): string {
  if (state.status === 'error' || !state.pending) return '';
  return `${formatNumber(state.pending.left)} ${SYMBOLS[state.pending.op]}`;
}

/**
 * Starts a request. Nothing is executed here: the state carries the request
 * as data and the hook runs it. Operands are parsed from the input but never
 * combined: results come only from EVAL_SUCCESS.
 */
function startEvaluation(
  state: State,
  request: { op: OperationName; operands: number[] },
  then: Continuation,
  pending: Pending | null,
  shown: string,
): State {
  return {
    status: 'evaluating',
    request: { id: state.nextId, ...request },
    then,
    pending,
    shown,
    nextId: state.nextId + 1,
  };
}

function digit(state: State, d: Digit): State {
  switch (state.status) {
    case 'editing': {
      const { input } = state;
      if (input === null || input === '0') return { ...state, input: d };
      if (countDigits(input) >= MAX_DIGITS) return state;
      return { ...state, input: input + d };
    }
    case 'result':
      return {
        status: 'editing',
        input: d,
        pending: state.pending,
        nextId: state.nextId,
      };
    case 'error':
      return {
        status: 'editing',
        input: d,
        pending: null,
        nextId: state.nextId,
      };
    case 'evaluating':
      return state;
    default:
      return assertNever(state);
  }
}

function decimal(state: State): State {
  switch (state.status) {
    case 'editing': {
      const { input } = state;
      if (input === null) return { ...state, input: '0.' };
      if (input.includes('.')) return state;
      return { ...state, input: input + '.' };
    }
    case 'result':
      return {
        status: 'editing',
        input: '0.',
        pending: state.pending,
        nextId: state.nextId,
      };
    case 'error':
    case 'evaluating':
      return state;
    default:
      return assertNever(state);
  }
}

function toggleSign(state: State): State {
  if (state.status !== 'editing') return state;
  const { input } = state;
  // Committed left operand (12a) or a zero input (12e, never "-0"): no-op.
  if (input === null || Number(input) === 0) return state;
  return {
    ...state,
    input: input.startsWith('-') ? input.slice(1) : `-${input}`,
  };
}

function backspace(state: State): State {
  if (state.status !== 'editing' || state.input === null) return state; // 12b
  const shorter = state.input.slice(0, -1);
  return {
    ...state,
    input: shorter === '' || shorter === '-' ? '0' : shorter,
  };
}

function binaryOp(state: State, op: BinaryOp): State {
  switch (state.status) {
    case 'editing': {
      const { input, pending } = state;
      if (input === null) {
        // Rule 3: an operator right after an operator replaces it.
        return pending ? { ...state, pending: { ...pending, op } } : state;
      }
      if (!pending) {
        // Rule 2: the current value becomes the left operand.
        return { ...state, input: null, pending: { op, left: Number(input) } };
      }
      // Rule 4: evaluate the pending operation, then chain the new operator.
      return startEvaluation(
        state,
        { op: pending.op, operands: [pending.left, Number(input)] },
        { kind: 'chain', next: op },
        pending,
        input,
      );
    }
    case 'result': {
      const { value, pending } = state;
      if (!pending) {
        return {
          status: 'editing',
          input: null,
          pending: { op, left: value },
          nextId: state.nextId,
        };
      }
      return startEvaluation(
        state,
        { op: pending.op, operands: [pending.left, value] },
        { kind: 'chain', next: op },
        pending,
        formatNumber(value),
      );
    }
    case 'error':
    case 'evaluating':
      return state;
    default:
      return assertNever(state);
  }
}

function equals(state: State): State {
  switch (state.status) {
    case 'editing': {
      const { input, pending } = state;
      if (input === null || !pending) return state;
      return startEvaluation(
        state,
        { op: pending.op, operands: [pending.left, Number(input)] },
        { kind: 'equals' },
        pending,
        input,
      );
    }
    case 'result': {
      const { value, pending } = state;
      if (!pending) return state;
      return startEvaluation(
        state,
        { op: pending.op, operands: [pending.left, value] },
        { kind: 'equals' },
        pending,
        formatNumber(value),
      );
    }
    case 'error':
    case 'evaluating':
      return state;
    default:
      return assertNever(state);
  }
}

function evalSuccess(state: State, id: number, result: number): State {
  // Rule 9: a response that is not for the current request is stale.
  if (state.status !== 'evaluating' || state.request.id !== id) return state;
  const { then } = state;
  switch (then.kind) {
    case 'chain':
      return {
        status: 'editing',
        input: null,
        pending: { op: then.next, left: result },
        nextId: state.nextId,
      };
    case 'equals':
      return {
        status: 'result',
        value: result,
        pending: null,
        nextId: state.nextId,
      };
    case 'unary-operand':
    case 'unary-left':
      return state; // next increment
    default:
      return assertNever(then);
  }
}

function evalFailure(state: State, id: number, error: ApiError): State {
  if (state.status !== 'evaluating' || state.request.id !== id) return state;
  return { status: 'error', error, nextId: state.nextId };
}

export function reduce(state: State, action: Action): State {
  switch (action.type) {
    case 'CLEAR':
      return { ...initialState, nextId: state.nextId };
    case 'DIGIT':
      return digit(state, action.digit);
    case 'DECIMAL':
      return decimal(state);
    case 'TOGGLE_SIGN':
      return toggleSign(state);
    case 'BACKSPACE':
      return backspace(state);
    case 'BINARY_OP':
      return binaryOp(state, action.op);
    case 'EQUALS':
      return equals(state);
    case 'EVAL_SUCCESS':
      return evalSuccess(state, action.id, action.result);
    case 'EVAL_FAILURE':
      return evalFailure(state, action.id, action.error);
    case 'UNARY_OP':
      return state; // next increment
    default:
      return assertNever(action);
  }
}
