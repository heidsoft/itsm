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
    changeOwnPassword: jest.fn(),
    getTicketStats: jest.fn(),
  },
}));

// F2 回归：改密成功必须走 AuthService.logout（清 httpOnly cookie）+ 硬导航到
// /login。此前只清前端 store：后端已吊销全部 token，用户停在半登出态。
const authLogoutSpy = jest.fn();
jest.mock('@/lib/services/auth-service', () => ({
  AuthService: { logout: (() => authLogoutSpy()) as unknown as () => void },
}));

const originalLocation = window.location;
beforeAll(() => {
  // jsdom 的 location 不可写，替换成可写的普通对象
  Object.defineProperty(window, 'location', {
    value: { ...originalLocation, href: '', assign: jest.fn() },
    writable: true,
  });
});
afterAll(() => {
  Object.defineProperty(window, 'location', { value: originalLocation, writable: true });
});

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

  // F2：改密成功后端吊销全部 token（reLogin:true），前端必须完整登出并硬导航。
  // 此前只清 store 不跳转：用户看到「请重新登录」却停在原页，旧 cookie 未清，
  // 直到下次 401 续签失败才被踢走。
  it('改密成功走 AuthService.logout 清 cookie 并硬导航到 /login', async () => {
    (UserApi.changeOwnPassword as jest.Mock).mockResolvedValueOnce(undefined);
    renderPage();

    // 改密卡片在 defaultActiveKey="basic" 之外的 security Tab 里，必须先切过去，
    // 否则整个表单（含「修改」按钮）根本没挂进 DOM。
    const securityTab = await screen.findByRole('tab', { name: /安全设置/u });
    fireEvent.click(securityTab);

    // 打开改密弹窗：安全设置卡片里的「修改」按钮（带 Key 图标，无其他文案）
    // 页面还有「保存修改」等文案按钮，必须精确匹配；antd 两字按钮的可访问名
    // 会插入空格（沉淀→沉 淀），所以按 textContent 归一化后用正则匹配。
    const openBtn = await waitFor(() => {
      const btns = Array.from(document.querySelectorAll('button'));
      const target = btns.find(b => /^修\s*改$/.test((b.textContent || '').trim()));
      expect(target).toBeTruthy();
      return target as HTMLButtonElement;
    });
    fireEvent.click(openBtn);

    // antd Modal 走 portal 渲染在 body 直下；等三个密码框出现
    const inputs = await waitFor(() => {
      const all = document.querySelectorAll('input[type="password"]');
      expect(all.length).toBe(3);
      return Array.from(all) as HTMLInputElement[];
    }, { timeout: 4000 });
    fireEvent.change(inputs[0], { target: { value: 'OldPass!2026x' } });
    fireEvent.change(inputs[1], { target: { value: 'NewPass!2026x' } });
    fireEvent.change(inputs[2], { target: { value: 'NewPass!2026x' } });

    // 确认按钮也在 portal 内：从 modal 容器内找
    const okBtn = await waitFor(() => {
      const btns = Array.from(document.querySelectorAll('button'));
      const ok = btns.find(b => /确认修改/u.test(b.textContent || ''));
      expect(ok).toBeTruthy();
      return ok as HTMLButtonElement;
    });
    fireEvent.click(okBtn);

    await waitFor(() =>
      expect(UserApi.changeOwnPassword).toHaveBeenCalledWith('OldPass!2026x', 'NewPass!2026x'),
    );
    await waitFor(() => expect(authLogoutSpy).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(window.location.href).toBe('/login'));
  });
});
