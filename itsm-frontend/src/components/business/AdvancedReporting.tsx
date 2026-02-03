'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Button,
  Space,
  Typography,
  Tabs,
  Form,
  Select,
  DatePicker,
  Input,
  Table,
  Tag,
  Modal,
  message,
  Switch,
  Popconfirm,
  Divider,
  Statistic,
  Progress,
  Alert,
  Spin,
  Empty,
  Upload,
  Dropdown,
  Menu,
} from 'antd';
import {
  BarChart3,
  TrendingUp,
  TrendingDown,
  Download,
  Eye,
  Filter,
  Calendar,
  Users,
  Clock,
  CheckCircle,
  AlertTriangle,
  Plus,
  Edit,
  Trash2,
  Save,
  Share2,
  RefreshCw,
  FileText,
  PieChart,
  LineChart,
  Settings,
  Database,
  CloudDownload,
  BarChart,
  Activity,
} from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;
const { TabPane } = Tabs;
const { RangePicker } = DatePicker;
const { TextArea } = Input;

interface ReportDefinition {
  id: number;
  name: string;
  description: string;
  category: 'ticket' | 'sla' | 'user' | 'system' | 'custom';
  data_source: string;
  query: string;
  parameters: ReportParameter[];
  visualization: 'table' | 'chart' | 'dashboard';
  chart_type?: 'bar' | 'line' | 'pie' | 'area' | 'scatter';
  schedule?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  created_by: string;
}

interface ReportParameter {
  name: string;
  type: 'string' | 'number' | 'date' | 'select' | 'boolean';
  label: string;
  required: boolean;
  default_value?: any;
  options?: { label: string; value: any }[];
}

interface ReportExecution {
  id: number;
  report_id: number;
  report_name: string;
  status: 'running' | 'completed' | 'failed';
  started_at: string;
  completed_at?: string;
  result_count?: number;
  error_message?: string;
  created_by: string;
}

interface ReportData {
  columns: string[];
  rows: any[][];
  total: number;
  summary: Record<string, any>;
}

interface DashboardWidget {
  id: string;
  title: string;
  type: 'chart' | 'metric' | 'table';
  config: any;
  position: { x: number; y: number; w: number; h: number };
}

