'use client';

import React from 'react';
import { Card, Typography, Alert } from 'antd';

const { Title, Text } = Typography;

export default function TicketTypesPage() {
  return (
    <div className='max-w-4xl mx-auto p-6'>
      <Card>
        <Title level={2} style={{ marginBottom: 0 }}>
          工单类型
        </Title>
        <Text type='secondary'>P0 构建稳定版：该页面暂时降级为占位，后续恢复类型/字段/审批等配置。</Text>
        <div className='mt-4'>
          <Alert
            type='info'
            showIcon
            message='功能待恢复'
            description='当前版本为了清零 TS 阻塞，移除了对 legacy business 组件链路的依赖。'
          />
        </div>
      </Card>
    </div>
  );
}


