/**
 * 字段设计器组件
 * 提供拖拽式字段配置、实时预览、条件逻辑设置等功能
 */

'use client';

import React, { useState, useCallback, useMemo } from 'react';
import {
  Card,
  Button,
  Space,
  Form,
  Input,
  Select,
  Switch,
  InputNumber,
  Tooltip,
  Modal,
  Tabs,
  Row,
  Col,
  Divider,
  Alert,
  Tag,
  Collapse,
  Radio,
  Checkbox,
  App,
  Empty,
  Badge,
} from 'antd';
import { ArrowUp, ArrowDown, Plus, Pencil, Trash2, Copy, Eye, Settings, AlertTriangle, HelpCircle, CheckCircle, GripVertical } from 'lucide-react';
import type {
  DragEndEvent} from '@dnd-kit/core';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import type {
  TemplateField,
  FieldType,
  FieldValidation,
  FieldOption,
  FieldConditional,
} from '@/types/template';
import { useI18n } from '@/lib/i18n/useI18n';

const { TextArea } = Input;
const { Panel } = Collapse;

// ==================== 字段类型配置工厂函数 ====================

interface FieldTypeConfig {
  type: FieldType;
  label: string;
  icon: string;
  description: string;
  category: 'basic' | 'advanced' | 'special';
  defaultConfig: Partial<TemplateField>;
}

