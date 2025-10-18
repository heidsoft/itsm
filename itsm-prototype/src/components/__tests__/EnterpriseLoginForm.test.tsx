/**
 * 企业级ITSM登录组件单元测试
 * 使用React Testing Library和Jest进行测试
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import EnterpriseLoginForm from '../EnterpriseLoginForm';

// Mock数据
const mockLoginResponse = {
  success: true,
  token: 'mock-jwt-token',
  refreshToken: 'mock-refresh-token',
  user: {
    id: '1',
    username: 'testuser',
    email: 'test@example.com',
    role: 'admin',
  },
};

const mockMfaResponse = {
  success: false,
  requiresMfa: true,
  mfaType: 'totp' as const,
};

// Mock函数
const mockOnLogin = jest.fn();
const mockOnForgotPassword = jest.fn();
const mockOnSSOLogin = jest.fn();
const mockOnWebAuthnLogin = jest.fn();

// 测试工具函数
const renderLoginForm = (props = {}) => {
  const defaultProps = {
    onLogin: mockOnLogin,
    onForgotPassword: mockOnForgotPassword,
    onSSOLogin: mockOnSSOLogin,
    onWebAuthnLogin: mockOnWebAuthnLogin,
    language: 'zh' as const,
    enableSSO: true,
    enableWebAuthn: true,
    csrfToken: 'mock-csrf-token',
  };

  return render(<EnterpriseLoginForm {...defaultProps} {...props} />);
};

describe('EnterpriseLoginForm', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('基础渲染测试', () => {
    test('应该正确渲染登录表单', () => {
      renderLoginForm();

      expect(screen.getByRole('heading', { name: /系统登录/i })).toBeInTheDocument();
      expect(screen.getByLabelText(/用户名\/邮箱/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/密码/i)).toBeInTheDocument();
      expect(screen.getByRole('checkbox', { name: /记住我/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /登录/i })).toBeInTheDocument();
    });

    test('应该正确渲染英文界面', () => {
      renderLoginForm({ language: 'en' });

      expect(screen.getByRole('heading', { name: /system login/i })).toBeInTheDocument();
      expect(screen.getByLabelText(/username\/email/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    });

    test('应该显示SSO和WebAuthn按钮', () => {
      renderLoginForm();

      expect(screen.getByRole('button', { name: /sso 单点登录/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /使用生物识别登录/i })).toBeInTheDocument();
    });

    test('当禁用SSO时不应显示SSO按钮', () => {
      renderLoginForm({ enableSSO: false });

      expect(screen.queryByRole('button', { name: /sso 单点登录/i })).not.toBeInTheDocument();
    });
  });

  describe('表单验证测试', () => {
    test('应该验证必填字段', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const submitButton = screen.getByRole('button', { name: /登录/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/此字段为必填项/i)).toBeInTheDocument();
      });
    });

    test('应该验证邮箱格式', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'invalid-email');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/请输入有效的邮箱地址/i)).toBeInTheDocument();
      });
    });

    test('应该验证密码长度', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'test@example.com');
      await user.type(passwordInput, '123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/密码长度至少8位/i)).toBeInTheDocument();
      });
    });

    test('应该接受有效的用户名格式', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);

      await user.type(usernameInput, 'validuser123');
      await user.type(passwordInput, 'password123');

      // 验证没有错误信息
      expect(screen.queryByText(/请输入有效的邮箱地址/i)).not.toBeInTheDocument();
    });
  });

  describe('用户交互测试', () => {
    test('应该切换密码可见性', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const passwordInput = screen.getByLabelText(/密码/i) as HTMLInputElement;
      const toggleButton = screen.getByRole('button', { name: /显示密码/i });

      expect(passwordInput.type).toBe('password');

      await user.click(toggleButton);
      expect(passwordInput.type).toBe('text');

      await user.click(toggleButton);
      expect(passwordInput.type).toBe('password');
    });

    test('应该处理记住我选项', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const rememberMeCheckbox = screen.getByRole('checkbox', { name: /记住我/i }) as HTMLInputElement;

      expect(rememberMeCheckbox.checked).toBe(false);

      await user.click(rememberMeCheckbox);
      expect(rememberMeCheckbox.checked).toBe(true);
    });

    test('应该清除字段错误当用户输入时', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      // 触发验证错误
      await user.click(submitButton);
      await waitFor(() => {
        expect(screen.getByText(/此字段为必填项/i)).toBeInTheDocument();
      });

      // 输入内容应该清除错误
      await user.type(usernameInput, 'test');
      expect(screen.queryByText(/此字段为必填项/i)).not.toBeInTheDocument();
    });
  });

  describe('登录流程测试', () => {
    test('应该成功提交登录表单', async () => {
      const user = userEvent.setup();
      mockOnLogin.mockResolvedValue(mockLoginResponse);
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);
      const rememberMeCheckbox = screen.getByRole('checkbox', { name: /记住我/i });
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'test@example.com');
      await user.type(passwordInput, 'password123');
      await user.click(rememberMeCheckbox);
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockOnLogin).toHaveBeenCalledWith({
          username: 'test@example.com',
          password: 'password123',
          rememberMe: true,
          totpCode: '',
        });
      });
    });

    test('应该显示加载状态', async () => {
      const user = userEvent.setup();
      mockOnLogin.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 1000)));
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'test@example.com');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      expect(screen.getByText(/登录中.../i)).toBeInTheDocument();
      expect(submitButton).toBeDisabled();
    });

    test('应该处理登录错误', async () => {
      const user = userEvent.setup();
      mockOnLogin.mockRejectedValue(new Error('Invalid credentials'));
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'test@example.com');
      await user.type(passwordInput, 'wrongpassword');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/用户名或密码错误/i)).toBeInTheDocument();
      });
    });

    test('应该防止重复提交', async () => {
      const user = userEvent.setup();
      mockOnLogin.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'test@example.com');
      await user.type(passwordInput, 'password123');

      // 快速点击两次
      await user.click(submitButton);
      await user.click(submitButton);

      // 应该只调用一次
      expect(mockOnLogin).toHaveBeenCalledTimes(1);
    });
  });

  describe('MFA测试', () => {
    test('应该显示MFA界面当需要时', async () => {
      const user = userEvent.setup();
      mockOnLogin.mockResolvedValue(mockMfaResponse);
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'test@example.com');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /多因素认证/i })).toBeInTheDocument();
        expect(screen.getByLabelText(/totp 验证码/i)).toBeInTheDocument();
      });
    });

    test('应该验证TOTP代码格式', async () => {
      const user = userEvent.setup();
      mockOnLogin.mockResolvedValue(mockMfaResponse);
      renderLoginForm();

      // 先触发MFA界面
      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'test@example.com');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByLabelText(/totp 验证码/i)).toBeInTheDocument();
      });

      // 测试无效的TOTP代码
      const totpInput = screen.getByLabelText(/totp 验证码/i);
      const verifyButton = screen.getByRole('button', { name: /验证/i });

      await user.type(totpInput, '123');
      await user.click(verifyButton);

      await waitFor(() => {
        expect(screen.getByText(/验证码错误或已过期/i)).toBeInTheDocument();
      });
    });

    test('应该只允许数字输入TOTP代码', async () => {
      const user = userEvent.setup();
      mockOnLogin.mockResolvedValue(mockMfaResponse);
      renderLoginForm();

      // 先触发MFA界面
      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'test@example.com');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByLabelText(/totp 验证码/i)).toBeInTheDocument();
      });

      const totpInput = screen.getByLabelText(/totp 验证码/i) as HTMLInputElement;
      await user.type(totpInput, 'abc123def');

      expect(totpInput.value).toBe('123');
    });

    test('应该能够返回登录界面', async () => {
      const user = userEvent.setup();
      mockOnLogin.mockResolvedValue(mockMfaResponse);
      renderLoginForm();

      // 先触发MFA界面
      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'test@example.com');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /多因素认证/i })).toBeInTheDocument();
      });

      // 点击返回按钮
      const backButton = screen.getByRole('button', { name: /返回登录/i });
      await user.click(backButton);

      expect(screen.getByRole('heading', { name: /系统登录/i })).toBeInTheDocument();
    });
  });

  describe('外部登录测试', () => {
    test('应该调用SSO登录', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const ssoButton = screen.getByRole('button', { name: /sso 单点登录/i });
      await user.click(ssoButton);

      expect(mockOnSSOLogin).toHaveBeenCalled();
    });

    test('应该调用WebAuthn登录', async () => {
      const user = userEvent.setup();
      mockOnWebAuthnLogin.mockResolvedValue(mockLoginResponse);
      renderLoginForm();

      const webauthnButton = screen.getByRole('button', { name: /使用生物识别登录/i });
      await user.click(webauthnButton);

      expect(mockOnWebAuthnLogin).toHaveBeenCalled();
    });

    test('应该处理WebAuthn登录错误', async () => {
      const user = userEvent.setup();
      mockOnWebAuthnLogin.mockRejectedValue(new Error('WebAuthn not supported'));
      renderLoginForm();

      const webauthnButton = screen.getByRole('button', { name: /使用生物识别登录/i });
      await user.click(webauthnButton);

      await waitFor(() => {
        expect(screen.getByText(/网络连接失败，请重试/i)).toBeInTheDocument();
      });
    });

    test('应该调用忘记密码', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const forgotPasswordButton = screen.getByRole('button', { name: /忘记密码？/i });
      await user.click(forgotPasswordButton);

      expect(mockOnForgotPassword).toHaveBeenCalled();
    });
  });

  describe('安全特性测试', () => {
    test('应该限制登录尝试次数', async () => {
      const user = userEvent.setup();
      mockOnLogin.mockRejectedValue(new Error('Invalid credentials'));
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'test@example.com');
      await user.type(passwordInput, 'wrongpassword');

      // 尝试登录5次
      for (let i = 0; i < 5; i++) {
        await user.click(submitButton);
        await waitFor(() => {
          expect(screen.getByText(/用户名或密码错误/i)).toBeInTheDocument();
        });
      }

      // 第6次应该被阻止
      await user.click(submitButton);
      await waitFor(() => {
        expect(screen.getByText(/登录尝试次数过多，请稍后再试/i)).toBeInTheDocument();
      });
    });

    test('应该包含CSRF token', async () => {
      const user = userEvent.setup();
      mockOnLogin.mockResolvedValue(mockLoginResponse);
      renderLoginForm({ csrfToken: 'test-csrf-token' });

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);
      const submitButton = screen.getByRole('button', { name: /登录/i });

      await user.type(usernameInput, 'test@example.com');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      // 验证隐藏的CSRF token字段存在
      expect(screen.getByDisplayValue('test-csrf-token')).toBeInTheDocument();
    });
  });

  describe('可访问性测试', () => {
    test('应该有正确的ARIA标签', () => {
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);

      expect(usernameInput).toHaveAttribute('aria-invalid', 'false');
      expect(passwordInput).toHaveAttribute('aria-invalid', 'false');
      expect(usernameInput).toHaveAttribute('required');
      expect(passwordInput).toHaveAttribute('required');
    });

    test('应该在错误时设置正确的ARIA属性', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const submitButton = screen.getByRole('button', { name: /登录/i });
      await user.click(submitButton);

      await waitFor(() => {
        const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
        expect(usernameInput).toHaveAttribute('aria-invalid', 'true');
        expect(usernameInput).toHaveAttribute('aria-describedby', 'username-error');
      });
    });

    test('应该有屏幕阅读器公告', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const submitButton = screen.getByRole('button', { name: /登录/i });
      await user.click(submitButton);

      await waitFor(() => {
        const announcer = document.querySelector('[aria-live="assertive"]');
        expect(announcer).toBeInTheDocument();
      });
    });

    test('应该支持键盘导航', () => {
      renderLoginForm();

      const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
      const passwordInput = screen.getByLabelText(/密码/i);

      // 测试Tab键导航
      usernameInput.focus();
      expect(document.activeElement).toBe(usernameInput);

      fireEvent.keyDown(usernameInput, { key: 'Tab' });
      expect(document.activeElement).toBe(passwordInput);
    });
  });

  describe('自定义样式测试', () => {
    test('应该应用自定义className', () => {
      const { container } = renderLoginForm({ className: 'custom-login-form' });
      
      expect(container.firstChild).toHaveClass('custom-login-form');
    });

    test('应该在错误状态下应用错误样式', async () => {
      const user = userEvent.setup();
      renderLoginForm();

      const submitButton = screen.getByRole('button', { name: /登录/i });
      await user.click(submitButton);

      await waitFor(() => {
        const usernameInput = screen.getByLabelText(/用户名\/邮箱/i);
        expect(usernameInput).toHaveClass('border-red-300', 'bg-red-50');
      });
    });
  });
});

// 集成测试
describe('EnterpriseLoginForm Integration', () => {
  test('完整的登录流程', async () => {
    const user = userEvent.setup();
    const mockLogin = jest.fn().mockResolvedValue(mockLoginResponse);
    
    render(
      <EnterpriseLoginForm
        onLogin={mockLogin}
        language="zh"
        enableSSO={true}
        enableWebAuthn={true}
      />
    );

    // 填写表单
    await user.type(screen.getByLabelText(/用户名\/邮箱/i), 'admin@example.com');
    await user.type(screen.getByLabelText(/密码/i), 'password123');
    await user.click(screen.getByRole('checkbox', { name: /记住我/i }));

    // 提交表单
    await user.click(screen.getByRole('button', { name: /登录/i }));

    // 验证调用
    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith({
        username: 'admin@example.com',
        password: 'password123',
        rememberMe: true,
        totpCode: '',
      });
    });
  });

  test('完整的MFA流程', async () => {
    const user = userEvent.setup();
    const mockLogin = jest.fn()
      .mockResolvedValueOnce(mockMfaResponse)
      .mockResolvedValueOnce(mockLoginResponse);
    
    render(<EnterpriseLoginForm onLogin={mockLogin} />);

    // 第一步：普通登录
    await user.type(screen.getByLabelText(/用户名\/邮箱/i), 'admin@example.com');
    await user.type(screen.getByLabelText(/密码/i), 'password123');
    await user.click(screen.getByRole('button', { name: /登录/i }));

    // 等待MFA界面出现
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /多因素认证/i })).toBeInTheDocument();
    });

    // 第二步：输入TOTP代码
    await user.type(screen.getByLabelText(/totp 验证码/i), '123456');
    await user.click(screen.getByRole('button', { name: /验证/i }));

    // 验证两次调用
    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledTimes(2);
      expect(mockLogin).toHaveBeenLastCalledWith({
        username: 'admin@example.com',
        password: 'password123',
        rememberMe: false,
        totpCode: '123456',
      });
    });
  });
});