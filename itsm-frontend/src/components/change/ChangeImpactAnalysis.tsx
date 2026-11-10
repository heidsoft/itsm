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
  Tag,
  Row,
  Col,
  Progress,
  Alert,
  Transfer,
  Checkbox,
  message,
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
import { CMDBApi } from '@/lib/api/cmdb-api';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

interface ChangeImpactAnalysisProps {
  changeId?: number;
  initialData?: Partial<ImpactAnalysisData>;
  onSave?: (data: ImpactAnalysisData) => void;
  readOnly?: boolean;
}

interface ImpactAnalysisData {
  businessImpact: string;
  technicalImpact: string;
  userImpact: string;
  affectedSystems: string[];
  affectedUsers: number;
  estimatedDowntime: number;
  dataRiskLevel: string;
  serviceDependencies: string[];
  backupStrategy: string;
  recoveryPlan: string;
  impactScore?: number;
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
  const [submitting, setSubmitting] = useState(false);
  const [systemOptions, setSystemOptions] = useState<SystemItem[]>([]);
  const [systemLoading, setSystemLoading] = useState(false);
  const [systemLoadError, setSystemLoadError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setSystemLoading(true);
      try {
        const items = await CMDBApi.getAllCIs({}, 200);
        const mapped: SystemItem[] = items.map(ci => {
          const ciType = ci.type || 'Configuration item';
          const criticality =
            ci.criticality === 'high' || ci.criticality === 'medium' || ci.criticality === 'low'
              ? ci.criticality
              : 'medium';

          return {
            key: String(ci.id),
            title: ci.name,
            description: ciType,
            category: ciType,
            criticality,
          };
        });

        if (cancelled) return;
        setSystemOptions(mapped);
        setSystemLoadError(null);
        setTargetKeys(prevKeys => {
          if (prevKeys.length === 0) return prevKeys;
          const keySet = new Set(mapped.map(i => i.key));
          if (prevKeys.every(k => keySet.has(k))) return prevKeys;

          const titleToKey = new Map(mapped.map(i => [i.title, i.key]));
          const nextKeys = prevKeys.map(k => titleToKey.get(k) ?? k).filter(k => keySet.has(k));
          return nextKeys.length > 0 ? nextKeys : prevKeys;
        });
      } catch (error) {
        console.error('Failed to load configuration items:', error);
        if (cancelled) return;
        setSystemOptions([]);
        setSystemLoadError('Failed to load configuration items');
        message.error('Failed to load configuration items');
      } finally {
        if (!cancelled) setSystemLoading(false);
      }
    };

