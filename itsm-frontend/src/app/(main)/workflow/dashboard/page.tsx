'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Table,
  Tag,
  Space,
  Button,
  DatePicker,
  Select,
  Spin,
  Alert,
  Empty,
} from 'antd';
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
import BPMNDashboardApi, { ProcessStat, TaskStat, TrendPoint } from '@/lib/api/bpmn-dashboard-api';
import { useAuthStore } from '@/lib/store/auth-store';
import { useI18n } from '@/lib/i18n';

const { RangePicker } = DatePicker;

export default function BPMNDashboardPage() {
  const { t } = useI18n();
  const currentTenant = useAuthStore(s => s.currentTenant);
  const [loading, setLoading] = useState(true);
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([dayjs().subtract(7, 'day'), dayjs()]);

  // 优先使用当前登录租户；未登录时回退到默认 1
  // TODO: 待接入用户/租户选择器后移除硬编码回退值，避免未登录态误指向租户 1
  const tenantId = currentTenant?.id ?? 1;

  const fetchMetrics = async () => {
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
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMetrics();
  }, [dateRange]);

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

  const topProcessColumns = [
    {
      title: t('workflow.processKey') || '流程Key',
      dataIndex: 'processDefinitionKey',
      key: 'processDefinitionKey',
    },
    {
      title: t('workflow.totalInstances') || '总实例数',
      dataIndex: 'totalInstances',
      key: 'totalInstances',
    },
    {
      title: t('workflow.running') || '进行中',
      dataIndex: 'runningInstances',
      key: 'runningInstances',
    },
    {
      title: t('workflow.completed') || '已完成',
      dataIndex: 'completedInstances',
      key: 'completedInstances',
    },
    {
      title: t('workflow.avgDuration') || '平均耗时(分钟)',
      dataIndex: 'avgDurationMinutes',
      key: 'avgDurationMinutes',
      render: (val: number) => val?.toFixed(1) || '-',
    },
  ];

  const taskDistColumns = [
    {
      title: t('workflow.status') || '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => <Tag color={getStatusColor(status)}>{status}</Tag>,
    },
    {
      title: t('workflow.count') || '数量',
      dataIndex: 'count',
      key: 'count',
    },
    {
      title: t('workflow.percentage') || '占比',
      dataIndex: 'percent',
      key: 'percent',
      render: (val: number) => `${val?.toFixed(1)}%`,
    },
  ];

  if (loading && !metrics) {
    return (
      <div className='flex items-center justify-center h-96'>
        <Spin size='large' />
      </div>
    );
  }

  return (
    <div className='p-6 space-y-6'>
      {/* Header */}
      <div className='flex justify-between items-center'>
        <h1 className='text-2xl font-bold'>
          {t('workflow.bpmnDashboard.title') || 'BPMN流程监控仪表盘'}
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
          <Button icon={<RefreshCw size={16} />} onClick={fetchMetrics}>
            {t('common.refresh') || '刷新'}
          </Button>
        </Space>
      </div>

      {/* Summary Cards */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('workflow.bpmnDashboard.totalProcesses') || '流程定义'}
              value={metrics?.totalProcesses || 0}
              prefix={<BarChart3 size={20} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('workflow.bpmnDashboard.activeInstances') || '运行实例'}
              value={metrics?.activeInstances || 0}
              prefix={<Activity size={20} />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('workflow.bpmnDashboard.completedToday') || '今日完成'}
              value={metrics?.completedToday || 0}
              prefix={<CheckCircle size={20} />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('workflow.bpmnDashboard.openTasks') || '待处理任务'}
              value={metrics?.openTasks || 0}
              prefix={<Clock size={20} />}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* Health & SLA */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title={t('workflow.bpmnDashboard.processHealth') || '流程健康度'}>
            <Row gutter={16}>
              <Col span={8}>
                <Statistic
                  title={t('workflow.bpmnDashboard.healthy') || '健康'}
                  value={metrics?.processHealth?.healthy || 0}
                  prefix={<CheckCircle size={16} />}
                  styles={{ content: { color: '#52c41a' } }}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title={t('workflow.bpmnDashboard.warning') || '警告'}
                  value={metrics?.processHealth?.warning || 0}
                  prefix={<AlertTriangle size={16} />}
                  styles={{ content: { color: '#faad14' } }}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title={t('workflow.bpmnDashboard.critical') || '严重'}
                  value={metrics?.processHealth?.critical || 0}
                  prefix={<XCircle size={16} />}
                  styles={{ content: { color: '#ff4d4f' } }}
                />
              </Col>
            </Row>
            <div className='mt-4 text-center'>
              <Statistic
                title={t('workflow.bpmnDashboard.healthScore') || '健康度评分'}
                value={metrics?.processHealth?.healthScore || 0}
                suffix='/100'
                styles={{
                  content: { color: getHealthColor(metrics?.processHealth?.healthScore || 0) },
                }}
              />
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('workflow.bpmnDashboard.slaCompliance') || 'SLA合规率'}>
            <div className='text-center py-8'>
              <Statistic
                value={metrics?.slaComplianceRate || 0}
                suffix='%'
                styles={{
                  content: {
                    fontSize: 48,
                    color:
                      (metrics?.slaComplianceRate || 0) >= 90
                        ? '#52c41a'
                        : (metrics?.slaComplianceRate || 0) >= 70
                          ? '#faad14'
                          : '#ff4d4f',
                  },
                }}
              />
              <p className='text-gray-500 mt-2'>
                {t('workflow.bpmnDashboard.slaComplianceRate') || 'SLA合规率'}
              </p>
            </div>
          </Card>
        </Col>
      </Row>

      {/* Top Processes & Task Distribution */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title={t('workflow.bpmnDashboard.topProcesses') || '热门流程'}>
            <Table
              dataSource={metrics?.topProcesses || []}
              columns={topProcessColumns}
              rowKey='processDefinitionKey'
              size='small'
              pagination={false}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('workflow.bpmnDashboard.taskDistribution') || '任务分布'}>
            <Table
              dataSource={metrics?.taskDistribution || []}
              columns={taskDistColumns}
              rowKey='status'
              size='small'
              pagination={false}
            />
          </Card>
        </Col>
      </Row>

      <Card title={t('workflow.bpmnDashboard.trend') || '流程趋势'}>
        <div className='h-48 flex items-center justify-center'>
          <Empty description='暂无流程趋势数据' />
        </div>
      </Card>
    </div>
  );
}
