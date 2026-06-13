'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Breadcrumb, Button, Card, Form, Input, Modal, Select, Space, Table, Tag, App } from 'antd';
import dayjs from 'dayjs';

import { CMDBApi } from '@/lib/api/cmdb-api';
import type { CloudResource, CloudService } from '@/types/biz/cmdb';

const { Option } = Select;

const providerOptions = [
  { value: 'aliyun', label: '阿里云' },
  { value: 'huawei', label: '华为云' },
  { value: 'tencent', label: '腾讯云' },
  { value: 'azure', label: 'Azure' },
  { value: 'onprem', label: '私有云' },
];

export default function CloudResourcePage() {
  const router = useRouter();
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [bindForm] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [resources, setResources] = useState<CloudResource[]>([]);
  const [services, setServices] = useState<CloudService[]>([]);
  const [binding, setBinding] = useState<CloudResource | null>(null);
  const [bindSubmitting, setBindSubmitting] = useState(false);

  const provider = Form.useWatch('provider', form);

  const serviceMap = useMemo(() => {
    return new Map(services.map(service => [service.id, service]));
  }, [services]);

  const loadServices = async () => {
    const isMounted = true;
    try {
      const list = await CMDBApi.getCloudServices();
      if (isMounted) {
        setServices(list || []);
      }
    } catch (error) {
      if (isMounted) {
        message.error('加载云服务目录失败');
      }
    }
  };

  const loadResources = async () => {
    const isMounted = true;
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const list = await CMDBApi.getCloudResources({
        provider: values.provider,
        service_id: values.service_id,
        region: values.region,
      });
      if (isMounted) {
        setResources(
          Array.isArray(list) ? (list as CloudResource[]) : ((list as any).items || (list as any).data || [])
        );
      }
    } catch (error) {
      if (isMounted) {
        message.error('加载云资源失败');
      }
    } finally {
      if (isMounted) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    let isMounted = true;
    loadServices();
    loadResources();
    return () => {
      isMounted = false;
    };
  }, []);

  const columns = [
    {
      title: '厂商',
      width: 110,
      render: (_: unknown, record: CloudResource) => {
        const serviceId = record.serviceId ?? record.service_id;
        const providerValue = serviceMap.get(serviceId)?.provider;
        return providerOptions.find(p => p.value === providerValue)?.label || providerValue || '-';
      },
    },
    {
      title: '服务',
      width: 160,
      render: (_: unknown, record: CloudResource) => {
        const serviceId = record.serviceId ?? record.service_id;
        return serviceMap.get(serviceId)?.serviceName ?? serviceMap.get(serviceId)?.service_name ?? `#${serviceId}`;
      },
    },
    {
      title: '资源类型',
      width: 160,
      render: (_: unknown, record: CloudResource) => {
        const serviceId = record.serviceId ?? record.service_id;
        const service = serviceMap.get(serviceId);
        return service?.resourceTypeName ?? service?.resource_type_name ?? '-';
      },
    },
    {
      title: '资源ID',
      dataIndex: 'resourceId',
      width: 200,
    },
    {
      title: '资源名称',
      width: 180,
      render: (_: unknown, record: CloudResource) => record.resourceName ?? record.resource_name ?? '-',
    },
    {
      title: 'Region',
      dataIndex: 'region',
      width: 120,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Zone',
      dataIndex: 'zone',
      width: 120,
      render: (value?: string) => value || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      render: (value?: string) => <Tag color={value ? 'green' : 'default'}>{value || '未知'}</Tag>,
    },
    {
      title: '最近发现',
      width: 160,
      render: (_: unknown, record: CloudResource) => {
        const value = record.lastSeenAt ?? record.last_seen_at;
        return value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-';
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: CloudResource) => (
        <Space>
          <Button
            type="link"
            onClick={() => router.push(`/cmdb/cis/create?cloud_resource_ref_id=${record.id}`)}
          >
            新建CI
          </Button>
          <Button
            type="link"
            onClick={() => {
              setBinding(record);
              bindForm.resetFields();
              bindForm.setFieldsValue({ ci_id: undefined });
            }}
          >
            绑定已有
          </Button>
        </Space>
      ),
    },
  ];

  const handleBindExisting = async () => {
    if (!binding) return;
    try {
      const values = await bindForm.validateFields();
      const service = serviceMap.get(binding.serviceId ?? binding.service_id);
      setBindSubmitting(true);
      await CMDBApi.updateCI(values.ci_id, {
        cloudResourceRefId: binding.id,
        cloudProvider: service?.provider,
        cloudAccountId: String(binding.cloudAccountId ?? binding.cloud_account_id),
        cloudRegion: binding.region,
        cloudZone: binding.zone,
        cloudResourceId: binding.resourceId ?? binding.resource_id,
        cloudResourceType: service?.resourceTypeCode ?? service?.resource_type_code,
        cloudMetadata: binding.metadata,
        cloudSyncStatus: 'success',
      });
      message.success('已绑定到配置项');
      setBinding(null);
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '绑定失败');
      } else {
        message.error('绑定失败');
      }
    } finally {
      setBindSubmitting(false);
    }
  };

  return (
    <Card>
      <div style={{ marginBottom: 16 }}>
        <h1 className="text-2xl font-bold">云资源列表</h1>
        <p className="text-gray-500 mt-1">查看已发现的云资源，并将资源新建或绑定为 CMDB 配置项。</p>
      </div>

      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: '首页' },
          { title: '配置管理' },
          { title: <a onClick={() => router.push('/cmdb')}>配置项列表</a> },
          { title: '云资源列表' },
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
        <Form.Item name="service_id">
          <Select
            placeholder="云服务"
            style={{ width: 200 }}
            allowClear
            showSearch
            optionFilterProp="label"
          >
            {services
              .filter(service => !provider || service.provider === provider)
              .map(service => (
                <Option
                  key={service.id}
                  value={service.id}
                  label={`${service.service_name} (${service.resource_type_name})`}
                >
                  {service.serviceName ?? service.service_name} ({service.resourceTypeName ?? service.resource_type_name})
                </Option>
              ))}
          </Select>
        </Form.Item>
        <Form.Item name="region">
          <Input placeholder="Region" style={{ width: 140 }} allowClear />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button onClick={loadResources}>查询</Button>
            <Button onClick={loadResources}>刷新</Button>
          </Space>
        </Form.Item>
      </Form>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={resources}
        columns={columns as any}
        pagination={{ pageSize: 10 }}
      />

      <Modal
        title="绑定已有配置项"
        open={Boolean(binding)}
        onCancel={() => setBinding(null)}
        onOk={handleBindExisting}
        confirmLoading={bindSubmitting}
        destroyOnClose
      >
        <Form form={bindForm} layout="vertical">
          <Form.Item
            name="ci_id"
            label="配置项ID"
            rules={[{ required: true, message: '请输入配置项ID' }]}
          >
            <Input placeholder="请输入已存在的配置项ID" />
          </Form.Item>
          {binding && (
            <div style={{ color: '#888' }}>
              将绑定资源：{binding.resourceName || binding.resource_name || binding.resourceId || binding.resource_id}
            </div>
          )}
        </Form>
      </Modal>
    </Card>
  );
}
