'use client';

import React, { useEffect, useState, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Shield, CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import { Typography, Card, ConfigProvider, message, Flex } from 'antd';
import { antdTheme } from '@/lib/antd-theme';
import { AuthService } from '@/lib/services/auth-service';
import { useAuthStore } from '@/lib/store/auth-store';
import { logger } from '@/lib/env';
import { API_BASE_URL, Tenant } from '@/lib/api/api-config';

const { Text, Title } = Typography;

/**
 * SSO回调内容组件
 * 包含useSearchParams使用，需要Suspense包装
 */
function SSOCallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [errorMessage, setErrorMessage] = useState('');

  const { login } = useAuthStore();

  useEffect(() => {
    const processCallback = async () => {
      try {
        const code = searchParams.get('code');
        const state = searchParams.get('state');
        const error = searchParams.get('error');
        const errorDescription = searchParams.get('error_description');

        // 处理错误情况
        if (error) {
          logger.error('SSO错误:', error, errorDescription);
          setStatus('error');
          setErrorMessage(errorDescription || error);
          return;
        }

        // 缺少授权码
        if (!code) {
          setStatus('error');
          setErrorMessage('未收到授权码，请重试');
          return;
        }

        logger.info('处理SSO回调:', { state });

        // 模拟SSO回调处理
        // 实际应调用后端 /api/v1/auth/sso/callback 端点
        await new Promise(resolve => setTimeout(resolve, 1500));

        // 模拟获取用户信息和token
        const mockUser = {
          id: 100,
          username: 'sso_user',
          name: 'SSO 用户',
          email: 'sso@example.com',
          role: 'admin',
          department: 'IT部门',
        };

        const mockAccessToken = 'sso_mock_token_' + Date.now();

        // 存储token
        AuthService.setTokens(mockAccessToken, 'mock_refresh_token');

        // 更新store状态
        login(
          {
            id: mockUser.id,
            username: mockUser.username,
            role: mockUser.role,
            email: mockUser.email,
            name: mockUser.name,
            tenantId: 1,
            department: mockUser.department,
            permissions: ['admin', 'ticket.*', 'incident.*', 'change.*', 'problem.*'],
          },
          mockAccessToken,
          {
            id: 1,
            name: '默认租户',
            code: 'default',
            type: 'standard',
            status: 'active',
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
          } as Tenant
        );

        setStatus('success');
        message.success('SSO登录成功');

        // 跳转到仪表盘
        setTimeout(() => {
          router.push('/dashboard');
        }, 1500);
      } catch (err) {
        logger.error('SSO回调处理失败:', err);
        setStatus('error');
        setErrorMessage(err instanceof Error ? err.message : 'SSO登录失败');
      }
    };

    processCallback();
  }, [searchParams, router, login]);

  if (status === 'loading') {
    return (
      <ConfigProvider theme={antdTheme}>
        <div className="min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50">
          <Card
            className="rounded-xl shadow-xl border-none w-full max-w-[400px]"
            styles={{ body: { padding: '48px' } }}
          >
            <div className="text-center">
              <div className="w-16 h-16 mx-auto mb-4 bg-blue-100 rounded-full flex items-center justify-center">
                <Loader2 size={32} className="text-blue-600 animate-spin" />
              </div>

              <Title level={3} className="!mb-2 !text-gray-900 !text-xl">
                正在处理SSO登录
              </Title>

              <Text className="!text-gray-500 !text-sm block">请稍候，正在验证您的身份...</Text>

              <Flex justify="center" className="mt-6">
                <div className="flex gap-2">
                  <div
                    className="w-2 h-2 bg-blue-600 rounded-full animate-bounce"
                    style={{ animationDelay: '0ms' }}
                  />
                  <div
                    className="w-2 h-2 bg-blue-600 rounded-full animate-bounce"
                    style={{ animationDelay: '150ms' }}
                  />
                  <div
                    className="w-2 h-2 bg-blue-600 rounded-full animate-bounce"
                    style={{ animationDelay: '300ms' }}
                  />
                </div>
              </Flex>
            </div>
          </Card>
        </div>
      </ConfigProvider>
    );
  }

  if (status === 'success') {
    return (
      <ConfigProvider theme={antdTheme}>
        <div className="min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50">
          <Card
            className="rounded-xl shadow-xl border-none w-full max-w-[400px]"
            styles={{ body: { padding: '48px' } }}
          >
            <div className="text-center">
              <div className="w-16 h-16 mx-auto mb-4 bg-green-100 rounded-full flex items-center justify-center">
                <CheckCircle size={32} className="text-green-600" />
              </div>

              <Title level={3} className="!mb-2 !text-gray-900 !text-xl">
                SSO登录成功
              </Title>

              <Text className="!text-gray-500 !text-sm block">正在跳转到仪表盘...</Text>

              <Flex justify="center" className="mt-6">
                <div className="flex gap-2">
                  <div
                    className="w-2 h-2 bg-green-600 rounded-full animate-bounce"
                    style={{ animationDelay: '0ms' }}
                  />
                  <div
                    className="w-2 h-2 bg-green-600 rounded-full animate-bounce"
                    style={{ animationDelay: '150ms' }}
                  />
                  <div
                    className="w-2 h-2 bg-green-600 rounded-full animate-bounce"
                    style={{ animationDelay: '300ms' }}
                  />
                </div>
              </Flex>
            </div>
          </Card>
        </div>
      </ConfigProvider>
    );
  }

  // Error state
  return (
    <ConfigProvider theme={antdTheme}>
      <div className="min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50">
        <Card
          className="rounded-xl shadow-xl border-none w-full max-w-[400px]"
          styles={{ body: { padding: '48px' } }}
        >
          <div className="text-center">
            <div className="w-16 h-16 mx-auto mb-4 bg-red-100 rounded-full flex items-center justify-center">
              <AlertCircle size={32} className="text-red-600" />
            </div>

            <Title level={3} className="!mb-2 !text-gray-900 !text-xl">
              SSO登录失败
            </Title>

            <Text className="!text-gray-500 !text-sm block !mb-6">
              {errorMessage || 'SSO登录过程中出现错误'}
            </Text>

            <button
              onClick={() => router.push('/login')}
              className="px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors text-sm font-medium"
            >
              返回登录页
            </button>
          </div>
        </Card>
      </div>
    </ConfigProvider>
  );
}

/**
 * SSO回调页面组件
 * 处理OAuth2/SAML等SSO登录后的回调
 */
export default function SSOCallbackPage() {
  return (
    <Suspense
      fallback={
        <ConfigProvider theme={antdTheme}>
          <div className="min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50">
            <Card
              className="rounded-xl shadow-xl border-none w-full max-w-[400px]"
              styles={{ body: { padding: '48px' } }}
            >
              <div className="text-center">
                <div className="w-16 h-16 mx-auto mb-4 bg-blue-100 rounded-full flex items-center justify-center">
                  <Loader2 size={32} className="text-blue-600 animate-spin" />
                </div>
                <Title level={3} className="!mb-2 !text-gray-900 !text-xl">
                  正在加载...
                </Title>
              </div>
            </Card>
          </div>
        </ConfigProvider>
      }
    >
      <SSOCallbackContent />
    </Suspense>
  );
}
