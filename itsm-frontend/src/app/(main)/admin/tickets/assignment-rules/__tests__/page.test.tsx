import { render, screen } from '@testing-library/react';
import { App } from 'antd';
import { ApiRequestError } from '@/lib/api/http-client';
import { TicketAssignmentApi } from '@/lib/api/ticket-assignment-api';
import AssignmentRulesPage from '../page';

jest.mock('@/lib/api/ticket-assignment-api', () => ({
  TicketAssignmentApi: {
    listRules: jest.fn(),
    createRule: jest.fn(),
    updateRule: jest.fn(),
    deleteRule: jest.fn(),
    testRule: jest.fn(),
  },
}));

const mockedApi = TicketAssignmentApi as jest.Mocked<typeof TicketAssignmentApi>;

function renderPage() {
  return render(
    <App>
      <AssignmentRulesPage />
    </App>,
  );
}

describe('AdminAssignmentRulesPage 加载失败语义', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  // 403 曾被渲染成「暂无分配规则」，把权限问题说成没有数据。
  it('无权限时说明权限不足，不渲染空状态', async () => {
    mockedApi.listRules.mockRejectedValue(new ApiRequestError('权限不足', 403, 2003));
    renderPage();

    expect(await screen.findByText("No permission to view allocation rules")).toBeInTheDocument();
    expect(screen.getByText(/The current role is not granted read permission for the assignment rules\. This is not "there are no rules in the system"\. Please contact the administrator to configure permissions for your role's opening rules\./)).toBeInTheDocument();
    expect(screen.queryByText("No allocation rules yet")).not.toBeInTheDocument();
    expect(document.querySelector('.ant-table')).toBeNull();
  });

  it('确实没有规则时才显示空状态', async () => {
    mockedApi.listRules.mockResolvedValue({ items: [], total: 0 });
    renderPage();

    expect(await screen.findByText("No allocation rules yet")).toBeInTheDocument();
    expect(screen.queryByText(/No permission/)).not.toBeInTheDocument();
  });
});
