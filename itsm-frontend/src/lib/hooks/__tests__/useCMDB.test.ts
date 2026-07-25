import { renderHook, waitFor, act } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useCIsQuery, useCIQuery, useCIRelationshipsQuery,
  useRelationshipGraphQuery, useImpactAnalysisQuery,
  useCITypesQuery, useCIChangeHistoryQuery, useCMDBStatsQuery,
  useDiscoveryRulesQuery, useDiscoveryHistoryQuery,
  useCreateCIMutation, useUpdateCIMutation, useDeleteCIMutation,
  useBatchCreateCIsMutation, useCreateRelationshipMutation,
  useDeleteRelationshipMutation, useRunDiscoveryRuleMutation,
  CMDB_KEYS,
} from '../useCMDB';

jest.mock('antd', () => ({ message: { success: jest.fn(), error: jest.fn() } }));

jest.mock('@/lib/api/cmdb-api', () => ({
  CMDBApi: {
    getCIs: jest.fn(), getCI: jest.fn(), createCI: jest.fn(),
    updateCI: jest.fn(), deleteCI: jest.fn(), batchCreateCIs: jest.fn(),
    getCIRelationships: jest.fn(), getCITopology: jest.fn(),
    analyzeImpact: jest.fn(), getCITypes: jest.fn(),
    getCIChangeHistory: jest.fn(), getCMDBStats: jest.fn(),
    getDiscoveryRules: jest.fn(), getDiscoveryHistory: jest.fn(),
    createRelationship: jest.fn(), deleteRelationship: jest.fn(),
    runDiscoveryRule: jest.fn(),
  },
}));

import { CMDBApi } from '@/lib/api/cmdb-api';
import { message } from 'antd';
const mockApi = CMDBApi as jest.Mocked<typeof CMDBApi>;

const createWrapper = () => {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: qc }, children);
};

