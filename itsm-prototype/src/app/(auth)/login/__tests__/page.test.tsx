import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import LoginPage from '../page';
import { AuthService } from '@/lib/services/auth-service';
import { useNotifications } from '@/lib/store/ui-store';

// Mock dependencies
const mockAuthService = {
    login: jest.fn(),
};

jest.mock('@/lib/services/auth-service', () => ({
  AuthService: mockAuthService,
}));
jest.mock('@/lib/store/ui-store');
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    replace: jest.fn(),
  }),
}));

// Mock Ant Design components with proper types
jest.mock('antd', () => ({
  Form: ({
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
  ),
  Input: ({
    placeholder,
    type,
    ...props
  }: {
    placeholder?: string;
    type?: string;
    [key: string]: unknown;
  }) => (
    <input 
      placeholder={placeholder} 
      type={type || 'text'} 
      data-testid={`input-${placeholder?.toLowerCase()}`}
      {...props} 
    />
  ),
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
}));

// Mock Lucide React icons
jest.mock('lucide-react', () => ({
  Lock: () => <div data-testid='lock-icon'>Lock</div>,
  User: () => <div data-testid='user-icon'>User</div>,
}));

const mockUseNotifications = useNotifications as jest.MockedFunction<typeof useNotifications>;

describe('LoginPage', () => {
  const mockNotifications = {
    success: jest.fn(),
    error: jest.fn(),
    warning: jest.fn(),
    info: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockUseNotifications.mockReturnValue(mockNotifications);
  });

  describe('Rendering', () => {
    it('should render login form with all required elements', () => {
      render(<LoginPage />);
      
      expect(screen.getByTestId('login-card')).toBeInTheDocument();
      expect(screen.getByTestId('login-title')).toHaveTextContent('登录');
      expect(screen.getByTestId('input-用户名')).toBeInTheDocument();
      expect(screen.getByTestId('input-密码')).toBeInTheDocument();
      expect(screen.getByTestId('login-button')).toBeInTheDocument();
    });

    it('should render icons for username and password fields', () => {
      render(<LoginPage />);
      
      expect(screen.getByTestId('user-icon')).toBeInTheDocument();
      expect(screen.getByTestId('lock-icon')).toBeInTheDocument();
    });

    it('should have proper form structure', () => {
      render(<LoginPage />);
      
      const form = screen.getByRole('form');
      expect(form).toBeInTheDocument();
      
      const usernameInput = screen.getByTestId('input-用户名');
      const passwordInput = screen.getByTestId('input-密码');
      
      expect(usernameInput).toHaveAttribute('placeholder', '用户名');
      expect(passwordInput).toHaveAttribute('type', 'password');
      expect(passwordInput).toHaveAttribute('placeholder', '密码');
    });
  });

  describe('Form Interaction', () => {
    it('should allow user to type in username and password fields', async () => {
      const user = userEvent.setup();
      render(<LoginPage />);
      
      const usernameInput = screen.getByTestId('input-用户名');
      const passwordInput = screen.getByTestId('input-密码');
      
      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'testpass');
      
      expect(usernameInput).toHaveValue('testuser');
      expect(passwordInput).toHaveValue('testpass');
    });

    it('should handle form submission', async () => {
      const user = userEvent.setup();
      mockAuthService.login = jest.fn().mockResolvedValue({
        success: true,
        user: { id: 1, username: 'testuser' },
        token: 'mock-token',
      });

      render(<LoginPage />);
      
      const usernameInput = screen.getByTestId('input-用户名');
      const passwordInput = screen.getByTestId('input-密码');
      const loginButton = screen.getByTestId('login-button');
      
      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'testpass');
      await user.click(loginButton);
      
      expect(mockAuthService.login).toHaveBeenCalledWith('testuser', 'testpass');
    });
  });

  describe('Authentication Flow', () => {
    it('should show loading state during login', async () => {
      const user = userEvent.setup();
      mockAuthService.login = jest
        .fn()
        .mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));

      render(<LoginPage />);
      
      const usernameInput = screen.getByTestId('input-用户名');
      const passwordInput = screen.getByTestId('input-密码');
      const loginButton = screen.getByTestId('login-button');
      
      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'testpass');
      await user.click(loginButton);
      
      expect(loginButton).toHaveTextContent('Loading...');
      expect(loginButton).toBeDisabled();
    });

    it('should handle successful login', async () => {
      const user = userEvent.setup();
      mockAuthService.login = jest.fn().mockResolvedValue({
        success: true,
        user: { id: 1, username: 'testuser' },
        token: 'mock-token',
      });

      render(<LoginPage />);
      
      const usernameInput = screen.getByTestId('input-用户名');
      const passwordInput = screen.getByTestId('input-密码');
      const loginButton = screen.getByTestId('login-button');
      
      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'testpass');
      await user.click(loginButton);
      
      await waitFor(() => {
        expect(mockNotifications.success).toHaveBeenCalledWith('登录成功');
      });
    });

    it('should handle login failure with error message', async () => {
      const user = userEvent.setup();
      const errorMessage = '用户名或密码错误';
      mockAuthService.login = jest.fn().mockRejectedValue(new Error(errorMessage));

      render(<LoginPage />);
      
      const usernameInput = screen.getByTestId('input-用户名');
      const passwordInput = screen.getByTestId('input-密码');
      const loginButton = screen.getByTestId('login-button');
      
      await user.type(usernameInput, 'wronguser');
      await user.type(passwordInput, 'wrongpass');
      await user.click(loginButton);
      
      await waitFor(() => {
        expect(mockNotifications.error).toHaveBeenCalledWith(errorMessage);
      });
    });

    it('should handle network error gracefully', async () => {
      const user = userEvent.setup();
      mockAuthService.login = jest.fn().mockRejectedValue(new Error('Network Error'));

      render(<LoginPage />);
      
      const usernameInput = screen.getByTestId('input-用户名');
      const passwordInput = screen.getByTestId('input-密码');
      const loginButton = screen.getByTestId('login-button');
      
      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'testpass');
      await user.click(loginButton);
      
      await waitFor(() => {
        expect(mockNotifications.error).toHaveBeenCalledWith('Network Error');
      });
    });
  });

  describe('Form Validation', () => {
    it('should handle empty form submission', async () => {
      const user = userEvent.setup();
      render(<LoginPage />);
      
      const loginButton = screen.getByTestId('login-button');
      await user.click(loginButton);
      
      // Form should still be submittable but AuthService should handle validation
      expect(mockAuthService.login).toHaveBeenCalledWith('', '');
    });

    it('should handle partial form data', async () => {
      const user = userEvent.setup();
      render(<LoginPage />);
      
      const usernameInput = screen.getByTestId('input-用户名');
      const loginButton = screen.getByTestId('login-button');
      
      await user.type(usernameInput, 'testuser');
      await user.click(loginButton);
      
      expect(mockAuthService.login).toHaveBeenCalledWith('testuser', '');
    });
  });

  describe('Accessibility', () => {
    it('should have proper form accessibility', () => {
      render(<LoginPage />);
      
      const form = screen.getByRole('form');
      expect(form).toBeInTheDocument();
      
      const loginButton = screen.getByRole('button');
      expect(loginButton).toBeInTheDocument();
    });

    it('should have proper input types for security', () => {
      render(<LoginPage />);
      
      const passwordInput = screen.getByTestId('input-密码');
      expect(passwordInput).toHaveAttribute('type', 'password');
    });
  });

  describe('Edge Cases', () => {
    it('should handle AuthService returning undefined', async () => {
      const user = userEvent.setup();
      mockAuthService.login = jest.fn().mockResolvedValue(undefined);

      render(<LoginPage />);
      
      const usernameInput = screen.getByTestId('input-用户名');
      const passwordInput = screen.getByTestId('input-密码');
      const loginButton = screen.getByTestId('login-button');
      
      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'testpass');
      await user.click(loginButton);
      
      await waitFor(() => {
        expect(mockAuthService.login).toHaveBeenCalled();
      });
    });

    it('should handle very long input values', async () => {
      const user = userEvent.setup();
      render(<LoginPage />);
      
      const usernameInput = screen.getByTestId('input-用户名');
      const longUsername = 'a'.repeat(1000);
      
      await user.type(usernameInput, longUsername);
      expect(usernameInput).toHaveValue(longUsername);
    });

    it('should handle special characters in input', async () => {
      const user = userEvent.setup();
      render(<LoginPage />);
      
      const usernameInput = screen.getByTestId('input-用户名');
      const passwordInput = screen.getByTestId('input-密码');
      
      const specialUsername = 'user@domain.com';
      const specialPassword = 'P@ssw0rd!#$';
      
      await user.type(usernameInput, specialUsername);
      await user.type(passwordInput, specialPassword);
      
      expect(usernameInput).toHaveValue(specialUsername);
      expect(passwordInput).toHaveValue(specialPassword);
    });
  });
});
