import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useServicesQuery, useServiceQuery, useCreateServiceMutation, SERVICE_CATALOG_KEYS } from '../useServiceCatalog';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock ServiceCatalogApi
jest.mock('@/lib/api/service-catalog-api', () => ({
  ServiceCatalogApi: {
    getServices: jest.fn(),
    getService: jest.fn(),
    createService: jest.fn(),
    updateService: jest.fn(),
    publishService: jest.fn(),
    getServiceRatings: jest.fn(),
    getServiceAnalytics: jest.fn(),
    getServiceRequests: jest.fn(),
    getServiceRequest: jest.fn(),
    createServiceRequest: jest.fn(),
    approveServiceRequest: jest.fn(),
    rejectServiceRequest: jest.fn(),
    getFavorites: jest.fn(),
    addFavorite: jest.fn(),
    removeFavorite: jest.fn(),
    getPortalConfig: jest.fn(),
    getCatalogStats: jest.fn(),
    rateService: jest.fn(),
  },
}));

import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
const mockServiceCatalogApi = ServiceCatalogApi as jest.Mocked<typeof ServiceCatalogApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useServiceCatalog hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('SERVICE_CATALOG_KEYS', () => {
    it('should generate correct query keys', () => {
      expect(SERVICE_CATALOG_KEYS.all).toEqual(['service-catalog']);
      expect(SERVICE_CATALOG_KEYS.services()).toEqual(['service-catalog', 'services']);
      expect(SERVICE_CATALOG_KEYS.serviceDetail('svc-1')).toEqual(['service-catalog', 'services', 'detail', 'svc-1']);
    });
  });

  describe('useServicesQuery', () => {
    it('should fetch services', async () => {
      mockServiceCatalogApi.getServices.mockResolvedValue({ items: [], total: 0 });

      const { result } = renderHook(() => useServicesQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockServiceCatalogApi.getServices).toHaveBeenCalled();
    });
  });

  describe('useServiceQuery', () => {
    it('should fetch a single service', async () => {
      mockServiceCatalogApi.getService.mockResolvedValue({ id: 'svc-1', name: 'Email Service' });

      const { result } = renderHook(() => useServiceQuery('svc-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockServiceCatalogApi.getService).toHaveBeenCalledWith('svc-1');
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useServiceQuery('svc-1', false), {
        wrapper: createWrapper(),
      });

      expect(mockServiceCatalogApi.getService).not.toHaveBeenCalled();
    });
  });

  describe('useCreateServiceMutation', () => {
    it('should create a service', async () => {
      mockServiceCatalogApi.createService.mockResolvedValue({ id: 'new-svc' });

      const { result } = renderHook(() => useCreateServiceMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'New Service', category: 'IT' } as any);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockServiceCatalogApi.createService).toHaveBeenCalledWith({ name: 'New Service', category: 'IT' }, expect.anything());
    });
  });
});
