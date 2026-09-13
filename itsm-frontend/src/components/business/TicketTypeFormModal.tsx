'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Alert,
  App,
  Badge,
  Button,
  Card,
  Collapse,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Switch,
  Tabs,
  Tag,
} from 'antd';
import { ArrowDown, ArrowUp, Info, Plus, Trash2 } from 'lucide-react';
import type {
  AssignmentRule,
  CustomFieldDefinition,
  TicketTypeDefinition,
} from '@/types/ticket-type';
import { CustomFieldType } from '@/types/ticket-type';
import { useI18n } from '@/lib/i18n/useI18n';

const { TextArea } = Input;

interface TicketTypeFormModalProps {
  visible: boolean;
  editingType: TicketTypeDefinition | null;
  onCancel: () => void;
  onSubmit: (values: unknown) => Promise<void>;
}

const OPTION_FIELD_TYPES: CustomFieldType[] = [
  CustomFieldType.SELECT,
  CustomFieldType.MULTI_SELECT,
  CustomFieldType.RADIO,
];

// 字段名是 schema 存储 key，仅接受 snake_case（与后端 Ent/JSONB 一致），避免与接口 JSON 驼峰命名冲突
const FIELD_KEY_PATTERN = /^[a-z][a-z0-9_]*$/;

// 将中文 label 转为粗略的 snake_case key，作为用户新建字段时的默认提示。
// 不保证独特性，仅在用户尚未填写 name 时提供可编辑起点。
const labelToSnakeCaseKey = (label: string): string => {
  if (!label) return '';
  const ascii = label.replace(/[^A-Za-z0-9]+/g, '_').replace(/_+/g, '_').replace(/^_|_$/g, '');
  return ascii ? ascii.toLowerCase() : '';
};

interface BoundApprovalNode {
  id: string;
  name: string;
  purpose: string;
  approvalMode: string;
  approverSource: string;
}

// 审批级次的权威来源是绑定流程里的 BPMN 审批节点，不是工单类型上的 JSON 字段。
// 这里解析已绑定流程定义的人工节点用于只读预览，避免用户再配一份运行时不生效的审批链。
const parseApprovalNodes = (bpmnXml?: string): BoundApprovalNode[] => {
  if (!bpmnXml) return [];
  const doc = new DOMParser().parseFromString(bpmnXml, 'application/xml');
  if (doc.getElementsByTagName('parsererror').length > 0) return [];

  // 属性按 localName 匹配，兼容 itsm:taskPurpose 与无前缀两种序列化形式
  const attr = (el: Element, localName: string): string => {
    for (let i = 0; i < el.attributes.length; i += 1) {
      const item = el.attributes[i];
      if ((item.localName || item.name) === localName) return item.value;
    }
    return '';
  };

  const nodes: BoundApprovalNode[] = [];
  Array.from(doc.getElementsByTagName('*')).forEach(el => {
    if ((el.localName || el.nodeName) !== 'userTask') return;
    const id = el.getAttribute('id') || '';
    nodes.push({
      id,
      name: el.getAttribute('name') || id,
      purpose: attr(el, 'taskPurpose'),
      approvalMode: attr(el, 'approvalMode'),
      approverSource:
        attr(el, 'assignee') || attr(el, 'candidateGroups') || attr(el, 'candidateUsers'),
    });
  });
  return nodes;
};

// BPMN 审批节点的 approvalMode 取值，用显式映射避免动态拼接 i18n key
const APPROVAL_MODE_LABEL_KEYS: Record<string, string> = {
  single: 'ticketTypeForm.modeSingle',
  any: 'ticketTypeForm.modeAny',
  all: 'ticketTypeForm.modeAll',
  sequential: 'ticketTypeForm.modeSequential',
};

const FieldLabel: React.FC<{ label: string; required?: boolean }> = ({ label, required }) => (
  <div className="mb-1 text-xs text-gray-500">
    {label}
    {required && <span className="ml-0.5 text-red-500">*</span>}
  </div>
);

