'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Breadcrumb, Button, Card, Divider, Form, Input, Select, Space, message } from 'antd';

import { CMDBApi } from '@/modules/cmdb/api';
import { CIStatus, CIStatusLabels } from '@/modules/cmdb/constants';
import type { CIType, CloudResource, CloudService } from '@/modules/cmdb/types';

const { TextArea } = Input;

const statusOptions = [
  CIStatus.ACTIVE,
  CIStatus.INACTIVE,
  CIStatus.MAINTENANCE,
];

type SchemaField = {
  key: string;
  label?: string;
  type?: string;
  required?: boolean;
  options?: string[];
  placeholder?: string;
};

const normalizeSchemaFields = (schema: unknown): SchemaField[] => {
  if (!schema) return [];
  if (Array.isArray(schema)) {
    return schema
      .map((item) => {
        if (typeof item !== 'object' || item === null) return null;
        const record = item as Record<string, any>;
        const key = record.key || record.name;
        if (!key) return null;
        return {
          key,
          label: record.label || key,
          type: record.type,
          required: Boolean(record.required),
          options: Array.isArray(record.options) ? record.options : undefined,
          placeholder: record.placeholder,
        };
      })
      .filter((item): item is SchemaField => Boolean(item));
  }
  if (typeof schema === 'object') {
    const record = schema as Record<string, any>;
    if (Array.isArray(record.fields)) {
      return normalizeSchemaFields(record.fields);
    }
    return Object.entries(record)
      .map(([key, value]) => {
        if (typeof value === 'string') {
          return { key, label: value };
        }
        if (typeof value === 'object' && value !== null) {
          const entry = value as Record<string, any>;
          return {
            key,
            label: entry.label || key,
            type: entry.type,
            required: Boolean(entry.required),
            options: Array.isArray(entry.options) ? entry.options : undefined,
            placeholder: entry.placeholder,
          };
        }
        return { key, label: key };
      });
  }
  return [];
};

