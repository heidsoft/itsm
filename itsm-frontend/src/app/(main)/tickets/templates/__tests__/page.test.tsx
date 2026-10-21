import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import { ApiRequestError } from '@/lib/api/http-client';
import { TicketApi } from '@/lib/api/ticket-api';
import TicketTemplatesPage from '../page';

jest.mock('@/lib/api/ticket-api', () => ({
  TicketApi: {
    getTemplates: jest.fn(),
    createTemplate: jest.fn(),
    updateTemplate: jest.fn(),
    deleteTemplate: jest.fn(),
    copyTemplate: jest.fn(),
  },
}));

// 页面按后端同名权限码决定按钮是否渲染，测试里显式给出角色权限集。
let permissionCodes: string[] = ['ticket_template:read', 'ticket_template:delete'];

jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({
    hasPermission: (resource: string, action: string) =>
      permissionCodes.includes(`${resource}:${action}`),
  }),
}));

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), replace: jest.fn(), back: jest.fn() }),
}));

const mockedApi = TicketApi as jest.Mocked<typeof TicketApi>;

const TEMPLATE = {
  id: 7,
  name: '网络故障模板',
  description: '用于网络类工单',
  category: 'IT Support',
  priority: 'high',
  formFields: { type: 'incident', sla: '2 hours', slaType: 'hours' },
  isActive: true,
  createdAt: '2026-10-01T00:00:00Z',
  updatedAt: '2026-10-01T00:00:00Z',
};

function renderPage() {
  return render(
    <App>
      <TicketTemplatesPage />
    </App>,
  );
}

describe('TicketTemplatesPage 加载失败语义', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    permissionCodes = ['ticket_template:read', 'ticket_template:delete'];
  });

  // 403 曾被渲染成三个 0 统计卡 + 「暂无模板」，使用者会以为系统里没有模板。
  it('无权限时说明权限不足，不渲染 0 统计卡与空状态', async () => {
    mockedApi.getTemplates.mockRejectedValue(new ApiRequestError('权限不足', 403, 2003));
    renderPage();

    expect(await screen.findByText('没有权限查看工单模板')).toBeInTheDocument();
    expect(screen.getByText(/这里不是「系统里没有模板」/)).toBeInTheDocument();
    expect(screen.queryByText('暂无模板')).not.toBeInTheDocument();
    expect(screen.queryByText('模板总数')).not.toBeInTheDocument();
    expect(screen.queryByText('模板管理')).not.toBeInTheDocument();
  });

  it('服务端错误时给出原因，重试成功后恢复列表', async () => {
    mockedApi.getTemplates.mockRejectedValue(
      new ApiRequestError('HTTP error! status: 500', 500, 5001),
    );
    renderPage();

    expect(await screen.findByText('模板列表加载失败')).toBeInTheDocument();
    expect(screen.getByText('HTTP error! status: 500')).toBeInTheDocument();

    mockedApi.getTemplates.mockResolvedValue({ items: [TEMPLATE], total: 1 });
    await userEvent.click(screen.getByRole('button', { name: '重试' }));

    await waitFor(() => expect(mockedApi.getTemplates).toHaveBeenCalledTimes(2));
    expect(await screen.findByText('网络故障模板')).toBeInTheDocument();
    expect(screen.queryByText('模板列表加载失败')).not.toBeInTheDocument();
  });

  it('加载成功且确实没有模板时才显示统计卡与空状态', async () => {
    mockedApi.getTemplates.mockResolvedValue({ items: [], total: 0 });
    renderPage();

    await waitFor(() => expect(mockedApi.getTemplates).toHaveBeenCalledTimes(1));
    expect(screen.getByText('模板总数')).toBeInTheDocument();
    expect(screen.getByText('暂无模板')).toBeInTheDocument();
    expect(screen.queryByText(/没有权限/)).not.toBeInTheDocument();
  });

  // 权限不足时仍显示可用按钮，点下去只会拿到 403，属于假可操作状态。
  it('写按钮按权限码渲染，列表仍可读', async () => {
    permissionCodes = ['ticket_template:read'];
    mockedApi.getTemplates.mockResolvedValue({ items: [TEMPLATE], total: 1 });
    renderPage();

    expect(await screen.findByText('网络故障模板')).toBeInTheDocument();
    expect(screen.queryByLabelText('新建模板')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('编辑模板')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('复制模板')).not.toBeInTheDocument();
    expect(screen.getByLabelText('查看模板详情')).toBeInTheDocument();
  });

  it('有写权限时按钮齐全，删除仍受 delete 权限控制', async () => {
    permissionCodes = [
      'ticket_template:read',
      'ticket_template:create',
      'ticket_template:update',
    ];
    mockedApi.getTemplates.mockResolvedValue({ items: [TEMPLATE], total: 1 });
    renderPage();

    expect(await screen.findByLabelText('新建模板')).toBeInTheDocument();
    expect(screen.getByLabelText('编辑模板')).toBeInTheDocument();
    expect(screen.getByLabelText('复制模板')).toBeInTheDocument();
    expect(screen.queryByLabelText('删除模板')).not.toBeInTheDocument();
  });
});
