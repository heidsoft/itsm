'use client';

/**
 * 知识库文章详情组件
 */

import React, { useState, useEffect } from 'react';
import {
  Card,
  Tag,
  Button,
  Skeleton,
  Result,
  Typography,
  Space,
  Breadcrumb,
  message,
  Divider,
  Tabs,
  Modal,
  Rate,
  Input,
  Tooltip,
} from 'antd';
import { ArrowLeft, Pencil, User, Folder, Calendar, CheckCircle, Link2 } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import { usePermissions } from '@/lib/hooks/use-permissions';
import {
  KnowledgeStatus,
  KnowledgeStatusLabels,
  KnowledgeStatusColors,
  ReviewStatusLabels,
  ReviewStatusColors,
} from '@/constants/knowledge';
import type { KnowledgeArticle } from '@/types/biz/knowledge';
import ArticleVersionControl from './ArticleVersionControl';
// 文章内容用 react-markdown 渲染，rehype-sanitize 负责防 XSS。
// 相比纯文本按行拆 + <br/>，这里能保留标题、加粗、行内代码、列表和表格。
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeSanitize from 'rehype-sanitize';

const { Title, Paragraph, Text } = Typography;

const ArticleDetail: React.FC = () => {
  const { id } = useParams() as { id: string };
  const router = useRouter();
  const { hasPermission } = usePermissions();
  // 发布/取消发布/更新在后端统一挂 knowledge:write；最终用户只有 knowledge:read，
  // 此前按钮照常渲染，点下去只剩 403。
  const canWriteArticle = hasPermission('knowledge', 'write');
  const [loading, setLoading] = useState(true);
  const [article, setArticle] = useState<KnowledgeArticle | null>(null);
  const [actionLoading, setActionLoading] = useState<
    'publish' | 'unpublish' | 'submit' | null
  >(null);

  useEffect(() => {
    if (id) {
      loadDetail();
    }

  }, [id]);

  const loadDetail = async () => {
    setLoading(true);
    try {
      const data = await KnowledgeBaseApi.getArticle(id);
      setArticle(data as unknown as KnowledgeArticle);
      // 设置页面标题
      if (typeof document !== 'undefined' && data?.title) {
        document.title = `${data.title} - Knowledge Base`;
      }
    } catch (error) {
      // console.error(error);
      message.error('Failed to load article details');
    } finally {
      setLoading(false);
    }
  };

  const handlePublish = () => {
    if (!article) return;
    Modal.confirm({
      title: 'Confirm to publish this article?',
      content: 'Once published, it will be visible to all users with permissions.',
      okText: 'Release',
      cancelText: 'Cancel',
      onOk: async () => {
        setActionLoading('publish');
        try {
          const updated = await KnowledgeBaseApi.publishArticle(article.id);
          setArticle(updated as unknown as KnowledgeArticle);
          message.success('Article published');
        } catch (e: any) {
          message.error('Publishing failed:' + (e?.message || 'Unknown error'));
        } finally {
          setActionLoading(null);
        }
      },
    });
  };

  const handleUnpublish = () => {
    if (!article) return;
    Modal.confirm({
      title: 'Confirm to cancel publishing?',
      content: 'The article will return to draft status after unpublishing.',
      okText: 'Unpublish',
      cancelText: 'Back',
      onOk: async () => {
        setActionLoading('unpublish');
        try {
          const updated = await KnowledgeBaseApi.unpublishArticle(article.id);
          setArticle(updated as unknown as KnowledgeArticle);
          message.success('Unpublished');
        } catch (e: any) {
          message.error('Operation failed:' + (e?.message || 'Unknown error'));
        } finally {
          setActionLoading(null);
        }
      },
    });
  };

  // 提交评审：评审流的前半段此前只有后端端点、UI 无入口，
  // 作者只能把草稿留在本地状态，审批页永远收不到待办。
  const handleSubmitReview = () => {
    if (!article) return;
    Modal.confirm({
      title: 'Confirm submission for review?',
      content: 'After submission, it enters the review queue and must be approved by other reviewers (self-review is not possible) before it can be visible to the outside world.',
      okText: 'Submit for review',
      cancelText: 'Cancel',
      onOk: async () => {
        setActionLoading('submit');
        try {
          const updated = await KnowledgeBaseApi.submitReview(String(article.id));
          setArticle(updated as unknown as KnowledgeArticle);
          message.success('Submitted for review, waiting for approval by other reviewers');
        } catch (e: unknown) {
          const err = e as { message?: string };
          message.error('Submission for review failed:' + (err?.message || 'Unknown error'));
        } finally {
          setActionLoading(null);
        }
      },
    });
  };

  if (loading)
    return (
      <Card>
        <Skeleton active />
      </Card>
    );

  if (!article) {
    return (
      <Card>
        <Result
          status="404"
          title="404"
          subTitle="Sorry, the article you visited does not exist"
          extra={
            <Button type="primary" onClick={() => router.push('/knowledge')}>

              Back to List
            </Button>
          }
        />
      </Card>
    );
  }

  const status =
    article.status === KnowledgeStatus.PUBLISHED
      ? KnowledgeStatus.PUBLISHED
      : KnowledgeStatus.DRAFT;

  // 评审阶段是发布按钮的唯一依据（后端 /publish 对未过评审返回 409/4090）。
  // reviewStatus 缺失时按可见性推断：已发布≈approved，其余≈draft。
  const reviewStatus: 'draft' | 'under_review' | 'approved' | 'rejected' =
    article.reviewStatus ?? (status === KnowledgeStatus.PUBLISHED ? 'approved' : 'draft');
  const canPublish = reviewStatus === 'approved';
  const canSubmitReview = reviewStatus === 'draft' || reviewStatus === 'rejected';

  // 作者/分类展示值：后端可能只回填名称之一，缺失时不渲染「作者: 」空壳。
  const authorName = (article.authorName || article.author || '').trim();
  const rawCategory = (article.categoryName || article.category || '').trim();
  // 旧版曾把分类 ID 当名称写入，纯数字不是有效分类名，按未分类展示。
  const articleCategory = /^\d+$/.test(rawCategory) ? '' : rawCategory;

  return (
    <div style={{ padding: '0 0 24px' }}>
      <Breadcrumb style={{ marginBottom: 16 }}>
        <Breadcrumb.Item>Home page</Breadcrumb.Item>
        <Breadcrumb.Item>Knowledge Base</Breadcrumb.Item>
        <Breadcrumb.Item onClick={() => router.push('/knowledge')}>Articles</Breadcrumb.Item>
        <Breadcrumb.Item>Article details</Breadcrumb.Item>
      </Breadcrumb>

      <Card>
        <div style={{ marginBottom: 24 }}>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: 16,
            }}
          >
            <Button icon={<ArrowLeft />} onClick={() => router.push('/knowledge')}>

              Back to List
            </Button>
            <Space>
              {/* 发布 = 评审通过的可见性投影：未过评审时只给「提交评审」，
                  不再渲染一个必然 409 的发布按钮（后端门禁见 Service.SetArticlePublished）。 */}
              {canWriteArticle && status === KnowledgeStatus.DRAFT && canPublish && (
                <Button
                  type="primary"
                  icon={<CheckCircle />}
                  onClick={handlePublish}
                  loading={actionLoading === 'publish'}
                >

                  Publish
                </Button>
              )}
              {canWriteArticle && status === KnowledgeStatus.DRAFT && canSubmitReview && (
                <Button
                  type="primary"
                  icon={<CheckCircle />}
                  onClick={handleSubmitReview}
                  loading={actionLoading === 'submit'}
                >

                  Submit for review
                </Button>
              )}
              {canWriteArticle && status === KnowledgeStatus.DRAFT && reviewStatus === 'under_review' && (
                <Tooltip title="Submitted for review, waiting for approval from other reviewers (self-review is not allowed)">
                  <Button disabled icon={<CheckCircle />}>

                    Under review
                  </Button>
                </Tooltip>
              )}
              {canWriteArticle && status === KnowledgeStatus.PUBLISHED && (
                <Button
                  icon={<CheckCircle />}
                  onClick={handleUnpublish}
                  loading={actionLoading === 'unpublish'}
                >

                  Unpublish
                </Button>
              )}
              {canWriteArticle && (
                <Button
                  type="primary"
                  icon={<Pencil />}
                  onClick={() => router.push(`/knowledge/articles/${article.id}/edit`)}
                >

                  Edit Article
                </Button>
              )}
            </Space>
          </div>

          <Title level={2}>{article.title}</Title>

          <Space split={<Divider type="vertical" />} wrap>
            {authorName && (
              <Space>
                <User />
                <Text type="secondary">Author: {authorName}</Text>
              </Space>
            )}
            <Space>
              <Calendar />
              <Text type="secondary">{dayjs(article.createdAt).format('YYYY-MM-DD HH:mm')}</Text>
            </Space>
            <Space>
              <Folder />
              <Text type="secondary">{articleCategory || 'Uncategorized'}</Text>
            </Space>
            {/* 来源追溯（2026-10-22 沉淀闭环）：由问题沉淀而来的文章回链问题详情，
                与问题详情页「知识沉淀」Tab 构成双向闭环；手工创建的文章不显示此行。 */}
            {article.sourceProblemId != null && (
              <Space>
                <Link2 />
                <Button
                  type="link"
                  size="small"
                  style={{ padding: 0 }}
                  onClick={() => router.push(`/problems/${article.sourceProblemId}`)}
                >

                  Source problem #{article.sourceProblemId}
                </Button>
              </Space>
            )}
            {/* 可见性与评审阶段是两条轴：已发布单独标注，评审阶段始终展示，
                避免出现「徽标=草稿、按钮=评审中」的语义冲突。 */}
            {status === KnowledgeStatus.PUBLISHED && (
              <Tag color={KnowledgeStatusColors[KnowledgeStatus.PUBLISHED]}>
                {KnowledgeStatusLabels[KnowledgeStatus.PUBLISHED]}
              </Tag>
            )}
            <Tag color={ReviewStatusColors[reviewStatus]}>{ReviewStatusLabels[reviewStatus]}</Tag>
          </Space>
        </div>

        <div style={{ marginBottom: 16 }}>
          {article.tags?.map(tag => (
            <Tag key={tag}>{tag}</Tag>
          ))}
        </div>

        <Divider />

        <Tabs
          defaultActiveKey="content"
          items={[
            {
              key: 'content',
              label: 'Article Content',
              children: (
                <div className="article-content" style={{ minHeight: 400 }}>
                  {/*
                    文章内容采用 Markdown 格式。渲染走 react-markdown + remark-gfm（表格/任务列表/删除线）
                    + rehype-sanitize（默认白名单剥离 <script>/<iframe>/onerror 等危险元素与属性），
                    这样既保留原始语义，又避免 XSS。原文逐行 <br/> 拆分不能呈现标题、列表、代码块。
                  */}
                  {article.content ? (
                    <div className="prose max-w-none">
                      <ReactMarkdown
                        remarkPlugins={[remarkGfm]}
                        rehypePlugins={[rehypeSanitize]}
                      >
                        {article.content}
                      </ReactMarkdown>
                    </div>
                  ) : (
                    <Paragraph type="secondary">This article has no content yet.</Paragraph>
                  )}
                </div>
              ),
            },
            {
              key: 'versions',
              label: 'Version History',
              children: (
                <ArticleVersionControl
                  articleId={article.id.toString()}
                  currentVersion={1}
                  onVersionChange={version => {
                    message.info(`Switch to version ${version}`);
                  }}
                />
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
};

export default ArticleDetail;
