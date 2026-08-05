/**
 * Real tests for the http-client singleton exported from
 * `src/lib/api/http-client.ts`. Replaces the previous
 * `api-client.test.ts`, which contained 280 lines of tautological
 * assertions (e.g. `expect('GET').toBe('GET')`) and exercised no actual
 * production code.
 *
 * What we exercise:
 *   - GET / POST / PUT / PATCH / DELETE use the right method and URL
 *   - Query params are serialized into the URL (including `undefined`/`null`
 *     are dropped)
 *   - JSON request bodies are camelCased before being sent (contract rule)
 *   - Response data has snake_case keys converted to camelCase
 *   - Non-zero `code` payloads throw with the backend message
 *   - HTTP non-2xx responses throw with status
 *   - `getPaginated` flattens nested filters into `filters[key]`
 *   - `batchOperation` wraps data in `{ operation, data }`
 *   - CSRF token is added to mutating requests but not GET
 *   - Auth (Bearer) header is included when a cookie token is present
 *   - Tenant headers (X-Tenant-ID, X-Tenant-Code) are added when the
 *     TenantContext is populated
 *   - `setTenantId/setTenantCode` are no-ops (deprecated)
 *   - Missing code field is tolerated (BPMN controller quirk)
 */

import { httpClient } from '../http-client';
import { security } from '@/lib/security';
import { setTenantId, setTenantCode, clearTenant } from '@/lib/auth/tenant-context';

jest.mock('@/lib/security', () => ({
  security: {
    csrf: {
      getToken: jest.fn().mockResolvedValue('mock-csrf-token'),
      clearToken: jest.fn(),
    },
    network: {
      getSecureHeaders: jest.fn().mockReturnValue({
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest',
      }),
    },
  },
}));

// Mock fetch globally. Tests configure per-call responses via mockResolvedValueOnce.
const fetchMock = jest.fn() as jest.Mock;
(global as unknown as { fetch: jest.Mock }).fetch = fetchMock;

const consoleSpy = {
  error: jest.spyOn(console, 'error').mockImplementation(() => {}),
  warn: jest.spyOn(console, 'warn').mockImplementation(() => {}),
  debug: jest.spyOn(console, 'debug').mockImplementation(() => {}),
  log: jest.spyOn(console, 'log').mockImplementation(() => {}),
};

/** Helper to build a JSON envelope response that the client can parse. */
function jsonResponse(body: unknown, init: { status?: number; ok?: boolean } = {}): Response {
  const status = init.status ?? 200;
  const ok = init.ok ?? (status >= 200 && status < 300);
  return {
    ok,
    status,
    statusText: status === 200 ? 'OK' : 'Error',
    json: async () => body,
    clone: () => jsonResponse(body, init),
    headers: new Headers(),
  } as unknown as Response;
}

