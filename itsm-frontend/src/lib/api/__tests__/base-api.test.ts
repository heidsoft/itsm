import { ApiError, ApiErrorCode, ApiResult, ApiResponseHandler, BaseApi } from '../base-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    getBaseURL: jest.fn(() => 'http://localhost:8090'),
    getAuthToken: jest.fn(() => 'test-token'),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;
const mockPatch = (httpClient as any).patch as jest.Mock;

// Create a concrete implementation of the abstract BaseApi for testing
class TestApi extends BaseApi {
  protected static readonly basePath = '/api/v1/test';

  static async testGetList(params?: any) { return this.getList('/items', params); }
  static async testGetById(id: string | number) { return this.getById('/items', id); }
  static async testCreate(data: any) { return this.create('/items', data); }
  static async testUpdate(id: string | number, data: any) { return this.update('/items', id, data); }
  static async testPatch(id: string | number, data: any) { return this.patch('/items', id, data); }
  static async testDelete(id: string | number) { return this.delete('/items', id); }
  static async testBatchDelete(ids: any[]) { return this.batchDelete('/items', ids); }
  static async testBatchUpdate(ids: any[], data: any) { return this.batchUpdate('/items', ids, data); }
  static async testSearch(params: any) { return this.search('/items', params); }
  static async testGetStats(params?: any) { return this.getStats('/items', params); }
  static async testUpload(file: File, data?: any) { return this.upload('/upload', file, data); }
  static testBuildQueryString(p: any) { return this.buildQueryString(p); }
  static testSafeParse(json: string, fallback: any) { return this.safeParse(json, fallback); }
}

