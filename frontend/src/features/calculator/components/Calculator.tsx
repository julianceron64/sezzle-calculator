import { errorMessage } from '../errorMessages';
import { getDisplay, getExpression } from '../model/reducer';
import { useCalculator } from '../useCalculator';
import styles from './Calculator.module.css';
import { Display } from './Display';
import { Keypad } from './Keypad';

interface CalculatorProps {
  /** Forwarded to the API client; tests inject a short timeout. */
  timeoutMs?: number;
}

export function Calculator({ timeoutMs }: CalculatorProps) {
  const { state, dispatch } = useCalculator({ timeoutMs });

  return (
    <main className={styles.card}>
      <h1 className={styles.title}>Calculator</h1>
      <Display
        expression={getExpression(state)}
        value={getDisplay(state)}
        error={state.status === 'error' ? errorMessage(state.error) : null}
      />
      <Keypad busy={state.status === 'evaluating'} dispatch={dispatch} />
    </main>
  );
}
