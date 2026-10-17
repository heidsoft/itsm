import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { MenuAdminAPI } from '@/lib/api/menu-api';
import { RoleAPI } from '@/lib/api/role-api';
import type { MenuItem } from '@/lib/api/menu-api';
import MenusPage from '../page';

jest.mock('@/lib/api/menu-api', () => ({
  MenuAdminAPI: {
    list: jest.fn(),
    get: jest.fn(),
    create: jest.fn(),
    update: jest.fn(),
    remove: jest.fn(),
    export: jest.fn(),
    initDefaults: jest.fn(),
  },
  notifyMenusUpdated: jest.fn(),
}));

jest.mock('@/lib/api/role-api', () => ({
  RoleAPI: { getPermissions: jest.fn() },
}));

const mockedMenuAPI = MenuAdminAPI as jest.Mocked<typeof MenuAdminAPI>;
const mockedGetPermissions = RoleAPI.getPermissions as jest.Mock;

// antd Table + Form 在 jsdom 下挂载慢，默认 5s 预算不够。
jest.setTimeout(30000);

const dashboard: MenuItem = {
  id: 1,
  name: '服务台',
  path: '/dashboard',
  icon: 'LayoutDashboard',
  sortOrder: 10,
  tenantId: 1,
  isVisible: true,
  isEnabled: true,
};

const incidents: MenuItem = {
  id: 2,
  name: '事件管理',
  path: '/incidents',
  parentId: 1,
  permissionCode: 'incident:read',
  sortOrder: 30,
  tenantId: 1,
  isVisible: true,
  isEnabled: true,
};

const settings: MenuItem = {
  id: 3,
  name: '系统设置',
  path: '/admin',
  sortOrder: 90,
  tenantId: 1,
  isVisible: false,
  isEnabled: false,
};

function listResult(items: MenuItem[]) {
  return { items, total: items.length, page: 1, pageSize: items.length, totalPages: 1 };
}

describe('菜单管理页 — 列表契约', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedGetPermissions.mockResolvedValue(['incident:read']);
  });

  it('按后端 items 信封渲染整份菜单，不再读取已废弃的 menus 字段', async () => {
    // 服务台是根节点，事件管理挂在它下面，系统设置独立成根
    mockedMenuAPI.list.mockResolvedValue(listResult([dashboard, incidents, settings]));

    render(
      <App>
        <MenusPage />
      </App>
    );

    await waitFor(() => expect(mockedMenuAPI.list).toHaveBeenCalled());

    expect(await screen.findByText('服务台')).toBeInTheDocument();
    expect(screen.getByText('系统设置')).toBeInTheDocument();
    // 后端一次返回整份租户菜单，统计口径基于全量而不是当前过滤结果
    const totalCard = screen.getByText('总菜单数').closest('.ant-statistic') as HTMLElement;
    expect(within(totalCard).getByText('3')).toBeInTheDocument();
  }, 30000);

  it('搜索命中子级菜单时结果仍然可见', async () => {
    mockedMenuAPI.list.mockResolvedValue(listResult([dashboard, incidents, settings]));

    render(
      <App>
        <MenusPage />
      </App>
    );

    const search = await screen.findByPlaceholderText('搜索菜单名称或路径...');
    fireEvent.change(search, { target: { value: '事件' } });

    const row = (await screen.findByText('事件管理')).closest('tr') as HTMLElement;
    expect(within(row).getByText('/incidents')).toBeInTheDocument();
    expect(within(row).getByText('incident:read')).toBeInTheDocument();
    expect(screen.queryByText('系统设置')).not.toBeInTheDocument();
  }, 30000);

  it('加载失败时提示后端契约错误而不是留下空表', async () => {
    mockedMenuAPI.list.mockRejectedValue(new Error('menu list unavailable'));

    render(
      <App>
        <MenusPage />
      </App>
    );

    expect(await screen.findByText('加载菜单列表失败')).toBeInTheDocument();
  }, 30000);

  it('保存顶级菜单时按空值显式清除上级与权限码，而不是省略字段', async () => {
    mockedMenuAPI.list.mockResolvedValue(listResult([dashboard, incidents]));
    mockedMenuAPI.update.mockResolvedValue(dashboard);

    render(
      <App>
        <MenusPage />
      </App>
    );

    const row = (await screen.findByText('服务台')).closest('tr') as HTMLElement;
    fireEvent.click(within(row).getByRole('button', { name: '编辑' }));

    await screen.findByText('编辑菜单项');
    // antd 在两个 CJK 字符间插入空格，按钮可访问名是「保 存」
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => expect(mockedMenuAPI.update).toHaveBeenCalled());
    // PUT 是指针语义：未传保持原值，只有显式空值才能清掉上级菜单和权限码
    expect(mockedMenuAPI.update).toHaveBeenCalledWith(
      1,
      expect.objectContaining({ parentId: 0, permissionCode: '', icon: 'LayoutDashboard' }),
    );
  }, 30000);
});
