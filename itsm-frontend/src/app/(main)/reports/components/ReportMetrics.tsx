'use client';

import React from 'react';
import { Card, Col, Progress, Row, Statistic } from 'antd';
import { FileText, Clock, CheckCircle, Star } from 'lucide-react';

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
  if (compliance >= 95) return { color: 'success', text: 'Excellent' };
  if (compliance >= 85) return { color: 'normal', text: 'Good' };
  if (compliance >= 75) return { color: 'exception', text: 'Needs improvement' };
  return { color: 'exception', text: 'Critical' };
};

const getSatisfactionColor = (score: number) => {
  if (score >= 4.5) return '#52c41a';
  if (score >= 4.0) return '#1890ff';
  if (score >= 3.5) return '#faad14';
  return '#f5222d';
};

export const ReportMetrics: React.FC<ReportMetricsProps> = ({ metrics }) => {
  return (
    <Row gutter={[16, 16]} className="mb-6">
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title="Total number of work orders"
            value={metrics?.totalTickets}
            prefix={<FileText style={{ color: '#1890ff' }} />}
            suffix={
              <div className="text-xs text-gray-500">
                <div>Resolved: {metrics?.resolvedTickets}</div>
                <div>Pending: {metrics?.pendingTickets}</div>
              </div>
            }
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title="Avg Resolution Time"
            value={metrics?.avgResolutionTime}
            prefix={<Clock style={{ color: '#52c41a' }} />}
            suffix="hours"
          />
          <div className="mt-2">
            <Progress
              percent={Math.min(((metrics?.avgResolutionTime || 0) / 24) * 100, 100)}
              size="small"
              status={metrics && metrics.avgResolutionTime <= 12 ? 'success' : 'normal'}
            />
          </div>
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title="SLA compliance rate"
            value={metrics?.slaCompliance}
            prefix={<CheckCircle style={{ color: '#1890ff' }} />}
            suffix="%"
            styles={{
              content: {
                color:
                  getSLAStatus(metrics?.slaCompliance || 0).color === 'success'
                    ? '#52c41a'
                    : '#1890ff',
              },
            }}
          />
          <div className="mt-2">
            <Progress
              percent={metrics?.slaCompliance || 0}
              status={
                getSLAStatus(metrics?.slaCompliance || 0).color as
                  | 'success'
                  | 'normal'
                  | 'exception'
              }
              size="small"
            />
          </div>
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title="Satisfaction Score"
            value={metrics?.satisfactionScore}
            precision={1}
            prefix={
              <Star
                style={{
                  color: getSatisfactionColor(metrics?.satisfactionScore || 0),
                }}
              />
            }
            suffix="/5.0"
          />
          <div className="mt-2">
            <Progress
              percent={((metrics?.satisfactionScore || 0) / 5) * 100}
              status="success"
              size="small"
              strokeColor={getSatisfactionColor(metrics?.satisfactionScore || 0)}
            />
          </div>
        </Card>
      </Col>
    </Row>
  );
};
