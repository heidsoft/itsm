'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { AuthService } from '@/lib/services/auth-service';
import { Card, Typography, Spin, Alert, Button } from 'antd';
import { Shield, RefreshCw, ArrowRight, CheckCircle, AlertCircle } from 'lucide-react';

const { Title, Text } = Typography;

export default function Home() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [authStatus, setAuthStatus] = useState<'checking' | 'authenticated' | 'unauthenticated'>(
    'checking'
  );

  useEffect(() => {
    const checkAuthAndRedirect = async () => {
      try {
        setLoading(true);
        setError(null);

        // 模拟网络延迟，避免闪烁
        await new Promise(resolve => setTimeout(resolve, 800));

        if (AuthService.isAuthenticated()) {
          setAuthStatus('authenticated');
          // 延迟重定向，让用户看到成功状态
          setTimeout(() => {
            router.push('/dashboard');
          }, 1000);
        } else {
          setAuthStatus('unauthenticated');
          // 延迟重定向，让用户看到状态
          setTimeout(() => {
            router.push('/login');
          }, 1000);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : '认证检查失败');
        setAuthStatus('unauthenticated');
      } finally {
        setLoading(false);
      }
    };

    checkAuthAndRedirect();
  }, [router]);

  const handleRetry = () => {
    setLoading(true);
    setError(null);
    setAuthStatus('checking');
    // 重新检查认证状态
    setTimeout(() => {
      if (AuthService.isAuthenticated()) {
        setAuthStatus('authenticated');
        setTimeout(() => router.push('/dashboard'), 1000);
      } else {
        setAuthStatus('unauthenticated');
        setTimeout(() => router.push('/login'), 1000);
      }
      setLoading(false);
    }, 800);
  };

  const handleManualRedirect = (path: string) => {
    router.push(path);
  };

  // 渲染加载状态
  if (loading) {
    return (
      <div className='min-h-screen bg-gradient-to-br from-blue-50 via-white to-indigo-50 flex items-center justify-center p-6'>
        <Card
          className='w-full max-w-md text-center shadow-2xl border-0'
          style={{ borderRadius: '20px' }}
        >
          <div className='mb-6'>
            <div className='w-20 h-20 mx-auto mb-4 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-2xl flex items-center justify-center shadow-lg'>
              <Shield className='w-10 h-10 text-white' />
            </div>
            <Title
              level={2}
              className='mb-2 bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent'
            >
              ITSM Platform
            </Title>
            <Text className='text-gray-600'>IT服务管理平台</Text>
          </div>

          <div className='mb-6'>
            <Spin size='large' />
            <div className='mt-4'>
              <Text className='text-gray-500'>正在检查认证状态...</Text>
            </div>
          </div>

          <div className='space-y-3'>
            <div className='flex items-center justify-center space-x-2 text-sm text-gray-500'>
              <div className='w-2 h-2 bg-blue-400 rounded-full animate-pulse'></div>
              <span>验证用户身份</span>
            </div>
            <div className='flex items-center justify-center space-x-2 text-sm text-gray-500'>
              <div className='w-2 h-2 bg-indigo-400 rounded-full animate-pulse'></div>
              <span>检查权限状态</span>
            </div>
            <div className='flex items-center justify-center space-x-2 text-sm text-gray-500'>
              <div className='w-2 h-2 bg-purple-400 rounded-full animate-pulse'></div>
              <span>准备重定向</span>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  // 渲染错误状态
  if (error) {
    return (
      <div className='min-h-screen bg-gradient-to-br from-red-50 via-white to-pink-50 flex items-center justify-center p-6'>
        <Card
          className='w-full max-w-md text-center shadow-2xl border-0'
          style={{ borderRadius: '20px' }}
        >
          <div className='mb-6'>
            <div className='w-20 h-20 mx-auto mb-4 bg-gradient-to-br from-red-500 to-pink-600 rounded-2xl flex items-center justify-center shadow-lg'>
              <AlertCircle className='w-10 h-10 text-white' />
            </div>
            <Title level={2} className='mb-2 text-red-600'>
              认证检查失败
            </Title>
            <Text className='text-gray-600'>无法验证用户身份</Text>
          </div>

          <Alert description={error} type='error' showIcon className='mb-6' />

          <div className='space-y-3'>
            <Button
              type='primary'
              size='large'
              icon={<RefreshCw />}
              onClick={handleRetry}
              className='w-full'
            >
              重试
            </Button>
            <Button
              size='large'
              icon={<ArrowRight />}
              onClick={() => handleManualRedirect('/login')}
              className='w-full'
            >
              前往登录页面
            </Button>
          </div>
        </Card>
      </div>
    );
  }

  // 渲染认证状态
  if (authStatus === 'authenticated') {
    return (
      <div className='min-h-screen bg-gradient-to-br from-green-50 via-white to-emerald-50 flex items-center justify-center p-6'>
        <Card
          className='w-full max-w-md text-center shadow-2xl border-0'
          style={{ borderRadius: '20px' }}
        >
          <div className='mb-6'>
            <div className='w-20 h-20 mx-auto mb-4 bg-gradient-to-br from-green-500 to-emerald-600 rounded-2xl flex items-center justify-center shadow-lg'>
              <CheckCircle className='w-10 h-10 text-white' />
            </div>
            <Title level={2} className='mb-2 text-green-600'>
              认证成功
            </Title>
            <Text className='text-gray-600'>正在跳转到仪表盘...</Text>
          </div>

          <div className='mb-6'>
            <Spin size='large' />
            <div className='mt-4'>
              <Text className='text-gray-500'>准备中...</Text>
            </div>
          </div>

          <div className='space-y-3'>
            <div className='flex items-center justify-center space-x-2 text-sm text-green-500'>
              <CheckCircle className='w-4 h-4' />
              <span>用户身份已验证</span>
            </div>
            <div className='flex items-center justify-center space-x-2 text-sm text-green-500'>
              <CheckCircle className='w-4 h-4' />
              <span>权限检查通过</span>
            </div>
            <div className='flex items-center justify-center space-x-2 text-sm text-blue-500'>
              <div className='w-2 h-2 bg-blue-400 rounded-full animate-pulse'></div>
              <span>即将跳转</span>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  // 渲染未认证状态
  return (
    <div className='min-h-screen bg-gradient-to-br from-orange-50 via-white to-amber-50 flex items-center justify-center p-6'>
      <Card
        className='w-full max-w-md text-center shadow-2xl border-0'
        style={{ borderRadius: '20px' }}
      >
        <div className='mb-6'>
          <div className='w-20 h-20 mx-auto mb-4 bg-gradient-to-br from-orange-500 to-amber-600 rounded-2xl flex items-center justify-center shadow-lg'>
            <Shield className='w-10 h-10 text-white' />
          </div>
          <Title level={2} className='mb-2 text-orange-600'>
            需要登录
          </Title>
          <Text className='text-gray-600'>请先登录以访问系统</Text>
        </div>

        <div className='mb-6'>
          <div className='w-16 h-16 mx-auto mb-4'>
            <div className='w-full h-full border-4 border-orange-200 border-t-orange-500 rounded-full animate-spin'></div>
          </div>
          <div className='mt-4'>
            <Text className='text-gray-500'>即将跳转到登录页面...</Text>
          </div>
        </div>

        <div className='space-y-3'>
          <div className='flex items-center justify-center space-x-2 text-sm text-orange-500'>
            <AlertCircle className='w-4 h-4' />
            <span>用户未认证</span>
          </div>
          <div className='flex items-center justify-center space-x-2 text-sm text-orange-500'>
            <AlertCircle className='w-4 h-4' />
            <span>需要重新登录</span>
          </div>
          <div className='flex items-center justify-center space-x-2 text-sm text-blue-500'>
            <div className='w-2 h-2 bg-blue-400 rounded-full animate-pulse'></div>
            <span>即将跳转</span>
          </div>
        </div>

        <div className='mt-6 pt-4 border-t border-gray-200'>
          <Button
            type='primary'
            size='large'
            icon={<ArrowRight />}
            onClick={() => handleManualRedirect('/login')}
            className='w-full'
          >
            立即前往登录
          </Button>
        </div>
      </Card>
    </div>
  );
}
