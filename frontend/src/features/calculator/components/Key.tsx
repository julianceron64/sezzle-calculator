import styles from './Key.module.css';

export type KeyVariant = 'digit' | 'operator' | 'function' | 'equals';

interface KeyProps {
  label: string;
  /** Accessible name for symbol keys; digits use their own number. */
  name?: string;
  variant: KeyVariant;
  wide?: boolean;
  /** While a request is pending: aria-disabled, never the native attribute, so focus is kept. */
  busy: boolean;
  onPress: () => void;
}

export function Key({ label, name, variant, wide, busy, onPress }: KeyProps) {
  const className = [styles.key, styles[variant], wide ? styles.wide : '']
    .filter(Boolean)
    .join(' ');
  return (
    <button
      type="button"
      className={className}
      aria-label={name}
      aria-disabled={busy ? true : undefined}
      onClick={onPress}
    >
      {label}
    </button>
  );
}
