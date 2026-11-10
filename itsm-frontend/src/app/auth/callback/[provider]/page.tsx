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
          throw new Error('Authorization failed, authorization code not obtained');
        }

        // 调用后端API完成登录
        await AuthService.thirdPartyLogin(provider, code, state);

        message.success('Login Successful');
        // 跳转到首页
        try {
          router.push('/dashboard');
        } catch (navErr) {
          console.error('[nav]', navErr);
          message.error('Navigation failed, please try again later');
        }
      } catch (err) {
        console.error('Third-party login failed:', err);
        setError(err instanceof Error ? err.message : 'Login failed, please try again later.');
        message.error('Login failed, please try again later.');
        // 3秒后跳转到登录页
        setTimeout(() => {
          try {
            router.push('/login');
          } catch (navErr) {
            console.error('[nav]', navErr);
            message.error('Navigation failed, please try again later');
          }
        }, 3000);
      }
    };

    handleCallback();
  }, [provider, router]);

  if (error) {
    return (
      <div className='flex flex-col items-center justify-center min-h-screen'>
        <p className='text-red-500 mb-4'>{error}</p>
        <p className='text-gray-500'>Jumping to the login page...</p>
      </div>
    );
  }

  return (
    <div className='flex flex-col items-center justify-center min-h-screen'>
      <Spin size='large' className='mb-4' />
      <p className='text-gray-600'>Completing login, please wait...</p>
    </div>
  );
}
