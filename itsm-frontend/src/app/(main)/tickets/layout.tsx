'use client';

import React, { PropsWithChildren } from 'react';
import { Breadcrumb } from 'antd';
import Link from 'next/link';
import { HomeOutlined } from '@ant-design/icons';

const TicketLayout: React.FC<PropsWithChildren> = ({ children }) => {
  const breadcrumbItems = [
    {
      title: (
        <Link href="/">
          <HomeOutlined />
        </Link>
      ),
    },
    {
      title: (
        <Link href="/tickets">
          工单管理
        </Link>
      ),
    },
  ];

  return (
    <div className="min-h-screen bg-gray-50">
      {/* 面包屑导航 */}
      <div className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-6 py-3">
          <Breadcrumb items={breadcrumbItems} />
        </div>
      </div>

      {/* 页面内容 */}
      {children}
    </div>
  );
};

export default TicketLayout;
