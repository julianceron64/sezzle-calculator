import { describe, expect, it } from 'vitest';
import { keyToAction } from './keymap';
import type { Action } from './types';

const press = (key: string, modifiers: Partial<KeyboardEvent> = {}) =>
  keyToAction({
    key,
    ctrlKey: false,
    altKey: false,
    metaKey: false,
    ...modifiers,
  });

// FE-08
describe('keyToAction', () => {
  const cases: [string, Action][] = [
    ['0', { type: 'DIGIT', digit: '0' }],
    ['7', { type: 'DIGIT', digit: '7' }],
    ['9', { type: 'DIGIT', digit: '9' }],
    ['.', { type: 'DECIMAL' }],
    [',', { type: 'DECIMAL' }],
    ['+', { type: 'BINARY_OP', op: 'add' }],
    ['-', { type: 'BINARY_OP', op: 'subtract' }],
    ['*', { type: 'BINARY_OP', op: 'multiply' }],
    ['/', { type: 'BINARY_OP', op: 'divide' }],
    ['^', { type: 'BINARY_OP', op: 'power' }],
    ['%', { type: 'BINARY_OP', op: 'percentage' }],
    ['r', { type: 'UNARY_OP', op: 'sqrt' }],
    ['R', { type: 'UNARY_OP', op: 'sqrt' }],
    ['Enter', { type: 'EQUALS' }],
    ['=', { type: 'EQUALS' }],
    ['Backspace', { type: 'BACKSPACE' }],
    ['Escape', { type: 'CLEAR' }],
    ['Delete', { type: 'CLEAR' }],
  ];

  it.each(cases)('maps %s', (key, action) => {
    expect(press(key)).toEqual(action);
  });

  it.each(['a', 'x', 'Tab', ' ', 'Shift', 'ArrowUp', 'F5', '10'])(
    'does not handle %j',
    (key) => {
      expect(press(key)).toBeNull();
    },
  );

  it.each(['ctrlKey', 'altKey', 'metaKey'] as const)(
    'ignores keys while %s is held',
    (modifier) => {
      expect(press('1', { [modifier]: true })).toBeNull();
      expect(press('Enter', { [modifier]: true })).toBeNull();
    },
  );

  it('still handles keys typed with Shift (+, *, ^ and % need it on many layouts)', () => {
    expect(press('+', { shiftKey: true })).toEqual({
      type: 'BINARY_OP',
      op: 'add',
    });
    expect(press('%', { shiftKey: true })).toEqual({
      type: 'BINARY_OP',
      op: 'percentage',
    });
  });
});
