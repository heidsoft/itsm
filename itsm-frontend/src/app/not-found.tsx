'use client';

import React from 'react';
import { Result, Button } from 'antd';
import { HomeOutlined, DashboardOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';

/**
 * 404 页面
 * 使用 Ant Design Result 组件，保持与系统视觉风格一致
 */
export default function NotFound() {
  const router = useRouter();

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <Result
        status="404"
        title="404"
        subTitle="抱歉，您访问的页面不存在。"
        extra={[
          <Button
            key="dashboard"
            type="primary"
            icon={<DashboardOutlined />}
            onClick={() => router.push('/dashboard')}
          >
            返回仪表盘
          </Button>,
          <Button key="home" icon={<HomeOutlined />} onClick={() => router.push('/')}>
            返回首页
          </Button>,
        ]}
      />
    </div>
  );
}
