'use client';

import React from 'react';
import { Button, Collapse, Form, Input, InputNumber, Select, Space, Switch } from 'antd';
import type { FormInstance } from 'antd/es/form';

import type { CIType, CloudResource, CloudService } from '@/types/biz/cmdb';

import type { CIFormValues, SchemaField } from './ci-editor-shared';
import {
  buildCloudResourceOptions,
  cloudProviderOptions,
  cloudSyncStatusOptions,
  criticalityOptions,
  environmentOptions,
  getStatusSelectOptions,
  sourceOptions,
} from './ci-editor-shared';

const { TextArea } = Input;

interface CIEditorFormProps {
  form: FormInstance<CIFormValues>;
  types: CIType[];
  typesLoading: boolean;
  cloudResources: CloudResource[];
  cloudServices: CloudService[];
  cloudLoading: boolean;
  schemaFields: SchemaField[];
  typeSchemaFields: SchemaField[];
  saving: boolean;
  submitText: string;
  onSubmit: (values: CIFormValues) => Promise<void> | void;
  onCancel: () => void;
  onCITypeChange: (value?: number) => void;
  onCloudResourceChange: (value?: number) => void;
  /** 表单值变化回调（用于脏数据离开守卫） */
  onValuesChange?: () => void;
}

/** 扩展属性 JSON 实时校验：错误时红字提示，不阻塞输入 */
const jsonValidator = (_: unknown, value: unknown) => {
  if (!value || typeof value !== 'string' || !value.trim()) {
    return Promise.resolve();
  }
  try {
    JSON.parse(value);
    return Promise.resolve();
  } catch {
    return Promise.reject(new Error('JSON 格式不合法，请检查引号、逗号与括号'));
  }
};

const DateTimeSchemaInput = ({
  value,
  onChange,
  label,
}: {
  value?: string;
  onChange?: (value?: string) => void;
  label: string;
}) => {
  let localValue = '';
  if (value) {
    const date = new Date(value);
    if (!Number.isNaN(date.getTime())) {
      const localDate = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
      localValue = localDate.toISOString().slice(0, 16);
    }
  }
  return (
    <Input
      type='datetime-local'
      value={localValue}
      aria-label={label}
      onChange={event => {
        const next = event.target.value;
        onChange?.(next ? new Date(next).toISOString() : undefined);
      }}
    />
  );
};

const renderSchemaFieldInput = (field: SchemaField) => {
  const placeholder = field.placeholder || `请输入${field.label || field.key}`;
  switch ((field.type || 'string').toLowerCase()) {
    case 'select':
    case 'enum':
      return (
        <Select
          placeholder={field.placeholder || `请选择${field.label || field.key}`}
          allowClear
          options={(field.options || []).map(option => ({ label: option, value: option }))}
        />
      );
    case 'number':
    case 'integer':
    case 'int':
    case 'float':
      return (
        <InputNumber
          className='w-full'
          placeholder={placeholder}
          min={field.validation?.minValue}
          max={field.validation?.maxValue}
          precision={field.type === 'integer' || field.type === 'int' ? 0 : undefined}
        />
      );
    case 'boolean':
    case 'bool':
      return <Switch checkedChildren='是' unCheckedChildren='否' />;
    case 'date':
      return <Input type='date' aria-label={field.label || field.key} />;
    case 'datetime':
      return <DateTimeSchemaInput label={field.label || field.key} />;
    case 'text':
      return <TextArea rows={3} placeholder={placeholder} allowClear />;
    default:
      return <Input placeholder={placeholder} allowClear />;
  }
};

