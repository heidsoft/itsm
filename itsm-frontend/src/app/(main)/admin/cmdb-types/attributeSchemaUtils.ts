// CI 类型属性模板（attribute schema）构建/解析/校验工具
// 独立于页面组件，便于复用与单元测试

export type AttributeFieldType = 'string' | 'number' | 'boolean' | 'date' | 'select';

export type AttributeTemplateField = {
  key?: string;
  label?: string;
  type?: AttributeFieldType;
  options?: string;
  required?: boolean;
  placeholder?: string;
};

export const ATTRIBUTE_FIELD_TYPES: AttributeFieldType[] = [
  'string',
  'number',
  'boolean',
  'date',
  'select',
];

export const ATTRIBUTE_FIELD_TYPE_OPTIONS: { value: AttributeFieldType; label: string }[] = [
  { value: 'string', label: '文本' },
  { value: 'number', label: '数字' },
  { value: 'boolean', label: '布尔' },
  { value: 'date', label: '日期' },
  { value: 'select', label: '枚举选择' },
];

export const DEFAULT_ATTRIBUTE_SCHEMA = JSON.stringify(
  {
    fields: [
      {
        key: 'environment',
        label: '环境',
        type: 'select',
        options: ['production', 'staging', 'development'],
        required: true,
      },
      {
        key: 'owner',
        label: '负责人',
        type: 'select',
        options: ['ops', 'platform', 'security'],
      },
    ],
  },
  null,
  2
);

const isAttributeFieldType = (value: unknown): value is AttributeFieldType =>
  typeof value === 'string' && (ATTRIBUTE_FIELD_TYPES as string[]).includes(value);

export const isAttributeSchemaSafelyEditable = (value?: string): boolean => {
  if (!value || !value.trim()) return true;
  try {
    const parsed = JSON.parse(value);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return false;
    const fields = (parsed as Record<string, unknown>).fields;
    if (!Array.isArray(fields)) return false;
    const allowedKeys = new Set([
      'key',
      'name',
      'label',
      'type',
      'options',
      'required',
      'placeholder',
    ]);
    return fields.every(field => {
      if (!field || typeof field !== 'object' || Array.isArray(field)) return false;
      const entry = field as Record<string, unknown>;
      return (
        Object.keys(entry).every(key => allowedKeys.has(key)) &&
        (entry.type === undefined || isAttributeFieldType(entry.type))
      );
    });
  } catch {
    return false;
  }
};

export const validateAttributeSchema = (value?: string): string | null => {
  if (!value || !value.trim()) {
    return null;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(value);
  } catch {
    return '请输入合法的 JSON';
  }

  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return '属性模式必须是对象 JSON';
  }

  const record = parsed as Record<string, unknown>;
  if (record.fields === undefined) {
    return null;
  }

  if (!Array.isArray(record.fields)) {
    return 'attribute_schema.fields 必须是数组';
  }

  for (let index = 0; index < record.fields.length; index += 1) {
    const field = record.fields[index];
    if (!field || typeof field !== 'object' || Array.isArray(field)) {
      return `attribute_schema.fields[${index}] 必须是对象`;
    }
    const entry = field as Record<string, unknown>;
    const fieldKey = entry.key || entry.name || `字段${index + 1}`;
    const fieldType = entry.type ?? 'string';
    if (!isAttributeFieldType(fieldType)) {
      return `attribute_schema.fields[${index}]（${fieldKey}）type 仅支持 ${ATTRIBUTE_FIELD_TYPES.join('/')}`;
    }
    if (fieldType === 'select' && (!Array.isArray(entry.options) || entry.options.length === 0)) {
      return `attribute_schema.fields[${index}]（${fieldKey}）type=select 时必须提供非空 options`;
    }
  }

  return null;
};

export const normalizeAttributeTemplateFields = (fields?: AttributeTemplateField[]) =>
  (fields || [])
    .map(field => {
      const type: AttributeFieldType = isAttributeFieldType(field.type) ? field.type : 'string';
      return {
        key: field.key?.trim(),
        label: field.label?.trim(),
        type,
        required: Boolean(field.required),
        placeholder: field.placeholder?.trim(),
        options:
          type === 'select'
            ? (field.options || '')
                .split(/[\n,，]/)
                .map(option => option.trim())
                .filter(Boolean)
            : [],
      };
    })
    .filter(field => field.key || field.label || field.options.length > 0);

export const buildAttributeSchemaFromFields = (fields?: AttributeTemplateField[]) => {
  const normalizedFields = normalizeAttributeTemplateFields(fields);
  if (normalizedFields.length === 0) {
    return '';
  }

  return JSON.stringify(
    {
      fields: normalizedFields.map(field => ({
        key: field.key,
        label: field.label || field.key,
        type: field.type,
        ...(field.type === 'select' ? { options: field.options } : {}),
        required: field.required,
        ...(field.placeholder ? { placeholder: field.placeholder } : {}),
      })),
    },
    null,
    2
  );
};

export const parseAttributeSchemaToFields = (schemaText?: string): AttributeTemplateField[] => {
  if (!schemaText || !schemaText.trim()) {
    return [];
  }

  try {
    const parsed = JSON.parse(schemaText);
    const fields = Array.isArray(parsed?.fields) ? parsed.fields : [];
    return fields
      .filter((field: unknown) => field && typeof field === 'object' && !Array.isArray(field))
      .map((field: Record<string, unknown>) => {
        const type: AttributeFieldType = isAttributeFieldType(field.type) ? field.type : 'string';
        return {
          key:
            typeof field.key === 'string'
              ? field.key
              : typeof field.name === 'string'
                ? field.name
                : '',
          label: typeof field.label === 'string' ? field.label : '',
          type,
          options:
            type === 'select' && Array.isArray(field.options)
              ? field.options
                  .filter((option): option is string => typeof option === 'string')
                  .join('\n')
              : '',
          required: Boolean(field.required),
          placeholder: typeof field.placeholder === 'string' ? field.placeholder : '',
        };
      });
  } catch {
    return [];
  }
};

export const getAttributeSchemaFieldCount = (value?: string) => {
  if (!value || !value.trim()) {
    return 0;
  }

  try {
    const parsed = JSON.parse(value);
    if (Array.isArray(parsed?.fields)) {
      return parsed.fields.length;
    }
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return Object.keys(parsed).length;
    }
  } catch {
    return 0;
  }

  return 0;
};
