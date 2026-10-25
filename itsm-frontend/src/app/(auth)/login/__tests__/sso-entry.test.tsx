import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import LoginPage from '../page';

/**
 * 登录页 SSO 入口的行为锁。
 *
 * 背景：后端 SSO（OIDC）此前是死代码（装配缺失，路由不注册），登录页也没有
 * 任何入口。本文件锁定「提供方发现 → 点击发起 → 整页跳转 / 失败留原地」这条
 * 用户可见闭环，防止它再次退化为不可达功能。
 *
 * 只 mock 网络边界与副作用依赖（AuthAPI / AuthService / router / logger），
 * antd 用真组件渲染（与其余页面测试一致）。
 */
const mockListSSOProviders = jest.fn();
const mockInitiateSSOLogin = jest.fn();

jest.mock('@/lib/api/auth-api', () => ({
  AuthAPI: {
    listSSOProviders: (...args: unknown[]) => mockListSSOProviders(...args),
    initiateSSOLogin: (...args: unknown[]) => mockInitiateSSOLogin(...args),
  },
}));

jest.mock('@/lib/services/auth-service', () => ({
  AuthService: {
    login: jest.fn(),
  },
}));

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStoreHydration: jest.fn(),
}));

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), replace: jest.fn() }),
  useSearchParams: () => new URLSearchParams(),
}));

jest.mock('@/lib/env', () => ({
  logger: { info: jest.fn(), warn: jest.fn(), error: jest.fn() },
}));

jest.mock('@/lib/i18n/useI18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

describe('登录页 SSO 入口', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockListSSOProviders.mockResolvedValue([]);
  });

  it('存在启用中的提供方时按名称渲染按钮', async () => {
    mockListSSOProviders.mockResolvedValue([
      { code: 'okta', name: 'Okta 企业登录', protocol: 'oidc' },
      { code: 'azure', name: 'Azure AD', protocol: 'oidc' },
    ]);

    render(<LoginPage />);

    expect(await screen.findByRole('button', { name: /Okta 企业登录/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Azure AD/ })).toBeInTheDocument();
    // useI18n 被 mock 成恒等函数，分隔文案断言 i18n key 本身
    expect(screen.getByText('auth.login.ssoDivider')).toBeInTheDocument();
  });

  it('没有提供方时不渲染 SSO 区块（账号密码登录保持纯净）', async () => {
    render(<LoginPage />);

    await waitFor(() => expect(mockListSSOProviders).toHaveBeenCalled());
    expect(screen.queryByText('auth.login.ssoDivider')).not.toBeInTheDocument();
  });

  it('点击提供方按钮发起登录并整页跳转到授权 URL', async () => {
    mockListSSOProviders.mockResolvedValue([{ code: 'okta', name: 'Okta 企业登录' }]);
    mockInitiateSSOLogin.mockResolvedValue({
      success: true,
      redirectUrl: 'https://idp.example.com/authorize?state=st',
    });

    // jsdom 不允许真实导航：临时替换 location 以断言跳转目标。
    const originalLocation = window.location;
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, href: originalLocation.href },
      writable: true,
    });

    try {
      render(<LoginPage />);
      const button = await screen.findByRole('button', { name: /Okta 企业登录/ });
      await userEvent.click(button);

      await waitFor(() => expect(mockInitiateSSOLogin).toHaveBeenCalledWith('okta'));
      await waitFor(() =>
        expect(window.location.href).toBe('https://idp.example.com/authorize?state=st')
      );
    } finally {
      Object.defineProperty(window, 'location', { value: originalLocation, writable: true });
    }
  });

  it('发起失败时展示错误并让按钮恢复可用，不离开登录页', async () => {
    mockListSSOProviders.mockResolvedValue([{ code: 'okta', name: 'Okta 企业登录' }]);
    mockInitiateSSOLogin.mockResolvedValue({ success: false, error: 'SSO 登录发起失败' });

    render(<LoginPage />);
    const button = await screen.findByRole('button', { name: /Okta 企业登录/ });
    await userEvent.click(button);

    expect(await screen.findByText('SSO 登录发起失败')).toBeInTheDocument();
    await waitFor(() => expect(button).toBeEnabled());
  });
});
