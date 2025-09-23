import { ReactNode } from 'react';
import { ValidationRule } from '../../utils/validation';

// 基础表单字段属性
export interface BaseFormFieldProps {
  name: string;
  label?: string;
  placeholder?: string;
  required?: boolean;
  disabled?: boolean;
  readonly?: boolean;
  error?: string | string[];
  help?: string;
  className?: string;
  onChange?: (value: unknown) => void;
  onBlur?: () => void;
  onFocus?: () => void;
}

// 表单字段组件属性
export interface FormFieldProps extends BaseFormFieldProps {
  children: ReactNode;
  layout?: 'vertical' | 'horizontal';
  labelWidth?: string;
}

// 输入框组件属性
export interface FormInputProps extends BaseFormFieldProps {
  type?: 'text' | 'password' | 'email' | 'number' | 'tel' | 'url';
  value?: string;
  maxLength?: number;
  minLength?: number;
  pattern?: string;
  autoComplete?: string;
  autoFocus?: boolean;
  prefix?: ReactNode;
  suffix?: ReactNode;
}

// 文本域组件属性
export interface FormTextareaProps extends BaseFormFieldProps {
  value?: string;
  rows?: number;
  cols?: number;
  maxLength?: number;
  minLength?: number;
  resize?: 'none' | 'both' | 'horizontal' | 'vertical';
  autoResize?: boolean;
}

// 选择框选项
export interface SelectOption {
  label: string;
  value: string | number;
  disabled?: boolean;
  group?: string;
}

// 选择框组件属性
export interface FormSelectProps extends BaseFormFieldProps {
  value?: string | number | string[] | number[];
  options: SelectOption[];
  multiple?: boolean;
  searchable?: boolean;
  clearable?: boolean;
  loading?: boolean;
  loadingText?: string;
  noOptionsText?: string;
  maxTagCount?: number;
  onSearch?: (query: string) => void;
}

// 复选框组件属性
export interface FormCheckboxProps extends BaseFormFieldProps {
  checked?: boolean;
  value?: string | number;
  indeterminate?: boolean;
  children?: ReactNode;
}

// 单选框选项
export interface RadioOption {
  label: string;
  value: string | number;
  disabled?: boolean;
}

// 单选框组件属性
export interface FormRadioProps extends BaseFormFieldProps {
  checked?: boolean;
  value?: string | number;
  children?: ReactNode;
}

// 单选框组属性
export interface FormRadioGroupProps extends BaseFormFieldProps {
  value?: string | number;
  options: RadioOption[];
  direction?: 'horizontal' | 'vertical';
}

// 日期选择器组件属性
export interface FormDatePickerProps extends BaseFormFieldProps {
  value?: string | Date;
  format?: string;
  showTime?: boolean;
  showToday?: boolean;
  disabledDate?: (date: Date) => boolean;
  minDate?: Date;
  maxDate?: Date;
  picker?: 'date' | 'week' | 'month' | 'quarter' | 'year';
}

// 文件上传组件属性
export interface FormUploadProps extends BaseFormFieldProps {
  value?: File[] | string[];
  accept?: string;
  multiple?: boolean;
  maxSize?: number;
  maxCount?: number;
  listType?: 'text' | 'picture' | 'picture-card';
  showUploadList?: boolean;
  beforeUpload?: (file: File) => boolean | Promise<boolean>;
  onUpload?: (file: File) => Promise<string>;
  onRemove?: (file: File | string) => void;
}

// 表单字段配置
export interface FormFieldConfig {
  name: string;
  type: 'text' | 'email' | 'password' | 'number' | 'tel' | 'url' | 'textarea' | 'select' | 'checkbox' | 'radio' | 'date' | 'upload' | 'custom';
  label?: string;
  placeholder?: string;
  required?: boolean;
  disabled?: boolean;
  readonly?: boolean;
  help?: string;
  defaultValue?: unknown;
  rules?: ValidationRule[];
  props?: Record<string, unknown>;
  dependencies?: string[];
  visible?: boolean | ((values: Record<string, unknown>) => boolean);
  render?: (props: BaseFormFieldProps) => ReactNode;
  
  // Input specific
  maxLength?: number;
  minLength?: number;
  pattern?: string;
  
  // Textarea specific
  rows?: number;
  autoResize?: boolean;
  
  // Select specific
  options?: SelectOption[];
  multiple?: boolean;
  searchable?: boolean;
  clearable?: boolean;
  
  // Radio specific
  direction?: 'horizontal' | 'vertical';
  
  // Date picker specific
  showTime?: boolean;
  showToday?: boolean;
  disabledDate?: (date: Date) => boolean;
}

// 表单模式配置
export interface FormSchema {
  fields: FormFieldConfig[];
  layout?: 'vertical' | 'horizontal' | 'inline';
  labelWidth?: string;
  validateTrigger?: 'onChange' | 'onBlur' | 'onSubmit';
  submitText?: string;
  resetText?: string;
  showReset?: boolean;
  showSubmit?: boolean;
}

// 表单构建器组件属性
export interface FormBuilderProps {
  fields: FormFieldConfig[];
  values?: Record<string, unknown>;
  errors?: Record<string, string | string[]>;
  onChange?: (name: string, value: unknown) => void;
  onSubmit?: (values: Record<string, unknown>) => void | Promise<void>;
  onReset?: () => void;
  loading?: boolean;
  submitText?: string;
  resetText?: string;
  showReset?: boolean;
  layout?: 'vertical' | 'horizontal';
  className?: string;
}