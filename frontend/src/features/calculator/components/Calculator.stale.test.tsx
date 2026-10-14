import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { calculate } from '../../../api/client';
import type { ApiError, CalculateResponse, Result } from '../../../api/types';
import { Calculator } from './Calculator';

// The real client rejects as soon as the hook aborts, so a late body can never
// reach the hook through it. This file replaces `calculate` with a fake that
// ignores the abort signal, to prove the reducer's id guard (rules 9 and 12f).
vi.mock('../../../api/client', () => ({ calculate: vi.fn() }));

type Outcome = Result<CalculateResponse, ApiError>;

function pendingCall() {
  let settle: (outcome: Outcome) => void = () => {};
  const promise = new Promise<Outcome>((resolve) => {
    settle = resolve;
  });
  return { promise, settle };
}

const success = (result: number): Outcome => ({
  ok: true,
  value: { operation: 'add', a: 0, result },
});

const display = () => screen.getByRole('status');

beforeEach(() => {
  vi.mocked(calculate).mockReset();
});

// FE-09, rules 9 and 12f
describe('Calculator: stale responses', () => {
  it('ignores a late response of a cleared request that reaches the hook', async () => {
    const first = pendingCall();
    const second = pendingCall();
    vi.mocked(calculate)
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);
    const user = userEvent.setup();
    render(<Calculator />);
    const press = async (...names: string[]) => {
      for (const name of names) {
        await user.click(screen.getByRole('button', { name }));
      }
    };

    await press('1', 'add', '2', 'equals'); // request id 1, never aborted by the fake
    await press('clear');
    expect(display()).toHaveTextContent(/^0$/);

    await press('3', 'multiply', '3', 'equals'); // request id 2
    expect(calculate).toHaveBeenCalledTimes(2);
    expect(screen.getByRole('button', { name: '7' })).toHaveAttribute(
      'aria-disabled',
      'true',
    );

    // The late answer to request 1 arrives while request 2 is in flight.
    await act(async () => {
      first.settle(success(99));
      await first.promise;
    });
    expect(display()).toHaveTextContent(/^3$/); // would be 99 without the id guard
    expect(screen.getByRole('button', { name: '7' })).toHaveAttribute(
      'aria-disabled',
      'true',
    );

    await act(async () => {
      second.settle(success(9));
      await second.promise;
    });
    expect(display()).toHaveTextContent(/^9$/);
  });

  it('ignores a late failure of a cleared request too', async () => {
    const first = pendingCall();
    const second = pendingCall();
    vi.mocked(calculate)
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);
    const user = userEvent.setup();
    render(<Calculator />);
    const press = async (...names: string[]) => {
      for (const name of names) {
        await user.click(screen.getByRole('button', { name }));
      }
    };

    await press('1', 'divide', '0', 'equals');
    await press('clear');
    await press('4', 'add', '4', 'equals');

    await act(async () => {
      first.settle({
        ok: false,
        error: { code: 'DIVISION_BY_ZERO', status: 422 },
      });
      await first.promise;
    });
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(display()).toHaveTextContent(/^4$/);
  });
});
