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
        document.title = `${data.title} - 知识库`;
      }
    } catch (error) {
      // console.error(error);
      message.error('加载文章详情失败');
    } finally {
      setLoading(false);
    }
  };

  const handlePublish = () => {
    if (!article) return;
    Modal.confirm({
      title: '确认发布该文章？',
      content: '发布后将对所有有权限的用户可见。',
      okText: '发布',
      cancelText: '取消',
      onOk: async () => {
        setActionLoading('publish');
        try {
          const updated = await KnowledgeBaseApi.publishArticle(article.id);
          setArticle(updated as unknown as KnowledgeArticle);
          message.success('文章已发布');
        } catch (e: any) {
          message.error('发布失败：' + (e?.message || '未知错误'));
        } finally {
          setActionLoading(null);
        }
      },
    });
  };

  const handleUnpublish = () => {
    if (!article) return;
    Modal.confirm({
      title: '确认取消发布？',
      content: '取消发布后文章将变回草稿状态。',
      okText: '取消发布',
      cancelText: '返回',
      onOk: async () => {
        setActionLoading('unpublish');
        try {
          const updated = await KnowledgeBaseApi.unpublishArticle(article.id);
          setArticle(updated as unknown as KnowledgeArticle);
          message.success('已取消发布');
        } catch (e: any) {
          message.error('操作失败：' + (e?.message || '未知错误'));
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
      title: '确认提交评审？',
      content: '提交后进入评审队列，须由其他评审人审批通过（不能自审）才会对外可见。',
      okText: '提交评审',
      cancelText: '取消',
      onOk: async () => {
        setActionLoading('submit');
        try {
          const updated = await KnowledgeBaseApi.submitReview(String(article.id));
          setArticle(updated as unknown as KnowledgeArticle);
          message.success('已提交评审，等待其他评审人审批');
        } catch (e: unknown) {
          const err = e as { message?: string };
          message.error('提交评审失败：' + (err?.message || '未知错误'));
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
          subTitle="抱歉，您访问的文章不存在"
          extra={
            <Button type="primary" onClick={() => router.push('/knowledge')}>
              返回列表
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

  return (
    <div style={{ padding: '0 0 24px' }}>
      <Breadcrumb style={{ marginBottom: 16 }}>
        <Breadcrumb.Item>首页</Breadcrumb.Item>
        <Breadcrumb.Item>知识库</Breadcrumb.Item>
        <Breadcrumb.Item onClick={() => router.push('/knowledge')}>文章列表</Breadcrumb.Item>
        <Breadcrumb.Item>文章详情</Breadcrumb.Item>
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
              返回列表
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
                  发布文章
                </Button>
              )}
              {canWriteArticle && status === KnowledgeStatus.DRAFT && canSubmitReview && (
                <Button
                  type="primary"
                  icon={<CheckCircle />}
                  onClick={handleSubmitReview}
                  loading={actionLoading === 'submit'}
                >
                  提交评审
                </Button>
              )}
              {canWriteArticle && status === KnowledgeStatus.DRAFT && reviewStatus === 'under_review' && (
                <Tooltip title="已送审，等待其他评审人审批（不能自审）">
                  <Button disabled icon={<CheckCircle />}>
                    评审中
                  </Button>
                </Tooltip>
              )}
              {canWriteArticle && status === KnowledgeStatus.PUBLISHED && (
                <Button
                  icon={<CheckCircle />}
                  onClick={handleUnpublish}
                  loading={actionLoading === 'unpublish'}
                >
                  取消发布
                </Button>
              )}
              {canWriteArticle && (
                <Button
                  type="primary"
                  icon={<Pencil />}
                  onClick={() => router.push(`/knowledge/articles/${article.id}/edit`)}
                >
                  编辑文章
                </Button>
              )}
            </Space>
          </div>

          <Title level={2}>{article.title}</Title>

          <Space split={<Divider type="vertical" />} wrap>
            <Space>
              <User />
              <Text type="secondary">作者: {article.author}</Text>
            </Space>
            <Space>
              <Calendar />
              <Text type="secondary">{dayjs(article.createdAt).format('YYYY-MM-DD HH:mm')}</Text>
            </Space>
            <Space>
              <Folder />
              <Text type="secondary">{article.category || '未分类'}</Text>
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
                  来源问题 #{article.sourceProblemId}
                </Button>
              </Space>
            )}
            <Tag color={KnowledgeStatusColors[status]}>{KnowledgeStatusLabels[status]}</Tag>
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
              label: '文章内容',
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
                    <Paragraph type="secondary">本文暂无内容。</Paragraph>
                  )}
                </div>
              ),
            },
            {
              key: 'versions',
              label: '版本历史',
              children: (
                <ArticleVersionControl
                  articleId={article.id.toString()}
                  currentVersion={1}
                  onVersionChange={version => {
                    message.info(`切换到版本 ${version}`);
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
