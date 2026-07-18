'use client';

import React, { useEffect, useState, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Shield, CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import { Typography, Card, ConfigProvider, message, Flex } from 'antd';
import { antdTheme } from '@/lib/antd-theme';
import { AuthService } from '@/lib/services/auth-service';
import { useAuthStore } from '@/lib/store/auth-store';
import { logger } from '@/lib/env';
import type { Tenant } from '@/lib/api/api-config';
import { API_BASE_URL } from '@/lib/api/api-config';

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
        const provider = searchParams.get('provider') || 'default';
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

        logger.info('处理SSO回调:', { state, provider });

        // ⚠️ 安全修复：禁止在前端硬编码任何 SSO 用户、token、tenantId、permissions。
        // 必须由后端 /api/v1/auth/sso/callback 基于 OAuth/SAML code 进行真实校验后下发会话。
        // 在开源版本尚未对接 SSO 提供商时，此路由必须显示"SSO 暂未启用"并禁止本地登录。
        const ssoCallbackResponse = await fetch(
          `${API_BASE_URL}/api/v1/auth/sso/callback?provider=${encodeURIComponent(provider)}`,
          {
            method: 'POST',
            credentials: 'include',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ code, state }),
          }
        );

        if (!ssoCallbackResponse.ok) {
          throw new Error(
            ssoCallbackResponse.status === 404 || ssoCallbackResponse.status === 501
              ? 'SSO 登录暂未启用，请使用账号密码登录'
              : 'SSO 登录失败，请稍后重试'
          );
        }

        const ssoCallbackData = await ssoCallbackResponse.json();
        if (ssoCallbackData?.code !== 0 || !ssoCallbackData?.data) {
          throw new Error(ssoCallbackData?.message || 'SSO 登录失败');
        }

        const { user, tenant, accessToken } = ssoCallbackData.data as {
          user: {
            id: number;
            username: string;
            role?: string;
            email?: string;
            name?: string;
            tenantId?: number;
            department?: string;
            permissions?: string[];
          };
          tenant: Tenant;
          accessToken: string;
        };

        // 后端 SSO 响应里 email/name 可能缺失，这里兜底默认值
        const safeUser = {
          id: user.id,
          username: user.username,
          email: user.email ?? '',
          name: user.name ?? user.username,
          tenantId: user.tenantId,
          role: user.role,
          department: user.department,
          permissions: user.permissions,
        };

        // 设置 auth-token cookie 供 middleware 路由守卫使用（与 AuthService.login 保持一致）
        if (typeof window !== 'undefined' && accessToken) {
          document.cookie = `auth-token=${accessToken}; path=/; SameSite=Lax`;
        }

        login(safeUser, accessToken, tenant);

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
