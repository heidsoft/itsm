'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Table, Select, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useRouter } from 'next/navigation';
import { BusinessPageTemplate } from '@/components/layout/BusinessPageTemplate';
import { useAuthStore } from '@/lib/store/auth-store';
import { useI18n } from '@/lib/i18n/useI18n';
import {
  queryWorkbench,
  type WorkbenchItem,
  type WorkbenchQuery,
} from '@/lib/api/workbench-api';

const { Text } = Typography;

const PHASE_OPTIONS = ['draft', 'submitted', 'active', 'resolved', 'closed'] as const;
const PRIORITY_OPTIONS = ['critical', 'high', 'medium', 'low'] as const;
const RECORD_TYPE_OPTIONS = ['incident', 'change', 'problem', 'ticket'] as const;

const DEFAULT_PHASES = ['draft', 'submitted', 'active'];

const priorityColor: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'blue',
  low: 'green',
};

const phaseColor: Record<string, string> = {
  draft: 'default',
  submitted: 'processing',
  active: 'processing',
  resolved: 'success',
  closed: 'default',
};

const recordTypeColor: Record<string, string> = {
  incident: '#f5222d',
  change: '#1890ff',
  problem: '#722ed1',
  ticket: '#13c2c2',
};

const detailPath: Record<string, string> = {
  incident: '/incidents',
  change: '/changes',
  problem: '/problems',
  ticket: '/tickets',
};

export default function WorkbenchPage() {
  const router = useRouter();
  const { t } = useI18n();
  const user = useAuthStore(s => s.user);

  const [items, setItems] = useState<WorkbenchItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const [phases, setPhases] = useState<string[]>(DEFAULT_PHASES);
  const [priorities, setPriorities] = useState<string[]>([]);
  const [recordTypes, setRecordTypes] = useState<string[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const query: WorkbenchQuery = {
        page,
        pageSize,
        phase: phases.length ? phases : undefined,
        priority: priorities.length ? priorities : undefined,
        recordType: recordTypes.length ? recordTypes : undefined,
      };
      if (user?.id) {
        query.assigneeId = user.id;
      }
      const resp = await queryWorkbench(query);
      setItems(resp.items);
      setTotal(resp.total);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load workbench');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, phases, priorities, recordTypes, user?.id]);

  useEffect(() => {
    void fetchData();
  }, [fetchData]);

  const handleRowClick = useCallback(
    (record: WorkbenchItem) => {
      const base = detailPath[record.recordType];
      if (base) {
        router.push(`${base}/${record.id}`);
      }
    },
    [router]
  );

  const handleRefresh = useCallback(() => {
    void fetchData();
  }, [fetchData]);

  const columns = useMemo<ColumnsType<WorkbenchItem>>(
    () => [
      {
        title: t('workbench.columns.type'),
        dataIndex: 'recordType',
        width: 100,
        render: (type: string) => (
          <Tag color={recordTypeColor[type]}>
            {t(`workbench.recordTypes.${type}`)}
          </Tag>
        ),
      },
      {
        title: t('workbench.columns.title'),
        dataIndex: 'title',
        ellipsis: true,
        render: (text: string) => <Text strong>{text}</Text>,
      },
      {
        title: t('workbench.columns.priority'),
        dataIndex: 'priority',
        width: 100,
        render: (p: string) => (
          <Tag color={priorityColor[p]}>
            {t(`workbench.phases.${p}`) === p ? p : t(`workbench.phases.${p}`)}
          </Tag>
        ),
      },
      {
        title: t('workbench.columns.status'),
        dataIndex: 'status',
        width: 120,
      },
      {
        title: t('workbench.columns.phase'),
        dataIndex: 'phase',
        width: 100,
        render: (p: string) => (
          <Tag color={phaseColor[p]}>
            {t(`workbench.phases.${p}`)}
          </Tag>
        ),
      },
      {
        title: t('workbench.columns.createdAt'),
        dataIndex: 'createdAt',
        width: 180,
        render: (v: string) => new Date(v).toLocaleString(),
      },
    ],
    [t]
  );

  const filterContent = (
    <div className='flex flex-wrap gap-3'>
      <div>
        <Text type='secondary' className='mr-2'>
          {t('workbench.filters.phase')}
        </Text>
        <Select
          mode='multiple'
          allowClear
          placeholder={t('workbench.filters.all')}
          value={phases}
          onChange={v => {
            setPhases(v);
            setPage(1);
          }}
          style={{ minWidth: 180 }}
          options={PHASE_OPTIONS.map(p => ({
            value: p,
            label: t(`workbench.phases.${p}`),
          }))}
        />
      </div>
      <div>
        <Text type='secondary' className='mr-2'>
          {t('workbench.filters.priority')}
        </Text>
        <Select
          mode='multiple'
          allowClear
          placeholder={t('workbench.filters.all')}
          value={priorities}
          onChange={v => {
            setPriorities(v);
            setPage(1);
          }}
          style={{ minWidth: 160 }}
          options={PRIORITY_OPTIONS.map(p => ({
            value: p,
            label: p,
          }))}
        />
      </div>
      <div>
        <Text type='secondary' className='mr-2'>
          {t('workbench.filters.recordType')}
        </Text>
        <Select
          mode='multiple'
          allowClear
          placeholder={t('workbench.filters.all')}
          value={recordTypes}
          onChange={v => {
            setRecordTypes(v);
            setPage(1);
          }}
          style={{ minWidth: 180 }}
          options={RECORD_TYPE_OPTIONS.map(r => ({
            value: r,
            label: t(`workbench.recordTypes.${r}`),
          }))}
        />
      </div>
    </div>
  );

  return (
    <BusinessPageTemplate
      title={t('workbench.title')}
      description={t('workbench.description')}
      filters={{
        visible: true,
        onToggle: () => {},
        content: filterContent,
      }}
      extraActions={[
        {
          key: 'refresh',
          label: t('incidents.refresh'),
          onClick: handleRefresh,
        },
      ]}
      loading={loading}
      error={!!error}
      empty={items.length === 0 && !loading}
      emptyDescription={t('workbench.emptyDescription')}
    >
      <Table<WorkbenchItem>
        rowKey='id'
        columns={columns}
        dataSource={items}
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
          showSizeChanger: true,
          showTotal: (t: number) => `${t} items`,
        }}
        onRow={record => ({
          onClick: () => handleRowClick(record),
          style: { cursor: 'pointer' },
        })}
      />
    </BusinessPageTemplate>
  );
}
