'use client';

import React, { useCallback } from 'react';
import { Button, Modal, Pagination, Select, Form, message } from 'antd';
import { Plus, RotateCcw, Download } from 'lucide-react';
import { useRouter } from 'next/navigation';

import { BusinessPageTemplate } from '@/components/layout/BusinessPageTemplate';
import { BatchActionBar } from '@/components/business/BatchActionBar';
import {
  UnifiedKanbanBoard,
  type KanbanColumnConfig,
} from '@/components/business/UnifiedKanbanBoard';
import type { PageStats } from '@/components/layout/BusinessPageTemplate';
import type { Incident } from '@/lib/api/types';
import { IncidentAPI } from '@/lib/api/incident-api';
import { useI18n } from '@/lib/i18n/useI18n';

import { IncidentList } from './components/IncidentList';
import { IncidentFilters } from './components/IncidentFilters';
import { useIncidentsQuery } from '@/lib/hooks/useIncidentsQuery';
import { useIncidentStats } from '@/lib/hooks/useIncidentStats';
import { useIncidentFilters } from '@/lib/hooks/useIncidentFilters';
import { useIncidentBatchOps } from '@/lib/hooks/useIncidentBatchOps';

const KANBAN_COLUMNS: KanbanColumnConfig<Incident>[] = [
  { key: 'new', title: '新建', color: '#3b82f6' },
  { key: 'acknowledged', title: '已确认', color: '#722ed1' },
  { key: 'assigned', title: '已分配', color: '#13c2c2' },
  { key: 'in_progress', title: '处理中', color: '#3b82f6' },
  { key: 'resolved', title: '已解决', color: '#52c41a' },
  { key: 'closed', title: '已关闭', color: '#d9d9d9' },
];

type View = 'list' | 'kanban';

