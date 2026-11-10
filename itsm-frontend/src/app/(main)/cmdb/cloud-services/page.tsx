'use client';

import React, { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  App,
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
} from 'antd';

import { CMDBApi } from '@/lib/api/cmdb-api';
import { CMDB_KEYS, useCloudServicesQuery } from '@/lib/hooks/useCMDB';
import type { CloudService } from '@/types/biz/cmdb';
import { useI18n } from '@/lib/i18n';

const { TextArea } = Input;

// 与后端 cloud.KnownProviders 同源（后端 dto 枚举由守卫测试强制对齐）。
// 火山引擎与 k8s 的发现适配器已接入，缺这两项时前端根本选不到、能力等于零。
const providerOptions = [
  { value: 'aliyun', label: 'Aliyun' },
  { value: 'huawei', label: 'Huawei Cloud' },
  { value: 'tencent', label: 'Tencent Cloud' },
  { value: 'azure', label: 'Azure' },
  { value: 'aws', label: 'AWS' },
  { value: 'volcengine', label: 'Volcengine' },
  { value: 'k8s', label: 'Kubernetes' },
  { value: 'onprem', label: 'Private Cloud' },
];

export default function CloudServicePage() {
  const { t } = useI18n();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [createForm] = Form.useForm();

  // React Query：云服务目录（provider 进 queryKey，切换自动重取）
  const [providerFilter, setProviderFilter] = useState<string | undefined>(undefined);
  const servicesQuery = useCloudServicesQuery(providerFilter);
  const serviceItems = servicesQuery.data?.items;
  const data = useMemo(() => serviceItems ?? [], [serviceItems]);
  const loading = servicesQuery.isPending;

  const serviceMap = useMemo(() => {
    return new Map(data.map(service => [service.id, service]));
  }, [data]);

  const createProvider = Form.useWatch('provider', createForm);

  const [createOpen, setCreateOpen] = useState(false);
  const [editModal, setEditModal] = useState(false);
  const [editingService, setEditingService] = useState<CloudService | null>(null);

  const invalidateServices = () =>
    queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'cloud-services'] });

  const buildPayload = (values: Record<string, unknown>) => {
    const payload: Record<string, unknown> = {
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
      const parsed: { fields?: unknown } =
        typeof values.attribute_schema === 'string'
          ? JSON.parse(values.attribute_schema)
          : (values.attribute_schema as { fields?: unknown });
      payload.attributeSchema = parsed;
      const fields = parsed.fields;
      if (fields) {
        if (!Array.isArray(fields)) {
          throw new Error(t('cmdb.propertyTemplateMustBeArray'));
        }
        for (const field of fields as Array<{ type?: string; options?: unknown }>) {
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

  // 查询：提交表单当前值
  const handleSearch = () => {
    const values = form.getFieldsValue();
    setProviderFilter(values.provider);
  };

  // 新增云服务（SyntaxError = JSON 模板非法，走独立提示）
  const createMutation = useMutation({
    mutationFn: async () => {
      const values = await createForm.validateFields();
      return CMDBApi.createCloudService(buildPayload(values));
    },
    onSuccess: () => {
      message.success(t('cmdb.cloudServiceCreated'));
      setCreateOpen(false);
      createForm.resetFields();
      invalidateServices();
    },
    onError: error => {
      if (error instanceof SyntaxError) {
        message.error(t('cmdb.propertyTemplateInvalidJSON'));
        return;
      }
      if (error instanceof Error) {
        message.error(error.message || t('cmdb.cloudServiceCreateFailed'));
      }
    },
  });

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

  const deleteMutation = useMutation({
    mutationFn: (service: CloudService) => CMDBApi.deleteCloudService(service.id),
    onSuccess: () => {
      message.success('Cloud service has been deleted');
      invalidateServices();
    },
    onError: error => {
      message.error(error instanceof Error ? error.message : 'Failed to delete cloud service');
    },
  });

  const updateMutation = useMutation({
    mutationFn: async () => {
      if (!editingService) throw new Error('no editing service');
      const values = await createForm.validateFields();
      return CMDBApi.updateCloudService(editingService.id, buildPayload(values));
    },
    onSuccess: () => {
      message.success('Cloud service has been updated');
      setEditModal(false);
      setEditingService(null);
      createForm.resetFields();
      invalidateServices();
    },
    onError: error => {
      message.error(
        error instanceof SyntaxError
          ? t('cmdb.propertyTemplateInvalidJSON')
          : error instanceof Error
            ? error.message
            : 'Failed to update cloud service'
      );
    },
  });

  const columns = [
    {
      title: 'Vendor',
      dataIndex: 'provider',
      width: 110,
      render: (value: string) => providerOptions.find(p => p.value === value)?.label || value,
    },
    {
      title: 'Superior service',
      dataIndex: 'parentId',
      width: 160,
      render: (value?: number) => (value ? serviceMap.get(value)?.serviceName || `#${value}` : '-'),
    },
    {
      title: 'Service code',
      dataIndex: 'serviceCode',
      width: 140,
    },
    {
      title: 'Category',
      dataIndex: 'category',
      width: 120,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Service Name',
      dataIndex: 'serviceName',
      width: 160,
    },
    {
      title: 'Resource type code',
      dataIndex: 'resourceTypeCode',
      width: 160,
    },
    {
      title: 'Resource type name',
      dataIndex: 'resourceTypeName',
      width: 160,
    },
    {
      title: 'API version',
      dataIndex: 'apiVersion',
      width: 120,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      width: 80,
      render: (value: boolean) => (
        <Tag color={value ? 'green' : 'default'}>{value ? 'Enabled' : 'Disable'}</Tag>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      fixed: 'right' as const,
      width: 140,
      render: (_: unknown, record: CloudService) => (
        <Space size='small'>
          <Button type='link' size='small' onClick={() => handleEdit(record)}>

            Edit
          </Button>
          <Popconfirm
            title="Confirm to delete this cloud service?"
            description="Cannot be restored after deletion."
            okText="Delete"
            cancelText="Cancel"
            onConfirm={() => deleteMutation.mutate(record)}
          >
            <Button
              type='link'
              danger
              size='small'
              loading={deleteMutation.isPending && deleteMutation.variables?.id === record.id}
              aria-label={`Delete cloud service ${record.serviceName}`}
            >

              Delete
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
          { title: 'Home page' },
          { title: 'Configuration Management' },
          { title: <a onClick={() => router.push('/cmdb')}>Configuration Item List</a> },
          { title: 'Cloud Service Directory' },
        ]}
      />

      <Form form={form} layout='inline' style={{ marginBottom: 24 }}>
        <Form.Item name='provider'>
          <Select placeholder="Cloud Provider" style={{ width: 160 }} allowClear options={providerOptions} />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button onClick={handleSearch}>Query</Button>
            <Button type='primary' onClick={() => setCreateOpen(true)}>

              New cloud service
            </Button>
          </Space>
        </Form.Item>
      </Form>

      <Table
        rowKey='id'
        loading={loading}
        dataSource={data}
        columns={columns}
        pagination={{ pageSize: 10 }}
      />

      <Modal
        title="New cloud service"
        open={createOpen}
        onCancel={() => {
          setCreateOpen(false);
          createForm.resetFields();
        }}
        onOk={() => createMutation.mutate()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={createForm} layout='vertical'>
          <Form.Item
            name='provider'
            label="Cloud Provider"
            rules={[{ required: true, message: 'Please select a cloud vendor' }]}
          >
            <Select placeholder="Please select a cloud vendor" options={providerOptions} />
          </Form.Item>
          <Form.Item
            name='service_code'
            label="Service code"
            rules={[{ required: true, message: 'Please enter service code' }]}
          >
            <Input placeholder="For example ecs/rds/oss" />
          </Form.Item>
          <Form.Item name='parent_id' label="Superior service">
            <Select
              placeholder="Select parent service (optional)"
              allowClear
              showSearch
              optionFilterProp='label'
              options={data
                .filter(service => !createProvider || service.provider === createProvider)
                .map(service => ({
                  value: service.id,
                  label: `${service.serviceName} (${service.serviceCode})`,
                }))}
            />
          </Form.Item>
          <Form.Item name='category' label="Service Category">
            <Input placeholder="e.g. compute/storage/network" />
          </Form.Item>
          <Form.Item
            name='service_name'
            label="Service Name"
            rules={[{ required: true, message: 'Please enter service name' }]}
          >
            <Input placeholder="For example, elastic computing ECS" />
          </Form.Item>
          <Form.Item
            name='resource_type_code'
            label="Resource type code"
            rules={[{ required: true, message: 'Please enter the resource type code' }]}
          >
            <Input placeholder="For example instance/volume/vpc" />
          </Form.Item>
          <Form.Item
            name='resource_type_name'
            label="Resource type name"
            rules={[{ required: true, message: 'Please enter a resource type name' }]}
          >
            <Input placeholder="For example, cloud server instance" />
          </Form.Item>
          <Form.Item name='api_version' label="API version">
            <Input placeholder="For example 2014-05-26" />
          </Form.Item>
          <Form.Item name='attribute_schema' label="Property Template (JSON)">
            <TextArea
              rows={4}
              placeholder="Only enumeration types are supported, for example {&quot;fields&quot;:[{&quot;key&quot;:&quot;instance_type&quot;,&quot;label&quot;:&quot;instance specification&quot;,&quot;type&quot;:&quot;select&quot;,&quot;options&quot;:[&quot;c6.large&quot;,&quot;c6.xlarge&quot;]}]}"
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="Editing Cloud Service"
        open={editModal}
        onCancel={() => {
          setEditModal(false);
          setEditingService(null);
          createForm.resetFields();
        }}
        onOk={() => updateMutation.mutate()}
        confirmLoading={updateMutation.isPending}
      >
        <Form form={createForm} layout='vertical'>
          <Form.Item
            name='provider'
            label="Cloud Provider"
            rules={[{ required: true, message: 'Please select a cloud vendor' }]}
          >
            <Select placeholder="Please select a cloud vendor" options={providerOptions} />
          </Form.Item>
          <Form.Item
            name='service_code'
            label="Service code"
            rules={[{ required: true, message: 'Please enter service code' }]}
          >
            <Input placeholder="For example ecs/rds/oss" />
          </Form.Item>
          <Form.Item name='parent_id' label="Superior service">
            <Select
              placeholder="Select parent service (optional)"
              allowClear
              showSearch
              optionFilterProp='label'
              options={data
                .filter(
                  service =>
                    service.id !== editingService?.id &&
                    (!createProvider || service.provider === createProvider)
                )
                .map(service => ({
                  value: service.id,
                  label: `${service.serviceName} (${service.serviceCode})`,
                }))}
            />
          </Form.Item>
          <Form.Item name='category' label="Service Category">
            <Input placeholder="e.g. compute/storage/network" />
          </Form.Item>
          <Form.Item
            name='service_name'
            label="Service Name"
            rules={[{ required: true, message: 'Please enter service name' }]}
          >
            <Input placeholder="For example, elastic computing ECS" />
          </Form.Item>
          <Form.Item
            name='resource_type_code'
            label="Resource type code"
            rules={[{ required: true, message: 'Please enter the resource type code' }]}
          >
            <Input placeholder="For example instance/volume/vpc" />
          </Form.Item>
          <Form.Item
            name='resource_type_name'
            label="Resource type name"
            rules={[{ required: true, message: 'Please enter a resource type name' }]}
          >
            <Input placeholder="For example, cloud server instance" />
          </Form.Item>
          <Form.Item name='api_version' label="API version">
            <Input placeholder="For example 2014-05-26" />
          </Form.Item>
          <Form.Item name='attribute_schema' label="Property Template (JSON)">
            <TextArea
              rows={4}
              placeholder="Only enumeration types are supported, for example {&quot;fields&quot;:[{&quot;key&quot;:&quot;instance_type&quot;,&quot;label&quot;:&quot;instance specification&quot;,&quot;type&quot;:&quot;select&quot;,&quot;options&quot;:[&quot;c6.large&quot;,&quot;c6.xlarge&quot;]}]}"
            />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
