'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Breadcrumb, Button, Card, Form, Input, Modal, Select, Space, Table, Tag, message } from 'antd';

import { CMDBApi } from '@/modules/cmdb/api';
import type { CloudService } from '@/modules/cmdb/types';

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
  const router = useRouter();
  const [form] = Form.useForm();
  const [createForm] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<CloudService[]>([]);
  const [createOpen, setCreateOpen] = useState(false);

  const serviceMap = useMemo(() => {
    return new Map(data.map((service) => [service.id, service]));
  }, [data]);

  const createProvider = Form.useWatch('provider', createForm);

  const loadData = async () => {
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const list = await CMDBApi.getCloudServices(values.provider);
      setData(list || []);
    } catch (error) {
      message.error('加载云服务目录失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleCreate = async () => {
    try {
      const values = await createForm.validateFields();
      const payload = { ...values } as Record<string, any>;
      if (values.attribute_schema) {
        try {
          payload.attribute_schema =
            typeof values.attribute_schema === 'string'
              ? JSON.parse(values.attribute_schema)
              : values.attribute_schema;
          const fields = payload.attribute_schema?.fields;
          if (fields) {
            if (!Array.isArray(fields)) {
              message.error('属性模板字段必须为数组');
              return;
            }
            for (const field of fields) {
              const type = field?.type;
              if (type !== 'select') {
                message.error('属性模板字段类型仅支持 select');
                return;
              }
              if (!Array.isArray(field?.options) || field.options.length === 0) {
                message.error('枚举类型必须提供非空 options');
                return;
              }
            }
          }
        } catch (error) {
          message.error('属性模板需要是有效的 JSON');
          return;
        }
      }
      await CMDBApi.createCloudService(payload);
      message.success('云服务已创建');
      setCreateOpen(false);
      createForm.resetFields();
      loadData();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '创建失败');
      }
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
      dataIndex: 'parent_id',
      width: 160,
      render: (value?: number) => (value ? serviceMap.get(value)?.service_name || `#${value}` : '-'),
    },
    {
      title: '服务代码',
      dataIndex: 'service_code',
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
      dataIndex: 'service_name',
      width: 160,
    },
    {
      title: '资源类型代码',
      dataIndex: 'resource_type_code',
      width: 160,
    },
    {
      title: '资源类型名称',
      dataIndex: 'resource_type_name',
      width: 160,
    },
    {
      title: 'API版本',
      dataIndex: 'api_version',
      width: 120,
      render: (value?: string) => value || '-',
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      width: 80,
      render: (value: boolean) => (
        <Tag color={value ? 'green' : 'default'}>{value ? '启用' : '停用'}</Tag>
      ),
    },
  ];

  return (
    <Card variant="borderless">
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
            <Button type="primary" onClick={() => setCreateOpen(true)}>新增云服务</Button>
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
        onCancel={() => setCreateOpen(false)}
        onOk={handleCreate}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical">
          <Form.Item name="provider" label="云厂商" rules={[{ required: true, message: '请选择云厂商' }]}>
            <Select placeholder="请选择云厂商">
              {providerOptions.map(item => (
                <Option key={item.value} value={item.value}>
                  {item.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="service_code" label="服务代码" rules={[{ required: true, message: '请输入服务代码' }]}>
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
                .filter((service) => !createProvider || service.provider === createProvider)
                .map((service) => (
                  <Option
                    key={service.id}
                    value={service.id}
                    label={`${service.service_name} (${service.service_code})`}
                  >
                    {service.service_name} ({service.service_code})
                  </Option>
                ))}
            </Select>
          </Form.Item>
          <Form.Item name="category" label="服务分类">
            <Input placeholder="例如 计算/存储/网络" />
          </Form.Item>
          <Form.Item name="service_name" label="服务名称" rules={[{ required: true, message: '请输入服务名称' }]}>
            <Input placeholder="例如 弹性计算 ECS" />
          </Form.Item>
          <Form.Item name="resource_type_code" label="资源类型代码" rules={[{ required: true, message: '请输入资源类型代码' }]}>
            <Input placeholder="例如 instance/volume/vpc" />
          </Form.Item>
          <Form.Item name="resource_type_name" label="资源类型名称" rules={[{ required: true, message: '请输入资源类型名称' }]}>
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
    </Card>
  );
}
