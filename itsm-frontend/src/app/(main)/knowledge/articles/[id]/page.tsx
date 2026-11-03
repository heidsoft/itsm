'use client';

/**
 * 知识库文章详情页面
 *
 * 发布/下架/提交评审全部由 ArticleDetail 内的评审状态机负责，
 * 页面层不再维护第二套 `status` 判断，避免草稿+评审中时渲染出必然 409 的发布按钮。
 */

import React from 'react';
import { useParams } from 'next/navigation';
import ArticleDetail from '@/components/knowledge/ArticleDetail';

export default function KnowledgeArticleDetailPage() {
  const { id } = useParams() as { id: string };

  return <ArticleDetail key={id} />;
}
