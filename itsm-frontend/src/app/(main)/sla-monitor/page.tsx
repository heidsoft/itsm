'use client';

import React, { useState, useEffect, useMemo } from 'react';
import {
  Card,
  Typography,
  Row,
  Col,
  Statistic,
  Button,
  Space,
  Select,
  Tabs,
  Table,
  Tag,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { SLAMonitorDashboard } from '@/components/business/SLAMonitorDashboard';
import { AlertTriangle, CheckCircle, Clock, RefreshCw, TrendingUp } from 'lucide-react';
import { SLAApi } from '@/lib/api/sla-api';
import type { SLADefinition, SLAViolation } from '@/lib/api/sla-api';

const { Title, Text } = Typography;

// 后端 priority / severity 返回英文枚举，展示层只做文案本地化，
// 不得用中文字符串匹配后端值（那样会永远命中不到而显示为空）。
const PRIORITY_META: Record<string, { color: string; label: string }> = {
  urgent: { color: 'red', label: '紧急' },
  critical: { color: 'red', label: '紧急' },
  high: { color: 'orange', label: '高' },
  medium: { color: 'blue', label: '中' },
  low: { color: 'default', label: '低' },
};

const SEVERITY_META: Record<string, { color: string; label: string }> = {
  critical: { color: 'red', label: '严重' },
  high: { color: 'orange', label: '高' },
  medium: { color: 'gold', label: '中' },
  low: { color: 'blue', label: '低' },
};

const VIOLATION_TYPE_LABELS: Record<string, string> = {
  response: '响应超时',
  resolution: '解决超时',
};

const formatDateTime = (value?: string): string =>
  value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-';

/** 单个 SLA 定义的违规聚合，全部来自后端违规列表，不做任何推算 */
interface ServiceSLARow {
  id: number;
  serviceName: string;
  serviceType: string;
  priority: string;
  isActive: boolean;
  responseTime: number;
  resolutionTime: number;
  violationTotal: number;
  violationOpen: number;
}

const SLAMonitorPage = () => {
  const [autoRefresh] = useState(true);
  const [refreshInterval, setRefreshInterval] = useState(30);
  const [activeTab, setActiveTab] = useState('overview');
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState({
    totalDefinitions: 0,
    activeDefinitions: 0,
    totalViolations: 0,
    openViolations: 0,
    overallComplianceRate: 0,
  });
  const [violations, setViolations] = useState<SLAViolation[]>([]);
  const [serviceSLA, setServiceSLA] = useState<ServiceSLARow[]>([]);

  const fetchStats = async () => {
    if (loading) return;
    setLoading(true);
    try {
      const [statsData, violationResp, definitionResp] = await Promise.all([
        SLAApi.getSLAStats(),
        SLAApi.getSLAViolations({ page: 1, pageSize: 200 }),
        SLAApi.getSLADefinitions({ page: 1, pageSize: 200 }),
      ]);

      setStats({
        totalDefinitions: statsData.totalDefinitions || 0,
        activeDefinitions: statsData.activeDefinitions || 0,
        totalViolations: statsData.totalViolations || 0,
        openViolations: statsData.openViolations || 0,
        overallComplianceRate: statsData.overallComplianceRate || 0,
      });

      const allViolations = violationResp?.items || [];
      setViolations(allViolations);

      // 服务 SLA 绩效：按 slaDefinitionId 聚合真实违规记录。
      // 后端没有“每服务请求总数/合规率”的数据源，因此不再伪造 95% 之类的数字。
      const violationCountByDef = new Map<number, { total: number; open: number }>();
      allViolations.forEach((v) => {
        const key = v.slaDefinitionId;
        const bucket = violationCountByDef.get(key) || { total: 0, open: 0 };
        bucket.total += 1;
        if (!v.isResolved) bucket.open += 1;
        violationCountByDef.set(key, bucket);
      });

      const services: ServiceSLARow[] = (definitionResp?.items || []).map((sla: SLADefinition) => {
        const bucket = violationCountByDef.get(sla.id) || { total: 0, open: 0 };
        return {
          id: sla.id,
          serviceName: sla.name,
          serviceType: sla.serviceType || '-',
          priority: sla.priority || '-',
          isActive: !!sla.isActive,
          responseTime: sla.responseTime || 0,
          resolutionTime: sla.resolutionTime || 0,
          violationTotal: bucket.total,
          violationOpen: bucket.open,
        };
      });
      setServiceSLA(services);
    } catch (error) {
      console.error('Failed to fetch SLA stats:', error);
      message.error('获取SLA统计数据失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStats();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const ticketColumns: ColumnsType<SLAViolation> = [
    {
      title: '工单号',
      key: 'ticketNumber',
      width: 150,
      render: (_, record) => <a>{record.ticketNumber || `#${record.ticketId}`}</a>,
    },
    {
      title: '标题',
      dataIndex: 'ticketTitle',
      key: 'ticketTitle',
      ellipsis: true,
      render: (title: string | undefined, record) => (
        <span>{title || `Ticket #${record.ticketId}`}</span>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'ticketPriority',
      key: 'ticketPriority',
      width: 90,
      render: (priority: string | undefined) => {
        const meta = priority ? PRIORITY_META[priority.toLowerCase()] : undefined;
        return <Tag color={meta?.color || 'default'}>{meta?.label || priority || '-'}</Tag>;
      },
    },
    {
      title: 'SLA 目标',
      dataIndex: 'slaName',
      key: 'slaName',
      width: 160,
      ellipsis: true,
      render: (name: string | undefined) => name || '-',
    },
    {
      title: '违规类型',
      dataIndex: 'violationType',
      key: 'violationType',
      width: 110,
      render: (type: string) => VIOLATION_TYPE_LABELS[type] || type || '-',
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (severity: string) => {
        const meta = SEVERITY_META[(severity || '').toLowerCase()];
        return <Tag color={meta?.color || 'default'}>{meta?.label || severity || '-'}</Tag>;
      },
    },
    {
      title: '违规时间',
      dataIndex: 'violationTime',
      key: 'violationTime',
      width: 180,
      render: formatDateTime,
    },
    {
      title: '状态',
      key: 'status',
      width: 90,
      render: (_, record) =>
        record.isResolved ? <Tag color="green">已解决</Tag> : <Tag color="red">未解决</Tag>,
    },
  ];

  const serviceColumns: ColumnsType<ServiceSLARow> = [
    {
      title: '服务名称',
      dataIndex: 'serviceName',
      key: 'serviceName',
      ellipsis: true,
      render: (text: string) => <a>{text}</a>,
    },
    {
      title: '服务类型',
      dataIndex: 'serviceType',
      key: 'serviceType',
      width: 130,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (priority: string) => {
        const meta = PRIORITY_META[(priority || '').toLowerCase()];
        return <Tag color={meta?.color || 'default'}>{meta?.label || priority}</Tag>;
      },
    },
    {
      title: '响应目标',
      dataIndex: 'responseTime',
      key: 'responseTime',
      width: 110,
      render: (minutes: number) => `${minutes} 分钟`,
    },
    {
      title: '解决目标',
      dataIndex: 'resolutionTime',
      key: 'resolutionTime',
      width: 110,
      render: (minutes: number) => `${minutes} 分钟`,
    },
    {
      title: '违规总数',
      dataIndex: 'violationTotal',
      key: 'violationTotal',
      width: 110,
      sorter: (a, b) => a.violationTotal - b.violationTotal,
      render: (val: number) => <span className={val > 0 ? 'text-red-600' : ''}>{val}</span>,
    },
    {
      title: '未解决违规',
      dataIndex: 'violationOpen',
      key: 'violationOpen',
      width: 120,
      sorter: (a, b) => a.violationOpen - b.violationOpen,
      render: (val: number) => <span className={val > 0 ? 'text-orange-600' : 'text-green-600'}>{val}</span>,
    },
    {
      title: '状态',
      key: 'isActive',
      width: 90,
      render: (_, record) =>
        record.isActive ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>,
    },
  ];

  const serviceSummary = useMemo(
    () => ({
      total: serviceSLA.length,
      active: serviceSLA.filter((s) => s.isActive).length,
      complianceRate: stats.overallComplianceRate || 0,
    }),
    [serviceSLA, stats.overallComplianceRate],
  );

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      {/* 页面头部 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <Title level={2} style={{ marginBottom: 4 }}>
            SLA实时监控
          </Title>
          <Text type="secondary">实时监控SLA执行情况和关键指标</Text>
        </div>
        <Space>
          <Select
            value={refreshInterval}
            onChange={setRefreshInterval}
            style={{ width: 120 }}
            options={[
              { value: 10, label: '10秒' },
              { value: 30, label: '30秒' },
              { value: 60, label: '1分钟' },
              { value: 300, label: '5分钟' },
            ]}
          />
          <Button icon={<RefreshCw size={16} />} onClick={() => fetchStats()} loading={loading}>
            刷新
          </Button>
        </Space>
      </div>

      {/* 统计卡片：Card 高度撑满 Col，保证四张卡片等高 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" style={{ height: '100%' }}>
            <Statistic
              title="SLA总数"
              value={stats.totalDefinitions}
              prefix={<Clock size={18} className="text-blue-500 mr-2" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" style={{ height: '100%' }}>
            <Statistic
              title="启用中"
              value={stats.activeDefinitions}
              prefix={<CheckCircle size={18} className="text-green-500 mr-2" />}
              styles={{ content: { color: '#52c41a' } }}
              suffix={`/ ${stats.totalDefinitions}`}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" style={{ height: '100%' }}>
            <Statistic
              title="违规总数"
              value={stats.totalViolations}
              prefix={<AlertTriangle size={18} className="text-orange-500 mr-2" />}
              styles={{ content: { color: '#fa8c16' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" style={{ height: '100%' }}>
            <Statistic
              title="未解决违规"
              value={stats.openViolations}
              prefix={<AlertTriangle size={18} className="text-red-500 mr-2" />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* SLA监控内容 */}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'overview',
              label: (
                <span className="flex items-center gap-2">
                  <Clock size={16} />
                  概览
                </span>
              ),
              children: (
                <SLAMonitorDashboard
                  autoRefresh={autoRefresh}
                  refreshInterval={refreshInterval}
                />
              ),
            },
            {
              key: 'tickets',
              label: (
                <span className="flex items-center gap-2">
                  <AlertTriangle size={16} />
                  工单SLA ({violations.filter((v) => !v.isResolved).length})
                </span>
              ),
              children: (
                <Table<SLAViolation>
                  columns={ticketColumns}
                  dataSource={violations}
                  rowKey="id"
                  loading={loading}
                  pagination={{ pageSize: 10, showSizeChanger: true }}
                  size="small"
                  scroll={{ x: 1100 }}
                  locale={{ emptyText: '暂无 SLA 违规记录' }}
                />
              ),
            },
            {
              key: 'services',
              label: (
                <span className="flex items-center gap-2">
                  <CheckCircle size={16} />
                  服务SLA ({serviceSLA.length})
                </span>
              ),
              children: (
                <div>
                  {/* 服务SLA 统计：三张卡片等高 */}
                  <Row gutter={[16, 16]} className="mb-4">
                    <Col xs={24} sm={8}>
                      <Card size="small" className="bg-blue-50" style={{ height: '100%' }}>
                        <Statistic
                          title="服务总数"
                          value={serviceSummary.total}
                          prefix={<CheckCircle size={18} className="text-blue-500" />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Card size="small" className="bg-green-50" style={{ height: '100%' }}>
                        <Statistic
                          title="启用中服务"
                          value={serviceSummary.active}
                          prefix={<TrendingUp size={18} className="text-green-500" />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={8}>
                      <Card size="small" className="bg-purple-50" style={{ height: '100%' }}>
                        <Statistic
                          title="总体合规率"
                          value={serviceSummary.complianceRate}
                          precision={1}
                          suffix="%"
                          prefix={<CheckCircle size={18} className="text-purple-500" />}
                          styles={{ content: { color: '#722ed1' } }}
                        />
                      </Card>
                    </Col>
                  </Row>

                  <Table<ServiceSLARow>
                    columns={serviceColumns}
                    dataSource={serviceSLA}
                    rowKey="id"
                    loading={loading}
                    pagination={{ pageSize: 10, showSizeChanger: true }}
                    scroll={{ x: 1000 }}
                    locale={{ emptyText: '暂无 SLA 定义，请先在 SLA 策略中创建' }}
                  />
                </div>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
};

export default SLAMonitorPage;
