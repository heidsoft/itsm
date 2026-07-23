'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Button, Calendar as AntCalendar, message, Pagination, Select, Card, Empty, Tag, Spin } from 'antd';
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
import { ChangeApi, type Change, type ChangeCalendarItem } from '@/lib/api/change-api';
import { useI18n } from '@/lib/i18n/useI18n';
import {
  UnifiedKanbanBoard,
  type KanbanColumnConfig,
} from '@/components/business/UnifiedKanbanBoard';

// 看板列配置
const KANBAN_COLUMNS: KanbanColumnConfig<Change>[] = [
  { key: 'draft', title: '草稿', color: '#d9d9d9' },
  { key: 'pending', title: '待审批', color: '#fa8c16' },
  { key: 'approved', title: '已批准', color: '#1890ff' },
  { key: 'scheduled', title: '已排期', color: '#722ed1' },
  { key: 'in_progress', title: '实施中', color: '#13c2c2' },
  { key: 'completed', title: '已完成', color: '#52c41a' },
  { key: 'cancelled', title: '已取消', color: '#ff4d4f' },
];

// 筛选选项
const statusOptions = [
  { value: 'draft', label: '草稿' },
  { value: 'pending', label: '待审批' },
  { value: 'approved', label: '已批准' },
  { value: 'in_progress', label: '实施中' },
  { value: 'completed', label: '已完成' },
  { value: 'rejected', label: '已拒绝' },
  { value: 'cancelled', label: '已取消' },
];

const riskOptions = [
  { value: 'high', label: '高风险' },
  { value: 'medium', label: '中风险' },
  { value: 'low', label: '低风险' },
];

const CHANGE_STATUS_CONFIG: Record<string, { label: string; color: string }> = {
  draft: { label: '草稿', color: 'default' },
  pending: { label: '待审批', color: 'orange' },
  approved: { label: '已批准', color: 'blue' },
  scheduled: { label: '已排期', color: 'purple' },
  in_progress: { label: '实施中', color: 'cyan' },
  completed: { label: '已完成', color: 'green' },
  rejected: { label: '已拒绝', color: 'red' },
  failed: { label: '失败', color: 'red' },
  rolled_back: { label: '已回滚', color: 'volcano' },
  cancelled: { label: '已取消', color: 'default' },
};

