import { describe, expect, it } from 'vitest';
import { getDisplay, initialState, reduce } from './reducer';
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
