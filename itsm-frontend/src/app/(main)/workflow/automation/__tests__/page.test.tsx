import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import { ApiRequestError } from '@/lib/api/http-client';
import { TicketAutomationRuleApi } from '@/lib/api/ticket-automation-rule-api';
import type { AutomationRule } from '@/lib/api/ticket-automation-rule-api';
import WorkflowAutomationPage from '../page';

jest.mock('@/lib/api/ticket-automation-rule-api', () => ({
  TicketAutomationRuleApi: {
    listRules: jest.fn(),
    createRule: jest.fn(),
    updateRule: jest.fn(),
    deleteRule: jest.fn(),
    testRule: jest.fn(),
  },
}));

const mockedApi = TicketAutomationRuleApi as jest.Mocked<typeof TicketAutomationRuleApi>;

const RULE: AutomationRule = {
  id: 12,
  name: 'P1 自动派值班',
  priority: 3,
  conditions: [],
  actions: [],
  isActive: true,
  executionCount: 4,
  createdBy: 1,
  tenantId: 1,
  createdAt: '2026-10-01T00:00:00Z',
  updatedAt: '2026-10-01T00:00:00Z',
  type: 'assignment',
};

function renderPage() {
  return render(
    <App>
      <WorkflowAutomationPage />
    </App>,
  );
}

// antd 表格空占位；测试环境没有 ConfigProvider，不能依赖「暂无数据」文案。
function hasEmptyTable(): boolean {
  return document.querySelector('.ant-table-placeholder') !== null;
}

// antd 的 Spin 容器始终存在（idle 也有 .ant-spin），只有 spinning 态才代表真在加载。
function isSpinning(): boolean {
  return document.querySelector('.ant-spin-spinning') !== null;
}

describe('WorkflowAutomationPage 加载失败语义', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  // 403 曾被渲染成四个 0 统计卡 + 表格「暂无数据」，用户以为系统里没有规则。
  it('无权限时说明权限不足，不渲染 0 统计卡与空表格', async () => {
    mockedApi.listRules.mockRejectedValue(new ApiRequestError('权限不足', 403, 2003));
    renderPage();

    expect(await screen.findByText('没有权限查看自动化规则')).toBeInTheDocument();
    expect(screen.getByText(/这里不是「系统里没有规则」/)).toBeInTheDocument();
    expect(screen.queryByText('自动分配规则')).not.toBeInTheDocument();
    expect(hasEmptyTable()).toBe(false);
    expect(screen.queryByText('新建规则')).not.toBeInTheDocument();
  });

  it('服务端错误时给出原因并提供重试', async () => {
    mockedApi.listRules.mockRejectedValue(new ApiRequestError('HTTP error! status: 500', 500));
    renderPage();

    expect(await screen.findByText('加载规则失败')).toBeInTheDocument();
    expect(screen.getByText('HTTP error! status: 500')).toBeInTheDocument();
    expect(screen.queryByText('自动升级规则')).not.toBeInTheDocument();

    mockedApi.listRules.mockResolvedValue({ items: [RULE], total: 1 });
    await userEvent.click(screen.getByRole('button', { name: '重试' }));

    await waitFor(() => expect(mockedApi.listRules).toHaveBeenCalledTimes(2));
    expect(await screen.findByText('P1 自动派值班')).toBeInTheDocument();
    expect(screen.queryByText('加载规则失败')).not.toBeInTheDocument();
  });

  it('加载成功且确实没有规则时才显示统计卡与空表', async () => {
    mockedApi.listRules.mockResolvedValue({ items: [], total: 0 });
    renderPage();

    await waitFor(() => expect(mockedApi.listRules).toHaveBeenCalledTimes(1));
    expect(screen.getByText('自动分配规则')).toBeInTheDocument();
    expect(hasEmptyTable()).toBe(true);
    expect(screen.queryByText(/权限/)).not.toBeInTheDocument();
  });
});

describe('WorkflowAutomationPage 首帧加载语义', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  // 修复前：listLoading 初值 false，首帧先渲染「0 条规则」统计卡与空表，再被真实数据覆盖。
  it('首次拉取未完成时显示 loading，不渲染 0 统计卡', async () => {
    mockedApi.listRules.mockImplementation(
      () =>
        new Promise(() => {
          // 保持 pending：列表处于 loading 中
        }),
    );

    renderPage();

    // 统计卡不在 Table 的 spin 容器内，是「假 0」的可靠信号
    expect(screen.queryByText('自动分配规则')).not.toBeInTheDocument();
    expect(screen.queryByText('智能路由规则')).not.toBeInTheDocument();
    expect(screen.queryByText('自动升级规则')).not.toBeInTheDocument();
    expect(screen.queryByText('活跃规则')).not.toBeInTheDocument();
  });

  // 修复前：loadRules 未完成时 loading=false，页面无任何加载指示，用户以为已加载完。
  it('首次拉取未完成时表格处于 spinning 态', async () => {
    mockedApi.listRules.mockImplementation(
      () =>
        new Promise(() => {
          // 保持 pending：列表处于 loading 中
        }),
    );

    renderPage();

    expect(isSpinning()).toBe(true);
  });
});