const createFieldTypes = (t: (key: string) => string): FieldTypeConfig[] => [
  // 基础类型
  {
    type: 'text' as FieldType,
    label: t('fieldDesigner.fieldTypes.text.label'),
    icon: '📝',
    description: t('fieldDesigner.fieldTypes.text.description'),
    category: 'basic',
    defaultConfig: {
      placeholder: t('fieldDesigner.fieldTypes.text.defaultPlaceholder'),
      validation: { maxLength: 200 },
    },
  },
  {
    type: 'textarea' as FieldType,
    label: t('fieldDesigner.fieldTypes.textarea.label'),
    icon: '📄',
    description: t('fieldDesigner.fieldTypes.textarea.description'),
    category: 'basic',
    defaultConfig: {
      placeholder: t('fieldDesigner.fieldTypes.textarea.defaultPlaceholder'),
      validation: { maxLength: 2000 },
    },
  },
  {
    type: 'number' as FieldType,
    label: t('fieldDesigner.fieldTypes.number.label'),
    icon: '🔢',
    description: t('fieldDesigner.fieldTypes.number.description'),
    category: 'basic',
    defaultConfig: {
      placeholder: t('fieldDesigner.fieldTypes.number.defaultPlaceholder'),
    },
  },
  {
    type: 'date' as FieldType,
    label: t('fieldDesigner.fieldTypes.date.label'),
    icon: '📅',
    description: t('fieldDesigner.fieldTypes.date.description'),
    category: 'basic',
    defaultConfig: {},
  },
  {
    type: 'datetime' as FieldType,
    label: t('fieldDesigner.fieldTypes.datetime.label'),
    icon: '🕐',
    description: t('fieldDesigner.fieldTypes.datetime.description'),
    category: 'basic',
    defaultConfig: {},
  },

  // 选择类型
  {
    type: 'select' as FieldType,
    label: t('fieldDesigner.fieldTypes.select.label'),
    icon: '📋',
    description: t('fieldDesigner.fieldTypes.select.description'),
    category: 'basic',
    defaultConfig: {
      options: [
        { label: t('fieldDesigner.fieldTypes.select.defaultOption1'), value: 'option1' },
        { label: t('fieldDesigner.fieldTypes.select.defaultOption2'), value: 'option2' },
      ],
      showSearch: true,
    },
  },
  {
    type: 'multi_select' as FieldType,
    label: t('fieldDesigner.fieldTypes.multiselect.label'),
    icon: '☑️',
    description: t('fieldDesigner.fieldTypes.multiselect.description'),
    category: 'basic',
    defaultConfig: {
      options: [
        { label: t('fieldDesigner.fieldTypes.multiselect.defaultOption1'), value: 'option1' },
        { label: t('fieldDesigner.fieldTypes.multiselect.defaultOption2'), value: 'option2' },
      ],
      multiple: true,
    },
  },
  {
    type: 'radio' as FieldType,
    label: t('fieldDesigner.fieldTypes.radio.label'),
    icon: '🔘',
    description: t('fieldDesigner.fieldTypes.radio.description'),
    category: 'basic',
    defaultConfig: {
      options: [
        { label: t('fieldDesigner.fieldTypes.radio.defaultOption1'), value: 'option1' },
        { label: t('fieldDesigner.fieldTypes.radio.defaultOption2'), value: 'option2' },
      ],
    },
  },
  {
    type: 'checkbox' as FieldType,
    label: t('fieldDesigner.fieldTypes.checkbox.label'),
    icon: '✅',
    description: t('fieldDesigner.fieldTypes.checkbox.description'),
    category: 'basic',
    defaultConfig: {
      options: [
        { label: t('fieldDesigner.fieldTypes.checkbox.defaultOption1'), value: 'option1' },
        { label: t('fieldDesigner.fieldTypes.checkbox.defaultOption2'), value: 'option2' },
      ],
    },
  },

  // 高级类型
  {
    type: 'user_picker' as FieldType,
    label: t('fieldDesigner.fieldTypes.user.label'),
    icon: '👤',
    description: t('fieldDesigner.fieldTypes.user.description'),
    category: 'advanced',
    defaultConfig: {
      showSearch: true,
      multiple: false,
    },
  },
  {
    type: 'department_picker' as FieldType,
    label: t('fieldDesigner.fieldTypes.dept.label'),
    icon: '🏢',
    description: t('fieldDesigner.fieldTypes.dept.description'),
    category: 'advanced',
    defaultConfig: {
      showSearch: true,
    },
  },
  {
    type: 'file_upload' as FieldType,
    label: t('fieldDesigner.fieldTypes.file.label'),
    icon: '📎',
    description: t('fieldDesigner.fieldTypes.file.description'),
    category: 'advanced',
    defaultConfig: {
      maxFileSize: 10 * 1024 * 1024, // 10MB
      acceptedFileTypes: ['image/*', 'application/pdf', '.doc', '.docx'],
      multiple: true,
    },
  },
  {
    type: 'rich_text' as FieldType,
    label: t('fieldDesigner.fieldTypes.richtext.label'),
    icon: '✏️',
    description: t('fieldDesigner.fieldTypes.richtext.description'),
    category: 'advanced',
    defaultConfig: {
      richTextConfig: {
        toolbar: ['bold', 'italic', 'underline', 'link', 'image'],
        height: 300,
      },
    },
  },
  {
    type: 'rating' as FieldType,
    label: t('fieldDesigner.fieldTypes.rate.label'),
    icon: '⭐',
    description: t('fieldDesigner.fieldTypes.rate.description'),
    category: 'advanced',
    defaultConfig: {
      validation: { min: 1, max: 5 },
    },
  },
  {
    type: 'slider' as FieldType,
    label: t('fieldDesigner.fieldTypes.slider.label'),
    icon: '🎚️',
    description: t('fieldDesigner.fieldTypes.slider.description'),
    category: 'advanced',
    defaultConfig: {
      validation: { min: 0, max: 100 },
    },
  },

  // 特殊类型
  {
    type: 'divider' as FieldType,
    label: t('fieldDesigner.fieldTypes.divider.label'),
    icon: '➖',
    description: t('fieldDesigner.fieldTypes.divider.description'),
    category: 'special',
    defaultConfig: {
      required: false,
    },
  },
  {
    type: 'section_title' as FieldType,
    label: t('fieldDesigner.fieldTypes.title.label'),
    icon: '📌',
    description: t('fieldDesigner.fieldTypes.title.description'),
    category: 'special',
    defaultConfig: {
      required: false,
    },
  },
];

// ==================== 可排序字段项组件 ====================

