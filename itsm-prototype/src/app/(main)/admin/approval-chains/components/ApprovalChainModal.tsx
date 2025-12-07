/**
 * 审批链模态框组件
 */

import React, { useState, useEffect, useCallback } from 'react';
import {
  Modal,
  Form,
  Input,
  Switch,
  Button,
  Steps,
  Card,
  Space,
  Select,
  InputNumber,
  Divider,
  Typography,
  message,
  Row,
  Col,
} from 'antd';
import { PlusOutlined, MinusCircleOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  ApprovalChain,
  ApprovalStep,
  CreateApprovalChainRequest,
  UpdateApprovalChainRequest,
} from '@/types/approval-chain';
import { FormField } from '@/types/common';

const { TextArea } = Input;
const { Option } = Select;
const { Text, Title } = Typography;
const { Step } = Steps;

interface ApprovalChainModalProps {
  visible: boolean;
  editingChain: ApprovalChain | null;
  onCancel: () => void;
  onSubmit: (data: CreateApprovalChainRequest | UpdateApprovalChainRequest) => Promise<void>;
  loading?: boolean;
}

export function ApprovalChainModal({
  visible,
  editingChain,
  onCancel,
  onSubmit,
  loading = false,
}: ApprovalChainModalProps) {
  const [form] = Form.useForm();
  const [currentStep, setCurrentStep] = useState(0);
  const [steps, setSteps] = useState<
    Omit<ApprovalStep, 'id' | 'chainId' | 'createdAt' | 'updatedAt'>[]
  >([]);

  // 重置表单
  const resetForm = useCallback(() => {
    form.resetFields();
    setCurrentStep(0);
    setSteps([]);
  }, [form]);

  // 初始化表单
  useEffect(() => {
    if (visible) {
      if (editingChain) {
        form.setFieldsValue({
          name: editingChain.name,
          description: editingChain.description,
          isActive: editingChain.isActive,
        });
        setSteps(
          editingChain.steps.map((step: ApprovalStep) => ({
            stepOrder: step.stepOrder,
            stepName: step.stepName,
            approverType: step.approverType,
            approverId: step.approverId,
            approverName: step.approverName,
            isRequired: step.isRequired,
            timeoutHours: step.timeoutHours,
            conditions: step.conditions,
          }))
        );
      } else {
        resetForm();
      }
    }
  }, [visible, editingChain, form, resetForm]);

  // 处理表单提交
  const handleSubmit = useCallback(async () => {
    try {
      const values = await form.validateFields();

      const data = {
        ...values,
        steps: steps.map((step, index) => ({
          ...step,
          stepOrder: index + 1,
        })),
      };

      await onSubmit(data);
      resetForm();
    } catch (error) {
      console.error('Form validation failed:', error);
    }
  }, [form, steps, onSubmit, resetForm]);

  // 添加步骤
  const handleAddStep = useCallback(() => {
    const newStep: Omit<ApprovalStep, 'id' | 'chainId' | 'createdAt' | 'updatedAt'> = {
      stepOrder: steps.length + 1,
      stepName: `步骤 ${steps.length + 1}`,
      approverType: 'user',
      approverId: 0,
      approverName: '',
      isRequired: true,
      timeoutHours: 24,
      conditions: [],
    };
    setSteps(prev => [...prev, newStep]);
  }, [steps.length]);

  // 删除步骤
  const handleRemoveStep = useCallback((index: number) => {
    setSteps(prev => prev.filter((_, i) => i !== index));
  }, []);

  // 更新步骤
  const handleUpdateStep = useCallback((index: number, field: string, value: unknown) => {
    setSteps(prev => prev.map((step, i) => (i === index ? { ...step, [field]: value } : step)));
  }, []);

  // 渲染步骤配置
  const renderStepConfig = useCallback(() => {
    return (
      <div className='space-y-4'>
        <div className='flex justify-between items-center'>
          <Title level={5}>审批步骤配置</Title>
          <Button type='dashed' icon={<PlusOutlined />} onClick={handleAddStep}>
            添加步骤
          </Button>
        </div>

        {steps.map((step, index) => (
          <Card key={index} size='small' className='relative'>
            <div className='flex justify-between items-start mb-4'>
              <div className='flex items-center gap-2'>
                <span className='font-medium'>步骤 {index + 1}</span>
                <Input
                  value={step.stepName}
                  onChange={e => handleUpdateStep(index, 'stepName', e.target.value)}
                  placeholder='步骤名称'
                  style={{ width: 200 }}
                />
              </div>
              <Button
                type='text'
                danger
                icon={<MinusCircleOutlined />}
                onClick={() => handleRemoveStep(index)}
                size='small'
              />
            </div>

            <Row gutter={16}>
              <Col span={8}>
                <div className='mb-2'>
                  <Text strong>审批人类型</Text>
                </div>
                <Select
                  value={step.approverType}
                  onChange={value => handleUpdateStep(index, 'approverType', value)}
                  style={{ width: '100%' }}
                >
                  <Option value='user'>用户</Option>
                  <Option value='role'>角色</Option>
                  <Option value='group'>组</Option>
                </Select>
              </Col>

              <Col span={8}>
                <div className='mb-2'>
                  <Text strong>审批人</Text>
                </div>
                <Input
                  value={step.approverName}
                  onChange={e => handleUpdateStep(index, 'approverName', e.target.value)}
                  placeholder='审批人名称'
                />
              </Col>

              <Col span={8}>
                <div className='mb-2'>
                  <Text strong>超时时间(小时)</Text>
                </div>
                <InputNumber
                  value={step.timeoutHours}
                  onChange={value => handleUpdateStep(index, 'timeoutHours', value)}
                  min={1}
                  max={168}
                  style={{ width: '100%' }}
                />
              </Col>
            </Row>

            <div className='mt-4'>
              <Switch
                checked={step.isRequired}
                onChange={checked => handleUpdateStep(index, 'isRequired', checked)}
                checkedChildren='必审'
                unCheckedChildren='可选'
              />
            </div>
          </Card>
        ))}

        {steps.length === 0 && (
          <div className='text-center py-8 text-gray-500'>
            <Text>暂无审批步骤，请添加步骤</Text>
          </div>
        )}
      </div>
    );
  }, [steps, handleAddStep, handleRemoveStep, handleUpdateStep]);

  // 步骤配置
  const stepItems = [
    {
      title: '基本信息',
      description: '设置审批链基本信息',
    },
    {
      title: '步骤配置',
      description: '配置审批步骤',
    },
  ];

  return (
    <Modal
      title={editingChain ? '编辑审批链' : '创建审批链'}
      open={visible}
      onCancel={onCancel}
      onOk={handleSubmit}
      confirmLoading={loading}
      width={800}
      destroyOnHidden
    >
      <Steps current={currentStep} className='mb-6'>
        {stepItems.map((item, index) => (
          <Step key={index} title={item.title} description={item.description} />
        ))}
      </Steps>

      {currentStep === 0 && (
        <Form form={form} layout='vertical' preserve={false}>
          <Form.Item
            name='name'
            label='审批链名称'
            rules={[
              { required: true, message: '请输入审批链名称' },
              { max: 100, message: '名称不能超过100个字符' },
            ]}
          >
            <Input placeholder='请输入审批链名称' />
          </Form.Item>

          <Form.Item
            name='description'
            label='描述'
            rules={[{ max: 500, message: '描述不能超过500个字符' }]}
          >
            <TextArea rows={3} placeholder='请输入审批链描述' />
          </Form.Item>

          <Form.Item name='isActive' label='状态' valuePropName='checked' initialValue={true}>
            <Switch checkedChildren='启用' unCheckedChildren='禁用' />
          </Form.Item>
        </Form>
      )}

      {currentStep === 1 && renderStepConfig()}

      <div className='flex justify-between mt-6'>
        <Button onClick={onCancel}>取消</Button>

        <Space>
          {currentStep > 0 && (
            <Button onClick={() => setCurrentStep(currentStep - 1)}>上一步</Button>
          )}

          {currentStep < stepItems.length - 1 ? (
            <Button type='primary' onClick={() => setCurrentStep(currentStep + 1)}>
              下一步
            </Button>
          ) : (
            <Button type='primary' onClick={handleSubmit} loading={loading}>
              {editingChain ? '更新' : '创建'}
            </Button>
          )}
        </Space>
      </div>
    </Modal>
  );
}
