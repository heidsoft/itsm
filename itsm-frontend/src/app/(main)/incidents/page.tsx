'use client';

import React, { useState, useEffect } from 'react';
import { Button, Input, Space, Card, Pagination, message, Empty } from 'antd';
import { Plus, Search, Filter } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { IncidentList } from './components/IncidentList';
import { IncidentFilters } from './components/IncidentFilters';
import { IncidentStats } from './components/IncidentStats';
import { Incident } from '@/lib/api/types';
import { IncidentAPI } from '@/lib/api/incident-api';

// Stats data format expected by IncidentStats component
interface IncidentStatsData {
  totalIncidents?: number;
  openIncidents?: number;
  criticalIncidents?: number;
  majorIncidents?: number;
  avgResolutionTime?: number;
}

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

  // Fetch incidents
  const fetchIncidents = async () => {
    setLoading(true);
    try {
      const response = await IncidentAPI.listIncidents({
        page,
        page_size: pageSize,
      });
      // HTTP client already extracts data, so use response directly
      // Backend returns items array, not incidents
      const resp = response as unknown as Record<string, unknown>;
      const items = resp.items as Incident[] | undefined;
      const totalCount = resp.total as number | undefined;
      setIncidents((items || []) as unknown as Incident[]);
      setTotal(totalCount || 0);
    } catch (error) {
      console.error('Failed to fetch incidents:', error);
      // Use empty data on error
      setIncidents([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  };

  // Fetch stats
  const fetchStats = async () => {
    try {
      const stats = await IncidentAPI.getIncidentMetrics() as unknown as Record<string, unknown>;
      // Transform snake_case to camelCase for IncidentStats component
      setMetrics({
        totalIncidents: stats.total_incidents as number,
        openIncidents: stats.open_incidents as number,
        criticalIncidents: stats.critical_incidents as number,
        majorIncidents: stats.major_incidents as number,
        avgResolutionTime: stats.avg_resolution_time as number,
      });
    } catch (error) {
      console.error('Failed to fetch incident stats:', error);
    }
  };

  useEffect(() => {
    fetchIncidents();
    fetchStats();
  }, [page, pageSize]);

  const handleEdit = (incident: Incident) => {
    console.log('Edit incident:', incident);
  };

  const handleView = (incident: Incident) => {
    console.log('View incident:', incident);
  };

  return (
    <div className='p-6 min-h-screen bg-gray-50'>
      <div className='mb-6 flex justify-between items-center'>
        <div>
          <h1 className='text-2xl font-bold text-gray-900'>事件管理</h1>
          <p className='text-gray-500 mt-1'>管理和追踪系统中的所有事件记录</p>
        </div>
        <Button type='primary' icon={<Plus className='w-4 h-4' />} size='large' onClick={() => router.push('/incidents/new')}>
          新建事件
        </Button>
      </div>

      <IncidentStats metrics={metrics || undefined} />

      <Card className='rounded-lg shadow-sm border border-gray-200' variant="borderless">
        <div className='mb-4 flex justify-between items-center'>
          <Space size='middle'>
            <Input
              prefix={<Search className='w-4 h-4 text-gray-400' />}
              placeholder='搜索事件ID、标题或描述...'
              className='w-64'
              allowClear
            />
            <Button
              icon={<Filter className='w-4 h-4' />}
              onClick={() => setShowFilters(!showFilters)}
              type={showFilters ? 'primary' : 'default'}
              ghost={showFilters}
            >
              筛选
            </Button>
          </Space>
        </div>

        {showFilters && (
          <div className='mb-4 p-4 bg-gray-50 rounded-lg border border-gray-100'>
            <IncidentFilters />
          </div>
        )}

        {incidents.length === 0 && !loading ? (
          <Empty
            description='暂无事件记录'
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          >
            <Button type='primary' onClick={() => router.push('/incidents/new')}>
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
          <div className='mt-4 flex justify-end'>
            <Pagination
              current={page}
              pageSize={pageSize}
              total={total}
              onChange={(p, ps) => {
                setPage(p);
                setPageSize(ps);
              }}
              showSizeChanger
              showTotal={(total) => `共 ${total} 条记录`}
            />
          </div>
        )}
      </Card>
    </div>
  );
}
