/**
 * 工作流实例列表的交互回归锁。
 *
 * 锁两件事：
 *   1) 搜索框防抖：连续输入不应每个字符都打出「实例 + 统计」两次请求；
 *   2) 乱序响应丢弃：先发后到的慢响应不能覆盖新关键字的查询结果。
 */

import React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';

import WorkflowInstancesPage from '../page';
import { WorkflowApi } from '@/lib/api/workflow-api';

jest.mock('next/navigation', () => ({
  useSearchParams: () => new URLSearchParams(),
}));

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: {
    getInstances: jest.fn(),
    getInstanceStats: jest.fn(),
    suspendWorkflow: jest.fn(),
    terminateWorkflow: jest.fn(),
    resumeWorkflow: jest.fn(),
    startWorkflow: jest.fn(),
    getWorkflows: jest.fn(),
  },
}));

jest.mock('@/components/workflow/WorkflowInstanceDetail', () => ({
  WorkflowInstanceDetail: () => <div data-testid="instance-detail" />,
  __esModule: true,
  default: () => <div data-testid="instance-detail" />,
}));

jest.setTimeout(30000);

const mockGetInstances = WorkflowApi.getInstances as jest.MockedFunction<typeof WorkflowApi.getInstances>;
const mockGetInstanceStats = WorkflowApi.getInstanceStats as jest.MockedFunction<
  typeof WorkflowApi.getInstanceStats
>;

const emptyStats = { total: 0, running: 0, completed: 0, suspended: 0, terminated: 0 };

const renderPage = () =>
  render(
    <App>
      <WorkflowInstancesPage />
    </App>
  );

describe('WorkflowInstancesPage 交互', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetInstances.mockResolvedValue({ instances: [], total: 0 });
    mockGetInstanceStats.mockResolvedValue(emptyStats);
  });

  it('连续输入关键字只在停顿后查询一次（请求防抖）', async () => {
    jest.useFakeTimers();
    try {
      renderPage();

      // 首屏查询一次
      await act(async () => {
        await Promise.resolve();
      });
      expect(mockGetInstances).toHaveBeenCalledTimes(1);

      const input = screen.getByPlaceholderText('按流程 Key 搜索');
      fireEvent.change(input, { target: { value: 'inc' } });
      fireEvent.change(input, { target: { value: 'incident' } });
      fireEvent.change(input, { target: { value: 'incident_flow' } });

      // 停顿时间未到：不应为每一次按键打请求
      expect(mockGetInstances).toHaveBeenCalledTimes(1);

      await act(async () => {
        jest.advanceTimersByTime(300);
        await Promise.resolve();
      });

      expect(mockGetInstances).toHaveBeenCalledTimes(2);
      expect(mockGetInstances).toHaveBeenLastCalledWith(
        expect.objectContaining({ workflowId: 'incident_flow' })
      );
      expect(mockGetInstanceStats).toHaveBeenCalledTimes(2);
    } finally {
      jest.useRealTimers();
    }
  });

  it('先发后到的慢响应不覆盖新关键字的结果（乱序守卫）', async () => {
    let resolveFirst!: (value: { instances: []; total: number }) => void;
    const first = new Promise<{ instances: []; total: number }>(resolve => {
      resolveFirst = resolve;
    });
    mockGetInstances.mockReturnValueOnce(first).mockResolvedValueOnce({ instances: [], total: 0 });

    renderPage();

    await waitFor(() => expect(mockGetInstances).toHaveBeenCalledTimes(1));

    const input = screen.getByPlaceholderText('按流程 Key 搜索');
    fireEvent.change(input, { target: { value: 'new-key' } });

    await waitFor(() => expect(mockGetInstances).toHaveBeenCalledTimes(2));

    // 第二个请求先完成，随后第一个（旧关键字）的慢响应才到。
    resolveFirst({ instances: [], total: 0 });
    await act(async () => {
      await Promise.resolve();
    });

    expect(mockGetInstances).toHaveBeenLastCalledWith(
      expect.objectContaining({ workflowId: 'new-key' })
    );
  });
});