interface SortableFieldItemProps {
  field: TemplateField;
  index: number;
  onEdit: (field: TemplateField) => void;
  onDelete: (id: string) => void;
  onDuplicate: (field: TemplateField) => void;
  onMoveUp: (index: number) => void;
  onMoveDown: (index: number) => void;
  isFirst: boolean;
  isLast: boolean;
  fieldTypes: FieldTypeConfig[];
  t: (key: string) => string;
}

const SortableFieldItem: React.FC<SortableFieldItemProps> = ({
  field,
  index,
  onEdit,
  onDelete,
  onDuplicate,
  onMoveUp,
  onMoveDown,
  isFirst,
  isLast,
  fieldTypes,
  t,
}) => {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: field.id,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const fieldTypeConfig = fieldTypes.find(cfg => cfg.type === field.type);

  return (
    <div ref={setNodeRef} style={style} className="mb-2">
      <Card
        size="small"
        className="hover:shadow-md transition-shadow"
        styles={{ body: { padding: '12px 16px' } }}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3 flex-1">
            <div
              {...attributes}
              {...listeners}
              className="cursor-move text-gray-400 hover:text-gray-600"
            >
              <GripVertical style={{ fontSize: 16 }} />
            </div>

            <div className="flex items-center gap-2">
              <span className="text-xl">{fieldTypeConfig?.icon || '📝'}</span>
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-medium">{field.label}</span>
                  {field.required && (
                    <Tag color="red" style={{ margin: 0 }}>
                      {t('fieldDesigner.flags.required')}
                    </Tag>
                  )}
                  {field.conditional && (
                    <Tag color="blue" style={{ margin: 0 }}>
                      {t('fieldDesigner.flags.conditionalShow')}
                    </Tag>
                  )}
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  {fieldTypeConfig?.label} • {field.name}
                </div>
              </div>
            </div>
          </div>

          <Space size="small">
            <Tooltip title={t('fieldDesigner.actions.moveUp')}>
              <Button
                type="text"
                size="small"
                icon={<ArrowUp />}
                onClick={() => onMoveUp(index)}
                disabled={isFirst}
              />
            </Tooltip>
            <Tooltip title={t('fieldDesigner.actions.moveDown')}>
              <Button
                type="text"
                size="small"
                icon={<ArrowDown />}
                onClick={() => onMoveDown(index)}
                disabled={isLast}
              />
            </Tooltip>
            <Tooltip title={t('fieldDesigner.actions.edit')}>
              <Button
                type="text"
                size="small"
                icon={<Pencil />}
                onClick={() => onEdit(field)}
              />
            </Tooltip>
            <Tooltip title={t('fieldDesigner.actions.copy')}>
              <Button
                type="text"
                size="small"
                icon={<Copy />}
                onClick={() => onDuplicate(field)}
              />
            </Tooltip>
            <Tooltip title={t('fieldDesigner.actions.delete')}>
              <Button
                type="text"
                size="small"
                danger
                icon={<Trash2 />}
                onClick={() => onDelete(field.id)}
              />
            </Tooltip>
          </Space>
        </div>
      </Card>
    </div>
  );
};

// ==================== 字段配置面板 ====================

interface FieldConfigPanelProps {
  field: TemplateField | null;
  allFields: TemplateField[];
  onSave: (field: TemplateField) => void;
  onCancel: () => void;
  fieldTypes: FieldTypeConfig[];
  t: (key: string) => string;
}

