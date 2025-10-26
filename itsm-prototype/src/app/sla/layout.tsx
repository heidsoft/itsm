import React from 'react';
import { Button } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import AppLayout from '@/components/layout/AppLayout';

export default function SLALayout({ children }: { children: React.ReactNode }) {
  return (
    <AppLayout
      title='SLA管理'
      description='服务级别协议管理，符合ITIL 4.0标准'
      showPageHeader={true}
      extra={
        <Button type='primary' icon={<PlusOutlined />}>
          创建SLA
        </Button>
      }
    >
      {children}
    </AppLayout>
  );
}
