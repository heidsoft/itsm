import {
  handleApiResponse,
  handlePaginatedResponse,
  handleArrayResponse,
  handleObjectResponse,
  validateResponse,
  safeGet,
} from '../api-response-handler';

describe('handleApiResponse', () => {
  it('returns defaultValue for null/undefined', () => {
    expect(handleApiResponse(null, 'fallback')).toBe('fallback');
    expect(handleApiResponse(undefined, 'fallback')).toBe('fallback');
  });

  it('extracts data from a successful ApiResponse (code 0)', () => {
    const resp = { code: 0, message: 'ok', data: { id: 1 } };
    expect(handleApiResponse(resp, { id: 0 })).toEqual({ id: 1 });
  });

  it('returns defaultValue when code != 0', () => {
    const resp = { code: 500, message: 'err', data: { id: 1 } };
    expect(handleApiResponse(resp, { id: 0 })).toEqual({ id: 0 });
  });

  it('returns raw value when not an ApiResponse shape', () => {
    expect(handleApiResponse('hello' as any, 'default')).toBe('hello');
  });
});

describe('handlePaginatedResponse', () => {
  it('returns default pagination for null', () => {
    const result = handlePaginatedResponse(null);
    expect(result).toEqual({ data: [], total: 0, page: 1, pageSize: 20, totalPages: 0 });
  });

  it('extracts paginated data from ApiResponse', () => {
    const resp = { code: 0, message: 'ok', data: { data: [1, 2], total: 2, page: 1, pageSize: 10, totalPages: 1 } };
    expect(handlePaginatedResponse(resp)).toEqual({ data: [1, 2], total: 2, page: 1, pageSize: 10, totalPages: 1 });
  });

  it('returns default on non-zero code', () => {
    const resp = { code: 1, message: 'err', data: { data: [1], total: 1, page: 1, pageSize: 10, totalPages: 1 } };
    expect(handlePaginatedResponse(resp).data).toEqual([]);
  });

  it('handles direct PaginatedResponse object', () => {
    const direct = { data: ['a'], total: 1, page: 2, pageSize: 5, totalPages: 1 };
    expect(handlePaginatedResponse(direct)).toEqual({ data: ['a'], total: 1, page: 2, pageSize: 5, totalPages: 1 });
  });
});

describe('handleArrayResponse', () => {
  it('returns [] for null/undefined', () => {
    expect(handleArrayResponse(null)).toEqual([]);
    expect(handleArrayResponse(undefined)).toEqual([]);
  });

  it('extracts array from ApiResponse', () => {
    expect(handleArrayResponse({ code: 0, message: '', data: [1, 2, 3] })).toEqual([1, 2, 3]);
  });

  it('returns [] for non-zero code', () => {
    expect(handleArrayResponse({ code: 1, message: '', data: [1] })).toEqual([]);
  });

  it('returns direct array', () => {
    expect(handleArrayResponse([4, 5])).toEqual([4, 5]);
  });
});

describe('handleObjectResponse', () => {
  const defaultVal = { name: '', age: 0 };

  it('returns default for null', () => {
    expect(handleObjectResponse(null, defaultVal)).toEqual(defaultVal);
  });

  it('merges ApiResponse data with defaults', () => {
    const resp = { code: 0, message: '', data: { name: 'Alice' } };
    expect(handleObjectResponse(resp, defaultVal)).toEqual({ name: 'Alice', age: 0 });
  });

  it('merges direct object with defaults', () => {
    expect(handleObjectResponse({ name: 'Bob' } as any, defaultVal)).toEqual({ name: 'Bob', age: 0 });
  });
});

describe('validateResponse', () => {
  it('returns false for null/undefined', () => {
    expect(validateResponse(null, () => true)).toBe(false);
    expect(validateResponse(undefined, () => true)).toBe(false);
  });

  it('runs validator on truthy data', () => {
    expect(validateResponse({ x: 1 }, (d) => 'x' in d)).toBe(true);
    expect(validateResponse({ x: 1 }, (d) => 'y' in d)).toBe(false);
  });
});

describe('safeGet', () => {
  it('returns defaultValue for non-object', () => {
    expect(safeGet(null, 'a.b', 42)).toBe(42);
    expect(safeGet(undefined, 'a', 'x')).toBe('x');
  });

  it('navigates nested paths', () => {
    expect(safeGet({ a: { b: { c: 99 } } }, 'a.b.c', 0)).toBe(99);
  });

  it('returns default when path breaks', () => {
    expect(safeGet({ a: { b: null } }, 'a.b.c', 'def')).toBe('def');
  });
});
