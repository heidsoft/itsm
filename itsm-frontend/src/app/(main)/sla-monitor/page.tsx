'use client';

import React, { useState, useEffect } from 'react';
import { Card, Typography, Row, Col, Statistic, Button, Space, Select, Tabs, Table, Tag, Progress, Tooltip, message } from 'antd';
import { SLAMonitorDashboard } from '@/components/business/SLAMonitorDashboard';
import { AlertTriangle, CheckCircle, Clock, RefreshCw } from 'lucide-react';
import { WarningOutlined, LineChartOutlined } from '@ant-design/icons';
import { SLAApi } from '@/lib/api/sla-api';

const { Title, Text } = Typography;

const priorityColors: Record<string, string> = {
  '紧急': 'red',
  '高': 'orange',
  '中': 'blue',
  '低': 'default',
};

const SLAMonitorPage = () => {
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [refreshInterval, setRefreshInterval] = useState(30);
  const [activeTab, setActiveTab] = useState('overview');
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState({
    totalDefinitions: 0,
    activeDefinitions: 0,
    atRiskTickets: 0,
    totalViolations: 0,
    openViolations: 0,
    overallComplianceRate: 0,
  });
  const [violations, setViolations] = useState<any[]>([]);
  const [serviceSLA, setServiceSLA] = useState<any[]>([]);

  // Fetch stats
  const fetchStats = async () => {
    setLoading(true);
    try {
      const [statsData, monitoringData, violationsData] = await Promise.all([
        SLAApi.getSLAStats(),
        SLAApi.getSLAMonitoring({ start_time: '30d', end_time: 'now' }),
        SLAApi.getSLAViolations({ page: 1, size: 20, is_resolved: false })
      ]);

      setStats({
        totalDefinitions: statsData.totalDefinitions || 0,
        activeDefinitions: statsData.activeDefinitions || 0,
        atRiskTickets: Math.round((monitoringData.atRiskTickets) || 0),
        totalViolations: statsData.totalViolations || 0,
        openViolations: statsData.openViolations || 0,
        overallComplianceRate: statsData.overallComplianceRate || 0,
      });

      // Set violations for ticket SLA table
      if (violationsData?.items) {
        setViolations(violationsData.items.map((v: any) => ({
          id: v.id,
          ticketNo: v.ticketNumber || v.ticket_number || `#${v.ticketId || v.ticket_id}`,
          title: v.ticketTitle || 'Unknown',
          priority: v.priority || 'medium',
          status: v.isResolved ? '已解决' : '处理中',
          slaTarget: v.slaName || 'SLA',
          remaining: v.isResolved ? '已解决' : `${v.delayMinutes || 0}分钟`,
          progress: Math.min(100, Math.round((v.delayMinutes || 0) / 60 * 100)),
        })));
      }

      // Fetch service SLA data from definitions
      const slaDefinitions = await SLAApi.getSLADefinitions({ size: 100 });
      if (slaDefinitions?.items) {
        const services = slaDefinitions.items.map((sla: any) => ({
          id: sla.id,
          serviceName: sla.name,
          totalRequests: 0,
          compliant: 0,
          atRisk: 0,
          breached: 0,
          complianceRate: sla.isActive ? 95 : 0,
          avgResponse: `${sla.responseTimeMinutes || 0}分钟`,
        }));
        setServiceSLA(services);
      }
    } catch (error) {
      console.error('Failed to fetch SLA stats:', error);
      message.error('获取SLA统计数据失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  const ticketColumns = [
    {
      title: '工单号',
      dataIndex: 'ticketNo',
      key: 'ticketNo',
      render: (text: string) => <a>{text}</a>,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      render: (priority: string) => <Tag color={priorityColors[priority]}>{priority}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
    },
    {
      title: 'SLA目标',
      dataIndex: 'slaTarget',
      key: 'slaTarget',
    },
    {
      title: '剩余时间',
      dataIndex: 'remaining',
      key: 'remaining',
      render: (remaining: string) => (
        <span className={remaining === '已超时' ? 'text-red-500' : ''}>
          {remaining}
        </span>
      ),
    },
    {
      title: '进度',
      dataIndex: 'progress',
      key: 'progress',
      render: (progress: number) => (
        <Progress
          percent={progress}
          size="small"
          status={progress >= 100 ? 'exception' : progress >= 80 ? 'normal' : 'active'}
        />
      ),
    },
  ];

  const serviceColumns = [
    {
      title: '服务名称',
      dataIndex: 'serviceName',
      key: 'serviceName',
      render: (text: string) => <a>{text}</a>,
    },
    {
      title: '总请求数',
      dataIndex: 'totalRequests',
      key: 'totalRequests',
      sorter: (a: any, b: any) => a.totalRequests - b.totalRequests,
    },
    {
      title: '合规',
      dataIndex: 'compliant',
      key: 'compliant',
      render: (val: number) => <span className="text-green-600">{val}</span>,
    },
    {
      title: '风险中',
      dataIndex: 'atRisk',
      key: 'atRisk',
      render: (val: number) => <span className="text-orange-600">{val}</span>,
    },
    {
      title: '已超时',
      dataIndex: 'breached',
      key: 'breached',
      render: (val: number) => <span className="text-red-600">{val}</span>,
    },
    {
      title: '合规率',
      dataIndex: 'complianceRate',
      key: 'complianceRate',
      render: (rate: number) => (
        <Progress
          percent={rate}
          size="small"
          format={(percent) => `${percent}%`}
          status={rate === undefined ? 'exception' : rate >= 95 ? 'success' : rate >= 90 ? 'normal' : 'exception'}
        />
      ),
    },
    {
      title: '平均响应时间',
      dataIndex: 'avgResponse',
      key: 'avgResponse',
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Button size="small" type="link">详情</Button>
      ),
    },
  ];

  // Calculate summary stats for service SLA
  const serviceSummary = React.useMemo(() => {
    const total = serviceSLA.reduce((sum, s) => sum + s.totalRequests, 0);
    const compliant = serviceSLA.reduce((sum, s) => sum + s.compliant, 0);
    const rate = total > 0 ? (compliant / total * 100).toFixed(1) : (stats.overallComplianceRate || 0).toString();
    return { total, compliant, rate };
  }, [serviceSLA, stats.overallComplianceRate]);

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      {/* 页面头部 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <Title level={2} style={{ marginBottom: 4 }}>
            SLA实时监控
          </Title>
          <Text type="secondary">
            实时监控SLA执行情况和关键指标
          </Text>
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
          <Button
            icon={<RefreshCw className="w-4 h-4" />}
            onClick={() => fetchStats()}
          >
            刷新
          </Button>
        </Space>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="SLA总数"
              value={stats.totalDefinitions}
              prefix={<Clock className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="合规"
              value={stats.activeDefinitions}
              prefix={<CheckCircle className="text-green-500 mr-2" />}
              valueStyle={{ color: '#52c41a' }}
              suffix={`/ ${stats.totalDefinitions}`}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="风险中"
              value={stats.atRiskTickets}
              prefix={<WarningOutlined className="text-orange-500 mr-2" />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="已超时"
              value={stats.openViolations}
              prefix={<AlertTriangle className="text-red-500 mr-2" />}
              valueStyle={{ color: '#ff4d4f' }}
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
                  <Clock />
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
                  <AlertTriangle />
                  工单SLA ({violations.length})
                </span>
              ),
              children: (
                <div>
                  <Table
                    columns={ticketColumns}
                    dataSource={violations}
                    rowKey="id"
                    pagination={false}
                    size="small"
                  />
                </div>
              ),
            },
            {
              key: 'services',
              label: (
                <span className="flex items-center gap-2">
                  <CheckCircle />
                  服务SLA ({serviceSLA.length})
                </span>
              ),
              children: (
                <div>
                  {/* 服务SLA 统计 */}
                  <Row gutter={16} className="mb-4">
                    <Col span={8}>
                      <Card size="small" className="bg-blue-50">
                        <Statistic
                          title="服务总数"
                          value={serviceSLA.length}
                          prefix={<CheckCircle />}
                        />
                      </Card>
                    </Col>
                    <Col span={8}>
                      <Card size="small" className="bg-green-50">
                        <Statistic
                          title="总请求数"
                          value={serviceSummary.total}
                          prefix={<LineChartOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={8}>
                      <Card size="small" className="bg-purple-50">
                        <Statistic
                          title="总体合规率"
                          value={Number(serviceSummary.rate)}
                          suffix="%"
                          prefix={<CheckCircle className="text-purple-500" />}
                          valueStyle={{ color: '#722ed1' }}
                        />
                      </Card>
                    </Col>
                  </Row>

                  <Table
                    columns={serviceColumns}
                    dataSource={serviceSLA}
                    rowKey="id"
                    pagination={{ pageSize: 10 }}
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
