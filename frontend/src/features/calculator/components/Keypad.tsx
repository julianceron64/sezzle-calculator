import type { Dispatch } from 'react';
import type { Action, Digit } from '../model/types';
import { Key, type KeyVariant } from './Key';
import styles from './Keypad.module.css';

interface KeySpec {
  label: string;
  name?: string;
  action: Action;
  variant: KeyVariant;
  wide?: boolean;
}

const digit = (d: Digit, wide = false): KeySpec => ({
  label: d,
  action: { type: 'DIGIT', digit: d },
  variant: 'digit',
  wide,
});

const KEYS: KeySpec[] = [
  {
    label: 'xʸ',
    name: 'power',
    action: { type: 'BINARY_OP', op: 'power' },
    variant: 'function',
  },
  {
    label: '√',
    name: 'square root',
    action: { type: 'UNARY_OP', op: 'sqrt' },
    variant: 'function',
  },
  {
    label: '%',
    name: 'percent of',
    action: { type: 'BINARY_OP', op: 'percentage' },
    variant: 'function',
    wide: true,
  },
  { label: 'C', name: 'clear', action: { type: 'CLEAR' }, variant: 'function' },
  {
    label: '⌫',
    name: 'backspace',
    action: { type: 'BACKSPACE' },
    variant: 'function',
  },
  {
    label: '±',
    name: 'toggle sign',
    action: { type: 'TOGGLE_SIGN' },
    variant: 'function',
  },
  {
    label: '÷',
    name: 'divide',
    action: { type: 'BINARY_OP', op: 'divide' },
    variant: 'operator',
  },
  digit('7'),
  digit('8'),
  digit('9'),
  {
    label: '×',
    name: 'multiply',
    action: { type: 'BINARY_OP', op: 'multiply' },
    variant: 'operator',
  },
  digit('4'),
  digit('5'),
  digit('6'),
  {
    label: '−',
    name: 'subtract',
    action: { type: 'BINARY_OP', op: 'subtract' },
    variant: 'operator',
  },
  digit('1'),
  digit('2'),
  digit('3'),
  {
    label: '+',
    name: 'add',
    action: { type: 'BINARY_OP', op: 'add' },
    variant: 'operator',
  },
  digit('0', true),
  {
    label: '.',
    name: 'decimal point',
    action: { type: 'DECIMAL' },
    variant: 'digit',
  },
  {
    label: '=',
    name: 'equals',
    action: { type: 'EQUALS' },
    variant: 'equals',
  },
];

interface KeypadProps {
  busy: boolean;
  dispatch: Dispatch<Action>;
}

export function Keypad({ busy, dispatch }: KeypadProps) {
  return (
    <div className={styles.keypad}>
      {KEYS.map((key) => (
        <Key
          key={key.label}
          label={key.label}
          name={key.name}
          variant={key.variant}
          wide={key.wide}
          // C stays enabled so a pending request can always be cancelled.
          busy={busy && key.action.type !== 'CLEAR'}
          onPress={() => dispatch(key.action)}
        />
      ))}
    </div>
  );
}
