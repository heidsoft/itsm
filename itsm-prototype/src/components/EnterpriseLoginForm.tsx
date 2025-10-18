'use client';

import React, { useState, useRef, useEffect, useCallback } from 'react';
import { 
  User, 
  Lock, 
  Eye, 
  EyeOff, 
  Mail, 
  AlertCircle, 
  CheckCircle, 
  Loader2,
  ArrowRight,
  Fingerprint,
  Smartphone,
  Key,
  Globe
} from 'lucide-react';

// 国际化文本配置
const i18nTexts = {
  zh: {
    title: '系统登录',
    subtitle: '请输入您的账户信息',
    usernameLabel: '用户名/邮箱',
    usernamePlaceholder: '请输入用户名或邮箱',
    passwordLabel: '密码',
    passwordPlaceholder: '请输入密码',
    rememberMe: '记住我',
    forgotPassword: '忘记密码？',
    loginButton: '登录',
    loggingIn: '登录中...',
    ssoLogin: 'SSO 单点登录',
    mfaTitle: '多因素认证',
    totpCode: 'TOTP 验证码',
    totpPlaceholder: '请输入6位验证码',
    webauthnButton: '使用生物识别登录',
    backToLogin: '返回登录',
    errors: {
      required: '此字段为必填项',
      invalidEmail: '请输入有效的邮箱地址',
      passwordTooShort: '密码长度至少8位',
      invalidCredentials: '用户名或密码错误',
      networkError: '网络连接失败，请重试',
      tooManyAttempts: '登录尝试次数过多，请稍后再试',
      mfaRequired: '需要多因素认证',
      invalidMfaCode: '验证码错误或已过期'
    }
  },
  en: {
    title: 'System Login',
    subtitle: 'Please enter your account information',
    usernameLabel: 'Username/Email',
    usernamePlaceholder: 'Enter username or email',
    passwordLabel: 'Password',
    passwordPlaceholder: 'Enter password',
    rememberMe: 'Remember me',
    forgotPassword: 'Forgot password?',
    loginButton: 'Login',
    loggingIn: 'Logging in...',
    ssoLogin: 'SSO Login',
    mfaTitle: 'Multi-Factor Authentication',
    totpCode: 'TOTP Code',
    totpPlaceholder: 'Enter 6-digit code',
    webauthnButton: 'Use Biometric Login',
    backToLogin: 'Back to Login',
    errors: {
      required: 'This field is required',
      invalidEmail: 'Please enter a valid email address',
      passwordTooShort: 'Password must be at least 8 characters',
      invalidCredentials: 'Invalid username or password',
      networkError: 'Network connection failed, please try again',
      tooManyAttempts: 'Too many login attempts, please try again later',
      mfaRequired: 'Multi-factor authentication required',
      invalidMfaCode: 'Invalid or expired verification code'
    }
  }
};

// 类型定义
interface LoginFormData {
  username: string;
  password: string;
  rememberMe: boolean;
  totpCode?: string;
}

interface LoginResponse {
  success: boolean;
  token?: string;
  refreshToken?: string;
  requiresMfa?: boolean;
  mfaType?: 'totp' | 'webauthn';
  user?: {
    id: string;
    username: string;
    email: string;
    role: string;
  };
}

interface EnterpriseLoginFormProps {
  onLogin?: (data: LoginFormData) => Promise<LoginResponse>;
  onForgotPassword?: () => void;
  onSSOLogin?: () => void;
  onWebAuthnLogin?: () => Promise<LoginResponse>;
  language?: 'zh' | 'en';
  enableSSO?: boolean;
  enableMFA?: boolean;
  enableWebAuthn?: boolean;
  csrfToken?: string;
  className?: string;
}

// 验证函数
const validateEmail = (email: string): boolean => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
};

const validateUsername = (username: string): boolean => {
  return username.length >= 3 && /^[a-zA-Z0-9_.-]+$/.test(username);
};

const validatePassword = (password: string): boolean => {
  return password.length >= 8;
};

const validateTOTPCode = (code: string): boolean => {
  return /^\d{6}$/.test(code);
};

