import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { delay, http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import type { OperationName } from '../../../api/types';
import { apiUrl, server } from '../../../test/msw';
import { Calculator } from './Calculator';

/** Accessible names from spec 003. Digits use their own number. */
const NAMES = {
  '÷': 'divide',
  '×': 'multiply',
  '−': 'subtract',
  '+': 'add',
  xʸ: 'power',
  '√': 'square root',
  '%': 'percent of',
  '±': 'toggle sign',
  '⌫': 'backspace',
  C: 'clear',
  '.': 'decimal point',
  '=': 'equals',
} as const;

type CalcRequest = { operation: OperationName; body: unknown };

/** Mocks operations with fixed results and records requests in order. */
function mockResults(results: Partial<Record<OperationName, number>>) {
  const requests: CalcRequest[] = [];
  for (const [operation, result] of Object.entries(results)) {
    server.use(
      http.post(apiUrl(operation), async ({ request }) => {
        const body: unknown = await request.json();
        requests.push({ operation: operation as OperationName, body });
        return HttpResponse.json({ operation, result, ...(body as object) });
      }),
    );
  }
  return requests;
}

function mockProblem(operation: OperationName, code: string, status: number) {
  server.use(
    http.post(apiUrl(operation), () =>
      HttpResponse.json(
        { type: `/problems/x`, title: 'T', status, detail: 'D', code },
        { status, headers: { 'Content-Type': 'application/problem+json' } },
      ),
    ),
  );
}

function setup(props: { timeoutMs?: number } = {}) {
  const user = userEvent.setup();
  render(<Calculator {...props} />);
  const key = (label: string) =>
    screen.getByRole('button', {
      name: label in NAMES ? NAMES[label as keyof typeof NAMES] : label,
    });
  /** Clicks keys by label, e.g. click('1', '2', '+', '3', '='). */
  const click = async (...labels: string[]) => {
    for (const label of labels) await user.click(key(label));
  };
  return { user, key, click };
}

const display = () => screen.getByRole('status');

describe('Calculator: basic flows', () => {
  // FE-05
  it('1 2 + 3 = shows 15 and sends {a:12,b:3} to /api/v1/add', async () => {
    const requests = mockResults({ add: 15 });
    const { click } = setup();

    await click('1', '2', '+', '3', '=');

    await waitFor(() => expect(display()).toHaveTextContent(/^15$/));
    expect(requests).toEqual([{ operation: 'add', body: { a: 12, b: 3 } }]);
  });

  // FE-06
  it('chains 2 + 3 × 4 = as two requests in order and shows the final result', async () => {
    const requests = mockResults({ add: 5, multiply: 20 });
    const { click } = setup();

    await click('2', '+', '3', '×');
    await waitFor(() => expect(display()).toHaveTextContent(/^5$/));
    await click('4', '=');

    await waitFor(() => expect(display()).toHaveTextContent(/^20$/));
    expect(requests).toEqual([
      { operation: 'add', body: { a: 2, b: 3 } },
      { operation: 'multiply', body: { a: 5, b: 4 } },
    ]);
  });

  it('shows the pending operation on the secondary line', async () => {
    const { click } = setup();
    await click('2', '+');
    expect(screen.getByText('2 +')).toBeInTheDocument();
  });

  it('applies √ to the typed value: 2 + √ 9 = shows 5', async () => {
    const requests = mockResults({ sqrt: 3, add: 5 });
    const { click } = setup();

    await click('2', '+', '9', '√');
    await waitFor(() => expect(display()).toHaveTextContent(/^3$/));
    await click('=');

    await waitFor(() => expect(display()).toHaveTextContent(/^5$/));
    expect(requests).toEqual([
      { operation: 'sqrt', body: { a: 9 } },
      { operation: 'add', body: { a: 2, b: 3 } },
    ]);
  });

  it('sends percent of: 20 % 150 =', async () => {
    const requests = mockResults({ percentage: 30 });
    const { click } = setup();
    await click('2', '0', '%', '1', '5', '0', '=');
    await waitFor(() => expect(display()).toHaveTextContent(/^30$/));
    expect(requests).toEqual([
      { operation: 'percentage', body: { a: 20, b: 150 } },
    ]);
  });

  it('edits input with ±, ⌫ and the decimal point', async () => {
    const { click } = setup();
    await click('1', '.', '5', '±');
    expect(display()).toHaveTextContent(/^-1\.5$/);
    await click('⌫', '⌫');
    expect(display()).toHaveTextContent(/^-1$/);
  });

  it('formats long results for display (0.1 + 0.2 shows 0.3)', async () => {
    server.use(
      http.post(apiUrl('add'), () =>
        HttpResponse.json({
          operation: 'add',
          a: 0.1,
          b: 0.2,
          result: 0.30000000000000004,
        }),
      ),
    );
    const { click } = setup();
    await click('0', '.', '1', '+', '0', '.', '2', '=');
    await waitFor(() => expect(display()).toHaveTextContent(/^0\.3$/));
  });
});

// FE-11
describe('Calculator: keys', () => {
  it('exposes every key by its accessible name', () => {
    setup();
    for (const digit of '0123456789') {
      expect(screen.getByRole('button', { name: digit })).toBeEnabled();
    }
    for (const name of Object.values(NAMES)) {
      expect(screen.getByRole('button', { name })).toBeInTheDocument();
    }
    expect(screen.getAllByRole('button')).toHaveLength(22);
  });

  it('renders every key as a button of type="button"', () => {
    setup();
    for (const button of screen.getAllByRole('button')) {
      expect(button).toHaveAttribute('type', 'button');
    }
  });
});

describe('Calculator: errors', () => {
  // FE-07
  it('shows the mapped message in an alert, and the next digit recovers', async () => {
    mockProblem('divide', 'DIVISION_BY_ZERO', 422);
    const { click } = setup();

    await click('1', '÷', '0', '=');

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Cannot divide by zero.',
    );
    expect(display()).toHaveTextContent(/^Error$/);

    await click('7');
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(display()).toHaveTextContent(/^7$/);
  });

  it('maps negative square roots by code', async () => {
    mockProblem('sqrt', 'NEGATIVE_SQUARE_ROOT', 422);
    const { click } = setup();
    await click('5', '±', '√');
    expect(await screen.findByRole('alert')).toHaveTextContent(
      /square root of a negative number/i,
    );
  });

  it('C clears an error', async () => {
    mockProblem('divide', 'DIVISION_BY_ZERO', 422);
    const { click } = setup();
    await click('1', '÷', '0', '=');
    await screen.findByRole('alert');
    await click('C');
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(display()).toHaveTextContent(/^0$/);
  });

  // FE-10
  it('shows the network error message', async () => {
    server.use(http.post(apiUrl('add'), () => HttpResponse.error()));
    const { click } = setup();
    await click('1', '+', '2', '=');
    expect(await screen.findByRole('alert')).toHaveTextContent(
      /could not reach the server/i,
    );
  });

  // FE-10
  it('shows the timeout message (short injected timeoutMs)', async () => {
    server.use(
      http.post(apiUrl('add'), async () => {
        await delay('infinite');
        return HttpResponse.json({});
      }),
    );
    const { click } = setup({ timeoutMs: 30 });
    await click('1', '+', '2', '=');
    expect(await screen.findByRole('alert')).toHaveTextContent(
      /took too long/i,
    );
  });

  it('falls back to generic copy for an unexpected server error', async () => {
    server.use(
      http.post(apiUrl('add'), () => new HttpResponse('boom', { status: 500 })),
    );
    const { click } = setup();
    await click('1', '+', '2', '=');
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Something went wrong. Please try again.',
    );
  });
});

