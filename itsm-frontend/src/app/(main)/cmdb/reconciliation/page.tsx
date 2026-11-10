'use client';

import React, { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Breadcrumb,
  Button,
  Card,
  Modal,
  Space,
  Statistic,
  Table,
  Tag,
} from 'antd';
import dayjs from 'dayjs';
import type { ColumnsType } from 'antd/es/table';

import CISearchSelect, { type CISelectOption } from '@/components/cmdb/CISearchSelect';
import { CMDBApi } from '@/lib/api/cmdb-api';
import { CMDB_KEYS, useCloudServicesQuery, useReconciliationQuery } from '@/lib/hooks/useCMDB';
import { serverErrorMessage } from '@/lib/api/http-client';
import type { CloudResource, ConfigurationItem } from '@/types/biz/cmdb';

const summaryLabels = [
  { key: 'resourceTotal', label: 'Total number of resources' },
  { key: 'boundResourceCount', label: 'Resource bound' },
  { key: 'unboundResourceCount', label: 'Resources to be bound' },
  { key: 'orphanCICount', label: 'Orphan CI' },
  { key: 'unlinkedCICount', label: 'CI not associated' },
];

export default function ReconciliationPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  // React Query：对账结果 + 云服务目录（替代手写 loadData + setState）
  const reconQuery = useReconciliationQuery();
  const servicesQuery = useCloudServicesQuery();

  const loading = reconQuery.isLoading || servicesQuery.isLoading;
  // 失败必须显式呈现：只消费 data 时，接口 500/403 会让「资源总数/孤儿CI」等
  // 指标一律显示 0，看起来像是「对账很干净」，实际是根本没取到数据。
  const reconError = reconQuery.isError;
  const reconErrorText = serverErrorMessage(reconQuery.error, 'Failed to load reconciliation data, please try again');

  const summary = reconQuery.data?.summary ?? null;
  const unboundResources = reconQuery.data?.unboundResources ?? [];
  const orphanCIs = reconQuery.data?.orphanCIs ?? [];
  const unlinkedCIs = reconQuery.data?.unlinkedCIs ?? [];

  const services = servicesQuery.data?.items ?? [];
  const serviceMap = useMemo(() => new Map(services.map(item => [item.id, item])), [services]);

  const [bindOpen, setBindOpen] = useState(false);
  const [bindResource, setBindResource] = useState<CloudResource | null>(null);
  const [bindCIID, setBindCIID] = useState<number | undefined>(undefined);
  const [bindCIOption, setBindCIOption] = useState<CISelectOption | undefined>(undefined);

  // 绑定已有 CI：写回云资源信息后同时失效对账与 CI 缓存
  const bindMutation = useMutation({
    mutationFn: ({ ciId, data }: { ciId: string; data: Parameters<typeof CMDBApi.updateCI>[1] }) =>
      CMDBApi.updateCI(ciId, data),
    onSuccess: (_result, variables) => {
      message.success(`The resource has been bound to the configuration item "${bindCIOption?.name ?? variables.ciId}」`);
      setBindOpen(false);
      setBindResource(null);
      setBindCIID(undefined);
      setBindCIOption(undefined);
      queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'reconciliation'] });
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.cis() });
    },
    onError: error => {
      message.error(error instanceof Error ? error.message || 'Binding failed' : 'Binding failed');
    },
  });

  const handleBindExisting = () => {
    if (!bindResource || !bindCIID) {
      message.warning('Please select the configuration item to be bound first');
      return;
    }
    const service = serviceMap.get(bindResource.serviceId);
    bindMutation.mutate({
      ciId: String(bindCIID),
      data: {
        cloudResourceRefId: bindResource.id,
        cloudProvider: service?.provider,
        cloudAccountId: bindResource.cloudAccountId ? String(bindResource.cloudAccountId) : undefined,
        cloudRegion: bindResource.region,
        cloudZone: bindResource.zone,
        cloudResourceId: bindResource.resourceId,
        cloudResourceType: service?.resourceTypeCode,
        cloudMetadata: bindResource.metadata,
        cloudSyncStatus: 'success',
      },
    });
  };

  const resourceColumns: ColumnsType<CloudResource> = [
    {
      title: 'Resource name',
      width: 180,
      render: (_: unknown, record: CloudResource) =>
        record.resourceName || record.resourceId,
    },
    {
      title: 'Resource ID',
      dataIndex: 'resourceId',
      width: 200,
    },
    {
      title: 'Resource Type',
      width: 160,
      render: (_: unknown, record: CloudResource) => {
        const service = serviceMap.get(record.serviceId);
        return service?.resourceTypeName || `#${record.serviceId}`;
      },
    },
    {
      title: 'Region',
      dataIndex: 'region',
      width: 120,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 110,
      render: (value?: string) => <Tag color={value ? 'green' : 'default'}>{value || 'Unknown'}</Tag>,
    },
    {
      title: 'Recently discovered',
      width: 160,
      render: (_: unknown, record: CloudResource) => {
        return record.lastSeenAt ? dayjs(record.lastSeenAt).format('YYYY-MM-DD HH:mm') : '-';
      },
    },
    {
      title: 'Actions',
      key: 'action',
      width: 160,
      render: (_: unknown, record: CloudResource) => (
        <Space>
          <Button
            type="link"
            onClick={() => router.push(`/cmdb/cis/create?cloudResourceRefId=${record.id}`)}
          >

            Create new CI
          </Button>
          <Button
            type="link"
            onClick={() => {
              setBindResource(record);
              setBindCIID(undefined);
              setBindCIOption(undefined);
              setBindOpen(true);
            }}
          >

            Binding already exists
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
      title: 'Name',
      dataIndex: 'name',
      width: 200,
      render: (value: string, record: ConfigurationItem) => (
        <Button type="link" onClick={() => router.push(`/cmdb/cis/${record.id}`)}>
          {value}
        </Button>
      ),
    },
    {
      title: 'Cloud resource ID',
      width: 200,
      render: (_: unknown, record: ConfigurationItem) => record.cloudResourceId || '-',
    },
    {
      title: 'Cloud Provider',
      width: 120,
      render: (_: unknown, record: ConfigurationItem) => record.cloudProvider || '-',
    },
  ];

  return (
    <Card>
      <div style={{ marginBottom: 16 }}>
        <h1 className="text-2xl font-bold">Cloud resource check</h1>
        <p className="text-gray-500 mt-1">Check the binding status of cloud resources and CMDB configuration items, and handle unbound, orphaned and unassociated assets.</p>
      </div>

      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: 'Home page' },
          { title: 'Configuration Management' },
          { title: <a onClick={() => router.push('/cmdb')}>Configuration Item List</a> },
          { title: 'Reconciliation Center' },
        ]}
      />

      {reconError && (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          title={reconErrorText}
          action={
            <Button size="small" onClick={() => reconQuery.refetch()}>

              Retry
            </Button>
          }
        />
      )}

      <Card style={{ marginBottom: 16 }}>
        <Space size="large" wrap>
          {summaryLabels.map(item => (
            <Statistic
              key={item.key}
              title={item.label}
              // 取数失败时显示「—」而不是 0：0 会被读成「没有问题」，
              // 而真实含义是「不知道」。
              value={
                reconError
                  ? '—'
                  : summary
                    ? ((summary as unknown as Record<string, number>)[item.key] ?? 0)
                    : 0
              }
            />
          ))}
        </Space>
      </Card>

      <Card title="Cloud resources to be bound" style={{ marginBottom: 16 }} loading={loading}>
        <Table
          rowKey="id"
          columns={resourceColumns}
          dataSource={unboundResources}
          pagination={{ pageSize: 8 }}
        />
      </Card>

      <Card title="Orphan configuration item (referenced resource does not exist)" style={{ marginBottom: 16 }} loading={loading}>
        <Table
          rowKey="id"
          columns={ciColumns}
          dataSource={orphanCIs}
          pagination={{ pageSize: 8 }}
        />
      </Card>

      <Card title="Configuration item is not associated (there is a cloud resource ID but it is not bound)" loading={loading}>
        <Table
          rowKey="id"
          columns={ciColumns}
          dataSource={unlinkedCIs}
          pagination={{ pageSize: 8 }}
        />
      </Card>

      <Modal
        title="Bind existing configuration items"
        open={bindOpen}
        onCancel={() => setBindOpen(false)}
        onOk={handleBindExisting}
        okButtonProps={{ disabled: !bindCIID }}
        confirmLoading={bindMutation.isPending}
        okText="Binding"
        destroyOnHidden
      >
        <div style={{ marginBottom: 12 }}>

          Search and select an existing configuration item, and the cloud resource information will be written to the configuration item.
        </div>
        <CISearchSelect
          value={bindCIID}
          onChange={(value, option) => {
            setBindCIID(value);
            setBindCIOption(option);
          }}
          style={{ width: '100%' }}
          placeholder="Enter name to search for configuration items"
        />
        {bindResource && (
          <div className="p-3 bg-gray-50 rounded text-sm text-gray-600" style={{ marginTop: 12 }}>
            <div className="font-medium mb-1">Resources will be bound:</div>
            <div>{bindResource.resourceName || bindResource.resourceId}</div>
            <div className="text-gray-400 mt-1">
              {bindResource.region || '-'} / {bindResource.zone || '-'}
            </div>
          </div>
        )}
      </Modal>
    </Card>
  );
}
