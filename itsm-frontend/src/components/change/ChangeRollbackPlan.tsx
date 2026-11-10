'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Input,
  Button,
  Space,
  Typography,
  Table,
  Steps,
  Tag,
  Row,
  Col,
  Alert,
  Select,
  TimePicker,
  Checkbox,
  message,
} from 'antd';
import {
  RotateCcw,
  Clock,
  AlertTriangle,
  CheckCircle,
  Users,
  Server,
  Database,
  Network,
  FileText,
  PlayCircle,
  PauseCircle,
  StopCircle,
} from 'lucide-react';
import dayjs from 'dayjs';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;
// Steps.Step 已被移除，直接使用 Steps 组件

const _Step = Steps;

interface ChangeRollbackPlanProps {
  changeId?: number;
  initialData?: Partial<RollbackPlanData>;
  onSave?: (data: RollbackPlanData) => void;
  readOnly?: boolean;
}

interface RollbackPlanData {
  rollbackStrategy: 'full' | 'partial' | 'data_only' | 'config_only';
  rollbackTriggers: string[];
  rollbackSteps: RollbackStep[];
  rollbackTimeWindow: {
    startTime: string;
    endTime: string;
  };
  rollbackTeam: string[];
  rollbackTestPlan: string;
  rollbackVerification: string;
  rollbackRiskAssessment: string;
}

interface RollbackStep {
  id: string;
  title: string;
  description: string;
  type: 'system' | 'data' | 'config' | 'network' | 'service';
  estimatedTime: number;
  dependencies: string[];
  responsiblePerson: string;
  verificationMethod: string;
  rollbackCommand?: string;
  successCriteria: string;
}