/** A response the test releases by hand, to hold a request in flight. */
function deferredResponse() {
  let release: () => void = () => {};
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });
  let markStarted: () => void = () => {};
  const started = new Promise<void>((resolve) => {
    markStarted = resolve;
  });
  return { gate, release: () => release(), started, markStarted };
}

// FE-09
describe('Calculator: while a request is pending', () => {
  it('uses aria-disabled (not disabled), ignores presses, keeps C enabled', async () => {
    const pending = deferredResponse();
    server.use(
      http.post(apiUrl('add'), async () => {
        pending.markStarted();
        await pending.gate;
        return HttpResponse.json({ operation: 'add', a: 1, b: 2, result: 3 });
      }),
    );
    const { user, key, click } = setup();

    await click('1', '+', '2', '=');
    await pending.started;

    for (const label of ['7', '+', '=', '√', '±', '⌫', '.', 'xʸ']) {
      expect(key(label)).toHaveAttribute('aria-disabled', 'true');
      expect(key(label)).not.toHaveAttribute('disabled');
    }
    expect(key('C')).not.toHaveAttribute('aria-disabled', 'true');

    await user.click(key('7'));
    await user.keyboard('9+=');
    expect(display()).toHaveTextContent(/^2$/);

    pending.release();
    await waitFor(() => expect(display()).toHaveTextContent(/^3$/));
    expect(key('7')).not.toHaveAttribute('aria-disabled', 'true');
  });

  it('keeps keyboard focus on a key that becomes aria-disabled', async () => {
    const pending = deferredResponse();
    server.use(
      http.post(apiUrl('add'), async () => {
        pending.markStarted();
        await pending.gate;
        return HttpResponse.json({ operation: 'add', a: 1, b: 2, result: 3 });
      }),
    );
    const { key, click } = setup();
    await click('1', '+', '2');
    key('=').focus();
    await click('=');
    await pending.started;
    expect(key('=')).toHaveFocus();
    pending.release();
    await waitFor(() => expect(display()).toHaveTextContent(/^3$/));
  });

  // The real client rejects once aborted, so a late body never reaches the
  // hook here; the id guard itself is proven in Calculator.stale.test.tsx.
  it('C aborts the in-flight request and the next request completes normally', async () => {
    const first = deferredResponse();
    let firstSignal: AbortSignal | undefined;
    server.use(
      http.post(apiUrl('add'), async ({ request }) => {
        firstSignal = request.signal;
        first.markStarted();
        await first.gate;
        return HttpResponse.json({ operation: 'add', a: 1, b: 2, result: 3 });
      }),
    );
    const { click } = setup();

    await click('1', '+', '2', '=');
    await first.started;
    await click('C');
    expect(display()).toHaveTextContent(/^0$/);

    // A second request, answered after the first one is released late.
    const requests = mockResults({ multiply: 9 });
    await click('3', '×', '3', '=');
    first.release();
    await waitFor(() => expect(display()).toHaveTextContent(/^9$/));
    expect(requests).toEqual([{ operation: 'multiply', body: { a: 3, b: 3 } }]);
    expect(firstSignal?.aborted).toBe(true);
  });
});

