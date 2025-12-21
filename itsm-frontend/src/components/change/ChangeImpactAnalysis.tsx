'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Select,
  Input,
  Button,
  Space,
  Typography,
  Table,
  Tag,
  Row,
  Col,
  Progress,
  Alert,
  Transfer,
  Checkbox,
} from 'antd';
import {
  Target,
  Server,
  Globe,
  Users,
  Clock,
  Database,
  Network,
  Shield,
  AlertTriangle,
  CheckCircle,
  TrendingUp,
} from 'lucide-react';
import type { ChangeImpact } from '@/lib/api/change-api';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;
const { Option } = Select;

interface ChangeImpactAnalysisProps {
  changeId?: number;
  initialData?: Partial<ImpactAnalysisData>;
  onSave?: (data: ImpactAnalysisData) => void;
  readOnly?: boolean;
}

interface ImpactAnalysisData {
  business_impact: string;
  technical_impact: string;
  user_impact: string;
  affected_systems: string[];
  affected_users: number;
  estimated_downtime: number;
  data_risk_level: string;
  service_dependencies: string[];
  backup_strategy: string;
  recovery_plan: string;
  impact_score?: number;
}

interface SystemItem {
  key: string;
  title: string;
  description: string;
  category: string;
  criticality: 'high' | 'medium' | 'low';
}

