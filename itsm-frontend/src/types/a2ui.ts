/**
 * A2UI Protocol Type Definitions
 * Agent-Driven UI Protocol
 * https://a2ui.org/
 */

// ==================== 消息类型 ====================

export interface A2UISurfaceUpdate {
  surfaceUpdate: {
    surfaceId: string;
    components: A2UIComponent[];
  };
}

export interface A2UIDataModelUpdate {
  dataModelUpdate: {
    surfaceId: string;
    path: string;
    contents: A2UIValue[];
  };
}

export interface A2UIBeginRendering {
  beginRendering: {
    surfaceId: string;
    root: string;
  };
}

export interface A2UIDeleteSurface {
  deleteSurface: {
    surfaceId: string;
  };
}

export interface A2UIUserAction {
  userAction: {
    name: string;
    surfaceId: string;
    context: Record<string, unknown>;
  };
}

export type A2UIMessage =
  | A2UISurfaceUpdate
  | A2UIDataModelUpdate
  | A2UIBeginRendering
  | A2UIDeleteSurface
  | A2UIUserAction;

// ==================== 组件定义 ====================

export interface A2UIComponent {
  id: string;
  component: A2UIComponentDef;
}

export type A2UIComponentDef =
  | { Column: ColumnProps }
  | { Row: RowProps }
  | { List: ListProps }
  | { Text: TextProps }
  | { TextField: TextFieldProps }
  | { TextAreaField: TextAreaFieldProps }
  | { PickerSelect: PickerSelectProps }
  | { Button: ButtonProps }
  | { Card: CardProps }
  | { Checkbox: CheckboxProps }
  | { DateTimeInput: DateTimeInputProps };

// 布局组件
export interface ColumnProps {
  children: ChildrenDef;
}

export interface RowProps {
  children: ChildrenDef;
  gap?: number;
}

export interface ListProps {
  children: TemplateDef;
}

export type ChildrenDef =
  | { explicitList: string[] }
  | { template: TemplateDef };

export interface TemplateDef {
  dataBinding: string; // JSON Pointer 路径，如 /products
  componentId: string; // 模板组件 ID
}

// 显示组件
export interface TextProps {
  text: ValueDef;
  usageHint?: 'h1' | 'h2' | 'h3' | 'body' | 'caption';
  color?: ValueDef | string;
  visible?: ValueDef;
}

// 输入组件 - 支持双向绑定
export interface TextFieldProps {
  label?: { literalString: string };
  text: ValueDef; // 读取路径
  placeholder?: { literalString?: string };
  enabled?: ValueDef; // 启用状态
  required?: ValueDef;
  error?: ValueDef; // 错误信息
}

export interface TextAreaFieldProps {
  label?: { literalString: string };
  text: ValueDef;
  rows?: number;
  enabled?: ValueDef;
}

export interface PickerSelectProps {
  label?: { literalString: string };
  selection: ValueDef; // 选中值路径（双向绑定）
  options: OptionsDef;
  enabled?: ValueDef;
}

export interface CheckboxProps {
  label?: { literalString: string };
  value: ValueDef; // 布尔值路径（双向绑定）
  enabled?: ValueDef;
}

export interface DateTimeInputProps {
  label?: { literalString: string };
  value: ValueDef; // 日期值路径（双向绑定）
  enabled?: ValueDef;
}

// 按钮组件
export interface ButtonProps {
  child: string;
  action?: ActionDef;
  variant?: 'filled' | 'outlined' | 'text';
  enabled?: ValueDef; // 按钮启用状态
}

// 容器组件
export interface CardProps {
  child: string;
  elevation?: number;
}

// ==================== 值定义（核心）====================

// 值可以是字面量或数据绑定路径
export type ValueDef =
  | { literalString: string }
  | { literalNumber: number }
  | { literalBoolean: boolean }
  | { path: string }; // JSON Pointer 路径

// 选项定义
export type OptionsDef =
  | { explicitList: OptionItem[] }
  | { dataBinding: string }; // 从数据模型加载选项

export interface OptionItem {
  id: string;
  text: string;
  disabled?: boolean;
}

// 动作定义
export interface ActionDef {
  name: string;
  context: { key: string; value: ValueDef }[];
}

// 值内容
export interface A2UIValue {
  key: string;
  valueString?: string;
  valueNumber?: number;
  valueBoolean?: boolean;
}

// ==================== 数据模型工具函数 ====================

/**
 * 解析 JSON Pointer 路径
 * /ticket/title -> { ticket: { title: value } }
 */
export function getValueByPath(model: Record<string, unknown>, path: string): unknown {
  if (!path.startsWith('/')) return undefined;

  const segments = path.split('/').filter(Boolean);
  let current: unknown = model;

  for (const segment of segments) {
    if (current === null || current === undefined) return undefined;

    // 数组索引
    if (/^\d+$/.test(segment)) {
      current = (current as unknown[])[parseInt(segment, 10)];
    } else {
      current = (current as Record<string, unknown>)[segment];
    }
  }

  return current;
}

/**
 * 设置值到 JSON Pointer 路径
 */
export function setValueByPath(model: Record<string, unknown>, path: string, value: unknown): void {
  if (!path.startsWith('/')) return;

  const segments = path.split('/').filter(Boolean);
  let current: Record<string, unknown> = model;

  for (let i = 0; i < segments.length - 1; i++) {
    const segment = segments[i];

    if (!(segment in current)) {
      // 创建中间路径
      const nextSegment = segments[i + 1];
      current[segment] = /^\d+$/.test(nextSegment) ? [] : {};
    }
    current = current[segment] as Record<string, unknown>;
  }

  const lastSegment = segments[segments.length - 1];
  current[lastSegment] = value;
}

/**
 * 解析值定义（字面量或路径）
 */
export function resolveValue(def: ValueDef | undefined, model: Record<string, unknown>): unknown {
  if (!def) return undefined;

  if ('literalString' in def) return def.literalString;
  if ('literalNumber' in def) return def.literalNumber;
  if ('literalBoolean' in def) return def.literalBoolean;
  if ('path' in def) return getValueByPath(model, def.path);

  return undefined;
}

// ==================== A2UI 服务类型 ====================

export interface A2UIFormRequest {
  intent: string;
  surfaceId?: string;
}

export interface A2UIFormResponse {
  messages: string[];
}

export interface A2UIActionRequest {
  action: string;
  surfaceId: string;
  context: Record<string, unknown>;
}

export interface A2UIActionResponse {
  messages: string[];
  success: boolean;
}

// ==================== 组件类型映射 ====================

export const A2UI_COMPONENT_TYPES = {
  LAYOUT: ['Column', 'Row', 'List'],
  DISPLAY: ['Text', 'Image', 'Icon', 'Divider'],
  INTERACTIVE: ['Button', 'TextField', 'TextAreaField', 'PickerSelect', 'Checkbox', 'DateTimeInput'],
  CONTAINER: ['Card', 'Tabs', 'Modal'],
} as const;

// 组件渲染映射（用于前端渲染）
export interface ComponentRenderMap {
  [key: string]: string; // A2UI 组件名 -> Ant Design 组件名
}

export const A2UI_TO_ANTD_MAP: ComponentRenderMap = {
  Text: 'Text',
  TextField: 'Input',
  TextAreaField: 'Input.TextArea',
  PickerSelect: 'Select',
  Button: 'Button',
  Checkbox: 'Input.Checkbox',
  DateTimeInput: 'DatePicker',
  Card: 'Card',
  Column: 'Flex',
  Row: 'Flex',
};
