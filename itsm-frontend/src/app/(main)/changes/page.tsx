'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Button, Calendar as AntCalendar, App, Pagination, Select, Card, Empty, Tag, Spin } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import {
  Calendar,
  CheckCircle2,
  ClipboardList,
  Clock,
  Download,
  LayoutGrid,
  Plus,
  RefreshCw,
  RotateCcw,
  Search,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import {
  BusinessPageTemplate,
  type PageStats,
} from '@/components/layout/BusinessPageTemplate';
import ChangeList from '@/components/change/ChangeList';
import { ChangeApi, type Change, type ChangeCalendarItem, type ChangeStatus } from '@/lib/api/change-api';
import type { Change as ChangeRow } from '@/types/biz/change';
import { buildCsv, csvDate, downloadCsv, fetchAllPages, type CsvColumn } from '@/lib/utils/csv-export';
import { useI18n } from '@/lib/i18n/useI18n';
import {
  UnifiedKanbanBoard,
  type KanbanColumnConfig,
} from '@/components/business/UnifiedKanbanBoard';

type View = 'list' | 'kanban' | 'calendar';

export default function ChangesPage() {
  const router = useRouter();
  const { t } = useI18n();
  const { message } = App.useApp();

  const kanbanColumns = useMemo<KanbanColumnConfig<Change>[]>(
    () => [
      { key: 'draft', title: t('changes.kanbanColumns.draft'), color: '#d9d9d9' },
      { key: 'pending', title: t('changes.kanbanColumns.pending'), color: '#fa8c16' },
      { key: 'approved', title: t('changes.kanbanColumns.approved'), color: '#3b82f6' },
      { key: 'scheduled', title: t('changes.kanbanColumns.scheduled'), color: '#722ed1' },
      { key: 'in_progress', title: t('changes.kanbanColumns.in_progress'), color: '#13c2c2' },
      { key: 'completed', title: t('changes.kanbanColumns.completed'), color: '#52c41a' },
      { key: 'cancelled', title: t('changes.kanbanColumns.cancelled'), color: '#ff4d4f' },
    ],
    [t]
  );

  const statusOptions = useMemo(
    () => [
      { value: 'draft', label: t('changes.statusOptions.draft') },
      { value: 'pending', label: t('changes.statusOptions.pending') },
      { value: 'approved', label: t('changes.statusOptions.approved') },
      { value: 'in_progress', label: t('changes.statusOptions.in_progress') },
      { value: 'completed', label: t('changes.statusOptions.completed') },
      { value: 'rejected', label: t('changes.statusOptions.rejected') },
      { value: 'cancelled', label: t('changes.statusOptions.cancelled') },
    ],
    [t]
  );

  const riskOptions = useMemo(
    () => [
      { value: 'high', label: t('changes.riskOptions.high') },
      { value: 'medium', label: t('changes.riskOptions.medium') },
      { value: 'low', label: t('changes.riskOptions.low') },
    ],
    [t]
  );

  const changeStatusConfig = useMemo<Record<string, { label: string; color: string }>>(
    () => ({
      draft: { label: t('changes.draft'), color: 'default' },
      pending: { label: t('changes.pending'), color: 'orange' },
      approved: { label: t('changes.approved'), color: 'blue' },
      scheduled: { label: t('changes.scheduled'), color: 'purple' },
      in_progress: { label: t('changes.inProgress'), color: 'cyan' },
      completed: { label: t('changes.completed'), color: 'green' },
      rejected: { label: t('changes.rejected'), color: 'red' },
      failed: { label: t('changes.failed'), color: 'red' },
      rolled_back: { label: t('changes.rolledBack'), color: 'volcano' },
      cancelled: { label: t('changes.statusOptions.cancelled'), color: 'default' },
    }),
    [t]
  );

  // ====== 状态管理 ======
  const [stats, setStats] = useState({
    total: 0,
    pending: 0,
    inProgress: 0,
    completed: 0,
  });

  const pageStats: PageStats[] = useMemo(
    () => [
      {
        label: t('changes.total'),
        value: stats.total,
        color: '#3b82f6',
        icon: <ClipboardList size={20} strokeWidth={1.8} />,
      },
      {
        label: t('changes.pending'),
        value: stats.pending,
        color: '#fa8c16',
        icon: <Clock size={20} strokeWidth={1.8} />,
      },
      {
        label: t('changes.inProgress'),
        value: stats.inProgress,
        color: '#3b82f6',
        icon: <RefreshCw size={20} strokeWidth={1.8} />,
      },
      {
        label: t('changes.completed'),
        value: stats.completed,
        color: '#52c41a',
        icon: <CheckCircle2 size={20} strokeWidth={1.8} />,
      },
    ],
    [t, stats]
  );

  const [statsLoading, setStatsLoading] = useState(false);

  const [searchKeyword, setSearchKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<ChangeStatus | undefined>(undefined);
  const [riskFilter, setRiskFilter] = useState<string | undefined>(undefined);
  const [showFilters, setShowFilters] = useState(false);

  const [activeView, setActiveView] = useState<View>('list');
  const [changes, setChanges] = useState<Change[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const [calendarLoading, setCalendarLoading] = useState(false);
  const [calendarData, setCalendarData] = useState<ChangeCalendarItem[]>([]);
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([
    dayjs().startOf('month'),
    dayjs().endOf('month'),
  ]);
  const [selectedDate, setSelectedDate] = useState<Dayjs>(dayjs());

  // 分页
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);

  // ====== 数据获取 ======
  const fetchChanges = useCallback(async () => {
    setLoading(true);
    setLoadError(false);
    try {
      const response = await ChangeApi.getChanges({
        page,
        pageSize,
        status: statusFilter,
        riskLevel: riskFilter,
        search: searchKeyword,
      });
      setChanges(response.items);
      // total 是信封必填字段，0 是合法值，禁止用 || 回退到当前页条数。
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to fetch changes:', error);
      message.error(t('changes.getFailed'));
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, statusFilter, riskFilter, searchKeyword, t]);

  // 导出：后端无变更导出端点，按当前筛选条件全量拉取后浏览器端生成 CSV
  // （2026-10-14 审查：原为 message.info 占位）。导出口径 = 当前筛选结果。
  const handleExport = async () => {
    try {
      const rows = await fetchAllPages((p, pageSize) =>
        ChangeApi.getChanges({
          page: p,
          pageSize,
          status: statusFilter,
          riskLevel: riskFilter,
          search: searchKeyword,
        }),
      );
      if (rows.length === 0) {
        message.warning(t('changes.exportEmpty'));
        return;
      }
      const columns: CsvColumn<Change>[] = [
        { title: '编号', render: (row) => `#${row.id}` },
        { title: '标题', render: (row) => row.title },
        { title: '类型', render: (row) => row.type },
        { title: '状态', render: (row) => row.status },
        { title: '优先级', render: (row) => row.priority },
        { title: '风险等级', render: (row) => row.riskLevel },
        { title: '影响范围', render: (row) => row.impactScope },
        { title: '处理人', render: (row) => row.assigneeName },
        { title: '创建人', render: (row) => row.createdByName },
        { title: '计划开始', render: (row) => csvDate(row.plannedStartDate) },
        { title: '计划结束', render: (row) => csvDate(row.plannedEndDate) },
      ];
      downloadCsv(`变更列表-${new Date().toISOString().slice(0, 10)}`, buildCsv(columns, rows));
      message.success(t('changes.exportSuccess', { count: rows.length }));
    } catch (error) {
      console.error('Failed to export changes:', error);
      message.error(t('changes.exportFailed'));
    }
  };

  const fetchChangesForKanban = useCallback(async () => {
    setLoading(true);
    setLoadError(false);
    try {
      const response = await ChangeApi.getChanges({
        page: 1,
        pageSize: 100,
        status: statusFilter,
        riskLevel: riskFilter,
        search: searchKeyword,
      });
      setChanges(response.items);
    } catch (error) {
      console.error('Failed to fetch changes for kanban:', error);
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  }, [statusFilter, riskFilter, searchKeyword]);

  const fetchStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const statsData = await ChangeApi.getChangeStats();
      setStats({
        total: statsData.total || 0,
        pending: statsData.pending || 0,
        inProgress: statsData.inProgress || 0,
        completed: statsData.completed || 0,
      });
    } catch (error) {
      console.error('Failed to fetch change stats:', error);
      message.error(t('changes.getStatsFailed'));
    } finally {
      setStatsLoading(false);
    }
  }, [t]);

  const fetchCalendarData = useCallback(async () => {
    setCalendarLoading(true);
    try {
      const response = await ChangeApi.getCalendar({
        startDate: dateRange[0].format('YYYY-MM-DD'),
        endDate: dateRange[1].format('YYYY-MM-DD'),
        status: statusFilter,
      });
      setCalendarData(response.items || []);
    } catch (error) {
      console.error('Failed to fetch change calendar:', error);
      setCalendarData([]);
      message.error(t('changes.calendarLoadFailed'));
    } finally {
      setCalendarLoading(false);
    }
  }, [dateRange, statusFilter, t]);

  useEffect(() => {
    if (activeView === 'kanban') {
      fetchChangesForKanban();
    } else if (activeView === 'calendar') {
      fetchCalendarData();
    } else if (activeView === 'list') {
      fetchChanges();
    }
  }, [activeView, fetchCalendarData, fetchChanges, fetchChangesForKanban]);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  // ====== 事件处理 ======
  const handleSearch = useCallback((value: string) => {
    setSearchKeyword(value);
    setPage(1);
  }, []);

  const handleCreate = useCallback(() => {
    router.push('/changes/new');
  }, [router]);

  const handleRefresh = useCallback(() => {
    if (activeView === 'kanban') {
      fetchChangesForKanban();
    } else if (activeView === 'calendar') {
      fetchCalendarData();
    } else {
      fetchChanges();
    }
    fetchStats();
  }, [activeView, fetchCalendarData, fetchChanges, fetchChangesForKanban, fetchStats]);

  const handleResetFilters = useCallback(() => {
    setSearchKeyword('');
    setStatusFilter(undefined);
    setRiskFilter(undefined);
    setPage(1);
  }, []);

  const handlePageChange = useCallback((newPage: number, newPageSize: number) => {
    setPage(newPage);
    setPageSize(newPageSize);
  }, []);

  const renderFilters = () => (
    <div className="flex flex-wrap gap-3">
      <Select
        placeholder={t('problems.statusPlaceholder')}
        value={statusFilter}
        onChange={(val) => {
          setStatusFilter(val);
          setPage(1);
        }}
        allowClear
        options={statusOptions}
        style={{ width: 150 }}
      />
      <Select
        placeholder={t('changes.riskPlaceholder')}
        value={riskFilter}
        onChange={(val) => {
          setRiskFilter(val);
          setPage(1);
        }}
        allowClear
        options={riskOptions}
        style={{ width: 150 }}
      />
      <Button onClick={handleResetFilters}>{t('changes.reset')}</Button>
    </div>
  );

  // 受控数据源：列表由本页统一拉取一次，ChangeList 只负责渲染。
  //
  // 此前 ChangeList 内部还会再拉一次（用于表格），本页另拉一份（用于分页与空态），
  // 同一份数据两个源、两次请求；筛选/翻页时两个响应竞态，会出现「分页显示 37 条
  // 但表体还是上一页」这类自相矛盾的界面。传入 items 即声明数据由本页负责。
  const renderListContent = () => (
    <ChangeList
      showHeader={false}
      search={searchKeyword}
      status={statusFilter}
      risk={riskFilter}
      page={page}
      pageSize={pageSize}
      // 边界显式转换：本页持有 API 层的 Change，ChangeList 的列定义按
      // types/biz/change 编写（两者的 type 枚举口径不同，同一份数据在
      // ChangeList 内部取数时本来就要做一次 as unknown as 转换）。
      items={changes as unknown as ChangeRow[]}
      total={total}
      loading={loading}
      onDataChange={fetchChanges}
    />
  );

  const renderKanbanContent = () => (
    <UnifiedKanbanBoard<Change>
      items={changes}
      loading={loading}
      getItemId={(change: Change) => change.id}
      getItemStatus={(change: Change) => change.status || 'draft'}
      getItemTitle={(change: Change) =>
        change.title || t('changes.itemTitleFallback', { id: change.id })
      }
      getItemNumber={(change: Change) => {
        const data = change as unknown as Record<string, unknown>;
        return (data.changeNumber as string) || `C-${change.id}`;
      }}
      getItemDescription={(change: Change) => change.description || ''}
      getItemPriority={(change: Change) => change.priority || 'medium'}
      getItemAssignee={(change: Change) => {
        const assigneeId = change.assigneeId;
        if (!assigneeId) return null;
        return { name: change.assigneeName || t('changes.userFallback', { id: assigneeId }) };
      }}
      getItemCreatedAt={(change: Change) => change.createdAt || ''}
      getItemUpdatedAt={(change: Change) => change.updatedAt || ''}
      onItemClick={(change: Change) => router.push(`/changes/${change.id}`)}
      onItemEdit={(change: Change) => router.push(`/changes/${change.id}/edit`)}
      columnConfigs={kanbanColumns}
      showToolbar={false}
      searchPlaceholder={t('changes.searchPlaceholder')}
      priorityOptions={[
        { value: 'critical', label: t('problems.priorityCritical'), color: 'red' },
        { value: 'high', label: t('incidents.priorityHigh'), color: 'orange' },
        { value: 'medium', label: t('incidents.priorityMedium'), color: 'blue' },
        { value: 'low', label: t('incidents.priorityLow'), color: 'green' },
      ]}
    />
  );

  const getChangesForDate = (date: Dayjs) =>
    calendarData.filter((change) => {
      const startISO = change.plannedStart;
      const endISO = change.plannedEnd || change.plannedStart;
      if (!startISO || !endISO) return false;
      const start = dayjs(startISO);
      const end = dayjs(endISO);
      if (start.year() <= 1 || end.year() <= 1) return false;
      return (
        start.isValid() &&
        end.isValid() &&
        !date.startOf('day').isBefore(start.startOf('day')) &&
        !date.startOf('day').isAfter(end.endOf('day'))
      );
    });

  const renderCalendarContent = () => {
    const selectedChanges = getChangesForDate(selectedDate);

    return (
      <Spin spinning={calendarLoading}>
        <Card styles={{ body: { padding: 16 } }}>
          <AntCalendar
            value={selectedDate}
            onSelect={setSelectedDate}
            onPanelChange={(date) => {
              setSelectedDate(date);
              setDateRange([date.startOf('month'), date.endOf('month')]);
            }}
            cellRender={(current, info) => {
              if (info.type !== 'date') return info.originNode;
              const dayChanges = getChangesForDate(current);

              return (
                <div className="h-full overflow-hidden">
                  {dayChanges.slice(0, 3).map((change) => {
                    const status = changeStatusConfig[change.status] || {
                      label: change.status,
                      color: 'default',
                    };
                    return (
                      <Tag
                        key={change.id}
                        color={status.color}
                        className="mb-1 block max-w-full overflow-hidden text-ellipsis whitespace-nowrap"
                        title={`${change.changeNumber} ${change.title}`}
                      >
                        {change.title}
                      </Tag>
                    );
                  })}
                  {dayChanges.length > 3 && (
                    <span className="text-xs text-gray-500">
                      {t('changes.calendar.moreCount', { count: dayChanges.length - 3 })}
                    </span>
                  )}
                </div>
              );
            }}
          />
        </Card>

        <Card
          title={t('changes.calendar.dateTitle', { date: selectedDate.format('YYYY年MM月DD日') })}
          className="mt-4"
        >
          {selectedChanges.length === 0 ? (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('changes.calendar.noChangesOnDate')} />
          ) : (
            <div className="space-y-2">
              {selectedChanges.map((change) => {
                const status = changeStatusConfig[change.status] || {
                  label: change.status,
                  color: 'default',
                };
                return (
                  <button
                    key={change.id}
                    type="button"
                    className="flex w-full items-center gap-3 rounded-md border border-gray-200 px-3 py-2 text-left transition-colors hover:bg-gray-50"
                    onClick={() => router.push(`/changes/${change.id}`)}
                  >
                    <Tag color={status.color}>{status.label}</Tag>
                    <span className="shrink-0 text-sm text-gray-500">{change.changeNumber}</span>
                    <span className="min-w-0 flex-1 truncate">{change.title}</span>
                    <span className="shrink-0 text-sm text-gray-500">
                      {change.assigneeName || t('changes.calendar.notAssigned')}
                    </span>
                  </button>
                );
              })}
            </div>
          )}
        </Card>
      </Spin>
    );
  };

  const renderContent = () => {
    if (activeView === 'kanban') {
      return renderKanbanContent();
    } else if (activeView === 'calendar') {
      return renderCalendarContent();
    }
    return renderListContent();
  };

  return (
    <BusinessPageTemplate
      title={t('changes.title')}
      description={t('changes.description')}
      stats={pageStats}
      statsLoading={statsLoading}
      searchPlaceholder={t('changes.searchPlaceholder')}
      searchValue={searchKeyword}
      onSearch={handleSearch}
      searchLoading={loading}
      filters={{
        visible: showFilters,
        onToggle: () => setShowFilters(!showFilters),
        content: renderFilters(),
      }}
      showViewSwitch={false}
      primaryAction={{
        label: t('changes.create'),
        onClick: handleCreate,
        icon: <Plus className="w-4 h-4" />,
      }}
      extraActions={[
        {
          key: 'refresh',
          label: t('changes.refresh'),
          icon: <RotateCcw className="w-4 h-4" />,
          onClick: handleRefresh,
        },
        {
          key: 'export',
          label: t('changes.export'),
          icon: <Download className="w-4 h-4" />,
          onClick: handleExport,
        },
      ]}
      loading={activeView === 'calendar' ? false : loading}
      error={activeView !== 'calendar' && loadError}
      errorDescription={t('changes.loadError')}
      onRetry={handleRefresh}
      empty={activeView !== 'calendar' && changes.length === 0 && !loading}
      emptyDescription={t('changes.emptyText')}
      emptyAction={{
        label: t('changes.createFirst'),
        onClick: handleCreate,
      }}
    >
      <div className="flex gap-2 mb-4">
        <Button
          type={activeView === 'list' ? 'primary' : 'default'}
          icon={<Search className="w-4 h-4" />}
          onClick={() => setActiveView('list')}
        >
          {t('changes.listView')}
        </Button>
        <Button
          type={activeView === 'kanban' ? 'primary' : 'default'}
          icon={<LayoutGrid className="w-4 h-4" />}
          onClick={() => setActiveView('kanban')}
        >
          {t('changes.kanbanView')}
        </Button>
        <Button
          type={activeView === 'calendar' ? 'primary' : 'default'}
          icon={<Calendar className="w-4 h-4" />}
          onClick={() => setActiveView('calendar')}
        >
          {t('changes.calendarView')}
        </Button>
      </div>

      {renderContent()}

      {activeView === 'list' && changes.length > 0 && (
        <div className="mt-4 flex justify-end">
          <Pagination
            current={page}
            pageSize={pageSize}
            total={total}
            onChange={handlePageChange}
            showSizeChanger
            showTotal={totalCount => t('changes.totalLabel', { total: totalCount })}
            pageSizeOptions={['10', '20', '50', '100']}
          />
        </div>
      )}
    </BusinessPageTemplate>
  );
}
