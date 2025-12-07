/**
 * 审批链统计卡片组件
 */

import React from 'react';
import { Card, Row, Col, Statistic, Typography } from 'antd';
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  UserOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { ApprovalChainStats } from '@/types/approval-chain';

const { Title } = Typography;

interface ApprovalChainStatsProps {
  stats: ApprovalChainStats;
  loading?: boolean;
}

export function ApprovalChainStatsCards({ stats, loading = false }: ApprovalChainStatsProps) {
  const statCards = [
    {
      title: '总审批链数',
      value: stats.total,
      icon: <SettingOutlined className='text-blue-500' />,
      color: '#1890ff',
    },
    {
      title: '活跃审批链',
      value: stats.active,
      icon: <CheckCircleOutlined className='text-green-500' />,
      color: '#52c41a',
    },
    {
      title: '非活跃审批链',
      value: stats.inactive,
      icon: <ClockCircleOutlined className='text-orange-500' />,
      color: '#faad14',
    },
    {
      title: '平均步骤数',
      value: stats.avgStepsPerChain.toFixed(1),
      icon: <UserOutlined className='text-purple-500' />,
      color: '#722ed1',
    },
  ];

  return (
    <div className='mb-6'>
      <Title level={4} className='mb-4'>
        审批链统计
      </Title>
      <Row gutter={[16, 16]}>
        {statCards.map((card, index) => (
          <Col xs={24} sm={12} lg={6} key={index}>
            <Card loading={loading} className='h-full'>
              <Statistic
                title={card.title}
                value={card.value}
                prefix={card.icon}
                valueStyle={{ color: card.color }}
              />
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  );
}
