'use client';

import React from 'react';
import { Card, Row, Col, Skeleton } from 'antd';
import {
  FileTextOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';

interface TicketStatsProps {
  stats: {
    total: number;
    open: number;
    resolved: number;
    highPriority: number;
  };
  loading?: boolean;
}

interface StatCardProps {
  title: string;
  value: number;
  icon: React.ReactNode;
  color: string;
  bgColor: string;
}

const StatCard: React.FC<StatCardProps> = React.memo(({ title, value, icon, color, bgColor }) => (
  <Col xs={24} sm={12} md={6} lg={6} xl={6}>
    <Card
      className={`border-0 shadow-sm hover:shadow-md transition-all duration-300 bg-gradient-to-br ${bgColor}`}
      styles={{ body: { padding: '16px' } }}
    >
      <div className='flex items-center justify-between'>
        <div>
          <div className='text-xs text-gray-600 mb-1 font-medium'>{title}</div>
          <div className={`text-xl font-bold ${color}`}>{value.toLocaleString()}</div>
        </div>
        <div
          className={`w-10 h-10 ${color.replace(
            'text-',
            'bg-'
          )} rounded-lg flex items-center justify-center shadow-sm`}
          style={{ opacity: 0.9 }}
        >
          <span className="text-white text-xl">{icon}</span>
        </div>
      </div>
    </Card>
  </Col>
));

StatCard.displayName = 'StatCard';

const TicketStatsSkeleton: React.FC = () => (
    <Row gutter={[12, 12]}>
        {Array.from({ length: 4 }).map((_, index) => (
            <Col key={index} xs={24} sm={12} md={6} lg={6} xl={6}>
                <Card className='border-0 shadow-sm' styles={{ body: { padding: '16px' } }}>
                    <div className='flex items-center justify-between'>
                        <div>
                            <Skeleton.Input active style={{ width: '80px', height: '16px' }} />
                            <Skeleton.Input active style={{ width: '50px', height: '24px', marginTop: '8px' }} />
                        </div>
                        <Skeleton.Avatar active size='large' shape='square' />
                    </div>
                </Card>
            </Col>
        ))}
    </Row>
);

export const TicketStats: React.FC<TicketStatsProps> = React.memo(({ stats, loading }) => {
    if (loading) {
        return <TicketStatsSkeleton />;
    }

  const statCards = [
    {
      title: '总工单数',
      value: stats.total,
      icon: <FileTextOutlined />,
      color: 'text-blue-600',
      bgColor: 'from-blue-50 to-blue-100',
    },
    {
      title: '待处理',
      value: stats.open,
      icon: <ClockCircleOutlined />,
      color: 'text-orange-600',
      bgColor: 'from-orange-50 to-orange-100',
    },
    {
      title: '已解决',
      value: stats.resolved,
      icon: <CheckCircleOutlined />,
      color: 'text-green-600',
      bgColor: 'from-green-50 to-green-100',
    },
    {
      title: '高优先级',
      value: stats.highPriority,
      icon: <ExclamationCircleOutlined />,
      color: 'text-red-600',
      bgColor: 'from-red-50 to-red-100',
    },
  ];

  return (
    <div className='mb-4'>
      <Row gutter={[12, 12]}>
        {statCards.map((card, index) => (
          <StatCard
            key={index}
            title={card.title}
            value={card.value}
            icon={card.icon}
            color={card.color}
            bgColor={card.bgColor}
          />
        ))}
      </Row>
    </div>
  );
});

TicketStats.displayName = 'TicketStats';
