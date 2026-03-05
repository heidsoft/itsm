'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Button, Input, Space, Card, Pagination, message, Empty, Tabs, Table, Badge, Tag, Avatar, Tooltip, Row, Col } from 'antd';
import { Plus, Search, Filter, Edit, Eye, AlertCircle, Table as TableIcon, Clock, User, MoreOutlined } from 'lucide-react';
import { AppstoreOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import { IncidentList } from './components/IncidentList';
import { IncidentFilters } from './components/IncidentFilters';
import { IncidentStats } from './components/IncidentStats';
import { Incident } from '@/lib/api/types';
import { IncidentAPI } from '@/lib/api/incident-api';
import dayjs from 'dayjs';

// Stats data format expected by IncidentStats component
interface IncidentStatsData {
  totalIncidents?: number;
  openIncidents?: number;
  criticalIncidents?: number;
  majorIncidents?: number;
  avgResolutionTime?: number;
}

// Kanban status config
const KANBAN_STATUS_CONFIG = [
  { key: 'new', title: '新建', color: '#1890ff' },
  { key: 'investigating', title: '调查中', color: '#722ed1' },
  { key: 'identified', title: '已识别', color: '#fa8c16' },
  { key: 'resolved', title: '已解决', titleEn: 'Resolved', color: '#52c41a' },
  { key: 'closed', title: '已关闭', color: '#d9d9d9' },
];

// Priority config
const PRIORITY_CONFIG = {
  critical: { color: 'red', text: '紧急', icon: '🔴' },
  high: { color: 'orange', text: '高', icon: '🟠' },
  medium: { color: 'blue', text: '中', icon: '🔵' },
  low: { color: 'green', text: '低', icon: '🟢' },
};

// Incident Kanban Card Component
const IncidentKanbanCard: React.FC<{
  incident: Incident;
  onClick: () => void;
}> = ({ incident, onClick }) => {
  const priority = PRIORITY_CONFIG[incident.priority as keyof typeof PRIORITY_CONFIG] || PRIORITY_CONFIG.medium;

  return (
    <Card
      size="small"
      className="mb-2 cursor-pointer hover:shadow-md transition-shadow"
      onClick={onClick}
      styles={{ body: { padding: '12px' } }}
    >
      <div className="flex items-start gap-2">
        <Tag color={priority.color} className="!m-0 text-xs">
          {priority.text}
        </Tag>
        <div className="flex-1 min-w-0">
          <div className="font-medium text-sm truncate">
            {incident.title || `事件 #${incident.id}`}
          </div>
          <div className="text-xs text-gray-500 mt-1 flex items-center gap-2">
            <span>ID: {incident.id}</span>
            <span>•</span>
            <span>{dayjs(incident.created_at).format('MM-DD HH:mm')}</span>
          </div>
        </div>
      </div>
      {incident.assignee_name && (
        <div className="mt-2 flex items-center gap-1">
          <Avatar size="small" className="!text-xs">{incident.assignee_name[0]}</Avatar>
          <span className="text-xs text-gray-500">{incident.assignee_name}</span>
        </div>
      )}
    </Card>
  );
};

// Incident Kanban Board Component
const IncidentKanbanBoard: React.FC<{
  incidents: Incident[];
  onIncidentClick: (incident: Incident) => void;
  loading: boolean;
}> = ({ incidents, onIncidentClick, loading }) => {
  // Group incidents by status
  const incidentsByStatus = useMemo(() => {
    const grouped: Record<string, Incident[]> = {};
    KANBAN_STATUS_CONFIG.forEach(status => {
      grouped[status.key] = incidents.filter(
        inc => (inc.status as string) === status.key || inc.status === status.title
      );
    });
    return grouped;
  }, [incidents]);

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <span>加载中...</span>
      </div>
    );
  }

  return (
    <Row gutter={[16, 16]} className="!mt-4">
      {KANBAN_STATUS_CONFIG.map(status => (
        <Col key={status.key} xs={24} sm={12} md={8} lg={4}>
          <div
            className="rounded-lg h-full"
            style={{ backgroundColor: '#f5f5f5' }}
          >
            <div
              className="p-3 rounded-t-lg flex items-center justify-between"
              style={{ backgroundColor: status.color }}
            >
              <span className="font-medium text-white">{status.title}</span>
              <Badge
                count={incidentsByStatus[status.key]?.length || 0}
                style={{ backgroundColor: 'rgba(255,255,255,0.3)' }}
              />
            </div>
            <div className="p-2 min-h-[400px] max-h-[600px] overflow-y-auto">
              {incidentsByStatus[status.key]?.length === 0 ? (
                <div className="text-center text-gray-400 py-8 text-sm">
                  暂无事件
                </div>
              ) : (
                incidentsByStatus[status.key]?.map(incident => (
                  <IncidentKanbanCard
                    key={incident.id}
                    incident={incident}
                    onClick={() => onIncidentClick(incident)}
                  />
                ))
              )}
            </div>
          </div>
        </Col>
      ))}
    </Row>
  );
};

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
  const [activeTab, setActiveTab] = useState('list');

  // Fetch incidents
  const fetchIncidents = async () => {
    setLoading(true);
    try {
      const response = await IncidentAPI.listIncidents({
        page: 1,
        page_size: 100, // Get more for Kanban view
        search: searchKeyword || undefined,
      });
      const resp = response as unknown as Record<string, unknown>;
      const items = resp.items as Incident[] | undefined;
      const totalCount = resp.total as number | undefined;
      setIncidents((items || []) as unknown as Incident[]);
      setTotal(totalCount || 0);
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
  }, [searchKeyword]);

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
          <IncidentKanbanBoard
            incidents={incidents}
            onIncidentClick={handleView}
            loading={loading}
          />
        )}
      </Card>
    </div>
  );
}
