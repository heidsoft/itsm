'use client';

import React, { useEffect, useState } from 'react';
import { Spin, message } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { AuthService } from '@/lib/services/auth-service';

export default function AuthCallbackPage() {
  const params = useParams();
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const provider = params.provider as string;

  useEffect(() => {
    const handleCallback = async () => {
      try {
        // 获取URL中的code参数
        const urlParams = new URLSearchParams(window.location.search);
        const code = urlParams.get('code');
        const state = urlParams.get('state');

        if (!code) {
          throw new Error('授权失败，未获取到授权码');
        }

        // 调用后端API完成登录
        await AuthService.thirdPartyLogin(provider, code, state);

        message.success('登录成功');
        // 跳转到首页
        router.push('/dashboard');
      } catch (err) {
        console.error('第三方登录失败:', err);
        setError(err instanceof Error ? err.message : '登录失败，请稍后重试');
        message.error('登录失败，请稍后重试');
        // 3秒后跳转到登录页
        setTimeout(() => {
          router.push('/login');
        }, 3000);
      }
    };

    handleCallback();
  }, [provider, router]);

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen">
        <p className="text-red-500 mb-4">{error}</p>
        <p className="text-gray-500">正在跳转到登录页面...</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-screen">
      <Spin size="large" className="mb-4" />
      <p className="text-gray-600">正在完成登录，请稍候...</p>
    </div>
  );
}
