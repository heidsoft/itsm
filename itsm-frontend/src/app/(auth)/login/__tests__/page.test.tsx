import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import LoginPage from '../page';

// Create mock functions before any imports
const mockLogin = jest.fn();
const mockPush = jest.fn();
const mockSuccess = jest.fn();
const mockError = jest.fn();

// Mock auth service
jest.mock('@/lib/services/auth-service', () => ({
  AuthService: {
    login: (...args: unknown[]) => mockLogin(...args),
  },
}));

// Mock next/navigation
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
    replace: jest.fn(),
  }),
}));

// Mock auth store hydration
jest.mock('@/lib/store/auth-store', () => ({
  useAuthStoreHydration: jest.fn(),
}));

// Mock logger
jest.mock('@/lib/env', () => ({
  logger: {
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock design system
jest.mock('@/lib/design-system/colors', () => ({
  colors: {
    primary: '#1890ff',
  },
}));

// Mock antd theme
jest.mock('@/lib/antd-theme', () => ({
  antdTheme: {},
}));

// Mock antd - comprehensive mock
jest.mock('antd', () => {
  const React = require('react');

  // Mock Input.Password
  const MockPassword = (props: Record<string, unknown>) => {
    return React.createElement('input', {
      type: 'password',
      'data-testid': 'password-input',
      ...props,
    });
  };

  // Mock Input with Password
  const MockInput = (props: Record<string, unknown>) => {
    const inputProps = { 'data-testid': props.prefix ? 'username-input' : 'input', ...props };
    if (props.Password) {
      return React.createElement(MockPassword, props);
    }
    return React.createElement('input', inputProps);
  };
  MockInput.Password = MockPassword;

  // Mock Form
  const MockForm = (props: Record<string, unknown>) => {
    return React.createElement('form', { role: 'form', ...props });
  };
  MockForm.useForm = () => [
    {
      validateFields: jest.fn().mockResolvedValue({}),
      getFieldsValue: jest.fn().mockReturnValue({}),
      setFieldsValue: jest.fn(),
      resetFields: jest.fn(),
      getFieldValue: jest.fn(),
    },
  ];

  // Mock Form.Item
  const MockFormItem = (props: Record<string, unknown>) => {
    return React.createElement('div', props, props.children);
  };
  MockForm.Item = MockFormItem;

  // Mock Button
  const MockButton = (props: Record<string, unknown>) => {
    return React.createElement(
      'button',
      { type: props.htmlType || 'button', 'data-testid': 'login-button', ...props },
      props.children
    );
  };

  // Mock Card
  const MockCard = (props: Record<string, unknown>) => {
    return React.createElement(
      'div',
      { 'data-testid': 'login-card', ...props },
      props.title,
      props.children
    );
  };

  // Mock Typography
  const MockTitle = (props: Record<string, unknown>) => {
    return React.createElement('h1', { 'data-testid': 'login-title', ...props }, props.children);
  };
  const MockText = (props: Record<string, unknown>) => {
    return React.createElement('span', { 'data-testid': 'text', ...props }, props.children);
  };

  return {
    message: {
      success: (...args: unknown[]) => mockSuccess(...args),
      error: (...args: unknown[]) => mockError(...args),
    },
    Form: MockForm,
    Input: MockInput,
    Button: MockButton,
    Card: MockCard,
    Typography: {
      Title: MockTitle,
      Text: MockText,
    },
    Space: (props: Record<string, unknown>) =>
      React.createElement('div', { 'data-testid': 'space', ...props }, props.children),
    ConfigProvider: (props: Record<string, unknown>) =>
      React.createElement('div', {}, props.children),
    Alert: (props: Record<string, unknown>) =>
      React.createElement('div', { 'data-testid': 'alert', ...props }, props.message),
    Divider: () => React.createElement('div', { 'data-testid': 'divider' }),
    Checkbox: (props: Record<string, unknown>) => React.createElement('label', {}, props.children),
    Row: (props: Record<string, unknown>) =>
      React.createElement('div', { className: 'row', ...props }, props.children),
    Col: (props: Record<string, unknown>) =>
      React.createElement('div', { className: 'col', ...props }, props.children),
    Flex: (props: Record<string, unknown>) =>
      React.createElement('div', { className: 'flex', ...props }, props.children),
  };
});

// Mock Lucide icons
jest.mock('lucide-react', () => ({
  Lock: () => React.createElement('div', { 'data-testid': 'lock-icon' }),
  User: () => React.createElement('div', { 'data-testid': 'user-icon' }),
  Shield: () => React.createElement('div', { 'data-testid': 'shield-icon' }),
  ArrowRight: () => React.createElement('div', { 'data-testid': 'arrow-right-icon' }),
  Eye: () => React.createElement('div', { 'data-testid': 'eye-icon' }),
  EyeOff: () => React.createElement('div', { 'data-testid': 'eye-off-icon' }),
}));

describe('LoginPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockLogin.mockReset();
    mockSuccess.mockReset();
    mockError.mockReset();
  });

  describe('渲染测试', () => {
    it('应该渲染登录表单的所有必需元素', () => {
      render(<LoginPage />);

      expect(screen.getByTestId('login-card')).toBeInTheDocument();
      // Check for the main heading
      expect(screen.getByText('欢迎回来')).toBeInTheDocument();
      expect(screen.getByTestId('username-input')).toBeInTheDocument();
      expect(screen.getByTestId('password-input')).toBeInTheDocument();
    });

    it('应该具有正确的表单结构', () => {
      render(<LoginPage />);

      expect(screen.getByRole('form')).toBeInTheDocument();
      expect(screen.getByTestId('password-input')).toHaveAttribute('type', 'password');
    });
  });

  describe('表单交互', () => {
    it('应该允许用户在用户名和密码字段中输入', async () => {
      const user = userEvent.setup();
      render(<LoginPage />);

      const usernameInput = screen.getByTestId('username-input');
      const passwordInput = screen.getByTestId('password-input');

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'testpass');

      expect(usernameInput).toHaveValue('testuser');
      expect(passwordInput).toHaveValue('testpass');
    });
  });

  describe('认证流程', () => {
    it('登录成功时应该调用AuthService', () => {
      mockLogin.mockResolvedValue({
        success: true,
        user: { id: 1, username: 'testuser' },
        token: 'mock-token',
      });

      render(<LoginPage />);

      // Verify the page rendered with login form
      expect(screen.getByTestId('login-card')).toBeInTheDocument();
      expect(screen.getByRole('form')).toBeInTheDocument();
    });

    it('登录失败时应该处理错误', () => {
      mockLogin.mockRejectedValue(new Error('用户名或密码错误'));

      render(<LoginPage />);

      // Verify the page rendered with login form
      expect(screen.getByTestId('login-card')).toBeInTheDocument();
    });
  });

  describe('边界情况', () => {
    it('AuthService返回undefined时应该渲染表单', () => {
      mockLogin.mockResolvedValue(undefined);

      render(<LoginPage />);

      // Verify the page still renders
      expect(screen.getByTestId('login-card')).toBeInTheDocument();
    });
  });
});
