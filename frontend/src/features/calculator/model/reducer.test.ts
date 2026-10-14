import { describe, expect, it } from 'vitest';
import { getDisplay, getExpression, initialState, reduce } from './reducer';
import type { Action, BinaryOp, Digit, State } from './types';

const BINARY: Record<string, BinaryOp> = {
  '+': 'add',
  '−': 'subtract',
  '×': 'multiply',
  '÷': 'divide',
  '^': 'power',
  '%': 'percentage',
};

/** Turns a key string such as "12+3=" into actions (spaces are ignored). */
function actionsFor(keys: string): Action[] {
  return [...keys.replace(/ /g, '')].map((key): Action => {
    if (/^[0-9]$/.test(key)) return { type: 'DIGIT', digit: key as Digit };
    const op = BINARY[key];
    if (op) return { type: 'BINARY_OP', op };
    switch (key) {
      case '.':
        return { type: 'DECIMAL' };
      case '±':
        return { type: 'TOGGLE_SIGN' };
      case '⌫':
        return { type: 'BACKSPACE' };
      case 'C':
        return { type: 'CLEAR' };
      case '√':
        return { type: 'UNARY_OP', op: 'sqrt' };
      case '=':
        return { type: 'EQUALS' };
      default:
        throw new Error(`unknown test key ${key}`);
    }
  });
}

/** Applies a key string to a state. */
function press(state: State, keys: string): State {
  return actionsFor(keys).reduce(reduce, state);
}

const fresh = (keys: string) => press(initialState, keys);

function evaluating(state: State) {
  if (state.status !== 'evaluating') {
    throw new Error(`expected evaluating, got ${state.status}`);
  }
  return state;
}

/** Delivers a successful response for the request currently in flight. */
function resolve(state: State, result: number): State {
  return reduce(evaluating(state), {
    type: 'EVAL_SUCCESS',
    id: evaluating(state).request.id,
    result,
  });
}

describe('initial state', () => {
  it('is editing "0" with no pending operator and ids starting at 1', () => {
    expect(initialState).toEqual({
      status: 'editing',
      input: '0',
      pending: null,
      nextId: 1,
    });
    expect(getDisplay(initialState)).toBe('0');
  });
});

// FE-01, rule 1
describe('rule 1: digits and decimal point', () => {
  it('appends digits', () => {
    expect(getDisplay(fresh('123'))).toBe('123');
  });

  it('replaces a lone 0', () => {
    expect(getDisplay(fresh('5'))).toBe('5');
    expect(getDisplay(fresh('00'))).toBe('0');
    expect(getDisplay(fresh('007'))).toBe('7');
  });

  it('allows at most 15 digits', () => {
    expect(getDisplay(fresh('1'.repeat(20)))).toBe('1'.repeat(15));
  });

  it('does not count the sign or the decimal point toward the 15 digits', () => {
    const decimal = fresh('1.' + '2'.repeat(20));
    expect(getDisplay(decimal)).toBe('1.' + '2'.repeat(14));
    const negative = fresh('±' + '9'.repeat(20)); // ± on 0 is a no-op, so it stays positive
    expect(getDisplay(negative)).toBe('9'.repeat(15));
    const signed = press(fresh('1'), '±' + '1'.repeat(20));
    expect(getDisplay(signed)).toBe('-' + '1'.repeat(15));
  });

  it('ignores a second decimal point', () => {
    expect(getDisplay(fresh('1.2.3'))).toBe('1.23');
  });

  it('turns "." on empty input into "0."', () => {
    expect(getDisplay(fresh('.'))).toBe('0.');
    expect(getDisplay(fresh('.5'))).toBe('0.5');
  });
});