// FE-08
describe('Calculator: keyboard only', () => {
  it('12+3= works with the keyboard', async () => {
    const requests = mockResults({ add: 15 });
    const { user } = setup();
    await user.keyboard('12+3=');
    await waitFor(() => expect(display()).toHaveTextContent(/^15$/));
    expect(requests).toEqual([{ operation: 'add', body: { a: 12, b: 3 } }]);
  });

  it('Enter equals, - and * and / are operators, comma is a decimal point', async () => {
    const requests = mockResults({ subtract: 4, multiply: 8, divide: 2 });
    const { user } = setup();

    await user.keyboard('9-5');
    await user.keyboard('{Enter}');
    await waitFor(() => expect(display()).toHaveTextContent(/^4$/));
    await user.keyboard('{Escape}');
    await user.keyboard('2*4=');
    await waitFor(() => expect(display()).toHaveTextContent(/^8$/));
    await user.keyboard('{Delete}');
    await user.keyboard('1,5/3=');

    await waitFor(() => expect(display()).toHaveTextContent(/^2$/));
    expect(requests).toEqual([
      { operation: 'subtract', body: { a: 9, b: 5 } },
      { operation: 'multiply', body: { a: 2, b: 4 } },
      { operation: 'divide', body: { a: 1.5, b: 3 } },
    ]);
  });

  it('^, % and r use power, percent of and square root', async () => {
    const requests = mockResults({ power: 8, percentage: 5, sqrt: 3 });
    const { user } = setup();
    await user.keyboard('2^3=');
    await waitFor(() => expect(display()).toHaveTextContent(/^8$/));
    await user.keyboard('{Escape}10%50=');
    await waitFor(() => expect(display()).toHaveTextContent(/^5$/));
    await user.keyboard('{Escape}9r');
    await waitFor(() => expect(display()).toHaveTextContent(/^3$/));
    expect(requests.map((r) => r.operation)).toEqual([
      'power',
      'percentage',
      'sqrt',
    ]);
  });

  it('Backspace edits and the error flow recovers from the keyboard', async () => {
    mockProblem('divide', 'DIVISION_BY_ZERO', 422);
    const { user } = setup();
    await user.keyboard('12{Backspace}');
    expect(display()).toHaveTextContent(/^1$/);
    await user.keyboard('/0=');
    expect(await screen.findByRole('alert')).toHaveTextContent(
      /divide by zero/i,
    );
    await user.keyboard('5');
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(display()).toHaveTextContent(/^5$/);
  });

  it('removes the listener on unmount', async () => {
    const { unmount } = render(<Calculator />);
    unmount();
    // A key after unmount must not throw or dispatch into a dead component.
    expect(fireEvent.keyDown(window, { key: '5' })).toBe(true);
  });
});

// FE-13
describe('Calculator: preventDefault on handled keys', () => {
  it('prevents the default action of handled keys only', () => {
    setup();
    for (const key of [
      '5',
      '+',
      'Enter',
      '=',
      'Backspace',
      'Escape',
      'r',
      ',',
    ]) {
      expect(fireEvent.keyDown(window, { key })).toBe(false); // false: defaultPrevented
    }
    for (const key of ['a', 'Tab', 'F5']) {
      expect(fireEvent.keyDown(window, { key })).toBe(true);
    }
    expect(fireEvent.keyDown(window, { key: '5', ctrlKey: true })).toBe(true);
    expect(fireEvent.keyDown(window, { key: '5', metaKey: true })).toBe(true);
  });

  it('Enter on a focused digit key triggers only EQUALS, not that key', async () => {
    const { user, key, click } = setup();
    await click('5');
    key('7').focus();
    await user.keyboard('{Enter}');
    expect(display()).toHaveTextContent(/^5$/); // would be 57 if the button also clicked
  });
});
