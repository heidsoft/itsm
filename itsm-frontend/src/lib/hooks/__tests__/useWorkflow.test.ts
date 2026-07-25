import { renderHook, waitFor, act } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useWorkflowsQuery, useWorkflowQuery, useWorkflowVersionsQuery,
  useWorkflowStatsQuery, useWorkflowInstancesQuery, useWorkflowInstanceQuery,
  useNodeInstancesQuery, useWorkflowTemplatesQuery,
  useCreateWorkflowMutation, useUpdateWorkflowMutation, useDeleteWorkflowMutation,
  useActivateWorkflowMutation, useDeactivateWorkflowMutation,
  useCloneWorkflowMutation, useStartWorkflowMutation,
  useCancelInstanceMutation, useCompleteNodeMutation,
  WORKFLOW_KEYS,
} from '../useWorkflow';

jest.mock('antd', () => ({ message: { success: jest.fn(), error: jest.fn() } }));

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: {
    getWorkflows: jest.fn(), getWorkflow: jest.fn(), createWorkflow: jest.fn(),
    updateWorkflow: jest.fn(), deleteWorkflow: jest.fn(),
    activateWorkflow: jest.fn(), deactivateWorkflow: jest.fn(),
    cloneWorkflow: jest.fn(), startWorkflow: jest.fn(),
    cancelInstance: jest.fn(), completeNode: jest.fn(),
    getWorkflowVersions: jest.fn(), getWorkflowStats: jest.fn(),
    getInstances: jest.fn(), getInstance: jest.fn(),
    getNodeInstances: jest.fn(), getTemplates: jest.fn(),
  },
}));

import { WorkflowApi } from '@/lib/api/workflow-api';
import { message } from 'antd';
const mockApi = WorkflowApi as jest.Mocked<typeof WorkflowApi>;

const createWrapper = () => {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: qc }, children);
};