describe('backspace and toggle sign while editing', () => {
  it('removes the last character and falls back to "0"', () => {
    expect(getDisplay(fresh('123⌫'))).toBe('12');
    expect(getDisplay(fresh('5⌫'))).toBe('0');
    expect(getDisplay(fresh('1.5⌫⌫'))).toBe('1');
    expect(getDisplay(fresh('5⌫⌫'))).toBe('0');
    expect(getDisplay(fresh('5±⌫'))).toBe('0');
    expect(getDisplay(fresh('⌫'))).toBe('0');
  });

  it('flips the sign of a non-zero input', () => {
    expect(getDisplay(fresh('5±'))).toBe('-5');
    expect(getDisplay(fresh('5±±'))).toBe('5');
    expect(getDisplay(fresh('1.5±'))).toBe('-1.5');
  });

  it('never produces a negative zero input (12e)', () => {
    for (const keys of ['±', '.±', '0.0±', '00±']) {
      expect(getDisplay(fresh(keys))).toBe(
        getDisplay(fresh(keys.slice(0, -1))),
      );
    }
  });
});

// FE-01, rule 11
describe('rule 11: CLEAR', () => {
  it('resets to the initial state but keeps nextId', () => {
    const busy: State = {
      status: 'editing',
      input: '123',
      pending: { op: 'add', left: 4 },
      nextId: 7,
    };
    expect(reduce(busy, { type: 'CLEAR' })).toEqual({
      ...initialState,
      nextId: 7,
    });
  });
});

// FE-01, rule 2
describe('rule 2: first binary operator', () => {
  it('commits the current value as the left operand', () => {
    const state = fresh('2+');
    expect(state).toMatchObject({
      status: 'editing',
      input: null,
      pending: { op: 'add', left: 2 },
    });
    expect(getDisplay(state)).toBe('2');
    expect(getExpression(state)).toBe('2 +');
  });

  it('starts a new input with the next digit or decimal point', () => {
    expect(getDisplay(fresh('2+3'))).toBe('3');
    expect(getDisplay(fresh('2+.'))).toBe('0.');
  });

  it('shows the operator symbol for every binary operation', () => {
    expect(getExpression(fresh('2−'))).toBe('2 −');
    expect(getExpression(fresh('2×'))).toBe('2 ×');
    expect(getExpression(fresh('2÷'))).toBe('2 ÷');
    expect(getExpression(fresh('2^'))).toBe('2 ^');
    expect(getExpression(fresh('20%'))).toBe('20 % of');
    expect(getExpression(initialState)).toBe('');
  });
});

// FE-01, rule 3
describe('rule 3: operator right after an operator', () => {
  it('replaces the pending operator without a request', () => {
    expect(fresh('2+×')).toMatchObject({
      status: 'editing',
      input: null,
      pending: { op: 'multiply', left: 2 },
    });
  });
});

// FE-01, rule 4
describe('rule 4: operator with a pending operation', () => {
  it('evaluates the pending operation as data, then chains the new operator', () => {
    const state = fresh('2+3×');
    expect(evaluating(state).request).toEqual({
      id: 1,
      op: 'add',
      operands: [2, 3],
    });
    expect(getDisplay(state)).toBe('3');

    const chained = resolve(state, 5);
    expect(chained).toMatchObject({
      status: 'editing',
      input: null,
      pending: { op: 'multiply', left: 5 },
    });
    expect(getDisplay(chained)).toBe('5');
    expect(getExpression(chained)).toBe('5 ×');
  });

  it('evaluates left to right without precedence (2 + 3 × 4 = 20)', () => {
    const second = press(resolve(fresh('2+3×'), 5), '4=');
    expect(evaluating(second).request).toEqual({
      id: 2,
      op: 'multiply',
      operands: [5, 4],
    });
    expect(resolve(second, 20)).toMatchObject({ status: 'result', value: 20 });
  });
});

// FE-01, rule 5
describe('rule 5: equals', () => {
  it('evaluates the pending operation and shows the result', () => {
    const state = fresh('20%150=');
    expect(evaluating(state).request).toEqual({
      id: 1,
      op: 'percentage',
      operands: [20, 150],
    });
    const done = resolve(state, 30);
    expect(done).toEqual({
      status: 'result',
      value: 30,
      pending: null,
      nextId: 2,
    });
    expect(getDisplay(done)).toBe('30');
    expect(getExpression(done)).toBe('');
  });

  it('is a no-op without a pending operator or without a new operand', () => {
    const typed = fresh('5');
    expect(reduce(typed, { type: 'EQUALS' })).toBe(typed);
    const committed = fresh('2+');
    expect(reduce(committed, { type: 'EQUALS' })).toBe(committed);
    const result = resolve(fresh('2+3='), 5);
    expect(reduce(result, { type: 'EQUALS' })).toBe(result); // no "repeat last op"
  });
});

