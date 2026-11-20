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
  // httpClient 在请求中止/超时降级路径会返回 null（见 http-client.ts 的 AbortError 分支），
  // 此时 stats 可能为 null 或字段缺失；直接 stats.avgStepsPerChain.toFixed(1) 会抛
  // TypeError 导致整页白屏。渲染层一律走数值兜底，空值显示 0.0 而不是崩溃。
  const safeStats = stats ?? ({} as Partial<ApprovalChainStats>);
  const avgSteps = Number(safeStats.avgStepsPerChain);
  const statCards = [
    {
      title: '总审批链数',
      value: safeStats.total ?? 0,
      icon: <Settings className="w-5 h-5 text-blue-500" />,
      color: '#1890ff',
    },
    {
      title: '活跃审批链',
      value: safeStats.active ?? 0,
      icon: <CheckCircle className="w-5 h-5 text-green-500" />,
      color: '#52c41a',
    },
    {
      title: '非活跃审批链',
      value: safeStats.inactive ?? 0,
      icon: <Clock className="w-5 h-5 text-orange-500" />,
      color: '#faad14',
    },
    {
      title: '平均步骤数',
      value: (Number.isFinite(avgSteps) ? avgSteps : 0).toFixed(1),
      icon: <Users className="w-5 h-5 text-purple-500" />,
      color: '#722ed1',
    },
  ];

  return (
    <div className="mb-6">
      <Title level={4} className="mb-4">
        审批链统计
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
