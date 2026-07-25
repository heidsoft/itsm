import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useMatrixConfigsQuery,
  useActiveMatrixConfigQuery,
  useMatrixDataQuery,
  usePriorityRulesQuery,
  usePrioritySuggestionQuery,
  usePriorityDistributionQuery,
  useCalculatePriorityMutation,
  useCreateMatrixConfigMutation,
  useActivateMatrixConfigMutation,
  useCreatePriorityRuleMutation,
  PRIORITY_MATRIX_KEYS,
} from '../usePriorityMatrix';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock PriorityMatrixApi
jest.mock('@/lib/api/priority-matrix-api', () => ({
  PriorityMatrixApi: {
    getMatrixConfigs: jest.fn(),
    getActiveMatrixConfig: jest.fn(),
    getMatrixData: jest.fn(),
    getPriorityRules: jest.fn(),
    getPrioritySuggestion: jest.fn(),
    getPriorityDistribution: jest.fn(),
    calculatePriority: jest.fn(),
    createMatrixConfig: jest.fn(),
    activateMatrixConfig: jest.fn(),
    createPriorityRule: jest.fn(),
  },
}));

import { PriorityMatrixApi } from '@/lib/api/priority-matrix-api';
const mockApi = PriorityMatrixApi as jest.Mocked<typeof PriorityMatrixApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('usePriorityMatrix hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('PRIORITY_MATRIX_KEYS', () => {
    it('should generate correct query keys', () => {
      expect(PRIORITY_MATRIX_KEYS.all).toEqual(['priority-matrix']);
      expect(PRIORITY_MATRIX_KEYS.configs()).toEqual(['priority-matrix', 'configs']);
      expect(PRIORITY_MATRIX_KEYS.activeConfig()).toEqual(['priority-matrix', 'active-config']);
      expect(PRIORITY_MATRIX_KEYS.matrixData('cfg-1')).toEqual(['priority-matrix', 'matrix-data', 'cfg-1']);
      expect(PRIORITY_MATRIX_KEYS.rules({ active: true } as any)).toEqual(['priority-matrix', 'rules', { active: true }]);
      expect(PRIORITY_MATRIX_KEYS.suggestion(5)).toEqual(['priority-matrix', 'suggestion', 5]);
      expect(PRIORITY_MATRIX_KEYS.distribution({ period: '7d' } as any)).toEqual(['priority-matrix', 'distribution', { period: '7d' }]);
    });
  });

  describe('useMatrixConfigsQuery', () => {
    it('should fetch matrix configs', async () => {
      mockApi.getMatrixConfigs.mockResolvedValue([{ id: 'cfg-1' }]);

      const { result } = renderHook(() => useMatrixConfigsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getMatrixConfigs).toHaveBeenCalled();
    });

    it('should handle error', async () => {
      mockApi.getMatrixConfigs.mockRejectedValue(new Error('Failed'));

      const { result } = renderHook(() => useMatrixConfigsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  describe('useActiveMatrixConfigQuery', () => {
    it('should fetch active config', async () => {
      mockApi.getActiveMatrixConfig.mockResolvedValue({ id: 'active-cfg', active: true });

      const { result } = renderHook(() => useActiveMatrixConfigQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getActiveMatrixConfig).toHaveBeenCalled();
      expect(result.current.data).toEqual({ id: 'active-cfg', active: true });
    });
  });

  describe('useMatrixDataQuery', () => {
    it('should fetch matrix data with configId', async () => {
      mockApi.getMatrixData.mockResolvedValue({ matrix: [] });

      const { result } = renderHook(() => useMatrixDataQuery('cfg-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getMatrixData).toHaveBeenCalledWith('cfg-1');
    });

    it('should fetch matrix data without configId', async () => {
      mockApi.getMatrixData.mockResolvedValue({ matrix: [] });

      const { result } = renderHook(() => useMatrixDataQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getMatrixData).toHaveBeenCalledWith(undefined);
    });
  });

  describe('usePriorityRulesQuery', () => {
    it('should fetch priority rules', async () => {
      mockApi.getPriorityRules.mockResolvedValue([{ id: 'r1' }]);

      const { result } = renderHook(() => usePriorityRulesQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getPriorityRules).toHaveBeenCalledWith(undefined);
    });

    it('should pass query params', async () => {
      mockApi.getPriorityRules.mockResolvedValue([]);
      const query = { active: true } as any;

      const { result } = renderHook(() => usePriorityRulesQuery(query), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getPriorityRules).toHaveBeenCalledWith(query);
    });
  });

  describe('usePrioritySuggestionQuery', () => {
    it('should fetch suggestion for a ticket', async () => {
      mockApi.getPrioritySuggestion.mockResolvedValue({ priority: 'high' });

      const { result } = renderHook(() => usePrioritySuggestionQuery(10), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getPrioritySuggestion).toHaveBeenCalledWith(10);
    });

    it('should not fetch when ticketId is 0', () => {
      renderHook(() => usePrioritySuggestionQuery(0), {
        wrapper: createWrapper(),
      });

      expect(mockApi.getPrioritySuggestion).not.toHaveBeenCalled();
    });

    it('should not fetch when disabled', () => {
      renderHook(() => usePrioritySuggestionQuery(10, false), {
        wrapper: createWrapper(),
      });

      expect(mockApi.getPrioritySuggestion).not.toHaveBeenCalled();
    });
  });

  describe('usePriorityDistributionQuery', () => {
    it('should fetch priority distribution', async () => {
      const query = { period: '7d' } as any;
      mockApi.getPriorityDistribution.mockResolvedValue({ data: [] });

      const { result } = renderHook(() => usePriorityDistributionQuery(query), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getPriorityDistribution).toHaveBeenCalledWith(query);
    });
  });

  describe('useCalculatePriorityMutation', () => {
    it('should calculate priority', async () => {
      mockApi.calculatePriority.mockResolvedValue({ priority: 'urgent', score: 95 });

      const { result } = renderHook(() => useCalculatePriorityMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ impact: 'high', urgency: 'high' } as any);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.calculatePriority).toHaveBeenCalled();
    });

    it('should handle calculation error', async () => {
      mockApi.calculatePriority.mockRejectedValue(new Error('Calc failed'));

      const { result } = renderHook(() => useCalculatePriorityMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ impact: 'low' } as any);

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  describe('useCreateMatrixConfigMutation', () => {
    it('should create matrix config', async () => {
      mockApi.createMatrixConfig.mockResolvedValue({ id: 'new-cfg' });

      const { result } = renderHook(() => useCreateMatrixConfigMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'New Config' } as any);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.createMatrixConfig).toHaveBeenCalled();
    });
  });

  describe('useActivateMatrixConfigMutation', () => {
    it('should activate a config', async () => {
      mockApi.activateMatrixConfig.mockResolvedValue({ id: 'cfg-1', active: true });

      const { result } = renderHook(() => useActivateMatrixConfigMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('cfg-1');

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.activateMatrixConfig).toHaveBeenCalledWith('cfg-1');
    });
  });

  describe('useCreatePriorityRuleMutation', () => {
    it('should create a priority rule', async () => {
      mockApi.createPriorityRule.mockResolvedValue({ id: 'rule-1' });

      const { result } = renderHook(() => useCreatePriorityRuleMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'Rule', condition: {} } as any);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.createPriorityRule).toHaveBeenCalled();
    });

    it('should handle create rule error', async () => {
      mockApi.createPriorityRule.mockRejectedValue(new Error('Failed'));

      const { result } = renderHook(() => useCreatePriorityRuleMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'Bad Rule' } as any);

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });
});
