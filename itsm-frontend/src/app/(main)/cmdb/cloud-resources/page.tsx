'use client';

import React, { useEffect, useMemo, useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { Breadcrumb, Button, Card, Form, Input, Modal, Select, Space, Table, Tag, App, Tooltip } from 'antd';
import { Search, Plus, Eye, RotateCcw, Link } from 'lucide-react';
import dayjs from 'dayjs';

import { CMDBApi } from '@/lib/api/cmdb-api';
import type { CloudResource, CloudService } from '@/types/biz/cmdb';

const { Option } = Select;

const providerOptions = [
  { value: 'aliyun', label: '阿里云' },
  { value: 'huawei', label: '华为云' },
  { value: 'tencent', label: '腾讯云' },
  { value: 'azure', label: 'Azure' },
  { value: 'aws', label: 'AWS' },
  { value: 'onprem', label: '私有云' },
];

const statusColors: Record<string, string> = {
  running: 'green',
  stopped: 'default',
  active: 'green',
  inactive: 'default',
  available: 'green',
  unavailable: 'red',
};

// 云资源状态中文映射
const cloudResourceStatusTextMap: Record<string, string> = {
  running: '运行中',
  stopped: '已停止',
  active: '活跃',
  inactive: '未激活',
  available: '可用',
  unavailable: '不可用',
  pending: '处理中',
  failed: '失败',
};

export default function CloudResourcePage() {
  const router = useRouter();
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [bindForm] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [resources, setResources] = useState<CloudResource[]>([]);
  const [total, setTotal] = useState(0);
  const [services, setServices] = useState<CloudService[]>([]);
  const [binding, setBinding] = useState<CloudResource | null>(null);
  const [bindSubmitting, setBindSubmitting] = useState(false);
  const [selectedRow, setSelectedRow] = useState<CloudResource | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10 });

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

  const loadResources = async (page = 1, pageSize = 10) => {
    const isMounted = true;
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const list = await CMDBApi.getCloudResources({
        provider: values.provider,
        serviceId: values.serviceId,
        region: values.region,
        offset: (page - 1) * pageSize,
        limit: pageSize,
      });
      if (isMounted) {
        // API 返回兼容处理：可能是数组或 { items, total } 格式
        type ApiListResponse = CloudResource[] | { items?: CloudResource[]; data?: CloudResource[]; total?: number };
        const response = list as ApiListResponse;
        const items = Array.isArray(response)
          ? response
          : (response.items || response.data || []);
        const totalCount = Array.isArray(response)
          ? items.length
          : (response.total ?? items.length);
        setResources(items);
        setTotal(totalCount);
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

  // 分页变化处理
  const handleTableChange = (page: number, pageSize: number) => {
    setPagination({ current: page, pageSize });
    loadResources(page, pageSize);
  };

  // 查看资源详情
  const handleViewDetail = (record: CloudResource) => {
    setSelectedRow(record);
    setDetailOpen(true);
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
      dataIndex: 'provider',
      width: 100,
      render: (value: string) => {
        const provider = providerOptions.find(p => p.value === value);
        return <Tag color="blue">{provider?.label || value || '-'}</Tag>;
      },
    },
    {
      title: '服务',
      width: 140,
      render: (_: unknown, record: CloudResource) => {
        const serviceId = record.serviceId ?? record.serviceId;
        const service = serviceMap.get(serviceId);
        return service?.serviceName ?? service?.serviceName ?? '-';
      },
    },
    {
      title: '资源类型',
      width: 120,
      render: (_: unknown, record: CloudResource) => {
        const serviceId = record.serviceId ?? record.serviceId;
        const service = serviceMap.get(serviceId);
        return service?.resourceTypeName ?? service?.resourceTypeName ?? '-';
      },
    },
    {
      title: '资源ID',
      dataIndex: 'resourceId',
      width: 180,
      ellipsis: true,
    },
    {
      title: '资源名称',
      dataIndex: 'resourceName',
      width: 160,
      ellipsis: true,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Region',
      dataIndex: 'region',
      width: 100,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Zone',
      dataIndex: 'zone',
      width: 100,
      render: (value?: string) => value || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (value?: string) => (
        <Tag color={statusColors[value || ''] || 'default'}>
          {value || '未知'}
        </Tag>
      ),
    },
    {
      title: '最近发现',
      width: 150,
      render: (_: unknown, record: CloudResource) => {
        const value = record.lastSeenAt ?? record.lastSeenAt;
        return value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-';
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: CloudResource) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<Eye />}
              onClick={() => handleViewDetail(record)}
              size="small"
            />
          </Tooltip>
          <Button
            type="link"
            size="small"
            onClick={() => router.push(`/cmdb/cis/create?cloud_resource_ref_id=${record.id}`)}
          >
            新建CI
          </Button>
          <Button
            type="link"
            size="small"
            onClick={() => {
              setBinding(record);
              bindForm.resetFields();
              bindForm.setFieldsValue({ ciId: undefined });
            }}
          >
            绑定
          </Button>
        </Space>
      ),
    },
  ];

  const handleBindExisting = async () => {
    if (!binding) return;
    try {
      const values = await bindForm.validateFields();
      const service = serviceMap.get(binding.serviceId ?? binding.serviceId);
      setBindSubmitting(true);
      await CMDBApi.updateCI(values.ciId, {
        cloudResourceRefId: binding.id,
        cloudProvider: service?.provider,
        cloudAccountId: String(binding.cloudAccountId ?? binding.cloudAccountId),
        cloudRegion: binding.region,
        cloudZone: binding.zone,
        cloudResourceId: binding.resourceId ?? binding.resourceId,
        cloudResourceType: service?.resourceTypeCode ?? service?.resourceTypeCode,
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
      <div className="mb-4">
        <h1 className="text-2xl font-bold">云资源列表</h1>
        <p className="text-gray-500 mt-1">查看已发现的云资源，并将资源新建或绑定为 CMDB 配置项。</p>
      </div>

      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: '首页' },
          { title: '配置管理' },
          { title: <a onClick={() => router.push('/cmdb')}>CMDB</a> },
          { title: '云资源列表' },
        ]}
      />

      {/* 搜索工具栏 */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <Form form={form} layout="inline" className="flex-wrap gap-2">
          <Form.Item name="provider" className="!mb-0">
            <Select placeholder="云厂商" style={{ width: 140 }} allowClear>
              {providerOptions.map(item => (
                <Option key={item.value} value={item.value}>
                  {item.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="service_id" className="!mb-0">
            <Select
              placeholder="云服务"
              style={{ width: 180 }}
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
                    label={`${service.serviceName} (${service.resourceTypeName})`}
                  >
                    {service.serviceName ?? service.serviceName} ({service.resourceTypeName ?? service.resourceTypeName})
                  </Option>
                ))}
            </Select>
          </Form.Item>
          <Form.Item name="region" className="!mb-0">
            <Input placeholder="Region" style={{ width: 120 }} allowClear />
          </Form.Item>
          <Form.Item className="!mb-0">
            <Space>
              <Button type="primary" icon={<Search />} onClick={() => loadResources(1, pagination.pageSize)}>
                查询
              </Button>
              <Button icon={<RotateCcw />} onClick={() => loadResources(pagination.current, pagination.pageSize)} loading={loading}>
                刷新
              </Button>
            </Space>
          </Form.Item>
        </Form>
        <span className="ml-auto text-sm text-gray-500">
          共 {total} 条资源
        </span>
      </div>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={resources}
        columns={columns as unknown as React.ComponentProps<typeof Table>['columns']}
        pagination={{
          current: pagination.current,
          pageSize: pagination.pageSize,
          total: total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: total => `共 ${total} 条记录`,
          pageSizeOptions: ['10', '20', '50', '100'],
          onChange: handleTableChange,
        }}
        scroll={{ x: 1200 }}
      />

      {/* 绑定已有配置项模态框 */}
      <Modal
        title="绑定已有配置项"
        open={Boolean(binding)}
        onCancel={() => setBinding(null)}
        onOk={handleBindExisting}
        confirmLoading={bindSubmitting}
        width={480}
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
            <div className="p-3 bg-gray-50 rounded text-sm text-gray-600">
              <div className="font-medium mb-1">将绑定资源：</div>
              <div>{binding.resourceName || binding.resourceName || binding.resourceId || binding.resourceId}</div>
              <div className="text-gray-400 mt-1">
                {providerOptions.find(p => p.value === (binding as any).provider)?.label} / {binding.region} / {binding.zone}
              </div>
            </div>
          )}
        </Form>
      </Modal>

      {/* 资源详情模态框 */}
      <Modal
        title="云资源详情"
        open={detailOpen}
        onCancel={() => {
          setDetailOpen(false);
          setSelectedRow(null);
        }}
        footer={[
          <Button key="close" onClick={() => setDetailOpen(false)}>
            关闭
          </Button>,
          <Button
            key="create"
            type="primary"
            icon={<Plus />}
            onClick={() => {
              if (selectedRow) {
                router.push(`/cmdb/cis/create?cloud_resource_ref_id=${selectedRow.id}`);
              }
            }}
          >
            新建CI
          </Button>,
        ]}
        width={560}
      >
        {selectedRow && (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <div className="text-sm text-gray-500">云厂商</div>
                <div>{providerOptions.find(p => p.value === (selectedRow as any).provider)?.label || (selectedRow as any).provider || '-'}</div>
              </div>
              <div>
                <div className="text-sm text-gray-500">服务类型</div>
                <div>{serviceMap.get(selectedRow.serviceId ?? selectedRow.serviceId)?.serviceName || '-'}</div>
              </div>
              <div>
                <div className="text-sm text-gray-500">资源ID</div>
                <div className="font-mono text-sm">{selectedRow.resourceId || selectedRow.resourceId || '-'}</div>
              </div>
              <div>
                <div className="text-sm text-gray-500">资源名称</div>
                <div>{selectedRow.resourceName || selectedRow.resourceName || '-'}</div>
              </div>
              <div>
                <div className="text-sm text-gray-500">Region</div>
                <div>{selectedRow.region || '-'}</div>
              </div>
              <div>
                <div className="text-sm text-gray-500">Zone</div>
                <div>{selectedRow.zone || '-'}</div>
              </div>
              <div>
                <div className="text-sm text-gray-500">状态</div>
                <Tag color={statusColors[selectedRow.status || ''] || 'default'}>
                  {cloudResourceStatusTextMap[selectedRow.status || ''] || selectedRow.status || '未知'}
                </Tag>
              </div>
              <div>
                <div className="text-sm text-gray-500">最近发现</div>
                <div>{selectedRow.lastSeenAt ? dayjs(selectedRow.lastSeenAt).format('YYYY-MM-DD HH:mm:ss') : '-'}</div>
              </div>
            </div>
          </div>
        )}
      </Modal>
    </Card>
  );
}
