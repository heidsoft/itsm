'use client';

/**
 * 审批链模态框组件
 */

import React, { useState, useEffect, useCallback } from 'react';
import {
  Modal,
  Form,
  Input,
  InputNumber,
  Switch,
  Button,
  Steps,
  Card,
  Space,
  Select,
  Collapse,
  Typography,
  App,
  Row,
  Col,
  Spin,
} from 'antd';
import { Plus, MinusCircle, Edit, Trash2 } from 'lucide-react';
import type {
  ApprovalChain,
  ApprovalStep,
  CreateApprovalChainRequest,
  UpdateApprovalChainRequest,
  FallbackAction,
} from '@/types/approval-chain';
import {
  APPROVAL_TYPE_LABELS,
  QUORUM_APPROVAL_TYPES,
  FALLBACK_ACTION_LABELS,
  APPROVAL_CONDITION_PRIORITIES,
  PRIORITY_LABELS,
  ENTITY_TYPE_LABELS,
  type ApprovalType,
} from '@/types/approval-chain';
import { FormField } from '@/types/common';
import { UserApi } from '@/lib/api/user-api';
import type { User } from '@/lib/api/user-api';
import { RoleAPI } from '@/lib/api/role-api';
import type { Role } from '@/lib/api/api-config';

const { TextArea } = Input;

const { Text, Title } = Typography;
// Step removed - antd 5+ uses items prop instead

