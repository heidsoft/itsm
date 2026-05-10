'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Card, Row, Col, Statistic, Button, message, Input, Select, Space, Tabs } from 'antd';
import {
  GitPullRequest,
  CheckCircle,
  Clock,
  Plus,
  Search,
  Filter,
  Table as TableIcon,
  Calendar,
} from 'lucide-react';
import { AppstoreOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import ChangeList from '@/components/change/ChangeList';
import { ChangeApi, type Change } from '@/lib/api/change-api';
import { useI18n } from '@/lib/i18n/useI18n';
import { PageContainer } from '@/components/layout/PageContainer';
import { useDebounce } from '@/lib/component-utils';
import {
  UnifiedKanbanBoard,
  type KanbanColumnConfig,
} from '@/components/business/UnifiedKanbanBoard';

export default function ChangesPage() {
  const router = useRouter();
  const { t } = useI18n();
  const [stats, setStats] = useState({
    total: 0,
    pending: 0,
    inProgress: 0,
    completed: 0,
  });

  // 搜索和筛选状态
  const [searchKeyword, setSearchKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined);
  const [riskFilter, setRiskFilter] = useState<string | undefined>(undefined);
  const [showFilters, setShowFilters] = useState(false);
  const [activeTab, setActiveTab] = useState('list');
  const [changes, setChanges] = useState<Change[]>([]);
  const [kanbanLoading, setKanbanLoading] = useState(false);
  const debouncedSearch = useDebounce(searchKeyword, 300);

  // 看板列配置
  const KANBAN_COLUMNS: KanbanColumnConfig<Change>[] = [
    { key: 'draft', title: '草稿', color: '#d9d9d9' },
    { key: 'pending', title: '待审批', color: '#fa8c16' },
    { key: 'approved', title: '已批准', color: '#1890ff' },
    { key: 'scheduled', title: '已排期', color: '#722ed1' },
    { key: 'implementing', title: '实施中', color: '#13c2c2' },
    { key: 'completed', title: '已完成', color: '#52c41a' },
    { key: 'cancelled', title: '已取消', color: '#ff4d4f' },
  ];

  // 获取变更列表用于看板视图
  const fetchChangesForKanban = useCallback(async () => {
    setKanbanLoading(true);
    try {
      const response = await ChangeApi.getChanges({
        page: 1,
        page_size: 100,
        status: statusFilter as any,  // ChangeStatus is string enum
        search: debouncedSearch,
      });
      const items = (response as any).items || (response as any).changes || [];
      setChanges(items as Change[]);
    } catch (error) {
      console.error('Failed to fetch changes for kanban:', error);
      setChanges([]);
    } finally {
      setKanbanLoading(false);
    }
  }, [statusFilter, debouncedSearch]);

  useEffect(() => {
    if (activeTab === 'kanban') {
      fetchChangesForKanban();
    }
  }, [activeTab, fetchChangesForKanban]);

  const fetchStats = async () => {
    try {
      const stats = await ChangeApi.getChangeStats();
      setStats({
        total: stats.total || 0,
        pending: stats.pending || 0,
        inProgress: stats.inProgress || 0,
        completed: stats.completed || 0,
      });
    } catch (error) {
      console.error('Failed to fetch change stats:', error);
      message.error(t('changes.getStatsFailed') || '获取变更统计失败，请稍后重试');
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  // 筛选器配置
  const statusOptions = [
    { value: 'draft', label: '草稿' },
    { value: 'pending', label: '待审批' },
    { value: 'approved', label: '已批准' },
    { value: 'implementing', label: '实施中' },
    { value: 'completed', label: '已完成' },
    { value: 'rejected', label: '已拒绝' },
    { value: 'cancelled', label: '已取消' },
  ];

  const riskOptions = [
    { value: 'high', label: '高风险' },
    { value: 'medium', label: '中风险' },
    { value: 'low', label: '低风险' },
  ];

  const handleResetFilters = () => {
    setSearchKeyword('');
    setStatusFilter(undefined);
    setRiskFilter(undefined);
  };

  const statsContent = (
    <Row gutter={[16, 16]} className="mb-6">
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="总变更数"
            value={stats.total}
            prefix={<GitPullRequest className="text-blue-500 mr-2" />}
            styles={{ content: { color: '#1890ff' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="待审批"
            value={stats.pending}
            prefix={<Clock className="text-orange-500 mr-2" />}
            styles={{ content: { color: '#fa8c16' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="进行中"
            value={stats.inProgress}
            prefix={<GitPullRequest className="text-blue-500 mr-2" />}
            styles={{ content: { color: '#1890ff' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="已完成"
            value={stats.completed}
            prefix={<CheckCircle className="text-green-500 mr-2" />}
            styles={{ content: { color: '#52c41a' } }}
          />
        </Card>
      </Col>
    </Row>
  );

  const filterContent = showFilters && (
    <Card className="mb-4 rounded-lg shadow-sm">
      <Space wrap size="middle">
        <Input
          placeholder="搜索变更标题或描述..."
          prefix={<Search className="w-4 h-4 text-gray-400" />}
          value={searchKeyword}
          onChange={e => setSearchKeyword(e.target.value)}
          allowClear
          style={{ width: 250 }}
        />
        <Select
          placeholder="状态筛选"
          value={statusFilter}
          onChange={setStatusFilter}
          allowClear
          options={statusOptions}
          style={{ width: 150 }}
        />
        <Select
          placeholder="风险等级"
          value={riskFilter}
          onChange={setRiskFilter}
          allowClear
          options={riskOptions}
          style={{ width: 150 }}
        />
        <Button onClick={handleResetFilters}>重置</Button>
      </Space>
    </Card>
  );

  return (
    <div
      className="p-6 min-h-screen"
      style={{ backgroundColor: 'var(--color-bg-secondary, #f9fafb)' }}
    >
      <PageContainer
        title="变更管理"
        description="管理IT基础架构和服务的变更请求，最小化变更风险"
        extra={
          <Space>
            <Button
              icon={<Search className="w-4 h-4" />}
              onClick={() => setShowFilters(!showFilters)}
              type={showFilters ? 'primary' : 'default'}
            >
              搜索筛选
            </Button>
            <Button
              type="primary"
              icon={<Plus className="w-4 h-4" />}
              size="large"
              onClick={() => router.push('/changes/new')}
            >
              新建变更
            </Button>
          </Space>
        }
        showStats
        stats={statsContent}
      >
        {filterContent}

        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          size="large"
          className="mb-4"
          items={[
            {
              key: 'list',
              label: (
                <span className="flex items-center gap-2">
                  <TableIcon className="w-4 h-4" />
                  列表视图
                </span>
              ),
              children: (
                <ChangeList
                  showHeader={false}
                  search={debouncedSearch}
                  status={statusFilter}
                  risk={riskFilter}
                />
              ),
            },
            {
              key: 'kanban',
              label: (
                <span className="flex items-center gap-2">
                  <AppstoreOutlined />
                  看板视图
                </span>
              ),
              children: (
                <UnifiedKanbanBoard<Change>
                  items={changes}
                  loading={kanbanLoading}
                  getItemId={(change: Change) => change.id}
                  getItemStatus={(change: Change) => change.status || 'draft'}
                  getItemTitle={(change: Change) => change.title || `变更 #${change.id}`}
                  getItemNumber={(change: Change) => {
                    const data = change as unknown as Record<string, unknown>;
                    return (
                      (data.change_number as string) ||
                      (data.changeNumber as string) ||
                      `C-${change.id}`
                    );
                  }}
                  getItemDescription={(change: Change) => change.description || ''}
                  getItemPriority={(change: Change) => change.priority || 'medium'}
                  getItemAssignee={(change: Change) => {
                    const assigneeId = change.assigneeId || (change as any).assignee_id;
                    if (!assigneeId) return null;
                    const data = change as unknown as Record<string, unknown>;
                    const assigneeName =
                      (data.assignee_name as string) || (data.assigneeName as string);
                    return { name: assigneeName || `用户 #${assigneeId}` };
                  }}
                  getItemCreatedAt={(change: Change) =>
                    change.createdAt || (change as any).created_at || ''
                  }
                  getItemUpdatedAt={(change: Change) =>
                    change.updatedAt || (change as any).updated_at || ''
                  }
                  onItemClick={(change: Change) => router.push(`/changes/${change.id}`)}
                  onItemEdit={(change: Change) => router.push(`/changes/${change.id}/edit`)}
                  columnConfigs={KANBAN_COLUMNS}
                  searchPlaceholder="搜索变更标题或描述..."
                  priorityOptions={[
                    { value: 'critical', label: '紧急', color: 'red' },
                    { value: 'high', label: '高', color: 'orange' },
                    { value: 'medium', label: '中', color: 'blue' },
                    { value: 'low', label: '低', color: 'green' },
                  ]}
                />
              ),
            },
            {
              key: 'calendar',
              label: (
                <span className="flex items-center gap-2">
                  <Calendar className="w-4 h-4" />
                  日历视图
                </span>
              ),
              children: (
                <Card className="text-center py-12">
                  <Calendar className="w-16 h-16 mx-auto text-gray-400 mb-4" />
                  <p className="text-gray-500">日历视图开发中，敬请期待...</p>
                </Card>
              ),
            },
          ]}
        />
      </PageContainer>
    </div>
  );
}