const CreateCIPage: React.FC = () => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [form] = Form.useForm();
  const [types, setTypes] = useState<CIType[]>([]);
  const [typesLoading, setTypesLoading] = useState(true);
  const [cloudResources, setCloudResources] = useState<CloudResource[]>([]);
  const [cloudServices, setCloudServices] = useState<CloudService[]>([]);
  const [cloudLoading, setCloudLoading] = useState(true);
  const [schemaFields, setSchemaFields] = useState<SchemaField[]>([]);
  const [saving, setSaving] = useState(false);

  const cloudServiceMap = useMemo(() => {
    return new Map(cloudServices.map((service) => [service.id, service]));
  }, [cloudServices]);

  useEffect(() => {
    const loadTypes = async () => {
      try {
        const res = await CMDBApi.getTypes();
        setTypes(res || []);
      } catch (error) {
        message.error('加载资产类型失败');
      } finally {
        setTypesLoading(false);
      }
    };
    loadTypes();
  }, []);

  useEffect(() => {
    const loadCloudData = async () => {
      setCloudLoading(true);
      try {
        const [resources, services] = await Promise.all([
          CMDBApi.getCloudResources(),
          CMDBApi.getCloudServices(),
        ]);
        setCloudResources(resources || []);
        setCloudServices(services || []);
      } catch (error) {
        message.error('加载云资源数据失败');
      } finally {
        setCloudLoading(false);
      }
    };
    loadCloudData();
  }, []);

  useEffect(() => {
    if (!cloudResources.length || !cloudServices.length) return;
    const resourceRefId = searchParams.get('cloud_resource_ref_id');
    if (!resourceRefId) return;
    const parsed = Number(resourceRefId);
    if (Number.isNaN(parsed)) return;
    form.setFieldsValue({ cloud_resource_ref_id: parsed });
    const resource = cloudResources.find((item) => item.id === parsed);
    if (!resource) return;
    const service = cloudServiceMap.get(resource.service_id);
    setSchemaFields(normalizeSchemaFields(service?.attribute_schema));
    form.setFieldsValue({
      cloud_resource_id: resource.resource_id,
      cloud_region: resource.region,
      cloud_zone: resource.zone,
      cloud_account_id: String(resource.cloud_account_id),
      cloud_provider: service?.provider,
      cloud_resource_type: service?.resource_type_code,
    });
  }, [cloudResources, cloudServices, cloudServiceMap, form, searchParams]);

  const handleCloudResourceChange = (value?: number) => {
    if (!value) {
      setSchemaFields([]);
      return;
    }
    const resource = cloudResources.find((item) => item.id === value);
    const service = resource ? cloudServiceMap.get(resource.service_id) : undefined;
    setSchemaFields(normalizeSchemaFields(service?.attribute_schema));
    if (!resource) return;
    form.setFieldsValue({
      cloud_resource_id: resource.resource_id,
      cloud_region: resource.region,
      cloud_zone: resource.zone,
      cloud_account_id: String(resource.cloud_account_id),
      cloud_provider: service?.provider,
      cloud_resource_type: service?.resource_type_code,
    });
  };

  const handleSubmit = async (values: {
    name: string;
    ci_type_id: number;
    status: CIStatus;
    description?: string;
    attributes?: string;
    serial_number?: string;
    model?: string;
    vendor?: string;
    location?: string;
    asset_tag?: string;
    assigned_to?: string;
    owned_by?: string;
    environment?: string;
    criticality?: string;
    discovery_source?: string;
    source?: string;
    cloud_provider?: string;
    cloud_account_id?: string;
    cloud_region?: string;
    cloud_zone?: string;
    cloud_resource_id?: string;
    cloud_resource_type?: string;
    cloud_sync_status?: string;
    cloud_resource_ref_id?: number;
    cloud_metadata?: Record<string, any>;
  }) => {
    try {
      let attributes: Record<string, any> | undefined;
      if (values.attributes) {
        try {
          attributes =
            typeof values.attributes === 'string'
              ? JSON.parse(values.attributes)
              : values.attributes;
        } catch (parseError) {
          message.error('扩展属性需要是有效的 JSON');
          return;
        }
      }
      setSaving(true);
      await CMDBApi.createCI({
        name: values.name,
        ci_type_id: values.ci_type_id,
        status: values.status,
        description: values.description,
        attributes,
        serial_number: values.serial_number,
        model: values.model,
        vendor: values.vendor,
        location: values.location,
        asset_tag: values.asset_tag,
        assigned_to: values.assigned_to,
        owned_by: values.owned_by,
        environment: values.environment,
        criticality: values.criticality,
        discovery_source: values.discovery_source,
        source: values.source,
        cloud_provider: values.cloud_provider,
        cloud_account_id: values.cloud_account_id,
        cloud_region: values.cloud_region,
        cloud_zone: values.cloud_zone,
        cloud_resource_id: values.cloud_resource_id,
        cloud_resource_type: values.cloud_resource_type,
        cloud_sync_status: values.cloud_sync_status,
        cloud_resource_ref_id: values.cloud_resource_ref_id,
        cloud_metadata: values.cloud_metadata,
      });
      message.success('配置项创建成功');
      router.push('/cmdb');
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '创建配置项失败');
      } else {
        message.error('创建配置项失败');
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card variant="borderless">
      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: '首页' },
          { title: '配置管理' },
          { title: <a onClick={() => router.push('/cmdb')}>配置项列表</a> },
          { title: '录入资产' },
        ]}
      />

      <Form form={form} layout="vertical" onFinish={handleSubmit}>
        <Form.Item
          label="资产名称"
          name="name"
          rules={[{ required: true, message: '请输入资产名称' }]}
        >
          <Input placeholder="请输入资产名称" />
        </Form.Item>

        <Form.Item
          label="资产类型"
          name="ci_type_id"
          rules={[{ required: true, message: '请选择资产类型' }]}
        >
          <Select placeholder="请选择资产类型" loading={typesLoading}>
            {types.map((type) => (
              <Select.Option key={type.id} value={type.id}>
                {type.name}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item
          label="状态"
          name="status"
          rules={[{ required: true, message: '请选择资产状态' }]}
        >
          <Select placeholder="请选择资产状态">
            {statusOptions.map((status) => (
              <Select.Option key={status} value={status}>
                {CIStatusLabels[status]}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item label="描述" name="description">
          <TextArea rows={4} placeholder="补充描述信息（可选）" />
        </Form.Item>

        <Form.Item label="序列号" name="serial_number">
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

        <Form.Item label="资产标签" name="asset_tag">
          <Input placeholder="请输入资产标签（可选）" />
        </Form.Item>

        <Form.Item label="分配给" name="assigned_to">
          <Input placeholder="请输入分配人（可选）" />
        </Form.Item>

        <Form.Item label="拥有者" name="owned_by">
          <Input placeholder="请输入拥有者（可选）" />
        </Form.Item>

        <Form.Item label="环境" name="environment">
          <Select placeholder="请选择环境" allowClear>
            <Select.Option value="production">生产</Select.Option>
            <Select.Option value="staging">预发布</Select.Option>
            <Select.Option value="development">开发</Select.Option>
          </Select>
        </Form.Item>

        <Form.Item label="重要性" name="criticality">
          <Select placeholder="请选择重要性" allowClear>
            <Select.Option value="low">低</Select.Option>
            <Select.Option value="medium">中</Select.Option>
            <Select.Option value="high">高</Select.Option>
            <Select.Option value="critical">关键</Select.Option>
          </Select>
        </Form.Item>

        <Form.Item label="发现源" name="discovery_source">
          <Input placeholder="请输入发现源（可选）" />
        </Form.Item>

        <Form.Item label="数据来源" name="source">
          <Select placeholder="请选择数据来源" allowClear>
            <Select.Option value="manual">手工录入</Select.Option>
            <Select.Option value="discovery">自动发现</Select.Option>
            <Select.Option value="import">批量导入</Select.Option>
          </Select>
        </Form.Item>

        <Divider orientation="left">云资源信息</Divider>

        <Form.Item label="云厂商" name="cloud_provider">
          <Select placeholder="请选择云厂商" allowClear>
            <Select.Option value="aliyun">阿里云</Select.Option>
            <Select.Option value="huawei">华为云</Select.Option>
            <Select.Option value="tencent">腾讯云</Select.Option>
            <Select.Option value="azure">Azure</Select.Option>
            <Select.Option value="onprem">私有云</Select.Option>
          </Select>
        </Form.Item>

        <Form.Item label="云资源引用" name="cloud_resource_ref_id">
          <Select
            placeholder="请选择云资源（可选）"
            allowClear
            loading={cloudLoading}
            showSearch
            optionFilterProp="label"
            onChange={handleCloudResourceChange}
          >
            {cloudResources.map((resource) => {
              const service = cloudServiceMap.get(resource.service_id);
              const label = `${resource.resource_name || resource.resource_id}（${service?.resource_type_name || '未知类型'}）`;
              return (
                <Select.Option key={resource.id} value={resource.id} label={label}>
                  {label}
                </Select.Option>
              );
            })}
          </Select>
        </Form.Item>

        <Form.Item label="云账号ID" name="cloud_account_id">
          <Input placeholder="请输入云账号ID（可选）" />
        </Form.Item>

        <Form.Item label="Region" name="cloud_region">
          <Input placeholder="请输入Region（可选）" />
        </Form.Item>

        <Form.Item label="Zone" name="cloud_zone">
          <Input placeholder="请输入Zone（可选）" />
        </Form.Item>

        <Form.Item label="云资源ID" name="cloud_resource_id">
          <Input placeholder="请输入云资源ID（可选）" />
        </Form.Item>

        <Form.Item label="云资源类型" name="cloud_resource_type">
          <Input placeholder="请输入云资源类型（可选）" />
        </Form.Item>

        {schemaFields.length > 0 && (
          <>
            <Divider orientation="left">云资源动态属性</Divider>
            <div style={{ marginBottom: 12, color: '#8c8c8c' }}>
              动态属性仅支持枚举选择。
            </div>
            {schemaFields.map((field) => (
              <Form.Item
                key={field.key}
                label={field.label || field.key}
                name={['cloud_metadata', field.key]}
                rules={field.required ? [{ required: true, message: `请输入${field.label || field.key}` }] : undefined}
              >
                {field.type === 'select' ? (
                  <Select placeholder={field.placeholder || `请选择${field.label || field.key}`} allowClear>
                    {(field.options || []).map((option) => (
                      <Select.Option key={option} value={option}>
                        {option}
                      </Select.Option>
                    ))}
                  </Select>
                ) : (
                  <Input placeholder={field.placeholder || `请输入${field.label || field.key}`} />
                )}
              </Form.Item>
            ))}
          </>
        )}

        <Form.Item label="同步状态" name="cloud_sync_status">
          <Select placeholder="请选择同步状态" allowClear>
            <Select.Option value="success">成功</Select.Option>
            <Select.Option value="failed">失败</Select.Option>
            <Select.Option value="unknown">未知</Select.Option>
          </Select>
        </Form.Item>

        <Form.Item label="扩展属性" name="attributes">
          <TextArea rows={4} placeholder='请输入扩展属性 JSON（可选）' />
        </Form.Item>

        <Space>
          <Button onClick={() => router.back()}>取消</Button>
          <Button type="primary" htmlType="submit" loading={saving}>
            保存
          </Button>
        </Space>
      </Form>
    </Card>
  );
};

export default CreateCIPage;
