'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Select,
  message,
  Modal,
  Descriptions,
  Divider,
  Empty,
  Typography,
  Tabs,
} from 'antd';
import { Pencil, Eye, Clock, MessageSquare, AlertCircle, CheckCircle, XCircle } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { PageContainer } from '@/components/layout/PageContainer';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import type { ReviewArticleRequest } from '@/types/knowledge-base';

// 本地定义的类型以匹配API响应
interface ArticleItem {
  id: number | string;
  title: string;
  content: string;
  summary?: string;
  category?: string;
  tags?: string[];
  status: string;
  author?: string;
  authorName?: string;
  authorId?: number;
  submittedAt?: string;
  createdAt?: string;
  updatedAt?: string;
}
import { useI18n } from '@/lib/i18n/useI18n';
import dayjs from 'dayjs';
import type { ColumnsType } from 'antd/es/table';

const { Text, Paragraph } = Typography;

// 状态标签映射
const statusTagMap: Record<string, { color: string; text: string; icon?: React.ReactNode }> = {
  draft: { color: 'default', text: '草稿' },
  pendingReview: { color: 'orange', text: '待审核', icon: <Clock /> },
  approved: { color: 'green', text: '已发布', icon: <CheckCircle /> },
  rejected: { color: 'red', text: '已拒绝', icon: <XCircle /> },
  archived: { color: 'default', text: '已归档' },
};