describe('base-api', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('ApiError', () => {
    it('should create with all params', () => {
      const err = new ApiError('msg', 404, '/test', new Error('orig'));
      expect(err.message).toBe('msg');
      expect(err.code).toBe(404);
      expect(err.endpoint).toBe('/test');
      expect(err.originalError).toBeInstanceOf(Error);
      expect(err.name).toBe('ApiError');
    });

    it('should create with defaults', () => {
      const err = new ApiError('msg');
      expect(err.code).toBe(ApiErrorCode.INTERNAL_ERROR);
      expect(err.endpoint).toBe('');
    });

    it('fromResponse should create from ApiResponse', () => {
      const err = ApiError.fromResponse({ code: 1001, message: 'Bad param', data: null } as any, '/ep');
      expect(err.message).toBe('Bad param');
      expect(err.code).toBe(1001);
      expect(err.endpoint).toBe('/ep');
    });

    it('fromError should return same ApiError', () => {
      const orig = new ApiError('orig', 400, '/a');
      expect(ApiError.fromError(orig, '/b')).toBe(orig);
    });

    it('fromError should wrap Error', () => {
      const err = ApiError.fromError(new Error('regular'), '/ep');
      expect(err.message).toBe('regular');
      expect(err.code).toBe(ApiErrorCode.INTERNAL_ERROR);
    });

    it('fromError should handle non-Error', () => {
      const err = ApiError.fromError('string error', '/ep');
      expect(err.message).toBe('Unknown error occurred');
    });

    it('isAuthError should check code', () => {
      expect(new ApiError('', ApiErrorCode.AUTH_FAILED).isAuthError).toBe(true);
      expect(new ApiError('', 400).isAuthError).toBe(false);
    });

    it('isNotFound should check code', () => {
      expect(new ApiError('', ApiErrorCode.NOT_FOUND).isNotFound).toBe(true);
      expect(new ApiError('', 400).isNotFound).toBe(false);
    });

    it('isParamError should check code range', () => {
      expect(new ApiError('', ApiErrorCode.PARAM_ERROR).isParamError).toBe(true);
      expect(new ApiError('', 1999).isParamError).toBe(true);
      expect(new ApiError('', 2000).isParamError).toBe(false);
    });
  });

  describe('ApiResult', () => {
    it('ok should create success result', () => {
      const r = ApiResult.ok({ id: 1 });
      expect(r.success).toBe(true);
      expect(r.data).toEqual({ id: 1 });
      expect(r.error).toBeNull();
    });

    it('fail should create error result', () => {
      const err = new ApiError('fail', 500);
      const r = ApiResult.fail(err);
      expect(r.success).toBe(false);
      expect(r.data).toBeNull();
      expect(r.error).toBe(err);
    });

    it('fromResponse should return ok for code 0', () => {
      const r = ApiResult.fromResponse({ code: 0, message: 'ok', data: { id: 1 } } as any);
      expect(r.success).toBe(true);
      expect(r.data).toEqual({ id: 1 });
    });

    it('fromResponse should return fail for non-zero code', () => {
      const r = ApiResult.fromResponse({ code: 1001, message: 'error', data: null } as any);
      expect(r.success).toBe(false);
      expect(r.error).not.toBeNull();
    });

    it('getOrThrow should return data on success', () => {
      const r = ApiResult.ok(42);
      expect(r.getOrThrow()).toBe(42);
    });

    it('getOrThrow should throw on failure', () => {
      const r = ApiResult.fail(new ApiError('fail', 500));
      expect(() => r.getOrThrow()).toThrow();
    });

    it('getOrNull should return data or null', () => {
      expect(ApiResult.ok(42).getOrNull()).toBe(42);
      expect(ApiResult.fail(new ApiError('f', 500)).getOrNull()).toBeNull();
    });

    it('getOrDefault should return data or default', () => {
      expect(ApiResult.ok(42).getOrDefault(0)).toBe(42);
      expect(ApiResult.fail<number>(new ApiError('f', 500)).getOrDefault(0)).toBe(0);
    });
  });

  describe('ApiResponseHandler', () => {
    it('isSuccess should check code 0', () => {
      expect(ApiResponseHandler.isSuccess({ code: 0, message: '', data: null } as any)).toBe(true);
      expect(ApiResponseHandler.isSuccess({ code: 1, message: '', data: null } as any)).toBe(false);
    });

    it('extractData should return data on success', () => {
      expect(ApiResponseHandler.extractData({ code: 0, message: '', data: 'val' } as any)).toBe('val');
    });

    it('extractData should throw on failure', () => {
      expect(() => ApiResponseHandler.extractData({ code: 1, message: 'err', data: null } as any)).toThrow();
    });

    it('handlePaginationResponse should extract paginated data', () => {
      const pagData = { items: [1, 2], total: 2, page: 1, pageSize: 10 };
      expect(ApiResponseHandler.handlePaginationResponse({ code: 0, message: '', data: pagData } as any)).toEqual(pagData);
    });
  });

  describe('BaseApi CRUD operations', () => {
    it('getList should call GET', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await TestApi.testGetList({ page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/test/items', { page: 1 });
    });

    it('getById should call GET with id', async () => {
      mockGet.mockResolvedValue({ id: 1 });
      await TestApi.testGetById(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/test/items/1', undefined);
    });

    it('create should call POST', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await TestApi.testCreate({ name: 'New' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/test/items', { name: 'New' });
    });

    it('update should call PUT', async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await TestApi.testUpdate(1, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/test/items/1', { name: 'Updated' });
    });

    it('patch should call PATCH', async () => {
      mockPatch.mockResolvedValue({ id: 1 });
      await TestApi.testPatch(1, { name: 'Patched' });
      expect(mockPatch).toHaveBeenCalledWith('/api/v1/test/items/1', { name: 'Patched' });
    });

    it('delete should call DELETE', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TestApi.testDelete(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/test/items/1', undefined);
    });

    it('batchDelete should call POST with ids', async () => {
      mockPost.mockResolvedValue({ deleted: 2 });
      await TestApi.testBatchDelete([1, 2]);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/test/items/batch', { ids: [1, 2] });
    });

    it('batchUpdate should call PUT', async () => {
      mockPut.mockResolvedValue({ updated: 2 });
      await TestApi.testBatchUpdate([1, 2], { status: 'active' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/test/items/batch', { ids: [1, 2], data: { status: 'active' } });
    });

    it('search should call GET /search', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0 });
      await TestApi.testSearch({ keyword: 'test' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/test/items/search', { keyword: 'test' });
    });

    it('getStats should call GET /stats', async () => {
      mockGet.mockResolvedValue({ total: 100 });
      await TestApi.testGetStats({ period: 'week' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/test/items/stats', { period: 'week' });
    });

    it('upload should send FormData', async () => {
      mockPost.mockResolvedValue({ url: '/files/1.pdf' });
      const file = new File(['data'], 'test.pdf');
      await TestApi.testUpload(file, { category: 'docs' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/test/upload', expect.any(FormData));
    });

    it('should wrap errors as ApiError', async () => {
      mockGet.mockRejectedValue(new Error('Network fail'));
      await expect(TestApi.testGetById(1)).rejects.toThrow();
    });
  });

  describe('BaseApi utility methods', () => {
    it('buildQueryString should filter null/undefined', () => {
      const qs = TestApi.testBuildQueryString({ a: 'val', b: null, c: undefined, d: 123 });
      expect(qs).toContain('a=val');
      expect(qs).toContain('d=123');
      expect(qs).not.toContain('b=');
      expect(qs).not.toContain('c=');
    });

    it('safeParse should parse valid JSON', () => {
      expect(TestApi.testSafeParse('{"a":1}', {})).toEqual({ a: 1 });
    });

    it('safeParse should return fallback for invalid JSON', () => {
      expect(TestApi.testSafeParse('invalid', { fallback: true })).toEqual({ fallback: true });
    });
  });
});
