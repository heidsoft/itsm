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
  message,
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

interface ChangeRiskAssessmentProps {
  changeId?: number;
  initialData?: Partial<RiskAssessmentData>;
  onSave?: (data: RiskAssessmentData) => void;
  readOnly?: boolean;
}

interface RiskAssessmentData {
  riskLevel: ChangeRisk;
  riskDescription: string;
  impactAnalysis: string;
  mitigationMeasures: string;
  contingencyPlan: string;
  riskOwner: string;
  riskScore?: number;
  riskFactors?: string[];
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
  const [submitting, setSubmitting] = useState(false);

  // 风险等级配置
  const riskLevels = {
    low: { value: 'low', label: 'Low Risk', color: 'green', score: 1 },
    medium: { value: 'medium', label: 'Medium Risk', color: 'orange', score: 2 },
    high: { value: 'high', label: 'High Risk', color: 'red', score: 3 },
  };

  // 风险因子选项
  const riskFactorOptions = [
    'System compatibility issue',
    'Data loss risk',
    'Service interruption',
    'Performance impact',
    'Security Vulnerability',
    'User operation complexity',
    'Dependency complexity',
    'Difficulty rolling back',
    'Insufficient resources',
    'Time pressure',
  ];

  // 计算风险分数
  useEffect(() => {
    const riskLevel = form.getFieldValue('riskLevel') as ChangeRisk;
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
      if (initialData.riskFactors) {
        setRiskFactors(initialData.riskFactors);
      }
    }
  }, [initialData, form]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      const assessmentData: RiskAssessmentData = {
        ...values,
        riskScore: riskScore,
        riskFactors: riskFactors,
      };

      onSave?.(assessmentData);
    } catch (error) {
      message.error('Form validation failed');
    } finally {
      setSubmitting(false);
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
    if (score <= 30) return 'Low Risk';
    if (score <= 60) return 'Medium Risk';
    return 'High Risk';
  };

  return (
    <Card
      title={
        <Space>
          <Shield className="w-5 h-5 text-blue-600" />
          <span>Change Risk Assessment</span>
        </Space>
      }
      extra={
        !readOnly && (
          <Button type="primary" onClick={handleSubmit} loading={submitting}>

            Save assessment
          </Button>
        )
      }
      className="change-risk-assessment"
    >
      <Form form={form} layout="vertical" disabled={readOnly}>
        <Row gutter={[24, 16]}>
          {/* 风险等级选择 */}
          <Col xs={24} lg={8}>
            <Form.Item
              label="Risk Level"
              name="riskLevel"
              rules={[{ required: true, message: 'Please select a risk level' }]}
            >
              <Select placeholder="Select risk level" onChange={() => setRiskFactors([])} options={Object.entries(riskLevels).map(([key, config]) => ({
                value: config.value as ChangeRisk,
                label: (
                  <Space>
                    {getRiskIcon(config.value as ChangeRisk)}
                    <span>{config.label}</span>
                    <Tag color={config.color}>{config.score}points</Tag>
                  </Space>
                ),
              }))} />
            </Form.Item>

            {/* 风险分数显示 */}
            <Card size="small" className="mt-4">
              <div className="text-center">
                <Title
                  level={4}
                  style={{
                    color: getRiskColor(riskScore),
                    margin: '0 0 8px 0',
                  }}
                >
                  {riskScore}points
                </Title>
                <Text type="secondary">Comprehensive risk rating:{getRiskStatus(riskScore)}</Text>
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
              label="Risk description"
              name="riskDescription"
              rules={[{ required: true, message: 'Please describe the risk content' }]}
            >
              <TextArea
                rows={4}
                placeholder="Describe in detail the risks that may arise from the change..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 影响分析 */}
          <Col xs={24}>
            <Form.Item
              label="Impact Analysis"
              name="impactAnalysis"
              rules={[{ required: true, message: 'Please perform impact analysis' }]}
            >
              <TextArea
                rows={6}
                placeholder="Analyze the impact of changes on systems, businesses, users, etc..."
                maxLength={2000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 风险因子 */}
          <Col xs={24} lg={12}>
            <Form.Item label="Risk factors">
              <Select
                mode="multiple"
                placeholder="Select applicable risk factors"
                value={riskFactors}
                onChange={setRiskFactors}
                style={{ width: '100%' }}
                options={riskFactorOptions.map(factor => ({ value: factor, label: factor }))}
              />
              <Text type="secondary" className="text-xs">

                Selecting the risk factors involved in this change will affect the risk score calculation
              </Text>
            </Form.Item>
          </Col>

          {/* 风险责任人 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="Risk Owner"
              name="riskOwner"
              rules={[{ required: true, message: 'Please designate the risk owner' }]}
            >
              <Input placeholder="Enter the name of the risk owner" prefix={<Users className="w-4 h-4" />} />
            </Form.Item>
          </Col>

          {/* 缓解措施 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="Mitigation measures"
              name="mitigationMeasures"
              rules={[{ required: true, message: 'Please develop mitigation measures' }]}
            >
              <TextArea
                rows={4}
                placeholder="Develop specific risk mitigation measures..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 应急计划 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="Contingency Plan"
              name="contingencyPlan"
              rules={[{ required: true, message: 'Please make a contingency plan' }]}
            >
              <TextArea
                rows={4}
                placeholder="Develop an emergency response plan in the event of a risk..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>
        </Row>

        {/* 风险提示 */}
        {riskScore >= 60 && (
          <Alert
            title="High Risk Warning"
            description="This change is assessed as high risk. It is recommended to formulate a contingency plan, arrange for implementation during off-peak periods, and prepare a rollback plan."
            type="error"
            showIcon
            icon={<AlertTriangle className="w-4 h-4" />}
            className="mt-4"
          />
        )}

        {riskScore >= 30 && riskScore < 60 && (
          <Alert
            title="Medium risk warning"
            description="This change involves certain risks, and it is recommended to carefully check the completeness of the mitigation measures and contingency plans."
            type="warning"
            showIcon
            className="mt-4"
          />
        )}

        {riskScore < 30 && (
          <Alert
            title="Low risk confirmation"
            description="This change is low risk, but it is still recommended to implement it according to standard procedures."
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
