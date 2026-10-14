import type { ApiError, OperationName } from '../../../api/types';

export type BinaryOp = Exclude<OperationName, 'sqrt'>;
export type UnaryOp = Extract<OperationName, 'sqrt'>;

export type Digit = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9';

/** A binary operator waiting for its right operand. */
export interface Pending {
  op: BinaryOp;
  left: number;
}

/** The effect an `evaluating` state asks the hook to run (effects as data). */
export interface EvalRequest {
  id: number;
  op: OperationName;
  operands: number[];
}

/** What to do with the result of a successful request. */
export type Continuation =
  /** Rule 4: the result becomes the left operand and `next` is pending. */
  | { kind: 'chain'; next: BinaryOp }
  /** Rule 5: show the result, clear the pending operator. */
  | { kind: 'equals' }
  /** Rule 6: the result replaces the current value; the pending operator is kept. */
  | { kind: 'unary-operand' }
  /** Rule 12c: the result replaces the committed left operand. */
  | { kind: 'unary-left' };

interface Base {
  /** Id of the next request. Lives in every status and is never reset (12f). */
  nextId: number;
}

export type State =
  | (Base & {
      status: 'editing';
      /** Digits as typed. `null`: a committed left operand, no digits typed yet. */
      input: string | null;
      pending: Pending | null;
    })
  | (Base & { status: 'result'; value: number; pending: Pending | null })
  | (Base & {
      status: 'evaluating';
      request: EvalRequest;
      then: Continuation;
      pending: Pending | null;
      /** The main-line text that was on screen when the request started. */
      shown: string;
    })
  | (Base & { status: 'error'; error: ApiError });

export type Action =
  | { type: 'DIGIT'; digit: Digit }
  | { type: 'DECIMAL' }
  | { type: 'TOGGLE_SIGN' }
  | { type: 'BACKSPACE' }
  | { type: 'CLEAR' }
  | { type: 'BINARY_OP'; op: BinaryOp }
  | { type: 'UNARY_OP'; op: UnaryOp }
  | { type: 'EQUALS' }
  | { type: 'EVAL_SUCCESS'; id: number; result: number }
  | { type: 'EVAL_FAILURE'; id: number; error: ApiError };