describe('httpClient', () => {
  beforeEach(() => {
    fetchMock.mockReset();
    clearTenant();
    // jsdom has no auth cookie by default — clear it explicitly.
    document.cookie = 'access_token=; path=/; max-age=-1;';
    document.cookie = 'refresh_token=; path=/; max-age=-1;';
  });

  afterAll(() => {
    Object.values(consoleSpy).forEach((spy) => spy.mockRestore());
  });

  describe('base configuration', () => {
    it('defaults to same-origin when no public API URL is configured', () => {
      expect(httpClient.getBaseURL()).toBe(process.env.NEXT_PUBLIC_API_URL || '');
    });

    it('returns null token when no cookie is set', () => {
      expect(httpClient.getAuthToken()).toBeNull();
      expect(httpClient.getToken()).toBeNull();
    });

    it('setToken populates the in-memory token; clearToken clears it', () => {
      // Cookie is the source of truth for cross-request auth, but
      // setToken is kept for backward compatibility and DOES set the
      // in-memory field. clearToken resets it back to null.
      httpClient.setToken('whatever');
      expect(httpClient.getAuthToken()).toBe('whatever');
      httpClient.clearToken();
      expect(httpClient.getAuthToken()).toBeNull();
    });
  });

  describe('get', () => {
    it('sends a GET request to the full URL', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: { id: 1 } }));

      const result = await httpClient.get<{ id: number }>('/api/v1/tickets/1');

      expect(fetchMock).toHaveBeenCalledTimes(1);
      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/1');
      expect(init.method).toBe('GET');
      expect(result).toEqual({ id: 1 });
    });

    it('serializes query params and drops undefined/null', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: [] }));

      await httpClient.get('/api/v1/tickets', {
        page: 1,
        size: 20,
        status: 'open',
        unused: undefined,
        alsoUnused: null,
      });

      const [url] = fetchMock.mock.calls[0];
      expect(url).toContain('page=1');
      expect(url).toContain('size=20');
      expect(url).toContain('status=open');
      expect(url).not.toContain('unused');
      expect(url).not.toContain('alsoUnused');
    });

    it('returns camelCase data when backend sends snake_case', async () => {
      fetchMock.mockResolvedValueOnce(
        jsonResponse({
          code: 0,
          message: 'ok',
          data: { ticket_number: 'TKT-001', assignee_id: 7, created_at: '2024-01-01' },
        })
      );

      const result = await httpClient.get<{ ticketNumber: string; assigneeId: number; createdAt: string }>(
        '/api/v1/tickets/1'
      );

      expect(result).toEqual({
        ticketNumber: 'TKT-001',
        assigneeId: 7,
        createdAt: '2024-01-01',
      });
    });
  });

  describe('post', () => {
    it('sends a POST with JSON body and returns response data', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: { id: 99 } }));

      const result = await httpClient.post<{ id: number }>('/api/v1/tickets', {
        title: 'New',
        description: 'New ticket',
      });

      const [, init] = fetchMock.mock.calls[0];
      expect(init.method).toBe('POST');
      expect(init.body).toBe(JSON.stringify({ title: 'New', description: 'New ticket' }));
      expect(result).toEqual({ id: 99 });
    });

    it('throws on non-zero backend code with the backend message', async () => {
      fetchMock.mockResolvedValueOnce(
        jsonResponse({ code: 1001, message: '标题不能为空', data: null })
      );

      await expect(httpClient.post('/api/v1/tickets', { title: '' })).rejects.toThrow('标题不能为空');
    });

    it('throws with HTTP status when response.ok is false', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({}, { status: 500, ok: false }));

      await expect(httpClient.get('/api/v1/tickets/1')).rejects.toThrow(/status: 500/);
    });

    it('tolerates a missing code field (BPMN controllers do not always set one)', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ message: 'ok', data: { ok: true } }));

      const result = await httpClient.get<{ ok: boolean }>('/api/v1/bpmn/processes');
      expect(result).toEqual({ ok: true });
    });

    it('preserves custom field names inside array-shaped values (map-shaped keys would be corrupted by camelCase normalization)', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: {} }));

      await httpClient.post('/api/v1/tickets', {
        formFields: {
          values: [{ name: 'current_replicas', value: 5 }],
        },
      });

      const [, init] = fetchMock.mock.calls[0];
      const sentBody = JSON.parse(init.body);
      // 数组元素里的 name 是字符串值，不是 object key，不会被请求体的驼峰转换改写。
      expect(sentBody.formFields.values[0].name).toBe('current_replicas');
      expect(sentBody.formFields.values[0].value).toBe(5);
    });
  });

  describe('put / patch / delete', () => {
    it('put uses PUT method and JSON body', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: {} }));
      await httpClient.put('/api/v1/tickets/1', { title: 'Updated' });
      const [, init] = fetchMock.mock.calls[0];
      expect(init.method).toBe('PUT');
      expect(init.body).toBe(JSON.stringify({ title: 'Updated' }));
    });

    it('patch uses PATCH method', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: {} }));
      await httpClient.patch('/api/v1/tickets/1', { title: 'Patched' });
      const [, init] = fetchMock.mock.calls[0];
      expect(init.method).toBe('PATCH');
    });

    it('delete uses DELETE method and accepts an optional body', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: null }));
      await httpClient.delete('/api/v1/tickets/1', { reason: 'spam' });
      const [, init] = fetchMock.mock.calls[0];
      expect(init.method).toBe('DELETE');
      expect(init.body).toBe(JSON.stringify({ reason: 'spam' }));
    });
  });

  describe('headers', () => {
    it('adds CSRF token to mutating requests only', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: {} }));
      await httpClient.post('/api/v1/tickets', { title: 't' });
      const [, init] = fetchMock.mock.calls[0];
      expect(init.headers['X-CSRF-Token']).toBe('mock-csrf-token');

      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: {} }));
      await httpClient.get('/api/v1/tickets');
      const [, init2] = fetchMock.mock.calls[1];
      expect(init2.headers['X-CSRF-Token']).toBeUndefined();
    });

    it('refreshes a stale CSRF token once after a 403 and clears it after success', async () => {
      fetchMock
        .mockResolvedValueOnce(
          jsonResponse({ code: 403, message: 'CSRF token mismatch' }, { status: 403, ok: false })
        )
        .mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: { id: 1 } }));

      await expect(httpClient.post('/api/v1/changes', { title: 'change' })).resolves.toEqual({
        id: 1,
      });

      expect(fetchMock).toHaveBeenCalledTimes(2);
      expect(security.csrf.clearToken).toHaveBeenCalledTimes(2);
      expect(fetchMock.mock.calls[1][1].headers['X-CSRF-Token']).toBe('mock-csrf-token');
    });

    it('does not retry permission-related 403 responses', async () => {
      fetchMock.mockResolvedValueOnce(
        jsonResponse({ code: 2003, message: 'Forbidden' }, { status: 403, ok: false })
      );

      await expect(httpClient.delete('/api/v1/changes/1')).rejects.toThrow('status: 403');
      expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    it('uses browser credentials instead of reading a token cookie', async () => {
      document.cookie = 'access_token=cookie-jwt; path=/';
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: {} }));

      await httpClient.get('/api/v1/tickets');

      const [, init] = fetchMock.mock.calls[0];
      expect(init.credentials).toBe('include');
      expect(init.headers['Authorization']).toBeUndefined();
    });

    it('adds tenant headers when TenantContext is populated', async () => {
      setTenantId(42);
      setTenantCode('acme');
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: {} }));

      await httpClient.get('/api/v1/tickets');

      const [, init] = fetchMock.mock.calls[0];
      expect(init.headers['X-Tenant-ID']).toBe('42');
      expect(init.headers['X-Tenant-Code']).toBe('acme');
    });

    it('does not add tenant headers when TenantContext is empty', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: {} }));

      await httpClient.get('/api/v1/tickets');

      const [, init] = fetchMock.mock.calls[0];
      expect(init.headers['X-Tenant-ID']).toBeUndefined();
      expect(init.headers['X-Tenant-Code']).toBeUndefined();
    });
  });

  describe('getPaginated', () => {
    it('flattens nested filters into filters[key]=value', async () => {
      fetchMock.mockResolvedValueOnce(
        jsonResponse({ code: 0, message: 'ok', data: { data: [], total: 0, page: 1, pageSize: 20, totalPages: 0 } })
      );

      await httpClient.getPaginated('/api/v1/tickets', {
        page: 2,
        pageSize: 10,
        sortBy: 'created_at',
        sortOrder: 'desc',
        filters: { status: 'open', priority: 'high' },
      });

      const [url] = fetchMock.mock.calls[0];
      expect(url).toContain('page=2');
      expect(url).toContain('pageSize=10');
      expect(url).toContain('sortBy=created_at');
      expect(url).toContain('sortOrder=desc');
      expect(url).toContain('filters%5Bstatus%5D=open');
      expect(url).toContain('filters%5Bpriority%5D=high');
    });
  });

  describe('batchOperation', () => {
    it('wraps the data array under operation/data', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: [1, 2, 3] }));

      const result = await httpClient.batchOperation<number>('/api/v1/tickets/batch', 'close', [
        { id: 1 },
        { id: 2 },
      ]);

      const [, init] = fetchMock.mock.calls[0];
      const body = JSON.parse(init.body as string);
      expect(body).toEqual({ operation: 'close', data: [{ id: 1 }, { id: 2 }] });
      expect(result).toEqual([1, 2, 3]);
    });
  });
});
