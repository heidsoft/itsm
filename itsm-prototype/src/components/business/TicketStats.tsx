'use client';

import React from 'react';
import { Card, Row, Col } from 'antd';
import { FileText, Clock, CheckCircle, AlertTriangle } from 'lucide-react';

interface TicketStatsProps {
  stats: {
    total: number;
    open: number;
    resolved: number;
    highPriority: number;
  };
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
          {React.cloneElement(icon as React.ReactElement, { size: 20 })}
        </div>
      </div>
    </Card>
  </Col>
));

StatCard.displayName = 'StatCard';

export const TicketStats: React.FC<TicketStatsProps> = React.memo(({ stats }) => {
  const statCards = [
    {
      title: '总工单数',
      value: stats.total,
      icon: <FileText size={20} className='text-white' />,
      color: 'text-blue-600',
      bgColor: 'from-blue-50 to-blue-100',
    },
    {
      title: '待处理',
      value: stats.open,
      icon: <Clock size={20} className='text-white' />,
      color: 'text-orange-600',
      bgColor: 'from-orange-50 to-orange-100',
    },
    {
      title: '已解决',
      value: stats.resolved,
      icon: <CheckCircle size={20} className='text-white' />,
      color: 'text-green-600',
      bgColor: 'from-green-50 to-green-100',
    },
    {
      title: '高优先级',
      value: stats.highPriority,
      icon: <AlertTriangle size={20} className='text-white' />,
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
