'use client';

import React, { useState } from 'react';
import { LoginForm } from '@/components/EnterpriseLoginForm';

interface LoginUser {
  username?: string;
  email?: string;
  id?: string;
}

interface LoginFormData {
  username: string;
  password: string;
  rememberMe: boolean;
  totpCode?: string;
}

export default function EnterpriseLoginDemo() {
  const [language, setLanguage] = useState<'zh' | 'en'>('zh');
  const [loginResult, setLoginResult] = useState<string>('');
  const [showResult, setShowResult] = useState(false);

  const handleLoginSuccess = (user: LoginUser) => {
    setLoginResult(`登录成功！欢迎 ${user.username || user.email}`);
    setShowResult(true);
    setTimeout(() => setShowResult(false), 5000);
  };

  // 模拟登录API调用
  const handleLogin = async (data: LoginFormData) => {
    try {
      // 模拟网络延迟
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      // 模拟登录逻辑
      if (data.username === 'admin' && data.password === 'password123') {
        const user = {
          id: '1',
          username: data.username,
          email: 'admin@example.com',
          role: 'admin'
        };
        handleLoginSuccess(user);
        return {
          success: true,
          token: 'mock-jwt-token',
          user
        };
      } else if (data.username === 'mfa-user' && data.password === 'password123') {
        return {
          success: true,
          requiresMfa: true,
          mfaType: 'totp' as const
        };
      } else {
        throw new Error('用户名或密码错误');
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '登录失败';
      setLoginResult(`登录失败：${errorMessage}`);
      setShowResult(true);
      setTimeout(() => setShowResult(false), 5000);
      throw error;
    }
  };

  const handleForgotPassword = () => {
    alert('忘记密码功能 - 通常会跳转到密码重置页面');
  };

  const handleSSOLogin = () => {
    alert('SSO登录 - 通常会跳转到SSO提供商');
  };

  const handleWebAuthnLogin = async () => {
    try {
      // 模拟WebAuthn登录
      await new Promise(resolve => setTimeout(resolve, 1000));
      const user = {
        id: '2',
        username: 'webauthn-user',
        email: 'webauthn@example.com',
        role: 'user'
      };
      handleLoginSuccess(user);
      return {
        success: true,
        token: 'mock-webauthn-token',
        user
      };
    } catch (error) {
      const errorMessage = 'WebAuthn登录失败';
      setLoginResult(`登录失败：${errorMessage}`);
      setShowResult(true);
      setTimeout(() => setShowResult(false), 5000);
      throw error;
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-indigo-50 via-sky-50 to-slate-100">
      {/* 顶部导航 */}
      <nav className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center">
              <h1 className="text-xl font-semibold text-gray-900">
                企业级登录组件演示
              </h1>
            </div>
            <div className="flex items-center space-x-4">
              <button
                onClick={() => setLanguage(language === 'zh' ? 'en' : 'zh')}
                className="px-3 py-1 text-sm bg-blue-100 text-blue-700 rounded-md hover:bg-blue-200 transition-colors"
              >
                {language === 'zh' ? 'English' : '中文'}
              </button>
            </div>
          </div>
        </div>
      </nav>

      {/* 结果提示 */}
      {showResult && (
        <div className="fixed top-20 left-1/2 transform -translate-x-1/2 z-50">
          <div className={`px-6 py-3 rounded-lg shadow-lg ${
            loginResult.includes('成功') 
              ? 'bg-green-100 text-green-800 border border-green-200' 
              : 'bg-red-100 text-red-800 border border-red-200'
          }`}>
            {loginResult}
          </div>
        </div>
      )}

      <div className="flex-1 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-6xl w-full">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            {/* 左侧：功能介绍 */}
            <div className="bg-white/80 backdrop-blur-sm border border-gray-200 shadow-xl rounded-2xl p-8">
              <h2 className="text-2xl font-bold text-gray-900 mb-6">
                {language === 'zh' ? '功能特性' : 'Features'}
              </h2>
              <div className="space-y-4">
                <div className="flex items-start space-x-3">
                  <div className="w-2 h-2 bg-blue-500 rounded-full mt-2"></div>
                  <div>
                    <h3 className="font-medium text-gray-900">
                      {language === 'zh' ? '多种登录方式' : 'Multiple Login Methods'}
                    </h3>
                    <p className="text-sm text-gray-600">
                      {language === 'zh' ? '支持用户名/邮箱登录、SSO单点登录、WebAuthn生物识别' : 'Username/Email, SSO, WebAuthn biometric authentication'}
                    </p>
                  </div>
                </div>
                <div className="flex items-start space-x-3">
                  <div className="w-2 h-2 bg-green-500 rounded-full mt-2"></div>
                  <div>
                    <h3 className="font-medium text-gray-900">
                      {language === 'zh' ? '多因素认证' : 'Multi-Factor Authentication'}
                    </h3>
                    <p className="text-sm text-gray-600">
                      {language === 'zh' ? '支持TOTP验证码和WebAuthn双重认证' : 'TOTP codes and WebAuthn two-factor authentication'}
                    </p>
                  </div>
                </div>
                <div className="flex items-start space-x-3">
                  <div className="w-2 h-2 bg-purple-500 rounded-full mt-2"></div>
                  <div>
                    <h3 className="font-medium text-gray-900">
                      {language === 'zh' ? '安全特性' : 'Security Features'}
                    </h3>
                    <p className="text-sm text-gray-600">
                      {language === 'zh' ? 'CSRF保护、登录尝试限制、安全的密码处理' : 'CSRF protection, rate limiting, secure password handling'}
                    </p>
                  </div>
                </div>
                <div className="flex items-start space-x-3">
                  <div className="w-2 h-2 bg-orange-500 rounded-full mt-2"></div>
                  <div>
                    <h3 className="font-medium text-gray-900">
                      {language === 'zh' ? '无障碍访问' : 'Accessibility'}
                    </h3>
                    <p className="text-sm text-gray-600">
                      {language === 'zh' ? '完整的键盘导航、屏幕阅读器支持、ARIA标签' : 'Full keyboard navigation, screen reader support, ARIA labels'}
                    </p>
                  </div>
                </div>
              </div>

              <div className="mt-8 p-4 bg-gray-50 rounded-lg">
                <h3 className="font-medium text-gray-900 mb-2">
                  {language === 'zh' ? '测试账户' : 'Test Accounts'}
                </h3>
                <div className="text-sm text-gray-600 space-y-1">
                  <p><strong>{language === 'zh' ? '普通登录' : 'Regular Login'}:</strong> admin / password123</p>
                  <p><strong>{language === 'zh' ? 'MFA登录' : 'MFA Login'}:</strong> mfa-user / password123</p>
                  <p><strong>{language === 'zh' ? 'WebAuthn' : 'WebAuthn'}:</strong> {language === 'zh' ? '点击生物识别按钮' : 'Click biometric button'}</p>
                </div>
              </div>
            </div>

            {/* 右侧：登录组件 */}
            <div className="bg-white/80 backdrop-blur-sm border border-gray-200 shadow-xl rounded-2xl">
              <div className="p-6">
                <LoginForm
                  onLogin={handleLogin}
                  onForgotPassword={handleForgotPassword}
                  onSSOLogin={handleSSOLogin}
                  onWebAuthnLogin={handleWebAuthnLogin}
                  enableSSO={true}
                  enableMFA={true}
                  enableWebAuthn={true}
                  language={language}
                  className="w-full"
                />
              </div>
            </div>
          </div>

          {/* 底部使用说明 */}
          <div className="mt-12 bg-white/80 backdrop-blur-sm border border-gray-200 shadow-xl rounded-2xl p-8">
            <h2 className="text-xl font-bold text-gray-900 mb-4">
              {language === 'zh' ? '使用说明' : 'Usage Guide'}
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <h3 className="font-medium text-gray-900 mb-2">
                  {language === 'zh' ? '基本使用' : 'Basic Usage'}
                </h3>
                <pre className="text-sm bg-gray-100 p-3 rounded overflow-x-auto">
{`import EnterpriseLoginForm from './EnterpriseLoginForm';

<EnterpriseLoginForm
  onLogin={handleLogin}
  onForgotPassword={handleForgotPassword}
  language="zh"
  enableSSO={true}
  enableMFA={true}
/>`}
                </pre>
              </div>
              <div>
                <h3 className="font-medium text-gray-900 mb-2">
                  {language === 'zh' ? '功能测试' : 'Feature Testing'}
                </h3>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• {language === 'zh' ? '尝试输入错误密码查看验证' : 'Try wrong password to see validation'}</li>
                  <li>• {language === 'zh' ? '使用mfa-user测试多因素认证' : 'Use mfa-user to test MFA'}</li>
                  <li>• {language === 'zh' ? '点击SSO按钮测试单点登录' : 'Click SSO button to test single sign-on'}</li>
                  <li>• {language === 'zh' ? '测试WebAuthn生物识别登录' : 'Test WebAuthn biometric login'}</li>
                  <li>• {language === 'zh' ? '切换语言查看国际化' : 'Switch language to see i18n'}</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}