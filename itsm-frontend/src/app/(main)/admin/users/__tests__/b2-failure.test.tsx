import { fireEvent, render, screen, waitFor } from '@testing-library/react';
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

beforeEach(() => {
  mockedRoleApi.getRoles.mockResolvedValue({ items: [] } as never);
  mockedUserApi.getUserStats.mockResolvedValue({ total: 0, active: 0, inactive: 0 });
});
it('列表失败显示可重试错误，重试成功后才显示真空态', async () => {
  mockedUserApi.getUsers.mockRejectedValueOnce(new Error('offline')).mockResolvedValue({ items: [], total: 0 } as never);
  renderPage();
  const retry = await screen.findByRole('button', { name: /重\s*试/ });
  expect(screen.queryByText('暂无用户数据')).not.toBeInTheDocument();
  fireEvent.click(retry);
  expect(await screen.findByText('暂无用户数据')).toBeInTheDocument();
});
