import { assertNever } from '../../../lib/assertNever';
import { formatNumber } from '../../../lib/formatNumber';
import type { Action, Digit, State } from './types';

const MAX_DIGITS = 15;

export const initialState: State = {
  status: 'editing',
  input: '0',
  pending: null,
  nextId: 1,
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
    case 'UNARY_OP':
    case 'EQUALS':
    case 'EVAL_SUCCESS':
    case 'EVAL_FAILURE':
      return state; // next increments
    default:
      return assertNever(action);
  }
}
