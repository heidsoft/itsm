'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Select,
  App,
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
import { SafeContent } from '@/components/common/SafeContent';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import { ArticleStatus } from '@/types/knowledge-base';
import type { ReviewArticleRequest } from '@/types/knowledge-base';

// 本地定义的类型以匹配API响应
interface ArticleItem {
  id: number | string;
  title: string;
  content: string;
  summary?: string;
  category?: string;
  tags?: string[];
  /** 后端 review_status 的真实值（draft/under_review/approved/rejected） */
  reviewStatus?: KnowledgeReviewStatus;
  /** 对外可见投影，由 reviewStatus 派生 */
  isPublished?: boolean;
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
import { KnowledgeReviewStatus } from '@/types/knowledge-base';

const { Text, Paragraph } = Typography;

// 状态标签映射：权威来源是后端 reviewStatus（KnowledgeReviewStatus 枚举）。
//
// 这里原先用的是前端自造的 ArticleStatus，而**后端文章实体上不存在 status 字段**
// （只有 is_published 布尔），导致本页过滤条件永远命中不到任何数据 ——
// 页面看着正常，实际上恒为空。改为读真实 review_status 后才真正可用。
const statusTagMap: Record<string, { color: string; text: string; icon?: React.ReactNode }> = {
  [KnowledgeReviewStatus.DRAFT]: { color: 'default', text: '草稿' },
  [KnowledgeReviewStatus.UNDER_REVIEW]: { color: 'orange', text: '待审核', icon: <Clock /> },
  [KnowledgeReviewStatus.APPROVED]: { color: 'green', text: '已通过', icon: <CheckCircle /> },
  [KnowledgeReviewStatus.REJECTED]: { color: 'red', text: '已驳回', icon: <XCircle /> },
};

/** 是否处于评审中（唯一允许 approve/reject 的状态）。 */
const isPendingReview = (reviewStatus?: string): boolean =>
  reviewStatus === KnowledgeReviewStatus.UNDER_REVIEW;

export default function KnowledgeReviewListPage() {
  const router = useRouter();
  const { t } = useI18n();
  const { message } = App.useApp();

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
        status: statusFilter as unknown as ArticleStatus,
      });
      // 过滤出待审核的文章（兼容下划线/驼峰两种历史取值）
      const items = (response.items ?? []) as unknown as ArticleItem[];
      // 用真实 reviewStatus 过滤，不再读不存在的 a.status。
      const pendingReview = items.filter(a => isPendingReview(a.reviewStatus));

      if (isPendingReview(statusFilter) || !statusFilter) {
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
      // 驳回必须给理由：后端 ReviewRequiresComment 会拒绝空意见，
      // 前端先拦一道给出明确提示，而不是等一次 400。
      if (reviewAction === 'reject' && !reviewComment.trim()) {
        message.error('驳回必须填写评审意见');
        return;
      }

      const id = String(selectedArticle.id);
      // 此前调用的 /review/decision 后端从未注册（断链），改为按动作打真实端点。
      if (reviewAction === 'approve') {
        await KnowledgeBaseApi.approveReview(id, reviewComment || undefined);
      } else {
        await KnowledgeBaseApi.rejectReview(id, reviewComment);
      }
      message.success(reviewAction === 'approve' ? '已批准并发布' : '已驳回');
      setReviewModalVisible(false);
      setSelectedArticle(null);
      setReviewComment('');
      fetchArticles();
    } catch (error: unknown) {
      message.error((error instanceof Error ? error.message : undefined) || '操作失败');
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
      title: '评审状态',
      dataIndex: 'reviewStatus',
      key: 'reviewStatus',
      width: 100,
      render: (reviewStatus: string) => getStatusTag(reviewStatus),
    },
    {
      title: '最近评审时间',
      dataIndex: 'reviewedAt',
      key: 'reviewedAt',
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
          {isPendingReview(record.reviewStatus) && (
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

  const statusOptions: Array<{ value: KnowledgeReviewStatus | string; label: string }> = [
    { value: KnowledgeReviewStatus.UNDER_REVIEW, label: '待审核' },
    { value: KnowledgeReviewStatus.APPROVED, label: '已通过' },
    { value: KnowledgeReviewStatus.REJECTED, label: '已驳回' },
    { value: KnowledgeReviewStatus.DRAFT, label: '草稿' },
    { value: 'rejected', label: '已拒绝' },
    { value: ArticleStatus.DRAFT, label: '草稿' },
  ];

  return (
    <PageContainer title="知识库审核" description="审核待发布的知识文章">
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
              isPendingReview(statusFilter) || !statusFilter
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
          isPendingReview(selectedArticle?.reviewStatus) && (
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
          isPendingReview(selectedArticle?.reviewStatus) && (
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
              <Descriptions.Item label="评审状态">
                {getStatusTag(selectedArticle.reviewStatus ?? KnowledgeReviewStatus.DRAFT)}
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
                <SafeContent
                  className="prose max-w-none"
                  sanitizeHtml
                  content={
                    selectedArticle.content.substring(0, 500) +
                    (selectedArticle.content.length > 500 ? '...' : '')
                  }
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
