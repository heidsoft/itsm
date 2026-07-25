import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useTicketRelationsQuery,
  useRelationQuery,
  useTicketHierarchyQuery,
  useTicketDependenciesQuery,
  useDependencyGraphQuery,
  useRelationStatsQuery,
  useRelationGraphQuery,
  useRelationSuggestionsQuery,
  useRelationPermissionsQuery,
  useCreateRelationMutation,
  useBatchCreateRelationsMutation,
  useUpdateRelationMutation,
  useDeleteRelationMutation,
  useSetParentMutation,
  useRemoveParentMutation,
  useAddDependencyMutation,
  useRemoveDependencyMutation,
  TICKET_RELATION_KEYS,
} from '../useTicketRelations';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock TicketRelationsApi
jest.mock('@/lib/api/ticket-relations-api', () => ({
  TicketRelationsApi: {
    getTicketRelations: jest.fn(),
    getRelation: jest.fn(),
    getHierarchy: jest.fn(),
    getDependencies: jest.fn(),
    getDependencyGraph: jest.fn(),
    getRelationStats: jest.fn(),
    getRelationGraph: jest.fn(),
    getRelationSuggestions: jest.fn(),
    getRelationPermissions: jest.fn(),
    createRelation: jest.fn(),
    batchCreateRelations: jest.fn(),
    updateRelation: jest.fn(),
    deleteRelation: jest.fn(),
    setParent: jest.fn(),
    removeParent: jest.fn(),
    addDependency: jest.fn(),
    removeDependency: jest.fn(),
  },
}));

