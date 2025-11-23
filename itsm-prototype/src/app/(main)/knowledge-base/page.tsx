'use client';

import React, { useState } from 'react';
import { Empty, Skeleton } from 'antd';
import { useKnowledgeBaseData } from './hooks/useKnowledgeBaseData';
import { KnowledgeBaseStats } from './components/KnowledgeBaseStats';
import { KnowledgeBaseFilters } from './components/KnowledgeBaseFilters';
import { ArticleList } from './components/ArticleList';
import { useI18n } from '@/lib/i18n';

const KnowledgeBaseSkeleton: React.FC = () => (
  <div>
    <Skeleton active paragraph={{ rows: 2 }} />
    <Skeleton active paragraph={{ rows: 2 }} style={{ marginTop: 24 }} />
    <Skeleton active paragraph={{ rows: 8 }} style={{ marginTop: 24 }} />
  </div>
);

const KnowledgeBasePage = () => {
  const { t } = useI18n();
  const { articles, loading, stats, setSearchText, setCategoryFilter, loadArticles } =
    useKnowledgeBaseData();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const handleCreateArticle = () => {
    window.location.href = '/knowledge-base/new';
  };

  if (loading) {
    return <KnowledgeBaseSkeleton />;
  }

  return (
    <div>
      <KnowledgeBaseStats stats={stats} />
      <KnowledgeBaseFilters
        loading={loading}
        onSearch={setSearchText}
        onCategoryFilterChange={setCategoryFilter}
        onCreateArticle={handleCreateArticle}
      />
      {articles.length === 0 ? (
        <Empty description={t('knowledgeBase.noMatchingArticles')} />
      ) : (
        <ArticleList
          articles={articles}
          loading={loading}
          selectedRowKeys={selectedRowKeys}
          onSelectedRowKeysChange={setSelectedRowKeys}
        />
      )}
    </div>
  );
};

export default KnowledgeBasePage;
