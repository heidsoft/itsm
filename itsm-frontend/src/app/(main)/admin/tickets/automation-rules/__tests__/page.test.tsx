import { render, screen } from '@testing-library/react';
import { App } from 'antd';
import { ApiRequestError } from '@/lib/api/http-client';
import { TicketAutomationRuleApi } from '@/lib/api/ticket-automation-rule-api';
import AutomationRulesPage from '../page';

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

describe('AdminAutomationRulesPage 加载失败语义', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  // 403 曾被渲染成「暂无自动化规则，点击上方按钮创建」，把权限问题说成没有数据。
  it('无权限时说明权限不足，不渲染空状态', async () => {
    mockedApi.listRules.mockRejectedValue(new ApiRequestError('权限不足', 403, 2003));
    render(
      <App>
        <AutomationRulesPage />
      </App>,
    );

    expect(await screen.findByText('没有权限查看自动化规则')).toBeInTheDocument();
    expect(screen.getByText(/这里不是「系统里没有规则」/)).toBeInTheDocument();
    expect(screen.queryByText('暂无自动化规则，点击上方按钮创建')).not.toBeInTheDocument();
    expect(document.querySelector('.ant-table')).toBeNull();
  });

  it('确实没有规则时才显示空状态', async () => {
    mockedApi.listRules.mockResolvedValue({ items: [], total: 0 });
    render(
      <App>
        <AutomationRulesPage />
      </App>,
    );

    expect(await screen.findByText('暂无自动化规则，点击上方按钮创建')).toBeInTheDocument();
    expect(screen.queryByText(/没有权限/)).not.toBeInTheDocument();
  });
});