const FieldConfigPanel: React.FC<FieldConfigPanelProps> = ({
  field,
  allFields,
  onSave,
  onCancel,
  fieldTypes,
  t,
}) => {
  const [form] = Form.useForm();
  const [currentField, setCurrentField] = useState<TemplateField | null>(field);
  const [activeTab, setActiveTab] = useState('basic');
  const { message } = App.useApp();

  React.useEffect(() => {
    if (field) {
      form.setFieldsValue(field);
      setCurrentField(field);
    }
  }, [field, form]);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      onSave({ ...currentField!, ...values });
      message.success(t('fieldDesigner.messages.saveSuccess'));
    } catch (error) {
      console.error('Form validation failed:', error);
    }
  };

  if (!field) {
    return (
      <Card className="h-full">
        <Empty description={t('fieldDesigner.empty.noFieldSelected')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
      </Card>
    );
  }

  const fieldTypeConfig = fieldTypes.find(cfg => cfg.type === field.type);
  const hasOptions = ['select', 'multi_select', 'radio', 'checkbox'].includes(field.type);

  return (
    <Card
      title={
        <div className="flex items-center gap-2">
          <span className="text-xl">{fieldTypeConfig?.icon}</span>
          <span>{t('fieldDesigner.section.config')}</span>
          <Tag color="blue">{fieldTypeConfig?.label}</Tag>
        </div>
      }
      extra={
        <Space>
          <Button onClick={onCancel}>{t('fieldDesigner.actions.cancel')}</Button>
          <Button type="primary" onClick={handleSave}>
            {t('fieldDesigner.actions.save')}
          </Button>
        </Space>
      }
      className="h-full"
      styles={{ body: { height: 'calc(100% - 57px)', overflowY: 'auto' } }}
    >
      <Form form={form} layout="vertical">
        <Tabs
          activeKey={activeTab} onChange={setActiveTab}
          items={[
                  {
                    key: 'basic',
                    label: t('fieldDesigner.section.basic'),
                    children: (
                      <>
            <Form.Item
              label={t('fieldDesigner.form.fieldName')}
              name="name"
              rules={[
                { required: true, message: t('fieldDesigner.form.fieldNameRequired') },
                {
                  pattern: /^[a-zA-Z_][a-zA-Z0-9_]*$/,
                  message: t('fieldDesigner.form.fieldNamePattern'),
                },
              ]}
              tooltip={t('fieldDesigner.form.fieldNameTooltip')}
            >
              <Input placeholder={t('fieldDesigner.form.fieldNamePlaceholder')} />
            </Form.Item>

            <Form.Item
              label={t('fieldDesigner.form.fieldLabel')}
              name="label"
              rules={[{ required: true, message: t('fieldDesigner.form.fieldLabelRequired') }]}
              tooltip={t('fieldDesigner.form.fieldLabelTooltip')}
            >
              <Input placeholder={t('fieldDesigner.form.fieldLabelPlaceholder')} />
            </Form.Item>

            <Form.Item label={t('fieldDesigner.form.placeholder')} name="placeholder">
              <Input placeholder={t('fieldDesigner.form.placeholderHint')} />
            </Form.Item>

            <Form.Item label={t('fieldDesigner.form.helpText')} name="helpText">
              <TextArea rows={2} placeholder={t('fieldDesigner.form.helpTextPlaceholder')} />
            </Form.Item>

            <Form.Item label={t('fieldDesigner.form.tooltip')} name="tooltip">
              <Input placeholder={t('fieldDesigner.form.tooltipPlaceholder')} />
            </Form.Item>

            <Row gutter={16}>
              <Col span={8}>
                <Form.Item label={t('fieldDesigner.form.required')} name="required" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item label={t('fieldDesigner.form.disabled')} name="disabled" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item label={t('fieldDesigner.form.hidden')} name="hidden" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item label={t('fieldDesigner.form.width')} name="width">
              <Select
                placeholder={t('fieldDesigner.form.widthPlaceholder')}
                options={[
                  { value: 24, label: t('fieldDesigner.form.widthFull') },
                  { value: 12, label: t('fieldDesigner.form.widthHalf') },
                  { value: 8, label: t('fieldDesigner.form.widthThird') },
                  { value: 6, label: t('fieldDesigner.form.widthQuarter') },
                ]}
              />
            </Form.Item>

            <Form.Item label={t('fieldDesigner.form.defaultValue')} name="defaultValue">
              <Input placeholder={t('fieldDesigner.form.defaultValuePlaceholder')} />
            </Form.Item>
                      </>
                    ),
                  },
                  {
                    key: 'validation',
                    label: t('fieldDesigner.section.validation'),
                    children: (
                      <>
            <Alert
              message={t('fieldDesigner.form.validationIntro')}
              type="info"
              showIcon
              className="mb-4"
            />

            <Form.Item label={t('fieldDesigner.form.minLength')} name={['validation', 'minLength']}>
              <InputNumber placeholder={t('fieldDesigner.form.minLengthPlaceholder')} min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item label={t('fieldDesigner.form.maxLength')} name={['validation', 'maxLength']}>
              <InputNumber placeholder={t('fieldDesigner.form.maxLengthPlaceholder')} min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item label={t('fieldDesigner.form.minValue')} name={['validation', 'minValue']}>
              <InputNumber placeholder={t('fieldDesigner.form.minValuePlaceholder')} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item label={t('fieldDesigner.form.maxValue')} name={['validation', 'maxValue']}>
              <InputNumber placeholder={t('fieldDesigner.form.maxValuePlaceholder')} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item label={t('fieldDesigner.form.pattern')} name={['validation', 'pattern']}>
              <Input placeholder={t('fieldDesigner.form.patternPlaceholder')} />
            </Form.Item>

            <Form.Item label={t('fieldDesigner.form.message')} name={['validation', 'customMessage']}>
              <TextArea rows={2} placeholder={t('fieldDesigner.form.messagePlaceholder')} />
            </Form.Item>

            <Row gutter={16}>
              <Col span={8}>
                <Form.Item label={t('fieldDesigner.form.errorEmail')} name={['validation', 'email']} valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item label={t('fieldDesigner.form.errorUrl')} name={['validation', 'url']} valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item label={t('fieldDesigner.form.errorPhone')} name={['validation', 'phone']} valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
            </Row>
                      </>
                    ),
                  },
                  {
                    key: 'options',
                    label: t('fieldDesigner.section.optionsConfig'),
                    children: (
                      <>
              <Alert
                message={t('fieldDesigner.form.optionsIntro')}
                type="info"
                showIcon
                className="mb-4"
              />

              <Form.List name="options">
                {(fields, { add, remove }) => (
                  <>
                    {fields.map((fieldItem, index) => (
                      <Card
                        key={fieldItem.key}
                        size="small"
                        className="mb-2"
                        extra={
                          <Button
                            type="text"
                            danger
                            size="small"
                            icon={<Trash2 />}
                            onClick={() => remove(fieldItem.name)}
                          />
                        }
                      >
                        <Row gutter={16}>
                          <Col span={10}>
                            <Form.Item
                              {...fieldItem}
                              label={t('fieldDesigner.form.optionLabel')}
                              name={[fieldItem.name, 'label']}
                              rules={[{ required: true, message: t('fieldDesigner.form.optionLabelRequired') }]}
                            >
                              <Input placeholder={t('fieldDesigner.form.optionLabelPlaceholder')} />
                            </Form.Item>
                          </Col>
                          <Col span={10}>
                            <Form.Item
                              {...fieldItem}
                              label={t('fieldDesigner.form.optionValue')}
                              name={[fieldItem.name, 'value']}
                              rules={[{ required: true, message: t('fieldDesigner.form.optionValueRequired') }]}
                            >
                              <Input placeholder={t('fieldDesigner.form.optionValuePlaceholder')} />
                            </Form.Item>
                          </Col>
                          <Col span={4}>
                            <Form.Item {...fieldItem} label={t('fieldDesigner.form.optionColor')} name={[fieldItem.name, 'color']}>
                              <Input type="color" />
                            </Form.Item>
                          </Col>
                        </Row>
                      </Card>
                    ))}
                    <Button
                      type="dashed"
                      onClick={() => add({ label: '', value: '' })}
                      block
                      icon={<Plus />}
                    >
                      {t('fieldDesigner.form.addOption')}
                    </Button>
                  </>
                )}
              </Form.List>
                      </>
                    ),
                  },
                  {
                    key: 'conditional',
                    label: t('fieldDesigner.section.conditional'),
                    children: (
                      <>
            <Alert
              message={t('fieldDesigner.form.conditionalIntro')}
              type="info"
              showIcon
              className="mb-4"
            />

            <Form.Item label={t('fieldDesigner.form.conditionalField')} name={['conditional', 'field']}>
              <Select
                placeholder={t('fieldDesigner.form.conditionalFieldPlaceholder')}
                allowClear
                showSearch
                optionFilterProp="children"
                options={allFields
                  .filter(f => f.id !== field.id)
                  .map(f => ({
                    value: f.name,
                    label: `${f.label} (${f.name})`,
                  }))}
              />
            </Form.Item>

            <Form.Item label={t('fieldDesigner.form.conditionalOperator')} name={['conditional', 'operator']}>
              <Select
                placeholder={t('fieldDesigner.form.conditionalOperatorPlaceholder')}
                options={[
                  { value: 'equals', label: t('fieldDesigner.form.operatorEquals') },
                  { value: 'not_equals', label: t('fieldDesigner.form.operatorNotEquals') },
                  { value: 'contains', label: t('fieldDesigner.form.operatorContains') },
                  { value: 'not_contains', label: t('fieldDesigner.form.operatorNotContains') },
                  { value: 'greater_than', label: t('fieldDesigner.form.operatorGreaterThan') },
                  { value: 'less_than', label: t('fieldDesigner.form.operatorLessThan') },
                  { value: 'in', label: t('fieldDesigner.form.operatorIn') },
                  { value: 'not_in', label: t('fieldDesigner.form.operatorNotIn') },
                ]}
              />
            </Form.Item>

            <Form.Item label={t('fieldDesigner.form.conditionalValue')} name={['conditional', 'value']}>
              <Input placeholder={t('fieldDesigner.form.conditionalValuePlaceholder')} />
            </Form.Item>
                      </>
                    ),
                  },
                  {
                    key: 'advanced',
                    label: t('fieldDesigner.section.advanced'),
                    children: (
                      <>
            {field.type === 'file_upload' && (
              <>
                <Form.Item label={t('fieldDesigner.form.maxFileSize')} name="maxFileSize">
                  <InputNumber min={1} max={100} placeholder="10" style={{ width: '100%' }} />
                </Form.Item>

                <Form.Item label={t('fieldDesigner.form.fileTypes')} name="acceptedFileTypes">
                  <Select
                    mode="tags"
                    placeholder={t('fieldDesigner.form.fileTypesPlaceholder')}
                    options={[
                      { value: 'image/*', label: t('fieldDesigner.form.fileImage') },
                      { value: 'application/pdf', label: t('fieldDesigner.form.filePdf') },
                      { value: '.doc', label: t('fieldDesigner.form.fileDoc') },
                      { value: '.docx', label: t('fieldDesigner.form.fileDocx') },
                      { value: '.xls', label: t('fieldDesigner.form.fileXls') },
                      { value: '.xlsx', label: t('fieldDesigner.form.fileXlsx') },
                    ]}
                  />
                </Form.Item>

                <Form.Item label={t('fieldDesigner.form.allowMultiple')} name="multiple" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </>
            )}

            {['select', 'multi_select', 'user_picker', 'department_picker'].includes(
              field.type
            ) && (
              <>
                <Form.Item label={t('fieldDesigner.form.showSearch')} name="showSearch" valuePropName="checked">
                  <Switch />
                </Form.Item>

                <Form.Item label={t('fieldDesigner.form.allowClear')} name="allowClear" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </>
            )}

            {field.type === 'multi_select' && (
              <Form.Item label={t('fieldDesigner.form.multipleMode')} name="multiple" valuePropName="checked">
                <Switch />
              </Form.Item>
            )}

            {field.type === 'rich_text' && (
              <>
                <Form.Item label={t('fieldDesigner.form.editorHeight')} name={['richTextConfig', 'height']}>
                  <InputNumber min={200} max={800} placeholder="300" style={{ width: '100%' }} />
                </Form.Item>

                <Form.Item label={t('fieldDesigner.form.toolbar')} name={['richTextConfig', 'toolbar']}>
                  <Select
                    mode="multiple"
                    placeholder={t('fieldDesigner.form.toolbarPlaceholder')}
                    options={[
                      { value: 'bold', label: t('fieldDesigner.form.toolbarBold') },
                      { value: 'italic', label: t('fieldDesigner.form.toolbarItalic') },
                      { value: 'underline', label: t('fieldDesigner.form.toolbarUnderline') },
                      { value: 'link', label: t('fieldDesigner.form.toolbarLink') },
                      { value: 'image', label: t('fieldDesigner.form.toolbarImage') },
                      { value: 'code', label: t('fieldDesigner.form.toolbarCode') },
                      { value: 'list', label: t('fieldDesigner.form.toolbarList') },
                    ]}
                  />
                </Form.Item>
              </>
            )}
                      </>
                    ),
                  },
          ]}
        />
      </Form>
    </Card>
  );
};

