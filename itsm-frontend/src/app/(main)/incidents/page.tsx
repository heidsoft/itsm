'use client';

import React, { useCallback, useMemo } from 'react';
import { Button, Modal, Pagination, Form, message } from 'antd';
import AppSelect from '@/components/ui/AppSelect';
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

type View = 'list' | 'kanban';

export default function IncidentsPage() {
  const router = useRouter();
  const { t } = useI18n();

  const kanbanColumns = useMemo<KanbanColumnConfig<Incident>[]>(
    () => [
      { key: 'new', title: t('incidents.kanbanColumns.new'), color: '#3b82f6' },
      { key: 'acknowledged', title: t('incidents.kanbanColumns.acknowledged'), color: '#722ed1' },
      { key: 'assigned', title: t('incidents.kanbanColumns.assigned'), color: '#13c2c2' },
      { key: 'in_progress', title: t('incidents.kanbanColumns.in_progress'), color: '#3b82f6' },
      { key: 'resolved', title: t('incidents.kanbanColumns.resolved'), color: '#52c41a' },
      { key: 'closed', title: t('incidents.kanbanColumns.closed'), color: '#d9d9d9' },
    ],
    [t]
  );

  const priorityOptions = useMemo(
    () => [
      { value: 'critical', label: t('incidents.priorityCritical'), color: 'red' },
      { value: 'high', label: t('incidents.priorityHigh'), color: 'orange' },
      { value: 'medium', label: t('incidents.priorityMedium'), color: 'blue' },
      { value: 'low', label: t('incidents.priorityLow'), color: 'green' },
    ],
    [t]
  );

  // UI state
  const filters = useIncidentFilters();
  const [activeView, setActiveView] = React.useState<View>('list');

  // Server state — query is bound to filter values, so changing search or
  // status automatically re-fetches. The error message is localized here so
  // the data hook stays free of i18n coupling.
  const query = useIncidentsQuery(filters.values, {
    errorMessage: t('incidents.getFailed'),
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
        t('incidents.batchAssignSuccess')
      );
    } catch {
      // validation failed — leave modal open so the user can correct input
    }
  }, [batch, t]);

  const renderListContent = () => {
    if (query.incidents.length === 0 && !query.loading) {
      return (
        <div className='py-12 text-center'>
          <div className='text-gray-400 mb-4'>{t('incidents.emptyText')}</div>
          <Button type='primary' onClick={handleCreate}>
            {t('incidents.createFirst')}
          </Button>
        </div>
      );
    }
    return (
      <>
        <BatchActionBar
          selectedCount={batch.selectedRowKeys.length}
          itemLabel={t('incidents.title')}
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
      getItemTitle={(incident: Incident) =>
        incident.title || t('incidents.itemTitleFallback', { id: incident.id })
      }
      getItemNumber={(incident: Incident) => incident.incidentNumber || String(incident.id)}
      getItemDescription={(incident: Incident) => incident.description || ''}
      getItemPriority={(incident: Incident) => incident.priority || incident.severity || 'medium'}
      getItemAssignee={(incident: Incident) =>
        incident.assignee
          ? { name: incident.assignee.name || incident.assigneeName || t('incidents.unassigned') }
          : null
      }
      getItemCreatedAt={(incident: Incident) => incident.createdAt}
      getItemUpdatedAt={(incident: Incident) => incident.updatedAt}
      onItemClick={handleView}
      onItemEdit={handleEdit}
      columnConfigs={kanbanColumns}
      showToolbar={false}
      priorityOptions={priorityOptions}
    />
  );

  return (
    <BusinessPageTemplate
      title={t('incidents.title')}
      description={t('incidents.description')}
      stats={stats.stats}
      statsLoading={stats.loading}
      searchPlaceholder={t('incidents.searchPlaceholder')}
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
        label: t('incidents.create'),
        onClick: handleCreate,
        icon: <Plus className='w-4 h-4' />,
      }}
      extraActions={[
        {
          key: 'refresh',
          label: t('incidents.refresh'),
          icon: <RotateCcw className='w-4 h-4' />,
          onClick: handleRefresh,
        },
        {
          key: 'export',
          label: t('incidents.export'),
          icon: <Download className='w-4 h-4' />,
          onClick: () => message.info(t('incidents.exportPending')),
        },
      ]}
      loading={query.loading}
      error={query.loadError}
      errorDescription={t('incidents.emptyDescription')}
      onRetry={handleRefresh}
      empty={query.incidents.length === 0 && !query.loading}
      emptyDescription={t('incidents.emptyText')}
      emptyAction={{
        label: t('incidents.createFirst'),
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
            showTotal={total => t('incidents.totalLabel', { total })}
            pageSizeOptions={['10', '20', '50', '100']}
          />
        </div>
      )}

      <Modal
        title={t('incidents.batchAssign')}
        open={batch.assignModalOpen}
        onOk={handleAssignConfirm}
        onCancel={batch.closeAssignModal}
        okText={t('incidents.batchAssignConfirm')}
        cancelText={t('common.cancel')}
        confirmLoading={batch.batchLoading}
      >
        <div className='mb-3 text-sm text-gray-500'>
          {t('incidents.batchAssignDesc', { count: batch.selectedRowKeys.length })}
        </div>
        <Form form={batch.assignForm} layout='vertical'>
          <Form.Item
            name='assigneeId'
            label={t('incidents.assignee')}
            rules={[{ required: true, message: t('incidents.assigneeRequired') }]}
          >
            <AppSelect
              placeholder={t('incidents.selectAssignee')}
              options={batch.assignUserOptions}
            />
          </Form.Item>
        </Form>
      </Modal>
    </BusinessPageTemplate>
  );
}
