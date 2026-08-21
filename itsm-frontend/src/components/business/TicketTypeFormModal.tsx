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
  App,
  InputNumber,
  Radio,
  Divider,
  Tag,
  Collapse,
} from 'antd';
import { ArrowUp, ArrowDown, Plus, Trash2, Info } from 'lucide-react';
import type {
  TicketTypeDefinition,
  CustomFieldDefinition,
  ApprovalChainDefinition,
  AssignmentRule,
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

export const TicketTypeFormModal: React.FC<TicketTypeFormModalProps> = ({
  visible,
  editingType,
  onCancel,
  onSubmit,
}) => {
  const { t } = useI18n();
  const { message } = App.useApp();
  const [form] = Form.useForm();
	const approvalEnabled = Form.useWatch('approvalEnabled', form);
	const slaEnabled = Form.useWatch('slaEnabled', form);
	const autoAssignEnabled = Form.useWatch('autoAssignEnabled', form);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('basic');
  const [customFields, setCustomFields] = useState<CustomFieldDefinition[]>([]);
  const [approvalChain, setApprovalChain] = useState<ApprovalChainDefinition[]>([]);
  const [assignmentRules, setAssignmentRules] = useState<AssignmentRule[]>([]);
  const [slas, setSlas] = useState<any[]>([]);
  const [users, setUsers] = useState<any[]>([]);
	const [categories, setCategories] = useState<any[]>([]);
	const [workflows, setWorkflows] = useState<any[]>([]);
	const [assignmentRuleOptions, setAssignmentRuleOptions] = useState<any[]>([]);

  useEffect(() => {
    if (visible) {
      void loadDependencies();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible]);

  const loadDependencies = async () => {
    try {
      const { SLAApi } = await import('@/lib/api/sla-api');
      const { UserApi } = await import('@/lib/api/user-api');

	  const { httpClient } = await import('@/lib/api/http-client');
      const [slaResponse, userResponse, categoryResponse, workflowResponse, ruleResponse] = await Promise.all([
        SLAApi.getSLADefinitions(),
        UserApi.getUsers({ page: 1, pageSize: 100, status: 'active' }),
		httpClient.get<any>('/api/v1/ticket-categories', { page: 1, pageSize: 200, isActive: true }),
		httpClient.get<any>('/api/v1/bpmn/process-definitions', { page: 1, pageSize: 200, isActive: true }),
		httpClient.get<any>('/api/v1/tickets/assignment-rules'),
      ]);

      setSlas(slaResponse.items);
      setUsers(userResponse.users);
	  setCategories(categoryResponse.categories ?? categoryResponse.items ?? []);
	  setWorkflows(workflowResponse.definitions ?? workflowResponse.items ?? []);
	  setAssignmentRuleOptions(ruleResponse.rules ?? ruleResponse.items ?? []);
    } catch (error) {
      console.error('Failed to load dependencies:', error);
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
		  categoryId: editingType.categoryId,
		  defaultPriority: editingType.defaultPriority,
		  sortOrder: editingType.sortOrder,
		  workflowDefinitionKey: editingType.workflowDefinitionKey,
		  assignmentRuleId: editingType.assignmentRuleId,
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
      newFields[index].order = index;
      newFields[targetIndex].order = targetIndex;
      setCustomFields(newFields);
    }
  };

  const addApprovalLevel = () => {
    const newLevel: ApprovalChainDefinition = {
      id: `level_${Date.now()}`,
      level: approvalChain.length + 1,
      name: `${t('ticketTypeForm.approvalLevelPrefix')} ${approvalChain.length + 1}`,
      approvers: [],
      approvalType: 'any',
      allowReject: true,
      allowDelegate: true,
      rejectAction: 'return',
    };
    setApprovalChain([...approvalChain, newLevel]);
  };

  const removeApprovalLevel = (index: number) => {
    setApprovalChain(approvalChain.filter((_, i) => i !== index));
  };

  const updateApprovalLevel = (index: number, level: Partial<ApprovalChainDefinition>) => {
    const newChain = [...approvalChain];
    newChain[index] = { ...newChain[index], ...level };
    setApprovalChain(newChain);
  };

  return (
    <Modal
      title={editingType ? t('ticketTypeForm.editTitle') : t('ticketTypeForm.createTitle')}
      open={visible}
      onCancel={onCancel}
      width={1000}
      footer={[
        <Button key="cancel" onClick={onCancel}>
          {t('common.cancel')}
        </Button>,
        <Button key="submit" type="primary" loading={loading} onClick={handleSubmit}>
          {editingType ? t('common.update') : t('common.create')}
        </Button>,
      ]}
    >
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'basic',
            label: t('ticketTypeForm.tabBasic'),
            children: (
              <Form form={form} layout="vertical">
                <Form.Item
                  label={t('ticketTypeForm.code')}
                  name="code"
                  rules={[
                    { required: true, message: t('ticketTypeForm.codeRequired') },
                    { pattern: /^[a-z_]+$/, message: t('ticketTypeForm.codePattern') },
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
				  <Form.Item label="分类" name="categoryId"><Select allowClear options={categories.map(item => ({ value: item.id, label: item.name }))} /></Form.Item>
				  <Form.Item label="默认优先级" name="defaultPriority" initialValue="medium"><Select options={['low', 'medium', 'high', 'urgent', 'critical'].map(value => ({ value, label: value }))} /></Form.Item>
				  <Form.Item label="排序" name="sortOrder" initialValue={0}><InputNumber min={0} className="w-full" /></Form.Item>
                  <Form.Item label={t('ticketTypeForm.icon')} name="icon">
                    <Select
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
              </Form>
            ),
          },
          {
            key: 'fields',
            label: t('ticketTypeForm.tabFields'),
            children: (
              <div className="space-y-4">
                <div className="flex justify-between items-center mb-4">
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
                            <Button
                              type="text"
                              size="small"
                              danger
                              icon={<Trash2 />}
                              onClick={() => removeCustomField(index)}
                            />
                          </Space>
                        }
                      >
                        <div className="grid grid-cols-2 gap-3">
                          <Input
                            placeholder={t('ticketTypeForm.fieldNameEnPlaceholder')}
                            value={field.name}
                            onChange={e =>
                              updateCustomField(index, { name: e.target.value })
                            }
                          />
						  {['select', 'multi_select', 'radio'].includes(field.type) && <Input className="col-span-2" placeholder="选项，使用英文逗号分隔" value={field.options?.map(option => option.label).join(',') ?? ''} onChange={e => updateCustomField(index, { options: e.target.value.split(',').map(value => value.trim()).filter(Boolean).map(value => ({ label: value, value })) })} />}
						  <Input className="col-span-2" placeholder="占位提示" value={field.placeholder} onChange={e => updateCustomField(index, { placeholder: e.target.value })} />
						  <Input className="col-span-2" placeholder="默认值（可选）" value={typeof field.defaultValue === 'string' || typeof field.defaultValue === 'number' ? String(field.defaultValue) : ''} onChange={e => updateCustomField(index, { defaultValue: e.target.value })} />
						  <InputNumber className="w-full" placeholder="最小长度/值" value={field.validation?.min} onChange={value => updateCustomField(index, { validation: { ...field.validation, min: value ?? undefined } })} />
						  <InputNumber className="w-full" placeholder="最大长度/值" value={field.validation?.max} onChange={value => updateCustomField(index, { validation: { ...field.validation, max: value ?? undefined } })} />
						  <Input className="col-span-2" placeholder="正则校验表达式（可选）" value={field.validation?.pattern} onChange={e => updateCustomField(index, { validation: { ...field.validation, pattern: e.target.value } })} />
						  <div><Switch checked={field.visible !== false} onChange={visible => updateCustomField(index, { visible })} /> <span className="ml-2">可见</span></div>
						  <div><Switch checked={field.readonly === true} onChange={readonly => updateCustomField(index, { readonly })} /> <span className="ml-2">只读</span></div>
                          <Input
                            placeholder={t('ticketTypeForm.fieldLabelPlaceholder')}
                            value={field.label}
                            onChange={e =>
                              updateCustomField(index, { label: e.target.value })
                            }
                          />
                          <Select
                            placeholder={t('ticketTypeForm.fieldType')}
                            value={field.type}
                            onChange={type => updateCustomField(index, { type })}
                            options={[
                              { value: CustomFieldType.TEXT, label: t('ticketTypeForm.typeText') },
                              {
                                value: CustomFieldType.TEXTAREA,
                                label: t('ticketTypeForm.typeTextarea'),
                              },
                              { value: CustomFieldType.NUMBER, label: t('ticketTypeForm.typeNumber') },
                              { value: CustomFieldType.DATE, label: t('ticketTypeForm.typeDate') },
                              {
                                value: CustomFieldType.DATETIME,
                                label: t('ticketTypeForm.typeDatetime'),
                              },
                              {
                                value: CustomFieldType.SELECT,
                                label: t('ticketTypeForm.typeSelect'),
                              },
                              {
                                value: CustomFieldType.MULTI_SELECT,
                                label: t('ticketTypeForm.typeMultiSelect'),
                              },
                              {
                                value: CustomFieldType.CHECKBOX,
                                label: t('ticketTypeForm.typeCheckbox'),
                              },
                              {
                                value: CustomFieldType.RADIO,
                                label: t('ticketTypeForm.typeRadio'),
                              },
                              {
                                value: CustomFieldType.USER_PICKER,
                                label: t('ticketTypeForm.typeUserPicker'),
                              },
							  { value: CustomFieldType.BOOLEAN, label: '布尔值' },
							  { value: CustomFieldType.DEPARTMENT, label: '部门' },
							  { value: CustomFieldType.CI, label: '配置项（CI）' },
                            ]}
                          />
                          <div className="flex items-center">
                            <Switch
                              checked={field.required}
                              onChange={required => updateCustomField(index, { required })}
                            />
                            <span className="ml-2">{t('ticketTypeForm.required')}</span>
                          </div>
                          <Input
                            className="col-span-2"
                            placeholder={t('ticketTypeForm.fieldDescriptionPlaceholder')}
                            value={field.description}
                            onChange={e =>
                              updateCustomField(index, { description: e.target.value })
                            }
                          />
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
            label: t('ticketTypeForm.tabApproval'),
            children: (
              <>
				<Form.Item label="绑定 BPMN Workflow" name="workflowDefinitionKey"><Select allowClear showSearch optionFilterProp="label" options={workflows.map(item => ({ value: item.key, label: `${item.name} (${item.key})` }))} /></Form.Item>
                <Form.Item
                  label={t('ticketTypeForm.enableApproval')}
                  name="approvalEnabled"
                  valuePropName="checked"
                >
                  <Switch />
                </Form.Item>

				{approvalEnabled && (
                  <div className="space-y-4 mt-4">
                    <div className="flex justify-between items-center">
                      <span className="text-gray-600">
                        {t('ticketTypeForm.approvalLevelsCount', {
                          count: approvalChain.length,
                        })}
                      </span>
                      <Button type="dashed" icon={<Plus />} onClick={addApprovalLevel}>
                        {t('ticketTypeForm.addApprovalLevel')}
                      </Button>
                    </div>

                    {approvalChain.length === 0 ? (
                      <div className="text-center py-8 text-gray-400">
                        <Info className="text-4xl mb-2" />
                        <div>{t('ticketTypeForm.noApprovalLevels')}</div>
                      </div>
                    ) : (
                      <div className="space-y-3">
                        {approvalChain.map((level, index) => (
                          <Card
                            key={level.id}
                            title={t('detailTabs.levelLabel', { level: level.level })}
                            size="small"
                            extra={
                              <Button
                                type="text"
                                size="small"
                                danger
                                icon={<Trash2 />}
                                onClick={() => removeApprovalLevel(index)}
                              />
                            }
                          >
                            <div className="space-y-3">
                              <Input
                                placeholder={t('ticketTypeForm.approvalLevelName')}
                                value={level.name}
                                onChange={e =>
                                  updateApprovalLevel(index, { name: e.target.value })
                                }
                              />

                              <div>
                                <div className="text-sm text-gray-600 mb-2">
                                  {t('ticketTypeForm.approvalMode')}
                                </div>
                                <Radio.Group
                                  value={level.approvalType}
                                  onChange={e =>
                                    updateApprovalLevel(index, {
                                      approvalType: e.target.value,
                                    })
                                  }
                                >
                                  <Radio value="any">{t('ticketTypeForm.approvalAny')}</Radio>
                                  <Radio value="all">{t('ticketTypeForm.approvalAll')}</Radio>
                                  <Radio value="majority">
                                    {t('ticketTypeForm.approvalMajority')}
                                  </Radio>
                                </Radio.Group>
                              </div>

                              <div>
                                <div className="text-sm text-gray-600 mb-2">
                                  {t('ticketTypeForm.approvers')}
                                </div>
                                <Select
                                  mode="multiple"
                                  placeholder={t('ticketTypeForm.selectApprovers')}
                                  style={{ width: '100%' }}
                                  value={level.approvers.map(a => a.value)}
                                  onChange={values => {
                                    const approvers = values.map(v => {
                                      const user = users.find(u => u.id === v);
                                      return {
                                        type: 'user' as const,
                                        value: v as number,
                                        name: user ? user.name || user.username : `${t('ticketTypeForm.userPrefix')} ${v}`,
                                      };
                                    });
                                    updateApprovalLevel(index, { approvers });
                                  }}
                                  options={users.map(user => ({
                                    value: user.id,
                                    label: user.name || user.username,
                                  }))}
                                />
                              </div>

                              <div className="grid grid-cols-2 gap-3">
                                <div className="flex items-center">
                                  <Switch
                                    checked={level.allowReject}
                                    onChange={allowReject =>
                                      updateApprovalLevel(index, { allowReject })
                                    }
                                  />
                                  <span className="ml-2">
                                    {t('ticketTypeForm.allowReject')}
                                  </span>
                                </div>
                                <div className="flex items-center">
                                  <Switch
                                    checked={level.allowDelegate}
                                    onChange={allowDelegate =>
                                      updateApprovalLevel(index, { allowDelegate })
                                    }
                                  />
                                  <span className="ml-2">
                                    {t('ticketTypeForm.allowDelegate')}
                                  </span>
                                </div>
                              </div>

                              {level.allowReject && (
                                <div>
                                  <div className="text-sm text-gray-600 mb-2">
                                    {t('ticketTypeForm.rejectAction')}
                                  </div>
                                  <Radio.Group
                                    value={level.rejectAction}
                                    onChange={e =>
                                      updateApprovalLevel(index, {
                                        rejectAction: e.target.value,
                                      })
                                    }
                                  >
                                    <Radio value="end">{t('ticketTypeForm.rejectEnd')}</Radio>
                                    <Radio value="return">
                                      {t('ticketTypeForm.rejectReturn')}
                                    </Radio>
                                    <Radio value="custom">
                                      {t('ticketTypeForm.rejectCustom')}
                                    </Radio>
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
                  <Form.Item label={t('ticketTypeForm.defaultSla')} name="defaultSlaId">
                    <Select
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
                  <div className="mt-4">
                    <div className="text-sm text-gray-600 mb-2">
                      <Info className="mr-1" />
                      {t('ticketTypeForm.autoAssignHint')}
                    </div>
					<Form.Item label="绑定分配规则" name="assignmentRuleId"><Select allowClear options={assignmentRuleOptions.map(rule => ({ value: rule.id, label: rule.name }))} /></Form.Item>
                  </div>
                )}
              </>
            ),
          },
        ]}
      />
    </Modal>
  );
};
