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
    login: (...args: any[]) => mockLogin(...args),
  },
}));

// Mock next/navigation
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
    replace: jest.fn(),
  }),
}));

// Mock Ant Design components
jest.mock('antd', () => {
  const actual = jest.requireActual('antd');

  const MockForm = ({
    children,
    onFinish,
    ...props
  }: {
    children: React.ReactNode;
    onFinish?: (values: Record<string, string>) => void;
    [key: string]: unknown;
  }) => (
    <form
      onSubmit={e => {
        e.preventDefault();
        onFinish?.({});
      }}
      {...props}
    >
      {children}
    </form>
  );

  // Attach static methods to MockForm
  MockForm.useForm = () => [
    {
      validateFields: jest.fn().mockResolvedValue({}),
      getFieldsValue: jest.fn(),
      setFieldsValue: jest.fn(),
      resetFields: jest.fn(),
    },
  ];

  MockForm.Item = ({ children }: any) => <div>{children}</div>;

  return {
    ...actual,
    message: {
      success: (...args: any[]) => mockSuccess(...args),
      error: (...args: any[]) => mockError(...args),
    },
    Form: MockForm,
    Input: (() => {
      const MockInput = ({
        placeholder,
        type,
        prefix,
        ...props
      }: {
        placeholder?: string;
        type?: string;
        prefix?: React.ReactNode;
        [key: string]: unknown;
      }) => (
        <div>
          {prefix}
          <input
            placeholder={placeholder}
            type={type || 'text'}
            data-testid={`input-${placeholder?.toLowerCase()}`}
            {...props}
          />
        </div>
      );

      MockInput.Password = ({
        placeholder,
        prefix,
        ...props
      }: {
        placeholder?: string;
        prefix?: React.ReactNode;
        [key: string]: unknown;
      }) => (
        <div>
          {prefix}
          <input
            type='password'
            placeholder={placeholder}
            data-testid={`input-${placeholder?.toLowerCase()}`}
            {...props}
          />
        </div>
      );

      return MockInput;
    })(),
    Button: ({
      children,
      loading,
      htmlType,
      ...props
    }: {
      children: React.ReactNode;
      loading?: boolean;
      htmlType?: 'button' | 'submit' | 'reset';
      [key: string]: unknown;
    }) => (
      <button type={htmlType || 'button'} disabled={loading} data-testid='login-button' {...props}>
        {loading ? 'Loading...' : children}
      </button>
    ),
    Card: ({
      children,
      title,
      ...props
    }: {
      children: React.ReactNode;
      title?: string;
      [key: string]: unknown;
    }) => (
      <div data-testid='login-card' {...props}>
        {title && <h2>{title}</h2>}
        {children}
      </div>
    ),
    Typography: {
      Title: ({ children, ...props }: { children: React.ReactNode; [key: string]: unknown }) => (
        <h1 data-testid='login-title' {...props}>
          {children}
        </h1>
      ),
      Text: ({
        children,
        type,
        ...props
      }: {
        children: React.ReactNode;
        type?: string;
        [key: string]: unknown;
      }) => (
        <span data-testid={`text-${type || 'default'}`} {...props}>
          {children}
        </span>
      ),
    },
    Space: ({ children, ...props }: { children: React.ReactNode; [key: string]: unknown }) => (
      <div data-testid='space' {...props}>
        {children}
      </div>
    ),
    ConfigProvider: ({ children }: any) => <div>{children}</div>,
    Alert: ({ message }: any) => <div data-testid='alert'>{message}</div>,
    Divider: ({ children }: any) => <div>{children}</div>,
    Checkbox: ({ children, ...props }: any) => (
      <label>
        <input type='checkbox' {...props} />
        {children}
      </label>
    ),
    Row: ({ children }: any) => <div className='row'>{children}</div>,
    Col: ({ children }: any) => <div className='col'>{children}</div>,
    Flex: ({ children }: any) => <div className='flex'>{children}</div>,
  };
});

