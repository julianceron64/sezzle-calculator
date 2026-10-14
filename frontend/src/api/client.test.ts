import { delay, http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { apiUrl, server } from '../test/msw';
import { calculate } from './client';

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
});

const problem = (code: string, status: number) =>
  HttpResponse.json(
    {
      type: `/problems/${code.toLowerCase()}`,
      title: 'Title',
      status,
      detail: 'Detail',
      code,
    },
    { status, headers: { 'Content-Type': 'application/problem+json' } },
  );

// FE-03
describe('calculate: success', () => {
  it('posts named operands as JSON and returns the response body', async () => {
    let seen: {
      method: string;
      contentType: string | null;
      body: unknown;
    } | null = null;
    server.use(
      http.post(apiUrl('add'), async ({ request }) => {
        seen = {
          method: request.method,
          contentType: request.headers.get('Content-Type'),
          body: await request.json(),
        };
        return HttpResponse.json({ operation: 'add', a: 1, b: 2, result: 3 });
      }),
    );

    const result = await calculate('add', [1, 2]);

    expect(result).toEqual({
      ok: true,
      value: { operation: 'add', a: 1, b: 2, result: 3 },
    });
    expect(seen).toEqual({
      method: 'POST',
      contentType: 'application/json',
      body: { a: 1, b: 2 },
    });
  });

  it('sends only `a` for a unary operation and accepts a response without `b`', async () => {
    let body: unknown;
    server.use(
      http.post(apiUrl('sqrt'), async ({ request }) => {
        body = await request.json();
        return HttpResponse.json({ operation: 'sqrt', a: 9, result: 3 });
      }),
    );

    const result = await calculate('sqrt', [9]);

    expect(body).toEqual({ a: 9 });
    expect(result).toEqual({
      ok: true,
      value: { operation: 'sqrt', a: 9, result: 3 },
    });
  });
});

// FE-03
describe('calculate: errors', () => {
  it('maps a problem+json response to its backend code and status', async () => {
    server.use(
      http.post(apiUrl('divide'), () => problem('DIVISION_BY_ZERO', 422)),
    );
    expect(await calculate('divide', [1, 0])).toEqual({
      ok: false,
      error: { code: 'DIVISION_BY_ZERO', status: 422 },
    });
  });

  it('maps an unrecognised problem code to UNKNOWN_ERROR with the status', async () => {
    server.use(http.post(apiUrl('add'), () => problem('BRAND_NEW_CODE', 400)));
    expect(await calculate('add', [1, 2])).toEqual({
      ok: false,
      error: { code: 'UNKNOWN_ERROR', status: 400 },
    });
  });

  it('maps a malformed problem body to UNKNOWN_ERROR', async () => {
    server.use(
      http.post(
        apiUrl('add'),
        () =>
          new HttpResponse('{not json', {
            status: 400,
            headers: { 'Content-Type': 'application/problem+json' },
          }),
      ),
    );
    expect(await calculate('add', [1, 2])).toEqual({
      ok: false,
      error: { code: 'UNKNOWN_ERROR', status: 400 },
    });
  });

  it('maps a non-problem error response to UNKNOWN_ERROR', async () => {
    server.use(
      http.post(
        apiUrl('add'),
        () =>
          new HttpResponse('<h1>Bad gateway</h1>', {
            status: 502,
            headers: { 'Content-Type': 'text/html' },
          }),
      ),
    );
    expect(await calculate('add', [1, 2])).toEqual({
      ok: false,
      error: { code: 'UNKNOWN_ERROR', status: 502 },
    });
  });

  it.each([
    ['an empty object', {}],
    ['a string result', { operation: 'add', a: 1, b: 2, result: '3' }],
    ['an unknown operation', { operation: 'mod', a: 1, b: 2, result: 3 }],
    ['a missing result', { operation: 'add', a: 1, b: 2 }],
    ['a non-numeric b', { operation: 'add', a: 1, b: '2', result: 3 }],
    ['an array', [1, 2, 3]],
    ['null', null],
  ])('reports INVALID_RESPONSE for a 2xx with %s', async (_name, body) => {
    server.use(http.post(apiUrl('add'), () => HttpResponse.json(body)));
    expect(await calculate('add', [1, 2])).toEqual({
      ok: false,
      error: { code: 'INVALID_RESPONSE' },
    });
  });

  it('reports INVALID_RESPONSE for a 2xx that is not JSON', async () => {
    server.use(http.post(apiUrl('add'), () => new HttpResponse('ok')));
    expect(await calculate('add', [1, 2])).toEqual({
      ok: false,
      error: { code: 'INVALID_RESPONSE' },
    });
  });

  it('reports NETWORK_ERROR when the request fails', async () => {
    server.use(http.post(apiUrl('add'), () => HttpResponse.error()));
    expect(await calculate('add', [1, 2])).toEqual({
      ok: false,
      error: { code: 'NETWORK_ERROR' },
    });
  });

  it('reports TIMEOUT when the server does not answer within timeoutMs', async () => {
    server.use(
      http.post(apiUrl('add'), async () => {
        await delay('infinite');
        return HttpResponse.json({});
      }),
    );
    expect(await calculate('add', [1, 2], { timeoutMs: 30 })).toEqual({
      ok: false,
      error: { code: 'TIMEOUT' },
    });
  });

  it('reports ABORTED when the caller aborts an in-flight request', async () => {
    let markStarted: () => void = () => {};
    const started = new Promise<void>((resolve) => {
      markStarted = resolve;
    });
    server.use(
      http.post(apiUrl('add'), async () => {
        markStarted();
        await delay('infinite');
        return HttpResponse.json({});
      }),
    );
    const controller = new AbortController();

    const pending = calculate('add', [1, 2], { signal: controller.signal });
    await started;
    controller.abort();

    expect(await pending).toEqual({ ok: false, error: { code: 'ABORTED' } });
  });

  it('reports ABORTED for a signal that is already aborted', async () => {
    const controller = new AbortController();
    controller.abort();
    expect(
      await calculate('add', [1, 2], { signal: controller.signal }),
    ).toEqual({ ok: false, error: { code: 'ABORTED' } });
  });
});

describe('calculate: configuration', () => {
  it('uses a 5000 ms timeout by default', async () => {
    const timeout = vi.spyOn(AbortSignal, 'timeout');
    server.use(
      http.post(apiUrl('add'), () =>
        HttpResponse.json({ operation: 'add', a: 1, b: 2, result: 3 }),
      ),
    );
    await calculate('add', [1, 2]);
    expect(timeout).toHaveBeenCalledWith(5000);
  });

  it('is same-origin when VITE_API_BASE_URL is not set', async () => {
    vi.stubEnv('VITE_API_BASE_URL', '');
    const fetchSpy = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValue(
        Response.json({ operation: 'add', a: 1, b: 2, result: 3 }),
      );
    await calculate('add', [1, 2]);
    expect(fetchSpy.mock.calls[0]?.[0]).toBe('/api/v1/add');
  });
});
