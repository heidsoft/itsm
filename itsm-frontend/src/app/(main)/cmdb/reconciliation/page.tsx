'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Breadcrumb,
  Button,
  Card,
  Input,
  Modal,
  Space,
  Statistic,
  Table,
  Tag,
  message,
} from 'antd';
import dayjs from 'dayjs';
import type { ColumnsType } from 'antd/es/table';

import { CMDBApi } from '@/lib/api/cmdb-api';
import type {
  CloudResource,
  CloudService,
  ConfigurationItem,
  ReconciliationResponse,
} from '@/types/biz/cmdb';

const summaryLabels = [
  { key: 'resourceTotal', label: '资源总数' },
  { key: 'boundResourceCount', label: '已绑定资源' },
  { key: 'unboundResourceCount', label: '待绑定资源' },
  { key: 'orphanCICount', label: '孤儿CI' },
  { key: 'unlinkedCICount', label: '未关联CI' },
];

export default function ReconciliationPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [summary, setSummary] = useState<ReconciliationResponse['summary'] | null>(null);
  const [unboundResources, setUnboundResources] = useState<CloudResource[]>([]);
  const [orphanCIs, setOrphanCIs] = useState<ConfigurationItem[]>([]);
  const [unlinkedCIs, setUnlinkedCIs] = useState<ConfigurationItem[]>([]);
  const [services, setServices] = useState<CloudService[]>([]);
  const [bindOpen, setBindOpen] = useState(false);
  const [bindResource, setBindResource] = useState<CloudResource | null>(null);
  const [bindCIID, setBindCIID] = useState<string>('');

  const serviceMap = useMemo(() => new Map(services.map(item => [item.id, item])), [services]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [recon, serviceList] = await Promise.all([
        CMDBApi.getReconciliationResults(),
        CMDBApi.getCloudServices(),
      ]);
      const r = recon as unknown as ReconciliationResponse;
      setSummary(r.summary || null);
      setUnboundResources(r.unboundResources || []);
      setOrphanCIs(r.orphanCIs || []);
      setUnlinkedCIs(r.unlinkedCIs || []);
      setServices(serviceList || []);
    } catch (error) {
      message.error('加载对账数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleBindExisting = async () => {
    if (!bindResource || !bindCIID) return;
    const ciID = Number(bindCIID);
    if (Number.isNaN(ciID)) {
      message.error('请输入有效的配置项ID');
      return;
    }
    try {
      const service = serviceMap.get(bindResource.serviceId ?? bindResource.serviceId);
      await CMDBApi.updateCI(String(ciID), {
        cloudResourceRefId: bindResource.id,
        cloudProvider: service?.provider,
        cloudAccountId: String(bindResource.cloudAccountId ?? bindResource.cloudAccountId),
        cloudRegion: bindResource.region,
        cloudZone: bindResource.zone,
        cloudResourceId: bindResource.resourceId ?? bindResource.resourceId,
        cloudResourceType: service?.resourceTypeCode ?? service?.resourceTypeCode,
        cloudMetadata: bindResource.metadata,
        cloudSyncStatus: 'success',
      });
      message.success('绑定成功');
      setBindOpen(false);
      setBindResource(null);
      setBindCIID('');
      loadData();
    } catch (error) {
      message.error('绑定失败');
    }
  };

  const resourceColumns: ColumnsType<CloudResource> = [
    {
      title: '资源名称',
      width: 180,
      render: (_: unknown, record: CloudResource) =>
        record.resourceName || record.resourceId,
    },
    {
      title: '资源ID',
      dataIndex: 'resourceId',
      width: 200,
    },
    {
      title: '资源类型',
      width: 160,
      render: (_: unknown, record: CloudResource) => {
        const serviceID = record.serviceId ?? record.serviceId;
        const service = serviceMap.get(serviceID);
        return service?.resourceTypeName ?? service?.resourceTypeName ?? `#${serviceID}`;
      },
    },
    {
      title: 'Region',
      dataIndex: 'region',
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
          <Button
            type="link"
            onClick={() => router.push(`/cmdb/cis/create?cloudResourceRefId=${record.id}`)}
          >
            新建CI
          </Button>
          <Button
            type="link"
            onClick={() => {
              setBindResource(record);
              setBindCIID('');
              setBindOpen(true);
            }}
          >
            绑定已有
          </Button>
        </Space>
      ),
    },
  ];

  const ciColumns: ColumnsType<ConfigurationItem> = [
    {
      title: 'CI ID',
      dataIndex: 'id',
      width: 90,
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 200,
      render: (value: string, record: ConfigurationItem) => (
        <Button type="link" onClick={() => router.push(`/cmdb/cis/${record.id}`)}>
          {value}
        </Button>
      ),
    },
    {
      title: '云资源ID',
      width: 200,
      render: (_: unknown, record: ConfigurationItem) =>
        record.cloudResourceId ?? record.cloudResourceId ?? '-',
    },
    {
      title: '云厂商',
      width: 120,
      render: (_: unknown, record: ConfigurationItem) =>
        record.cloudProvider ?? record.cloudProvider ?? '-',
    },
  ];

  return (
    <Card>
      <div style={{ marginBottom: 16 }}>
        <h1 className="text-2xl font-bold">云资源核对</h1>
        <p className="text-gray-500 mt-1">核对云资源与 CMDB 配置项绑定状态，处理待绑定、孤儿和未关联资产。</p>
      </div>

      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: '首页' },
          { title: '配置管理' },
          { title: <a onClick={() => router.push('/cmdb')}>配置项列表</a> },
          { title: '对账中心' },
        ]}
      />

      <Card style={{ marginBottom: 16 }}>
        <Space size="large" wrap>
          {summaryLabels.map(item => (
            <Statistic
              key={item.key}
              title={item.label}
              value={summary ? ((summary as unknown as Record<string, number>)[item.key] ?? 0) : 0}
            />
          ))}
        </Space>
      </Card>

      <Card title="待绑定云资源" style={{ marginBottom: 16 }} loading={loading}>
        <Table
          rowKey="id"
          columns={resourceColumns}
          dataSource={unboundResources}
          pagination={{ pageSize: 8 }}
        />
      </Card>

      <Card title="孤儿配置项（引用资源不存在）" style={{ marginBottom: 16 }} loading={loading}>
        <Table
          rowKey="id"
          columns={ciColumns}
          dataSource={orphanCIs}
          pagination={{ pageSize: 8 }}
        />
      </Card>

      <Card title="未关联配置项（有云资源ID但未绑定）" loading={loading}>
        <Table
          rowKey="id"
          columns={ciColumns}
          dataSource={unlinkedCIs}
          pagination={{ pageSize: 8 }}
        />
      </Card>

      <Modal
        title="绑定已有配置项"
        open={bindOpen}
        onCancel={() => setBindOpen(false)}
        onOk={handleBindExisting}
      >
        <div style={{ marginBottom: 12 }}>请输入配置项ID后进行绑定。</div>
        <Input
          value={bindCIID}
          onChange={event => setBindCIID(event.target.value)}
          placeholder="配置项ID"
        />
      </Modal>
    </Card>
  );
}
