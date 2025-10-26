/**
 * 统一表单组件
 * 提供功能完整的表单组件，支持动态字段、验证、布局等
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Form,
  Input,
  Select,
  DatePicker,
  TimePicker,
  InputNumber,
  Switch,
  Radio,
  Checkbox,
  Upload,
  Button,
  Space,
  Row,
  Col,
  Card,
  Divider,
  Typography,
  message,
} from 'antd';
import {
  PlusOutlined,
  MinusCircleOutlined,
  UploadOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import { FormField, Option } from '../../types/common';
import { useFormValidation } from '../../lib/component-utils';

const { TextArea } = Input;
const { Option: SelectOption } = Select;
const { RangePicker } = DatePicker;
const { Group: RadioGroup } = Radio;
const { Group: CheckboxGroup } = Checkbox;
const { Text, Title } = Typography;

// 表单配置接口
export interface FormConfig {
  fields: FormField[];
  layout?: 'horizontal' | 'vertical' | 'inline';
  labelCol?: { span: number };
  wrapperCol?: { span: number };
  size?: 'small' | 'middle' | 'large';
  disabled?: boolean;
  readonly?: boolean;
}

// 表单属性接口
export interface UnifiedFormProps<T = Record<string, unknown>> {
  // 表单配置
  config: FormConfig;

  // 初始值
  initialValues?: T;

  // 事件处理
  onSubmit?: (values: T) => void | Promise<void>;
  onCancel?: () => void;
  onChange?: (values: T, changedFields: string[]) => void;

  // 按钮配置
  submitText?: string;
  cancelText?: string;
  showCancel?: boolean;
  showReset?: boolean;

  // 状态
  loading?: boolean;
  disabled?: boolean;

  // 样式
  className?: string;
  title?: string;
  extra?: React.ReactNode;

  // 其他
  preserve?: boolean;
  scrollToFirstError?: boolean;
}

// 统一表单组件
export function UnifiedForm<T extends Record<string, unknown>>({
  config,
  initialValues = {} as T,
  onSubmit,
  onCancel,
  onChange,
  submitText = '提交',
  cancelText = '取消',
  showCancel = true,
  showReset = true,
  loading = false,
  disabled = false,
  className,
  title,
  extra,
  preserve = false,
  scrollToFirstError = true,
}: UnifiedFormProps<T>) {
  const [form] = Form.useForm();
  const [values, setValues] = useState<T>(initialValues);
  const [changedFields, setChangedFields] = useState<string[]>([]);

  // 表单验证规则
  const validationRules = useMemo(() => {
    const rules: Record<string, (value: unknown) => string | null> = {};

    config.fields.forEach(field => {
      if (field.validation) {
        rules[field.name] = (value: unknown) => {
          if (field.required && (!value || (typeof value === 'string' && !value.trim()))) {
            return `${field.label}是必填项`;
          }

          if (field.validation?.min && typeof value === 'number' && value < field.validation.min) {
            return `${field.label}不能小于${field.validation.min}`;
          }

          if (field.validation?.max && typeof value === 'number' && value > field.validation.max) {
            return `${field.label}不能大于${field.validation.max}`;
          }

          if (
            field.validation?.pattern &&
            typeof value === 'string' &&
            !field.validation.pattern.test(value)
          ) {
            return field.validation.message || `${field.label}格式不正确`;
          }

          return null;
        };
      }
    });

    return rules;
  }, [config.fields]);

  // 表单验证Hook
  const {
    values: validatedValues,
    errors,
    setValue,
    validateForm,
    resetForm,
  } = useFormValidation(initialValues, validationRules);

  // 处理表单值变化
  const handleValuesChange = useCallback(
    (changedValues: Partial<T>, allValues: T) => {
      setValues(allValues);

      const changedKeys = Object.keys(changedValues);
      setChangedFields(prev => [...new Set([...prev, ...changedKeys])]);

      // 更新验证状态
      changedKeys.forEach(key => {
        setValue(key as keyof T, allValues[key as keyof T]);
      });

      onChange?.(allValues, changedKeys);
    },
    [onChange, setValue]
  );

  // 处理表单提交
  const handleSubmit = useCallback(async () => {
    try {
      const isValid = validateForm();
      if (!isValid) {
        message.error('请检查表单填写是否正确');
        return;
      }

      await onSubmit?.(values);
    } catch (error) {
      console.error('Form submission error:', error);
      message.error('提交失败，请重试');
    }
  }, [values, onSubmit, validateForm]);

  // 处理表单重置
  const handleReset = useCallback(() => {
    form.resetFields();
    setValues(initialValues);
    setChangedFields([]);
    resetForm();
  }, [form, initialValues, resetForm]);

  // 处理取消
  const handleCancel = useCallback(() => {
    onCancel?.();
  }, [onCancel]);

  // 渲染表单字段
  const renderField = useCallback(
    (field: FormField) => {
      const fieldValue = values[field.name as keyof T];
      const fieldError = errors[field.name as keyof T];
      const isDisabled = disabled || config.disabled || field.disabled;

      const commonProps = {
        placeholder: field.placeholder,
        disabled: isDisabled,
        readOnly: config.readonly,
      };

      switch (field.type) {
        case 'text':
          return (
            <Input
              {...commonProps}
              value={fieldValue as string}
              onChange={e =>
                handleValuesChange({ [field.name]: e.target.value } as Partial<T>, values)
              }
            />
          );

        case 'textarea':
          return (
            <TextArea
              {...commonProps}
              rows={4}
              value={fieldValue as string}
              onChange={e =>
                handleValuesChange({ [field.name]: e.target.value } as Partial<T>, values)
              }
            />
          );

        case 'select':
          return (
            <Select
              {...commonProps}
              value={fieldValue}
              onChange={value => handleValuesChange({ [field.name]: value } as Partial<T>, values)}
              allowClear
            >
              {field.options?.map(option => (
                <SelectOption key={option.value} value={option.value} disabled={option.disabled}>
                  {option.label}
                </SelectOption>
              ))}
            </Select>
          );

        case 'multiselect':
          return (
            <Select
              mode='multiple'
              {...commonProps}
              value={fieldValue as string[]}
              onChange={value => handleValuesChange({ [field.name]: value } as Partial<T>, values)}
              allowClear
            >
              {field.options?.map(option => (
                <SelectOption key={option.value} value={option.value} disabled={option.disabled}>
                  {option.label}
                </SelectOption>
              ))}
            </Select>
          );

        case 'date':
          return (
            <DatePicker
              {...commonProps}
              value={fieldValue}
              onChange={date =>
                handleValuesChange(
                  { [field.name]: date?.format('YYYY-MM-DD') } as Partial<T>,
                  values
                )
              }
              style={{ width: '100%' }}
            />
          );

        case 'datetime':
          return (
            <DatePicker
              showTime
              {...commonProps}
              value={fieldValue}
              onChange={date =>
                handleValuesChange(
                  { [field.name]: date?.format('YYYY-MM-DD HH:mm:ss') } as Partial<T>,
                  values
                )
              }
              style={{ width: '100%' }}
            />
          );

        case 'daterange':
          return (
            <RangePicker
              {...commonProps}
              value={fieldValue}
              onChange={dates =>
                handleValuesChange(
                  { [field.name]: dates?.map(d => d?.format('YYYY-MM-DD')) } as Partial<T>,
                  values
                )
              }
              style={{ width: '100%' }}
            />
          );

        case 'number':
          return (
            <InputNumber
              {...commonProps}
              value={fieldValue as number}
              onChange={value => handleValuesChange({ [field.name]: value } as Partial<T>, values)}
              style={{ width: '100%' }}
              min={field.validation?.min}
              max={field.validation?.max}
            />
          );

        case 'boolean':
          return (
            <Switch
              checked={fieldValue as boolean}
              onChange={checked =>
                handleValuesChange({ [field.name]: checked } as Partial<T>, values)
              }
              disabled={isDisabled}
            />
          );

        case 'radio':
          return (
            <RadioGroup
              value={fieldValue}
              onChange={e =>
                handleValuesChange({ [field.name]: e.target.value } as Partial<T>, values)
              }
              disabled={isDisabled}
            >
              {field.options?.map(option => (
                <Radio key={option.value} value={option.value} disabled={option.disabled}>
                  {option.label}
                </Radio>
              ))}
            </RadioGroup>
          );

        case 'checkbox':
          return (
            <CheckboxGroup
              value={fieldValue as string[]}
              onChange={checkedValues =>
                handleValuesChange({ [field.name]: checkedValues } as Partial<T>, values)
              }
              disabled={isDisabled}
            >
              {field.options?.map(option => (
                <Checkbox key={option.value} value={option.value} disabled={option.disabled}>
                  {option.label}
                </Checkbox>
              ))}
            </CheckboxGroup>
          );

        case 'file':
          return (
            <Upload
              beforeUpload={() => false}
              onChange={info => {
                const file = info.file.originFileObj;
                handleValuesChange({ [field.name]: file } as Partial<T>, values);
              }}
              disabled={isDisabled}
            >
              <Button icon={<UploadOutlined />} disabled={isDisabled}>
                选择文件
              </Button>
            </Upload>
          );

        default:
          return null;
      }
    },
    [values, errors, disabled, config.disabled, config.readonly, handleValuesChange]
  );

  // 渲染表单字段
  const renderFormFields = useMemo(() => {
    return config.fields.map(field => {
      const fieldError = errors[field.name as keyof T];

      return (
        <Form.Item
          key={field.name}
          label={field.label}
          name={field.name}
          rules={[
            { required: field.required, message: `${field.label}是必填项` },
            ...(field.validation?.pattern
              ? [
                  {
                    pattern: field.validation.pattern,
                    message: field.validation.message || `${field.label}格式不正确`,
                  },
                ]
              : []),
          ]}
          validateStatus={fieldError ? 'error' : ''}
          help={fieldError}
        >
          {renderField(field)}
        </Form.Item>
      );
    });
  }, [config.fields, errors, renderField]);

  // 渲染表单按钮
  const renderFormButtons = useMemo(() => {
    return (
      <Form.Item>
        <Space>
          <Button
            type='primary'
            htmlType='submit'
            loading={loading}
            disabled={disabled || config.disabled}
          >
            {submitText}
          </Button>

          {showCancel && (
            <Button onClick={handleCancel} disabled={loading}>
              {cancelText}
            </Button>
          )}

          {showReset && (
            <Button onClick={handleReset} disabled={loading}>
              重置
            </Button>
          )}
        </Space>
      </Form.Item>
    );
  }, [
    loading,
    disabled,
    config.disabled,
    submitText,
    showCancel,
    cancelText,
    handleCancel,
    showReset,
    handleReset,
  ]);

  // 渲染表单头部
  const renderFormHeader = useMemo(() => {
    if (!title && !extra) return null;

    return (
      <div className='flex justify-between items-center mb-6'>
        {title && <Title level={4}>{title}</Title>}
        {extra && <div>{extra}</div>}
      </div>
    );
  }, [title, extra]);

  return (
    <Card className={className}>
      {renderFormHeader}

      <Form
        form={form}
        layout={config.layout || 'vertical'}
        labelCol={config.labelCol}
        wrapperCol={config.wrapperCol}
        size={config.size || 'middle'}
        initialValues={initialValues}
        onValuesChange={handleValuesChange}
        onFinish={handleSubmit}
        preserve={preserve}
        scrollToFirstError={scrollToFirstError}
      >
        {renderFormFields}
        {renderFormButtons}
      </Form>
    </Card>
  );
}

// 动态表单组件（支持动态添加/删除字段）
export interface DynamicFormProps<T = Record<string, unknown>> extends UnifiedFormProps<T> {
  onAddField?: (field: FormField) => void;
  onRemoveField?: (fieldName: string) => void;
  allowAddRemove?: boolean;
}

export function DynamicForm<T extends Record<string, unknown>>({
  config,
  onAddField,
  onRemoveField,
  allowAddRemove = false,
  ...props
}: DynamicFormProps<T>) {
  const [dynamicFields, setDynamicFields] = useState<FormField[]>(config.fields);

  const handleAddField = useCallback(() => {
    const newField: FormField = {
      name: `field_${Date.now()}`,
      label: '新字段',
      type: 'text',
      required: false,
    };

    setDynamicFields(prev => [...prev, newField]);
    onAddField?.(newField);
  }, [onAddField]);

  const handleRemoveField = useCallback(
    (fieldName: string) => {
      setDynamicFields(prev => prev.filter(field => field.name !== fieldName));
      onRemoveField?.(fieldName);
    },
    [onRemoveField]
  );

  const updatedConfig = useMemo(
    () => ({
      ...config,
      fields: dynamicFields,
    }),
    [config, dynamicFields]
  );

  return (
    <div>
      <UnifiedForm {...props} config={updatedConfig} />

      {allowAddRemove && (
        <div className='mt-4'>
          <Button type='dashed' icon={<PlusOutlined />} onClick={handleAddField} block>
            添加字段
          </Button>
        </div>
      )}
    </div>
  );
}

// 导出组件
export default UnifiedForm;
export { DynamicForm };
