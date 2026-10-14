const SIGNIFICANT_DIGITS = 12;
const EXPONENTIAL_DIGITS = 10;

// toExponential always prints "d.ddde±x"; the defaults only satisfy noUncheckedIndexedAccess.
function splitExponential(value: number, fractionDigits: number) {
  const [mantissa = '', exponent = '0'] = value
    .toExponential(fractionDigits)
    .split('e');
  return { mantissa, exp: Number(exponent), exponentText: exponent };
}

const trimFraction = (s: string) => s.replace(/0+$/, '');

/**
 * Display formatting (ADR-006, spec 003): 12 significant digits in plain
 * notation, exponential with 10 significant digits at the extremes. The
 * plain/exponential threshold is decided on the *rounded* value. Never uses
 * toPrecision, which switches to exponential on its own.
 */
export function formatNumber(n: number): string {
  if (n === 0) return '0';
  const sign = n < 0 ? '-' : '';
  const abs = Math.abs(n);

  const rounded = splitExponential(abs, SIGNIFICANT_DIGITS - 1);
  if (rounded.exp >= 12 || rounded.exp < -9) {
    const { mantissa, exponentText } = splitExponential(
      abs,
      EXPONENTIAL_DIGITS - 1,
    );
    const [whole = '', fraction = ''] = mantissa.split('.');
    const trimmed = trimFraction(fraction);
    return `${sign}${whole}${trimmed ? `.${trimmed}` : ''}e${exponentText}`;
  }

  const digits = rounded.mantissa.replace('.', '');
  if (rounded.exp >= 0) {
    const whole = digits.slice(0, rounded.exp + 1);
    const fraction = trimFraction(digits.slice(rounded.exp + 1));
    return `${sign}${whole}${fraction ? `.${fraction}` : ''}`;
  }
  const fraction = trimFraction('0'.repeat(-rounded.exp - 1) + digits);
  return `${sign}0.${fraction}`;
}
