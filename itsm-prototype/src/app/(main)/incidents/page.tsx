'use client';

import React, { useState } from 'react';
import { Button, Space, Pagination, Skeleton } from 'antd';
import { DownloadOutlined, PlusOutlined } from '@ant-design/icons';
import { useIncidentsData } from './hooks/useIncidentsData';
import { IncidentStats } from './components/IncidentStats';
import { IncidentFilters } from './components/IncidentFilters';
import { IncidentList } from './components/IncidentList';
import { useI18n } from '@/lib/i18n';

const IncidentsPageSkeleton: React.FC = () => (
  <div>
    <Skeleton active paragraph={{ rows: 4 }} />
    <Skeleton active paragraph={{ rows: 2 }} style={{ marginTop: 24 }} />
    <Skeleton active paragraph={{ rows: 8 }} style={{ marginTop: 24 }} />
  </div>
);

export default function IncidentsPage() {
  const { t } = useI18n();
  const {
    incidents,
    loading,
    total,
    currentPage,
    pageSize,
    metrics,
    setCurrentPage,
    setPageSize,
    handleSearch,
    handleFilterChange,
    loadIncidents,
  } = useIncidentsData();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const handleCreateIncident = () => {
    window.location.href = '/incidents/new';
  };

  if (loading) {
    return <IncidentsPageSkeleton />;
  }

  return (
    <div>
      <IncidentStats metrics={metrics} />
      <IncidentFilters
        loading={loading}
        onSearch={handleSearch}
        onFilterChange={handleFilterChange}
        onRefresh={loadIncidents}
      />
      <div className='bg-white/60 backdrop-blur-sm rounded-2xl p-6 border border-white/20 shadow-lg mb-6'>
        <div className='flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6'>
          <div className='flex items-center space-x-4'>
            <div>
              <h3 className='text-lg font-semibold text-gray-800'>{t('incidents.listTitle')}</h3>
              <p className='text-sm text-gray-600'>{t('incidents.listDescription')}</p>
            </div>
          </div>
          <Space size='middle' className='flex-wrap'>
            <Button
              icon={<DownloadOutlined />}
              size='large'
              className='bg-white border-gray-200 text-gray-700 hover:bg-gray-50 hover:border-gray-300 shadow-sm hover:shadow-md transition-all duration-200 rounded-lg font-medium'
            >
              {t('incidents.export')}
            </Button>
            <Button
              type='primary'
              icon={<PlusOutlined />}
              size='large'
              onClick={handleCreateIncident}
              className='bg-gradient-to-r from-green-500 to-emerald-600 border-0 text-white hover:from-green-600 hover:to-emerald-700 shadow-lg hover:shadow-xl transition-all duration-200 rounded-lg font-medium'
            >
              {t('incidents.create')}
            </Button>
          </Space>
        </div>
        <IncidentList
          incidents={incidents}
          loading={loading}
          selectedRowKeys={selectedRowKeys}
          onSelectedRowKeysChange={setSelectedRowKeys}
          onEdit={incident => (window.location.href = `/incidents/${incident.id}/edit`)}
          onView={incident => (window.location.href = `/incidents/${incident.id}`)}
        />
        <Pagination
          current={currentPage}
          pageSize={pageSize}
          total={total}
          showSizeChanger
          showQuickJumper
          showTotal={(total, range) => t('incidents.paginationItems', { start: range[0], end: range[1], total })}
          onChange={(page, size) => {
            setCurrentPage(page);
            setPageSize(size);
          }}
          style={{ marginTop: 16, textAlign: 'right' }}
        />
      </div>
    </div>
  );
}
