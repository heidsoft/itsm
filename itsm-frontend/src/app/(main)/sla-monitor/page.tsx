'use client';

import React, { useState, useEffect } from 'react';
import { Card, Typography, Row, Col, Statistic, Button, Space, Select, Tabs, Table, Tag, Progress, Tooltip } from 'antd';
import { SLAMonitorDashboard } from '@/components/business/SLAMonitorDashboard';
import { AlertTriangle, CheckCircle, Clock, RefreshCw } from 'lucide-react';
import { WarningOutlined, LineChartOutlined } from '@ant-design/icons';

const { Title, Text } = Typography;

// Mock SLA ticket data
const mockSLATickets = [
  { id: 1, ticket_no: 'INC-001', title: '服务器宕机', priority: '紧急', status: '响应中', sla_target: '1小时', remaining: '15分钟', progress: 85 },
  { id: 2, ticket_no: 'INC-002', title: '网络无法访问', priority: '高', status: '处理中', sla_target: '2小时', remaining: '45分钟', progress: 60 },
  { id: 3, ticket_no: 'INC-003', title: '应用响应慢', priority: '中', status: '已响应', sla_target: '4小时', remaining: '2小时', progress: 30 },
  { id: 4, ticket_no: 'SR-001', title: '账号申请', priority: '低', status: '处理中', sla_target: '8小时', remaining: '6小时', progress: 20 },
  { id: 5, ticket_no: 'INC-004', title: '数据库连接失败', priority: '紧急', status: '响应中', sla_target: '1小时', remaining: '已超时', progress: 100 },
];

// Mock Service SLA data
const mockServiceSLA = [
  { id: 1, service_name: '云服务器 ECS', total_requests: 1250, compliant: 1180, at_risk: 45, breached: 25, compliance_rate: 94.4, avg_response: '12分钟' },
  { id: 2, service_name: '云数据库 RDS', total_requests: 856, compliant: 825, at_risk: 20, breached: 11, compliance_rate: 96.4, avg_response: '8分钟' },
  { id: 3, service_name: '对象存储 OSS', total_requests: 542, compliant: 530, at_risk: 8, breached: 4, compliance_rate: 97.8, avg_response: '5分钟' },
  { id: 4, service_name: '负载均衡 SLB', total_requests: 320, compliant: 298, at_risk: 15, breached: 7, compliance_rate: 93.1, avg_response: '15分钟' },
  { id: 5, service_name: 'VPN 网关', total_requests: 180, compliant: 175, at_risk: 3, breached: 2, compliance_rate: 97.2, avg_response: '3分钟' },
  { id: 6, service_name: '企业邮箱', total_requests: 95, compliant: 93, at_risk: 1, breached: 1, compliance_rate: 97.9, avg_response: '10分钟' },
];

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
  const [stats, setStats] = useState({
    totalSLA: 0,
    compliant: 0,
    atRisk: 0,
    breached: 0,
    complianceRate: 0,
  });

  // Fetch stats
  const fetchStats = async () => {
    try {
      // Mock data - in real app would call API
      setStats({
        totalSLA: 45,
        compliant: 38,
        atRisk: 5,
        breached: 2,
        complianceRate: 84.4,
      });
    } catch (error) {
      console.error('Failed to fetch SLA stats:', error);
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  const ticketColumns = [
    {
      title: '工单号',
      dataIndex: 'ticket_no',
      key: 'ticket_no',
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
      dataIndex: 'sla_target',
      key: 'sla_target',
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
      dataIndex: 'service_name',
      key: 'service_name',
      render: (text: string) => <a>{text}</a>,
    },
    {
      title: '总请求数',
      dataIndex: 'total_requests',
      key: 'total_requests',
      sorter: (a: any, b: any) => a.total_requests - b.total_requests,
    },
    {
      title: '合规',
      dataIndex: 'compliant',
      key: 'compliant',
      render: (val: number) => <span className="text-green-600">{val}</span>,
    },
    {
      title: '风险中',
      dataIndex: 'at_risk',
      key: 'at_risk',
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
      dataIndex: 'compliance_rate',
      key: 'compliance_rate',
      render: (rate: number) => (
        <Progress
          percent={rate}
          size="small"
          format={(p) => `${p}%`}
          status={p === undefined ? 'exception' : p >= 95 ? 'success' : p >= 90 ? 'normal' : 'exception'}
        />
      ),
    },
    {
      title: '平均响应时间',
      dataIndex: 'avg_response',
      key: 'avg_response',
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
    const total = mockServiceSLA.reduce((sum, s) => sum + s.total_requests, 0);
    const compliant = mockServiceSLA.reduce((sum, s) => sum + s.compliant, 0);
    const rate = total > 0 ? (compliant / total * 100).toFixed(1) : 0;
    return { total, compliant, rate };
  }, []);

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
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="SLA总数"
              value={stats.totalSLA}
              prefix={<Clock className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="合规"
              value={stats.compliant}
              prefix={<CheckCircle className="text-green-500 mr-2" />}
              valueStyle={{ color: '#52c41a' }}
              suffix={`/ ${stats.totalSLA}`}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="风险中"
              value={stats.atRisk}
              prefix={<WarningOutlined className="text-orange-500 mr-2" />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="已超时"
              value={stats.breached}
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
                  工单SLA ({mockSLATickets.length})
                </span>
              ),
              children: (
                <div>
                  <Table
                    columns={ticketColumns}
                    dataSource={mockSLATickets}
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
                  服务SLA ({mockServiceSLA.length})
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
                          value={mockServiceSLA.length}
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
                    dataSource={mockServiceSLA}
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
