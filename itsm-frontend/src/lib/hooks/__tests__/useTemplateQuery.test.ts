import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useTemplatesQuery,
  useTemplateQuery,
  useTemplateStatsQuery,
  useRecentTemplatesQuery,
  usePopularTemplatesQuery,
  useRecommendedTemplatesQuery,
  useFavoriteTemplatesQuery,
  useTemplateCategoriesQuery,
  useTemplateRatingsQuery,
  useTemplateVersionsQuery,
  useCreateTemplateMutation,
  useUpdateTemplateMutation,
  useDeleteTemplateMutation,
  usePublishTemplateMutation,
  useDuplicateTemplateMutation,
  useCreateTicketFromTemplateMutation,
  useRateTemplateMutation,
  useFavoriteTemplateMutation,
  useUnfavoriteTemplateMutation,
  useImportTemplateMutation,
  useArchiveTemplateMutation,
  useBatchDeleteTemplatesMutation,
  useBatchToggleTemplatesMutation,
  templateKeys,
} from '../useTemplateQuery';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
    warning: jest.fn(),
  },
}));

// Mock TemplateApi
jest.mock('@/lib/api/template-api', () => ({
  TemplateApi: {
    getTemplates: jest.fn(),
    getTemplate: jest.fn(),
    getTemplateStats: jest.fn(),
    getRecentTemplates: jest.fn(),
    getPopularTemplates: jest.fn(),
    getRecommendedTemplates: jest.fn(),
    getFavoriteTemplates: jest.fn(),
    getCategories: jest.fn(),
    getTemplateRatings: jest.fn(),
    getTemplateVersions: jest.fn(),
    createTemplate: jest.fn(),
    updateTemplate: jest.fn(),
    deleteTemplate: jest.fn(),
    publishTemplate: jest.fn(),
    duplicateTemplate: jest.fn(),
    createTicketFromTemplate: jest.fn(),
    recordTemplateUsage: jest.fn(),
    rateTemplate: jest.fn(),
    favoriteTemplate: jest.fn(),
    unfavoriteTemplate: jest.fn(),
    importTemplate: jest.fn(),
    archiveTemplate: jest.fn(),
    batchDeleteTemplates: jest.fn(),
    batchToggleTemplates: jest.fn(),
  },
}));