const ChangeImpactAnalysis: React.FC<ChangeImpactAnalysisProps> = ({
  changeId,
  initialData,
  onSave,
  readOnly = false,
}) => {
  const [form] = Form.useForm();
  const [targetKeys, setTargetKeys] = useState<string[]>([]);
  const [impactScore, setImpactScore] = useState(0);

  // 预定义的系统列表
  const mockData: SystemItem[] = [
    { key: 'web-server-01', title: 'Web服务器01', description: '主Web服务器', category: '服务器', criticality: 'high' },
    { key: 'db-primary', title: '主数据库', description: 'MySQL主库', category: '数据库', criticality: 'high' },
    { key: 'db-replica', title: '数据库从库', description: 'MySQL从库', category: '数据库', criticality: 'medium' },
    { key: 'cache-redis', title: 'Redis缓存', description: 'Redis缓存服务器', category: '缓存', criticality: 'medium' },
    { key: 'api-gateway', title: 'API网关', description: '微服务网关', category: '网络', criticality: 'high' },
    { key: 'file-storage', title: '文件存储', description: '对象存储服务', category: '存储', criticality: 'medium' },
    { key: 'monitoring', title: '监控系统', description: 'Zabbix监控系统', category: '监控', criticality: 'low' },
    { key: 'backup-server', title: '备份服务器', description: '数据备份服务', category: '备份', criticality: 'medium' },
  ];

  // 影响程度选项
  const impactLevels = {
    none: { label: '无影响', score: 0, color: 'green' },
    low: { label: '轻微影响', score: 1, color: 'blue' },
    medium: { label: '中等影响', score: 2, color: 'orange' },
    high: { label: '严重影响', score: 3, color: 'red' },
    critical: { label: '关键影响', score: 4, color: 'purple' },
  };

  // 计算影响分数
  const calculateImpactScore = (values: any) => {
    let score = 0;
    
    // 业务影响分数
    const businessImpact = values.business_impact;
    if (businessImpact) {
      if (businessImpact.includes('关键') || businessImpact.includes('严重')) score += 20;
      else if (businessImpact.includes('中等')) score += 15;
      else if (businessImpact.includes('轻微')) score += 10;
    }
    
    // 受影响系统数量
    score += targetKeys.length * 5;
    
    // 受影响用户数
    const affectedUsers = values.affected_users || 0;
    if (affectedUsers > 10000) score += 15;
    else if (affectedUsers > 1000) score += 10;
    else if (affectedUsers > 100) score += 5;
    
    // 预计停机时间
    const downtime = values.estimated_downtime || 0;
    if (downtime > 60) score += 20;
    else if (downtime > 10) score += 15;
    else if (downtime > 1) score += 10;
    else if (downtime > 0) score += 5;
    
    return Math.min(score, 100);
  };

  useEffect(() => {
    if (initialData) {
      form.setFieldsValue(initialData);
      if (initialData.affected_systems) {
        setTargetKeys(initialData.affected_systems);
      }
    }
  }, [initialData, form]);

  // 监听表单变化计算影响分数
  const onValuesChange = (changedValues: any, allValues: any) => {
    const score = calculateImpactScore(allValues);
    setImpactScore(score);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      const analysisData: ImpactAnalysisData = {
        ...values,
        affected_systems: targetKeys,
        impact_score: impactScore,
      };
      
      onSave?.(analysisData);
    } catch (error) {
      console.error('Form validation failed:', error);
    }
  };

  // Transfer组件的渲染函数
  const renderSystemItem = (item: SystemItem) => (
    <div className="p-2 border border-gray-200 rounded mb-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          {getSystemIcon(item.category)}
          <div>
            <div className="font-medium">{item.title}</div>
            <Text type="secondary" className="text-xs">{item.description}</Text>
          </div>
        </div>
        <Tag color={item.criticality === 'high' ? 'red' : item.criticality === 'medium' ? 'orange' : 'blue'}>
          {item.criticality === 'high' ? '关键' : item.criticality === 'medium' ? '重要' : '一般'}
        </Tag>
      </div>
    </div>
  );

  const getSystemIcon = (category: string) => {
    switch (category) {
      case '服务器':
        return <Server className="w-4 h-4 text-blue-500" />;
      case '数据库':
        return <Database className="w-4 h-4 text-green-500" />;
      case '网络':
        return <Network className="w-4 h-4 text-purple-500" />;
      case '存储':
        return <Database className="w-4 h-4 text-orange-500" />;
      case '缓存':
        return <Database className="w-4 h-4 text-red-500" />;
      case '监控':
        return <Target className="w-4 h-4 text-gray-500" />;
      case '备份':
        return <Shield className="w-4 h-4 text-teal-500" />;
      default:
        return <Server className="w-4 h-4 text-gray-500" />;
    }
  };

  const getImpactColor = (score: number) => {
    if (score <= 30) return '#52c41a';
    if (score <= 60) return '#faad14';
    if (score <= 80) return '#ff7a45';
    return '#ff4d4f';
  };

  const getImpactStatus = (score: number) => {
    if (score <= 30) return '低影响';
    if (score <= 60) return '中等影响';
    if (score <= 80) return '高影响';
    return '关键影响';
  };

  return (
    <Card
      title={
        <Space>
          <Target className="w-5 h-5 text-blue-600" />
          <span>变更影响分析</span>
        </Space>
      }
      extra={
        !readOnly && (
          <Button type="primary" onClick={handleSubmit}>
            保存分析
          </Button>
        )
      }
      className="change-impact-analysis"
    >
      <Form
        form={form}
        layout="vertical"
        disabled={readOnly}
        onValuesChange={onValuesChange}
      >
        {/* 影响分数显示 */}
        <Card size="small" className="mb-4">
          <Row gutter={16} align="middle">
            <Col span={6}>
              <div className="text-center">
                <Title level={3} style={{ 
                  color: getImpactColor(impactScore),
                  margin: '0 0 8px 0'
                }}>
                  {impactScore}分
                </Title>
                <Text type="secondary">综合影响评分</Text>
              </div>
            </Col>
            <Col span={6}>
              <div className="text-center">
                <div className="text-2xl font-bold text-blue-600">{targetKeys.length}</div>
                <Text type="secondary">受影响系统</Text>
              </div>
            </Col>
            <Col span={6}>
              <div className="text-center">
                <div className="text-2xl font-bold text-orange-600">
                  {form.getFieldValue('affected_users') || 0}
                </div>
                <Text type="secondary">受影响用户</Text>
              </div>
            </Col>
            <Col span={6}>
              <div className="text-center">
                <div className="text-2xl font-bold text-red-600">
                  {form.getFieldValue('estimated_downtime') || 0}分钟
                </div>
                <Text type="secondary">预计停机时间</Text>
              </div>
            </Col>
          </Row>
          <Progress
            percent={impactScore}
            strokeColor={getImpactColor(impactScore)}
            showInfo={false}
            className="mt-4"
          />
        </Card>

        <Row gutter={[24, 16]}>
          {/* 业务影响 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="业务影响分析"
              name="business_impact"
              rules={[{ required: true, message: '请分析业务影响' }]}
            >
              <TextArea
                rows={4}
                placeholder="分析变更对业务流程、服务质量、收入等方面的影响..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 技术影响 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="技术影响分析"
              name="technical_impact"
              rules={[{ required: true, message: '请分析技术影响' }]}
            >
              <TextArea
                rows={4}
                placeholder="分析变更对系统架构、性能、安全性等方面的影响..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 用户影响 */}
          <Col xs={24}>
            <Form.Item
              label="用户影响分析"
              name="user_impact"
              rules={[{ required: true, message: '请分析用户影响' }]}
            >
              <TextArea
                rows={4}
                placeholder="分析变更对用户体验、操作习惯、培训需求等方面的影响..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 受影响系统选择 */}
          <Col xs={24} lg={12}>
            <Form.Item label="受影响系统">
              <Transfer
                dataSource={mockData}
                titles={['可用系统', '受影响系统']}
                targetKeys={targetKeys}
                onChange={setTargetKeys}
                render={renderSystemItem}
                listStyle={{
                  width: 300,
                  height: 300,
                }}
                showSearch
                searchPlaceholder="搜索系统..."
                disabled={readOnly}
              />
            </Form.Item>
          </Col>

          <Col xs={24} lg={12}>
            {/* 受影响用户数 */}
            <Form.Item
              label="受影响用户数"
              name="affected_users"
              rules={[{ required: true, message: '请输入受影响用户数' }]}
            >
              <Input
                type="number"
                placeholder="估计受影响的用户数量"
                suffix={<Users className="w-4 h-4" />}
              />
            </Form.Item>

            {/* 预计停机时间 */}
            <Form.Item
              label="预计停机时间（分钟）"
              name="estimated_downtime"
              rules={[{ required: true, message: '请输入预计停机时间' }]}
            >
              <Input
                type="number"
                placeholder="预计服务中断时间"
                suffix={<Clock className="w-4 h-4" />}
              />
            </Form.Item>

            {/* 数据风险等级 */}
            <Form.Item
              label="数据风险等级"
              name="data_risk_level"
              rules={[{ required: true, message: '请选择数据风险等级' }]}
            >
              <Select placeholder="选择数据风险等级">
                <Option value="low">低风险 - 无数据丢失风险</Option>
                <Option value="medium">中风险 - 可能需要数据恢复</Option>
                <Option value="high">高风险 - 有数据丢失可能</Option>
                <Option value="critical">极高风险 - 严重数据损失风险</Option>
              </Select>
            </Form.Item>

            {/* 服务依赖 */}
            <Form.Item
              label="服务依赖"
              name="service_dependencies"
            >
              <Select
                mode="tags"
                placeholder="选择或输入依赖的服务"
                style={{ width: '100%' }}
              >
                <Option value="认证服务">认证服务</Option>
                <Option value="支付服务">支付服务</Option>
                <Option value="通知服务">通知服务</Option>
                <Option value="日志服务">日志服务</Option>
                <Option value="文件服务">文件服务</Option>
              </Select>
            </Form.Item>
          </Col>

          {/* 备份策略 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="备份策略"
              name="backup_strategy"
              rules={[{ required: true, message: '请制定备份策略' }]}
            >
              <TextArea
                rows={4}
                placeholder="描述变更前的数据备份和系统备份策略..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 恢复计划 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="恢复计划"
              name="recovery_plan"
              rules={[{ required: true, message: '请制定恢复计划' }]}
            >
              <TextArea
                rows={4}
                placeholder="制定变更失败后的系统恢复计划..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>
        </Row>

        {/* 影响程度提示 */}
        {impactScore >= 80 && (
          <Alert
            message="关键影响警告"
            description="该变更被评定为关键影响，建议：1. 安排在业务低峰期实施 2. 准备完整的回滚方案 3. 通知所有相关方 4. 准备应急预案 5. 增加监控和巡检"
            type="error"
            showIcon
            icon={<AlertTriangle className="w-4 h-4" />}
            className="mt-4"
          />
        )}

        {impactScore >= 60 && impactScore < 80 && (
          <Alert
            message="高影响提示"
            description="该变更具有较高影响，建议仔细评估实施时间和风险控制措施。"
            type="warning"
            showIcon
            className="mt-4"
          />
        )}

        {impactScore < 30 && (
          <Alert
            message="低影响确认"
            description="该变更影响较小，可按标准流程实施。"
            type="success"
            showIcon
            className="mt-4"
          />
        )}
      </Form>
    </Card>
  );
};

export default ChangeImpactAnalysis;