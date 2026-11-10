'use client';

import React, { useState, useEffect } from 'react';
import {
  App,
  Card,
  Row,
  Col,
  Statistic,
  Table,
  Tag,
  Space,
  Button,
  DatePicker,
  Spin,
  Alert,
  Empty,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import {
  Activity,
  Clock,
  CheckCircle,
  AlertTriangle,
  XCircle,
  BarChart3,
  RefreshCw,
} from 'lucide-react';

import type { DashboardMetrics } from '@/lib/api/bpmn-dashboard-api';
import BPMNDashboardApi, { ProcessStat, TaskStat } from '@/lib/api/bpmn-dashboard-api';
import { useAuthStore } from '@/lib/store/auth-store';
import { useI18n } from '@/lib/i18n';

const { RangePicker } = DatePicker;

// 任务状态是后端枚举（created/assigned/completed/...），统一映射中文，不在表格里直出英文。
const taskStatusLabel: Record<string, string> = {
  created: 'Created',
  pending: 'Pending',
  assigned: 'Assigned',
  claimed: 'Received',
  in_progress: 'In Progress',
  running: 'Running',
  completed: 'Completed',
  cancelled: 'Cancelled',
  terminated: 'Terminated',
  failed: 'Failed',
  skipped: 'Skipped',
  suspended: 'Suspended',
};

export default function BPMNDashboardPage() {
  const { t } = useI18n();
  const { message } = App.useApp();
  const currentTenant = useAuthStore(s => s.currentTenant);
  const [loading, setLoading] = useState(true);
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([dayjs().subtract(7, 'day'), dayjs()]);

  // 使用当前登录租户；未登录时不加载数据
  const tenantId = currentTenant?.id;

  const fetchMetrics = async () => {
    if (!tenantId) {
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const data = await BPMNDashboardApi.getDashboardMetrics(
        tenantId,
        dateRange[0].format('YYYY-MM-DD'),
        dateRange[1].format('YYYY-MM-DD')
      );
      setMetrics(data);
    } catch (error) {
      console.error('Failed to fetch dashboard metrics:', error);
      message.error('Failed to load workflow dashboard metrics');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMetrics();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dateRange, tenantId]);

  const getHealthColor = (score: number) => {
    if (score >= 80) return 'green';
    if (score >= 60) return 'orange';
    return 'red';
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'blue';
      case 'completed':
        return 'green';
      case 'assigned':
        return 'cyan';
      case 'created':
        return 'default';
      case 'cancelled':
        return 'red';
      default:
        return 'default';
    }
  };

  const topProcessColumns: ColumnsType<ProcessStat> = [
    {
      title: t('workflow.processKey') || 'Process Key',
      dataIndex: 'processDefinitionKey',
      key: 'processDefinitionKey',
      ellipsis: true,
    },
    {
      title: t('workflow.totalInstances') || 'Total Instances',
      dataIndex: 'totalInstances',
      key: 'totalInstances',
      width: 100,
      align: 'right',
    },
    {
      title: t('workflow.running') || 'In Progress',
      dataIndex: 'runningInstances',
      key: 'runningInstances',
      width: 90,
      align: 'right',
    },
    {
      title: t('workflow.completed') || 'Completed',
      dataIndex: 'completedInstances',
      key: 'completedInstances',
      width: 90,
      align: 'right',
    },
    {
      title: t('workflow.avgDuration') || 'Avg Duration (min)',
      dataIndex: 'avgDurationMinutes',
      key: 'avgDurationMinutes',
      width: 130,
      align: 'right',
      render: (val: number) => (typeof val === 'number' ? val.toFixed(1) : '-'),
    },
  ];

  const taskDistColumns: ColumnsType<TaskStat> = [
    {
      title: t('workflow.status') || 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{taskStatusLabel[status] ?? status}</Tag>
      ),
    },
    {
      title: t('workflow.count') || 'Count',
      dataIndex: 'count',
      key: 'count',
      width: 90,
      align: 'right',
    },
    {
      title: t('workflow.percentage') || 'Percentage',
      dataIndex: 'percent',
      key: 'percent',
      width: 100,
      align: 'right',
      render: (val: number) => (typeof val === 'number' ? `${val.toFixed(1)}%` : '-'),
    },
  ];

  if (loading && !metrics) {
    return (
      <div className='flex items-center justify-center h-96'>
        <Spin size='large' />
      </div>
    );
  }

  if (!tenantId) {
    return (
      <div className='p-6'>
        <Alert
          title={t('notificationCenter.typeLabels.warning') || 'Warning'}
          description={t('auth.login.subtitle') || 'Please log in first to view the workflow dashboard'}
          type='warning'
          showIcon
        />
      </div>
    );
  }

  const healthScore = metrics?.processHealth?.healthScore ?? 0;
  const slaRate = metrics?.slaComplianceRate ?? 0;
  // 无已完成样本时后端诚实返回 0；必须与“真实 0% 合规”区分展示。
  const slaTotalSamples = metrics?.slaTotalSamples ?? 0;
  const slaCompliantSamples = metrics?.slaCompliantSamples ?? 0;
  const trendData = metrics?.trendData || [];

  const slaColor =
    slaRate >= 90 ? '#52c41a' : slaRate >= 70 ? '#faad14' : '#ff4d4f';

  return (
    <div className='p-6 space-y-6'>
      {/* Header */}
      <div className='flex justify-between items-center'>
        <h1 className='text-2xl font-bold'>
          {t('workflow.bpmnDashboard.title') || 'BPMN Process Monitor Dashboard'}
        </h1>
        <Space>
          <RangePicker
            onChange={(dates, dateStrings) => {
              if (dates && dates[0] && dates[1]) {
                setDateRange([dates[0], dates[1]]);
              }
            }}
            value={[dateRange[0], dateRange[1]]}
          />
          <Button icon={<RefreshCw size={16} />} onClick={fetchMetrics} loading={loading}>
            {t('common.refresh') || 'Refresh'}
          </Button>
        </Space>
      </div>

      {/* Summary Cards：Card 撑满 Col，保证四张卡片等高 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card style={{ height: '100%' }}>
            <Statistic
              title={t('workflow.bpmnDashboard.totalProcesses') || 'Process Definitions'}
              value={metrics?.totalProcesses || 0}
              prefix={<BarChart3 size={18} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card style={{ height: '100%' }}>
            <Statistic
              title={t('workflow.bpmnDashboard.activeInstances') || 'Active Instances'}
              value={metrics?.activeInstances || 0}
              prefix={<Activity size={18} />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card style={{ height: '100%' }}>
            <Statistic
              title={t('workflow.bpmnDashboard.completedToday') || 'Completed Today'}
              value={metrics?.completedToday || 0}
              prefix={<CheckCircle size={18} />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card style={{ height: '100%' }}>
            <Statistic
              title={t('workflow.bpmnDashboard.openTasks') || 'Open Tasks'}
              value={metrics?.openTasks || 0}
              prefix={<Clock size={18} />}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* Health & SLA */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card
            title={t('workflow.bpmnDashboard.processHealth') || 'Process Health'}
            style={{ height: '100%' }}
          >
            <Row gutter={16}>
              <Col span={8}>
                <Statistic
                  title={t('workflow.bpmnDashboard.healthy') || 'Healthy'}
                  value={metrics?.processHealth?.healthy || 0}
                  prefix={<CheckCircle size={16} />}
                  styles={{ content: { color: '#52c41a' } }}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title={t('workflow.bpmnDashboard.warning') || 'Warning'}
                  value={metrics?.processHealth?.warning || 0}
                  prefix={<AlertTriangle size={16} />}
                  styles={{ content: { color: '#faad14' } }}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title={t('workflow.bpmnDashboard.critical') || 'Critical'}
                  value={metrics?.processHealth?.critical || 0}
                  prefix={<XCircle size={16} />}
                  styles={{ content: { color: '#ff4d4f' } }}
                />
              </Col>
            </Row>
            <div className='mt-4 text-center'>
              <Statistic
                title={t('workflow.bpmnDashboard.healthScore') || 'Health Score'}
                value={healthScore}
                precision={1}
                suffix='/100'
                styles={{ content: { color: getHealthColor(healthScore) } }}
              />
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title={t('workflow.bpmnDashboard.slaCompliance') || 'SLA Compliance Rate'}
            style={{ height: '100%' }}
          >
            {slaTotalSamples > 0 ? (
              <div className='text-center py-4'>
                <Statistic
                  value={slaRate}
                  precision={1}
                  suffix='%'
                  styles={{ content: { fontSize: 48, color: slaColor } }}
                />
                <p className='text-gray-500 mt-2'>
                  {t('workflow.bpmnDashboard.slaComplianceRate') || 'SLA Compliance Rate'} ·{' '}
                  {slaCompliantSamples}/{slaTotalSamples}  Completed instances meet the criteria
                </p>
              </div>
            ) : (
              <div className='h-full flex items-center justify-center py-8'>
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description="There are no completed process instances in the selected time range, and the SLA compliance rate cannot be calculated."
                />
              </div>
            )}
          </Card>
        </Col>
      </Row>

      {/* Top Processes & Task Distribution */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card
            title={t('workflow.bpmnDashboard.topProcesses') || 'Top Processes'}
            style={{ height: '100%' }}
          >
            <Table<ProcessStat>
              dataSource={metrics?.topProcesses || []}
              columns={topProcessColumns}
              rowKey='processDefinitionKey'
              size='small'
              pagination={false}
              scroll={{ x: 620 }}
              locale={{ emptyText: 'No process instance data yet' }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title={t('workflow.bpmnDashboard.taskDistribution') || 'Task Distribution'}
            style={{ height: '100%' }}
          >
            <Table<TaskStat>
              dataSource={metrics?.taskDistribution || []}
              columns={taskDistColumns}
              rowKey='status'
              size='small'
              pagination={false}
              locale={{ emptyText: 'No task data yet' }}
            />
          </Card>
        </Col>
      </Row>

      {/* Trend：渲染后端真实 trendData，不再固定显示空态 */}
      <Card title={t('workflow.bpmnDashboard.trend') || 'Process Trend'}>
        {trendData.length > 0 ? (
          <div style={{ width: '100%', height: 280 }}>
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={trendData} margin={{ top: 8, right: 24, bottom: 8, left: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="date" tick={{ fontSize: 12 }} />
                <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                <Tooltip
                  formatter={(value) => [`${value ?? 0}`, 'Number of instances']}
                  labelFormatter={(label) => `Date:${label}`}
                />
                <Legend formatter={() => 'Number of instances started per day'} />
                <Line
                  type="monotone"
                  dataKey="count"
                  stroke="#1890ff"
                  strokeWidth={2}
                  dot={{ r: 3 }}
                  activeDot={{ r: 5 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        ) : (
          <div className='h-48 flex items-center justify-center'>
            <Empty description="There is no process trend data in the selected time range." />
          </div>
        )}
      </Card>
    </div>
  );
}
