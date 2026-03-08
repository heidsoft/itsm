'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Button, Input, Space, Card, Pagination, message, Empty, Tabs, Badge, Tag, Avatar } from 'antd';
import { Plus, Search, Filter, Table as TableIcon } from 'lucide-react';
import { AppstoreOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import { IncidentList } from './components/IncidentList';
import { IncidentFilters } from './components/IncidentFilters';
import { IncidentStats } from './components/IncidentStats';
import { Incident } from '@/lib/api/types';
import { IncidentAPI } from '@/lib/api/incident-api';
import { UserApi } from '@/lib/api/user-api';
import { UnifiedKanbanBoard, type KanbanColumnConfig } from '@/components/business/UnifiedKanbanBoard';
import { useDebounce } from '@/lib/component-utils';
import dayjs from 'dayjs';

// Stats data format expected by IncidentStats component
interface IncidentStatsData {
  totalIncidents?: number;
  openIncidents?: number;
  criticalIncidents?: number;
  majorIncidents?: number;
  avgResolutionTime?: number;
}

// Priority config
const PRIORITY_CONFIG = {
  critical: { color: 'red', text: '紧急', label: '紧急', icon: '🔴' },
  high: { color: 'orange', text: '高', label: '高', icon: '🟠' },
  medium: { color: 'blue', text: '中', label: '中', icon: '🔵' },
  low: { color: 'green', text: '低', label: '低', icon: '🟢' },
};

// Kanban column configurations (与Tickets看板保持一致的视觉风格)
const KANBAN_COLUMNS: KanbanColumnConfig<Incident>[] = [
  { key: 'new', title: '新建', color: '#1890ff' },
  { key: 'investigating', title: '调查中', color: '#722ed1' },
  { key: 'identified', title: '已识别', color: '#fa8c16' },
  { key: 'resolved', title: '已解决', color: '#52c41a' },
  { key: 'closed', title: '已关闭', color: '#d9d9d9' },
];

export default function IncidentsPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [showFilters, setShowFilters] = useState(false);
  const [metrics, setMetrics] = useState<IncidentStatsData | null>(null);
  const [searchKeyword, setSearchKeyword] = useState('');
  const debouncedSearch = useDebounce(searchKeyword, 300);
  const [activeTab, setActiveTab] = useState('list');

  // Fetch incidents
  const fetchIncidents = async () => {
    setLoading(true);
    try {
      const response = await IncidentAPI.listIncidents({
        page: 1,
        page_size: 100, // Get more for Kanban view
        search: debouncedSearch || undefined,
      });
      const resp = response as unknown as Record<string, unknown>;
      const items = (resp.items || resp.incidents || []) as Incident[];
      
      // Fetch users to map reporter names
      const userMap = new Map<number, string>();
      try {
        const usersResponse = await UserApi.getUsers({ page_size: 1000 });
        usersResponse.users.forEach(user => userMap.set(user.id, user.name));
      } catch (e) {
        console.warn('Failed to fetch users for reporter names', e);
      }
      
      // Enrich incidents with reporter object if user exists
      const enriched = items.map(inc => {
        const reporterId = inc.reporter_id;
        const reporterName = userMap.get(reporterId);
        return {
          ...inc,
          ...(reporterId && reporterName ? { reporter: { id: reporterId, name: reporterName } } : {}),
        };
      });
      
      setIncidents(enriched);
      const totalCount = (resp as any).total || enriched.length;
      setTotal(totalCount);
    } catch (error) {
      console.error('Failed to fetch incidents:', error);
      setIncidents([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  };

  // Search handler
  const handleSearch = useCallback((value: string) => {
    setSearchKeyword(value);
    setPage(1);
  }, []);

  // Fetch stats
  const fetchStats = async () => {
    try {
      const stats = (await IncidentAPI.getIncidentMetrics()) as unknown as Record<string, unknown>;
      setMetrics({
        totalIncidents: Number(stats.total_incidents) || 0,
        openIncidents: Number(stats.open_incidents) || 0,
        criticalIncidents: Number(stats.critical_incidents) || 0,
        majorIncidents: Number(stats.major_incidents) || 0,
        avgResolutionTime: Number(stats.avg_resolution_time) || 0,
      });
    } catch (error) {
      console.error('Failed to fetch incident stats:', error);
    }
  };

  useEffect(() => {
    let isMounted = true;
    fetchIncidents();
    fetchStats();
    return () => {
      isMounted = false;
    };
  }, [debouncedSearch]);

  const handleEdit = (incident: Incident) => {
    router.push(`/incidents/${incident.id}/edit`);
  };

  const handleView = (incident: Incident) => {
    router.push(`/incidents/${incident.id}`);
  };

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">事件管理</h1>
          <p className="text-gray-500 mt-1">管理和追踪系统中的所有事件记录</p>
        </div>
        <Button
          type="primary"
          icon={<Plus className="w-4 h-4" />}
          size="large"
          onClick={() => router.push('/incidents/new')}
        >
          新建事件
        </Button>
      </div>

      <IncidentStats metrics={metrics || undefined} />

      <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
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
                  <TableIcon />
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

        <div className="mb-4 flex justify-between items-center">
          <Space size="middle">
            <Input.Search
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              placeholder="搜索事件ID、标题或描述..."
              className="w-64"
              allowClear
              onSearch={handleSearch}
              defaultValue={searchKeyword}
            />
            <Button
              icon={<Filter className="w-4 h-4" />}
              onClick={() => setShowFilters(!showFilters)}
              type={showFilters ? 'primary' : 'default'}
              ghost={showFilters}
            >
              筛选
            </Button>
          </Space>
        </div>

        {showFilters && (
          <div className="mb-4 p-4 bg-gray-50 rounded-lg border border-gray-100">
            <IncidentFilters />
          </div>
        )}

        {activeTab === 'list' && (
          <>
            {incidents.length === 0 && !loading ? (
              <Empty description="暂无事件记录" image={Empty.PRESENTED_IMAGE_SIMPLE}>
                <Button type="primary" onClick={() => router.push('/incidents/new')}>
                  创建第一个事件
                </Button>
              </Empty>
            ) : (
              <IncidentList
                incidents={incidents}
                loading={loading}
                selectedRowKeys={selectedRowKeys}
                onSelectedRowKeysChange={setSelectedRowKeys}
                onEdit={handleEdit}
                onView={handleView}
              />
            )}

            {incidents.length > 0 && (
              <div className="mt-4 flex justify-end">
                <Pagination
                  current={page}
                  pageSize={pageSize}
                  total={total}
                  onChange={(p, ps) => {
                    setPage(p);
                    setPageSize(ps);
                  }}
                  showSizeChanger
                  showTotal={total => `共 ${total} 条记录`}
                />
              </div>
            )}
          </>
        )}

        {activeTab === 'kanban' && (
          <UnifiedKanbanBoard<Incident>
            items={incidents}
            loading={loading}
            getItemId={(incident: Incident) => incident.id}
            getItemStatus={(incident: Incident) => incident.status}
            getItemTitle={(incident: Incident) => incident.title || `事件 #${incident.id}`}
            getItemNumber={(incident: Incident) => incident.incident_number || String(incident.id)}
            getItemDescription={(incident: Incident) => incident.description || ''}
            getItemPriority={(incident: Incident) => incident.priority || incident.severity || 'medium'}
            getItemAssignee={(incident: Incident) =>
              incident.assignee
                ? { name: incident.assignee.name || incident.assignee_name || '未分配' }
                : null
            }
            getItemCreatedAt={(incident: Incident) => incident.created_at}
            getItemUpdatedAt={(incident: Incident) => incident.updated_at}
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
        )}
      </Card>
    </div>
  );
}