    load();
    return () => {
      cancelled = true;
    };
  }, []);

  // 影响程度选项
  const impactLevels = {
    none: { label: 'No impact', score: 0, color: 'green' },
    low: { label: 'Minor impact', score: 1, color: 'blue' },
    medium: { label: 'Moderate impact', score: 2, color: 'orange' },
    high: { label: 'Severe impact', score: 3, color: 'red' },
    critical: { label: 'Key impact', score: 4, color: 'purple' },
  };

  // 计算影响分数
  const calculateImpactScore = (values: any) => {
    let score = 0;

    // 业务影响分数
    const businessImpact = values.businessImpact;
    if (businessImpact) {
      if (businessImpact.includes('关键') || businessImpact.includes('严重')) score += 20;
      else if (businessImpact.includes('中等')) score += 15;
      else if (businessImpact.includes('轻微')) score += 10;
    }

    // 受影响系统数量
    score += targetKeys.length * 5;

    // 受影响用户数
    const affectedUsers = values.affectedUsers || 0;
    if (affectedUsers > 10000) score += 15;
    else if (affectedUsers > 1000) score += 10;
    else if (affectedUsers > 100) score += 5;

    // 预计停机时间
    const downtime = values.estimatedDowntime || 0;
    if (downtime > 60) score += 20;
    else if (downtime > 10) score += 15;
    else if (downtime > 1) score += 10;
    else if (downtime > 0) score += 5;

    return Math.min(score, 100);
  };

  useEffect(() => {
    if (initialData) {
      form.setFieldsValue(initialData);
      if (initialData.affectedSystems) {
        setTargetKeys(initialData.affectedSystems);
      }
    }
  }, [initialData, form]);

  useEffect(() => {
    const values = form.getFieldsValue();
    const score = calculateImpactScore(values);
    setImpactScore(score);
  }, [form, targetKeys]);

  // 监听表单变化计算影响分数
  const onValuesChange = (changedValues: unknown, allValues: unknown) => {
    const score = calculateImpactScore(allValues);
    setImpactScore(score);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      const analysisData: ImpactAnalysisData = {
        ...values,
        affectedSystems: targetKeys,
        impactScore: impactScore,
      };

      onSave?.(analysisData);
    } catch (error) {
      message.error('Form validation failed');
    } finally {
      setSubmitting(false);
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
            <Text type="secondary" className="text-xs">
              {item.description}
            </Text>
          </div>
        </div>
        <Tag
          color={
            item.criticality === 'high' ? 'red' : item.criticality === 'medium' ? 'orange' : 'blue'
          }
        >
          {item.criticality === 'high' ? 'Critical' : item.criticality === 'medium' ? 'Important' : 'Average'}
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
    if (score <= 30) return 'Low impact';
    if (score <= 60) return 'Moderate impact';
    if (score <= 80) return 'High impact';
    return 'Key impact';
  };

  return (
    <Card
      title={
        <Space>
          <Target className="w-5 h-5 text-blue-600" />
          <span>Change impact analysis</span>
        </Space>
      }
      extra={
        !readOnly && (
          <Button type="primary" onClick={handleSubmit} loading={submitting}>

            Save analysis
          </Button>
        )
      }
      className="change-impact-analysis"
    >
      <Form form={form} layout="vertical" disabled={readOnly} onValuesChange={onValuesChange}>
        {/* 影响分数显示 */}
        <Card size="small" className="mb-4">
          <Row gutter={16} align="middle">
            <Col span={6}>
              <div className="text-center">
                <Title
                  level={3}
                  style={{
                    color: getImpactColor(impactScore),
                    margin: '0 0 8px 0',
                  }}
                >
                  {impactScore}points
                </Title>
                <Text type="secondary">Comprehensive impact score</Text>
              </div>
            </Col>
            <Col span={6}>
              <div className="text-center">
                <div className="text-2xl font-bold text-blue-600">{targetKeys.length}</div>
                <Text type="secondary">Affected systems</Text>
              </div>
            </Col>
            <Col span={6}>
              <div className="text-center">
                <div className="text-2xl font-bold text-orange-600">
                  {form.getFieldValue('affected_users') || 0}
                </div>
                <Text type="secondary">Affected Users</Text>
              </div>
            </Col>
            <Col span={6}>
              <div className="text-center">
                <div className="text-2xl font-bold text-red-600">
                  {form.getFieldValue('estimated_downtime') || 0}minutes
                </div>
                <Text type="secondary">Estimated downtime</Text>
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
              label="Business Impact Analysis"
              name="businessImpact"
              rules={[{ required: true, message: 'Please analyze the business impact' }]}
            >
              <TextArea
                rows={4}
                placeholder="Analyze the impact of changes on business processes, service quality, revenue, etc..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 技术影响 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="Technology Impact Analysis"
              name="technicalImpact"
              rules={[{ required: true, message: 'Please analyze the technical impact' }]}
            >
              <TextArea
                rows={4}
                placeholder="Analyze the impact of changes on system architecture, performance, security, etc..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 用户影响 */}
          <Col xs={24}>
            <Form.Item
              label="User Impact Analysis"
              name="userImpact"
              rules={[{ required: true, message: 'Please analyze the user impact' }]}
            >
              <TextArea
                rows={4}
                placeholder="Analyze the impact of changes on user experience, operating habits, training needs, etc..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 受影响系统选择 */}
          <Col xs={24} lg={12}>
            <Form.Item label="Affected systems">
              {systemLoadError && (
                <Alert
                  title="System list loading failed"
                  description={systemLoadError}
                  type="error"
                  showIcon
                  className="mb-3"
                />
              )}
              <Transfer
                dataSource={systemOptions}
                titles={['Available systems', 'Affected systems']}
                targetKeys={targetKeys}
                onChange={keys => setTargetKeys(keys as string[])}
                render={renderSystemItem}
                listStyle={{
                  width: 300,
                  height: 300,
                }}
                showSearch
                disabled={readOnly || systemLoading}
              />
            </Form.Item>
          </Col>

          <Col xs={24} lg={12}>
            {/* 受影响用户数 */}
            <Form.Item
              label="Affected User Count"
              name="affectedUsers"
              rules={[{ required: true, message: 'Please enter the number of affected users' }]}
            >
              <Input
                type="number"
                placeholder="Estimated number of affected users"
                suffix={<Users className="w-4 h-4" />}
              />
            </Form.Item>

            {/* 预计停机时间 */}
            <Form.Item
              label="Estimated downtime (minutes)"
              name="estimatedDowntime"
              rules={[{ required: true, message: 'Please enter estimated downtime' }]}
            >
              <Input
                type="number"
                placeholder="Estimated service outage time"
                suffix={<Clock className="w-4 h-4" />}
              />
            </Form.Item>

            {/* 数据风险等级 */}
            <Form.Item
              label="Data risk level"
              name="dataRiskLevel"
              rules={[{ required: true, message: 'Please select data risk level' }]}
            >
              <Select placeholder="Select data risk level" options={[
                { value: 'low', label: 'Low Risk - No risk of data loss' },
                { value: 'medium', label: 'Medium Risk - Data recovery may be required' },
                { value: 'high', label: 'High Risk - Possibility of data loss' },
                { value: 'critical', label: 'Extremely High Risk - Risk of severe data loss' },
              ]} />
            </Form.Item>

            {/* 服务依赖 */}
            <Form.Item label="Service dependency" name="serviceDependencies">
              <Select mode="tags" placeholder="Select or enter dependent services" style={{ width: '100%' }} options={[
                { value: '认证服务', label: 'Authentication Service' },
                { value: '支付服务', label: 'Payment service' },
                { value: '通知服务', label: 'Notification service' },
                { value: '日志服务', label: 'Log service' },
                { value: '文件服务', label: 'File Service' },
              ]} />
            </Form.Item>
          </Col>

          {/* 备份策略 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="Backup strategy"
              name="backupStrategy"
              rules={[{ required: true, message: 'Please develop a backup strategy' }]}
            >
              <TextArea
                rows={4}
                placeholder="Describe the data backup and system backup strategy before the change..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 恢复计划 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="Recovery Plan"
              name="recoveryPlan"
              rules={[{ required: true, message: 'Please develop a recovery plan' }]}
            >
              <TextArea
                rows={4}
                placeholder="Develop a system recovery plan after a failed change..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>
        </Row>

        {/* 影响程度提示 */}
        {impactScore >= 80 && (
          <Alert
            title="Critical Impact Warning"
            description="This change is assessed as having a critical impact. It is recommended to arrange implementation during off-peak periods, prepare a rollback plan, and notify all relevant parties."
            type="error"
            showIcon
            icon={<AlertTriangle className="w-4 h-4" />}
            className="mt-4"
          />
        )}

        {impactScore >= 60 && impactScore < 80 && (
          <Alert
            title="High Impact Tip"
            description="This change has high impact and it is recommended to carefully evaluate implementation time and risk control measures."
            type="warning"
            showIcon
            className="mt-4"
          />
        )}

        {impactScore < 30 && (
          <Alert
            title="Low impact confirmation"
            description="This change has a small impact and can be implemented according to standard procedures."
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
