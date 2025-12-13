'use client';

import React from 'react';
import { Card, Typography, Space, Button } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import Link from 'next/link';
import TicketList from '@/components/ticket/TicketList';

const { Title, Text } = Typography;

export default function TicketsPage() {
  return (
    <div className='max-w-7xl mx-auto p-6'>
      <Space direction='vertical' size={16} style={{ width: '100%' }}>
        <Card>
          <div className='flex items-center justify-between'>
            <div>
              <Title level={2} style={{ marginBottom: 0 }}>
                工单
              </Title>
              <Text type='secondary'>轻量列表页（P0 构建稳定版），高级看板/分析后续逐步恢复</Text>
            </div>
            <Link href='/tickets/create'>
              <Button type='primary' icon={<PlusOutlined />}>
                新建工单
              </Button>
            </Link>
          </div>
        </Card>

        <TicketList />
      </Space>
    </div>
  );
}


