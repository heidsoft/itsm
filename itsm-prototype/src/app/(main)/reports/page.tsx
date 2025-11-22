'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Space,
  Typography,
  Row,
  Col,
  Statistic,
  Button,
  Select,
  DatePicker,
  Tabs,
  Alert,
  Badge,
  Progress,
  Tag,
} from 'antd';
import {
  BarChartOutlined,
  RiseOutlined,
  FallOutlined,
  DownloadOutlined,
  HeartOutlined,
  StarOutlined,
  FileTextOutlined,
  PartitionOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  EyeOutlined,
  CalendarOutlined,
} from '@ant-design/icons';
import { SmartSLAMonitor } from '@/components/business/SmartSLAMonitor';
import { PredictiveAnalytics } from '@/components/business/PredictiveAnalytics';
import { SatisfactionDashboard } from '@/components/business/SatisfactionDashboard';
import { TicketAssociation } from '../components/TicketAssociation';

const { Title, Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;

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

export default function ReportsPage() {
  const [metrics, setMetrics] = useState<ReportMetrics | null>(null);
  const [timeRange, setTimeRange] = useState<[string, string] | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadReportMetrics();
  }, [timeRange]);

  const loadReportMetrics = async () => {
    try {
      const mockMetrics: ReportMetrics = {
        totalTickets: 1247,
        resolvedTickets: 1189,
        avgResolutionTime: 8.7,
        slaCompliance: 87.3,
        satisfactionScore: 4.2,
        activeAgents: 12,
        pendingTickets: 58,
        urgentTickets: 23,
      };

      setMetrics(mockMetrics);
    } catch (error) {
      console.error('加载报表数据失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const getTrendIcon = (current: number, previous: number) => {
    if (current > previous) return <RiseOutlined style={{ color: '#52c41a' }} />;
    if (current < previous) return <FallOutlined style={{ color: '#f5222d' }} />;
    return <RiseOutlined style={{ color: '#1890ff' }} />;
  };

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

  if (loading) {
    return (
      <Card>
        <div className='animate-pulse space-y-4'>
          <div className='h-4 bg-gray-200 rounded w-1/3'></div>
          <div className='space-y-3'>
            <div className='h-3 bg-gray-200 rounded'></div>
            <div className='h-3 bg-gray-200 rounded w-5/6'></div>
            <div className='h-3 bg-gray-200 rounded w-4/6'></div>
          </div>
        </div>
      </Card>
    );
  }

  return (
    <div className='max-w-7xl mx-auto p-6'>
      {/* 页面标题 */}
      <div className='mb-6'>
        <Title level={2}>
          <BarChartOutlined style={{ marginRight: 12, color: '#1890ff' }} />
          智能报表中心
        </Title>
        <Text type='secondary'>
          全面的ITSM数据分析和智能洞察，帮助您优化服务质量和运营效率
        </Text>
      </div>

      {/* 时间范围选择器 */}
      <Card className='mb-6'>
        <div className='flex items-center justify-between'>
          <Space>
            <CalendarOutlined style={{ color: '#666' }} />
            <Text strong>报表时间范围</Text>
          </Space>
          <Space>
            <RangePicker
              value={timeRange}
              onChange={dates => {
                if (dates) {
                  setTimeRange([dates[0]?.toISOString() || '', dates[1]?.toISOString() || '']);
                }
              }}
            />
            <Select
              defaultValue='30d'
              style={{ width: 120 }}
              onChange={value => console.log('快速选择:', value)}
            >
              <Option value='7d'>近7天</Option>
              <Option value='30d'>近30天</Option>
              <Option value='90d'>近90天</Option>
              <Option value='1y'>近一年</Option>
            </Select>
            <Button icon={<DownloadOutlined />} type='primary'>
              导出报表
            </Button>
          </Space>
        </div>
      </Card>

      {/* 关键指标概览 */}
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
              valueStyle={{
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

      {/* 紧急工单提醒 */}
      {metrics && metrics.urgentTickets > 0 && (
        <Alert
          message={`当前有 ${metrics.urgentTickets} 个紧急工单需要立即处理`}
          description='请优先处理高优先级工单，以确保SLA达标'
          type='warning'
          showIcon
          icon={<WarningOutlined />}
          className='mb-6'
          action={
            <Button size='small' type='link'>
              查看详情
            </Button>
          }
        />
      )}

      {/* 主报表内容 */}
      <Tabs
        defaultActiveKey='overview'
        size='large'
        type='card'
        items={[
          {
            key: 'overview',
            label: (
              <Space>
                <BarChartOutlined />
                概览
              </Space>
            ),
            children: (
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
                                ((metrics?.resolvedTickets || 0) / (metrics?.totalTickets || 1)) *
                                  100
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
                                ((metrics?.pendingTickets || 0) / (metrics?.totalTickets || 1)) *
                                  100
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
                              color={
                                metrics && metrics.avgResolutionTime <= 12 ? 'green' : 'orange'
                              }
                            >
                              {metrics && metrics.avgResolutionTime <= 12
                                ? '优秀'
                                : '需改进'}
                            </Tag>
                            <Text strong className='ml-2'>
                              {metrics?.avgResolutionTime}小时
                            </Text>
                          </div>
                        </div>
                        <div className='flex items-center justify-between'>
                          <Text>满意度评分</Text>
                          <div className='flex items-center'>
                            <Tag
                              color={
                                metrics && metrics.satisfactionScore >= 4.0 ? 'green' : 'orange'
                              }
                            >
                              {metrics && metrics.satisfactionScore >= 4.0
                                ? '满意'
                                : '需改进'}
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
            ),
          },
          {
            key: 'sla',
            label: (
              <Space>
                <ClockCircleOutlined />
                SLA监控
                <Badge count={metrics?.urgentTickets} />
              </Space>
            ),
            children: <SmartSLAMonitor />,
          },
          {
            key: 'satisfaction',
            label: (
              <Space>
                <HeartOutlined />
                满意度分析
              </Space>
            ),
            children: <SatisfactionDashboard />,
          },
          {
            key: 'prediction',
            label: (
              <Space>
                <RiseOutlined />
                智能预测
              </Space>
            ),
            children: <PredictiveAnalytics />,
          },
          {
            key: 'association',
            label: (
              <Space>
                <PartitionOutlined />
                工单关联
              </Space>
            ),
            children: <TicketAssociation />,
          },
        ]}
      />

      {/* 快速操作 */}
      <Card className='mt-6'>
        <div className='flex items-center justify-between'>
          <div>
            <Title level={5}>快速操作</Title>
            <Text type='secondary'>快速访问常用报表功能</Text>
          </div>
          <Space>
            <Button icon={<DownloadOutlined />}>导出综合报表</Button>
            <Button icon={<EyeOutlined />} type='primary'>
              查看详细数据
            </Button>
            <Button icon={<ThunderboltOutlined />}>生成分析报告</Button>
          </Space>
        </div>
      </Card>
    </div>
  );
}
