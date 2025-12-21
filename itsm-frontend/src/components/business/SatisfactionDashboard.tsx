'use client';

import React from 'react';
import { Card, Row, Col, Statistic, Progress, Skeleton } from 'antd';
import { Star, Users, ThumbsUp, MessageSquare } from 'lucide-react';
import { useSatisfactionData } from './hooks/useSatisfactionData';

const SatisfactionDashboardSkeleton: React.FC = () => (
  <Card title='客户满意度仪表盘' className='shadow-sm'>
    <Row gutter={[16, 16]}>
      {Array.from({ length: 4 }).map((_, index) => (
        <Col key={index} xs={24} sm={12} lg={6}>
          <Card className='text-center'>
            <Skeleton active paragraph={{ rows: 2 }} />
          </Card>
        </Col>
      ))}
    </Row>
  </Card>
);

const SatisfactionDashboard: React.FC = () => {
  const { metrics, loading } = useSatisfactionData();

  const getSatisfactionColor = (score: number) => {
    if (score >= 4.5) return '#52c41a';
    if (score >= 4.0) return '#1890ff';
    if (score >= 3.5) return '#faad14';
    return '#ff4d4f';
  };

  const getSatisfactionText = (score: number) => {
    if (score >= 4.5) return '非常满意';
    if (score >= 4.0) return '满意';
    if (score >= 3.5) return '一般';
    return '不满意';
  };

  if (loading || !metrics) {
    return <SatisfactionDashboardSkeleton />;
  }

  return (
    <div className='space-y-4'>
      <Card title='客户满意度仪表盘' className='shadow-sm'>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <Card className='text-center'>
              <Statistic
                title='总体满意度'
                value={metrics.overall}
                precision={1}
                suffix='/ 5.0'
                valueStyle={{ color: getSatisfactionColor(metrics.overall) }}
                prefix={<Star className='w-5 h-5' />}
              />
              <Progress
                percent={(metrics.overall / 5) * 100}
                strokeColor={getSatisfactionColor(metrics.overall)}
                size='small'
                showInfo={false}
                style={{ marginTop: 8 }}
              />
              <div className='text-sm text-gray-600 mt-1'>
                {getSatisfactionText(metrics.overall)}
              </div>
            </Card>
          </Col>

          <Col xs={24} sm={12} lg={6}>
            <Card className='text-center'>
              <Statistic
                title='响应时间满意度'
                value={metrics.responseTime}
                precision={1}
                suffix='/ 5.0'
                valueStyle={{ color: getSatisfactionColor(metrics.responseTime) }}
                prefix={<MessageSquare className='w-5 h-5' />}
              />
              <Progress
                percent={(metrics.responseTime / 5) * 100}
                strokeColor={getSatisfactionColor(metrics.responseTime)}
                size='small'
                showInfo={false}
                style={{ marginTop: 8 }}
              />
            </Card>
          </Col>

          <Col xs={24} sm={12} lg={6}>
            <Card className='text-center'>
              <Statistic
                title='解决质量满意度'
                value={metrics.resolutionQuality}
                precision={1}
                suffix='/ 5.0'
                valueStyle={{ color: getSatisfactionColor(metrics.resolutionQuality) }}
                prefix={<ThumbsUp className='w-5 h-5' />}
              />
              <Progress
                percent={(metrics.resolutionQuality / 5) * 100}
                strokeColor={getSatisfactionColor(metrics.resolutionQuality)}
                size='small'
                showInfo={false}
                style={{ marginTop: 8 }}
              />
            </Card>
          </Col>

          <Col xs={24} sm={12} lg={6}>
            <Card className='text-center'>
              <Statistic
                title='沟通满意度'
                value={metrics.communication}
                precision={1}
                suffix='/ 5.0'
                valueStyle={{ color: getSatisfactionColor(metrics.communication) }}
                prefix={<Users className='w-5 h-5' />}
              />
              <Progress
                percent={(metrics.communication / 5) * 100}
                strokeColor={getSatisfactionColor(metrics.communication)}
                size='small'
                showInfo={false}
                style={{ marginTop: 8 }}
              />
            </Card>
          </Col>
        </Row>

        <div className='mt-4 text-center text-gray-600'>
          <span>
            基于 {metrics.totalResponses} 份反馈数据
          </span>
        </div>
      </Card>
    </div>
  );
};

export { SatisfactionDashboard };
