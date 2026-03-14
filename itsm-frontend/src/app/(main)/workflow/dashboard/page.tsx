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
} from 'antd';
import {
  Activity,
  Clock,
  CheckCircle,
  AlertTriangle,
  XCircle,
  BarChart3,
  RefreshCw,
  TrendingUp,
} from 'lucide-react';

import BPMNDashboardApi, { DashboardMetrics, ProcessStat, TaskStat, TrendPoint } from '@/lib/api/bpmn-dashboard-api';
import { useI18n } from '@/lib/i18n';

const { RangePicker } = DatePicker;

export default function BPMNDashboardPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [dateRange, setDateRange] = useState<[string, string]>([
    new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
    new Date().toISOString().split('T')[0],
  ]);

  // 默认租户ID
  const tenantId = 1;

  const fetchMetrics = async () => {
    setLoading(true);
    try {
      const data = await BPMNDashboardApi.getDashboardMetrics(tenantId, dateRange[0], dateRange[1]);
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
      case 'running': return 'blue';
      case 'completed': return 'green';
      case 'assigned': return 'cyan';
      case 'created': return 'default';
      case 'cancelled': return 'red';
      default: return 'default';
    }
  };

  const topProcessColumns = [
    {
      title: t('workflow.process_key') || '流程Key',
      dataIndex: 'process_definition_key',
      key: 'process_definition_key',
    },
    {
      title: t('workflow.total_instances') || '总实例数',
      dataIndex: 'total_instances',
      key: 'total_instances',
    },
    {
      title: t('workflow.running') || '进行中',
      dataIndex: 'running_instances',
      key: 'running_instances',
    },
    {
      title: t('workflow.completed') || '已完成',
      dataIndex: 'completed_instances',
      key: 'completed_instances',
    },
    {
      title: t('workflow.avg_duration') || '平均耗时(分钟)',
      dataIndex: 'avg_duration_minutes',
      key: 'avg_duration_minutes',
      render: (val: number) => val?.toFixed(1) || '-',
    },
  ];

  const taskDistColumns = [
    {
      title: t('workflow.status') || '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{status}</Tag>
      ),
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
      <div className="flex items-center justify-center h-96">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">
          {t('bpmn.dashboard.title') || 'BPMN流程监控仪表盘'}
        </h1>
        <Space>
          <RangePicker
            onChange={(dates, dateStrings) => {
              if (dateStrings[0] && dateStrings[1]) {
                setDateRange([dateStrings[0], dateStrings[1]]);
              }
            }}
            defaultValue={[dateRange[0], dateRange[1]]}
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
              title={t('bpmn.dashboard.total_processes') || '流程定义'}
              value={metrics?.total_processes || 0}
              prefix={<BarChart3 size={20} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('bpmn.dashboard.active_instances') || '运行实例'}
              value={metrics?.active_instances || 0}
              prefix={<Activity size={20} />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('bpmn.dashboard.completed_today') || '今日完成'}
              value={metrics?.completed_today || 0}
              prefix={<CheckCircle size={20} />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('bpmn.dashboard.open_tasks') || '待处理任务'}
              value={metrics?.open_tasks || 0}
              prefix={<Clock size={20} />}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
      </Row>

      {/* Health & SLA */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card
            title={t('bpmn.dashboard.process_health') || '流程健康度'}
          >
            <Row gutter={16}>
              <Col span={8}>
                <Statistic
                  title={t('bpmn.dashboard.healthy') || '健康'}
                  value={metrics?.process_health?.healthy || 0}
                  prefix={<CheckCircle size={16} />}
                  valueStyle={{ color: '#52c41a' }}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title={t('bpmn.dashboard.warning') || '警告'}
                  value={metrics?.process_health?.warning || 0}
                  prefix={<AlertTriangle size={16} />}
                  valueStyle={{ color: '#faad14' }}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title={t('bpmn.dashboard.critical') || '严重'}
                  value={metrics?.process_health?.critical || 0}
                  prefix={<XCircle size={16} />}
                  valueStyle={{ color: '#ff4d4f' }}
                />
              </Col>
            </Row>
            <div className="mt-4 text-center">
              <Statistic
                title={t('bpmn.dashboard.health_score') || '健康度评分'}
                value={metrics?.process_health?.health_score || 0}
                suffix="/100"
                valueStyle={{ color: getHealthColor(metrics?.process_health?.health_score || 0) }}
              />
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title={t('bpmn.dashboard.sla_compliance') || 'SLA合规率'}
          >
            <div className="text-center py-8">
              <Statistic
                value={metrics?.sla_compliance_rate || 0}
                suffix="%"
                valueStyle={{
                  fontSize: 48,
                  color: (metrics?.sla_compliance_rate || 0) >= 90 ? '#52c41a' : (metrics?.sla_compliance_rate || 0) >= 70 ? '#faad14' : '#ff4d4f'
                }}
              />
              <p className="text-gray-500 mt-2">
                {t('bpmn.dashboard.sla_compliance_rate') || 'SLA合规率'}
              </p>
            </div>
          </Card>
        </Col>
      </Row>

      {/* Top Processes & Task Distribution */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card
            title={t('bpmn.dashboard.top_processes') || '热门流程'}
          >
            <Table
              dataSource={metrics?.top_processes || []}
              columns={topProcessColumns}
              rowKey="process_definition_key"
              size="small"
              pagination={false}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title={t('bpmn.dashboard.task_distribution') || '任务分布'}
          >
            <Table
              dataSource={metrics?.task_distribution || []}
              columns={taskDistColumns}
              rowKey="status"
              size="small"
              pagination={false}
            />
          </Card>
        </Col>
      </Row>

      {/* Trend Chart Placeholder */}
      <Card
        title={t('bpmn.dashboard.trend') || '流程趋势'}
      >
        <div className="h-48 flex items-center justify-center text-gray-400">
          <Space>
            <TrendingUp size={24} />
            <span>{t('bpmn.dashboard.trend_chart') || '趋势图表（待开发）'}</span>
          </Space>
        </div>
      </Card>
    </div>
  );
}
