'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Button, message, Pagination, Select } from 'antd';
import {
  Plus,
  RotateCcw,
  Download,
  Bug,
  CircleAlert,
  ScanSearch,
  CircleCheckBig,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import {
  BusinessPageTemplate,
  type PageStats,
} from '@/components/layout/BusinessPageTemplate';
import ProblemList from '@/components/problem/ProblemList';
import { ProblemApi, type Problem } from '@/lib/api/problem-api';
import { useI18n } from '@/lib/i18n/useI18n';
import {
  UnifiedKanbanBoard,
  type KanbanColumnConfig,
} from '@/components/business/UnifiedKanbanBoard';

type View = 'list' | 'kanban';

export default function ProblemListPage() {
  const router = useRouter();
  const { t } = useI18n();

  const kanbanColumns = useMemo<KanbanColumnConfig<Problem>[]>(
    () => [
      { key: 'open', title: t('problems.kanbanColumns.open'), color: '#ff4d4f' },
      { key: 'investigating', title: t('problems.kanbanColumns.investigating'), color: '#722ed1' },
      { key: 'identified', title: t('problems.kanbanColumns.identified'), color: '#fa8c16' },
      { key: 'resolved', title: t('problems.kanbanColumns.resolved'), color: '#52c41a' },
      { key: 'closed', title: t('problems.kanbanColumns.closed'), color: '#d9d9d9' },
    ],
    [t]
  );

  const statusOptions = useMemo(
    () => [
      { value: 'open', label: t('problems.statusOptions.open') },
      { value: 'investigating', label: t('problems.statusOptions.investigating') },
      { value: 'identified', label: t('problems.statusOptions.identified') },
      { value: 'resolved', label: t('problems.statusOptions.resolved') },
      { value: 'closed', label: t('problems.statusOptions.closed') },
    ],
    [t]
  );

  const priorityOptions = useMemo(
    () => [
      { value: 'critical', label: t('problems.priorityCritical') },
      { value: 'high', label: t('incidents.priorityHigh') },
      { value: 'medium', label: t('incidents.priorityMedium') },
      { value: 'low', label: t('incidents.priorityLow') },
    ],
    [t]
  );

  // ====== 状态管理 ======
  const [stats, setStats] = useState({
    total: 0,
    open: 0,
    inProgress: 0,
    resolved: 0,
  });

  const pageStats: PageStats[] = useMemo(
    () => [
      {
        label: t('problems.total'),
        value: stats.total,
        color: '#3b82f6',
        icon: <Bug size={20} strokeWidth={1.8} />,
      },
      {
        label: t('problems.open'),
        value: stats.open,
        color: '#ff4d4f',
        icon: <CircleAlert size={20} strokeWidth={1.8} />,
      },
      {
        label: t('problems.investigatingLabel'),
        value: stats.inProgress,
        color: '#fa8c16',
        icon: <ScanSearch size={20} strokeWidth={1.8} />,
      },
      {
        label: t('problems.resolved'),
        value: stats.resolved,
        color: '#52c41a',
        icon: <CircleCheckBig size={20} strokeWidth={1.8} />,
      },
    ],
    [t, stats]
  );

  const [statsLoading, setStatsLoading] = useState(false);

  const [searchKeyword, setSearchKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined);
  const [priorityFilter, setPriorityFilter] = useState<string | undefined>(undefined);
  const [showFilters, setShowFilters] = useState(false);

  const [activeView, setActiveView] = useState<View>('list');
  const [problems, setProblems] = useState<Problem[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState(false);

  // 分页
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);

  // ====== 数据获取 ======
  const fetchProblems = useCallback(async () => {
    setLoading(true);
    setLoadError(false);
    try {
      const response = await ProblemApi.getProblems({
        page,
        pageSize,
        status: statusFilter,
        priority: priorityFilter,
        search: searchKeyword,
      });
      const items = response.problems || [];
      setProblems(items);
      setTotal(response.total || items.length);
    } catch (error) {
      console.error('Failed to fetch problems:', error);
      message.error(t('problems.getFailed'));
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, statusFilter, priorityFilter, searchKeyword, t]);

  const fetchProblemsForKanban = useCallback(async () => {
    setLoading(true);
    setLoadError(false);
    try {
      const response = await ProblemApi.getProblems({
        page: 1,
        pageSize: 100,
        status: statusFilter,
        priority: priorityFilter,
        search: searchKeyword,
      });
      setProblems(response.problems || []);
    } catch (error) {
      console.error('Failed to fetch problems for kanban:', error);
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  }, [statusFilter, priorityFilter, searchKeyword]);

  const fetchStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const statsData = await ProblemApi.getProblemStats();
      setStats({
        total: statsData.total || 0,
        open: statsData.open || 0,
        inProgress: statsData.inProgress || 0,
        resolved: statsData.resolved || 0,
      });
    } catch (error) {
      console.error('Failed to fetch problem stats:', error);
      message.error(t('problems.getStatsFailed'));
    } finally {
      setStatsLoading(false);
    }
  }, [t]);

  useEffect(() => {
    if (activeView === 'kanban') {
      fetchProblemsForKanban();
    } else {
      fetchProblems();
    }
  }, [activeView, fetchProblems, fetchProblemsForKanban]);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  const handleSearch = useCallback((value: string) => {
    setSearchKeyword(value);
    setPage(1);
  }, []);

  const handleCreate = useCallback(() => {
    router.push('/problems/new');
  }, [router]);

  const handleRefresh = useCallback(() => {
    if (activeView === 'kanban') {
      fetchProblemsForKanban();
    } else {
      fetchProblems();
    }
    fetchStats();
  }, [activeView, fetchProblems, fetchProblemsForKanban, fetchStats]);

  const handleResetFilters = useCallback(() => {
    setSearchKeyword('');
    setStatusFilter(undefined);
    setPriorityFilter(undefined);
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
        placeholder={t('problems.priorityPlaceholder')}
        value={priorityFilter}
        onChange={(val) => {
          setPriorityFilter(val);
          setPage(1);
        }}
        allowClear
        options={priorityOptions}
        style={{ width: 150 }}
      />
      <Button onClick={handleResetFilters}>{t('problems.reset')}</Button>
    </div>
  );

  const renderListContent = () => (
    <ProblemList
      showHeader={false}
      keyword={searchKeyword}
      status={statusFilter}
      priority={priorityFilter}
    />
  );

  const renderKanbanContent = () => (
    <UnifiedKanbanBoard<Problem>
      items={problems}
      loading={loading}
      getItemId={(problem: Problem) => problem.id}
      getItemStatus={(problem: Problem) => problem.status || 'open'}
      getItemTitle={(problem: Problem) =>
        problem.title || t('problems.itemTitleFallback', { id: problem.id })
      }
      getItemNumber={(problem: Problem) => {
        const data = problem as unknown as Record<string, unknown>;
        return (data.problemNumber as string) || `P-${problem.id}`;
      }}
      getItemDescription={(problem: Problem) => problem.description || ''}
      getItemPriority={(problem: Problem) => problem.priority || problem.severity || 'medium'}
      getItemAssignee={(problem: Problem) => {
        const assigneeId = problem.assigneeId;
        if (!assigneeId) return null;
        const data = problem as unknown as Record<string, unknown>;
        const assigneeName = data.assigneeName as string;
        return { name: assigneeName || t('problems.userFallback', { id: assigneeId }) };
      }}
      getItemCreatedAt={(problem: Problem) => problem.createdAt || ''}
      getItemUpdatedAt={(problem: Problem) => problem.updatedAt || ''}
      onItemClick={(problem: Problem) => router.push(`/problems/${problem.id}`)}
      onItemEdit={(problem: Problem) => router.push(`/problems/${problem.id}/edit`)}
      columnConfigs={kanbanColumns}
      showToolbar={false}
      searchPlaceholder={t('problems.searchPlaceholder')}
      priorityOptions={[
        { value: 'critical', label: t('problems.priorityCritical'), color: 'red' },
        { value: 'high', label: t('incidents.priorityHigh'), color: 'orange' },
        { value: 'medium', label: t('incidents.priorityMedium'), color: 'blue' },
        { value: 'low', label: t('incidents.priorityLow'), color: 'green' },
      ]}
    />
  );

  return (
    <BusinessPageTemplate
      title={t('problems.title')}
      description={t('problems.description')}
      stats={pageStats}
      statsLoading={statsLoading}
      searchPlaceholder={t('problems.searchPlaceholder')}
      searchValue={searchKeyword}
      onSearch={handleSearch}
      searchLoading={loading}
      filters={{
        visible: showFilters,
        onToggle: () => setShowFilters(!showFilters),
        content: renderFilters(),
      }}
      showViewSwitch={true}
      activeView={activeView}
      onViewChange={view => setActiveView(view as View)}
      primaryAction={{
        label: t('problems.create'),
        onClick: handleCreate,
        icon: <Plus className="w-4 h-4" />,
      }}
      extraActions={[
        {
          key: 'refresh',
          label: t('problems.refresh'),
          icon: <RotateCcw className="w-4 h-4" />,
          onClick: handleRefresh,
        },
        {
          key: 'export',
          label: t('problems.export'),
          icon: <Download className="w-4 h-4" />,
          onClick: () => message.info(t('problems.exportPending')),
        },
      ]}
      loading={loading}
      error={loadError}
      errorDescription={t('problems.emptyDescription')}
      onRetry={handleRefresh}
      empty={problems.length === 0 && !loading}
      emptyDescription={t('problems.emptyText')}
      emptyAction={{
        label: t('problems.createFirst'),
        onClick: handleCreate,
      }}
    >
      {activeView === 'list' ? renderListContent() : renderKanbanContent()}

      {activeView === 'list' && problems.length > 0 && (
        <div className="mt-4 flex justify-end">
          <Pagination
            current={page}
            pageSize={pageSize}
            total={total}
            onChange={handlePageChange}
            showSizeChanger
            showTotal={totalCount => t('problems.totalLabel', { total: totalCount })}
            pageSizeOptions={['10', '20', '50', '100']}
          />
        </div>
      )}
    </BusinessPageTemplate>
  );
}
