'use client';

import React from 'react';
import { Button, Divider, Form, Input, Select, Space } from 'antd';
import type { FormInstance } from 'antd/es/form';

import type { CIType, CloudResource, CloudService } from '@/types/biz/cmdb';

import type {
  CIFormValues,
  SchemaField} from './ci-editor-shared';
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
}

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
}: CIEditorFormProps) {
  const cloudServiceMap = React.useMemo(
    () => new Map(cloudServices.map(service => [service.id, service])),
    [cloudServices]
  );

  return (
    <Form form={form} layout="vertical" onFinish={onSubmit} className="space-y-1">
      <div className="grid gap-4 lg:grid-cols-2">
        <Form.Item
          label="资产名称"
          name="name"
          rules={[{ required: true, message: '请输入资产名称' }]}
        >
          <Input placeholder="请输入资产名称" />
        </Form.Item>

        <Form.Item
          label="资产类型"
          name="ciTypeId"
          rules={[{ required: true, message: '请选择资产类型' }]}
        >
          <Select
            placeholder="请选择资产类型"
            loading={typesLoading}
            showSearch
            optionFilterProp="label"
            onChange={onCITypeChange}
            options={types.map(type => ({
              label: type.name,
              value: type.id,
            }))}
          />
        </Form.Item>

        <Form.Item
          label="状态"
          name="status"
          rules={[{ required: true, message: '请选择资产状态' }]}
        >
          <Select placeholder="请选择资产状态" options={getStatusSelectOptions()} />
        </Form.Item>

        <Form.Item label="环境" name="environment">
          <Select placeholder="请选择环境" allowClear options={environmentOptions} />
        </Form.Item>
      </div>

      <Form.Item label="描述" name="description">
        <TextArea rows={4} placeholder="补充描述信息（可选）" />
      </Form.Item>

      <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
        <Form.Item label="序列号" name="serialNumber">
          <Input placeholder="请输入序列号（可选）" />
        </Form.Item>
        <Form.Item label="型号" name="model">
          <Input placeholder="请输入型号（可选）" />
        </Form.Item>
        <Form.Item label="厂商" name="vendor">
          <Input placeholder="请输入厂商（可选）" />
        </Form.Item>
        <Form.Item label="位置" name="location">
          <Input placeholder="请输入位置（可选）" />
        </Form.Item>
        <Form.Item label="资产标签" name="assetTag">
          <Input placeholder="请输入资产标签（可选）" />
        </Form.Item>
        <Form.Item label="重要性" name="criticality">
          <Select placeholder="请选择重要性" allowClear options={criticalityOptions} />
        </Form.Item>
        <Form.Item label="分配给" name="assignedTo">
          <Input placeholder="请输入分配人（可选）" />
        </Form.Item>
        <Form.Item label="拥有者" name="ownedBy">
          <Input placeholder="请输入拥有者（可选）" />
        </Form.Item>
        <Form.Item label="发现源" name="discoverySource">
          <Input placeholder="请输入发现源（可选）" />
        </Form.Item>
      </div>

      <Form.Item label="数据来源" name="source">
        <Select placeholder="请选择数据来源" allowClear options={sourceOptions} />
      </Form.Item>

      {typeSchemaFields.length > 0 && (
        <>
          <Divider>类型扩展属性</Divider>
          <div className="mb-3 text-sm text-slate-500">
            这些字段来自所选 CI 类型模板，会保存到配置项扩展属性中，用于统一检索、报表和后续流程引用。
          </div>
          <div className="grid gap-4 lg:grid-cols-2">
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
              >
                {field.type === 'select' ? (
                  <Select
                    placeholder={field.placeholder || `请选择${field.label || field.key}`}
                    allowClear
                    options={(field.options || []).map(option => ({
                      label: option,
                      value: option,
                    }))}
                  />
                ) : (
                  <Input placeholder={field.placeholder || `请输入${field.label || field.key}`} allowClear />
                )}
              </Form.Item>
            ))}
          </div>
        </>
      )}

      <Divider>云资源信息</Divider>

      <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
        <Form.Item label="云厂商" name="cloudProvider">
          <Select placeholder="请选择云厂商" allowClear options={cloudProviderOptions} />
        </Form.Item>

        <Form.Item label="云资源引用" name="cloudResourceRefId" className="xl:col-span-2">
          <Select
            placeholder="请选择云资源（可选）"
            allowClear
            loading={cloudLoading}
            showSearch
            optionFilterProp="label"
            onChange={onCloudResourceChange}
            options={buildCloudResourceOptions(cloudResources, cloudServiceMap)}
          />
        </Form.Item>

        <Form.Item label="云账号 ID" name="cloudAccountId">
          <Input placeholder="请输入云账号 ID（可选）" />
        </Form.Item>
        <Form.Item label="Region" name="cloudRegion">
          <Input placeholder="请输入 Region（可选）" />
        </Form.Item>
        <Form.Item label="Zone" name="cloudZone">
          <Input placeholder="请输入 Zone（可选）" />
        </Form.Item>
        <Form.Item label="云资源 ID" name="cloudResourceId">
          <Input placeholder="请输入云资源 ID（可选）" />
        </Form.Item>
        <Form.Item label="云资源类型" name="cloudResourceType">
          <Input placeholder="请输入云资源类型（可选）" />
        </Form.Item>
        <Form.Item label="同步状态" name="cloudSyncStatus">
          <Select placeholder="请选择同步状态" allowClear options={cloudSyncStatusOptions} />
        </Form.Item>
      </div>

      {schemaFields.length > 0 && (
        <>
          <Divider>云资源动态属性</Divider>
          <div className="mb-3 text-sm text-slate-500">
            动态属性会跟随所选云资源类型变化，并保存到云资源元数据中，优先使用枚举选择，减少手填错误。
          </div>
          <div className="grid gap-4 lg:grid-cols-2">
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
              >
                {field.type === 'select' ? (
                  <Select
                    placeholder={field.placeholder || `请选择${field.label || field.key}`}
                    allowClear
                    options={(field.options || []).map(option => ({
                      label: option,
                      value: option,
                    }))}
                  />
                ) : (
                  <Input placeholder={field.placeholder || `请输入${field.label || field.key}`} />
                )}
              </Form.Item>
            ))}
          </div>
        </>
      )}

      <Form.Item label="扩展属性" name="attributes">
        <TextArea rows={5} placeholder="请输入扩展属性 JSON（可选）" />
      </Form.Item>

      <Space>
        <Button onClick={onCancel}>取消</Button>
        <Button type="primary" htmlType="submit" loading={saving}>
          {submitText}
        </Button>
      </Space>
    </Form>
  );
}