describe('useWorkflow hooks', () => {
  beforeEach(() => jest.clearAllMocks());

  describe('WORKFLOW_KEYS', () => {
    it('generates all key shapes', () => {
      expect(WORKFLOW_KEYS.all).toEqual(['workflows']);
      expect(WORKFLOW_KEYS.lists()).toEqual(['workflows', 'list']);
      expect(WORKFLOW_KEYS.detail('w1')).toEqual(['workflows', 'detail', 'w1']);
      expect(WORKFLOW_KEYS.versions('w1')).toEqual(['workflows', 'versions', 'w1']);
      expect(WORKFLOW_KEYS.stats('w1')).toEqual(['workflows', 'stats', 'w1']);
      expect(WORKFLOW_KEYS.instances()).toEqual(['workflows', 'instances']);
      expect(WORKFLOW_KEYS.instance('i1')).toEqual(['workflows', 'instances', 'i1']);
      expect(WORKFLOW_KEYS.nodeInstances('i1')).toEqual(['workflows', 'node-instances', 'i1']);
      expect(WORKFLOW_KEYS.templates()).toEqual(['workflows', 'templates']);
    });
  });

  describe('useWorkflowsQuery', () => {
    it('fetches workflows', async () => {
      mockApi.getWorkflows.mockResolvedValue({ items: [] } as any);
      const { result } = renderHook(() => useWorkflowsQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useWorkflowQuery', () => {
    it('fetches a workflow', async () => {
      mockApi.getWorkflow.mockResolvedValue({ id: 'w1' } as any);
      const { result } = renderHook(() => useWorkflowQuery('w1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useWorkflowQuery('w1', false), { wrapper: createWrapper() });
      expect(mockApi.getWorkflow).not.toHaveBeenCalled();
    });
  });

  describe('useWorkflowVersionsQuery', () => {
    it('fetches versions', async () => {
      mockApi.getWorkflowVersions.mockResolvedValue([] as any);
      const { result } = renderHook(() => useWorkflowVersionsQuery('w1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useWorkflowVersionsQuery('w1', false), { wrapper: createWrapper() });
      expect(mockApi.getWorkflowVersions).not.toHaveBeenCalled();
    });
  });

  describe('useWorkflowStatsQuery', () => {
    it('fetches stats', async () => {
      mockApi.getWorkflowStats.mockResolvedValue({ total: 5 } as any);
      const { result } = renderHook(() => useWorkflowStatsQuery('w1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useWorkflowStatsQuery('w1', undefined, false), { wrapper: createWrapper() });
      expect(mockApi.getWorkflowStats).not.toHaveBeenCalled();
    });
  });

  describe('useWorkflowInstancesQuery', () => {
    it('fetches instances', async () => {
      mockApi.getInstances.mockResolvedValue({ items: [] } as any);
      const { result } = renderHook(() => useWorkflowInstancesQuery({ status: 'running' }), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useWorkflowInstanceQuery', () => {
    it('fetches a single instance', async () => {
      mockApi.getInstance.mockResolvedValue({ id: 'i1' } as any);
      const { result } = renderHook(() => useWorkflowInstanceQuery('i1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useWorkflowInstanceQuery('i1', false), { wrapper: createWrapper() });
      expect(mockApi.getInstance).not.toHaveBeenCalled();
    });
  });

  describe('useNodeInstancesQuery', () => {
    it('fetches node instances', async () => {
      mockApi.getNodeInstances.mockResolvedValue([] as any);
      const { result } = renderHook(() => useNodeInstancesQuery('i1'), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
    it('does not fetch when disabled', () => {
      renderHook(() => useNodeInstancesQuery('i1', false), { wrapper: createWrapper() });
      expect(mockApi.getNodeInstances).not.toHaveBeenCalled();
    });
  });

  describe('useWorkflowTemplatesQuery', () => {
    it('fetches templates', async () => {
      mockApi.getTemplates.mockResolvedValue([] as any);
      const { result } = renderHook(() => useWorkflowTemplatesQuery(), { wrapper: createWrapper() });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
    });
  });

  describe('useCreateWorkflowMutation', () => {
    it('creates workflow', async () => {
      mockApi.createWorkflow.mockResolvedValue({ id: 'new' } as any);
      const { result } = renderHook(() => useCreateWorkflowMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ name: 'WF' } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useUpdateWorkflowMutation', () => {
    it('updates workflow', async () => {
      mockApi.updateWorkflow.mockResolvedValue({ id: 'w1' } as any);
      const { result } = renderHook(() => useUpdateWorkflowMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ id: 'w1', data: { name: 'Updated' } } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useDeleteWorkflowMutation', () => {
    it('deletes workflow', async () => {
      mockApi.deleteWorkflow.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useDeleteWorkflowMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('w1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useActivateWorkflowMutation', () => {
    it('activates workflow', async () => {
      mockApi.activateWorkflow.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useActivateWorkflowMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('w1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useDeactivateWorkflowMutation', () => {
    it('deactivates workflow', async () => {
      mockApi.deactivateWorkflow.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useDeactivateWorkflowMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate('w1'); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useCloneWorkflowMutation', () => {
    it('clones workflow', async () => {
      mockApi.cloneWorkflow.mockResolvedValue({ id: 'clone-1' } as any);
      const { result } = renderHook(() => useCloneWorkflowMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ id: 'w1', name: 'Clone' }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useStartWorkflowMutation', () => {
    it('starts workflow', async () => {
      mockApi.startWorkflow.mockResolvedValue({ instanceId: 'i1' } as any);
      const { result } = renderHook(() => useStartWorkflowMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ workflowId: 'w1', ticketId: 1 } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useCancelInstanceMutation', () => {
    it('cancels instance', async () => {
      mockApi.cancelInstance.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useCancelInstanceMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ instanceId: 'i1', reason: 'no longer needed' }); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });

  describe('useCompleteNodeMutation', () => {
    it('completes node', async () => {
      mockApi.completeNode.mockResolvedValue(undefined as any);
      const { result } = renderHook(() => useCompleteNodeMutation(), { wrapper: createWrapper() });
      act(() => { result.current.mutate({ instanceId: 'i1', nodeId: 'n1', output: {} } as any); });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(message.success).toHaveBeenCalled();
    });
  });
});
