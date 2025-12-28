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
const { Option } = Select;
const { Step } = Steps;

interface ChangeRollbackPlanProps {
  changeId?: number;
  initialData?: Partial<RollbackPlanData>;
  onSave?: (data: RollbackPlanData) => void;
  readOnly?: boolean;
}

interface RollbackPlanData {
  rollback_strategy: 'full' | 'partial' | 'data_only' | 'config_only';
  rollback_triggers: string[];
  rollback_steps: RollbackStep[];
  rollback_time_window: {
    start_time: string;
    end_time: string;
  };
  rollback_team: string[];
  rollback_test_plan: string;
  rollback_verification: string;
  rollback_risk_assessment: string;
}

interface RollbackStep {
  id: string;
  title: string;
  description: string;
  type: 'system' | 'data' | 'config' | 'network' | 'service';
  estimated_time: number;
  dependencies: string[];
  responsible_person: string;
  verification_method: string;
  rollback_command?: string;
  success_criteria: string;
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

  // 回滚策略选项
  const rollbackStrategies = {
    full: {
      label: '完整回滚',
      description: '恢复所有变更，包括系统、数据和配置',
      risk: 'high',
      time: 'high',
    },
    partial: {
      label: '部分回滚',
      description: '只回滚特定的变更组件',
      risk: 'medium',
      time: 'medium',
    },
    data_only: {
      label: '仅数据回滚',
      description: '只恢复数据变更，保持系统和配置变更',
      risk: 'low',
      time: 'low',
    },
    config_only: {
      label: '仅配置回滚',
      description: '只恢复配置变更，保持数据和系统变更',
      risk: 'medium',
      time: 'low',
    },
  };

