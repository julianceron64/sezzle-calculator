import styles from './Display.module.css';

interface DisplayProps {
  /** Pending operation, e.g. "2 +". */
  expression: string;
  value: string;
  /** User-facing error copy, shown below the display with role="alert". */
  error: string | null;
}

export function Display({ expression, value, error }: DisplayProps) {
  return (
    <div className={styles.display}>
      <div className={styles.expression}>{expression}</div>
      <div role="status" aria-live="polite" className={styles.value}>
        {value}
      </div>
      {error && (
        <p role="alert" className={styles.error}>
          {error}
        </p>
      )}
    </div>
  );
}
