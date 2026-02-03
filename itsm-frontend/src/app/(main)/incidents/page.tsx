'use client';

import React, { useState, useEffect } from 'react';
import { Button, Input, Space, Card, Pagination, message } from 'antd';
import { Plus, Search, Filter } from 'lucide-react';
import { IncidentList } from './components/IncidentList';
import { IncidentFilters } from './components/IncidentFilters';
import { IncidentStats } from './components/IncidentStats';
import { Incident } from '@/lib/api/types';
import { incidentApi } from '@/lib/api/incident-api';

export default function IncidentsPage() {
  const [loading, setLoading] = useState(false);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [showFilters, setShowFilters] = useState(false);

  // Fetch incidents
  const fetchIncidents = async () => {
    setLoading(true);
    try {
      // TODO: Replace with actual API call with filters
      const response = await incidentApi.list({
        page,
        page_size: pageSize,
      });
      setIncidents(response.items);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to fetch incidents:', error);
      message.error('获取事件列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchIncidents();
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
        <Button type='primary' icon={<Plus className='w-4 h-4' />} size='large'>
          新建事件
        </Button>
      </div>

      <IncidentStats className='mb-6' />

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

        <IncidentList
          incidents={incidents}
          loading={loading}
          selectedRowKeys={selectedRowKeys}
          onSelectedRowKeysChange={setSelectedRowKeys}
          onEdit={handleEdit}
          onView={handleView}
        />

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
      </Card>
    </div>
  );
}