export default function IncidentsPage() {
  const router = useRouter();
  const { t } = useI18n();

  // UI state
  const filters = useIncidentFilters();
  const [activeView, setActiveView] = React.useState<View>('list');

  // Server state — query is bound to filter values, so changing search or
  // status automatically re-fetches. The error message is localized here so
  // the data hook stays free of i18n coupling.
  const query = useIncidentsQuery(filters.values, {
    errorMessage: t('incidents.getFailed') || '加载事件列表失败，请稍后重试',
  });
  const stats = useIncidentStats();

  const batch = useIncidentBatchOps({
    onAfterBatch: async () => {
      await query.refresh();
    },
  });

  const handleView = useCallback(
    (incident: Incident) => {
      router.push(`/incidents/${incident.id}`);
    },
    [router]
  );

  const handleEdit = useCallback(
    (incident: Incident) => {
      router.push(`/incidents/${incident.id}/edit`);
    },
    [router]
  );

  const handleCreate = useCallback(() => {
    router.push('/incidents/create');
  }, [router]);

  const handleRefresh = useCallback(() => {
    void query.refresh();
    void stats.refresh();
  }, [query, stats]);

  const handleSearch = useCallback(
    (value: string) => {
      filters.setSearch(value);
      query.setPage(1);
    },
    [filters, query]
  );

  const handleFilterChange = useCallback(
    (status?: string, priority?: string, source?: string) => {
      filters.setFilter({ status, priority, source });
      query.setPage(1);
    },
    [filters, query]
  );

  const handlePageChange = useCallback(
    (next: number, nextSize: number) => {
      query.setPage(next, nextSize);
    },
    [query]
  );

  const handleAssignConfirm = useCallback(async () => {
    try {
      const values = await batch.assignForm.validateFields();
      batch.closeAssignModal();
      await batch.runBatch(
        batch.selectedRowKeys,
        id => IncidentAPI.assignIncident(Number(id), values.assigneeId),
        '批量分派成功'
      );
    } catch {
      // validation failed — leave modal open so the user can correct input
    }
  }, [batch]);

  const renderListContent = () => {
    if (query.incidents.length === 0 && !query.loading) {
      return (
        <div className='py-12 text-center'>
          <div className='text-gray-400 mb-4'>暂无事件记录</div>
          <Button type='primary' onClick={handleCreate}>
            创建第一个事件
          </Button>
        </div>
      );
    }
    return (
      <>
        <BatchActionBar
          selectedCount={batch.selectedRowKeys.length}
          itemLabel='事件'
          onClear={() => batch.setSelectedRowKeys([])}
          actions={batch.batchActions}
          loading={batch.batchLoading}
        />
        <IncidentList
          // `query.incidents` is incident-api.Incident[] (backend-accurate,
          // optional incidentNumber); IncidentList expects types.Incident[]
          // (required incidentNumber). The two Incident types are separately
          // maintained, so a crossing cast is required here. See review note.
          incidents={query.incidents as unknown as Incident[]}
          loading={query.loading}
          selectedRowKeys={batch.selectedRowKeys}
          onSelectedRowKeysChange={batch.setSelectedRowKeys}
          onEdit={handleEdit}
          onRefresh={handleRefresh}
        />
      </>
    );
  };

  const renderKanbanContent = () => (
    <UnifiedKanbanBoard<Incident>
      items={query.incidents as unknown as Incident[]}
      loading={query.loading}
      getItemId={(incident: Incident) => incident.id}
      getItemStatus={(incident: Incident) => incident.status}
      getItemTitle={(incident: Incident) => incident.title || `事件 #${incident.id}`}
      getItemNumber={(incident: Incident) => incident.incidentNumber || String(incident.id)}
      getItemDescription={(incident: Incident) => incident.description || ''}
      getItemPriority={(incident: Incident) => incident.priority || incident.severity || 'medium'}
      getItemAssignee={(incident: Incident) =>
        incident.assignee
          ? { name: incident.assignee.name || incident.assigneeName || '未分配' }
          : null
      }
      getItemCreatedAt={(incident: Incident) => incident.createdAt}
      getItemUpdatedAt={(incident: Incident) => incident.updatedAt}
      onItemClick={handleView}
      onItemEdit={handleEdit}
      columnConfigs={KANBAN_COLUMNS}
      showToolbar={false}
      priorityOptions={[
        { value: 'critical', label: '紧急', color: 'red' },
        { value: 'high', label: '高', color: 'orange' },
        { value: 'medium', label: '中', color: 'blue' },
        { value: 'low', label: '低', color: 'green' },
      ]}
    />
  );

  return (
    <BusinessPageTemplate
      title='事件管理'
      description='管理和追踪系统中的所有事件记录'
      stats={stats.stats}
      statsLoading={stats.loading}
      searchPlaceholder='搜索事件ID、标题或描述...'
      searchValue={filters.values.search}
      onSearch={handleSearch}
      searchLoading={query.loading}
      filters={{
        visible: filters.visible,
        onToggle: filters.toggleVisible,
        content: (
          <IncidentFilters
            loading={query.loading}
            status={filters.values.status}
            priority={filters.values.priority}
            source={filters.values.source}
            onFilterChange={handleFilterChange}
            onRefresh={handleRefresh}
          />
        ),
      }}
      showViewSwitch={true}
      activeView={activeView}
      onViewChange={view => setActiveView(view as View)}
      primaryAction={{
        label: '新建事件',
        onClick: handleCreate,
        icon: <Plus className='w-4 h-4' />,
      }}
      extraActions={[
        {
          key: 'refresh',
          label: '刷新',
          icon: <RotateCcw className='w-4 h-4' />,
          onClick: handleRefresh,
        },
        {
          key: 'export',
          label: '导出',
          icon: <Download className='w-4 h-4' />,
          onClick: () => message.info('导出功能开发中'),
        },
      ]}
      loading={query.loading}
      error={query.loadError}
      errorDescription='加载事件列表失败'
      onRetry={handleRefresh}
      empty={query.incidents.length === 0 && !query.loading}
      emptyDescription='暂无事件记录'
      emptyAction={{
        label: '创建第一个事件',
        onClick: handleCreate,
      }}
    >
      {activeView === 'list' ? renderListContent() : renderKanbanContent()}

      {query.incidents.length > 0 && (
        <div className='mt-4 flex justify-end'>
          <Pagination
            current={query.page}
            pageSize={query.pageSize}
            total={query.total}
            onChange={handlePageChange}
            showSizeChanger
            showTotal={t => `共 ${t} 条记录`}
            pageSizeOptions={['10', '20', '50', '100']}
          />
        </div>
      )}

      <Modal
        title='批量分派事件'
        open={batch.assignModalOpen}
        onOk={handleAssignConfirm}
        onCancel={batch.closeAssignModal}
        okText='确定分派'
        cancelText='取消'
        confirmLoading={batch.batchLoading}
      >
        <div className='mb-3 text-sm text-gray-500'>
          将为已选择的{' '}
          <span className='text-blue-600 font-semibold'>{batch.selectedRowKeys.length}</span>{' '}
          个事件分派处理人
        </div>
        <Form form={batch.assignForm} layout='vertical'>
          <Form.Item
            name='assigneeId'
            label='处理人'
            rules={[{ required: true, message: '请选择处理人' }]}
          >
            <Select
              placeholder='请选择处理人'
              showSearch
              optionFilterProp='label'
              options={batch.assignUserOptions}
            />
          </Form.Item>
        </Form>
      </Modal>
    </BusinessPageTemplate>
  );
}