  // 回滚触发条件
  const rollbackTriggers = [
    '系统错误率超过阈值',
    '性能显著下降',
    '数据完整性问题',
    '安全漏洞发现',
    '用户体验严重下降',
    '业务流程中断',
    '关键服务不可用',
    '第三方服务异常',
    '监控告警超过阈值',
    '用户投诉激增',
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
      if (initialData.rollback_steps) {
        setRollbackSteps(initialData.rollback_steps);
      }
    }
  }, [initialData, form]);

  const addRollbackStep = () => {
    const newStep: RollbackStep = {
      id: `step_${Date.now()}`,
      title: '',
      description: '',
      type: 'system',
      estimated_time: 10,
      dependencies: [],
      responsible_person: '',
      verification_method: '',
      success_criteria: '',
    };
    setRollbackSteps([...rollbackSteps, newStep]);
  };

  const updateRollbackStep = (index: number, field: keyof RollbackStep, value: any) => {
    const updatedSteps = [...rollbackSteps];
    updatedSteps[index] = { ...updatedSteps[index], [field]: value };
    setRollbackSteps(updatedSteps);
  };

  const deleteRollbackStep = (index: number) => {
    setRollbackSteps(rollbackSteps.filter((_, i) => i !== index));
  };

  const getTotalRollbackTime = () => {
    return rollbackSteps.reduce((total, step) => total + (step.estimated_time || 0), 0);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      const rollbackData: RollbackPlanData = {
        ...values,
        rollback_steps: rollbackSteps,
      };
      
      onSave?.(rollbackData);
    } catch (error) {
      console.error('Form validation failed:', error);
    }
  };

  // 回滚步骤表格列
  const stepColumns = [
    {
      title: '步骤',
      dataIndex: 'id',
      key: 'id',
      width: 80,
      render: (_: any, __: any, index: number) => index + 1,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      width: 200,
      render: (text: string, record: RollbackStep, index: number) => (
        <Input
          value={text}
          onChange={(e) => updateRollbackStep(index, 'title', e.target.value)}
          placeholder="输入步骤标题"
          disabled={readOnly}
        />
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string, record: RollbackStep, index: number) => (
        <Select
          value={type}
          onChange={(value) => updateRollbackStep(index, 'type', value)}
          style={{ width: '100%' }}
          disabled={readOnly}
        >
          {Object.entries(stepTypeIcons).map(([key, icon]) => (
            <Option key={key} value={key}>
              <Space>
                {icon}
                <span>{key.charAt(0).toUpperCase() + key.slice(1)}</span>
              </Space>
            </Option>
          ))}
        </Select>
      ),
    },
    {
      title: '预计时间(分钟)',
      dataIndex: 'estimated_time',
      key: 'estimated_time',
      width: 130,
      render: (time: number, record: RollbackStep, index: number) => (
        <Input
          type="number"
          value={time}
          onChange={(e) => updateRollbackStep(index, 'estimated_time', parseInt(e.target.value) || 0)}
          disabled={readOnly}
          min={1}
        />
      ),
    },
    {
      title: '负责人',
      dataIndex: 'responsible_person',
      key: 'responsible_person',
      width: 120,
      render: (person: string, record: RollbackStep, index: number) => (
        <Input
          value={person}
          onChange={(e) => updateRollbackStep(index, 'responsible_person', e.target.value)}
          placeholder="负责人"
          disabled={readOnly}
        />
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 80,
      render: (_: any, record: RollbackStep, index: number) => (
        !readOnly && (
          <Button
            type="text"
            danger
            size="small"
            onClick={() => deleteRollbackStep(index)}
          >
            删除
          </Button>
        ),
    },
  ];

  return (
    <Card
      title={
        <Space>
          <RotateCcw className="w-5 h-5 text-blue-600" />
          <span>变更回滚计划</span>
        </Space>
      }
      extra={
        !readOnly && (
          <Button type="primary" onClick={handleSubmit}>
            保存回滚计划
          </Button>
        )
      }
      className="change-rollback-plan"
    >
      <Form
        form={form}
        layout="vertical"
        disabled={readOnly}
      >
        {/* 回滚策略选择 */}
        <Row gutter={[24, 16]}>
          <Col xs={24}>
            <Form.Item
              label="回滚策略"
              name="rollback_strategy"
              rules={[{ required: true, message: '请选择回滚策略' }]}
            >
              <Select placeholder="选择回滚策略">
                {Object.entries(rollbackStrategies).map(([key, strategy]) => (
                  <Option key={key} value={key}>
                    <div className="p-2">
                      <div className="font-medium">{strategy.label}</div>
                      <Text type="secondary" className="text-xs">{strategy.description}</Text>
                      <div className="mt-1">
                        <Tag color={strategy.risk === 'high' ? 'red' : strategy.risk === 'medium' ? 'orange' : 'green'}>
                          风险: {strategy.risk}
                        </Tag>
                        <Tag color={strategy.time === 'high' ? 'red' : strategy.time === 'medium' ? 'orange' : 'green'}>
                          时间: {strategy.time}
                        </Tag>
                      </div>
                    </div>
                  </Option>
                ))}
              </Select>
            </Form.Item>
          </Col>

          {/* 回滚触发条件 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="回滚触发条件"
              name="rollback_triggers"
              rules={[{ required: true, message: '请选择触发条件' }]}
            >
              <Select
                mode="multiple"
                placeholder="选择触发回滚的条件"
                style={{ width: '100%' }}
              >
                {rollbackTriggers.map(trigger => (
                  <Option key={trigger} value={trigger}>
                    <Space>
                      <AlertTriangle className="w-4 h-4 text-orange-500" />
                      {trigger}
                    </Space>
                  </Option>
                ))}
              </Select>
            </Form.Item>
          </Col>

          {/* 回滚团队 */}
          <Col xs={24} lg={12}>
            <Form.Item
              label="回滚团队"
              name="rollback_team"
              rules={[{ required: true, message: '请指定回滚团队成员' }]}
            >
              <Select
                mode="tags"
                placeholder="指定回滚团队成员"
                style={{ width: '100%' }}
              >
                <Option value="系统管理员">系统管理员</Option>
                <Option value="数据库管理员">数据库管理员</Option>
                <Option value="网络工程师">网络工程师</Option>
                <Option value="应用开发">应用开发</Option>
                <Option value="安全专员">安全专员</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>

        {/* 回滚步骤 */}
        <Card size="small" className="mb-4">
          <div className="flex justify-between items-center mb-4">
            <Title level={5} className="mb-0">
              回滚步骤明细
            </Title>
            <Space>
              <Text type="secondary">
                总预计时间: {getTotalRollbackTime()} 分钟
              </Text>
              {!readOnly && (
                <Button type="dashed" icon={<FileText />} onClick={addRollbackStep}>
                  添加步骤
                </Button>
              )}
            </Space>
          </div>

          {rollbackSteps.length === 0 ? (
            <div className="text-center py-8">
              <RotateCcw className="w-12 h-12 text-gray-400 mx-auto mb-4" />
              <Text type="secondary">暂无回滚步骤，请添加详细的回滚步骤</Text>
            </div>
          ) : (
            <Table
              columns={stepColumns}
              dataSource={rollbackSteps}
              rowKey="id"
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
              label="回滚测试计划"
              name="rollback_test_plan"
              rules={[{ required: true, message: '请制定测试计划' }]}
            >
              <TextArea
                rows={4}
                placeholder="描述回滚前的测试方案..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          <Col xs={24} lg={12}>
            <Form.Item
              label="回滚验证方案"
              name="rollback_verification"
              rules={[{ required: true, message: '请制定验证方案' }]}
            >
              <TextArea
                rows={4}
                placeholder="描述回滚成功后的验证方法..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>

          {/* 风险评估 */}
          <Col xs={24}>
            <Form.Item
              label="回滚风险评估"
              name="rollback_risk_assessment"
              rules={[{ required: true, message: '请进行风险评估' }]}
            >
              <TextArea
                rows={4}
                placeholder="评估回滚过程可能遇到的风险和应对措施..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>
        </Row>

        {/* 时间窗口建议 */}
        <Alert
          message="回滚时间窗口建议"
          description={
            <div>
              <p>建议设置以下时间窗口进行回滚操作：</p>
              <ul>
                <li>业务低峰期（通常为凌晨2-5点）</li>
                <li>避开重要业务活动和结算周期</li>
                <li>确保有足够的时间完成回滚和验证</li>
                <li>提前通知所有相关方</li>
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