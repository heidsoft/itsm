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
import {
  PlusOutlined,
  DeleteOutlined,
  CopyOutlined,
  EditOutlined,
  DragOutlined,
  EyeOutlined,
  SettingOutlined,
  QuestionCircleOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
} from '@ant-design/icons';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
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

const { TextArea } = Input;
const { Panel } = Collapse;

// ==================== 字段类型配置 ====================

interface FieldTypeConfig {
  type: FieldType;
  label: string;
  icon: string;
  description: string;
  category: 'basic' | 'advanced' | 'special';
  defaultConfig: Partial<TemplateField>;
}

const FIELD_TYPES: FieldTypeConfig[] = [
  // 基础类型
  {
    type: 'text' as FieldType,
    label: '单行文本',
    icon: '📝',
    description: '单行文本输入框',
    category: 'basic',
    defaultConfig: {
      placeholder: '请输入内容',
      validation: { maxLength: 200 },
    },
  },
  {
    type: 'textarea' as FieldType,
    label: '多行文本',
    icon: '📄',
    description: '多行文本输入框',
    category: 'basic',
    defaultConfig: {
      placeholder: '请输入详细内容',
      validation: { maxLength: 2000 },
    },
  },
  {
    type: 'number' as FieldType,
    label: '数字',
    icon: '🔢',
    description: '数字输入框',
    category: 'basic',
    defaultConfig: {
      placeholder: '请输入数字',
    },
  },
  {
    type: 'date' as FieldType,
    label: '日期',
    icon: '📅',
    description: '日期选择器',
    category: 'basic',
    defaultConfig: {},
  },
  {
    type: 'datetime' as FieldType,
    label: '日期时间',
    icon: '🕐',
    description: '日期时间选择器',
    category: 'basic',
    defaultConfig: {},
  },

  // 选择类型
  {
    type: 'select' as FieldType,
    label: '下拉选择',
    icon: '📋',
    description: '单选下拉框',
    category: 'basic',
    defaultConfig: {
      options: [
        { label: '选项1', value: 'option1' },
        { label: '选项2', value: 'option2' },
      ],
      showSearch: true,
    },
  },
  {
    type: 'multi_select' as FieldType,
    label: '多选下拉',
    icon: '☑️',
    description: '多选下拉框',
    category: 'basic',
    defaultConfig: {
      options: [
        { label: '选项1', value: 'option1' },
        { label: '选项2', value: 'option2' },
      ],
      multiple: true,
    },
  },
  {
    type: 'radio' as FieldType,
    label: '单选按钮',
    icon: '🔘',
    description: '单选按钮组',
    category: 'basic',
    defaultConfig: {
      options: [
        { label: '选项1', value: 'option1' },
        { label: '选项2', value: 'option2' },
      ],
    },
  },
  {
    type: 'checkbox' as FieldType,
    label: '复选框',
    icon: '✅',
    description: '复选框组',
    category: 'basic',
    defaultConfig: {
      options: [
        { label: '选项1', value: 'option1' },
        { label: '选项2', value: 'option2' },
      ],
    },
  },

  // 高级类型
  {
    type: 'user_picker' as FieldType,
    label: '用户选择',
    icon: '👤',
    description: '选择用户',
    category: 'advanced',
    defaultConfig: {
      showSearch: true,
      multiple: false,
    },
  },
  {
    type: 'department_picker' as FieldType,
    label: '部门选择',
    icon: '🏢',
    description: '选择部门',
    category: 'advanced',
    defaultConfig: {
      showSearch: true,
    },
  },
  {
    type: 'file_upload' as FieldType,
    label: '文件上传',
    icon: '📎',
    description: '文件上传',
    category: 'advanced',
    defaultConfig: {
      maxFileSize: 10 * 1024 * 1024, // 10MB
      acceptedFileTypes: ['image/*', 'application/pdf', '.doc', '.docx'],
      multiple: true,
    },
  },
  {
    type: 'rich_text' as FieldType,
    label: '富文本',
    icon: '✏️',
    description: '富文本编辑器',
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
    label: '评分',
    icon: '⭐',
    description: '星级评分',
    category: 'advanced',
    defaultConfig: {
      validation: { min: 1, max: 5 },
    },
  },
  {
    type: 'slider' as FieldType,
    label: '滑块',
    icon: '🎚️',
    description: '数值滑块',
    category: 'advanced',
    defaultConfig: {
      validation: { min: 0, max: 100 },
    },
  },

  // 特殊类型
  {
    type: 'divider' as FieldType,
    label: '分隔线',
    icon: '➖',
    description: '视觉分隔',
    category: 'special',
    defaultConfig: {
      required: false,
    },
  },
  {
    type: 'section_title' as FieldType,
    label: '章节标题',
    icon: '📌',
    description: '表单章节',
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
}) => {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: field.id,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const fieldTypeConfig = FIELD_TYPES.find(t => t.type === field.type);

  return (
    <div ref={setNodeRef} style={style} className='mb-2'>
      <Card
        size='small'
        className='hover:shadow-md transition-shadow'
        styles={{ body: { padding: '12px 16px' } }}
      >
        <div className='flex items-center justify-between'>
          <div className='flex items-center gap-3 flex-1'>
            <div
              {...attributes}
              {...listeners}
              className='cursor-move text-gray-400 hover:text-gray-600'
            >
              <DragOutlined style={{ fontSize: 16 }} />
            </div>

            <div className='flex items-center gap-2'>
              <span className='text-xl'>{fieldTypeConfig?.icon || '📝'}</span>
              <div>
                <div className='flex items-center gap-2'>
                  <span className='font-medium'>{field.label}</span>
                  {field.required && (
                    <Tag color='red' style={{ margin: 0 }}>
                      必填
                    </Tag>
                  )}
                  {field.conditional && (
                    <Tag color='blue' style={{ margin: 0 }}>
                      条件显示
                    </Tag>
                  )}
                </div>
                <div className='text-xs text-gray-500 mt-1'>
                  {fieldTypeConfig?.label} • {field.name}
                </div>
              </div>
            </div>
          </div>

          <Space size='small'>
            <Tooltip title='上移'>
              <Button
                type='text'
                size='small'
                icon={<ArrowUpOutlined />}
                onClick={() => onMoveUp(index)}
                disabled={isFirst}
              />
            </Tooltip>
            <Tooltip title='下移'>
              <Button
                type='text'
                size='small'
                icon={<ArrowDownOutlined />}
                onClick={() => onMoveDown(index)}
                disabled={isLast}
              />
            </Tooltip>
            <Tooltip title='编辑'>
              <Button
                type='text'
                size='small'
                icon={<EditOutlined />}
                onClick={() => onEdit(field)}
              />
            </Tooltip>
            <Tooltip title='复制'>
              <Button
                type='text'
                size='small'
                icon={<CopyOutlined />}
                onClick={() => onDuplicate(field)}
              />
            </Tooltip>
            <Tooltip title='删除'>
              <Button
                type='text'
                size='small'
                danger
                icon={<DeleteOutlined />}
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
}

const FieldConfigPanel: React.FC<FieldConfigPanelProps> = ({
  field,
  allFields,
  onSave,
  onCancel,
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
      message.success('字段配置已保存');
    } catch (error) {
      console.error('表单验证失败:', error);
    }
  };

  if (!field) {
    return (
      <Card className='h-full'>
        <Empty description='请选择或添加字段进行配置' image={Empty.PRESENTED_IMAGE_SIMPLE} />
      </Card>
    );
  }

  const fieldTypeConfig = FIELD_TYPES.find(t => t.type === field.type);
  const hasOptions = ['select', 'multi_select', 'radio', 'checkbox'].includes(field.type);

  return (
    <Card
      title={
        <div className='flex items-center gap-2'>
          <span className='text-xl'>{fieldTypeConfig?.icon}</span>
          <span>字段配置</span>
          <Tag color='blue'>{fieldTypeConfig?.label}</Tag>
        </div>
      }
      extra={
        <Space>
          <Button onClick={onCancel}>取消</Button>
          <Button type='primary' onClick={handleSave}>
            保存
          </Button>
        </Space>
      }
      className='h-full'
      styles={{ body: { height: 'calc(100% - 57px)', overflowY: 'auto' } }}
    >
      <Form form={form} layout='vertical'>
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          {/* 基础设置 */}
          <Tabs.TabPane tab='基础设置' key='basic'>
            <Form.Item
              label='字段名称'
              name='name'
              rules={[
                { required: true, message: '请输入字段名称' },
                {
                  pattern: /^[a-zA-Z_][a-zA-Z0-9_]*$/,
                  message: '只能包含字母、数字和下划线，且以字母或下划线开头',
                },
              ]}
              tooltip='用于数据存储的字段标识符，创建后不建议修改'
            >
              <Input placeholder='如：customer_name' />
            </Form.Item>

            <Form.Item
              label='字段标签'
              name='label'
              rules={[{ required: true, message: '请输入字段标签' }]}
              tooltip='显示给用户的字段名称'
            >
              <Input placeholder='如：客户姓名' />
            </Form.Item>

            <Form.Item label='占位符' name='placeholder'>
              <Input placeholder='输入框的占位提示文字' />
            </Form.Item>

            <Form.Item label='帮助文本' name='helpText'>
              <TextArea rows={2} placeholder='帮助用户理解如何填写此字段' />
            </Form.Item>

            <Form.Item label='工具提示' name='tooltip'>
              <Input placeholder='鼠标悬停时显示的提示' />
            </Form.Item>

            <Row gutter={16}>
              <Col span={8}>
                <Form.Item label='必填' name='required' valuePropName='checked'>
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item label='禁用' name='disabled' valuePropName='checked'>
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item label='隐藏' name='hidden' valuePropName='checked'>
                  <Switch />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item label='字段宽度' name='width'>
              <Select
                placeholder='选择字段宽度'
                options={[
                  { value: 24, label: '100% (全宽)' },
                  { value: 12, label: '50% (半宽)' },
                  { value: 8, label: '33% (三分之一)' },
                  { value: 6, label: '25% (四分之一)' },
                ]}
              />
            </Form.Item>

            <Form.Item label='默认值' name='defaultValue'>
              <Input placeholder='字段的默认值' />
            </Form.Item>
          </Tabs.TabPane>

          {/* 验证规则 */}
          <Tabs.TabPane tab='验证规则' key='validation'>
            <Alert
              message='配置字段的验证规则，确保用户输入符合要求'
              type='info'
              showIcon
              className='mb-4'
            />

            <Form.Item label='最小长度' name={['validation', 'minLength']}>
              <InputNumber placeholder='字符最小长度' min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item label='最大长度' name={['validation', 'maxLength']}>
              <InputNumber placeholder='字符最大长度' min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item label='最小值' name={['validation', 'minValue']}>
              <InputNumber placeholder='数值最小值' style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item label='最大值' name={['validation', 'maxValue']}>
              <InputNumber placeholder='数值最大值' style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item label='正则表达式' name={['validation', 'pattern']}>
              <Input placeholder='如：^[0-9]{11}$' />
            </Form.Item>

            <Form.Item label='自定义错误消息' name={['validation', 'customMessage']}>
              <TextArea rows={2} placeholder='验证失败时显示的错误消息' />
            </Form.Item>

            <Row gutter={16}>
              <Col span={8}>
                <Form.Item label='邮箱格式' name={['validation', 'email']} valuePropName='checked'>
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item label='URL格式' name={['validation', 'url']} valuePropName='checked'>
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item label='电话格式' name={['validation', 'phone']} valuePropName='checked'>
                  <Switch />
                </Form.Item>
              </Col>
            </Row>
          </Tabs.TabPane>

          {/* 选项配置 */}
          {hasOptions && (
            <Tabs.TabPane tab='选项配置' key='options'>
              <Alert
                message='配置下拉选择、单选、多选等字段的选项'
                type='info'
                showIcon
                className='mb-4'
              />

              <Form.List name='options'>
                {(fields, { add, remove }) => (
                  <>
                    {fields.map((fieldItem, index) => (
                      <Card
                        key={fieldItem.key}
                        size='small'
                        className='mb-2'
                        extra={
                          <Button
                            type='text'
                            danger
                            size='small'
                            icon={<DeleteOutlined />}
                            onClick={() => remove(fieldItem.name)}
                          />
                        }
                      >
                        <Row gutter={16}>
                          <Col span={10}>
                            <Form.Item
                              {...fieldItem}
                              label='标签'
                              name={[fieldItem.name, 'label']}
                              rules={[{ required: true, message: '请输入标签' }]}
                            >
                              <Input placeholder='显示文本' />
                            </Form.Item>
                          </Col>
                          <Col span={10}>
                            <Form.Item
                              {...fieldItem}
                              label='值'
                              name={[fieldItem.name, 'value']}
                              rules={[{ required: true, message: '请输入值' }]}
                            >
                              <Input placeholder='实际值' />
                            </Form.Item>
                          </Col>
                          <Col span={4}>
                            <Form.Item {...fieldItem} label='颜色' name={[fieldItem.name, 'color']}>
                              <Input type='color' />
                            </Form.Item>
                          </Col>
                        </Row>
                      </Card>
                    ))}
                    <Button
                      type='dashed'
                      onClick={() => add({ label: '', value: '' })}
                      block
                      icon={<PlusOutlined />}
                    >
                      添加选项
                    </Button>
                  </>
                )}
              </Form.List>
            </Tabs.TabPane>
          )}

          {/* 条件显示 */}
          <Tabs.TabPane tab='条件显示' key='conditional'>
            <Alert
              message='根据其他字段的值决定是否显示此字段'
              type='info'
              showIcon
              className='mb-4'
            />

            <Form.Item label='依赖字段' name={['conditional', 'field']}>
              <Select
                placeholder='选择依赖的字段'
                allowClear
                showSearch
                optionFilterProp='children'
                options={allFields
                  .filter(f => f.id !== field.id)
                  .map(f => ({
                    value: f.name,
                    label: `${f.label} (${f.name})`,
                  }))}
              />
            </Form.Item>

            <Form.Item label='条件运算符' name={['conditional', 'operator']}>
              <Select
                placeholder='选择运算符'
                options={[
                  { value: 'equals', label: '等于 (=)' },
                  { value: 'not_equals', label: '不等于 (≠)' },
                  { value: 'contains', label: '包含' },
                  { value: 'not_contains', label: '不包含' },
                  { value: 'greater_than', label: '大于 (> )' },
                  { value: 'less_than', label: '小于 (< )' },
                  { value: 'in', label: '在列表中' },
                  { value: 'not_in', label: '不在列表中' },
                ]}
              />
            </Form.Item>

            <Form.Item label='比较值' name={['conditional', 'value']}>
              <Input placeholder='要比较的值' />
            </Form.Item>
          </Tabs.TabPane>

          {/* 高级配置 */}
          <Tabs.TabPane tab='高级配置' key='advanced'>
            {field.type === 'file_upload' && (
              <>
                <Form.Item label='最大文件大小（MB）' name='maxFileSize'>
                  <InputNumber min={1} max={100} placeholder='10' style={{ width: '100%' }} />
                </Form.Item>

                <Form.Item label='允许的文件类型' name='acceptedFileTypes'>
                  <Select
                    mode='tags'
                    placeholder='如：image/*, .pdf, .docx'
                    options={[
                      { value: 'image/*', label: '图片 (image/*)' },
                      { value: 'application/pdf', label: 'PDF' },
                      { value: '.doc', label: 'Word (.doc)' },
                      { value: '.docx', label: 'Word (.docx)' },
                      { value: '.xls', label: 'Excel (.xls)' },
                      { value: '.xlsx', label: 'Excel (.xlsx)' },
                    ]}
                  />
                </Form.Item>

                <Form.Item label='允许多个文件' name='multiple' valuePropName='checked'>
                  <Switch />
                </Form.Item>
              </>
            )}

            {['select', 'multi_select', 'user_picker', 'department_picker'].includes(
              field.type
            ) && (
              <>
                <Form.Item label='显示搜索框' name='showSearch' valuePropName='checked'>
                  <Switch />
                </Form.Item>

                <Form.Item label='允许清空' name='allowClear' valuePropName='checked'>
                  <Switch />
                </Form.Item>
              </>
            )}

            {field.type === 'multi_select' && (
              <Form.Item label='多选模式' name='multiple' valuePropName='checked'>
                <Switch />
              </Form.Item>
            )}

            {field.type === 'rich_text' && (
              <>
                <Form.Item label='编辑器高度' name={['richTextConfig', 'height']}>
                  <InputNumber min={200} max={800} placeholder='300' style={{ width: '100%' }} />
                </Form.Item>

                <Form.Item label='工具栏' name={['richTextConfig', 'toolbar']}>
                  <Select
                    mode='multiple'
                    placeholder='选择工具栏按钮'
                    options={[
                      { value: 'bold', label: '粗体' },
                      { value: 'italic', label: '斜体' },
                      { value: 'underline', label: '下划线' },
                      { value: 'link', label: '链接' },
                      { value: 'image', label: '图片' },
                      { value: 'code', label: '代码' },
                      { value: 'list', label: '列表' },
                    ]}
                  />
                </Form.Item>
              </>
            )}
          </Tabs.TabPane>
        </Tabs>
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
  const [fields, setFields] = useState<TemplateField[]>(value);
  const [selectedField, setSelectedField] = useState<TemplateField | null>(null);
  const [fieldTypeFilter, setFieldTypeFilter] = useState<string>('all');
  const { message } = App.useApp();

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
      label: `新字段 ${fields.length + 1}`,
      type: typeConfig.type,
      required: false,
      order: fields.length,
      ...typeConfig.defaultConfig,
    };

    handleFieldsChange([...fields, newField]);
    setSelectedField(newField);
    message.success('字段已添加');
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
      title: '确认删除',
      content: '确定要删除这个字段吗？此操作不可撤销。',
      onOk: () => {
        const newFields = fields.filter(f => f.id !== id);
        handleFieldsChange(newFields);
        if (selectedField?.id === id) {
          setSelectedField(null);
        }
        message.success('字段已删除');
      },
    });
  };

  const handleDuplicateField = (field: TemplateField) => {
    const newField: TemplateField = {
      ...field,
      id: `field_${Date.now()}`,
      name: `${field.name}_copy`,
      label: `${field.label} (副本)`,
      order: fields.length,
    };

    handleFieldsChange([...fields, newField]);
    message.success('字段已复制');
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
    <div className='field-designer'>
      <Row gutter={16} style={{ height: 'calc(100vh - 200px)' }}>
        {/* 左侧：字段类型面板 */}
        <Col span={5}>
          <Card
            title='字段类型'
            extra={
              <Select
                size='small'
                value={fieldTypeFilter}
                onChange={setFieldTypeFilter}
                style={{ width: 100 }}
                options={[
                  { value: 'all', label: '全部' },
                  { value: 'basic', label: '基础' },
                  { value: 'advanced', label: '高级' },
                  { value: 'special', label: '特殊' },
                ]}
              />
            }
            className='h-full'
            styles={{ body: { height: 'calc(100% - 57px)', overflowY: 'auto' } }}
          >
            <Space orientation='vertical' style={{ width: '100%' }} size='small'>
              {filteredFieldTypes.map(typeConfig => (
                <Card
                  key={typeConfig.type}
                  size='small'
                  hoverable
                  onClick={() => handleAddField(typeConfig)}
                  className='cursor-pointer'
                >
                  <div className='flex items-center gap-2'>
                    <span className='text-xl'>{typeConfig.icon}</span>
                    <div className='flex-1'>
                      <div className='font-medium text-sm'>{typeConfig.label}</div>
                      <div className='text-xs text-gray-500'>{typeConfig.description}</div>
                    </div>
                    <PlusOutlined className='text-blue-500' />
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
              <div className='flex items-center justify-between'>
                <span>字段列表</span>
                <Badge count={fields.length} showZero color='blue' />
              </div>
            }
            className='h-full'
            styles={{ body: { height: 'calc(100% - 57px)', overflowY: 'auto' } }}
          >
            {fields.length === 0 ? (
              <Empty
                description='暂无字段，请从左侧添加字段'
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
          />
        </Col>
      </Row>
    </div>
  );
};

export default FieldDesigner;
