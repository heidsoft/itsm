'use client';

import React from 'react';
import {
  Card,
  Space,
  Typography,
  Button,
  Select,
  DatePicker,
  Tabs,
  Alert,
  Badge,
  Skeleton,
} from 'antd';
import {
  BarChartOutlined,
  DownloadOutlined,
  HeartOutlined,
  StarOutlined,
  PartitionOutlined,
  RiseOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
  WarningOutlined,
  EyeOutlined,
  CalendarOutlined,
} from '@ant-design/icons';
import { SmartSLAMonitor } from '@/components/business/SmartSLAMonitor';
import { PredictiveAnalytics } from '@/components/business/PredictiveAnalytics';
import { SatisfactionDashboard } from '@/components/business/SatisfactionDashboard';
import { TicketAssociation } from '@/components/business/TicketAssociation';
import { useReportData } from './hooks/useReportData';
import { ReportMetrics } from './components/ReportMetrics';
import { OverviewTab } from './components/OverviewTab';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;

const ReportsPageSkeleton: React.FC = () => (
  <div className='max-w-7xl mx-auto p-6'>
    <div className='mb-6'>
      <Skeleton.Input style={{ width: '300px' }} active />
    </div>
    <Card className='mb-6'>
      <Skeleton active paragraph={{ rows: 1 }} />
    </Card>
    <Skeleton active paragraph={{ rows: 4 }} />
    <Card className='mt-6'>
      <Skeleton active paragraph={{ rows: 2 }} />
    </Card>
  </div>
);

export default function ReportsPage() {
  const { t } = useI18n();
  const { metrics, timeRange, setTimeRange, loading } = useReportData();

  if (loading) {
    return <ReportsPageSkeleton />;
  }

  return (
    <div className='max-w-7xl mx-auto p-6'>
      {/* 页面标题 */}
      <div className='mb-6'>
        <Title level={2}>
          <BarChartOutlined style={{ marginRight: 12, color: '#1890ff' }} />
          {t('reports.title')}
        </Title>
        <Text type='secondary'>
          {t('reports.description')}
        </Text>
      </div>

      {/* 时间范围选择器 */}
      <Card className='mb-6'>
        <div className='flex items-center justify-between'>
          <Space>
            <CalendarOutlined style={{ color: '#666' }} />
            <Text strong>{t('reports.timeRange')}</Text>
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
              onChange={value => console.log(t('reports.quickSelect'), value)}
            >
              <Option value='7d'>{t('reports.last7Days')}</Option>
              <Option value='30d'>{t('reports.last30Days')}</Option>
              <Option value='90d'>{t('reports.last90Days')}</Option>
              <Option value='1y'>{t('reports.last1Year')}</Option>
            </Select>
            <Button icon={<DownloadOutlined />} type='primary'>
              {t('reports.export')}
            </Button>
          </Space>
        </div>
      </Card>

      <ReportMetrics metrics={metrics} />

      {/* 紧急工单提醒 */}
      {metrics && metrics.urgentTickets > 0 && (
        <Alert
          message={t('reports.urgentTicketsAlert', { count: metrics.urgentTickets })}
          description={t('reports.urgentTicketsDescription')}
          type='warning'
          showIcon
          icon={<WarningOutlined />}
          className='mb-6'
          action={
            <Button size='small' type='link'>
              {t('reports.viewDetails')}
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
                {t('reports.overview')}
              </Space>
            ),
            children: <OverviewTab metrics={metrics} />,
          },
          {
            key: 'sla',
            label: (
              <Space>
                <ClockCircleOutlined />
                {t('reports.slaMonitor')}
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
                {t('reports.satisfactionAnalysis')}
              </Space>
            ),
            children: <SatisfactionDashboard />,
          },
          {
            key: 'prediction',
            label: (
              <Space>
                <RiseOutlined />
                {t('reports.intelligentPrediction')}
              </Space>
            ),
            children: <PredictiveAnalytics />,
          },
          {
            key: 'association',
            label: (
              <Space>
                <PartitionOutlined />
                {t('reports.ticketAssociation')}
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
            <Title level={5}>{t('reports.quickActions')}</Title>
            <Text type='secondary'>{t('reports.quickActionsDescription')}</Text>
          </div>
          <Space>
            <Button icon={<DownloadOutlined />}>{t('reports.exportComprehensiveReport')}</Button>
            <Button icon={<EyeOutlined />} type='primary'>
              {t('reports.viewDetailedData')}
            </Button>
            <Button icon={<ThunderboltOutlined />}>{t('reports.generateAnalysisReport')}</Button>
          </Space>
        </div>
      </Card>
    </div>
  );
}
