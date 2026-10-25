import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 审批中心（/approvals）文案锁。
 *
 * 两条回归：
 * - English 模式下面包屑/表头/弹窗全部走 i18n，页面主体不得残留中文；
 * - 工作节点（taskPurpose 未声明或为 work）用「完成」文案，只有显式 approval 才用「批准」，
 *   对应引擎侧 normalizeTaskPurpose 把空值按 work 处理的语义（P0-1 D2）。
 */

let language: 'zh-CN' | 'en-US' = 'zh-CN';

jest.mock('@/lib/user-preferences', () => ({
  userPreferences: {
    get: () => ({ language }),
    subscribe: () => () => {},
  },
}));

const mockListMyApprovalTasks = jest.fn();
const mockSubmitTaskDecision = jest.fn();

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: {
    listMyApprovalTasks: (...args: unknown[]) => mockListMyApprovalTasks(...args),
    claimMyTask: jest.fn(),
    submitTaskDecision: (...args: unknown[]) => mockSubmitTaskDecision(...args),
  },
}));

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: () => Promise.resolve({ items: [] }),
  },
}));

jest.mock('@/lib/api/user-api', () => ({
  UserApi: {
    getUsers: () => Promise.resolve({ items: [] }),
  },
}));

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: () => ({ user: { id: 1, username: 'alice', name: 'Alice' } }),
}));

import ApprovalsCenterPage from '../page';

function task(overrides: Record<string, unknown>) {
  return {
    id: 101,
    taskId: 'TASK-101',
    taskDefinitionKey: 'review',
    taskName: 'Technical Review',
    taskType: 'user_task',
    status: 'assigned',
    assignee: 'alice',
    assigneeName: 'Alice',
    processInstanceId: 7,
    processInstanceKey: 'PI-7',
    processDefinitionKey: 'change-review',
    createdTime: '2026-10-20T08:00:00Z',
    ...overrides,
  };
}

async function renderPage() {
  // 页面通过 App.useApp() 取 message 实例，必须挂在 antd App 上下文下渲染
  const utils = render(
    <App>
      <ApprovalsCenterPage />
    </App>
  );
  await waitFor(() => expect(mockListMyApprovalTasks).toHaveBeenCalled());
  return utils;
}

describe('审批中心文案', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    language = 'zh-CN';
    mockSubmitTaskDecision.mockResolvedValue(undefined);
  });

  it('English 模式下表头与操作使用英文，页面主体不残留中文', async () => {
    language = 'en-US';
    mockListMyApprovalTasks.mockResolvedValue({
      items: [task({ taskPurpose: 'approval' })],
      total: 1,
      page: 1,
      size: 100,
    });

    await renderPage();

    expect(await screen.findByText('Approval Center')).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Status' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Approve' })).toBeInTheDocument();

    // 未命中的 key 会被 useI18n 原样返回，中文 locale 串也不应出现在英文界面
    const body = document.body.textContent || '';
    expect(body).not.toMatch(/approvalCenter\./);
    expect(body).not.toMatch(/[一-龥]/);
  });

  it('工作节点用「完成」文案，弹窗标题与确认按钮不得写成批准', async () => {
    mockListMyApprovalTasks.mockResolvedValue({
      items: [task({ taskPurpose: '' })],
      total: 1,
      page: 1,
      size: 100,
    });

    await renderPage();

    const button = await screen.findByRole('button', { name: '完成' });
    await userEvent.click(button);

    expect(await screen.findByText('完成任务')).toBeInTheDocument();
    expect(screen.getByText('处理意见（可选）')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '确认完成' })).toBeInTheDocument();
    expect(screen.queryByText('批准任务')).not.toBeInTheDocument();
    expect(screen.queryByText('审批意见（可选）')).not.toBeInTheDocument();
  });

  it('显式 approval 节点仍用「批准」文案', async () => {
    mockListMyApprovalTasks.mockResolvedValue({
      items: [task({ taskPurpose: 'approval' })],
      total: 1,
      page: 1,
      size: 100,
    });

    await renderPage();

    const button = await screen.findByRole('button', { name: '批准' });
    await userEvent.click(button);

    expect(await screen.findByText('批准任务')).toBeInTheDocument();
    expect(screen.getByText('审批意见（可选）')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '确认批准' })).toBeInTheDocument();
  });

  it('提交工作节点时调用 decisions 接口并反馈已完成', async () => {
    mockListMyApprovalTasks.mockResolvedValue({
      items: [task({ taskPurpose: 'work' })],
      total: 1,
      page: 1,
      size: 100,
    });

    await renderPage();

    await userEvent.click(await screen.findByRole('button', { name: '完成' }));
    await userEvent.click(await screen.findByRole('button', { name: '确认完成' }));

    await waitFor(() =>
      expect(mockSubmitTaskDecision).toHaveBeenCalledWith(101, { action: 'approve', comment: '' })
    );
    expect(await screen.findByText('已完成')).toBeInTheDocument();
    // 审批中心页面较重：提交后触发任务与遗留单两路全量重渲染，jsdom 实测 ~11-12s，
    // 默认 10s testTimeout 贴线必红（非用例逻辑问题）
  }, 20000);
});