export default function ChangesPage() {
  const router = useRouter();
  const { t } = useI18n();

  // ====== 状态管理 ======
  const [stats, setStats] = useState({
    total: 0,
    pending: 0,
    inProgress: 0,
    completed: 0,
  });
  const [statsLoading, setStatsLoading] = useState(false);

  const [searchKeyword, setSearchKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined);
  const [riskFilter, setRiskFilter] = useState<string | undefined>(undefined);
  const [showFilters, setShowFilters] = useState(false);

  const [activeView, setActiveView] = useState<'list' | 'kanban' | 'calendar'>('list');
  const [changes, setChanges] = useState<Change[]>([]);
  const [loading, setLoading] = useState(false);
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
    try {
      const response = await ChangeApi.getChanges({
        page,
        pageSize,
        status: statusFilter as any,
        risk: riskFilter,
        search: searchKeyword,
      });
      const items = response.changes || [];
      setChanges(items);
      setTotal(response.total || items.length);
    } catch (error) {
      console.error('Failed to fetch changes:', error);
      message.error('加载变更列表失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, statusFilter, riskFilter, searchKeyword]);

  const fetchChangesForKanban = useCallback(async () => {
    setLoading(true);
    try {
      const response = await ChangeApi.getChanges({
        page: 1,
        pageSize: 100,
        status: statusFilter as any,
        risk: riskFilter,
        search: searchKeyword,
      });
      setChanges(response.changes || []);
    } catch (error) {
      console.error('Failed to fetch changes for kanban:', error);
      setChanges([]);
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
      message.error(t('changes.getStatsFailed') || '获取变更统计失败，请稍后重试');
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
      message.error('加载变更日历失败，请稍后重试');
    } finally {
      setCalendarLoading(false);
    }
  }, [dateRange, statusFilter]);

  // 加载数据
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

  // ====== 统计数据转换 ======
  const pageStats: PageStats[] = [
    {
      label: '总变更数',
      value: stats.total,
      color: '#1890ff',
      icon: <ClipboardList size={20} strokeWidth={1.8} />,
    },
    {
      label: '待审批',
      value: stats.pending,
      color: '#fa8c16',
      icon: <Clock size={20} strokeWidth={1.8} />,
    },
    {
      label: '进行中',
      value: stats.inProgress,
      color: '#1890ff',
      icon: <RefreshCw size={20} strokeWidth={1.8} />,
    },
    {
      label: '已完成',
      value: stats.completed,
      color: '#52c41a',
      icon: <CheckCircle2 size={20} strokeWidth={1.8} />,
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
        placeholder="风险等级"
        value={riskFilter}
        onChange={(val) => {
          setRiskFilter(val);
          setPage(1);
        }}
        allowClear
        options={riskOptions}
        style={{ width: 150 }}
      />
      <Button onClick={handleResetFilters}>重置</Button>
    </div>
  );

  // ====== 渲染内容 ======
  const renderListContent = () => (
    <ChangeList
      showHeader={false}
      search={searchKeyword}
      status={statusFilter}
      risk={riskFilter}
    />
  );

  const renderKanbanContent = () => (
    <UnifiedKanbanBoard<Change>
      items={changes}
      loading={loading}
      getItemId={(change: Change) => change.id}
      getItemStatus={(change: Change) => change.status || 'draft'}
      getItemTitle={(change: Change) => change.title || `变更 #${change.id}`}
      getItemNumber={(change: Change) => {
        const data = change as unknown as Record<string, unknown>;
        return (data.changeNumber as string) || `C-${change.id}`;
      }}
      getItemDescription={(change: Change) => change.description || ''}
      getItemPriority={(change: Change) => change.priority || 'medium'}
      getItemAssignee={(change: Change) => {
        const assigneeId = change.assigneeId;
        if (!assigneeId) return null;
        return { name: change.assigneeName || `用户 #${assigneeId}` };
      }}
      getItemCreatedAt={(change: Change) => change.createdAt || ''}
      getItemUpdatedAt={(change: Change) => change.updatedAt || ''}
      onItemClick={(change: Change) => router.push(`/changes/${change.id}`)}
      onItemEdit={(change: Change) => router.push(`/changes/${change.id}/edit`)}
      columnConfigs={KANBAN_COLUMNS}
      showToolbar={false}
      searchPlaceholder="搜索变更标题或描述..."
      priorityOptions={[
        { value: 'critical', label: '紧急', color: 'red' },
        { value: 'high', label: '高', color: 'orange' },
        { value: 'medium', label: '中', color: 'blue' },
        { value: 'low', label: '低', color: 'green' },
      ]}
    />
  );

  const getChangesForDate = (date: Dayjs) =>
    calendarData.filter((change) => {
      const start = dayjs(change.plannedStart);
      const end = dayjs(change.plannedEnd || change.plannedStart);
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
                    const status = CHANGE_STATUS_CONFIG[change.status] || {
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
                    <span className="text-xs text-gray-500">另有 {dayChanges.length - 3} 项</span>
                  )}
                </div>
              );
            }}
          />
        </Card>

        <Card title={`${selectedDate.format('YYYY年MM月DD日')}的变更`} className="mt-4">
          {selectedChanges.length === 0 ? (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="当天暂无变更" />
          ) : (
            <div className="space-y-2">
              {selectedChanges.map((change) => {
                const status = CHANGE_STATUS_CONFIG[change.status] || {
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
                    <span className="shrink-0 text-sm text-gray-500">{change.assigneeName || '未分配'}</span>
                  </button>
                );
              })}
            </div>
          )}
        </Card>
      </Spin>
    );
  };

  // ====== 渲染视图 ======
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
      // 页面信息
      title="变更管理"
      description="管理IT基础架构和服务的变更请求，最小化变更风险"

      // 统计
      stats={pageStats}
      statsLoading={statsLoading}

      // 搜索
      searchPlaceholder="搜索变更标题或描述..."
      searchValue={searchKeyword}
      onSearch={handleSearch}
      searchLoading={loading}

      // 筛选
      filters={{
        visible: showFilters,
        onToggle: () => setShowFilters(!showFilters),
        content: renderFilters(),
      }}

      // 视图切换（使用 Tabs 模拟）
      showViewSwitch={false}
      // 自定义 Tab 渲染在 children 中

      // 操作
      primaryAction={{
        label: '新建变更',
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
      loading={activeView === 'calendar' ? false : loading}
      empty={activeView !== 'calendar' && changes.length === 0 && !loading}
      emptyDescription="暂无变更记录"
      emptyAction={{
        label: '创建第一个变更',
        onClick: handleCreate,
      }}
    >
      {/* 自定义视图切换 */}
      <div className="flex gap-2 mb-4">
        <Button
          type={activeView === 'list' ? 'primary' : 'default'}
          icon={<Search className="w-4 h-4" />}
          onClick={() => setActiveView('list')}
        >
          列表视图
        </Button>
        <Button
          type={activeView === 'kanban' ? 'primary' : 'default'}
          icon={<LayoutGrid className="w-4 h-4" />}
          onClick={() => setActiveView('kanban')}
        >
          看板视图
        </Button>
        <Button
          type={activeView === 'calendar' ? 'primary' : 'default'}
          icon={<Calendar className="w-4 h-4" />}
          onClick={() => setActiveView('calendar')}
        >
          日历视图
        </Button>
      </div>

      {/* 视图内容 */}
      {renderContent()}

      {/* 分页 */}
      {activeView === 'list' && changes.length > 0 && (
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
