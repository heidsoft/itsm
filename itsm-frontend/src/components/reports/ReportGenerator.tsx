'use client';

import React, { useState } from 'react';
import {
  Form,
  Select,
  Button,
  Card,
  Row,
  Col,
  Space,
  Typography,
  Divider,
  Alert,
  Switch,
  InputNumber,
  DatePicker,
  Input,
  message
} from 'antd';
import {
  BarChart3,
  PieChart,
  TrendingUp,
  Activity,
  FileText,
  PlayCircle,
  Settings,
} from 'lucide-react';
import { TicketAnalyticsApi, type AnalyticsConfig, type AnalyticsResponse } from '@/lib/api/ticket-analytics-api';
import ReportsCharts from './ReportsCharts';

const { Title, Text } = Typography;
const { Option } = Select;

interface ReportGeneratorProps {
  onGenerate: (config: Partial<AnalyticsConfig>) => void;
  loading?: boolean;
  timeRange: [string, string];
}

const ReportGenerator: React.FC<ReportGeneratorProps> = ({
  onGenerate,
  loading = false,
  timeRange,
}) => {
  const [form] = Form.useForm();
  const [previewData, setPreviewData] = useState<AnalyticsResponse | null>(null);
  const [showPreview, setShowPreview] = useState(false);

  // 维度选项
  const dimensionOptions = [
    { value: 'created_date', label: '创建日期', icon: <FileText size={16} /> },
    { value: 'status', label: '状态', icon: <Activity size={16} /> },
    { value: 'priority', label: '优先级', icon: <TrendingUp size={16} /> },
    { value: 'category', label: '分类', icon: <BarChart3 size={16} /> },
    { value: 'sla_status', label: 'SLA状态', icon: <PieChart size={16} /> },
    { value: 'resolution_category', label: '解决分类', icon: <Settings size={16} /> },
    { value: 'department', label: '部门', icon: <BarChart3 size={16} /> },
    { value: 'assignee', label: '处理人', icon: <Activity size={16} /> },
  ];

  // 指标选项
  const metricOptions = [
    { value: 'count', label: '数量统计', description: '工单总数' },
    { value: 'avg_response_time', label: '平均响应时间', description: '平均响应时长' },
    { value: 'avg_resolution_time', label: '平均解决时间', description: '平均解决时长' },
    { value: 'sla_compliance_rate', label: 'SLA合规率', description: 'SLA达成百分比' },
    { value: 'customer_satisfaction', label: '客户满意度', description: '客户评分' },
    { value: 'first_call_resolution', label: '首次解决率', description: '首次通话解决率' },
  ];

  // 图表类型选项
  const chartTypeOptions = [
    { value: 'bar', label: '柱状图', icon: <BarChart3 size={16} /> },
    { value: 'line', label: '折线图', icon: <TrendingUp size={16} /> },
    { value: 'pie', label: '饼图', icon: <PieChart size={16} /> },
    { value: 'area', label: '面积图', icon: <Activity size={16} /> },
    { value: 'table', label: '表格', icon: <FileText size={16} /> },
  ];

  // 生成预览
  const handleGeneratePreview = async () => {
    try {
      const values = await form.validateFields();

      const config: Partial<AnalyticsConfig> = {
        dimensions: values.dimensions,
        metrics: values.metrics,
        chart_type: values.chart_type,
        time_range: timeRange,
        filters: values.filters || {},
        group_by: values.group_by,
      };

      // 使用API获取真实数据预览
      const response = await TicketAnalyticsApi.getDeepAnalytics(config as AnalyticsConfig);
      setPreviewData(response);
      setShowPreview(true);
      message.success('预览生成成功');
    } catch (error) {
      message.error('生成预览失败');
    }
  };

  // 应用配置
  const handleApplyConfig = async () => {
    try {
      const values = await form.validateFields();

      const config: Partial<AnalyticsConfig> = {
        dimensions: values.dimensions,
        metrics: values.metrics,
        chart_type: values.chart_type,
        time_range: timeRange,
        filters: values.filters || {},
        group_by: values.group_by,
      };

      onGenerate(config);
      message.success('报告配置已应用');
    } catch (error) {
      message.error('配置验证失败');
    }
  };

  // 重置表单
  const handleReset = () => {
    form.resetFields();
    setShowPreview(false);
    setPreviewData(null);
  };

  return (
    <div className="space-y-6">
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          dimensions: ['created_date'],
          metrics: ['count'],
          chart_type: 'bar',
          page: 1,
          page_size: 20,
        }}
      >
        <Row gutter={[16, 16]}>
          {/* 维度选择 */}
          <Col xs={24} md={8}>
            <Form.Item
              label="分析维度"
              name="dimensions"
              rules={[{ required: true, message: '请选择至少一个维度' }]}
            >
              <Select
                mode="multiple"
                placeholder="选择分析维度"
                style={{ width: '100%' }}
              >
                {dimensionOptions.map(option => (
                  <Option key={option.value} value={option.value}>
                    <Space>
                      {option.icon}
                      {option.label}
                    </Space>
                  </Option>
                ))}
              </Select>
            </Form.Item>
          </Col>

          {/* 指标选择 */}
          <Col xs={24} md={8}>
            <Form.Item
              label="分析指标"
              name="metrics"
              rules={[{ required: true, message: '请选择至少一个指标' }]}
            >
              <Select
                mode="multiple"
                placeholder="选择分析指标"
                style={{ width: '100%' }}
              >
                {metricOptions.map(option => (
                  <Option key={option.value} value={option.value}>
                    <div>
                      <div>{option.label}</div>
                      <Text type="secondary" className="text-xs">
                        {option.description}
                      </Text>
                    </div>
                  </Option>
                ))}
              </Select>
            </Form.Item>
          </Col>

          {/* 图表类型 */}
          <Col xs={24} md={8}>
            <Form.Item
              label="图表类型"
              name="chart_type"
              rules={[{ required: true, message: '请选择图表类型' }]}
            >
              <Select placeholder="选择图表类型" style={{ width: '100%' }}>
                {chartTypeOptions.map(option => (
                  <Option key={option.value} value={option.value}>
                    <Space>
                      {option.icon}
                      {option.label}
                    </Space>
                  </Option>
                ))}
              </Select>
            </Form.Item>
          </Col>

          {/* 分组字段 */}
          <Col xs={24} md={8}>
            <Form.Item
              label="分组字段（可选）"
              name="group_by"
            >
              <Select placeholder="选择分组字段" allowClear style={{ width: '100%' }}>
                {dimensionOptions.map(option => (
                  <Option key={option.value} value={option.value}>
                    {option.label}
                  </Option>
                ))}
              </Select>
            </Form.Item>
          </Col>

          {/* 分页设置 */}
          <Col xs={24} md={8}>
            <Form.Item label="分页设置">
              <Space.Compact style={{ width: '100%' }}>
                <Form.Item name="page" noStyle>
                  <InputNumber
                    placeholder="页码"
                    min={1}
                    style={{ width: '50%' }}
                  />
                </Form.Item>
                <Form.Item name="page_size" noStyle>
                  <InputNumber
                    placeholder="每页数量"
                    min={1}
                    style={{ width: '50%' }}
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>

          {/* 操作按钮 */}
          <Col xs={24} md={8}>
            <Form.Item label="操作">
              <Space>
                <Button
                  type="primary"
                  icon={<PlayCircle size={16} />}
                  onClick={handleGeneratePreview}
                  loading={loading}
                >
                  预览
                </Button>
                <Button
                  icon={<Settings size={16} />}
                  onClick={handleApplyConfig}
                >
                  应用配置
                </Button>
                <Button onClick={handleReset}>
                  重置
                </Button>
              </Space>
            </Form.Item>
          </Col>
        </Row>
      </Form>

      <Divider />

      {/* 预览区域 */}
      {showPreview && previewData && (
        <Card title="数据预览" className="mb-4">
          <Alert
            message="预览模式"
            description="这是基于当前配置生成的数据预览，点击 '应用配置' 将更新主图表。"
            type="info"
            showIcon
            className="mb-4"
          />

          <ReportsCharts
            data={previewData.data}
            chartType={form.getFieldValue('chart_type')}
            height={300}
          />

          {/* 统计摘要 */}
          <Row gutter={[16, 16]} className="mt-6">
            <Col span={6}>
              <Card size="small">
                <div className="text-center">
                  <div className="text-2xl font-bold text-blue-600">
                    {previewData.summary.total.toLocaleString()}
                  </div>
                  <div className="text-gray-500">总数据量</div>
                </div>
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">
                    {previewData.summary.resolved.toLocaleString()}
                  </div>
                  <div className="text-gray-500">已解决</div>
                </div>
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <div className="text-center">
                  <div className="text-2xl font-bold text-orange-600">
                    {previewData.summary.avg_response_time.toFixed(1)}h
                  </div>
                  <div className="text-gray-500">平均响应时间</div>
                </div>
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <div className="text-center">
                  <div className="text-2xl font-bold text-purple-600">
                    {previewData.summary.sla_compliance.toFixed(1)}%
                  </div>
                  <div className="text-gray-500">SLA合规率</div>
                </div>
              </Card>
            </Col>
          </Row>
        </Card>
      )}

      {!showPreview && (
        <Alert
          message="配置报告参数"
          description="选择分析维度、指标和图表类型，然后点击 '预览' 查看效果。"
          type="info"
          showIcon
        />
      )}
    </div>
  );
};

export default ReportGenerator;