import { renderHook, waitFor, act } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useServicesQuery, useServiceQuery, useServiceRatingsQuery,
  useServiceAnalyticsQuery, useServiceRequestsQuery, useServiceRequestQuery,
  useFavoritesQuery, usePortalConfigQuery, useCatalogStatsQuery,
  useCreateServiceMutation, useUpdateServiceMutation, usePublishServiceMutation,
  useCreateServiceRequestMutation, useApproveServiceRequestMutation,
  useRejectServiceRequestMutation, useAddFavoriteMutation,
  useRemoveFavoriteMutation, useRateServiceMutation,
  SERVICE_CATALOG_KEYS,
} from '../useServiceCatalog';

jest.mock('antd', () => ({ message: { success: jest.fn(), error: jest.fn() } }));

jest.mock('@/lib/api/service-catalog-api', () => ({
  ServiceCatalogApi: {
    getServices: jest.fn(), getService: jest.fn(), createService: jest.fn(),
    updateService: jest.fn(), publishService: jest.fn(),
    getServiceRatings: jest.fn(), getServiceAnalytics: jest.fn(),
    getServiceRequests: jest.fn(), getServiceRequest: jest.fn(),
    createServiceRequest: jest.fn(), approveServiceRequest: jest.fn(),
    rejectServiceRequest: jest.fn(), getFavorites: jest.fn(),
    addFavorite: jest.fn(), removeFavorite: jest.fn(),
    getPortalConfig: jest.fn(), getCatalogStats: jest.fn(),
    rateService: jest.fn(),
  },
}));

import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { message } from 'antd';
const mockApi = ServiceCatalogApi as jest.Mocked<typeof ServiceCatalogApi>;

const createWrapper = () => {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: qc }, children);
};

describe('useServiceCatalog hooks', () => {
  beforeEach(() => jest.clearAllMocks());

  describe('SERVICE_CATALOG_KEYS', () => {
    it('generates all key shapes', () => {
      expect(SERVICE_CATALOG_KEYS.all).toEqual(['service-catalog']);
      expect(SERVICE_CATALOG_KEYS.services()).toEqual(['service-catalog', 'services']);
      expect(SERVICE_CATALOG_KEYS.serviceDetail('s1')).toEqual(['service-catalog', 'services', 'detail', 's1']);
      expect(SERVICE_CATALOG_KEYS.serviceRatings('s1')).toEqual(['service-catalog', 'services', 'ratings', 's1']);
      expect(SERVICE_CATALOG_KEYS.serviceAnalytics('s1')).toEqual(['service-catalog', 'services', 'analytics', 's1']);
      expect(SERVICE_CATALOG_KEYS.requests()).toEqual(['service-catalog', 'requests']);
      expect(SERVICE_CATALOG_KEYS.requestDetail(1)).toEqual(['service-catalog', 'requests', 'detail', 1]);
      expect(SERVICE_CATALOG_KEYS.favorites()).toEqual(['service-catalog', 'favorites']);
      expect(SERVICE_CATALOG_KEYS.portal()).toEqual(['service-catalog', 'portal']);
      expect(SERVICE_CATALOG_KEYS.stats()).toEqual(['service-catalog', 'stats']);
    });
  });

  describe('useServicesQuery', () => {
    it('fetches services', async () => {
      mockApi.getServices.mockResolvedValue({ items: [] } as any);
      const { result } = renderHook(() => useServicesQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useServiceQuery', () => {
    it('fetches a service', async () => {
      mockApi.getService.mockResolvedValue({ id: 's1' } as any);
      const { result } = renderHook(() => useServiceQuery('s1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useServiceQuery('s1', false), { wrapper: createWrapper() });
      expect(mockApi.getService).not.toHaveBeenCalled();
    });
  });

  describe('useServiceRatingsQuery', () => {
    it('fetches ratings', async () => {
      mockApi.getServiceRatings.mockResolvedValue({ items: [] } as any);
      const { result } = renderHook(() => useServiceRatingsQuery('s1', { page: 1 }), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useServiceRatingsQuery('s1', undefined, false), { wrapper: createWrapper() });
      expect(mockApi.getServiceRatings).not.toHaveBeenCalled();
    });
  });

  describe('useServiceAnalyticsQuery', () => {
    it('fetches analytics', async () => {
      mockApi.getServiceAnalytics.mockResolvedValue({ views: 10 } as any);
      const { result } = renderHook(() => useServiceAnalyticsQuery('s1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useServiceAnalyticsQuery('s1', undefined, false), { wrapper: createWrapper() });
      expect(mockApi.getServiceAnalytics).not.toHaveBeenCalled();
    });
  });

  describe('useServiceRequestsQuery', () => {
    it('fetches requests', async () => {
      mockApi.getServiceRequests.mockResolvedValue({ items: [] } as any);
      const { result } = renderHook(() => useServiceRequestsQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useServiceRequestQuery', () => {
    it('fetches a request', async () => {
      mockApi.getServiceRequest.mockResolvedValue({ id: 1 } as any);
      const { result } = renderHook(() => useServiceRequestQuery(1), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useServiceRequestQuery(1, false), { wrapper: createWrapper() });
      expect(mockApi.getServiceRequest).not.toHaveBeenCalled();
    });
  });

  describe('useFavoritesQuery', () => {
    it('fetches favorites', async () => {
      mockApi.getFavorites.mockResolvedValue([] as any);
      const { result } = renderHook(() => useFavoritesQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('usePortalConfigQuery', () => {
    it('fetches portal config', async () => {
      mockApi.getPortalConfig.mockResolvedValue({} as any);
      const { result } = renderHook(() => usePortalConfigQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useCatalogStatsQuery', () => {
    it('fetches catalog stats', async () => {
      mockApi.getCatalogStats.mockResolvedValue({ total: 10 } as any);
      const { result } = renderHook(() => useCatalogStatsQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useCreateServiceMutation', () => {
    it('creates service', async () => {
      mockApi.createService.mockResolvedValue({ id: 'new' } as any);
      const { result } = renderHook(() => useCreateServiceMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ name: 'Svc' } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useUpdateServiceMutation', () => {
    it('updates service', async () => {
      mockApi.updateService.mockResolvedValue({ id: 's1' } as any);
      const { result } = renderHook(() => useUpdateServiceMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ id: 's1', data: { name: 'Updated' } } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('usePublishServiceMutation', () => {
    it('publishes service', async () => {
      mockApi.publishService.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => usePublishServiceMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('s1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useCreateServiceRequestMutation', () => {
    it('creates service request', async () => {
      mockApi.createServiceRequest.mockResolvedValue({ id: 1 } as any);
      const { result } = renderHook(() => useCreateServiceRequestMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ serviceId: 's1' } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useApproveServiceRequestMutation', () => {
    it('approves request', async () => {
      mockApi.approveServiceRequest.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useApproveServiceRequestMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ id: 1, comment: 'ok' }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useRejectServiceRequestMutation', () => {
    it('rejects request', async () => {
      mockApi.rejectServiceRequest.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useRejectServiceRequestMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ id: 1, reason: 'invalid' }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useAddFavoriteMutation', () => {
    it('adds favorite', async () => {
      mockApi.addFavorite.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useAddFavoriteMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('s1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useRemoveFavoriteMutation', () => {
    it('removes favorite', async () => {
      mockApi.removeFavorite.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useRemoveFavoriteMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('s1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useRateServiceMutation', () => {
    it('rates service', async () => {
      mockApi.rateService.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useRateServiceMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ serviceId: 's1', rating: 5, comment: 'Great' }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });
});
