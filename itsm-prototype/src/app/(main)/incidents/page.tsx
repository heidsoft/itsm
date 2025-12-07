'use client';

import React, { useState } from 'react';
import { Button, Space, Pagination } from 'antd';
import { DownloadOutlined, PlusOutlined } from '@ant-design/icons';
import { useIncidentsData } from './hooks/useIncidentsData';
import { IncidentStats } from './components/IncidentStats';
import { IncidentFilters } from './components/IncidentFilters';
import { IncidentList } from './components/IncidentList';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { useI18n } from '@/lib/i18n';

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

  // 使用统一的 LoadingEmptyError 组件处理加载和空态
  const renderContent = () => {
    if (loading) {
      return <LoadingEmptyError state='loading' loadingText={t('incidents.loading')} />;
    }

    if (!incidents || incidents.length === 0) {
      return (
        <LoadingEmptyError
          state='empty'
          empty={{
            title: t('incidents.emptyTitle') || '暂无事件',
            description: t('incidents.emptyDescription') || '当前没有事件数据',
            actionText: t('incidents.create') || '创建事件',
            onAction: handleCreateIncident,
            showAction: true,
          }}
        />
      );
    }

    return (
      <>
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
          showTotal={(total, range) =>
            t('incidents.paginationItems', { start: range[0], end: range[1], total })
          }
          onChange={(page, size) => {
            setCurrentPage(page);
            setPageSize(size);
          }}
          style={{ marginTop: 16, textAlign: 'right' }}
        />
      </>
    );
  };

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
        {renderContent()}
      </div>
    </div>
  );
}
