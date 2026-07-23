'use client';

/**
 * 知识库文章详情页面
 * B6 修复：原本 /knowledge/articles/[id] 路由 404
 */

import React, { useEffect, useState } from 'react';
import { Button, Space, message } from 'antd';
import { Archive, Send } from 'lucide-react';
import { useParams } from 'next/navigation';
import ArticleDetail from '@/components/knowledge/ArticleDetail';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import { ArticleStatus } from '@/types/knowledge-base';

export default function KnowledgeArticleDetailPage() {
  const { id } = useParams() as { id: string };
  const [status, setStatus] = useState<ArticleStatus>();
  const [submitting, setSubmitting] = useState(false);
  const [detailVersion, setDetailVersion] = useState(0);

  useEffect(() => {
    if (!id) return;

    KnowledgeBaseApi.getArticle(id)
      .then(article => setStatus(article.status))
      .catch(() => {
        // ArticleDetail displays the page-level load error.
      });
  }, [id]);

  const updatePublishStatus = async (publish: boolean) => {
    const articleId = Number(id);
    if (!Number.isSafeInteger(articleId) || articleId <= 0) {
      message.error('无效的文章 ID');
      return;
    }

    setSubmitting(true);
    try {
      const article = publish
        ? await KnowledgeBaseApi.publishArticle(articleId)
        : await KnowledgeBaseApi.unpublishArticle(articleId);
      setStatus(article.status);
      setDetailVersion(version => version + 1);
      message.success(publish ? '文章已发布' : '文章已下架');
    } catch {
      message.error(publish ? '发布文章失败' : '下架文章失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <Space style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
        {status === ArticleStatus.PUBLISHED ? (
          <Button
            danger
            icon={<Archive size={16} />}
            loading={submitting}
            onClick={() => updatePublishStatus(false)}
          >
            下架文章
          </Button>
        ) : (
          <Button
            type="primary"
            icon={<Send size={16} />}
            loading={submitting}
            onClick={() => updatePublishStatus(true)}
          >
            发布文章
          </Button>
        )}
      </Space>
      <ArticleDetail key={detailVersion} />
    </>
  );
}
