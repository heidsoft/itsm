'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Progress,
  Table,
  Tag,
  Button,
  Select,
  DatePicker,
  Space,
  Typography,
  Divider,
  Badge,
  List,
  theme,
  Tabs,
} from 'antd';
import {
  AlertTriangle,
  Clock,
  CheckCircle,
  TrendingUp,
  Activity,
  Target,
  Bell,
  BarChart3,
  FileText,
  RefreshCw,
  BarChart,
  Shield,
} from 'lucide-react';
import SLAApi from '@/lib/api/sla-api';
import type { SLADefinition, SLAViolation, SLAComplianceReport } from '@/lib/api/sla-api';
import SLADashboardCharts from '@/components/charts/SLADashboardCharts';
import SLAViolationMonitor from '@/components/business/SLAViolationMonitor';
import { SLAMonitorDashboard } from '@/components/business/SLAMonitorDashboard';

const { Title, Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;

const SLADashboardPage = () => {
  const { token } = theme.useToken();

  // 状态管理
  const [slaStats, setSlaStats] = useState({
    total_definitions: 0,
    active_definitions: 0,
    total_violations: 0,
    open_violations: 0,
    overall_compliance_rate: 95.5,
  });
  const [slaDefinitions, setSlaDefinitions] = useState<SLADefinition[]>([]);
  const [slaViolations, setSlaViolations] = useState<SLAViolation[]>([]);
  const [complianceReport, setComplianceReport] = useState<SLAComplianceReport | null>(null);
  const [slaAlerts, setSlaAlerts] = useState<
    Array<{
      ticket_id: number;
      ticket_title: string;
      priority: string;
      sla_definition: string;
      time_remaining: number;
      alert_level: 'warning' | 'critical';
      created_at: string;
    }>
  >([]);
  const [slaMetrics, setSlaMetrics] = useState<{
    response_time_avg: number;
    resolution_time_avg: number;
    compliance_rate: number;
    violation_count: number;
  } | null>(null);

  // 加载SLA统计数据
  const loadSLAStats = async () => {
    try {
      const stats = await SLAApi.getSLAStats();
      setSlaStats(stats);
    } catch (error) {
      console.error('加载SLA统计失败:', error);
    }
  };

  // 加载SLA定义
  const loadSLADefinitions = async () => {
    try {
      const response = await SLAApi.getSLADefinitions({ page: 1, page_size: 10 });
      setSlaDefinitions(response.items);
    } catch (error) {
      console.error('加载SLA定义失败:', error);
    }
  };

  // 加载SLA违规
  const loadSLAViolations = async () => {
    try {
      const response = await SLAApi.getSLAViolations({ page: 1, page_size: 10, status: 'open' });
      setSlaViolations(response.items);
    } catch (error) {
      console.error('加载SLA违规失败:', error);
    }
  };

  // 加载SLA预警
  const loadSLAAlerts = async () => {
    try {
      const alerts = await SLAApi.getSLAAlerts();
      setSlaAlerts(alerts);
    } catch (error) {
      console.error('加载SLA预警失败:', error);
    }
  };

  // 加载SLA指标
  const loadSLAMetrics = async () => {
    try {
      const metrics = await SLAApi.getSLAMetrics({ period: 'week' });
      setSlaMetrics(metrics);
    } catch (error) {
      console.error('加载SLA指标失败:', error);
    }
  };

  // 加载合规报告
  const loadComplianceReport = async () => {
    try {
      const endDate = new Date();
      const startDate = new Date();
      startDate.setDate(endDate.getDate() - 30); // 最近30天

      const report = await SLAApi.getSLAComplianceReport({
        start_date: startDate.toISOString(),
        end_date: endDate.toISOString(),
      });
      setComplianceReport(report);
    } catch (error) {
      console.error('加载合规报告失败:', error);
    }
  };

  useEffect(() => {
    const loadAllData = async () => {
      await Promise.all([
        loadSLAStats(),
        loadSLADefinitions(),
        loadSLAViolations(),
        loadSLAAlerts(),
        loadSLAMetrics(),
        loadComplianceReport(),
      ]);
    };

    loadAllData();
  }, []);

  // 获取合规率颜色
  const getComplianceColor = (rate: number) => {
    if (rate >= 95) return token.colorSuccess;
    if (rate >= 85) return token.colorWarning;
    return token.colorError;
  };

  // 获取违规级别颜色
  const getViolationLevelColor = (level: string) => {
    switch (level) {
      case 'critical':
        return 'red';
      case 'warning':
        return 'orange';
      default:
        return 'blue';
    }
  };

  // SLA定义表格列
  const slaDefinitionColumns = [
    {
      title: 'SLA名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: SLADefinition) => (
        <div>
          <div style={{ fontWeight: 500 }}>{name}</div>
          <Text type='secondary' style={{ fontSize: 12 }}>
            {record.service_type} - {record.priority}
          </Text>
        </div>
      ),
    },
    {
      title: '响应时间',
      dataIndex: 'response_time_minutes',
      key: 'response_time_minutes',
      render: (minutes: number) => (
        <Space>
          <Clock style={{ width: 16, height: 16 }} />
          <Text>{minutes}分钟</Text>
        </Space>
      ),
    },
    {
      title: '解决时间',
      dataIndex: 'resolution_time_minutes',
      key: 'resolution_time_minutes',
      render: (minutes: number) => (
        <Space>
          <Target style={{ width: 16, height: 16 }} />
          <Text>{minutes}分钟</Text>
        </Space>
      ),
    },
    {
      title: '可用性目标',
      dataIndex: 'availability_target',
      key: 'availability_target',
      render: (target: number) => <Tag color='blue'>{target}%</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (isActive: boolean) => (
        <Tag color={isActive ? 'green' : 'red'}>{isActive ? '激活' : '禁用'}</Tag>
      ),
    },
  ];

  // SLA违规表格列
  const slaViolationColumns = [
    {
      title: '工单ID',
      dataIndex: 'ticket_id',
      key: 'ticket_id',
      render: (id: number) => <Text code>#{String(id).padStart(5, '0')}</Text>,
    },
    {
      title: '违规类型',
      dataIndex: 'violation_type',
      key: 'violation_type',
      render: (type: string) => (
        <Tag color={type === 'response_time' ? 'orange' : 'red'}>
          {type === 'response_time' ? '响应超时' : '解决超时'}
        </Tag>
      ),
    },
    {
      title: '延迟时间',
      dataIndex: 'delay_minutes',
      key: 'delay_minutes',
      render: (minutes: number) => <Text type='danger'>{minutes}分钟</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'open' ? 'red' : 'green'}>
          {status === 'open' ? '待处理' : '已处理'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => <Text>{new Date(date).toLocaleString()}</Text>,
    },
  ];

  return (
    <div style={{ padding: token.padding }}>
      {/* 页面标题 */}
      <div style={{ marginBottom: token.marginLG }}>
        <Title
          level={2}
          style={{ margin: 0, display: 'flex', alignItems: 'center', gap: token.marginSM }}
        >
          <BarChart3 style={{ color: token.colorPrimary }} />
          SLA监控仪表盘
        </Title>
        <Text type='secondary'>实时监控服务级别协议的执行情况和合规性</Text>
      </div>

      {/* 操作栏 */}
      <Card style={{ marginBottom: token.marginLG }}>
        <Row justify='space-between' align='middle'>
          <Col>
            <Space>
              <RangePicker />
              <Select defaultValue='all' style={{ width: 120 }}>
                <Option value='all'>全部服务</Option>
                <Option value='it'>IT服务</Option>
                <Option value='hr'>HR服务</Option>
                <Option value='finance'>财务服务</Option>
              </Select>
            </Space>
          </Col>
          <Col>
            <Space>
              <Button
                icon={<RefreshCw style={{ width: 16, height: 16 }} />}
                onClick={() => window.location.reload()}
              >
                刷新
              </Button>
              <Button type='primary' icon={<Bell style={{ width: 16, height: 16 }} />}>
                SLA预警设置
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* SLA概览统计 */}
      <Row gutter={[16, 16]} style={{ marginBottom: token.marginLG }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title='总体合规率'
              value={slaStats.overall_compliance_rate}
              precision={1}
              suffix='%'
              valueStyle={{ color: getComplianceColor(slaStats.overall_compliance_rate) }}
              prefix={<Target style={{ width: 20, height: 20 }} />}
            />
            <Progress
              percent={slaStats.overall_compliance_rate}
              strokeColor={getComplianceColor(slaStats.overall_compliance_rate)}
              size='small'
              style={{ marginTop: token.marginSM }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title='活跃SLA定义'
              value={slaStats.active_definitions}
              suffix={`/ ${slaStats.total_definitions}`}
              valueStyle={{ color: token.colorSuccess }}
              prefix={<FileText style={{ width: 20, height: 20 }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title='待处理违规'
              value={slaStats.open_violations}
              valueStyle={{ color: token.colorError }}
              prefix={<AlertTriangle style={{ width: 20, height: 20 }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title='总违规数'
              value={slaStats.total_violations}
              valueStyle={{ color: token.colorWarning }}
              prefix={<Activity style={{ width: 20, height: 20 }} />}
            />
          </Card>
        </Col>
      </Row>

      {/* SLA预警和趋势 */}
      <Row gutter={[16, 16]} style={{ marginBottom: token.marginLG }}>
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <Bell style={{ width: 16, height: 16 }} />
                SLA预警
              </Space>
            }
            extra={<Badge count={slaAlerts.length} />}
          >
            <List
              dataSource={slaAlerts.slice(0, 5)}
              renderItem={alert => (
                <List.Item>
                  <List.Item.Meta
                    avatar={
                      <AlertTriangle
                        style={{
                          width: 20,
                          height: 20,
                          color: getViolationLevelColor(alert.alert_level),
                        }}
                      />
                    }
                    title={`工单 #${String(alert.ticket_id).padStart(5, '0')}`}
                    description={
                      <div>
                        <div>{alert.ticket_title}</div>
                        <Text type='secondary' style={{ fontSize: 12 }}>
                          剩余时间: {alert.time_remaining}分钟 | 级别: {alert.alert_level}
                        </Text>
                      </div>
                    }
                  />
                </List.Item>
              )}
              locale={{ emptyText: '暂无SLA预警' }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <TrendingUp style={{ width: 16, height: 16 }} />
                SLA性能趋势
              </Space>
            }
          >
            {slaMetrics && (
              <div>
                <Row gutter={16} style={{ marginBottom: token.marginMD }}>
                  <Col span={12}>
                    <Statistic
                      title='平均响应时间'
                      value={slaMetrics.response_time_avg}
                      precision={1}
                      suffix='分钟'
                      valueStyle={{ fontSize: 16 }}
                    />
                  </Col>
                  <Col span={12}>
                    <Statistic
                      title='平均解决时间'
                      value={slaMetrics.resolution_time_avg}
                      precision={1}
                      suffix='分钟'
                      valueStyle={{ fontSize: 16 }}
                    />
                  </Col>
                </Row>
                <Divider />
                <div style={{ textAlign: 'center' }}>
                  <Text type='secondary'>近7天违规: {slaMetrics.violation_count} 次</Text>
                </div>
              </div>
            )}
          </Card>
        </Col>
      </Row>

      {/* 合规报告 */}
      {complianceReport && (
        <Row gutter={[16, 16]} style={{ marginBottom: token.marginLG }}>
          <Col span={24}>
            <Card
              title={
                <Space>
                  <BarChart3 style={{ width: 16, height: 16 }} />
                  SLA合规报告 (近30天)
                </Space>
              }
            >
              <Row gutter={16}>
                <Col xs={24} sm={8}>
                  <Statistic
                    title='总工单数'
                    value={complianceReport.total_tickets}
                    prefix={<FileText style={{ width: 16, height: 16 }} />}
                  />
                </Col>
                <Col xs={24} sm={8}>
                  <Statistic
                    title='达标工单'
                    value={complianceReport.met_sla}
                    valueStyle={{ color: token.colorSuccess }}
                    prefix={<CheckCircle style={{ width: 16, height: 16 }} />}
                  />
                </Col>
                <Col xs={24} sm={8}>
                  <Statistic
                    title='违规工单'
                    value={complianceReport.violated_sla}
                    valueStyle={{ color: token.colorError }}
                    prefix={<AlertTriangle style={{ width: 16, height: 16 }} />}
                  />
                </Col>
              </Row>
              <Divider />
              <Row gutter={16}>
                <Col xs={24} sm={12}>
                  <Statistic
                    title='平均响应时间'
                    value={complianceReport.avg_response_time}
                    precision={1}
                    suffix='分钟'
                  />
                </Col>
                <Col xs={24} sm={12}>
                  <Statistic
                    title='平均解决时间'
                    value={complianceReport.avg_resolution_time}
                    precision={1}
                    suffix='分钟'
                  />
                </Col>
              </Row>
            </Card>
          </Col>
        </Row>
      )}

      {/* 主要内容区域 - 使用Tabs布局 */}
      <Card>
        <Tabs
          defaultActiveKey="overview"
          items={[
            {
              key: 'overview',
              label: (
                <Space>
                  <BarChart3 size={16} />
                  概览
                </Space>
              ),
              children: (
                <div className="space-y-6">
                  {/* SLA定义和违规表格 */}
                  <Row gutter={[16, 16]}>
                    <Col xs={24} lg={12}>
                      <Card
                        title='SLA定义'
                        extra={
                          <Button type='primary' size='small'>
                            管理SLA
                          </Button>
                        }
                      >
                        <Table
                          dataSource={slaDefinitions}
                          columns={slaDefinitionColumns}
                          rowKey='id'
                          pagination={{ pageSize: 5, size: 'small' }}
                          size='small'
                        />
                      </Card>
                    </Col>
                    <Col xs={24} lg={12}>
                      <Card
                        title='最近SLA违规'
                        extra={
                          <Button type='primary' size='small'>
                            查看全部
                          </Button>
                        }
                      >
                        <Table
                          dataSource={slaViolations}
                          columns={slaViolationColumns}
                          rowKey='id'
                          pagination={{ pageSize: 5, size: 'small' }}
                          size='small'
                        />
                      </Card>
                    </Col>
                  </Row>
                </div>
              ),
            },
            {
              key: 'charts',
              label: (
                <Space>
                  <BarChart size={16} />
                  数据分析
                </Space>
              ),
              children: <SLADashboardCharts />,
            },
            {
              key: 'violations',
              label: (
                <Space>
                  <AlertTriangle size={16} />
                  违规监控
                </Space>
              ),
              children: <SLAViolationMonitor />,
            },
            {
              key: 'monitor',
              label: (
                <Space>
                  <Shield size={16} />
                  实时监控
                </Space>
              ),
              children: <SLAMonitorDashboard autoRefresh={true} refreshInterval={30} />,
            },
          ]}
        />
      </Card>
    </div>
  );
};

export default SLADashboardPage;
