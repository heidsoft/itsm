'use client';

import React, { useState } from 'react';
import { Button, Skeleton, Badge } from 'antd';
import { PlusCircle } from 'lucide-react';
import { useProblemsData } from './hooks/useProblemsData';
import { ProblemStats } from './components/ProblemStats';
import { ProblemFilters } from './components/ProblemFilters';
import { ProblemList } from './components/ProblemList';
import { useI18n } from '@/lib/i18n';
import { useRouter } from 'next/navigation';

const ProblemListPageSkeleton: React.FC = () => (
  <div>
    <Skeleton active paragraph={{ rows: 4 }} />
    <Skeleton active paragraph={{ rows: 2 }} style={{ marginTop: 24 }} />
    <Skeleton active paragraph={{ rows: 8 }} style={{ marginTop: 24 }} />
  </div>
);

const ProblemListPage = () => {
  const { t } = useI18n();
  const router = useRouter();
  const {
    problems,
    loading,
    filter,
    setFilter,
    searchText,
    setSearchText,
    pagination,
    handleTableChange,
    stats,
    fetchProblems,
  } = useProblemsData();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const handleCreateProblem = () => {
    router.push('/problems/new');
  };

  if (loading) {
    return <ProblemListPageSkeleton />;
  }

  return (
    <div>
      <ProblemStats metrics={stats} />
      <ProblemFilters
        loading={loading}
        onSearch={setSearchText}
        onFilterChange={setFilter}
        onRefresh={fetchProblems}
      />
      <div className='mb-6 flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4'>
        <div className='flex flex-col sm:flex-row sm:items-center gap-4'>
          <div>
            <h2 className='text-xl font-semibold text-gray-900'>{t('problems.listTitle')}</h2>
            <p className='text-sm text-gray-500 mt-1'>{t('problems.listDescription')}</p>
          </div>
          {selectedRowKeys.length > 0 && (
            <Badge
              count={selectedRowKeys.length}
              className='bg-blue-100 text-blue-800 px-3 py-1 rounded-full text-sm font-medium'
            >
              <span className='text-sm text-gray-600'>
                {t('problems.selected', { count: selectedRowKeys.length })}
              </span>
            </Badge>
          )}
        </div>
        <div className='flex gap-3'>
          <Button
            type='primary'
            icon={<PlusCircle size={16} />}
            onClick={handleCreateProblem}
            className='bg-blue-600 hover:bg-blue-700 border-blue-600 hover:border-blue-700 rounded-md shadow-sm transition-colors duration-200 flex items-center gap-2'
          >
            {t('problems.create')}
          </Button>
        </div>
      </div>
      <ProblemList
        problems={problems}
        loading={loading}
        selectedRowKeys={selectedRowKeys}
        onSelectedRowKeysChange={setSelectedRowKeys}
        pagination={pagination}
        onTableChange={handleTableChange}
      />
    </div>
  );
};

export default ProblemListPage;
