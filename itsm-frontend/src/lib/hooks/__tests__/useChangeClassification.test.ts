import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useClassificationsQuery,
  useClassificationQuery,
  useClassificationSuggestionQuery,
  useClassificationRulesQuery,
  useChangeTemplatesQuery,
  useCreateClassificationMutation,
  useUpdateClassificationMutation,
  useAssessRiskMutation,
  useAnalyzeImpactMutation,
  useApplyClassificationMutation,
  CHANGE_CLASSIFICATION_KEYS,
} from '../useChangeClassification';
import { ChangeType } from '@/types/change-classification';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock ChangeClassificationApi
jest.mock('@/lib/api/change-classification-api', () => ({
  ChangeClassificationApi: {
    getClassifications: jest.fn(),
    getClassification: jest.fn(),
    getClassificationSuggestion: jest.fn(),
    getClassificationRules: jest.fn(),
    getChangeTemplates: jest.fn(),
    createClassification: jest.fn(),
    updateClassification: jest.fn(),
    assessRisk: jest.fn(),
    analyzeImpact: jest.fn(),
    applyClassificationSuggestion: jest.fn(),
  },
}));

import { ChangeClassificationApi } from '@/lib/api/change-classification-api';
const mockApi = ChangeClassificationApi as jest.Mocked<typeof ChangeClassificationApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useChangeClassification hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('CHANGE_CLASSIFICATION_KEYS', () => {
    it('should generate correct query keys', () => {
      expect(CHANGE_CLASSIFICATION_KEYS.all).toEqual(['change-classifications']);
      expect(CHANGE_CLASSIFICATION_KEYS.lists()).toEqual(['change-classifications', 'list']);
      expect(CHANGE_CLASSIFICATION_KEYS.list({ page: 1 } as any)).toEqual([
        'change-classifications', 'list', { page: 1 },
      ]);
      expect(CHANGE_CLASSIFICATION_KEYS.details()).toEqual(['change-classifications', 'detail']);
      expect(CHANGE_CLASSIFICATION_KEYS.detail('abc')).toEqual(['change-classifications', 'detail', 'abc']);
      expect(CHANGE_CLASSIFICATION_KEYS.suggestion(1)).toEqual(['change-classifications', 'suggestion', 1]);
      expect(CHANGE_CLASSIFICATION_KEYS.rules()).toEqual(['change-classifications', 'rules']);
      expect(CHANGE_CLASSIFICATION_KEYS.templates('cls-1')).toEqual(['change-classifications', 'templates', 'cls-1']);
    });
  });

  describe('useClassificationsQuery', () => {
    it('should fetch classifications list', async () => {
      mockApi.getClassifications.mockResolvedValue({ classifications: [{ id: '1', name: 'Test', code: 'test', type: ChangeType.NORMAL, riskLevel: 'low', approvalRequired: false, cabRequired: false, testingRequired: false, backoutPlanRequired: false, businessJustificationRequired: false, isActive: true, createdAt: new Date(), updatedAt: new Date() }], total: 1 });

      const { result } = renderHook(() => useClassificationsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getClassifications).toHaveBeenCalledWith(undefined);
    });

    it('should pass query params', async () => {
      mockApi.getClassifications.mockResolvedValue({ classifications: [], total: 0 });
      const query = { page: 1, pageSize: 10 } as any;

      const { result } = renderHook(() => useClassificationsQuery(query), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getClassifications).toHaveBeenCalledWith(query);
    });

    it('should handle error state', async () => {
      mockApi.getClassifications.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useClassificationsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toBeDefined();
    });
  });

  describe('useClassificationQuery', () => {
    it('should fetch a single classification', async () => {
      mockApi.getClassification.mockResolvedValue({ id: 'cls-1', name: 'Standard', code: 'standard', type: ChangeType.NORMAL, riskLevel: 'low', approvalRequired: false, cabRequired: false, testingRequired: false, backoutPlanRequired: false, businessJustificationRequired: false, isActive: true, createdAt: new Date(), updatedAt: new Date() });

      const { result } = renderHook(() => useClassificationQuery('cls-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getClassification).toHaveBeenCalledWith('cls-1');
    });

    it('should not fetch when id is empty', () => {
      renderHook(() => useClassificationQuery(''), {
        wrapper: createWrapper(),
      });

      expect(mockApi.getClassification).not.toHaveBeenCalled();
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useClassificationQuery('cls-1', false), {
        wrapper: createWrapper(),
      });

      expect(mockApi.getClassification).not.toHaveBeenCalled();
    });
  });

  describe('useClassificationSuggestionQuery', () => {
    it('should fetch suggestion for a change', async () => {
      mockApi.getClassificationSuggestion.mockResolvedValue({ changeId: 42, suggestedClassification: { id: 'cls-1', name: 'Standard', code: 'standard', type: ChangeType.NORMAL, riskLevel: 'low', approvalRequired: false, cabRequired: false, testingRequired: false, backoutPlanRequired: false, businessJustificationRequired: false, isActive: true, createdAt: new Date(), updatedAt: new Date() }, confidence: 85, reasoning: 'Based on historical data', basedOn: 'historical' });

      const { result } = renderHook(() => useClassificationSuggestionQuery(42), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getClassificationSuggestion).toHaveBeenCalledWith(42);
    });

    it('should not fetch when changeId is 0', () => {
      renderHook(() => useClassificationSuggestionQuery(0), {
        wrapper: createWrapper(),
      });

      expect(mockApi.getClassificationSuggestion).not.toHaveBeenCalled();
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useClassificationSuggestionQuery(42, false), {
        wrapper: createWrapper(),
      });

      expect(mockApi.getClassificationSuggestion).not.toHaveBeenCalled();
    });
  });

  describe('useClassificationRulesQuery', () => {
    it('should fetch classification rules', async () => {
      mockApi.getClassificationRules.mockResolvedValue([{ id: 'r1', name: 'Rule 1', priority: 1, conditions: [], resultClassification: 'cls-1', enabled: true, createdAt: new Date(), updatedAt: new Date() }]);

      const { result } = renderHook(() => useClassificationRulesQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getClassificationRules).toHaveBeenCalled();
    });
  });

  describe('useChangeTemplatesQuery', () => {
    it('should fetch templates', async () => {
      mockApi.getChangeTemplates.mockResolvedValue([]);

      const { result } = renderHook(() => useChangeTemplatesQuery('cls-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getChangeTemplates).toHaveBeenCalledWith({ classificationId: 'cls-1' });
    });

    it('should fetch without classificationId', async () => {
      mockApi.getChangeTemplates.mockResolvedValue([]);

      const { result } = renderHook(() => useChangeTemplatesQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getChangeTemplates).toHaveBeenCalledWith({ classificationId: undefined });
    });
  });

  describe('useCreateClassificationMutation', () => {
    it('should create classification on success', async () => {
      mockApi.createClassification.mockResolvedValue({ id: 'new-1', name: 'New Classification', code: 'new-classification', type: ChangeType.NORMAL, riskLevel: 'low', approvalRequired: false, cabRequired: false, testingRequired: false, backoutPlanRequired: false, businessJustificationRequired: false, isActive: true, createdAt: new Date(), updatedAt: new Date() });

      const { result } = renderHook(() => useCreateClassificationMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'New Classification' } as any);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.createClassification).toHaveBeenCalled();
    });

    it('should handle create error', async () => {
      mockApi.createClassification.mockRejectedValue(new Error('Create failed'));

      const { result } = renderHook(() => useCreateClassificationMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'Fail' } as any);

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  describe('useUpdateClassificationMutation', () => {
    it('should update classification', async () => {
      mockApi.updateClassification.mockResolvedValue({ id: 'cls-1', name: 'Updated', code: 'updated', type: ChangeType.NORMAL, riskLevel: 'low', approvalRequired: false, cabRequired: false, testingRequired: false, backoutPlanRequired: false, businessJustificationRequired: false, isActive: true, createdAt: new Date(), updatedAt: new Date() });

      const { result } = renderHook(() => useUpdateClassificationMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ id: 'cls-1', data: { name: 'Updated' } as any });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.updateClassification).toHaveBeenCalledWith('cls-1', { name: 'Updated' });
    });

    it('should handle update error', async () => {
      mockApi.updateClassification.mockRejectedValue(new Error('Update failed'));

      const { result } = renderHook(() => useUpdateClassificationMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ id: 'cls-1', data: {} as any });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  describe('useAssessRiskMutation', () => {
    it('should assess risk', async () => {
      mockApi.assessRisk.mockResolvedValue({ riskLevel: 'medium' } as any);

      const { result } = renderHook(() => useAssessRiskMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ changeId: 1 } as any);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.assessRisk).toHaveBeenCalled();
    });
  });

  describe('useAnalyzeImpactMutation', () => {
    it('should analyze impact', async () => {
      mockApi.analyzeImpact.mockResolvedValue({ impactLevel: 'high' } as any);

      const { result } = renderHook(() => useAnalyzeImpactMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ changeId: 1 } as any);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.analyzeImpact).toHaveBeenCalled();
    });
  });

  describe('useApplyClassificationMutation', () => {
    it('should apply classification', async () => {
      mockApi.applyClassificationSuggestion.mockResolvedValue({ success: true } as any);

      const { result } = renderHook(() => useApplyClassificationMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ changeId: 1, classificationId: 'cls-1' });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.applyClassificationSuggestion).toHaveBeenCalledWith(1, 'cls-1');
    });

    it('should handle apply error', async () => {
      mockApi.applyClassificationSuggestion.mockRejectedValue(new Error('Apply failed'));

      const { result } = renderHook(() => useApplyClassificationMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ changeId: 1, classificationId: 'cls-x' });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });
});
