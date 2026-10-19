import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';

import { UserApi } from '@/lib/api/user-api';
import ProfilePage from '../page';

jest.mock('@/lib/api/user-api', () => ({
  UserApi: {
    getOwnProfile: jest.fn(),
    updateOwnProfile: jest.fn(),
    updateUser: jest.fn(),
    changePassword: jest.fn(),
    getTicketStats: jest.fn(),
  },
}));

jest.mock('@/lib/api/ticket-api', () => ({
  TicketApi: { getTicketStats: jest.fn().mockResolvedValue({}) },
}));

jest.mock('@/lib/api/notification-preference-api', () => ({
  NotificationPreferenceApi: {
    getPreferences: jest.fn().mockResolvedValue([]),
    bulkUpdate: jest.fn().mockResolvedValue(undefined),
  },
}));

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({ t: (key: string) => key, locale: 'zh-CN' }),
}));

// 会话里的用户**没有 phone**：这正是缺陷的来源——旧实现用 store 值预填编辑表单，
// 只改姓名的提交会把电话/部门抹成空串。
const storeUser = {
  id: 19,
  username: 'walkthru01',
  email: 'walkthru01@example.com',
  name: '韦小明',
  role: 'end_user',
  department: '财务部',
  permissions: ['ticket:read'],
};

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: Object.assign(
    () => ({ user: storeUser }),
    { getState: () => ({ user: storeUser, logout: jest.fn(), updateUser: jest.fn() }) },
  ),
  useAuthStoreHydration: jest.fn(),
}));

const mockedUserApi = UserApi as jest.Mocked<typeof UserApi>;

const serverProfile = {
  id: 19,
  username: 'walkthru01',
  email: 'walkthru01@example.com',
  name: '韦小明',
  role: 'end_user',
  mspRole: '',
  department: '财务部',
  departmentId: 0,
  phone: '13900001111',
  active: true,
  tenantId: 1,
  createdAt: '2026-10-20T00:00:00Z',
  updatedAt: '2026-10-20T00:00:00Z',
};

function renderPage() {
  return render(
    <App>
      <ProfilePage />
    </App>,
  );
}

beforeEach(() => {
  jest.clearAllMocks();
  mockedUserApi.getOwnProfile.mockResolvedValue(serverProfile);
  mockedUserApi.updateOwnProfile.mockResolvedValue({
    ...serverProfile,
    name: '改名成功',
  });
  mockedUserApi.updateUser.mockRejectedValue(new Error('403 权限不足'));
});

describe('/profile 自助资料面', () => {
  it('进页读服务端权威资料，电话用 /auth/me 的值预填而不是 store 的空值', async () => {
    renderPage();

    await waitFor(() => expect(mockedUserApi.getOwnProfile).toHaveBeenCalledTimes(1));

    const phone = await screen.findByLabelText('电话');
    expect(phone).toHaveValue('13900001111');
    expect(screen.getByLabelText('部门')).toHaveValue('财务部');
  });

  it('保存走 PUT /api/v1/account/profile，不再打需要 user:write 的 /users/:id', async () => {
    renderPage();
    const nameInput = await screen.findByTestId('profile-name-input');
    await waitFor(() => expect(screen.getByLabelText('电话')).toHaveValue('13900001111'));

    fireEvent.change(nameInput, { target: { value: '改名成功' } });
    fireEvent.click(screen.getByRole('button', { name: '保存修改' }));

    await waitFor(() => expect(mockedUserApi.updateOwnProfile).toHaveBeenCalledTimes(1));
    const payload = mockedUserApi.updateOwnProfile.mock.calls[0][0];
    expect(payload).toEqual({
      name: '改名成功',
      email: serverProfile.email,
      phone: '13900001111',
      department: '财务部',
    });
    expect(mockedUserApi.updateUser).not.toHaveBeenCalled();
  });

  it('保存后以响应 DTO 回填，不保留输入框里的原始值', async () => {
    mockedUserApi.updateOwnProfile.mockResolvedValue({
      ...serverProfile,
      name: '  服务端修剪后  ',
      phone: '13800000000',
    });

    renderPage();
    const nameInput = await screen.findByTestId('profile-name-input');
    await waitFor(() => expect(screen.getByLabelText('电话')).toHaveValue('13900001111'));

    fireEvent.change(nameInput, { target: { value: '用户输入的原值' } });
    fireEvent.click(screen.getByRole('button', { name: '保存修改' }));

    await waitFor(() => expect(mockedUserApi.updateOwnProfile).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.getByText(/服务端修剪后/)).toBeInTheDocument());
    // 表单值同样来自响应 DTO（服务端 trim 后的原样），不是用户键入的 '用户输入的原值'。
    expect(screen.getByTestId('profile-name-input')).toHaveValue('  服务端修剪后  ');
    expect(screen.getByLabelText('电话')).toHaveValue('13800000000');
  });

  it('保存失败时透出服务端原因，不只说「请重试」', async () => {
    mockedUserApi.updateOwnProfile.mockRejectedValue(
      new Error('该邮箱已被其他账号使用'),
    );

    renderPage();
    const nameInput = await screen.findByTestId('profile-name-input');
    await waitFor(() => expect(screen.getByLabelText('电话')).toHaveValue('13900001111'));

    fireEvent.change(nameInput, { target: { value: '撞邮箱' } });
    fireEvent.click(screen.getByRole('button', { name: '保存修改' }));

    expect(await screen.findByText(/该邮箱已被其他账号使用/)).toBeInTheDocument();
  });
});
