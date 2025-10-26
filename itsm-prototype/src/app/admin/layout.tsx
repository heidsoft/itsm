'use client';

import React from 'react';
import { Button } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import AppLayout from '@/components/layout/AppLayout';

interface AdminLayoutProps {
  children: React.ReactNode;
}

const AdminLayout = ({ children }: AdminLayoutProps) => {
  return (
    <AppLayout
      title='系统管理'
      description='系统配置和管理功能'
      breadcrumb={[{ title: '系统管理' }]}
      showPageHeader={true}
      extra={
        <Button type='primary' icon={<PlusOutlined />}>
          创建
        </Button>
      }
    >
      {children}
    </AppLayout>
  );
};

export default AdminLayout;