const AdvancedReporting: React.FC = () => {
  const [activeTab, setActiveTab] = useState('reports');
  const [reports, setReports] = useState<ReportDefinition[]>([]);
  const [executions, setExecutions] = useState<ReportExecution[]>([]);
  const [currentReport, setCurrentReport] = useState<ReportDefinition | null>(null);
  const [reportData, setReportData] = useState<ReportData | null>(null);
  const [loading, setLoading] = useState(false);
  const [showReportModal, setShowReportModal] = useState(false);
  const [showExecutionModal, setShowExecutionModal] = useState(false);
  const [selectedReport, setSelectedReport] = useState<ReportDefinition | null>(null);
  const [form] = Form.useForm();
  const [executionForm] = Form.useForm();
  const [dashboardWidgets, setDashboardWidgets] = useState<DashboardWidget[]>([]);
  const [isEditing, setIsEditing] = useState(false);

  // 模拟数据
  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));

      // 模拟报表定义数据
      const mockReports: ReportDefinition[] = [
        {
          id: 1,
          name: '工单处理效率分析',
          description: '分析工单处理效率、响应时间和解决时间',
          category: 'ticket',
          data_source: 'tickets',
          query:
            'SELECT * FROM tickets WHERE created_at >= :start_date AND created_at <= :end_date',
          parameters: [
            {
              name: 'start_date',
              type: 'date',
              label: '开始日期',
              required: true,
            },
            {
              name: 'end_date',
              type: 'date',
              label: '结束日期',
              required: true,
            },
            {
              name: 'priority',
              type: 'select',
              label: '优先级',
              required: false,
              options: [
                { label: '高', value: 'high' },
                { label: '中', value: 'medium' },
                { label: '低', value: 'low' },
              ],
            },
          ],
          visualization: 'chart',
          chart_type: 'bar',
          is_active: true,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          created_by: '张三',
        },
        {
          id: 2,
          name: 'SLA合规率报表',
          description: '统计SLA合规率、违规情况和趋势分析',
          category: 'sla',
          data_source: 'sla_metrics',
          query: 'SELECT * FROM sla_metrics WHERE period = :period',
          parameters: [
            {
              name: 'period',
              type: 'select',
              label: '统计周期',
              required: true,
              options: [
                { label: '日', value: 'daily' },
                { label: '周', value: 'weekly' },
                { label: '月', value: 'monthly' },
              ],
              default_value: 'monthly',
            },
          ],
          visualization: 'dashboard',
          is_active: true,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          created_by: '李四',
        },
        {
          id: 3,
          name: '用户工作量统计',
          description: '统计用户工作量、处理工单数量和效率',
          category: 'user',
          data_source: 'user_workload',
          query: 'SELECT * FROM user_workload WHERE user_id = :user_id',
          parameters: [
            {
              name: 'user_id',
              type: 'select',
              label: '用户',
              required: true,
              options: [
                { label: '张三', value: '1' },
                { label: '李四', value: '2' },
                { label: '王五', value: '3' },
              ],
            },
          ],
          visualization: 'table',
          is_active: true,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          created_by: '王五',
        },
      ];

      const mockExecutions: ReportExecution[] = [
        {
          id: 1,
          report_id: 1,
          report_name: '工单处理效率分析',
          status: 'completed',
          started_at: '2024-01-15T10:00:00Z',
          completed_at: '2024-01-15T10:05:00Z',
          result_count: 1250,
          created_by: '张三',
        },
        {
          id: 2,
          report_id: 2,
          report_name: 'SLA合规率报表',
          status: 'running',
          started_at: '2024-01-15T10:30:00Z',
          created_by: '李四',
        },
      ];

      setReports(mockReports);
      setExecutions(mockExecutions);

      // 初始化仪表盘组件
      const initialWidgets: DashboardWidget[] = [
        {
          id: 'widget1',
          title: '工单处理趋势',
          type: 'chart',
          config: { chartType: 'line', dataSource: 'ticket_trends' },
          position: { x: 0, y: 0, w: 6, h: 4 },
        },
        {
          id: 'widget2',
          title: 'SLA合规率',
          type: 'metric',
          config: { metric: 'sla_compliance', format: 'percentage' },
          position: { x: 6, y: 0, w: 3, h: 2 },
        },
        {
          id: 'widget3',
          title: '用户工作量排行',
          type: 'table',
          config: { dataSource: 'user_workload_ranking', maxRows: 5 },
          position: { x: 9, y: 0, w: 3, h: 4 },
        },
      ];

      setDashboardWidgets(initialWidgets);
    } catch (error) {
      console.error('加载报表数据失败:', error);
      message.error('加载数据失败');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateReport = async () => {
    try {
      const values = await form.validateFields();

      const newReport: ReportDefinition = {
        id: Date.now(),
        ...values,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        created_by: '当前用户',
      };

      setReports(prev => [...prev, newReport]);
      message.success('报表创建成功');
      setShowReportModal(false);
      form.resetFields();
    } catch (error) {
      console.error('创建报表失败:', error);
    }
  };

  const handleEditReport = async () => {
    try {
      const values = await form.validateFields();

      if (selectedReport) {
        setReports(prev =>
          prev.map(r =>
            r.id === selectedReport.id
              ? { ...r, ...values, updated_at: new Date().toISOString() }
              : r
          )
        );
        message.success('报表更新成功');
        setShowReportModal(false);
        setSelectedReport(null);
        form.resetFields();
      }
    } catch (error) {
      console.error('更新报表失败:', error);
    }
  };

  const handleDeleteReport = async (id: number) => {
    try {
      setReports(prev => prev.filter(r => r.id !== id));
      message.success('报表删除成功');
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleExecuteReport = async (report: ReportDefinition) => {
    try {
      setCurrentReport(report);
      setLoading(true);

      // 模拟执行报表
      await new Promise(resolve => setTimeout(resolve, 2000));

      // 模拟报表数据
      const mockData: ReportData = {
        columns: ['日期', '工单数量', '平均响应时间', '平均解决时间', 'SLA合规率'],
        rows: [
          ['2024-01-01', 45, '2.3小时', '8.5小时', '95.6%'],
          ['2024-01-02', 52, '2.1小时', '7.8小时', '96.2%'],
          ['2024-01-03', 38, '2.5小时', '9.2小时', '94.8%'],
          ['2024-01-04', 61, '1.9小时', '7.1小时', '97.1%'],
          ['2024-01-05', 47, '2.2小时', '8.1小时', '95.9%'],
        ],
        total: 5,
        summary: {
          total_tickets: 243,
          avg_response_time: '2.2小时',
          avg_resolution_time: '8.1小时',
          overall_sla_compliance: '95.9%',
        },
      };

      setReportData(mockData);
      setShowExecutionModal(true);
    } catch (error) {
      console.error('执行报表失败:', error);
      message.error('执行报表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleExportReport = async (format: 'excel' | 'csv' | 'pdf') => {
    try {
      // 模拟导出
      await new Promise(resolve => setTimeout(resolve, 1000));

      const formatNames = {
        excel: 'Excel',
        csv: 'CSV',
        pdf: 'PDF',
      };

      message.success(`${formatNames[format]}格式导出成功`);
    } catch (error) {
      message.error('导出失败');
    }
  };

  const handleSaveDashboard = async () => {
    try {
      // 模拟保存仪表盘配置
      await new Promise(resolve => setTimeout(resolve, 500));
      message.success('仪表盘配置保存成功');
      setIsEditing(false);
    } catch (error) {
      message.error('保存失败');
    }
  };

  const getCategoryColor = (category: string) => {
    switch (category) {
      case 'ticket':
        return 'blue';
      case 'sla':
        return 'green';
      case 'user':
        return 'purple';
      case 'system':
        return 'orange';
      case 'custom':
        return 'default';
      default:
        return 'default';
    }
  };

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case 'ticket':
        return <FileText className='w-4 h-4' />;
      case 'sla':
        return <CheckCircle className='w-4 h-4' />;
      case 'user':
        return <Users className='w-4 h-4' />;
      case 'system':
        return <Settings className='w-4 h-4' />;
      case 'custom':
        return <BarChart3 className='w-4 h-4' />;
      default:
        return <BarChart3 className='w-4 h-4' />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'success';
      case 'running':
        return 'processing';
      case 'failed':
        return 'error';
      default:
        return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className='w-4 h-4' />;
      case 'running':
        return <Clock className='w-4 h-4' />;
      case 'failed':
        return <AlertTriangle className='w-4 h-4' />;
      default:
        return <Clock className='w-4 h-4' />;
    }
  };

  const reportColumns: ColumnsType<ReportDefinition> = [
    {
      title: '报表信息',
      key: 'info',
      render: (_, record) => (
        <div className='space-y-1'>
          <div className='font-medium'>{record.name}</div>
          <div className='text-sm text-gray-600'>{record.description}</div>
          <div className='flex items-center gap-2 text-xs text-gray-500'>
            {getCategoryIcon(record.category)}
            <span>
              {record.category === 'ticket'
                ? '工单'
                : record.category === 'sla'
                ? 'SLA'
                : record.category === 'user'
                ? '用户'
                : record.category === 'system'
                ? '系统'
                : '自定义'}
            </span>
            <span>•</span>
            <span>创建者：{record.created_by}</span>
          </div>
        </div>
      ),
    },
    {
      title: '可视化类型',
      key: 'visualization',
      width: 120,
      render: (_, record) => (
        <div className='space-y-1'>
          <Tag color='blue'>
            {record.visualization === 'table'
              ? '表格'
              : record.visualization === 'chart'
              ? '图表'
              : '仪表盘'}
          </Tag>
          {record.chart_type && (
            <div className='text-xs text-gray-500'>
              {record.chart_type === 'bar'
                ? '柱状图'
                : record.chart_type === 'line'
                ? '折线图'
                : record.chart_type === 'pie'
                ? '饼图'
                : record.chart_type === 'area'
                ? '面积图'
                : '散点图'}
            </div>
          )}
        </div>
      ),
    },
    {
      title: '状态',
      key: 'is_active',
      width: 80,
      render: (_, record) => (
        <Tag color={record.is_active ? 'success' : 'default'}>
          {record.is_active ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 200,
      render: (_, record) => (
        <Space size='small'>
          <Button size='small' type='primary' onClick={() => handleExecuteReport(record)}>
            执行
          </Button>
          <Button
            size='small'
            icon={<Edit className='w-3 h-3' />}
            onClick={() => {
              setSelectedReport(record);
              form.setFieldsValue(record);
              setShowReportModal(true);
            }}
          >
            编辑
          </Button>
          <Popconfirm
            title='确认删除'
            description='确定要删除这个报表吗？'
            onConfirm={() => handleDeleteReport(record.id)}
            okText='确认'
            cancelText='取消'
          >
            <Button size='small' danger icon={<Trash2 className='w-3 h-3' />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const executionColumns: ColumnsType<ReportExecution> = [
    {
      title: '报表名称',
      key: 'report_name',
      render: (_, record) => <div className='font-medium'>{record.report_name}</div>,
    },
    {
      title: '状态',
      key: 'status',
      width: 120,
      render: (_, record) => (
        <Tag color={getStatusColor(record.status)}>
          {getStatusIcon(record.status)}
          {record.status === 'completed'
            ? '已完成'
            : record.status === 'running'
            ? '执行中'
            : '执行失败'}
        </Tag>
      ),
    },
    {
      title: '执行时间',
      key: 'execution_time',
      width: 150,
      render: (_, record) => (
        <div className='text-sm'>
          <div>开始：{new Date(record.started_at).toLocaleString()}</div>
          {record.completed_at && <div>完成：{new Date(record.completed_at).toLocaleString()}</div>}
        </div>
      ),
    },
    {
      title: '结果',
      key: 'result',
      width: 100,
      render: (_, record) => (
        <div className='text-sm'>
          {record.status === 'completed' && record.result_count && (
            <div>记录数：{record.result_count}</div>
          )}
          {record.status === 'failed' && record.error_message && (
            <div className='text-red-500'>错误：{record.error_message}</div>
          )}
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_, record) => (
        <Space size='small'>
          {record.status === 'completed' && (
            <Button size='small' icon={<Eye className='w-3 h-3' />}>
              查看结果
            </Button>
          )}
          {record.status === 'running' && (
            <Button size='small' icon={<RefreshCw className='w-3 h-3' />}>
              刷新状态
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className='space-y-6'>
      <div className='flex justify-between items-center'>
        <div>
          <Title level={2}>
            <BarChart3 className='inline-block w-6 h-6 mr-2' />
            高级报表分析
          </Title>
          <Text type='secondary'>创建自定义报表、分析数据和构建实时仪表盘</Text>
        </div>
        <Space>
          <Button icon={<RefreshCw className='w-4 h-4' />} onClick={loadData}>
            刷新
          </Button>
        </Space>
      </div>

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab='报表管理' key='reports'>
          <div className='space-y-4'>
            <div className='flex justify-between items-center'>
              <Title level={4}>报表定义</Title>
              <Button
                type='primary'
                icon={<Plus className='w-4 h-4' />}
                onClick={() => {
                  setSelectedReport(null);
                  form.resetFields();
                  setShowReportModal(true);
                }}
              >
                新建报表
              </Button>
            </div>

            <Table
              columns={reportColumns}
              dataSource={reports}
              rowKey='id'
              loading={loading}
              size='small'
            />
          </div>
        </TabPane>

        <TabPane tab='执行历史' key='executions'>
          <div className='space-y-4'>
            <Title level={4}>报表执行历史</Title>
            <Table columns={executionColumns} dataSource={executions} rowKey='id' size='small' />
          </div>
        </TabPane>

        <TabPane tab='实时仪表盘' key='dashboard'>
          <div className='space-y-4'>
            <div className='flex justify-between items-center'>
              <Title level={4}>实时数据仪表盘</Title>
              <Space>
                {isEditing ? (
                  <>
                    <Button onClick={() => setIsEditing(false)}>取消编辑</Button>
                    <Button type='primary' onClick={handleSaveDashboard}>
                      保存配置
                    </Button>
                  </>
                ) : (
                  <Button icon={<Edit className='w-4 h-4' />} onClick={() => setIsEditing(true)}>
                    编辑布局
                  </Button>
                )}
              </Space>
            </div>

            <div className='grid grid-cols-12 gap-4' style={{ minHeight: '600px' }}>
              {dashboardWidgets.map(widget => (
                <div
                  key={widget.id}
                  className='col-span-3'
                  style={{
                    gridColumn: `span ${widget.position.w}`,
                    gridRow: `span ${widget.position.h}`,
                  }}
                >
                  <Card
                    title={widget.title}
                    size='small'
                    className='h-full'
                    extra={
                      isEditing && (
                        <Space>
                          <Button size='small' icon={<Settings className='w-3 h-3' />} />
                          <Button size='small' icon={<Trash2 className='w-3 h-3' />} />
                        </Space>
                      )
                    }
                  >
                    <div className='h-full flex items-center justify-center'>
                      {widget.type === 'chart' && (
                        <div className='text-center text-gray-500'>
                          <BarChart3 className='w-12 h-12 mx-auto mb-2' />
                          <div>图表组件</div>
                          <div className='text-xs'>{widget.config.chartType}</div>
                        </div>
                      )}
                      {widget.type === 'metric' && (
                        <div className='text-center'>
                          <Statistic
                            title={widget.title}
                            value={95.6}
                            suffix='%'
                            styles={{ content: { color: '#52c41a' } }}
                          />
                        </div>
                      )}
                      {widget.type === 'table' && (
                        <div className='text-center text-gray-500'>
                          <Table className='w-full' size='small' />
                        </div>
                      )}
                    </div>
                  </Card>
                </div>
              ))}
            </div>
          </div>
        </TabPane>
      </Tabs>

      {/* 报表编辑模态框 */}
      <Modal
        title={selectedReport ? '编辑报表' : '新建报表'}
        open={showReportModal}
        onOk={selectedReport ? handleEditReport : handleCreateReport}
        onCancel={() => {
          setShowReportModal(false);
          setSelectedReport(null);
          form.resetFields();
        }}
        width={800}
        okText='保存'
        cancelText='取消'
      >
        <Form form={form} layout='vertical'>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label='报表名称'
                name='name'
                rules={[{ required: true, message: '请输入报表名称' }]}
              >
                <Input placeholder='请输入报表名称' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label='报表分类'
                name='category'
                rules={[{ required: true, message: '请选择报表分类' }]}
              >
                <Select placeholder='请选择报表分类'>
                  <Option value='ticket'>工单</Option>
                  <Option value='sla'>SLA</Option>
                  <Option value='user'>用户</Option>
                  <Option value='system'>系统</Option>
                  <Option value='custom'>自定义</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item label='描述' name='description'>
            <TextArea rows={2} placeholder='请输入报表描述' />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label='数据源'
                name='data_source'
                rules={[{ required: true, message: '请输入数据源' }]}
              >
                <Input placeholder='请输入数据源' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label='可视化类型'
                name='visualization'
                rules={[{ required: true, message: '请选择可视化类型' }]}
              >
                <Select placeholder='请选择可视化类型'>
                  <Option value='table'>表格</Option>
                  <Option value='chart'>图表</Option>
                  <Option value='dashboard'>仪表盘</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label='SQL查询'
            name='query'
            rules={[{ required: true, message: '请输入SQL查询' }]}
          >
            <TextArea rows={4} placeholder='请输入SQL查询语句' />
          </Form.Item>

          <Form.Item label='启用状态' name='is_active' valuePropName='checked'>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      {/* 报表执行结果模态框 */}
      <Modal
        title={`报表执行结果 - ${currentReport?.name}`}
        open={showExecutionModal}
        onCancel={() => setShowExecutionModal(false)}
        width={1200}
        footer={[
          <Button key='close' onClick={() => setShowExecutionModal(false)}>
            关闭
          </Button>,
          <Dropdown
            key='export'
            overlay={
              <Menu>
                <Menu.Item key='excel' onClick={() => handleExportReport('excel')}>
                  <Download className='w-4 h-4 mr-2' />
                  导出Excel
                </Menu.Item>
                <Menu.Item key='csv' onClick={() => handleExportReport('csv')}>
                  <Download className='w-4 h-4 mr-2' />
                  导出CSV
                </Menu.Item>
                <Menu.Item key='pdf' onClick={() => handleExportReport('pdf')}>
                  <Download className='w-4 h-4 mr-2' />
                  导出PDF
                </Menu.Item>
              </Menu>
            }
          >
            <Button type='primary' icon={<Download className='w-4 h-4' />}>
              导出数据
            </Button>
          </Dropdown>,
        ]}
      >
        {reportData && (
          <div className='space-y-4'>
            {/* 数据摘要 */}
            <Card title='数据摘要' size='small'>
              <Row gutter={[16, 16]}>
                {Object.entries(reportData.summary).map(([key, value]) => (
                  <Col span={6} key={key}>
                    <Statistic
                      title={key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
                      value={value}
                    />
                  </Col>
                ))}
              </Row>
            </Card>

            {/* 数据表格 */}
            <Card title='详细数据' size='small'>
              <Table
                columns={reportData.columns.map(col => ({
                  title: col,
                  dataIndex: col,
                  key: col,
                }))}
                dataSource={reportData.rows.map((row, index) => {
                  const obj: any = { key: index };
                  row.forEach((cell, cellIndex) => {
                    obj[reportData.columns[cellIndex]] = cell;
                  });
                  return obj;
                })}
                pagination={{
                  pageSize: 10,
                  showSizeChanger: true,
                  showQuickJumper: true,
                  showTotal: total => `共 ${total} 条记录`,
                }}
                size='small'
                scroll={{ x: true }}
              />
            </Card>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default AdvancedReporting;
