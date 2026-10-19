'use client';

/**
 * 知识库文章详情页面
 */

import React, { useEffect, useState } from 'react';
import { App, Button, Space } from 'antd';
import { Archive, Send } from 'lucide-react';
import { useParams } from 'next/navigation';
import ArticleDetail from '@/components/knowledge/ArticleDetail';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import { ArticleStatus } from '@/types/knowledge-base';
import { useI18n } from '@/lib/i18n/useI18n';
import { usePermissions } from '@/lib/hooks/use-permissions';

export default function KnowledgeArticleDetailPage() {
  const { t } = useI18n();
  const { message } = App.useApp();
  const { hasPermission } = usePermissions();
  // 发布/下架在 router 里挂 knowledge:write，只读角色不应看到可点按钮。
  const canWriteArticle = hasPermission('knowledge', 'write');
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
      message.error(t('knowledgeBase.invalidArticleId'));
      return;
    }

    setSubmitting(true);
    try {
      const article = publish
        ? await KnowledgeBaseApi.publishArticle(articleId)
        : await KnowledgeBaseApi.unpublishArticle(articleId);
      setStatus(article.status);
      setDetailVersion(version => version + 1);
      message.success(publish ? t('knowledgeBase.articlePublished') : t('knowledgeBase.articleUnpublished'));
    } catch {
      message.error(publish ? t('knowledgeBase.publishFailed') : t('knowledgeBase.unpublishFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <Space style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
        {canWriteArticle &&
          (status === ArticleStatus.PUBLISHED ? (
            <Button
              danger
              icon={<Archive size={16} />}
              loading={submitting}
              onClick={() => updatePublishStatus(false)}
            >
              {t('knowledgeBase.unpublish')}
            </Button>
          ) : (
            <Button
              type="primary"
              icon={<Send size={16} />}
              loading={submitting}
              onClick={() => updatePublishStatus(true)}
            >
              {t('knowledgeBase.publish')}
            </Button>
          ))}
      </Space>
      <ArticleDetail key={detailVersion} />
    </>
  );
}