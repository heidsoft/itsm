'use client';

import React, { useState, useEffect } from 'react';
import {
  Modal,
  Form,
  Input,
  Select,
  Switch,
  Button,
  Space,
  Tabs,
  Card,
  message,
  InputNumber,
  Radio,
  Divider,
  Tag,
  Collapse,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import {
  TicketTypeDefinition,
  CustomFieldDefinition,
  CustomFieldType,
  ApprovalChainDefinition,
  AssignmentRule,
} from '@/types/ticket-type';

const { TextArea } = Input;
const { Option } = Select;
const { Panel } = Collapse;

interface TicketTypeFormModalProps {
  visible: boolean;
  editingType: TicketTypeDefinition | null;
  onCancel: () => void;
  onSubmit: (values: any) => Promise<void>;
}

export const TicketTypeFormModal: React.FC<TicketTypeFormModalProps> = ({
  visible,
  editingType,
  onCancel,
  onSubmit,
}) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('basic');
  const [customFields, setCustomFields] = useState<CustomFieldDefinition[]>([]);
  const [approvalChain, setApprovalChain] = useState<ApprovalChainDefinition[]>([]);
  const [assignmentRules, setAssignmentRules] = useState<AssignmentRule[]>([]);
  const [slas, setSlas] = useState<any[]>([]);
  const [users, setUsers] = useState<any[]>([]);

  useEffect(() => {
    if (visible) {
      loadDependencies();
      // ... existing code ...
    }
  }, [visible]);

  const loadDependencies = async () => {
    try {
      const { SLAApi } = await import('@/lib/api/sla-api');
      const { UserApi } = await import('@/lib/api/user-api');

      const [slaResponse, userResponse] = await Promise.all([
        SLAApi.getSLADefinitions(),
        UserApi.getUsers({ page: 1, page_size: 100, status: 'active' }),
      ]);

      setSlas(slaResponse.items);
      setUsers(userResponse.users);
    } catch (error) {
      console.error('Failed to load dependencies:', error);
      // Fail silently to avoid interrupting user flow
    }
  };

  useEffect(() => {
    if (visible) {
      if (editingType) {
        form.setFieldsValue({
          code: editingType.code,
          name: editingType.name,
          description: editingType.description,
          icon: editingType.icon,
          color: editingType.color,
          approvalEnabled: editingType.approvalEnabled,
          slaEnabled: editingType.slaEnabled,
          defaultSlaId: editingType.defaultSlaId,
          autoAssignEnabled: editingType.autoAssignEnabled,
        });
        setCustomFields(editingType.customFields || []);
        setApprovalChain(editingType.approvalChain || []);
        setAssignmentRules(editingType.assignmentRules || []);
      } else {
        form.resetFields();
        setCustomFields([]);
        setApprovalChain([]);
        setAssignmentRules([]);
      }
    }
  }, [visible, editingType, form]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      const submitData = {
        ...values,
        customFields,
        approvalChain,
        assignmentRules,
      };

      await onSubmit(submitData);
      form.resetFields();
      setCustomFields([]);
      setApprovalChain([]);
      setAssignmentRules([]);
    } catch (error) {
      console.error('Form validation failed:', error);
    } finally {
      setLoading(false);
    }
  };

  // 添加自定义字段
  const addCustomField = () => {
    const newField: CustomFieldDefinition = {
      id: `field_${Date.now()}`,
      name: '',
      label: '',
      type: CustomFieldType.TEXT,
      required: false,
      order: customFields.length,
    };
    setCustomFields([...customFields, newField]);
  };

  // 删除自定义字段
  const removeCustomField = (index: number) => {
    setCustomFields(customFields.filter((_, i) => i !== index));
  };

  // 更新自定义字段
  const updateCustomField = (index: number, field: Partial<CustomFieldDefinition>) => {
    const newFields = [...customFields];
    newFields[index] = { ...newFields[index], ...field };
    setCustomFields(newFields);
  };

  // 移动字段顺序
  const moveField = (index: number, direction: 'up' | 'down') => {
    const newFields = [...customFields];
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    if (targetIndex >= 0 && targetIndex < newFields.length) {
      [newFields[index], newFields[targetIndex]] = [newFields[targetIndex], newFields[index]];
      newFields[index].order = index;
      newFields[targetIndex].order = targetIndex;
      setCustomFields(newFields);
    }
  };

  // 添加审批级别
  const addApprovalLevel = () => {
    const newLevel: ApprovalChainDefinition = {
      id: `level_${Date.now()}`,
      level: approvalChain.length + 1,
      name: `审批级别 ${approvalChain.length + 1}`,
      approvers: [],
      approvalType: 'any',
      allowReject: true,
      allowDelegate: true,
      rejectAction: 'return',
    };
    setApprovalChain([...approvalChain, newLevel]);
  };

  // 删除审批级别
  const removeApprovalLevel = (index: number) => {
    setApprovalChain(approvalChain.filter((_, i) => i !== index));
  };

  // 更新审批级别
  const updateApprovalLevel = (index: number, level: Partial<ApprovalChainDefinition>) => {
    const newChain = [...approvalChain];
    newChain[index] = { ...newChain[index], ...level };
    setApprovalChain(newChain);
  };

  return (
    <Modal
      title={editingType ? '编辑工单类型' : '创建工单类型'}
      open={visible}
      onCancel={onCancel}
      width={1000}
      footer={[
        <Button key="cancel" onClick={onCancel}>
          取消
        </Button>,
        <Button key="submit" type="primary" loading={loading} onClick={handleSubmit}>
          {editingType ? '更新' : '创建'}
        </Button>,
      ]}
    >
      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        {/* 基本信息 */}
        <Tabs.TabPane tab="基本信息" key="basic">
          <Form form={form} layout="vertical">
            <Form.Item
              label="类型编码"
              name="code"
              rules={[
                { required: true, message: '请输入类型编码' },
                { pattern: /^[a-z_]+$/, message: '只能包含小写字母和下划线' },
              ]}
              tooltip="唯一标识，创建后不可修改"
            >
              <Input placeholder="例如: incident, request, change" disabled={!!editingType} />
            </Form.Item>

            <Form.Item
              label="类型名称"
              name="name"
              rules={[{ required: true, message: '请输入类型名称' }]}
            >
              <Input placeholder="例如: 故障工单, 服务请求" />
            </Form.Item>

            <Form.Item label="描述" name="description">
              <TextArea rows={3} placeholder="描述此工单类型的用途" />
            </Form.Item>

            <div className="grid grid-cols-2 gap-4">
              <Form.Item label="图标" name="icon">
                <Select placeholder="选择图标">
                  <Option value="BugOutlined">🐛 故障</Option>
                  <Option value="CustomerServiceOutlined">🎧 服务</Option>
                  <Option value="ToolOutlined">🔧 维护</Option>
                  <Option value="QuestionCircleOutlined">❓ 问题</Option>
                  <Option value="ThunderboltOutlined">⚡ 紧急</Option>
                </Select>
              </Form.Item>

              <Form.Item label="颜色" name="color">
                <Select placeholder="选择颜色">
                  <Option value="#ff4d4f">🔴 红色</Option>
                  <Option value="#1890ff">🔵 蓝色</Option>
                  <Option value="#52c41a">🟢 绿色</Option>
                  <Option value="#faad14">🟡 黄色</Option>
                  <Option value="#722ed1">🟣 紫色</Option>
                </Select>
              </Form.Item>
            </div>
          </Form>
        </Tabs.TabPane>

        {/* 自定义字段 */}
        <Tabs.TabPane tab="自定义字段" key="fields">
          <div className="space-y-4">
            <div className="flex justify-between items-center mb-4">
              <span className="text-gray-600">
                配置此工单类型的自定义字段（共 {customFields.length} 个）
              </span>
              <Button type="dashed" icon={<PlusOutlined />} onClick={addCustomField}>
                添加字段
              </Button>
            </div>

            {customFields.length === 0 ? (
              <div className="text-center py-8 text-gray-400">
                <InfoCircleOutlined className="text-4xl mb-2" />
                <div>暂无自定义字段，点击上方按钮添加</div>
              </div>
            ) : (
              <div className="space-y-3">
                {customFields.map((field, index) => (
                  <Card
                    key={field.id}
                    size="small"
                    extra={
                      <Space>
                        <Button
                          type="text"
                          size="small"
                          icon={<ArrowUpOutlined />}
                          disabled={index === 0}
                          onClick={() => moveField(index, 'up')}
                        />
                        <Button
                          type="text"
                          size="small"
                          icon={<ArrowDownOutlined />}
                          disabled={index === customFields.length - 1}
                          onClick={() => moveField(index, 'down')}
                        />
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={() => removeCustomField(index)}
                        />
                      </Space>
                    }
                  >
                    <div className="grid grid-cols-2 gap-3">
                      <Input
                        placeholder="字段名称（英文）"
                        value={field.name}
                        onChange={e => updateCustomField(index, { name: e.target.value })}
                      />
                      <Input
                        placeholder="字段标签（显示名称）"
                        value={field.label}
                        onChange={e => updateCustomField(index, { label: e.target.value })}
                      />
                      <Select
                        placeholder="字段类型"
                        value={field.type}
                        onChange={type => updateCustomField(index, { type })}
                      >
                        <Option value={CustomFieldType.TEXT}>单行文本</Option>
                        <Option value={CustomFieldType.TEXTAREA}>多行文本</Option>
                        <Option value={CustomFieldType.NUMBER}>数字</Option>
                        <Option value={CustomFieldType.DATE}>日期</Option>
                        <Option value={CustomFieldType.DATETIME}>日期时间</Option>
                        <Option value={CustomFieldType.SELECT}>下拉选择</Option>
                        <Option value={CustomFieldType.MULTI_SELECT}>多选</Option>
                        <Option value={CustomFieldType.CHECKBOX}>复选框</Option>
                        <Option value={CustomFieldType.RADIO}>单选按钮</Option>
                        <Option value={CustomFieldType.USER_PICKER}>用户选择器</Option>
                      </Select>
                      <div className="flex items-center">
                        <Switch
                          checked={field.required}
                          onChange={required => updateCustomField(index, { required })}
                        />
                        <span className="ml-2">必填</span>
                      </div>
                      <Input
                        className="col-span-2"
                        placeholder="字段描述（可选）"
                        value={field.description}
                        onChange={e => updateCustomField(index, { description: e.target.value })}
                      />
                    </div>
                  </Card>
                ))}
              </div>
            )}
          </div>
        </Tabs.TabPane>

        {/* 审批流程 */}
        <Tabs.TabPane tab="审批流程" key="approval">
          <Form.Item label="启用审批流程" name="approvalEnabled" valuePropName="checked">
            <Switch />
          </Form.Item>

          {form.getFieldValue('approvalEnabled') && (
            <div className="space-y-4 mt-4">
              <div className="flex justify-between items-center">
                <span className="text-gray-600">配置审批级别（共 {approvalChain.length} 级）</span>
                <Button type="dashed" icon={<PlusOutlined />} onClick={addApprovalLevel}>
                  添加审批级别
                </Button>
              </div>

              {approvalChain.length === 0 ? (
                <div className="text-center py-8 text-gray-400">
                  <InfoCircleOutlined className="text-4xl mb-2" />
                  <div>暂无审批级别，点击上方按钮添加</div>
                </div>
              ) : (
                <div className="space-y-3">
                  {approvalChain.map((level, index) => (
                    <Card
                      key={level.id}
                      title={`第 ${level.level} 级审批`}
                      size="small"
                      extra={
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={() => removeApprovalLevel(index)}
                        />
                      }
                    >
                      <div className="space-y-3">
                        <Input
                          placeholder="审批级别名称"
                          value={level.name}
                          onChange={e => updateApprovalLevel(index, { name: e.target.value })}
                        />

                        <div>
                          <div className="text-sm text-gray-600 mb-2">审批方式</div>
                          <Radio.Group
                            value={level.approvalType}
                            onChange={e =>
                              updateApprovalLevel(index, {
                                approvalType: e.target.value,
                              })
                            }
                          >
                            <Radio value="any">任一通过</Radio>
                            <Radio value="all">全部通过</Radio>
                            <Radio value="majority">多数通过</Radio>
                          </Radio.Group>
                        </div>

                        <div>
                          <div className="text-sm text-gray-600 mb-2">审批人</div>
                          <Select
                            mode="multiple"
                            placeholder="选择审批人"
                            style={{ width: '100%' }}
                            value={level.approvers.map(a => a.value)}
                            onChange={values => {
                              // 注意：审批人详细信息需要从用户API获取
                              const approvers = values.map(v => {
                                const user = users.find(u => u.id === v);
                                return {
                                  type: 'user' as const,
                                  value: v as number,
                                  name: user ? user.name || user.username : `用户 ${v}`,
                                };
                              });
                              updateApprovalLevel(index, { approvers });
                            }}
                          >
                            {users.map(user => (
                              <Option key={user.id} value={user.id}>
                                {user.name || user.username}
                              </Option>
                            ))}
                          </Select>
                        </div>

                        <div className="grid grid-cols-2 gap-3">
                          <div className="flex items-center">
                            <Switch
                              checked={level.allowReject}
                              onChange={allowReject => updateApprovalLevel(index, { allowReject })}
                            />
                            <span className="ml-2">允许驳回</span>
                          </div>
                          <div className="flex items-center">
                            <Switch
                              checked={level.allowDelegate}
                              onChange={allowDelegate =>
                                updateApprovalLevel(index, { allowDelegate })
                              }
                            />
                            <span className="ml-2">允许委派</span>
                          </div>
                        </div>

                        {level.allowReject && (
                          <div>
                            <div className="text-sm text-gray-600 mb-2">驳回后操作</div>
                            <Radio.Group
                              value={level.rejectAction}
                              onChange={e =>
                                updateApprovalLevel(index, {
                                  rejectAction: e.target.value,
                                })
                              }
                            >
                              <Radio value="end">结束流程</Radio>
                              <Radio value="return">返回上一级</Radio>
                              <Radio value="custom">自定义</Radio>
                            </Radio.Group>
                          </div>
                        )}
                      </div>
                    </Card>
                  ))}
                </div>
              )}
            </div>
          )}
        </Tabs.TabPane>

        {/* SLA配置 */}
        <Tabs.TabPane tab="SLA配置" key="sla">
          <Form.Item label="启用SLA" name="slaEnabled" valuePropName="checked">
            <Switch />
          </Form.Item>

          {form.getFieldValue('slaEnabled') && (
            <Form.Item label="默认SLA" name="defaultSlaId">
              <Select placeholder="选择默认SLA">
                {slas.map(sla => (
                  <Option key={sla.id} value={sla.id}>
                    {sla.name}
                  </Option>
                ))}
              </Select>
            </Form.Item>
          )}
        </Tabs.TabPane>

        {/* 自动分配 */}
        <Tabs.TabPane tab="自动分配" key="assignment">
          <Form.Item label="启用自动分配" name="autoAssignEnabled" valuePropName="checked">
            <Switch />
          </Form.Item>

          {form.getFieldValue('autoAssignEnabled') && (
            <div className="mt-4">
              <div className="text-sm text-gray-600 mb-2">
                <InfoCircleOutlined className="mr-1" />
                配置自动分配规则，系统将根据规则自动分配工单
              </div>
              <Button type="dashed" block icon={<PlusOutlined />}>
                添加分配规则
              </Button>
            </div>
          )}
        </Tabs.TabPane>
      </Tabs>
    </Modal>
  );
};
