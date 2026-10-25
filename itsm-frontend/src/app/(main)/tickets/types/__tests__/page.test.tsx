import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import { ApiRequestError } from '@/lib/api/http-client';
import { TicketTypeApi } from '@/lib/api/ticketTypeApi';
import { TicketTypeStatus, type TicketTypeDefinition, type TicketTypeListResponse, type TicketTypePresetDefinition } from '@/types/ticket-type';
import TicketTypesPage from '../page';

jest.mock('@/lib/api/ticketTypeApi', () => ({
  TicketTypeApi: {
    list: jest.fn(),
    create: jest.fn(),
    update: jest.fn(),
    setEnabled: jest.fn(),
    restore: jest.fn(),
    clone: jest.fn(),
    listPresets: jest.fn(),
    installPreset: jest.fn(),
  },
}));

// 表单弹窗会拉流程定义与 SLA 目录，本页用例只关心按钮门禁与 403 文案。
jest.mock('@/components/business/TicketTypeFormModal', () => ({
  TicketTypeFormModal: () => null,
}));

// 页面按后端同名权限码决定按钮是否渲染，测试里显式给出角色权限集。
let permissionCodes: string[] = [];

jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({
    hasPermission: (resource: string, action: string) =>
      permissionCodes.includes(`${resource}:${action}`),
  }),
}));

const mockedApi = TicketTypeApi as jest.Mocked<typeof TicketTypeApi>;

const TYPE: TicketTypeDefinition = {
  id: 12,
  code: 'general',
  name: '其他工单',
  description: '通用工单类型',
  status: TicketTypeStatus.ACTIVE,
  defaultPriority: 'medium',
  sortOrder: 0,
  customFields: [],
  approvalEnabled: false,
  slaEnabled: false,
  autoAssignEnabled: false,
  createdBy: 1,
  createdByName: '管理员',
  createdAt: '2026-10-01T00:00:00Z',
  updatedAt: '2026-10-01T00:00:00Z',
  tenantId: 1,
};

const PRESET: TicketTypePresetDefinition = {
  id: 'general-incident',
  name: '通用事件',
  description: '通用故障与中断上报',
  category: '运维',
  definition: { code: 'general_incident', name: '通用事件' } as TicketTypePresetDefinition['definition'],
};

const LIST_BODY: TicketTypeListResponse = {
  items: [TYPE],
  total: 1,
  page: 1,
  pageSize: 100,
  totalPages: 1,
};

function renderPage() {
  return render(
    <App>
      <TicketTypesPage />
    </App>,
  );
}

const denied = () => new ApiRequestError('权限不足', 403, 2003);

describe('TicketTypesPage 权限门禁', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedApi.list.mockResolvedValue(LIST_BODY);
    mockedApi.listPresets.mockResolvedValue([PRESET]);
    permissionCodes = ['ticket:read', 'ticket_type:read'];
  });

  // 坐席只有 ticket_type:read：整页应降为只读，而不是把 6 个写按钮都点成 403。
  it('只读角色不渲染写操作，并说明需要哪个权限码', async () => {
    renderPage();

    expect(await screen.findByText('只读')).toBeInTheDocument();
    expect(screen.getByText(/新建、编辑、复制、启停和恢复需要 ticket_type:manage/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '新建类型' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '编辑' })).not.toBeInTheDocument();
  });

  // 预设库降为 ticket_type:read 后，坐席可以浏览内容，但安装按钮不出现。
  it('只读角色能打开预设库，但看不到安装按钮', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole('button', { name: '从预设安装' }));

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('通用事件')).toBeInTheDocument();
    expect(mockedApi.listPresets).toHaveBeenCalledTimes(1);
    expect(within(dialog).queryByRole('button', { name: '安装' })).not.toBeInTheDocument();
    expect(within(dialog).getByText(/安装需要 ticket_type:install_preset/)).toBeInTheDocument();
  });

  // it_admin 口径：manage + install_preset 都补齐后，写按钮与安装按钮恢复。
  it('配置角色渲染写操作并可以安装预设', async () => {
    permissionCodes = ['ticket:read', 'ticket_type:read', 'ticket_type:manage', 'ticket_type:install_preset'];
    mockedApi.installPreset.mockResolvedValue(TYPE);
    const user = userEvent.setup();
    renderPage();

    expect(screen.getByRole('button', { name: /新建类型/ })).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: '编辑' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '从预设安装' }));
    const installButton = await screen.findByRole('button', { name: '安装' });
    await user.click(installButton);
    await waitFor(() => expect(mockedApi.installPreset).toHaveBeenCalledWith('general-incident'));
    // 安装成功后关弹窗并回读列表，使用者能在表格里看到新类型
    await waitFor(() => expect(mockedApi.list).toHaveBeenCalledTimes(2));
  }, 30000);

  // 提交期间的重复点击不能发第二发请求。
  it('安装请求进行中禁用按钮', async () => {
    permissionCodes = ['ticket:read', 'ticket_type:read', 'ticket_type:manage', 'ticket_type:install_preset'];
    let resolveInstall: (value: TicketTypeDefinition) => void = () => {};
    mockedApi.installPreset.mockImplementation(
      () => new Promise<TicketTypeDefinition>((resolve) => { resolveInstall = resolve; }),
    );
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole('button', { name: '从预设安装' }));
    const installButton = await screen.findByRole('button', { name: '安装' });
    await user.click(installButton);
    expect(mockedApi.installPreset).toHaveBeenCalledTimes(1);
    expect(installButton).toBeDisabled();
    await user.click(installButton).catch(() => undefined);
    expect(mockedApi.installPreset).toHaveBeenCalledTimes(1);

    resolveInstall(TYPE);
    await waitFor(() => expect(mockedApi.list).toHaveBeenCalledTimes(2));
  }, 30000);

  // 403 仍然要点名权限码：后端 message 只有「权限不足」。
  it('预设列表 403 时提示需要的读取权限码', async () => {
    permissionCodes = [];
    mockedApi.listPresets.mockRejectedValue(denied());
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole('button', { name: '从预设安装' }));

    expect(await screen.findByText(/权限不足。该操作需要 ticket_type:read/)).toBeInTheDocument();
    expect(screen.queryByText('通用事件')).not.toBeInTheDocument();
  });
});
