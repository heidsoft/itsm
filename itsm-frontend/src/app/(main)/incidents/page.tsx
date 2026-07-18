'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Button, Space, message, Pagination, Badge, Modal, Select, Input, Form } from 'antd';
import { Plus, Search, Bell, RotateCcw, Download, Filter, UserPlus, CheckCircle, XCircle, Trash2 } from 'lucide-react';
import { useRouter } from 'next/navigation';
import {
  BusinessPageTemplate,
  type PageStats,
} from '@/components/layout/BusinessPageTemplate';
import { IncidentList } from './components/IncidentList';
import { IncidentFilters } from './components/IncidentFilters';
import { IncidentStats } from './components/IncidentStats';
import type { Incident } from '@/lib/api/types';
import { IncidentAPI } from '@/lib/api/incident-api';
import { UserApi } from '@/lib/api/user-api';
import { BatchActionBar, type BatchAction } from '@/components/business/BatchActionBar';
import {
  UnifiedKanbanBoard,
  type KanbanColumnConfig,
} from '@/components/business/UnifiedKanbanBoard';
import { useI18n } from '@/lib/i18n/useI18n';

// 看板列配置
const KANBAN_COLUMNS: KanbanColumnConfig<Incident>[] = [
  { key: 'new', title: '新建', color: '#1890ff' },
  { key: 'acknowledged', title: '已确认', color: '#722ed1' },
  { key: 'assigned', title: '已分配', color: '#13c2c2' },
  { key: 'inProgress', title: '处理中', color: '#1890ff' },
  { key: 'resolved', title: '已解决', color: '#52c41a' },
  { key: 'closed', title: '已关闭', color: '#d9d9d9' },
];

