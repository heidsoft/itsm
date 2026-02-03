'use client';

import React from 'react';
import { Card, Col, Progress, Row, Statistic } from 'antd';
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  FileTextOutlined,
  StarOutlined,
} from '@ant-design/icons';

interface ReportMetrics {
  totalTickets: number;
  resolvedTickets: number;
  avgResolutionTime: number;
  slaCompliance: number;
  satisfactionScore: number;
  activeAgents: number;
  pendingTickets: number;
  urgentTickets: number;
}

interface ReportMetricsProps {
  metrics: ReportMetrics | null;
}

const getSLAStatus = (compliance: number) => {
  if (compliance >= 95) return { color: 'success', text: '优秀' };
  if (compliance >= 85) return { color: 'normal', text: '良好' };
  if (compliance >= 75) return { color: 'exception', text: '需改进' };
  return { color: 'exception', text: '严重' };
};

const getSatisfactionColor = (score: number) => {
  if (score >= 4.5) return '#52c41a';
  if (score >= 4.0) return '#1890ff';
  if (score >= 3.5) return '#faad14';
  return '#f5222d';
};

export const ReportMetrics: React.FC<ReportMetricsProps> = ({ metrics }) => {
  return (
    <Row gutter={[16, 16]} className='mb-6'>
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title='工单总数'
            value={metrics?.totalTickets}
            prefix={<FileTextOutlined style={{ color: '#1890ff' }} />}
            suffix={
              <div className='text-xs text-gray-500'>
                <div>已解决: {metrics?.resolvedTickets}</div>
                <div>待处理: {metrics?.pendingTickets}</div>
              </div>
            }
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title='平均解决时间'
            value={metrics?.avgResolutionTime}
            prefix={<ClockCircleOutlined style={{ color: '#52c41a' }} />}
            suffix='小时'
          />
          <div className='mt-2'>
            <Progress
              percent={Math.min(((metrics?.avgResolutionTime || 0) / 24) * 100, 100)}
              size='small'
              status={metrics && metrics.avgResolutionTime <= 12 ? 'success' : 'normal'}
            />
          </div>
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title='SLA达标率'
            value={metrics?.slaCompliance}
            prefix={<CheckCircleOutlined style={{ color: '#1890ff' }} />}
            suffix='%'
            styles={{ content: {
              color:
                getSLAStatus(metrics?.slaCompliance || 0).color === 'success'
                  ? '#52c41a'
                  : '#1890ff',
            }}
          />
          <div className='mt-2'>
            <Progress
              percent={metrics?.slaCompliance || 0}
              status={
                getSLAStatus(metrics?.slaCompliance || 0).color as
                  | 'success'
                  | 'normal'
                  | 'exception'
              }
              size='small'
            />
          </div>
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title='满意度评分'
            value={metrics?.satisfactionScore}
            precision={1}
            prefix={
              <StarOutlined
                style={{
                  color: getSatisfactionColor(metrics?.satisfactionScore || 0),
                }}
              />
            }
            suffix='/5.0'
          />
          <div className='mt-2'>
            <Progress
              percent={((metrics?.satisfactionScore || 0) / 5) * 100}
              status='success'
              size='small'
              strokeColor={getSatisfactionColor(metrics?.satisfactionScore || 0)}
            />
          </div>
        </Card>
      </Col>
    </Row>
  );
};
