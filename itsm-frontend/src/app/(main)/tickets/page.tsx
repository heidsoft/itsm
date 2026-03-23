'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Card, Typography, Space, Button, Tabs, Badge, Alert } from 'antd';
import {
  PlusOutlined,
  TableOutlined,
  AppstoreOutlined,
  BarChartOutlined,
  SearchOutlined,
  BellOutlined,
} from '@ant-design/icons';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import TicketList from '@/components/ticket/TicketList';
import TicketKanban from '@/components/ticket/TicketKanban';
import TicketAdvancedSearch, {
  type AdvancedSearchFilters,
} from '@/components/ticket/TicketAdvancedSearch';
import type { TicketQueryFilters } from '@/lib/hooks/useTickets';
import {
  saveFilters,
  restoreFilters,
  clearFilters,
  getDefaultFilters,
} from '@/lib/utils/filter-persistence';

const { Title, Text } = Typography;

export default function TicketsPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [activeTab, setActiveTab] = useState('list');
  const [showAdvancedSearch, setShowAdvancedSearch] = useState(false);

  // 从 localStorage 恢复筛选条件
  const [advancedFilters, setAdvancedFilters] = useState<Partial<TicketQueryFilters>>(() => {
    const defaults = getDefaultFilters('tickets') as Partial<TicketQueryFilters>;
    return restoreFilters('tickets', defaults);
  });

  const [ticketStats, setTicketStats] = useState({
    total: 0,
    open: 0,
    overdue: 0,
    today: 0,
  });

  // 保存筛选条件到 localStorage
  useEffect(() => {
    saveFilters('tickets', advancedFilters);
  }, [advancedFilters]);

  // 从 URL 参数获取当前标签页和高级搜索状态
  useEffect(() => {
    const tab = searchParams.get('tab');
    if (tab && ['list', 'kanban', 'analytics', 'search'].includes(tab)) {
      setActiveTab(tab);
    }
    // 从 URL 恢复高级搜索面板状态
    const search = searchParams.get('search');
    if (search === 'advanced') {
      setShowAdvancedSearch(true);
    }
  }, [searchParams]);

  // 获取工单统计数据
  const fetchTicketStats = useCallback(async () => {
    try {
      const { ticketService } = await import('@/lib/services/ticket-service');
      const stats = await ticketService.getTicketStats();
      setTicketStats({
        total: stats.total,
        open: stats.open,
        overdue: stats.overdue || 0,
        today: 0, // 暂时没有今日新增的 API
      });
    } catch (error) {
      console.error('Failed to fetch ticket stats:', error);
    }
  }, []);

  useEffect(() => {
    fetchTicketStats();
  }, [fetchTicketStats]);

  // 处理标签页切换
  const handleTabChange = (tab: string) => {
    setActiveTab(tab);
    const newParams = new URLSearchParams(searchParams.toString());
    newParams.set('tab', tab);
    router.push(`/tickets?${newParams.toString()}`, { scroll: false });
  };

  const mapAdvancedToQueryFilters = useCallback(
    (filters: AdvancedSearchFilters): Partial<TicketQueryFilters> => {
      const result: Partial<TicketQueryFilters> = {};

      if (filters.keyword) {
        result.keyword = filters.keyword;
      } else if (filters.ticket_number) {
        result.keyword = filters.ticket_number;
      } else if (filters.title) {
        result.keyword = filters.title;
      } else if (filters.description) {
        result.keyword = filters.description;
      }

      if (filters.status && filters.status.length > 0) {
        result.status = filters.status[0] as TicketQueryFilters['status'];
      }

      if (filters.priority && filters.priority.length > 0) {
        result.priority = filters.priority[0] as TicketQueryFilters['priority'];
      }

      if (filters.type && filters.type.length > 0) {
        result.type = filters.type[0] as TicketQueryFilters['type'];
      }

      if (filters.category && filters.category.length > 0) {
        result.category = filters.category[0];
      }

      if (typeof filters.assignee_id === 'number') {
        result.assignee_id = filters.assignee_id;
      }

      if (filters.created_after && filters.created_before) {
        result.dateRange = [filters.created_after, filters.created_before];
      }

      return result;
    },
    []
  );

  const handleAdvancedSearch = (filters: AdvancedSearchFilters) => {
    const mapped = mapAdvancedToQueryFilters(filters);
    setAdvancedFilters(mapped);
    setActiveTab('list');
  };

  const handleSearchReset = () => {
    clearFilters('tickets');
    setAdvancedFilters({});
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* 页面头部 */}
      <div className="bg-white border-b border-gray-200">
        <div className="w-full px-6 py-4">
          <div className="flex items-center justify-between">
            <div>
              <Title level={2} style={{ marginBottom: 0 }}>
                工单管理
              </Title>
              <Text type="secondary">
                统一的工单处理平台，支持多维度视图切换、全生命周期管理、SLA 监控与智能分派
              </Text>
            </div>
            <Space>
              <Button
                icon={<SearchOutlined />}
                onClick={() => {
                  const newShow = !showAdvancedSearch;
                  setShowAdvancedSearch(newShow);
                  const newParams = new URLSearchParams(searchParams.toString());
                  if (newShow) {
                    newParams.set('search', 'advanced');
                  } else {
                    newParams.delete('search');
                  }
                  router.push(`/tickets?${newParams.toString()}`, { scroll: false });
                }}
              >
                高级搜索
              </Button>
              <Badge count={ticketStats.overdue} size="small">
                <Button icon={<BellOutlined />}>SLA 预警</Button>
              </Badge>
              <Link href="/tickets/create">
                <Button type="primary" icon={<PlusOutlined />}>
                  新建工单
                </Button>
              </Link>
            </Space>
          </div>

          {/* 统计数据栏 */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mt-4">
            <Card size="small" className="rounded-lg shadow-sm">
              <div className="flex items-center justify-between">
                <div>
                  <Text type="secondary">总工单</Text>
                  <div className="text-2xl font-bold">{ticketStats.total}</div>
                </div>
                <TableOutlined className="text-2xl text-blue-500" />
              </div>
            </Card>
            <Card size="small" className="rounded-lg shadow-sm">
              <div className="flex items-center justify-between">
                <div>
                  <Text type="secondary">待处理</Text>
                  <div className="text-2xl font-bold text-orange-500">{ticketStats.open}</div>
                </div>
                <BellOutlined className="text-2xl text-orange-500" />
              </div>
            </Card>
            <Card size="small" className="rounded-lg shadow-sm">
              <div className="flex items-center justify-between">
                <div>
                  <Text type="secondary">超时工单</Text>
                  <div className="text-2xl font-bold text-red-500">{ticketStats.overdue}</div>
                </div>
                <BellOutlined className="text-2xl text-red-500" />
              </div>
            </Card>
            <Card size="small" className="rounded-lg shadow-sm">
              <div className="flex items-center justify-between">
                <div>
                  <Text type="secondary">今日新增</Text>
                  <div className="text-2xl font-bold text-green-500">{ticketStats.today}</div>
                </div>
                <PlusOutlined className="text-2xl text-green-500" />
              </div>
            </Card>
          </div>
        </div>
      </div>

      {/* 高级搜索面板 */}
      {showAdvancedSearch && (
        <div className="bg-gray-50 border-b border-gray-200">
          <div className="w-full px-6 py-4">
            <TicketAdvancedSearch onSearch={handleAdvancedSearch} onReset={handleSearchReset} />
          </div>
        </div>
      )}

      {/* 主内容区域 */}
      <div className="w-full px-6 py-6">
        {/* 标签页导航 */}
        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          size="large"
          className="mb-6"
          items={[
            {
              key: 'list',
              label: (
                <span className="flex items-center gap-2">
                  <TableOutlined />
                  列表视图
                </span>
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
            },
          ]}
        />

        {/* 标签页内容 */}
        {activeTab === 'list' && (
          <TicketList showHeader={false} pageSize={20} advancedFilters={advancedFilters} />
        )}

        {activeTab === 'kanban' && (
          <TicketKanban onTicketSelect={ticket => router.push(`/tickets/${ticket.id}`)} />
        )}
      </div>

      {/* 快捷操作浮动按钮 */}
      <div className="fixed bottom-6 right-6 z-50">
        <Space orientation="vertical" size="middle">
          <Button
            type="primary"
            shape="circle"
            size="large"
            icon={<PlusOutlined />}
            onClick={() => router.push('/tickets/create')}
            className="shadow-lg hover:scale-110 transition-transform"
          />
        </Space>
      </div>
    </div>
  );
}
