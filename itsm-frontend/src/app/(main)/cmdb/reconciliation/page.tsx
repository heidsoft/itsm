'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Breadcrumb, Button, Card, Input, Modal, Space, Statistic, Table, Tag, message } from 'antd';
import dayjs from 'dayjs';

import { CMDBApi } from '@/modules/cmdb/api';
import type { CloudResource, CloudService, ConfigurationItem, ReconciliationResponse } from '@/modules/cmdb/types';

const summaryLabels = [
  { key: 'resource_total', label: '资源总数' },
  { key: 'bound_resource_count', label: '已绑定资源' },
  { key: 'unbound_resource_count', label: '待绑定资源' },
  { key: 'orphan_ci_count', label: '孤儿CI' },
  { key: 'unlinked_ci_count', label: '未关联CI' },
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

  const serviceMap = useMemo(() => new Map(services.map((item) => [item.id, item])), [services]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [recon, serviceList] = await Promise.all([
        CMDBApi.getReconciliation(),
        CMDBApi.getCloudServices(),
      ]);
      setSummary(recon.summary);
      setUnboundResources(recon.unbound_resources || []);
      setOrphanCIs(recon.orphan_cis || []);
      setUnlinkedCIs(recon.unlinked_cis || []);
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
      const service = serviceMap.get(bindResource.service_id);
      await CMDBApi.updateCI(ciID, {
        cloud_resource_ref_id: bindResource.id,
        cloud_provider: service?.provider,
        cloud_account_id: String(bindResource.cloud_account_id),
        cloud_region: bindResource.region,
        cloud_zone: bindResource.zone,
        cloud_resource_id: bindResource.resource_id,
        cloud_resource_type: service?.resource_type_code,
        cloud_metadata: bindResource.metadata,
        cloud_sync_status: 'success',
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

  const resourceColumns = [
    {
      title: '资源名称',
      dataIndex: 'resource_name',
      width: 180,
      render: (value: string | undefined, record: CloudResource) => value || record.resource_id,
    },
    {
      title: '资源ID',
      dataIndex: 'resource_id',
      width: 200,
    },
    {
      title: '资源类型',
      dataIndex: 'service_id',
      width: 160,
      render: (serviceID: number) => serviceMap.get(serviceID)?.resource_type_name || `#${serviceID}`,
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
      dataIndex: 'last_seen_at',
      width: 160,
      render: (value?: string) => (value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: CloudResource) => (
        <Space>
          <Button type="link" onClick={() => router.push(`/cmdb/cis/create?cloud_resource_ref_id=${record.id}`)}>
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

  const ciColumns = [
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
      dataIndex: 'cloud_resource_id',
      width: 200,
      render: (value?: string) => value || '-',
    },
    {
      title: '云厂商',
      dataIndex: 'cloud_provider',
      width: 120,
      render: (value?: string) => value || '-',
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
          { title: '对账中心' },
        ]}
      />

      <Card style={{ marginBottom: 16 }}>
        <Space size="large" wrap>
          {summaryLabels.map((item) => (
            <Statistic
              key={item.key}
              title={item.label}
              value={summary ? (summary as any)[item.key] ?? 0 : 0}
            />
          ))}
        </Space>
      </Card>

      <Card title="待绑定云资源" style={{ marginBottom: 16 }} loading={loading}>
        <Table
          rowKey="id"
          columns={resourceColumns as any}
          dataSource={unboundResources}
          pagination={{ pageSize: 8 }}
        />
      </Card>

      <Card title="孤儿配置项（引用资源不存在）" style={{ marginBottom: 16 }} loading={loading}>
        <Table rowKey="id" columns={ciColumns as any} dataSource={orphanCIs} pagination={{ pageSize: 8 }} />
      </Card>

      <Card title="未关联配置项（有云资源ID但未绑定）" loading={loading}>
        <Table rowKey="id" columns={ciColumns as any} dataSource={unlinkedCIs} pagination={{ pageSize: 8 }} />
      </Card>

      <Modal
        title="绑定已有配置项"
        open={bindOpen}
        onCancel={() => setBindOpen(false)}
        onOk={handleBindExisting}
      >
        <div style={{ marginBottom: 12 }}>
          请输入配置项ID后进行绑定。
        </div>
        <Input
          value={bindCIID}
          onChange={(event) => setBindCIID(event.target.value)}
          placeholder="配置项ID"
        />
      </Modal>
    </Card>
  );
}
