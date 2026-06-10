'use client';

/**
 * 知识库文章详情页面
 * B6 修复：原本 /knowledge/articles/[id] 路由 404
 */

import React from 'react';
import { useParams } from 'next/navigation';
import ArticleDetail from '@/components/knowledge/ArticleDetail';

export default function KnowledgeArticleDetailPage() {
  const params = useParams();
  const id = (params?.id as string) || '';

  return <ArticleDetail id={id} />;
}