const ChangeRollbackPlan: React.FC<ChangeRollbackPlanProps> = ({
  changeId,
  initialData,
  onSave,
  readOnly = false,
}) => {
  const [form] = Form.useForm();
  const [rollbackSteps, setRollbackSteps] = useState<RollbackStep[]>([]);
  const [currentStep, setCurrentStep] = useState(0);
  const [submitting, setSubmitting] = useState(false);

  // 回滚策略选项
  const rollbackStrategies = {
    full: {
      label: 'Full rollback',
      description: 'Revert all changes including system, data and configuration',
      risk: 'high',
      time: 'high',
    },
    partial: {
      label: 'Partial rollback',
      description: 'Only roll back specific changed components',
      risk: 'medium',
      time: 'medium',
    },
    dataOnly: {
      label: 'Data rollback only',
      description: 'Only restore data changes, keep system and configuration changes',
      risk: 'low',
      time: 'low',
    },
    configOnly: {
      label: 'Configuration rollback only',
      description: 'Only restore configuration changes, keep data and system changes',
      risk: 'medium',
      time: 'low',
    },
  };

  // 回滚触发条件
  const rollbackTriggers = [
    'System error rate exceeds threshold',
    'Significant performance degradation',
    'Data integrity issue',
    'Security vulnerability discovered',
    'User experience is severely degraded',
    'Business process interrupted',
    'Critical service unavailable',
    'Third-party service exception',
    'Monitoring alarm exceeds threshold',
    'Increase in user complaints',
  ];

  // 步骤类型图标
  const stepTypeIcons = {
    system: <Server className="w-4 h-4 text-blue-500" />,
    data: <Database className="w-4 h-4 text-green-500" />,
    config: <FileText className="w-4 h-4 text-orange-500" />,
    network: <Network className="w-4 h-4 text-purple-500" />,
    service: <CheckCircle className="w-4 h-4 text-cyan-500" />,
  };

  useEffect(() => {
    if (initialData) {
      form.setFieldsValue(initialData);
      if (initialData.rollbackSteps) {
        setRollbackSteps(initialData.rollbackSteps);
      }
    }
  }, [initialData, form]);

  const addRollbackStep = () => {
    const newStep: RollbackStep = {
      id: `step_${Date.now()}`,
      title: '',
      description: '',
      type: 'system',
      estimatedTime: 10,
      dependencies: [],
      responsiblePerson: '',
      verificationMethod: '',
      successCriteria: '',
    };
    setRollbackSteps([...rollbackSteps, newStep]);
  };

  const updateRollbackStep = (index: number, field: keyof RollbackStep, value: unknown) => {
    const updatedSteps = [...rollbackSteps];
    updatedSteps[index] = { ...updatedSteps[index], [field]: value };
    setRollbackSteps(updatedSteps);
  };

  const deleteRollbackStep = (index: number) => {
    setRollbackSteps(rollbackSteps.filter((_, i) => i !== index));
  };

  const getTotalRollbackTime = () => {
    return rollbackSteps.reduce((total, step) => total + (step.estimatedTime || 0), 0);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      const rollbackData: RollbackPlanData = {
        ...values,
        rollbackSteps: rollbackSteps,
      };

      onSave?.(rollbackData);
    } catch (error) {
      message.error('Form validation failed');
    } finally {
      setSubmitting(false);
    }
  };

  // 回滚步骤表格列
  const stepColumns = [
    {
      title: 'Step',
      dataIndex: 'id',
      key: 'id',
      width: 80,
      render: (_: unknown, __: unknown, index: number) => index + 1,
    },
    {
      title: 'Title',
      dataIndex: 'title',
      key: 'title',
      width: 200,
      render: (text: string, record: RollbackStep, index: number) => (
        <Input
          value={text}
          onChange={e => updateRollbackStep(index, 'title', e.target.value)}
          placeholder="Enter step title"
          disabled={readOnly}
        />
      ),
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string, record: RollbackStep, index: number) => (
        <Select
          value={type}
          onChange={value => updateRollbackStep(index, 'type', value)}
          style={{ width: '100%' }}
          disabled={readOnly}
          options={Object.entries(stepTypeIcons).map(([key]) => ({
            value: key,
            label: key.charAt(0).toUpperCase() + key.slice(1),
          }))}
        />
      ),
    },
    {
      title: 'Estimated time (minutes)',
      dataIndex:'estimatedTime',
      key:'estimatedTime',
      width: 130,
      render: (time: number, record: RollbackStep, index: number) => (
        <Input
          type="number"
          value={time}
          onChange={e => updateRollbackStep(index, 'estimated_time' as any, parseInt(e.target.value) || 0)}
          disabled={readOnly}
          min={1}
        />
      ),
    },
    {
      title: 'Assignee',
      dataIndex:'responsiblePerson',
      key:'responsiblePerson',
      width: 120,
      render: (person: string, record: RollbackStep, index: number) => (
        <Input
          value={person}
          onChange={e => updateRollbackStep(index, 'responsible_person' as any, e.target.value)}
          placeholder="Assignee"
          disabled={readOnly}
        />
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 80,
      render: (_: unknown, record: RollbackStep, index: number) =>
        !readOnly ? (
          <Button type="text" danger size="small" onClick={() => deleteRollbackStep(index)}>

            Delete
          </Button>
        ) : null,
    },
  ];

  return (
    <Card
      title={
        <Space>
          <RotateCcw className="w-5 h-5 text-blue-600" />
          <span>Change rollback plan</span>
        </Space>
      }
      extra={
        !readOnly && (
          <Button type="primary" onClick={handleSubmit} loading={submitting}>

            Save rollback plan
          </Button>
        )
      }
      className="change-rollback-plan"
    >
      <Form form={form} layout="vertical" disabled={readOnly}>
        {/* 回滚策略选择 */}
        <Row gutter={[24, 16]}>
          <Col xs={24}>
            <Form.Item
              label="Rollback policy"
              name="rollbackStrategy"
              rules={[{ required: true, message: 'Please select a rollback strategy' }]}
            >
              <Select placeholder="Select rollback strategy" options={Object.entries(rollbackStrategies).map(([key, strategy]) => ({
                value: key,
                label: (
                  <div className="p-2">
                    <div className="font-medium">{strategy.label}</div>
                    <Text type="secondary" className="text-xs">
                      {strategy.description}
                    </Text>
                    <div className="mt-1">
                      <Tag
                        color={
                          strategy.risk === 'high'
                            ? 'red'
                            : strategy.risk === 'medium'
                              ? 'orange'
                              : 'green'
                        }
                      >

                        Risk: {strategy.risk}
                      </Tag>
                      <Tag
                        color={
                          strategy.time === 'high'
                            ? 'red'
                            : strategy.time === 'medium'
                              ? 'orange'
                              : 'green'
                        }
                      >

                        Time: {strategy.time}
                      </Tag>
                    </div>
                  </div>
                ),
              }))} />
            </Form.Item>
          </Col>

          {/* 回滚触发条件 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="Rollback trigger condition"
              name="rollbackTriggers"
              rules={[{ required: true, message: 'Please select trigger conditions' }]}
            >
              <Select mode="multiple" placeholder="Select conditions that trigger rollback" style={{ width: '100%' }} options={rollbackTriggers.map(trigger => ({
                value: trigger,
                label: (
                  <Space>
                    <AlertTriangle className="w-4 h-4 text-orange-500" />
                    {trigger}
                  </Space>
                ),
              }))} />
            </Form.Item>
          </Col>

          {/* 回滚团队 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="Rollback Team"
              name="rollbackTeam"
              rules={[{ required: true, message: 'Please specify rollback team members' }]}
            >
              <Select mode="tags" placeholder="Specify rollback team members" style={{ width: '100%' }} options={[
                { value: '系统管理员', label: 'System Administrator' },
                { value: '数据库管理员', label: 'Database Administrator' },
                { value: '网络工程师', label: 'Network Engineer' },
                { value: '应用开发', label: 'Application Development' },
                { value: '安全专员', label: 'Security Specialist' },
              ]} />
            </Form.Item>
          </Col>
        </Row>

        {/* 回滚步骤 */}
        <Card size="small" className="mb-4">
          <div className="flex justify-between items-center mb-4">
            <Title level={5} className="mb-0">

              Rollback step details
            </Title>
            <Space>
              <Text type="secondary">Total estimated time: {getTotalRollbackTime()}  minutes</Text>
              {!readOnly && (
                <Button type="dashed" icon={<FileText />} onClick={addRollbackStep}>

                  Add Step
                </Button>
              )}
            </Space>
          </div>

          {rollbackSteps.length === 0 ? (
            <div className="text-center py-8">
              <RotateCcw className="w-12 h-12 text-gray-400 mx-auto mb-4" />
              <Text type="secondary">There are no rollback steps yet. Please add detailed rollback steps.</Text>
            </div>
          ) : (
            <Table
              columns={stepColumns}
              dataSource={rollbackSteps}
              rowKey="id"
              scroll={{ x: 'max-content' }}
              pagination={false}
              size="small"
              className="rollback-steps-table"
            />
          )}
        </Card>

        {/* 测试计划 */}
        <Row gutter={[24, 16]}>
          <Col xs={24} lg={12}>
            <Form.Item
              label="Rollback test plan"
              name="rollbackTestPlan"
              rules={[{ required: true, message: 'Please develop a test plan' }]}
            >
              <TextArea rows={4} placeholder="Describe the test plan before rollback..." maxLength={1000} showCount />
            </Form.Item>
          </Col>

          <Col xs={24} lg={12}>
            <Form.Item
              label="Rollback verification scheme"
              name="rollbackVerification"
              rules={[{ required: true, message: 'Please develop a verification plan' }]}
            >
              <TextArea
                rows={4}
                placeholder="Describe the verification method after successful rollback..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 风险评估 */}
          <Col xs={24}>
            <Form.Item
              label="Rollback risk assessment"
              name="rollbackRiskAssessment"
              rules={[{ required: true, message: 'Please conduct a risk assessment' }]}
            >
              <TextArea
                rows={4}
                placeholder="Assess possible risks and countermeasures during rollback..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>
        </Row>

        {/* 时间窗口建议 */}
        <Alert
          title="Rollback time window suggestion"
          description={
            <div>
              <p>It is recommended to set the following time window for rollback operation:</p>
              <ul>
                <li>Low business peak period (usually 2-5 am)</li>
                <li>Avoid important business activities and billing cycles</li>
                <li>Ensure sufficient time to complete rollback and verification</li>
                <li>Notify all relevant parties in advance</li>
              </ul>
            </div>
          }
          type="info"
          showIcon
          className="mt-4"
        />
      </Form>
    </Card>
  );
};

export default ChangeRollbackPlan;
