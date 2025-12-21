'use client';

import React from 'react';
import { Card, Col, Progress, Row, Tag, Typography } from 'antd';
import { ReportMetrics } from './ReportMetrics';

const { Text } = Typography;

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

interface OverviewTabProps {
  metrics: ReportMetrics | null;
}

const getSLAStatus = (compliance: number) => {
  if (compliance >= 95) return { color: 'success', text: '优秀' };
  if (compliance >= 85) return { color: 'normal', text: '良好' };
  if (compliance >= 75) return { color: 'exception', text: '需改进' };
  return { color: 'exception', text: '严重' };
};

export const OverviewTab: React.FC<OverviewTabProps> = ({ metrics }) => {
  return (
    <div className='space-y-6'>
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title='工单状态分布'>
            <div className='space-y-4'>
              <div className='flex items-center justify-between'>
                <Text>已解决</Text>
                <div className='flex items-center'>
                  <Progress
                    percent={Math.round(
                      ((metrics?.resolvedTickets || 0) / (metrics?.totalTickets || 1)) * 100
                    )}
                    size='small'
                    status='success'
                    className='mr-2'
                    style={{ width: 100 }}
                  />
                  <Text strong>{metrics?.resolvedTickets}</Text>
                </div>
              </div>
              <div className='flex items-center justify-between'>
                <Text>待处理</Text>
                <div className='flex items-center'>
                  <Progress
                    percent={Math.round(
                      ((metrics?.pendingTickets || 0) / (metrics?.totalTickets || 1)) * 100
                    )}
                    size='small'
                    status='normal'
                    className='mr-2'
                    style={{ width: 100 }}
                  />
                  <Text strong>{metrics?.pendingTickets}</Text>
                </div>
              </div>
              <div className='flex items-center justify-between'>
                <Text>紧急工单</Text>
                <div className='flex items-center'>
                  <Progress
                    percent={Math.round(
                      ((metrics?.urgentTickets || 0) / (metrics?.totalTickets || 1)) * 100
                    )}
                    size='small'
                    status='exception'
                    className='mr-2'
                    style={{ width: 100 }}
                  />
                  <Text strong style={{ color: '#f5222d' }}>
                    {metrics?.urgentTickets}
                  </Text>
                </div>
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title='服务质量指标'>
            <div className='space-y-4'>
              <div className='flex items-center justify-between'>
                <Text>SLA达标率</Text>
                <div className='flex items-center'>
                  <Tag
                    color={
                      getSLAStatus(metrics?.slaCompliance || 0).color === 'success'
                        ? 'green'
                        : 'blue'
                    }
                  >
                    {getSLAStatus(metrics?.slaCompliance || 0).text}
                  </Tag>
                  <Text strong className='ml-2'>
                    {metrics?.slaCompliance}%
                  </Text>
                </div>
              </div>
              <div className='flex items-center justify-between'>
                <Text>平均解决时间</Text>
                <div className='flex items-center'>
                  <Tag
                    color={metrics && metrics.avgResolutionTime <= 12 ? 'green' : 'orange'}
                  >
                    {metrics && metrics.avgResolutionTime <= 12 ? '优秀' : '需改进'}
                  </Tag>
                  <Text strong className='ml-2'>
                    {metrics?.avgResolutionTime}小时
                  </Text>
                </div>
              </div>
              <div className='flex items-center justify-between'>
                <Text>满意度评分</Text>
                <div className='flex items-center'>
                  <Tag color={metrics && metrics.satisfactionScore >= 4.0 ? 'green' : 'orange'}>
                    {metrics && metrics.satisfactionScore >= 4.0 ? '满意' : '需改进'}
                  </Tag>
                  <Text strong className='ml-2'>
                    {metrics?.satisfactionScore}/5.0
                  </Text>
                </div>
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
};