import { TicketRelationsApi } from '@/lib/api/ticket-relations-api';
const mockApi = TicketRelationsApi as jest.Mocked<typeof TicketRelationsApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useTicketRelations hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('TICKET_RELATION_KEYS', () => {
    it('should generate correct query keys', () => {
      expect(TICKET_RELATION_KEYS.all).toEqual(['ticket-relations']);
      expect(TICKET_RELATION_KEYS.lists()).toEqual(['ticket-relations', 'list']);
      expect(TICKET_RELATION_KEYS.detail('rel-1')).toEqual(['ticket-relations', 'detail', 'rel-1']);
      expect(TICKET_RELATION_KEYS.hierarchy(1)).toEqual(['ticket-relations', 'hierarchy', 1]);
      expect(TICKET_RELATION_KEYS.dependencies(1)).toEqual(['ticket-relations', 'dependencies', 1]);
      expect(TICKET_RELATION_KEYS.dependencyGraph(1)).toEqual(['ticket-relations', 'dependency-graph', 1]);
      expect(TICKET_RELATION_KEYS.stats(1)).toEqual(['ticket-relations', 'stats', 1]);
      expect(TICKET_RELATION_KEYS.suggestions(1)).toEqual(['ticket-relations', 'suggestions', 1]);
      expect(TICKET_RELATION_KEYS.permissions(1)).toEqual(['ticket-relations', 'permissions', 1]);
    });
  });

  describe('useTicketRelationsQuery', () => {
    it('should fetch ticket relations', async () => {
      mockApi.getTicketRelations.mockResolvedValue([]);

      const { result } = renderHook(() => useTicketRelationsQuery(1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getTicketRelations).toHaveBeenCalledWith(1, {
        relationType: undefined,
        direction: undefined,
        includeDetails: undefined,
      });
    });

    it('should pass options to API', async () => {
      mockApi.getTicketRelations.mockResolvedValue([]);

      const { result } = renderHook(
        () => useTicketRelationsQuery(1, { relationType: 'blocks', direction: 'outgoing', includeDetails: true }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getTicketRelations).toHaveBeenCalledWith(1, {
        relationType: 'blocks',
        direction: 'outgoing',
        includeDetails: true,
      });
    });

    it('should not fetch when disabled', () => {
      renderHook(
        () => useTicketRelationsQuery(1, { enabled: false }),
        { wrapper: createWrapper() }
      );
      expect(mockApi.getTicketRelations).not.toHaveBeenCalled();
    });

    it('should not fetch when ticketId is 0', () => {
      renderHook(() => useTicketRelationsQuery(0), { wrapper: createWrapper() });
      expect(mockApi.getTicketRelations).not.toHaveBeenCalled();
    });
  });

  describe('useRelationQuery', () => {
    it('should fetch a single relation', async () => {
      mockApi.getRelation.mockResolvedValue({ id: 'rel-1' });

      const { result } = renderHook(() => useRelationQuery('rel-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getRelation).toHaveBeenCalledWith('rel-1');
    });

    it('should not fetch when id is empty', () => {
      renderHook(() => useRelationQuery(''), { wrapper: createWrapper() });
      expect(mockApi.getRelation).not.toHaveBeenCalled();
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useRelationQuery('rel-1', false), { wrapper: createWrapper() });
      expect(mockApi.getRelation).not.toHaveBeenCalled();
    });
  });

  describe('useTicketHierarchyQuery', () => {
    it('should fetch hierarchy', async () => {
      mockApi.getHierarchy.mockResolvedValue({ parent: null, children: [] });

      const { result } = renderHook(() => useTicketHierarchyQuery(1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getHierarchy).toHaveBeenCalledWith(1);
    });

    it('should not fetch when ticketId is 0', () => {
      renderHook(() => useTicketHierarchyQuery(0), { wrapper: createWrapper() });
      expect(mockApi.getHierarchy).not.toHaveBeenCalled();
    });
  });

  describe('useTicketDependenciesQuery', () => {
    it('should fetch dependencies', async () => {
      mockApi.getDependencies.mockResolvedValue([]);

      const { result } = renderHook(() => useTicketDependenciesQuery(1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getDependencies).toHaveBeenCalledWith(1);
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useTicketDependenciesQuery(1, false), { wrapper: createWrapper() });
      expect(mockApi.getDependencies).not.toHaveBeenCalled();
    });
  });

  describe('useDependencyGraphQuery', () => {
    it('should fetch dependency graph', async () => {
      mockApi.getDependencyGraph.mockResolvedValue({ nodes: [], edges: [] });

      const { result } = renderHook(() => useDependencyGraphQuery(1, 3), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getDependencyGraph).toHaveBeenCalledWith(1, 3);
    });

    it('should not fetch when ticketId is 0', () => {
      renderHook(() => useDependencyGraphQuery(0), { wrapper: createWrapper() });
      expect(mockApi.getDependencyGraph).not.toHaveBeenCalled();
    });
  });

  describe('useRelationStatsQuery', () => {
    it('should fetch relation stats', async () => {
      mockApi.getRelationStats.mockResolvedValue({ total: 5 });

      const { result } = renderHook(() => useRelationStatsQuery(1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getRelationStats).toHaveBeenCalledWith(1);
    });
  });

  describe('useRelationGraphQuery', () => {
    it('should fetch relation graph', async () => {
      mockApi.getRelationGraph.mockResolvedValue({ nodes: [], edges: [] });

      const { result } = renderHook(
        () => useRelationGraphQuery(1, { maxDepth: 2, relationTypes: ['blocks'] }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getRelationGraph).toHaveBeenCalledWith(1, { maxDepth: 2, relationTypes: ['blocks'] });
    });
  });

  describe('useRelationSuggestionsQuery', () => {
    it('should fetch suggestions', async () => {
      mockApi.getRelationSuggestions.mockResolvedValue([]);

      const { result } = renderHook(() => useRelationSuggestionsQuery(1, 5), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getRelationSuggestions).toHaveBeenCalledWith(1, 5);
    });
  });

  describe('useRelationPermissionsQuery', () => {
    it('should fetch permissions', async () => {
      mockApi.getRelationPermissions.mockResolvedValue({ canCreate: true });

      const { result } = renderHook(() => useRelationPermissionsQuery(1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getRelationPermissions).toHaveBeenCalledWith(1);
    });
  });

  describe('useCreateRelationMutation', () => {
    it('should create a relation', async () => {
      mockApi.createRelation.mockResolvedValue({ id: 'new-rel' });

      const { result } = renderHook(() => useCreateRelationMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ sourceTicketId: 1, targetTicketId: 2, relationType: 'blocks' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.createRelation).toHaveBeenCalled();
    });

    it('should handle create error', async () => {
      mockApi.createRelation.mockRejectedValue(new Error('Create failed'));

      const { result } = renderHook(() => useCreateRelationMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ sourceTicketId: 1, targetTicketId: 2 } as any);

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useBatchCreateRelationsMutation', () => {
    it('should batch create relations', async () => {
      mockApi.batchCreateRelations.mockResolvedValue({ created: 3, failed: 0 });

      const { result } = renderHook(() => useBatchCreateRelationsMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ sourceTicketId: 1, relations: [] } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });

    it('should handle batch create error', async () => {
      mockApi.batchCreateRelations.mockRejectedValue(new Error('Batch failed'));

      const { result } = renderHook(() => useBatchCreateRelationsMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ sourceTicketId: 1, relations: [] } as any);

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useUpdateRelationMutation', () => {
    it('should update a relation', async () => {
      mockApi.updateRelation.mockResolvedValue({ id: 'rel-1' });

      const { result } = renderHook(() => useUpdateRelationMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ relationId: 'rel-1', request: { notes: 'Updated' } as any });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.updateRelation).toHaveBeenCalledWith('rel-1', { notes: 'Updated' });
    });
  });

  describe('useDeleteRelationMutation', () => {
    it('should delete a relation', async () => {
      mockApi.deleteRelation.mockResolvedValue(undefined);

      const { result } = renderHook(() => useDeleteRelationMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ relationId: 'rel-1', reason: 'No longer needed' });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.deleteRelation).toHaveBeenCalledWith('rel-1', 'No longer needed');
    });

    it('should handle delete error', async () => {
      mockApi.deleteRelation.mockRejectedValue(new Error('Delete failed'));

      const { result } = renderHook(() => useDeleteRelationMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ relationId: 'rel-1' });

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useSetParentMutation', () => {
    it('should set parent', async () => {
      mockApi.setParent.mockResolvedValue(undefined);

      const { result } = renderHook(() => useSetParentMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ childTicketId: 2, parentTicketId: 1 });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.setParent).toHaveBeenCalledWith(2, 1);
    });

    it('should handle set parent error', async () => {
      mockApi.setParent.mockRejectedValue(new Error('Circular'));

      const { result } = renderHook(() => useSetParentMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ childTicketId: 1, parentTicketId: 1 });

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useRemoveParentMutation', () => {
    it('should remove parent', async () => {
      mockApi.removeParent.mockResolvedValue(undefined);

      const { result } = renderHook(() => useRemoveParentMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(2);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.removeParent).toHaveBeenCalledWith(2);
    });
  });

  describe('useAddDependencyMutation', () => {
    it('should add dependency', async () => {
      mockApi.addDependency.mockResolvedValue(undefined);

      const { result } = renderHook(() => useAddDependencyMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ ticketId: 1, dependsOnTicketId: 2, dependencyType: 'hard' });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.addDependency).toHaveBeenCalledWith({
        ticketId: 1, dependsOnTicketId: 2, dependencyType: 'hard',
      });
    });
  });

  describe('useRemoveDependencyMutation', () => {
    it('should remove dependency', async () => {
      mockApi.removeDependency.mockResolvedValue(undefined);

      const { result } = renderHook(() => useRemoveDependencyMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ ticketId: 1, dependencyId: 'dep-1' });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.removeDependency).toHaveBeenCalledWith(1, 'dep-1');
    });

    it('should handle remove dependency error', async () => {
      mockApi.removeDependency.mockRejectedValue(new Error('Not found'));

      const { result } = renderHook(() => useRemoveDependencyMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ ticketId: 1, dependencyId: 'bad' });

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });
});
