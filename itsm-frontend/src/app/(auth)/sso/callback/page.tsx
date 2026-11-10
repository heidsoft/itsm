'use client';

import React, { useEffect, useState, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Shield, CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import { Typography, Card, ConfigProvider, Flex } from 'antd';
import { antdTheme } from '@/lib/antd-theme';
import { AuthService } from '@/lib/services/auth-service';
import { logger } from '@/lib/env';
import { AuthAPI } from '@/lib/api/auth-api';

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

  // 成功后延迟 1.5 秒展示结果页再跳转；定时器随组件卸载清理，避免卸载后仍在后台导航
  useEffect(() => {
    if (status !== 'success') return;
    const timer = setTimeout(() => router.replace('/dashboard'), 1500);
    return () => clearTimeout(timer);
  }, [status, router]);

  useEffect(() => {
    let cancelled = false;
    const processCallback = async () => {
      try {
        const code = searchParams.get('code');
        const state = searchParams.get('state');
        const provider = searchParams.get('provider') || 'default';
        const error = searchParams.get('error');
        const errorDescription = searchParams.get('error_description');

        // 处理错误情况
        if (error) {
          logger.error('SSO error:', error, errorDescription);
          setStatus('error');
          setErrorMessage(errorDescription || error);
          return;
        }

        // 缺少授权码
        if (!code) {
          setStatus('error');
          setErrorMessage('Authorization code not received, please try again');
          return;
        }

        logger.info('Handling SSO callback:', { state, provider });

        const ssoResult = await AuthAPI.ssoCallback(code, state, provider);

        if (!ssoResult.success || !ssoResult.data) {
          throw new Error(ssoResult.error || 'SSO login failed');
        }

        // 回调只负责让后端基于授权码签发会话；身份、租户范围与权限一律回后端会话端点读取，
        // 不采信回调响应里的 user/tenant/accessToken 字段。
        const outcome = await AuthService.syncSession();
        if (outcome.state === 'unauthenticated') {
          throw new Error('SSO callback has been accepted, but the backend did not establish a session, please log in again');
        }
        if (outcome.state === 'unavailable') {
          throw new Error('Unable to confirm session, please try again later');
        }

        if (cancelled) return;
        setStatus('success');
      } catch (err) {
        logger.error('SSO callback processing failed:', err);
        if (cancelled) return;
        setStatus('error');
        setErrorMessage(err instanceof Error ? err.message : 'SSO login failed');
      }
    };

    processCallback();
    return () => {
      cancelled = true;
    };
  }, [searchParams, router]);

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

                Processing SSO login
              </Title>

              <Text className="!text-gray-500 !text-sm block">Please wait, your identity is being verified...</Text>

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

                SSO login successful
              </Title>

              <Text className="!text-gray-500 !text-sm block">Jumping to dashboard...</Text>

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

              SSO login failed
            </Title>

            <Text className="!text-gray-500 !text-sm block !mb-6">
              {errorMessage || 'Error occurred during SSO login process'}
            </Text>

            <button
              onClick={() => router.push('/login')}
              className="px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors text-sm font-medium"
            >

              Return to login page
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

                  Loading...
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
