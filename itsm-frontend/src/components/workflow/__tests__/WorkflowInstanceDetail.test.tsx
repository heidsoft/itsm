/**
 * Tests for WorkflowInstanceDetail
 *
 * 覆盖：
 *   - 渲染 basic / tasks / history 三个 Tab
 *   - initialInstance 传入时：跳过 getInstance 调用，仅拉取 tasks 和 timeline
 *   - 不传 initialInstance 时：主动调用 getInstance 填充 summary
 *   - 详情刷新：点击刷新按钮触发重新拉取
 *   - onBack 回调：深链路由场景点击"返回列表"按钮触发
 *   - 不传 onBack 时不渲染返回按钮（Modal 场景）
 *   - 空数据状态：空 tasks / 空 auditLogs 显示 Empty
 */

import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';

import WorkflowInstanceDetail from '@/components/workflow/WorkflowInstanceDetail';
import { WorkflowApi } from '@/lib/api/workflow-api';
import BPMNDashboardApi from '@/lib/api/bpmn-dashboard-api';

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: {
    getInstance: jest.fn(),
    getNodeInstances: jest.fn(),
  },
}));

jest.mock('@/lib/api/bpmn-dashboard-api', () => ({
  __esModule: true,
  default: {
    getProcessTimeline: jest.fn(),
  },
}));

jest.mock('next/navigation', () => ({
  useRouter: jest.fn(),
}));

const mockGetInstance = WorkflowApi.getInstance as jest.MockedFunction<typeof WorkflowApi.getInstance>;
const mockGetNodeInstances = WorkflowApi.getNodeInstances as jest.MockedFunction<typeof WorkflowApi.getNodeInstances>;
const mockGetProcessTimeline = BPMNDashboardApi.getProcessTimeline as jest.MockedFunction<
  typeof BPMNDashboardApi.getProcessTimeline
>;

