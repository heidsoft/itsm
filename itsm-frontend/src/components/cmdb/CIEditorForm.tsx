'use client';

import React from 'react';
import { Button, Collapse, Form, Input, InputNumber, Select, Space, Switch } from 'antd';
import type { FormInstance } from 'antd/es/form';

import type { CIType, CloudResource, CloudService } from '@/types/biz/cmdb';

import type { CIFormValues, SchemaField, TranslationFn } from './ci-editor-shared';
import {
  buildCloudProviderOptions,
  buildCloudResourceOptions,
  buildCloudSyncStatusOptions,
  buildCriticalityOptions,
  buildEnvironmentOptions,
  buildSourceOptions,
  getStatusSelectOptions,
} from './ci-editor-shared';
import { useI18n } from '@/lib/i18n/useI18n';

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
const jsonValidator = (t: TranslationFn) => (_: unknown, value: unknown) => {
  if (!value || typeof value !== 'string' || !value.trim()) {
    return Promise.resolve();
  }
  try {
    JSON.parse(value);
    return Promise.resolve();
  } catch {
    return Promise.reject(new Error(t('ciEditor.jsonInvalid')));
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
  const { t } = useI18n();
  const cloudServiceMap = React.useMemo(
    () => new Map(cloudServices.map(service => [service.id, service])),
    [cloudServices]
  );

  const renderSchemaFieldInput = React.useCallback(
    (field: SchemaField) => {
      const labelText = field.label || field.key;
      const placeholder = field.placeholder || t('ciEditor.placeholderInput', { label: labelText });
      switch ((field.type || 'string').toLowerCase()) {
        case 'select':
        case 'enum':
          return (
            <Select
              placeholder={field.placeholder || t('ciEditor.placeholderSelect', { label: labelText })}
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
          return <Switch checkedChildren={t('ciEditor.boolYes')} unCheckedChildren={t('ciEditor.boolNo')} />;
        case 'date':
          return <Input type='date' aria-label={field.label || field.key} />;
        case 'datetime':
          return <DateTimeSchemaInput label={field.label || field.key} />;
        case 'text':
          return <TextArea rows={3} placeholder={placeholder} allowClear />;
        default:
          return <Input placeholder={placeholder} allowClear />;
      }
    },
    [t]
  );

  // 基础信息
  const basicSection = (
    <>
      <div className='grid gap-4 lg:grid-cols-2'>
        <Form.Item
          label={t('ciEditor.assetName')}
          name='name'
          rules={[{ required: true, message: t('ciEditor.assetNameRequired') }]}
        >
          <Input placeholder={t('ciEditor.assetNameRequired')} size='middle' />
        </Form.Item>

        <Form.Item
          label={t('ciEditor.assetType')}
          name='ciTypeId'
          rules={[{ required: true, message: t('ciEditor.assetTypeRequired') }]}
        >
          <Select
            placeholder={t('ciEditor.assetTypeRequired')}
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
          label={t('ciEditor.status')}
          name='status'
          rules={[{ required: true, message: t('ciEditor.statusRequired') }]}
        >
          <Select placeholder={t('ciEditor.statusRequired')} size='middle' options={getStatusSelectOptions(t)} />
        </Form.Item>

        <Form.Item label={t('ciEditor.serialNumber')} name='serialNumber'>
          <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.serialNumber') })} />
        </Form.Item>
        <Form.Item label={t('ciEditor.model')} name='model'>
          <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.model') })} />
        </Form.Item>
        <Form.Item label={t('ciEditor.vendor')} name='vendor'>
          <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.vendor') })} />
        </Form.Item>
        <Form.Item label={t('ciEditor.assetTag')} name='assetTag'>
          <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.assetTag') })} />
        </Form.Item>
        <Form.Item label={t('ciEditor.dataSource')} name='source'>
          <Select placeholder={t('ciEditor.placeholderOptionalSelect', { label: t('ciEditor.dataSource') })} allowClear options={buildSourceOptions(t)} />
        </Form.Item>
      </div>

      <Form.Item label={t('ciEditor.description')} name='description'>
        <TextArea rows={3} placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.description') })} />
      </Form.Item>
    </>
  );

  // 归属与环境
  const ownershipSection = (
    <div className='grid gap-4 lg:grid-cols-2 xl:grid-cols-3'>
      <Form.Item label={t('ciEditor.environment')} name='environment'>
        <Select placeholder={t('ciEditor.placeholderOptionalSelect', { label: t('ciEditor.environment') })} allowClear size='middle' options={buildEnvironmentOptions(t)} />
      </Form.Item>
      <Form.Item label={t('ciEditor.criticality')} name='criticality'>
        <Select placeholder={t('ciEditor.placeholderOptionalSelect', { label: t('ciEditor.criticality') })} allowClear options={buildCriticalityOptions(t)} />
      </Form.Item>
      <Form.Item label={t('ciEditor.location')} name='location'>
        <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.location') })} />
      </Form.Item>
      <Form.Item label={t('ciEditor.assignedTo')} name='assignedTo'>
        <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.assignedTo') })} />
      </Form.Item>
      <Form.Item label={t('ciEditor.ownedBy')} name='ownedBy'>
        <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.ownedBy') })} />
      </Form.Item>
      <Form.Item label={t('ciEditor.discoverySource')} name='discoverySource'>
        <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.discoverySource') })} />
      </Form.Item>
    </div>
  );

  // 云资源
  const cloudSection = (
    <>
      <div className='mb-3 text-sm text-slate-500'>
        {t('ciEditor.cloudResourceHelp')}
      </div>

      <div className='grid gap-4 lg:grid-cols-2 xl:grid-cols-3'>
        <Form.Item label={t('ciEditor.cloudProvider')} name='cloudProvider'>
          <Select placeholder={t('ciEditor.placeholderOptionalSelect', { label: t('ciEditor.cloudProvider') })} allowClear options={buildCloudProviderOptions(t)} />
        </Form.Item>

        <Form.Item label={t('ciEditor.cloudResourceRef')} name='cloudResourceRefId' className='xl:col-span-2'>
          <Select
            placeholder={t('ciEditor.cloudResourcePlaceholder')}
            allowClear
            loading={cloudLoading}
            onChange={onCloudResourceChange}
            options={buildCloudResourceOptions(cloudResources, cloudServiceMap, t)}
          />
        </Form.Item>

        <Form.Item label={t('ciEditor.cloudAccountId')} name='cloudAccountId'>
          <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.cloudAccountId') })} />
        </Form.Item>
        <Form.Item label={t('ciEditor.cloudRegion')} name='cloudRegion'>
          <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.cloudRegion') })} />
        </Form.Item>
        <Form.Item label={t('ciEditor.cloudZone')} name='cloudZone'>
          <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.cloudZone') })} />
        </Form.Item>
        <Form.Item label={t('ciEditor.cloudResourceId')} name='cloudResourceId'>
          <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.cloudResourceId') })} />
        </Form.Item>
        <Form.Item label={t('ciEditor.cloudResourceType')} name='cloudResourceType'>
          <Input placeholder={t('ciEditor.placeholderOptionalInput', { label: t('ciEditor.cloudResourceType') })} />
        </Form.Item>
        <Form.Item label={t('ciEditor.cloudSyncStatus')} name='cloudSyncStatus'>
          <Select placeholder={t('ciEditor.placeholderOptionalSelect', { label: t('ciEditor.cloudSyncStatus') })} allowClear options={buildCloudSyncStatusOptions(t)} />
        </Form.Item>
      </div>

      {schemaFields.length > 0 && (
        <>
          <div className='mb-3 mt-2 text-sm text-slate-500'>
            {t('ciEditor.dynamicAttributesHelp')}
          </div>
          <div className='grid gap-4 lg:grid-cols-2'>
            {schemaFields.map(field => (
              <Form.Item
                key={field.key}
                label={field.label || field.key}
                name={['cloudMetadata', field.key]}
                rules={
                  field.required
                    ? [
                        {
                          required: true,
                          message: t('ciEditor.required', { label: field.label || field.key }),
                        },
                      ]
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
            {t('ciEditor.typeSchemaFieldsHelp')}
          </div>
          <div className='grid gap-4 lg:grid-cols-2'>
            {typeSchemaFields.map(field => (
              <Form.Item
                key={field.key}
                label={field.label || field.key}
                name={['customAttributes', field.key]}
                rules={
                  field.required
                    ? [
                        {
                          required: true,
                          message: t('ciEditor.required', { label: field.label || field.key }),
                        },
                      ]
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
        label={t('ciEditor.extensionAttributesJson')}
        name='attributes'
        rules={[{ validator: jsonValidator(t) }]}
        validateTrigger='onChange'
        extra={t('ciEditor.extensionAttributesExtra')}
      >
        <TextArea rows={5} placeholder={t('ciEditor.extensionAttributesPlaceholder')} />
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
            label: <span className='font-medium'>{t('ciEditor.basicInfo')}</span>,
            children: basicSection,
          },
          {
            key: 'ownership',
            forceRender: true,
            label: <span className='font-medium'>{t('ciEditor.ownershipEnvironment')}</span>,
            children: ownershipSection,
          },
          {
            key: 'cloud',
            forceRender: true,
            label: <span className='font-medium'>{t('ciEditor.cloudResources')}</span>,
            children: cloudSection,
          },
          {
            key: 'extension',
            forceRender: true,
            label: <span className='font-medium'>{t('ciEditor.extensionAttributes')}</span>,
            children: extensionSection,
          },
        ]}
      />

      <Space className='mt-4'>
        <Button onClick={onCancel}>{t('ciEditor.cancel')}</Button>
        <Button type='primary' htmlType='submit' loading={saving}>
          {submitText}
        </Button>
      </Space>
    </Form>
  );
}
