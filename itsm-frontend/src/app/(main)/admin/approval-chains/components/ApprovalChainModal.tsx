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
        message.error('获取用户列表失败');
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
      message.error('获取角色列表失败');
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
        message.error('请至少添加一个审批步骤');
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
          `步骤 ${invalidIndex + 1}（${step.stepName}）未选择${
            step.approverType === 'user' ? '审批用户' : '审批角色'
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
      // 表单验证失败
    }
  }, [form, steps, onSubmit, resetForm, message]);

  // 添加步骤
  const handleAddStep = useCallback(() => {
    const newStep: Omit<ApprovalStep, 'id' | 'chainId' | 'createdAt' | 'updatedAt'> = {
      level: steps.length + 1,
      stepOrder: steps.length + 1,
      stepName: `步骤 ${steps.length + 1}`,
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
          <Title level={5}>审批步骤配置</Title>
          <Button type="dashed" icon={<Plus className="w-4 h-4" />} onClick={handleAddStep}>
            添加步骤
          </Button>
        </div>

        {steps.map((step, index) => (
          <Card key={index} size="small" className="relative">
            <div className="flex justify-between items-start mb-4">
              <div className="flex items-center gap-2">
                <span className="font-medium">步骤 {index + 1}</span>
                <Input
                  value={step.stepName}
                  onChange={e => handleUpdateStep(index, 'stepName', e.target.value)}
                  placeholder="步骤名称"
                  style={{ width: 200 }}
                />
              </div>
              <Button
                type="text"
                aria-label={`删除第 ${index + 1} 个审批步骤`}
                danger
                icon={<MinusCircle className="w-4 h-4" />}
                onClick={() => handleRemoveStep(index)}
                size="small"
              />
            </div>

            <Row gutter={16}>
              <Col span={8}>
                <div className="mb-2">
                  <Text strong>审批层级</Text>
                </div>
                <InputNumber
                  aria-label={`步骤 ${index + 1} 审批层级`}
                  value={step.level ?? index + 1}
                  onChange={value => handleUpdateStep(index, 'level', value ?? index + 1)}
                  min={1}
                  max={20}
                  style={{ width: '100%' }}
                />
                <Text type="secondary" className="text-xs mt-1 block">
                  同一层级的多个步骤按「审批方式」合并
                </Text>
              </Col>

              <Col span={8}>
                <div className="mb-2">
                  <Text strong>审批人类型</Text>
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
                    { value: 'user', label: '用户' },
                    { value: 'role', label: '角色' },
                  ]}
                />
              </Col>

              <Col span={8}>
                <div className="mb-2">
                  <Text strong>审批人</Text>
                </div>
                {step.approverType === 'user' ? (
                  <Select
                    showSearch
                    filterOption={false}
                    onSearch={fetchUsers}
                    value={step.approverId || undefined}
                    placeholder="搜索并选择审批用户"
                    style={{ width: '100%' }}
                    loading={usersLoading}
                    notFoundContent={usersLoading ? <Spin size="small" /> : '未找到用户'}
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
                    placeholder="选择审批角色"
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
                  <Text strong>审批方式</Text>
                </div>
                <Select
                  aria-label={`步骤 ${index + 1} 审批方式`}
                  value={step.approvalType || 'serial'}
                  onChange={value => handleUpdateStep(index, 'approvalType', value)}
                  style={{ width: '100%' }}
                  options={approvalTypeOptions}
                />
              </Col>

              <Col span={6}>
                <div className="mb-2">
                  <Text strong>通过阈值</Text>
                </div>
                <InputNumber
                  aria-label={`步骤 ${index + 1} 通过阈值`}
                  value={step.threshold}
                  onChange={value => handleUpdateStep(index, 'threshold', value ?? undefined)}
                  min={1}
                  max={50}
                  placeholder="默认按本层人数"
                  disabled={!QUORUM_APPROVAL_TYPES.includes(step.approvalType || 'serial')}
                  style={{ width: '100%' }}
                />
              </Col>

              <Col span={6}>
                <div className="mb-2">
                  <Text strong>是否必审</Text>
                </div>
                <Switch
                  aria-label="该步骤是否为必审步骤"
                  checked={step.isRequired}
                  onChange={checked => handleUpdateStep(index, 'isRequired', checked)}
                  checkedChildren="必审"
                  unCheckedChildren="可选"
                />
              </Col>
            </Row>

            <Collapse
              size="small"
              className="mt-3"
              items={[
                {
                  key: 'advanced',
                  label: '高级配置（兜底策略 / 生效条件）',
                  children: (
                    <div className="space-y-3">
                      <Row gutter={16}>
                        <Col span={8}>
                          <div className="mb-2">
                            <Text strong>无人可审时的兜底</Text>
                          </div>
                          <Select
                            aria-label={`步骤 ${index + 1} 兜底策略`}
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
                                <Text strong>兜底审批人</Text>
                              </div>
                              <Select
                                showSearch
                                filterOption={false}
                                onSearch={fetchUsers}
                                value={step.fallbackApproverId || undefined}
                                placeholder="搜索并选择兜底审批用户"
                                style={{ width: '100%' }}
                                loading={usersLoading}
                                notFoundContent={usersLoading ? <Spin size="small" /> : '未找到用户'}
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
                                <Text strong>兜底审批角色</Text>
                              </div>
                              <Select
                                showSearch
                                optionFilterProp="label"
                                value={step.fallbackRole || undefined}
                                placeholder="选择兜底审批角色"
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
                            <Text strong>生效优先级</Text>
                          </div>
                          <Select
                            mode="multiple"
                            aria-label={`步骤 ${index + 1} 生效优先级`}
                            value={step.conditionPriorities}
                            placeholder="留空表示不按优先级过滤"
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
                            <Text strong>金额下限</Text>
                          </div>
                          <InputNumber
                            aria-label={`步骤 ${index + 1} 金额下限`}
                            value={step.conditionAmountMin}
                            onChange={value =>
                              handleUpdateStep(index, 'conditionAmountMin', value ?? undefined)
                            }
                            min={0}
                            placeholder="不限"
                            style={{ width: '100%' }}
                          />
                        </Col>

                        <Col span={6}>
                          <div className="mb-2">
                            <Text strong>金额上限</Text>
                          </div>
                          <InputNumber
                            aria-label={`步骤 ${index + 1} 金额上限`}
                            value={step.conditionAmountMax}
                            onChange={value =>
                              handleUpdateStep(index, 'conditionAmountMax', value ?? undefined)
                            }
                            min={0}
                            placeholder="不限"
                            style={{ width: '100%' }}
                          />
                        </Col>
                      </Row>

                      <Text type="secondary" className="text-xs block">
                        留空的兜底审批人/角色表示按角色解析；金额与优先级条件命中才适用本步骤。
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
            <Text>暂无审批步骤，请添加步骤</Text>
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
        <div style={{ display: currentStep === 0 ? 'block' : 'none' }}>
          <Form.Item
            name="name"
            label="审批链名称"
            rules={[
              { required: true, message: '请输入审批链名称' },
              { max: 100, message: '名称不能超过100个字符' },
            ]}
          >
            <Input placeholder="请输入审批链名称" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
            rules={[{ max: 500, message: '描述不能超过500个字符' }]}
          >
            <TextArea rows={3} placeholder="请输入审批链描述" />
          </Form.Item>

          <Form.Item
            name="entityType"
            label="适用对象"
            initialValue="ticket"
            rules={[{ required: true, message: '请选择适用对象' }]}
          >
            <Select placeholder="请选择适用对象" options={Object.entries(ENTITY_TYPE_LABELS).map(([value, label]) => ({ value, label }))} />
          </Form.Item>

          <Form.Item name="isActive" label="状态" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </div>
      </Form>

      <div style={{ display: currentStep === 1 ? 'block' : 'none' }}>{renderStepConfig()}</div>

      <div className="flex justify-between mt-6">
        <Button onClick={onCancel}>取消</Button>

        <Space>
          {currentStep > 0 && (
            <Button onClick={() => setCurrentStep(currentStep - 1)}>上一步</Button>
          )}

          {currentStep < stepItems.length - 1 ? (
            <Button type="primary" onClick={() => setCurrentStep(currentStep + 1)}>
              下一步
            </Button>
          ) : (
            <Button type="primary" onClick={handleSubmit} loading={loading}>
              {editingChain ? '更新' : '创建'}
            </Button>
          )}
        </Space>
      </div>
    </Modal>
  );
}