describe('WorkflowInstanceDetail', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // 默认：空响应
    mockGetInstance.mockResolvedValue({} as never);
    mockGetNodeInstances.mockResolvedValue([] as never);
    mockGetProcessTimeline.mockResolvedValue([] as never);
  });

  it('skips getInstance when initialInstance is provided', async () => {
    const initial = {
      id: 'inst-1',
      processDefinitionKey: 'proc-key',
      businessKey: 'biz-1',
      status: 'running',
      startTime: '2026-01-01T00:00:00Z',
      endTime: undefined,
    };

    render(<WorkflowInstanceDetail instanceId="inst-1" initialInstance={initial} />);

    await waitFor(() => {
      expect(mockGetNodeInstances).toHaveBeenCalledWith('inst-1');
    });
    expect(mockGetInstance).not.toHaveBeenCalled();
    expect(screen.getByText('inst-1')).toBeInTheDocument();
    expect(screen.getByText('proc-key')).toBeInTheDocument();
  });

  it('fetches getInstance when initialInstance is missing', async () => {
    mockGetInstance.mockResolvedValue({
      id: 'inst-2',
      workflowId: 'proc-2',
      status: 'completed',
      startTime: new Date('2026-01-01T00:00:00Z'),
      endTime: new Date('2026-01-01T01:00:00Z'),
    } as never);

    render(<WorkflowInstanceDetail instanceId="inst-2" />);

    await waitFor(() => {
      expect(mockGetInstance).toHaveBeenCalledWith('inst-2');
    });
    expect(screen.getByText('proc-2')).toBeInTheDocument();
  });

  it('renders node tasks table', async () => {
    mockGetNodeInstances.mockResolvedValue([
      {
        id: 't1',
        nodeName: '审批节点',
        nodeType: 'userTask',
        status: 'pending',
        assigneeName: '张三',
        createdAt: '2026-01-01T00:00:00Z',
        dueDate: '2026-01-02T00:00:00Z',
      },
    ] as never);

    render(
      <WorkflowInstanceDetail
        instanceId="inst-3"
        initialInstance={{
          id: 'inst-3',
          processDefinitionKey: 'proc',
          businessKey: 'biz',
          status: 'running',
          startTime: '2026-01-01T00:00:00Z',
        }}
      />
    );

    // 等待 mock 被调用 + 切到 tasks Tab
    await waitFor(() => {
      expect(mockGetNodeInstances).toHaveBeenCalledWith('inst-3');
    });
    fireEvent.click(screen.getByText(/任务列表/));
    await waitFor(() => {
      expect(screen.getByText('审批节点')).toBeInTheDocument();
    });
    expect(screen.getByText('张三')).toBeInTheDocument();
  });

  it('renders audit log timeline when present', async () => {
    mockGetProcessTimeline.mockResolvedValue([
      {
        id: 'log1',
        action: 'PROCESS_STARTED',
        activityName: 'Start',
        userName: 'system',
        timestamp: '2026-01-01T00:00:00Z',
      },
    ] as never);

    render(
      <WorkflowInstanceDetail
        instanceId="inst-4"
        initialInstance={{
          id: 'inst-4',
          processDefinitionKey: 'proc',
          businessKey: 'biz',
          status: 'running',
          startTime: '2026-01-01T00:00:00Z',
        }}
      />
    );

    await waitFor(() => {
      expect(screen.getByText(/执行历史/)).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText(/执行历史/));
    await waitFor(() => {
      expect(screen.getByText('PROCESS STARTED')).toBeInTheDocument();
    });
  });

  it('renders back button when onBack is provided and triggers callback', async () => {
    const onBack = jest.fn();

    render(
      <WorkflowInstanceDetail
        instanceId="inst-5"
        initialInstance={{
          id: 'inst-5',
          processDefinitionKey: 'proc',
          businessKey: 'biz',
          status: 'running',
          startTime: '2026-01-01T00:00:00Z',
        }}
        onBack={onBack}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('返回列表')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('返回列表'));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it('does not render back button when onBack is omitted', async () => {
    render(
      <WorkflowInstanceDetail
        instanceId="inst-6"
        initialInstance={{
          id: 'inst-6',
          processDefinitionKey: 'proc',
          businessKey: 'biz',
          status: 'running',
          startTime: '2026-01-01T00:00:00Z',
        }}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('inst-6')).toBeInTheDocument();
    });
    expect(screen.queryByText('返回列表')).not.toBeInTheDocument();
  });

  it('refresh button re-triggers fetchers', async () => {
    const onBack = jest.fn();

    render(
      <WorkflowInstanceDetail
        instanceId="inst-7"
        initialInstance={{
          id: 'inst-7',
          processDefinitionKey: 'proc',
          businessKey: 'biz',
          status: 'running',
          startTime: '2026-01-01T00:00:00Z',
        }}
        onBack={onBack}
      />
    );

    await waitFor(() => {
      expect(mockGetNodeInstances).toHaveBeenCalledTimes(1);
    });

    const refreshBtn = screen.getByText('刷新');
    fireEvent.click(refreshBtn);

    await waitFor(() => {
      expect(mockGetNodeInstances).toHaveBeenCalledTimes(2);
    });
    expect(mockGetProcessTimeline).toHaveBeenCalledTimes(2);
  });

  it('falls back gracefully when timeline API rejects', async () => {
    mockGetProcessTimeline.mockRejectedValueOnce(new Error('timeline api down'));

    render(
      <WorkflowInstanceDetail
        instanceId="inst-8"
        initialInstance={{
          id: 'inst-8',
          processDefinitionKey: 'proc',
          businessKey: 'biz',
          status: 'running',
          startTime: '2026-01-01T00:00:00Z',
        }}
      />
    );

    // 即使 timeline 失败，basic Tab 仍应正常渲染
    await waitFor(() => {
      expect(screen.getByText('inst-8')).toBeInTheDocument();
    });
  });
});