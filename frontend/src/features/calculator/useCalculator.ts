import { useEffect, useReducer } from 'react';
import { calculate } from '../../api/client';
import { keyToAction } from './model/keymap';
import { initialState, reduce } from './model/reducer';

interface Options {
  /** Request timeout; tests inject a short one. Defaults to the client's 5 s. */
  timeoutMs?: number;
}

/**
 * Owns every side effect of the calculator: it runs the request the reducer
 * describes (aborting it when the state moves on) and listens to the keyboard.
 */
export function useCalculator({ timeoutMs }: Options = {}) {
  const [state, dispatch] = useReducer(reduce, initialState);

  // `state` keeps the same object for ignored actions, so `request` is stable
  // for as long as one request is in flight.
  const request = state.status === 'evaluating' ? state.request : null;
  useEffect(() => {
    if (!request) return;
    const controller = new AbortController();
    void calculate(request.op, request.operands, {
      signal: controller.signal,
      timeoutMs,
    }).then((result) => {
      if (result.ok) {
        dispatch({
          type: 'EVAL_SUCCESS',
          id: request.id,
          result: result.value.result,
        });
      } else if (result.error.code !== 'ABORTED') {
        dispatch({ type: 'EVAL_FAILURE', id: request.id, error: result.error });
      }
    });
    return () => controller.abort(); // CLEAR or unmount cancels the request
  }, [request, timeoutMs]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const action = keyToAction(event);
      if (!action) return;
      event.preventDefault(); // e.g. Enter on a focused button must not also click it
      dispatch(action);
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  return { state, dispatch };
}