describe('useCMDB hooks', () => {
  beforeEach(() => jest.clearAllMocks());

  describe('CMDB_KEYS', () => {
    it('generates all key shapes', () => {
      expect(CMDB_KEYS.all).toEqual(['cmdb']);
      expect(CMDB_KEYS.cis()).toEqual(['cmdb', 'cis']);
      expect(CMDB_KEYS.ciDetail('c1')).toEqual(['cmdb', 'cis', 'detail', 'c1']);
      expect(CMDB_KEYS.ciRelationships('c1')).toEqual(['cmdb', 'cis', 'relationships', 'c1']);
      expect(CMDB_KEYS.ciChanges('c1')).toEqual(['cmdb', 'cis', 'changes', 'c1']);
      expect(CMDB_KEYS.ciTypes()).toEqual(['cmdb', 'ci-types']);
      expect(CMDB_KEYS.stats()).toEqual(['cmdb', 'stats']);
      expect(CMDB_KEYS.discovery()).toEqual(['cmdb', 'discovery']);
    });
  });

  describe('useCIsQuery', () => {
    it('fetches CIs', async () => {
      mockApi.getCIs.mockResolvedValue({ items: [] } as any);
      const { result } = renderHook(() => useCIsQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useCIQuery', () => {
    it('fetches a CI', async () => {
      mockApi.getCI.mockResolvedValue({ id: 'c1' } as any);
      const { result } = renderHook(() => useCIQuery('c1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useCIQuery('c1', false), { wrapper: createWrapper() });
      expect(mockApi.getCI).not.toHaveBeenCalled();
    });
  });

  describe('useCIRelationshipsQuery', () => {
    it('fetches relationships', async () => {
      mockApi.getCIRelationships.mockResolvedValue([] as any);
      const { result } = renderHook(() => useCIRelationshipsQuery('c1', { direction: 'both' }), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useCIRelationshipsQuery('c1', undefined, false), { wrapper: createWrapper() });
      expect(mockApi.getCIRelationships).not.toHaveBeenCalled();
    });
  });

  describe('useRelationshipGraphQuery', () => {
    it('fetches graph', async () => {
      mockApi.getCITopology.mockResolvedValue({ nodes: [], edges: [] } as any);
      const { result } = renderHook(() => useRelationshipGraphQuery({ rootCI: '1', depth: 2 } as any), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useRelationshipGraphQuery({ rootCI: '1', depth: 2 } as any, false), { wrapper: createWrapper() });
      expect(mockApi.getCITopology).not.toHaveBeenCalled();
    });
  });

  describe('useImpactAnalysisQuery', () => {
    it('fetches impact analysis', async () => {
      mockApi.analyzeImpact.mockResolvedValue({ impactedCIs: [] } as any);
      const { result } = renderHook(() => useImpactAnalysisQuery({ ciId: 'c1' } as any), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useImpactAnalysisQuery({ ciId: 'c1' } as any, false), { wrapper: createWrapper() });
      expect(mockApi.analyzeImpact).not.toHaveBeenCalled();
    });
  });

  describe('useCITypesQuery', () => {
    it('fetches CI types', async () => {
      mockApi.getCITypes.mockResolvedValue([] as any);
      const { result } = renderHook(() => useCITypesQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useCIChangeHistoryQuery', () => {
    it('fetches change history', async () => {
      mockApi.getCIChangeHistory.mockResolvedValue({ items: [] } as any);
      const { result } = renderHook(() => useCIChangeHistoryQuery('c1', { page: 1 }), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useCIChangeHistoryQuery('c1', undefined, false), { wrapper: createWrapper() });
      expect(mockApi.getCIChangeHistory).not.toHaveBeenCalled();
    });
  });

  describe('useCMDBStatsQuery', () => {
    it('fetches CMDB stats', async () => {
      mockApi.getCMDBStats.mockResolvedValue({ total: 100 } as any);
      const { result } = renderHook(() => useCMDBStatsQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useDiscoveryRulesQuery', () => {
    it('fetches discovery rules', async () => {
      mockApi.getDiscoveryRules.mockResolvedValue([] as any);
      const { result } = renderHook(() => useDiscoveryRulesQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useDiscoveryHistoryQuery', () => {
    it('fetches discovery history', async () => {
      mockApi.getDiscoveryHistory.mockResolvedValue([] as any);
      const { result } = renderHook(() => useDiscoveryHistoryQuery('rule-1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useCreateCIMutation', () => {
    it('creates CI', async () => {
      mockApi.createCI.mockResolvedValue({ id: 'new-ci' } as any);
      const { result } = renderHook(() => useCreateCIMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ name: 'Server' } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useUpdateCIMutation', () => {
    it('updates CI', async () => {
      mockApi.updateCI.mockResolvedValue({ id: 'c1' } as any);
      const { result } = renderHook(() => useUpdateCIMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ id: 'c1', data: { name: 'Updated' } } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useDeleteCIMutation', () => {
    it('deletes CI', async () => {
      mockApi.deleteCI.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useDeleteCIMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('c1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useBatchCreateCIsMutation', () => {
    it('batch creates CIs', async () => {
      mockApi.batchCreateCIs.mockResolvedValue([{ id: 'a' }, { id: 'b' }] as any);
      const { result } = renderHook(() => useBatchCreateCIsMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate([{ name: 'A' }, { name: 'B' }] as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useCreateRelationshipMutation', () => {
    it('creates relationship', async () => {
      mockApi.createRelationship.mockResolvedValue({ parentId: 'c1', childId: 'c2' } as any);
      const { result } = renderHook(() => useCreateRelationshipMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ sourceCI: 'c1', targetCI: 'c2', type: 'depends_on' } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useDeleteRelationshipMutation', () => {
    it('deletes relationship', async () => {
      mockApi.deleteRelationship.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useDeleteRelationshipMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('rel-1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useRunDiscoveryRuleMutation', () => {
    it('runs discovery rule', async () => {
      mockApi.runDiscoveryRule.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useRunDiscoveryRuleMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('rule-1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });
});
