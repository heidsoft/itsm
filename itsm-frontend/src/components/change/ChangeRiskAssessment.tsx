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
  Alert,
  Row,
  Col,
  Divider,
  Tag,
  Progress,
} from 'antd';
import {
  Shield,
  AlertTriangle,
  CheckCircle,
  Clock,
  Users,
  FileText,
  TrendingUp,
} from 'lucide-react';
import type { ChangeRisk, ChangeImpact } from '@/lib/api/change-api';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;
const { Option } = Select;

interface ChangeRiskAssessmentProps {
  changeId?: number;
  initialData?: Partial<RiskAssessmentData>;
  onSave?: (data: RiskAssessmentData) => void;
  readOnly?: boolean;
}

interface RiskAssessmentData {
  risk_level: ChangeRisk;
  risk_description: string;
  impact_analysis: string;
  mitigation_measures: string;
  contingency_plan: string;
  risk_owner: string;
  risk_score?: number;
  risk_factors?: string[];
}

const ChangeRiskAssessment: React.FC<ChangeRiskAssessmentProps> = ({
  changeId,
  initialData,
  onSave,
  readOnly = false,
}) => {
  const [form] = Form.useForm();
  const [riskScore, setRiskScore] = useState(0);
  const [riskFactors, setRiskFactors] = useState<string[]>([]);

  // 风险等级配置
  const riskLevels = {
    low: { value: 'low', label: '低风险', color: 'green', score: 1 },
    medium: { value: 'medium', label: '中风险', color: 'orange', score: 2 },
    high: { value: 'high', label: '高风险', color: 'red', score: 3 },
  };

  // 风险因子选项
  const riskFactorOptions = [
    '系统兼容性问题',
    '数据丢失风险',
    '服务中断',
    '性能影响',
    '安全漏洞',
    '用户操作复杂度',
    '依赖关系复杂性',
    '回滚困难',
    '资源不足',
    '时间压力',
  ];

  // 计算风险分数
  useEffect(() => {
    const riskLevel = form.getFieldValue('risk_level') as ChangeRisk;
    const factorsCount = riskFactors.length;
    
    let score = 0;
    if (riskLevel === 'low') score = 10;
    else if (riskLevel === 'medium') score = 30;
    else if (riskLevel === 'high') score = 60;
    
    // 风险因子增加分数
    score += factorsCount * 5;
    
    setRiskScore(Math.min(score, 100));
  }, [riskFactors]);

  useEffect(() => {
    if (initialData) {
      form.setFieldsValue(initialData);
      if (initialData.risk_factors) {
        setRiskFactors(initialData.risk_factors);
      }
    }
  }, [initialData, form]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      const assessmentData: RiskAssessmentData = {
        ...values,
        risk_score: riskScore,
        risk_factors: riskFactors,
      };
      
      onSave?.(assessmentData);
    } catch (error) {
      console.error('Form validation failed:', error);
    }
  };

  const getRiskIcon = (level: ChangeRisk) => {
    switch (level) {
      case 'low':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'medium':
        return <AlertTriangle className="w-4 h-4 text-orange-500" />;
      case 'high':
        return <Shield className="w-4 h-4 text-red-500" />;
      default:
        return null;
    }
  };

  const getRiskColor = (score: number) => {
    if (score <= 30) return '#52c41a';
    if (score <= 60) return '#faad14';
    return '#ff4d4f';
  };

  const getRiskStatus = (score: number) => {
    if (score <= 30) return '低风险';
    if (score <= 60) return '中风险';
    return '高风险';
  };

  return (
    <Card
      title={
        <Space>
          <Shield className="w-5 h-5 text-blue-600" />
          <span>变更风险评估</span>
        </Space>
      }
      extra={
        !readOnly && (
          <Button type="primary" onClick={handleSubmit}>
            保存评估
          </Button>
        )
      }
      className="change-risk-assessment"
    >
      <Form
        form={form}
        layout="vertical"
        disabled={readOnly}
      >
        <Row gutter={[24, 16]}>
          {/* 风险等级选择 */}
          <Col xs={24} lg={8}>
            <Form.Item
              label="风险等级"
              name="risk_level"
              rules={[{ required: true, message: '请选择风险等级' }]}
            >
              <Select
                placeholder="选择风险等级"
                onChange={() => setRiskFactors([])}
              >
                {Object.entries(riskLevels).map(([key, config]) => (
                  <Option key={key} value={config.value as ChangeRisk}>
                    <Space>
                      {getRiskIcon(config.value as ChangeRisk)}
                      <span>{config.label}</span>
                      <Tag color={config.color}>{config.score}分</Tag>
                    </Space>
                  </Option>
                ))}
              </Select>
            </Form.Item>

            {/* 风险分数显示 */}
            <Card size="small" className="mt-4">
              <div className="text-center">
                <Title level={4} style={{ 
                  color: getRiskColor(riskScore),
                  margin: '0 0 8px 0'
                }}>
                  {riskScore}分
                </Title>
                <Text type="secondary">
                  综合风险评级：{getRiskStatus(riskScore)}
                </Text>
                <Progress
                  percent={riskScore}
                  strokeColor={getRiskColor(riskScore)}
                  showInfo={false}
                  className="mt-2"
                />
              </div>
            </Card>
          </Col>

          {/* 风险描述 */}
          <Col xs={24} lg={16}>
            <Form.Item
              label="风险描述"
              name="risk_description"
              rules={[{ required: true, message: '请描述风险内容' }]}
            >
              <TextArea
                rows={4}
                placeholder="详细描述变更可能带来的风险..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 影响分析 */}
          <Col xs={24}>
            <Form.Item
              label="影响分析"
              name="impact_analysis"
              rules={[{ required: true, message: '请进行影响分析' }]}
            >
              <TextArea
                rows={6}
                placeholder="分析变更对系统、业务、用户等方面的影响..."
                maxLength={2000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 风险因子 */}
          <Col xs={24} lg={12}>
            <Form.Item label="风险因子">
              <Select
                mode="multiple"
                placeholder="选择适用的风险因子"
                value={riskFactors}
                onChange={setRiskFactors}
                style={{ width: '100%' }}
              >
                {riskFactorOptions.map(factor => (
                  <Option key={factor} value={factor}>
                    {factor}
                  </Option>
                ))}
              </Select>
              <Text type="secondary" className="text-xs">
                选择该变更涉及的风险因子，将影响风险分数计算
              </Text>
            </Form.Item>
          </Col>

          {/* 风险责任人 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="风险责任人"
              name="risk_owner"
              rules={[{ required: true, message: '请指定风险责任人' }]}
            >
              <Input
                placeholder="输入风险责任人姓名"
                prefix={<Users className="w-4 h-4" />}
              />
            </Form.Item>
          </Col>

          {/* 缓解措施 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="缓解措施"
              name="mitigation_measures"
              rules={[{ required: true, message: '请制定缓解措施' }]}
            >
              <TextArea
                rows={4}
                placeholder="制定具体的风险缓解措施..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 应急计划 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="应急计划"
              name="contingency_plan"
              rules={[{ required: true, message: '请制定应急计划' }]}
            >
              <TextArea
                rows={4}
                placeholder="制定风险发生时的应急响应计划..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>
        </Row>

        {/* 风险提示 */}
        {riskScore >= 60 && (
          <Alert
            message="高风险警告"
            description="该变更被评定为高风险，建议：1. 制定详细的应急计划 2. 安排在业务低峰期实施 3. 准备充分的回滚方案 4. 增加监控和巡检频率"
            type="error"
            showIcon
            icon={<AlertTriangle className="w-4 h-4" />}
            className="mt-4"
          />
        )}

        {riskScore >= 30 && riskScore < 60 && (
          <Alert
            message="中等风险提示"
            description="该变更存在一定风险，建议仔细检查缓解措施和应急计划的完备性。"
            type="warning"
            showIcon
            className="mt-4"
          />
        )}

        {riskScore < 30 && (
          <Alert
            message="低风险确认"
            description="该变更风险较低，但仍建议按照标准流程实施。"
            type="success"
            showIcon
            className="mt-4"
          />
        )}
      </Form>
    </Card>
  );
};

export default ChangeRiskAssessment;