// 主组件
export const EnterpriseLoginForm: React.FC<EnterpriseLoginFormProps> = ({
  onLogin,
  onForgotPassword,
  onSSOLogin,
  onWebAuthnLogin,
  language = 'zh',
  enableSSO = true,
  enableMFA = true,
  enableWebAuthn = true,
  csrfToken,
  className = ''
}) => {
  // 状态管理
  const [formData, setFormData] = useState<LoginFormData>({
    username: '',
    password: '',
    rememberMe: false,
    totpCode: ''
  });
  
  const [isLoading, setIsLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loginAttempts, setLoginAttempts] = useState(0);
  const [isBlocked, setIsBlocked] = useState(false);
  const [showMFA, setShowMFA] = useState(false);
  const [mfaType, setMfaType] = useState<'totp' | 'webauthn'>('totp');
  
  // 引用
  const usernameRef = useRef<HTMLInputElement>(null);
  const passwordRef = useRef<HTMLInputElement>(null);
  const totpRef = useRef<HTMLInputElement>(null);
  const errorAnnouncerRef = useRef<HTMLDivElement>(null);
  
  // 获取当前语言文本
  const t = i18nTexts[language];
  
  // 防抖提交
  const [submitTimeout, setSubmitTimeout] = useState<NodeJS.Timeout | null>(null);
  
  // 清理定时器
  useEffect(() => {
    return () => {
      if (submitTimeout) {
        clearTimeout(submitTimeout);
      }
    };
  }, [submitTimeout]);
  
  // 处理登录尝试限制
  useEffect(() => {
    if (loginAttempts >= 5) {
      setIsBlocked(true);
      const timer = setTimeout(() => {
        setIsBlocked(false);
        setLoginAttempts(0);
      }, 300000); // 5分钟后解除限制
      
      return () => clearTimeout(timer);
    }
  }, [loginAttempts]);
  
  // 表单验证
  const validateForm = useCallback((): boolean => {
    const newErrors: Record<string, string> = {};
    
    // 用户名验证
    if (!formData.username.trim()) {
      newErrors.username = t.errors.required;
    } else if (formData.username.includes('@')) {
      if (!validateEmail(formData.username)) {
        newErrors.username = t.errors.invalidEmail;
      }
    } else if (!validateUsername(formData.username)) {
      newErrors.username = t.errors.invalidEmail; // 复用邮箱错误信息
    }
    
    // 密码验证
    if (!formData.password) {
      newErrors.password = t.errors.required;
    } else if (!validatePassword(formData.password)) {
      newErrors.password = t.errors.passwordTooShort;
    }
    
    // MFA验证码验证
    if (showMFA && mfaType === 'totp') {
      if (!formData.totpCode) {
        newErrors.totpCode = t.errors.required;
      } else if (!validateTOTPCode(formData.totpCode)) {
        newErrors.totpCode = t.errors.invalidMfaCode;
      }
    }
    
    setErrors(newErrors);
    
    // 宣布错误给屏幕阅读器
    if (Object.keys(newErrors).length > 0 && errorAnnouncerRef.current) {
      errorAnnouncerRef.current.textContent = Object.values(newErrors).join(', ');
    }
    
    return Object.keys(newErrors).length === 0;
  }, [formData, showMFA, mfaType, t]);
  
  // 处理输入变化
  const handleInputChange = useCallback((field: keyof LoginFormData, value: string | boolean) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
    
    // 清除对应字段的错误
    if (errors[field]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[field];
        return newErrors;
      });
    }
  }, [errors]);
  
  // 处理表单提交
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (isBlocked) {
      setErrors({ general: t.errors.tooManyAttempts });
      return;
    }
    
    if (!validateForm() || isLoading) {
      return;
    }
    
    // 防重入提交
    if (submitTimeout) {
      clearTimeout(submitTimeout);
    }
    
    setIsLoading(true);
    setErrors({});
    
    try {
      if (!onLogin) {
        throw new Error('Login handler not provided');
      }
      
      const response = await onLogin(formData);
      
      if (response.success) {
        // 登录成功，重置尝试次数
        setLoginAttempts(0);
        
        // 如果需要MFA且启用了MFA功能
        if (response.requiresMfa && enableMFA) {
          setShowMFA(true);
          setMfaType(response.mfaType || 'totp');
          
          // 聚焦到MFA输入框
          setTimeout(() => {
            if (response.mfaType === 'totp' && totpRef.current) {
              totpRef.current.focus();
            }
          }, 100);
        }
      } else {
        throw new Error(t.errors.invalidCredentials);
      }
    } catch (error) {
      setLoginAttempts(prev => prev + 1);
      
      let errorMessage = t.errors.networkError;
      if (error instanceof Error) {
        if (error.message.includes('credentials')) {
          errorMessage = t.errors.invalidCredentials;
        } else if (error.message.includes('network')) {
          errorMessage = t.errors.networkError;
        } else if (error.message.includes('mfa')) {
          errorMessage = t.errors.invalidMfaCode;
        }
      }
      
      setErrors({ general: errorMessage });
    } finally {
      setIsLoading(false);
      
      // 设置防抖延迟
      const timeout = setTimeout(() => {
        setSubmitTimeout(null);
      }, 1000);
      setSubmitTimeout(timeout);
    }
  };
  
  // 处理WebAuthn登录
  const handleWebAuthnLogin = async () => {
    if (!onWebAuthnLogin || isLoading) return;
    
    setIsLoading(true);
    setErrors({});
    
    try {
      const response = await onWebAuthnLogin();
      if (response.success) {
        setLoginAttempts(0);
      }
    } catch {
      setErrors({ general: t.errors.networkError });
    } finally {
      setIsLoading(false);
    }
  };
  
  // 切换密码可见性
  const togglePasswordVisibility = () => {
    setShowPassword(!showPassword);
  };
  
  // 返回登录表单
  const backToLogin = () => {
    setShowMFA(false);
    setFormData(prev => ({ ...prev, totpCode: '' }));
    setErrors({});
  };
  
  return (
    <div className={`w-full max-w-md mx-auto ${className}`}>
      {/* 错误宣布器（屏幕阅读器） */}
      <div
        ref={errorAnnouncerRef}
        className="sr-only"
        aria-live="assertive"
        aria-atomic="true"
      />
      
      {/* 表单标题 */}
      <div className="text-center mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">
          {showMFA ? t.mfaTitle : t.title}
        </h1>
        <p className="text-gray-600">
          {showMFA ? '请完成多因素认证' : t.subtitle}
        </p>
      </div>
      
      {/* 通用错误提示 */}
      {errors.general && (
        <div 
          className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start"
          role="alert"
          aria-live="polite"
        >
          <AlertCircle className="w-5 h-5 text-red-500 mr-3 flex-shrink-0 mt-0.5" />
          <span className="text-red-700 text-sm">{errors.general}</span>
        </div>
      )}
      
      {/* 登录表单 */}
      {!showMFA ? (
        <form onSubmit={handleSubmit} className="space-y-6" noValidate>
          {/* CSRF Token */}
          {csrfToken && (
            <input type="hidden" name="_token" value={csrfToken} />
          )}
          
          {/* 用户名/邮箱输入 */}
          <div>
            <label 
              htmlFor="username" 
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              {t.usernameLabel}
              <span className="text-red-500 ml-1" aria-label="必填">*</span>
            </label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                {formData.username.includes('@') ? (
                  <Mail className="h-5 w-5 text-gray-400" aria-hidden="true" />
                ) : (
                  <User className="h-5 w-5 text-gray-400" aria-hidden="true" />
                )}
              </div>
              <input
                ref={usernameRef}
                id="username"
                name="username"
                type="text"
                autoComplete="username"
                value={formData.username}
                onChange={(e) => handleInputChange('username', e.target.value)}
                className={`block w-full pl-10 pr-3 py-3 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors ${
                  errors.username 
                    ? 'border-red-300 bg-red-50' 
                    : 'border-gray-300 bg-white'
                }`}
                placeholder={t.usernamePlaceholder}
                aria-invalid={errors.username ? 'true' : 'false'}
                aria-describedby={errors.username ? 'username-error' : undefined}
                required
              />
            </div>
            {errors.username && (
              <p 
                id="username-error" 
                className="mt-2 text-sm text-red-600"
                role="alert"
              >
                {errors.username}
              </p>
            )}
          </div>
          
          {/* 密码输入 */}
          <div>
            <label 
              htmlFor="password" 
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              {t.passwordLabel}
              <span className="text-red-500 ml-1" aria-label="必填">*</span>
            </label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Lock className="h-5 w-5 text-gray-400" aria-hidden="true" />
              </div>
              <input
                ref={passwordRef}
                id="password"
                name="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                value={formData.password}
                onChange={(e) => handleInputChange('password', e.target.value)}
                className={`block w-full pl-10 pr-12 py-3 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors ${
                  errors.password 
                    ? 'border-red-300 bg-red-50' 
                    : 'border-gray-300 bg-white'
                }`}
                placeholder={t.passwordPlaceholder}
                aria-invalid={errors.password ? 'true' : 'false'}
                aria-describedby={errors.password ? 'password-error' : undefined}
                required
              />
              <button
                type="button"
                onClick={togglePasswordVisibility}
                className="absolute inset-y-0 right-0 pr-3 flex items-center"
                aria-label={showPassword ? '隐藏密码' : '显示密码'}
              >
                {showPassword ? (
                  <EyeOff className="h-5 w-5 text-gray-400 hover:text-gray-600" />
                ) : (
                  <Eye className="h-5 w-5 text-gray-400 hover:text-gray-600" />
                )}
              </button>
            </div>
            {errors.password && (
              <p 
                id="password-error" 
                className="mt-2 text-sm text-red-600"
                role="alert"
              >
                {errors.password}
              </p>
            )}
          </div>
          
          {/* 记住我和忘记密码 */}
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <input
                id="remember-me"
                name="remember-me"
                type="checkbox"
                checked={formData.rememberMe}
                onChange={(e) => handleInputChange('rememberMe', e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="remember-me" className="ml-2 block text-sm text-gray-700">
                {t.rememberMe}
              </label>
            </div>
            
            {onForgotPassword && (
              <button
                type="button"
                onClick={onForgotPassword}
                className="text-sm text-blue-600 hover:text-blue-700 focus:outline-none focus:underline transition-colors"
              >
                {t.forgotPassword}
              </button>
            )}
          </div>
          
          {/* 登录按钮 */}
          <button
            type="submit"
            disabled={isLoading || isBlocked}
            className="w-full flex justify-center items-center px-4 py-3 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-5 h-5 mr-2 animate-spin" />
                <span>{t.loggingIn}</span>
              </>
            ) : (
              <>
                <span>{t.loginButton}</span>
                <ArrowRight className="w-5 h-5 ml-2" />
              </>
            )}
          </button>
          
          {/* 分隔线 */}
          {(enableSSO || enableWebAuthn) && (
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-gray-300" />
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-2 bg-white text-gray-500">或</span>
              </div>
            </div>
          )}
          
          {/* SSO登录 */}
          {enableSSO && onSSOLogin && (
            <button
              type="button"
              onClick={onSSOLogin}
              disabled={isLoading}
              className="w-full flex justify-center items-center px-4 py-3 border border-gray-300 rounded-lg shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <Globe className="w-5 h-5 mr-2" />
              <span>{t.ssoLogin}</span>
            </button>
          )}
          
          {/* WebAuthn登录 */}
          {enableWebAuthn && onWebAuthnLogin && (
            <button
              type="button"
              onClick={handleWebAuthnLogin}
              disabled={isLoading}
              className="w-full flex justify-center items-center px-4 py-3 border border-gray-300 rounded-lg shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <Fingerprint className="w-5 h-5 mr-2" />
              <span>{t.webauthnButton}</span>
            </button>
          )}
        </form>
      ) : (
        /* MFA表单 */
        <form onSubmit={handleSubmit} className="space-y-6" noValidate>
          {mfaType === 'totp' ? (
            /* TOTP验证码输入 */
            <div>
              <label 
                htmlFor="totp-code" 
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                {t.totpCode}
                <span className="text-red-500 ml-1" aria-label="必填">*</span>
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <Smartphone className="h-5 w-5 text-gray-400" aria-hidden="true" />
                </div>
                <input
                  ref={totpRef}
                  id="totp-code"
                  name="totp-code"
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]*"
                  maxLength={6}
                  value={formData.totpCode || ''}
                  onChange={(e) => handleInputChange('totpCode', e.target.value.replace(/\D/g, ''))}
                  className={`block w-full pl-10 pr-3 py-3 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors text-center text-lg tracking-widest ${
                    errors.totpCode 
                      ? 'border-red-300 bg-red-50' 
                      : 'border-gray-300 bg-white'
                  }`}
                  placeholder={t.totpPlaceholder}
                  aria-invalid={errors.totpCode ? 'true' : 'false'}
                  aria-describedby={errors.totpCode ? 'totp-error' : undefined}
                  required
                />
              </div>
              {errors.totpCode && (
                <p 
                  id="totp-error" 
                  className="mt-2 text-sm text-red-600"
                  role="alert"
                >
                  {errors.totpCode}
                </p>
              )}
            </div>
          ) : (
            /* WebAuthn验证 */
            <div className="text-center py-8">
              <Fingerprint className="w-16 h-16 text-blue-600 mx-auto mb-4" />
              <p className="text-gray-600 mb-6">请使用您的生物识别设备完成验证</p>
              <button
                type="button"
                onClick={handleWebAuthnLogin}
                disabled={isLoading}
                className="inline-flex items-center px-6 py-3 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {isLoading ? (
                  <Loader2 className="w-5 h-5 mr-2 animate-spin" />
                ) : (
                  <Key className="w-5 h-5 mr-2" />
                )}
                <span>开始验证</span>
              </button>
            </div>
          )}
          
          {/* MFA操作按钮 */}
          <div className="flex space-x-4">
            <button
              type="button"
              onClick={backToLogin}
              disabled={isLoading}
              className="flex-1 px-4 py-3 border border-gray-300 rounded-lg shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {t.backToLogin}
            </button>
            
            {mfaType === 'totp' && (
              <button
                type="submit"
                disabled={isLoading || !formData.totpCode}
                className="flex-1 flex justify-center items-center px-4 py-3 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {isLoading ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  <CheckCircle className="w-5 h-5" />
                )}
                <span className="ml-2">验证</span>
              </button>
            )}
          </div>
        </form>
      )}
    </div>
  );
};

export default EnterpriseLoginForm;