// 第一步「基本信息」的 Form 字段名：用于分步校验与创建失败时的自动回跳定位
const STEP_BASIC_FIELDS = ['name', 'description', 'entityType', 'isActive'];

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
  // 反馈 API 必须取自 App 上下文，静态 message 不继承主题与 locale
  const { message } = App.useApp();
  const [currentStep, setCurrentStep] = useState(0);
  const [steps, setSteps] = useState<
    Omit<ApprovalStep, 'id' | 'chainId' | 'createdAt' | 'updatedAt'>[]
  >([]);
  const [users, setUsers] = useState<User[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [roles, setRoles] = useState<Role[]>([]);
  const [rolesLoading, setRolesLoading] = useState(false);

  // 获取可选审批用户（远程搜索）
  const fetchUsers = useCallback(
    async (search?: string) => {
      try {
        setUsersLoading(true);
        const response = await UserApi.getUsers({ page: 1, pageSize: 100, search: search || '' });
        setUsers(response.items || []);
      } catch (error) {
        message.error('Failed to obtain user list');
      } finally {
        setUsersLoading(false);
      }
    },
    [message]
  );

  // 获取可选审批角色
  const fetchRoles = useCallback(async () => {
    try {
      setRolesLoading(true);
      const response = await RoleAPI.getRoles({ page: 1, pageSize: 100 });
      setRoles(response.items);
    } catch (error) {
      message.error('Failed to obtain role list');
    } finally {
      setRolesLoading(false);
    }
  }, [message]);

  useEffect(() => {
    if (visible) {
      fetchUsers();
      fetchRoles();
    }
  }, [visible, fetchUsers, fetchRoles]);

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
          entityType: editingChain.entityType || 'ticket',
          isActive: editingChain.isActive,
        });
        setSteps(
          editingChain.steps.map((step: ApprovalStep) => ({
            level: step.level,
            stepOrder: step.stepOrder,
            stepName: step.stepName,
            approverType: step.approverType,
            approverId: step.approverId,
            approverName: step.approverName,
            isRequired: step.isRequired,
            approvalType: step.approvalType,
            threshold: step.threshold,
            fallbackAction: step.fallbackAction,
            fallbackApproverId: step.fallbackApproverId,
            fallbackRole: step.fallbackRole,
            conditionPriorities: step.conditionPriorities,
            conditionAmountMin: step.conditionAmountMin,
            conditionAmountMax: step.conditionAmountMax,
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

      if (steps.length === 0) {
        message.error('Please add at least one approval step');
        return;
      }

      // 每个步骤必须绑定真实审批人，避免落库后审批路由失效
      const invalidIndex = steps.findIndex(step =>
        step.approverType === 'user'
          ? !step.approverId || step.approverId <= 0
          : !step.approverName?.trim()
      );
      if (invalidIndex >= 0) {
        const step = steps[invalidIndex];
        message.error(
          `Step ${invalidIndex + 1}（${step.stepName}) not selected ${
            step.approverType === 'user' ? 'Approval user' : 'Approval role'
          }`
        );
        return;
      }

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
      // D52：表单校验失败时，若错误落在第一步字段（隐藏容器内渲染，用户不可见），
      // 必须自动跳回第一步并给出可见提示，否则表现为「点了没反应」。
      const errorFields = (error as { errorFields?: { name: (string | number)[] }[] })
        ?.errorFields;
      if (
        errorFields?.some(field => STEP_BASIC_FIELDS.includes(String(field.name[0])))
      ) {
        setCurrentStep(0);
        message.error('Basic information is invalid. Please review it and try again.');
      }
    }
  }, [form, steps, onSubmit, resetForm, message]);

  // 「Next」必须先通过当前步骤的字段校验（D52：修复前直接放行，
  // 导致空名称也能进入步骤配置，创建时错误又渲染在隐藏容器里不可见）
  const handleNextStep = useCallback(async () => {
    try {
      await form.validateFields(STEP_BASIC_FIELDS);
      setCurrentStep(prev => prev + 1);
    } catch {
      // 校验错误已渲染在字段旁，停留在当前步骤
    }
  }, [form]);

  // 添加步骤
  const handleAddStep = useCallback(() => {
    const newStep: Omit<ApprovalStep, 'id' | 'chainId' | 'createdAt' | 'updatedAt'> = {
      level: steps.length + 1,
      stepOrder: steps.length + 1,
      stepName: `Step ${steps.length + 1}`,
      approverType: 'user',
      approverId: 0,
      approverName: '',
      isRequired: true,
      approvalType: 'serial',
      fallbackAction: 'block',
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
    const approvalTypeOptions = (Object.keys(APPROVAL_TYPE_LABELS) as ApprovalType[]).map(value => ({
      value,
      label: APPROVAL_TYPE_LABELS[value],
    }));

    return (
      <div className="space-y-4">
        <div className="flex justify-between items-center">
          <Title level={5}>Approval step configuration</Title>
          <Button type="dashed" icon={<Plus className="w-4 h-4" />} onClick={handleAddStep}>

            Add Step
          </Button>
        </div>

        {steps.map((step, index) => (
          <Card key={index} size="small" className="relative">
            <div className="flex justify-between items-start mb-4">
              <div className="flex items-center gap-2">
                <span className="font-medium">Step {index + 1}</span>
                <Input
                  value={step.stepName}
                  onChange={e => handleUpdateStep(index, 'stepName', e.target.value)}
                  placeholder="step name"
                  style={{ width: 200 }}
                />
              </div>
              <Button
                type="text"
                aria-label={`Delete the ${index + 1} approval steps`}
                danger
                icon={<MinusCircle className="w-4 h-4" />}
                onClick={() => handleRemoveStep(index)}
                size="small"
              />
            </div>

            <Row gutter={16}>
              <Col span={8}>
                <div className="mb-2">
                  <Text strong>Approval level</Text>
                </div>
                <InputNumber
                  aria-label={`Step ${index + 1} Approval level`}
                  value={step.level ?? index + 1}
                  onChange={value => handleUpdateStep(index, 'level', value ?? index + 1)}
                  min={1}
                  max={20}
                  style={{ width: '100%' }}
                />
                <Text type="secondary" className="text-xs mt-1 block">

                  Multiple steps at the same level are merged according to "Approval Method"
                </Text>
              </Col>

              <Col span={8}>
                <div className="mb-2">
                  <Text strong>Approver type</Text>
                </div>
                <Select
                  value={step.approverType}
                  onChange={value => {
                    // 切换类型时清空原有绑定，防止残留错误审批人
                    handleUpdateStep(index, 'approverType', value);
                    handleUpdateStep(index, 'approverId', 0);
                    handleUpdateStep(index, 'approverName', '');
                  }}
                  style={{ width: '100%' }}
                  options={[
                    { value: 'user', label: 'Users' },
                    { value: 'role', label: 'Role' },
                  ]}
                />
              </Col>

              <Col span={8}>
                <div className="mb-2">
                  <Text strong>Approver</Text>
                </div>
                {step.approverType === 'user' ? (
                  <Select
                    showSearch
                    filterOption={false}
                    onSearch={fetchUsers}
                    value={step.approverId || undefined}
                    placeholder="Search and select approval users"
                    style={{ width: '100%' }}
                    loading={usersLoading}
                    notFoundContent={usersLoading ? <Spin size="small" /> : 'User not found'}
                    optionLabelProp="label"
                    options={users.map(user => ({
                      value: user.id,
                      label: user.name || user.username,
                    }))}
                    onChange={(userId: number) => {
                      const user = users.find(u => u.id === userId);
                      handleUpdateStep(index, 'approverId', userId);
                      handleUpdateStep(index, 'approverName', user?.name || user?.username || '');
                    }}
                  />
                ) : (
                  <Select
                    showSearch
                    optionFilterProp="label"
                    value={step.approverName || undefined}
                    placeholder="Select approval role"
                    style={{ width: '100%' }}
                    loading={rolesLoading}
                    options={roles.map(role => ({
                      value: role.code || role.name,
                      label: role.name,
                    }))}
                    onChange={(roleKey: string) => {
                      const role = roles.find(r => (r.code || r.name) === roleKey);
                      handleUpdateStep(index, 'approverName', role?.code || role?.name || roleKey);
                      handleUpdateStep(index, 'approverId', 0);
                    }}
                  />
                )}
              </Col>
            </Row>

            <Row gutter={16} className="mt-4">
              <Col span={12}>
                <div className="mb-2">
                  <Text strong>Approval method</Text>
                </div>
                <Select
                  aria-label={`Step ${index + 1} Approval method`}
                  value={step.approvalType || 'serial'}
                  onChange={value => handleUpdateStep(index, 'approvalType', value)}
                  style={{ width: '100%' }}
                  options={approvalTypeOptions}
                />
              </Col>

              <Col span={6}>
                <div className="mb-2">
                  <Text strong>Passed threshold</Text>
                </div>
                <InputNumber
                  aria-label={`Step ${index + 1} Passed threshold`}
                  value={step.threshold}
                  onChange={value => handleUpdateStep(index, 'threshold', value ?? undefined)}
                  min={1}
                  max={50}
                  placeholder="Default is based on the number of people on this floor"
                  disabled={!QUORUM_APPROVAL_TYPES.includes(step.approvalType || 'serial')}
                  style={{ width: '100%' }}
                />
              </Col>

              <Col span={6}>
                <div className="mb-2">
                  <Text strong>Is it required to be reviewed?</Text>
                </div>
                <Switch
                  aria-label="Is this step a required step?"
                  checked={step.isRequired}
                  onChange={checked => handleUpdateStep(index, 'isRequired', checked)}
                  checkedChildren="Required"
                  unCheckedChildren="Optional"
                />
              </Col>
            </Row>

            <Collapse
              size="small"
              className="mt-3"
              items={[
                {
                  key: 'advanced',
                  label: 'Advanced configuration (covering policy/validation conditions)',
                  children: (
                    <div className="space-y-3">
                      <Row gutter={16}>
                        <Col span={8}>
                          <div className="mb-2">
                            <Text strong>Keeping secrets when no one can judge</Text>
                          </div>
                          <Select
                            aria-label={`Step ${index + 1} Cover-up strategy`}
                            value={step.fallbackAction || 'block'}
                            onChange={value =>
                              handleUpdateStep(index, 'fallbackAction', value as FallbackAction)
                            }
                            style={{ width: '100%' }}
                            options={(Object.keys(FALLBACK_ACTION_LABELS) as FallbackAction[]).map(
                              value => ({ value, label: FALLBACK_ACTION_LABELS[value] })
                            )}
                          />
                        </Col>

                        {step.fallbackAction === 'escalate' && (
                          <>
                            <Col span={8}>
                              <div className="mb-2">
                                <Text strong>Secret approver</Text>
                              </div>
                              <Select
                                showSearch
                                filterOption={false}
                                onSearch={fetchUsers}
                                value={step.fallbackApproverId || undefined}
                                placeholder="Search and select users for approval"
                                style={{ width: '100%' }}
                                loading={usersLoading}
                                notFoundContent={usersLoading ? <Spin size="small" /> : 'User not found'}
                                optionLabelProp="label"
                                allowClear
                                options={users.map(user => ({
                                  value: user.id,
                                  label: user.name || user.username,
                                }))}
                                onChange={(userId: number) => {
                                  handleUpdateStep(index, 'fallbackApproverId', userId || 0);
                                }}
                              />
                            </Col>

                            <Col span={8}>
                              <div className="mb-2">
                                <Text strong>Secret approval role</Text>
                              </div>
                              <Select
                                showSearch
                                optionFilterProp="label"
                                value={step.fallbackRole || undefined}
                                placeholder="Select the overall approval role"
                                style={{ width: '100%' }}
                                loading={rolesLoading}
                                allowClear
                                options={roles.map(role => ({
                                  value: role.code || role.name,
                                  label: role.name,
                                }))}
                                onChange={(roleKey: string) =>
                                  handleUpdateStep(index, 'fallbackRole', roleKey || '')
                                }
                              />
                            </Col>
                          </>
                        )}
                      </Row>

                      <Row gutter={16}>
                        <Col span={12}>
                          <div className="mb-2">
                            <Text strong>Effective priority</Text>
                          </div>
                          <Select
                            mode="multiple"
                            aria-label={`Step ${index + 1} Effective priority`}
                            value={step.conditionPriorities}
                            placeholder="Leave blank to indicate no priority filtering"
                            style={{ width: '100%' }}
                            allowClear
                            options={APPROVAL_CONDITION_PRIORITIES.map(value => ({
                              value,
                              label: PRIORITY_LABELS[value] ?? value,
                            }))}
                            onChange={value =>
                              handleUpdateStep(
                                index,
                                'conditionPriorities',
                                (value as string[]).length > 0 ? value : undefined
                              )
                            }
                          />
                        </Col>

                        <Col span={6}>
                          <div className="mb-2">
                            <Text strong>Minimum amount</Text>
                          </div>
                          <InputNumber
                            aria-label={`Step ${index + 1} Minimum amount`}
                            value={step.conditionAmountMin}
                            onChange={value =>
                              handleUpdateStep(index, 'conditionAmountMin', value ?? undefined)
                            }
                            min={0}
                            placeholder="No limit"
                            style={{ width: '100%' }}
                          />
                        </Col>

                        <Col span={6}>
                          <div className="mb-2">
                            <Text strong>Amount limit</Text>
                          </div>
                          <InputNumber
                            aria-label={`Step ${index + 1} Amount limit`}
                            value={step.conditionAmountMax}
                            onChange={value =>
                              handleUpdateStep(index, 'conditionAmountMax', value ?? undefined)
                            }
                            min={0}
                            placeholder="No limit"
                            style={{ width: '100%' }}
                          />
                        </Col>
                      </Row>

                      <Text type="secondary" className="text-xs block">

                        The blank approver/role indicates analysis by role; this step is applicable only if the amount and priority conditions are met.
                      </Text>
                    </div>
                  ),
                },
              ]}
            />
          </Card>
        ))}

        {steps.length === 0 && (
          <div className="text-center py-8 text-gray-500">
            <Text>There are no approval steps yet, please add steps.</Text>
          </div>
        )}
      </div>
    );
  }, [
    steps,
    handleAddStep,
    handleRemoveStep,
    handleUpdateStep,
    fetchUsers,
    users,
    usersLoading,
    roles,
    rolesLoading,
  ]);

  // 步骤配置
  const stepItems = [
    {
      title: 'Basic Info',
      description: 'Set basic information of the approval chain',
    },
    {
      title: 'Step configuration',
      description: 'Configure approval steps',
    },
  ];

  return (
    <Modal
      title={editingChain ? 'EditApproval Chains' : 'Create approval chain'}
      open={visible}
      onCancel={onCancel}
      // 只保留向导内部的一套操作区：默认 footer 的「确定」与「创建/更新」重复提交
      footer={null}
      width={800}
      styles={{ body: { maxHeight: '65vh', overflowY: 'auto' } }}
      destroyOnHidden
    >
      <Steps
        current={currentStep}
        className="mb-6"
        items={stepItems.map((item, index) => ({
          key: index,
          title: item.title,
          description: item.description,
        }))}
      />

      {/*
        Form 必须始终挂载，两个步骤仅用 display:none 切换：
        否则 step 0 ↔ step 1 切换时 Form 卸载，name/description/entityType/isActive 被清空。
      */}
      <Form form={form} layout="vertical" component="div">
        <div data-testid="approval-chain-step-basic" style={{ display: currentStep === 0 ? 'block' : 'none' }}>
          <Form.Item
            name="name"
            label="Approval ChainsName"
            rules={[
              { required: true, message: 'Please enter Approval ChainsName' },
              { max: 100, message: 'Name cannot exceed 100 characters' },
            ]}
          >
            <Input placeholder="Please enter Approval ChainsName" />
          </Form.Item>

          <Form.Item
            name="description"
            label="Description"
            rules={[{ max: 500, message: 'Description must not exceed 500 characters' }]}
          >
            <TextArea rows={3} placeholder="Please enter the approval chain description" />
          </Form.Item>

          <Form.Item
            name="entityType"
            label="Applicable objects"
            initialValue="ticket"
            rules={[{ required: true, message: 'Please select applicable objects' }]}
          >
            <Select placeholder="Please select applicable objects" options={Object.entries(ENTITY_TYPE_LABELS).map(([value, label]) => ({ value, label }))} />
          </Form.Item>

          <Form.Item name="isActive" label="Status" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="Enabled" unCheckedChildren="Disabled" />
          </Form.Item>
        </div>
      </Form>

      <div style={{ display: currentStep === 1 ? 'block' : 'none' }}>{renderStepConfig()}</div>

      <div className="flex justify-between mt-6">
        <Button onClick={onCancel}>Cancel</Button>

        <Space>
          {currentStep > 0 && (
            <Button onClick={() => setCurrentStep(currentStep - 1)}>Previous</Button>
          )}

          {currentStep < stepItems.length - 1 ? (
            <Button type="primary" onClick={handleNextStep}>
              Next
            </Button>
          ) : (
            <Button type="primary" onClick={handleSubmit} loading={loading}>
              {editingChain ? 'Update' : 'Create'}
            </Button>
          )}
        </Space>
      </div>
    </Modal>
  );
}
