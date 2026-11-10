/**
 * 审批链统计卡片组件
 */

import React from 'react';
import { Card, Row, Col, Statistic, Typography } from 'antd';
import {
  CheckCircle,
  Clock,
  Users,
  Settings,
} from 'lucide-react';
import type { ApprovalChainStats } from '@/types/approval-chain';

const { Title } = Typography;

interface ApprovalChainStatsProps {
  stats: ApprovalChainStats;
  loading?: boolean;
}

export function ApprovalChainStatsCards({ stats, loading = false }: ApprovalChainStatsProps) {
  const statCards = [
    {
      title: 'Total number of approval chains',
      value: stats.total,
      icon: <Settings className="w-5 h-5 text-blue-500" />,
      color: '#1890ff',
    },
    {
      title: 'Active approval chain',
      value: stats.active,
      icon: <CheckCircle className="w-5 h-5 text-green-500" />,
      color: '#52c41a',
    },
    {
      title: 'Inactive approval chain',
      value: stats.inactive,
      icon: <Clock className="w-5 h-5 text-orange-500" />,
      color: '#faad14',
    },
    {
      title: 'average number of steps',
      value: stats.avgStepsPerChain.toFixed(1),
      icon: <Users className="w-5 h-5 text-purple-500" />,
      color: '#722ed1',
    },
  ];

  return (
    <div className="mb-6">
      <Title level={4} className="mb-4">

        Approval chain statistics
      </Title>
      <Row gutter={[16, 16]}>
        {statCards.map((card, index) => (
          <Col xs={24} sm={12} lg={6} key={index}>
            <Card loading={loading} className="h-full">
              <Statistic
                title={card.title}
                value={card.value}
                prefix={card.icon}
                styles={{ content: { color: card.color } }}
              />
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  );
}
