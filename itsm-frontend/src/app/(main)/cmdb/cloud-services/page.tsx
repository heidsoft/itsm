'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Breadcrumb,
  Button,
  Card,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  message,
} from 'antd';

import { CMDBApi } from '@/lib/api/cmdb-api';
import type { CloudService } from '@/types/biz/cmdb';
import { useI18n } from '@/lib/i18n';

const { Option } = Select;
const { TextArea } = Input;

const providerOptions = [
  { value: 'aliyun', label: '阿里云' },
  { value: 'huawei', label: '华为云' },
  { value: 'tencent', label: '腾讯云' },
  { value: 'azure', label: 'Azure' },
  { value: 'onprem', label: '私有云' },
];

export default function CloudServicePage() {
  const { t } = useI18n();
  const router = useRouter();
  const [form] = Form.useForm();
  const [createForm] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<CloudService[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [editModal, setEditModal] = useState(false);
  const [editingService, setEditingService] = useState<CloudService | null>(null);

  const serviceMap = useMemo(() => {
    return new Map(data.map(service => [service.id, service]));
  }, [data]);

  const createProvider = Form.useWatch('provider', createForm);

  const buildPayload = (values: Record<string, any>) => {
    const payload: Record<string, any> = {
      provider: values.provider,
      serviceCode: values.service_code,
      parentId: values.parent_id,
      category: values.category,
      serviceName: values.service_name,
      resourceTypeCode: values.resource_type_code,
      resourceTypeName: values.resource_type_name,
      apiVersion: values.api_version,
      isActive: values.is_active ?? editingService?.isActive,
    };

    if (values.attribute_schema) {
      payload.attributeSchema =
        typeof values.attribute_schema === 'string'
          ? JSON.parse(values.attribute_schema)
          : values.attribute_schema;
      const fields = payload.attributeSchema?.fields;
      if (fields) {
        if (!Array.isArray(fields)) {
          throw new Error(t('cmdb.propertyTemplateMustBeArray'));
        }
        for (const field of fields) {
          if (field?.type !== 'select') {
            throw new Error(t('cmdb.propertyTemplateOnlySelect'));
          }
          if (!Array.isArray(field?.options) || field.options.length === 0) {
            throw new Error(t('cmdb.enumMustHaveOptions'));
          }
        }
      }
    }

    return payload;
  };

  const loadData = async () => {
    const isMounted = true;
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const list = await CMDBApi.getCloudServices(values.provider);
      if (isMounted) {
        setData(list || []);
      }
    } catch (error) {
      if (isMounted) {
        message.error(t('cmdb.loadCloudServicesFailed'));
      }
    } finally {
      if (isMounted) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    let isMounted = true;
    loadData();
    return () => {
      isMounted = false;
    };
  }, []);

  const handleCreate = async () => {
    try {
      const values = await createForm.validateFields();
      const payload = buildPayload(values);
      await CMDBApi.createCloudService(payload);
      message.success(t('cmdb.cloudServiceCreated'));
      setCreateOpen(false);
      createForm.resetFields();
      loadData();
    } catch (error) {
      if (error instanceof SyntaxError) {
        message.error(t('cmdb.propertyTemplateInvalidJSON'));
        return;
      }
      if (error instanceof Error) {
        message.error(error.message || t('cmdb.cloudServiceCreateFailed'));
      }
    }
  };

  const handleEdit = (service: CloudService) => {
    setEditingService(service);
    createForm.setFieldsValue({
      provider: service.provider,
      service_code: service.serviceCode,
      parent_id: service.parentId || undefined,
      category: service.category,
      service_name: service.serviceName,
      resource_type_code: service.resourceTypeCode,
      resource_type_name: service.resourceTypeName,
      api_version: service.apiVersion,
      attribute_schema: service.attributeSchema
        ? JSON.stringify(service.attributeSchema, null, 2)
        : undefined,
      is_active: service.isActive,
    });
    setEditModal(true);
  };

  const handleDelete = async (service: CloudService) => {
    try {
      await CMDBApi.deleteCloudService(service.id);
      message.success('云服务已删除');
      loadData();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除云服务失败');
    }
  };

  const handleUpdate = async () => {
    if (!editingService) return;
    try {
      const values = await createForm.validateFields();
      await CMDBApi.updateCloudService(editingService.id, buildPayload(values));
      message.success('云服务已更新');
      setEditModal(false);
      setEditingService(null);
      createForm.resetFields();
      loadData();
    } catch (error) {
      message.error(
        error instanceof SyntaxError
          ? t('cmdb.propertyTemplateInvalidJSON')
          : error instanceof Error
            ? error.message
            : '更新云服务失败',
      );
    }
  };

  const columns = [
    {
      title: '厂商',
      dataIndex: 'provider',
      width: 110,
      render: (value: string) => providerOptions.find(p => p.value === value)?.label || value,
    },
    {
      title: '上级服务',
      dataIndex: 'parentId',
      width: 160,
      render: (value?: number) =>
        value ? serviceMap.get(value)?.serviceName || `#${value}` : '-',
    },
    {
      title: '服务代码',
      dataIndex: 'serviceCode',
      width: 140,
    },
    {
      title: '分类',
      dataIndex: 'category',
      width: 120,
      render: (value?: string) => value || '-',
    },
    {
      title: '服务名称',
      dataIndex: 'serviceName',
      width: 160,
    },
    {
      title: '资源类型代码',
      dataIndex: 'resourceTypeCode',
      width: 160,
    },
    {
      title: '资源类型名称',
      dataIndex: 'resourceTypeName',
      width: 160,
    },
    {
      title: 'API版本',
      dataIndex: 'apiVersion',
      width: 120,
      render: (value?: string) => value || '-',
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      width: 80,
      render: (value: boolean) => (
        <Tag color={value ? 'green' : 'default'}>{value ? '启用' : '停用'}</Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      fixed: 'right' as const,
      width: 140,
      render: (_: unknown, record: CloudService) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确认删除该云服务？"
            description="删除后无法恢复。"
            okText="删除"
            cancelText="取消"
            onConfirm={() => handleDelete(record)}
          >
            <Button type="link" danger size="small">
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: '首页' },
          { title: '配置管理' },
          { title: <a onClick={() => router.push('/cmdb')}>配置项列表</a> },
          { title: '云服务目录' },
        ]}
      />

      <Form form={form} layout="inline" style={{ marginBottom: 24 }}>
        <Form.Item name="provider">
          <Select placeholder="云厂商" style={{ width: 160 }} allowClear>
            {providerOptions.map(item => (
              <Option key={item.value} value={item.value}>
                {item.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item>
          <Space>
            <Button onClick={loadData}>查询</Button>
            <Button type="primary" onClick={() => setCreateOpen(true)}>
              新增云服务
            </Button>
          </Space>
        </Form.Item>
      </Form>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={data}
        columns={columns as any}
        pagination={{ pageSize: 10 }}
      />

      <Modal
        title="新增云服务"
        open={createOpen}
        onCancel={() => {
          setCreateOpen(false);
          createForm.resetFields();
        }}
        onOk={handleCreate}
      >
        <Form form={createForm} layout="vertical">
          <Form.Item
            name="provider"
            label="云厂商"
            rules={[{ required: true, message: '请选择云厂商' }]}
          >
            <Select placeholder="请选择云厂商">
              {providerOptions.map(item => (
                <Option key={item.value} value={item.value}>
                  {item.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            name="service_code"
            label="服务代码"
            rules={[{ required: true, message: '请输入服务代码' }]}
          >
            <Input placeholder="例如 ecs/rds/oss" />
          </Form.Item>
          <Form.Item name="parent_id" label="上级服务">
            <Select
              placeholder="选择父级服务（可选）"
              allowClear
              showSearch
              optionFilterProp="label"
            >
              {data
                .filter(service => !createProvider || service.provider === createProvider)
                .map(service => (
                  <Option
                    key={service.id}
                    value={service.id}
                    label={`${service.serviceName} (${service.serviceCode})`}
                  >
                    {service.serviceName} ({service.serviceCode})
                  </Option>
                ))}
            </Select>
          </Form.Item>
          <Form.Item name="category" label="服务分类">
            <Input placeholder="例如 计算/存储/网络" />
          </Form.Item>
          <Form.Item
            name="service_name"
            label="服务名称"
            rules={[{ required: true, message: '请输入服务名称' }]}
          >
            <Input placeholder="例如 弹性计算 ECS" />
          </Form.Item>
          <Form.Item
            name="resource_type_code"
            label="资源类型代码"
            rules={[{ required: true, message: '请输入资源类型代码' }]}
          >
            <Input placeholder="例如 instance/volume/vpc" />
          </Form.Item>
          <Form.Item
            name="resource_type_name"
            label="资源类型名称"
            rules={[{ required: true, message: '请输入资源类型名称' }]}
          >
            <Input placeholder="例如 云服务器实例" />
          </Form.Item>
          <Form.Item name="api_version" label="API版本">
            <Input placeholder="例如 2014-05-26" />
          </Form.Item>
          <Form.Item name="attribute_schema" label="属性模板(JSON)">
            <TextArea
              rows={4}
              placeholder='仅支持枚举类型，例如 {"fields":[{"key":"instance_type","label":"实例规格","type":"select","options":["c6.large","c6.xlarge"]}]}'
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="编辑云服务"
        open={editModal}
        onCancel={() => {
          setEditModal(false);
          setEditingService(null);
          createForm.resetFields();
        }}
        onOk={handleUpdate}
      >
        <Form form={createForm} layout="vertical">
          <Form.Item name="provider" label="云厂商" rules={[{ required: true, message: '请选择云厂商' }]}>
            <Select placeholder="请选择云厂商">
              {providerOptions.map(item => <Option key={item.value} value={item.value}>{item.label}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="service_code" label="服务代码" rules={[{ required: true, message: '请输入服务代码' }]}>
            <Input placeholder="例如 ecs/rds/oss" />
          </Form.Item>
          <Form.Item name="parent_id" label="上级服务">
            <Select placeholder="选择父级服务（可选）" allowClear showSearch optionFilterProp="label">
              {data
                .filter(service => service.id !== editingService?.id && (!createProvider || service.provider === createProvider))
                .map(service => (
                  <Option key={service.id} value={service.id} label={`${service.serviceName} (${service.serviceCode})`}>
                    {service.serviceName} ({service.serviceCode})
                  </Option>
                ))}
            </Select>
          </Form.Item>
          <Form.Item name="category" label="服务分类"><Input placeholder="例如 计算/存储/网络" /></Form.Item>
          <Form.Item name="service_name" label="服务名称" rules={[{ required: true, message: '请输入服务名称' }]}>
            <Input placeholder="例如 弹性计算 ECS" />
          </Form.Item>
          <Form.Item name="resource_type_code" label="资源类型代码" rules={[{ required: true, message: '请输入资源类型代码' }]}>
            <Input placeholder="例如 instance/volume/vpc" />
          </Form.Item>
          <Form.Item name="resource_type_name" label="资源类型名称" rules={[{ required: true, message: '请输入资源类型名称' }]}>
            <Input placeholder="例如 云服务器实例" />
          </Form.Item>
          <Form.Item name="api_version" label="API版本"><Input placeholder="例如 2014-05-26" /></Form.Item>
          <Form.Item name="attribute_schema" label="属性模板(JSON)">
            <TextArea rows={4} placeholder='仅支持枚举类型，例如 {"fields":[{"key":"instance_type","label":"实例规格","type":"select","options":["c6.large","c6.xlarge"]}]}' />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