export default function IncidentsPage() {
  const router = useRouter();
  const { t } = useI18n();

  // ====== 状态管理 ======
  const [loading, setLoading] = useState(false);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const [searchKeyword, setSearchKeyword] = useState('');
  const [activeView, setActiveView] = useState<'list' | 'kanban'>('list');

  // 统计数据
  const [metrics, setMetrics] = useState<{
    totalIncidents?: number;
    openIncidents?: number;
    criticalIncidents?: number;
    majorIncidents?: number;
    avgResolutionTime?: number;
  } | null>(null);
  const [statsLoading, setStatsLoading] = useState(false);

  // 筛选状态
  const [filtersVisible, setFiltersVisible] = useState(false);

  // ====== 数据获取 ======
  const fetchIncidents = useCallback(async () => {
    setLoading(true);
    try {
      const response = await IncidentAPI.listIncidents({
        page,
        pageSize,
        search: searchKeyword || undefined,
      });
      const items = response.incidents || response.data || [];

      // 获取用户映射
      const userMap = new Map<number, string>();
      try {
        const usersResponse = await UserApi.getUsers({ pageSize: 100 });
        usersResponse.users.forEach((user) => userMap.set(user.id, user.name));
      } catch (e) {
        console.warn('Failed to fetch users for reporter names', e);
      }

      // 丰富事件数据
      const enriched = items.map((inc) => {
        const reporterId = inc.reporterId || (inc as any).reporterId;
        const reporterName = reporterId ? userMap.get(reporterId) : undefined;
        return {
          ...inc,
          ...(reporterId && reporterName
            ? { reporter: { id: reporterId, name: reporterName } }
            : {}),
        };
      });

      setIncidents(enriched as any);
      setTotal(response.total || enriched.length);
    } catch (error) {
      console.error('Failed to fetch incidents:', error);
      message.error(t('incidents.getFailed') || '加载事件列表失败，请稍后重试');
      setIncidents([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, searchKeyword, t]);

  const fetchStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const stats = await IncidentAPI.getIncidentMetrics();
      setMetrics({
        totalIncidents: stats.totalIncidents || 0,
        openIncidents: stats.openIncidents || 0,
        criticalIncidents: stats.criticalIncidents || 0,
        majorIncidents: stats.majorIncidents || 0,
        avgResolutionTime: stats.avgResolutionTime || 0,
      });
    } catch (error) {
      console.error('Failed to fetch incident stats:', error);
    } finally {
      setStatsLoading(false);
    }
  }, []);

  // 初始加载
  useEffect(() => {
    fetchIncidents();
    fetchStats();
  }, [fetchIncidents, fetchStats]);

  // ====== 事件处理 ======
  const handleSearch = useCallback((value: string) => {
    setSearchKeyword(value);
    setPage(1);
  }, []);

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
    router.push('/incidents/new');
  }, [router]);

  const handleRefresh = useCallback(() => {
    fetchIncidents();
    fetchStats();
  }, [fetchIncidents, fetchStats]);

  const handlePageChange = useCallback((newPage: number, newPageSize: number) => {
    setPage(newPage);
    setPageSize(newPageSize);
  }, []);

  // ====== 批量操作 ======
  const [batchLoading, setBatchLoading] = useState(false);
  const [assignModalOpen, setAssignModalOpen] = useState(false);
  const [assignUserOptions, setAssignUserOptions] = useState<
    { label: string; value: number }[]
  >([]);
  const [assignForm] = Form.useForm<{ assigneeId: number }>();

  // 逐条循环兜底：后端目前尚未提供 incident 批量端点，此处封装 Promise.allSettled，失败逐条汇总
  const runIncidentBatch = useCallback(
    async (
      ids: React.Key[],
      handler: (id: number) => Promise<unknown>,
      successMsg: string,
      failPrefix = '部分失败',
    ) => {
      if (ids.length === 0) return;
      setBatchLoading(true);
      try {
        const results = await Promise.allSettled(ids.map((id) => handler(Number(id))));
        const failCount = results.filter((r) => r.status === 'rejected').length;
        if (failCount === 0) {
          message.success(`${successMsg}：${ids.length} 项`);
        } else {
          message.warning(`${failPrefix}：成功 ${ids.length - failCount} 项，失败 ${failCount} 项`);
        }
        setSelectedRowKeys([]);
        await fetchIncidents();
      } finally {
        setBatchLoading(false);
      }
    },
    [fetchIncidents],
  );

  const openAssignModal = useCallback(async () => {
    assignForm.resetFields();
    setAssignModalOpen(true);
    if (assignUserOptions.length === 0) {
      try {
        const res = await UserApi.getUsers({ pageSize: 100 });
        setAssignUserOptions(
          (res.users || []).map((u) => ({ label: u.name || u.username, value: u.id })),
        );
      } catch (e) {
        console.warn('load users for assign failed', e);
      }
    }
  }, [assignForm, assignUserOptions.length]);

  const handleBatchAssign = useCallback(async () => {
    const values = await assignForm.validateFields();
    setAssignModalOpen(false);
    await runIncidentBatch(
      selectedRowKeys,
      (id) => IncidentAPI.assignIncident(id, values.assigneeId),
      '批量分派成功',
    );
  }, [assignForm, selectedRowKeys, runIncidentBatch]);

  const handleBatchResolve = useCallback(async () => {
    await runIncidentBatch(
      selectedRowKeys,
      (id) => IncidentAPI.resolveIncident(id, { resolution: '批量解决' }),
      '批量解决成功',
    );
  }, [selectedRowKeys, runIncidentBatch]);

  const handleBatchClose = useCallback(async () => {
    await runIncidentBatch(
      selectedRowKeys,
      (id) => IncidentAPI.closeIncident(id, { closeNotes: '批量关闭' }),
      '批量关闭成功',
    );
  }, [selectedRowKeys, runIncidentBatch]);

  const handleBatchDelete = useCallback(async () => {
    await runIncidentBatch(
      selectedRowKeys,
      (id) => IncidentAPI.deleteIncident(id),
      '批量删除成功',
    );
  }, [selectedRowKeys, runIncidentBatch]);

  const batchActions: BatchAction[] = [
    {
      key: 'assign',
      label: '批量分派',
      icon: <UserPlus size={14} />,
      onClick: openAssignModal,
      type: 'primary',
    },
    {
      key: 'resolve',
      label: '批量解决',
      icon: <CheckCircle size={14} />,
      onClick: handleBatchResolve,
    },
    {
      key: 'close',
      label: '批量关闭',
      icon: <XCircle size={14} />,
      onClick: handleBatchClose,
    },
    {
      key: 'delete',
      label: '批量删除',
      icon: <Trash2 size={14} />,
      danger: true,
      confirmTitle: `确定删除选中的 ${selectedRowKeys.length} 个事件？此操作不可撤销`,
      onClick: handleBatchDelete,
      overflow: true,
    },
  ];

  // ====== 统计数据转换 ======
  const pageStats: PageStats[] = metrics
    ? [
        {
          label: '总事件',
          value: metrics.totalIncidents || 0,
          color: '#1890ff',
          icon: <span className="text-2xl">📊</span>,
        },
        {
          label: '待处理',
          value: metrics.openIncidents || 0,
          color: '#faad14',
          icon: <span className="text-2xl">⏳</span>,
        },
        {
          label: '紧急事件',
          value: metrics.criticalIncidents || 0,
          color: '#ff4d4f',
          icon: <span className="text-2xl">🚨</span>,
        },
        {
          label: '平均解决时间',
          value: Math.round((metrics.avgResolutionTime || 0) / 60),
          color: '#52c41a',
          icon: <span className="text-2xl">⏱️</span>,
        },
      ]
    : [];

  // ====== 渲染内容 ======
  const renderListContent = () => {
    if (incidents.length === 0 && !loading) {
      return (
        <div className="py-12 text-center">
          <div className="text-gray-400 mb-4">暂无事件记录</div>
          <Button type="primary" onClick={handleCreate}>
            创建第一个事件
          </Button>
        </div>
      );
    }

    return (
      <>
        <BatchActionBar
          selectedCount={selectedRowKeys.length}
          itemLabel="事件"
          onClear={() => setSelectedRowKeys([])}
          actions={batchActions}
          loading={batchLoading}
        />
        <IncidentList
          incidents={incidents}
          loading={loading}
          selectedRowKeys={selectedRowKeys}
          onSelectedRowKeysChange={setSelectedRowKeys}
          onEdit={handleEdit}
          onView={handleView}
          onRefresh={handleRefresh}
        />
      </>
    );
  };

  const renderKanbanContent = () => (
    <UnifiedKanbanBoard<Incident>
      items={incidents}
      loading={loading}
      getItemId={(incident: Incident) => incident.id}
      getItemStatus={(incident: Incident) => incident.status}
      getItemTitle={(incident: Incident) => incident.title || `事件 #${incident.id}`}
      getItemNumber={(incident: Incident) =>
        incident.incidentNumber || incident.incidentNumber || String(incident.id)
      }
      getItemDescription={(incident: Incident) => incident.description || ''}
      getItemPriority={(incident: Incident) =>
        incident.priority || incident.severity || 'medium'
      }
      getItemAssignee={(incident: Incident) =>
        incident.assignee
          ? { name: incident.assignee.name || incident.assigneeName || '未分配' }
          : null
      }
      getItemCreatedAt={(incident: Incident) => incident.createdAt || incident.createdAt}
      getItemUpdatedAt={(incident: Incident) => incident.updatedAt || incident.updatedAt}
      onItemClick={handleView}
      onItemEdit={handleEdit}
      columnConfigs={KANBAN_COLUMNS}
      searchPlaceholder="搜索事件ID、标题或描述..."
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
      title="事件管理"
      description="管理和追踪系统中的所有事件记录"

      // 统计
      stats={pageStats}
      statsLoading={statsLoading}

      // 搜索
      searchPlaceholder="搜索事件ID、标题或描述..."
      searchValue={searchKeyword}
      onSearch={handleSearch}
      searchLoading={loading}

      // 筛选
      filters={{
        visible: filtersVisible,
        onToggle: () => setFiltersVisible(!filtersVisible),
        content: <IncidentFilters />,
      }}

      // 视图切换
      showViewSwitch={true}
      activeView={activeView}
      onViewChange={setActiveView}

      // 操作
      primaryAction={{
        label: '新建事件',
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
      empty={incidents.length === 0 && !loading}
      emptyDescription="暂无事件记录"
      emptyAction={{
        label: '创建第一个事件',
        onClick: handleCreate,
      }}
    >
      {/* 视图内容 */}
      {activeView === 'list' ? renderListContent() : renderKanbanContent()}

      {/* 分页 */}
      {incidents.length > 0 && (
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
      {/* 批量分派 Modal */}
      <Modal
        title="批量分派事件"
        open={assignModalOpen}
        onOk={handleBatchAssign}
        onCancel={() => setAssignModalOpen(false)}
        okText="确定分派"
        cancelText="取消"
        confirmLoading={batchLoading}
      >
        <div className="mb-3 text-sm text-gray-500">
          将为已选择的 <span className="text-blue-600 font-semibold">{selectedRowKeys.length}</span> 个事件分派处理人
        </div>
        <Form form={assignForm} layout="vertical">
          <Form.Item
            name="assigneeId"
            label="处理人"
            rules={[{ required: true, message: '请选择处理人' }]}
          >
            <Select
              placeholder="请选择处理人"
              showSearch
              optionFilterProp="label"
              options={assignUserOptions}
            />
          </Form.Item>
        </Form>
      </Modal>
    </BusinessPageTemplate>
  );
}
