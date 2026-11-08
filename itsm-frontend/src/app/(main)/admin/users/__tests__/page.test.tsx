import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { UserApi } from '@/lib/api/user-api';
import { RoleAPI } from '@/lib/api/role-api';
import UserManagementPage from '../page';

jest.mock('@/lib/api/user-api', () => ({
  UserApi: {
    getUsers: jest.fn(),
    getUserStats: jest.fn(),
    createUser: jest.fn(),
    updateUser: jest.fn(),
    deleteUser: jest.fn(),
    changeUserStatus: jest.fn(),
    resetPassword: jest.fn(),
  },
  PRIMARY_ROLE_OPTIONS: [],
  resolveRoleTags: () => [],
}));

jest.mock('@/lib/api/role-api', () => ({
  RoleAPI: {
    getRoles: jest.fn(),
  },
}));

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: () => ({ currentTenant: { id: 1, name: '默认租户' } }),
  useAuthStoreHydration: () => undefined,
}));

const mockedUserApi = UserApi as jest.Mocked<typeof UserApi>;
const mockedRoleApi = RoleAPI as jest.Mocked<typeof RoleAPI>;

function renderPage() {
  return render(
    <App>
      <UserManagementPage />
    </App>,
  );
}

// antd 的 Spin 容器始终存在（idle 也有 .ant-spin），只有 spinning 态才代表真在加载。
function isSpinning(): boolean {
  return document.querySelector('.ant-spin-spinning') !== null;
}

// antd Table 在 loading 时也会渲染 .ant-table-placeholder（被 spin 遮罩盖住），
// 且其内部同样使用 .ant-empty 结构，因此空表占位存在与否不是「假空态」的可靠信号。
// 页面自带的 Empty 块是 .ant-card-body 的直接子节点，且带「创建第一个用户」按钮。
function hasStandaloneEmptyBlock(): boolean {
  return document.querySelector('.ant-card-body > .ant-empty') !== null;
}

describe('UserManagement 首帧加载语义', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedRoleApi.getRoles.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 100,
      totalPages: 0,
    });
    mockedUserApi.getUserStats.mockResolvedValue({ total: 0, active: 0, inactive: 0 });
  });

  // 修复前：listLoading 初值 false，首帧把「还没拉取」画成「暂无用户数据」Empty 块。
  it('首次拉取未完成时不渲染 Empty 空态块', async () => {
    mockedUserApi.getUsers.mockImplementation(
      () =>
        new Promise(() => {
          // 保持 pending：列表处于 loading 中
        }),
    );

    renderPage();

    expect(hasStandaloneEmptyBlock()).toBe(false);
    expect(screen.queryByText('暂无用户数据')).not.toBeInTheDocument();
    // 统计卡同样不能先把「还没拉到」画成「总用户数 0」
    expect(screen.queryByText('总用户数')).not.toBeInTheDocument();
    expect(screen.queryByText('活跃用户')).not.toBeInTheDocument();
    expect(screen.queryByText('禁用用户')).not.toBeInTheDocument();
  });

  // 修复前：loadUsers 未完成时 loading=false，页面无任何加载指示，用户以为已加载完。
  it('首次拉取未完成时表格处于 spinning 态', async () => {
    mockedUserApi.getUsers.mockImplementation(
      () =>
        new Promise(() => {
          // 保持 pending：列表处于 loading 中
        }),
    );

    renderPage();

    expect(isSpinning()).toBe(true);
  });

  // 加载确实完成后，才允许出现真正的空态。
  it('拉取完成且确实没有用户时才显示 Empty 空态', async () => {
    mockedUserApi.getUsers.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
      totalPages: 0,
    });

    renderPage();

    await waitFor(() => expect(mockedUserApi.getUsers).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(hasStandaloneEmptyBlock()).toBe(true));
    expect(screen.getByText('暂无用户数据')).toBeInTheDocument();
  });

  // 修复前：列表 loading 与提交按钮共用一个状态，列表转圈时提交按钮也跟着转。
  it('列表加载期间提交按钮不进入 loading 态', async () => {
    mockedUserApi.getUsers.mockImplementation(
      () =>
        new Promise(() => {
          // 保持 pending：列表处于 loading 中
        }),
    );

    renderPage();

    expect(isSpinning()).toBe(true);
    const submitButtons = document.querySelectorAll('button.ant-btn-primary');
    submitButtons.forEach(btn => {
      expect(btn.classList.contains('ant-btn-loading')).toBe(false);
    });
  });
});