import { TemplateApi } from '@/lib/api/template-api';
const mockApi = TemplateApi as jest.Mocked<typeof TemplateApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useTemplateQuery hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('templateKeys', () => {
    it('should generate correct query keys', () => {
      expect(templateKeys.all).toEqual(['templates']);
      expect(templateKeys.lists()).toEqual(['templates', 'list']);
      expect(templateKeys.details()).toEqual(['templates', 'detail']);
      expect(templateKeys.detail('t1')).toEqual(['templates', 'detail', 't1']);
      expect(templateKeys.stats('t1')).toEqual(['templates', 'stats', 't1']);
      expect(templateKeys.recent()).toEqual(['templates', 'recent']);
      expect(templateKeys.popular()).toEqual(['templates', 'popular']);
      expect(templateKeys.recommended()).toEqual(['templates', 'recommended']);
      expect(templateKeys.favorites()).toEqual(['templates', 'favorites']);
      expect(templateKeys.categories()).toEqual(['template-categories']);
      expect(templateKeys.ratings('t1')).toEqual(['templates', 'ratings', 't1']);
      expect(templateKeys.versions('t1')).toEqual(['templates', 'versions', 't1']);
    });
  });

  describe('useTemplatesQuery', () => {
    it('should fetch templates', async () => {
      mockApi.getTemplates.mockResolvedValue({ items: [], total: 0 } as any);

      const { result } = renderHook(() => useTemplatesQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getTemplates).toHaveBeenCalledWith(undefined);
    });

    it('should pass query params', async () => {
      mockApi.getTemplates.mockResolvedValue({ items: [], total: 0 } as any);
      const query = { category: 'incident' } as any;

      const { result } = renderHook(() => useTemplatesQuery(query), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getTemplates).toHaveBeenCalledWith(query);
    });

    it('should handle error', async () => {
      mockApi.getTemplates.mockRejectedValue(new Error('Failed'));

      const { result } = renderHook(() => useTemplatesQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useTemplateQuery', () => {
    it('should fetch a single template', async () => {
      mockApi.getTemplate.mockResolvedValue({ id: 't1', name: 'Template 1' } as any);

      const { result } = renderHook(() => useTemplateQuery('t1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getTemplate).toHaveBeenCalledWith('t1');
    });

    it('should not fetch when templateId is empty', () => {
      renderHook(() => useTemplateQuery(''), { wrapper: createWrapper() });
      expect(mockApi.getTemplate).not.toHaveBeenCalled();
    });
  });

  describe('useTemplateStatsQuery', () => {
    it('should fetch template stats', async () => {
      mockApi.getTemplateStats.mockResolvedValue({ usageCount: 10 } as any);

      const { result } = renderHook(() => useTemplateStatsQuery('t1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getTemplateStats).toHaveBeenCalledWith('t1');
    });

    it('should not fetch when templateId is empty', () => {
      renderHook(() => useTemplateStatsQuery(''), { wrapper: createWrapper() });
      expect(mockApi.getTemplateStats).not.toHaveBeenCalled();
    });
  });

  describe('useRecentTemplatesQuery', () => {
    it('should fetch recent templates with default limit', async () => {
      mockApi.getRecentTemplates.mockResolvedValue([]);

      const { result } = renderHook(() => useRecentTemplatesQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getRecentTemplates).toHaveBeenCalledWith(10);
    });

    it('should pass custom limit', async () => {
      mockApi.getRecentTemplates.mockResolvedValue([]);

      const { result } = renderHook(() => useRecentTemplatesQuery(5), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getRecentTemplates).toHaveBeenCalledWith(5);
    });
  });

  describe('usePopularTemplatesQuery', () => {
    it('should fetch popular templates', async () => {
      mockApi.getPopularTemplates.mockResolvedValue([]);

      const { result } = renderHook(() => usePopularTemplatesQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getPopularTemplates).toHaveBeenCalledWith(10);
    });
  });

  describe('useRecommendedTemplatesQuery', () => {
    it('should fetch recommended templates', async () => {
      mockApi.getRecommendedTemplates.mockResolvedValue([]);

      const { result } = renderHook(() => useRecommendedTemplatesQuery('user-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getRecommendedTemplates).toHaveBeenCalledWith('user-1');
    });
  });

  describe('useFavoriteTemplatesQuery', () => {
    it('should fetch favorite templates', async () => {
      mockApi.getFavoriteTemplates.mockResolvedValue([]);

      const { result } = renderHook(() => useFavoriteTemplatesQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getFavoriteTemplates).toHaveBeenCalled();
    });
  });

  describe('useTemplateCategoriesQuery', () => {
    it('should fetch categories', async () => {
      mockApi.getCategories.mockResolvedValue([{ id: 'c1', name: 'Incident' }]);

      const { result } = renderHook(() => useTemplateCategoriesQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getCategories).toHaveBeenCalled();
    });
  });

  describe('useTemplateRatingsQuery', () => {
    it('should fetch ratings', async () => {
      mockApi.getTemplateRatings.mockResolvedValue([]);

      const { result } = renderHook(() => useTemplateRatingsQuery('t1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getTemplateRatings).toHaveBeenCalledWith('t1');
    });

    it('should not fetch when templateId is empty', () => {
      renderHook(() => useTemplateRatingsQuery(''), { wrapper: createWrapper() });
      expect(mockApi.getTemplateRatings).not.toHaveBeenCalled();
    });
  });

  describe('useTemplateVersionsQuery', () => {
    it('should fetch versions', async () => {
      mockApi.getTemplateVersions.mockResolvedValue([]);

      const { result } = renderHook(() => useTemplateVersionsQuery('t1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.getTemplateVersions).toHaveBeenCalledWith('t1');
    });
  });

  describe('useCreateTemplateMutation', () => {
    it('should create a template', async () => {
      mockApi.createTemplate.mockResolvedValue({ id: 'new-t' } as any);

      const { result } = renderHook(() => useCreateTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'New Template' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.createTemplate).toHaveBeenCalled();
    });

    it('should handle create error', async () => {
      mockApi.createTemplate.mockRejectedValue(new Error('Create failed'));

      const { result } = renderHook(() => useCreateTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'Bad' } as any);

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useUpdateTemplateMutation', () => {
    it('should update a template', async () => {
      mockApi.updateTemplate.mockResolvedValue({ id: 't1', name: 'Updated' } as any);

      const { result } = renderHook(() => useUpdateTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ id: 't1', data: { name: 'Updated' } as any });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.updateTemplate).toHaveBeenCalledWith('t1', { name: 'Updated' });
    });
  });

  describe('useDeleteTemplateMutation', () => {
    it('should delete a template', async () => {
      mockApi.deleteTemplate.mockResolvedValue(undefined);

      const { result } = renderHook(() => useDeleteTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('t1');

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.deleteTemplate).toHaveBeenCalledWith('t1');
    });
  });

  describe('usePublishTemplateMutation', () => {
    it('should publish a template', async () => {
      mockApi.publishTemplate.mockResolvedValue({ id: 't1', published: true } as any);

      const { result } = renderHook(() => usePublishTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ templateId: 't1', changelog: 'v2.0' });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.publishTemplate).toHaveBeenCalledWith('t1', 'v2.0');
    });
  });

  describe('useDuplicateTemplateMutation', () => {
    it('should duplicate a template', async () => {
      mockApi.duplicateTemplate.mockResolvedValue({ id: 'dup-t' } as any);

      const { result } = renderHook(() => useDuplicateTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ templateId: 't1', name: 'Copy of T1' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.duplicateTemplate).toHaveBeenCalled();
    });
  });

  describe('useCreateTicketFromTemplateMutation', () => {
    it('should create ticket from template', async () => {
      mockApi.createTicketFromTemplate.mockResolvedValue({ id: 1 } as any);
      mockApi.recordTemplateUsage.mockResolvedValue(undefined);

      const { result } = renderHook(() => useCreateTicketFromTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ templateId: 't1', title: 'New Ticket' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.createTicketFromTemplate).toHaveBeenCalled();
    });
  });

  describe('useRateTemplateMutation', () => {
    it('should rate a template', async () => {
      mockApi.rateTemplate.mockResolvedValue({ rating: 5 });

      const { result } = renderHook(() => useRateTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ templateId: 't1', rating: 5, comment: 'Great!' });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.rateTemplate).toHaveBeenCalledWith('t1', 5, 'Great!');
    });
  });

  describe('useFavoriteTemplateMutation', () => {
    it('should favorite a template', async () => {
      mockApi.favoriteTemplate.mockResolvedValue(undefined);

      const { result } = renderHook(() => useFavoriteTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('t1');

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.favoriteTemplate).toHaveBeenCalledWith('t1');
    });
  });

  describe('useUnfavoriteTemplateMutation', () => {
    it('should unfavorite a template', async () => {
      mockApi.unfavoriteTemplate.mockResolvedValue(undefined);

      const { result } = renderHook(() => useUnfavoriteTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('t1');

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.unfavoriteTemplate).toHaveBeenCalledWith('t1');
    });
  });

  describe('useImportTemplateMutation', () => {
    it('should import a template', async () => {
      mockApi.importTemplate.mockResolvedValue({ id: 'imported' });

      const { result } = renderHook(() => useImportTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ data: '{}', format: 'json' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useArchiveTemplateMutation', () => {
    it('should archive a template', async () => {
      mockApi.archiveTemplate.mockResolvedValue({ id: 't1', archived: true } as any);

      const { result } = renderHook(() => useArchiveTemplateMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('t1');

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.archiveTemplate).toHaveBeenCalledWith('t1');
    });
  });

  describe('useBatchDeleteTemplatesMutation', () => {
    it('should batch delete templates', async () => {
      mockApi.batchDeleteTemplates.mockResolvedValue({ success: 3, failed: 0 });

      const { result } = renderHook(() => useBatchDeleteTemplatesMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(['t1', 't2', 't3']);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.batchDeleteTemplates).toHaveBeenCalledWith(['t1', 't2', 't3']);
    });

    it('should handle partial failure', async () => {
      mockApi.batchDeleteTemplates.mockResolvedValue({ success: 2, failed: 1 });

      const { result } = renderHook(() => useBatchDeleteTemplatesMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(['t1', 't2', 't3']);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useBatchToggleTemplatesMutation', () => {
    it('should batch toggle templates', async () => {
      mockApi.batchToggleTemplates.mockResolvedValue({ success: 2, failed: 0 });

      const { result } = renderHook(() => useBatchToggleTemplatesMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ templateIds: ['t1', 't2'], isActive: true });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockApi.batchToggleTemplates).toHaveBeenCalledWith(['t1', 't2'], true);
    });

    it('should handle toggle error', async () => {
      mockApi.batchToggleTemplates.mockRejectedValue(new Error('Toggle failed'));

      const { result } = renderHook(() => useBatchToggleTemplatesMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ templateIds: ['t1'], isActive: false });

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });
});
