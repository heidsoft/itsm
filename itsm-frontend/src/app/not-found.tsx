'use client';

import React from 'react';
import { Result, Button, message } from 'antd';
import { LayoutDashboard, Home } from 'lucide-react';
import { useRouter } from 'next/navigation';

/**
 * 404 页面
 * 使用 Ant Design Result 组件，保持与系统视觉风格一致
 */
export default function NotFound() {
  const router = useRouter();

  return (
    <div className='min-h-screen flex items-center justify-center bg-gray-50'>
      <Result
        status='404'
        title='404'
        subTitle="Sorry, the page you visited does not exist."
        extra={[
          <Button
            key='dashboard'
            type='primary'
            icon={<LayoutDashboard />}
            onClick={() => {
              try {
                router.push('/dashboard');
              } catch (navErr) {
                console.error('[nav]', navErr);
                message.error('Navigation failed, please try again later');
              }
            }}
          >

            Return to dashboard
          </Button>,
          <Button
            key='home'
            icon={<Home />}
            onClick={() => {
              try {
                router.push('/');
              } catch (navErr) {
                console.error('[nav]', navErr);
                message.error('Navigation failed, please try again later');
              }
            }}
          >

            Return to homepage
          </Button>,
        ]}
      />
    </div>
  );
}
