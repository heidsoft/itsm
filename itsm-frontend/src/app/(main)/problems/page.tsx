'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Card, Row, Col, Statistic, Button, message, Input, Select, Space, Tabs } from 'antd';
import {
  Bug,
  CheckCircle,
  Clock,
  AlertTriangle,
  Plus,
  Search,
  Filter,
  Table as TableIcon,
} from 'lucide-react';
import { LayoutGrid } from 'lucide-react';
import { useRouter } from 'next/navigation';
import ProblemList from '@/components/problem/ProblemList';
import { ProblemApi, type Problem } from '@/lib/api/problem-api';
import { useI18n } from '@/lib/i18n/useI18n';
import { PageContainer } from '@/components/layout/PageContainer';
import { useDebounce } from '@/lib/component-utils';
import {
  UnifiedKanbanBoard,
  type KanbanColumnConfig,
} from '@/components/business/UnifiedKanbanBoard';

export default function ProblemListPage() {
  const router = useRouter();
  const { t } = useI18n();
  const [stats, setStats] = useState({
    total: 0,
    open: 0,
    inProgress: 0,
    resolved: 0,
  });

  // 搜索和筛选状态
  const [searchKeyword, setSearchKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined);
  const [priorityFilter, setPriorityFilter] = useState<string | undefined>(undefined);
  const [showFilters, setShowFilters] = useState(false);
  const [activeTab, setActiveTab] = useState('list');
  const [problems, setProblems] = useState<Problem[]>([]);
  const [loading, setLoading] = useState(false);
  const debouncedSearch = useDebounce(searchKeyword, 300);

  // 看板列配置
  const KANBAN_COLUMNS: KanbanColumnConfig<Problem>[] = [
    { key: 'open', title: '待处理', color: '#ff4d4f' },
    { key: 'investigating', title: '调查中', color: '#722ed1' },
    { key: 'identified', title: '已识别', color: '#fa8c16' },
    { key: 'resolved', title: '已解决', color: '#52c41a' },
    { key: 'closed', title: '已关闭', color: '#d9d9d9' },
  ];

  // 获取问题列表用于看板视图
  const fetchProblemsForKanban = useCallback(async () => {
    setLoading(true);
    try {
      const response = await ProblemApi.getProblems({
        page: 1,
        page_size: 100, // 获取足够的数据用于看板
        status: statusFilter,
        priority: priorityFilter,
        search: debouncedSearch,
      });
      const items = response.problems || [];
      setProblems(items);
    } catch (error) {
      console.error('Failed to fetch problems for kanban:', error);
      setProblems([]);
    } finally {
      setLoading(false);
    }
  }, [statusFilter, priorityFilter, debouncedSearch]);

  useEffect(() => {
    if (activeTab === 'kanban') {
      fetchProblemsForKanban();
    }
  }, [activeTab, fetchProblemsForKanban]);

  const fetchStats = async () => {
    try {
      const stats = await ProblemApi.getProblemStats();
      setStats({
        total: stats.total || 0,
        open: stats.open || 0,
        inProgress: stats.inProgress || 0,
        resolved: stats.resolved || 0,
      });
    } catch (error) {
      console.error('Failed to fetch problem stats:', error);
      message.error(t('problems.getStatsFailed') || '获取问题统计失败，请稍后重试');
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  // 筛选器配置
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

  const handleResetFilters = () => {
    setSearchKeyword('');
    setStatusFilter(undefined);
    setPriorityFilter(undefined);
  };

  const statsContent = (
    <Row gutter={[16, 16]} className="mb-6">
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="总问题数"
            value={stats.total}
            prefix={<Bug className="text-blue-500 mr-2" />}
            styles={{ content: { color: '#1890ff' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="待处理"
            value={stats.open}
            prefix={<AlertTriangle className="text-red-500 mr-2" />}
            styles={{ content: { color: '#ff4d4f' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="调查中"
            value={stats.inProgress}
            prefix={<Clock className="text-orange-500 mr-2" />}
            styles={{ content: { color: '#fa8c16' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="已解决"
            value={stats.resolved}
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
          placeholder="搜索问题标题或描述..."
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
          placeholder="优先级筛选"
          value={priorityFilter}
          onChange={setPriorityFilter}
          allowClear
          options={priorityOptions}
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
        title="问题管理"
        description="识别、分析和消除事件发生的根本原因"
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
              onClick={() => router.push('/problems/new')}
            >
              新建问题
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
                <ProblemList
                  showHeader={false}
                  keyword={debouncedSearch}
                  status={statusFilter}
                  priority={priorityFilter}
                />
              ),
            },
            {
              key: 'kanban',
              label: (
                <span className="flex items-center gap-2">
                  <LayoutGrid />
                  看板视图
                </span>
              ),
              children: (
                <UnifiedKanbanBoard<Problem>
                  items={problems}
                  loading={loading}
                  getItemId={(problem: Problem) => problem.id}
                  getItemStatus={(problem: Problem) => problem.status || 'open'}
                  getItemTitle={(problem: Problem) => problem.title || `问题 #${problem.id}`}
                  getItemNumber={(problem: Problem) => {
                    const data = problem as unknown as Record<string, unknown>;
                    return (
                      (data.problem_number as string) ||
                      (data.problemNumber as string) ||
                      `P-${problem.id}`
                    );
                  }}
                  getItemDescription={(problem: Problem) => problem.description || ''}
                  getItemPriority={(problem: Problem) =>
                    problem.priority || problem.severity || 'medium'
                  }
                  getItemAssignee={(problem: Problem) => {
                    const assigneeId = problem.assigneeId || problem.assignee_id;
                    if (!assigneeId) return null;
                    const data = problem as unknown as Record<string, unknown>;
                    const assigneeName =
                      (data.assignee_name as string) || (data.assigneeName as string);
                    return { name: assigneeName || `用户 #${assigneeId}` };
                  }}
                  getItemCreatedAt={(problem: Problem) => problem.createdAt || problem.created_at}
                  getItemUpdatedAt={(problem: Problem) => problem.updatedAt || problem.updated_at}
                  onItemClick={(problem: Problem) => router.push(`/problems/${problem.id}`)}
                  onItemEdit={(problem: Problem) => router.push(`/problems/${problem.id}/edit`)}
                  columnConfigs={KANBAN_COLUMNS}
                  searchPlaceholder="搜索问题标题或描述..."
                  priorityOptions={[
                    { value: 'critical', label: '紧急', color: 'red' },
                    { value: 'high', label: '高', color: 'orange' },
                    { value: 'medium', label: '中', color: 'blue' },
                    { value: 'low', label: '低', color: 'green' },
                  ]}
                />
              ),
            },
          ]}
        />
      </PageContainer>
    </div>
  );
}
