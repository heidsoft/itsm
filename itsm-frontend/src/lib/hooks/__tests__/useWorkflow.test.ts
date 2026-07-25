import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useWorkflowsQuery, useWorkflowQuery, useCreateWorkflowMutation, WORKFLOW_KEYS } from '../useWorkflow';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock WorkflowApi
jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: {
    getWorkflows: jest.fn(),
    getWorkflow: jest.fn(),
    createWorkflow: jest.fn(),
    updateWorkflow: jest.fn(),
    deleteWorkflow: jest.fn(),
    activateWorkflow: jest.fn(),
    deactivateWorkflow: jest.fn(),
    cloneWorkflow: jest.fn(),
    startWorkflow: jest.fn(),
    cancelInstance: jest.fn(),
    completeNode: jest.fn(),
    getWorkflowVersions: jest.fn(),
    getWorkflowStats: jest.fn(),
    getInstances: jest.fn(),
    getInstance: jest.fn(),
    getNodeInstances: jest.fn(),
    getTemplates: jest.fn(),
  },
}));

import { WorkflowApi } from '@/lib/api/workflow-api';
const mockWorkflowApi = WorkflowApi as jest.Mocked<typeof WorkflowApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useWorkflow hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('WORKFLOW_KEYS', () => {
    it('should generate correct query keys', () => {
      expect(WORKFLOW_KEYS.all).toEqual(['workflows']);
      expect(WORKFLOW_KEYS.lists()).toEqual(['workflows', 'list']);
      expect(WORKFLOW_KEYS.detail('123')).toEqual(['workflows', 'detail', '123']);
    });
  });

  describe('useWorkflowsQuery', () => {
    it('should fetch workflows', async () => {
      mockWorkflowApi.getWorkflows.mockResolvedValue({ items: [], total: 0 });

      const { result } = renderHook(() => useWorkflowsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockWorkflowApi.getWorkflows).toHaveBeenCalled();
    });
  });

  describe('useWorkflowQuery', () => {
    it('should fetch a single workflow', async () => {
      mockWorkflowApi.getWorkflow.mockResolvedValue({ id: '123', name: 'Test' });

      const { result } = renderHook(() => useWorkflowQuery('123'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockWorkflowApi.getWorkflow).toHaveBeenCalledWith('123');
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useWorkflowQuery('123', false), {
        wrapper: createWrapper(),
      });

      expect(mockWorkflowApi.getWorkflow).not.toHaveBeenCalled();
    });
  });

  describe('useCreateWorkflowMutation', () => {
    it('should create a workflow', async () => {
      mockWorkflowApi.createWorkflow.mockResolvedValue({ id: 'new-1' });

      const { result } = renderHook(() => useCreateWorkflowMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'New Workflow' } as any);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockWorkflowApi.createWorkflow).toHaveBeenCalledWith({ name: 'New Workflow' }, expect.anything());
    });
  });
});
