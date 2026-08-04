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
} from 'antd';
import { ArrowLeft, Pencil, User, Folder, Calendar, CheckCircle, Archive, ThumbsUp, ThumbsDown } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import {
  KnowledgeStatus,
  KnowledgeStatusLabels,
  KnowledgeStatusColors,
} from '@/constants/knowledge';
import type { KnowledgeArticle } from '@/types/biz/knowledge';
import ArticleVersionControl from './ArticleVersionControl';

const { Title, Paragraph, Text } = Typography;

const ArticleDetail: React.FC = () => {
  const { id } = useParams() as { id: string };
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [article, setArticle] = useState<KnowledgeArticle | null>(null);
  const [actionLoading, setActionLoading] = useState<'publish' | 'unpublish' | 'archive' | null>(null);
  const [helpful, setHelpful] = useState<boolean | null>(null);
  const [feedbackSubmitted, setFeedbackSubmitted] = useState(false);
  const [feedbackComment, setFeedbackComment] = useState('');

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

  const handleArchive = () => {
    if (!article) return;
    Modal.confirm({
      title: '确认归档该文章？',
      content: '归档后将从默认列表中隐藏，可在归档管理中找回。',
      okText: '归档',
      cancelText: '取消',
      onOk: async () => {
        setActionLoading('archive');
        try {
          const updated = await KnowledgeBaseApi.archiveArticle(article.id.toString());
          setArticle(updated as unknown as KnowledgeArticle);
          message.success('文章已归档');
        } catch (e: any) {
          message.error('归档失败：' + (e?.message || '未知错误'));
        } finally {
          setActionLoading(null);
        }
      },
    });
  };

  const handleFeedback = (isHelpful: boolean) => {
    setHelpful(isHelpful);
    setFeedbackSubmitted(true);
    message.success(isHelpful ? '感谢您的肯定！' : '感谢您的反馈，我们会改进');
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
              {status === KnowledgeStatus.DRAFT && (
                <Button
                  type="primary"
                  icon={<CheckCircle />}
                  onClick={handlePublish}
                  loading={actionLoading === 'publish'}
                >
                  发布文章
                </Button>
              )}
              {status === KnowledgeStatus.PUBLISHED && (
                <Button
                  icon={<CheckCircle />}
                  onClick={handleUnpublish}
                  loading={actionLoading === 'unpublish'}
                >
                  取消发布
                </Button>
              )}
              <Button
                icon={<Archive />}
                onClick={handleArchive}
                loading={actionLoading === 'archive'}
              >
                归档
              </Button>
              <Button
                type="primary"
                icon={<Pencil />}
                onClick={() => router.push(`/knowledge/articles/${article.id}/edit`)}
              >
                编辑文章
              </Button>
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
                  <Paragraph>
                    {article.content.split('\n').map((line, i) => (
                      <React.Fragment key={i}>
                        {line}
                        <br />
                      </React.Fragment>
                    ))}
                  </Paragraph>
                  {/* Helpfulness Feedback */}
                  <Divider />
                  <div style={{ padding: '16px 0', background: '#f9f9f9', borderRadius: 8, textAlign: 'center' }}>
                    {feedbackSubmitted ? (
                      <div>
                        <Text type="secondary">感谢您的反馈！</Text>
                      </div>
                    ) : (
                      <Space direction="vertical" size="small">
                        <Text strong>这篇文章对您有帮助吗？</Text>
                        <Space>
                          <Button
                            icon={<ThumbsUp size={14} />}
                            onClick={() => handleFeedback(true)}
                            type={helpful === true ? 'primary' : 'default'}
                          >
                            有帮助
                          </Button>
                          <Button
                            icon={<ThumbsDown size={14} />}
                            onClick={() => handleFeedback(false)}
                            type={helpful === false ? 'primary' : 'default'}
                          >
                            需要改进
                          </Button>
                        </Space>
                      </Space>
                    )}
                  </div>
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
