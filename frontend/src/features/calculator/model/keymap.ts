import type { Action, BinaryOp, Digit } from './types';

const BINARY_KEYS: Record<string, BinaryOp> = {
  '+': 'add',
  '-': 'subtract',
  '*': 'multiply',
  '/': 'divide',
  '^': 'power',
  '%': 'percentage',
};

const isDigit = (key: string): key is Digit => /^[0-9]$/.test(key);

/**
 * Keyboard key to calculator action, or null for keys the calculator does not
 * handle. Ctrl/Alt/Meta combinations are left to the browser. Shift is not a
 * blocker: `+`, `*`, `^` and `%` need it on many layouts, and the key identity
 * already comes from `event.key`.
 */
export function keyToAction(
  event: Pick<KeyboardEvent, 'key' | 'ctrlKey' | 'altKey' | 'metaKey'>,
): Action | null {
  if (event.ctrlKey || event.altKey || event.metaKey) return null;
  const { key } = event;

  if (isDigit(key)) return { type: 'DIGIT', digit: key };
  const op = BINARY_KEYS[key];
  if (op) return { type: 'BINARY_OP', op };

  switch (key) {
    case '.':
    case ',':
      return { type: 'DECIMAL' };
    case 'r':
    case 'R':
      return { type: 'UNARY_OP', op: 'sqrt' };
    case 'Enter':
    case '=':
      return { type: 'EQUALS' };
    case 'Backspace':
      return { type: 'BACKSPACE' };
    case 'Escape':
    case 'Delete':
      return { type: 'CLEAR' };
    default:
      return null;
  }
}
