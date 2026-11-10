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

// jest.setup 全局 mock 的 dayjs 没有 fromNow/extend，「创建时间」列命中断点后会渲染崩溃；
// 本页面不使用 DatePicker，这里回到真实 dayjs 并挂 relativeTime 插件。
jest.mock('dayjs', () => {
  const actual = jest.requireActual('dayjs');
  const plugin = jest.requireActual('dayjs/plugin/relativeTime');
  const factory = actual.default ?? actual;
  factory.extend(plugin.default ?? plugin);
  return actual;
});

// 全局 jest.setup 的 matchMedia 恒 false，会让 responsive:['lg'/'md'] 列（流程/负责人）
// 在 jsdom 中整列消失；本文件在首次渲染前改为命中全部断点，保证所有列可断言。
// antd responsiveObserver 在首次订阅时缓存断点结果，因此必须在文件级、任何 render 前覆盖。
window.matchMedia = jest.fn().mockImplementation(query => ({
  matches: true,
  media: query,
  onchange: null,
  addListener: jest.fn(),
  removeListener: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  dispatchEvent: jest.fn(),
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

    // 标题与菜单同名（菜单 2026-10-26 改「我的待办」后页面标题同步，避免点菜单进到另一个名字）
    expect(await screen.findByText('My To-dos')).toBeInTheDocument();
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

    const button = await screen.findByRole('button', { name: "Complete" });
    await userEvent.click(button);

    expect(await screen.findByText("Complete task")).toBeInTheDocument();
    expect(screen.getByText("Work comment (optional)")).toBeInTheDocument();
    expect(screen.getByRole('button', { name: "Confirm completion" })).toBeInTheDocument();
    expect(screen.queryByText("Approve task")).not.toBeInTheDocument();
    expect(screen.queryByText("Approval comment (optional)")).not.toBeInTheDocument();
  });

  it('显式 approval 节点仍用「批准」文案', async () => {
    mockListMyApprovalTasks.mockResolvedValue({
      items: [task({ taskPurpose: 'approval' })],
      total: 1,
      page: 1,
      size: 100,
    });

    await renderPage();

    const button = await screen.findByRole('button', { name: "Approve" });
    await userEvent.click(button);

    expect(await screen.findByText("Approve task")).toBeInTheDocument();
    expect(screen.getByText("Approval comment (optional)")).toBeInTheDocument();
    expect(screen.getByRole('button', { name: "Confirm approval" })).toBeInTheDocument();
  });

  it('流程列优先显示定义显示名，缺失时回退 Key（D10）', async () => {
    mockListMyApprovalTasks.mockResolvedValue({
      items: [
        task({
          id: 201,
          taskId: 'TASK-201',
          processDefinitionKey: 'ticket_general_flow',
          processDefinitionName: '工单通用流程',
        }),
        task({ id: 202, taskId: 'TASK-202', processDefinitionKey: 'ghost_flow' }),
      ],
      total: 2,
      page: 1,
      size: 100,
    });

    await renderPage();

    expect(await screen.findByText('工单通用流程')).toBeInTheDocument();
    // 有显示名的不再暴露工程 Key；无定义的回退展示 Key 便于排查
    expect(screen.queryByText('ticket_general_flow')).not.toBeInTheDocument();
    expect(screen.getByText('ghost_flow')).toBeInTheDocument();
  });

  it('提交工作节点时调用 decisions 接口并反馈已完成', async () => {
    mockListMyApprovalTasks.mockResolvedValue({
      items: [task({ taskPurpose: 'work' })],
      total: 1,
      page: 1,
      size: 100,
    });

    await renderPage();

    await userEvent.click(await screen.findByRole('button', { name: "Complete" }));
    await userEvent.click(await screen.findByRole('button', { name: "Confirm completion" }));

    await waitFor(() =>
      expect(mockSubmitTaskDecision).toHaveBeenCalledWith(101, { action: 'approve', comment: '' })
    );
    expect(await screen.findByText("Completed")).toBeInTheDocument();
    // 审批中心页面较重：提交后触发任务与遗留单两路全量重渲染，jsdom 实测 ~11-12s，
    // 默认 10s testTimeout 贴线必红（非用例逻辑问题）
  }, 20000);
});
