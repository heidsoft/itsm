/**
 * Tests for templates module
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

jest.mock('zustand', () => ({
  create: jest.fn((fn: any) => {
    if (typeof fn === 'function') return fn;
    return fn;
  }),
}));

jest.mock('zustand/middleware', () => ({
  persist: (fn: any) => fn,
}));

jest.mock('@tanstack/react-query', () => ({
  useQuery: jest.fn(),
  useMutation: jest.fn(() => ({ mutate: jest.fn() })),
  useQueryClient: jest.fn(() => ({ invalidateQueries: jest.fn() })),
}));

import { httpClient } from '@/lib/api/http-client';
import { createCrudApi, createTenantApi } from '../api-service';

const mockedHttpClient = httpClient as jest.Mocked<typeof httpClient>;

describe('Templates - API Service', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('createCrudApi', () => {
    it('should create CRUD API with all methods', () => {
      const api = createCrudApi<{ id: number; name: string }>('/users');

      expect(api.list).toBeDefined();
      expect(api.get).toBeDefined();
      expect(api.create).toBeDefined();
      expect(api.update).toBeDefined();
      expect(api.delete).toBeDefined();
      expect(api.batchDelete).toBeDefined();
    });

    it('should call httpClient.get for list', () => {
      const api = createCrudApi<{ id: number }>('/users');
      api.list({ page: 1 });
      expect(mockedHttpClient.get).toHaveBeenCalledWith('/users', { page: 1 });
    });

    it('should call httpClient.get for get by id', () => {
      const api = createCrudApi<{ id: number }>('/users');
      api.get(1);
      expect(mockedHttpClient.get).toHaveBeenCalledWith('/users/1');
    });

    it('should call httpClient.post for create', () => {
      const api = createCrudApi<{ id: number; name: string }>('/users');
      api.create({ name: 'John' });
      expect(mockedHttpClient.post).toHaveBeenCalledWith('/users', { name: 'John' });
    });

    it('should call httpClient.put for update', () => {
      const api = createCrudApi<{ id: number; name: string }>('/users');
      api.update(1, { name: 'Updated' });
      expect(mockedHttpClient.put).toHaveBeenCalledWith('/users/1', { name: 'Updated' });
    });

    it('should call httpClient.delete for delete', () => {
      const api = createCrudApi<{ id: number }>('/users');
      api.delete(1);
      expect(mockedHttpClient.delete).toHaveBeenCalledWith('/users/1');
    });

    it('should call batch delete with custom endpoint', () => {
      const api = createCrudApi<{ id: number }>('/users', { batchEndpoint: '/users/batch' });
      api.batchDelete([1, 2, 3]);
      expect(mockedHttpClient.post).toHaveBeenCalledWith('/users/batch/delete', { ids: [1, 2, 3] });
    });

    it('should use default batch endpoint', () => {
      const api = createCrudApi<{ id: number }>('/users');
      api.batchDelete([1, 2]);
      expect(mockedHttpClient.post).toHaveBeenCalledWith('/users/batch/delete', { ids: [1, 2] });
    });

    it('should include custom endpoints', () => {
      const customFn = jest.fn();
      const api = createCrudApi<{ id: number }>('/users', {
        customEndpoints: { customAction: customFn },
      });
      expect((api as any).customAction).toBe(customFn);
    });
  });

  describe('createTenantApi', () => {
    it('should create tenant API with CRUD methods', () => {
      const api = createTenantApi<{ id: number; name: string }>('/tenants');

      expect(api.list).toBeDefined();
      expect(api.get).toBeDefined();
      expect(api.create).toBeDefined();
      expect(api.update).toBeDefined();
      expect(api.delete).toBeDefined();
    });

    it('should call httpClient correctly', () => {
      const api = createTenantApi<{ id: number }>('/tenants');
      api.list({ page: 1 });
      expect(mockedHttpClient.get).toHaveBeenCalledWith('/tenants', { page: 1 });
    });

    it('should call get by id', () => {
      const api = createTenantApi<{ id: number }>('/tenants');
      api.get(5);
      expect(mockedHttpClient.get).toHaveBeenCalledWith('/tenants/5');
    });

    it('should call create', () => {
      const api = createTenantApi<{ id: number; name: string }>('/tenants');
      api.create({ name: 'New' });
      expect(mockedHttpClient.post).toHaveBeenCalledWith('/tenants', { name: 'New' });
    });

    it('should call update', () => {
      const api = createTenantApi<{ id: number; name: string }>('/tenants');
      api.update(3, { name: 'Updated' });
      expect(mockedHttpClient.put).toHaveBeenCalledWith('/tenants/3', { name: 'Updated' });
    });

    it('should call delete', () => {
      const api = createTenantApi<{ id: number }>('/tenants');
      api.delete(7);
      expect(mockedHttpClient.delete).toHaveBeenCalledWith('/tenants/7');
    });
  });
});

describe('Templates - Store', () => {
  it('should export createSimpleStore', async () => {
    const { createSimpleStore } = await import('../store');
    expect(createSimpleStore).toBeDefined();
  });

  it('should export useAuthStore', async () => {
    const { useAuthStore } = await import('../store');
    expect(useAuthStore).toBeDefined();
  });
});

describe('Templates - Query', () => {
  it('should export useList', async () => {
    const { useList } = await import('../query');
    expect(useList).toBeDefined();
  });

  it('should export useDetail', async () => {
    const { useDetail } = await import('../query');
    expect(useDetail).toBeDefined();
  });

  it('should export useCreate', async () => {
    const { useCreate } = await import('../query');
    expect(useCreate).toBeDefined();
  });

  it('should export useUpdate', async () => {
    const { useUpdate } = await import('../query');
    expect(useUpdate).toBeDefined();
  });

  it('should export useDelete', async () => {
    const { useDelete } = await import('../query');
    expect(useDelete).toBeDefined();
  });
});
