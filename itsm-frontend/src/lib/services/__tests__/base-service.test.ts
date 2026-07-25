import { BaseService, ApiError, buildQueryString, joinPath } from '../base-service';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    request: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;
const mockPatch = httpClient.patch as jest.Mock;
const mockRequest = (httpClient as any).request as jest.Mock;

// Concrete implementation for testing the abstract BaseService
interface TestEntity {
  id: number;
  name: string;
}

class TestService extends BaseService<TestEntity> {
  protected readonly basePath = '/api/v1/tests';
}

const testService = new TestService();

describe('BaseService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('list', () => {
    it('should fetch paginated list', async () => {
      const response = { data: [{ id: 1, name: 'Test' }], total: 1, page: 1, pageSize: 10, totalPages: 1 };
      mockGet.mockResolvedValue(response);

      const result = await testService.list({ page: 1, pageSize: 10 });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/tests', { page: 1, pageSize: 10 });
      expect(result.data).toHaveLength(1);
      expect(result.total).toBe(1);
    });

    it('should list without params', async () => {
      mockGet.mockResolvedValue({ data: [], total: 0, page: 1, pageSize: 10, totalPages: 0 });

      await testService.list();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/tests', undefined);
    });
  });

  describe('getById', () => {
    it('should fetch entity by id', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Test Item' });

      const result = await testService.getById(1);

      expect(mockGet).toHaveBeenCalledWith('/api/v1/tests/1');
      expect(result.name).toBe('Test Item');
    });
  });

  describe('create', () => {
    it('should create an entity', async () => {
      const data = { name: 'New Item' };
      mockPost.mockResolvedValue({ id: 2, name: 'New Item' });

      const result = await testService.create(data);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/tests', data);
      expect(result.id).toBe(2);
    });
  });

  describe('update', () => {
    it('should update an entity', async () => {
      const data = { name: 'Updated' };
      mockPut.mockResolvedValue({ id: 1, name: 'Updated' });

      const result = await testService.update(1, data);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/tests/1', data);
      expect(result.name).toBe('Updated');
    });
  });

  describe('patch', () => {
    it('should partially update an entity', async () => {
      const data = { name: 'Patched' };
      mockPatch.mockResolvedValue({ id: 1, name: 'Patched' });

      const result = await testService.patch(1, data);

      expect(mockPatch).toHaveBeenCalledWith('/api/v1/tests/1', data);
      expect(result.name).toBe('Patched');
    });
  });

  describe('delete', () => {
    it('should delete an entity by id', async () => {
      mockDelete.mockResolvedValue(undefined);

      await testService.delete(1);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tests/1');
    });
  });

  describe('batchDelete', () => {
    it('should batch delete entities', async () => {
      mockRequest.mockResolvedValue(undefined);

      await testService.batchDelete([1, 2, 3]);

      expect(mockRequest).toHaveBeenCalledWith({
        method: 'DELETE',
        url: '/api/v1/tests/batch',
        data: { ids: [1, 2, 3] },
      });
    });
  });

  describe('exists', () => {
    it('should return true when entity exists', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Exists' });

      const result = await testService.exists(1);

      expect(result).toBe(true);
    });

    it('should return false when entity does not exist', async () => {
      mockGet.mockRejectedValue(new Error('Not found'));

      const result = await testService.exists(999);

      expect(result).toBe(false);
    });
  });

  describe('search', () => {
    it('should search entities', async () => {
      mockGet.mockResolvedValue([{ id: 1, name: 'Found' }]);

      const result = await testService.search('test query', { page: 1 });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/tests/search', { q: 'test query', page: 1 });
      expect(result).toHaveLength(1);
    });

    it('should search without extra params', async () => {
      mockGet.mockResolvedValue([]);

      await testService.search('keyword');

      expect(mockGet).toHaveBeenCalledWith('/api/v1/tests/search', { q: 'keyword' });
    });
  });
});

describe('ApiError', () => {
  it('should create error with code', () => {
    const error = new ApiError('Not found', 404);
    expect(error.message).toBe('Not found');
    expect(error.code).toBe(404);
    expect(error.name).toBe('ApiError');
  });

  it('should detect auth errors', () => {
    expect(new ApiError('Unauthorized', 401).isAuthError).toBe(true);
    expect(new ApiError('Token expired', 2001).isAuthError).toBe(true);
    expect(new ApiError('Forbidden', 403).isAuthError).toBe(false);
  });

  it('should detect forbidden errors', () => {
    expect(new ApiError('Forbidden', 403).isForbidden).toBe(true);
    expect(new ApiError('No permission', 2003).isForbidden).toBe(true);
    expect(new ApiError('Not found', 404).isForbidden).toBe(false);
  });

  it('should detect not found errors', () => {
    expect(new ApiError('Not found', 404).isNotFound).toBe(true);
    expect(new ApiError('Resource missing', 3001).isNotFound).toBe(true);
    expect(new ApiError('Error', 500).isNotFound).toBe(false);
  });

  it('should detect conflict errors', () => {
    expect(new ApiError('Conflict', 409).isConflict).toBe(true);
    expect(new ApiError('Error', 500).isConflict).toBe(false);
  });

  it('should store requestId and details', () => {
    const error = new ApiError('Error', 500, { requestId: 'req-123', details: { field: 'name' } });
    expect(error.requestId).toBe('req-123');
    expect(error.details).toEqual({ field: 'name' });
  });
});

describe('buildQueryString', () => {
  it('should build query string from params', () => {
    const result = buildQueryString({ page: 1, search: 'test', status: 'active' });
    expect(result).toContain('page=1');
    expect(result).toContain('search=test');
    expect(result).toContain('status=active');
  });

  it('should skip undefined, null, and empty values', () => {
    const result = buildQueryString({ page: 1, name: undefined, empty: '', nil: null as any });
    expect(result).toBe('page=1');
  });

  it('should return empty string for empty params', () => {
    const result = buildQueryString({});
    expect(result).toBe('');
  });
});

describe('joinPath', () => {
  it('should join path segments', () => {
    expect(joinPath('/api/v1', 'users', '123')).toBe('/api/v1/users/123');
  });

  it('should handle trailing/leading slashes', () => {
    expect(joinPath('/api/v1/', '/users/', '/123')).toBe('/api/v1/users/123');
  });

  it('should handle single segment', () => {
    expect(joinPath('/api')).toBe('/api');
  });
});
