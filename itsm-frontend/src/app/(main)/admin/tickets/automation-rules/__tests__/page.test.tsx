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

    expect(await screen.findByText("No permission to view automation rules")).toBeInTheDocument();
    expect(screen.getByText(/The current role is not granted read permission for automation rules\. This is not "there are no rules in the system"\. Please contact the administrator to configure permissions for your role's opening rules\./)).toBeInTheDocument();
    expect(screen.queryByText("There is no automation rule yet, click the button above to create it")).not.toBeInTheDocument();
    expect(document.querySelector('.ant-table')).toBeNull();
  });

  it('确实没有规则时才显示空状态', async () => {
    mockedApi.listRules.mockResolvedValue({ items: [], total: 0 });
    render(
      <App>
        <AutomationRulesPage />
      </App>,
    );

    expect(await screen.findByText("There is no automation rule yet, click the button above to create it")).toBeInTheDocument();
    expect(screen.queryByText(/No permission/)).not.toBeInTheDocument();
  });
});