export default function KnowledgeReviewListPage() {
  const router = useRouter();
  const { t } = useI18n();

  const [loading, setLoading] = useState(false);
  const [articles, setArticles] = useState<ArticleItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [reviewModalVisible, setReviewModalVisible] = useState(false);
  const [selectedArticle, setSelectedArticle] = useState<ArticleItem | null>(null);
  const [reviewAction, setReviewAction] = useState<'approve' | 'reject'>('approve');
  const [reviewComment, setReviewComment] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const fetchArticles = useCallback(async () => {
    setLoading(true);
    try {
      const response = await KnowledgeBaseApi.getArticles({
        page,
        pageSize,
        status: statusFilter as any,
      });
      // 过滤出待审核的文章（如果选择了状态）或所有需要审核的文章
      const items = (response.articles || []) as unknown as ArticleItem[];
      const pendingReview = items.filter(a => a.status === 'pending_review');

      if (statusFilter === 'pending_review' || !statusFilter) {
        setArticles(pendingReview);
        setTotal(pendingReview.length);
      } else {
        setArticles(items);
        setTotal(response.total || 0);
      }
    } catch (error) {
      console.error('Failed to fetch articles:', error);
      message.error('获取文章列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, statusFilter]);

  useEffect(() => {
    fetchArticles();
  }, [fetchArticles]);

  const handleReview = async () => {
    if (!selectedArticle) return;
    setSubmitting(true);
    try {
      const request: ReviewArticleRequest = {
        action: reviewAction,
        comment: reviewComment || undefined,
      };
      await KnowledgeBaseApi.reviewArticle(String(selectedArticle.id), request);
      message.success(reviewAction === 'approve' ? '已批准发布' : '已拒绝');
      setReviewModalVisible(false);
      setSelectedArticle(null);
      setReviewComment('');
      fetchArticles();
    } catch (error: any) {
      message.error(error?.message || '操作失败');
    } finally {
      setSubmitting(false);
    }
  };

  const getStatusTag = (status: string) => {
    const config = statusTagMap[status] || { color: 'default', text: status };
    return (
      <Tag icon={config.icon} color={config.color}>
        {config.text}
      </Tag>
    );
  };

  const columns: ColumnsType<ArticleItem> = [
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      render: (title: string, record) => (
        <div>
          <Text strong>{title}</Text>
          {record.summary && (
            <Paragraph type="secondary" ellipsis={{ rows: 1 }} className="text-xs mt-1">
              {record.summary}
            </Paragraph>
          )}
        </div>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      width: 120,
    },
    {
      title: '作者',
      dataIndex: 'authorName',
      key: 'authorName',
      width: 120,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => getStatusTag(status),
    },
    {
      title: '提交时间',
      dataIndex: 'submittedAt',
      key: 'submittedAt',
      width: 180,
      render: (date: string) => (date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            icon={<Eye />}
            onClick={() => {
              setSelectedArticle(record);
              setDetailModalVisible(true);
            }}
          >
            查看
          </Button>
          {record.status === 'pending_review' && (
            <>
              <Button
                type="link"
                icon={<CheckCircle />}
                onClick={() => {
                  setSelectedArticle(record);
                  setReviewAction('approve');
                  setReviewModalVisible(true);
                }}
              >
                批准
              </Button>
              <Button
                type="link"
                danger
                icon={<XCircle />}
                onClick={() => {
                  setSelectedArticle(record);
                  setReviewAction('reject');
                  setReviewModalVisible(true);
                }}
              >
                拒绝
              </Button>
            </>
          )}
        </Space>
      ),
    },
  ];

  const statusOptions = [
    { value: 'pending_review', label: '待审核' },
    { value: 'approved', label: '已发布' },
    { value: 'rejected', label: '已拒绝' },
    { value: 'draft', label: '草稿' },
  ];

  return (
    <PageContainer title="知识库审核" description="审核和批准待发布的知识库文章">
      <Card className="shadow-sm rounded-lg">
        <div className="mb-4">
          <Space wrap>
            <Select
              placeholder="状态筛选"
              value={statusFilter}
              onChange={setStatusFilter}
              allowClear
              options={statusOptions}
              style={{ width: 150 }}
            />
            <Button onClick={fetchArticles}>刷新</Button>
          </Space>
        </div>

        {articles.length > 0 ? (
          <Table
            columns={columns}
            dataSource={articles}
            rowKey="id"
            loading={loading}
            pagination={{
              current: page,
              pageSize: pageSize,
              total,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: total => `共 ${total} 条记录`,
              onChange: (p, ps) => {
                setPage(p);
                setPageSize(ps);
              },
            }}
          />
        ) : (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={
              statusFilter === 'pending_review' || !statusFilter
                ? '暂没有待审核的文章'
                : '没有找到符合条件的文章'
            }
          />
        )}
      </Card>

      {/* 文章详情弹窗 */}
      <Modal
        title="文章详情"
        open={detailModalVisible}
        onCancel={() => {
          setDetailModalVisible(false);
          setSelectedArticle(null);
        }}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            关闭
          </Button>,
          selectedArticle?.status === 'pending_review' && (
            <Button
              key="reject"
              danger
              onClick={() => {
                setDetailModalVisible(false);
                setSelectedArticle(selectedArticle);
                setReviewAction('reject');
                setReviewModalVisible(true);
              }}
            >
              拒绝
            </Button>
          ),
          selectedArticle?.status === 'pending_review' && (
            <Button
              key="approve"
              type="primary"
              onClick={() => {
                setDetailModalVisible(false);
                setSelectedArticle(selectedArticle);
                setReviewAction('approve');
                setReviewModalVisible(true);
              }}
            >
              批准发布
            </Button>
          ),
        ]}
        width={800}
      >
        {selectedArticle && (
          <div>
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="标题" span={2}>
                {selectedArticle.title}
              </Descriptions.Item>
              <Descriptions.Item label="分类">{selectedArticle.category || '-'}</Descriptions.Item>
              <Descriptions.Item label="状态">
                {getStatusTag(selectedArticle.status)}
              </Descriptions.Item>
              <Descriptions.Item label="作者">
                {selectedArticle.authorName || selectedArticle.author || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="提交时间">
                {selectedArticle.submittedAt
                  ? dayjs(selectedArticle.submittedAt).format('YYYY-MM-DD HH:mm')
                  : '-'}
              </Descriptions.Item>
            </Descriptions>

            {selectedArticle.summary && (
              <>
                <Divider>摘要</Divider>
                <Paragraph>{selectedArticle.summary}</Paragraph>
              </>
            )}

            {selectedArticle.content && (
              <>
                <Divider>内容预览</Divider>
                <div
                  className="prose max-w-none"
                  dangerouslySetInnerHTML={{
                    __html:
                      selectedArticle.content.substring(0, 500) +
                      (selectedArticle.content.length > 500 ? '...' : ''),
                  }}
                />
              </>
            )}

            {selectedArticle.tags && selectedArticle.tags.length > 0 && (
              <>
                <Divider>标签</Divider>
                <Space>
                  {selectedArticle.tags.map((tag, idx) => (
                    <Tag key={idx}>{tag}</Tag>
                  ))}
                </Space>
              </>
            )}
          </div>
        )}
      </Modal>

      {/* 审核弹窗 */}
      <Modal
        title={
          <Space>
            <AlertCircle
              style={{ color: reviewAction === 'approve' ? '#52c41a' : '#ff4d4f' }}
            />
            {reviewAction === 'approve' ? '批准发布' : '拒绝发布'}
          </Space>
        }
        open={reviewModalVisible}
        onOk={handleReview}
        onCancel={() => {
          setReviewModalVisible(false);
          setSelectedArticle(null);
          setReviewComment('');
        }}
        okText={reviewAction === 'approve' ? '批准' : '拒绝'}
        okButtonProps={{ danger: reviewAction === 'reject', loading: submitting }}
        cancelText="取消"
      >
        <div className="py-4">
          <p className="mb-4">
            确定要{reviewAction === 'approve' ? '批准' : '拒绝'}发布文章 &quot;
            <Text strong>{selectedArticle?.title}</Text>&quot; 吗？
          </p>
          <div>
            <Text type="secondary" className="mb-2 block">
              审核意见（可选）：
            </Text>
            <textarea
              className="w-full border rounded p-2 min-h-[80px]"
              placeholder="请输入审核意见..."
              value={reviewComment}
              onChange={e => setReviewComment(e.target.value)}
            />
          </div>
        </div>
      </Modal>
    </PageContainer>
  );
}
