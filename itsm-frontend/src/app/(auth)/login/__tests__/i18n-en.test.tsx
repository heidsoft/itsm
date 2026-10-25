import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

/**
 * 登录页英文模式文案锁。
 *
 * 此前页面把开发账号提示、会话过期提示、SSO 分隔文案和品牌卖点写死成中文，
 * 切到 English 后整页中英混排。useI18n 走真实现（只 mock 语言偏好），
 * 所以这里能同时抓到「漏接 t()」和「key 没定义在 en-US」两种回归。
 */
jest.mock('@/lib/user-preferences', () => ({
  userPreferences: {
    get: () => ({ language: 'en-US' }),
    subscribe: () => () => {},
  },
}));

jest.mock('@/lib/api/auth-api', () => ({
  AuthAPI: {
    listSSOProviders: () => Promise.resolve([{ code: 'okta', name: 'Okta Enterprise', protocol: 'oidc' }]),
    initiateSSOLogin: jest.fn(),
  },
}));

jest.mock('@/lib/services/auth-service', () => ({
  AuthService: { login: jest.fn() },
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

import LoginPage from '../page';

describe('登录页英文模式', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('渲染文案不含中文，也不泄漏未命中的 i18n key', async () => {
    render(<LoginPage />);

    expect(await screen.findByText('Welcome Back')).toBeInTheDocument();
    expect(screen.getByText('or sign in with your enterprise account')).toBeInTheDocument();
    expect(screen.getByText('AI-driven IT service management')).toBeInTheDocument();

    const text = document.body.textContent || '';
    // 未定义在 en-US 的 key 会被 useI18n 原样返回，等同于文案缺失
    expect(text).not.toMatch(/auth\.login\.[a-zA-Z]/);
    expect(text).not.toMatch(/[一-龥]/);
  });
});