// Mock Lucide icons
jest.mock('lucide-react', () => ({
  Lock: () => <div data-testid='lock-icon'>Lock</div>,
  User: () => <div data-testid='user-icon'>User</div>,
  Shield: () => <div data-testid='shield-icon'>Shield</div>,
  ArrowRight: () => <div data-testid='arrow-right-icon'>ArrowRight</div>,
  Eye: () => <div data-testid='eye-icon'>Eye</div>,
  EyeOff: () => <div data-testid='eye-off-icon'>EyeOff</div>,
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
      expect(screen.getByTestId('input-请输入用户名')).toBeInTheDocument();
      expect(screen.getByTestId('input-请输入密码')).toBeInTheDocument();
      expect(screen.getAllByTestId('login-button')[0]).toBeInTheDocument();
    });

    it('应该渲染用户名和密码字段的图标', () => {
      render(<LoginPage />);

      expect(screen.getByTestId('user-icon')).toBeInTheDocument();
      expect(screen.getByTestId('lock-icon')).toBeInTheDocument();
    });

    it('应该具有正确的表单结构', () => {
      render(<LoginPage />);

      expect(screen.getByRole('form')).toBeInTheDocument();
      expect(screen.getByTestId('input-请输入密码')).toHaveAttribute('type', 'password');
    });
  });

  describe('表单交互', () => {
    it('应该允许用户在用户名和密码字段中输入', async () => {
      const user = userEvent.setup();
      render(<LoginPage />);

      const usernameInput = screen.getByTestId('input-请输入用户名');
      const passwordInput = screen.getByTestId('input-请输入密码');

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'testpass');

      expect(usernameInput).toHaveValue('testuser');
      expect(passwordInput).toHaveValue('testpass');
    });

    it('应该处理表单提交', async () => {
      const user = userEvent.setup();
      mockLogin.mockResolvedValue({
        success: true,
        user: { id: 1, username: 'testuser' },
        token: 'mock-token',
      });

      render(<LoginPage />);

      const usernameInput = screen.getByTestId('input-请输入用户名');
      const passwordInput = screen.getByTestId('input-请输入密码');
      // The button with "登录" text is the submit button
      const loginButton = screen.getByRole('button', { name: '登录' });

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'testpass');
      await user.click(loginButton);

      expect(mockLogin).toHaveBeenCalledWith('testuser', 'testpass');
    });
  });

  describe('认证流程', () => {
    it('应该处理成功登录', async () => {
      const user = userEvent.setup();
      mockLogin.mockResolvedValue({
        success: true,
        user: { id: 1, username: 'testuser' },
        token: 'mock-token',
      });

      render(<LoginPage />);

      const usernameInput = screen.getByTestId('input-请输入用户名');
      const passwordInput = screen.getByTestId('input-请输入密码');
      const loginButton = screen.getByRole('button', { name: '登录' });

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'testpass');
      await user.click(loginButton);

      await waitFor(() => {
        expect(mockSuccess).toHaveBeenCalledWith('登录成功');
      });

      // Should redirect to dashboard
      await waitFor(() => {
        expect(mockPush).toHaveBeenCalledWith('/dashboard');
      });
    });

    it('应该处理登录失败并显示错误信息', async () => {
      const user = userEvent.setup();
      const errorMessage = '用户名或密码错误';
      mockLogin.mockRejectedValue(new Error(errorMessage));

      render(<LoginPage />);

      const usernameInput = screen.getByTestId('input-请输入用户名');
      const passwordInput = screen.getByTestId('input-请输入密码');
      const loginButton = screen.getByRole('button', { name: '登录' });

      await user.type(usernameInput, 'wronguser');
      await user.type(passwordInput, 'wrongpass');
      await user.click(loginButton);

      await waitFor(() => {
        expect(screen.getByTestId('alert')).toHaveTextContent(errorMessage);
      });
    });
  });

  describe('边界情况', () => {
    it('应该处理AuthService返回undefined', async () => {
      const user = userEvent.setup();
      mockLogin.mockResolvedValue(undefined);

      render(<LoginPage />);

      const usernameInput = screen.getByTestId('input-请输入用户名');
      const passwordInput = screen.getByTestId('input-请输入密码');
      const loginButton = screen.getByRole('button', { name: '登录' });

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'testpass');
      await user.click(loginButton);

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalled();
      });
    });
  });
});
