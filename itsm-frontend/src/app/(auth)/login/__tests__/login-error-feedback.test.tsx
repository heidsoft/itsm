import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import LoginPage from '../page';

/**
 * 登录失败反馈的行为锁（缺陷 #79）。
 *
 * 背景：登录失败此前只向页面返回 false，Alert 又把「登录失败」同时作为标题和
 * 正文渲染，用户看到的是「登录失败 / 登录失败」，后端已经返回的
 * `{code:2001, message:"用户名或密码错误"}` 与限流 retryAfterSeconds 全部丢失。
 *
 * 只 mock 网络边界与副作用依赖（AuthService / router / logger），antd 用真组件渲染。
 */
const mockLogin = jest.fn();
const mockListSSOProviders = jest.fn();

jest.mock('@/lib/services/auth-service', () => ({
  AuthService: {
    login: (...args: unknown[]) => mockLogin(...args),
  },
}));

jest.mock('@/lib/api/auth-api', () => ({
  AuthAPI: {
    listSSOProviders: (...args: unknown[]) => mockListSSOProviders(...args),
    initiateSSOLogin: jest.fn(),
  },
}));

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStoreHydration: jest.fn(),
}));

const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  useSearchParams: () => new URLSearchParams(),
}));

jest.mock('@/lib/env', () => ({
  logger: { info: jest.fn(), warn: jest.fn(), error: jest.fn() },
}));

jest.mock('@/lib/i18n/useI18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string | number>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}));

async function submit(username: string, password: string) {
  const usernameInput = document.getElementById('username') as HTMLInputElement;
  const passwordInput = document.getElementById('password') as HTMLInputElement;
  await userEvent.clear(usernameInput);
  await userEvent.clear(passwordInput);
  await userEvent.type(usernameInput, username);
  await userEvent.type(passwordInput, password);
  await userEvent.click(screen.getByRole('button', { name: /auth\.login\.loginButton/ }));
}

describe('登录失败反馈', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockListSSOProviders.mockResolvedValue([]);
  });

  it('透出后端失败原因，不再显示重复的通用「登录失败」', async () => {
    mockLogin.mockResolvedValueOnce({ ok: false, message: '用户名或密码错误' });

    render(<LoginPage />);
    await submit('admin', 'wrongpass');

    expect(await screen.findByText('用户名或密码错误')).toBeInTheDocument();
    expect(screen.queryByText('auth.login.loginFailedUnknown')).not.toBeInTheDocument();
  });

  it('失败后无需刷新即可再次提交，第二次提交仍然发起登录请求', async () => {
    mockLogin.mockResolvedValueOnce({ ok: false, message: '用户名或密码错误' });
    mockLogin.mockResolvedValueOnce({ ok: true });

    render(<LoginPage />);
    await submit('admin', 'wrongpass');
    await screen.findByText('用户名或密码错误');

    await submit('admin', 'rightpass');

    await waitFor(() => expect(mockLogin).toHaveBeenCalledTimes(2));
    expect(mockPush).toHaveBeenCalledWith('/dashboard');
  });

  it('限流失败展示倒计时并保留后端原因', async () => {
    mockLogin.mockResolvedValueOnce({
      ok: false,
      message: '登录请求过于频繁，请稍后再试',
      retryAfterSeconds: 30,
    });

    render(<LoginPage />);
    await submit('admin', 'againpass');

    expect(
      await screen.findByText('登录请求过于频繁，请稍后再试auth.login.retryIn:{"seconds":30}')
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /auth\.login\.waitSeconds/ })).toBeDisabled();
  });
});