// FE-01, rule 7
describe('rule 7: after a result', () => {
  const result = () => resolve(fresh('2+3='), 5);

  it('starts a fresh input with a digit or decimal point', () => {
    expect(result()).toMatchObject({ status: 'result', value: 5 });
    expect(press(result(), '7')).toMatchObject({
      status: 'editing',
      input: '7',
      pending: null,
    });
    expect(getDisplay(press(result(), '.'))).toBe('0.');
  });

  it('ignores backspace', () => {
    const state = result();
    expect(reduce(state, { type: 'BACKSPACE' })).toBe(state);
  });

  it('uses the result as the left operand of the next operator', () => {
    expect(press(result(), '×')).toMatchObject({
      status: 'editing',
      input: null,
      pending: { op: 'multiply', left: 5 },
    });
  });
});

// FE-01, rule 8
describe('rule 8: while evaluating', () => {
  const busy = () => fresh('2+3=');

  it('ignores every action except CLEAR and EVAL_*', () => {
    const state = busy();
    const ignored: Action[] = [
      { type: 'DIGIT', digit: '4' },
      { type: 'DECIMAL' },
      { type: 'TOGGLE_SIGN' },
      { type: 'BACKSPACE' },
      { type: 'BINARY_OP', op: 'multiply' },
      { type: 'UNARY_OP', op: 'sqrt' },
      { type: 'EQUALS' },
    ];
    for (const action of ignored) {
      expect(reduce(state, action)).toBe(state);
    }
  });

  it('CLEAR resets to the initial state but keeps nextId', () => {
    expect(reduce(busy(), { type: 'CLEAR' })).toEqual({
      ...initialState,
      nextId: 2,
    });
  });
});

// FE-01, rule 9
describe('rule 9: stale responses', () => {
  const error = { code: 'TIMEOUT' } as const;

  it('ignores EVAL_* whose id is not the current request', () => {
    const state = fresh('2+3=');
    expect(reduce(state, { type: 'EVAL_SUCCESS', id: 99, result: 5 })).toBe(
      state,
    );
    expect(reduce(state, { type: 'EVAL_FAILURE', id: 99, error })).toBe(state);
  });

  it('ignores EVAL_* when nothing is in flight', () => {
    const state = fresh('5');
    expect(reduce(state, { type: 'EVAL_SUCCESS', id: 1, result: 5 })).toBe(
      state,
    );
    expect(reduce(state, { type: 'EVAL_FAILURE', id: 1, error })).toBe(state);
  });
});

// FE-01, rule 10
describe('rule 10: failure', () => {
  const failed = () => {
    const state = fresh('1÷0=');
    return reduce(state, {
      type: 'EVAL_FAILURE',
      id: evaluating(state).request.id,
      error: { code: 'DIVISION_BY_ZERO', status: 422 },
    });
  };

  it('moves to the error status and displays "Error"', () => {
    expect(failed()).toEqual({
      status: 'error',
      error: { code: 'DIVISION_BY_ZERO', status: 422 },
      nextId: 2,
    });
    expect(getDisplay(failed())).toBe('Error');
    expect(getExpression(failed())).toBe('');
  });

  it('starts clean on the next digit', () => {
    expect(press(failed(), '7')).toEqual({
      status: 'editing',
      input: '7',
      pending: null,
      nextId: 2,
    });
  });

  it('starts clean on CLEAR', () => {
    expect(reduce(failed(), { type: 'CLEAR' })).toEqual({
      ...initialState,
      nextId: 2,
    });
  });

  it('ignores other keys', () => {
    const state = failed();
    const ignored: Action[] = [
      { type: 'DECIMAL' },
      { type: 'TOGGLE_SIGN' },
      { type: 'BACKSPACE' },
      { type: 'BINARY_OP', op: 'add' },
      { type: 'UNARY_OP', op: 'sqrt' },
      { type: 'EQUALS' },
    ];
    for (const action of ignored) {
      expect(reduce(state, action)).toBe(state);
    }
  });
});
