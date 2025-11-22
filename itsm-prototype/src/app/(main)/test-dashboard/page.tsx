'use client';

import React from 'react';
import { Card, Row, Col, Typography } from 'antd';
import { AIMetrics } from '@/components/business/AIMetrics';
import { SmartSLAMonitor } from '@/components/business/SmartSLAMonitor';
import { PredictiveAnalytics } from '@/components/business/PredictiveAnalytics';
import { ResourceDistributionChart, ResourceHealthPieChart } from '../dashboard/charts';

const { Title } = Typography;

export default function TestDashboardPage() {
  return (
    <div className='p-6'>
      <Title level={2} className='mb-6'>
        Dashboard组件测试页面
      </Title>

      <Row gutter={[24, 24]}>
        <Col xs={24} lg={8}>
          <Card title='AI 使用指标' className='h-full'>
            <AIMetrics />
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title='智能SLA监控' className='h-full'>
            <SmartSLAMonitor />
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title='智能预测分析' className='h-full'>
            <PredictiveAnalytics />
          </Card>
        </Col>
      </Row>

      <Row gutter={[24, 24]} className='mt-6'>
        <Col xs={24} lg={12}>
          <Card title='多云资源分布'>
            <ResourceDistributionChart />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title='资源健康状态'>
            <ResourceHealthPieChart />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