// ==================== 字段设计器主组件 ====================

export interface FieldDesignerProps {
  value?: TemplateField[];
  onChange?: (fields: TemplateField[]) => void;
  categoryId?: string;
}

export const FieldDesigner: React.FC<FieldDesignerProps> = ({
  value = [],
  onChange,
  categoryId,
}) => {
  const { t } = useI18n();
  const [fields, setFields] = useState<TemplateField[]>(value);
  const [selectedField, setSelectedField] = useState<TemplateField | null>(null);
  const [fieldTypeFilter, setFieldTypeFilter] = useState<string>('all');
  const { message } = App.useApp();

  const FIELD_TYPES = React.useMemo(() => createFieldTypes(t as unknown as (key: string) => string), [t]);

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  React.useEffect(() => {
    setFields(value);
  }, [value]);

  const handleFieldsChange = useCallback(
    (newFields: TemplateField[]) => {
      setFields(newFields);
      onChange?.(newFields);
    },
    [onChange]
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      const oldIndex = fields.findIndex(f => f.id === active.id);
      const newIndex = fields.findIndex(f => f.id === over.id);

      const newFields = arrayMove(fields, oldIndex, newIndex).map((f, idx) => ({
        ...f,
        order: idx,
      }));

      handleFieldsChange(newFields);
    }
  };

  const handleAddField = (typeConfig: FieldTypeConfig) => {
    const newField: TemplateField = {
      id: `field_${Date.now()}`,
      name: `field_${fields.length + 1}`,
      label: t('fieldDesigner.categories.newField') + ' ' + (fields.length + 1),
      type: typeConfig.type,
      required: false,
      order: fields.length,
      ...typeConfig.defaultConfig,
    };

    handleFieldsChange([...fields, newField]);
    setSelectedField(newField);
    message.success(t('fieldDesigner.messages.fieldAdded'));
  };

  const handleEditField = (field: TemplateField) => {
    setSelectedField(field);
  };

  const handleSaveField = (updatedField: TemplateField) => {
    const newFields = fields.map(f => (f.id === updatedField.id ? updatedField : f));
    handleFieldsChange(newFields);
    setSelectedField(null);
  };

  const handleDeleteField = (id: string) => {
    Modal.confirm({
      title: t('fieldDesigner.messages.confirmDeleteTitle'),
      content: t('fieldDesigner.messages.confirmDeleteContent'),
      onOk: () => {
        const newFields = fields.filter(f => f.id !== id);
        handleFieldsChange(newFields);
        if (selectedField?.id === id) {
          setSelectedField(null);
        }
        message.success(t('fieldDesigner.messages.fieldDeleted'));
      },
    });
  };

  const handleDuplicateField = (field: TemplateField) => {
    const newField: TemplateField = {
      ...field,
      id: `field_${Date.now()}`,
      name: `${field.name}_copy`,
      label: `${field.label} ${t('fieldDesigner.categories.copySuffix')}`,
      order: fields.length,
    };

    handleFieldsChange([...fields, newField]);
    message.success(t('fieldDesigner.messages.fieldDuplicated'));
  };

  const handleMoveUp = (index: number) => {
    if (index > 0) {
      const newFields = [...fields];
      [newFields[index - 1], newFields[index]] = [newFields[index], newFields[index - 1]];
      handleFieldsChange(newFields.map((f, idx) => ({ ...f, order: idx })));
    }
  };

  const handleMoveDown = (index: number) => {
    if (index < fields.length - 1) {
      const newFields = [...fields];
      [newFields[index], newFields[index + 1]] = [newFields[index + 1], newFields[index]];
      handleFieldsChange(newFields.map((f, idx) => ({ ...f, order: idx })));
    }
  };

  const filteredFieldTypes = useMemo(() => {
    if (fieldTypeFilter === 'all') return FIELD_TYPES;
    return FIELD_TYPES.filter(t => t.category === fieldTypeFilter);
  }, [fieldTypeFilter]);

  return (
    <div className="field-designer">
      <Row gutter={16} style={{ height: 'calc(100vh - 200px)' }}>
        {/* 左侧：字段类型面板 */}
        <Col span={5}>
          <Card
            title={t('fieldDesigner.categories.fieldTypes')}
            extra={
              <Select
                size="small"
                value={fieldTypeFilter}
                onChange={setFieldTypeFilter}
                style={{ width: 100 }}
                options={[
                  { value: 'all', label: t('fieldDesigner.categories.all') },
                  { value: 'basic', label: t('fieldDesigner.categories.basic') },
                  { value: 'advanced', label: t('fieldDesigner.categories.advanced') },
                  { value: 'special', label: t('fieldDesigner.categories.special') },
                ]}
              />
            }
            className="h-full"
            styles={{ body: { height: 'calc(100% - 57px)', overflowY: 'auto' } }}
          >
            <Space orientation="vertical" style={{ width: '100%' }} size="small">
              {filteredFieldTypes.map(typeConfig => (
                <Card
                  key={typeConfig.type}
                  size="small"
                  hoverable
                  onClick={() => handleAddField(typeConfig)}
                  className="cursor-pointer"
                >
                  <div className="flex items-center gap-2">
                    <span className="text-xl">{typeConfig.icon}</span>
                    <div className="flex-1">
                      <div className="font-medium text-sm">{typeConfig.label}</div>
                      <div className="text-xs text-gray-500">{typeConfig.description}</div>
                    </div>
                    <Plus className="text-blue-500" />
                  </div>
                </Card>
              ))}
            </Space>
          </Card>
        </Col>

        {/* 中间：字段列表 */}
        <Col span={10}>
          <Card
            title={
              <div className="flex items-center justify-between">
                <span>{t('fieldDesigner.categories.fieldList')}</span>
                <Badge count={fields.length} showZero color="blue" />
              </div>
            }
            className="h-full"
            styles={{ body: { height: 'calc(100% - 57px)', overflowY: 'auto' } }}
          >
            {fields.length === 0 ? (
              <Empty
                description={t('fieldDesigner.messages.noFields')}
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              />
            ) : (
              <DndContext
                sensors={sensors}
                collisionDetection={closestCenter}
                onDragEnd={handleDragEnd}
              >
                <SortableContext
                  items={fields.map(f => f.id)}
                  strategy={verticalListSortingStrategy}
                >
                  {fields.map((field, index) => (
                    <SortableFieldItem
                      key={field.id}
                      field={field}
                      index={index}
                      onEdit={handleEditField}
                      onDelete={handleDeleteField}
                      onDuplicate={handleDuplicateField}
                      onMoveUp={handleMoveUp}
                      onMoveDown={handleMoveDown}
                      isFirst={index === 0}
                      isLast={index === fields.length - 1}
                      fieldTypes={FIELD_TYPES}
                      t={t}
                    />
                  ))}
                </SortableContext>
              </DndContext>
            )}
          </Card>
        </Col>

        {/* 右侧：字段配置面板 */}
        <Col span={9}>
          <FieldConfigPanel
            field={selectedField}
            allFields={fields}
            onSave={handleSaveField}
            onCancel={() => setSelectedField(null)}
            fieldTypes={FIELD_TYPES}
            t={t}
          />
        </Col>
      </Row>
    </div>
  );
};

export default FieldDesigner;
