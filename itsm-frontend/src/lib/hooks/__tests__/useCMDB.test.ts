import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useCIsQuery, useCIQuery, useCreateCIMutation, CMDB_KEYS } from '../useCMDB';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock CMDBApi
jest.mock('@/lib/api/cmdb-api', () => ({
  CMDBApi: {
    getCIs: jest.fn(),
    getCI: jest.fn(),
    createCI: jest.fn(),
    updateCI: jest.fn(),
    deleteCI: jest.fn(),
    batchCreateCIs: jest.fn(),
    getCIRelationships: jest.fn(),
    getCITopology: jest.fn(),
    analyzeImpact: jest.fn(),
    getCITypes: jest.fn(),
    getCIChangeHistory: jest.fn(),
    getCMDBStats: jest.fn(),
    getDiscoveryRules: jest.fn(),
    getDiscoveryHistory: jest.fn(),
    createRelationship: jest.fn(),
    deleteRelationship: jest.fn(),
    runDiscoveryRule: jest.fn(),
  },
}));

import { CMDBApi } from '@/lib/api/cmdb-api';
const mockCMDBApi = CMDBApi as jest.Mocked<typeof CMDBApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useCMDB hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('CMDB_KEYS', () => {
    it('should generate correct query keys', () => {
      expect(CMDB_KEYS.all).toEqual(['cmdb']);
      expect(CMDB_KEYS.cis()).toEqual(['cmdb', 'cis']);
      expect(CMDB_KEYS.ciDetail('abc')).toEqual(['cmdb', 'cis', 'detail', 'abc']);
    });
  });

  describe('useCIsQuery', () => {
    it('should fetch CI list', async () => {
      mockCMDBApi.getCIs.mockResolvedValue({ items: [], total: 0 });

      const { result } = renderHook(() => useCIsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockCMDBApi.getCIs).toHaveBeenCalled();
    });
  });

  describe('useCIQuery', () => {
    it('should fetch a single CI', async () => {
      mockCMDBApi.getCI.mockResolvedValue({ id: 'ci-1', name: 'Server' });

      const { result } = renderHook(() => useCIQuery('ci-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockCMDBApi.getCI).toHaveBeenCalledWith('ci-1');
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useCIQuery('ci-1', false), {
        wrapper: createWrapper(),
      });

      expect(mockCMDBApi.getCI).not.toHaveBeenCalled();
    });
  });

  describe('useCreateCIMutation', () => {
    it('should create a CI', async () => {
      mockCMDBApi.createCI.mockResolvedValue({ id: 'new-ci' });

      const { result } = renderHook(() => useCreateCIMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'New Server', type: 'server' } as any);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockCMDBApi.createCI).toHaveBeenCalledWith({ name: 'New Server', type: 'server' }, expect.anything());
    });
  });
});