export const TicketTypeFormModal: React.FC<TicketTypeFormModalProps> = ({
  visible,
  editingType,
  onCancel,
  onSubmit,
}) => {
  const { t } = useI18n();
  const { message } = App.useApp();
  const router = useRouter();
  const [form] = Form.useForm();
  const workflowDefinitionKey = Form.useWatch('workflowDefinitionKey', form);
  const slaEnabled = Form.useWatch('slaEnabled', form);
  const autoAssignEnabled = Form.useWatch('autoAssignEnabled', form);
  const [loading, setLoading] = useState(false);
  const [depsLoading, setDepsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('basic');
  const [customFields, setCustomFields] = useState<CustomFieldDefinition[]>([]);
  const [approvalNodes, setApprovalNodes] = useState<BoundApprovalNode[]>([]);
  const [approvalNodesLoading, setApprovalNodesLoading] = useState(false);
  const [approvalNodesError, setApprovalNodesError] = useState('');
  const [assignmentRules, setAssignmentRules] = useState<AssignmentRule[]>([]);
  const [slas, setSlas] = useState<any[]>([]);
  const [categories, setCategories] = useState<any[]>([]);
  const [workflows, setWorkflows] = useState<any[]>([]);
  const [assignmentRuleOptions, setAssignmentRuleOptions] = useState<any[]>([]);

  useEffect(() => {
    if (visible) {
      void loadDependencies();
    }
  }, [visible]);

  const loadDependencies = async () => {
    setDepsLoading(true);
    try {
      const { SLAApi } = await import('@/lib/api/sla-api');
      const { httpClient } = await import('@/lib/api/http-client');

      const [slaResponse, categoryResponse, workflowResponse, ruleResponse] = await Promise.all([
        SLAApi.getSLADefinitions(),
        httpClient.get<any>('/api/v1/ticket-categories', { page: 1, pageSize: 200, isActive: true }),
        httpClient.get<any>('/api/v1/bpmn/process-definitions', { page: 1, pageSize: 200, isActive: true }),
        httpClient.get<any>('/api/v1/tickets/assignment-rules'),
      ]);

      setSlas(slaResponse.items ?? []);
      setCategories(categoryResponse.items ?? categoryResponse.categories ?? []);
      setWorkflows(workflowResponse.definitions ?? workflowResponse.items ?? []);
      setAssignmentRuleOptions(ruleResponse.rules ?? ruleResponse.items ?? []);
    } catch (error) {
      console.error('Failed to load dependencies:', error);
      message.error(t('ticketTypeForm.depsLoadFailed'));
    } finally {
      setDepsLoading(false);
    }
  };

  useEffect(() => {
    if (!visible) return;
    setActiveTab('basic');
    if (editingType) {
      form.setFieldsValue({
        code: editingType.code,
        name: editingType.name,
        description: editingType.description,
        icon: editingType.icon,
        color: editingType.color,
        slaEnabled: editingType.slaEnabled,
        defaultSlaId: editingType.defaultSlaId,
        autoAssignEnabled: editingType.autoAssignEnabled,
        categoryId: editingType.categoryId,
        defaultPriority: editingType.defaultPriority,
        sortOrder: editingType.sortOrder,
        workflowDefinitionKey: editingType.workflowDefinitionKey,
        assignmentRuleId: editingType.assignmentRuleId,
      });
      setCustomFields(editingType.customFields || []);
      setAssignmentRules(editingType.assignmentRules || []);
    } else {
      form.resetFields();
      setCustomFields([]);
      setAssignmentRules([]);
    }
    setApprovalNodes([]);
    setApprovalNodesError('');
  }, [visible, editingType, form]);

  // 绑定的工作流变化时拉取流程定义，解析出审批节点做只读预览
  useEffect(() => {
    if (!visible) return;
    const key = typeof workflowDefinitionKey === 'string' ? workflowDefinitionKey.trim() : '';
    if (!key) {
      setApprovalNodes([]);
      setApprovalNodesError('');
      setApprovalNodesLoading(false);
      return;
    }

    let cancelled = false;
    setApprovalNodesLoading(true);
    setApprovalNodesError('');

    void (async () => {
      try {
        const { WorkflowApi } = await import('@/lib/api/workflow-api');
        const definition = await WorkflowApi.getProcessDefinition(key);
        if (cancelled) return;
        setApprovalNodes(parseApprovalNodes(definition.bpmnXml));
      } catch (error) {
        if (cancelled) return;
        console.error('Failed to load approval nodes of bound workflow:', error);
        setApprovalNodes([]);
        setApprovalNodesError(t('ticketTypeForm.approvalNodesLoadFailed'));
      } finally {
        if (!cancelled) setApprovalNodesLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [visible, workflowDefinitionKey, t]);

  const handleSubmit = async () => {
    let values;
    try {
      values = await form.validateFields();
    } catch {
      // 基础信息必填项未通过校验，跳回第一个页签定位错误
      setActiveTab('basic');
      return;
    }

    // 自定义字段完整性校验
    for (let i = 0; i < customFields.length; i++) {
      const field = customFields[i];
      if (!field.label?.trim()) {
        setActiveTab('fields');
        message.warning(t('ticketTypeForm.fieldLabelRequired', { index: i + 1 }));
        return;
      }
      if (!field.name?.trim()) {
        setActiveTab('fields');
        message.warning(t('ticketTypeForm.fieldNameRequired', { index: i + 1 }));
        return;
      }
      if (!FIELD_KEY_PATTERN.test(field.name)) {
        setActiveTab('fields');
        message.warning(t('ticketTypeForm.fieldKeyInvalid', { index: i + 1 }));
        return;
      }
      if (OPTION_FIELD_TYPES.includes(field.type) && !field.options?.length) {
        setActiveTab('fields');
        message.warning(t('ticketTypeForm.fieldOptionsRequired', { index: i + 1 }));
        return;
      }
    }

    // SLA 完整性校验
    if (values.slaEnabled && !values.defaultSlaId) {
      setActiveTab('sla');
      message.warning(t('ticketTypeForm.slaPolicyRequired'));
      return;
    }

    const submitData = {
      ...values,
      customFields: customFields.map((field, index) => ({ ...field, order: index })),
      assignmentRules,
    };

    setLoading(true);
    try {
      await onSubmit(submitData);
      form.resetFields();
      setCustomFields([]);
      setAssignmentRules([]);
    } catch (error) {
      // 错误提示由父组件展示，保持弹窗打开以便修改重试
      console.error('Submit ticket type failed:', error);
    } finally {
      setLoading(false);
    }
  };

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

  const removeCustomField = (index: number) => {
    setCustomFields(customFields.filter((_, i) => i !== index));
  };

  const updateCustomField = (index: number, field: Partial<CustomFieldDefinition>) => {
    const newFields = [...customFields];
    newFields[index] = { ...newFields[index], ...field };
    setCustomFields(newFields);
  };

  const moveField = (index: number, direction: 'up' | 'down') => {
    const newFields = [...customFields];
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    if (targetIndex >= 0 && targetIndex < newFields.length) {
      [newFields[index], newFields[targetIndex]] = [newFields[targetIndex], newFields[index]];
      setCustomFields(newFields.map((field, i) => ({ ...field, order: i })));
    }
  };

  // 设计器按数字主键打开，而工单类型绑定的是流程 key，需从已加载列表反查
  const boundWorkflow = workflows.find(item => item.key === workflowDefinitionKey);

  const priorityOptions = [
    { value: 'low', label: t('ticketTypeForm.priorityLow') },
    { value: 'medium', label: t('ticketTypeForm.priorityMedium') },
    { value: 'high', label: t('ticketTypeForm.priorityHigh') },
    { value: 'urgent', label: t('ticketTypeForm.priorityUrgent') },
    { value: 'critical', label: t('ticketTypeForm.priorityCritical') },
  ];

  const fieldTypeOptions = [
    { value: CustomFieldType.TEXT, label: t('ticketTypeForm.typeText') },
    { value: CustomFieldType.TEXTAREA, label: t('ticketTypeForm.typeTextarea') },
    { value: CustomFieldType.NUMBER, label: t('ticketTypeForm.typeNumber') },
    { value: CustomFieldType.DATE, label: t('ticketTypeForm.typeDate') },
    { value: CustomFieldType.DATETIME, label: t('ticketTypeForm.typeDatetime') },
    { value: CustomFieldType.SELECT, label: t('ticketTypeForm.typeSelect') },
    { value: CustomFieldType.MULTI_SELECT, label: t('ticketTypeForm.typeMultiSelect') },
    { value: CustomFieldType.CHECKBOX, label: t('ticketTypeForm.typeCheckbox') },
    { value: CustomFieldType.RADIO, label: t('ticketTypeForm.typeRadio') },
    { value: CustomFieldType.USER_PICKER, label: t('ticketTypeForm.typeUserPicker') },
    { value: CustomFieldType.BOOLEAN, label: t('ticketTypeForm.typeBoolean') },
    { value: CustomFieldType.DEPARTMENT, label: t('ticketTypeForm.typeDepartment') },
    { value: CustomFieldType.CI, label: t('ticketTypeForm.typeCi') },
  ];

  return (
    <Modal
      title={editingType ? t('ticketTypeForm.editTitle') : t('ticketTypeForm.createTitle')}
      open={visible}
      onCancel={onCancel}
      width={1000}
      mask={{ closable: false }}
      keyboard={false}
      styles={{ body: { maxHeight: '65vh', overflowY: 'auto', paddingRight: 8 } }}
      footer={[
        <Button key="cancel" onClick={onCancel}>
          {t('common.cancel')}
        </Button>,
        <Button key="submit" type="primary" loading={loading} onClick={handleSubmit}>
          {editingType ? t('common.update') : t('common.create')}
        </Button>,
      ]}
    >
      {/* Form 必须包裹全部页签，否则其余页签中的 Form.Item 无法注册到表单实例 */}
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          defaultPriority: 'medium',
          sortOrder: 0,
          slaEnabled: false,
          autoAssignEnabled: false,
        }}
      >
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'basic',
              label: t('ticketTypeForm.tabBasic'),
              children: (
                <>
                  <Form.Item
                    label={t('ticketTypeForm.code')}
                    name="code"
                    rules={[
                      { required: true, message: t('ticketTypeForm.codeRequired') },
                      { pattern: /^[a-z][a-z0-9_]*$/, message: t('ticketTypeForm.codePattern') },
                    ]}
                    tooltip={t('ticketTypeForm.codeTooltip')}
                  >
                    <Input
                      placeholder={t('ticketTypeForm.codePlaceholder')}
                      disabled={!!editingType}
                    />
                  </Form.Item>

                  <Form.Item
                    label={t('ticketTypeForm.name')}
                    name="name"
                    rules={[{ required: true, message: t('ticketTypeForm.nameRequired') }]}
                  >
                    <Input placeholder={t('ticketTypeForm.namePlaceholder')} />
                  </Form.Item>

                  <Form.Item label={t('ticketTypeForm.description')} name="description">
                    <TextArea rows={3} placeholder={t('ticketTypeForm.descriptionPlaceholder')} />
                  </Form.Item>

                  <div className="grid grid-cols-2 gap-4">
                    <Form.Item label={t('ticketTypeForm.category')} name="categoryId">
                      <Select
                        allowClear
                        loading={depsLoading}
                        placeholder={t('ticketTypeForm.categoryPlaceholder')}
                        options={categories.map(item => ({ value: item.id, label: item.name }))}
                      />
                    </Form.Item>
                    <Form.Item
                      label={t('ticketTypeForm.defaultPriority')}
                      name="defaultPriority"
                    >
                      <Select options={priorityOptions} />
                    </Form.Item>
                    <Form.Item label={t('ticketTypeForm.sortOrder')} name="sortOrder">
                      <InputNumber min={0} className="w-full" />
                    </Form.Item>
                    <Form.Item label={t('ticketTypeForm.icon')} name="icon">
                      <Select
                        allowClear
                        placeholder={t('ticketTypeForm.selectIcon')}
                        options={[
                          { value: 'Bug', label: t('ticketTypeForm.iconBug') },
                          { value: 'Headphones', label: t('ticketTypeForm.iconService') },
                          { value: 'Wrench', label: t('ticketTypeForm.iconMaintenance') },
                          { value: 'HelpCircle', label: t('ticketTypeForm.iconQuestion') },
                          { value: 'Zap', label: t('ticketTypeForm.iconUrgent') },
                        ]}
                      />
                    </Form.Item>
                    <Form.Item label={t('ticketTypeForm.color')} name="color">
                      <Select
                        allowClear
                        placeholder={t('ticketTypeForm.selectColor')}
                        options={[
                          { value: '#ff4d4f', label: t('ticketTypeForm.colorRed') },
                          { value: '#1890ff', label: t('ticketTypeForm.colorBlue') },
                          { value: '#52c41a', label: t('ticketTypeForm.colorGreen') },
                          { value: '#faad14', label: t('ticketTypeForm.colorYellow') },
                          { value: '#722ed1', label: t('ticketTypeForm.colorPurple') },
                        ]}
                      />
                    </Form.Item>
                  </div>
                </>
              ),
            },
            {
              key: 'fields',
              label: (
                <Badge count={customFields.length} size="small" offset={[10, -2]}>
                  {t('ticketTypeForm.tabFields')}
                </Badge>
              ),
              children: (
                <div className="space-y-4">
                  <div className="flex justify-between items-center">
                    <span className="text-gray-600">
                      {t('ticketTypeForm.fieldsCount', { count: customFields.length })}
                    </span>
                    <Button type="dashed" icon={<Plus />} onClick={addCustomField}>
                      {t('ticketTypeForm.addField')}
                    </Button>
                  </div>

                  {customFields.length === 0 ? (
                    <div className="text-center py-8 text-gray-400">
                      <Info className="text-4xl mb-2" />
                      <div>{t('ticketTypeForm.noFields')}</div>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {customFields.map((field, index) => (
                        <Card
                          key={field.id}
                          size="small"
                          title={
                            <Space size={4}>
                              <span className="text-gray-400">#{index + 1}</span>
                              <span>{field.label?.trim() || t('ticketTypeForm.unnamedField')}</span>
                              {field.required && (
                                <Tag color="red" className="!mr-0">
                                  {t('ticketTypeForm.required')}
                                </Tag>
                              )}
                              {!field.visible && (
                                <Tag className="!mr-0">{t('ticketTypeForm.invisibleTag')}</Tag>
                              )}
                            </Space>
                          }
                          extra={
                            <Space>
                              <Button
                                type="text"
                                size="small"
                                icon={<ArrowUp />}
                                disabled={index === 0}
                                onClick={() => moveField(index, 'up')}
                              />
                              <Button
                                type="text"
                                size="small"
                                icon={<ArrowDown />}
                                disabled={index === customFields.length - 1}
                                onClick={() => moveField(index, 'down')}
                              />
                              <Popconfirm
                                title={t('ticketTypeForm.deleteFieldConfirm')}
                                onConfirm={() => removeCustomField(index)}
                              >
                                <Button type="text" size="small" danger icon={<Trash2 />} />
                              </Popconfirm>
                            </Space>
                          }
                        >
                          <div className="grid grid-cols-2 gap-3">
                            <div>
                              <FieldLabel label={t('ticketTypeForm.fieldDisplayName')} required />
                              <Input
                                placeholder={t('ticketTypeForm.fieldLabelPlaceholder')}
                                value={field.label}
                                onChange={e => {
                                  const newLabel = e.target.value;
                                  // 仅在用户尚未手工填写过 field.name 时才提供 snake_case 默认值，
                                  // 避免后续输入覆盖用户选择的 key。
                                  const shouldAutoFill = !field.name || field.name === labelToSnakeCaseKey(field.label);
                                  updateCustomField(index, {
                                    label: newLabel,
                                    ...(shouldAutoFill ? { name: labelToSnakeCaseKey(newLabel) } : {}),
                                  });
                                }}
                              />
                            </div>
                            <div>
                              <FieldLabel label={t('ticketTypeForm.fieldType')} />
                              <Select
                                className="w-full"
                                value={field.type}
                                onChange={type => updateCustomField(index, { type })}
                                options={fieldTypeOptions}
                              />
                            </div>
                            <div>
                              <FieldLabel
                                label={t('ticketTypeForm.fieldKey')}
                                required
                              />
                              <Input
                                placeholder={t('ticketTypeForm.fieldKeyPlaceholder')}
                                value={field.name}
                                onChange={e =>
                                  updateCustomField(index, { name: e.target.value })
                                }
                              />
                              <div className="mt-1 text-xs text-gray-400">
                                {t('ticketTypeForm.fieldKeyTooltip')}
                              </div>
                            </div>
                            <div>
                              <FieldLabel label={t('ticketTypeForm.fieldPlaceholderLabel')} />
                              <Input
                                value={field.placeholder}
                                onChange={e =>
                                  updateCustomField(index, { placeholder: e.target.value })
                                }
                              />
                            </div>

                            {OPTION_FIELD_TYPES.includes(field.type) && (
                              <div className="col-span-2">
                                <FieldLabel
                                  label={t('ticketTypeForm.fieldOptionsLabel')}
                                  required
                                />
                                <Input
                                  placeholder={t('ticketTypeForm.fieldOptionsPlaceholder')}
                                  value={field.options?.map(option => option.label).join(',') ?? ''}
                                  onChange={e =>
                                    updateCustomField(index, {
                                      options: e.target.value
                                        .split(',')
                                        .map(value => value.trim())
                                        .filter(Boolean)
                                        .map(value => ({ label: value, value })),
                                    })
                                  }
                                />
                              </div>
                            )}

                            <div className="col-span-2">
                              <Collapse
                                ghost
                                size="small"
                                items={[
                                  {
                                    key: 'advanced',
                                    label: t('ticketTypeForm.advancedSettings'),
                                    children: (
                                      <div className="grid grid-cols-2 gap-3">
                                        <div>
                                          <FieldLabel label={t('ticketTypeForm.fieldDefaultValueLabel')} />
                                          <Input
                                            value={
                                              typeof field.defaultValue === 'string' ||
                                              typeof field.defaultValue === 'number'
                                                ? String(field.defaultValue)
                                                : ''
                                            }
                                            onChange={e =>
                                              updateCustomField(index, {
                                                defaultValue: e.target.value,
                                              })
                                            }
                                          />
                                        </div>
                                        <div>
                                          <FieldLabel label={t('ticketTypeForm.fieldDescriptionLabel')} />
                                          <Input
                                            placeholder={t('ticketTypeForm.fieldDescriptionPlaceholder')}
                                            value={field.description}
                                            onChange={e =>
                                              updateCustomField(index, {
                                                description: e.target.value,
                                              })
                                            }
                                          />
                                        </div>
                                        <div>
                                          <FieldLabel label={t('ticketTypeForm.validationMinLabel')} />
                                          <InputNumber
                                            className="w-full"
                                            value={field.validation?.min}
                                            onChange={value =>
                                              updateCustomField(index, {
                                                validation: { ...field.validation, min: value ?? undefined },
                                              })
                                            }
                                          />
                                        </div>
                                        <div>
                                          <FieldLabel label={t('ticketTypeForm.validationMaxLabel')} />
                                          <InputNumber
                                            className="w-full"
                                            value={field.validation?.max}
                                            onChange={value =>
                                              updateCustomField(index, {
                                                validation: { ...field.validation, max: value ?? undefined },
                                              })
                                            }
                                          />
                                        </div>
                                        <div className="col-span-2">
                                          <FieldLabel label={t('ticketTypeForm.validationPatternLabel')} />
                                          <Input
                                            value={field.validation?.pattern}
                                            onChange={e =>
                                              updateCustomField(index, {
                                                validation: { ...field.validation, pattern: e.target.value },
                                              })
                                            }
                                          />
                                        </div>
                                      </div>
                                    ),
                                  },
                                ]}
                              />
                            </div>
                          </div>

                          <div className="mt-2 flex items-center gap-6 border-t border-gray-100 pt-3">
                            <div className="flex items-center">
                              <Switch
                                size="small"
                                checked={field.required}
                                onChange={required => updateCustomField(index, { required })}
                              />
                              <span className="ml-2 text-sm text-gray-600">
                                {t('ticketTypeForm.required')}
                              </span>
                            </div>
                            <div className="flex items-center">
                              <Switch
                                size="small"
                                checked={field.visible !== false}
                                onChange={visible => updateCustomField(index, { visible })}
                              />
                              <span className="ml-2 text-sm text-gray-600">
                                {t('ticketTypeForm.visibleLabel')}
                              </span>
                            </div>
                            <div className="flex items-center">
                              <Switch
                                size="small"
                                checked={field.readonly === true}
                                onChange={readonly => updateCustomField(index, { readonly })}
                              />
                              <span className="ml-2 text-sm text-gray-600">
                                {t('ticketTypeForm.readonlyLabel')}
                              </span>
                            </div>
                          </div>
                        </Card>
                      ))}
                    </div>
                  )}
                </div>
              ),
            },
            {
              key: 'approval',
              label: (
                <Badge count={approvalNodes.length} size="small" offset={[10, -2]}>
                  {t('ticketTypeForm.tabApproval')}
                </Badge>
              ),
              children: (
                <>
                  <Alert
                    type="info"
                    showIcon
                    className="!mb-4"
                    message={t('ticketTypeForm.approvalSourceHint')}
                    description={t('ticketTypeForm.approvalSourceHintDetail')}
                  />

                  <Form.Item
                    label={t('ticketTypeForm.bindWorkflow')}
                    name="workflowDefinitionKey"
                    tooltip={t('ticketTypeForm.bindWorkflowTooltip')}
                  >
                    <Select
                      allowClear
                      showSearch
                      optionFilterProp="label"
                      loading={depsLoading}
                      placeholder={t('ticketTypeForm.workflowPlaceholder')}
                      options={workflows.map(item => ({
                        value: item.key,
                        label: `${item.name} (${item.key})`,
                      }))}
                    />
                  </Form.Item>

                  <div className="flex justify-between items-center mb-2">
                    <span className="text-gray-600">
                      {t('ticketTypeForm.approvalNodesCount', { count: approvalNodes.length })}
                    </span>
                    {boundWorkflow && (
                      <Button
                        type="link"
                        className="!px-0"
                        onClick={() => router.push(`/workflow/designer?id=${boundWorkflow.id}`)}
                      >
                        {t('ticketTypeForm.openDesigner')}
                      </Button>
                    )}
                  </div>

                  {approvalNodesError ? (
                    <Alert type="warning" showIcon message={approvalNodesError} />
                  ) : approvalNodesLoading ? (
                    <div className="text-center py-8">
                      <Spin />
                    </div>
                  ) : !workflowDefinitionKey ? (
                    <div className="text-center py-8 text-gray-400">
                      <Info className="text-4xl mb-2" />
                      <div>{t('ticketTypeForm.noWorkflowBound')}</div>
                    </div>
                  ) : approvalNodes.length === 0 ? (
                    <div className="text-center py-8 text-gray-400">
                      <Info className="text-4xl mb-2" />
                      <div>{t('ticketTypeForm.noApprovalNodes')}</div>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {approvalNodes.map((node, index) => (
                        <Card
                          key={node.id}
                          size="small"
                          title={
                            <Space size={4}>
                              <span className="text-gray-400">
                                {t('detailTabs.levelLabel', { level: index + 1 })}
                              </span>
                              <span>{node.name}</span>
                            </Space>
                          }
                          extra={
                            node.purpose === 'approval' ? (
                              <Tag color="blue">{t('ticketTypeForm.purposeApproval')}</Tag>
                            ) : (
                              <Tag>{t('ticketTypeForm.purposeManual')}</Tag>
                            )
                          }
                        >
                          <div className="text-sm text-gray-600 space-y-1">
                            <div>
                              <span className="text-gray-400">
                                {t('ticketTypeForm.approvalMode')}
                                {'：'}
                              </span>
                              {t(
                                APPROVAL_MODE_LABEL_KEYS[node.approvalMode] ??
                                  'ticketTypeForm.notConfigured',
                              )}
                            </div>
                            <div>
                              <span className="text-gray-400">
                                {t('ticketTypeForm.approverSource')}
                                {'：'}
                              </span>
                              {node.approverSource || t('ticketTypeForm.notConfigured')}
                            </div>
                          </div>
                        </Card>
                      ))}
                    </div>
                  )}
                </>
              ),
            },
            {
              key: 'sla',
              label: t('ticketTypeForm.tabSla'),
              children: (
                <>
                  <Form.Item
                    label={t('ticketTypeForm.enableSla')}
                    name="slaEnabled"
                    valuePropName="checked"
                  >
                    <Switch />
                  </Form.Item>

                  {slaEnabled && (
                    <Form.Item
                      label={t('ticketTypeForm.defaultSla')}
                      name="defaultSlaId"
                      required
                    >
                      <Select
                        loading={depsLoading}
                        placeholder={t('ticketTypeForm.selectDefaultSla')}
                        options={slas.map(sla => ({
                          value: sla.id,
                          label: sla.name,
                        }))}
                      />
                    </Form.Item>
                  )}
                </>
              ),
            },
            {
              key: 'assignment',
              label: t('ticketTypeForm.tabAssignment'),
              children: (
                <>
                  <Form.Item
                    label={t('ticketTypeForm.enableAutoAssign')}
                    name="autoAssignEnabled"
                    valuePropName="checked"
                  >
                    <Switch />
                  </Form.Item>

                  {autoAssignEnabled && (
                    <div className="mt-2">
                      <div className="text-sm text-gray-600 mb-3">
                        <Info className="mr-1 inline" />
                        {t('ticketTypeForm.autoAssignHint')}
                      </div>
                      <Form.Item
                        label={t('ticketTypeForm.assignmentRuleLabel')}
                        name="assignmentRuleId"
                      >
                        <Select
                          allowClear
                          loading={depsLoading}
                          placeholder={t('ticketTypeForm.assignmentRulePlaceholder')}
                          options={assignmentRuleOptions.map(rule => ({
                            value: rule.id,
                            label: rule.name,
                          }))}
                        />
                      </Form.Item>
                    </div>
                  )}
                </>
              ),
            },
          ]}
        />
      </Form>
    </Modal>
  );
};
