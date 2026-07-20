'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Button, Space, message, Pagination, Select } from 'antd';
import { Plus, Search, RotateCcw, Download, Filter } from 'lucide-react';
import { LayoutGrid } from 'lucide-react';
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

// 看板列配置
const KANBAN_COLUMNS: KanbanColumnConfig<Problem>[] = [
  { key: 'open', title: '待处理', color: '#ff4d4f' },
  { key: 'investigating', title: '调查中', color: '#722ed1' },
  { key: 'identified', title: '已识别', color: '#fa8c16' },
  { key: 'resolved', title: '已解决', color: '#52c41a' },
  { key: 'closed', title: '已关闭', color: '#d9d9d9' },
];

// 筛选选项
const statusOptions = [
  { value: 'open', label: '待处理' },
  { value: 'investigating', label: '调查中' },
  { value: 'identified', label: '已识别' },
  { value: 'resolved', label: '已解决' },
  { value: 'closed', label: '已关闭' },
];

const priorityOptions = [
  { value: 'critical', label: '紧急' },
  { value: 'high', label: '高' },
  { value: 'medium', label: '中' },
  { value: 'low', label: '低' },
];

export default function ProblemListPage() {
  const router = useRouter();
  const { t } = useI18n();

  // ====== 状态管理 ======
  const [stats, setStats] = useState({
    total: 0,
    open: 0,
    inProgress: 0,
    resolved: 0,
  });
  const [statsLoading, setStatsLoading] = useState(false);

  const [searchKeyword, setSearchKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined);
  const [priorityFilter, setPriorityFilter] = useState<string | undefined>(undefined);
  const [showFilters, setShowFilters] = useState(false);

  const [activeView, setActiveView] = useState<'list' | 'kanban'>('list');
  const [problems, setProblems] = useState<Problem[]>([]);
  const [loading, setLoading] = useState(false);

  // 分页
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);

  // ====== 数据获取 ======
  const fetchProblems = useCallback(async () => {
    setLoading(true);
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
      message.error('加载问题列表失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, statusFilter, priorityFilter, searchKeyword]);

  const fetchProblemsForKanban = useCallback(async () => {
    setLoading(true);
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
      setProblems([]);
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
      message.error(t('problems.getStatsFailed') || '获取问题统计失败，请稍后重试');
    } finally {
      setStatsLoading(false);
    }
  }, [t]);

  // 加载数据
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

  // ====== 事件处理 ======
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

  // ====== 统计数据转换 ======
  const pageStats: PageStats[] = [
    {
      label: '总问题数',
      value: stats.total,
      color: '#1890ff',
      icon: <span className="text-2xl">🐛</span>,
    },
    {
      label: '待处理',
      value: stats.open,
      color: '#ff4d4f',
      icon: <span className="text-2xl">⚠️</span>,
    },
    {
      label: '调查中',
      value: stats.inProgress,
      color: '#fa8c16',
      icon: <span className="text-2xl">🔍</span>,
    },
    {
      label: '已解决',
      value: stats.resolved,
      color: '#52c41a',
      icon: <span className="text-2xl">✅</span>,
    },
  ];

  // ====== 筛选面板内容 ======
  const renderFilters = () => (
    <div className="flex flex-wrap gap-3">
      <Select
        placeholder="状态筛选"
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
        placeholder="优先级筛选"
        value={priorityFilter}
        onChange={(val) => {
          setPriorityFilter(val);
          setPage(1);
        }}
        allowClear
        options={priorityOptions}
        style={{ width: 150 }}
      />
      <Button onClick={handleResetFilters}>重置</Button>
    </div>
  );

  // ====== 渲染内容 ======
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
      getItemTitle={(problem: Problem) => problem.title || `问题 #${problem.id}`}
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
        return { name: assigneeName || `用户 #${assigneeId}` };
      }}
      getItemCreatedAt={(problem: Problem) => problem.createdAt || ''}
      getItemUpdatedAt={(problem: Problem) => problem.updatedAt || ''}
      onItemClick={(problem: Problem) => router.push(`/problems/${problem.id}`)}
      onItemEdit={(problem: Problem) => router.push(`/problems/${problem.id}/edit`)}
      columnConfigs={KANBAN_COLUMNS}
      showToolbar={false}
      searchPlaceholder="搜索问题标题或描述..."
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
      // 页面信息
      title="问题管理"
      description="识别、分析和消除事件发生的根本原因"

      // 统计
      stats={pageStats}
      statsLoading={statsLoading}

      // 搜索
      searchPlaceholder="搜索问题标题或描述..."
      searchValue={searchKeyword}
      onSearch={handleSearch}
      searchLoading={loading}

      // 筛选
      filters={{
        visible: showFilters,
        onToggle: () => setShowFilters(!showFilters),
        content: renderFilters(),
      }}

      // 视图切换
      showViewSwitch={true}
      activeView={activeView}
      onViewChange={setActiveView}

      // 操作
      primaryAction={{
        label: '新建问题',
        onClick: handleCreate,
        icon: <Plus className="w-4 h-4" />,
      }}

      extraActions={[
        {
          key: 'refresh',
          label: '刷新',
          icon: <RotateCcw className="w-4 h-4" />,
          onClick: handleRefresh,
        },
        {
          key: 'export',
          label: '导出',
          icon: <Download className="w-4 h-4" />,
          onClick: () => message.info('导出功能开发中'),
        },
      ]}

      // 内容
      loading={loading}
      empty={problems.length === 0 && !loading}
      emptyDescription="暂无问题记录"
      emptyAction={{
        label: '创建第一个问题',
        onClick: handleCreate,
      }}
    >
      {/* 视图内容 */}
      {activeView === 'list' ? renderListContent() : renderKanbanContent()}

      {/* 分页 */}
      {activeView === 'list' && problems.length > 0 && (
        <div className="mt-4 flex justify-end">
          <Pagination
            current={page}
            pageSize={pageSize}
            total={total}
            onChange={handlePageChange}
            showSizeChanger
            showTotal={(total) => `共 ${total} 条记录`}
            pageSizeOptions={['10', '20', '50', '100']}
          />
        </div>
      )}
    </BusinessPageTemplate>
  );
}
