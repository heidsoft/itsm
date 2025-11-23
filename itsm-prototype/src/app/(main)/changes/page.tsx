'use client';

import React, { useState } from 'react';
import { Button, Skeleton, Badge } from 'antd';
import { Plus } from 'lucide-react';
import { useChangesData } from './hooks/useChangesData';
import { ChangeStats } from './components/ChangeStats';
import { ChangeFilters } from './components/ChangeFilters';
import { ChangeList } from './components/ChangeList';
import { useI18n } from '@/lib/i18n';

const ChangeListPageSkeleton: React.FC = () => (
  <div>
    <Skeleton active paragraph={{ rows: 4 }} />
    <Skeleton active paragraph={{ rows: 2 }} style={{ marginTop: 24 }} />
    <Skeleton active paragraph={{ rows: 8 }} style={{ marginTop: 24 }} />
  </div>
);

const ChangeListPage = () => {
  const { t } = useI18n();
  const {
    changes,
    loading,
    stats,
    filter,
    setFilter,
    searchText,
    setSearchText,
    pagination,
    handleTableChange,
    fetchChanges,
  } = useChangesData();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const handleCreateChange = () => {
    console.log('创建变更');
  };

  if (loading) {
    return <ChangeListPageSkeleton />;
  }

  return (
    <div className='p-6 bg-gray-50 min-h-screen'>
      <ChangeStats stats={stats} />
      <ChangeFilters
        loading={loading}
        searchText={searchText}
        onSearchTextChange={setSearchText}
        filter={filter}
        onFilterChange={setFilter}
        onFetchChanges={fetchChanges}
      />
      <div className='flex justify-between items-center mb-6'>
        <div>
          <h3 className='text-xl font-semibold text-gray-800 mb-1'>{t('changes.listTitle')}</h3>
          <p className='text-gray-600'>{t('changes.listDescription')}</p>
        </div>
        <div className='flex items-center space-x-3'>
          {selectedRowKeys.length > 0 && (
            <Badge
              count={selectedRowKeys.length}
              className='bg-blue-100 text-blue-800 px-3 py-1 rounded-full text-sm font-medium'
            >
              {t('changes.selected', { count: selectedRowKeys.length })}
            </Badge>
          )}
          <Button
            type='primary'
            icon={<Plus size={20} />}
            onClick={handleCreateChange}
            size='large'
            className='bg-blue-600 hover:bg-blue-700 border-blue-600 hover:border-blue-700 shadow-sm'
          >
            {t('changes.create')}
          </Button>
        </div>
      </div>
      <ChangeList
        changes={changes}
        loading={loading}
        pagination={pagination}
        selectedRowKeys={selectedRowKeys}
        onSelectedRowKeysChange={setSelectedRowKeys}
        onTableChange={handleTableChange}
      />
    </div>
  );
};

export default ChangeListPage;