export function CIEditorForm({
  form,
  types,
  typesLoading,
  cloudResources,
  cloudServices,
  cloudLoading,
  schemaFields,
  typeSchemaFields,
  saving,
  submitText,
  onSubmit,
  onCancel,
  onCITypeChange,
  onCloudResourceChange,
  onValuesChange,
}: CIEditorFormProps) {
  const cloudServiceMap = React.useMemo(
    () => new Map(cloudServices.map(service => [service.id, service])),
    [cloudServices]
  );

  // 基础信息
  const basicSection = (
    <>
      <div className='grid gap-4 lg:grid-cols-2'>
        <Form.Item
          label='资产名称'
          name='name'
          rules={[{ required: true, message: '请输入资产名称' }]}
        >
          <Input placeholder='请输入资产名称' size='middle' />
        </Form.Item>

        <Form.Item
          label='资产类型'
          name='ciTypeId'
          rules={[{ required: true, message: '请选择资产类型' }]}
        >
          <Select
            placeholder='请选择资产类型'
            loading={typesLoading}
            showSearch
            optionFilterProp='label'
            onChange={onCITypeChange}
            size='middle'
            options={types.map(type => ({
              label: type.name,
              value: type.id,
            }))}
          />
        </Form.Item>

        <Form.Item
          label='状态'
          name='status'
          rules={[{ required: true, message: '请选择资产状态' }]}
        >
          <Select placeholder='请选择资产状态' size='middle' options={getStatusSelectOptions()} />
        </Form.Item>

        <Form.Item label='序列号' name='serialNumber'>
          <Input placeholder='请输入序列号（可选）' />
        </Form.Item>
        <Form.Item label='型号' name='model'>
          <Input placeholder='请输入型号（可选）' />
        </Form.Item>
        <Form.Item label='厂商' name='vendor'>
          <Input placeholder='请输入厂商（可选）' />
        </Form.Item>
        <Form.Item label='资产标签' name='assetTag'>
          <Input placeholder='请输入资产标签（可选）' />
        </Form.Item>
        <Form.Item label='数据来源' name='source'>
          <Select placeholder='请选择数据来源' allowClear options={sourceOptions} />
        </Form.Item>
      </div>

      <Form.Item label='描述' name='description'>
        <TextArea rows={3} placeholder='补充描述信息（可选）' />
      </Form.Item>
    </>
  );

  // 归属与环境
  const ownershipSection = (
    <div className='grid gap-4 lg:grid-cols-2 xl:grid-cols-3'>
      <Form.Item label='环境' name='environment'>
        <Select placeholder='请选择环境' allowClear size='middle' options={environmentOptions} />
      </Form.Item>
      <Form.Item label='重要性' name='criticality'>
        <Select placeholder='请选择重要性' allowClear options={criticalityOptions} />
      </Form.Item>
      <Form.Item label='位置' name='location'>
        <Input placeholder='请输入位置（可选）' />
      </Form.Item>
      <Form.Item label='分配给' name='assignedTo'>
        <Input placeholder='请输入分配人（可选）' />
      </Form.Item>
      <Form.Item label='拥有者' name='ownedBy'>
        <Input placeholder='请输入拥有者（可选）' />
      </Form.Item>
      <Form.Item label='发现源' name='discoverySource'>
        <Input placeholder='请输入发现源（可选）' />
      </Form.Item>
    </div>
  );

  // 云资源
  const cloudSection = (
    <>
      <div className='mb-3 text-sm text-slate-500'>
        如果配置项来自云平台，请先选择「云资源引用」，系统会自动填充
        Region、Zone、资源类型等字段。如不涉及云资源，此部分可留空。
      </div>

      <div className='grid gap-4 lg:grid-cols-2 xl:grid-cols-3'>
        <Form.Item label='云厂商' name='cloudProvider'>
          <Select placeholder='请选择云厂商' allowClear options={cloudProviderOptions} />
        </Form.Item>

        <Form.Item label='云资源引用' name='cloudResourceRefId' className='xl:col-span-2'>
          <Select
            placeholder='请选择云资源（可选）'
            allowClear
            loading={cloudLoading}
            showSearch
            optionFilterProp='label'
            onChange={onCloudResourceChange}
            options={buildCloudResourceOptions(cloudResources, cloudServiceMap)}
          />
        </Form.Item>

        <Form.Item label='云账号 ID' name='cloudAccountId'>
          <Input placeholder='请输入云账号 ID（可选）' />
        </Form.Item>
        <Form.Item label='Region' name='cloudRegion'>
          <Input placeholder='请输入 Region（可选）' />
        </Form.Item>
        <Form.Item label='Zone' name='cloudZone'>
          <Input placeholder='请输入 Zone（可选）' />
        </Form.Item>
        <Form.Item label='云资源 ID' name='cloudResourceId'>
          <Input placeholder='请输入云资源 ID（可选）' />
        </Form.Item>
        <Form.Item label='云资源类型' name='cloudResourceType'>
          <Input placeholder='请输入云资源类型（可选）' />
        </Form.Item>
        <Form.Item label='同步状态' name='cloudSyncStatus'>
          <Select placeholder='请选择同步状态' allowClear options={cloudSyncStatusOptions} />
        </Form.Item>
      </div>

      {schemaFields.length > 0 && (
        <>
          <div className='mb-3 mt-2 text-sm text-slate-500'>
            动态属性会跟随所选云资源类型变化，并保存到云资源元数据中，优先使用枚举选择，减少手填错误。
          </div>
          <div className='grid gap-4 lg:grid-cols-2'>
            {schemaFields.map(field => (
              <Form.Item
                key={field.key}
                label={field.label || field.key}
                name={['cloudMetadata', field.key]}
                rules={
                  field.required
                    ? [{ required: true, message: `请输入${field.label || field.key}` }]
                    : undefined
                }
                valuePropName={
                  ['boolean', 'bool'].includes((field.type || '').toLowerCase())
                    ? 'checked'
                    : 'value'
                }
              >
                {renderSchemaFieldInput(field)}
              </Form.Item>
            ))}
          </div>
        </>
      )}
    </>
  );

  // 扩展属性
  const extensionSection = (
    <>
      {typeSchemaFields.length > 0 && (
        <>
          <div className='mb-3 text-sm text-slate-500'>
            这些字段来自所选 CI
            类型模板，会保存到配置项扩展属性中，用于统一检索、报表和后续流程引用。
          </div>
          <div className='grid gap-4 lg:grid-cols-2'>
            {typeSchemaFields.map(field => (
              <Form.Item
                key={field.key}
                label={field.label || field.key}
                name={['customAttributes', field.key]}
                rules={
                  field.required
                    ? [{ required: true, message: `请选择${field.label || field.key}` }]
                    : undefined
                }
                valuePropName={
                  ['boolean', 'bool'].includes((field.type || '').toLowerCase())
                    ? 'checked'
                    : 'value'
                }
              >
                {renderSchemaFieldInput(field)}
              </Form.Item>
            ))}
          </div>
        </>
      )}

      <Form.Item
        label='扩展属性（JSON）'
        name='attributes'
        rules={[{ validator: jsonValidator }]}
        validateTrigger='onChange'
        extra='自定义扩展属性，JSON 对象格式，例如 {"cpu": "8C", "memory": "32G"}'
      >
        <TextArea rows={5} placeholder='请输入扩展属性 JSON（可选）' />
      </Form.Item>
    </>
  );

  return (
    <Form
      form={form}
      layout='vertical'
      onFinish={onSubmit}
      onValuesChange={onValuesChange}
      className='space-y-1'
    >
      <Collapse
        defaultActiveKey={['basic', 'ownership', 'cloud', 'extension']}
        ghost
        items={[
          {
            key: 'basic',
            forceRender: true,
            label: <span className='font-medium'>基础信息</span>,
            children: basicSection,
          },
          {
            key: 'ownership',
            forceRender: true,
            label: <span className='font-medium'>归属与环境</span>,
            children: ownershipSection,
          },
          {
            key: 'cloud',
            forceRender: true,
            label: <span className='font-medium'>云资源</span>,
            children: cloudSection,
          },
          {
            key: 'extension',
            forceRender: true,
            label: <span className='font-medium'>扩展属性</span>,
            children: extensionSection,
          },
        ]}
      />

      <Space className='mt-4'>
        <Button onClick={onCancel}>取消</Button>
        <Button type='primary' htmlType='submit' loading={saving}>
          {submitText}
        </Button>
      </Space>
    </Form>
  );
}
