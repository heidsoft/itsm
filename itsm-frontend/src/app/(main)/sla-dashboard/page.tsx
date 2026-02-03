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
  Badge,
  List,
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

const { Option } = Select;
const { RangePicker } = DatePicker;

const SLADashboardPage = () => {
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
    if (rate >= 95) return '#52c41a'; // success
    if (rate >= 85) return '#faad14'; // warning
    return '#ff4d4f'; // error
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
          <div className="font-medium">{name}</div>
          <span className="text-xs text-gray-500">
            {record.service_type} - {record.priority}
          </span>
        </div>
      ),
    },
    {
      title: '响应时间',
      dataIndex: 'response_time_minutes',
      key: 'response_time_minutes',
      render: (minutes: number) => (
        <Space>
          <Clock className="w-4 h-4" />
          <span>{minutes}分钟</span>
        </Space>
      ),
    },
    {
      title: '解决时间',
      dataIndex: 'resolution_time_minutes',
      key: 'resolution_time_minutes',
      render: (minutes: number) => (
        <Space>
          <Target className="w-4 h-4" />
          <span>{minutes}分钟</span>
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
      render: (id: number) => <span className="font-mono bg-gray-100 px-1 rounded">#{String(id).padStart(5, '0')}</span>,
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
      render: (minutes: number) => <span className="text-red-500">{minutes}分钟</span>,
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
      render: (date: string) => <span>{new Date(date).toLocaleString()}</span>,
    },
  ];

  return (
    <div className="p-6">
      {/* 页面标题 */}
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900 m-0 flex items-center gap-2">
          <BarChart3 className="text-blue-500" />
          SLA监控仪表盘
        </h2>
        <p className="text-gray-500 mt-1">实时监控服务级别协议的执行情况和合规性</p>
      </div>

      {/* 操作栏 */}
      <Card className="mb-6 rounded-lg shadow-sm border border-gray-200" variant="borderless">
        <Row justify='space-between' align='middle'>
          <Col>
            <Space>
              <RangePicker />
              <Select defaultValue='all' className="w-32">
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
                icon={<RefreshCw className="w-4 h-4" />}
                onClick={() => window.location.reload()}
              >
                刷新
              </Button>
              <Button type='primary' icon={<Bell className="w-4 h-4" />}>
                SLA预警设置
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* SLA概览统计 */}
      <div className="mb-6">
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
              <Statistic
                title='总体合规率'
                value={slaStats.overall_compliance_rate}
                precision={1}
                suffix='%'
                styles={{ content: { color: getComplianceColor(slaStats.overall_compliance_rate) } }}
                prefix={<Target className="w-5 h-5" />}
              />
              <Progress
                percent={slaStats.overall_compliance_rate}
                strokeColor={getComplianceColor(slaStats.overall_compliance_rate)}
                size='small'
                className="mt-2"
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
              <Statistic
                title='活跃SLA定义'
                value={slaStats.active_definitions}
                suffix={`/ ${slaStats.total_definitions}`}
                styles={{ content: { color: '#52c41a' } }}
                prefix={<FileText className="w-5 h-5" />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
              <Statistic
                title='待处理违规'
                value={slaStats.open_violations}
                styles={{ content: { color: '#ff4d4f' } }}
                prefix={<AlertTriangle className="w-5 h-5" />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
              <Statistic
                title='总违规数'
                value={slaStats.total_violations}
                styles={{ content: { color: '#faad14' } }}
                prefix={<Activity className="w-5 h-5" />}
              />
            </Card>
          </Col>
        </Row>
      </div>

      {/* SLA预警和趋势 */}
      <div className="mb-6">
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}>
            <Card
              className="rounded-lg shadow-sm border border-gray-200"
              variant="borderless"
              title={
                <Space>
                  <Bell className="w-4 h-4" />
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
                          <span className="text-xs text-gray-500">
                            剩余时间: {alert.time_remaining}分钟 | 级别: {alert.alert_level}
                          </span>
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
              className="rounded-lg shadow-sm border border-gray-200"
              variant="borderless"
              title={
                <Space>
                  <TrendingUp className="w-4 h-4" />
                  SLA性能趋势
                </Space>
              }
            >
              {slaMetrics && (
                <div>
                  <Row gutter={16} className="mb-4">
                    <Col span={12}>
                      <Statistic
                        title='平均响应时间'
                        value={slaMetrics.response_time_avg}
                        precision={1}
                        suffix='分钟'
                        styles={{ content: { fontSize: 16 } }}
                      />
                    </Col>
                    <Col span={12}>
                      <Statistic
                        title='平均解决时间'
                        value={slaMetrics.resolution_time_avg}
                        precision={1}
                        suffix='分钟'
                        styles={{ content: { fontSize: 16 } }}
                      />
                    </Col>
                  </Row>
                  <div className="h-px bg-gray-200 my-4" />
                  <div className="text-center">
                    <span className="text-gray-500">近7天违规: {slaMetrics.violation_count} 次</span>
                  </div>
                </div>
              )}
            </Card>
          </Col>
        </Row>
      </div>

      {/* 合规报告 */}
      {complianceReport && (
        <div className="mb-6">
          <Row gutter={[16, 16]}>
            <Col span={24}>
              <Card
                className="rounded-lg shadow-sm border border-gray-200"
                variant="borderless"
                title={
                  <Space>
                    <BarChart3 className="w-4 h-4" />
                    SLA合规报告 (近30天)
                  </Space>
                }
              >
                <Row gutter={16}>
                  <Col xs={24} sm={8}>
                    <Statistic
                      title='总工单数'
                      value={complianceReport.total_tickets}
                      prefix={<FileText className="w-4 h-4" />}
                    />
                  </Col>
                  <Col xs={24} sm={8}>
                    <Statistic
                      title='达标工单'
                      value={complianceReport.met_sla}
                      styles={{ content: { color: '#52c41a' } }}
                      prefix={<CheckCircle className="w-4 h-4" />}
                    />
                  </Col>
                  <Col xs={24} sm={8}>
                    <Statistic
                      title='违规工单'
                      value={complianceReport.violated_sla}
                      styles={{ content: { color: '#ff4d4f' } }}
                      prefix={<AlertTriangle className="w-4 h-4" />}
                    />
                  </Col>
                </Row>
                <div className="h-px bg-gray-200 my-4" />
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
        </div>
      )}

      {/* 主要内容区域 - 使用Tabs布局 */}
      <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
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
                        className="rounded-lg shadow-sm border border-gray-200"
                        variant="borderless"
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
                        className="rounded-lg shadow-sm border border-gray-200"
                        variant="borderless"
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
